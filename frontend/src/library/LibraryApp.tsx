import { useEffect, useMemo, useRef, useState, type DragEvent } from "react";
import {
  ChevronDown,
  ChevronRight,
  ClipboardPaste,
  Copy,
  Download,
  FolderTree,
  GripVertical,
  Layers,
  MoreHorizontal,
  Plus,
  Search,
  Trash2,
  Upload,
  Wand2,
} from "lucide-react";
import { Document, Library, Soap, events, type Action, type ActionInfo, type Boot, type CustomCleaner, type RunResult } from "../mygo";
import { useBoot } from "../lib/boot";
import { MenuButton, Segmented, Toasts, errorText, toast } from "../lib/ui";
import { debounce } from "../lib/text";
import { ActionForm, summary, type FormContext } from "./ActionForm";
import { definitions, getAt, insertAt, moveTo, moveWithin, newAction, removeAt, updateAt, type Path } from "./tree";
import { GroupEditor } from "./GroupEditor";
import "./library.css";

const key = (p: Path) => p.join(".");
const DRAG_PATH = "application/x-textrouter-path";
const DRAG_NEW = "application/x-textrouter-action";

export default function LibraryApp() {
  const boot = useBoot();
  if (!boot) return <div className="page" />;
  return <LibraryWindow boot={boot} />;
}

function LibraryWindow({ boot }: { boot: Boot }) {
  const [tab, setTab] = useState<"cleaners" | "groups">("cleaners");
  const [selected, setSelected] = useState<string>(boot.custom[0]?.uuid ?? "");
  const [query, setQuery] = useState("");

  useEffect(() => {
    const go = (id: string) => {
      if (!id) return;
      if (id === "groups") setTab("groups");
      else {
        setTab("cleaners");
        setSelected(id);
      }
    };
    Library.pendingSelection().then(go);
    return events.librarySelect.on(go);
  }, []);

  useEffect(() => {
    if (selected && !boot.custom.some((c) => c.uuid === selected)) setSelected(boot.custom[0]?.uuid ?? "");
    if (!selected && boot.custom.length) setSelected(boot.custom[0].uuid);
  }, [boot.custom, selected]);

  const list = boot.custom.filter((c) => !query || c.name.toLowerCase().includes(query.toLowerCase()) || (c.comment ?? "").toLowerCase().includes(query.toLowerCase()));
  const current = boot.custom.find((c) => c.uuid === selected);

  const create = async () => {
    const c = await Library.newCleaner("新清理器");
    setSelected(c.uuid);
  };
  const remove = async (c: CustomCleaner) => {
    if (!(await Document.confirm(`删除清理器「${c.name}」？`, "此操作不能撤销。", "删除"))) return;
    await Library.deleteCleaner(c.uuid);
    toast(`已删除「${c.name}」`);
  };
  const doImport = async () => {
    try {
      const r = await Library.import();
      if (r.added.length) {
        toast(`已导入 ${r.added.length} 个清理器${r.skipped ? `，跳过 ${r.skipped} 个` : ""}`);
      }
    } catch (e) {
      toast(errorText(e), "err", undefined, 5000);
    }
  };
  const doExport = async (uuids: string[]) => {
    try {
      const p = await Library.export(uuids);
      if (p) toast("已导出到 " + p);
    } catch (e) {
      toast(errorText(e), "err");
    }
  };

  return (
    <div className="page lib">
      <header className="titlebar">
        <span className="page-title">我的库</span>
        <div className="no-drag" style={{ marginLeft: 14, width: 220 }}>
          <Segmented
            small
            value={tab}
            onChange={setTab}
            options={[
              { value: "cleaners", label: "清理器", icon: <Layers /> },
              { value: "groups", label: "分组", icon: <FolderTree /> },
            ]}
          />
        </div>
        <span className="spacer" />
        <button className="tool" onClick={doImport} title="导入 TextSoap 的 .textsoap9Lib / .textsoapCleaner 文件">
          <Download />
          <span>导入</span>
        </button>
        <MenuButton
          className="tool"
          icon={<Upload />}
          label={<span>导出</span>}
          align="right"
          items={[
            { label: current ? `导出「${current.name}」…` : "导出所选清理器…", disabled: !current, onSelect: () => current && doExport([current.uuid]) },
            { label: "导出全部清理器…", disabled: !boot.custom.length, onSelect: () => doExport(boot.custom.map((c) => c.uuid)) },
            "sep",
            { label: "在文件夹中显示资料库", onSelect: () => Library.revealLibrary() },
          ]}
        />
      </header>

      {tab === "groups" ? (
        <GroupEditor boot={boot} />
      ) : (
        <div className="lib-body">
          <aside className="lib-list">
            <div style={{ padding: 10 }}>
              <div className="search">
                <Search />
                <input className="input" placeholder="搜索我的清理器" value={query} onChange={(e) => setQuery(e.target.value)} />
              </div>
            </div>
            <div className="lib-items">
              {list.map((c) => (
                <button key={c.uuid} className={"lib-item" + (c.uuid === selected ? " on" : "")} onClick={() => setSelected(c.uuid)}>
                  <Layers />
                  <span className="names">
                    <span className="n">{c.name}</span>
                    <span className="s">{c.actions.length} 个动作{c.comment ? ` · ${c.comment}` : ""}</span>
                  </span>
                </button>
              ))}
              {!list.length && <div className="empty">{query ? "没有找到" : "还没有自定义清理器"}</div>}
            </div>
            <div className="lib-foot">
              <button className="btn small" onClick={create} title="新建清理器">
                <Plus /> 新建
              </button>
              <button className="btn small icon" disabled={!current} title="复制" onClick={() => current && Library.duplicateCleaner(current.uuid).then((c) => setSelected(c.uuid))}>
                <Copy />
              </button>
              <button className="btn small icon danger" disabled={!current} title="删除" onClick={() => current && remove(current)}>
                <Trash2 />
              </button>
            </div>
          </aside>
          {current ? (
            <CleanerEditor key={current.uuid} boot={boot} cleaner={current} />
          ) : (
            <div className="lib-empty">
              <Layers />
              <div>自定义清理器把多个动作串在一起，一次完成整套清理。</div>
              <button className="btn primary" onClick={create}>
                <Plus /> 新建清理器
              </button>
            </div>
          )}
        </div>
      )}
      <Toasts />
    </div>
  );
}

// ---------------------------------------------------------------------------

function CleanerEditor({ boot, cleaner }: { boot: Boot; cleaner: CustomCleaner }) {
  const [draft, setDraft] = useState<CustomCleaner>(cleaner);
  const [sel, setSel] = useState<Path | null>(null);
  const [open, setOpen] = useState<Set<string>>(new Set());
  const [sample, setSample] = useState(() => localStorage.getItem("sample") ?? "");
  const [result, setResult] = useState<RunResult | null>(null);
  const [view, setView] = useState<"final" | "step">("final");
  const [drop, setDrop] = useState<{ parent: Path; index: number } | null>(null);
  const [previewH, setPreviewH] = useState(() => Number(localStorage.getItem("previewH")) || 230);
  const catalog = useMemo(() => new Map(boot.catalog.map((a) => [a.id, a])), [boot.catalog]);
  const saving = useRef(false);

  const save = useMemo(
    () =>
      debounce(async (c: CustomCleaner) => {
        saving.current = true;
        try {
          const s = await Library.saveCleaner(c);
          if (s.name !== c.name) {
            setDraft((d) => ({ ...d, name: s.name }));
            toast(`已有同名清理器，已改名为「${s.name}」`, "info");
          }
        } catch (e) {
          toast(errorText(e), "err");
        } finally {
          saving.current = false;
        }
      }, 450),
    [],
  );
  const change = (fn: (c: CustomCleaner) => CustomCleaner) =>
    setDraft((d) => {
      const n = fn(d);
      save(n);
      return n;
    });
  const setActions = (fn: (a: Action[]) => Action[]) => change((c) => ({ ...c, actions: fn(c.actions ?? []) }));

  // Preview.
  useEffect(() => {
    localStorage.setItem("sample", sample);
    const t = setTimeout(() => Soap.run(draft.actions ?? [], sample).then(setResult), 220);
    return () => clearTimeout(t);
  }, [draft.actions, sample]);

  const defs = definitions(draft.actions ?? []);
  const ctx: FormContext = { lists: defs.lists, macros: defs.macros, cleaners: boot.cleaners, allowScripts: boot.prefs.allowShellScripts, self: draft.name };

  // Where the toolbox inserts: into the selected container, else after the selected action.
  const target = (): { parent: Path; index: number } => {
    if (!sel) return { parent: [], index: (draft.actions ?? []).length };
    const a = getAt(draft.actions ?? [], sel);
    if (a && catalog.get(a.identifier)?.container) return { parent: sel, index: (a.actionList ?? []).length };
    return { parent: sel.slice(0, -1), index: sel[sel.length - 1] + 1 };
  };
  const add = (info: ActionInfo, where = target()) => {
    setActions((as) => insertAt(as, where.parent, where.index, newAction(info)));
    const p = [...where.parent, where.index];
    setSel(p);
    setOpen((o) => new Set(o).add(key(p)));
  };

  const onDrop = (e: DragEvent, parent: Path, index: number) => {
    e.preventDefault();
    e.stopPropagation();
    setDrop(null);
    const id = e.dataTransfer.getData(DRAG_NEW);
    if (id) {
      const info = catalog.get(id);
      if (info) add(info, { parent, index });
      return;
    }
    const from = e.dataTransfer.getData(DRAG_PATH);
    if (from) {
      const fp = from.split(".").map(Number);
      setDraft((d) => {
        const r = moveTo(d.actions ?? [], fp, parent, index);
        const n = { ...d, actions: r.list };
        save(n);
        setSel(r.path);
        return n;
      });
    }
  };
  const dragOver = (e: DragEvent, parent: Path, index: number) => {
    e.preventDefault();
    e.stopPropagation();
    e.dataTransfer.dropEffect = e.dataTransfer.types.includes(DRAG_NEW) ? "copy" : "move";
    if (!drop || key(drop.parent) !== key(parent) || drop.index !== index) setDrop({ parent, index });
  };

  const steps = result?.steps ?? [];
  const stepView = view === "step" && sel && sel.length === 1 && steps[sel[0]];
  const shown = stepView ? steps[sel![0]].output : result?.text ?? "";

  const renderList = (actions: Action[], parent: Path, depth: number) => (
    <div className={"alist" + (depth ? " nested" : "")} onDragOver={(e) => dragOver(e, parent, actions.length)} onDrop={(e) => onDrop(e, parent, actions.length)}>
      {actions.map((a, i) => {
        const path = [...parent, i];
        const k = key(path);
        const info = catalog.get(a.identifier);
        const isOpen = open.has(k);
        const isSel = sel !== null && key(sel) === k;
        const step = depth === 0 ? steps[i] : undefined;
        const before = depth === 0 ? (i === 0 ? sample : steps[i - 1]?.output) : undefined;
        const sum = summary(a, ctx);
        const dropHere = drop && key(drop.parent) === key(parent) && drop.index === i;
        return (
          <div key={k}>
            {dropHere && <div className="drop-line" />}
            <div
              className={"acard" + (isSel ? " sel" : "") + (a.enabled ? "" : " off") + (info?.support !== "full" ? " unsupported" : "")}
              onDragOver={(e) => {
                const r = e.currentTarget.getBoundingClientRect();
                const after = e.clientY > r.top + Math.min(r.height / 2, 22);
                dragOver(e, parent, after ? i + 1 : i);
              }}
              onDrop={(e) => {
                const r = e.currentTarget.getBoundingClientRect();
                onDrop(e, parent, e.clientY > r.top + Math.min(r.height / 2, 22) ? i + 1 : i);
              }}
            >
              <div
                className="acard-head"
                draggable
                onDragStart={(e) => {
                  e.dataTransfer.setData(DRAG_PATH, k);
                  e.dataTransfer.effectAllowed = "move";
                }}
                onDragEnd={() => setDrop(null)}
                onClick={() => {
                  setSel(path);
                  setOpen((o) => {
                    const n = new Set(o);
                    if (n.has(k) && isSel) n.delete(k);
                    else n.add(k);
                    return n;
                  });
                }}
              >
                <GripVertical className="grip" />
                <input
                  type="checkbox"
                  checked={a.enabled}
                  title={a.enabled ? "停用这个动作" : "启用这个动作"}
                  onClick={(e) => e.stopPropagation()}
                  onChange={(e) => setActions((as) => updateAt(as, path, (x) => ({ ...x, enabled: e.target.checked })))}
                />
                <span className="acard-num">{depth === 0 ? i + 1 : "•"}</span>
                <span className="acard-title">{info?.title ?? a.identifier}</span>
                {!isOpen && sum && <span className="acard-sum">{sum}</span>}
                <span className="spacer" />
                {info?.support === "rich" && <span className="chip gray">富文本</span>}
                {info?.support === "unavailable" && <span className="chip gray">不支持</span>}
                {step && a.enabled && (step.error ? (
                  <span className="badge err" title={step.error}>出错</span>
                ) : step.output === before ? (
                  <span className="badge none">无变化</span>
                ) : (
                  <span className="badge ok">{step.count > 0 ? `改动 ${step.count} 处` : "已改变"}</span>
                ))}
                {isOpen ? <ChevronDown className="chev" /> : <ChevronRight className="chev" />}
                <span onClick={(e) => e.stopPropagation()}>
                  <MenuButton
                    className="btn ghost small icon"
                    icon={<MoreHorizontal />}
                    caret={false}
                    align="right"
                    items={[
                      { label: "上移", disabled: i === 0, onSelect: () => setDraft((d) => { const r = moveWithin(d.actions, path, -1); const n = { ...d, actions: r.list }; save(n); setSel(r.path); return n; }) },
                      { label: "下移", disabled: i === actions.length - 1, onSelect: () => setDraft((d) => { const r = moveWithin(d.actions, path, 1); const n = { ...d, actions: r.list }; save(n); setSel(r.path); return n; }) },
                      { label: "复制一份", onSelect: () => setActions((as) => insertAt(as, parent, i + 1, structuredClone(a))) },
                      "sep",
                      { label: "删除", onSelect: () => { setActions((as) => removeAt(as, path)); setSel(null); } },
                    ]}
                  />
                </span>
              </div>
              {isOpen && (
                <div className="acard-body">
                  {info && <div className="hint" style={{ marginBottom: 8 }}>{info.desc}</div>}
                  <ActionForm action={a} info={info} ctx={ctx} onChange={(p) => setActions((as) => updateAt(as, path, (x) => ({ ...x, parameters: p })))} />
                </div>
              )}
              {info?.container && (
                <div className="acard-children">
                  {renderList(a.actionList ?? [], path, depth + 1)}
                  {(a.actionList ?? []).length === 0 && <div className="drop-empty">把动作拖到这里，或选中此项后在右侧点击动作</div>}
                </div>
              )}
            </div>
          </div>
        );
      })}
      {drop && key(drop.parent) === key(parent) && drop.index === actions.length && <div className="drop-line" />}
    </div>
  );

  const startResize = (e: React.MouseEvent) => {
    const y0 = e.clientY;
    const h0 = previewH;
    let h = h0;
    const move = (ev: MouseEvent) => {
      h = Math.max(120, Math.min(window.innerHeight - 260, h0 - (ev.clientY - y0)));
      setPreviewH(h);
    };
    const up = () => {
      window.removeEventListener("mousemove", move);
      window.removeEventListener("mouseup", up);
      localStorage.setItem("previewH", String(h));
    };
    window.addEventListener("mousemove", move);
    window.addEventListener("mouseup", up);
  };

  const t = target();
  const targetLabel = sel ? (t.parent.length === sel.length ? `插入到「${catalog.get(getAt(draft.actions, sel)?.identifier ?? "")?.title}」里面` : `插入到第 ${sel.map((x) => x + 1).join(".")} 步之后`) : "添加到末尾";

  return (
    <>
      <section className="lib-editor">
        <div className="lib-head">
          <input className="name-input" value={draft.name} onChange={(e) => change((c) => ({ ...c, name: e.target.value }))} placeholder="清理器名称" />
          <input className="comment-input" value={draft.comment ?? ""} onChange={(e) => change((c) => ({ ...c, comment: e.target.value }))} placeholder="说明（可选）：这个清理器做什么" />
        </div>
        <div className="lib-actions" onClick={(e) => e.target === e.currentTarget && setSel(null)}>
          {(draft.actions ?? []).length === 0 ? (
            <div
              className="drop-empty big"
              onDragOver={(e) => dragOver(e, [], 0)}
              onDrop={(e) => onDrop(e, [], 0)}
            >
              <Wand2 />
              <div>从右侧动作库点击或拖入动作，按从上到下的顺序执行。</div>
            </div>
          ) : (
            renderList(draft.actions, [], 0)
          )}
        </div>
        <div className="preview" style={{ height: previewH }}>
          <div className="preview-resizer" onMouseDown={startResize} />
          <div className="preview-pane">
            <div className="preview-title">
              <span>样例文本</span>
              <span className="spacer" />
              <button className="btn ghost small" onClick={() => Soap.readClipboard().then(setSample)}>
                <ClipboardPaste /> 从剪贴板载入
              </button>
            </div>
            <textarea className="preview-text mono" value={sample} onChange={(e) => setSample(e.target.value)} placeholder="在这里输入一段样例，右侧会实时显示清理结果" spellCheck={false} />
          </div>
          <div className="preview-pane">
            <div className="preview-title">
              <span>结果</span>
              {result && (shown === sample ? <span className="badge none">无变化</span> : <span className="badge ok">已改变</span>)}
              <span className="spacer" />
              {sel && sel.length === 1 && steps[sel[0]] && (
                <Segmented
                  small
                  value={view}
                  onChange={setView}
                  options={[
                    { value: "final", label: "最终结果" },
                    { value: "step", label: `第 ${sel[0] + 1} 步之后` },
                  ]}
                />
              )}
              <button className="btn ghost small" onClick={() => Soap.writeClipboard(shown).then(() => toast("已拷贝结果"))}>
                <Copy /> 拷贝
              </button>
            </div>
            <textarea className="preview-text mono" readOnly value={shown} spellCheck={false} />
            {result?.warnings?.length ? <div className="preview-warn">{result.warnings.join("；")}</div> : null}
          </div>
        </div>
      </section>
      <Toolbox boot={boot} onAdd={(i) => add(i)} targetLabel={targetLabel} />
    </>
  );
}

// ---------------------------------------------------------------------------

function Toolbox({ boot, onAdd, targetLabel }: { boot: Boot; onAdd: (a: ActionInfo) => void; targetLabel: string }) {
  const [q, setQ] = useState("");
  const [showAll, setShowAll] = useState(false);
  const [closed, setClosed] = useState<Set<string>>(new Set());
  const items = boot.catalog.filter(
    (a) =>
      (showAll || a.support === "full") &&
      !["if.end", "runtime.macrolist", "applycleaner.static", "find.regex"].includes(a.id) &&
      (!q || a.title.includes(q) || a.en.toLowerCase().includes(q.toLowerCase()) || a.desc.includes(q)),
  );
  return (
    <aside className="toolbox">
      <div className="toolbox-head">
        <div className="label" style={{ marginBottom: 6 }}>动作库</div>
        <div className="search">
          <Search />
          <input className="input" placeholder="搜索动作" value={q} onChange={(e) => setQ(e.target.value)} />
        </div>
        <div className="hint" style={{ marginTop: 6 }}>点击添加（{targetLabel}），或拖到左侧任意位置。</div>
      </div>
      <div className="toolbox-list">
        {boot.categories.map((cat) => {
          const list = items.filter((a) => a.category === cat);
          if (!list.length) return null;
          const isClosed = closed.has(cat) && !q;
          return (
            <div key={cat}>
              <button className="toolbox-cat" onClick={() => setClosed((c) => { const n = new Set(c); if (n.has(cat)) n.delete(cat); else n.add(cat); return n; })}>
                {isClosed ? <ChevronRight /> : <ChevronDown />}
                {cat}
                <span className="muted" style={{ fontWeight: 400 }}>{list.length}</span>
              </button>
              {!isClosed &&
                list.map((a) => (
                  <button
                    key={a.id}
                    className={"tool-item" + (a.support !== "full" ? " dim" : "")}
                    draggable
                    onDragStart={(e) => {
                      e.dataTransfer.setData(DRAG_NEW, a.id);
                      e.dataTransfer.effectAllowed = "copy";
                    }}
                    onClick={() => onAdd(a)}
                    title={a.en}
                  >
                    <span className="t">
                      {a.title}
                      {a.container && <span className="chip gray" style={{ marginLeft: 6 }}>容器</span>}
                    </span>
                    <span className="d">{a.desc}</span>
                  </button>
                ))}
            </div>
          );
        })}
      </div>
      <label className="check toolbox-foot">
        <input type="checkbox" checked={showAll} onChange={(e) => setShowAll(e.target.checked)} />
        显示富文本和不支持的动作
      </label>
    </aside>
  );
}

