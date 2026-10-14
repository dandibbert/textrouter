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

func main() {
	lib := engine.LoadLibrary(engine.LibraryPath())
	d := &desktop{}
	a := gui.New(lib, d)
	quitting := false

	mygo.App.SetName("TextRouter")
	d.SetTheme(lib.Settings.Theme)
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })
	mygo.App.WhenReady(func() {
		d.win = mygo.NewWindow(mygo.WindowOptions{
			Title:     "TextRouter",
			Width:     1120,
			Height:    720,
			MinWidth:  820,
			MinHeight: 520,
			StateKey:  "main",
			// Inset title bar: the traffic lights float over the sidebar,
			// which shows the window's sidebar material (macOS).
			TitleBarStyle:  mygo.TitleBarHiddenInset,
			TitleBarHeight: 52,
			Vibrancy:       mygo.VibrancySidebar,
			Content:        ui.View(a.View),
		})
		// Open Clipboard Space with the current clipboard: from the Dock,
		// the menu bar icon or the global shortcut.
		d.show = func() {
			text := mygo.Clipboard.ReadText()
			d.win.Update(func() {
				a.LoadClipboard(text)
				a.Page = gui.PageClipboard
			})
			d.win.Show()
			d.win.Focus()
			mygo.App.Focus()
		}
		d.win.OnClose(func(e *mygo.CloseEvent) {
			if !quitting {
				e.PreventDefault()
				d.win.Hide()
			}
		})
		mygo.App.OnActivate(func(hasVisible bool) {
			if !hasVisible {
				d.show()
			}
		})
		goPage := func(p string) func(*mygo.MenuItem, *mygo.Window) {
			return func(*mygo.MenuItem, *mygo.Window) {
				d.show()
				d.win.Update(func() { a.Page = p })
			}
		}
		mygo.App.SetMenu(mygo.NewMenu([]*mygo.MenuItem{
			{Role: mygo.RoleAppMenu},
			{Role: mygo.RoleEditMenu},
			{Label: "前往", Submenu: []*mygo.MenuItem{
				{Label: "剪贴板空间", Accelerator: "CmdOrCtrl+1", Click: goPage(gui.PageClipboard)},
				{Label: "查找替换", Accelerator: "CmdOrCtrl+2", Click: goPage(gui.PageFind)},
				{Label: "批量替换", Accelerator: "CmdOrCtrl+3", Click: goPage(gui.PageBatch)},
				{Label: "动作链", Accelerator: "CmdOrCtrl+4", Click: goPage(gui.PageChains)},
				{Label: "清理器库", Accelerator: "CmdOrCtrl+5", Click: goPage(gui.PageCleaners)},
				{Label: "设置", Accelerator: "CmdOrCtrl+,", Click: goPage(gui.PageSettings)},
			}},
			{Role: mygo.RoleWindowMenu},
		}))
		tray, err := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), IconIsTemplate: true, ToolTip: "TextRouter — 打开剪贴板空间"})
		if err != nil {
			log.Println("tray:", err)
		} else {
			menu := mygo.NewMenu([]*mygo.MenuItem{
				{Label: "打开剪贴板空间", Click: func(*mygo.MenuItem, *mygo.Window) { d.show() }},
				{Label: "查找替换", Click: goPage(gui.PageFind)},
				{Label: "批量替换", Click: goPage(gui.PageBatch)},
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
