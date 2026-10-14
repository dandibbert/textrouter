import { useEffect, useMemo, useState } from "react";
import { ArrowDown, ArrowUp, Copy, FolderOpen, FolderPlus, Lock, Minus, Plus, Tag, Trash2 } from "lucide-react";
import { Document, Library, type Boot, type Group, type GroupItem } from "../mygo";
import { MenuButton, toast } from "../lib/ui";
import { debounce } from "../lib/text";

const COLORS = ["", "#e5484d", "#f76b15", "#ffc53d", "#30a46c", "#0090ff", "#8e4ec6", "#8d8d8d"];

export function GroupEditor({ boot }: { boot: Boot }) {
  const [sel, setSel] = useState(boot.groups.find((g) => !g.builtIn)?.id ?? boot.groups[0]?.id);
  const group = boot.groups.find((g) => g.id === sel);
  const builtIn = boot.groups.filter((g) => g.builtIn);
  const mine = boot.groups.filter((g) => !g.builtIn);

  const create = async (from?: Group) => {
    const g = await Library.saveGroup({ id: "", name: from ? from.name + " 副本" : "新分组", items: from ? structuredClone(from.items) : [] });
    setSel(g.id);
  };

  return (
    <div className="lib-body groups-body">
      <aside className="lib-list">
        <div className="lib-items" style={{ paddingTop: 10 }}>
          <div className="list-label">我的分组</div>
          {mine.map((g) => (
            <button key={g.id} className={"lib-item" + (g.id === sel ? " on" : "")} onClick={() => setSel(g.id)}>
              <FolderOpen />
              <span className="names">
                <span className="n">{g.name}</span>
                <span className="s">{g.items.filter((i) => i.kind === "cleaner").length} 个清理器</span>
              </span>
            </button>
          ))}
          {!mine.length && <div className="hint" style={{ padding: "4px 10px 8px" }}>还没有自己的分组</div>}
          <div className="list-label">内置分组</div>
          {builtIn.map((g) => (
            <button key={g.id} className={"lib-item" + (g.id === sel ? " on" : "")} onClick={() => setSel(g.id)}>
              <Lock />
              <span className="names">
                <span className="n">{g.name}</span>
              </span>
            </button>
          ))}
        </div>
        <div className="lib-foot">
          <button className="btn small" onClick={() => create()}>
            <FolderPlus /> 新建分组
          </button>
        </div>
      </aside>
      {group ? (
        group.builtIn ? (
          <BuiltinView key={group.id} boot={boot} group={group} onCopy={() => create(group)} />
        ) : (
          <GroupForm key={group.id} boot={boot} group={group} />
        )
      ) : (
        <div className="lib-empty">选择一个分组</div>
      )}
    </div>
  );
}

function BuiltinView({ boot, group, onCopy }: { boot: Boot; group: Group; onCopy: () => void }) {
  const zh = useMemo(() => new Map(boot.cleaners.map((c) => [c.name, c.zh])), [boot.cleaners]);
  return (
    <section className="group-editor">
      <div className="lib-head">
        <div className="row">
          <div className="name-input" style={{ flex: 1 }}>{group.name}</div>
          <button className="btn" onClick={onCopy}>
            <Copy /> 复制为我的分组
          </button>
        </div>
        <div className="hint">内置分组不能修改。复制一份后可以自由增删、排序，并加入标签和分隔线。</div>
      </div>
      <div className="group-items">
        {group.items.map((it, i) =>
          it.kind === "cleaner" ? (
            <div key={i} className="gitem">
              {zh.get(it.name ?? "") ?? it.name}
            </div>
          ) : it.kind === "label" ? (
            <div key={i} className="gitem label">{it.name}</div>
          ) : (
            <div key={i} className="gitem divider" />
          ),
        )}
      </div>
    </section>
  );
}

function GroupForm({ boot, group }: { boot: Boot; group: Group }) {
  const [draft, setDraft] = useState(group);
  const [sel, setSel] = useState(-1);
  const save = useMemo(() => debounce((g: Group) => Library.saveGroup(g), 400), []);
  useEffect(() => setDraft(group), [group.id]); // eslint-disable-line react-hooks/exhaustive-deps
  const change = (fn: (g: Group) => Group) =>
    setDraft((d) => {
      const n = fn(d);
      save(n);
      return n;
    });
  const items = draft.items ?? [];
  const insert = (it: GroupItem) => {
    const at = sel >= 0 ? sel + 1 : items.length;
    change((g) => ({ ...g, items: [...items.slice(0, at), it, ...items.slice(at)] }));
    setSel(at);
  };
  const move = (d: number) => {
    if (sel < 0) return;
    const to = Math.max(0, Math.min(items.length - 1, sel + d));
    const next = [...items];
    const [x] = next.splice(sel, 1);
    next.splice(to, 0, x);
    change((g) => ({ ...g, items: next }));
    setSel(to);
  };
  const zh = new Map(boot.cleaners.map((c) => [c.name, c.zh]));
  const present = new Set(items.filter((i) => i.kind === "cleaner").map((i) => i.name));
  const cats = [...new Set(boot.cleaners.map((c) => (c.custom ? "我的清理器" : c.category)))];

  const addMenu = () =>
    cats.flatMap((cat) => [
      { head: cat },
      ...boot.cleaners
        .filter((c) => (c.custom ? "我的清理器" : c.category) === cat && !present.has(c.name))
        .map((c) => ({ label: c.zh, onSelect: () => insert({ kind: "cleaner", name: c.name }) })),
    ]);

  return (
    <section className="group-editor">
      <div className="lib-head">
        <div className="row">
          <input className="name-input" style={{ flex: 1 }} value={draft.name} onChange={(e) => change((g) => ({ ...g, name: e.target.value }))} />
          <button
            className="btn danger"
            onClick={async () => {
              if (!(await Document.confirm(`删除分组「${draft.name}」？`, "分组里的清理器不会被删除。", "删除"))) return;
              await Library.deleteGroup(draft.id);
              toast("已删除分组");
            }}
          >
            <Trash2 /> 删除分组
          </button>
        </div>
        <div className="row" style={{ marginTop: 8 }}>
          <MenuButton className="btn" icon={<Plus />} label="添加清理器" items={addMenu} />
          <button className="btn" onClick={() => insert({ kind: "label", name: "标签" })}>
            <Tag /> 添加标签
          </button>
          <button className="btn" onClick={() => insert({ kind: "divider" })}>
            <Minus /> 添加分隔线
          </button>
          <span className="spacer" />
          <button className="btn icon" disabled={sel <= 0} onClick={() => move(-1)} title="上移">
            <ArrowUp />
          </button>
          <button className="btn icon" disabled={sel < 0 || sel >= items.length - 1} onClick={() => move(1)} title="下移">
            <ArrowDown />
          </button>
          <button
            className="btn icon danger"
            disabled={sel < 0}
            title="从分组移除"
            onClick={() => {
              change((g) => ({ ...g, items: items.filter((_, k) => k !== sel) }));
              setSel(-1);
            }}
          >
            <Trash2 />
          </button>
        </div>
      </div>
      <div className="group-items">
        {items.map((it, i) => (
          <div key={i} className={"gitem" + (it.kind !== "cleaner" ? " " + it.kind : "") + (i === sel ? " on" : "")} onClick={() => setSel(i)}>
            {it.kind === "cleaner" && (zh.get(it.name ?? "") ?? it.name)}
            {it.kind === "label" && (
              <>
                <input
                  className="label-input"
                  style={{ color: it.color || undefined }}
                  value={it.name ?? ""}
                  onChange={(e) => change((g) => ({ ...g, items: items.map((x, k) => (k === i ? { ...x, name: e.target.value } : x)) }))}
                />
                <span className="colors">
                  {COLORS.map((c) => (
                    <button
                      key={c || "none"}
                      className={"swatch" + ((it.color ?? "") === c ? " on" : "")}
                      style={{ background: c || "var(--text-2)" }}
                      title={c ? c : "默认颜色"}
                      onClick={() => change((g) => ({ ...g, items: items.map((x, k) => (k === i ? { ...x, color: c } : x)) }))}
                    />
                  ))}
                </span>
              </>
            )}
          </div>
        ))}
        {!items.length && <div className="empty">点“添加清理器”把常用的清理器放进来。</div>}
      </div>
    </section>
  );
}
