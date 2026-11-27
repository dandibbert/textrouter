package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActivationLoadsClipboard(t *testing.T) {
	a, tt := newApp(t, "")
	tt.Frame()
	// Launch into empty workspace.
	if !a.LoadClipboard("first copy") || a.Text != "first copy" || a.toast != "" {
		t.Fatalf("launch: %q toast %q", a.Text, a.toast)
	}
	tt.Frame()
	// Unchanged clipboard: the document, carets and undo are left alone.
	a.ed.Select(2, 5)
	ver := a.ed.Buffer.Version()
	if a.LoadClipboard("first copy") || a.ed.Buffer.Version() != ver {
		t.Fatal("unchanged clipboard touched the document")
	}
	if s := a.ed.Selection(); s.Start() != 2 || s.End() != 5 {
		t.Fatalf("selection moved: %+v", s)
	}
	// New clipboard while workspace already has content (even unedited):
	// never replaces automatically; only the pill appears.
	if a.LoadClipboard("second copy") || a.Text != "first copy" || a.ed.Buffer.Version() != ver || a.toast != "" {
		t.Fatalf("displayed doc was touched: %q", a.Text)
	}
	if s := a.ed.Selection(); s.Start() != 2 || s.End() != 5 {
		t.Fatalf("selection moved: %+v", s)
	}
	tt.Frame()
	if !tt.HasText("剪贴板有新内容 · 载入") {
		t.Fatal("pill missing")
	}
	// Clicking the pill loads it, undoably.
	if err := tt.Click("剪贴板有新内容 · 载入"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Text != "second copy" || tt.HasText("剪贴板有新内容 · 载入") {
		t.Fatalf("pill load: %q", a.Text)
	}
	a.Undo()
	if a.Text != "first copy" {
		t.Fatalf("undo load: %q", a.Text)
	}
	// Edit the document, then a new clipboard arrives: nothing changes,
	// only the pill appears.
	a.Text = "second copy, edited by me"
	tt.Frame()
	a.ed.Select(3, 6)
	ver = a.ed.Buffer.Version()
	if a.LoadClipboard("third copy") || a.Text != "second copy, edited by me" || a.ed.Buffer.Version() != ver || a.toast != "" {
		t.Fatalf("edited doc was touched: %q", a.Text)
	}
	if s := a.ed.Selection(); s.Start() != 3 || s.End() != 6 {
		t.Fatalf("selection moved: %+v", s)
	}
	tt.Frame()
	if !tt.HasText("剪贴板有新内容 · 载入") {
		t.Fatal("pill missing")
	}
	// Activating again with the same clipboard: still untouched.
	if a.LoadClipboard("third copy") || a.Text != "second copy, edited by me" {
		t.Fatal("second activation loaded")
	}
	// Empty / non-text clipboard clears the stale pill without wiping content.
	if a.LoadClipboard("") || a.Text != "second copy, edited by me" {
		t.Fatal("empty clipboard touched the document")
	}
	tt.Frame()
	if tt.HasText("剪贴板有新内容 · 载入") {
		t.Fatal("pill stayed for empty clipboard")
	}
	// Restore third copy as pending and click the pill.
	if a.LoadClipboard("third copy") || a.Text != "second copy, edited by me" {
		t.Fatal("third copy replaced automatically")
	}
	tt.Frame()
	if err := tt.Click("剪贴板有新内容 · 载入"); err != nil {
		t.Fatal(err)
	}
	tt.Frame()
	if a.Text != "third copy" || tt.HasText("剪贴板有新内容 · 载入") {
		t.Fatalf("pill load: %q", a.Text)
	}
	a.Undo()
	if a.Text != "second copy, edited by me" {
		t.Fatalf("undo load: %q", a.Text)
	}
	// What we copy ourselves does not count as new.
	a.toast = ""
	a.writeClip("second copy, edited by me")
	if a.LoadClipboard("second copy, edited by me") || a.toast != "" {
		t.Fatal("own copy reloaded")
	}
	// An empty workspace still loads new clipboard content directly.
	a.Text = ""
	tt.Frame()
	if !a.LoadClipboard("fourth copy") || a.Text != "fourth copy" {
		t.Fatalf("empty workspace load: %q", a.Text)
	}
}

func TestActivationSwitchesToClipboardTab(t *testing.T) {
	a, tt := newApp(t, "")
	tt.Frame()
	a.LoadClipboard("clip")
	tt.Frame()
	p := filepath.Join(t.TempDir(), "f.txt")
	os.WriteFile(p, []byte("file text"), 0o644)
	a.loadFile(p)
	tt.Frame()
	// Unchanged clipboard: back to the 剪贴板 tab, file untouched.
	a.LoadClipboard("clip")
	if a.cur != 0 || a.Text != "clip" || a.docs[1].text != "file text" {
		t.Fatalf("cur %d text %q file %q", a.cur, a.Text, a.docs[1].text)
	}
	a.switchDoc(1)
	tt.Frame()
	// New clipboard switches to the 剪贴板 tab and offers the pill without
	// overwriting either the clipboard workspace or the file.
	a.LoadClipboard("new clip")
	if a.cur != 0 || a.Text != "clip" || !a.clipHasPending || a.clipPending != "new clip" || a.docs[1].text != "file text" {
		t.Fatalf("cur %d text %q pending %v/%q file %q", a.cur, a.Text, a.clipHasPending, a.clipPending, a.docs[1].text)
	}
	tt.Frame()
	if err := tt.Click("剪贴板有新内容 · 载入"); err != nil {
		t.Fatal(err)
	}
	if a.Text != "new clip" || a.docs[1].text != "file text" {
		t.Fatalf("after click: text %q file %q", a.Text, a.docs[1].text)
	}
}

func TestActivationAutoReadOff(t *testing.T) {
	a, tt := newApp(t, "")
	tt.Frame()
	a.LoadClipboard("clip")
	a.Lib.Settings.NoAutoReadClipboard = true
	if a.LoadClipboard("other") || a.Text != "clip" {
		t.Fatalf("auto-read off still loaded: %q", a.Text)
	}
	a.OpenTool(PageSettings)
	tt.Frame()
	if !tt.HasText("激活时自动读取剪贴板") {
		t.Fatal("setting missing")
	}
}
