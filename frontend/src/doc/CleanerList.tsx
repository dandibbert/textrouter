import { forwardRef, useImperativeHandle, useMemo, useRef, useState } from "react";
import { FolderOpen, Layers, Pencil, Search, Sparkle, Wand2 } from "lucide-react";
import type { Boot, CleanerInfo, Group, GroupItem } from "../mygo";
import { Document } from "../mygo";
import { MenuButton, useHoverTip, type MenuEntry } from "../lib/ui";

export type CleanerListHandle = { focusSearch: () => void };

type Row = { kind: "cleaner"; c: CleanerInfo } | { kind: "label"; name: string; color?: string } | { kind: "divider" };

function matches(c: CleanerInfo, q: string) {
  const s = q.toLowerCase();
  return c.zh.toLowerCase().includes(s) || c.name.toLowerCase().includes(s) || c.desc.toLowerCase().includes(s);
}

export const CleanerList = forwardRef<CleanerListHandle, {
  boot: Boot;
  onApply: (c: CleanerInfo) => void;
}>(function CleanerList({ boot, onApply }, ref) {
  const [groupId, setGroupId] = useState(boot.prefs.defaultGroup || "standard");
  const [query, setQuery] = useState("");
  const [flash, setFlash] = useState("");
  const search = useRef<HTMLInputElement>(null);
  const tip = useHoverTip();
  useImperativeHandle(ref, () => ({ focusSearch: () => (search.current?.focus(), search.current?.select()) }));

  const byName = useMemo(() => new Map(boot.cleaners.map((c) => [c.name, c])), [boot.cleaners]);
  const group: Group = boot.groups.find((g) => g.id === groupId) ?? boot.groups[0];

  const rows = useMemo(() => {
    const toRow = (it: GroupItem): Row | null => {
      if (it.kind === "label") return { kind: "label", name: it.name ?? "", color: it.color };
      if (it.kind === "divider") return { kind: "divider" };
      const c = byName.get(it.name ?? "");
      return c ? { kind: "cleaner", c } : null;
    };
    const inGroup = (group?.items ?? []).map(toRow).filter((r): r is Row => r !== null);
    if (!query.trim()) return { main: inGroup, more: [] as CleanerInfo[] };
    const q = query.trim();
    const main = inGroup.filter((r) => r.kind === "cleaner" && matches(r.c, q));
    const seen = new Set(main.map((r) => (r.kind === "cleaner" ? r.c.name : "")));
    const more = boot.cleaners.filter((c) => !seen.has(c.name) && matches(c, q));
    return { main, more };
  }, [group, query, byName, boot.cleaners]);

  const apply = (c: CleanerInfo) => {
    setFlash(c.name);
    setTimeout(() => setFlash(""), 260);
    onApply(c);
  };

  const groupMenu = (): MenuEntry[] => {
    const builtIn = boot.groups.filter((g) => g.builtIn);
    const mine = boot.groups.filter((g) => !g.builtIn);
    const entry = (g: Group): MenuEntry => ({ label: g.name, checked: g.id === group?.id, onSelect: () => setGroupId(g.id) });
    const out: MenuEntry[] = [{ head: "内置分组" }, ...builtIn.map(entry)];
    if (mine.length) out.push("sep", { head: "我的分组" }, ...mine.map(entry));
    out.push("sep", { label: "编辑分组…", onSelect: () => Document.openLibrary("groups") });
    return out;
  };

  const cleanerRow = (c: CleanerInfo, key: string) => (
    <button
      key={key}
      className={"cleaner" + (c.custom ? " custom" : "") + (c.implemented ? "" : " disabled") + (flash === c.name ? " flash" : "")}
      onClick={() => apply(c)}
      {...tip.bind(() => (
        <>
          <b>{c.zh}</b>
          {!c.custom && <div className="mono" style={{ fontSize: 11, marginBottom: 4 }}>{c.name}</div>}
          {c.desc || (c.custom ? "自定义清理器" : "")}
          {!c.implemented && <div style={{ marginTop: 4, color: "var(--orange)" }}>只对富文本有效，纯文本不会改变。</div>}
        </>
      ))}
    >
      <span className={"ico-tile" + (c.custom ? " custom" : c.name === "MyScrub" || c.name === "Scrub" ? " scrub" : "")}>
        {c.custom ? <Layers /> : c.name === "MyScrub" || c.name === "Scrub" ? <Sparkle /> : <Wand2 />}
      </span>
      <span className="names">
        <span className="zh">{c.zh}</span>
        <span className="sub">{c.custom ? c.desc || "自定义清理器" : c.name}</span>
      </span>
      {c.custom && (
        <span
          className="btn ghost small icon edit"
          title="在我的库中编辑"
          onClick={(e) => {
            e.stopPropagation();
            Document.openLibrary(c.uuid ?? "");
          }}
        >
          <Pencil />
        </span>
      )}
    </button>
  );

  return (
    <div className={"cleaners" + (boot.prefs.compactDisplay ? " compact" : "")}>
      <div className="cleaners-head">
        <MenuButton
          className="select-btn"
          icon={<FolderOpen />}
          label={<span className="select-label">{group?.name ?? "分组"}</span>}
          items={groupMenu}
          title="切换清理器分组"
        />
        <div className="search">
          <Search />
          <input
            ref={search}
            className="input"
            placeholder="搜索清理器"
            value={query}
            onChange={(e) => setQuery(e.target.value)}
            onKeyDown={(e) => {
              if (e.key === "Escape") setQuery("");
              if (e.key === "Enter") {
                const first = rows.main.find((r) => r.kind === "cleaner") as { c: CleanerInfo } | undefined;
                const c = first?.c ?? rows.more[0];
                if (c) apply(c);
              }
            }}
          />
        </div>
      </div>
      <div className="cleaner-scroll">
        {rows.main.map((r, i) =>
          r.kind === "cleaner" ? (
            cleanerRow(r.c, "m" + i)
          ) : r.kind === "label" ? (
            <div key={"l" + i} className="list-label" style={r.color ? { color: r.color } : undefined}>
              {r.name}
            </div>
          ) : (
            <div key={"d" + i} className="list-divider" />
          ),
        )}
        {query && rows.more.length > 0 && (
          <>
            <div className="list-label">库中的其他结果</div>
            {rows.more.map((c) => cleanerRow(c, "o" + c.name))}
          </>
        )}
        {rows.main.length === 0 && rows.more.length === 0 && (
          <div className="empty">{query ? "没有找到清理器" : "这个分组是空的"}</div>
        )}
      </div>
      {tip.node}
    </div>
  );
});
