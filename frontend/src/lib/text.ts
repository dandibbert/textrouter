export type Token = { from: number; to: number; cls: string };

/** Highlights a TextSoap (ICU) regular expression. */
export function tokenizeRegex(p: string): Token[] {
  const out: Token[] = [];
  const push = (from: number, to: number, cls: string) => to > from && out.push({ from, to, cls });
  let i = 0;
  while (i < p.length) {
    const c = p[i];
    if (c === "\\") {
      const start = i;
      const n = p[i + 1] ?? "";
      if (n === "Q") {
        const end = p.indexOf("\\E", i + 2);
        const stop = end < 0 ? p.length : end + 2;
        push(start, i + 2, "rx-escape");
        if (end >= 0) push(end, stop, "rx-escape");
        i = stop;
        continue;
      }
      i += 2;
      if ("xpPNk".includes(n) && (p[i] === "{" || p[i] === "<")) {
        const close = p.indexOf(p[i] === "{" ? "}" : ">", i);
        i = close < 0 ? p.length : close + 1;
      } else if (n === "u") i = Math.min(p.length, i + 4);
      else if (n === "U") i = Math.min(p.length, i + 8);
      else if (n === "x") i = Math.min(p.length, i + 2);
      const cls = /[1-9]/.test(n) || n === "k" ? "rx-ref" : "bBAzZG".includes(n) ? "rx-anchor" : /[a-zA-Z]/.test(n) ? "rx-escape" : "";
      if (cls) push(start, i, cls);
      continue;
    }
    if (c === "[") {
      const start = i;
      let depth = 0;
      i++;
      if (p[i] === "^") i++;
      if (p[i] === "]") i++;
      while (i < p.length) {
        if (p[i] === "\\") i += 2;
        else if (p[i] === "[") {
          depth++;
          i++;
        } else if (p[i] === "]") {
          i++;
          if (depth-- === 0) break;
        } else i++;
      }
      push(start, Math.min(i, p.length), "rx-class");
      continue;
    }
    if (c === "(") {
      const m = /^\((\?(?:<[=!]|<[A-Za-z][A-Za-z0-9]*>|'[A-Za-z][A-Za-z0-9]*'|[:=!>|]|[-imsxwU]+[:)])?)?/.exec(p.slice(i));
      const len = m ? m[0].length : 1;
      push(i, i + len, "rx-group");
      i += len;
      continue;
    }
    if (c === ")") {
      push(i, i + 1, "rx-group");
      i++;
      continue;
    }
    if (c === "*" || c === "+" || c === "?") {
      let j = i + 1;
      if (p[j] === "?" || p[j] === "+") j++;
      push(i, j, "rx-quant");
      i = j;
      continue;
    }
    if (c === "{") {
      const m = /^\{\d+(,\d*)?\}[?+]?/.exec(p.slice(i));
      if (m) {
        push(i, i + m[0].length, "rx-quant");
        i += m[0].length;
        continue;
      }
    }
    if (c === "^" || c === "$") {
      push(i, i + 1, "rx-anchor");
      i++;
      continue;
    }
    if (c === "|") {
      push(i, i + 1, "rx-alt");
      i++;
      continue;
    }
    if (c === ".") {
      push(i, i + 1, "rx-escape");
      i++;
      continue;
    }
    i++;
  }
  return out;
}

/** Highlights a replace template ($1, $u, \n, $v{A}...). */
export function tokenizeTemplate(p: string, regex: boolean): Token[] {
  const out: Token[] = [];
  const re = regex ? /\$v\{[^}]*\}|\$[0-9uUlLEe$]|\\x\{[0-9A-Fa-f]+\}|\\./g : /\$v\{[^}]*\}|\\x\{[0-9A-Fa-f]+\}|\\[tns\\]/g;
  for (const m of p.matchAll(re)) {
    const t = m[0];
    const cls = t.startsWith("$v") ? "rx-var" : t.startsWith("$") ? "rx-ref" : "rx-escape";
    out.push({ from: m.index!, to: m.index! + t.length, cls });
  }
  return out;
}

/** Escapes text for an ICU regular expression. */
export function escapeRegex(s: string): string {
  return s.replace(/[\\^$.*+?()[\]{}|]/g, "\\$&").replace(/\n/g, "\\n").replace(/\t/g, "\\t");
}

/** Escapes text for a literal find field (which understands \t \n \\). */
export function escapeText(s: string): string {
  return s.replace(/\\/g, "\\\\").replace(/\n/g, "\\n").replace(/\t/g, "\\t");
}

const cjk = /[\u3040-\u30ff\u3400-\u4dbf\u4e00-\u9fff\uf900-\ufaff\uac00-\ud7af]/u;

export type Stats = { paragraphs: number; words: number; chars: number; charsNoSpace: number; lines: number };

export function textStats(s: string): Stats {
  let words = 0;
  let chars = 0;
  let noSpace = 0;
  let inWord = false;
  for (const ch of s) {
    chars++;
    const space = /\s/.test(ch);
    if (!space) noSpace++;
    if (cjk.test(ch)) {
      words++;
      inWord = false;
    } else if (/[\p{L}\p{N}_'’-]/u.test(ch)) {
      if (!inWord) words++;
      inWord = true;
    } else inWord = false;
  }
  const lines = s === "" ? 0 : s.split("\n");
  return {
    paragraphs: lines === 0 ? 0 : lines.filter((l) => l.trim() !== "").length,
    words,
    chars,
    charsNoSpace: noSpace,
    lines: lines === 0 ? 0 : lines.length,
  };
}

/** Splits text into plain and invisible-character segments for display. */
export function visibleParts(s: string, max = 240): { text: string; invisible: boolean }[] {
  const out: { text: string; invisible: boolean }[] = [];
  const cut = s.length > max ? s.slice(0, max) : s;
  let buf = "";
  for (const ch of cut) {
    const sym = ch === "\n" ? "⏎" : ch === "\t" ? "⇥" : ch === "\r" ? "␍" : ch === "\u00a0" ? "⍽" : "";
    if (sym) {
      if (buf) out.push({ text: buf, invisible: false });
      buf = "";
      out.push({ text: sym + (ch === "\n" ? "\n" : ""), invisible: true });
    } else buf += ch;
  }
  if (buf) out.push({ text: buf, invisible: false });
  if (s.length > max) out.push({ text: "…", invisible: true });
  return out;
}

export function debounce<A extends unknown[]>(fn: (...a: A) => void, ms: number) {
  let t = 0;
  const d = (...a: A) => {
    clearTimeout(t);
    t = window.setTimeout(() => fn(...a), ms);
  };
  d.cancel = () => clearTimeout(t);
  return d;
}

/** A single family name gets quoted; anything that already looks like a CSS
 * font list ("Menlo, monospace") is used as typed. */
export function fontStack(family: string) {
  const f = family.trim().replace(/;/g, "");
  if (!f) return "var(--mono)";
  return `${/[,"']/.test(f) ? f : JSON.stringify(f)}, var(--mono)`;
}
