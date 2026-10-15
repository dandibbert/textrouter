import { useEffect, useLayoutEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { Search } from "lucide-react";

export type PaletteItem = {
  id: string;
  section: string;
  title: string;
  sub?: string;
  icon: ReactNode;
  tile?: "accent" | "scrub" | "plain";
  keys?: string;
  search?: string;
  run: () => void;
};

function score(it: PaletteItem, q: string): number {
  const t = it.title.toLowerCase();
  if (t.startsWith(q)) return 3;
  if (t.includes(q)) return 2;
  if ((it.sub ?? "").toLowerCase().includes(q) || (it.search ?? "").toLowerCase().includes(q)) return 1;
  return 0;
}

/** Raycast-style command palette: type to filter, ↑↓ to move, ↵ to run. */
export function Palette({ items, onClose, placeholder = "搜索清理器和操作…" }: { items: PaletteItem[]; onClose: () => void; placeholder?: string }) {
  const [q, setQ] = useState("");
  const [active, setActive] = useState(0);
  const input = useRef<HTMLInputElement>(null);
  const list = useRef<HTMLDivElement>(null);

  const shown = useMemo(() => {
    const s = q.trim().toLowerCase();
    if (!s) return items;
    return items
      .map((it, i) => ({ it, i, sc: score(it, s) }))
      .filter((x) => x.sc > 0)
      .sort((a, b) => b.sc - a.sc || a.i - b.i)
      .map((x) => ({ ...x.it, section: "结果" }));
  }, [items, q]);

  useEffect(() => setActive(0), [q]);
  useEffect(() => input.current?.focus(), []);
  useLayoutEffect(() => {
    list.current?.querySelector(`[data-i="${active}"]`)?.scrollIntoView({ block: "nearest" });
  }, [active]);

  const run = (it?: PaletteItem) => {
    if (!it) return;
    onClose();
    it.run();
  };

  const rows: ReactNode[] = [];
  let section = "";
  shown.forEach((it, i) => {
    if (it.section !== section) {
      section = it.section;
      rows.push(
        <div key={"s" + i} className="pal-section">
          {section}
        </div>,
      );
    }
    rows.push(
      <button
        key={it.id}
        data-i={i}
        className={"pal-row" + (i === active ? " on" : "")}
        onMouseMove={() => setActive(i)}
        onClick={() => run(it)}
      >
        <span className={"ico-tile " + (it.tile ?? "plain")}>{it.icon}</span>
        <span className="pal-title">{it.title}</span>
        {it.sub && <span className="pal-sub">{it.sub}</span>}
        <span className="spacer" />
        {it.keys && <kbd className="keycap">{it.keys}</kbd>}
      </button>,
    );
  });

  return (
    <div className="pal-backdrop" onMouseDown={(e) => e.target === e.currentTarget && onClose()}>
      <div className="pal" role="dialog">
        <div className="pal-search">
          <Search />
          <input
            ref={input}
            value={q}
            placeholder={placeholder}
            onChange={(e) => setQ(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Escape") onClose();
              else if (e.key === "ArrowDown") {
                e.preventDefault();
                setActive((a) => Math.min(a + 1, shown.length - 1));
              } else if (e.key === "ArrowUp") {
                e.preventDefault();
                setActive((a) => Math.max(a - 1, 0));
              } else if (e.key === "Enter") {
                e.preventDefault();
                run(shown[active]);
              }
            }}
          />
        </div>
        <div className="pal-list" ref={list}>
          {rows.length ? rows : <div className="empty">没有匹配的清理器或操作</div>}
        </div>
        <div className="pal-foot">
          <span className="pal-brand">TextRouter</span>
          <span className="spacer" />
          <span className="pal-hint">
            执行 <kbd className="keycap">↵</kbd>
          </span>
          <span className="pal-sep" />
          <span className="pal-hint">
            关闭 <kbd className="keycap">esc</kbd>
          </span>
        </div>
      </div>
    </div>
  );
}
