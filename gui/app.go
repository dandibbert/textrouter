// Package gui is TextRouter's native interface, built with MyGo's ui
// package. It only talks to the engine package and to a small Platform
// interface, so every screen can be tested without a window.
package gui

import (
	"fmt"
	"strings"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
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

	history []snapshot // undo stack of applied operations
	future  []snapshot
	last    *lastOp

	paletteOpen  bool
	paletteQuery string
	paletteSel   int

	find    findState
	batch   batchState
	chains  chainsState
	clean   cleanersState
	files   filesState
	setting settingsState

	sections  map[string]*bool
	status    string
	writeClip func(string)
}

type snapshot struct {
	text, label string
}

type lastOp struct {
	label    string
	before   string
	after    string
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
	return a
}

// LoadClipboard replaces the workspace text (called when the window opens).
func (a *App) LoadClipboard(s string) {
	if s == a.Text {
		return
	}
	a.push("读取剪贴板")
	a.Text = s
	a.last = nil
}

func (a *App) push(label string) {
	a.history = append(a.history, snapshot{a.Text, label})
	if len(a.history) > 200 {
		a.history = a.history[1:]
	}
	a.future = nil
}

// Transform applies fn to the workspace text as one undoable operation.
func (a *App) Transform(label string, fn func(string) string) {
	before := a.Text
	after := fn(before)
	if after == before {
		a.status = "「" + label + "」没有改变文本"
		return
	}
	a.push(label)
	a.Text = after
	a.last = &lastOp{label: label, before: before, after: after}
	ins, del := engine.DiffStats(engine.Diff(before, after))
	a.status = fmt.Sprintf("已应用「%s」：+%d −%d 字符", label, ins, del)
}

// Undo restores the text before the last operation.
func (a *App) Undo() {
	if len(a.history) == 0 {
		return
	}
	s := a.history[len(a.history)-1]
	a.history = a.history[:len(a.history)-1]
	a.future = append(a.future, snapshot{a.Text, s.label})
	a.Text = s.text
	a.last = nil
	a.status = "已撤销「" + s.label + "」"
}

// Redo re-applies an undone operation.
func (a *App) Redo() {
	if len(a.future) == 0 {
		return
	}
	s := a.future[len(a.future)-1]
	a.future = a.future[:len(a.future)-1]
	a.history = append(a.history, snapshot{a.Text, s.label})
	a.Text = s.text
	a.status = "已重做「" + s.label + "」"
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

// View is the window content.
func (a *App) View(c *ui.Context) {
	t := c.Theme()
	a.writeClip = c.WriteClipboard
	c.OnShortcut(ui.Cmd, ui.KeyK, func() { a.openPalette() })
	c.OnShortcut(ui.Cmd|ui.Shift, ui.KeyZ, a.Redo)
	c.OnShortcut(ui.Cmd, ui.Key1, func() { a.Page = PageClipboard })
	ui.Row(c).Fill().Background(t.Background).Children(func() {
		ui.Column(c).Width(212).FillHeight().Background(t.Surface).Children(func() {
			ui.Row(c).Padding(18, 16, 10, 16).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(22, 22).Radius(6).Gradient(ui.Hex("#7C5CFF"), ui.Hex("#22C3EE"), 135)
				ui.Text(c, "TextRouter").FontSize(16).Bold()
			})
			ui.Sidebar(c, &a.Page, func() {
				ui.SidebarSection(c, "日常", nil, func() {
					ui.SidebarItem(c, PageClipboard, nil, "剪贴板空间")
					ui.SidebarItem(c, PageFind, nil, "查找替换")
					ui.SidebarItem(c, PageBatch, nil, "批量替换")
				})
				ui.SidebarSection(c, "工具箱", nil, func() {
					ui.SidebarItem(c, PageChains, nil, "动作链").Children(func() {
						ui.Badge(c, fmt.Sprint(len(a.Lib.Chains)))
					})
					ui.SidebarItem(c, PageCleaners, nil, "清理器库")
					ui.SidebarItem(c, PageFiles, nil, "批量处理文件")
					ui.SidebarItem(c, PageSettings, nil, "设置")
				})
			}).Grow(1)
			ui.Column(c).Padding(12, 16).Gap(4).Children(func() {
				if ui.Button(c, "全部动作  ⌘K").FillWidth().Clicked() {
					a.openPalette()
				}
			})
		})
		ui.Divider(c).Vertical()
		ui.Column(c).Grow(1).FillHeight().MinWidth(0).Children(func() {
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
}

func (a *App) statusBar(c *ui.Context) {
	t := c.Theme()
	ui.Divider(c)
	ui.Row(c).Padding(6, 16).Gap(12).AlignItems(ui.Center).Children(func() {
		st := a.status
		if st == "" {
			st = "就绪"
		}
		ui.Text(c, st).FontSize(11).TextColor(t.TextMuted).Grow(1).MaxLines(1)
		lines := strings.Count(a.Text, "\n") + 1
		if a.Text == "" {
			lines = 0
		}
		ui.Textf(c, "%d 字符 · %d 行", engine.RuneLen(a.Text), lines).FontSize(11).TextColor(t.TextMuted)
	})
}

// header draws a page title with a subtitle and right-side actions.
func header(c *ui.Context, title, subtitle string, actions func()) {
	t := c.Theme()
	ui.Row(c).Padding(18, 20, 12, 20).Gap(10).AlignItems(ui.Center).Children(func() {
		ui.Column(c).Grow(1).Gap(3).MinWidth(0).Children(func() {
			ui.Text(c, title).FontSize(20).Bold()
			if subtitle != "" {
				ui.Text(c, subtitle).FontSize(12).TextColor(t.TextMuted)
			}
		})
		if actions != nil {
			actions()
		}
	})
}

// card is a rounded panel.
func card(c *ui.Context) ui.Element {
	t := c.Theme()
	return ui.Column(c).Background(t.Surface).Radius(10).Border(1, t.Border).Padding(12).Gap(8)
}

func muted(c *ui.Context, s string) ui.Element {
	return ui.Text(c, s).FontSize(12).TextColor(c.Theme().TextMuted)
}

// colors used for highlights, adapted to light/dark.
func hlAdd(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(46, 160, 67, 0.35)
	}
	return ui.RGBA(46, 160, 67, 0.22)
}

func hlDel(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(248, 81, 73, 0.35)
	}
	return ui.RGBA(248, 81, 73, 0.2)
}

func hlMatch(t *ui.Theme) ui.Color {
	if t.Dark {
		return ui.RGBA(250, 204, 21, 0.35)
	}
	return ui.RGBA(250, 204, 21, 0.45)
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
		ui.Text(c, after).Selectable().Font("monospace").FontSize(13)
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
			spans = append(spans, ui.Span{Text: visible(s.Text), Background: hlDel(t), Strikethrough: true, Color: t.TextMuted})
		}
	}
	if len(spans) == 0 {
		spans = []ui.Span{{Text: " "}}
	}
	ui.RichText(c, spans...).Selectable().Font("monospace").FontSize(13)
}

// diffBadge summarises a change.
func diffBadge(c *ui.Context, before, after string) {
	if before == after {
		ui.Badge(c, "无变化")
		return
	}
	ins, del := engine.DiffStats(cachedDiff(before, after))
	ui.Badge(c, fmt.Sprintf("+%d −%d", ins, del))
}
