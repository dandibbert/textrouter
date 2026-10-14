package gui

import (
	"sort"
	"strings"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

// clipboardView is the main screen: the clipboard text, one-click
// cleaners, and copy back.
func (a *App) clipboardView(c *ui.Context) {
	t := c.Theme()
	c.OnShortcut(ui.Cmd, ui.KeyEnter, a.done)
	header(c, "剪贴板空间", "改好后按 ⌘↩ 复制回剪贴板并收起窗口；⌘K 搜索全部动作", func() {
		if ui.Button(c, "读取剪贴板").Tooltip("用当前剪贴板内容替换").Clicked() {
			a.LoadClipboard(c.ReadClipboard())
		}
		if ui.Button(c, "撤销").Disabled(len(a.history) == 0).Tooltip("撤销上一步清理").Clicked() {
			a.Undo()
		}
		if ui.PrimaryButton(c, "复制并完成").Tooltip("⌘↩").Clicked() {
			a.done()
		}
	})
	// Quick actions: pinned cleaners and chains.
	ui.Row(c).PaddingX(20).Gap(6).Wrap().Children(func() {
		for _, id := range a.Lib.Settings.Pinned {
			label, fn := a.resolveAction(id)
			if fn == nil {
				continue
			}
			if ui.Button(c.Key("pin:"+id), label).Clicked() {
				a.Transform(label, fn)
			}
		}
		for i := range a.Lib.Chains {
			ch := &a.Lib.Chains[i]
			if ui.Button(c.Key("chain:"+ch.ID), "⛓ "+ch.Name).Clicked() {
				a.Transform(ch.Name, a.chainFn(ch))
			}
		}
		if ui.Button(c, "＋ 更多…").Clicked() {
			a.openPalette()
		}
	})
	ui.Column(c).Grow(1).Padding(12, 20, 12, 20).Gap(8).MinHeight(0).Children(func() {
		if a.last != nil && a.last.showDiff {
			ui.Split(c, &a.setting.splitClip, func() {
				ui.TextArea(c, &a.Text).Fill().Font("monospace").FontSize(13).Label("剪贴板文本").Placeholder("剪贴板是空的。粘贴或输入文字…")
			}, func() {
				ui.Column(c).Fill().Gap(6).Padding(0, 0, 0, 10).Children(func() {
					muted(c, "「"+a.last.label+"」的改动（红=删除，绿=新增）")
					ui.Scroll(c).Grow(1).Background(t.Surface).Radius(8).Padding(10).Children(func() {
						diffView(c, a.last.before, a.last.after)
					})
				})
			}).Grow(1)
		} else {
			ui.TextArea(c, &a.Text).Grow(1).Font("monospace").FontSize(13).Label("剪贴板文本").Placeholder("剪贴板是空的。粘贴或输入文字…").AutoFocus()
		}
		if a.last != nil {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				muted(c, "上一步：「"+a.last.label+"」")
				diffBadge(c, a.last.before, a.last.after)
				ui.Spacer(c)
				label := "显示改动"
				if a.last.showDiff {
					label = "隐藏改动"
				}
				if ui.Button(c, label).Clicked() {
					a.last.showDiff = !a.last.showDiff
				}
			})
		}
	})
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
			return "⛓ " + ch.Name, a.chainFn(ch)
		}
	case strings.HasPrefix(id, "rules:"):
		for i := range a.Lib.RuleSets {
			rs := &a.Lib.RuleSets[i]
			if rs.ID == strings.TrimPrefix(id, "rules:") {
				return "⇄ " + rs.Name, func(s string) string { out, _, _ := engine.ApplyRules(s, rs.Rules); return out }
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
	score              int
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
		add("chain:"+a.Lib.Chains[i].ID, "⛓ "+a.Lib.Chains[i].Name, "动作链", "")
	}
	for i := range a.Lib.RuleSets {
		add("rules:"+a.Lib.RuleSets[i].ID, "⇄ "+a.Lib.RuleSets[i].Name, "批量替换规则", "")
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
	t := c.Theme()
	ui.Modal(c, &a.paletteOpen, func() {
		items := a.paletteItems()
		if a.paletteSel >= len(items) {
			a.paletteSel = max(0, len(items)-1)
		}
		ui.Column(c).Width(560).Gap(8).Children(func() {
			ui.Text(c, "全部动作").Bold()
			field := ui.TextInput(c, &a.paletteQuery).Placeholder("搜索清理器、动作链、规则（中文/英文/拼写片段）").AutoFocus().Label("搜索动作")
			if field.Changed() {
				a.paletteSel = 0
			}
			field.OnShortcut(0, ui.KeyDown, func() { a.paletteSel = min(a.paletteSel+1, len(items)-1) })
			field.OnShortcut(0, ui.KeyUp, func() { a.paletteSel = max(a.paletteSel-1, 0) })
			if field.Submitted() && len(items) > 0 {
				a.runPaletteItem(items[a.paletteSel])
			}
			ui.Scroll(c).Height(360).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					for i, it := range items {
						if i >= 80 {
							muted(c, "…继续输入以缩小范围")
							break
						}
						row := ui.ButtonBase(c.Key("pal:"+it.id)).Justify(ui.Start).AlignItems(ui.Stretch).Padding(7, 10).Radius(6).FillWidth()
						if i == a.paletteSel {
							row.Background(t.Selection)
						}
						row.Label(it.title).Children(func() {
							ui.Row(c).Gap(8).AlignItems(ui.Center).FillWidth().Children(func() {
								ui.Column(c).Grow(1).Gap(1).Children(func() {
									ui.Text(c, it.title).FontSize(13)
									ui.Text(c, it.sub).FontSize(11).TextColor(t.TextMuted).MaxLines(1)
								})
								if it.badge != "" {
									ui.Badge(c, it.badge)
								}
							})
						})
						if row.Clicked() {
							a.runPaletteItem(it)
						}
					}
					if len(items) == 0 {
						muted(c, "没有匹配的动作")
					}
				})
			})
			muted(c, "↑↓ 选择 · ↩ 应用到剪贴板空间文本 · Esc 关闭")
		})
	})
}
