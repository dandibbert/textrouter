package gui

import (
	"fmt"
	"strings"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

const inspectorW = 340

// document is one text the window edits: the clipboard workspace or a
// file. Peers: each keeps its own editor, undo history and last action.
type document struct {
	name, path string
	clip       bool
	text       string
	ed         *editor.State
	edVer      uint64
	edSynced   string
	last       *lastOp
}

type inspectorState struct {
	query    string
	group    string
	findSel  int // current match in the find panel
	inSel    bool
	selRange [2]int
	explain  bool
}

func (a *App) ensureDocs() {
	if len(a.docs) == 0 {
		a.docs = []*document{{name: "剪贴板", clip: true}}
		a.cur = 0
	}
}

func (a *App) doc() *document {
	a.ensureDocs()
	return a.docs[a.cur]
}

// switchDoc makes document i current, keeping the state of the other.
func (a *App) switchDoc(i int) {
	a.ensureDocs()
	if i == a.cur || i < 0 || i >= len(a.docs) {
		return
	}
	a.syncEditor()
	d := a.docs[a.cur]
	d.text, d.ed, d.edVer, d.edSynced, d.last = a.Text, a.ed, a.edVer, a.edSynced, a.last
	n := a.docs[i]
	a.cur = i
	a.Text, a.ed, a.edVer, a.edSynced, a.last = n.text, n.ed, n.edVer, n.edSynced, n.last
	a.changes = nil
	a.syncEditor()
}

func (a *App) closeDoc(i int) {
	if i <= 0 || i >= len(a.docs) {
		return // the clipboard workspace stays
	}
	if i == a.cur {
		a.switchDoc(0)
	}
	a.docs = append(a.docs[:i], a.docs[i+1:]...)
	if a.cur > i {
		a.cur--
	}
}

// ShowClipboard makes the clipboard workspace current.
func (a *App) ShowClipboard() { a.switchDoc(0) }

func (a *App) showFind() { a.inspector = "find" }

func (a *App) toggleInspector(tab string) {
	if a.inspector == tab {
		a.inspector = ""
	} else {
		a.inspector = tab
	}
}

// OpenTool opens a second-level tool as a sheet over the document.
func (a *App) OpenTool(page string) { a.Page = page }

// ---------------------------------------------------------------- title bar

// titleBar holds the document tabs and the toolbar (TextSoap: MyScrub,
// Copy All, Paste Over, MyLibrary), in the window's hidden title bar.
func (a *App) titleBar(c *ui.Context, k tokens) {
	bar := c.TitleBar()
	ui.Row(c).Height(44).Padding(0, sp3, 0, max(bar.Left, 0)+sp3).Gap(4).AlignItems(ui.Center).Background(k.window).BorderWidth(0, 0, 1, 0).BorderColor(k.hair).DragWindow().Children(func() {
		for i, d := range a.docs {
			a.docTab(c, k, i, d)
		}
		if b := iconButton(c, "opendoc", "plus", "打开文件", "打开文件 ⌘O").Size(26, 26); b.Clicked() {
			a.openFile()
		}
		ui.Spacer(c)
		a.scopePill(c, k)
		ui.Box(c).Width(sp2)
		if ghost(c, "scrub", "一键清理", "sparkles", false).Tooltip("应用「一键清理 Scrub」（可在设置中更换）").Clicked() {
			if cl := engine.CleanerByName("Scrub"); cl != nil {
				a.Transform(cl.ZhName, cl.Fn)
			}
		}
		if iconButton(c, "pasteover", "clipboard-paste", "粘贴覆盖", "用剪贴板内容替换全部文本").Clicked() {
			a.replaceWithUndo("粘贴覆盖", c.ReadClipboard())
		}
		if iconButton(c, "copyall", "copy", "全部复制", "全部复制到剪贴板").Clicked() {
			c.WriteClipboard(a.Text)
			a.status = "已复制全部文本"
		}
		a.toolsMenu(c, k)
		ui.Box(c).Size(1, 16).Background(k.hair).Margin(0, sp1)
		for _, tab := range []struct{ id, icon, tip string }{{"cleaners", "wand-sparkles", "清理器 ⌥⌘S"}, {"find", "text-search", "查找替换 ⌘F"}} {
			b := iconButton(c, "insp-"+tab.id, tab.icon, tab.tip, tab.tip).Size(28, 28)
			if a.inspector == tab.id {
				b.Background(k.accentSoft)
			}
			if b.Clicked() {
				a.toggleInspector(tab.id)
			}
		}
		if a.doc().clip {
			ui.Box(c).Width(sp1)
			d := ui.ButtonBase(c.Key("done")).Height(28).Padding(0, 10).Radius(rMd).Gap(6).AlignItems(ui.Center).Label("复制并完成").Tooltip("复制回剪贴板并收起窗口")
			pressable(c, d, k, ui.Transparent)
			d.Children(func() {
				ui.Text(c, "复制并完成").FontSize(12.5).FontWeight(600).TextColor(k.accent).SingleLine()
				ui.Text(c, "⌘↩").FontSize(11.5).TextColor(k.accent.Alpha(0.6))
			})
			if d.Clicked() {
				a.done()
			}
		}
	})
}

func (a *App) docTab(c *ui.Context, k tokens, i int, d *document) {
	sel := i == a.cur
	b := ui.ButtonBase(c.Key(fmt.Sprintf("doc%d:%s", i, d.path))).Height(28).Padding(0, 8, 0, 10).Radius(rMd).Gap(6).AlignItems(ui.Center).Label(d.name).Selected(sel).MaxWidth(200)
	bg := ui.Transparent
	switch {
	case sel:
		bg = k.canvas
		b.Shadow(0, 0, 0, 1, k.hair).Shadow(0, 1, 2, 0, k.shadowNear)
	case b.Hovered():
		bg = k.hover
	}
	b.Background(bg)
	b.Children(func() {
		icon := "file-text"
		if d.clip {
			icon = "clipboard-paste"
		}
		col := k.faint
		if sel {
			col = k.accent
		}
		ui.Icon(c, ic(icon)).FontSize(13).TextColor(col)
		w := 500
		if sel {
			w = 600
		}
		ui.Text(c, d.name).FontSize(12.5).FontWeight(w).TextColor(map[bool]ui.Color{true: k.text, false: k.muted}[sel]).SingleLine()
		if !d.clip {
			x := ui.ButtonBase(c.Key("x")).Size(16, 16).Radius(4).Center().Label("关闭 " + d.name)
			if x.Hovered() {
				x.Background(k.press)
			}
			x.Children(func() { ui.Icon(c, ic("x")).FontSize(11).TextColor(k.faint) })
			if x.Clicked() {
				a.closeDoc(i)
			}
		}
	})
	if b.Clicked() {
		a.switchDoc(i)
	}
}

// toolsMenu is the Tools menu: every second-level feature.
func (a *App) toolsMenu(c *ui.Context, k tokens) {
	b := ui.ButtonBase(c.Key("tools")).Height(28).Padding(0, 8, 0, 10).Radius(rMd).Gap(5).AlignItems(ui.Center).Label("工具").Tooltip("批量替换、我的清理器、清理器库、批量处理文件")
	pressable(c, b, k, ui.Transparent)
	b.Children(func() {
		ui.Text(c, "工具").FontSize(12.5).FontWeight(500).TextColor(k.text)
		ui.Icon(c, ic("chevron-down")).FontSize(12).TextColor(k.faint)
	})
	b.Menu(func(m *ui.Menu) { a.toolsMenuItems(m) })
}

func (a *App) toolsMenuItems(m *ui.Menu) {
	for _, t := range tools[:4] {
		if m.Item(t.label + "…").Chosen() {
			a.OpenTool(t.id)
		}
	}
	m.Separator()
	if m.Item("打开文件…").Chosen() {
		a.openFile()
	}
	if m.Item("存储为…").Chosen() {
		a.saveFile()
	}
	m.Separator()
	if m.Item("设置…").Chosen() {
		a.OpenTool(PageSettings)
	}
}

// replaceWithUndo replaces the whole document as one undoable action.
func (a *App) replaceWithUndo(label, s string) {
	a.syncEditor()
	before := a.Text
	if s == before {
		return
	}
	a.replaceAll(s)
	a.last = &lastOp{label: label, before: before, after: a.Text, scope: "全文"}
	a.status = "已" + label
}

// ---------------------------------------------------------------- document

func (a *App) documentView(c *ui.Context, k tokens) {
	a.syncEditor()
	a.configureEditor(c, k)
	c.OnShortcut(ui.Cmd, ui.KeyEnter, a.done)
	c.OnShortcut(ui.Cmd, ui.KeyEqual, func() { a.zoomBy(1) })
	c.OnShortcut(ui.Cmd, ui.KeyMinus, func() { a.zoomBy(-1) })
	c.OnShortcut(ui.Cmd, ui.Key0, func() { a.view.zoom = 1 })
	c.OnShortcut(ui.Cmd, ui.KeyO, a.openFile)
	c.OnShortcut(ui.Cmd, ui.KeyS, a.saveFile)
	c.OnShortcut(ui.Cmd|ui.Alt, ui.KeyI, func() { a.view.invisibles = !a.view.invisibles })
	zone := ui.Column(c.Key(fmt.Sprintf("docview%d", a.cur))).Grow(1).MinWidth(0).FillHeight().Background(k.canvas)
	if files := zone.DroppedFiles(); len(files) > 0 {
		for _, f := range files {
			a.loadFile(f)
		}
	}
	zone.Children(func() {
		ui.Row(c).Grow(1).MinHeight(0).Padding(sp2, 0, 0, sp2).Children(func() {
			editor.View(c, a.ed).Grow(1).FillHeight().AutoFocus()
		})
	})
	a.syncEditor()
}

// ---------------------------------------------------------------- inspector

func (a *App) inspectorView(c *ui.Context, k tokens) {
	ui.Column(c).Width(inspectorW).FillHeight().Background(k.window).BorderWidth(0, 0, 0, 1).BorderColor(k.hair).Children(func() {
		ui.Row(c).Padding(sp3).Children(func() {
			idx := map[string]int{"cleaners": 0, "find": 1}[a.inspector]
			ui.Row(c).Grow(1).Children(func() {
				if segmented(c, "insp", &idx, "清理器", "查找替换") {
					a.inspector = []string{"cleaners", "find"}[idx]
				}
			})
		})
		if a.inspector == "find" {
			a.findPanel(c, k)
		} else {
			a.cleanerPanel(c, k)
		}
	})
}

// cleanerPanel is TextSoap's cleaner list: groups, search, click to apply
// (to the selection, or the whole text).
func (a *App) cleanerPanel(c *ui.Context, k tokens) {
	s := &a.insp
	if s.group == "" {
		s.group = "常用"
	}
	ui.Row(c).Padding(0, sp3, sp2, sp3).Gap(6).Children(func() {
		sf := ui.Row(c).Grow(1).Height(28).Padding(0, 8).Gap(6).Radius(rMd).Background(k.field).AlignItems(ui.Center)
		sf.Children(func() {
			ui.Icon(c, ic("search")).FontSize(12).TextColor(k.faint)
			ui.TextInputBase(c, &s.query).Placeholder("搜索清理器").Label("搜索清理器").Grow(1).FontSize(12.5).TextColor(k.text)
		})
		groups := append([]string{"常用", "全部"}, engine.Categories()...)
		ui.Select(c, &s.group, groups).Label("分组").Width(104).Height(28).FontSize(12.5)
	})
	type row struct{ id, title, sub, badge string }
	var rows []row
	q := strings.ToLower(strings.TrimSpace(s.query))
	match := func(t string) bool { return q == "" || strings.Contains(strings.ToLower(t), q) }
	if s.group == "常用" && q == "" {
		for _, id := range a.pins(100) {
			label, _ := a.resolveAction(id)
			sub := "清理器"
			if strings.HasPrefix(id, "chain:") {
				sub = "我的清理器"
			}
			rows = append(rows, row{id, label, sub, ""})
		}
		for i := range a.Lib.RuleSets {
			rows = append(rows, row{"rules:" + a.Lib.RuleSets[i].ID, a.Lib.RuleSets[i].Name, "批量替换规则", ""})
		}
	} else {
		for i := range a.Lib.Chains {
			if match(a.Lib.Chains[i].Name) {
				rows = append(rows, row{"chain:" + a.Lib.Chains[i].ID, a.Lib.Chains[i].Name, "我的清理器", ""})
			}
		}
		for _, cl := range engine.Cleaners() {
			if !cl.Implemented || (s.group != "全部" && s.group != "常用" && cl.Category != s.group) {
				continue
			}
			if !match(cl.ZhName + " " + cl.Name) {
				continue
			}
			b := ""
			if !cl.Measured {
				b = "未验证"
			}
			rows = append(rows, row{cl.Name, cl.ZhName, cl.Name, b})
		}
	}
	ui.Row(c).Padding(0, sp4, sp1, sp4).Children(func() {
		ui.Textf(c, "%d 个 · 单击应用到%s", len(rows), a.Scope()).FontSize(11).TextColor(k.faint)
	})
	ui.List(c, nil, len(rows), func(i int) {
		r := rows[i]
		b := ui.ButtonBase(c.Key("cl:"+r.id)).Justify(ui.Start).AlignItems(ui.Center).Height(40).Padding(0, sp2).Radius(rMd).Gap(sp2).FillWidth().Label(r.title)
		pressable(c, b, k, ui.Transparent)
		b.Children(func() {
			ui.Icon(c, ic(actionIcon(r.id))).FontSize(14).TextColor(k.muted)
			ui.Column(c).Grow(1).Gap(0).MinWidth(0).Children(func() {
				ui.Text(c, r.title).FontSize(13).FontWeight(500).TextColor(k.text).SingleLine()
				ui.Text(c, r.sub).FontSize(10.5).TextColor(k.faint).SingleLine()
			})
			if r.badge != "" {
				ui.Box(c).Size(6, 6).Radius(3).Background(k.amber).Tooltip("未验证：未与 TextSoap 实测输出比对")
			}
		})
		if b.Clicked() {
			if label, fn := a.resolveAction(r.id); fn != nil {
				a.Transform(label, fn)
			}
		}
	}).Grow(1).Padding(0, sp2, sp2, sp2)
	ui.Row(c).Padding(sp2, sp3).BorderWidth(1, 0, 0, 0).BorderColor(k.hair).Gap(sp2).Children(func() {
		if ghost(c, "mylib", "我的清理器…", "workflow", false).Grow(1).Clicked() {
			a.OpenTool(PageChains)
		}
		if ghost(c, "lib", "清理器库…", "library-big", false).Grow(1).Clicked() {
			a.OpenTool(PageCleaners)
		}
	})
}

// findPanel is the first-level find/replace, inline next to the
// document, with the regex helper in place.
func (a *App) findPanel(c *ui.Context, k tokens) {
	f := &a.find
	s := &a.insp
	o := f.options()
	res := a.runFind(o)
	if s.findSel >= len(res.matches) {
		s.findSel = 0
	}
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Padding(0, sp3, sp3, sp3).Gap(sp2).Children(func() {
			ui.Row(c).AlignItems(ui.Center).Gap(2).Children(func() {
				re := f.modeIdx == 1
				if flag(c, "re", ".*", "正则表达式", &re) {
					f.modeIdx = map[bool]int{true: 1, false: 0}[re]
				}
				flag(c, "case", "Aa", "区分大小写", &f.opts.MatchCase)
				if f.modeIdx == 1 {
					flag(c, "ml", "^$", "多行：^ 和 $ 按行匹配（选项 m）", &f.multiline)
					flag(c, "dot", ".↵", ". 也匹配换行（选项 s）", &f.dotall)
				}
				if flag(c, "insel", "⌶", "只在选区中查找", &s.inSel) && s.inSel {
					if sels := a.selections(); len(sels) > 0 {
						s.selRange = [2]int{sels[0].Start(), sels[0].End()}
					} else {
						s.inSel = false
					}
				}
				ui.Spacer(c)
				switch {
				case res.err != nil:
					badge(c, "规则有误", toneRed)
				case o.Find != "":
					badge(c, fmt.Sprintf("%d 处", len(res.matches)), toneAccent)
				}
			})
			in := ui.TextInputBase(c, &f.opts.Find).Font(monoFont).FontSize(13.5).Label("查找").Placeholder(placeholderFind(f.modeIdx)).AutoFocus().TextColor(k.text).Padding(8, 10).Radius(rMd).Background(k.canvas).Border(1, k.hairStrong)
			if o.Regex {
				in.TextRanges(patternRanges(k, o.Find)...)
			}
			if in.Focused() {
				in.Border(1, k.accent.Alpha(0.7))
			}
			if in.Changed() {
				s.findSel = 0
			}
			if in.Submitted() && len(res.matches) > 0 {
				a.gotoMatch(res, s.findSel)
				s.findSel = (s.findSel + 1) % len(res.matches)
			}
			rep := ui.TextInputBase(c, &f.opts.Replace).Font(monoFont).FontSize(13.5).Label("替换为").Placeholder("替换为（留空 = 删除）").TextColor(k.text).Padding(8, 10).Radius(rMd).Background(k.canvas).Border(1, k.hairStrong)
			if o.Regex {
				rep.TextRanges(templateRanges(k, o.Replace)...)
			}
			if rep.Focused() {
				rep.Border(1, k.accent.Alpha(0.7))
			}
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				n := len(res.matches)
				if iconButton(c, "fprev", "arrow-up", "上一个", "上一个").Size(28, 28).Clicked() && n > 0 {
					s.findSel = (s.findSel - 1 + n) % n
					a.gotoMatch(res, s.findSel)
				}
				if iconButton(c, "fnext", "arrow-down", "下一个", "下一个 ↩").Size(28, 28).Clicked() && n > 0 {
					a.gotoMatch(res, s.findSel)
					s.findSel = (s.findSel + 1) % n
				}
				ui.Spacer(c)
				if ghost(c, "rone", "替换", "", n == 0).Tooltip("替换当前这一处，然后跳到下一处").Clicked() && n > 0 {
					a.replaceOne(o, res, s.findSel)
				}
				label := fmt.Sprintf("全部替换（%d 处）", n)
				if primary(c, "rall", label, "", "", n == 0).Clicked() && n > 0 {
					a.replaceAllFind(o)
				}
			})
			// Regex helper: presets and plain-language explanation.
			if f.modeIdx == 1 {
				h := ui.ButtonBase(c.Key("help-h")).Justify(ui.Start).AlignItems(ui.Center).Gap(6).FillWidth().Label("正则助手").Padding(sp2, 0, 0, 0)
				h.Children(func() {
					rot := h.AnimateTo("rot", map[bool]float32{true: 90, false: 0}[!s.explain], quick)
					ui.Icon(c, ic("chevron-right")).FontSize(12).TextColor(k.faint).Rotate(rot)
					eyebrow(c, "正则助手：解释与常用模式").Grow(1)
				})
				if h.Clicked() {
					s.explain = !s.explain
				}
				if !s.explain {
					ui.Column(c).Gap(sp2).Padding(sp3).Radius(rMd).Background(k.canvas).Border(1, k.hair).Children(func() {
						if o.Find == "" {
							hint(c, "输入正则后，这里逐段说明它匹配什么；也可以从下面的常用模式开始。")
						}
						explainParts(c, "f", engine.ExplainPattern(o.Find, o.Regex))
						if o.Find != "" {
							ui.Box(c).Height(1).FillWidth().Background(k.hair)
							explainParts(c, "r", engine.ExplainTemplate(o.Replace, o.Regex))
						}
						ui.Text(c, "$0 整个匹配 · $1…$9 第几组 · $U/$L 之后大写/小写 · $u/$l 下一个字 · $E 结束 · \\$ 美元符号").FontSize(11).TextColor(k.faint).LineHeight(1.5)
					})
					ui.Row(c).Gap(4).Wrap().Children(func() {
						for _, p := range engine.Presets() {
							t := ui.ButtonBase(c.Key("preset:"+p.Name)).Height(22).Padding(0, 8).Radius(999).Center().Label(p.Name).Tooltip(p.Desc + "\n" + p.Pattern)
							active := f.opts.Find == p.Pattern
							bg, col := k.field, k.muted
							if active {
								bg, col = k.accentSoft, k.accent
							} else if t.Hovered() {
								bg = k.fieldHover
							}
							t.Background(bg).Children(func() { ui.Text(c, p.Name).FontSize(11).FontWeight(500).TextColor(col).SingleLine() })
							if t.Clicked() {
								f.opts.Find, f.opts.Replace = p.Pattern, p.Replace
								if p.Name == "行首行尾空白" || p.Name == "空行" || p.Name == "每行首字母大写" {
									f.multiline = true
								}
							}
						}
					})
				}
			}
			if res.err != nil {
				ui.Text(c, res.err.Error()).FontSize(11.5).TextColor(k.red).Wrap()
			}
			if len(res.matches) > 0 {
				eyebrow(c, "匹配与替换结果").Padding(sp2, 0, 0, 0)
				ui.Column(c).Gap(1).Children(func() {
					for i, m := range res.matches {
						if i >= 200 {
							hint(c, fmt.Sprintf("…还有 %d 处", len(res.matches)-i))
							break
						}
						row := ui.ButtonBase(c.Key(fmt.Sprint("mr", i))).Justify(ui.Start).FillWidth().Radius(rSm).Label(fmt.Sprintf("第 %d 处", i+1))
						if i == s.findSel {
							row.Background(k.accentSoft)
						}
						row.Children(func() { matchRow(c, i, m) })
						if row.Clicked() {
							s.findSel = i
							a.gotoMatch(res, i)
						}
					}
				})
			}
		})
	})
}

// findBase is the text the find panel searches and its byte offset in
// the document.
func (a *App) findBase() (string, int) {
	s := &a.insp
	if s.inSel && s.selRange[1] <= len(a.Text) && s.selRange[0] < s.selRange[1] {
		return a.Text[s.selRange[0]:s.selRange[1]], s.selRange[0]
	}
	return a.Text, 0
}

// gotoMatch selects match i in the editor.
func (a *App) gotoMatch(res findResult, i int) {
	if i >= len(res.matches) || a.ed == nil {
		return
	}
	base, off := a.findBase()
	r := []rune(base)
	m := res.matches[i]
	if m.End > len(r) {
		return
	}
	st := off + len(string(r[:m.Start]))
	en := st + len(string(r[m.Start:m.End]))
	a.ed.Select(st, en)
	a.ed.ScrollTo(a.ed.Buffer.Pos(st).Line)
}

func (a *App) replaceOne(o engine.FindOptions, res findResult, i int) {
	if i >= len(res.matches) {
		return
	}
	base, off := a.findBase()
	r := []rune(base)
	m := res.matches[i]
	st := off + len(string(r[:m.Start]))
	en := st + len(string(r[m.Start:m.End]))
	a.syncEditor()
	before := a.Text
	a.ed.Buffer.Edit(editor.Edit{Start: st, End: en, Text: m.Result})
	if a.insp.inSel {
		a.insp.selRange[1] += len(m.Result) - (en - st)
	}
	a.syncEditor()
	a.last = &lastOp{label: "替换一处", before: before, after: a.Text, scope: "一处"}
	a.markChanges(before, a.Text)
	a.status = "已替换 1 处"
}

func (a *App) replaceAllFind(o engine.FindOptions) {
	a.syncEditor()
	base, off := a.findBase()
	out, n, err := o.Apply(base)
	if err != nil || n == 0 {
		return
	}
	before := a.Text
	a.ed.Buffer.Edit(editor.Edit{Start: off, End: off + len(base), Text: out})
	if a.insp.inSel {
		a.insp.selRange[1] = off + len(out)
	}
	a.syncEditor()
	a.last = &lastOp{label: "全部替换", before: before, after: a.Text, scope: "全文"}
	a.markChanges(before, a.Text)
	a.ed.DiffBase = editor.NewBuffer(before)
	a.status = fmt.Sprintf("已替换 %d 处", n)
	a.toast, a.toastUndo = fmt.Sprintf("已替换 %d 处", n), true
}

// ---------------------------------------------------------------- sheets

// sheetView shows a second-level tool as a sheet over the document.
func (a *App) sheetView(c *ui.Context, k tokens) {
	ui.Overlay(c, func() {
		scrim := ui.Box(c.Key("sheet-scrim")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(ui.RGBA(0, 0, 0, map[bool]float32{true: 0.45, false: 0.22}[k.dark]))
		if scrim.Clicked() {
			a.Page = PageClipboard
		}
		card := ui.Column(c.Key("sheet")).Absolute().Top(44).Left(48).Right(48).Bottom(36).Radius(rXl).Background(k.window).Clip().Border(1, k.hairStrong).
			Shadow(0, 30, 80, -10, ui.RGBA(0, 0, 0, map[bool]float32{true: 0.6, false: 0.3}[k.dark]))
		card.OnShortcut(0, ui.KeyEscape, func() { a.Page = PageClipboard })
		y := card.AnimateTo("y", 0, ui.Snappy)
		if a.sheetFresh != a.Page {
			a.sheetFresh = a.Page
		}
		card.Translate(0, y)
		prev := headerLead
		headerLead = func() {
			if iconButton(c, "sheet-close", "x", "关闭", "关闭 esc").Size(28, 28).Clicked() {
				a.Page = PageClipboard
			}
		}
		card.Children(func() {
			switch a.Page {
			case PageBatch:
				a.batchView(c)
			case PageChains:
				a.chainsView(c)
			case PageCleaners:
				a.cleanersView(c)
			case PageFiles:
				a.filesView(c)
			case PageSettings:
				a.settingsView(c)
			}
		})
		headerLead = prev
	})
}

func (a *App) findText() string { t, _ := a.findBase(); return t }

// Exported entry points for the app menu bar.
func (a *App) OpenFileDialog() { a.openFile() }
func (a *App) SaveFileDialog() { a.saveFile() }
func (a *App) ShowFind()       { a.showFind() }
func (a *App) OpenPalette()    { a.openPalette() }

// fakeMenuView paints a stand-in for the native popup menu under the Tools
// button. The real menu is an NSMenu that headless rendering cannot capture;
// this is used only by the snapshot test.
func (a *App) fakeMenuView(c *ui.Context, k tokens) {
	ui.Overlay(c, func() {
		bg := ui.RGBA(246, 246, 246, 0.98)
		if k.dark {
			bg = ui.RGBA(44, 44, 48, 0.98)
		}
		m := ui.Column(c.Key("fakemenu")).Absolute().Top(40).Right(170).Width(220).Padding(5, 5, 5, 5).Radius(8).
			Background(bg).Border(1, k.hairStrong).Shadow(0, 10, 30, 0, ui.RGBA(0, 0, 0, 0.25))
		m.Children(func() {
			for i, l := range a.fakeMenu {
				if l == "-" {
					ui.Box(c.Key(fmt.Sprint("fm-sep", i))).Height(1).FillWidth().Margin(5, 6, 5, 6).Background(k.hairStrong)
					continue
				}
				r := ui.Row(c.Key(fmt.Sprint("fm", i))).FillWidth().Height(22).Padding(0, 10, 0, 10).Radius(4).AlignItems(ui.Center)
				col := k.text
				if i == 0 {
					r.Background(k.accent)
					col = k.accentText
				}
				r.Children(func() { ui.Text(c, l).FontSize(13).TextColor(col) })
			}
		})
	})
}
