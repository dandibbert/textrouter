package engine

import (
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

func TestLibraryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "lib.json")
	lib := LoadLibrary(p)
	if len(lib.Cleaners) == 0 {
		t.Fatal("first launch should have example cleaners")
	}
	c := lib.PutCleaner(CustomCleaner{Name: "X", Actions: []Action{act("applycleaner", Params{"cleaner": "Uppercase"})}})
	lib.PutGroup(Group{Name: "Mine", Items: []GroupItem{{Kind: "cleaner", Name: "X"}, {Kind: "divider"}}})
	lib.Prefs.ShowInvisibles = true
	lib.RememberSearch(SearchEntry{Find: "a"})
	if err := lib.Save(); err != nil {
		t.Fatal(err)
	}
	back := LoadLibrary(p)
	got := back.CleanerByUUID(c.UUID)
	if got == nil || !back.Prefs.ShowInvisibles || len(back.Groups) != 1 || len(back.History) != 1 {
		t.Fatalf("library did not round trip: %+v", back)
	}
	if out, _ := got.Run("abc", back.Env(nil)); out != "ABC" {
		t.Errorf("got %q", out)
	}
	back.PutCleaner(CustomCleaner{UUID: c.UUID, Name: "Y", Actions: got.Actions})
	if back.Groups[0].Items[0].Name != "Y" {
		t.Error("renaming a cleaner should update groups")
	}
}
