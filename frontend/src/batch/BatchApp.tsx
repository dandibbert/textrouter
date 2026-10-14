import { useEffect, useState } from "react";
import { CheckCircle2, File, FilePlus, Folder, FolderOpen, FolderPlus, Play, Square, Trash2, XCircle } from "lucide-react";
import { Channel, onFileDrop } from "mygo-runtime";
import { Batch, type BatchItem, type BatchProgress, type BatchSettings, type BatchSummary, type Boot } from "../mygo";
import { useBoot } from "../lib/boot";
import { Segmented, Toasts, errorText, toast } from "../lib/ui";

export default function BatchApp() {
  const boot = useBoot();
  if (!boot) return <div className="page" />;
  return <BatchWindow boot={boot} />;
}

function BatchWindow({ boot }: { boot: Boot }) {
  const [s, setS] = useState<BatchSettings>(boot.batch);
  const [items, setItems] = useState<BatchItem[]>([]);
  const [log, setLog] = useState<BatchProgress[]>([]);
  const [running, setRunning] = useState<Channel<BatchProgress> | null>(null);
  const [summary, setSummary] = useState<BatchSummary | null>(null);
  const set = <K extends keyof BatchSettings>(k: K, v: BatchSettings[K]) => setS((x) => ({ ...x, [k]: v }));
  const add = (more: BatchItem[]) => setItems((cur) => [...cur, ...more.filter((m) => !cur.some((c) => c.path === m.path))]);

  useEffect(() => onFileDrop(({ paths }) => Batch.describe(paths).then(add)), []);

  const absDest = s.destFolder.startsWith("/") || /^[A-Za-z]:\\/.test(s.destFolder);
  const example = (() => {
    const name = "notes.txt";
    const ext = s.newExt ? "." + s.newExt.replace(/^\./, "") : ".txt";
    const file = `${s.prefix}notes${s.suffix}${ext}`;
    if (!s.destFolder) return name === file ? "notes.txt（覆盖原文件！）" : file;
    return absDest ? `${s.destFolder}/${file}` : `<原文件夹>/${s.destFolder}/${file}`;
  })();

  const run = async () => {
    setLog([]);
    setSummary(null);
    const ch = new Channel<BatchProgress>((p) => setLog((l) => [...l, p]));
    setRunning(ch);
    try {
      const r = await Batch.run(s, items, ch);
      setSummary(r);
      toast(`处理完成：${r.cleaned} 个文件`, r.failed ? "warn" : "ok");
    } catch (e) {
      toast(errorText(e), "err", undefined, 5000);
    } finally {
      setRunning(null);
    }
  };

  const last = log[log.length - 1];

  return (
    <div className="page">
      <header className="titlebar">
        <span className="page-title">批量处理文件</span>
        <span className="spacer" />
        {running ? (
          <button className="tool" onClick={() => running.close()}>
            <Square />
            <span>停止</span>
          </button>
        ) : (
          <button className="tool primary" disabled={!items.length} onClick={run}>
            <Play />
            <span>开始处理</span>
          </button>
        )}
      </header>
      <div style={{ flex: 1, overflow: "auto", padding: "8px 22px 24px" }}>
        <div className="section-title">要处理的文件</div>
        <div className="panel" style={{ padding: 10 }}>
          <div className="row" style={{ marginBottom: 8 }}>
            <button className="btn" onClick={() => Batch.choose(false).then(add)}>
              <FilePlus /> 添加文件…
            </button>
            <button className="btn" onClick={() => Batch.choose(true).then(add)}>
              <FolderPlus /> 添加文件夹…
            </button>
            <span className="spacer" />
            {items.length > 0 && (
              <button className="btn ghost" onClick={() => setItems([])}>
                清空
              </button>
            )}
          </div>
          {items.length === 0 ? (
            <div className="drop-empty" style={{ padding: 26 }}>
              把文件或文件夹拖到这里
            </div>
          ) : (
            <div style={{ maxHeight: 180, overflow: "auto" }}>
              {items.map((it) => (
                <div key={it.path} className="row" style={{ padding: "4px 6px" }}>
                  {it.isDir ? <Folder style={{ width: 15, color: "var(--accent)" }} /> : <File style={{ width: 15, color: "var(--text-3)" }} />}
                  <span className="mono" style={{ fontSize: 12, flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={it.path}>
                    {it.path}
                  </span>
                  <button className="btn ghost small icon" onClick={() => setItems(items.filter((x) => x.path !== it.path))}>
                    <Trash2 />
                  </button>
                </div>
              ))}
            </div>
          )}
        </div>

        <div className="section-title">处理方式</div>
        <div className="panel">
          <div className="form-row">
            <div className="k">清理器</div>
            <div className="v">
              <select className="select" style={{ maxWidth: 340 }} value={s.cleaner} onChange={(e) => set("cleaner", e.target.value)}>
                <optgroup label="我的清理器">
                  {boot.cleaners.filter((c) => c.custom).map((c) => (
                    <option key={c.name} value={c.name}>{c.zh}</option>
                  ))}
                </optgroup>
                <optgroup label="内置清理器">
                  {boot.cleaners.filter((c) => !c.custom && c.implemented).map((c) => (
                    <option key={c.name} value={c.name}>{c.zh}</option>
                  ))}
                </optgroup>
              </select>
            </div>
          </div>
          <div className="form-row">
            <div className="k">文件夹里的文件</div>
            <div className="v">
              <Segmented small value={s.extMode || "additional"} onChange={(v) => set("extMode", v)} options={[{ value: "additional", label: "常见文本文件 + 以下扩展名" }, { value: "only", label: "只处理以下扩展名" }]} />
              <input className="input mono" style={{ maxWidth: 340 }} placeholder="例如 rtf, adoc" value={s.extensions} onChange={(e) => set("extensions", e.target.value)} />
              <label className="check">
                <input type="checkbox" checked={s.includeSubfolders} onChange={(e) => set("includeSubfolders", e.target.checked)} />
                包含子文件夹
              </label>
            </div>
          </div>
        </div>

        <div className="section-title">输出</div>
        <div className="panel">
          <div className="form-row">
            <div className="k">保存到</div>
            <div className="v">
              <div className="row">
                <input className="input mono" style={{ maxWidth: 340 }} value={s.destFolder} onChange={(e) => set("destFolder", e.target.value)} placeholder="留空 = 原文件夹" />
                <button className="btn" onClick={() => Batch.chooseDestination().then((p) => p && set("destFolder", p))}>
                  <FolderOpen /> 选择…
                </button>
              </div>
              <div className="hint">填名字（如 results）表示每个文件旁边的子文件夹；也可以选择一个固定的文件夹。原文件不会被修改。</div>
            </div>
          </div>
          <div className="form-row">
            <div className="k">文件名</div>
            <div className="v">
              <div className="row">
                <input className="input" style={{ width: 110 }} placeholder="前缀" value={s.prefix} onChange={(e) => set("prefix", e.target.value)} />
                <span className="muted">原文件名</span>
                <input className="input" style={{ width: 110 }} placeholder="后缀" value={s.suffix} onChange={(e) => set("suffix", e.target.value)} />
                <input className="input" style={{ width: 90 }} placeholder="新扩展名" value={s.newExt} onChange={(e) => set("newExt", e.target.value)} />
              </div>
              <div className="hint">
                例：notes.txt → <span className="mono">{example}</span>
              </div>
            </div>
          </div>
        </div>

        {(log.length > 0 || summary) && (
          <>
            <div className="section-title">
              进度 {last && `${last.done} / ${last.total}`}
              {summary && ` · 完成 ${summary.cleaned}，失败 ${summary.failed}，跳过 ${summary.skipped}`}
            </div>
            <div className="panel" style={{ padding: 8, maxHeight: 220, overflow: "auto" }}>
              {log.map((l, i) => (
                <div key={i} className="row" style={{ padding: "3px 4px", fontSize: 12 }}>
                  {l.error ? <XCircle style={{ width: 14, color: "var(--red)" }} /> : <CheckCircle2 style={{ width: 14, color: "var(--green)" }} />}
                  <span className="mono" style={{ flex: 1, overflow: "hidden", textOverflow: "ellipsis", whiteSpace: "nowrap" }} title={l.output || l.source}>
                    {l.error ? `${l.source}：${l.error}` : l.output}
                  </span>
                  {l.output && !l.error && (
                    <button className="btn ghost small" onClick={() => Batch.reveal(l.output!)}>
                      显示
                    </button>
                  )}
                </div>
              ))}
            </div>
          </>
        )}
      </div>
      <Toasts />
    </div>
  );
}
