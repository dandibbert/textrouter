package gui

import (
	"sort"
	"strings"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

// clipboardView is the main screen: the clipboard text on a large canvas,
// a compact action bar, and one primary "copy & close".
func (a *App) clipboardView(c *ui.Context) {
	k := tok(c)
	c.OnShortcut(ui.Cmd, ui.KeyEnter, a.done)
	pageHeader(c, "剪贴板空间", "", func() {
		if iconButton(c, "read", "clipboard-paste", "读取剪贴板", "重新读取剪贴板").Clicked() {
			a.LoadClipboard(c.ReadClipboard())
		}
		u := iconButton(c, "undo", "undo-2", "撤销", "撤销 ⌘Z").Disabled(len(a.history) == 0)
		if len(a.history) == 0 {
			u.Opacity(0.35)
		}
		if u.Clicked() {
			a.Undo()
		}
		r := iconButton(c, "redo", "redo-2", "重做", "重做 ⇧⌘Z").Disabled(len(a.future) == 0)
		if len(a.future) == 0 {
			r.Opacity(0.35)
		}
		if r.Clicked() {
			a.Redo()
		}
		ui.Box(c).Width(sp2)
		if primary(c, "done", "复制并完成", "copy", "⌘↩", false).Tooltip("复制回剪贴板并收起窗口").Clicked() {
			a.done()
		}
	})
	// Action bar: the palette launcher and a few pinned chips.
	ui.Row(c).Padding(0, sp5, sp3, sp5).Gap(6).AlignItems(ui.Center).Children(func() {
		launcher := ui.ButtonBase(c.Key("apply")).Height(30).Width(232).Padding(0, 8, 0, 10).Radius(rMd).Gap(8).AlignItems(ui.Center).Label("应用动作").Tooltip("搜索全部清理器、动作链和规则")
		pressable(c, launcher, k, k.canvas)
		launcher.Shadow(0, 0, 0, 1, k.hairStrong).Shadow(0, 1, 2, 0, k.shadowNear)
		launcher.Children(func() {
			ui.Icon(c, ic("wand-sparkles")).FontSize(14).TextColor(k.accent)
			ui.Text(c, "应用动作…").FontSize(12.5).TextColor(k.muted).Grow(1).SingleLine()
			kbd(c, "⌘K")
		})
		if launcher.Clicked() {
			a.openPalette()
		}
		ui.Box(c).Size(1, 18).Background(k.hairStrong).Margin(0, 4)
		ui.Row(c).Gap(6).Grow(1).MinWidth(0).Clip().AlignItems(ui.Center).Children(func() {
			for _, id := range a.pins(4) {
				label, fn := a.resolveAction(id)
				name, icon := strings.TrimPrefix(strings.TrimPrefix(label, "⛓ "), "⇄ "), actionIcon(id)
				if chip(c, "pin:"+id, icon, name, "应用「"+name+"」").Clicked() {
					a.Transform(name, fn)
				}
			}
		})
		if iconButton(c, "more", "ellipsis", "更多动作", "全部动作 ⌘K").Clicked() {
			a.openPalette()
		}
	})
	// The canvas.
	ui.Column(c).Grow(1).Padding(0, sp5, sp4, sp5).MinHeight(0).Children(func() {
		canvasCard(c).Grow(1).MinHeight(0).Clip().Children(func() {
			if a.last != nil && a.last.showDiff {
				ui.Scroll(c).Grow(1).Padding(sp6, sp8).Children(func() {
					diffView(c, a.last.before, a.last.after)
				})
			} else {
				ed := ui.TextAreaBase(c, &a.Text).Grow(1).Padding(sp6, sp8).Label("剪贴板文本").
					Placeholder("剪贴板是空的。粘贴或输入文字…").AutoFocus().FontSize(15).LineHeight(1.65).TextColor(k.text)
				if a.monoEdit {
					ed.Font(monoFont).FontSize(13.5)
				} else {
					ed.Font(editorFont)
				}
				if a.last != nil && a.last.after == a.Text {
					ed.TextRanges(insertRanges(a.last.before, a.last.after, k.accent)...)
				}
			}
			// Footer strip inside the card: what the last action did.
			ui.Row(c).Height(40).Padding(0, sp3, 0, sp5).Gap(sp2).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(k.hair).Children(func() {
				if a.last != nil {
					ui.Icon(c, ic("sparkles")).FontSize(13).TextColor(k.accent)
					ui.Text(c, a.last.label).FontSize(12).FontWeight(500).TextColor(k.text).SingleLine()
					diffBadge(c, a.last.before, a.last.after)
					if ins, _ := engine.DiffStats(cachedDiff(a.last.before, a.last.after)); ins > 0 && !a.last.showDiff {
						ui.Text(c, "改动的文字以强调色显示").FontSize(11.5).TextColor(k.faint).SingleLine().Shrink(1).MinWidth(0)
					}
				} else {
					ui.Text(c, "选一个动作，或直接编辑。改动会高亮显示，并可随时撤销。").FontSize(11.5).TextColor(k.faint).SingleLine().Shrink(1).MinWidth(0)
				}
				ui.Spacer(c)
				if a.last != nil {
					label, icon := "对照改动", "git-compare-arrows"
					if a.last.showDiff {
						label, icon = "返回编辑", "type"
					}
					if ghost(c, "diff", label, icon, false).Clicked() {
						a.last.showDiff = !a.last.showDiff
					}
				}
				mono := iconButton(c, "mono", "type", "等宽字体", "切换等宽字体")
				if a.monoEdit {
					mono.Background(k.accentSoft)
				}
				if mono.Clicked() {
					a.monoEdit = !a.monoEdit
				}
			})
		})
	})
}

// pins returns up to n pinned action ids that still resolve.
func (a *App) pins(n int) []string {
	var out []string
	for _, id := range a.Lib.Settings.Pinned {
		if _, fn := a.resolveAction(id); fn != nil {
			out = append(out, id)
		}
	}
	for i := range a.Lib.Chains {
		id := "chain:" + a.Lib.Chains[i].ID
		if !a.isPinned(id) {
			out = append(out, id)
		}
	}
	if len(out) > n {
		out = out[:n]
	}
	return out
}

// actionIcon picks an icon for an action id.
func actionIcon(id string) string {
	switch {
	case strings.HasPrefix(id, "chain:"):
		return "workflow"
	case strings.HasPrefix(id, "rules:"):
		return "replace-all"
	}
	l := strings.ToLower(id)
	switch {
	case strings.Contains(l, "scrub"):
		return "sparkles"
	case strings.Contains(l, "space"), strings.Contains(l, "trim"):
		return "eraser"
	case strings.Contains(l, "paragraph"), strings.Contains(l, "line"):
		return "pilcrow"
	case strings.Contains(l, "quote"):
		return "quote"
	case strings.Contains(l, "duplicate"), strings.Contains(l, "remove"):
		return "list-x"
	case strings.Contains(l, "case"):
		return "case-sensitive"
	case strings.Contains(l, "wrap"):
		return "wrap-text"
	}
	return "wand-sparkles"
}

// insertRanges returns the rune ranges of after that the diff inserted.
func insertRanges(before, after string, col ui.Color) []ui.TextRange {
	if engine.RuneLen(before)+engine.RuneLen(after) > 2*maxPreview {
		return nil
	}
	var out []ui.TextRange
	pos := 0
	for _, s := range cachedDiff(before, after) {
		n := engine.RuneLen(s.Text)
		switch s.Op {
		case '=':
			pos += n
		case '+':
			out = append(out, ui.TextRange{Start: pos, End: pos + n, Color: col, Weight: 600})
			pos += n
		}
	}
	return out
}

// done copies the text back and hides the window.
func (a *App) done() {
	if a.writeClip != nil {
		a.writeClip(a.Text)
	}
	a.status = "已复制到剪贴板"
	if a.Lib.Settings.CopyOnDone {
		a.Platform.HideWindow()
	}
}

func (a *App) chainFn(ch *engine.Chain) func(string) string {
	return func(s string) string {
		out, _ := ch.Run(s, a.Lib.Resolver())
		return out
	}
}

// resolveAction maps a palette/pinned id to a label and transform.
// Ids: a cleaner name, "chain:<id>" or "rules:<id>".
func (a *App) resolveAction(id string) (string, func(string) string) {
	switch {
	case strings.HasPrefix(id, "chain:"):
		if ch := a.Lib.ChainByID(strings.TrimPrefix(id, "chain:")); ch != nil {
			return ch.Name, a.chainFn(ch)
		}
	case strings.HasPrefix(id, "rules:"):
		for i := range a.Lib.RuleSets {
			rs := &a.Lib.RuleSets[i]
			if rs.ID == strings.TrimPrefix(id, "rules:") {
				return rs.Name, func(s string) string { out, _, _ := engine.ApplyRules(s, rs.Rules); return out }
			}
		}
	default:
		if cl := engine.CleanerByName(id); cl != nil {
			return cl.ZhName, cl.Fn
		}
	}
	return id, nil
}

// paletteItem is one row of the command palette.
type paletteItem struct {
	id, title, sub, badge string
	score                 int
}

func (a *App) paletteItems() []paletteItem {
	q := strings.ToLower(strings.TrimSpace(a.paletteQuery))
	var items []paletteItem
	add := func(id, title, sub, badge string) {
		score := 0
		if q != "" {
			hay := strings.ToLower(title + " " + sub)
			switch {
			case strings.HasPrefix(strings.ToLower(title), q):
				score = 3
			case strings.Contains(hay, q):
				score = 2
			case fuzzy(hay, q):
				score = 1
			default:
				return
			}
		}
		items = append(items, paletteItem{id, title, sub, badge, score})
	}
	for i := range a.Lib.Chains {
		add("chain:"+a.Lib.Chains[i].ID, a.Lib.Chains[i].Name, "动作链", "")
	}
	for i := range a.Lib.RuleSets {
		add("rules:"+a.Lib.RuleSets[i].ID, a.Lib.RuleSets[i].Name, "批量替换规则", "")
	}
	for _, cl := range engine.Cleaners() {
		if !cl.Implemented {
			continue
		}
		badge := ""
		if !cl.Measured {
			if !a.Lib.Settings.ShowUnmeasure {
				continue
			}
			badge = "未验证"
		}
		add(cl.Name, cl.ZhName, cl.Name+" · "+cl.Category, badge)
	}
	sort.SliceStable(items, func(i, j int) bool { return items[i].score > items[j].score })
	return items
}

func fuzzy(hay, q string) bool {
	i := 0
	hr := []rune(hay)
	for _, c := range q {
		for i < len(hr) && hr[i] != c {
			i++
		}
		if i == len(hr) {
			return false
		}
		i++
	}
	return true
}

func (a *App) openPalette() {
	a.paletteOpen = true
	a.paletteFresh = true
	a.paletteQuery = ""
	a.paletteSel = 0
}

func (a *App) runPaletteItem(it paletteItem) {
	a.paletteOpen = false
	label, fn := a.resolveAction(it.id)
	if fn != nil {
		a.Transform(label, fn)
		if a.Page != PageClipboard && a.Page != PageFind && a.Page != PageBatch {
			a.Page = PageClipboard
		}
	}
}

func (a *App) paletteView(c *ui.Context) {
	if !a.paletteOpen {
		return
	}
	k := tok(c)
	items := a.paletteItems()
	if a.paletteSel >= len(items) {
		a.paletteSel = max(0, len(items)-1)
	}
	ui.Overlay(c, func() {
		// Scrim: click to dismiss.
		ui.Box(c.Key("pal-blur")).Absolute().Top(0).Left(0).Right(0).Bottom(0).PassThrough().Material(glass.Blur{Radius: 6})
		scrim := ui.Box(c.Key("pal-scrim")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(ui.RGBA(0, 0, 0, map[bool]float32{true: 0.40, false: 0.14}[k.dark]))
		scrim.OnShortcut(0, ui.KeyEscape, func() { a.paletteOpen = false })
		if scrim.Clicked() {
			a.paletteOpen = false
		}
		ui.Column(c).Absolute().Top(0).Left(0).Right(0).Bottom(0).AlignItems(ui.Center).PassThrough().Padding(96, 0, 0, 0).Children(func() {
			pn := ui.Column(c.Key("pal")).Width(600).Radius(16).Clip()
			if k.dark {
				pn.Background(ui.Hex("#232327").Alpha(0.96)).Border(1, k.hairStrong)
			} else {
				pn.Background(ui.RGBA(252, 252, 253, 0.97)).Border(1, k.hair)
			}
			pn.Shadow(0, 24, 64, -12, ui.RGBA(0, 0, 0, map[bool]float32{true: 0.6, false: 0.28}[k.dark])).Shadow(0, 2, 6, 0, ui.RGBA(0, 0, 0, 0.08))
			target := float32(1)
			if a.paletteFresh {
				target, a.paletteFresh = 0, false
				c.Invalidate()
			}
			sc := pn.AnimateTo("in", target, ui.Snappy)
			pn.Children(func() {
				ui.Row(c).Height(56).Padding(0, sp5).Gap(sp3).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(k.hair).Children(func() {
					ui.Icon(c, ic("search")).FontSize(18).TextColor(k.faint)
					f := ui.TextInputBase(c, &a.paletteQuery).Grow(1).FontSize(16).Placeholder("搜索清理器、动作链、规则…").AutoFocus().Label("搜索动作").TextColor(k.text)
					if f.Changed() {
						a.paletteSel = 0
					}
					f.OnShortcut(0, ui.KeyDown, func() { a.paletteSel = min(a.paletteSel+1, len(items)-1) })
					f.OnShortcut(0, ui.KeyUp, func() { a.paletteSel = max(a.paletteSel-1, 0) })
					f.OnShortcut(0, ui.KeyEscape, func() { a.paletteOpen = false })
					if f.Submitted() && len(items) > 0 {
						a.runPaletteItem(items[a.paletteSel])
					}
					kbd(c, "esc")
				})
				ui.Text(c, "全部动作").FontSize(11).FontWeight(600).TextColor(k.faint).Padding(sp3, sp5, sp1, sp5)
				ui.Scroll(c).Height(380).Padding(0, sp2, sp2, sp2).Children(func() {
					ui.Column(c).Gap(1).Children(func() {
						for i, it := range items {
							if i >= 80 {
								hint(c, "…继续输入以缩小范围").Padding(sp2, sp3)
								break
							}
							sel := i == a.paletteSel
							row := ui.ButtonBase(c.Key("pal:"+it.id)).Justify(ui.Start).AlignItems(ui.Center).Height(44).Padding(0, sp3).Radius(10).Gap(sp3).FillWidth().Label(it.title)
							bg := ui.Transparent
							if sel {
								bg = k.accentSoft
							} else if row.Hovered() {
								bg = k.hover
							}
							row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 90e6}))
							if row.Hovered() && !sel {
								a.paletteSel = i
							}
							row.Children(func() {
								icol, ibg := k.muted, k.field
								if sel {
									icol, ibg = k.accentText, k.accent
								}
								ui.Box(c).Size(28, 28).Radius(7).Background(ibg).Center().Children(func() {
									ui.Icon(c, ic(actionIcon(it.id))).FontSize(15).TextColor(icol)
								})
								ui.Column(c).Grow(1).Gap(1).MinWidth(0).Children(func() {
									ui.Text(c, strings.TrimPrefix(strings.TrimPrefix(it.title, "⛓ "), "⇄ ")).FontSize(13.5).FontWeight(500).TextColor(k.text).SingleLine()
									ui.Text(c, it.sub).FontSize(11.5).TextColor(k.faint).SingleLine()
								})
								if it.badge != "" {
									badge(c, it.badge, toneAmber)
								}
								if sel {
									ui.Icon(c, ic("corner-down-left")).FontSize(14).TextColor(k.accent)
								}
							})
							if row.Clicked() {
								a.runPaletteItem(it)
							}
						}
						if len(items) == 0 {
							ui.Column(c).Padding(40).AlignItems(ui.Center).Gap(sp2).FillWidth().Children(func() {
								ui.Icon(c, ic("search")).FontSize(22).TextColor(k.faint)
								hint(c, "没有匹配的动作")
							})
						}
					})
				})
				ui.Row(c).Height(36).Padding(0, sp5).Gap(sp4).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(k.hair).Background(k.field).Children(func() {
					for _, p := range [][2]string{{"↑↓", "选择"}, {"↩", "应用到文本"}, {"esc", "关闭"}} {
						ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
							kbd(c, p[0])
							ui.Text(c, p[1]).FontSize(11.5).TextColor(k.muted)
						})
					}
					ui.Spacer(c)
					ui.Textf(c, "%d 个结果", len(items)).FontSize(11.5).TextColor(k.faint).FontFeatures("tnum")
				})
			})
			pn.Scale(0.96+0.04*sc, 0.96+0.04*sc).Opacity(sc)
		})
	})
}
