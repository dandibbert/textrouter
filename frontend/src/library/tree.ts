import type { Action, ActionInfo } from "../mygo";

export type Path = number[];

export function getAt(list: Action[], path: Path): Action | undefined {
  let cur: Action | undefined;
  let items = list;
  for (const i of path) {
    cur = items[i];
    if (!cur) return undefined;
    items = cur.actionList ?? [];
  }
  return cur;
}

/** Returns a copy of list with the children at parent replaced by fn(children). */
export function editChildren(list: Action[], parent: Path, fn: (children: Action[]) => Action[]): Action[] {
  if (parent.length === 0) return fn([...list]);
  const [i, ...rest] = parent;
  return list.map((a, k) => (k === i ? { ...a, actionList: editChildren(a.actionList ?? [], rest, fn) } : a));
}

export function updateAt(list: Action[], path: Path, fn: (a: Action) => Action): Action[] {
  const parent = path.slice(0, -1);
  const idx = path[path.length - 1];
  return editChildren(list, parent, (ch) => ch.map((a, k) => (k === idx ? fn(a) : a)));
}

export function removeAt(list: Action[], path: Path): Action[] {
  const idx = path[path.length - 1];
  return editChildren(list, path.slice(0, -1), (ch) => ch.filter((_, k) => k !== idx));
}

export function insertAt(list: Action[], parent: Path, index: number, a: Action): Action[] {
  return editChildren(list, parent, (ch) => {
    ch.splice(Math.max(0, Math.min(index, ch.length)), 0, a);
    return ch;
  });
}

export function moveWithin(list: Action[], path: Path, delta: number): { list: Action[]; path: Path } {
  const parent = path.slice(0, -1);
  const idx = path[path.length - 1];
  let to = idx;
  const out = editChildren(list, parent, (ch) => {
    to = Math.max(0, Math.min(ch.length - 1, idx + delta));
    const [a] = ch.splice(idx, 1);
    ch.splice(to, 0, a);
    return ch;
  });
  return { list: out, path: [...parent, to] };
}

/** Moves the action at `from` to position `index` among the children of `parent`. */
export function moveTo(list: Action[], from: Path, parent: Path, index: number): { list: Action[]; path: Path } {
  const a = getAt(list, from);
  if (!a) return { list, path: from };
  // Dropping into itself is not allowed.
  if (parent.length >= from.length && from.every((v, i) => parent[i] === v)) return { list, path: from };
  let target = index;
  const sameParent = from.length - 1 === parent.length && parent.every((v, i) => from[i] === v);
  if (sameParent && from[from.length - 1] < index) target--;
  let next = removeAt(list, from);
  // Removing may shift the parent path when it lies after `from` at the same level.
  const adj = [...parent];
  const lvl = from.length - 1;
  if (adj.length > lvl && from.slice(0, lvl).every((v, i) => adj[i] === v) && adj[lvl] > from[lvl]) adj[lvl]--;
  next = insertAt(next, adj, target, a);
  return { list: next, path: [...adj, target] };
}

export function newAction(info: ActionInfo): Action {
  return {
    identifier: info.id,
    parameters: structuredClone(info.defaults ?? {}),
    enabled: true,
    ...(info.container ? { actionList: [] } : {}),
  };
}

/** Names of lists and macros defined anywhere in a cleaner. */
export function definitions(list: Action[]) {
  const lists: string[] = [];
  const macros: string[] = [];
  const walk = (as: Action[]) => {
    for (const a of as) {
      if (a.identifier === "runtime.definelist") lists.push(String(a.parameters?.name ?? ""));
      if (a.identifier === "runtime.macro") macros.push(String(a.parameters?.name ?? ""));
      if (a.actionList) walk(a.actionList);
    }
  };
  walk(list);
  return { lists: lists.filter(Boolean), macros: macros.filter(Boolean) };
}

export const str = (v: unknown, def = "") => (v === undefined || v === null ? def : String(v));
export const num = (v: unknown, def = 0) => {
  const n = typeof v === "number" ? v : parseInt(String(v ?? ""), 10);
  return Number.isFinite(n) ? n : def;
};
export const bool = (v: unknown, def = false) => {
  if (v === undefined || v === null || v === "") return def;
  if (typeof v === "boolean") return v;
  if (typeof v === "number") return v !== 0;
  return ["1", "true", "yes"].includes(String(v).toLowerCase());
};
