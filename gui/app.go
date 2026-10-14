// Package gui is TextRouter's native interface, built with MyGo's ui
// package. It only talks to the engine package and to a small Platform
// interface, so every screen can be tested without a window.
package gui

import (
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

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
	sideOpen      bool
	docs          []*document
	cur           int
	inspector     string // "", "cleaners" or "find"
	insp          inspectorState
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

// LoadClipboard replaces the workspace text (called when the window opens).
// It is one undoable edit.
func (a *App) LoadClipboard(s string) {
	a.syncEditor()
	if s == a.Text {
		return
	}
	a.replaceAll(s)
	a.last = nil
	a.clearDiffBase()
}

// Transform applies fn as one undoable operation: to each selection of
// the editor when there is one, else to the whole text.
func (a *App) Transform(label string, fn func(string) string) {
	a.syncEditor()
	before := a.Text
	scope := a.applyScoped(fn)
	after := a.Text
	if after == before {
		a.status = "「" + label + "」没有改变" + scope
		return
	}
	a.last = &lastOp{label: label, before: before, after: after, scope: scope}
	a.ed.DiffBase = editor.NewBuffer(before)
	a.markChanges(before, after)
	ins, del := engine.DiffStats(engine.Diff(before, after))
	a.status = fmt.Sprintf("已对%s应用「%s」：+%d −%d 字符", scope, label, ins, del)
	a.toast, a.toastUndo = fmt.Sprintf("「%s」· %s · +%d −%d", label, scope, ins, del), true
}

// Undo undoes the last edit or action (one shared history).
func (a *App) Undo() {
	a.syncEditor()
	if !a.ed.Buffer.CanUndo() {
		return
	}
	a.ed.Buffer.Undo()
	a.syncEditor()
	a.last = nil
	a.clearDiffBase()
	a.status = "已撤销"
}

// Redo re-applies an undone edit.
func (a *App) Redo() {
	a.syncEditor()
	if !a.ed.Buffer.CanRedo() {
		return
	}
	a.ed.Buffer.Redo()
	a.syncEditor()
	a.status = "已重做"
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

// View is the window content: one text document (the clipboard or an
// opened file) with the cleaner/find inspector on the right.
func (a *App) View(c *ui.Context) {
	registerFonts()
	uiFontFamily = a.uiFamily()
	k := applyTheme(c)
	a.writeClip = c.WriteClipboard
	a.ensureDocs()
	a.persistView()
	if a.Page == PageFind {
		a.Page = PageClipboard
		a.showFind()
	}
	c.OnShortcut(ui.Cmd, ui.KeyK, func() { a.openPalette() })
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyZ, a.Redo)
	c.OnShortcut(ui.Cmd, ui.KeyF, a.showFind)
	c.OnShortcut(ui.Cmd|ui.Alt, ui.KeyS, func() { a.toggleInspector("cleaners") })
	c.OnShortcut(ui.Cmd, ui.KeyComma, func() { a.Page = PageSettings })
	if a.Page != PageClipboard && a.Page != "" && !a.paletteOpen {
		c.OnShortcut(0, ui.KeyEscape, func() { a.Page = PageClipboard })
	}
	a.syncEditor()
	ui.Column(c).Fill().Background(k.canvas).Children(func() {
		a.titleBar(c, k)
		ui.Row(c).Grow(1).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
			a.documentView(c, k)
			target := float32(0)
			if a.inspector != "" {
				target = inspectorW
			}
			holder := ui.Box(c.Key("inspwrap")).FillHeight().Clip().Width(target)
			if target > 0 {
				holder.Children(func() { a.inspectorView(c, k) })
			}
		})
		a.statusBar(c)
	})
	if a.Page != PageClipboard && a.Page != "" {
		a.sheetView(c, k)
	}
	if a.fakeMenu != nil {
		a.fakeMenuView(c, k)
	}
	a.paletteView(c)
	if a.toast != "" {
		msg := a.toast
		a.toast = ""
		if a.toastUndo {
			c.ToastAction(msg, "撤销", a.Undo)
		} else {
			c.Toast(msg)
		}
	}
}

// statusBar is the editor options bar under the text, as TextSoap's:
// zoom, line numbers, wrap, invisibles, font, and statistics.
func (a *App) statusBar(c *ui.Context) {
	k := tok(c)
	ui.Row(c).Height(30).Padding(0, sp3).Gap(2).AlignItems(ui.Center).Background(k.canvas).BorderWidth(1, 0, 0, 0).BorderColor(k.hair).Children(func() {
		small := func(key, icon, label string, on bool) bool {
			b := iconButton(c, key, icon, label, label).Size(24, 24)
			if on {
				b.Background(k.accentSoft)
			}
			return b.Clicked()
		}
		if small("zout", "minus", "缩小 ⌘-", false) {
			a.zoomBy(-1)
		}
		ui.Textf(c, "%d%%", int(a.view.zoom*100+0.5)).FontSize(11).TextColor(k.muted).FontFeatures("tnum").Width(36).TextAlign(ui.Center)
		if small("zin", "plus", "放大 ⌘+", false) {
			a.zoomBy(1)
		}
		ui.Box(c).Size(1, 14).Background(k.hair).Margin(0, sp1)
		if small("lnum", "list-ordered", "行号", a.view.lineNumbers) {
			a.view.lineNumbers = !a.view.lineNumbers
		}
		if small("wrap", "wrap-text", "自动换行", a.view.wrap) {
			a.view.wrap = !a.view.wrap
		}
		if small("invis", "pilcrow", "显示不可见字符（空格、制表符、换行、NBSP、全角空格）⌥⌘I", a.view.invisibles) {
			a.view.invisibles = !a.view.invisibles
		}
		if small("mono", "type", "等宽字体", a.view.mono) {
			a.view.mono = !a.view.mono
		}
		ui.Box(c).Width(sp3)
		st := a.status
		col := k.faint
		if strings.Contains(st, "失败") || strings.Contains(st, "错误") {
			col = k.red
		}
		ui.Text(c, st).FontSize(11).TextColor(col).Grow(1).Shrink(1).MinWidth(0).SingleLine()
		if a.last != nil {
			u := ui.ButtonBase(c.Key("sb-undo")).Height(20).Padding(0, 6).Radius(4).Label("撤销上一步")
			if u.Hovered() {
				u.Background(k.hover)
			}
			u.Children(func() { ui.Text(c, "撤销").FontSize(11).FontWeight(600).TextColor(k.accent) })
			if u.Clicked() {
				a.Undo()
			}
		}
		var parts []string
		if a.ed != nil {
			p := a.ed.CursorPos()
			ln := a.ed.Buffer.Line(p.Line)
			parts = append(parts, fmt.Sprintf("行 %d，列 %d", p.Line+1, utf8.RuneCountInString(ln[:min(p.Col, len(ln))])+1))
			if sels := a.selections(); len(sels) > 0 {
				n, lines := 0, 0
				for _, s := range sels {
					t := a.ed.Buffer.Slice(s.Start(), s.End())
					n += utf8.RuneCountInString(t)
					lines += strings.Count(t, "\n") + 1
				}
				parts = append(parts, fmt.Sprintf("已选 %d 字 · %d 行", n, lines))
			}
		}
		lines := strings.Count(a.Text, "\n") + 1
		if a.Text == "" {
			lines = 0
		}
		parts = append(parts, fmt.Sprintf("%d 字符", engine.RuneLen(a.Text)), fmt.Sprintf("%d 词", len(strings.Fields(a.Text))), fmt.Sprintf("%d 行", lines))
		for i, s := range parts {
			if i > 0 {
				ui.Text(c, "·").FontSize(11).TextColor(k.faint.Alpha(0.6)).Margin(0, 4)
			}
			c2 := k.faint
			if strings.HasPrefix(s, "已选") {
				c2 = k.accent
			}
			ui.Text(c, s).FontSize(11).TextColor(c2).FontFeatures("tnum").SingleLine()
		}
	})
}

// header is kept for pages: title, subtitle and right-side actions.
func header(c *ui.Context, title, subtitle string, actions func()) {
	pageHeader(c, title, subtitle, actions)
}

// card is a raised panel.
func card(c *ui.Context) ui.Element { return panel(c) }

func muted(c *ui.Context, s string) ui.Element { return hint(c, s) }

// colors used for highlights, adapted to light/dark.
func hlAdd(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(74, 222, 128, 0.20)
	}
	return ui.RGBA(34, 197, 94, 0.20)
}

func hlDel(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(248, 113, 113, 0.20)
	}
	return ui.RGBA(239, 68, 68, 0.14)
}

func hlMatch(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(250, 204, 21, 0.26)
	}
	return ui.RGBA(250, 204, 21, 0.38)
}

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
			spans = append(spans, ui.Span{Text: visible(s.Text), Background: hlDel(t), Strikethrough: true, Color: tok(c).red.Alpha(0.85), DecorationColor: tok(c).red.Alpha(0.6)})
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
