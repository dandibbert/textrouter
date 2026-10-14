import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { EditorView, ViewUpdate } from "@codemirror/view";
import {
  BookOpen,
  ClipboardCopy,
  ClipboardPaste,
  ClipboardType,
  Copy,
  FileText,
  ListOrdered,
  PanelLeft,
  Pilcrow,
  Replace,
  Search,
  Sparkles,
  WrapText,
  Wand2,
  ZoomIn,
} from "lucide-react";
import { onFileDrop } from "mygo-runtime";
import { Document, Soap, events, type Boot, type CleanerInfo, type DocInfo, type Prefs } from "../mygo";
import { keys, setPref, useBoot } from "../lib/boot";
import { MenuButton, Segmented, Toasts, errorText, toast } from "../lib/ui";
import { debounce, textStats, type Stats } from "../lib/text";
import { createEditor, reconfigure, type EditorOptions } from "./editor";
import { CleanerList, type CleanerListHandle } from "./CleanerList";
import { FindPanel, type FindPanelHandle } from "./FindPanel";

function editorOptions(p: Prefs): EditorOptions {
  return {
    lineNumbers: p.showLineNumbers,
    wrap: p.wrapText,
    invisibles: p.showInvisibles,
    activeLine: p.highlightCurrentLine,
    fontSize: p.fontSize,
    zoom: p.zoom,
    fontFamily: p.fontFamily,
  };
}

/** Replaces from..to with text as the smallest change, so the cursor,
 * scroll position and undo stay sensible. */
function minimalChange(view: EditorView, from: number, to: number, text: string) {
  const old = view.state.sliceDoc(from, to);
  let a = 0;
  const max = Math.min(old.length, text.length);
  while (a < max && old.charCodeAt(a) === text.charCodeAt(a)) a++;
  let b = 0;
  while (b < max - a && old.charCodeAt(old.length - 1 - b) === text.charCodeAt(text.length - 1 - b)) b++;
  return { from: from + a, to: to - b, insert: text.slice(a, text.length - b) };
}

export function DocumentApp() {
  const boot = useBoot();
  const [info, setInfo] = useState<DocInfo | null>(null);
  useEffect(() => {
    Document.info().then(setInfo);
    return events.docChanged.on((d) => setInfo((cur) => ({ ...d, text: cur?.text ?? d.text })));
  }, []);
  useEffect(() => {
    if (info) document.title = info.name;
  }, [info?.name]);
  if (!boot || !info) return <div className="doc" />;
  return <DocumentWindow boot={boot} info={info} />;
}

function DocumentWindow({ boot, info }: { boot: Boot; info: DocInfo }) {
  const prefs = boot.prefs;
  const host = useRef<HTMLDivElement>(null);
  const viewRef = useRef<EditorView | null>(null);
  const findRef = useRef<FindPanelHandle>(null);
  const cleanersRef = useRef<CleanerListHandle>(null);
  const saved = useRef(info.text);
  const [tab, setTab] = useState<"cleaners" | "find">("cleaners");
  const [stats, setStats] = useState<Stats>(() => textStats(info.text));
  const [selection, setSelection] = useState(0);
  const [dirty, setDirty] = useState(false);
  const [width, setWidth] = useState(prefs.sidebarWidth || 320);
  const [sidebar, setSidebar] = useState(!prefs.sidebarHidden);
  const kindRef = useRef(info.kind);
  kindRef.current = info.kind;
  const view = useCallback(() => viewRef.current, []);

  const sync = useMemo(
    () =>
      debounce((text: string) => {
        const d = kindRef.current !== "clipboard" && text !== saved.current;
        setDirty(d);
        Document.sync(text, d);
        setStats(textStats(text));
      }, 150),
    [],
  );

  // Editor.
  useEffect(() => {
    const onUpdate = (u: ViewUpdate) => {
      if (u.docChanged) {
        sync(u.state.doc.toString());
        findRef.current?.docChanged();
      }
      if (u.selectionSet || u.docChanged) {
        const sel = u.state.selection.main;
        setSelection(sel.to - sel.from);
      }
    };
    const v = createEditor(host.current!, info.text, editorOptions(prefs), onUpdate);
    viewRef.current = v;
    v.focus();
    return () => v.destroy();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    if (viewRef.current) reconfigure(viewRef.current, editorOptions(prefs));
  }, [prefs.showLineNumbers, prefs.wrapText, prefs.showInvisibles, prefs.highlightCurrentLine, prefs.fontSize, prefs.zoom, prefs.fontFamily]);

  useEffect(() => {
    if (info.kind === "file") {
      saved.current = viewRef.current?.state.doc.toString() ?? info.text;
      setDirty(false);
    }
  }, [info.kind, info.path]);

  const text = () => viewRef.current?.state.doc.toString() ?? "";

  const replaceAllText = (t: string, message?: string) => {
    const v = viewRef.current;
    if (!v) return;
    v.dispatch({ changes: minimalChange(v, 0, v.state.doc.length, t), userEvent: "input.paste" });
    if (message) toast(message, "ok", `${keys("CmdOrCtrl+Z")} 撤销`);
  };

  // ------------------------------------------------------------- actions

  const applyCleaner = async (c: { name: string; zh: string }) => {
    const v = viewRef.current;
    if (!v) return;
    const sel = v.state.selection.main;
    let { from, to } = sel;
    const whole = from === to;
    if (whole) {
      if (!prefs.applyToAllWhenNoSelection) return toast("先选中要清理的文本", "info");
      from = 0;
      to = v.state.doc.length;
    }
    try {
      const r = await Soap.apply(c.name, v.state.sliceDoc(from, to));
      if (!r.changed) {
        toast(`「${c.zh}」没有改变文本`, "info");
      } else {
        const change = minimalChange(v, from, to, r.text);
        v.dispatch({
          changes: change,
          selection: whole ? undefined : { anchor: from, head: from + r.text.length },
          userEvent: "input.cleaner",
          scrollIntoView: true,
        });
        toast(`已应用「${c.zh}」${whole ? "" : "到选中文本"}`, "ok", `${keys("CmdOrCtrl+Z")} 撤销`);
      }
      if (r.warnings?.length) toast(r.warnings.join("；"), "warn", undefined, 5000);
    } catch (e) {
      toast(errorText(e), "err");
    }
  };

  const myScrub = () => {
    const name = prefs.myScrub || "Scrub";
    const c = boot.cleaners.find((x) => x.name === name);
    applyCleaner({ name, zh: c?.zh ?? name });
  };

  const copy = async () => {
    const v = viewRef.current;
    if (!v) return;
    const { from, to } = v.state.selection.main;
    if (from === to) return copyAll();
    await Soap.writeClipboard(v.state.sliceDoc(from, to));
    toast(`已拷贝选中的 ${to - from} 个字符`);
  };
  const copyAll = async () => {
    const v = viewRef.current;
    if (!v) return;
    await Soap.writeClipboard(text());
    v.dispatch({ selection: { anchor: 0, head: v.state.doc.length } });
    toast("已拷贝全部文本");
  };
  const paste = async () => {
    const v = viewRef.current;
    if (!v) return;
    const clip = await Soap.readClipboard();
    if (!clip) return toast("剪贴板里没有文本", "info");
    v.dispatch(v.state.replaceSelection(clip), { userEvent: "input.paste", scrollIntoView: true });
    v.focus();
  };
  const pasteOver = async () => {
    const clip = await Soap.readClipboard();
    if (!clip) return toast("剪贴板里没有文本", "info");
    replaceAllText(clip, "已用剪贴板内容替换全部文本");
  };

  const save = async (as: boolean) => {
    try {
      const path = await Document.save(text(), as);
      if (path) {
        saved.current = text();
        setDirty(false);
        toast("已存储");
      }
    } catch (e) {
      toast(errorText(e), "err");
    }
  };

  const toggleSidebar = (v = !sidebar) => {
    setSidebar(v);
    setPref(prefs, "sidebarHidden", !v);
  };

  const [focusFind, setFocusFind] = useState(0);
  const showFind = () => {
    if (!sidebar) toggleSidebar(true);
    setTab("find");
    setFocusFind((n) => n + 1);
  };
  // The panel is display:none until the tab switch renders, so focus after it.
  useEffect(() => {
    if (focusFind) findRef.current?.focus();
  }, [focusFind]);

  const zoom = (z: number) => setPref(prefs, "zoom", Math.max(50, Math.min(300, z)));

  // Menu commands, window events.
  const handlers = useRef<Record<string, () => void>>({});
  handlers.current = {
    open: () => Document.open().catch((e) => toast(errorText(e), "err")),
    save: () => save(false),
    saveAs: () => save(true),
    copyAll,
    pasteOver,
    myScrub,
    find: showFind,
    findNext: () => findRef.current?.next(1),
    findPrevious: () => findRef.current?.next(-1),
    useSelection: () => {
      findRef.current?.useSelection();
      showFind();
    },
    replace: () => findRef.current?.replace(),
    replaceAll: () => findRef.current?.replaceAll(),
    showCleaners: () => {
      if (!sidebar) toggleSidebar(true);
      setTab("cleaners");
    },
    searchCleaners: () => {
      if (!sidebar) toggleSidebar(true);
      setTab("cleaners");
      setTimeout(() => cleanersRef.current?.focusSearch(), 0);
    },
    toggleSidebar: () => toggleSidebar(),
    zoomIn: () => zoom(prefs.zoom + 10),
    zoomOut: () => zoom(prefs.zoom - 10),
    zoomReset: () => zoom(100),
  };
  useEffect(() => {
    const offs = [
      events.command.on((c) => handlers.current[c]?.()),
      events.docLoad.on((t) => replaceAllText(t, "已载入剪贴板的新内容")),
      events.docCloseRequested.on(() => {
        sync.cancel();
        Document.close(text());
      }),
      onFileDrop(({ paths }) => Document.openPaths(paths).catch((e) => toast(errorText(e), "err"))),
    ];
    return () => offs.forEach((off) => off());
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Sidebar resizing.
  const startResize = (e: React.MouseEvent) => {
    const x0 = e.clientX;
    const w0 = width;
    let w = w0;
    const move = (ev: MouseEvent) => {
      w = Math.max(260, Math.min(560, w0 + ev.clientX - x0));
      setWidth(w);
    };
    const up = () => {
      window.removeEventListener("mousemove", move);
      window.removeEventListener("mouseup", up);
      setPref(prefs, "sidebarWidth", w);
    };
    window.addEventListener("mousemove", move);
    window.addEventListener("mouseup", up);
  };

  const myScrubName = boot.cleaners.find((c) => c.name === (prefs.myScrub || "Scrub"))?.zh ?? prefs.myScrub;

  return (
    <div className="doc">
      <div className="doc-body">
        {(
          <aside className="sidebar" style={{ width, display: sidebar ? "flex" : "none" }}>
            <div className="sidebar-top">
              <span className="spacer" />
              <button className="tool icon-only" onClick={() => toggleSidebar(false)} title={`隐藏侧栏 ${keys("CmdOrCtrl+Alt+S")}`}>
                <PanelLeft />
              </button>
            </div>
            <div className="sidebar-tabs">
              <Segmented
                value={tab}
                onChange={(t) => (t === "find" ? showFind() : setTab(t))}
                options={[
                  { value: "cleaners", label: "清理器", icon: <Wand2 />, title: `清理器列表 ${keys("CmdOrCtrl+1")}` },
                  { value: "find", label: "查找替换", icon: <Search />, title: `查找与替换 ${keys("CmdOrCtrl+F")}` },
                ]}
              />
            </div>
            <div style={{ display: tab === "cleaners" ? "flex" : "none", flex: 1, minHeight: 0, flexDirection: "column" }}>
              <CleanerList ref={cleanersRef} boot={boot} onApply={(c: CleanerInfo) => applyCleaner(c)} />
            </div>
            <div style={{ display: tab === "find" ? "flex" : "none", flex: 1, minHeight: 0, flexDirection: "column" }}>
              <FindPanel ref={findRef} boot={boot} view={view} onActivate={() => setTab("find")} />
            </div>
            <div className="resizer" onMouseDown={startResize} />
          </aside>
        )}
        <main className="editor-col">
          <header className={"titlebar main-toolbar" + (sidebar ? "" : " inset")}>
            {!sidebar && (
              <button className="tool icon-only" onClick={() => toggleSidebar(true)} title={`显示侧栏 ${keys("CmdOrCtrl+Alt+S")}`}>
                <PanelLeft />
              </button>
            )}
            <div className="title-block">
              <span className={"doc-icon" + (info.kind === "clipboard" ? " clip" : "")}>
                {info.kind === "clipboard" ? <ClipboardType /> : <FileText />}
              </span>
              <span className="name" title={info.path}>{info.name}</span>
              {dirty && <span className="dirty-dot" title="有未存储的更改" />}
              {info.kind === "clipboard" && prefs.copyBackOnClose && (
                <span className="chip" title="关闭窗口或退出时，文本会自动拷贝回剪贴板">
                  关闭时拷回
                </span>
              )}
            </div>
            <div className="spacer" />
            <button className="tool primary" onClick={myScrub} title={`MyScrub 一键清理：应用「${myScrubName}」${keys("CmdOrCtrl+J")}\n可在偏好设置中更换`}>
              <Sparkles />
              <span>一键清理</span>
            </button>
            <div className="tool-cluster">
              <button className="tool collapsible" onClick={copy} title="拷贝选中的文本（没有选中则拷贝全部）">
                <Copy />
                <span>拷贝</span>
              </button>
              <button className="tool collapsible" onClick={paste} title="在光标处粘贴剪贴板内容">
                <ClipboardPaste />
                <span>粘贴</span>
              </button>
              <button className="tool collapsible" onClick={copyAll} title={`全选并拷贝 ${keys("CmdOrCtrl+Shift+C")}`}>
                <ClipboardCopy />
                <span>全部拷贝</span>
              </button>
              <button className="tool collapsible" onClick={pasteOver} title={`用剪贴板内容替换全部文本 ${keys("CmdOrCtrl+Shift+V")}`}>
                <Replace />
                <span>粘贴覆盖</span>
              </button>
            </div>
            <button className="tool collapsible" onClick={() => Document.openLibrary("")} title={`我的库：自定义清理器和分组 ${keys("CmdOrCtrl+Shift+L")}`}>
              <BookOpen />
              <span>我的库</span>
            </button>
          </header>
          <div className="editor-host" ref={host} />
          <footer className="statusbar">
            <MenuButton
              className="status-btn"
              icon={<ZoomIn />}
              label={`${prefs.zoom}%`}
              title="缩放"
              items={[50, 75, 90, 100, 110, 125, 150, 175, 200, 250].map((z) => ({ label: `${z}%`, checked: prefs.zoom === z, onSelect: () => zoom(z) }))}
            />
            <button className={"status-btn" + (prefs.showLineNumbers ? " on" : "")} onClick={() => setPref(prefs, "showLineNumbers", !prefs.showLineNumbers)} title="行号">
              <ListOrdered />
            </button>
            <button className={"status-btn" + (prefs.wrapText ? " on" : "")} onClick={() => setPref(prefs, "wrapText", !prefs.wrapText)} title="自动换行">
              <WrapText />
            </button>
            <button className={"status-btn" + (prefs.showInvisibles ? " on" : "")} onClick={() => setPref(prefs, "showInvisibles", !prefs.showInvisibles)} title="显示不可见字符（空格、制表符、换行）">
              <Pilcrow />
            </button>
            <div className="spacer" />
            {prefs.showWordCount ? (
              <button className="status-btn" onClick={() => setPref(prefs, "showWordCount", false)} title="点击隐藏统计">
                <span className="stats">
                  {selection > 0 && (
                    <span>
                      已选 <b>{selection}</b>
                    </span>
                  )}
                  <span>
                    段落 <b>{stats.paragraphs}</b>
                  </span>
                  <span>
                    词 <b>{stats.words}</b>
                  </span>
                  <span>
                    字符 <b>{stats.chars}</b>
                  </span>
                  <span>
                    不含空白 <b>{stats.charsNoSpace}</b>
                  </span>
                </span>
              </button>
            ) : (
              <button className="status-btn" onClick={() => setPref(prefs, "showWordCount", true)}>
                显示统计
              </button>
            )}
          </footer>
        </main>
      </div>
      <Toasts />
    </div>
  );
}
