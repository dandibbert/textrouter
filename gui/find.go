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
	key := hashOf("find", a.findText(), o.Find, o.Replace, fmt.Sprint(o.Regex, o.MatchCase), o.RegexOptions)
	return findMemo.get(key, func() any {
		ms, err := o.FindAll(a.findText(), 0)
		r := findResult{matches: ms, err: err, out: a.findText()}
		if err == nil && len(ms) > 0 {
			r.out = engine.ApplyMatches(a.findText(), ms)
		}
		return r
	}).(findResult)
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
