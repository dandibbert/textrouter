package gui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// document is one text the window edits: the clipboard workspace or a
// file. Peers: each keeps its own editor, step trail and last action.
type document struct {
	name, path string
	clip       bool
	text       string
	ed         *editor.State
	edVer      uint64
	edSynced   string
	last       *lastOp
	tr         *trail
}

type inspectorState struct{}

func (a *App) ensureDocs() {
	if len(a.docs) == 0 {
		a.docs = []*document{{name: "剪贴板", clip: true}}
		a.cur = 0
	}
}

func (a *App) doc() *document {
	a.ensureDocs()
	return a.docs[a.cur]
}

// switchDoc makes document i current, keeping the state of the other.
func (a *App) switchDoc(i int) {
	a.ensureDocs()
	if i == a.cur || i < 0 || i >= len(a.docs) {
		return
	}
	a.syncEditor()
	d := a.docs[a.cur]
	d.text, d.ed, d.edVer, d.edSynced, d.last, d.tr = a.Text, a.ed, a.edVer, a.edSynced, a.last, a.tr
	n := a.docs[i]
	a.cur = i
	a.Text, a.ed, a.edVer, a.edSynced, a.last, a.tr = n.text, n.ed, n.edVer, n.edSynced, n.last, n.tr
	a.changes = nil
	a.ws.selLines, a.ws.applyFx, a.ws.hov = nil, nil, ""
	a.syncEditor()
}

func (a *App) closeDoc(i int) {
	if i <= 0 || i >= len(a.docs) {
		return // the clipboard workspace stays
	}
	if i == a.cur {
		a.switchDoc(0)
	}
	a.docs = append(a.docs[:i], a.docs[i+1:]...)
	if a.cur > i {
		a.cur--
	}
}

// ShowClipboard makes the clipboard workspace current.
func (a *App) ShowClipboard() { a.switchDoc(0) }

func (a *App) showFind() { a.setTab(1); a.ws.focusFnd = true }

// toggleInspector switches the right column to "cleaners" or "find".
func (a *App) toggleInspector(tab string) {
	if tab == "find" {
		a.setTab(1)
	} else {
		a.setTab(0)
	}
}

// OpenTool opens a second-level tool as a sheet over the document.
func (a *App) OpenTool(page string) { a.Page = page }

// ---------------------------------------------------------------- tabs and tools

// docTab is a document's tab in the title bar (shown once a file is open).
func (a *App) docTab(c *ui.Context, k tokens, i int, d *document) {
	sel := i == a.cur
	b := ui.ButtonBase(c.Key(fmt.Sprintf("doc%d:%s", i, d.path))).Height(30).Padding(0, 8, 0, 10).Radius(radiusCtl).Gap(6).AlignItems(ui.Center).Label(d.name).Selected(sel).MaxWidth(200)
	bg, bd := ui.Transparent, ui.Transparent
	switch {
	case sel:
		bg, bd = Accent, Ink
	case b.Hovered():
		bg = AccentSoft
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6})).Border(strokeW, bd)
	focusRing(b)
	b.Children(func() {
		icon := "file-text"
		if d.clip {
			icon = "clipboard-paste"
		}
		ui.Icon(c, ic(icon)).FontSize(13).TextColor(Ink)
		ui.Text(c, d.name).FontSize(sizeButton).FontWeight(map[bool]int{true: 700, false: 500}[sel]).TextColor(Ink).SingleLine()
		if !d.clip {
			x := ui.ButtonBase(c.Key("x")).Size(16, 16).Radius(radiusKey).Center().Label("关闭 " + d.name)
			if x.Hovered() {
				x.Background(Paper)
			}
			x.Children(func() { ui.Icon(c, ic("x")).FontSize(11).TextColor(Ink) })
			if x.Clicked() {
				a.closeDoc(i)
			}
		}
	})
	if b.Clicked() {
		a.switchDoc(i)
	}
}

// toolsMenu is the Tools menu: every second-level feature.
func (a *App) toolsMenu(c *ui.Context, k tokens) {
	b := ui.ButtonBase(c.Key("tools")).Height(34).Padding(0, 10).Gap(5).AlignItems(ui.Center).Radius(radiusCtl).Label("工具").Tooltip("批量替换、我的清理器、清理器库、批量处理文件")
	bg, bd := ui.Transparent, ui.Transparent
	if b.Hovered() {
		bg, bd = Paper, Ink
	}
	b.Background(bg).Border(strokeW, bd)
	focusRing(b)
	b.Children(func() {
		ui.Text(c, "工具").FontSize(sizeButton).FontWeight(500).TextColor(Ink)
		ui.Icon(c, ic("chevron-down")).FontSize(12).TextColor(Ink)
	})
	b.Menu(func(m *ui.Menu) { a.toolsMenuItems(c, m) })
}

func (a *App) toolsMenuItems(c *ui.Context, m *ui.Menu) {
	for _, t := range tools[:4] {
		if m.Item(t.label + "…").Chosen() {
			a.OpenTool(t.id)
		}
	}
	m.Separator()
	if m.Item("一键清理 Scrub").Chosen() {
		a.applyAction("Scrub")
	}
	if m.Item("粘贴覆盖（用剪贴板替换全部文本）").Chosen() {
		a.replaceWithUndo("粘贴覆盖", c.ReadClipboard())
	}
	if m.Item("全部复制").Chosen() {
		c.WriteClipboard(a.Text)
		a.noteCopied(a.Text)
		a.showToast("已复制全部文本", "")
	}
	m.Separator()
	if m.Item("打开文件…").Chosen() {
		a.openFile()
	}
	if m.Item("存储为…").Chosen() {
		a.saveFile()
	}
	m.Separator()
	if m.Item("设置…").Chosen() {
		a.OpenTool(PageSettings)
	}
}

// replaceWithUndo replaces the whole document as one step of the trail.
func (a *App) replaceWithUndo(label, s string) {
	a.catchUp()
	before := a.Text
	if s == before {
		return
	}
	a.replaceAll(s)
	a.last = &lastOp{label: label, before: before, after: a.Text, scope: "全文"}
	a.pushStep(label, nil)
	a.ws.startApplyFx(before, a.Text)
	a.status = "已" + label
}

// findBase is the text find/replace searches and its byte offset in the
// document: always the whole text.
func (a *App) findBase() (string, int) { return a.Text, 0 }

// ---------------------------------------------------------------- sheets

// sheetView shows a second-level tool as a sheet over the document.
func (a *App) sheetView(c *ui.Context, k tokens) {
	ui.Overlay(c, func() {
		scrim := ui.Box(c.Key("sheet-scrim")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(Ink.Alpha(0.35))
		if scrim.Clicked() {
			a.Page = PageClipboard
		}
		card := ui.Column(c.Key("sheet")).Absolute().Top(44).Left(48).Right(48).Bottom(36).Radius(radiusCtl).Background(Ground).Clip()
		boxed(card)
		hard(card, shadowLg)
		card.OnShortcut(0, ui.KeyEscape, func() { a.Page = PageClipboard })
		y := card.AnimateWith("y", 0, dPanel, EaseOut)
		if a.sheetFresh != a.Page {
			a.sheetFresh = a.Page
		}
		card.Translate(0, y)
		prev := headerLead
		headerLead = func() {
			if iconButton(c, "sheet-close", "x", "关闭", "关闭 esc").Size(30, 30).Clicked() {
				a.Page = PageClipboard
			}
		}
		card.Children(func() {
			switch a.Page {
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
			}
		})
		headerLead = prev
	})
}

func (a *App) findText() string { t, _ := a.findBase(); return t }

// Exported entry points for the app menu bar.
func (a *App) OpenFileDialog() { a.openFile() }
func (a *App) SaveFileDialog() { a.saveFile() }
func (a *App) ShowFind()       { a.showFind() }
func (a *App) OpenPalette()    { a.openPalette() }

// fakeMenuView paints a stand-in for the native popup menu under the Tools
// button. The real menu is an NSMenu that headless rendering cannot capture;
// this is used only by the snapshot test.
func (a *App) fakeMenuView(c *ui.Context, k tokens) {
	ui.Overlay(c, func() {
		m := ui.Column(c.Key("fakemenu")).Absolute().Top(44).Right(150).Width(260).Padding(5).Radius(radiusCtl).Background(Paper)
		boxed(m)
		hard(m, shadowLg)
		m.Children(func() {
			for i, l := range a.fakeMenu {
				if l == "-" {
					ui.Box(c.Key(fmt.Sprint("fm-sep", i))).Height(strokeW).FillWidth().Margin(5, 6, 5, 6).Background(Ink)
					continue
				}
				r := ui.Row(c.Key(fmt.Sprint("fm", i))).FillWidth().Height(26).Padding(0, 10, 0, 10).Radius(radiusKey).AlignItems(ui.Center)
				if i == 0 {
					r.Background(Accent)
				}
				r.Children(func() { ui.Text(c, l).FontSize(sizeButton).TextColor(Ink) })
			}
		})
	})
}

// pendingClipPill offers a newer clipboard that was not loaded because the
// 剪贴板 document has edits. Subtle and non-blocking; loads on click.
func (a *App) pendingClipPill(c *ui.Context, k tokens) {
	if !a.clipHasPending || a.cur != 0 {
		return
	}
	b := ui.ButtonBase(c.Key("clip-pending")).Height(28).Padding(0, 10).Radius(radiusPill).Gap(6).AlignItems(ui.Center).
		Label("剪贴板有新内容 · 载入").Tooltip("载入后仍可 ⌘Z 找回当前的编辑")
	bg := AccentSoft
	if b.Hovered() {
		bg = Accent
	}
	b.Background(bg).Border(strokeW, Ink)
	focusRing(b)
	b.Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(Ink)
		ui.Text(c, "剪贴板有新内容 · 载入").FontSize(11.5).FontWeight(700).TextColor(Ink)
	})
	if b.Clicked() {
		a.LoadPendingClipboard()
	}
}
