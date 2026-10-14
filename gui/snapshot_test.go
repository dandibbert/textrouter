package gui

import (
	"image/png"
	"os"
	"path/filepath"
	"testing"
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
			case PageCleaners:
				a.clean.selected = "Scrub"
			case PageClipboard:
				a.Transform("删除多余空格", func(s string) string { return s })
			}
			tt.Frame()
			tt.Frame()
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
	// palette
	a, tt := newApp(t, sample)
	tt.SetScale(2)
	a.openPalette()
	a.paletteQuery = "空格"
	tt.Frame()
	tt.Frame()
	f, _ := os.Create(filepath.Join(dir, "palette.png"))
	png.Encode(f, tt.Image())
	f.Close()
}
