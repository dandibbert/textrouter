package gui

import (
	"fmt"
	"time"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

// View is the window content: the clipboard workspace of
// docs/design/clipboard-redesign.md — a title bar with the step trail, the
// text in a white card, and the right column with 清理 and 查找替换.
func (a *App) View(c *ui.Context) {
	registerFonts()
	uiFontFamily = a.uiFamily()
	k := applyTheme(c)
	a.writeClip = func(t string) { c.WriteClipboard(t); a.noteCopied(t) }
	a.ensureDocs()
	a.persistView()
	a.syncEditor()
	a.refreshCaches()
	w := &a.ws
	if a.Page == PageFind {
		a.Page = PageClipboard
		a.setTab(1)
	}
	if fx := w.applyFx; fx != nil && (c.Preferences().ReduceMotion || time.Since(fx.at) > dApplyGone) {
		w.applyFx = nil
	}
	if w.tab == 0 && w.find.pat.hover != "" {
		w.find.pat.hover = ""
	}

	c.OnShortcut(ui.Cmd, ui.KeyK, a.openPalette)
	c.OnShortcut(ui.Cmd, ui.Key1, func() { a.setTab(0) })
	c.OnShortcut(ui.Cmd, ui.KeyF, func() { a.setTab(1); w.focusFnd = true })
	c.OnShortcut(ui.Cmd, ui.KeyG, a.NextMatch)
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyG, a.PrevMatch)
	c.OnShortcut(ui.Cmd|ui.Alt, ui.KeyEnter, func() {
		if w.tab == 1 {
			a.replaceCurrent()
		}
	})
	c.OnEditCommand("undo", a.Undo) // the Edit menu takes ⌘Z, whatever has the focus
	c.OnEditCommand("redo", a.Redo)
	c.OnShortcut(ui.Cmd, ui.KeyZ, a.Undo)
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyZ, a.Redo)
	c.OnShortcut(ui.Cmd, ui.KeyComma, func() { a.Page = PageSettings })
	c.OnShortcut(ui.Cmd, ui.KeyW, a.CloseCurrentTab) // the File menu takes ⌘W; this is the same action
	if a.Page != PageClipboard && a.Page != "" && !a.paletteOpen {
		c.OnShortcut(0, ui.KeyEscape, func() { a.Page = PageClipboard })
	}

	w.frameHov = ""
	ui.Column(c).Fill().Background(Ground).Children(func() {
		a.titleBar(c, k)
		ui.Row(c).Grow(1).MinHeight(0).Gap(bodyGap).Padding(padTop, padRight, padBottom, padLeft).AlignItems(ui.Stretch).Children(func() {
			a.bodyCard(c, k)
			a.rightColumn(c, k)
		})
	})
	if w.tab == 0 && w.hov != w.frameHov {
		w.hov, w.hovAt = w.frameHov, time.Now()
		c.Invalidate()
	} else if w.tab != 0 {
		w.hov = ""
	}
	if a.Page != PageClipboard && a.Page != "" {
		a.sheetView(c, k)
	}
	a.paletteView(c)
	a.toastView(c)
	a.closePromptView(c)
}

// ---------------------------------------------------------------- title bar

func (a *App) clipAge() string {
	if a.clipAt.IsZero() {
		return ""
	}
	d := time.Since(a.clipAt)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	}
	return fmt.Sprintf("%d 小时前", int(d.Hours()))
}

// titleBar is the 54px bar: the window's title, the step trail in the middle,
// then tools, undo, redo and the copy button.
func (a *App) titleBar(c *ui.Context, k tokens) {
	bar := c.TitleBar()
	ui.Row(c).Height(titleBarH).MinHeight(titleBarH).Padding(0, 14, 0, max(bar.Left, 0)+18).Gap(10).AlignItems(ui.Center).
		Background(Ground).BorderWidth(0, 0, strokeW, 0).BorderColor(Ink).DragWindow().Children(func() {
		if len(a.docs) > 1 {
			ui.Row(c.Key("doctabs")).Gap(6).AlignItems(ui.Center).Shrink(1).MinWidth(0).ClipX().Children(func() {
				for i, d := range a.docs {
					a.docTab(c, k, i, d)
				}
			})
		} else if d := a.doc(); !d.clip {
			ui.Icon(c, ic("file-text")).FontSize(16).TextColor(Ink)
			ui.Text(c, d.name).FontSize(16).FontWeight(700).TextColor(Ink).LetterSpacing(-0.3).SingleLine().MaxWidth(240).MinWidth(0).Shrink(1)
			a.unsavedDot(c, a.cur)
		} else {
			t := ui.ButtonBase(c.Key("cliptitle")).Height(30).Gap(10).AlignItems(ui.Center).Label("剪贴板").
				Tooltip("拖出标题栏，或右键，在新窗口中打开副本")
			a.tearOff(c, t, 0, d)
			t.Children(func() {
				ui.Text(c, "剪贴板").FontSize(16).FontWeight(700).TextColor(Ink).LetterSpacing(-0.3).SingleLine()
				if age := a.clipAge(); age != "" {
					ui.Text(c, age).FontSize(12).TextColor(Muted).SingleLine()
				}
			})
		}
		if b := iconButton(c, "opendoc", "plus", "打开文件", "打开文件 ⌘O").Size(28, 28); b.Clicked() {
			a.openFile()
		}
		a.crumbs(c, k)
		a.pendingClipPill(c, k)
		a.toolsMenu(c, k)
		if iconButton(c, "undo", "undo-2", "撤销", "撤销  ⌘Z").Disabled(!a.CanUndo()).Clicked() {
			a.Undo()
		}
		if iconButton(c, "redo", "redo-2", "重做", "重做  ⇧⌘Z").Disabled(!a.CanRedo()).Clicked() {
			a.Redo()
		}
		if a.doc().clip {
			a.copyButton(c, k)
		} else {
			a.saveButton(c, k)
		}
	})
}

// crumbs is the step trail: 原文 → step → [current]. The current step is
// yellow and boxed, earlier ones grey, undone ones struck through; a click
// goes to that step without losing the others.
func (a *App) crumbs(c *ui.Context, k tokens) {
	t := a.trail()
	saveChain := a.canSaveChain()
	row := ui.Row(c.Key("crumbs")).Grow(1).MinWidth(0).Gap(2).AlignItems(ui.Center).Justify(ui.Center).ClipX()
	lb := row.LayoutBounds()
	avail := lb.W
	if lb.H == 0 {
		avail = 480
		if len(t.steps) > 1 || saveChain {
			c.Invalidate()
		}
	} else {
		if avail != a.ws.crumbsW {
			a.ws.crumbsW = avail
			c.Invalidate()
		}
	}
	const saveChainW = float32(98) // button width (~88) + 10px left margin
	row.Children(func() {
		if lb.H > 0 && avail < 44 {
			return
		}
		showSave := saveChain && avail >= saveChainW
		crumbAvail := avail
		if showSave {
			crumbAvail -= saveChainW
		}
		if !showSave || crumbAvail >= 44 {
			from, to := crumbRange(t.steps, t.pos, crumbAvail)
			if from <= to {
				if from > 0 {
					ui.Text(c, "…").FontSize(12).TextColor(Faint).Padding(0, 4).SingleLine()
				}
				single := from == to
				for i := from; i <= to; i++ {
					s := t.steps[i]
					if i > from {
						ui.Text(c, "→").FontSize(12).TextColor(Faint).Padding(0, 2).SingleLine()
					}
					p := anim(c, s.at, dPop, EaseSpring)
					b := ui.ButtonBase(c.Key(fmt.Sprint("crumb", i, s.label))).Height(26).Padding(0, 9).Radius(radiusCtl).Center().Label(s.label).MaxWidth(150)
					if single && crumbAvail-8 < crumbWidthEst(s.label) {
						b.Shrink(1).MinWidth(0)
					}
					col, bg, bd := Muted, ui.Transparent, ui.Transparent
					strike := false
					switch {
					case i == t.pos:
						col, bg, bd = Ink, Accent, Ink
					case i > t.pos:
						col, strike = Faint, true
					}
					if b.Hovered() && i != t.pos {
						bg = AccentSoft
					}
					b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 150e6})).Border(strokeW, bd).Scale(0.6+0.4*p, 0.6+0.4*p).Opacity(p)
					focusRing(b)
					b.Children(func() {
						tx := ui.Text(c, s.label).FontSize(12).FontWeight(map[bool]int{true: 700, false: 400}[i == t.pos]).TextColor(col).SingleLine().MinWidth(0).Shrink(1)
						if strike {
							tx.Strikethrough()
						}
					})
					if b.Clicked() {
						a.goStep(i)
					}
				}
				if to < len(t.steps)-1 {
					ui.Text(c, "…").FontSize(12).TextColor(Faint).Padding(0, 4).SingleLine()
				}
			}
		}
		if showSave {
			b := ui.ButtonBase(c.Key("savechain")).Height(28).Padding(0, 10).Radius(radiusCtl).Center().Margin(0, 0, 0, 10).Label("存为动作链")
			b.Background(Paper)
			boxed(b)
			lift(b, 2)
			b.Children(func() { ui.Text(c, "存为动作链").FontSize(12).FontWeight(500).TextColor(Ink).SingleLine() })
			if b.Clicked() {
				a.saveTrailAsChain()
			}
		}
	})
}

func crumbWidthEst(label string) float32 {
	var w float32
	for _, r := range label {
		switch {
		case r >= 0x1100:
			w += 12.5
		case r >= 'A' && r <= 'Z':
			w += 8.5
		default:
			w += 7.5
		}
	}
	return min(150, w+24)
}

func crumbRange(steps []step, pos int, avail float32) (int, int) {
	n := len(steps)
	if n == 0 {
		return 0, -1
	}
	if pos < 0 {
		pos = 0
	} else if pos >= n {
		pos = n - 1
	}
	const (
		maxCrumbs = 7
		arrowW    = float32(22)
		ellipsisW = float32(24)
	)
	budget := avail - 8
	totalW := func(f, t int) float32 {
		w := float32(0)
		for i := f; i <= t; i++ {
			if i > f {
				w += arrowW
			}
			w += crumbWidthEst(steps[i].label)
		}
		if f > 0 {
			w += ellipsisW
		}
		if t < n-1 {
			w += ellipsisW
		}
		return w
	}
	from, to := pos, pos
	if totalW(from, to) > budget {
		return from, to
	}
	if to < n-1 && totalW(from, to+1) <= budget {
		to++
	}
	for to-from+1 < maxCrumbs {
		grew := false
		if from > 0 && totalW(from-1, to) <= budget {
			from--
			grew = true
		}
		if to-from+1 < maxCrumbs && to < n-1 && to-pos < 3 && totalW(from, to+1) <= budget {
			to++
			grew = true
		}
		if !grew && to < n-1 && totalW(from, to+1) <= budget {
			to++
			grew = true
		}
		if !grew {
			break
		}
	}
	return from, to
}

// canSaveChain is true once two or more cleaner steps stand on the trail.
func (a *App) canSaveChain() bool {
	t := a.trail()
	n := 0
	for _, s := range t.steps[1 : t.pos+1] {
		if s.act != nil {
			n++
		}
	}
	return n >= 2
}

// copyButton is 复制并完成: it draws a tick and says 已复制 once pressed.
func (a *App) copyButton(c *ui.Context, k tokens) {
	w := &a.ws
	b := ui.ButtonBase(c.Key("done")).Height(36).MinWidth(140).Padding(0, 12).Gap(8).Radius(radiusCtl).Center().Label("复制并完成").Tooltip("复制回剪贴板并收起窗口")
	bg := Accent
	if b.Hovered() {
		bg = AccentHover
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6}))
	boxed(b)
	lift(b, shadowSm)
	copied := !w.copiedAt.IsZero() && time.Since(w.copiedAt) < 1800*time.Millisecond
	b.Children(func() {
		if copied {
			p := anim(c, w.copiedAt.Add(60*time.Millisecond), dDraw, EaseOut)
			ui.Box(c).Size(15*p, 15).Clip().Children(func() {
				ui.Icon(c, ic("check")).FontSize(15).TextColor(Ink).Width(15)
			})
			f := anim(c, w.copiedAt, 220*time.Millisecond, nil)
			ui.Text(c, "已复制").FontSize(13).FontWeight(700).TextColor(Ink).Opacity(f).Translate(0, 4*(1-f))
			return
		}
		ui.Text(c, "复制并完成").FontSize(13).FontWeight(700).TextColor(Ink).SingleLine()
		kbd(c, "⌘↩")
	})
	if b.Clicked() {
		a.done()
	}
}

// saveButton is 存储 ⌘S on file tabs: highlighted in yellow while the file
// has unsaved edits, calm white once saved.
func (a *App) saveButton(c *ui.Context, k tokens) {
	dirty := a.docDirty(a.cur)
	tip := "存储文件 ⌘S"
	if a.doc().path == "" {
		tip = "存储为文件 ⌘S"
	}
	b := ui.ButtonBase(c.Key("savefile")).Height(36).Padding(0, 12).Gap(7).Radius(radiusCtl).Center().Label("保存文件").Tooltip(tip)
	bg := Paper
	switch {
	case dirty && b.Hovered():
		bg = AccentHover
	case dirty:
		bg = Accent
	case b.Hovered():
		bg = AccentWash
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6}))
	boxed(b)
	lift(b, shadowSm)
	b.Children(func() {
		ui.Text(c, "保存").FontSize(13).FontWeight(700).TextColor(Ink).SingleLine()
		kbd(c, "⌘S")
	})
	if b.Clicked() {
		a.saveFile()
	}
}

// done copies the text back, says so, and hides the window after a moment.
func (a *App) done() {
	if a.writeClip != nil {
		a.writeClip(a.Text)
	}
	a.status = "已复制到剪贴板"
	a.ws.copiedAt = time.Now()
	sub := "窗口即将收起，回到上一个应用直接 ⌘V"
	if !a.Lib.Settings.CopyOnDone {
		sub = "回到上一个应用直接 ⌘V"
	}
	a.showToast("已复制到剪贴板", sub)
	if a.Lib.Settings.CopyOnDone {
		time.AfterFunc(900*time.Millisecond, func() { a.Platform.Update(a.Platform.HideWindow) })
	}
}

// ---------------------------------------------------------------- right column

func (a *App) rightColumn(c *ui.Context, k tokens) {
	w := &a.ws
	ui.Column(c.Key("rail")).Width(rightColW).MinWidth(rightColW).FillHeight().Children(func() {
		a.tabSwitch(c, k)
		p := anim(c, w.tabAt, dPanel, EaseOut)
		ui.Column(c.Key(fmt.Sprint("panel", w.tab))).Grow(1).MinHeight(0).FillWidth().Translate(w.tabDir*10*(1-p), 0).Opacity(p).Children(func() {
			if w.tab == 0 {
				a.cleanTab(c, k)
			} else {
				a.findTab(c, k)
				a.patternMenu(c, k)
			}
		})
	})
}

// tabSwitch is the 清理 | 查找替换 control: a yellow slider springs between
// the two labels.
func (a *App) tabSwitch(c *ui.Context, k tokens) {
	w := &a.ws
	half := float32(rightColW-2*strokeW) / 2
	frame := ui.Box(c.Key("tabs")).FillWidth().Height(40).Margin(0, 0, 16, 0).Background(Paper).Radius(radiusSeg)
	hard(boxed(frame), shadowSm)
	frame.Children(func() {
		x := frame.AnimateWith("x", float32(w.tab)*half, dTab, EaseSpring)
		ui.Box(c.Key("thumb")).Absolute().Top(3).Left(3+x).Size(half-3, 30).Radius(radiusThumb).Background(Accent).Border(strokeW, Ink)
		ui.Row(c.Key("tabbtns")).Absolute().Top(3).Left(3).Right(3).Height(30).Children(func() {
			for i, it := range []struct{ label, key string }{{"清理", "⌘1"}, {"查找替换", "⌘F"}} {
				b := ui.ButtonBase(c.Key("tab" + it.label)).Grow(1).Basis(0).FillHeight().Gap(7).Radius(radiusThumb).Center().Label(it.label).Selected(w.tab == i)
				focusRing(b)
				b.Children(func() {
					ui.Text(c, it.label).FontSize(sizeBody).FontWeight(700).TextColor(Ink).SingleLine()
					kbd(c, it.key)
				})
				if b.Clicked() {
					a.setTab(i)
				}
			}
		})
	})
}

var _ = engine.RuneLen
