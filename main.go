// TextRouter: a TextSoap-style text cleaner with a Clipboard Space,
// readable find/replace, batch rule tables and action chains.
//
//	go tool mygo dev     # run with live reload
//	go tool mygo build -platform darwin/arm64
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"runtime"
	"time"

	"github.com/dandibbert/textrouter/engine"
	"github.com/dandibbert/textrouter/gui"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
)

// desktop implements gui.Platform with MyGo's desktop APIs.
type desktop struct {
	win    *mygo.Window
	hotkey string
	show   func()
	lib    *engine.Library
	parent *desktop // the main window's, for the windows made by tearing off a tab
}

func (d *desktop) OpenFiles(title string, exts []string, multiple, dirs bool) []string {
	opts := mygo.OpenDialogOptions{Parent: d.win, Title: title, Multiple: multiple, Directory: dirs}
	if len(exts) > 0 {
		opts.Filters = []mygo.FileFilter{{Name: "Files", Extensions: exts}}
	}
	paths, err := mygo.Dialog.Open(opts)
	if err != nil {
		log.Println("open dialog:", err)
	}
	return paths
}

func (d *desktop) SaveFile(title, name string) string {
	p, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: d.win, Title: title, DefaultPath: name})
	if err != nil {
		log.Println("save dialog:", err)
	}
	return p
}

// HideWindow hides the window; on macOS the whole app hides so the
// previous app comes back to the front, ready for ⌘V.
func (d *desktop) HideWindow() {
	if d.parent != nil {
		d.win.Close()
		return
	}
	if runtime.GOOS == "darwin" {
		mygo.App.Hide()
		return
	}
	d.win.Hide()
}

func (d *desktop) SetTheme(theme string) {
	switch theme {
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}

func (d *desktop) SetHotkey(acc string) error {
	if d.parent != nil {
		return d.parent.SetHotkey(acc)
	}
	if d.hotkey != "" {
		mygo.GlobalShortcut.Unregister(d.hotkey)
	}
	if acc == "" {
		d.hotkey = ""
		return nil
	}
	if err := mygo.GlobalShortcut.Register(acc, func() { d.show() }); err != nil {
		if d.hotkey != "" {
			_ = mygo.GlobalShortcut.Register(d.hotkey, func() { d.show() })
		}
		return err
	}
	d.hotkey = acc
	return nil
}

func (d *desktop) Update(fn func()) { d.win.Update(fn) }

// OpenWindow opens a file document in a window of its own, a little below
// and to the right of this one.
func (d *desktop) OpenWindow(name, path, text string) {
	root := d
	if d.parent != nil {
		root = d.parent
	}
	child := &desktop{parent: root, lib: root.lib}
	ca := gui.NewDetached(root.lib, child, name, path, text)
	x, y := d.win.Position()
	w, h := d.win.Size()
	child.win = mygo.NewWindow(mygo.WindowOptions{
		Title:          name,
		Width:          w,
		Height:         h,
		X:              x + 36,
		Y:              y + 36,
		MinWidth:       820,
		MinHeight:      520,
		TitleBarStyle:  mygo.TitleBarHiddenInset,
		TitleBarHeight: 42,
		Vibrancy:       mygo.VibrancySidebar,
		Content:        ui.View(ca.View),
	})
	child.win.Show()
	child.win.Focus()
}

// SetAppIcon changes the Dock icon; the .app bundle keeps its own.
func (d *desktop) SetAppIcon(png []byte) {
	if err := mygo.App.Dock.SetIcon(png); err != nil {
		log.Println("dock icon:", err)
	}
}

func main() {
	lib := engine.LoadLibrary(engine.LibraryPath())
	d := &desktop{lib: lib}
	a := gui.New(lib, d)
	quitting := false

	// Files dropped on the Dock icon, "Open With" and the command line: held
	// until the window exists, then opened as document tabs.
	var pendingFiles []string
	mygo.App.OnOpenFile(func(path string) {
		if d.win == nil {
			pendingFiles = append(pendingFiles, path)
			return
		}
		d.win.Update(func() { a.OpenPath(path) })
		d.win.Show()
	})

	mygo.App.SetName("TextRouter")
	d.SetTheme(lib.Settings.Theme)
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })
	mygo.App.WhenReady(func() {
		d.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "TextRouter",
			Width:     lib.Settings.View.WindowW,
			Height:    lib.Settings.View.WindowH,
			MinWidth:  820,
			MinHeight: 520,
			StateKey:  "main",
			// Inset title bar: the traffic lights float over the sidebar,
			// which shows the window's sidebar material (macOS).
			TitleBarStyle:  mygo.TitleBarHiddenInset,
			TitleBarHeight: 42,
			Vibrancy:       mygo.VibrancySidebar,
			Content:        ui.View(a.View),
		})
		for _, p := range pendingFiles {
			p := p
			d.win.Update(func() { a.OpenPath(p) })
		}
		pendingFiles = nil
		if id := gui.SavedAppIcon(lib.Path()); id != gui.DefaultAppIcon {
			d.SetAppIcon(gui.AppIconPNG(id))
		}
		// Open Clipboard Space with the current clipboard: from the Dock,
		// the menu bar icon or the global shortcut.
		appActive := false
		d.show = func() {
			text := mygo.Clipboard.ReadText()
			d.win.Update(func() { a.LoadClipboard(text) })
			if !d.win.IsVisible() {
				d.win.Show()
			}
			d.win.Focus()
			if !appActive {
				mygo.App.Focus() // didBecomeActive follows; it finds nothing new
			}
		}
		// Remember the window size in the settings, once resizing settles.
		var resizeTimer *time.Timer
		d.win.OnResize(func() {
			w, h := d.win.Size()
			if resizeTimer != nil {
				resizeTimer.Stop()
			}
			resizeTimer = time.AfterFunc(400*time.Millisecond, func() {
				d.win.Update(func() { a.SetWindowSize(w, h) })
			})
		})
		d.win.OnClose(func(e *mygo.CloseEvent) {
			if !quitting {
				e.PreventDefault()
				d.win.Hide()
			}
		})
		// Every activation reads the clipboard into the 剪贴板 document:
		//  - Dock icon click: applicationShouldHandleReopen:hasVisibleWindows:
		//    → OnActivate, whether or not the window is visible (before, a
		//    visible window ignored the click);
		//  - switching back to the app (Dock click while inactive, ⌘Tab,
		//    clicking its window): applicationDidBecomeActive: →
		//    OnDidBecomeActive;
		//  - menu bar icon, global shortcut and launch call d.show directly.
		// A Dock click on an inactive app fires both; the second finds the
		// clipboard unchanged and leaves the document alone.
		mygo.App.OnActivate(func(bool) { d.show() })
		mygo.App.OnDidBecomeActive(func() { appActive = true; d.show() })
		mygo.App.OnDidResignActive(func() { appActive = false })
		goPage := func(p string) func(*mygo.MenuItem, *mygo.Window) {
			return func(*mygo.MenuItem, *mygo.Window) {
				d.show()
				d.win.Update(func() { a.Page = p })
			}
		}
		run := func(fn func()) func(*mygo.MenuItem, *mygo.Window) {
			return func(*mygo.MenuItem, *mygo.Window) { d.win.Update(fn) }
		}
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Role: mygo.RoleEditMenu},
			{Label: "文件", Submenu: []*mygo.MenuItem{
				{Label: "剪贴板", Accelerator: "CmdOrCtrl+1", Click: run(a.ShowClipboard)},
				{Label: "打开…", Accelerator: "CmdOrCtrl+O", Click: run(a.OpenFileDialog)},
				{Label: "存储为…", Accelerator: "CmdOrCtrl+S", Click: run(a.SaveFileDialog)},
				mygo.Separator(),
				{Role: mygo.RoleClose, Label: "关闭窗口"},
			}},
			{Label: "查找", Submenu: []*mygo.MenuItem{
				{Label: "查找替换", Accelerator: "CmdOrCtrl+F", Click: run(a.ShowFind)},
			}},
			{Label: "工具", Submenu: []*mygo.MenuItem{
				{Label: "批量替换…", Accelerator: "CmdOrCtrl+Shift+R", Click: goPage(gui.PageBatch)},
				{Label: "我的清理器（动作链）…", Accelerator: "CmdOrCtrl+Shift+L", Click: goPage(gui.PageChains)},
				{Label: "清理器库…", Click: goPage(gui.PageCleaners)},
				{Label: "批量处理文件…", Click: goPage(gui.PageFiles)},
				mygo.Separator(),
				{Label: "全部动作…", Accelerator: "CmdOrCtrl+K", Click: run(a.OpenPalette)},
				{Label: "设置…", Accelerator: "CmdOrCtrl+,", Click: goPage(gui.PageSettings)},
			}},
			{Role: mygo.RoleWindowMenu},
		}))
		tray, err := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), IconIsTemplate: true, ToolTip: "TextRouter — 打开剪贴板"})
		if err != nil {
			log.Println("tray:", err)
		} else {
			menu := mygo.NewMenu([]*mygo.MenuItem{
				{Label: "打开剪贴板", Click: func(*mygo.MenuItem, *mygo.Window) { d.show() }},
				{Label: "批量替换…", Click: goPage(gui.PageBatch)},
				mygo.Separator(),
				{Role: mygo.RoleQuit},
			})
			if runtime.GOOS == "linux" {
				tray.SetMenu(menu)
			} else {
				tray.OnClick(d.show)
				tray.OnRightClick(func() {
					tray.SetMenu(menu)
					defer tray.SetMenu(nil)
					tray.PopUpMenu()
				})
			}
			mygo.App.OnWillQuit(func(*mygo.QuitEvent) { tray.Destroy() })
		}
		if err := d.SetHotkey(lib.Settings.Hotkey); err != nil {
			log.Println("global shortcut:", err)
		}
		d.show()
	})
	mygo.App.OnWindowAllClosed(func() {}) // keep running in the menu bar
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// trayIcon draws a template "T⇄" glyph: black on transparent for macOS.
func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	ink := color.NRGBA{A: 255}
	if runtime.GOOS != "darwin" {
		ink = color.NRGBA{R: 124, G: 92, B: 255, A: 255}
	}
	for _, r := range []image.Rectangle{
		image.Rect(3, 4, 19, 8), image.Rect(9, 8, 13, 28), // T
		image.Rect(17, 14, 29, 17), image.Rect(25, 11, 28, 20), // →
		image.Rect(17, 22, 29, 25), image.Rect(18, 19, 21, 28), // ←
	} {
		draw.Draw(img, r, image.NewUniform(ink), image.Point{}, draw.Src)
	}
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}
