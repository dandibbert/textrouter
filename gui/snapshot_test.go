package gui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"

	"github.com/dandibbert/textrouter/engine"
)

// TestSnapshots writes PNGs of every page when TEXTROUTER_SNAPSHOTS is a
// directory, for the README and design review.
func TestSnapshots(t *testing.T) {
	dir := os.Getenv("TEXTROUTER_SNAPSHOTS")
	if dir == "" {
		t.Skip("set TEXTROUTER_SNAPSHOTS=<dir>")
	}
	sample := "Hello  world, this is   a “sample” text.\n> quoted line from email\nprice 5元 and 12元\n中文English混排 test@example.com\n\nnew paragraph https://example.com"
	for _, dark := range []bool{false, true} {
		for _, page := range []string{PageClipboard, PageFind, PageBatch, PageChains, PageCleaners, PageFiles, PageSettings} {
			a, tt := newApp(t, sample)
			tt.SetScale(2)
			tt.SetDark(dark)
			a.Page = page
			switch page {
			case PageFind:
				a.find.modeIdx = 1
				a.find.opts.Find = `(\d+)元`
				a.find.opts.Replace = "¥$1"
			case PageBatch:
				a.Text = sample + "\nMy favourite colour is centre grey — organise it."
			case PageCleaners:
				a.clean.selected = "Scrub"
			case PageClipboard:
				cl := engine.CleanerByName("Capitalize with Title Case")
				a.Transform(cl.ZhName, cl.Fn)
			}
			for i := 0; i < 30; i++ {
				tt.Frame()
			}
			name := page
			if dark {
				name += "-dark"
			}
			f, err := os.Create(filepath.Join(dir, name+".png"))
			if err != nil {
				t.Fatal(err)
			}
			png.Encode(f, tt.Image())
			f.Close()
		}
	}
	for _, dark := range []bool{false, true} {
		a, tt := newApp(t, sample)
		tt.SetScale(2)
		tt.SetDark(dark)
		a.openPalette()
		a.paletteQuery = "空格"
		for i := 0; i < 40; i++ {
			tt.Frame()
		}
		name := "palette"
		if dark {
			name += "-dark"
		}
		f, _ := os.Create(filepath.Join(dir, name+".png"))
		png.Encode(f, tt.Image())
		f.Close()
	}
}
