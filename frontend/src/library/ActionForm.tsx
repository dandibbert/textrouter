import type { ReactNode } from "react";
import { Info } from "lucide-react";
import type { Action, ActionInfo, CleanerInfo } from "../mygo";
import { PatternField } from "../lib/PatternField";
import { Segmented } from "../lib/ui";
import { ListTable, rowsOf } from "./ListTable";
import { bool, num, str } from "./tree";

export type FormContext = {
  lists: string[];
  macros: string[];
  cleaners: CleanerInfo[];
  allowScripts: boolean;
  self: string;
};

type P = Record<string, unknown>;

function Field({ label, children, hint, wide }: { label: string; children: ReactNode; hint?: ReactNode; wide?: boolean }) {
  return (
    <label className={"af-field" + (wide ? " wide" : "")} onClick={(e) => e.target === e.currentTarget && e.preventDefault()}>
      <span className="af-label">{label}</span>
      <span className="af-control">{children}</span>
      {hint && <span className="hint">{hint}</span>}
    </label>
  );
}

function Check({ label, value, onChange, title }: { label: string; value: boolean; onChange: (v: boolean) => void; title?: string }) {
  return (
    <label className="check" title={title}>
      <input type="checkbox" checked={value} onChange={(e) => onChange(e.target.checked)} />
      {label}
    </label>
  );
}

function NumberInput({ value, onChange, min, max, width = 72 }: { value: number; onChange: (v: number) => void; min?: number; max?: number; width?: number }) {
  return <input className="input" type="number" style={{ width }} value={value} min={min} max={max} onChange={(e) => onChange(parseInt(e.target.value || "0", 10))} />;
}

function Select({ value, onChange, options }: { value: string; onChange: (v: string) => void; options: { value: string; label: string }[] }) {
  return (
    <select className="select" value={value} onChange={(e) => onChange(e.target.value)}>
      {options.map((o) => (
        <option key={o.value} value={o.value}>
          {o.label}
        </option>
      ))}
    </select>
  );
}

const granularities = ["包含", "开头是", "整个单词", "结尾是"];

function RegexOptions({ p, set }: { p: P; set: (k: string, v: unknown) => void }) {
  const o = str(p.regex_options);
  const flag = (f: string, on: boolean) => set("regex_options", on ? (o.includes(f) ? o : o + f) : o.replace(f, ""));
  return (
    <div className="row" style={{ gap: 14, flexWrap: "wrap" }}>
      <Check label="^ $ 匹配每行 (m)" value={o.includes("m")} onChange={(v) => flag("m", v)} />
      <Check label=". 匹配换行 (s)" value={o.includes("s")} onChange={(v) => flag("s", v)} />
      <Check label="忽略空白与注释 (x)" value={o.includes("x")} onChange={(v) => flag("x", v)} />
    </div>
  );
}

function ResultModifiers({ p, set }: { p: P; set: (k: string, v: unknown) => void }) {
  return (
    <div className="row" style={{ gap: 14, flexWrap: "wrap" }}>
      <Check label="反选（对不匹配的部分执行）" value={bool(p.invert)} onChange={(v) => set("invert", v)} />
      <span className="row">
        跳过前 <NumberInput value={num(p.skip)} min={0} onChange={(v) => set("skip", v)} width={60} /> 个
      </span>
      <span className="row">
        最多 <NumberInput value={num(p.limit)} min={0} onChange={(v) => set("limit", v)} width={60} /> 个
        <span className="muted">（0 = 不限）</span>
      </span>
    </div>
  );
}

function CleanerSelect({ value, onChange, ctx }: { value: string; onChange: (v: string) => void; ctx: FormContext }) {
  const custom = ctx.cleaners.filter((c) => c.custom && c.name !== ctx.self);
  const builtin = ctx.cleaners.filter((c) => !c.custom);
  const cats = [...new Set(builtin.map((c) => c.category))];
  return (
    <select className="select" value={value} onChange={(e) => onChange(e.target.value)}>
      {!ctx.cleaners.some((c) => c.name === value) && <option value={value}>{value || "选择清理器"}（找不到）</option>}
      {custom.length > 0 && (
        <optgroup label="我的清理器">
          {custom.map((c) => (
            <option key={c.name} value={c.name}>
              {c.name}
            </option>
          ))}
        </optgroup>
      )}
      {cats.map((cat) => (
        <optgroup key={cat} label={cat}>
          {builtin
            .filter((c) => c.category === cat)
            .map((c) => (
              <option key={c.name} value={c.name}>
                {c.zh}
              </option>
            ))}
        </optgroup>
      ))}
    </select>
  );
}

function ListSelect({ value, onChange, ctx, allowNone }: { value: string; onChange: (v: string) => void; ctx: FormContext; allowNone?: boolean }) {
  const opts = [...(allowNone ? [{ value: "", label: "不使用列表" }] : [{ value: "", label: "选择列表" }]), ...ctx.lists.map((l) => ({ value: l, label: `列表「${l}」` }))];
  if (value && !ctx.lists.includes(value)) opts.push({ value, label: `「${value}」（未定义）` });
  return <Select value={value} onChange={onChange} options={opts} />;
}

/** The find part shared by find/replace, conditions and Extract Text. */
function FindFields({ a, p, set, ctx }: { a: Action; p: P; set: (k: string, v: unknown) => void; ctx: FormContext }) {
  const id = a.identifier;
  const fixedRegex = id.endsWith(".regex") || id === "text.extract";
  const regex = fixedRegex || num(p.type, id === "find" ? 1 : 0) === 1;
  const hasReplace = id.startsWith("find") && id !== "find.delete";
  const hasList = id === "find" || id === "find.batch" || id === "find.batch.regex" || id === "find.delete";
  const isCondition = id.startsWith("if.");
  const listName = str(p.list_name);
  return (
    <>
      {!fixedRegex && (
        <Field label="查找方式">
          <Segmented
            small
            value={regex ? "1" : "0"}
            onChange={(v) => set("type", Number(v))}
            options={[
              { value: "0", label: "文字" },
              { value: "1", label: "正则表达式" },
            ]}
          />
        </Field>
      )}
      <Field label={regex ? "正则表达式" : "查找"} wide>
        <PatternField value={str(p.find)} onChange={(v) => set("find", v)} mode={regex ? "regex" : "text"} placeholder={regex ? "例如 (\\d+)-(\\d+)" : "要查找的文字"} />
      </Field>
      {hasReplace && (
        <Field label="替换为" wide hint={regex ? "$0 整个匹配，$1–$9 分组，$u/$l 下一个字母大/小写，\\n 换行" : "\\t 制表符，\\n 换行；留空即删除"}>
          <PatternField value={str(p.replace)} onChange={(v) => set("replace", v)} mode={regex ? "template" : "template-text"} placeholder="留空则删除" />
        </Field>
      )}
      <Field label="选项" wide>
        <div className="row" style={{ gap: 14, flexWrap: "wrap" }}>
          <Check label="区分大小写" value={bool(p.match_case, true)} onChange={(v) => set("match_case", v)} />
          {!regex && (
            <span className="row">
              匹配方式
              <Select value={String(num(p.granularity))} onChange={(v) => set("granularity", Number(v))} options={granularities.map((g, i) => ({ value: String(i), label: g }))} />
            </span>
          )}
        </div>
        {regex && <RegexOptions p={p} set={set} />}
      </Field>
      {hasList && (
        <Field
          label="按列表"
          hint={
            listName ? (
              <>
                对列表「{listName}」的每一行执行一次。在查找和替换中用 <code>$v{"{A}"}</code>、<code>$v{"{B}"}</code>… 表示这一行的 A、B… 列。
              </>
            ) : (
              "选一个本清理器里“定义列表”动作定义的列表，即可批量替换。"
            )
          }
        >
          <ListSelect value={listName} onChange={(v) => set("list_name", v)} ctx={ctx} allowNone={id === "find" || id === "find.delete"} />
        </Field>
      )}
      {isCondition && regex && (
        <Field label="只取分组" hint="0 = 整个匹配；1–9 = 只对该捕获分组执行里面的动作">
          <NumberInput value={num(p.group)} min={0} max={9} onChange={(v) => set("group", v)} />
        </Field>
      )}
      {isCondition && (
        <Field label="结果" wide>
          <ResultModifiers p={p} set={set} />
        </Field>
      )}
      {id === "text.extract" && (
        <>
          <Field label="只取分组" hint="0 = 整个匹配">
            <NumberInput value={num(p.group)} min={0} max={9} onChange={(v) => set("group", v)} />
          </Field>
          <Field label="输出">
            <div className="row" style={{ gap: 14 }}>
              <Check label="追加在原文之后（不替换原文）" value={bool(p.append)} onChange={(v) => set("append", v)} />
              <span className="row">
                分隔
                <input className="input mono" style={{ width: 70 }} value={str(p.separator, "\\n")} onChange={(e) => set("separator", e.target.value)} />
              </span>
            </div>
          </Field>
        </>
      )}
    </>
  );
}

export function ActionForm({ action, info, onChange, ctx }: { action: Action; info?: ActionInfo; onChange: (p: P) => void; ctx: FormContext }) {
  const p: P = action.parameters ?? {};
  const set = (k: string, v: unknown) => onChange({ ...p, [k]: v });
  const id = action.identifier;

  if (info && info.support !== "full") {
    return (
      <div className="af-note">
        <Info />
        {info.support === "rich"
          ? "这个动作只作用于富文本（字体、颜色、链接）。TextRouter 处理纯文本，运行时会跳过它。"
          : "这个动作依赖 macOS 服务或 TextSoap 的测试功能，TextRouter 暂不支持，运行时会跳过它。"}
      </div>
    );
  }

  switch (id) {
    case "find":
    case "find.regex":
    case "find.delete":
    case "find.batch":
    case "find.batch.regex":
    case "if.match.text":
    case "if.match.text.regex":
    case "text.extract":
      return (
        <div className="af">
          <FindFields a={action} p={p} set={set} ctx={ctx} />
        </div>
      );
    case "applycleaner":
    case "applycleaner.static":
      return (
        <div className="af">
          <Field label="清理器" wide>
            <CleanerSelect value={str(p.cleaner)} onChange={(v) => set("cleaner", v)} ctx={ctx} />
          </Field>
        </div>
      );
    case "text.insert":
      return (
        <div className="af">
          <Field label="位置">
            <Segmented small value={String(num(p.position))} onChange={(v) => set("position", Number(v))} options={[{ value: "0", label: "插在前面" }, { value: "1", label: "加在后面" }, { value: "2", label: "替换全部" }]} />
          </Field>
          <Field label="文字" wide hint="\t 制表符，\n 换行">
            <PatternField value={str(p.text)} onChange={(v) => set("text", v)} mode="template-text" />
          </Field>
        </div>
      );
    case "text.tag":
      return (
        <div className="af">
          <Field label="前缀" wide>
            <PatternField value={str(p.prefix)} onChange={(v) => set("prefix", v)} mode="template-text" placeholder="例如 <b>" />
          </Field>
          <Field label="后缀" wide>
            <PatternField value={str(p.suffix)} onChange={(v) => set("suffix", v)} mode="template-text" placeholder="例如 </b>" />
          </Field>
        </div>
      );
    case "text.substring": {
      const mode = num(p.mode);
      return (
        <div className="af">
          <Field label="截取">
            <Segmented small value={String(mode)} onChange={(v) => set("mode", Number(v))} options={[{ value: "0", label: "开头" }, { value: "1", label: "结尾" }, { value: "2", label: "指定位置" }]} />
          </Field>
          {mode < 2 ? (
            <Field label="字符数">
              <NumberInput value={num(p.count, 10)} min={0} onChange={(v) => set("count", v)} />
            </Field>
          ) : (
            <Field label="范围">
              <span className="row">
                从第 <NumberInput value={num(p.start, 1)} min={1} onChange={(v) => set("start", v)} /> 个字符起，取
                <NumberInput value={num(p.length, 10)} min={0} onChange={(v) => set("length", v)} /> 个
              </span>
            </Field>
          )}
        </div>
      );
    }
    case "text.wrap":
      return (
        <div className="af">
          <Field label="最大行宽">
            <span className="row">
              <NumberInput value={num(p.width, 70)} min={10} onChange={(v) => set("width", v)} /> 个字符
            </span>
          </Field>
        </div>
      );
    case "text.quote":
      return (
        <div className="af">
          <Field label="引用层级">
            <NumberInput value={num(p.level, 1)} min={1} max={9} onChange={(v) => set("level", v)} />
          </Field>
          <Field label="引用符">
            <span className="row">
              前缀 <input className="input mono" style={{ width: 60 }} value={str(p.prefix)} onChange={(e) => set("prefix", e.target.value)} />
              符号 <input className="input mono" style={{ width: 50 }} value={str(p.mark, ">")} onChange={(e) => set("mark", e.target.value)} />
              后缀 <input className="input mono" style={{ width: 50 }} value={str(p.suffix, " ")} onChange={(e) => set("suffix", e.target.value)} />
            </span>
          </Field>
          <Field label="折行">
            <span className="row">
              <Check label="同时折行，行宽" value={bool(p.wrap, true)} onChange={(v) => set("wrap", v)} />
              <NumberInput value={num(p.width, 70)} min={10} onChange={(v) => set("width", v)} />
            </span>
          </Field>
        </div>
      );
    case "text.titlecase":
      return (
        <div className="af">
          <Field label="额外的小词" wide hint="保持小写，逗号或空格分隔，例如 via, per">
            <input className="input" value={str(p.small_words)} onChange={(e) => set("small_words", e.target.value)} />
          </Field>
          <Field label="特殊词" wide hint="保持原样，例如 iPhone, macOS">
            <input className="input" value={str(p.special_words)} onChange={(e) => set("special_words", e.target.value)} />
          </Field>
        </div>
      );
    case "text.appendtoclip":
      return (
        <div className="af">
          <Field label="分隔" hint="加在剪贴板原内容与新文本之间，\n 为换行">
            <input className="input mono" style={{ width: 120 }} value={str(p.separator)} onChange={(e) => set("separator", e.target.value)} />
          </Field>
        </div>
      );
    case "line.addto":
    case "line.removefrom":
      return (
        <div className="af">
          <Field label="位置">
            <Segmented small value={String(num(p.position))} onChange={(v) => set("position", Number(v))} options={[{ value: "0", label: "行首" }, { value: "1", label: "行尾" }]} />
          </Field>
          <Field label="文字" wide hint="\t 制表符">
            <PatternField value={str(p.text)} onChange={(v) => set("text", v)} mode="template-text" />
          </Field>
        </div>
      );
    case "line.sort":
      return (
        <div className="af">
          <Field label="顺序">
            <Segmented small value={bool(p.descending) ? "1" : "0"} onChange={(v) => set("descending", v === "1")} options={[{ value: "0", label: "升序 A→Z" }, { value: "1", label: "降序 Z→A" }]} />
          </Field>
          <Field label="比较" wide>
            <div className="row" style={{ gap: 14, flexWrap: "wrap" }}>
              <Check label="区分大小写" value={bool(p.match_case)} onChange={(v) => set("match_case", v)} />
              <Check label="数字按数值" value={bool(p.numeric)} onChange={(v) => set("numeric", v)} />
              <Check label="忽略行首空白" value={bool(p.ignore_whitespace)} onChange={(v) => set("ignore_whitespace", v)} />
              <Check label="忽略变音符号" value={bool(p.ignore_diacritics)} onChange={(v) => set("ignore_diacritics", v)} />
            </div>
          </Field>
          <Field label="排序依据" wide>
            <Check label="按正则提取的内容排序" value={bool(p.use_expression)} onChange={(v) => set("use_expression", v)} />
          </Field>
          {bool(p.use_expression) && (
            <>
              <Field label="正则表达式" wide>
                <PatternField value={str(p.find)} onChange={(v) => set("find", v)} mode="regex" placeholder="例如 \\d+" />
              </Field>
              <Field label="排序键" wide hint="用 $1 等组合出比较用的内容">
                <PatternField value={str(p.replace, "$0")} onChange={(v) => set("replace", v)} mode="template" />
              </Field>
            </>
          )}
        </div>
      );
    case "line.number":
      return (
        <div className="af">
          <Field label="编号">
            <Segmented small value={String(num(p.lines))} onChange={(v) => set("lines", Number(v))} options={[{ value: "0", label: "所有行" }, { value: "1", label: "跳过空行" }]} />
          </Field>
          <Field label="起始 / 步长">
            <span className="row">
              <NumberInput value={num(p.start, 1)} onChange={(v) => set("start", v)} />
              <NumberInput value={num(p.increment, 1)} onChange={(v) => set("increment", v)} />
            </span>
          </Field>
          <Field label="格式" hint="连续的 0 代表编号和最少位数（000 → 001），\s 空格，\t 制表符">
            <input className="input mono" style={{ width: 140 }} value={str(p.format, "0.\\s")} onChange={(e) => set("format", e.target.value)} />
          </Field>
        </div>
      );
    case "if.match.url":
    case "if.match.date":
      return (
        <div className="af">
          <Field label="结果" wide>
            <ResultModifiers p={p} set={set} />
          </Field>
        </div>
      );
    case "modify.results.limit":
    case "modify.results.skip":
      return (
        <div className="af">
          <Field label="个数">
            <NumberInput value={num(p.count, 1)} min={0} onChange={(v) => set("count", v)} />
          </Field>
        </div>
      );
    case "runtime.group":
      return (
        <div className="af">
          <Field label="名称" wide>
            <input className="input" value={str(p.name)} placeholder="可选" onChange={(e) => set("name", e.target.value)} />
          </Field>
        </div>
      );
    case "runtime.repeat":
      return (
        <div className="af">
          <Field label="次数">
            <span className="row">
              <NumberInput value={num(p.count, 2)} min={1} onChange={(v) => set("count", v)} />
              <Check label="或一直重复到文本不再变化" value={bool(p.until_unchanged)} onChange={(v) => set("until_unchanged", v)} />
            </span>
          </Field>
        </div>
      );
    case "runtime.definelist": {
      const rows = rowsOf(p);
      return (
        <div className="af">
          <Field label="列表名称" wide hint="在“查找替换”的“按列表”或“处理列表”中选择它">
            <input className="input" value={str(p.name)} onChange={(e) => set("name", e.target.value)} />
          </Field>
          <div className="af-field wide">
            <ListTable
              rows={rows}
              columns={num(p.columns, Math.max(1, ...rows.map((r) => r.length)))}
              onChange={(r, c) => {
                const next: P = { ...p, rows: r, columns: c };
                delete next.text;
                onChange(next);
              }}
            />
          </div>
        </div>
      );
    }
    case "runtime.processlist":
      return (
        <div className="af">
          <Field label="列表" hint="对列表的每一行执行里面的动作，动作里的 $v{A}… 换成这一行的列">
            <ListSelect value={str(p.list_name)} onChange={(v) => set("list_name", v)} ctx={ctx} />
          </Field>
        </div>
      );
    case "runtime.macro":
      return (
        <div className="af">
          <Field label="宏名称" wide>
            <input className="input" value={str(p.name)} onChange={(e) => set("name", e.target.value)} />
          </Field>
        </div>
      );
    case "runtime.applymacro":
      return (
        <div className="af">
          <Field label="宏">
            <Select value={str(p.name)} onChange={(v) => set("name", v)} options={[{ value: "", label: "选择宏" }, ...ctx.macros.map((m) => ({ value: m, label: m }))]} />
          </Field>
        </div>
      );
    case "runtime.usernote":
      return (
        <div className="af">
          <Field label="备注" wide>
            <textarea className="textarea" rows={2} value={str(p.note)} onChange={(e) => set("note", e.target.value)} />
          </Field>
        </div>
      );
    case "runtime.shellscript":
      return (
        <div className="af">
          <Field label="脚本" wide hint={ctx.allowScripts ? "文本从标准输入传入，脚本的标准输出成为新文本（/bin/sh，15 秒超时）" : "需要先在 偏好设置 → 高级 中允许运行 Shell 脚本"}>
            <textarea className="textarea mono" rows={3} value={str(p.script)} onChange={(e) => set("script", e.target.value)} />
          </Field>
        </div>
      );
  }
  const keys = Object.keys(p);
  if (!keys.length) return null;
  return (
    <div className="af">
      {keys.map((k) => (
        <Field key={k} label={k} wide>
          <input className="input mono" value={str(p[k])} onChange={(e) => set(k, e.target.value)} />
        </Field>
      ))}
    </div>
  );
}

/** One-line summary shown on a collapsed action card. */
export function summary(a: Action, ctx: FormContext): string {
  const p: P = a.parameters ?? {};
  const q = (s: unknown) => `「${str(s).length > 30 ? str(s).slice(0, 30) + "…" : str(s)}」`;
  switch (a.identifier) {
    case "find":
    case "find.regex":
    case "find.batch":
    case "find.batch.regex":
      return `${q(p.find)} → ${str(p.replace) === "" ? "删除" : q(p.replace)}${str(p.list_name) ? ` · 列表 ${str(p.list_name)}` : ""}`;
    case "find.delete":
      return `删除 ${q(p.find)}`;
    case "if.match.text":
    case "if.match.text.regex":
      return `匹配 ${q(p.find)}${bool(p.invert) ? "（反选）" : ""}`;
    case "applycleaner":
    case "applycleaner.static": {
      const c = ctx.cleaners.find((x) => x.name === str(p.cleaner));
      return c ? c.zh : str(p.cleaner);
    }
    case "runtime.definelist":
      return `「${str(p.name)}」· ${rowsOf(p).length} 行`;
    case "runtime.processlist":
      return `列表「${str(p.list_name)}」`;
    case "runtime.repeat":
      return bool(p.until_unchanged) ? "直到不再变化" : `${num(p.count, 2)} 次`;
    case "runtime.usernote":
      return str(p.note);
    case "text.insert":
      return q(p.text);
    case "line.addto":
    case "line.removefrom":
      return `${num(p.position) ? "行尾" : "行首"} ${q(p.text)}`;
  }
  return "";
}
