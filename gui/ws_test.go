package gui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

// settle runs a few frames, as the pointer and previews take two to catch up.
func settle(tt *ui.Tester, n int) {
	for i := 0; i < n; i++ {
		tt.Frame()
	}
}

func moveTo(t *testing.T, tt *ui.Tester, text string) {
	t.Helper()
	r, ok := tt.Find(text)
	if !ok {
		t.Fatalf("no %q on screen: %q", text, tt.Texts())
	}
	tt.Move(r.X+r.W/2, r.Y+r.H/2)
	settle(tt, 3)
}

func rowID(t *testing.T, a *App, zh string) string {
	t.Helper()
	for _, s := range a.cleanSections() {
		for _, r := range s.rows {
			if r.name == zh {
				return r.id
			}
		}
	}
	t.Fatalf("no row %q", zh)
	return ""
}

func TestHoverPreviewThenApply(t *testing.T) {
	a, tt := newApp(t, "a   b\nc  d\nkeep")
	settle(tt, 2)
	id := rowID(t, a, "删除多余空格")
	moveTo(t, tt, "删除多余空格")
	if a.ws.hov != id {
		t.Fatalf("hover = %q, want %q", a.ws.hov, id)
	}
	pv, _ := a.shownPreview()
	if pv == nil || !tt.HasText("预览") || !tt.HasText("点击应用 · 移开即恢复") {
		t.Fatalf("no preview: %q", tt.Texts())
	}
	if a.Text != "a   b\nc  d\nkeep" {
		t.Fatalf("preview must not touch the text: %q", a.Text)
	}
	// What will go is struck through, what will come inserted.
	merged := a.ws.pvEd.st.Buffer.String()
	if !strings.Contains(merged, "··") {
		t.Errorf("deleted spaces should show as ·: %q", merged)
	}
	// Leaving the list restores the text.
	tt.Move(300, 300)
	settle(tt, 3)
	if a.ws.hov != "" {
		t.Fatalf("hover stayed: %q", a.ws.hov)
	}
	if pv, _ := a.shownPreview(); pv != nil {
		t.Fatal("preview stayed after leaving the list")
	}
	// Click applies, and the trail gains a step.
	if err := tt.Click("删除多余空格"); err != nil {
		t.Fatal(err)
	}
	settle(tt, 2)
	if a.Text != "a b\nc d\nkeep" {
		t.Fatalf("applied: %q", a.Text)
	}
	if len(a.trail().steps) != 2 || a.trail().steps[1].label != "删除多余空格" {
		t.Fatalf("trail: %+v", a.trail().steps)
	}
	if a.ws.applyFx == nil {
		t.Fatal("apply animation did not start")
	}
	if !tt.HasText("原文") {
		t.Fatalf("crumbs: %q", tt.Texts())
	}
}

func TestBadgeCountMatchesAppliedResult(t *testing.T) {
	a, tt := newApp(t, "a   b\n\n\nc  d\n\n  keep  \n")
	settle(tt, 2)
	for _, s := range a.cleanSections() {
		for _, r := range s.rows {
			res := a.actionResult(r.id)
			_, fn := a.resolveAction(r.id)
			after := fn(a.Text)
			if res.after != after {
				t.Errorf("%s: preview result differs from the applied one", r.id)
			}
			want := ""
			if after != a.Text {
				want = deltaText(a.Text, after, engine.Diff(a.Text, after))
			}
			if res.delta != want {
				t.Errorf("%s: badge %q, applied change %q", r.id, res.delta, want)
			}
		}
	}
	// Cached while the text does not change; recomputed when it does.
	r1 := a.actionResult("Remove Extra Spaces")
	tt.Frame()
	if r2 := a.actionResult("Remove Extra Spaces"); r1 != r2 {
		t.Error("badge recomputed for unchanged text")
	}
	a.Text = "x  y"
	tt.Frame()
	if r3 := a.actionResult("Remove Extra Spaces"); r3 == r1 || r3.delta != "1 处" {
		t.Errorf("badge after edit: %+v", r3)
	}
}

func TestTrailUndoRedoAndJump(t *testing.T) {
	a, tt := newApp(t, "a   b\n\nc")
	settle(tt, 2)
	a.applyAction("Remove Extra Spaces")
	a.ws.applyFx = nil
	settle(tt, 1)
	a.applyAction("Uppercase")
	settle(tt, 1)
	steps := a.trail().steps
	if len(steps) != 3 || a.Text != "A B\n\nC" {
		t.Fatalf("steps %+v text %q", steps, a.Text)
	}
	a.Undo()
	if a.Text != "a b\n\nc" || !a.CanRedo() {
		t.Fatalf("undo: %q", a.Text)
	}
	a.goStep(0) // jump to 原文 without losing the later steps
	if a.Text != "a   b\n\nc" || len(a.trail().steps) != 3 {
		t.Fatalf("jump: %q %d", a.Text, len(a.trail().steps))
	}
	a.goStep(2)
	if a.Text != "A B\n\nC" {
		t.Fatalf("forward: %q", a.Text)
	}
	// Two cleaner steps can be saved as an action chain.
	if !a.canSaveChain() {
		t.Fatal("save-as-chain not offered")
	}
	n := len(a.Lib.Chains)
	a.saveTrailAsChain()
	if len(a.Lib.Chains) != n+1 || a.ws.toast.text == "" {
		t.Fatalf("chain not saved: %d, toast %q", len(a.Lib.Chains), a.ws.toast.text)
	}
}

func TestSearchWithNoResultOffersFind(t *testing.T) {
	a, tt := newApp(t, "zzzz here")
	settle(tt, 2)
	a.ws.query = "zzzz"
	settle(tt, 2)
	if !tt.HasText("没有叫「zzzz」的动作") {
		t.Fatalf("empty state: %q", tt.Texts())
	}
	if err := tt.Click("用「zzzz」新建查找替换 →"); err != nil {
		t.Fatal(err)
	}
	settle(tt, 2)
	if a.ws.tab != 1 || a.find.opts.Find != "zzzz" || a.find.modeIdx != 0 {
		t.Fatalf("tab %d find %q mode %d", a.ws.tab, a.find.opts.Find, a.find.modeIdx)
	}
}

func TestSkippedMatchesAreNotReplaced(t *testing.T) {
	a, tt := newApp(t, "x1 x2 x3")
	a.setTab(1)
	a.find.opts.Find, a.find.opts.Replace = "x", "y"
	settle(tt, 3)
	if !tt.HasText("3 处") {
		t.Fatalf("count: %q", tt.Texts())
	}
	a.ws.find.skip = map[int]bool{1: true}
	settle(tt, 2)
	if !tt.HasText("2 处") {
		t.Fatalf("included count: %q", tt.Texts())
	}
	if err := tt.Click("全部替换 ·"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "y1 x2 y3" {
		t.Fatalf("replace all: %q", a.Text)
	}
}

func TestReplaceCurrentMovesToNext(t *testing.T) {
	a, tt := newApp(t, "x1 x2 x3")
	a.setTab(1)
	a.find.opts.Find, a.find.opts.Replace = "x", "y"
	settle(tt, 3)
	a.jumpTo(1)
	settle(tt, 2)
	a.replaceCurrent()
	settle(tt, 3)
	if a.Text != "x1 y2 x3" {
		t.Fatalf("replace one: %q", a.Text)
	}
	if a.ws.find.cur != 1 { // the third x is now the second match
		t.Fatalf("cur = %d, want 1", a.ws.find.cur)
	}
}

func TestJumpKeepsTextListGutterAndTicksTogether(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 40; i++ {
		fmt.Fprintf(&b, "line %d %s\n", i, map[bool]string{true: "needle", false: "hay"}[i%7 == 0])
	}
	a, tt := newApp(t, b.String())
	a.setTab(1)
	a.find.opts.Find, a.find.opts.Replace = "needle", "N"
	a.find.opts.MatchCase = true
	settle(tt, 3)
	a.jumpTo(2)
	settle(tt, 3)
	f := &a.ws.find
	pv, _ := a.shownPreview()
	if f.cur != 2 || pv == nil {
		t.Fatalf("cur %d pv %v", f.cur, pv)
	}
	line := pv.curLine
	if line != 14 {
		t.Fatalf("current line %d, want 14", line)
	}
	st := a.ws.pvEd.st
	gotRow := false
	for _, gl := range st.Gutter.Lines {
		if gl.Line == line && gl.NumberFill.A > 0 {
			gotRow = true
		}
	}
	if !gotRow {
		t.Error("gutter row of the current match not highlighted")
	}
	outlined := 0
	for _, h := range st.Highlights {
		if h.Outline.A > 0 {
			outlined++
		}
	}
	if outlined != 2 { // the struck match and its replacement
		t.Errorf("outlined spans = %d", outlined)
	}
	if !tt.HasText("第 3 / 6 处 · 第 15 行") {
		t.Errorf("status: %q", tt.Texts())
	}
	// ⌘G / ⇧⌘G step through and wrap.
	tt.Key(ui.Cmd, ui.KeyG)
	settle(tt, 2)
	if f.cur != 3 {
		t.Errorf("⌘G: %d", f.cur)
	}
	tt.Key(ui.Cmd|ui.Shift, ui.KeyG)
	tt.Key(ui.Cmd|ui.Shift, ui.KeyG)
	settle(tt, 2)
	if f.cur != 1 {
		t.Errorf("⇧⌘G: %d", f.cur)
	}
}

func TestPatternMenuApplyAndRestore(t *testing.T) {
	a, tt := newApp(t, "价格 5元 和 12元")
	a.setTab(1)
	a.find.opts.Find, a.find.opts.Replace = "5", "六"
	settle(tt, 2)
	p, ok := a.patternNamed("价格")
	if !ok {
		t.Fatal("no 价格 pattern")
	}
	a.applyPattern(p)
	settle(tt, 2)
	if a.find.opts.Find != `(\d+)元` || a.find.opts.Replace != "¥$1" || a.find.modeIdx != 1 || a.ws.find.pat.applied != "价格" {
		t.Fatalf("applied: %+v %+v", a.find.opts, a.ws.find.pat)
	}
	// A second pick takes it off and restores what was there.
	a.applyPattern(p)
	if a.find.opts.Find != "5" || a.find.opts.Replace != "六" || a.find.modeIdx != 0 || a.ws.find.pat.applied != "" {
		t.Fatalf("restored: %+v", a.find.opts)
	}
	// So does the × on the tag.
	a.applyPattern(p)
	settle(tt, 2)
	if err := tt.Click("取消这个模式，恢复原来的规则"); err != nil {
		t.Fatal(err)
	}
	if a.find.opts.Find != "5" || a.ws.find.pat.applied != "" {
		t.Fatalf("× restore: %+v", a.find.opts)
	}
	// Editing the find box by hand drops the tag.
	a.applyPattern(p)
	settle(tt, 2)
	a.find.opts.Find = "abc"
	settle(tt, 2)
	if a.ws.find.pat.applied != "" {
		t.Fatal("tag stayed after a hand edit")
	}
}

func TestPatternMenuHoverPreviewsMatches(t *testing.T) {
	a, tt := newApp(t, "tel 13812345678 and 13987654321")
	a.setTab(1)
	a.togglePatternMenu()
	settle(tt, 3)
	if !tt.HasText("悬停预览 · 点击使用 · 再点取消") {
		t.Fatalf("menu not open: %q", tt.Texts())
	}
	moveTo(t, tt, "中国大陆手机号")
	if a.ws.find.pat.hover != "中国大陆手机号" {
		t.Fatalf("hover = %q", a.ws.find.pat.hover)
	}
	pv, _ := a.shownPreview()
	if pv == nil || !pv.hitMode || !tt.HasText("模式「中国大陆手机号」") || !tt.HasText("2 处匹配") {
		t.Fatalf("pattern preview: %v %q", pv, tt.Texts())
	}
}

func TestRegexErrorDisablesReplace(t *testing.T) {
	a, tt := newApp(t, "abc")
	a.setTab(1)
	a.find.modeIdx = 1
	a.find.opts.Find = "(ab"
	settle(tt, 3)
	if !strings.Contains(strings.Join(tt.Texts(), " "), "正则写法有误") {
		t.Fatalf("no error box: %q", tt.Texts())
	}
	a.find.opts.Replace = "x"
	a.replaceAllIncluded()
	if a.Text != "abc" {
		t.Fatalf("replaced despite the error: %q", a.Text)
	}
	if shake(0.2) != -5 || shake(0.4) != 5 || shake(0.6) != -3 || shake(0.8) != 2 || shake(1) != 0 {
		t.Error("shake keyframes")
	}
}

func TestGutterClickPicksWholeLines(t *testing.T) {
	a, tt := newApp(t, "one\ntwo\nthree")
	settle(tt, 2)
	a.gutterClick(1)
	if !a.ws.selLines[1] || a.Scope() != "选区" || a.ed.Buffer.Slice(a.selections()[0].Start(), a.selections()[0].End()) != "two\n" {
		t.Fatalf("pick: %v %q", a.ws.selLines, a.scopeText())
	}
	a.gutterClick(1)
	if len(a.ws.selLines) != 0 || a.Scope() != "全文" {
		t.Fatalf("unpick: %v", a.ws.selLines)
	}
	a.gutterClick(0)
	a.Text = "changed"
	settle(tt, 2)
	if len(a.ws.selLines) != 0 {
		t.Fatal("picked lines survived a text change")
	}
}

func TestReduceMotionFinishesAnimationsAtOnce(t *testing.T) {
	a, tt := newApp(t, "a   b")
	tt.SetPreferences(ui.Preferences{ReduceMotion: true})
	settle(tt, 2)
	a.applyAction("Remove Extra Spaces")
	settle(tt, 2)
	if a.ws.applyFx != nil {
		t.Fatal("apply animation should be skipped")
	}
}

func TestLargeTextPreviewIsCached(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 3000; i++ {
		fmt.Fprintf(&b, "row %d  with   spaces   and 价格 %d元\n", i, i)
	}
	a, tt := newApp(t, b.String())
	settle(tt, 2)
	start := time.Now()
	r := a.actionResult("Remove Extra Spaces")
	pv := r.preview("Remove Extra Spaces")
	first := time.Since(start)
	if pv == nil || !r.changed() {
		t.Fatal("no preview")
	}
	start = time.Now()
	for i := 0; i < 100; i++ {
		_ = a.actionResult("Remove Extra Spaces").preview("Remove Extra Spaces")
	}
	if again := time.Since(start); again > first/2+5*time.Millisecond {
		t.Errorf("repeat lookups are not cached: first %v, 100 more %v", first, again)
	}
	// Many matches: the list renders at most 300 rows.
	a.setTab(1)
	a.find.modeIdx = 1
	a.find.opts.Find, a.find.opts.Replace = `\d+元`, "¥"
	start = time.Now()
	settle(tt, 3)
	if d := time.Since(start); d > 5*time.Second {
		t.Errorf("frames with 3000 matches took %v", d)
	}
	if !tt.HasText("3000 / 3000") && !tt.HasText("1 / 3000") {
		t.Errorf("counter: %q", tt.Texts()[:20])
	}
}

func TestSavedPatternPersists(t *testing.T) {
	a, tt := newApp(t, "x")
	a.setTab(1)
	a.find.modeIdx = 1
	a.find.opts.Find, a.find.opts.Replace = `(\d+)号`, "No.$1"
	settle(tt, 2)
	a.saveCurrentPattern()
	_, mine := a.patternItems()
	if len(mine) != 1 || mine[0].pattern != `(\d+)号` {
		t.Fatalf("mine: %+v", mine)
	}
}
