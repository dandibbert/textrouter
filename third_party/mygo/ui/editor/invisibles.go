package editor

import (
	"unicode/utf8"

	"github.com/egoist/mygo/ui"
)

// InvisibleKind is what kind of invisible character a mark shows.
// (TextRouter fork.)
type InvisibleKind uint8

const (
	InvisibleNone InvisibleKind = iota
	InvisibleNBSP
	InvisibleIdeographicSpace
	InvisibleZeroWidth
	InvisibleCR
)

// InvisibleOf says whether r is a character Invisibles marks.
func InvisibleOf(r rune) InvisibleKind {
	switch r {
	case '\u00a0', '\u202f', '\u2007':
		return InvisibleNBSP
	case '\u3000':
		return InvisibleIdeographicSpace
	case '\u200b', '\u200c', '\u200d', '\u2060', '\ufeff', '\u00ad':
		return InvisibleZeroWidth
	case '\r':
		return InvisibleCR
	}
	return InvisibleNone
}

// InvisibleMark is where a mark is drawn over a row: X and W are the
// character's own place, W 0 for a character of no width, so marks never
// take room of their own.
type InvisibleMark struct {
	Kind InvisibleKind
	X, W float32
}

// invisibleMarks returns the marks of a row, x relative to the text's
// left edge.
func (ed *editor) invisibleMarks(l *lineLayout, row *rowLayout, dst []InvisibleMark) []InvisibleMark {
	v := &ed.view
	s := l.text
	a, b := l.fromDisp(row.from), l.fromDisp(row.to)
	for i := a; i < b; {
		r, size := utf8.DecodeRuneInString(s[i:])
		k := InvisibleOf(r)
		if k != InvisibleNone {
			d0, d1 := l.toDisp(i, false), l.toDisp(i+size, false)
			if d0 >= row.from && d0 <= row.to {
				x0 := v.xOf(l, row, d0)
				x1 := v.xOf(l, row, min(d1, row.to))
				dst = append(dst, InvisibleMark{Kind: k, X: x0, W: max(x1-x0, 0)})
			}
		}
		i += size
	}
	return dst
}

// paintInvisibles draws the marks of Invisibles over a row.
func (ed *editor) paintInvisibles(p *ui.Painter, l *lineLayout, row *rowLayout, ox, y float32, lastRow bool, line int) {
	v := &ed.view
	ct := ed.theme
	lh := v.lineHeight
	cy := y + lh/2
	cw := v.charWidth
	soft := ct.Whitespace
	warn := ct.SeverityColor(SeverityWarning)
	for _, m := range ed.invisibleMarks(l, row, nil) {
		x0, x1 := ox+m.X, ox+m.X+m.W
		switch m.Kind {
		case InvisibleNBSP:
			// ⍽: an open box under the middle of the space.
			w := min(m.W-2, cw*0.7)
			cx := (x0 + x1) / 2
			var path ui.Path
			b := cy + lh*0.16
			path.MoveTo(cx-w/2, b-3).LineTo(cx-w/2, b).LineTo(cx+w/2, b).LineTo(cx+w/2, b-3)
			p.StrokePath(&path, 1.2, warn)
		case InvisibleIdeographicSpace:
			// □: a square inside the full-width space.
			sz := min(m.W-4, lh*0.5)
			p.Stroke(ui.Rect{X: (x0+x1)/2 - sz/2, Y: cy - sz/2, W: sz, H: sz}, soft, 1, 2)
		case InvisibleZeroWidth:
			// ‸: a caret-like tick standing on the character's place.
			var path ui.Path
			t := y + lh*0.18
			path.MoveTo(x0, t).LineTo(x0, y+lh*0.82)
			path.MoveTo(x0-3, t).LineTo(x0, t+3).LineTo(x0+3, t)
			p.StrokePath(&path, 1.2, warn)
		case InvisibleCR:
			// ␍: a short return arrow in the character's own place.
			ed.paintReturn(p, x0+max(m.W, cw*0.8)/2, cy, cw*0.35, warn)
		}
	}
	// ¬ after the text of a line's last row, past its end: no room taken.
	if lastRow && line < ed.buf().LineCount()-1 {
		x := ox + row.indent + row.width + 2
		h := lh * 0.18
		var path ui.Path
		path.MoveTo(x, cy-h/2).LineTo(x+cw*0.55, cy-h/2).LineTo(x+cw*0.55, cy+h)
		p.StrokePath(&path, 1.2, soft)
		if ed.buf().LineEnding() == "\r\n" {
			ed.paintReturn(p, x+cw*1.2, cy, cw*0.3, soft)
		}
	}
}

func (ed *editor) paintReturn(p *ui.Painter, cx, cy, r float32, c ui.Color) {
	var path ui.Path
	path.MoveTo(cx+r, cy-r).LineTo(cx+r, cy+r*0.4).LineTo(cx-r, cy+r*0.4)
	path.MoveTo(cx-r+2.5, cy+r*0.4-2.5).LineTo(cx-r, cy+r*0.4).LineTo(cx-r+2.5, cy+r*0.4+2.5)
	p.StrokePath(&path, 1.1, c)
}

// InvisibleMarks returns the marks Invisibles draws in line i, in DIPs
// from the text's left edge, for tests: each lies in its character's own
// place. (TextRouter fork.)
func (st *State) InvisibleMarks(i int) []InvisibleMark {
	ed := &st.ed
	l := ed.view.line(i)
	var out []InvisibleMark
	for ri := range l.rows {
		out = ed.invisibleMarks(l, &l.rows[ri], out)
	}
	return out
}

// RuneX returns the x of the byte offset off of line i from the text's
// left edge, as laid out, for tests. (TextRouter fork.)
func (st *State) RuneX(i, off int) float32 {
	ed := &st.ed
	l := ed.view.line(i)
	d := l.toDisp(off, false)
	row := &l.rows[l.rowOf(d, false)]
	return ed.view.xOf(l, row, d)
}
