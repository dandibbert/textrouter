package gui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
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
	if !tt.HasText("剪贴板空间") {
		t.Fatalf("texts %q", tt.Texts())
	}
	if err := tt.Click("删除多余空格"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "a b\n> quoted" {
		t.Fatalf("after cleaner: %q", a.Text)
	}
	if err := tt.Click("撤销"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "a   b\n> quoted" {
		t.Fatalf("after undo: %q", a.Text)
	}
	if err := tt.Click("复制并完成"); err != nil {
		t.Fatal(err)
	}
	if got := tt.Clipboard(); got != a.Text {
		t.Errorf("clipboard %q", got)
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
	if !tt.HasText("找到 2 处") {
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
	if err := tt.Click("应用到剪贴板空间"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "my favorite color, color" {
		t.Errorf("batch: %q", a.Text)
	}
	if err := tt.Click("＋ 添加规则"); err != nil {
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
	if err := tt.Click("应用到剪贴板空间"); err != nil {
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
	if !tt.HasText("打开剪贴板空间的全局快捷键") {
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
	if !strings.Contains(strings.Join(tt.Texts(), " "), "剪贴板空间") {
		t.Error("dark render")
	}
}
