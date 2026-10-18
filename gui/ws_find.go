package gui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// ---------------------------------------------------------------- state

// myPatternsID is the rule set that holds the user's saved patterns, each a
// rule whose note is the pattern's name.
const myPatternsID = "my-patterns"

type findUI struct {
	cur     int
	skip    map[int]bool
	sig     string // what the current matches were found with
	nextCur int    // where to point after a text change, -1 for the first
	explain bool
	explAt  time.Time

	errMsg string
	errAt  time.Time

	jumpAt        time.Time
	tickY         *float32
	tickFrom      float32
	scrollPending bool
	scrollAt      time.Time
	scrollFrom    float64

	list ui.ListState

	counter   string
	counterAt time.Time
	inclText  string
	inclAt    time.Time

	pvSig string
	pv    *preview

	pat patState

	// scope limits find/replace to the selection the editor had when the tab
	// opened, in runes; scoped says it is on.
	scope  [2]int
	scoped bool
}

func (f *findUI) reset() {
	f.cur, f.skip, f.sig, f.nextCur = 0, nil, "", -1
}

// patState is the pattern menu.
type patState struct {
	open      bool
	openAt    time.Time
	closing   bool
	hover     string // the item under the pointer: its matches preview
	frame     string
	applied   string // the name of the pattern in use
	appliedAt time.Time
	saved     *savedFind
}

// savedFind is what the find boxes held before a pattern took them over.
type savedFind struct {
	find, replace  string
	regex, matchCs bool
	multiline      bool
}

type patItem struct {
	name, desc, pattern, replace string
	mine                         bool
}

// patternItems lists the built-in patterns and the user's own.
func (a *App) patternItems() (builtin, mine []patItem) {
	for _, p := range engine.Presets() {
		builtin = append(builtin, patItem{name: p.Name, desc: p.Desc, pattern: p.Pattern, replace: p.Replace})
	}
	for i := range a.Lib.RuleSets {
		if a.Lib.RuleSets[i].ID != myPatternsID {
			continue
		}
		for _, r := range a.Lib.RuleSets[i].Rules {
			mine = append(mine, patItem{name: orDefault(r.Note, cut(r.Find, 8)), pattern: r.Find, replace: r.Replace, mine: true})
		}
	}
	return
}

func (a *App) patternNamed(name string) (patItem, bool) {
	b, m := a.patternItems()
	for _, p := range append(b, m...) {
		if p.name == name {
			return p, true
		}
	}
	return patItem{}, false
}

// ---------------------------------------------------------------- results

type findInfo struct {
	res   findResult
	lines []int // line of each match
}

// findNow is the matches of the find boxes in the text, found once per change.
func (a *App) findNow() *findInfo {
	o := a.find.options()
	sig := fmt.Sprint(o.Find, "\x00", o.Regex, o.MatchCase, o.RegexOptions)
	f0 := &a.ws.find
	if f0.scoped {
		sig += fmt.Sprint("\x00scope", f0.scope)
	}
	key := "f:" + sig + "\x00" + o.Replace
	if v, ok := a.ws.cache[key]; ok {
		return v.(*findInfo)
	}
	ms, err := o.FindAll(a.Text, 0)
	if f0.scoped {
		var in []engine.Match
		for _, m := range ms {
			if m.Start >= f0.scope[0] && m.End <= f0.scope[1] {
				in = append(in, m)
			}
		}
		ms = in
	}
	fi := &findInfo{res: findResult{matches: ms, err: err, out: a.Text}}
	r := []rune(a.Text)
	line, at := 0, 0
	for _, m := range ms {
		for ; at < m.Start && at < len(r); at++ {
			if r[at] == '\n' {
				line++
			}
		}
		fi.lines = append(fi.lines, line)
	}
	a.ws.cache[key] = fi
	// New matches: start again from the first (or from where a replacement
	// left off), and forget which were unchecked.
	f := &a.ws.find
	full := sig + fmt.Sprint(a.ws.textKey)
	if f.sig != full {
		f.sig, f.skip = full, nil
		f.cur = 0
		if f.nextCur > 0 {
			f.cur = f.nextCur
		}
		f.nextCur = -1
	}
	if f.cur >= len(ms) {
		f.cur = max(len(ms)-1, 0)
	}
	return fi
}

// included counts the matches that will be replaced: those left checked.
func (a *App) included(fi *findInfo) int {
	n := 0
	for i := range fi.res.matches {
		if !a.ws.find.skip[i] {
			n++
		}
	}
	return n
}

// findShown is the preview of find/replace for the body: the text with every
// match struck through and its replacement after it, or, while the pointer is
// on a pattern, only that pattern's matches.
func (a *App) findShown() *preview {
	f := &a.ws.find
	if f.pat.hover != "" {
		if p, ok := a.patternNamed(f.pat.hover); ok {
			o := engine.FindOptions{Find: p.pattern, Replace: p.replace, Regex: true}
			key := "pp:" + p.name
			if v, ok := a.ws.cache[key]; ok {
				return v.(*preview)
			}
			ms, err := o.FindAll(a.Text, 0)
			if err != nil || len(ms) == 0 {
				return nil
			}
			pv := findPreview(key, a.Text, ms, nil, true, -1)
			pv.label, pv.delta = p.name, fmt.Sprintf("%d 处匹配", len(ms))
			a.ws.cache[key] = pv
			return pv
		}
	}
	o := a.find.options()
	if o.Find == "" {
		return nil
	}
	fi := a.findNow()
	if fi.res.err != nil || len(fi.res.matches) == 0 {
		return nil
	}
	sig := fmt.Sprint(a.ws.textKey, "\x00", f.sig, o.Replace, f.cur, len(f.skip))
	for i := range f.skip {
		sig += fmt.Sprint(",", i)
	}
	if f.pv != nil && f.pvSig == sig {
		return f.pv
	}
	f.pv = findPreview("find", a.Text, fi.res.matches, func(i int) bool { return f.skip[i] }, false, f.cur)
	f.pvSig = sig
	return f.pv
}

func (a *App) findStatus() string {
	o := a.find.options()
	if o.Find == "" {
		return "输入要查找的文字"
	}
	fi := a.findNow()
	f := &a.ws.find
	switch {
	case fi.res.err != nil:
		return "规则有误"
	case len(fi.res.matches) == 0:
		return "没有匹配"
	}
	s := fmt.Sprintf("第 %d / %d 处 · 第 %d 行", f.cur+1, len(fi.res.matches), fi.lines[f.cur]+1)
	if n := len(f.skip); n > 0 {
		s += fmt.Sprintf(" · 跳过 %d", n)
	}
	return s
}

// ---------------------------------------------------------------- jumping

// jumpTo makes match i the current one: the text, the list, the line number
// and the scroll-bar tick all follow.
func (a *App) jumpTo(i int) {
	f := &a.ws.find
	fi := a.findNow()
	n := len(fi.res.matches)
	if n == 0 {
		return
	}
	i = (i%n + n) % n
	f.cur = i
	f.jumpAt = time.Now()
	f.scrollPending, f.scrollAt = true, time.Time{}
	f.list.ScrollTo(i, ui.Center)
}

// NextMatch and PrevMatch step through the matches (⌘G, ⇧⌘G).
func (a *App) NextMatch() {
	if a.ws.tab != 1 {
		a.setTab(1)
	}
	a.jumpTo(a.ws.find.cur + 1)
}

func (a *App) PrevMatch() {
	if a.ws.tab != 1 {
		a.setTab(1)
	}
	a.jumpTo(a.ws.find.cur - 1)
}

// setTab switches the right column to 0 清理 or 1 查找替换.
func (a *App) setTab(i int) {
	w := &a.ws
	if w.tab == i {
		return
	}
	w.tabDir = 1
	if i == 0 {
		w.tabDir = -1
	}
	w.tab, w.tabAt = i, time.Now()
	if i == 1 {
		a.captureScope()
	}
	w.hov, w.frameHov = "", ""
	w.find.pat.hover = ""
	if i == 1 {
		w.focusFnd = true
	}
	a.inspector = []string{"cleaners", "find"}[i]
}

// ---------------------------------------------------------------- replacing

// runeToByte returns the byte offset of rune offset r in s.
func runeToByte(s string, r int) int {
	n := 0
	for i := range s {
		if n == r {
			return i
		}
		n++
	}
	return len(s)
}

// commitFind records an edit made by find/replace as a step of the trail.
func (a *App) commitFind(label string, act *engine.Step, before string) {
	a.syncEditor()
	a.last = &lastOp{label: label, before: before, after: a.Text, scope: "全文"}
	a.markChanges(before, a.Text)
	a.pushStep(label, act)
	a.ws.startApplyFx(before, a.Text)
}

// replaceCurrent replaces the current match and points at the next one.
func (a *App) replaceCurrent() {
	f := &a.ws.find
	fi := a.findNow()
	if fi.res.err != nil || len(fi.res.matches) == 0 {
		return
	}
	a.catchUp()
	m := fi.res.matches[f.cur]
	st, en := runeToByte(a.Text, m.Start), runeToByte(a.Text, m.End)
	before := a.Text
	a.ed.Buffer.Edit(editor.Edit{Start: st, End: en, Text: m.Result})
	f.nextCur = f.cur
	a.commitFind("替换一处", nil, before)
	a.status = "已替换 1 处"
}

// replaceAllIncluded replaces every match that is still checked.
func (a *App) replaceAllIncluded() {
	fi := a.findNow()
	f := &a.ws.find
	if fi.res.err != nil {
		return
	}
	var ms []engine.Match
	for i, m := range fi.res.matches {
		if !f.skip[i] {
			ms = append(ms, m)
		}
	}
	if len(ms) == 0 {
		return
	}
	a.catchUp()
	before := a.Text
	a.replaceAll(engine.ApplyMatches(a.Text, ms))
	var act *engine.Step
	if len(f.skip) == 0 {
		o := a.find.options()
		act = &engine.Step{Kind: engine.KindFind, Find: o}
	}
	label := "替换「" + cut(a.find.opts.Find, 8) + "」"
	f.nextCur = -1
	a.commitFind(label, act, before)
	a.status = fmt.Sprintf("已替换 %d 处", len(ms))
}

// ---------------------------------------------------------------- patterns

// applyPattern puts a pattern into the find boxes, remembering what they held
// so that the pattern can be taken off again.
func (a *App) applyPattern(p patItem) {
	f := &a.ws.find
	if f.pat.applied == p.name {
		a.clearPattern()
		return
	}
	if f.pat.saved == nil {
		o := a.find.opts
		f.pat.saved = &savedFind{find: o.Find, replace: o.Replace, regex: a.find.modeIdx == 1, matchCs: o.MatchCase, multiline: a.find.multiline}
	}
	a.find.opts.Find, a.find.opts.Replace = p.pattern, p.replace
	a.find.modeIdx = 1
	a.find.multiline = strings.Contains(p.pattern, "^") || strings.Contains(p.pattern, "$")
	f.pat.applied, f.pat.appliedAt = p.name, time.Now()
	f.reset()
}

// clearPattern takes the applied pattern off: the boxes get back what they
// held (also when the user edits the find box by hand, the tag just goes).
func (a *App) clearPattern() {
	f := &a.ws.find
	if s := f.pat.saved; s != nil {
		a.find.opts.Find, a.find.opts.Replace = s.find, s.replace
		a.find.modeIdx = map[bool]int{true: 1, false: 0}[s.regex]
		a.find.opts.MatchCase, a.find.multiline = s.matchCs, s.multiline
	}
	f.pat.applied, f.pat.saved = "", nil
	f.reset()
}

func (a *App) saveCurrentPattern() {
	o := a.find.options()
	if o.Find == "" {
		return
	}
	var rs *engine.RuleSet
	for i := range a.Lib.RuleSets {
		if a.Lib.RuleSets[i].ID == myPatternsID {
			rs = &a.Lib.RuleSets[i]
		}
	}
	if rs == nil {
		a.Lib.RuleSets = append(a.Lib.RuleSets, engine.RuleSet{ID: myPatternsID, Name: "我的模式"})
		rs = &a.Lib.RuleSets[len(a.Lib.RuleSets)-1]
	}
	name := cut(o.Find, 8)
	rs.Rules = append(rs.Rules, engine.Rule{Find: o.Find, Replace: o.Replace, Regex: o.Regex, MatchCase: o.MatchCase, Note: name})
	a.save()
	a.showToast("已存为我的模式「"+name+"」", "在“模式”菜单的“我的模式”里")
}

// ---------------------------------------------------------------- view

// shake is the horizontal offset of the error box at progress p of its
// 380ms: −5, 5, −3, 2, then still.
func shake(p float32) float32 {
	keys := [][2]float32{{0, 0}, {.2, -5}, {.4, 5}, {.6, -3}, {.8, 2}, {1, 0}}
	for i := 1; i < len(keys); i++ {
		if p <= keys[i][0] {
			a, b := keys[i-1], keys[i]
			return a[1] + (b[1]-a[1])*(p-a[0])/(b[0]-a[0])
		}
	}
	return 0
}

// findTab is the 查找替换 tab.
func (a *App) findTab(c *ui.Context, k tokens) {
	f := &a.ws.find
	fo := &a.find
	o := fo.options()
	fi := a.findNow()
	res := fi.res
	n := len(res.matches)
	// A hand edit of the find box drops the pattern tag.
	if f.pat.applied != "" {
		if p, ok := a.patternNamed(f.pat.applied); !ok || p.pattern != fo.opts.Find {
			f.pat.applied, f.pat.saved = "", nil
		}
	}
	hasErr := res.err != nil && o.Find != ""
	if hasErr {
		msg := "正则写法有误：" + strings.TrimPrefix(res.err.Error(), "error parsing regexp: ")
		if f.errMsg != msg {
			f.errMsg, f.errAt = msg, time.Now()
		}
	} else {
		f.errMsg = ""
	}
	locked := f.pat.applied != ""

	ui.Column(c.Key("findtab")).Grow(1).MinHeight(0).Gap(12).Children(func() {
		a.findBoxes(c, k, locked)
		a.findToggles(c, k, n)
		if hasErr {
			shk := shake(anim(c, f.errAt, dShake, nil))
			ui.Row(c.Key("finderr")).Gap(8).AlignItems(ui.Start).Padding(8, 10).Radius(radiusCtl).Background(Del).Border(strokeW, Ink).Translate(shk, 0).Children(func() {
				ui.Text(c, "!").FontSize(12).FontWeight(700).TextColor(Ink)
				ui.Text(c, f.errMsg).FontSize(12).TextColor(Ink).Wrap().Grow(1).MinWidth(0)
			})
		}
		if fo.modeIdx == 1 && n > 0 && !hasErr {
			a.groupChips(c, res.matches[f.cur])
		}
		if f.explain {
			a.explainBox(c, o)
		}
		ui.Row(c).FillWidth().Padding(2, 2, 0, 2).AlignItems(ui.End).Children(func() {
			eyebrow(c, "匹配")
			ui.Spacer(c)
			ui.Text(c, "勾掉的不替换").FontSize(sizeSmall).TextColor(Muted)
		})
		a.matchList(c, k, fi, o)
		a.findButtons(c, k, fi, o)
	})
}

// findBoxes are the 查找 / 替换 boxes, joined in one frame.
func (a *App) findBoxes(c *ui.Context, k tokens, locked bool) {
	f := &a.ws.find
	w := &a.ws
	fo := &a.find
	frame := ui.Column(c.Key("findframe")).FillWidth().Background(Paper).Radius(radiusCtl).Border(strokeW, Ink)
	focused := frame.FocusWithin()
	t := frame.AnimateWith("lift", map[bool]float32{true: -2, false: 0}[focused || locked], dRelease, EaseSpring)
	frame.Translate(t, t)
	if locked {
		frame.Shadow(5, 5, 0, 2, Ink).Shadow(5, 5, 0, 0, Accent)
	} else {
		hard(frame, shadowSm-t)
	}
	frame.Children(func() {
		ui.Row(c.Key("findrow")).FillWidth().MinHeight(42).Padding(4, 6, 4, 12).Gap(8).AlignItems(ui.Center).BorderWidth(0, 0, strokeW, 0).BorderColor(Ink).Children(func() {
			ui.Text(c, "查找").FontSize(sizeSmall).FontWeight(700).TextColor(Ink).Width(28)
			if locked {
				p := anim(c, f.pat.appliedAt, dPop, EaseSpring)
				tag := ui.Row(c.Key("pattag")).Height(22).Padding(0, 2, 0, 8).Gap(2).AlignItems(ui.Center).Radius(radiusPill).Background(Ink)
				tag.Scale(0.6+0.4*p, 0.6+0.4*p).Opacity(p)
				tag.Children(func() {
					ui.Text(c, f.pat.applied).FontSize(11.5).FontWeight(700).TextColor(Accent).SingleLine()
					x := ui.ButtonBase(c.Key("patx")).Size(18, 18).Radius(9).Center().Label("取消这个模式，恢复原来的规则").Tooltip("取消这个模式，恢复原来的规则")
					if x.Hovered() {
						x.Background(Accent.Alpha(0.25))
					}
					x.Children(func() { ui.Text(c, "×").FontSize(13).FontWeight(700).TextColor(Accent) })
					if x.Clicked() {
						a.clearPattern()
					}
				})
			}
			in := ui.TextInputBase(c, &fo.opts.Find).Font(monoFont).FontSize(sizeBody).Label("查找").Placeholder(placeholderFind(fo.modeIdx)).
				Grow(1).MinWidth(0).TextColor(Ink)
			if fo.modeIdx == 1 {
				in.TextRanges(patternRanges(k, fo.opts.Find)...)
			}
			if w.focusFnd {
				in.Focus()
				w.focusFnd = false
			}
			if in.Changed() {
				f.reset()
			}
			if in.Submitted() && w.tab == 1 {
				a.NextMatch()
			}
			mb := ui.ButtonBase(c.Key("patbtn")).Height(26).Padding(0, 8).Gap(4).AlignItems(ui.Center).Radius(radiusCtl).Label("常用模式")
			bg := Transparent2()
			if f.pat.open || mb.Hovered() {
				bg = AccentSoft
			}
			mb.Background(bg)
			focusRing(mb)
			mb.Children(func() {
				ui.Text(c, "模式").FontSize(sizeButton).FontWeight(500).TextColor(Ink).SingleLine()
				rot := mb.AnimateWith("chev", map[bool]float32{true: 180, false: 0}[f.pat.open], dMenuIn, EaseSpring)
				ui.Icon(c, ic("chevron-down")).FontSize(11).TextColor(Ink).Rotate(rot)
			})
			if mb.Clicked() {
				a.togglePatternMenu()
			}
		})
		ui.Row(c.Key("reprow")).FillWidth().MinHeight(42).Padding(4, 12).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "替换").FontSize(sizeSmall).FontWeight(700).TextColor(Ink).Width(28)
			rep := ui.TextInputBase(c, &fo.opts.Replace).Font(monoFont).FontSize(sizeBody).Label("替换为").Placeholder("留空 = 删除匹配").
				Grow(1).MinWidth(0).TextColor(Ink)
			if fo.modeIdx == 1 {
				rep.TextRanges(templateRanges(k, fo.opts.Replace)...)
			}
		})
	})
}

// Transparent2 is a clear colour for backgrounds that only show on hover.
func Transparent2() ui.Color { return ui.Transparent }

func (a *App) togglePatternMenu() {
	p := &a.ws.find.pat
	if p.open && !p.closing {
		p.closing, p.openAt = true, time.Now()
		return
	}
	p.open, p.closing, p.openAt = true, false, time.Now()
}

// findToggles is the row of 正则 / Aa / 解释 switches and the ↑ 3 / 12 ↓ stepper.
func (a *App) findToggles(c *ui.Context, k tokens, n int) {
	f := &a.ws.find
	fo := &a.find
	ui.Row(c.Key("findtoggles")).FillWidth().Gap(6).AlignItems(ui.Center).Children(func() {
		re := fo.modeIdx == 1
		rb := ui.ButtonBase(c.Key("re")).Height(28).Padding(0, 10).Gap(5).AlignItems(ui.Center).Radius(radiusPill).Label("正则表达式").Tooltip("按正则表达式匹配").Selected(re)
		bg, col := Paper, Ink
		switch {
		case re:
			bg, col = Ink, Accent
		case rb.Hovered():
			bg = AccentSoft
		}
		rb.Background(rb.AnimateColor("bg", bg, ui.Animation{Duration: 140e6})).Border(strokeW, Ink)
		focusRing(rb)
		rb.Children(func() {
			ui.Text(c, ".*").Font(monoFont).FontSize(12).FontWeight(700).TextColor(col)
			ui.Text(c, "正则").FontSize(12).TextColor(col)
		})
		if rb.Clicked() {
			fo.modeIdx = map[bool]int{true: 0, false: 1}[re]
			f.reset()
		}
		if flag(c, "case", "Aa", "区分大小写", &fo.opts.MatchCase) {
			f.reset()
		}
		ex := f.explain
		if flag(c, "explain", "解释", "用中文解释这条规则", &ex) {
			f.explain, f.explAt = ex, time.Now()
		}
		if f.scoped {
			sb := ui.ButtonBase(c.Key("scope")).Height(28).Padding(0, 8).Gap(4).AlignItems(ui.Center).Radius(radiusPill).Label("只在选区中查找，点击取消").Tooltip("只在选区中查找 · 点击取消限制")
			sb.Background(AccentSoft).Border(strokeW, Ink)
			focusRing(sb)
			sb.Children(func() { ui.Text(c, "仅选区 ×").FontSize(12).FontWeight(700).TextColor(Ink).SingleLine() })
			if sb.Clicked() {
				f.scoped = false
				f.reset()
			}
		}
		ui.Spacer(c)
		prev := iconButton(c, "prev", "arrow-up", "上一处 ⇧⌘G", "上一处 ⇧⌘G").Size(28, 28).Disabled(n == 0)
		if prev.Clicked() {
			a.jumpTo(f.cur - 1)
		}
		cnt := "0 / 0"
		if n > 0 {
			cnt = fmt.Sprintf("%d / %d", f.cur+1, n)
		}
		if cnt != f.counter {
			f.counter, f.counterAt = cnt, time.Now()
		}
		p := anim(c, f.counterAt, dFlip, EaseOut)
		ui.Row(c.Key("counter")).Width(54).Height(20).Clip().Center().Children(func() {
			ui.Text(c, cnt).Font(monoFont).FontSize(12).FontWeight(700).TextColor(Ink).SingleLine().Translate(0, 12*(1-p)).Opacity(p)
		})
		next := iconButton(c, "next", "arrow-down", "下一处 ⌘G", "下一处 ⌘G").Size(28, 28).Disabled(n == 0)
		if next.Clicked() {
			a.jumpTo(f.cur + 1)
		}
	})
}

// groupChips lists the capture groups of the current match; a click puts the
// group's $n into the replacement.
func (a *App) groupChips(c *ui.Context, m engine.Match) {
	if len(m.Groups) == 0 {
		return
	}
	f := &a.ws.find
	ui.Row(c.Key("groups")).FillWidth().Gap(6).Wrap().AlignItems(ui.Center).Children(func() {
		ui.Text(c, fmt.Sprintf("第 %d 处的组", f.cur+1)).FontSize(11.5).TextColor(Muted).SingleLine()
		for gi, g := range m.Groups {
			if gi >= 5 {
				break
			}
			val := "（未匹配）"
			if g.Ok {
				val = visibleMatch(g.Text)
				if r := []rune(val); len(r) > 18 {
					val = string(r[:18]) + "…"
				}
			}
			key := fmt.Sprintf("$%d", gi)
			b := ui.ButtonBase(c.Key("grp"+key)).Height(24).Padding(0, 8).Gap(6).AlignItems(ui.Center).Radius(radiusPill).Label("插入 " + key).Tooltip("点击插入到替换框").MaxWidth(300)
			bg := Paper
			if b.Hovered() {
				bg = AccentSoft
			}
			b.Background(bg).Border(strokeW, Ink)
			focusRing(b)
			b.Children(func() {
				ui.Text(c, key).Font(monoFont).FontSize(11).FontWeight(700).TextColor(Ink)
				ui.Text(c, val).Font(monoFont).FontSize(11).TextColor(Ink).SingleLine()
			})
			if b.Clicked() {
				a.find.opts.Replace += key
			}
		}
	})
}

// visibleMatch makes line breaks and tabs in a match readable.
func visibleMatch(s string) string {
	return strings.NewReplacer("\n", "⏎", "\t", "→").Replace(s)
}

// explainBox is the plain-language reading of the rule: one row per token of
// the pattern (light yellow) and of the replacement (green).
func (a *App) explainBox(c *ui.Context, o engine.FindOptions) {
	f := &a.ws.find
	p := anim(c, f.explAt, dExpand, EaseOut)
	type row struct {
		tok, txt string
		bg       ui.Color
	}
	var rows []row
	for _, pt := range engine.ExplainPattern(o.Find, o.Regex) {
		bg := AccentSoft
		if pt.Kind == "error" {
			bg = Del
		}
		rows = append(rows, row{pt.Text, pt.Desc, bg})
	}
	for _, pt := range engine.ExplainTemplate(o.Replace, o.Regex) {
		bg := Ins
		if pt.Kind == "error" {
			bg = Del
		}
		rows = append(rows, row{pt.Text, pt.Desc, bg})
	}
	h := min(132, float32(len(rows))*22+20)
	sc := ui.Scroll(c.Key("explain")).FillWidth().Height(h * p).MinHeight(0).Radius(radiusCtl).Background(Paper)
	dashed(sc).Children(func() {
		ui.Column(c).FillWidth().Gap(5).Padding(10, 12).Children(func() {
			for i, r := range rows {
				rp := anim(c, f.explAt.Add(time.Duration(i)*dExpStep), 220*time.Millisecond, EaseOut)
				ui.Row(c.Key(fmt.Sprint("ex", i))).Gap(10).AlignItems(ui.Start).Opacity(rp).Translate(0, 4*(1-rp)).Children(func() {
					tok := r.tok
					if tok == "" {
						tok = "∅"
					}
					ui.Text(c, tok).Font(monoFont).FontSize(sizeSmall).TextColor(Ink).Background(r.bg).Border(keycapW, Ink).Radius(radiusKey).
						Padding(0, 5).MinWidth(30).MaxWidth(120).SingleLine().TextAlign(ui.Center)
					ui.Text(c, r.txt).FontSize(12).TextColor(Ink).Wrap().Grow(1).MinWidth(0)
				})
			}
		})
	})
}

// checkbox is the 18px square: yellow with a black tick that springs in.
func checkbox(c *ui.Context, key string, on bool) ui.Element {
	b := ui.ButtonBase(c.Key(key)).Size(18, 18).Radius(radiusKey).Center().Label("替换这一处").Selected(on)
	bg := Paper
	if on {
		bg = Accent
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 140e6})).Border(strokeW, Ink)
	press := b.AnimateWith("p", map[bool]float32{true: 0.88, false: 1}[b.Pressed()], 120*time.Millisecond, EaseOut)
	b.Scale(press, press)
	focusRing(b)
	t := b.AnimateWith("tick", map[bool]float32{true: 1, false: 0}[on], dCheck, EaseSpring)
	b.Children(func() {
		ui.Icon(c, ic("check")).FontSize(12).TextColor(Ink).Scale(t, t)
	})
	return b
}

// matchList is the list of matches, one 32px row each, at most 300.
func (a *App) matchList(c *ui.Context, k tokens, fi *findInfo, o engine.FindOptions) {
	f := &a.ws.find
	res := fi.res
	n := min(len(res.matches), 300)
	if o.Find == "" || res.err != nil || len(res.matches) == 0 {
		a.noMatches(c, o, res.err != nil)
		return
	}
	r := []rune(a.Text)
	ui.List(c, &f.list, n, func(i int) {
		m := res.matches[i]
		skip := f.skip[i]
		cur := i == f.cur
		row := ui.Row(c.Key(fmt.Sprint("m", i))).FillWidth().Height(32).Padding(0, 6).Gap(8).AlignItems(ui.Center).Radius(radiusCtl)
		bg, bd := ui.Transparent, ui.Transparent
		switch {
		case cur:
			bg, bd = Accent, Ink
		case row.Hovered():
			bg = AccentWash
		}
		row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 160e6})).Border(strokeW, row.AnimateColor("bd", bd, ui.Animation{Duration: 160e6}))
		row.Children(func() {
			if checkbox(c, fmt.Sprint("ck", i), !skip).Clicked() {
				if f.skip == nil {
					f.skip = map[int]bool{}
				}
				if skip {
					delete(f.skip, i)
				} else {
					f.skip[i] = true
				}
				f.counterAt = time.Time{}
			}
			pick := ui.ButtonBase(c.Key(fmt.Sprint("pick", i))).Grow(1).MinWidth(0).FillHeight().Gap(8).AlignItems(ui.Center).Justify(ui.Start).Label(fmt.Sprintf("第 %d 处", i+1))
			focusRing(pick)
			pick.Children(func() {
				ui.Text(c, fmt.Sprintf("L%d", fi.lines[i]+1)).Font(monoFont).FontSize(sizeSmall).TextColor(Muted).Width(28).SingleLine()
				before, hit, after := contextOf(r, m)
				hitBg, hitCol := Del, Ink
				if skip {
					hitBg, hitCol = ui.Transparent, Faint
				}
				ui.RichText(c,
					ui.Span{Text: before, Color: Muted},
					ui.Span{Text: hit, Background: hitBg, Color: hitCol},
					ui.Span{Text: after, Color: Muted},
				).Font(monoFont).FontSize(sizeSmall).SingleLine().Grow(1).MinWidth(0)
				to := "删除"
				if m.Result != "" {
					to = "→ " + visibleMatch(m.Result)
				}
				t := ui.Text(c, to).Font(monoFont).FontSize(sizeSmall).TextColor(Ink).Background(Ins).Radius(3).Padding(0, 4).MaxWidth(92).SingleLine()
				if skip {
					t.Opacity(0.45).Strikethrough()
				}
			})
			if pick.Clicked() {
				a.jumpTo(i)
			}
		})
	}).Grow(1).MinHeight(0).Margin(0, -4).Padding(0, 2)
	if len(res.matches) > n {
		hint(c, fmt.Sprintf("还有 %d 处没有列出（全部替换仍然会替换）", len(res.matches)-n)).FontSize(11)
	}
}

// contextOf cuts a match and the text around it, on its lines, to what a
// 32px row can show: 12 characters before, 26 of the match, 24 after.
func contextOf(r []rune, m engine.Match) (before, hit, after string) {
	s, e := m.Start, min(m.End, len(r))
	bs := s
	for bs > 0 && s-bs < 12 && r[bs-1] != '\n' {
		bs--
	}
	before = string(r[bs:s])
	if bs > 0 && r[bs-1] != '\n' {
		before = "…" + before
	}
	hr := r[s:e]
	if len(hr) > 26 {
		hr = append(append([]rune{}, hr[:26]...), '…')
	}
	hit = visibleMatch(string(hr))
	ae := e
	for ae < len(r) && ae-e < 24 && r[ae] != '\n' {
		ae++
	}
	after = string(r[e:ae])
	if ae < len(r) && r[ae] != '\n' {
		after += "…"
	}
	return
}

// noMatches is the dashed box shown when nothing matches, with what to try.
func (a *App) noMatches(c *ui.Context, o engine.FindOptions, bad bool) {
	f := &a.ws.find
	if o.Find == "" {
		ui.Column(c.Key("nomatch0")).Grow(1).Center().Children(func() {
			hint(c, "在上面输入要查找的文字，匹配会列在这里。")
		})
		return
	}
	if bad {
		ui.Box(c.Key("nomatchbad")).Grow(1)
		return
	}
	dashed(ui.Column(c.Key("nomatch")).Margin(8, 4).Padding(14).Gap(10).AlignItems(ui.Start).Radius(radiusSeg)).Children(func() {
		ui.Text(c, "没有匹配").FontSize(sizeBody).FontWeight(700).TextColor(Ink)
		ui.Row(c).Gap(6).Wrap().Children(func() {
			if a.find.opts.MatchCase {
				if suggestChip(c, "sg-cs", "关掉区分大小写") {
					a.find.opts.MatchCase = false
					f.reset()
				}
			}
			if a.find.modeIdx == 0 {
				if suggestChip(c, "sg-re", "改用正则") {
					a.find.modeIdx = 1
					f.reset()
				}
			}
			if suggestChip(c, "sg-pat", "从「模式」里挑一个") {
				a.togglePatternMenu()
			}
		})
	})
}

func suggestChip(c *ui.Context, key, label string) bool {
	b := ui.ButtonBase(c.Key(key)).Height(28).Padding(0, 10).Radius(radiusPill).Center().Label(label)
	bg := Paper
	if b.Hovered() {
		bg = AccentSoft
	}
	b.Background(bg).Border(strokeW, Ink)
	focusRing(b)
	b.Children(func() { ui.Text(c, label).FontSize(12).TextColor(Ink).SingleLine() })
	return b.Clicked()
}

// findButtons are 替换这一处 / 全部替换 · N 处 / ⋯.
func (a *App) findButtons(c *ui.Context, k tokens, fi *findInfo, o engine.FindOptions) {
	f := &a.ws.find
	bad := fi.res.err != nil || o.Find == ""
	n := len(fi.res.matches)
	inc := a.included(fi)
	ui.Row(c.Key("findbtns")).FillWidth().Gap(8).Padding(0, 4, 4, 0).AlignItems(ui.Center).Children(func() {
		one := ui.ButtonBase(c.Key("rone")).Height(38).Padding(0, 12).Radius(radiusCtl).Center().Label("替换这一处").Tooltip("⌥⌘↩  替换后跳到下一处").Disabled(bad || n == 0)
		one.Background(Paper)
		boxed(one)
		lift(one, shadowSm)
		one.Children(func() { ui.Text(c, "替换这一处").FontSize(sizeButton).FontWeight(500).TextColor(Ink).SingleLine() })
		if one.Clicked() {
			a.replaceCurrent()
		}
		lbl := fmt.Sprintf("%d 处", inc)
		if lbl != f.inclText {
			f.inclText, f.inclAt = lbl, time.Now()
		}
		p := anim(c, f.inclAt, dFlip, EaseOut)
		all := ui.ButtonBase(c.Key("rall")).Grow(1).Height(38).Radius(radiusCtl).Center().Label("全部替换").Disabled(bad || inc == 0)
		abg := Accent
		if !all.IsDisabled() && all.Hovered() {
			abg = AccentHover
		}
		all.Background(all.AnimateColor("bg", abg, ui.Animation{Duration: 120e6}))
		boxed(all)
		lift(all, shadowSm)
		all.Children(func() {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "全部替换 ·").FontSize(sizeBody).FontWeight(700).TextColor(Ink).SingleLine()
				ui.Box(c).Height(20).Clip().Center().Children(func() {
					ui.Text(c, lbl).FontSize(sizeBody).FontWeight(700).TextColor(Ink).SingleLine().Translate(0, 12*(1-p)).Opacity(p)
				})
			})
		})
		if all.Clicked() {
			a.replaceAllIncluded()
		}
		more := ui.ButtonBase(c.Key("more")).Size(38, 38).Radius(radiusCtl).Center().Label("更多").Tooltip("加入批量规则 · 存为动作链步骤").Disabled(o.Find == "")
		more.Background(Paper)
		boxed(more)
		lift(more, shadowSm)
		more.Children(func() { ui.Text(c, "⋯").FontSize(15).FontWeight(700).TextColor(Ink) })
		a.popMenu(c, "findmore", more, func(m *pm) {
			if m.Item("加入批量规则").Chosen() {
				a.addRuleFromFind(o)
			}
			if m.Item("存为动作链步骤").Chosen() {
				a.addFindToChain(o)
			}
		})
	})
}

// patternMenu floats under the find boxes: built-in patterns and the user's
// own, each with its rule and how often it matches here.
func (a *App) patternMenu(c *ui.Context, k tokens) {
	f := &a.ws.find
	p := &f.pat
	if !p.open {
		return
	}
	var in float32
	if p.closing {
		out := anim(c, p.openAt, dMenuOut, EaseIn)
		if out >= 1 || c.Preferences().ReduceMotion {
			p.open, p.closing, p.hover = false, false, ""
			return
		}
		in = 1 - out
	} else {
		in = anim(c, p.openAt, dMenuIn, EaseSpring)
	}
	scale := 0.95 + 0.05*in
	y := -8 * (1 - in)
	if p.closing {
		scale, y = 0.97+0.03*in, -6*(1-in)
	}
	builtin, mine := a.patternItems()
	menu := ui.Column(c.Key("patmenu")).Absolute().Top(48).Left(0).Right(4).Padding(6).Background(Paper).Radius(radiusSeg).Border(strokeW, Ink)
	hard(menu, shadowLg)
	menu.TransformOrigin(1, 0).Scale(scale, scale).Translate(0, y).Opacity(min(1, in*2))
	p.frame = ""
	if menu.PressedOutside() && !p.closing {
		p.closing, p.openAt = true, time.Now()
	}
	menu.OnShortcut(0, ui.KeyEscape, func() { p.closing, p.openAt = true, time.Now() })
	menu.Children(func() {
		ui.Row(c).FillWidth().Padding(4, 4, 6, 8).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "悬停预览 · 点击使用 · 再点取消").FontSize(11.5).TextColor(Muted).SingleLine()
			ui.Spacer(c)
			x := iconButton(c, "patclose", "x", "关闭", "关闭").Size(26, 26)
			if x.Clicked() {
				p.closing, p.openAt = true, time.Now()
			}
		})
		ui.Scroll(c.Key("patscroll")).FillWidth().MaxHeight(440).Children(func() {
			idx := 0
			section := func(title string, items []patItem) {
				ui.Text(c, strings.ToUpper(title)).FontSize(sizeSmall).FontWeight(700).TextColor(Ink).LetterSpacing(0.88).Padding(10, 8, 4, 8)
				if len(items) == 0 {
					hint(c, "还没有，点下面的按钮存一个").FontSize(11.5).Padding(0, 8, 6, 8)
				}
				for _, it := range items {
					a.patternRow(c, it, idx)
					idx++
				}
			}
			section("内置", builtin)
			section("我的模式", mine)
		})
		ui.Box(c).FillWidth().Height(strokeW).Background(Ink).Margin(6, 2)
		b := ui.ButtonBase(c.Key("patsave")).FillWidth().Height(34).Padding(0, 10).Radius(radiusCtl).AlignItems(ui.Center).Justify(ui.Start).Label("把当前规则存为我的模式")
		bg, bd := ui.Transparent, ui.Transparent
		if b.Hovered() {
			bg, bd = Accent, Ink
		}
		b.Background(bg).Border(strokeW, bd)
		focusRing(b)
		b.Children(func() {
			ui.Text(c, "＋ 把当前规则存为我的模式").FontSize(sizeButton).FontWeight(700).TextColor(Ink)
		})
		if b.Clicked() {
			a.saveCurrentPattern()
		}
	})
	if p.hover != p.frame {
		p.hover = p.frame
		c.Invalidate()
	}
}

func (a *App) patternRow(c *ui.Context, it patItem, idx int) {
	f := &a.ws.find
	p := &f.pat
	active := p.applied == it.name
	cnt := a.patternCount(it)
	in := anim(c, p.openAt.Add(time.Duration(idx)*dMenuStep), 280*time.Millisecond, EaseOut)
	row := ui.ButtonBase(c.Key("pat:"+it.name)).FillWidth().MinHeight(42).Padding(3, 8).Margin(0, 0, 2, 0).Gap(8).AlignItems(ui.Center).Justify(ui.Start).Radius(radiusCtl).Label(it.name)
	bg, bd := ui.Transparent, ui.Transparent
	switch {
	case row.Hovered():
		bg, bd = Accent, Ink
	case active:
		bg = AccentSoft
	}
	row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 100e6})).Border(strokeW, row.AnimateColor("bd", bd, ui.Animation{Duration: 100e6}))
	row.Translate(0, 6*(1-in)).Opacity(in)
	focusRing(row)
	if row.Hovered() {
		p.frame = it.name
	}
	row.Children(func() {
		chk := ""
		if active {
			chk = "✓"
		}
		ui.Text(c, chk).FontSize(sizeBody).FontWeight(700).TextColor(Ink).Width(14)
		ui.Column(c).Grow(1).MinWidth(0).Children(func() {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Text(c, it.name).FontSize(sizeBody).FontWeight(500).TextColor(Ink).SingleLine()
				if it.desc != "" {
					ui.Text(c, cut(it.desc, 14)).FontSize(sizeSmall).TextColor(Muted).SingleLine()
				}
			})
			ui.Text(c, it.pattern).Font(monoFont).FontSize(10.5).TextColor(Muted).SingleLine()
		})
		cbg, ccol := Paper, Ink
		if cnt == 0 {
			ccol = Faint
		} else if !row.Hovered() {
			cbg = AccentSoft
		}
		ui.Text(c, fmt.Sprint(cnt)).Font(monoFont).FontSize(sizeSmall).TextColor(ccol).Background(cbg).Radius(radiusPill).Padding(0, 6).SingleLine()
	})
	if row.Clicked() {
		a.applyPattern(it)
		p.closing, p.openAt = true, time.Now()
	}
}

// patternCount is how many times a pattern matches the text.
func (a *App) patternCount(it patItem) int {
	key := "pc:" + it.pattern
	if v, ok := a.ws.cache[key]; ok {
		return v.(int)
	}
	ms, err := engine.FindOptions{Find: it.pattern, Regex: true}.FindAll(a.Text, 0)
	n := len(ms)
	if err != nil {
		n = 0
	}
	a.ws.cache[key] = n
	return n
}

var _ = utf8.RuneLen

// captureScope limits find/replace to the editor's selection, if it has one
// (one range); the chip in the toggles row drops the limit.
func (a *App) captureScope() {
	f := &a.ws.find
	f.scoped = false
	if sels := a.selections(); len(sels) == 1 && a.ed != nil {
		s := sels[0]
		f.scope = [2]int{utf8.RuneCountInString(a.Text[:s.Start()]), utf8.RuneCountInString(a.Text[:s.End()])}
		f.scoped = true
		f.reset()
	}
}
