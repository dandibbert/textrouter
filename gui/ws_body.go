package gui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// bodyFont returns the text's font and line height multiple: 13px on 25px
// lines unless the font settings were changed from the older defaults.
func (a *App) bodyFont() (ui.Font, float32) {
	vs := &a.Lib.Settings.View
	size, lh := float32(sizeBody), float32(lineBody)/sizeBody
	if (vs.FontSize != 15 && vs.FontSize > 0) || (vs.LineHeight != 1.6 && vs.LineHeight > 0) {
		size, lh = float32(vs.FontSize), float32(vs.LineHeight)
	}
	return ui.Font{Family: a.editorFamily(), Size: size * a.view.zoom}, lh
}

// editorTheme colours the text editor from the tokens.
func editorTheme(c *ui.Context, k tokens) *editor.Theme {
	th := editor.DefaultTheme(c.Theme())
	th.Background, th.Text = Paper, Ink
	th.LineNumber, th.ActiveLineNumber = GutterNum, Ink
	th.Selection, th.InactiveSelection = Accent, Accent.Alpha(0.6)
	th.Cursor = Ink
	th.Match, th.CurrentMatch = Hit, Accent
	th.Whitespace = Faint
	th.Added, th.Deleted, th.Modified = MarkAdd, MarkDel, Ink
	th.ScrollbarOutline, th.ScrollbarThumb, th.ScrollbarHover, th.ScrollbarDrag = Ink, Accent, AccentDeep, Ink
	th.ScrollbarWidth, th.ScrollbarMin = scrollW+6, 44
	return th
}

// configureEditor sets the options of the text's editor (and of the preview
// editor, which has no gutter clicks) for this frame.
func (a *App) configureEditor(c *ui.Context, k tokens) {
	a.configureState(c, k, a.ed, false)
}

func (a *App) configureState(c *ui.Context, k tokens, st *editor.State, preview bool) {
	v := &a.view
	if v.zoom <= 0 {
		v.zoom = 1
	}
	st.Font, st.LineHeight = a.bodyFont()
	st.SoftWrap = true
	st.LineNumbers = editor.LineNumbersOff
	if v.lineNumbers {
		st.LineNumbers = editor.LineNumbersOn
	}
	st.Whitespace = editor.WhitespaceSelection
	if v.invisibles {
		st.Whitespace = editor.WhitespaceAll
	}
	st.Invisibles = v.invisibles
	st.NoIndentGuides, st.NoFolding, st.NoActiveLine = true, true, true
	st.TabSize = 4
	st.Placeholder = "剪贴板是空的。粘贴、输入，或把文本文件拖进来…"
	st.Label = "剪贴板文本"
	st.Theme = editorTheme(c, k)
	st.HighlightInset = 4
	st.ReadOnly = preview
	st.OnGutterClick = nil
	if !preview {
		st.OnGutterClick = a.gutterClick
		st.Bindings = []editor.Binding{
			{Mods: ui.Cmd, Key: ui.KeyK, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyF, Command: editor.NoCommand},
			{Mods: ui.Cmd | ui.Alt, Key: ui.KeyF, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyG, Command: editor.NoCommand},
			{Mods: ui.Cmd | ui.Shift, Key: ui.KeyG, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.Key1, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyZ, Command: editor.NoCommand},
			{Mods: ui.Cmd | ui.Shift, Key: ui.KeyZ, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyEnter, Command: editor.NoCommand},
			{Mods: ui.Cmd | ui.Alt, Key: ui.KeyEnter, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyBackslash, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyEqual, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyMinus, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.Key0, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyO, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyS, Command: editor.NoCommand},
			{Mods: ui.Cmd, Key: ui.KeyL, Command: editor.GoToLine},
		}
		st.RenderFindBar = nil
		st.RenderGoToLine = a.renderGoToLine
	}
}

// gutterStyle is the gutter of the 新粗野 look: width by digits, a 2px rule at
// its right, bars for changed lines, the current match's row and the rows
// picked by clicking their numbers.
func (a *App) gutterStyle(pv *preview, barP float32, curLine int) editor.GutterStyle {
	g := editor.GutterStyle{
		On: true, DigitWidth: gutterDigW, Pad: gutterPad, MinDigits: 2, NumberRight: 7, TextGap: 14,
		NumberSize: sizeSmall, NumberFont: fontMono, Background: GutterBg, Border: Ink, BorderWidth: strokeW, Hover: Ink,
	}
	if pv != nil {
		for l, m := range pv.marks {
			col := MarkDel
			switch m {
			case markAdd:
				col = MarkAdd
			case markHit:
				col = Ink
			}
			g.Bars = append(g.Bars, editor.GutterBar{Line: l, Color: col, Grow: barP})
		}
	}
	for l := range a.ws.selLines {
		g.Lines = append(g.Lines, editor.GutterLine{Line: l, Fill: AccentSoft})
	}
	if curLine >= 0 {
		g.Lines = append(g.Lines, editor.GutterLine{Line: curLine, Fill: AccentWash, NumberFill: Accent, Edge: Ink, NumberTxt: Ink, Bold: true})
	}
	return g
}

// gutterClick picks a line by its number, and drops it when clicked again; the
// picked lines are what the editor selects, so that actions work on them.
func (a *App) gutterClick(line int) {
	if a.ws.selLines == nil {
		a.ws.selLines = map[int]bool{}
	}
	if a.ws.selLines[line] {
		delete(a.ws.selLines, line)
	} else {
		a.ws.selLines[line] = true
	}
	var sels []editor.Selection
	for l := 0; l < a.ed.Buffer.LineCount(); l++ {
		if !a.ws.selLines[l] {
			continue
		}
		s := a.ed.Buffer.Offset(editor.Pos{Line: l})
		e := s + len(a.ed.Buffer.Line(l))
		if l+1 < a.ed.Buffer.LineCount() {
			e++
		}
		sels = append(sels, editor.Selection{Anchor: s, Head: e})
	}
	if len(sels) > 0 {
		a.ed.SetSelections(sels...)
	} else {
		p := a.ed.Cursor()
		a.ed.Select(p, p)
	}
}

// shownPreview decides what the body shows instead of the text, if anything.
func (a *App) shownPreview() (*preview, float32) {
	w := &a.ws
	if fx := w.applyFx; fx != nil {
		return fx.pv, 0
	}
	if w.tab == 0 && w.hov != "" {
		if r := a.actionResult(w.hov); r.changed() && r.segs != nil {
			return r.preview(w.hov), 0
		}
	}
	if w.tab == 1 {
		if pv := a.findShown(); pv != nil {
			return pv, 0
		}
	}
	return nil, 0
}

// bodyCard is the white card holding the text, its scroll-bar ticks and the
// status bar.
func (a *App) bodyCard(c *ui.Context, k tokens) {
	a.syncEditor()
	w := &a.ws
	c.OnShortcut(ui.Cmd, ui.KeyEnter, a.done)
	c.OnShortcut(ui.Cmd, ui.KeyEqual, func() { a.zoomBy(1) })
	c.OnShortcut(ui.Cmd, ui.KeyMinus, func() { a.zoomBy(-1) })
	c.OnShortcut(ui.Cmd, ui.Key0, func() { a.view.zoom = 1 })
	c.OnShortcut(ui.Cmd, ui.KeyO, a.openFile)
	c.OnShortcut(ui.Cmd, ui.KeyS, a.saveFile)
	c.OnShortcut(ui.Cmd|ui.Alt, ui.KeyI, func() { a.view.invisibles = !a.view.invisibles })

	pv, _ := a.shownPreview()
	if pv != nil && w.pvShown != pv.key+fmt.Sprint(w.textKey) {
		// A preview begins (or changes to another): its entrance plays again.
		w.pvShown, w.pvAt = pv.key+fmt.Sprint(w.textKey), time.Now()
	}
	if pv == nil {
		w.pvShown = ""
	}
	entered := anim(c, w.pvAt, dPreview, EaseOut)
	barP := anim(c, w.pvAt, dBar, EaseOut)

	st := a.ed
	curLine := -1
	var jump float32 = 1
	if pv != nil {
		st = a.pvState(pv)
		a.configureState(c, k, st, true)
		fx := w.applyFx
		p := entered
		if fx != nil {
			p = 1 - anim(c, fx.at, dApplyGone, ui.CubicBezier(.4, 0, .2, 1))
		}
		var ring, ringA float32
		if w.tab == 1 {
			jump = anim(c, w.find.jumpAt, dJump, EaseOut)
			if jump < 1 {
				ring, ringA = 12*jump, 0.7*(1-jump)
			}
		}
		st.Highlights, st.HighlightVersion = previewHighlights(pv, p, ring, ringA), st.Buffer.Version()
		if w.tab == 1 {
			curLine = pv.curLine
		}
		st.Gutter = a.gutterStyle(pv, barP, curLine)
		a.smoothScroll(c, st, pv)
	} else {
		a.configureEditor(c, k)
		a.updateHighlights(c, k)
		st.Gutter = a.gutterStyle(nil, 1, -1)
	}

	zone := ui.Column(c.Key(fmt.Sprintf("docview%d", a.cur))).Grow(1).MinWidth(0).FillHeight()
	if files := zone.DroppedFiles(); len(files) > 0 {
		for _, f := range files {
			a.loadFile(f)
		}
	}
	hard(boxed(zone.Background(Paper).Radius(radiusCtl).Clip()), shadowMd).Children(func() {
		ui.Box(c).Grow(1).MinHeight(0).Children(func() {
			ev := editor.View(c, st).Absolute().Top(0).Left(0).Right(0).Bottom(0)
			if pv == nil {
				ev.AutoFocus()
			}
			a.ticksView(c, st, pv, curLine)
		})
		a.statusBarView(c, k, pv)
	})
	if pv == nil {
		a.syncEditor()
	}
}

// ticksView draws the marks left of the scroll bar: one per match or changed
// line, the current match as a bigger yellow square that slides to the next.
func (a *App) ticksView(c *ui.Context, st *editor.State, pv *preview, cur int) {
	if pv == nil || len(pv.ticks) == 0 {
		return
	}
	w := &a.ws
	h := st.ViewHeight() - 24
	if h < 20 {
		return
	}
	n := max(pv.lines-1, 1)
	pos := func(line int) float32 { return float32(line) / float32(n) * h }
	// No marks while hovering a long list's rows would be noise: thin them to
	// what fits (one per 3px).
	last := float32(-10)
	ui.Box(c.Key("ticks")).Absolute().Top(12).Bottom(12).Right(18).Width(tickColW).PassThrough().Children(func() {
		for i, l := range pv.ticks {
			y := pos(l)
			if y-last < 3 && l != cur {
				continue
			}
			last = y
			delay := min(time.Duration(i)*dTickStep, dTickMax)
			p := anim(c, w.pvAt.Add(delay), 300*time.Millisecond, EaseOut)
			col, th := Ink, float32(3)
			if pv.hitMode == false && w.tab == 0 {
				col, th = MarkDel, 4
			}
			ui.Box(c.Key(fmt.Sprint("tick", i))).Absolute().Left(1).Top(y).Size(6, th).Radius(1).Background(col).Scale(p, 1).Opacity(p)
		}
		if w.tab == 1 && cur >= 0 {
			ty := w.find.tickY
			target := pos(cur)
			if ty == nil {
				ty = new(float32)
				*ty = target
				w.find.tickY = ty
			}
			slide := anim(c, w.find.jumpAt, dTickSlide, EaseOut)
			*ty = w.find.tickFrom + (target-w.find.tickFrom)*slide
			if slide >= 1 {
				w.find.tickFrom = target
			}
			ui.Box(c.Key("tick-cur")).Absolute().Left(-2).Top(*ty-2).Size(12, 7).Radius(2).Background(Accent).Border(strokeW, Ink)
		}
	})
}

// smoothScroll moves the shown editor to bring the current match to the
// middle, gliding in 550ms; the user can scroll away between jumps.
func (a *App) smoothScroll(c *ui.Context, st *editor.State, pv *preview) {
	f := &a.ws.find
	if !f.scrollPending || pv.curLine < 0 {
		return
	}
	if f.scrollAt.IsZero() {
		f.scrollAt, f.scrollFrom = time.Now(), st.ScrollOffset()
	}
	lh := float64(st.Font.Size * st.LineHeight)
	target := st.LineTop(pv.curLine) - float64(st.ViewHeight())/2 + lh/2
	if target < 0 {
		target = 0
	}
	p := float64(anim(c, f.scrollAt, dJump, EaseOut))
	st.SetScrollOffset(f.scrollFrom + (target-f.scrollFrom)*p)
	if p >= 1 {
		f.scrollPending, f.scrollAt = false, time.Time{}
	}
}

// statusBarView is the 46px bar under the text: what the pointer is on, the
// match position or a hint on the left; line numbers and invisibles toggles,
// and the size of the text on the right.
func (a *App) statusBarView(c *ui.Context, k tokens, pv *preview) {
	w := &a.ws
	ui.Row(c.Key("status")).Height(statusBarH).MinHeight(statusBarH).Padding(0, 12, 0, 18).Gap(10).AlignItems(ui.Center).
		Background(Paper).BorderWidth(strokeW, 0, 0, 0).BorderColor(Ink).Children(func() {
		ui.Row(c).Grow(1).MinWidth(0).Gap(10).AlignItems(ui.Center).Children(func() {
			switch {
			case pv != nil && w.applyFx == nil && w.tab == 0:
				p := anim(c, w.pvAt, dFlip, EaseOut)
				ui.Text(c, "预览").FontSize(11).FontWeight(700).TextColor(Accent).Background(Ink).Radius(radiusPill).Padding(1, 8).SingleLine()
				ui.Text(c, pv.label).FontSize(sizeButton).FontWeight(700).TextColor(Ink).SingleLine().Translate(0, 6*(1-p)).Opacity(p)
				ui.Text(c, pv.delta).Font(monoFont).FontSize(11.5).TextColor(Ink).SingleLine()
				a.legend(c)
				ui.Spacer(c)
				ui.Text(c, "点击应用 · 移开即恢复").FontSize(sizeButton).TextColor(Muted).SingleLine()
			case pv != nil && w.applyFx == nil && w.find.pat.hover != "":
				ui.Text(c, "预览").FontSize(11).FontWeight(700).TextColor(Accent).Background(Ink).Radius(radiusPill).Padding(1, 8).SingleLine()
				ui.Text(c, "模式「"+pv.label+"」").FontSize(sizeButton).FontWeight(700).TextColor(Ink).SingleLine()
				ui.Text(c, pv.delta).Font(monoFont).FontSize(11.5).TextColor(Ink).SingleLine()
				ui.Spacer(c)
				ui.Text(c, "点击使用 · 移开即恢复").FontSize(sizeButton).TextColor(Muted).SingleLine()
			case w.tab == 1:
				ui.Text(c, a.findStatus()).Font(monoFont).FontSize(11.5).TextColor(Ink).SingleLine().Shrink(1)
				ui.Spacer(c)
				ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
					kbd(c, "⌘G")
					ui.Text(c, "下一处").FontSize(sizeButton).TextColor(Muted).SingleLine()
					kbd(c, "⇧⌘G")
					ui.Text(c, "上一处").FontSize(sizeButton).TextColor(Muted).SingleLine()
				})
			default:
				msg := "悬停右侧动作先看改动，点击才应用"
				if sels := a.selections(); len(sels) > 0 {
					n := 0
					for _, s := range sels {
						n += utf8.RuneCountInString(a.ed.Buffer.Slice(s.Start(), s.End()))
					}
					msg = fmt.Sprintf("作用于选区 · %d 字 · 动作只改选中的文字", n)
				} else if a.status != "" {
					msg = a.status
				}
				col := Muted
				if strings.Contains(msg, "失败") || strings.Contains(msg, "错误") {
					col = Danger
				}
				ui.Text(c, msg).FontSize(sizeButton).TextColor(col).SingleLine()
			}
		})
		toggle := func(key, glyph, tip string, on bool, font string) bool {
			b := ui.ButtonBase(c.Key(key)).Size(30, 30).Radius(radiusCtl).Center().Label(tip).Tooltip(tip).Selected(on)
			bg, bd := ui.Transparent, ui.Transparent
			switch {
			case on:
				bg, bd = Accent, Ink
			case b.Hovered():
				bg, bd = Paper, Ink
			}
			b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6})).Border(strokeW, bd)
			focusRing(b)
			b.Children(func() { ui.Text(c, glyph).Font(font).FontSize(13).FontWeight(700).TextColor(Ink) })
			return b.Clicked()
		}
		if toggle("lnum", "#", "显示行号", a.view.lineNumbers, monoFont) {
			a.view.lineNumbers = !a.view.lineNumbers
		}
		if toggle("invis", "¶", "显示空格、制表符和换行 ⌥⌘I", a.view.invisibles, uiFontFamily) {
			a.view.invisibles = !a.view.invisibles
		}
		lines := strings.Count(a.Text, "\n") + 1
		if a.Text == "" {
			lines = 0
		}
		ui.Textf(c, "%d 字 · %d 行", engine.RuneLen(a.Text), lines).Font(monoFont).FontSize(11).TextColor(Muted).SingleLine()
	})
}

// legend is the 删 / 增 key of the preview colours.
func (a *App) legend(c *ui.Context) {
	ui.Row(c).Gap(10).Margin(0, 0, 0, 6).AlignItems(ui.Center).Children(func() {
		for _, it := range []struct {
			col ui.Color
			s   string
		}{{Del, "删"}, {Ins, "增"}} {
			ui.Row(c.Key("lg" + it.s)).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(11, 11).Radius(3).Background(it.col).Border(keycapW, Ink)
				ui.Text(c, it.s).FontSize(sizeButton).TextColor(Muted).SingleLine()
			})
		}
	})
}
