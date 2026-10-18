package gui

import (
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// changeSpan is a run of text an action changed, in byte offsets of the
// text after it. Del marks a place where text was only removed
// (Start == End).
type changeSpan struct {
	Start, End int
	Del        bool
}

// changeSpans returns the characters that differ between before and
// after, at character level, as spans of after. Equal runs of one or two
// letters between changes are folded into the change so a reworded word
// highlights as one span rather than as confetti.
func changeSpans(before, after string) []changeSpan {
	if before == after {
		return nil
	}
	segs := cleanupSegs(engine.Diff(before, after))
	var out []changeSpan
	pos := 0
	for _, s := range segs {
		switch s.Op {
		case '=':
			pos += len(s.Text)
		case '+':
			n := len(out)
			if n > 0 && out[n-1].End == pos {
				out[n-1].End, out[n-1].Del = pos+len(s.Text), false
			} else {
				out = append(out, changeSpan{Start: pos, End: pos + len(s.Text)})
			}
			pos += len(s.Text)
		case '-':
			if n := len(out); n == 0 || out[n-1].End != pos {
				out = append(out, changeSpan{Start: pos, End: pos, Del: true})
			}
		}
	}
	return out
}

// cleanupSegs folds short equal runs (≤ 2 runes, letters or digits)
// sandwiched between changes into those changes.
func cleanupSegs(segs []engine.Seg) []engine.Seg {
	if len(segs) < 3 {
		return segs
	}
	var out []engine.Seg
	for i := 0; i < len(segs); i++ {
		s := segs[i]
		if s.Op == '=' && i > 0 && i+1 < len(segs) && utf8.RuneCountInString(s.Text) <= 2 && wordy(s.Text) {
			out = append(out, engine.Seg{Op: '-', Text: s.Text}, engine.Seg{Op: '+', Text: s.Text})
			continue
		}
		out = append(out, s)
	}
	// Order each changed stretch as all deletions, then all insertions.
	var res []engine.Seg
	for i := 0; i < len(out); {
		if out[i].Op == '=' {
			res = append(res, out[i])
			i++
			continue
		}
		del, ins := "", ""
		for ; i < len(out) && out[i].Op != '='; i++ {
			if out[i].Op == '-' {
				del += out[i].Text
			} else {
				ins += out[i].Text
			}
		}
		if del != "" {
			res = append(res, engine.Seg{Op: '-', Text: del})
		}
		if ins != "" {
			res = append(res, engine.Seg{Op: '+', Text: ins})
		}
	}
	return res
}

func wordy(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return false
		}
	}
	return true
}

// changeFade is how long the new text's highlight takes to fade after an
// action (§6 "点击应用": 1.3s); for the first 35% of it the text is boxed in
// a 2px black frame. Any edit clears it at once.
const changeFade = dApplyNew

// markChanges remembers the spans of the last action for the editor.
func (a *App) markChanges(before, after string) {
	a.changes = changeSpans(before, after)
	a.changesVer = a.ed.Buffer.Version()
	a.changesAt = time.Now()
}

// changeProgress is how far the fade of the change highlight has come, 0 to 1.
func (a *App) changeProgress(c *ui.Context) float32 {
	if a.freezeChanges {
		return 0
	}
	if c.Preferences().ReduceMotion {
		return 1
	}
	return min(1, float32(time.Since(a.changesAt))/float32(changeFade))
}

// updateHighlights gives the editor the spans the last action changed, for
// the buffer's current version: new text is green and boxed in black at
// first, then fades away; deleted places show a thin bar.
func (a *App) updateHighlights(c *ui.Context, k tokens) {
	st := a.ed
	ver := st.Buffer.Version()
	var hs []editor.Highlight
	if len(a.changes) > 0 && a.changesVer == ver {
		p := a.changeProgress(c)
		if p >= 1 {
			a.changes = nil
		} else {
			if !a.freezeChanges {
				c.Invalidate()
			}
			fill := Ins.Alpha(1 - p)
			var frame ui.Color
			if p < 0.35 {
				frame = Ink
			}
			for _, s := range a.changes {
				if s.Del {
					hs = append(hs, editor.Highlight{Start: s.Start, End: s.End, Color: MarkDel.Alpha(1 - p)})
				} else {
					hs = append(hs, editor.Highlight{Start: s.Start, End: s.End, Color: fill, Outline: frame})
				}
			}
		}
	} else if len(a.changes) > 0 {
		a.changes = nil // edited since: drop it
	}
	st.Highlights, st.HighlightVersion = hs, ver
}

// byteMatches turns rune-offset matches of base into byte offsets of the
// document, base starting at off.
func byteMatches(base string, ms []engine.Match, off int) [][2]int {
	if len(ms) == 0 {
		return nil
	}
	idx := make([]int, 0, utf8.RuneCountInString(base)+1)
	for i := range base {
		idx = append(idx, i)
	}
	idx = append(idx, len(base))
	out := make([][2]int, 0, len(ms))
	for _, m := range ms {
		if m.Start < 0 || m.End >= len(idx) || m.Start > m.End {
			continue
		}
		out = append(out, [2]int{off + idx[m.Start], off + idx[m.End]})
	}
	return out
}
