package main

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/egoist/mygo"
	"golang.org/x/text/encoding/simplifiedchinese"
)

// Document kinds.
const (
	KindClipboard = "clipboard" // the Clipboard Workspace
	KindFile      = "file"
	KindUntitled  = "untitled"
)

// DocInfo describes the document of a window.
type DocInfo struct {
	Kind string `json:"kind"`
	Path string `json:"path,omitempty"`
	Name string `json:"name"`
	Text string `json:"text"`
}

type doc struct {
	win     *mygo.Window
	kind    string
	path    string
	text    string // last text the page synced
	dirty   bool   // changed since opened or saved (files and untitled)
	closing bool   // the page confirmed the close
	pending bool   // waiting for the page to answer CloseRequested
}

func (d *doc) info() DocInfo {
	return DocInfo{Kind: d.kind, Path: d.path, Name: d.name(), Text: d.text}
}

func (d *doc) name() string {
	switch d.kind {
	case KindClipboard:
		return "剪贴板工作区"
	case KindFile:
		return filepath.Base(d.path)
	}
	return "未命名"
}

var docs = struct {
	sync.Mutex
	byWin    map[int]*doc
	untitled int
}{byWin: map[int]*doc{}}

var quitting bool

func docOf(w *mygo.Window) *doc {
	if w == nil {
		return nil
	}
	docs.Lock()
	defer docs.Unlock()
	return docs.byWin[w.ID()]
}

func workspaceDoc() *doc {
	docs.Lock()
	defer docs.Unlock()
	for _, d := range docs.byWin {
		if d.kind == KindClipboard {
			return d
		}
	}
	return nil
}

func docWindows() []*doc {
	docs.Lock()
	defer docs.Unlock()
	out := make([]*doc, 0, len(docs.byWin))
	for _, d := range docs.byWin {
		out = append(out, d)
	}
	return out
}

func windowBackground() string { return "light-dark(#fbfbfc, #1c1c1f)" }

// openDocument opens a document window.
func openDocument(kind, path, text string) *mygo.Window {
	d := &doc{kind: kind, path: path, text: text}
	if kind == KindUntitled {
		docs.Lock()
		docs.untitled++
		docs.Unlock()
	}
	opts := mygo.WindowOptions{
		Title:           d.name(),
		URL:             "/",
		Width:           1180,
		Height:          760,
		MinWidth:        760,
		MinHeight:       460,
		TitleBarStyle:   mygo.TitleBarHiddenInset,
		TitleBarHeight:  52,
		BackgroundColor: windowBackground(),
	}
	if kind == KindClipboard {
		opts.StateKey = "workspace"
	} else if n := len(docWindows()); n > 0 {
		if focused := focusedDoc(); focused != nil {
			x, y := focused.win.Position()
			opts.X, opts.Y = x+28, y+28
		}
	}
	w := mygo.NewWindow(opts)
	d.win = w
	docs.Lock()
	docs.byWin[w.ID()] = d
	docs.Unlock()
	w.OnClose(func(e *mygo.CloseEvent) { onDocClose(d, e) })
	w.OnClosed(func() {
		docs.Lock()
		delete(docs.byWin, w.ID())
		docs.Unlock()
	})
	return w
}

func focusedDoc() *doc {
	for _, d := range docWindows() {
		if d.win.IsFocused() {
			return d
		}
	}
	return nil
}

// onDocClose copies the Clipboard Workspace back to the clipboard and asks
// about unsaved files. A close the user starts first asks the page for its
// final text; quitting uses the text the page last synced.
func onDocClose(d *doc, e *mygo.CloseEvent) {
	if d.closing {
		return
	}
	if !quitting {
		e.PreventDefault()
		if d.pending {
			return
		}
		d.pending = true
		if CloseRequested.Emit(d.win, true) != nil {
			d.win.Destroy()
			return
		}
		// A page that does not answer (still loading, hung) must not keep
		// its window open.
		go func() {
			time.Sleep(2 * time.Second)
			mygo.RunOnMain(func() {
				if d.pending {
					finishClose(d, d.text)
				}
			})
		}()
		return
	}
	if !confirmClose(d, d.text) {
		e.PreventDefault()
		return
	}
	d.closing = true
}

// finishClose runs once the page answered CloseRequested.
func finishClose(d *doc, text string) {
	mygo.RunOnMain(func() {
		if !d.pending || d.closing || d.win.IsDestroyed() {
			return
		}
		d.pending = false
		d.text = text
		if !confirmClose(d, text) {
			return
		}
		d.closing = true
		d.win.Destroy()
	})
}

// confirmClose does what closing a document needs and reports whether the
// window may close.
func confirmClose(d *doc, text string) bool {
	switch d.kind {
	case KindClipboard:
		if app.prefs().CopyBackOnClose && text != mygo.Clipboard.ReadText() {
			mygo.Clipboard.WriteText(text)
		}
		return true
	}
	if !d.dirty || (d.kind == KindUntitled && text == "") {
		return true
	}
	r, _ := mygo.Dialog.Message(mygo.MessageOptions{
		Parent: d.win, Type: mygo.MessageWarning,
		Message: "要存储对“" + d.name() + "”所做的更改吗？",
		Detail:  "如果不存储，你的更改将会丢失。",
		Buttons: []string{"存储…", "取消", "不存储"}, DefaultButton: 0, CancelButton: 1,
	})
	switch r.Button {
	case 0:
		path, err := saveDoc(d, text, false)
		return err == nil && path != ""
	case 2:
		return true
	}
	return false
}

func saveDoc(d *doc, text string, as bool) (string, error) {
	path := d.path
	if as || d.kind != KindFile || path == "" {
		name := d.name()
		if d.kind != KindFile {
			name += ".txt"
		}
		p, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: d.win, Title: "存储", DefaultPath: name})
		if err != nil || p == "" {
			return "", err
		}
		path = p
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		return "", err
	}
	d.kind, d.path, d.text, d.dirty = KindFile, path, text, false
	d.win.SetTitle(d.name())
	DocChanged.Emit(d.win, d.info())
	return path, nil
}

// showWorkspace brings up the Clipboard Workspace with the clipboard: a new
// window, or the open one, whose text is replaced (undoably) when the
// clipboard holds something else.
func showWorkspace() {
	if app.prefs().ClipboardWorkspace == "disabled" {
		openDocument(KindUntitled, "", "")
		return
	}
	clip := mygo.Clipboard.ReadText()
	if d := workspaceDoc(); d != nil {
		if clip != "" && clip != d.text {
			LoadText.Emit(d.win, clip)
		}
		d.win.Show()
		d.win.Focus()
		mygo.App.Focus()
		return
	}
	openDocument(KindClipboard, "", clip)
	mygo.App.Focus()
}

// openFiles opens files in document windows, or focuses their windows.
func openFiles(paths []string) error {
	var errs []error
	for _, p := range paths {
		if d := docForPath(p); d != nil {
			d.win.Show()
			d.win.Focus()
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			errs = append(errs, err)
			continue
		}
		openDocument(KindFile, p, decodeText(data))
	}
	return errors.Join(errs...)
}

func docForPath(p string) *doc {
	for _, d := range docWindows() {
		if d.kind == KindFile && d.path == p {
			return d
		}
	}
	return nil
}

// decodeText reads UTF-8 (with or without BOM) or UTF-16 with BOM, and
// falls back to GB18030 for other bytes.
func decodeText(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte{0xEF, 0xBB, 0xBF}):
		return string(b[3:])
	case bytes.HasPrefix(b, []byte{0xFF, 0xFE}), bytes.HasPrefix(b, []byte{0xFE, 0xFF}):
		return decodeUTF16(b)
	case utf8.Valid(b):
		return string(b)
	}
	if s, err := simplifiedchinese.GB18030.NewDecoder().Bytes(b); err == nil {
		return string(s)
	}
	return strings.ToValidUTF8(string(b), "\uFFFD")
}

func decodeUTF16(b []byte) string {
	le := b[0] == 0xFF
	b = b[2:]
	u := make([]uint16, 0, len(b)/2)
	for i := 0; i+1 < len(b); i += 2 {
		if le {
			u = append(u, uint16(b[i])|uint16(b[i+1])<<8)
		} else {
			u = append(u, uint16(b[i])<<8|uint16(b[i+1]))
		}
	}
	var sb strings.Builder
	for i := 0; i < len(u); i++ {
		r := rune(u[i])
		if r >= 0xD800 && r < 0xDC00 && i+1 < len(u) {
			r = (r-0xD800)<<10 + (rune(u[i+1]) - 0xDC00) + 0x10000
			i++
		}
		sb.WriteRune(r)
	}
	return sb.String()
}

// readImports reads the files to import; a directory (a TextSoap library
// package) contributes every file inside it.
func readImports(paths []string) [][]byte {
	var out [][]byte
	for _, p := range paths {
		_ = filepath.WalkDir(p, func(path string, de os.DirEntry, err error) error {
			if err != nil || de.IsDir() {
				return nil
			}
			if data, err := os.ReadFile(path); err == nil {
				out = append(out, data)
			}
			return nil
		})
	}
	return out
}

func platform() string {
	if runtime.GOOS == "windows" {
		return "win32"
	}
	return runtime.GOOS
}

// ---------------------------------------------------------------------------

// Document is the document of the calling window.
type Document struct{}

// Info returns the calling window's document, with its initial text.
func (Document) Info(ctx context.Context) DocInfo {
	if d := docOf(mygo.CallerWindow(ctx)); d != nil {
		return d.info()
	}
	return DocInfo{Kind: KindUntitled, Name: "未命名"}
}

// Sync tells Go the current text, which closing and quitting use.
func (Document) Sync(ctx context.Context, text string, dirty bool) {
	if d := docOf(mygo.CallerWindow(ctx)); d != nil {
		d.text, d.dirty = text, dirty
	}
}

// Close answers CloseRequested with the final text.
func (Document) Close(ctx context.Context, text string) {
	if d := docOf(mygo.CallerWindow(ctx)); d != nil {
		finishClose(d, text)
	}
}

// Save saves the document, asking for a path when it has none. It returns
// the path, or "" when canceled.
func (Document) Save(ctx context.Context, text string, as bool) (string, error) {
	d := docOf(mygo.CallerWindow(ctx))
	if d == nil {
		return "", errors.New("没有文档")
	}
	return saveDoc(d, text, as)
}

// Open asks for files and opens them.
func (Document) Open(ctx context.Context) error {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "打开", Multiple: true})
	if err != nil || len(paths) == 0 {
		return err
	}
	return openFiles(paths)
}

// OpenPaths opens dropped files.
func (Document) OpenPaths(paths []string) error { return openFiles(paths) }

// New opens an untitled document.
func (Document) New() { openDocument(KindUntitled, "", "") }

// Workspace brings up the Clipboard Workspace.
func (Document) Workspace() { showWorkspace() }

// OpenLibrary shows MyLibrary, optionally at a custom cleaner.
func (Document) OpenLibrary(uuid string) { openLibrary(uuid) }

// OpenPrefs shows Preferences.
func (Document) OpenPrefs() { openSingle("prefs") }

// OpenBatch shows Batch File Cleaning.
func (Document) OpenBatch() { openSingle("batch") }

// Confirm asks a yes/no question attached to the calling window.
func (Document) Confirm(ctx context.Context, message, detail, ok string) bool {
	if ok == "" {
		ok = "好"
	}
	r, err := mygo.Dialog.Message(mygo.MessageOptions{
		Parent: mygo.CallerWindow(ctx), Type: mygo.MessageWarning, Message: message, Detail: detail,
		Buttons: []string{ok, "取消"}, DefaultButton: 0, CancelButton: 1,
	})
	return err == nil && r.Button == 0
}

// Hide hides the app (macOS) or the window, so the previous app comes back
// for pasting.
func (Document) Hide(ctx context.Context) {
	if runtime.GOOS == "darwin" {
		mygo.App.Hide()
		return
	}
	if w := mygo.CallerWindow(ctx); w != nil {
		w.Minimize()
	}
}

// ---------------------------------------------------------------------------
// Secondary windows: one of each.

var singles = struct {
	sync.Mutex
	m map[string]*mygo.Window
}{m: map[string]*mygo.Window{}}

func openSingle(kind string) *mygo.Window {
	singles.Lock()
	w := singles.m[kind]
	singles.Unlock()
	if w != nil && !w.IsDestroyed() {
		w.Show()
		w.Focus()
		return w
	}
	opts := mygo.WindowOptions{
		URL: "/" + kind, StateKey: kind, BackgroundColor: windowBackground(),
		TitleBarStyle: mygo.TitleBarHiddenInset, TitleBarHeight: 52,
	}
	switch kind {
	case "library":
		opts.Title, opts.Width, opts.Height, opts.MinWidth, opts.MinHeight = "我的库", 1240, 800, 980, 560
	case "prefs":
		opts.Title, opts.Width, opts.Height, opts.MinWidth, opts.MinHeight = "偏好设置", 720, 600, 620, 480
	case "batch":
		opts.Title, opts.Width, opts.Height, opts.MinWidth, opts.MinHeight = "批量处理文件", 820, 640, 680, 520
	}
	w = mygo.NewWindow(opts)
	singles.Lock()
	singles.m[kind] = w
	singles.Unlock()
	w.OnClosed(func() {
		singles.Lock()
		if singles.m[kind] == w {
			delete(singles.m, kind)
		}
		singles.Unlock()
	})
	return w
}

func openLibrary(uuid string) {
	singles.Lock()
	existing := singles.m["library"]
	singles.Unlock()
	w := openSingle("library")
	if uuid != "" {
		if existing != nil && !existing.IsDestroyed() {
			SelectCleaner.Emit(w, uuid)
		} else {
			pendingSelect = uuid
		}
	}
}

var pendingSelect string

// PendingSelection returns (once) the cleaner MyLibrary should open at.
func (LibraryService) PendingSelection() string {
	s := pendingSelect
	pendingSelect = ""
	return s
}
