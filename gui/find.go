package gui

import (
	"fmt"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

type findState struct {
	opts       engine.FindOptions
	modeIdx    int // 0 literal, 1 regex
	multiline  bool
	dotall     bool
	preset     string
	showResult int // 0 改动对照, 1 结果
}

func (s *findState) options() engine.FindOptions {
	o := s.opts
	o.Regex = s.modeIdx == 1
	o.RegexOptions = ""
	if s.multiline {
		o.RegexOptions += "m"
	}
	if s.dotall {
		o.RegexOptions += "s"
	}
	return o
}

type findResult struct {
	matches []engine.Match
	out     string
	err     error
}

var findMemo memo

func (a *App) runFind(o engine.FindOptions) findResult {
	key := hashOf("find", a.Text, o.Find, o.Replace, fmt.Sprint(o.Regex, o.MatchCase), o.RegexOptions)
	return findMemo.get(key, func() any {
		ms, err := o.FindAll(a.Text, 0)
		r := findResult{matches: ms, err: err, out: a.Text}
		if err == nil && len(ms) > 0 {
			r.out = engine.ApplyMatches(a.Text, ms)
		}
		return r
	}).(findResult)
}

func (a *App) findView(c *ui.Context) {
	t := c.Theme()
	f := &a.find
	o := f.options()
	res := a.runFind(o)
	header(c, "查找替换", "默认按文字查找；需要模式匹配时切到「正则」。右侧实时显示每一处会变成什么。", func() {
		if ui.Button(c, "加入批量规则").Disabled(o.Find == "").Clicked() {
			a.addRuleFromFind(o)
		}
		if ui.Button(c, "存为动作链步骤").Disabled(o.Find == "").Clicked() {
			a.addFindToChain(o)
		}
		if ui.PrimaryButton(c, fmt.Sprintf("全部替换（%d 处）", len(res.matches))).Disabled(len(res.matches) == 0).Clicked() {
			a.Transform("查找替换", func(string) string { return res.out })
		}
	})
	ui.Row(c).Grow(1).PaddingX(20).PaddingY(4).Gap(16).MinHeight(0).Children(func() {
		// Left: the form and the plain-language explanation.
		ui.Scroll(c).Width(380).FillHeight().Children(func() {
			ui.Column(c).Gap(12).Padding(0, 6, 0, 0).Children(func() {
				card(c).Children(func() {
					ui.Segmented(c, &f.modeIdx, "文字", "正则表达式").Label("查找方式")
					ui.Text(c, "查找").FontSize(12).Bold()
					ui.TextInput(c, &f.opts.Find).Font("monospace").Label("查找").Placeholder(placeholderFind(f.modeIdx)).AutoFocus()
					ui.Text(c, "替换为").FontSize(12).Bold()
					ui.TextInput(c, &f.opts.Replace).Font("monospace").Label("替换为").Placeholder("留空 = 删除匹配内容")
					ui.Row(c).Gap(12).Wrap().Children(func() {
						ui.Checkbox(c, &f.opts.MatchCase, "区分大小写")
						if f.modeIdx == 1 {
							ui.Checkbox(c, &f.multiline, "多行（^ $ 按行）").Tooltip("TextSoap 选项 m")
							ui.Checkbox(c, &f.dotall, ". 匹配换行").Tooltip("TextSoap 选项 s")
						}
					})
				})
				if f.modeIdx == 1 {
					card(c).Children(func() {
						ui.Text(c, "常用模式").FontSize(12).Bold()
						ui.Row(c).Gap(6).Wrap().Children(func() {
							for _, p := range engine.Presets() {
								if ui.Button(c.Key("preset:"+p.Name), p.Name).Tooltip(p.Desc).Clicked() {
									f.opts.Find, f.opts.Replace = p.Pattern, p.Replace
									if p.Name == "行首行尾空白" || p.Name == "空行" || p.Name == "每行首字母大写" {
										f.multiline = true
									}
								}
							}
						})
					})
				}
				card(c).Children(func() {
					ui.Text(c, "这个规则的意思").FontSize(12).Bold()
					if o.Find == "" {
						muted(c, "在上面输入要查找的内容，这里会逐段解释。")
					}
					explainParts(c, "f", engine.ExplainPattern(o.Find, o.Regex))
					if o.Find != "" {
						ui.Divider(c)
						muted(c, "替换为：")
						explainParts(c, "r", engine.ExplainTemplate(o.Replace, o.Regex))
					}
					if o.Regex {
						ui.Divider(c)
						muted(c, "替换记号：$0 整个匹配 · $1…$9 第几组 · $U/$L 之后全大写/小写 · $u/$l 下一个字大写/小写 · $E 结束 · \\$ 美元符号。不支持 ${1}，$$ 会变成空。")
					}
				})
			})
		})
		// Right: live preview.
		ui.Column(c).Grow(1).FillHeight().Gap(8).MinWidth(0).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				switch {
				case res.err != nil:
					ui.Text(c, "⚠ 规则有误："+res.err.Error()).FontSize(12).TextColor(t.Danger).Grow(1)
				case o.Find == "":
					muted(c, "预览：剪贴板空间里的文本").Grow(1)
				default:
					ui.Textf(c, "找到 %d 处", len(res.matches)).Bold().Grow(1)
				}
				ui.Segmented(c, &f.showResult, "高亮对照", "替换结果").Label("预览方式")
			})
			ui.Scroll(c).Grow(1).Background(t.Surface).Radius(8).Border(1, t.Border).Padding(12).Children(func() {
				if f.showResult == 1 {
					ui.Text(c, res.out).Selectable().Font("monospace").FontSize(13)
				} else {
					matchView(c, a.Text, res.matches)
				}
			})
			if len(res.matches) > 0 {
				ui.Text(c, "逐条查看").FontSize(12).Bold()
				ui.Scroll(c).Height(170).Children(func() {
					ui.Column(c).Gap(4).Children(func() {
						for i, m := range res.matches {
							if i >= 200 {
								muted(c, fmt.Sprintf("…还有 %d 处", len(res.matches)-i))
								break
							}
							matchRow(c, i, m)
						}
					})
				})
			}
		})
	})
}

func placeholderFind(mode int) string {
	if mode == 1 {
		return `例如 (\d+)元`
	}
	return "要查找的文字"
}

// matchView shows the text with each match highlighted and its
// replacement right after it.
func matchView(c *ui.Context, text string, ms []engine.Match) {
	t := c.Theme()
	r := []rune(text)
	if len(r) > maxPreview {
		r = r[:maxPreview]
	}
	var spans []ui.Span
	last := 0
	for _, m := range ms {
		if m.End > len(r) {
			break
		}
		spans = append(spans, ui.Span{Text: string(r[last:m.Start])})
		old := string(r[m.Start:m.End])
		if old == "" {
			old = "∅"
		}
		spans = append(spans, ui.Span{Text: visible(old), Background: hlDel(t), Strikethrough: m.Result != old})
		if m.Result != string(r[m.Start:m.End]) {
			spans = append(spans, ui.Span{Text: visible(m.Result), Background: hlAdd(t)})
		}
		last = m.End
	}
	spans = append(spans, ui.Span{Text: string(r[last:])})
	ui.RichText(c, spans...).Selectable().Font("monospace").FontSize(13)
}

func matchRow(c *ui.Context, i int, m engine.Match) {
	t := c.Theme()
	ui.Row(c.Key(fmt.Sprint("m", i))).Gap(8).Padding(6, 8).Radius(6).Background(t.Surface).AlignItems(ui.Start).Children(func() {
		ui.Textf(c, "#%d", i+1).FontSize(11).TextColor(t.TextMuted).Width(34)
		ui.Column(c).Grow(1).Gap(3).MinWidth(0).Children(func() {
			ui.RichText(c,
				ui.Span{Text: visible(m.Groups[0].Text), Background: hlMatch(t)},
				ui.Span{Text: "  →  ", Color: t.TextMuted},
				ui.Span{Text: visible(m.Result), Background: hlAdd(t)},
			).Font("monospace").FontSize(12)
			if len(m.Groups) > 1 {
				var spans []ui.Span
				for gi, g := range m.Groups[1:] {
					v := g.Text
					if !g.Ok {
						v = "（未参与）"
					}
					spans = append(spans, ui.Span{Text: fmt.Sprintf("$%d=", gi+1), Color: t.TextMuted}, ui.Span{Text: visible(v) + "   "})
				}
				ui.RichText(c, spans...).Font("monospace").FontSize(11)
			}
		})
	})
}

func explainParts(c *ui.Context, prefix string, parts []engine.Part) {
	t := c.Theme()
	for i, p := range parts {
		ui.Row(c.Key(fmt.Sprint(prefix, i))).Gap(8).AlignItems(ui.Start).Children(func() {
			bg := t.SurfaceHover
			switch p.Kind {
			case "error":
				bg = hlDel(t)
			case "token", "group":
				bg = hlMatch(t)
			case "class", "quant", "anchor":
				bg = hlAdd(t)
			}
			txt := p.Text
			if txt == "" {
				txt = "∅"
			}
			ui.Text(c, txt).Font("monospace").FontSize(12).Background(bg).Radius(4).Padding(1, 5).MaxWidth(150)
			ui.Text(c, p.Desc).FontSize(12).Grow(1).Wrap()
		})
	}
}

func (a *App) addRuleFromFind(o engine.FindOptions) {
	if len(a.Lib.RuleSets) == 0 {
		a.Lib.RuleSets = append(a.Lib.RuleSets, engine.RuleSet{ID: engine.NewID(), Name: "我的规则"})
	}
	rs := a.currentRuleSet()
	if rs == nil {
		rs = &a.Lib.RuleSets[0]
		a.batch.setID = rs.ID
	}
	rs.Rules = append(rs.Rules, engine.Rule{Find: o.Find, Replace: o.Replace, Regex: o.Regex, MatchCase: o.MatchCase})
	a.save()
	a.status = "已加入批量规则「" + rs.Name + "」"
}

func (a *App) addFindToChain(o engine.FindOptions) {
	ch := a.currentChain()
	if ch == nil {
		a.Lib.Chains = append(a.Lib.Chains, engine.Chain{ID: engine.NewID(), Name: "新动作链"})
		ch = &a.Lib.Chains[len(a.Lib.Chains)-1]
		a.chains.selID = ch.ID
	}
	ch.Steps = append(ch.Steps, engine.Step{Kind: engine.KindFind, Find: o})
	a.save()
	a.status = "已加入动作链「" + ch.Name + "」"
}
