package gui

import (
	"fmt"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// document is one text the window edits: the clipboard workspace or a
// file. Peers: each keeps its own editor, step trail and last action.
type document struct {
	name, path string
	clip       bool
	text       string
	saved      string // text last read from or written to disk; files only
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

// docDirty reports whether document i differs from what was last read from
// or written to its file. The pinned 剪贴板 tab is not a file.
func (a *App) docDirty(i int) bool {
	if i < 0 || i >= len(a.docs) {
		return false
	}
	d := a.docs[i]
	if d.clip || d.path == "" {
		return false
	}
	text := d.text
	if i == a.cur {
		a.syncEditor()
		text = a.Text
	}
	return text != d.saved
}

// canCloseTab reports whether document i is a file tab ⌘W can close. The
// pinned 剪贴板 stays; a torn-off window's only document can close, and
// closing it closes that window.
func (a *App) canCloseTab(i int) bool {
	if i < 0 || i >= len(a.docs) || a.docs[i].clip {
		return false
	}
	if a.detached {
		return true
	}
	return i > 0
}

// CloseCurrentTab is ⌘W. A file tab closes; when it has unsaved edits a
// dialog asks first. The pinned 剪贴板 tab cannot close, so ⌘W hides the
// window instead, the same as its close button.
func (a *App) CloseCurrentTab() { a.requestClose(a.cur) }

func (a *App) requestClose(i int) {
	if a.closeAsk {
		return
	}
	a.syncEditor()
	if !a.canCloseTab(i) {
		if i == a.cur && i >= 0 && i < len(a.docs) && a.docs[i].clip {
			a.Platform.CloseWindow()
		}
		return
	}
	if a.docDirty(i) {
		a.closeAsk, a.closeOpen, a.closeIdx = true, true, i
		return
	}
	a.finishClose(i)
}

func (a *App) finishClose(i int) {
	a.closeAsk, a.closeOpen = false, false
	if i < 0 || i >= len(a.docs) || a.docs[i].clip {
		return
	}
	if a.detached {
		a.Platform.CloseWindow()
		return
	}
	a.closeDoc(i)
}

// unsavedDot is the mark on a file tab whose text is not what the file holds.
func (a *App) unsavedDot(c *ui.Context, i int) {
	if !a.docDirty(i) {
		return
	}
	ui.Box(c).Size(7, 7).Radius(4).Background(Ink).Tooltip("未存储的更改")
}

// closePromptView asks before a dirty tab is closed. Escape and the
// backdrop cancel; nothing is discarded unless 不存储 is chosen.
func (a *App) closePromptView(c *ui.Context) {
	if !a.closeAsk {
		return
	}
	if a.closeIdx < 0 || a.closeIdx >= len(a.docs) || !a.canCloseTab(a.closeIdx) {
		a.closeAsk, a.closeOpen = false, false
		return
	}
	idx := a.closeIdx
	name := a.docs[idx].name
	dlg := ui.DialogBase(c, &a.closeOpen, func(back, panel ui.Element) {
		back.Background(Ink.Alpha(0.35))
		hard(boxed(panel.Width(440).Padding(22).Radius(radiusCtl).Background(Ground)), shadowLg)
		panel.Label("未存储的更改")
		ui.Column(c).Width(396).Gap(12).Children(func() {
			ui.Text(c, "未存储的更改").FontSize(18).FontWeight(700).TextColor(Ink)
			ui.Text(c, "「"+name+"」有未存储的更改。关闭这个标签前要存储吗？").FontSize(14).TextColor(Ink)
			ui.Text(c, "不存储的话，这些更改会丢失。").FontSize(13).TextColor(Muted)
			ui.Row(c).Gap(8).Justify(ui.End).Padding(8, 0, 0, 0).Children(func() {
				if ghost(c, "close-discard", "不存储", "", false).Clicked() {
					a.finishClose(idx)
				}
				if ghost(c, "close-cancel", "取消", "", false).Clicked() {
					a.closeAsk, a.closeOpen = false, false
				}
				if primary(c, "close-save", "存储", "", "", false).AutoFocus().Clicked() {
					if a.writeDoc(idx) {
						a.finishClose(idx)
					} else {
						a.closeOpen = true
					}
				}
			})
		})
	})
	if dlg.Dismissed() {
		a.closeAsk = false
	}
}

// NewDetached is the interface of a window that holds one file document, as
// made by dragging its tab out of the main window. It shares the library
// (settings, action chains) with the main window. saved is what the file
// on disk was last known to hold.
func NewDetached(lib *engine.Library, p Platform, name, path, text, saved string) *App {
	a := New(lib, p)
	a.detached = true
	a.docs = []*document{{name: name, path: path, text: text, saved: saved}}
	a.cur = 0
	a.Text = text
	return a
}

// detachDoc moves file tab i into a window of its own, with its text as it
// is now (unsaved edits included).
func (a *App) detachDoc(i int) {
	if i <= 0 || i >= len(a.docs) {
		return // the clipboard workspace stays
	}
	a.syncEditor()
	d := a.docs[i]
	text := d.text
	if i == a.cur {
		text = a.Text
	}
	name, path, saved := d.name, d.path, d.saved
	a.closeDoc(i)
	a.Platform.OpenWindow(name, path, text, saved)
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
	if !d.clip && i > 0 {
		// A file tab can be torn off: drag it down out of the title bar, or
		// right-click it, to open it in a window of its own.
		b.Draggable(nil, ui.DraggableOptions{Snap: func(ui.Point) ui.Point { return ui.Point{} }})
		if b.Dragging() {
			a.tabDrag = b.DragOffset()
		}
		if _, _, ended := b.DragEnded(); ended {
			if off := a.tabDrag; off.Y > 36 || off.Y < -36 || off.X > 160 || off.X < -160 {
				a.detachDoc(i)
				return
			}
		}
		if b.RightClicked() {
			*a.menuFlag("tab" + d.path) = true
		}
		a.ctxMenu(c, "tab"+d.path, b, func(m *pm) {
			if m.Item("在新窗口中打开").Chosen() {
				a.detachDoc(i)
			}
			if m.Item("关闭标签").Chosen() {
				a.requestClose(i)
			}
		})
	}
	b.Children(func() {
		icon := "file-text"
		if d.clip {
			icon = "clipboard-paste"
		}
		ui.Icon(c, ic(icon)).FontSize(13).TextColor(Ink)
		ui.Text(c, d.name).FontSize(sizeButton).FontWeight(map[bool]int{true: 700, false: 500}[sel]).TextColor(Ink).SingleLine()
		a.unsavedDot(c, i)
		if !d.clip {
			x := ui.ButtonBase(c.Key("x")).Size(16, 16).Radius(radiusKey).Center().Label("关闭 " + d.name)
			if x.Hovered() {
				x.Background(Paper)
			}
			x.Children(func() { ui.Icon(c, ic("x")).FontSize(11).TextColor(Ink) })
			if x.Clicked() {
				a.requestClose(i)
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
	a.popMenu(c, "tools", b, func(m *pm) { a.toolsMenuItems(c, m) })
}

func (a *App) toolsMenuItems(c *ui.Context, m *pm) {
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
	if m.Item("存储").Chosen() {
		a.saveFile()
	}
	if m.Item("存储为…").Chosen() {
		a.saveFileAs()
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
func (a *App) OpenFileDialog()   { a.openFile() }
func (a *App) SaveFileDialog()   { a.saveFile() }
func (a *App) SaveFileAsDialog() { a.saveFileAs() }
func (a *App) ShowFind()         { a.showFind() }
func (a *App) OpenPalette()      { a.openPalette() }

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
