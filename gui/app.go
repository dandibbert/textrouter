// Package gui is TextRouter's native interface, built with MyGo's ui
// package. It only talks to the engine package and to a small Platform
// interface, so every screen can be tested without a window.
package gui

import (
	"fmt"
	"strings"
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

	ed       *editor.State // the Clipboard Space editor; its buffer holds Text
	edVer    uint64
	edSynced string
	invisVer uint64
	invisOn  bool
	view     editorPrefs
	sideOpen bool
	filePath string

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
	a.view = editorPrefs{lineNumbers: true, wrap: true, fontSize: 15}
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

// navItem describes one sidebar entry.
type navItem struct{ id, icon, label string }

var navMain = []navItem{
	{PageClipboard, "clipboard-paste", "剪贴板空间"},
	{PageFind, "text-search", "查找替换"},
	{PageBatch, "replace-all", "批量替换"},
}

var navTools = []navItem{
	{PageChains, "workflow", "动作链"},
	{PageCleaners, "library-big", "清理器库"},
	{PageFiles, "files", "批量处理文件"},
}

// View is the window content.
func (a *App) View(c *ui.Context) {
	registerFonts()
	k := applyTheme(c)
	a.writeClip = c.WriteClipboard
	c.OnShortcut(ui.Cmd, ui.KeyK, func() { a.openPalette() })
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyZ, a.Redo)
	for i, id := range []string{PageClipboard, PageFind, PageBatch, PageChains, PageCleaners} {
		id := id
		c.OnShortcut(ui.Cmd, []ui.Key{ui.Key1, ui.Key2, ui.Key3, ui.Key4, ui.Key5}[i], func() { a.Page = id })
	}
	c.OnShortcut(ui.Cmd, ui.KeyComma, func() { a.Page = PageSettings })
	c.OnShortcut(ui.Cmd, ui.KeyBackslash, func() { a.sideOpen = !a.sideOpen })
	a.syncEditor()
	vib := c.Vibrancy()
	if vib {
		c.Root().Background(ui.Transparent)
	}
	headerLead = func() {
		bar := c.TitleBar()
		if !a.sideOpen && bar.Left > 0 {
			ui.Box(c).Width(bar.Left - sp3)
		}
		a.sidebarToggle(c)
		ui.Box(c).Width(sp1)
	}
	ui.Row(c).Fill().AlignItems(ui.Stretch).Children(func() {
		target := float32(0)
		if a.sideOpen {
			target = sidebarW
		}
		holder := ui.Box(c.Key("sidewrap")).FillHeight().Clip()
		w := holder.AnimateTo("w", target, ui.Smooth)
		holder.Width(w)
		if w > 0.5 {
			holder.Children(func() {
				side := ui.Column(c).Width(sidebarW).FillHeight().BorderWidth(0, 1, 0, 0).BorderColor(k.hair)
				if !vib {
					side.Background(k.sidebar)
				}
				side.Children(func() { a.sidebar(c) })
			})
		}
		bg := k.window
		if a.Page == PageClipboard {
			bg = k.canvas
		}
		ui.Column(c).Grow(1).FillHeight().MinWidth(0).Background(bg).Children(func() {
			switch a.Page {
			case PageFind:
				a.findView(c)
			case PageBatch:
				a.batchView(c)
			case PageChains:
				a.chainsView(c)
			case PageCleaners:
				a.cleanersView(c)
			case PageFiles:
				a.filesView(c)
			case PageSettings:
				a.settingsView(c)
			default:
				a.clipboardView(c)
			}
			a.statusBar(c)
		})
	})
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

func (a *App) sidebar(c *ui.Context) {
	k := tok(c)
	bar := c.TitleBar()
	// Room for the traffic lights; draggable like a title bar.
	ui.Row(c).Height(max(bar.Height, titleBarH)).FillWidth().DragWindow()
	ui.Column(c).Padding(0, sp3).Gap(2).Grow(1).Children(func() {
		// The palette launcher, styled as a search field.
		s := ui.ButtonBase(c.Key("side-search")).Height(30).Padding(0, 8, 0, 10).Radius(rMd).Gap(8).AlignItems(ui.Center).FillWidth().Label("全部动作").Tooltip("搜索并应用任何清理器、动作链或规则 ⌘K")
		pressable(c, s, k, k.field)
		s.Border(1, k.hair)
		s.Children(func() {
			ui.Icon(c, ic("search")).FontSize(14).TextColor(k.faint)
			ui.Text(c, "搜索动作…").FontSize(12.5).TextColor(k.faint).Grow(1).SingleLine()
			kbd(c, "⌘K")
		})
		if s.Clicked() {
			a.openPalette()
		}
		ui.Box(c).Height(sp3)
		eyebrow(c, "日常").Padding(sp1, sp2)
		for _, it := range navMain {
			a.navRow(c, it, "")
		}
		ui.Box(c).Height(sp3)
		eyebrow(c, "工具箱").Padding(sp1, sp2)
		for _, it := range navTools {
			n := ""
			switch it.id {
			case PageChains:
				n = fmt.Sprint(len(a.Lib.Chains))
			case PageCleaners:
				_, impl, _ := engine.CleanerStats()
				n = fmt.Sprint(impl)
			}
			a.navRow(c, it, n)
		}
		ui.Spacer(c)
		a.navRow(c, navItem{PageSettings, "settings-2", "设置"}, "")
		ui.Row(c).Padding(sp3, sp2, sp3, sp2).Gap(sp2).AlignItems(ui.Center).Children(func() {
			ui.Image(c, appIconBmp()).Size(18, 18).Radius(5)
			ui.Text(c, "TextRouter").FontSize(12).FontWeight(600).TextColor(k.muted)
			ui.Text(c, "0.2").FontSize(11).TextColor(k.faint)
		})
	})
}

func (a *App) navRow(c *ui.Context, it navItem, count string) {
	k := tok(c)
	sel := a.Page == it.id
	b := ui.ButtonBase(c.Key("nav:"+it.id)).Height(30).Padding(0, 8).Radius(rMd).Gap(9).AlignItems(ui.Center).FillWidth().Label(it.label).Selected(sel)
	bg := ui.Transparent
	switch {
	case sel:
		bg = k.pill
	case b.Pressed():
		bg = k.press
	case b.Hovered():
		bg = k.hover
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 140e6}))
	if b.FocusVisible() {
		b.Shadow(0, 0, 0, 2, k.accent.Alpha(0.4))
	}
	b.Children(func() {
		col := k.muted
		if sel {
			col = k.accent
		}
		ui.Icon(c, ic(it.icon)).FontSize(16).TextColor(col)
		w := 500
		if sel {
			w = 600
		}
		ui.Text(c, it.label).FontSize(13).FontWeight(w).TextColor(k.text).Grow(1).SingleLine()
		if count != "" {
			ui.Text(c, count).FontSize(11).FontWeight(500).TextColor(k.faint).FontFeatures("tnum")
		}
	})
	if b.Clicked() {
		a.Page = it.id
	}
}

func (a *App) statusBar(c *ui.Context) {
	k := tok(c)
	bg := k.window
	if a.Page == PageClipboard {
		bg = k.canvas
	}
	ui.Row(c).Height(26).Padding(0, sp4).Gap(sp3).AlignItems(ui.Center).Background(bg).BorderWidth(1, 0, 0, 0).BorderColor(k.hair).Children(func() {
		st := a.status
		if st == "" {
			st = "就绪"
		}
		col := k.faint
		if strings.Contains(st, "失败") || strings.Contains(st, "错误") {
			col = k.red
		}
		ui.Text(c, st).FontSize(11).TextColor(col).Grow(1).SingleLine()
		if a.last != nil && a.Page == PageClipboard {
			u := ui.ButtonBase(c.Key("sb-undo")).Height(18).Padding(0, 6).Radius(4).Label("撤销上一步")
			if u.Hovered() {
				u.Background(k.hover)
			}
			u.Children(func() { ui.Text(c, "撤销").FontSize(11).FontWeight(600).TextColor(k.accent) })
			if u.Clicked() {
				a.Undo()
			}
		}
		parts := []string{}
		if a.Page == PageClipboard && a.ed != nil {
			p := a.ed.CursorPos()
			parts = append(parts, fmt.Sprintf("行 %d，列 %d", p.Line+1, utf8.RuneCountInString(a.ed.Buffer.Line(p.Line)[:min(p.Col, len(a.ed.Buffer.Line(p.Line)))])+1))
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
		if a.Page == PageClipboard && a.view.fontSize != 15 && a.view.fontSize != 0 {
			parts = append(parts, fmt.Sprintf("%d%%", int(a.view.fontSize/15*100+0.5)))
		}
		for i, s := range parts {
			if i > 0 {
				ui.Text(c, "·").FontSize(11).TextColor(k.faint.Alpha(0.6))
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
