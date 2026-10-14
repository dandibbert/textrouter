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
	key := hashOf("find", a.scopeText(), o.Find, o.Replace, fmt.Sprint(o.Regex, o.MatchCase), o.RegexOptions)
	return findMemo.get(key, func() any {
		ms, err := o.FindAll(a.scopeText(), 0)
		r := findResult{matches: ms, err: err, out: a.scopeText()}
		if err == nil && len(ms) > 0 {
			r.out = engine.ApplyMatches(a.scopeText(), ms)
		}
		return r
	}).(findResult)
}

func (a *App) findView(c *ui.Context) {
	k := tok(c)
	f := &a.find
	o := f.options()
	res := a.runFind(o)
	pageHeader(c, "查找替换", "", func() {
		a.scopePill(c, k)
		ui.Box(c).Width(sp2)
		if ghost(c, "torule", "加入批量规则", "replace-all", o.Find == "").Clicked() && o.Find != "" {
			a.addRuleFromFind(o)
		}
		if ghost(c, "tochain", "存为动作链步骤", "workflow", o.Find == "").Clicked() && o.Find != "" {
			a.addFindToChain(o)
		}
		ui.Box(c).Width(sp1)
		if primary(c, "replaceall", fmt.Sprintf("全部替换（%d 处）", len(res.matches)), "replace-all", "", len(res.matches) == 0).Clicked() && len(res.matches) > 0 {
			a.Transform("查找替换", func(in string) string { out, _, _ := o.Apply(in); return out })
		}
	})
	ui.Row(c).Grow(1).Padding(0, sp5, sp4, sp5).Gap(sp4).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
		// Left: the form, then the plain-language explanation.
		ui.Scroll(c).Width(372).FillHeight().Children(func() {
			ui.Column(c).Gap(sp3).Padding(0, 4, 4, 0).Children(func() {
				panel(c).Gap(sp2).Children(func() {
					ui.Row(c).AlignItems(ui.Center).Children(func() {
						eyebrow(c, "查找").Grow(1)
						segmented(c, "mode", &f.modeIdx, "文字", "正则表达式")
					})
					// Pattern field with syntax colors and inline flags.
					ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
						in := ui.TextInputBase(c, &f.opts.Find).Grow(1).Font(monoFont).FontSize(14).Label("查找").Placeholder(placeholderFind(f.modeIdx)).AutoFocus().TextColor(k.text).Padding(9, 0, 9, 11)
						if o.Regex {
							in.TextRanges(patternRanges(k, o.Find)...)
						}
						fieldShell(c, in)
					}).Radius(rMd).Background(fieldBg(c, f.opts.Find)).Padding(0, 6, 0, 0)
					ui.Row(c).Gap(2).AlignItems(ui.Center).Children(func() {
						flag(c, "case", "Aa", "区分大小写", &f.opts.MatchCase)
						if f.modeIdx == 1 {
							flag(c, "ml", "^$", "多行：^ 和 $ 按行匹配（选项 m）", &f.multiline)
							flag(c, "dot", ".↵", ". 也匹配换行（选项 s）", &f.dotall)
						}
						ui.Spacer(c)
						switch {
						case res.err != nil:
							badge(c, "规则有误", toneRed)
						case o.Find != "":
							badge(c, fmt.Sprintf("%d 处", len(res.matches)), toneAccent)
						}
					})
					ui.Box(c).Height(2)
					eyebrow(c, "替换为")
					rep := ui.TextInputBase(c, &f.opts.Replace).Font(monoFont).FontSize(14).Label("替换为").Placeholder("留空 = 删除匹配内容").TextColor(k.text).Padding(9, 11)
					if o.Regex {
						rep.TextRanges(templateRanges(k, o.Replace)...)
					}
					rep.Radius(rMd).Background(fieldBg(c, f.opts.Replace))
					fieldShell(c, rep)
				})
				if f.modeIdx == 1 {
					open := a.open("presets")
					panel(c).Gap(sp2).Children(func() {
						h := ui.ButtonBase(c.Key("presets-h")).Justify(ui.Start).AlignItems(ui.Center).Gap(6).FillWidth().Label("常用模式")
						h.Children(func() {
							rot := h.AnimateTo("rot", map[bool]float32{true: 90, false: 0}[*open], quick)
							ui.Icon(c, ic("chevron-right")).FontSize(13).TextColor(k.faint).Rotate(rot)
							eyebrow(c, "常用模式").Grow(1)
							ui.Textf(c, "%d", len(engine.Presets())).FontSize(11).TextColor(k.faint)
						})
						if h.Clicked() {
							*open = !*open
						}
						if *open {
							ui.Row(c).Gap(5).Wrap().Children(func() {
								for _, p := range engine.Presets() {
									t := ui.ButtonBase(c.Key("preset:"+p.Name)).Height(24).Padding(0, 9).Radius(999).Center().Label(p.Name).Tooltip(p.Desc + "\n" + p.Pattern)
									active := f.opts.Find == p.Pattern
									bg := k.field
									if active {
										bg = k.accentSoft
									} else if t.Hovered() {
										bg = k.fieldHover
									}
									t.Background(bg)
									col := k.muted
									if active {
										col = k.accent
									}
									t.Children(func() { ui.Text(c, p.Name).FontSize(11.5).FontWeight(500).TextColor(col).SingleLine() })
									if t.Clicked() {
										f.opts.Find, f.opts.Replace = p.Pattern, p.Replace
										if p.Name == "行首行尾空白" || p.Name == "空行" || p.Name == "每行首字母大写" {
											f.multiline = true
										}
									}
								}
							})
						}
					})
				}
				// Explanation: a calm side panel, no card chrome.
				ui.Column(c).Gap(sp2).Padding(sp2, sp1, 0, sp1).Children(func() {
					ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
						ui.Icon(c, ic("info")).FontSize(13).TextColor(k.faint)
						eyebrow(c, "这个规则的意思")
					})
					if o.Find == "" {
						hint(c, "在上面输入要查找的内容，这里会逐段解释它会匹配什么。")
					}
					explainParts(c, "f", engine.ExplainPattern(o.Find, o.Regex))
					if o.Find != "" {
						ui.Box(c).Height(sp1)
						eyebrow(c, "替换为")
						explainParts(c, "r", engine.ExplainTemplate(o.Replace, o.Regex))
					}
					if o.Regex {
						ui.Box(c).Height(sp1)
						ui.Text(c, "替换记号：$0 整个匹配 · $1…$9 第几组 · $U/$L 之后全大写/小写 · $u/$l 下一个字大写/小写 · $E 结束 · \\$ 美元符号。不支持 ${1}，$$ 会变成空。").FontSize(11.5).TextColor(k.faint).LineHeight(1.5)
					}
				})
			})
		})
		// Right: live preview on the canvas.
		canvasCard(c).Grow(1).FillHeight().MinWidth(0).Clip().Children(func() {
			ui.Row(c).Height(44).Padding(0, sp3, 0, sp5).Gap(sp2).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(k.hair).Children(func() {
				switch {
				case res.err != nil:
					ui.Icon(c, ic("triangle-alert")).FontSize(14).TextColor(k.red)
					ui.Text(c, "规则有误："+res.err.Error()).FontSize(12.5).TextColor(k.red).Grow(1).SingleLine()
				case o.Find == "":
					ui.Text(c, "预览").FontSize(13).FontWeight(600).TextColor(k.text)
					ui.Text(c, "剪贴板空间里的文本").FontSize(12).TextColor(k.faint).Grow(1)
				default:
					ui.Textf(c, "找到 %d 处", len(res.matches)).FontSize(13).FontWeight(600).TextColor(k.text)
					ui.Text(c, "黄色 = 匹配 · 绿色 = 替换后").FontSize(12).TextColor(k.faint).Grow(1).SingleLine()
				}
				segmented(c, "view", &f.showResult, "高亮对照", "替换结果")
			})
			ui.Scroll(c).Grow(1).Padding(sp5, sp6).Children(func() {
				if f.showResult == 1 {
					ui.Text(c, res.out).Selectable().Font(editorFont).FontSize(15).LineHeight(1.65).TextColor(k.text)
				} else {
					matchView(c, a.scopeText(), res.matches)
				}
			})
			if len(res.matches) > 0 {
				ui.Column(c).Height(190).BorderWidth(1, 0, 0, 0).BorderColor(k.hair).Background(k.field).Children(func() {
					ui.Row(c).Padding(sp3, sp5, sp1, sp5).Children(func() { eyebrow(c, "逐条查看") })
					ui.Scroll(c).Grow(1).Padding(0, sp3, sp3, sp3).Children(func() {
						ui.Column(c).Gap(2).Children(func() {
							for i, m := range res.matches {
								if i >= 200 {
									hint(c, fmt.Sprintf("…还有 %d 处", len(res.matches)-i))
									break
								}
								matchRow(c, i, m)
							}
						})
					})
				})
			}
		})
	})
}

// fieldBg is the fill of an inline-styled field.
func fieldBg(c *ui.Context, _ string) ui.Color { return tok(c).field }

// fieldShell adds the focus ring to the field's container.
func fieldShell(c *ui.Context, in ui.Element) {}

// patternRanges colors a regex by its explanation parts.
func patternRanges(k tokens, pat string) []ui.TextRange {
	var out []ui.TextRange
	parts := engine.ExplainPattern(pat, true)
	rs := []rune(pat)
	pos := 0
	for _, p := range parts {
		pr := []rune(p.Text)
		if len(pr) == 0 {
			continue
		}
		at := indexRunes(rs, pr, pos)
		if at < 0 {
			continue
		}
		if p.Kind != "literal" {
			w := 500
			if p.Kind == "group" || p.Kind == "error" {
				w = 700
			}
			out = append(out, ui.TextRange{Start: at, End: at + len(pr), Color: tokenColor(k, p.Kind), Weight: w})
		}
		pos = at + len(pr)
	}
	return out
}

// templateRanges colors $-tokens of a replacement template.
func templateRanges(k tokens, tpl string) []ui.TextRange {
	var out []ui.TextRange
	rs := []rune(tpl)
	for i := 0; i < len(rs); i++ {
		if (rs[i] == '$' || rs[i] == '\\') && i+1 < len(rs) {
			out = append(out, ui.TextRange{Start: i, End: i + 2, Color: tokenColor(k, "group"), Weight: 700})
			i++
		}
	}
	return out
}

func indexRunes(hay, needle []rune, from int) int {
	for i := from; i+len(needle) <= len(hay); i++ {
		ok := true
		for j := range needle {
			if hay[i+j] != needle[j] {
				ok = false
				break
			}
		}
		if ok {
			return i
		}
	}
	return -1
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
	ui.RichText(c, spans...).Selectable().Font(editorFont).FontSize(15).LineHeight(1.65).TextColor(tok(c).text)
}

func matchRow(c *ui.Context, i int, m engine.Match) {
	k := tok(c)
	t := c.Theme()
	row := ui.Row(c.Key(fmt.Sprint("m", i))).Gap(sp3).Padding(6, 8).Radius(rSm).AlignItems(ui.Start)
	if row.Hovered() {
		row.Background(k.hover)
	}
	row.Children(func() {
		ui.Textf(c, "%d", i+1).FontSize(11).FontWeight(600).TextColor(k.faint).Width(22).FontFeatures("tnum").TextAlign(ui.End)
		ui.Column(c).Grow(1).Gap(3).MinWidth(0).Children(func() {
			ui.RichText(c,
				ui.Span{Text: visible(m.Groups[0].Text), Background: hlMatch(t)},
				ui.Span{Text: "  →  ", Color: k.faint},
				ui.Span{Text: visible(m.Result), Background: hlAdd(t)},
			).Font(monoFont).FontSize(12.5)
			if len(m.Groups) > 1 {
				var spans []ui.Span
				for gi, g := range m.Groups[1:] {
					v := g.Text
					if !g.Ok {
						v = "（未参与）"
					}
					spans = append(spans, ui.Span{Text: fmt.Sprintf("$%d ", gi+1), Color: k.accent, Weight: 600}, ui.Span{Text: visible(v) + "    ", Color: k.muted})
				}
				ui.RichText(c, spans...).Font(monoFont).FontSize(11.5)
			}
		})
	})
}

func explainParts(c *ui.Context, prefix string, parts []engine.Part) {
	k := tok(c)
	for i, p := range parts {
		ui.Row(c.Key(fmt.Sprint(prefix, i))).Gap(sp3).AlignItems(ui.Start).Padding(3, 0).Children(func() {
			txt := p.Text
			if txt == "" {
				txt = "∅"
			}
			col := tokenColor(k, p.Kind)
			bg := k.field
			if p.Kind == "error" {
				bg = k.redSoft
			}
			ui.Text(c, txt).Font(monoFont).FontSize(12).FontWeight(600).TextColor(col).Background(bg).Radius(5).Padding(2, 6).MaxWidth(140)
			ui.Text(c, p.Desc).FontSize(12.5).TextColor(k.text).LineHeight(1.45).Grow(1).Wrap().Padding(2, 0, 0, 0)
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
