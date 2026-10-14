package gui

import (
	"fmt"
	"os"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

type batchState struct {
	setID     string
	list      ui.ListState
	renaming  bool
	name      string
	showDiff  bool
	pasteOpen bool
	pasteText string
}

func (a *App) currentRuleSet() *engine.RuleSet {
	for i := range a.Lib.RuleSets {
		if a.Lib.RuleSets[i].ID == a.batch.setID {
			return &a.Lib.RuleSets[i]
		}
	}
	return nil
}

type batchResult struct {
	out    string
	counts []int
	err    error
}

var batchMemo memo

func (a *App) runRules(rules []engine.Rule) batchResult {
	key := hashOf("batch", a.scopeText(), fmt.Sprintf("%+v", rules))
	return batchMemo.get(key, func() any {
		out, counts, err := engine.ApplyRules(a.scopeText(), rules)
		return batchResult{out, counts, err}
	}).(batchResult)
}

func (a *App) batchView(c *ui.Context) {
	k := tok(c)
	b := &a.batch
	rs := a.currentRuleSet()
	if rs == nil && len(a.Lib.RuleSets) > 0 {
		rs = &a.Lib.RuleSets[0]
		b.setID = rs.ID
	}
	var res batchResult
	if rs != nil {
		res = a.runRules(rs.Rules)
	}
	pageHeader(c, "批量替换", "从上到下依次执行，每条作用于上一条的结果", func() {
		a.scopePill(c, k)
		ui.Box(c).Width(sp2)
		if rs != nil && primary(c, "applybatch", "应用到文档", "play", "", res.out == a.scopeText()).Clicked() && res.out != a.scopeText() {
			rules := rs.Rules
			a.Transform("批量替换："+rs.Name, func(in string) string { out, _, _ := engine.ApplyRules(in, rules); return out })
		}
	})
	// Rule set bar.
	ui.Row(c).Padding(0, sp5, sp3, sp5).Gap(6).AlignItems(ui.Center).Children(func() {
		var names []string
		sel := ""
		for _, r := range a.Lib.RuleSets {
			names = append(names, r.Name)
			if r.ID == b.setID {
				sel = r.Name
			}
		}
		if len(names) > 0 && ui.Select(c, &sel, names).Label("规则集").Width(220).Height(30).Changed() {
			for _, r := range a.Lib.RuleSets {
				if r.Name == sel {
					b.setID = r.ID
				}
			}
		}
		if iconButton(c, "newset", "plus", "新建规则集", "新建规则集").Clicked() {
			a.Lib.RuleSets = append(a.Lib.RuleSets, engine.RuleSet{ID: engine.NewID(), Name: fmt.Sprintf("规则集 %d", len(a.Lib.RuleSets)+1),
				Rules: []engine.Rule{{MatchCase: true}}})
			b.setID = a.Lib.RuleSets[len(a.Lib.RuleSets)-1].ID
			a.save()
		}
		if rs == nil {
			return
		}
		if iconButton(c, "rename", "type", "重命名", "重命名规则集").Clicked() {
			b.name, b.renaming = rs.Name, true
		}
		ui.Spacer(c)
		if res.err != nil {
			ui.Icon(c, ic("triangle-alert")).FontSize(13).TextColor(k.red)
			ui.Text(c, res.err.Error()).FontSize(11.5).TextColor(k.red).MaxLines(1).MaxWidth(320)
		}
		ui.MenuButton(c, "导入 / 导出", func(m *ui.Menu) {
			if m.Item("从 CSV/TSV 文件导入…").Chosen() {
				a.importRules(rs)
			}
			if m.Item("从粘贴的表格导入…").Chosen() {
				b.pasteOpen, b.pasteText = true, ""
			}
			m.Separator()
			if m.Item("导出为 CSV…").Chosen() {
				a.exportRules(rs, ',')
			}
			if m.Item("导出为 TSV…").Chosen() {
				a.exportRules(rs, '\t')
			}
			if m.Item("复制为 TSV（可粘到表格软件）").Chosen() {
				if a.writeClip != nil {
					a.writeClip(string(engine.FormatRules(rs.Rules, '\t')))
					a.status = "已复制 TSV"
				}
			}
			m.Separator()
			if m.Item("删除此规则集").Chosen() {
				a.deleteRuleSet(rs.ID)
			}
		}).Height(30).Radius(rMd).Background(k.field).Border(1, ui.Transparent).FontSize(12.5)
	})
	if rs == nil {
		ui.Column(c).Grow(1).Center().Children(func() { hint(c, "还没有规则集，点「＋」新建。") })
		return
	}
	ui.Column(c).Grow(1).Padding(0, sp5, sp4, sp5).Gap(sp3).MinHeight(0).Children(func() {
		// The rule table.
		panel(c).Padding(0).Gap(0).Clip().Children(func() {
			ui.Row(c).Height(34).Gap(sp2).Padding(0, sp3, 0, sp3).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(k.hair).Children(func() {
				ui.Box(c).Width(14 + 22 + 22)
				eyebrow(c, "查找").Grow(1).Basis(0)
				ui.Box(c).Width(18)
				eyebrow(c, "替换为").Grow(1).Basis(0)
				eyebrow(c, "选项").Width(62)
				eyebrow(c, "命中").Width(56)
				ui.Box(c).Width(28)
			})
			b.list.Reorder = func(rows []int, to int) {
				rs.Rules = moveRules(rs.Rules, rows, to)
				a.save()
			}
			b.list.Key = func(i int) any { return fmt.Sprintf("%p", &rs.Rules[i]) }
			ui.List(c, &b.list, len(rs.Rules), func(i int) {
				a.ruleRow(c, rs, i, res)
			}).Height(min(float32(len(rs.Rules))*45, 315))
			ui.Row(c).Height(42).Padding(0, sp3).Gap(sp2).AlignItems(ui.Center).BorderWidth(1, 0, 0, 0).BorderColor(k.hair).Children(func() {
				add := ui.ButtonBase(c.Key("addrule")).Height(28).Padding(0, 10, 0, 8).Radius(rSm).Gap(6).AlignItems(ui.Center).Label("添加规则")
				pressable(c, add, k, ui.Transparent)
				add.Children(func() {
					ui.Icon(c, ic("plus")).FontSize(14).TextColor(k.accent)
					ui.Text(c, "添加规则").FontSize(12.5).FontWeight(600).TextColor(k.accent)
				})
				if add.Clicked() {
					rs.Rules = append(rs.Rules, engine.Rule{MatchCase: true})
					a.save()
				}
				ui.Spacer(c)
				total := 0
				for _, n := range res.counts {
					total += n
				}
				ui.Textf(c, "%d 条规则 · 共替换 %d 处 · 拖动左侧把手调整顺序", len(rs.Rules), total).FontSize(11.5).TextColor(k.faint)
			})
		})
		// Result preview.
		canvasCard(c).Grow(1).MinHeight(0).Clip().Children(func() {
			ui.Row(c).Height(44).Padding(0, sp3, 0, sp5).Gap(sp2).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(k.hair).Children(func() {
				ui.Text(c, "结果预览").FontSize(13).FontWeight(600).TextColor(k.text)
				diffBadge(c, a.scopeText(), res.out)
				ui.Spacer(c)
				idx := map[bool]int{false: 0, true: 1}[b.showDiff]
				if segmented(c, "bview", &idx, "结果", "改动对照") {
					b.showDiff = idx == 1
				}
			})
			ui.Scroll(c).Grow(1).Padding(sp5, sp6).Children(func() {
				if b.showDiff {
					diffView(c, a.scopeText(), res.out)
				} else {
					ui.Text(c, res.out).Selectable().Font(editorFont).FontSize(15).LineHeight(1.65).TextColor(k.text)
				}
			})
		})
	})
	ui.Modal(c, &b.renaming, func() {
		ui.Text(c, "重命名规则集").FontSize(15).FontWeight(650)
		in := field(c, ui.TextInput(c, &b.name).AutoFocus().Width(320))
		ui.Row(c).Gap(8).Justify(ui.End).FillWidth().Children(func() {
			if in.Submitted() || primary(c, "renameok", "确定", "", "", false).Clicked() {
				rs.Name, b.renaming = b.name, false
				a.save()
			}
		})
	})
	ui.Modal(c, &b.pasteOpen, func() {
		ui.Text(c, "粘贴表格").FontSize(15).FontWeight(650)
		hint(c, "每行一条：查找<Tab>替换<Tab>正则(1/0)<Tab>区分大小写(1/0)。也接受 CSV。可以直接从 Excel / Numbers 复制。")
		field(c, ui.TextArea(c, &b.pasteText).Size(540, 220).Font(monoFont).AutoFocus())
		ui.Row(c).Gap(8).Justify(ui.End).FillWidth().Children(func() {
			if ghost(c, "pastecancel", "取消", "", false).Clicked() {
				b.pasteOpen = false
			}
			if primary(c, "pasteok", "追加这些规则", "", "", false).Clicked() {
				rules, err := engine.ParseRules([]byte(b.pasteText))
				if err != nil {
					a.status = "解析失败：" + err.Error()
				} else {
					rs.Rules = append(rs.Rules, rules...)
					a.status = fmt.Sprintf("已追加 %d 条规则", len(rules))
					a.save()
				}
				b.pasteOpen = false
			}
		})
	})
}

func (a *App) ruleRow(c *ui.Context, rs *engine.RuleSet, i int, res batchResult) {
	k := tok(c)
	r := &rs.Rules[i]
	count := 0
	if i < len(res.counts) {
		count = res.counts[i]
	}
	valid := true
	if r.Regex && r.Find != "" {
		if _, err := r.Options().Compile(); err != nil {
			valid = false
		}
	}
	row := ui.Row(c).Height(45).Gap(sp2).Padding(0, sp3).AlignItems(ui.Center)
	if i > 0 {
		row.BorderWidth(1, 0, 0, 0).BorderColor(k.hair)
	}
	hov := row.Hovered()
	if hov {
		row.Background(k.hover)
	}
	if r.Disabled {
		row.Opacity(0.5)
	}
	row.Children(func() {
		ui.Icon(c, ic("grip-vertical")).FontSize(14).TextColor(map[bool]ui.Color{true: k.muted, false: k.faint.Alpha(0.5)}[hov]).Cursor(ui.CursorGrab)
		enabled := !r.Disabled
		if ui.Checkbox(c, &enabled, "").Label(fmt.Sprintf("启用第 %d 条", i+1)).Width(22).Changed() {
			r.Disabled = !enabled
			a.save()
		}
		ui.Textf(c, "%d", i+1).FontSize(11).FontWeight(600).TextColor(k.faint).Width(22).FontFeatures("tnum")
		in := ruleField(c, ui.TextInputBase(c, &r.Find).Placeholder("查找").Label(fmt.Sprintf("第 %d 条 查找", i+1)), !valid)
		if r.Regex && valid {
			in.TextRanges(patternRanges(k, r.Find)...)
		}
		if in.Changed() {
			a.save()
		}
		ui.Icon(c, ic("arrow-right")).FontSize(13).TextColor(k.faint).Width(18)
		rep := ruleField(c, ui.TextInputBase(c, &r.Replace).Placeholder("（删除）").Label(fmt.Sprintf("第 %d 条 替换", i+1)), false)
		if r.Regex {
			rep.TextRanges(templateRanges(k, r.Replace)...)
		}
		if rep.Changed() {
			a.save()
		}
		ui.Row(c).Width(62).Gap(2).Children(func() {
			if flag(c, "re", ".*", "按正则表达式匹配", &r.Regex) {
				a.save()
			}
			if flag(c, "cs", "Aa", "区分大小写", &r.MatchCase) {
				a.save()
			}
		})
		ui.Row(c).Width(56).Children(func() {
			switch {
			case !valid:
				badge(c, "错误", toneRed).Tooltip("正则有误")
			case count > 0:
				badge(c, fmt.Sprintf("%d 处", count), toneAccent)
			default:
				badge(c, "0 处", toneNeutral)
			}
		})
		del := iconButton(c, "del", "trash-2", fmt.Sprintf("删除第 %d 条", i+1), "删除这条规则").Size(28, 28)
		if !hov {
			del.Opacity(0.35)
		}
		if del.Clicked() {
			rs.Rules = append(rs.Rules[:i], rs.Rules[i+1:]...)
			a.save()
		}
	})
}

// ruleField styles an inline table cell editor.
func ruleField(c *ui.Context, e ui.Element, bad bool) ui.Element {
	k := tok(c)
	e.Grow(1).Basis(0).Font(monoFont).FontSize(13).TextColor(k.text).Padding(6, 8).Radius(rSm).MinWidth(0)
	switch {
	case e.Focused():
		e.Background(k.canvas).Border(1, k.accent.Alpha(0.7)).Shadow(0, 0, 0, 3, k.accent.Alpha(0.15))
	case bad:
		e.Background(k.redSoft).Border(1, k.red.Alpha(0.5))
	case e.Hovered():
		e.Background(k.fieldHover).Border(1, ui.Transparent)
	default:
		e.Background(k.field).Border(1, ui.Transparent)
	}
	return e
}

func moveRules(rules []engine.Rule, rows []int, to int) []engine.Rule {
	picked := map[int]bool{}
	var moving []engine.Rule
	for _, r := range rows {
		picked[r] = true
		moving = append(moving, rules[r])
	}
	var out []engine.Rule
	for i := 0; i <= len(rules); i++ {
		if i == to {
			out = append(out, moving...)
		}
		if i < len(rules) && !picked[i] {
			out = append(out, rules[i])
		}
	}
	return out
}

func (a *App) deleteRuleSet(id string) {
	for i, r := range a.Lib.RuleSets {
		if r.ID == id {
			a.Lib.RuleSets = append(a.Lib.RuleSets[:i], a.Lib.RuleSets[i+1:]...)
			break
		}
	}
	a.batch.setID = ""
	if len(a.Lib.RuleSets) > 0 {
		a.batch.setID = a.Lib.RuleSets[0].ID
	}
	a.save()
}

func (a *App) importRules(rs *engine.RuleSet) {
	go func() {
		paths := a.Platform.OpenFiles("导入规则（CSV/TSV）", []string{"csv", "tsv", "txt"}, false, false)
		if len(paths) == 0 {
			return
		}
		data, err := os.ReadFile(paths[0])
		a.Platform.Update(func() {
			if err != nil {
				a.status = err.Error()
				return
			}
			rules, err := engine.ParseRules(data)
			if err != nil {
				a.status = "解析失败：" + err.Error()
				return
			}
			rs.Rules = append(rs.Rules, rules...)
			a.status = fmt.Sprintf("已导入 %d 条规则", len(rules))
			a.save()
		})
	}()
}

func (a *App) exportRules(rs *engine.RuleSet, sep rune) {
	ext := "csv"
	if sep == '\t' {
		ext = "tsv"
	}
	data := engine.FormatRules(rs.Rules, sep)
	go func() {
		p := a.Platform.SaveFile("导出规则", rs.Name+"."+ext)
		if p == "" {
			return
		}
		err := os.WriteFile(p, data, 0o644)
		a.Platform.Update(func() {
			if err != nil {
				a.status = err.Error()
			} else {
				a.status = "已导出到 " + p
			}
		})
	}()
}
