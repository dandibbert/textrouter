import { forwardRef, useCallback, useEffect, useImperativeHandle, useMemo, useRef, useState } from "react";
import type { EditorView } from "@codemirror/view";
import { AlertCircle, ChevronLeft, ChevronRight, CornerDownRight, History, Lightbulb, Plus, Search, SlidersHorizontal, X, Zap } from "lucide-react";
import { Library, Soap, type Boot, type FindRequest, type FoundMatch, type Part, type SearchEntry } from "../mygo";
import { PatternField, type PatternFieldHandle } from "../lib/PatternField";
import { MenuButton, toast, type MenuEntry } from "../lib/ui";
import { debounce, escapeRegex, escapeText, visibleParts } from "../lib/text";
import { setHits } from "./editor";
import { keys } from "../lib/boot";

export type FindPanelHandle = {
  focus: () => void;
  next: (dir: 1 | -1) => void;
  useSelection: () => void;
  replace: () => void;
  replaceAll: () => void;
  docChanged: () => void;
};

const granularities = ["包含", "开头是", "整个单词", "结尾是"];
const granularityDesc = ["找到的文字可以在任何位置", "只匹配一个单词的开头", "只匹配完整的单词", "只匹配一个单词的结尾"];

function tokenClass(kind: string) {
  return { class: "rx-class", group: "rx-group", quant: "rx-quant", anchor: "rx-anchor", alt: "rx-alt", token: "rx-escape" }[kind] ?? "";
}

function Visible({ text }: { text: string }) {
  return (
    <>
      {visibleParts(text).map((p, i) =>
        p.invisible ? (
          <span key={i} className="invis">
            {p.text}
          </span>
        ) : (
          <span key={i}>{p.text}</span>
        ),
      )}
    </>
  );
}

export const FindPanel = forwardRef<FindPanelHandle, {
  boot: Boot;
  view: () => EditorView | null;
  onActivate: () => void;
}>(function FindPanel({ boot, view, onActivate }, ref) {
  const prefs = boot.prefs;
  const [find, setFind] = useState("");
  const [replace, setReplace] = useState("");
  const [regex, setRegex] = useState(prefs.searchRegex);
  const [matchCase, setMatchCase] = useState(prefs.searchMatchCase);
  const [live, setLive] = useState(prefs.searchLive);
  const [opts, setOpts] = useState(prefs.searchMatchLines ? "m" : "");
  const [gran, setGran] = useState(0);
  const [matches, setMatches] = useState<FoundMatch[]>([]);
  const [truncated, setTruncated] = useState(false);
  const [current, setCurrent] = useState(-1);
  const [error, setError] = useState("");
  const [searched, setSearched] = useState(false);
  const [explainOn, setExplainOn] = useState(false);
  const [parts, setParts] = useState<Part[]>([]);
  const [history, setHistory] = useState<SearchEntry[]>(boot.history);
  const [lastField, setLastField] = useState<"find" | "replace">("find");
  const findRef = useRef<PatternFieldHandle>(null);
  const replaceRef = useRef<PatternFieldHandle>(null);
  const listRef = useRef<HTMLDivElement>(null);
  const seq = useRef(0);

  useEffect(() => setHistory(boot.history), [boot.history]);

  const request = useCallback(
    (text: string): FindRequest => ({ text, find, replace, regex, matchCase, regexOptions: regex ? opts : "", granularity: regex ? 0 : gran, limit: 5000 }),
    [find, replace, regex, matchCase, opts, gran],
  );

  const showHits = (ms: FoundMatch[], cur: number) => {
    view()?.dispatch({ effects: setHits.of({ hits: ms.map((m) => ({ from: m.from, to: m.to })), current: cur }) });
  };

  const select = (ms: FoundMatch[], i: number, scroll = true) => {
    setCurrent(i);
    showHits(ms, i);
    const v = view();
    const m = ms[i];
    if (v && m) {
      v.dispatch({ selection: { anchor: m.from, head: m.to }, scrollIntoView: scroll });
      requestAnimationFrame(() => listRef.current?.querySelector(`[data-i="${i}"]`)?.scrollIntoView({ block: "nearest" }));
    }
  };

  /** Searches the text; then selects the first match at or after `from`
   * (or keeps the current index when `keep`). */
  const run = useCallback(
    async (then?: { from?: number; keep?: number; quiet?: boolean }) => {
      const v = view();
      if (!v) return [];
      const id = ++seq.current;
      if (!find) {
        setMatches([]);
        setCurrent(-1);
        setError("");
        setSearched(false);
        showHits([], -1);
        return [];
      }
      const r = await Soap.find(request(v.state.doc.toString()));
      if (id !== seq.current) return [];
      setError(r.error ?? "");
      setSearched(true);
      const ms = r.matches ?? [];
      setMatches(ms);
      setTruncated(r.truncated);
      let cur = -1;
      if (then?.keep !== undefined) cur = ms.length ? Math.min(then.keep, ms.length - 1) : -1;
      else if (then?.from !== undefined) {
        cur = ms.findIndex((m) => m.from >= then.from!);
        if (cur < 0 && ms.length) cur = 0;
      }
      if (cur >= 0 && then?.from !== undefined) select(ms, cur);
      else {
        setCurrent(cur);
        showHits(ms, cur);
      }
      return ms;
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [find, request, view],
  );

  const remember = () => {
    if (!find) return;
    Library.rememberSearch({ find, replace, regex, matchCase, regexOptions: opts, granularity: gran }).then(setHistory);
  };

  const doFind = () => {
    remember();
    const v = view();
    run({ from: v ? v.state.selection.main.from : 0 });
  };

  // Live search.
  const liveRun = useMemo(() => debounce((fn: () => void) => fn(), 140), []);
  useEffect(() => {
    if (live) liveRun(() => run({ keep: 0 }));
    else if (!find) run();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [find, replace, regex, matchCase, opts, gran, live]);

  // Explanation.
  useEffect(() => {
    if (!explainOn || !find) return setParts([]);
    const t = setTimeout(() => Soap.explain(find, regex, false).then(setParts), 150);
    return () => clearTimeout(t);
  }, [explainOn, find, regex]);

  const docChangedRun = useMemo(() => debounce((fn: () => void) => fn(), 260), []);

  const next = async (dir: 1 | -1) => {
    let ms = matches;
    const v = view();
    if (!ms.length || !searched) {
      ms = await run({ from: v ? v.state.selection.main.from : 0 });
      if (dir === 1 || !ms.length) return;
    }
    if (!ms.length) return;
    let i: number;
    if (current < 0 && v) {
      const pos = v.state.selection.main.head;
      i = dir === 1 ? ms.findIndex((m) => m.from >= pos) : ms.map((m) => m.to <= pos).lastIndexOf(true);
      if (i < 0) i = dir === 1 ? 0 : ms.length - 1;
    } else i = (current + dir + ms.length) % ms.length;
    select(ms, i);
  };

  const replaceOne = async () => {
    const v = view();
    if (!v) return;
    if (current < 0 || !matches[current]) return next(1);
    const m = matches[current];
    if (v.state.sliceDoc(m.from, m.to) !== m.text) {
      await run({ from: m.from });
      return;
    }
    remember();
    v.dispatch({ changes: { from: m.from, to: m.to, insert: m.result }, userEvent: "input.replace" });
    await run({ from: m.from + m.result.length });
  };

  const replaceAll = async () => {
    const v = view();
    if (!v || !find) return;
    remember();
    const r = await Soap.replaceAll(request(v.state.doc.toString()));
    if (r.error) return setError(r.error);
    if (!r.changes.length) return toast("没有找到匹配", "info");
    v.dispatch({ changes: r.changes.map((c) => ({ from: c.from, to: c.to, insert: c.insert })), userEvent: "input.replace", scrollIntoView: true });
    toast(`已替换 ${r.changes.length} 处`, "ok", `${keys("CmdOrCtrl+Z")} 撤销`);
    run();
  };

  const clear = () => {
    seq.current++;
    setMatches([]);
    setCurrent(-1);
    setSearched(false);
    setError("");
    showHits([], -1);
  };

  useImperativeHandle(ref, () => ({
    focus: () => {
      onActivate();
      setTimeout(() => findRef.current?.selectAll(), 0);
    },
    next,
    useSelection: () => {
      const v = view();
      if (!v) return;
      const { from, to } = v.state.selection.main;
      if (from === to) return;
      const s = v.state.sliceDoc(from, to);
      setFind(regex ? escapeRegex(s) : escapeText(s));
      onActivate();
    },
    replace: replaceOne,
    replaceAll,
    docChanged: () => {
      if (find && (live || searched)) docChangedRun(() => run({ keep: current < 0 ? undefined : current }));
    },
  }));

  // ---------------------------------------------------------------- menus

  const optionsMenu = (): MenuEntry[] => {
    const out: MenuEntry[] = [];
    if (regex) {
      const flag = (f: string) => opts.includes(f);
      const toggle = (f: string) => setOpts(flag(f) ? opts.replace(f, "") : opts + f);
      out.push(
        { head: "正则表达式选项" },
        { label: "^ 和 $ 匹配每一行的开头和结尾", code: "m", checked: flag("m"), onSelect: () => toggle("m") },
        { label: "点号 . 也匹配换行", code: "s", checked: flag("s"), onSelect: () => toggle("s") },
        { label: "忽略空格，允许 # 注释", code: "x", checked: flag("x"), onSelect: () => toggle("x") },
      );
    } else {
      out.push({ head: "匹配方式" });
      granularities.forEach((g, i) => out.push({ label: g, desc: granularityDesc[i], checked: gran === i, onSelect: () => setGran(i) }));
    }
    out.push(
      "sep",
      {
        label: "将当前选项存为默认",
        onSelect: () =>
          Library.setPrefs({ ...prefs, searchRegex: regex, searchMatchCase: matchCase, searchLive: live, searchMatchLines: opts.includes("m") }).then(() =>
            toast("已存为查找默认选项"),
          ),
      },
      {
        label: "恢复默认选项",
        onSelect: () => {
          setRegex(prefs.searchRegex);
          setMatchCase(prefs.searchMatchCase);
          setLive(prefs.searchLive);
          setOpts(prefs.searchMatchLines ? "m" : "");
          setGran(0);
        },
      },
    );
    return out;
  };

  const ins = (code: string, label: string, caret?: number, desc?: string): MenuEntry => ({
    code,
    label,
    desc,
    onSelect: () => (lastField === "find" ? findRef : replaceRef).current?.insert(code.replace(" ", ""), caret),
  });

  const insertMenu = (): MenuEntry[] => {
    if (lastField === "replace") {
      if (!regex)
        return [{ head: "插入到「替换为」" }, ins("\\t", "制表符"), ins("\\n", "换行"), ins("\\\\", "反斜杠 \\ 本身"), ins("\\x{hhhh}", "Unicode 字符", 3)];
      return [
        { head: "引用找到的内容" },
        ins("$0", "整个匹配"),
        ins("$1", "第 1 个分组"),
        ins("$2", "第 2 个分组"),
        ins("$3", "第 3 个分组"),
        { head: "改变大小写" },
        ins("$u", "下一个字母大写"),
        ins("$l", "下一个字母小写"),
        ins("$U", "之后全部大写"),
        ins("$L", "之后全部小写"),
        ins("$E", "结束大小写转换"),
        { head: "特殊字符" },
        ins("\\t", "制表符"),
        ins("\\n", "换行"),
        ins("\\$", "美元符号 $ 本身"),
        ins("\\\\", "反斜杠 \\ 本身"),
      ];
    }
    if (!regex)
      return [{ head: "插入到「查找」" }, ins("\\t", "制表符"), ins("\\n", "换行"), ins("\\\\", "反斜杠 \\ 本身"), ins("\\x{hhhh}", "Unicode 字符", 3)];
    return [
      { head: "字符" },
      ins(".", "任意一个字符（换行除外）"),
      ins("\\d", "一个数字"),
      ins("\\w", "一个字母、数字或下划线"),
      ins("\\s", "一个空白（空格、制表符、换行）"),
      ins("\\p{Han}", "一个汉字"),
      ins("[ ]", "其中任意一个字符", 1),
      ins("[^ ]", "不是其中任何一个字符", 2),
      { head: "位置" },
      ins("^", "行首"),
      ins("$", "行尾"),
      ins("\\b", "单词边界"),
      { head: "次数（加在字符或分组之后）" },
      ins("*", "0 次或多次"),
      ins("+", "1 次或多次"),
      ins("?", "0 次或 1 次（可有可无）"),
      ins("{2,5}", "2 到 5 次", 1),
      ins("*?", "尽量少地重复"),
      { head: "分组" },
      ins("( )", "分组，可在替换中用 $1 引用", 1),
      ins("(?: )", "只分组，不编号", 3),
      ins("|", "或者"),
      ins("(?= )", "后面紧跟着…（不计入匹配）", 3),
      ins("(?<= )", "前面紧挨着…（不计入匹配）", 4),
      { head: "特殊字符" },
      ins("\\t", "制表符"),
      ins("\\n", "换行"),
      ins("\\x{hhhh}", "Unicode 字符", 3),
      "sep",
      { head: "常用表达式（替换查找内容）" },
      ...boot.presets.map<MenuEntry>((p) => ({
        label: p.name,
        desc: p.desc,
        onSelect: () => {
          setFind(p.pattern);
          if (p.replace) setReplace(p.replace);
        },
      })),
    ];
  };

  const historyMenu = (): MenuEntry[] => {
    if (!history.length) return [{ label: "还没有查找记录", disabled: true, onSelect: () => {} }];
    return [
      { head: "最近的查找" },
      ...history.slice(0, 20).map<MenuEntry>((h) => ({
        label: (
          <span className="mono" style={{ fontSize: 12 }}>
            {h.find.length > 36 ? h.find.slice(0, 36) + "…" : h.find}
            {h.replace && <span className="muted"> → {h.replace.length > 20 ? h.replace.slice(0, 20) + "…" : h.replace}</span>}
          </span>
        ),
        desc: h.regex ? "正则" : undefined,
        onSelect: () => {
          setFind(h.find);
          setReplace(h.replace);
          setRegex(h.regex);
          setMatchCase(h.matchCase);
          setOpts(h.regexOptions ?? "");
          setGran(h.granularity ?? 0);
        },
      })),
      "sep",
      { label: "清除查找记录", onSelect: () => Library.clearHistory().then(() => setHistory([])) },
    ];
  };

  const m = current >= 0 ? matches[current] : undefined;
  const shown = matches.slice(0, 1000);

  return (
    <div className="find">
      <div className="find-form">
        <div className="find-toggles">
          <button className={"pill" + (regex ? " on" : "")} onClick={() => setRegex(!regex)} title="把查找内容当作正则表达式">
            <span className="pill-code">.*</span>正则
          </button>
          <button className={"pill" + (matchCase ? " on" : "")} onClick={() => setMatchCase(!matchCase)} title="区分大小写">
            <span className="pill-code">Aa</span>大小写
          </button>
          <button className={"pill" + (live ? " on" : "")} onClick={() => setLive(!live)} title="输入时即时查找">
            <Zap />
            实时
          </button>
          <span className="spacer" />
          <MenuButton className="btn ghost small" icon={<SlidersHorizontal />} label="选项" items={optionsMenu} align="right" />
        </div>

        <div className="field">
          <div className="field-title">
            <span className="row">
              查找
              {!regex && gran > 0 && <span className="chip gray">{granularities[gran]}</span>}
            </span>
            <span className="field-actions">
              {regex && (
                <button className={"link" + (explainOn ? " on" : "")} onClick={() => setExplainOn(!explainOn)} title="逐段解释这个正则表达式">
                  <Lightbulb />
                  解释
                </button>
              )}
              <MenuButton className="link" icon={<Plus />} label="插入" items={insertMenu} align="right" caret={false} title="插入常用片段到光标处" />
              <MenuButton className="link" icon={<History />} label="历史" items={historyMenu} align="right" caret={false} title="最近的查找" />
            </span>
          </div>
          <PatternField
            ref={findRef}
            value={find}
            onChange={setFind}
            mode={regex ? "regex" : "text"}
            placeholder={regex ? "正则表达式，例如 (\\d+)-(\\d+)" : "要查找的文字"}
            onEnter={doFind}
            onFocus={() => setLastField("find")}
            error={!!error}
            ariaLabel="查找"
          />
          {error && (
            <div className="find-error">
              <AlertCircle />
              <span>表达式有误：{error}</span>
            </div>
          )}
          {explainOn && regex && parts.length > 0 && (
            <div className="explain">
              {parts.map((p, i) => (
                <div key={i} className={"explain-part k-" + p.kind}>
                  <code className={tokenClass(p.kind)}>{p.text}</code>
                  <span>{p.desc}</span>
                </div>
              ))}
            </div>
          )}
        </div>
        <div className="field">
          <div className="field-title">
            <span>替换为</span>
            {regex && <span className="field-hint">$1 = 第 1 个分组</span>}
          </div>
          <PatternField
            ref={replaceRef}
            value={replace}
            onChange={setReplace}
            mode={regex ? "template" : "template-text"}
            placeholder="留空则删除找到的内容"
            onEnter={doFind}
            onFocus={() => setLastField("replace")}
            ariaLabel="替换为"
          />
        </div>
        <div className="find-buttons">
          <button className="btn primary" onClick={doFind} disabled={!find} title="查找全部匹配 ⏎">
            <Search />
            查找
          </button>
          <button className="btn" onClick={replaceOne} disabled={!find} title={`替换选中的匹配，然后选中下一个 ${keys("CmdOrCtrl+Alt+G")}`}>
            替换
          </button>
          <button className="btn" onClick={replaceAll} disabled={!find} title={`替换全部匹配（可撤销）${keys("CmdOrCtrl+Alt+Shift+G")}`}>
            全部替换
          </button>
        </div>
      </div>

      <div className="matches-head">
        <span>匹配</span>
        {searched && !error && (
          <span className={"count" + (matches.length ? "" : " none")}>
            {matches.length === 0 ? "无" : current >= 0 ? `${current + 1} / ${matches.length}${truncated ? "+" : ""}` : `${matches.length}${truncated ? "+" : ""}`}
          </span>
        )}
        <span className="spacer" />
        <button className="btn ghost small icon" onClick={() => next(-1)} disabled={!matches.length} title={`上一个 ${keys("CmdOrCtrl+Shift+G")}`}>
          <ChevronLeft />
        </button>
        <button className="btn ghost small icon" onClick={() => next(1)} disabled={!matches.length} title={`下一个 ${keys("CmdOrCtrl+G")}`}>
          <ChevronRight />
        </button>
        <button className="btn ghost small icon" onClick={clear} disabled={!searched} title="清除查找结果和高亮">
          <X />
        </button>
      </div>
      <div className="matches" ref={listRef}>
        {!searched && (
          <div className="empty">
            输入要查找的内容，按 <kbd>⏎</kbd> 查找。
            <br />
            每个匹配会列在这里，并显示替换后的样子。
          </div>
        )}
        {searched && !error && matches.length === 0 && <div className="empty">没有找到匹配</div>}
        {shown.map((mm, i) => (
          <button key={i} data-i={i} className={"match" + (i === current ? " on" : "")} onClick={() => select(matches, i)}>
            <span className="idx">{i + 1}</span>
            <span className="pair">
              <span className="from">{mm.text === "" ? <span className="invis">（空）</span> : <Visible text={mm.text} />}</span>
              {replace !== "" && (
                <span className="to-row">
                  <CornerDownRight />
                  {mm.result === "" ? <span className="to deleted">删除</span> : <span className="to"><Visible text={mm.result} /></span>}
                </span>
              )}
            </span>
          </button>
        ))}
        {matches.length > shown.length && <div className="empty">只列出前 {shown.length} 个</div>}
      </div>
      {m && m.groups.length > 1 && (
        <div className="groups">
          <div className="groups-title">捕获分组 · 第 {current + 1} 个匹配</div>
          {m.groups.slice(1).map((g) => (
            <div key={g.index} className="group-row">
              <span className="g">
                ${g.index}
                {g.name && <span className="gname">{g.name}</span>}
              </span>
              <span className="v">{g.ok ? g.text === "" ? <span className="invis">（空）</span> : <Visible text={g.text} /> : <span className="invis">未参与匹配</span>}</span>
            </div>
          ))}
        </div>
      )}
    </div>
  );
});
