package gui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/dandibbert/textrouter/engine"
	"github.com/dandibbert/textrouter/internal/fontlist"
	"github.com/egoist/mygo/ui"
)

func spansText(after string, sp []changeSpan) []string {
	var out []string
	for _, s := range sp {
		if s.Del {
			out = append(out, "^"+after[s.Start:])
			continue
		}
		out = append(out, after[s.Start:s.End])
	}
	return out
}

func TestChangeSpansCharacterLevel(t *testing.T) {
	cases := []struct {
		before, after string
		want          []string
	}{
		// One changed word in a long line: only that word.
		{"My favourite colour is grey", "My favorite color is grey", nil}, // two deletion points
		{"My favorite color is grey", "My favourite colour is grey", []string{"u", "u"}},
		{"Hello  world", "Hello world", nil}, // deletion only: a point
		{"price 5元 and 12元", "price ¥5 and ¥12", []string{"¥5", "¥12"}},
		{"abc\ndef\nghi", "abc\nDEF\nghi", []string{"DEF"}},
		{"中文English", "中文 English", []string{" "}},
		{"same", "same", nil},
	}
	for _, c := range cases {
		sp := changeSpans(c.before, c.after)
		var got []string
		for _, s := range sp {
			if !s.Del {
				got = append(got, c.after[s.Start:s.End])
			}
		}
		if strings.Join(got, "|") != strings.Join(c.want, "|") {
			t.Errorf("%q → %q: got %q want %q", c.before, c.after, got, c.want)
		}
		// Spans never cover a whole unchanged line.
		for _, s := range sp {
			if s.End-s.Start == len(c.after) && c.before != "" {
				t.Errorf("%q: span covers everything", c.after)
			}
		}
	}
	// A deletion leaves a point where the text was.
	sp := changeSpans("Hello  world", "Hello world")
	if len(sp) != 1 || !sp[0].Del || sp[0].Start != 6 {
		t.Fatalf("deletion point: %+v", sp)
	}
	// Reworded word: short equal runs inside are folded into one span.
	sp = changeSpans("organise it", "systemize it")
	if len(sp) != 1 || spansText("systemize it", sp)[0] != "systemize" && !strings.Contains("systemize", spansText("systemize it", sp)[0]) {
		t.Fatalf("fold: %q", spansText("systemize it", sp))
	}
}

func TestChangeHighlightShownThenClearedByEdit(t *testing.T) {
	a, tt := newApp(t, "a   b\nkeep this line")
	tt.Frame()
	cl := engine.CleanerByName("Remove Extra Spaces")
	a.Transform(cl.ZhName, cl.Fn)
	a.ws.applyFx = nil // the 500ms collapse shows a preview first
	tt.Frame()
	if len(a.ed.Highlights) != 1 || a.ed.HighlightVersion != a.ed.Buffer.Version() {
		t.Fatalf("highlights after action: %+v", a.ed.Highlights)
	}
	h := a.ed.Highlights[0]
	if h.Start != 2 || h.End != 2 { // only the removed spaces, as a point
		t.Fatalf("span %+v", h)
	}
	// Typing clears it.
	if err := tt.Click("剪贴板文本"); err != nil {
		t.Fatal(err)
	}
	tt.Type("x")
	tt.Frame()
	tt.Frame()
	if len(a.changes) != 0 || len(a.ed.Highlights) != 0 {
		t.Fatalf("not cleared after edit: %+v", a.ed.Highlights)
	}
}

func TestFindMatchesHighlightedInEditor(t *testing.T) {
	a, tt := newApp(t, "价格 5元 和 12元")
	a.showFind()
	a.find.modeIdx = 1
	a.find.opts.Find = `\d+元`
	tt.Frame()
	tt.Frame()
	// With matches the body shows the preview: matches struck through.
	st := a.ws.pvEd.st
	if st == nil {
		t.Fatal("no preview editor")
	}
	merged := st.Buffer.String()
	var got []string
	for _, h := range st.Highlights {
		if h.Strike.A > 0 {
			got = append(got, merged[h.Start:h.End])
		}
	}
	if strings.Join(got, ",") != "5元,12元" {
		t.Fatalf("find highlights %q", got)
	}
}

func TestInvisiblesTakeNoRoom(t *testing.T) {
	text := "a\u00a0b\u3000c\u200bd\re\nnext"
	a, tt := newApp(t, text)
	tt.Frame()
	xs := func() []float32 {
		var out []float32
		for i := 0; i <= strings.Index(text, "\n"); i++ {
			out = append(out, a.ed.RuneX(0, i))
		}
		return out
	}
	off := xs()
	a.view.invisibles = true
	tt.Frame()
	tt.Frame()
	if !a.ed.Invisibles {
		t.Fatal("invisibles not on")
	}
	on := xs()
	for i := range off {
		if off[i] != on[i] {
			t.Fatalf("byte %d moved: %v vs %v", i, off[i], on[i])
		}
	}
	marks := a.ed.InvisibleMarks(0)
	if len(marks) != 4 {
		t.Fatalf("marks %+v", marks)
	}
	// Each mark lies on its own character's place.
	for _, m := range marks {
		found := false
		for i := range on {
			if abs32(on[i]-m.X) < 0.01 {
				found = true
			}
		}
		if !found {
			t.Fatalf("mark %+v not at a character", m)
		}
	}
	// The zero-width space draws as a mark of no width.
	if marks[2].W > 0.5 {
		t.Fatalf("zero-width mark has width %v", marks[2].W)
	}
	// No inlays are used any more.
	a.view.invisibles = false
	tt.Frame()
}

func abs32(x float32) float32 {
	if x < 0 {
		return -x
	}
	return x
}

func TestViewSettingsRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "library.json")
	lib := engine.LoadLibrary(path)
	if !lib.Settings.View.LineNumbers || !lib.Settings.View.Wrap || lib.Settings.View.FontSize != 15 {
		t.Fatalf("defaults %+v", lib.Settings.View)
	}
	a := New(lib, nil)
	tt := ui.NewTester(a.View, 1100, 700)
	tt.Frame()
	a.view.lineNumbers, a.view.wrap, a.view.invisibles = false, false, true
	a.zoomBy(2)
	a.toggleInspector("find")
	a.find.modeIdx = 1
	a.Lib.Settings.View.EditorFont = "Georgia"
	a.Lib.Settings.View.CJKFont = "Songti SC"
	a.Lib.Settings.View.FontSize = 17
	a.Lib.Settings.View.LineHeight = 1.8
	a.SetWindowSize(1300, 840)
	tt.Frame()

	lib2 := engine.LoadLibrary(path)
	v := lib2.Settings.View
	if v.LineNumbers || v.Wrap || !v.Invisibles || v.Zoom != 1.2 || v.Inspector != "find" || !v.FindRegex ||
		v.EditorFont != "Georgia" || v.CJKFont != "Songti SC" || v.FontSize != 17 || v.LineHeight != 1.8 || v.WindowW != 1300 || v.WindowH != 840 {
		t.Fatalf("round trip %+v", v)
	}
	b := New(lib2, nil)
	tb := ui.NewTester(b.View, 1100, 700)
	tb.Frame()
	if b.view.lineNumbers || !b.view.invisibles || b.inspector != "find" || b.find.modeIdx != 1 || abs32(b.ed.Font.Size-17*1.2) > 0.01 || b.ed.LineHeight != 1.8 ||
		!strings.HasPrefix(b.ed.Font.Family, "Georgia, Songti SC") {
		t.Fatalf("restored: %+v font %+v lh %v", b.view, b.ed.Font, b.ed.LineHeight)
	}
}

func TestFontSettingsPicker(t *testing.T) {
	fontlist.SetForTest([]fontlist.Family{{Name: "Georgia"}, {Name: "Menlo", Mono: true}, {Name: "Songti SC", LocalName: "宋体-简", CJK: true}})
	a, tt := newApp(t, "x")
	a.OpenTool(PageSettings)
	tt.Frame()
	if !tt.HasText("实时预览") || !tt.HasText("中文后备字体") {
		t.Fatalf("font settings missing: %q", tt.Texts())
	}
	if err := tt.Click("苹方（系统默认）"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if tt.HasText("Georgia  中文字体示例") || !tt.HasText("Songti SC  中文字体示例") {
		t.Fatalf("CJK list should hold only CJK fonts: %q", tt.Texts())
	}
	if err := tt.Click("Songti SC  中文字体示例"); err != nil {
		t.Fatal(err)
	}
	if a.Lib.Settings.View.CJKFont != "Songti SC" {
		t.Fatalf("cjk = %q", a.Lib.Settings.View.CJKFont)
	}
}
