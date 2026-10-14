package engine

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDiff(t *testing.T) {
	segs := Diff("hello world", "hello brave world!")
	var a, b string
	for _, s := range segs {
		if s.Op != '+' {
			a += s.Text
		}
		if s.Op != '-' {
			b += s.Text
		}
	}
	if a != "hello world" || b != "hello brave world!" {
		t.Errorf("diff does not reconstruct: %q %q", a, b)
	}
	if ins, del := DiffStats(segs); ins != 7 || del != 0 {
		t.Errorf("stats %d %d", ins, del)
	}
}

func TestLibraryAndFiles(t *testing.T) {
	dir := t.TempDir()
	lib := LoadLibrary(filepath.Join(dir, "lib.json"))
	lib.Chains = append(lib.Chains, Chain{ID: "x", Name: "X", Steps: []Step{{Kind: KindCleaner, Cleaner: "Uppercase"}}})
	if err := lib.Save(); err != nil {
		t.Fatal(err)
	}
	back := LoadLibrary(filepath.Join(dir, "lib.json"))
	if back.ChainByID("x") == nil || back.Settings.Hotkey == "" {
		t.Fatal("library did not round trip")
	}
	f := filepath.Join(dir, "a.txt")
	os.WriteFile(f, []byte("abc"), 0o644)
	ch := back.ChainByID("x")
	res := ProcessFiles(ExpandPaths([]string{dir}), func(s string) string { o, _ := ch.Run(s, nil); return o }, true, true)
	var changed int
	for _, r := range res {
		if r.Changed {
			changed++
		}
	}
	data, _ := os.ReadFile(f)
	if string(data) != "ABC" || changed == 0 {
		t.Errorf("file %q changed %d", data, changed)
	}
}
