package gui

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActivationLoadsClipboard(t *testing.T) {
	a, tt := newApp(t, "")
	tt.Frame()
	// Launch.
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
	// New clipboard, no edits of ours: replaced quietly.
	if !a.LoadClipboard("second copy") || a.Text != "second copy" || a.toast != "" {
		t.Fatalf("new clipboard: %q toast %q", a.Text, a.toast)
	}
	tt.Frame()
	// Edit the document, then a new clipboard arrives: loaded, undoable, toast.
	a.Text = "second copy, edited by me"
	tt.Frame()
	if !a.LoadClipboard("third copy") || a.Text != "third copy" || a.toast != "已载入新剪贴板" || !a.toastUndo {
		t.Fatalf("edited: %q toast %q", a.Text, a.toast)
	}
	tt.Frame()
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
	// New clipboard goes into the 剪贴板 tab, never into the file.
	a.LoadClipboard("new clip")
	if a.cur != 0 || a.Text != "new clip" || a.docs[1].text != "file text" {
		t.Fatalf("cur %d text %q file %q", a.cur, a.Text, a.docs[1].text)
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
