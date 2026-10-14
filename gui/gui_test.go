package gui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

func newApp(t *testing.T, text string) (*App, *ui.Tester) {
	t.Helper()
	lib := engine.LoadLibrary(filepath.Join(t.TempDir(), "library.json"))
	a := New(lib, nil)
	a.Text = text
	return a, ui.NewTester(a.View, 1180, 760)
}

func TestClipboardSpaceApplyUndoCopy(t *testing.T) {
	a, tt := newApp(t, "a   b\n> quoted")
	if !tt.HasText("作用于全文") {
		t.Fatalf("texts %q", tt.Texts())
	}
	cl := engine.CleanerByName("Remove Extra Spaces")
	a.Transform(cl.ZhName, cl.Fn)
	tt.Frame()
	if a.Text != "a b\n> quoted" {
		t.Fatalf("after cleaner: %q", a.Text)
	}
	if err := tt.Click("撤销"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "a   b\n> quoted" {
		t.Fatalf("after undo: %q", a.Text)
	}
	// Typing in the editor and the action share one undo history.
	if err := tt.Click("剪贴板文本"); err != nil {
		t.Fatal(err)
	}
	tt.Key(ui.Ctrl, ui.KeyA)
	tt.Key(ui.Cmd, ui.KeyA)
	tt.Type("new")
	tt.Frame()
	if a.Text != "new" {
		t.Fatalf("typed: %q", a.Text)
	}
	a.Undo()
	if a.Text != "a   b\n> quoted" {
		t.Fatalf("undo typing: %q", a.Text)
	}
	if err := tt.Click("复制并完成"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != a.Text {
		t.Errorf("clipboard %q", got)
	}
}

func TestSelectionScopedActions(t *testing.T) {
	a, tt := newApp(t, "hello world\nsecond line")
	tt.Frame()
	a.ed.Select(6, 11) // "world"
	tt.Frame()
	if !tt.HasText("作用于选区 · 5 字") {
		t.Fatalf("scope pill: %q", tt.Texts())
	}
	up := engine.CleanerByName("Uppercase")
	a.Transform(up.ZhName, up.Fn)
	if a.Text != "hello WORLD\nsecond line" {
		t.Fatalf("selection action: %q", a.Text)
	}
	if a.last.scope != "选区" {
		t.Errorf("scope %q", a.last.scope)
	}
	// Find/replace page applies to the selection too.
	a.ed.Select(0, 5)
	a.Page = PageFind
	tt.Frame()
	a.insp.inSel, a.insp.selRange = true, [2]int{0, 5}
	a.find.opts.Find, a.find.opts.Replace = "l", "L"
	tt.Frame()
	if err := tt.Click("全部替换（2 处）"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "heLLo WORLD\nsecond line" {
		t.Fatalf("scoped replace: %q", a.Text)
	}
	// Two selections: both change in one undo step.
	a.insp.inSel = false
	tt.Frame()
	a.ed.SetSelections(editor.Selection{Anchor: 0, Head: 2}, editor.Selection{Anchor: 12, Head: 18})
	a.Transform(up.ZhName, up.Fn)
	if a.Text != "HELLo WORLD\nSECOND line" {
		t.Fatalf("multi selection: %q", a.Text)
	}
	a.Undo()
	if a.Text != "heLLo WORLD\nsecond line" {
		t.Fatalf("multi undo: %q", a.Text)
	}
	// No selection: whole text.
	a.ed.Select(0, 0)
	lo := engine.CleanerByName("lowercase")
	if lo == nil {
		lo = &engine.Cleaner{ZhName: "lower", Fn: strings.ToLower}
	}
	a.Transform("lower", strings.ToLower)
	if a.Text != "hello world\nsecond line" || a.last.scope != "全文" {
		t.Fatalf("whole: %q %q", a.Text, a.last.scope)
	}
}

func TestLineOperations(t *testing.T) {
	a, tt := newApp(t, "b\nc\na")
	tt.Frame()
	a.ed.Select(0, len(a.Text))
	a.RunEditorCommand(editor.SortLines)
	if a.Text != "a\nb\nc" {
		t.Fatalf("sort: %q", a.Text)
	}
	a.ed.Select(0, 0)
	a.RunEditorCommand(editor.DuplicateLinesDown)
	if a.Text != "a\na\nb\nc" {
		t.Fatalf("duplicate: %q", a.Text)
	}
	a.RunEditorCommand(editor.DeleteLine)
	if a.Text != "a\nb\nc" {
		t.Fatalf("delete line: %q", a.Text)
	}
	a.ed.Select(0, 0)
	a.RunEditorCommand(editor.MoveLinesDown)
	if a.Text != "b\na\nc" {
		t.Fatalf("move down: %q", a.Text)
	}
	a.ed.Select(0, len(a.Text))
	a.RunEditorCommand(editor.JoinLines)
	if a.Text != "b a c" {
		t.Fatalf("join: %q", a.Text)
	}
	// Palette runs editor commands too.
	a.paletteQuery = "转为大写"
	items := a.paletteItems()
	if len(items) == 0 {
		t.Fatal("no palette item")
	}
	a.ed.Select(0, len(a.Text))
	a.runPaletteItem(items[0])
	if a.Text != "B A C" {
		t.Fatalf("palette upper: %q (%s)", a.Text, items[0].id)
	}
}

func TestInvisibleInlays(t *testing.T) {
	in := invisibleInlays("a\u00a0b\u3000c\n")
	if len(in) != 3 || in[0].Text != "⍽" || in[1].Text != "□" || in[2].Text != "¬" {
		t.Fatalf("%+v", in)
	}
}

func TestPaletteSearchAndRun(t *testing.T) {
	a, tt := newApp(t, "hello world")
	tt.Key(ui.Cmd, ui.KeyK)
	tt.Frame()
	if !tt.HasText("全部动作") || !a.paletteOpen {
		t.Fatal("palette did not open")
	}
	tt.Frame()
	tt.Type("全部大写")
	tt.Frame()
	t.Logf("query=%q focused=%v", a.paletteQuery, tt.Focused("搜索动作"))
	tt.Key(0, ui.KeyEnter)
	tt.Frame()
	if a.Text != "HELLO WORLD" {
		t.Errorf("palette run: %q", a.Text)
	}
}

func TestFindPageLivePreview(t *testing.T) {
	a, tt := newApp(t, "price 5元, 12元")
	a.Page = PageFind
	tt.Frame()
	a.find.modeIdx = 1
	a.find.opts.Find = `(\d+)元`
	a.find.opts.Replace = "¥$1"
	tt.Frame()
	if !tt.HasText("2 处") || a.inspector != "find" {
		t.Fatalf("texts %q", tt.Texts())
	}
	if !tt.HasText("第 1 组开始（替换里用 $1 引用）") {
		t.Errorf("no explanation: %q", tt.Texts())
	}
	if err := tt.Click("全部替换（2 处）"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "price ¥5, ¥12" {
		t.Errorf("replace all: %q", a.Text)
	}
}

func TestBatchPageCounts(t *testing.T) {
	a, tt := newApp(t, "my favourite colour, colour")
	a.Page = PageBatch
	tt.Frame()
	if !tt.HasText("2 处") || !tt.HasText("1 处") {
		t.Fatalf("counts missing: %q", tt.Texts())
	}
	if err := tt.Click("应用到文档"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "my favorite color, color" {
		t.Errorf("batch: %q", a.Text)
	}
	if err := tt.Click("添加规则"); err != nil {
		t.Fatal(err)
	}
	if n := len(a.currentRuleSet().Rules); n != 5 {
		t.Errorf("rules %d", n)
	}
}

func TestChainsPage(t *testing.T) {
	a, tt := newApp(t, "中文English混排")
	a.Page = PageChains
	a.chains.selID = "demo-pangu"
	tt.Frame()
	if !tt.HasText("最终结果") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if err := tt.Click("应用到文档"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "中文 English 混排" {
		t.Errorf("chain: %q", a.Text)
	}
}

func TestCleanersAndSettingsAndFiles(t *testing.T) {
	a, tt := newApp(t, "“smart”")
	a.Page = PageCleaners
	a.clean.selected = "Straighten Quotes"
	tt.Frame()
	if !tt.HasText("已实测：与 TextSoap 9 在 39 个测试输入上的输出一致。") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if err := tt.Click("应用"); err != nil {
		t.Fatal(err)
	}
	if a.Text != `"smart"` {
		t.Errorf("cleaner: %q", a.Text)
	}
	a.Page = PageSettings
	tt.Frame()
	if !tt.HasText("打开剪贴板文档的全局快捷键") {
		t.Errorf("settings: %q", tt.Texts())
	}
	a.Page = PageFiles
	tt.Frame()
	if !tt.HasText("批量处理文件") {
		t.Errorf("files: %q", tt.Texts())
	}
	// Dark mode renders too.
	tt.SetDark(true)
	a.Page = PageClipboard
	tt.Frame()
	if !strings.Contains(strings.Join(tt.Texts(), " "), "复制并完成") {
		t.Error("dark render")
	}
}

func TestDocumentTabsAndToolSheet(t *testing.T) {
	a, tt := newApp(t, "from clipboard")
	tt.Frame()
	p := filepath.Join(t.TempDir(), "notes.txt")
	if err := os.WriteFile(p, []byte("file body"), 0o644); err != nil {
		t.Fatal(err)
	}
	a.loadFile(p)
	tt.Frame()
	if len(a.docs) != 2 || a.cur != 1 || a.Text != "file body" || !tt.HasText("notes.txt") {
		t.Fatalf("docs=%d cur=%d text=%q", len(a.docs), a.cur, a.Text)
	}
	if tt.HasText("复制并完成") {
		t.Fatal("file document should not offer copy-and-done")
	}
	if err := tt.Click("剪贴板"); err != nil {
		t.Fatal(err)
	}
	if a.cur != 0 || a.Text != "from clipboard" {
		t.Fatalf("switch back: cur=%d %q", a.cur, a.Text)
	}
	a.loadFile(p) // reopening switches to the existing tab
	tt.Frame()
	if len(a.docs) != 2 || a.cur != 1 {
		t.Fatalf("reopen: docs=%d cur=%d", len(a.docs), a.cur)
	}
	a.OpenTool(PageBatch)
	tt.Frame()
	if !tt.HasText("应用到文档") {
		t.Fatalf("batch sheet missing: %q", tt.Texts())
	}
	tt.Key(0, ui.KeyEscape)
	tt.Frame()
	if a.Page != PageClipboard {
		t.Fatalf("esc should close sheet, page=%q", a.Page)
	}
	if strings.Contains(strings.Join(tt.Texts(), "|"), "剪贴板空间") {
		t.Fatal("剪贴板空间 still in UI")
	}
}
