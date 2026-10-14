import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import { Compartment, EditorState, RangeSetBuilder } from "@codemirror/state";
import { Decoration, EditorView, ViewPlugin, keymap, placeholder as cmPlaceholder, type DecorationSet, type ViewUpdate } from "@codemirror/view";
import { defaultKeymap, history, historyKeymap } from "@codemirror/commands";
import { tokenizeRegex, tokenizeTemplate, type Token } from "./text";

export type FieldMode = "regex" | "text" | "template" | "template-text";

export type PatternFieldHandle = {
  focus: () => void;
  selectAll: () => void;
  insert: (text: string, cursorOffset?: number) => void;
};

function tokens(mode: FieldMode, s: string): Token[] {
  switch (mode) {
    case "regex":
      return tokenizeRegex(s);
    case "template":
      return tokenizeTemplate(s, true);
    case "template-text":
    case "text":
      return tokenizeTemplate(s, false);
  }
}

function highlighter(mode: FieldMode) {
  const build = (view: EditorView): DecorationSet => {
    const b = new RangeSetBuilder<Decoration>();
    const s = view.state.doc.toString();
    const ts = tokens(mode, s).sort((a, b) => a.from - b.from);
    let last = 0;
    for (const t of ts) {
      if (t.from < last) continue;
      b.add(t.from, t.to, Decoration.mark({ class: t.cls }));
      last = t.to;
    }
    return b.finish();
  };
  return ViewPlugin.fromClass(
    class {
      decorations: DecorationSet;
      constructor(view: EditorView) {
        this.decorations = build(view);
      }
      update(u: ViewUpdate) {
        if (u.docChanged) this.decorations = build(u.view);
      }
    },
    { decorations: (v) => v.decorations },
  );
}

/** A one-to-few-line code field with regex/template syntax colors. Enter
 * runs onEnter; Shift+Enter (or Option+Enter) inserts a line break. */
export const PatternField = forwardRef<PatternFieldHandle, {
  value: string;
  onChange: (v: string) => void;
  mode: FieldMode;
  placeholder?: string;
  onEnter?: () => void;
  onFocus?: () => void;
  error?: boolean;
  ariaLabel?: string;
}>(function PatternField({ value, onChange, mode, placeholder, onEnter, onFocus, error, ariaLabel }, ref) {
  const host = useRef<HTMLDivElement>(null);
  const view = useRef<EditorView | null>(null);
  const hl = useRef(new Compartment());
  const ph = useRef(new Compartment());
  const cb = useRef({ onChange, onEnter, onFocus });
  cb.current = { onChange, onEnter, onFocus };
  const [focused, setFocused] = useState(false);

  useEffect(() => {
    const v = new EditorView({
      parent: host.current!,
      state: EditorState.create({
        doc: value,
        extensions: [
          history(),
          keymap.of([
            { key: "Enter", run: () => (cb.current.onEnter?.(), true) },
            { key: "Shift-Enter", run: (v) => (v.dispatch(v.state.replaceSelection("\n")), true) },
            { key: "Alt-Enter", run: (v) => (v.dispatch(v.state.replaceSelection("\n")), true) },
            { key: "Tab", run: () => false },
            ...historyKeymap,
            ...defaultKeymap,
          ]),
          EditorView.lineWrapping,
          hl.current.of(highlighter(mode)),
          ph.current.of(cmPlaceholder(placeholder ?? "")),
          EditorView.contentAttributes.of({ "aria-label": ariaLabel ?? "", spellcheck: "false", autocorrect: "off", autocapitalize: "off" }),
          EditorView.updateListener.of((u) => {
            if (u.docChanged) cb.current.onChange(u.state.doc.toString());
            if (u.focusChanged) {
              setFocused(u.view.hasFocus);
              if (u.view.hasFocus) cb.current.onFocus?.();
            }
          }),
        ],
      }),
    });
    view.current = v;
    return () => v.destroy();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  useEffect(() => {
    view.current?.dispatch({ effects: hl.current.reconfigure(highlighter(mode)) });
  }, [mode]);
  useEffect(() => {
    view.current?.dispatch({ effects: ph.current.reconfigure(cmPlaceholder(placeholder ?? "")) });
  }, [placeholder]);
  useEffect(() => {
    const v = view.current;
    if (v && v.state.doc.toString() !== value) {
      v.dispatch({ changes: { from: 0, to: v.state.doc.length, insert: value }, selection: { anchor: value.length } });
    }
  }, [value]);

  useImperativeHandle(ref, () => ({
    focus: () => view.current?.focus(),
    selectAll: () => {
      const v = view.current;
      if (!v) return;
      v.focus();
      v.dispatch({ selection: { anchor: 0, head: v.state.doc.length } });
    },
    insert: (text, cursorOffset) => {
      const v = view.current;
      if (!v) return;
      const { from, to } = v.state.selection.main;
      const sel = v.state.sliceDoc(from, to);
      // "( )" style snippets wrap the selection.
      let ins = text;
      let caret = from + (cursorOffset ?? text.length);
      if (cursorOffset !== undefined && sel) {
        ins = text.slice(0, cursorOffset) + sel + text.slice(cursorOffset);
        caret = from + ins.length;
      }
      v.dispatch({ changes: { from, to, insert: ins }, selection: { anchor: caret } });
      v.focus();
    },
  }));

  return <div className={"field-box" + (focused ? " focused" : "") + (error ? " error" : "")} ref={host} />;
});
