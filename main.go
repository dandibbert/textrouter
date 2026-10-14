// TextRouter is a TextSoap-style text cleaner: a Clipboard Workspace editor
// with interactive find/replace, 190+ cleaners and custom cleaners built
// from TextSoap's actions.
//
//	go tool mygo dev     # run with the Vite dev server
//	go tool mygo build   # build the app
package main

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log"
	"runtime"
	"slices"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo"
)

const appName = "TextRouter"

func main() {
	mygo.App.SetName(appName)
	app.lib = engine.LoadLibrary(engine.LibraryPath())

	mygo.Bind(Soap{})
	mygo.Bind(Document{})
	mygo.Bind(Batch{})
	mygo.BindAs("Library", LibraryService{})

	setTheme(app.prefs().Appearance)
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { quitting = true })
	mygo.App.OnOpenFile(func(path string) {
		if err := openFiles([]string{path}); err != nil {
			log.Println("open:", err)
		}
	})
	mygo.App.WhenReady(func() {
		mygo.App.SetMenu(buildMenu())
		p := app.prefs()
		applyPrefs(engine.Prefs{}, p)
		showWorkspace()
	})
	mygo.App.OnActivate(func(hasVisibleWindows bool) {
		if !hasVisibleWindows {
			showWorkspace()
		}
	})
	mygo.App.OnWindowAllClosed(func() {
		if app.prefs().QuitWhenLastClosed || (runtime.GOOS != "darwin" && tray == nil) {
			mygo.App.Quit()
		}
	})
	if err := mygo.App.Run(); err != nil {
		log.Fatal(err)
	}
}

// ---------------------------------------------------------------------------
// Menus

type click = func(*mygo.MenuItem, *mygo.Window)

// command sends a menu command to the window it was chosen in, bringing up
// the Clipboard Workspace when no window is focused.
func command(name string) click {
	return func(_ *mygo.MenuItem, w *mygo.Window) {
		if w == nil || w.IsDestroyed() {
			if d := focusedDoc(); d != nil {
				w = d.win
			}
		}
		if w == nil {
			showWorkspace()
			if d := workspaceDoc(); d != nil {
				w = d.win
			} else {
				return
			}
		}
		_ = Command.Emit(w, name)
	}
}

func run(fn func()) click { return func(*mygo.MenuItem, *mygo.Window) { fn() } }

// togglePref flips an editor preference from the View menu.
func togglePref(set func(p *engine.Prefs, v bool)) click {
	return func(it *mygo.MenuItem, _ *mygo.Window) {
		p := app.prefs()
		set(&p, it.Checked)
		go func() { _, _ = LibraryService{}.SetPrefs(p) }()
	}
}

var appMenu *mygo.Menu

func buildMenu() *mygo.Menu {
	p := app.prefs()
	appItems := []*mygo.MenuItem{
		{Label: "关于 " + appName, Role: mygo.RoleAbout},
		mygo.Separator(),
		{Label: "偏好设置…", Accelerator: "CmdOrCtrl+,", Click: run(func() { openSingle("prefs") })},
		mygo.Separator(),
	}
	if runtime.GOOS == "darwin" {
		appItems = append(appItems,
			&mygo.MenuItem{Role: mygo.RoleServices}, mygo.Separator(),
			&mygo.MenuItem{Label: "隐藏 " + appName, Role: mygo.RoleHide},
			&mygo.MenuItem{Label: "隐藏其他", Role: mygo.RoleHideOthers},
			&mygo.MenuItem{Label: "全部显示", Role: mygo.RoleUnhide}, mygo.Separator())
	} else {
		appItems = appItems[2:]
	}
	appItems = append(appItems, &mygo.MenuItem{Label: "退出 " + appName, Role: mygo.RoleQuit})

	appMenu = mygo.NewMenu([]*mygo.MenuItem{
		{Label: appName, Submenu: appItems},
		{Label: "文件", Submenu: []*mygo.MenuItem{
			{Label: "新建", Accelerator: "CmdOrCtrl+N", Click: run(func() { openDocument(KindUntitled, "", "") })},
			{Label: "剪贴板工作区", Accelerator: "CmdOrCtrl+Shift+N", Click: run(showWorkspace)},
			{Label: "打开…", Accelerator: "CmdOrCtrl+O", Click: command("open")},
			mygo.Separator(),
			{Label: "关闭", Accelerator: "CmdOrCtrl+W", Role: mygo.RoleClose},
			{Label: "存储…", Accelerator: "CmdOrCtrl+S", Click: command("save")},
			{Label: "另存为…", Accelerator: "CmdOrCtrl+Shift+S", Click: command("saveAs")},
			mygo.Separator(),
			{Label: "批量处理文件…", Accelerator: "CmdOrCtrl+Shift+B", Click: run(func() { openSingle("batch") })},
		}},
		{Label: "编辑", Submenu: []*mygo.MenuItem{
			{Label: "撤销", Role: mygo.RoleUndo},
			{Label: "重做", Role: mygo.RoleRedo},
			mygo.Separator(),
			{Label: "剪切", Role: mygo.RoleCut},
			{Label: "拷贝", Role: mygo.RoleCopy},
			{Label: "粘贴", Role: mygo.RolePaste},
			{Label: "全选", Role: mygo.RoleSelectAll},
			mygo.Separator(),
			{Label: "全部拷贝", Accelerator: "CmdOrCtrl+Shift+C", Click: command("copyAll")},
			{Label: "粘贴覆盖", Accelerator: "CmdOrCtrl+Shift+V", Click: command("pasteOver")},
			mygo.Separator(),
			{Label: "查找", Submenu: []*mygo.MenuItem{
				{Label: "查找与替换…", Accelerator: "CmdOrCtrl+F", Click: command("find")},
				{Label: "查找下一个", Accelerator: "CmdOrCtrl+G", Click: command("findNext")},
				{Label: "查找上一个", Accelerator: "CmdOrCtrl+Shift+G", Click: command("findPrevious")},
				{Label: "用所选内容查找", Accelerator: "CmdOrCtrl+E", Click: command("useSelection")},
				mygo.Separator(),
				{Label: "替换并查找下一个", Accelerator: "CmdOrCtrl+Alt+G", Click: command("replace")},
				{Label: "全部替换", Accelerator: "CmdOrCtrl+Alt+Shift+G", Click: command("replaceAll")},
			}},
		}},
		{Label: "清理", Submenu: []*mygo.MenuItem{
			{Label: "MyScrub 一键清理", Accelerator: "CmdOrCtrl+J", Click: command("myScrub")},
			{Label: "命令面板…", Accelerator: "CmdOrCtrl+K", Click: command("palette")},
			mygo.Separator(),
			{Label: "清理器列表", Accelerator: "CmdOrCtrl+1", Click: command("showCleaners")},
			{Label: "查找面板", Accelerator: "CmdOrCtrl+2", Click: command("find")},
			{Label: "搜索清理器", Accelerator: "CmdOrCtrl+Alt+F", Click: command("searchCleaners")},
			mygo.Separator(),
			{Label: "我的库…", Accelerator: "CmdOrCtrl+Shift+L", Click: run(func() { openLibrary("") })},
		}},
		{Label: "显示", Submenu: []*mygo.MenuItem{
			{Label: "显示/隐藏侧栏", Accelerator: "CmdOrCtrl+Alt+S", Click: command("toggleSidebar")},
			mygo.Separator(),
			{ID: "lineNumbers", Label: "行号", Type: mygo.MenuItemCheckbox, Checked: p.ShowLineNumbers,
				Click: togglePref(func(p *engine.Prefs, v bool) { p.ShowLineNumbers = v })},
			{ID: "wrap", Label: "自动换行", Type: mygo.MenuItemCheckbox, Checked: p.WrapText,
				Click: togglePref(func(p *engine.Prefs, v bool) { p.WrapText = v })},
			{ID: "invisibles", Label: "不可见字符", Type: mygo.MenuItemCheckbox, Checked: p.ShowInvisibles,
				Click: togglePref(func(p *engine.Prefs, v bool) { p.ShowInvisibles = v })},
			{ID: "stats", Label: "统计信息", Type: mygo.MenuItemCheckbox, Checked: p.ShowWordCount,
				Click: togglePref(func(p *engine.Prefs, v bool) { p.ShowWordCount = v })},
			mygo.Separator(),
			{Label: "放大", Accelerator: "CmdOrCtrl+=", Click: command("zoomIn")},
			{Label: "缩小", Accelerator: "CmdOrCtrl+-", Click: command("zoomOut")},
			{Label: "实际大小", Accelerator: "CmdOrCtrl+0", Click: command("zoomReset")},
			mygo.Separator(),
			{Label: "进入全屏", Role: mygo.RoleToggleFullScreen},
			{Label: "开发者工具", Role: mygo.RoleToggleDevTools},
		}},
		{Label: "窗口", Submenu: []*mygo.MenuItem{
			{Label: "最小化", Role: mygo.RoleMinimize},
			{Label: "缩放", Role: mygo.RoleZoom},
			mygo.Separator(),
			{Label: "剪贴板工作区", Click: run(showWorkspace)},
			{Label: "我的库", Click: run(func() { openLibrary("") })},
			{Label: "批量处理文件", Click: run(func() { openSingle("batch") })},
		}},
	})
	return appMenu
}

func syncViewMenu(p engine.Prefs) {
	if appMenu == nil {
		return
	}
	for id, v := range map[string]bool{"lineNumbers": p.ShowLineNumbers, "wrap": p.WrapText,
		"invisibles": p.ShowInvisibles, "stats": p.ShowWordCount} {
		if it := appMenu.ItemByID(id); it != nil && it.Checked != v {
			it.Checked = v
		}
	}
	mygo.App.SetMenu(appMenu)
}

// ---------------------------------------------------------------------------
// Preferences that reach the system

func setTheme(appearance string) {
	switch appearance {
	case "dark":
		mygo.Theme.SetSource(mygo.ThemeDark)
	case "light":
		mygo.Theme.SetSource(mygo.ThemeLight)
	default:
		mygo.Theme.SetSource(mygo.ThemeSystem)
	}
}

var hotkey string

func setHotkey(acc string) error {
	if hotkey != "" {
		mygo.GlobalShortcut.Unregister(hotkey)
		hotkey = ""
	}
	if acc == "" {
		return nil
	}
	if err := mygo.GlobalShortcut.Register(acc, showWorkspace); err != nil {
		return err
	}
	hotkey = acc
	return nil
}

func applyPrefs(old, p engine.Prefs) {
	mygo.RunOnMain(func() {
		if old.Appearance != p.Appearance {
			setTheme(p.Appearance)
		}
		if old.GlobalHotkey != p.GlobalHotkey {
			if err := setHotkey(p.GlobalHotkey); err != nil {
				log.Println("global shortcut:", err)
			}
		}
		if p.MenuBarIcon && tray == nil {
			createTray()
		} else if !p.MenuBarIcon && tray != nil {
			tray.Destroy()
			tray = nil
		} else if tray != nil && old.MyScrub != p.MyScrub {
			refreshTrayMenu()
		}
		syncViewMenu(p)
	})
}

// ---------------------------------------------------------------------------
// Menu bar icon (what textsoapAgent provides): open the workspace, or clean
// the clipboard in place with a cleaner.

var tray *mygo.Tray

func createTray() {
	t, err := mygo.NewTray(mygo.TrayOptions{Icon: trayIcon(), IconIsTemplate: true, ToolTip: appName + " — 打开剪贴板工作区"})
	if err != nil {
		log.Println("tray:", err)
		return
	}
	tray = t
	if runtime.GOOS == "linux" {
		refreshTrayMenu()
		return
	}
	t.OnClick(showWorkspace)
	t.OnRightClick(func() {
		t.SetMenu(trayMenu())
		defer t.SetMenu(nil)
		t.PopUpMenu()
	})
}

func refreshTrayMenu() {
	if tray != nil && runtime.GOOS == "linux" {
		tray.SetMenu(trayMenu())
	}
}

// cleanClipboard applies a cleaner to the clipboard text in place.
func cleanClipboard(name string) click {
	return func(*mygo.MenuItem, *mygo.Window) {
		go func() {
			text := mygo.Clipboard.ReadText()
			if r := (Soap{}).Apply(name, text); r.Changed {
				mygo.Clipboard.WriteText(r.Text)
			}
		}()
	}
}

func trayMenu() *mygo.Menu {
	var cleaners []*mygo.MenuItem
	var custom []string
	app.read(func(l *engine.Library) {
		for _, c := range l.Cleaners {
			custom = append(custom, c.Name)
		}
	})
	slices.Sort(custom)
	for _, n := range custom {
		cleaners = append(cleaners, &mygo.MenuItem{Label: n, Click: cleanClipboard(n)})
	}
	if len(cleaners) > 0 {
		cleaners = append(cleaners, mygo.Separator())
	}
	for _, n := range []string{"Scrub", "Remove Extra Spaces", "Make Paragraphs", "Remove Forwarding (>) Characters",
		"Straighten Quotes", "Trim Lines", "Uppercase", "Lowercase", "Capitalize Sentences"} {
		if c := engine.CleanerByName(n); c != nil {
			cleaners = append(cleaners, &mygo.MenuItem{Label: c.ZhName, Click: cleanClipboard(n)})
		}
	}
	return mygo.NewMenu([]*mygo.MenuItem{
		{Label: "打开剪贴板工作区", Click: run(showWorkspace)},
		{Label: "MyScrub 清理剪贴板", Click: cleanClipboard("#MYSCRUB")},
		{Label: "清理剪贴板", Submenu: cleaners},
		mygo.Separator(),
		{Label: "新建文档", Click: run(func() { openDocument(KindUntitled, "", "") })},
		{Label: "我的库…", Click: run(func() { openLibrary("") })},
		{Label: "批量处理文件…", Click: run(func() { openSingle("batch") })},
		{Label: "偏好设置…", Click: run(func() { openSingle("prefs") })},
		mygo.Separator(),
		{Label: "退出 " + appName, Role: mygo.RoleQuit},
	})
}

// trayIcon draws a template glyph: a soap bar with a sparkle.
func trayIcon() []byte {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	ink := color.NRGBA{A: 255}
	if runtime.GOOS != "darwin" {
		ink = color.NRGBA{R: 90, G: 110, B: 255, A: 255}
	}
	fill := func(r image.Rectangle) { draw.Draw(img, r, image.NewUniform(ink), image.Point{}, draw.Src) }
	// Rounded bar outline.
	fill(image.Rect(5, 12, 27, 14))
	fill(image.Rect(5, 26, 27, 28))
	fill(image.Rect(3, 14, 5, 26))
	fill(image.Rect(27, 14, 29, 26))
	// Lines of text on the bar.
	fill(image.Rect(8, 17, 22, 19))
	fill(image.Rect(8, 21, 18, 23))
	// Sparkle.
	fill(image.Rect(22, 2, 24, 10))
	fill(image.Rect(19, 5, 27, 7))
	var b bytes.Buffer
	_ = png.Encode(&b, img)
	return b.Bytes()
}
