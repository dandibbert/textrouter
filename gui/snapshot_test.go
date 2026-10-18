package gui

import (
	"github.com/dandibbert/textrouter/engine"
	"github.com/dandibbert/textrouter/internal/fontlist"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

const snapSample = "Hello  world, this is   a “sample” text.\n> quoted line from email\n> price 5元 and 12元\n中文English混排\u3000test@example.com\n\tindented\u00a0with NBSP and a tab   \n\nnew paragraph https://example.com\nSort me: banana\nSort me: apple\nSort me: cherry"

// TestSnapshots writes PNGs of every screen when TEXTROUTER_SNAPSHOTS is a
// directory, for the README and design review.
func TestSnapshots(t *testing.T) {
	dir := os.Getenv("TEXTROUTER_SNAPSHOTS")
	if dir == "" {
		t.Skip("set TEXTROUTER_SNAPSHOTS=<dir>")
	}
	type scene struct {
		name  string
		setup func(a *App, tt *ui.Tester)
	}
	scenes := []scene{
		{"main-find-regex", func(a *App, tt *ui.Tester) {
			a.showFind()
			a.find.modeIdx = 1
			a.find.opts.Find = `(\d+)元`
			a.find.opts.Replace = "¥$1"
			a.ws.find.explain = false
		}},
		{"main-cleaners", func(a *App, tt *ui.Tester) {
			a.toggleInspector("cleaners")
			s := strings.Index(a.Text, "中文")
			a.ed.Select(s, strings.Index(a.Text, "\n\nnew"))
		}},
		{"tools-menu", func(a *App, tt *ui.Tester) {
			tt.Frame()
			if err := tt.Click("工具"); err != nil {
				t.Fatal(err)
			}
			// The real menu is native (NSMenu); paint its items in-app.
			a.fakeMenu = tt.Menu()
			if len(a.fakeMenu) == 0 {
				t.Fatal("tools menu did not open")
			}
		}},
		{"batch-sheet", func(a *App, tt *ui.Tester) {
			a.Text += "\nMy favourite colour is centre grey — organise it."
			a.OpenTool(PageBatch)
		}},
		{"doc-clipboard", func(a *App, tt *ui.Tester) {}},
		{"doc-file", func(a *App, tt *ui.Tester) {
			p := filepath.Join(t.TempDir(), "release-notes.md")
			os.WriteFile(p, []byte("# Release notes\n\n- Faster  find  panel\n- Document tabs: clipboard and files side by side\n- Batch replace as a sheet\n"), 0o644)
			a.loadFile(p)
			a.view.lineNumbers = true
		}},
		{"change-highlight", func(a *App, tt *ui.Tester) {
			a.freezeChanges = true
			rs := a.Lib.RuleSets[0]
			a.Text += "\nMy favourite colour is centre grey — organise it."
			tt.Frame()
			cl := engine.CleanerByName("Straighten Quotes")
			a.Transform("批量替换 + 直引号", func(s string) string {
				out, _, _ := engine.ApplyRules(s, rs.Rules)
				return cl.Fn(strings.ReplaceAll(out, "  ", " "))
			})
		}},
		{"invisibles", func(a *App, tt *ui.Tester) {
			a.Text = "全角空格\u3000在这里，NBSP\u00a0在这里。\n零宽空格\u200b藏在这里，回车\r在这里。\n\t制表符缩进   行尾空格   \n最后一行"
			a.view.invisibles = true
		}},
		{"settings-fonts", func(a *App, tt *ui.Tester) {
			fontlist.System()
			fontlist.Load(nil)
			for i := 0; i < 50; i++ {
				if _, ok := fontlist.Ready(); ok {
					break
				}
				time.Sleep(20 * time.Millisecond)
			}
			a.Lib.Settings.View.EditorFont = "Source Serif Pro"
			a.Lib.Settings.View.CJKFont = "Noto Serif CJK SC"
			a.Lib.Settings.View.FontSize = 16
			a.OpenTool(PageSettings)
			tt.Frame()
			a.fontUI.open = "editor"
			a.fontUI.query = "serif"
		}},
		{"clean-hover-preview", func(a *App, tt *ui.Tester) {
			a.Text += "\n\n\nmore   text  here"
			settle(tt, 3)
			moveTo(t, tt, "删除多余空格")
		}},
		{"clean-no-result", func(a *App, tt *ui.Tester) { a.ws.query = "没有这个动作" }},
		{"find-error-explain", func(a *App, tt *ui.Tester) {
			a.setTab(1)
			a.find.modeIdx = 1
			a.find.opts.Find = `(\d+元`
			a.ws.find.explain = true
		}},
		{"find-explain-skipped", func(a *App, tt *ui.Tester) {
			a.setTab(1)
			a.find.modeIdx = 1
			a.find.opts.Find = `(\d+)元`
			a.find.opts.Replace = "¥$1"
			a.ws.find.explain = true
			settle(tt, 2)
			a.ws.find.skip = map[int]bool{1: true}
			a.jumpTo(1)
		}},
		{"find-pattern-menu", func(a *App, tt *ui.Tester) {
			a.setTab(1)
			a.togglePatternMenu()
			settle(tt, 4)
			moveTo(t, tt, "价格")
		}},
		{"find-pattern-applied", func(a *App, tt *ui.Tester) {
			a.setTab(1)
			p, _ := a.patternNamed("价格")
			a.applyPattern(p)
		}},
		{"toast-copied", func(a *App, tt *ui.Tester) {
			a.Lib.Settings.CopyOnDone = false
			a.done()
		}},
		{"chains-sheet", func(a *App, tt *ui.Tester) { a.OpenTool(PageChains) }},
		{"palette", func(a *App, tt *ui.Tester) { a.openPalette(); a.paletteQuery = "替换" }},
	}
	for _, dark := range []bool{false} {
		for _, sc := range scenes {
			a, tt := newApp(t, snapSample)
			tt.SetScale(2)
			tt.SetDark(dark)
			tt.SetTitleBar(ui.TitleBar{Height: 42, Left: 78})
			tt.Frame()
			sc.setup(a, tt)
			for i := 0; i < 40; i++ {
				tt.Frame()
			}
			name := sc.name
			if dark {
				name += "-dark"
			}
			img := tt.Image()
			lights(img, 2)
			f, err := os.Create(filepath.Join(dir, name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			png.Encode(f, img)
			f.Close()
		}
	}
}

// lights paints stand-in macOS window controls where the title bar's
// controls would be, so screenshots read like the real window.
func lights(img image.Image, scale int) {
	m, ok := img.(draw.Image)
	if !ok {
		return
	}
	cols := []color.RGBA{{255, 95, 87, 255}, {254, 188, 46, 255}, {40, 200, 64, 255}}
	for i, col := range cols {
		cx, cy, r := (20+i*20)*scale, 21*scale, 6*scale
		for y := -r; y <= r; y++ {
			for x := -r; x <= r; x++ {
				if x*x+y*y <= r*r {
					m.Set(cx+x, cy+y, col)
				}
			}
		}
	}
}
