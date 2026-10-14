import { useState } from "react";
import { Plus, Table2, Trash2, Type } from "lucide-react";
import { Segmented } from "../lib/ui";

const COLS = ["A", "B", "C", "D"];

export function rowsOf(params: Record<string, unknown>): string[][] {
  const rows = params.rows;
  if (Array.isArray(rows)) return rows.map((r) => (Array.isArray(r) ? r.map((c) => String(c ?? "")) : [String(r ?? "")]));
  if (typeof params.text === "string") return parseTSV(params.text);
  return [];
}

export function parseTSV(s: string): string[][] {
  return s
    .replace(/\r\n?/g, "\n")
    .split("\n")
    .filter((l) => l !== "")
    .map((l) => l.split("\t"));
}

/** Editor for a Define List table: 1–4 columns named A–D. */
export function ListTable({ rows, columns, onChange }: {
  rows: string[][];
  columns: number;
  onChange: (rows: string[][], columns: number) => void;
}) {
  const [mode, setMode] = useState<"table" | "text">("table");
  const [draft, setDraft] = useState("");
  const cols = Math.max(1, Math.min(4, columns || 1));
  const set = (r: number, c: number, v: string) => {
    const next = rows.map((row) => [...row]);
    while (next[r].length < cols) next[r].push("");
    next[r][c] = v;
    onChange(next, cols);
  };
  return (
    <div className="list-table">
      <div className="row" style={{ marginBottom: 8 }}>
        <span className="label" style={{ margin: 0 }}>列数</span>
        <Segmented
          small
          value={String(cols)}
          onChange={(v) => onChange(rows, Number(v))}
          options={[1, 2, 3, 4].map((n) => ({ value: String(n), label: `${n} 列` }))}
        />
        <span className="spacer" />
        <Segmented
          small
          value={mode}
          onChange={(m) => {
            if (m === "text") setDraft(rows.map((r) => r.slice(0, cols).join("\t")).join("\n"));
            setMode(m);
          }}
          options={[
            { value: "table", label: "表格", icon: <Table2 /> },
            { value: "text", label: "文本", icon: <Type />, title: "每行一条，列之间用 Tab 分隔，可直接从表格软件粘贴" },
          ]}
        />
      </div>
      {mode === "text" ? (
        <>
          <textarea
            className="textarea mono"
            rows={Math.min(14, Math.max(5, rows.length + 1))}
            value={draft}
            placeholder={"colour\tcolor\nfavourite\tfavorite"}
            onChange={(e) => {
              setDraft(e.target.value);
              const parsed = parseTSV(e.target.value);
              const width = Math.max(cols, ...parsed.map((r) => Math.min(4, r.length)));
              onChange(parsed, width);
            }}
          />
          <div className="hint" style={{ marginTop: 4 }}>每行一条记录，列之间用 Tab 分隔（从 Excel / Numbers 复制的表格可直接粘贴）。</div>
        </>
      ) : (
        <div className="table-wrap">
          <table className="grid">
            <thead>
              <tr>
                <th className="n">#</th>
                {COLS.slice(0, cols).map((c) => (
                  <th key={c}>
                    {c} 列 <span className="muted mono">$v{"{" + c + "}"}</span>
                  </th>
                ))}
                <th className="x" />
              </tr>
            </thead>
            <tbody>
              {rows.map((row, r) => (
                <tr key={r}>
                  <td className="n">{r + 1}</td>
                  {COLS.slice(0, cols).map((_, c) => (
                    <td key={c}>
                      <input className="cell mono" value={row[c] ?? ""} onChange={(e) => set(r, c, e.target.value)} />
                    </td>
                  ))}
                  <td className="x">
                    <button className="btn ghost small icon" title="删除这一行" onClick={() => onChange(rows.filter((_, k) => k !== r), cols)}>
                      <Trash2 />
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
          {rows.length === 0 && <div className="empty" style={{ padding: 14 }}>列表是空的</div>}
          <button className="btn ghost small" style={{ margin: 6 }} onClick={() => onChange([...rows, Array(cols).fill("")], cols)}>
            <Plus /> 添加一行
          </button>
        </div>
      )}
    </div>
  );
}
