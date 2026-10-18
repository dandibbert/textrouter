package gui

import (
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// A preview is what the body shows in place of the text while the pointer is
// on a cleaner or a pattern, or while find/replace has matches: the text with
// what will go struck through, what will come inserted, and what is only
// matched marked. It is built from engine.Diff and engine.FindOptions.FindAll
// and cached for as long as the text does not change.

type spanKind int

const (
	spDel  spanKind = iota // will go: Del with a strike
	spIns                  // will come: Ins
	spHit                  // matched, not replaced: Hit with an underline
	spSkip                 // match the user unchecked: left as it is
)

type pvSpan struct {
	start, end int // bytes of the merged text
	kind       spanKind
	cur        bool // part of the current match
}

type markKind int

const (
	markDel markKind = iota + 1
	markAdd
	markHit
)

type preview struct {
	key     string
	label   string // the name in the status bar
	delta   string // "−3 行" / "5 处"
	hitMode bool   // only matches, as when previewing a pattern
	merged  string
	spans   []pvSpan
	marks   map[int]markKind // merged line → change bar
	ticks   []int            // merged lines to tick on the scroll bar
	curLine int              // line of the current match, -1 none
	lines   int
	at      time.Time // when it began showing
}

// visibleDel makes what a deleted run held visible: spaces as ·, line breaks
// as ⏎, tabs as →.
var visibleDel = strings.NewReplacer("\n", "⏎", " ", "·", "\t", "→")

// pvBuilder builds the merged text of a preview.
type pvBuilder struct {
	b         strings.Builder
	line      int
	spans     []pvSpan
	marks     map[int]markKind
	tickLines []int
	curLine   int
}

func newPvBuilder() *pvBuilder { return &pvBuilder{marks: map[int]markKind{}, curLine: -1} }

func (p *pvBuilder) mark(line int, k markKind) {
	// A line with anything deleted shows the deletion colour.
	if old, ok := p.marks[line]; ok && old < k {
		return
	}
	p.marks[line] = k
}

func (p *pvBuilder) same(s string) {
	p.b.WriteString(s)
	p.line += strings.Count(s, "\n")
}

func (p *pvBuilder) add(s string, k spanKind, cur bool, mk markKind) {
	if s == "" {
		return
	}
	if k == spDel {
		s = visibleDel.Replace(s)
	}
	start := p.b.Len()
	p.b.WriteString(s)
	p.spans = append(p.spans, pvSpan{start: start, end: p.b.Len(), kind: k, cur: cur})
	if mk != 0 {
		for i, n := 0, strings.Count(s, "\n"); i <= n; i++ {
			p.mark(p.line+i, mk)
		}
	}
	p.line += strings.Count(s, "\n")
}

func (p *pvBuilder) finish(key string) *preview {
	pv := &preview{key: key, merged: p.b.String(), spans: p.spans, marks: p.marks, curLine: -1, lines: p.line + 1}
	return pv
}

// diffPreview shows the change from before to after.
func diffPreview(key string, segs []engine.Seg) *preview {
	p := newPvBuilder()
	for _, s := range segs {
		switch s.Op {
		case '=':
			p.same(s.Text)
		case '-':
			p.add(s.Text, spDel, false, markDel)
		case '+':
			p.add(s.Text, spIns, false, markAdd)
		}
	}
	pv := p.finish(key)
	for l := range pv.marks {
		pv.ticks = append(pv.ticks, l)
	}
	sort.Ints(pv.ticks)
	return pv
}

// findPreview shows the matches of a find/replace in text: each struck
// through with its replacement after it, unless the box is unchecked (skip) or
// only matching (hitOnly). ms are in runes of text.
func findPreview(key, text string, ms []engine.Match, skip func(int) bool, hitOnly bool, cur int) *preview {
	p := newPvBuilder()
	r := []rune(text)
	last := 0
	for i, m := range ms {
		if m.Start < last || m.End > len(r) {
			continue
		}
		p.same(string(r[last:m.Start]))
		old := string(r[m.Start:m.End])
		isCur := i == cur
		startLine := p.line
		switch {
		case skip != nil && skip(i):
			p.add(old, spSkip, isCur, 0)
		case hitOnly:
			p.add(old, spHit, isCur, markHit)
		default:
			p.add(old, spDel, isCur, markDel)
			p.add(m.Result, spIns, isCur, markDel)
		}
		p.ticks(startLine, i, cur)
		last = m.End
	}
	p.same(string(r[last:]))
	pv := p.finish(key)
	pv.hitMode = hitOnly
	pv.ticks, pv.curLine = p.tickLines, p.curLine
	return pv
}

// ticks notes the line of match i for the scroll bar.
func (p *pvBuilder) ticks(line, i, cur int) {
	p.tickLines = append(p.tickLines, line)
	if i == cur {
		p.curLine = line
	}
}

// pvEditor keeps the read-only editor that shows previews.
type pvEditor struct {
	st  *editor.State
	key string
}

// state returns the editor showing pv's text, made again when the text of the
// preview changed.
func (a *App) pvState(pv *preview) *editor.State {
	pe := &a.ws.pvEd
	id := hashOf(pv.key, pv.merged)
	if pe.st == nil || pe.key != fmt.Sprint(id) {
		var off float64
		if a.ed != nil {
			off = a.ed.ScrollOffset()
		}
		pe.st = &editor.State{Buffer: editor.NewBuffer(pv.merged), ReadOnly: true}
		pe.key = fmt.Sprint(id)
		pe.st.SetScrollOffset(off)
	}
	return pe.st
}

// previewHighlights colours a preview's spans. p is how far its entrance has
// come: insertions wipe in from the left, deletions fade in.
func previewHighlights(pv *preview, p float32, ring float32, ringA float32) []editor.Highlight {
	hs := make([]editor.Highlight, 0, len(pv.spans))
	for _, s := range pv.spans {
		h := editor.Highlight{Start: s.start, End: s.end}
		switch s.kind {
		case spDel:
			h.Color, h.Strike = Del.Alpha(p), Ink.Alpha(p)
		case spIns:
			h.Color = Ins
			if p < 1 {
				h.Wipe = max(p, 0.001)
			}
		case spHit:
			h.Color, h.Underline = Hit.Alpha(p), Ink.Alpha(p)
		case spSkip:
			h.Underline = SkipLine
		}
		if s.cur {
			h.Outline = Ink
			if ring > 0 {
				h.Ring, h.RingColor = ring, Focus.Alpha(ringA)
			}
		}
		hs = append(hs, h)
	}
	return hs
}

// lineOf returns the number of line breaks before byte off of s.
func lineOf(s string, off int) int {
	if off > len(s) {
		off = len(s)
	}
	return strings.Count(s[:off], "\n")
}

// ---------------------------------------------------------------- caches

// textKey identifies the text for caches: it changes when the text does.
func (a *App) textKey() uint64 { return a.ws.textKey }

func (a *App) refreshCaches() {
	k := hashOf(a.Text)
	if k != a.ws.textKey || a.ws.cache == nil {
		a.ws.textKey = k
		a.ws.cache = map[string]any{}
		a.ws.selLines = nil
	}
}

// actionResult is what applying action id would make of the whole text: the
// result, the diff and the preview of it. It is computed once per text.
func (a *App) actionResult(id string) *actionRes {
	if v, ok := a.ws.cache["a:"+id]; ok {
		return v.(*actionRes)
	}
	r := &actionRes{}
	label, fn := a.resolveAction(id)
	r.label = label
	if fn != nil {
		r.after = fn(a.Text)
		if r.after != a.Text {
			if engine.RuneLen(a.Text)+engine.RuneLen(r.after) > 600000 {
				r.big = true
				r.delta = "有改动"
			} else {
				r.segs = engine.Diff(a.Text, r.after)
				r.delta = deltaText(a.Text, r.after, r.segs)
			}
		}
	}
	a.ws.cache["a:"+id] = r
	return r
}

type actionRes struct {
	label, after, delta string
	segs                []engine.Seg
	big                 bool
	pv                  *preview
}

// changed tells whether the action would change the text.
func (r *actionRes) changed() bool { return r.delta != "" }

// preview returns the preview of the action, built once.
func (r *actionRes) preview(id string) *preview {
	if r.pv == nil && r.segs != nil {
		r.pv = diffPreview("a:"+id, r.segs)
		r.pv.label, r.pv.delta = r.label, r.delta
	}
	return r.pv
}

// deltaText is the badge of a change: "−N 行" when it removes lines, else the
// number of separate changed blocks as "N 处".
func deltaText(before, after string, segs []engine.Seg) string {
	lb, la := strings.Count(before, "\n")+1, strings.Count(after, "\n")+1
	if la < lb {
		return fmt.Sprintf("−%d 行", lb-la)
	}
	n, prev := 0, byte('=')
	for _, s := range segs {
		if s.Op != '=' && prev == '=' {
			n++
		}
		prev = s.Op
	}
	if n == 0 {
		return ""
	}
	return fmt.Sprintf("%d 处", n)
}

var _ = utf8.RuneLen
var _ = ui.Transparent
