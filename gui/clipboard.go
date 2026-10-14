package gui

import (
	"sort"
	"strings"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
)

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
	case strings.HasPrefix(id, "page:"):
		return "arrow-right"
	case strings.HasPrefix(id, "cmd:"):
		return "type"
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
	for _, id := range a.Lib.Settings.Pinned {
		if label, fn := a.resolveAction(id); fn != nil && !strings.HasPrefix(id, "chain:") {
			add(id, label, "已固定", "")
		}
	}
	for i := range a.Lib.Chains {
		add("chain:"+a.Lib.Chains[i].ID, a.Lib.Chains[i].Name, "动作链", "")
	}
	for _, ec := range editorCommands {
		add("cmd:"+ec.name, ec.title, "编辑 "+ec.keys, "")
	}
	for _, it := range append(append([]navItem{}, navMain[1:]...), append(navTools, navItem{PageSettings, "settings-2", "设置"})...) {
		add("page:"+it.id, "前往："+it.label, "页面", "")
	}
	for i := range a.Lib.RuleSets {
		add("rules:"+a.Lib.RuleSets[i].ID, a.Lib.RuleSets[i].Name, "批量替换规则", "")
	}
	for _, cl := range engine.Cleaners() {
		if !cl.Implemented || a.isPinned(cl.Name) {
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
	if strings.HasPrefix(it.id, "page:") {
		a.Page = strings.TrimPrefix(it.id, "page:")
		return
	}
	if ec := editorCommandNamed(strings.TrimPrefix(it.id, "cmd:")); ec != nil && strings.HasPrefix(it.id, "cmd:") {
		a.Page = PageClipboard
		a.RunEditorCommand(ec.cmd)
		return
	}
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
