import { useEffect, useLayoutEffect, useRef, useState, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { AlertTriangle, Check, CheckCircle2, ChevronDown, Info, XCircle } from "lucide-react";

// ---------------------------------------------------------------------------
// Menus

export type MenuEntry =
  | {
      label: ReactNode;
      code?: string;
      desc?: ReactNode;
      checked?: boolean;
      disabled?: boolean;
      onSelect: () => void;
    }
  | "sep"
  | { head: string };

export function Menu({ anchor, items, onClose, align = "left" }: {
  anchor: DOMRect;
  items: MenuEntry[];
  onClose: () => void;
  align?: "left" | "right";
}) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ left: anchor.left, top: anchor.bottom + 4 });
  useLayoutEffect(() => {
    const el = ref.current;
    if (!el) return;
    const r = el.getBoundingClientRect();
    let left = align === "right" ? anchor.right - r.width : anchor.left;
    left = Math.max(6, Math.min(left, window.innerWidth - r.width - 6));
    let top = anchor.bottom + 4;
    if (top + r.height > window.innerHeight - 6) top = Math.max(6, anchor.top - r.height - 4);
    setPos({ left, top });
  }, [anchor, align]);
  useEffect(() => {
    const down = (e: MouseEvent) => {
      if (!ref.current?.contains(e.target as Node)) onClose();
    };
    const key = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("mousedown", down, true);
    window.addEventListener("keydown", key, true);
    window.addEventListener("blur", onClose);
    return () => {
      window.removeEventListener("mousedown", down, true);
      window.removeEventListener("keydown", key, true);
      window.removeEventListener("blur", onClose);
    };
  }, [onClose]);
  const hasTicks = items.some((i) => typeof i === "object" && "onSelect" in i && i.checked !== undefined);
  return createPortal(
    <div className="menu" ref={ref} style={pos} role="menu">
      {items.map((it, i) => {
        if (it === "sep") return <div key={i} className="menu-sep" />;
        if ("head" in it) return <div key={i} className="menu-head">{it.head}</div>;
        return (
          <button
            key={i}
            className="menu-item"
            disabled={it.disabled}
            role="menuitem"
            onMouseDown={(e) => e.preventDefault()}
            onClick={() => {
              onClose();
              it.onSelect();
            }}
          >
            {hasTicks && <span className="tick">{it.checked && <Check />}</span>}
            {it.code !== undefined && <span className="menu-code">{it.code}</span>}
            <span>{it.label}</span>
            {it.desc && <span className="menu-desc">{it.desc}</span>}
          </button>
        );
      })}
    </div>,
    document.body,
  );
}

/** A button that opens a menu. */
export function MenuButton({ label, icon, items, className = "btn ghost small", title, align, caret = true }: {
  label?: ReactNode;
  icon?: ReactNode;
  items: MenuEntry[] | (() => MenuEntry[]);
  className?: string;
  title?: string;
  align?: "left" | "right";
  caret?: boolean;
}) {
  const [rect, setRect] = useState<DOMRect | null>(null);
  return (
    <>
      <button
        className={className}
        title={title}
        onMouseDown={(e) => e.preventDefault()}
        onClick={(e) => setRect(rect ? null : e.currentTarget.getBoundingClientRect())}
      >
        {icon}
        {label}
        {caret && <ChevronDown className="caret" />}
      </button>
      {rect && <Menu anchor={rect} items={typeof items === "function" ? items() : items} onClose={() => setRect(null)} align={align} />}
    </>
  );
}

// ---------------------------------------------------------------------------
// Toasts

type ToastKind = "ok" | "info" | "warn" | "err";
type ToastItem = { id: number; kind: ToastKind; text: ReactNode; hint?: ReactNode };
let toastListener: ((t: ToastItem[]) => void) | null = null;
let toasts: ToastItem[] = [];
let toastId = 0;

export function toast(text: ReactNode, kind: ToastKind = "ok", hint?: ReactNode, ms = 2600) {
  const t = { id: ++toastId, kind, text, hint };
  toasts = [...toasts.slice(-2), t];
  toastListener?.(toasts);
  setTimeout(() => {
    toasts = toasts.filter((x) => x.id !== t.id);
    toastListener?.(toasts);
  }, ms);
}

export function Toasts() {
  const [list, setList] = useState<ToastItem[]>([]);
  useEffect(() => {
    toastListener = setList;
    return () => {
      toastListener = null;
    };
  }, []);
  const icon = { ok: <CheckCircle2 className="ok" />, info: <Info />, warn: <AlertTriangle className="warn" />, err: <XCircle className="err" /> };
  return (
    <div className="toast-wrap">
      {list.map((t) => (
        <div key={t.id} className="toast">
          {icon[t.kind]}
          <span>{t.text}</span>
          {t.hint && <kbd>{t.hint}</kbd>}
        </div>
      ))}
    </div>
  );
}

// ---------------------------------------------------------------------------
// Hover tips

export function useHoverTip(delay = 550) {
  const [tip, setTip] = useState<{ x: number; y: number; content: ReactNode } | null>(null);
  const timer = useRef<number>(0);
  const bind = (content: () => ReactNode) => ({
    onMouseEnter: (e: React.MouseEvent) => {
      const r = (e.currentTarget as HTMLElement).getBoundingClientRect();
      clearTimeout(timer.current);
      timer.current = window.setTimeout(() => setTip({ x: r.right + 8, y: r.top, content: content() }), delay);
    },
    onMouseLeave: () => {
      clearTimeout(timer.current);
      setTip(null);
    },
    onMouseDown: () => {
      clearTimeout(timer.current);
      setTip(null);
    },
  });
  const node = tip ? <TipBox {...tip} /> : null;
  return { bind, node };
}

function TipBox({ x, y, content }: { x: number; y: number; content: ReactNode }) {
  const ref = useRef<HTMLDivElement>(null);
  const [pos, setPos] = useState({ left: x, top: y });
  useLayoutEffect(() => {
    const r = ref.current?.getBoundingClientRect();
    if (!r) return;
    setPos({
      left: x + r.width > window.innerWidth - 8 ? Math.max(8, window.innerWidth - r.width - 8) : x,
      top: Math.min(y, window.innerHeight - r.height - 8),
    });
  }, [x, y]);
  return createPortal(
    <div className="tip" ref={ref} style={pos}>
      {content}
    </div>,
    document.body,
  );
}

// ---------------------------------------------------------------------------

export function Segmented<T extends string>({ value, options, onChange, small }: {
  value: T;
  options: { value: T; label: ReactNode; icon?: ReactNode; title?: string }[];
  onChange: (v: T) => void;
  small?: boolean;
}) {
  const i = Math.max(0, options.findIndex((o) => o.value === value));
  return (
    <div className={"segmented" + (small ? " small" : "")} role="tablist">
      <span
        className="segmented-thumb"
        aria-hidden
        style={{ width: `calc((100% - 6px) / ${options.length})`, transform: `translateX(${i * 100}%)` }}
      />
      {options.map((o) => (
        <button key={o.value} className={o.value === value ? "on" : ""} title={o.title} onClick={() => onChange(o.value)} role="tab">
          {o.icon}
          {o.label}
        </button>
      ))}
    </div>
  );
}

export function errorText(e: unknown): string {
  return e instanceof Error ? e.message : String(e);
}
