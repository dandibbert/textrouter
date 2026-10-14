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
	key := hashOf("batch", a.Text, fmt.Sprintf("%+v", rules))
	return batchMemo.get(key, func() any {
		out, counts, err := engine.ApplyRules(a.Text, rules)
		return batchResult{out, counts, err}
	}).(batchResult)
}

func (a *App) batchView(c *ui.Context) {
	t := c.Theme()
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
	header(c, "批量替换", "每一行是一条规则：左边找什么，右边换成什么。从上到下依次执行，下一条作用于上一条的结果。拖动行可调整顺序。", func() {
		if rs != nil && ui.PrimaryButton(c, "应用到剪贴板空间").Disabled(res.out == a.Text).Clicked() {
			out := res.out
			a.Transform("批量替换："+rs.Name, func(string) string { return out })
		}
	})
	// Rule set picker row.
	ui.Row(c).PaddingX(20).Gap(8).AlignItems(ui.Center).Children(func() {
		var names []string
		sel := ""
		for _, r := range a.Lib.RuleSets {
			names = append(names, r.Name)
			if r.ID == b.setID {
				sel = r.Name
			}
		}
		if len(names) > 0 && ui.Select(c, &sel, names).Label("规则集").Width(220).Changed() {
			for _, r := range a.Lib.RuleSets {
				if r.Name == sel {
					b.setID = r.ID
				}
			}
		}
		if ui.Button(c, "新建规则集").Clicked() {
			a.Lib.RuleSets = append(a.Lib.RuleSets, engine.RuleSet{ID: engine.NewID(), Name: fmt.Sprintf("规则集 %d", len(a.Lib.RuleSets)+1),
				Rules: []engine.Rule{{MatchCase: true}}})
			b.setID = a.Lib.RuleSets[len(a.Lib.RuleSets)-1].ID
			a.save()
		}
		if rs == nil {
			return
		}
		if ui.Button(c, "重命名").Clicked() {
			b.name, b.renaming = rs.Name, true
		}
		ui.MenuButton(c, "导入/导出", func(m *ui.Menu) {
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
		})
		ui.Spacer(c)
		if res.err != nil {
			ui.Text(c, "⚠ "+res.err.Error()).FontSize(11).TextColor(t.Danger).MaxLines(2).MaxWidth(360)
		}
	})
	if rs == nil {
		ui.Column(c).Grow(1).Center().Children(func() { muted(c, "还没有规则集，点「新建规则集」开始。") })
		return
	}
	ui.Column(c).Grow(1).Padding(10, 20, 12, 20).Gap(8).MinHeight(0).Children(func() {
		// Column headings, aligned with the row layout below.
		ui.Row(c).Gap(8).PaddingX(10).Children(func() {
			ui.Text(c, "启用").FontSize(11).TextColor(t.TextMuted).Width(34)
			ui.Text(c, "查找").FontSize(11).TextColor(t.TextMuted).Grow(1).Basis(0)
			ui.Box(c).Width(16)
			ui.Text(c, "替换为").FontSize(11).TextColor(t.TextMuted).Grow(1).Basis(0)
			ui.Text(c, "正则 / 区分大小写 / 命中").FontSize(11).TextColor(t.TextMuted).Width(210)
		})
		b.list.Reorder = func(rows []int, to int) {
			rs.Rules = moveRules(rs.Rules, rows, to)
			a.save()
		}
		b.list.Key = func(i int) any { return fmt.Sprintf("%p", &rs.Rules[i]) }
		ui.List(c, &b.list, len(rs.Rules), func(i int) {
			a.ruleRow(c, rs, i, res)
		}).Gap(4).Height(min(float32(len(rs.Rules))*46+8, 300))
		ui.Row(c).Gap(8).Children(func() {
			if ui.Button(c, "＋ 添加规则").Clicked() {
				rs.Rules = append(rs.Rules, engine.Rule{MatchCase: true})
				a.save()
			}
			total := 0
			for _, n := range res.counts {
				total += n
			}
			ui.Spacer(c)
			muted(c, fmt.Sprintf("共 %d 条规则，替换 %d 处", len(rs.Rules), total))
			ui.Toggle(c, &b.showDiff, "显示改动")
		})
		ui.Scroll(c).Grow(1).Background(t.Surface).Radius(8).Border(1, t.Border).Padding(12).Children(func() {
			if b.showDiff {
				diffView(c, a.Text, res.out)
			} else {
				ui.Text(c, res.out).Selectable().Font("monospace").FontSize(13)
			}
		})
	})
	ui.Modal(c, &b.renaming, func() {
		ui.Text(c, "重命名规则集").Bold()
		if ui.TextInput(c, &b.name).AutoFocus().Width(300).Submitted() || ui.PrimaryButton(c, "确定").Clicked() {
			rs.Name, b.renaming = b.name, false
			a.save()
		}
	})
	ui.Modal(c, &b.pasteOpen, func() {
		ui.Text(c, "粘贴表格").Bold()
		muted(c, "每行一条：查找<Tab>替换<Tab>正则(1/0)<Tab>区分大小写(1/0)。也接受 CSV。可以直接从 Excel/Numbers 复制。")
		ui.TextArea(c, &b.pasteText).Size(520, 220).Font("monospace").AutoFocus()
		ui.Row(c).Gap(8).Justify(ui.End).Children(func() {
			if ui.Button(c, "取消").Clicked() {
				b.pasteOpen = false
			}
			if ui.PrimaryButton(c, "追加这些规则").Clicked() {
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
	t := c.Theme()
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
	bg := t.Surface
	if r.Disabled {
		bg = t.Background
	}
	ui.Row(c).Gap(8).Padding(5, 10).Radius(8).Background(bg).Border(1, t.Border).AlignItems(ui.Center).Children(func() {
		enabled := !r.Disabled
		if ui.Checkbox(c, &enabled, "").Label(fmt.Sprintf("启用第 %d 条", i+1)).Width(34).Changed() {
			r.Disabled = !enabled
			a.save()
		}
		in := ui.TextInput(c, &r.Find).Font("monospace").Placeholder("查找").Label(fmt.Sprintf("第 %d 条 查找", i+1)).Grow(1).Basis(0)
		if !valid {
			in.Error("正则有误")
		}
		if in.Changed() {
			a.save()
		}
		ui.Text(c, "→").TextColor(t.TextMuted).Width(16)
		if ui.TextInput(c, &r.Replace).Font("monospace").Placeholder("（删除）").Label(fmt.Sprintf("第 %d 条 替换", i+1)).Grow(1).Basis(0).Changed() {
			a.save()
		}
		ui.Row(c).Width(210).Gap(6).AlignItems(ui.Center).Children(func() {
			if ui.Toggle(c, &r.Regex, ".*").Tooltip("按正则表达式匹配").Changed() {
				a.save()
			}
			if ui.Toggle(c, &r.MatchCase, "Aa").Tooltip("区分大小写").Changed() {
				a.save()
			}
			label := fmt.Sprintf("%d 处", count)
			if !valid {
				label = "错误"
			}
			ui.Badge(c, label)
			ui.Spacer(c)
			if ui.Button(c, "✕").Label(fmt.Sprintf("删除第 %d 条", i+1)).Clicked() {
				rs.Rules = append(rs.Rules[:i], rs.Rules[i+1:]...)
				a.save()
			}
		})
	})
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
