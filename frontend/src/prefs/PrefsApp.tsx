import { useState, type ReactNode } from "react";
import { Keyboard, Settings, SlidersHorizontal, Type } from "lucide-react";
import { Library, type Boot, type Prefs } from "../mygo";
import { keys, useBoot } from "../lib/boot";
import { Segmented, Toasts, errorText, toast } from "../lib/ui";

export default function PrefsApp() {
  const boot = useBoot();
  if (!boot) return <div className="page" />;
  return <PrefsWindow boot={boot} />;
}

function Row({ k, children, hint }: { k: string; children: ReactNode; hint?: ReactNode }) {
  return (
    <div className="form-row">
      <div className="k">{k}</div>
      <div className="v">
        {children}
        {hint && <div className="hint">{hint}</div>}
      </div>
    </div>
  );
}

function Check({ label, value, onChange }: { label: string; value: boolean; onChange: (v: boolean) => void }) {
  return (
    <label className="check">
      <input type="checkbox" checked={value} onChange={(e) => onChange(e.target.checked)} />
      {label}
    </label>
  );
}

/** Records a key combination as a MyGo accelerator. */
function HotkeyField({ value, onChange }: { value: string; onChange: (v: string) => void }) {
  const [rec, setRec] = useState(false);
  return (
    <div className="row">
      <button
        className={"btn" + (rec ? " primary" : "")}
        style={{ minWidth: 150, fontFamily: "var(--mono)" }}
        onClick={() => setRec(true)}
        onBlur={() => setRec(false)}
        onKeyDown={(e) => {
          if (!rec) return;
          e.preventDefault();
          if (e.key === "Escape") return setRec(false);
          if (["Meta", "Control", "Alt", "Shift"].includes(e.key)) return;
          const mods: string[] = [];
          if (e.metaKey || e.ctrlKey) mods.push("CmdOrCtrl");
          if (e.altKey) mods.push("Alt");
          if (e.shiftKey) mods.push("Shift");
          if (!mods.length) return;
          const k = e.code.startsWith("Key") ? e.code.slice(3) : e.code.startsWith("Digit") ? e.code.slice(5) : e.key.length === 1 ? e.key.toUpperCase() : e.key;
          onChange([...mods, k].join("+"));
          setRec(false);
        }}
      >
        {rec ? "请按下快捷键…" : value ? keys(value) : "未设置"}
      </button>
      {value && (
        <button className="btn ghost" onClick={() => onChange("")}>
          清除
        </button>
      )}
    </div>
  );
}

function PrefsWindow({ boot }: { boot: Boot }) {
  const [tab, setTab] = useState<"general" | "editor" | "search" | "agent">("general");
  const p = boot.prefs;
  const set = async <K extends keyof Prefs>(k: K, v: Prefs[K]) => {
    try {
      await Library.setPrefs({ ...p, [k]: v });
    } catch (e) {
      toast(errorText(e), "err");
    }
  };
  const scrubOptions = boot.cleaners.filter((c) => c.implemented);

  return (
    <div className="page">
      <header className="titlebar">
        <span className="page-title">偏好设置</span>
        <span className="spacer" />
        <div className="no-drag" style={{ width: 420 }}>
          <Segmented
            small
            value={tab}
            onChange={setTab}
            options={[
              { value: "general", label: "通用", icon: <Settings /> },
              { value: "editor", label: "编辑器", icon: <Type /> },
              { value: "search", label: "查找", icon: <SlidersHorizontal /> },
              { value: "agent", label: "快捷键与高级", icon: <Keyboard /> },
            ]}
          />
        </div>
        <span className="spacer" />
        <span style={{ width: 60 }} />
      </header>
      <div style={{ flex: 1, overflow: "auto", padding: "8px 22px 24px" }}>
        {tab === "general" && (
          <>
            <div className="section-title">外观</div>
            <div className="panel">
              <Row k="主题">
                <Segmented small value={p.appearance || "system"} onChange={(v) => set("appearance", v)} options={[{ value: "system", label: "跟随系统" }, { value: "light", label: "浅色" }, { value: "dark", label: "深色" }]} />
              </Row>
            </div>
            <div className="section-title">剪贴板工作区</div>
            <div className="panel">
              <Row k="启动时" hint="剪贴板工作区会在打开时自动载入剪贴板文本。停用后，启动时打开一个空白文档。">
                <Segmented small value={p.clipboardWorkspace || "plain"} onChange={(v) => set("clipboardWorkspace", v)} options={[{ value: "plain", label: "打开剪贴板工作区" }, { value: "disabled", label: "打开空白文档" }]} />
              </Row>
              <Row k="关闭时" hint="关闭剪贴板工作区窗口或退出应用时，把其中的文本放回剪贴板，切回原来的应用直接粘贴即可。">
                <Check label="把文本拷贝回剪贴板" value={p.copyBackOnClose} onChange={(v) => set("copyBackOnClose", v)} />
              </Row>
              <Row k="最后一个窗口关闭后">
                <Check label="退出 TextRouter" value={p.quitWhenLastClosed} onChange={(v) => set("quitWhenLastClosed", v)} />
              </Row>
            </div>
            <div className="section-title">清理器</div>
            <div className="panel">
              <Row k="MyScrub 一键清理" hint="工具栏“一键清理”按钮和菜单栏图标使用的清理器。">
                <select className="select" style={{ maxWidth: 320 }} value={p.myScrub || "Scrub"} onChange={(e) => set("myScrub", e.target.value)}>
                  <optgroup label="我的清理器">
                    {scrubOptions.filter((c) => c.custom).map((c) => (
                      <option key={c.name} value={c.name}>{c.zh}</option>
                    ))}
                  </optgroup>
                  <optgroup label="内置清理器">
                    {scrubOptions.filter((c) => !c.custom && c.name !== "MyScrub").map((c) => (
                      <option key={c.name} value={c.name}>{c.zh}</option>
                    ))}
                  </optgroup>
                </select>
              </Row>
              <Row k="没有选中文本时">
                <Check label="清理器作用于全部文本" value={p.applyToAllWhenNoSelection} onChange={(v) => set("applyToAllWhenNoSelection", v)} />
              </Row>
              <Row k="默认分组">
                <select className="select" style={{ maxWidth: 240 }} value={p.defaultGroup} onChange={(e) => set("defaultGroup", e.target.value)}>
                  {boot.groups.map((g) => (
                    <option key={g.id} value={g.id}>{g.name}</option>
                  ))}
                </select>
              </Row>
              <Row k="分组顺序">
                <Segmented small value={p.groupOrder || "builtin-custom"} onChange={(v) => set("groupOrder", v)} options={[{ value: "builtin-custom", label: "内置在前" }, { value: "custom-builtin", label: "我的在前" }, { value: "custom", label: "只显示我的" }]} />
              </Row>
              <Row k="清理器列表">
                <Check label="紧凑显示（不显示英文名）" value={p.compactDisplay} onChange={(v) => set("compactDisplay", v)} />
              </Row>
            </div>
          </>
        )}
        {tab === "editor" && (
          <>
            <div className="section-title">文本编辑</div>
            <div className="panel">
              <Row k="字体">
                <div className="row">
                  <input className="input" style={{ maxWidth: 240 }} placeholder="默认等宽字体" value={p.fontFamily} onChange={(e) => set("fontFamily", e.target.value)} />
                  <input className="input" type="number" style={{ width: 70 }} min={9} max={40} value={p.fontSize} onChange={(e) => set("fontSize", parseInt(e.target.value || "14", 10))} />
                  <span className="muted">pt</span>
                </div>
              </Row>
              <Row k="显示">
                <Check label="行号" value={p.showLineNumbers} onChange={(v) => set("showLineNumbers", v)} />
                <Check label="自动换行" value={p.wrapText} onChange={(v) => set("wrapText", v)} />
                <Check label="不可见字符（空格、制表符、换行）" value={p.showInvisibles} onChange={(v) => set("showInvisibles", v)} />
                <Check label="高亮当前行" value={p.highlightCurrentLine} onChange={(v) => set("highlightCurrentLine", v)} />
                <Check label="底部显示统计（段落、词、字符）" value={p.showWordCount} onChange={(v) => set("showWordCount", v)} />
              </Row>
              <Row k="缩放">
                <div className="row">
                  <input type="range" min={50} max={250} step={10} value={p.zoom} onChange={(e) => set("zoom", Number(e.target.value))} style={{ width: 220, accentColor: "var(--accent)" }} />
                  <span style={{ fontVariantNumeric: "tabular-nums" }}>{p.zoom}%</span>
                </div>
              </Row>
            </div>
          </>
        )}
        {tab === "search" && (
          <>
            <div className="section-title">查找面板的默认选项</div>
            <div className="panel">
              <Row k="新的查找" hint="打开新窗口时查找面板使用这些选项。也可以在查找面板的“选项”菜单里“将当前选项存为默认”。">
                <Check label="正则表达式" value={p.searchRegex} onChange={(v) => set("searchRegex", v)} />
                <Check label="区分大小写" value={p.searchMatchCase} onChange={(v) => set("searchMatchCase", v)} />
                <Check label="实时查找（边输入边查找）" value={p.searchLive} onChange={(v) => set("searchLive", v)} />
                <Check label="^ 和 $ 匹配每一行（多行模式）" value={p.searchMatchLines} onChange={(v) => set("searchMatchLines", v)} />
              </Row>
              <Row k="查找记录">
                <button className="btn" style={{ alignSelf: "flex-start" }} onClick={() => Library.clearHistory().then(() => toast("已清除查找记录"))}>
                  清除查找记录
                </button>
              </Row>
            </div>
          </>
        )}
        {tab === "agent" && (
          <>
            <div className="section-title">随时调出</div>
            <div className="panel">
              <Row k="全局快捷键" hint="在任何应用里按下，打开剪贴板工作区并载入当前剪贴板。">
                <HotkeyField value={p.globalHotkey} onChange={(v) => set("globalHotkey", v)} />
              </Row>
              <Row k="菜单栏图标" hint="从菜单栏打开剪贴板工作区，或直接用清理器处理剪贴板（不打开窗口）。">
                <Check label="在菜单栏显示图标" value={p.menuBarIcon} onChange={(v) => set("menuBarIcon", v)} />
              </Row>
            </div>
            <div className="section-title">高级</div>
            <div className="panel">
              <Row k="Shell 脚本" hint="允许自定义清理器中的“运行 Shell 脚本”动作执行命令。只运行你信任的清理器。">
                <Check label="允许运行 Shell 脚本" value={p.allowShellScripts} onChange={(v) => set("allowShellScripts", v)} />
              </Row>
              <Row k="资料库" hint={boot.libPath}>
                <div className="row">
                  <button className="btn" onClick={() => Library.revealLibrary()}>在文件夹中显示</button>
                  <button className="btn danger" onClick={() => Library.resetPrefs().then(() => toast("已恢复默认设置"))}>恢复默认设置</button>
                </div>
              </Row>
            </div>
          </>
        )}
      </div>
      <Toasts />
    </div>
  );
}
