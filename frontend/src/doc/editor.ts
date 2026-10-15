import { Compartment, EditorState, RangeSetBuilder, StateEffect, StateField, type Extension } from "@codemirror/state";
import {
  Decoration,
  EditorView,
  ViewPlugin,
  WidgetType,
  drawSelection,
  highlightActiveLine,
  highlightActiveLineGutter,
  highlightSpecialChars,
  highlightWhitespace,
  keymap,
  lineNumbers,
  placeholder,
  rectangularSelection,
  type DecorationSet,
  type ViewUpdate,
} from "@codemirror/view";
import { defaultKeymap, history, historyKeymap, indentWithTab } from "@codemirror/commands";
import { fontStack } from "../lib/text";

export type Hit = { from: number; to: number };
export type EditorOptions = {
  lineNumbers: boolean;
  wrap: boolean;
  invisibles: boolean;
  activeLine: boolean;
  fontSize: number;
  zoom: number;
  fontFamily: string;
};

// ---------------------------------------------------------------------------
// Find matches shown in the text

export const setHits = StateEffect.define<{ hits: Hit[]; current: number }>();

const hitMark = Decoration.mark({ class: "cm-hit" });
const currentMark = Decoration.mark({ class: "cm-hit-current" });

export const hitsField = StateField.define<{ hits: Hit[]; current: number; deco: DecorationSet }>({
  create: () => ({ hits: [], current: -1, deco: Decoration.none }),
  update(v, tr) {
    for (const e of tr.effects) {
      if (e.is(setHits)) {
        const b = new RangeSetBuilder<Decoration>();
        const len = tr.state.doc.length;
        e.value.hits.forEach((h, i) => {
          const from = Math.min(h.from, len);
          const to = Math.min(h.to, len);
          if (to > from) b.add(from, to, i === e.value.current ? currentMark : hitMark);
        });
        return { hits: e.value.hits, current: e.value.current, deco: b.finish() };
      }
    }
    if (tr.docChanged) {
      const hits = v.hits.map((h) => ({ from: tr.changes.mapPos(h.from, 1), to: tr.changes.mapPos(h.to, -1) }));
      return { hits, current: v.current, deco: v.deco.map(tr.changes) };
    }
    return v;
  },
  provide: (f) => EditorView.decorations.from(f, (v) => v.deco),
});

// ---------------------------------------------------------------------------
// Invisibles: spaces, tabs and line ends

class EolWidget extends WidgetType {
  toDOM() {
    const s = document.createElement("span");
    s.className = "cm-eol";
    s.setAttribute("aria-hidden", "true");
    return s;
  }
  ignoreEvent() {
    return false;
  }
  eq() {
    return true;
  }
}
const eol = Decoration.widget({ widget: new EolWidget(), side: 1 });

const lineEnds = ViewPlugin.fromClass(
  class {
    decorations: DecorationSet;
    constructor(view: EditorView) {
      this.decorations = this.build(view);
    }
    update(u: ViewUpdate) {
      if (u.docChanged || u.viewportChanged) this.decorations = this.build(u.view);
    }
    build(view: EditorView) {
      const b = new RangeSetBuilder<Decoration>();
      const doc = view.state.doc;
      for (const { from, to } of view.visibleRanges) {
        let pos = from;
        while (pos <= to) {
          const line = doc.lineAt(pos);
          if (line.number < doc.lines) b.add(line.to, line.to, eol);
          pos = line.to + 1;
        }
      }
      return b.finish();
    }
  },
  { decorations: (v) => v.decorations },
);

// ---------------------------------------------------------------------------

const compartments = {
  lineNumbers: new Compartment(),
  wrap: new Compartment(),
  invisibles: new Compartment(),
  activeLine: new Compartment(),
  font: new Compartment(),
};

function fontTheme(o: EditorOptions) {
  const size = Math.round(((o.fontSize || 14) * (o.zoom || 100)) / 100);
  return EditorView.editorAttributes.of({ style: `--editor-font: ${fontStack(o.fontFamily)}; --editor-size: ${size}px` });
}

function configure(o: EditorOptions) {
  return {
    lineNumbers: o.lineNumbers ? [lineNumbers(), highlightActiveLineGutter()] : [],
    wrap: o.wrap ? EditorView.lineWrapping : [],
    invisibles: o.invisibles ? [highlightWhitespace(), lineEnds] : [],
    activeLine: o.activeLine ? highlightActiveLine() : [],
    font: fontTheme(o),
  } satisfies Record<keyof typeof compartments, Extension>;
}

export function createEditor(parent: HTMLElement, doc: string, o: EditorOptions, onUpdate: (u: ViewUpdate) => void, extra: Extension[] = []) {
  const c = configure(o);
  return new EditorView({
    parent,
    state: EditorState.create({
      doc,
      extensions: [
        history(),
        drawSelection(),
        rectangularSelection(),
        highlightSpecialChars(),
        EditorState.allowMultipleSelections.of(true),
        keymap.of([...defaultKeymap, ...historyKeymap, indentWithTab]),
        placeholder("在这里输入或粘贴文本…"),
        EditorView.contentAttributes.of({ spellcheck: "false", autocorrect: "off", autocapitalize: "off" }),
        hitsField,
        compartments.lineNumbers.of(c.lineNumbers),
        compartments.wrap.of(c.wrap),
        compartments.invisibles.of(c.invisibles),
        compartments.activeLine.of(c.activeLine),
        compartments.font.of(c.font),
        EditorView.updateListener.of(onUpdate),
        ...extra,
      ],
    }),
  });
}

export function reconfigure(view: EditorView, o: EditorOptions) {
  const c = configure(o);
  view.dispatch({
    effects: (Object.keys(compartments) as (keyof typeof compartments)[]).map((k) => compartments[k].reconfigure(c[k])),
  });
}
