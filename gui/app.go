// Package gui is TextRouter's native interface, built with MyGo's ui
// package. It only talks to the engine package and to a small Platform
// interface, so every screen can be tested without a window.
package gui

import (
	"fmt"
	"strings"
	"time"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// Platform is what the interface needs from the desktop. All methods may
// be no-ops (tests use NopPlatform).
type Platform interface {
	OpenFiles(title string, exts []string, multiple, dirs bool) []string
	SaveFile(title, name string) string
	HideWindow()
	SetTheme(theme string)
	SetHotkey(acc string) error
	// Update runs fn on the UI thread and redraws (used after dialogs,
	// which block and so run on goroutines).
	Update(fn func())
}

// NopPlatform does nothing; used by tests.
type NopPlatform struct{}

func (NopPlatform) OpenFiles(string, []string, bool, bool) []string { return nil }
func (NopPlatform) SaveFile(string, string) string                  { return "" }
func (NopPlatform) HideWindow()                                     {}
func (NopPlatform) SetTheme(string)                                 {}
func (NopPlatform) SetHotkey(string) error                          { return nil }
func (NopPlatform) Update(fn func())                                { fn() }

// Page ids of the sidebar.
const (
	PageClipboard = "clipboard"
	PageFind      = "find"
	PageBatch     = "batch"
	PageChains    = "chains"
	PageCleaners  = "cleaners"
	PageFiles     = "files"
	PageSettings  = "settings"
)

// App is the whole interface state.
type App struct {
	Lib      *engine.Library
	Platform Platform

	Page string
	Text string // the Clipboard Space text, shared by every page as the sample

	last *lastOp

	ed            *editor.State // the Clipboard Space editor; its buffer holds Text
	edVer         uint64
	edSynced      string
	view          editorPrefs
	docs          []*document
	cur           int
	inspector     string // "cleaners" or "find": the tab of the right column
	insp          inspectorState
	tr            *trail
	ws            workspace
	clipAt        time.Time
	sheetFresh    string
	fakeMenu      []string
	changes       []changeSpan
	changesVer    uint64
	changesAt     time.Time
	viewSaved     bool
	fontUI        fontUI
	freezeChanges bool // screenshots: no fade // screenshots only: stand-in for the native popup menu

	paletteOpen  bool
	paletteQuery string
	paletteSel   int
	paletteFresh bool

	find    findState
	batch   batchState
	chains  chainsState
	clean   cleanersState
	files   filesState
	setting settingsState

	sections  map[string]*bool
	status    string
	toast     string
	toastUndo bool
	monoEdit  bool
	writeClip func(string)
	// clipLoaded is the clipboard text last loaded into (or copied from)
	// the 剪贴板 document; clipPrev is that document's text right after.
	clipSeen   bool
	clipLoaded string
	clipPrev   string
	// clipPending is a newer clipboard not loaded because the document
	// had edits; the toolbar offers it.
	clipPending    string
	clipHasPending bool
}

type lastOp struct {
	label    string
	before   string
	after    string
	scope    string
	showDiff bool
}

// New creates the interface state.
func New(lib *engine.Library, p Platform) *App {
	if p == nil {
		p = NopPlatform{}
	}
	a := &App{Lib: lib, Platform: p, Page: PageClipboard, sections: map[string]*bool{}}
	a.find.opts = engine.FindOptions{MatchCase: true}
	if len(lib.RuleSets) > 0 {
		a.batch.setID = lib.RuleSets[0].ID
	}
	if len(lib.Chains) > 0 {
		a.chains.selID = lib.Chains[0].ID
	}
	a.clean.category = "全部"
	a.batch.showDiff = true
	a.restoreView()
	return a
}

// LoadClipboard is what every activation calls (launch, Dock icon, app
// activation, menu bar icon, global shortcut) with the clipboard's text:
// it switches to the 剪贴板 document and reads the clipboard into it.
//
//   - Clipboard unchanged since it was last loaded or copied from here:
//     the document is left alone (carets, undo history, edits).
//   - Changed, document unedited: replaced silently (one undoable edit).
//   - Changed, document has edits not yet copied out: the document is
//     kept exactly as is (text, carets, undo); a pill in the toolbar,
//     「剪贴板有新内容 · 载入」, loads it on click as an undoable edit.
//   - Auto-read off in the settings: only switches to the document.
//
// It reports whether the document was replaced.
func (a *App) LoadClipboard(s string) bool {
	a.ShowClipboard()
	if a.Lib.Settings.NoAutoReadClipboard {
		return false
	}
	a.syncEditor()
	if a.clipSeen && s == a.clipLoaded {
		return false // nothing new on the clipboard
	}
	first := !a.clipSeen
	a.clipSeen, a.clipLoaded = true, s
	if s == a.Text {
		a.clipHasPending, a.clipPending = false, ""
		return false
	}
	edited := !first && a.Text != a.clipPrev && a.Text != ""
	if edited {
		// Edits not yet copied out: keep the document exactly as it is and
		// offer the new clipboard in a pill instead.
		a.clipPending, a.clipHasPending = s, true
		return false
	}
	a.applyClipboard(s)
	a.status = "已读取剪贴板"
	return true
}

// applyClipboard loads s into the 剪贴板 document as one undoable edit.
func (a *App) applyClipboard(s string) {
	a.clipHasPending, a.clipPending = false, ""
	a.replaceAll(s)
	a.clipPrev = a.Text
	a.clearDiffBase()
	a.changes = nil
	a.last = nil
	a.tr = nil
	a.clipAt = time.Now()
}

// LoadPendingClipboard loads the clipboard the pill offers (undoable).
func (a *App) LoadPendingClipboard() {
	if !a.clipHasPending {
		return
	}
	a.ShowClipboard()
	a.catchUp()
	old, before := a.tr, a.Text
	a.applyClipboard(a.clipPending)
	a.tr = old // the load is a step of the same trail, so ⌘Z finds the edits again
	a.pushStep("载入新剪贴板", nil)
	a.last = &lastOp{label: "载入新剪贴板", before: before, after: a.Text, scope: "全文"}
	a.status = "已载入新剪贴板（⌘Z 可找回之前的编辑）"
}

// noteCopied records text this app put on the clipboard, so the next
// activation sees nothing new and leaves the document alone.
func (a *App) noteCopied(s string) {
	a.clipSeen, a.clipLoaded = true, s
	a.clipHasPending, a.clipPending = false, ""
	if a.cur == 0 {
		a.syncEditor()
		a.clipPrev = a.Text
	}
}

// Transform applies fn as one step of the trail: to each selection of the
// editor when there is one, else to the whole text.
func (a *App) Transform(label string, fn func(string) string) { a.transformStep(label, nil, fn) }

// transformStep is Transform that also records the chain step that redoes it
// (nil when there is none), and plays the apply animation.
func (a *App) transformStep(label string, act *engine.Step, fn func(string) string) {
	a.catchUp()
	before := a.Text
	scope := a.applyScoped(fn)
	after := a.Text
	if after == before {
		a.status = "「" + label + "」没有改变" + scope
		return
	}
	a.last = &lastOp{label: label, before: before, after: after, scope: scope}
	a.ed.DiffBase = nil
	a.markChanges(before, after)
	ins, del := engine.DiffStats(cachedDiff(before, after))
	a.status = fmt.Sprintf("已对%s应用「%s」：+%d −%d 字符", scope, label, ins, del)
	a.pushStep(label, act)
	a.ws.startApplyFx(before, after)
}

func (a *App) save() {
	if err := a.Lib.Save(); err != nil {
		a.status = "保存失败：" + err.Error()
	}
}

func (a *App) open(name string) *bool {
	b, ok := a.sections[name]
	if !ok {
		v := true
		b = &v
		a.sections[name] = b
	}
	return b
}

// navItem describes a secondary tool (opened as a sheet).
type navItem struct{ id, icon, label string }

// Tools are second-level: opened as sheets over the document from the
// Tools menu, the app menu bar or ⌘K.
var tools = []navItem{
	{PageBatch, "replace-all", "批量替换"},
	{PageChains, "workflow", "我的清理器（动作链）"},
	{PageCleaners, "library-big", "清理器库"},
	{PageFiles, "files", "批量处理文件"},
	{PageSettings, "settings-2", "设置"},
}

// header is kept for pages: title, subtitle and right-side actions.
func header(c *ui.Context, title, subtitle string, actions func()) {
	pageHeader(c, title, subtitle, actions)
}

// card is a raised panel.
func card(c *ui.Context) ui.Element { return panel(c) }

func muted(c *ui.Context, s string) ui.Element { return hint(c, s) }

// colours used for highlights in the pages' diffs.
func hlAdd(t *ui.Theme) ui.Color { return Ins }

func hlDel(t *ui.Theme) ui.Color { return Del }

func hlMatch(t *ui.Theme) ui.Color { return Hit }

// visible makes whitespace visible inside highlighted snippets.
func visible(s string) string {
	return strings.NewReplacer("\n", "↵\n", "\t", "→", "\u00a0", "⍽").Replace(s)
}

const maxPreview = 20000

// diffView shows before/after as one text with deletions struck through
// in red and insertions in green.
func diffView(c *ui.Context, before, after string) {
	t := c.Theme()
	if engine.RuneLen(before)+engine.RuneLen(after) > 2*maxPreview {
		ui.Text(c, "文本较大，只显示结果：").FontSize(11).TextColor(t.TextMuted)
		ui.Text(c, after).Selectable().Font(editorFont).FontSize(15).LineHeight(1.65)
		return
	}
	segs := cachedDiff(before, after)
	var spans []ui.Span
	for _, s := range segs {
		switch s.Op {
		case '=':
			spans = append(spans, ui.Span{Text: s.Text})
		case '+':
			spans = append(spans, ui.Span{Text: visible(s.Text), Background: hlAdd(t)})
		case '-':
			spans = append(spans, ui.Span{Text: visible(s.Text), Background: hlDel(t), Strikethrough: true, Color: Ink, DecorationColor: Ink, DecorationThickness: strokeW})
		}
	}
	if len(spans) == 0 {
		spans = []ui.Span{{Text: " "}}
	}
	ui.RichText(c, spans...).Selectable().Font(editorFont).FontSize(15).LineHeight(1.65).TextColor(tok(c).text)
}

// diffBadge summarises a change.
func diffBadge(c *ui.Context, before, after string) {
	if before == after {
		badge(c, "无变化", toneNeutral)
		return
	}
	ins, del := engine.DiffStats(cachedDiff(before, after))
	ui.Row(c).Gap(4).Children(func() {
		if ins > 0 {
			badge(c, fmt.Sprintf("+%d", ins), toneGreen)
		}
		if del > 0 {
			badge(c, fmt.Sprintf("−%d", del), toneRed)
		}
	})
}
