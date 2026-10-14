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

// changeHold and changeFade are how long the change highlight stays, then
// how long it takes to fade out. Any edit clears it at once.
const (
	changeHold = 4 * time.Second
	changeFade = 1200 * time.Millisecond
)

// markChanges remembers the spans of the last action for the editor.
func (a *App) markChanges(before, after string) {
	a.changes = changeSpans(before, after)
	a.changesVer = a.ed.Buffer.Version()
	a.changesAt = time.Now()
}

// changeAlpha is how strongly the change highlight shows now.
func (a *App) changeAlpha() float32 {
	if a.freezeChanges {
		return 1
	}
	t := time.Since(a.changesAt)
	if t < changeHold {
		return 1
	}
	return max(0, 1-float32(t-changeHold)/float32(changeFade))
}

// updateHighlights gives the editor the change spans and the find
// panel's matches, for the buffer's current version.
func (a *App) updateHighlights(c *ui.Context, k tokens) {
	st := a.ed
	ver := st.Buffer.Version()
	var hs []editor.Highlight
	if len(a.changes) > 0 && a.changesVer == ver {
		al := a.changeAlpha()
		if al <= 0 {
			a.changes = nil
		} else {
			if al < 1 || time.Since(a.changesAt) < changeHold {
				c.Invalidate()
			}
			fill := k.green.Alpha(map[bool]float32{true: 0.38, false: 0.30}[k.dark] * al)
			line := k.green.Alpha(0.85 * al)
			for _, s := range a.changes {
				if s.Del {
					hs = append(hs, editor.Highlight{Start: s.Start, End: s.End, Color: k.red.Alpha(0.9 * al)})
				} else {
					hs = append(hs, editor.Highlight{Start: s.Start, End: s.End, Color: fill, Outline: line})
				}
			}
		}
	} else if len(a.changes) > 0 {
		a.changes = nil // edited since: drop it
	}
	if a.inspector == "find" && a.find.opts.Find != "" {
		res := a.runFind(a.find.options())
		base, off := a.findBase()
		cur := a.selections()
		for _, m := range byteMatches(base, res.matches, off) {
			h := editor.Highlight{Start: m[0], End: m[1], Color: st.Theme.Match}
			if len(cur) == 1 && cur[0].Start() == m[0] && cur[0].End() == m[1] {
				h.Color, h.Outline = st.Theme.CurrentMatch, st.Theme.CurrentMatch
			}
			hs = append(hs, h)
		}
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
