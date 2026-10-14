package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
	"github.com/egoist/mygo/ui/editor"
)

// editorPrefs are the Clipboard Space editor's view options.
type editorPrefs struct {
	lineNumbers bool
	wrap        bool
	invisibles  bool
	mono        bool
	fontSize    float32
}

// ---------------------------------------------------------------- text sync

// syncEditor keeps App.Text and the editor buffer equal: edits typed in
// the editor flow into Text, and Text set from elsewhere (tests, pages)
// flows into the buffer as an undoable edit.
func (a *App) syncEditor() {
	if a.ed == nil {
		a.ed = &editor.State{Buffer: editor.NewBuffer(a.Text)}
		a.edVer, a.edSynced = a.ed.Buffer.Version(), a.Text
		return
	}
	buf := a.ed.Buffer
	if v := buf.Version(); v != a.edVer {
		a.Text = buf.String()
		a.edVer, a.edSynced = v, a.Text
		return
	}
	if a.Text != a.edSynced {
		a.replaceAll(a.Text)
	}
}

// replaceAll sets the whole text as one minimal edit, so carets outside
// the changed span stay where they are.
func (a *App) replaceAll(s string) {
	if a.ed == nil {
		a.Text = s
		a.syncEditor()
		return
	}
	buf := a.ed.Buffer
	old := buf.String()
	if old != s {
		p := commonPrefix(old, s)
		q := commonSuffix(old[p:], s[p:])
		buf.Edit(editor.Edit{Start: p, End: len(old) - q, Text: s[p : len(s)-q]})
	}
	a.Text = buf.String()
	a.edVer, a.edSynced = buf.Version(), a.Text
}

func commonPrefix(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[n] == b[n] {
		n++
	}
	for n > 0 && n < len(a) && !utf8.RuneStart(a[n]) {
		n--
	}
	return n
}

func commonSuffix(a, b string) int {
	n := 0
	for n < len(a) && n < len(b) && a[len(a)-1-n] == b[len(b)-1-n] {
		n++
	}
	for n > 0 && n < len(a) && !utf8.RuneStart(a[len(a)-n]) {
		n--
	}
	return n
}

// selections returns the editor's non-empty selections in order.
func (a *App) selections() []editor.Selection {
	if a.ed == nil {
		return nil
	}
	var out []editor.Selection
	for _, s := range a.ed.Selections() {
		if s.Start() != s.End() {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start() < out[j].Start() })
	return out
}

// Scope tells what an action applies to: "选区" or "全文".
func (a *App) Scope() string {
	if len(a.selections()) > 0 {
		return "选区"
	}
	return "全文"
}

// applyScoped runs fn on each selection (one undo step) or on the whole
// text, and returns the scope it used.
func (a *App) applyScoped(fn func(string) string) string {
	sels := a.selections()
	if len(sels) == 0 {
		a.replaceAll(fn(a.Text))
		return "全文"
	}
	buf := a.ed.Buffer
	var edits []editor.Edit
	var next []editor.Selection
	delta := 0
	for _, s := range sels {
		in := buf.Slice(s.Start(), s.End())
		out := fn(in)
		edits = append(edits, editor.Edit{Start: s.Start(), End: s.End(), Text: out})
		next = append(next, editor.Selection{Anchor: s.Start() + delta, Head: s.Start() + delta + len(out)})
		delta += len(out) - (s.End() - s.Start())
	}
	buf.Transact(func() {
		for i := len(edits) - 1; i >= 0; i-- {
			buf.Edit(edits[i])
		}
	})
	a.ed.SetSelections(next...)
	a.Text = buf.String()
	a.edVer, a.edSynced = buf.Version(), a.Text
	return "选区"
}

// scopeText is the text actions would apply to: the selections (joined
// by newlines, for previews) or the whole text.
func (a *App) scopeText() string {
	sels := a.selections()
	if len(sels) == 0 {
		return a.Text
	}
	parts := make([]string, len(sels))
	for i, s := range sels {
		parts[i] = a.ed.Buffer.Slice(s.Start(), s.End())
	}
	return strings.Join(parts, "\n")
}

func (a *App) clearDiffBase() {
	if a.ed != nil {
		a.ed.DiffBase = nil
	}
}

// RunEditorCommand runs an editor command (line operations, case, find…).
func (a *App) RunEditorCommand(cmd editor.Command) {
	a.syncEditor()
	a.ed.Run(cmd)
	a.syncEditor()
}

// ---------------------------------------------------------------- view

// editorTheme matches the editor to the app tokens.
func editorTheme(c *ui.Context, k tokens) *editor.Theme {
	th := editor.DefaultTheme(c.Theme())
	th.Background = k.canvas
	th.Text = k.text
	th.LineNumber = k.faint.Alpha(0.7)
	th.ActiveLineNumber = k.muted
	if k.dark {
		th.ActiveLine = ui.RGBA(255, 255, 255, 0.035)
	} else {
		th.ActiveLine = ui.RGBA(0, 0, 0, 0.028)
	}
	th.Selection = k.accent.Alpha(0.22)
	th.InactiveSelection = k.accent.Alpha(0.12)
	th.Cursor = k.accent
	th.Match = ui.RGBA(250, 204, 21, map[bool]float32{true: 0.28, false: 0.40}[k.dark])
	th.CurrentMatch = ui.RGBA(249, 115, 22, map[bool]float32{true: 0.55, false: 0.55}[k.dark])
	th.Whitespace = k.faint.Alpha(0.75)
	th.Added = k.green.Alpha(0.75)
	th.Modified = k.accent.Alpha(0.75)
	th.Deleted = k.red.Alpha(0.75)
	return th
}

// configureEditor sets the editor's options from the prefs, each frame.
func (a *App) configureEditor(c *ui.Context, k tokens) {
	st := a.ed
	v := &a.view
	if v.fontSize == 0 {
		v.fontSize = 15
	}
	family, size := editorFont, v.fontSize
	if v.mono {
		family, size = monoFont, v.fontSize-1.5
	}
	st.Font = ui.Font{Family: family, Size: size}
	st.LineHeight = 1.6
	st.SoftWrap = v.wrap
	st.LineNumbers = editor.LineNumbersOff
	if v.lineNumbers {
		st.LineNumbers = editor.LineNumbersOn
	}
	st.Whitespace = editor.WhitespaceSelection
	if v.invisibles {
		st.Whitespace = editor.WhitespaceAll
	}
	st.NoIndentGuides = true
	st.NoFolding = true
	st.TabSize = 4
	st.Placeholder = "剪贴板是空的。粘贴、输入，或把文本文件拖进来…"
	st.Label = "剪贴板文本"
	st.Theme = editorTheme(c, k)
	st.Bindings = []editor.Binding{
		{Mods: ui.Cmd, Key: ui.KeyK, Command: editor.NoCommand},
		{Mods: ui.Cmd, Key: ui.KeyEnter, Command: editor.NoCommand},
		{Mods: ui.Cmd, Key: ui.KeyBackslash, Command: editor.NoCommand},
		{Mods: ui.Cmd, Key: ui.KeyEqual, Command: editor.NoCommand},
		{Mods: ui.Cmd, Key: ui.KeyMinus, Command: editor.NoCommand},
		{Mods: ui.Cmd, Key: ui.Key0, Command: editor.NoCommand},
		{Mods: ui.Cmd, Key: ui.KeyO, Command: editor.NoCommand},
		{Mods: ui.Cmd, Key: ui.KeyS, Command: editor.NoCommand},
		{Mods: ui.Cmd, Key: ui.KeyL, Command: editor.GoToLine},
	}
	st.RenderFindBar = a.renderFindBar
	st.RenderGoToLine = a.renderGoToLine
}

// updateInvisibles refreshes the invisible-character marks after the
// editor has attached its buffer.
func (a *App) updateInvisibles() {
	st := a.ed
	if a.view.invisibles {
		if ver := st.Buffer.Version(); !a.invisOn || ver != a.invisVer {
			st.SetInlays(invisibleInlays(st.Buffer.String()))
			a.invisVer, a.invisOn = ver, true
		}
	} else if a.invisOn {
		st.SetInlays(nil)
		a.invisOn = false
	}
}

// invisibleInlays marks line breaks, NBSP, full-width spaces and other
// invisible characters with visible glyphs (shown, never inserted).
func invisibleInlays(s string) []editor.Inlay {
	var out []editor.Inlay
	if len(s) > 2_000_000 {
		return nil
	}
	for i, r := range s {
		switch r {
		case '\n':
			out = append(out, editor.Inlay{Offset: i, Text: "¬"})
		case '\u00a0':
			out = append(out, editor.Inlay{Offset: i, Text: "⍽"})
		case '\u3000':
			out = append(out, editor.Inlay{Offset: i, Text: "□"})
		case '\u200b', '\u200c', '\u200d', '\ufeff':
			out = append(out, editor.Inlay{Offset: i, Text: "‸"})
		case '\r':
			out = append(out, editor.Inlay{Offset: i, Text: "␍"})
		}
	}
	return out
}

// clipboardView is the main screen: an edge-to-edge editor with one slim
// top bar and the status bar.
func (a *App) clipboardView(c *ui.Context) {
	k := tok(c)
	a.syncEditor()
	a.configureEditor(c, k)
	c.OnShortcut(ui.Cmd, ui.KeyEnter, a.done)
	c.OnShortcut(ui.Cmd, ui.KeyEqual, func() { a.view.fontSize = min(a.view.fontSize+1, 32) })
	c.OnShortcut(ui.Cmd, ui.KeyMinus, func() { a.view.fontSize = max(a.view.fontSize-1, 10) })
	c.OnShortcut(ui.Cmd, ui.Key0, func() { a.view.fontSize = 15 })
	c.OnShortcut(ui.Cmd, ui.KeyO, a.openFile)
	c.OnShortcut(ui.Cmd, ui.KeyS, a.saveFile)
	c.OnShortcut(ui.Cmd|ui.Alt, ui.KeyI, func() { a.view.invisibles = !a.view.invisibles })

	a.editorBar(c, k)
	zone := ui.Column(c).Grow(1).MinHeight(0).Background(k.canvas)
	if files := zone.DroppedFiles(); len(files) > 0 {
		a.loadFile(files[0])
	}
	zone.Children(func() {
		ui.Row(c).Grow(1).MinHeight(0).Padding(sp2, 0, 0, sp2).Children(func() {
			editor.View(c, a.ed).Grow(1).FillHeight().AutoFocus()
		})
	})
	a.updateInvisibles()
	a.syncEditor()
}

// editorBar is the one slim, quiet bar above the editor.
func (a *App) editorBar(c *ui.Context, k tokens) {
	bar := c.TitleBar()
	left := float32(sp3)
	if !a.sideOpen {
		left = max(bar.Left, 0) + sp3
		if bar.Left == 0 && bar.Height == 0 {
			left = sp3
		}
	}
	ui.Row(c).Height(42).Padding(0, sp3, 0, left).Gap(2).AlignItems(ui.Center).Background(k.canvas).BorderWidth(0, 0, 1, 0).BorderColor(k.hair).DragWindow().Children(func() {
		a.sidebarToggle(c)
		ui.Box(c).Width(sp2)
		// Palette launcher: quiet, text-like.
		l := ui.ButtonBase(c.Key("apply")).Height(28).Width(260).Padding(0, 8, 0, 10).Radius(rMd).Gap(8).AlignItems(ui.Center).Label("应用动作").Tooltip("搜索并应用清理器、动作链、规则和编辑命令")
		pressable(c, l, k, k.field)
		l.Children(func() {
			ui.Icon(c, ic("search")).FontSize(13).TextColor(k.faint)
			ui.Text(c, "应用动作…").FontSize(12.5).TextColor(k.faint).Grow(1).SingleLine()
			ui.Text(c, "⌘K").FontSize(11.5).TextColor(k.faint)
		})
		if l.Clicked() {
			a.openPalette()
		}
		ui.Box(c).Width(sp2)
		a.scopePill(c, k)
		ui.Spacer(c)
		toggle := func(key, icon, label, tip string, on bool) bool {
			b := iconButton(c, key, icon, label, tip).Size(28, 28)
			if on {
				b.Background(k.accentSoft)
			}
			return b.Clicked()
		}
		if toggle("find", "text-search", "查找", "查找 ⌘F · 替换 ⌥⌘F", false) {
			a.ed.Run(editor.Find)
		}
		if toggle("invis", "pilcrow", "显示不可见字符", "显示空格、制表符、换行、NBSP、全角空格 ⌥⌘I", a.view.invisibles) {
			a.view.invisibles = !a.view.invisibles
		}
		if toggle("lnum", "list-ordered", "行号", "显示行号", a.view.lineNumbers) {
			a.view.lineNumbers = !a.view.lineNumbers
		}
		if toggle("wrap", "wrap-text", "自动换行", "自动换行", a.view.wrap) {
			a.view.wrap = !a.view.wrap
		}
		if toggle("mono", "type", "等宽字体", "等宽 / 比例字体", a.view.mono) {
			a.view.mono = !a.view.mono
		}
		ui.Box(c).Size(1, 16).Background(k.hair).Margin(0, sp2)
		u := iconButton(c, "undo", "undo-2", "撤销", "撤销 ⌘Z（包括动作）").Size(28, 28)
		if !a.ed.Buffer.CanUndo() {
			u.Opacity(0.35)
		}
		if u.Clicked() {
			a.Undo()
		}
		ui.Box(c).Width(sp1)
		// Copy & close: quiet text button with an accent label.
		d := ui.ButtonBase(c.Key("done")).Height(28).Padding(0, 10).Radius(rMd).Gap(6).AlignItems(ui.Center).Label("复制并完成").Tooltip("复制回剪贴板并收起窗口")
		pressable(c, d, k, ui.Transparent)
		d.Children(func() {
			ui.Icon(c, ic("copy")).FontSize(13).TextColor(k.accent)
			ui.Text(c, "复制并完成").FontSize(12.5).FontWeight(600).TextColor(k.accent).SingleLine()
			ui.Text(c, "⌘↩").FontSize(11.5).TextColor(k.accent.Alpha(0.6))
		})
		if d.Clicked() {
			a.done()
		}
	})
}

// scopePill shows what the next action will apply to.
func (a *App) scopePill(c *ui.Context, k tokens) {
	sels := a.selections()
	if len(sels) == 0 {
		ui.Text(c, "作用于全文").FontSize(11.5).TextColor(k.faint).SingleLine().Tooltip("选中文字后，动作只作用于选区")
		return
	}
	n := 0
	for _, s := range sels {
		n += utf8.RuneCountInString(a.ed.Buffer.Slice(s.Start(), s.End()))
	}
	label := fmt.Sprintf("作用于选区 · %d 字", n)
	if len(sels) > 1 {
		label = fmt.Sprintf("作用于 %d 个选区 · %d 字", len(sels), n)
	}
	ui.Row(c).Gap(5).Height(22).Padding(0, 8).Radius(999).Background(k.accentSoft).AlignItems(ui.Center).Children(func() {
		ui.Box(c).Size(6, 6).Radius(3).Background(k.accent)
		ui.Text(c, label).FontSize(11.5).FontWeight(600).TextColor(k.accent).SingleLine()
	}).Tooltip("动作、查找替换和批量替换只作用于选中的文字")
}

func (a *App) sidebarToggle(c *ui.Context) {
	icon := "chevron-right"
	if a.sideOpen {
		icon = "chevron-down"
	}
	_ = icon
	if iconButton(c, "side", "panel-left", "侧边栏", "显示/隐藏侧边栏 ⌘\\").Size(28, 28).Clicked() {
		a.sideOpen = !a.sideOpen
	}
}

// renderFindBar draws the editor's find/replace bar as a floating card.
func (a *App) renderFindBar(c *ui.Context, bar ui.Element, f *editor.FindBar) {
	k := tok(c)
	bar.Column().Gap(6).Padding(8).Radius(rLg).Background(k.canvas).Border(1, k.hairStrong).
		Shadow(0, 10, 30, -8, ui.RGBA(0, 0, 0, map[bool]float32{true: 0.55, false: 0.18}[k.dark])).Shadow(0, 1, 3, 0, k.shadowNear).Width(440)
	in := func(e ui.Element) ui.Element {
		return e.Grow(1).FontSize(13).Font(monoFont).TextColor(k.text).Padding(5, 8).Radius(rSm).Background(k.field)
	}
	fl := func(key, text, tip string, on *bool) {
		b := ui.ButtonBase(c.Key(key)).Height(24).MinWidth(26).Padding(0, 6).Radius(5).Center().Label(tip).Tooltip(tip).KeepFocus()
		bg, col := ui.Transparent, k.faint
		if *on {
			bg, col = k.accentSoft, k.accent
		} else if b.Hovered() {
			bg = k.hover
		}
		b.Background(bg).Children(func() { ui.Text(c, text).FontSize(11.5).FontWeight(600).Font(monoFont).TextColor(col) })
		if b.Clicked() {
			*on = !*on
		}
	}
	nav := func(key, icon, tip string) bool {
		b := iconButton(c, key, icon, tip, tip).Size(26, 26).KeepFocus()
		return b.Clicked()
	}
	bar.Children(func() {
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			f.QueryInput(in(ui.TextInputBase(c, f.Query()).Placeholder("查找").Label("查找")))
			fl("fcase", "Aa", "区分大小写", f.CaseSensitive())
			fl("fword", "ab", "全词匹配", f.WholeWord())
			fl("fre", ".*", "正则表达式", f.Regexp())
			fl("fsel", "⌶", "只在选区中查找", f.InSelection())
		})
		ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
			st := f.Status()
			col := k.muted
			if f.Err() != nil {
				col = k.red
			}
			ui.Text(c, st).FontSize(11.5).TextColor(col).FontFeatures("tnum").Grow(1).SingleLine()
			if nav("fprev", "arrow-up", "上一个 ⇧↩") {
				f.Previous()
			}
			if nav("fnext", "arrow-down", "下一个 ↩") {
				f.Next()
			}
			if nav("fall", "check", "全部选中 ⌥↩") {
				f.SelectAll()
			}
			r := "replace-all"
			if nav("frep", r, "替换 ⌥⌘F") {
				f.SetReplacing(!f.Replacing())
			}
			if nav("fclose", "x", "关闭 esc") {
				f.Close()
			}
		})
		if f.Replacing() {
			ui.Row(c).Gap(4).AlignItems(ui.Center).Children(func() {
				f.ReplacementInput(in(ui.TextInputBase(c, f.Replacement()).Placeholder("替换为").Label("替换为")))
				if ghost(c, "rone", "替换", "", false).KeepFocus().Clicked() {
					f.Replace()
				}
				if ghost(c, "rall", "全部", "", false).KeepFocus().Clicked() {
					f.ReplaceAll()
				}
			})
		}
	})
}

func (a *App) renderGoToLine(c *ui.Context, bar ui.Element, g *editor.GoToLineBar) {
	k := tok(c)
	bar.Row().Gap(8).Padding(8).Radius(rLg).Background(k.canvas).Border(1, k.hairStrong).AlignItems(ui.Center).
		Shadow(0, 10, 30, -8, ui.RGBA(0, 0, 0, map[bool]float32{true: 0.55, false: 0.18}[k.dark])).Width(320)
	bar.Children(func() {
		ui.Icon(c, ic("corner-down-left")).FontSize(13).TextColor(k.faint)
		g.Input(ui.TextInputBase(c, g.Text()).Placeholder(fmt.Sprintf("转到行（1–%d）", g.LineCount())).Label("转到行").Grow(1).FontSize(13).TextColor(k.text).Padding(5, 8).Radius(rSm).Background(k.field))
	})
}

// ---------------------------------------------------------------- files

func (a *App) loadFile(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		a.status = "打开失败：" + err.Error()
		return
	}
	if !utf8.Valid(data) {
		a.status = "不是 UTF-8 文本文件：" + filepath.Base(path)
		return
	}
	a.syncEditor()
	a.replaceAll(string(data))
	a.filePath = path
	a.last = nil
	a.clearDiffBase()
	a.status = "已打开 " + filepath.Base(path)
}

func (a *App) openFile() {
	go func() {
		paths := a.Platform.OpenFiles("打开文本文件", nil, false, false)
		if len(paths) == 0 {
			return
		}
		a.Platform.Update(func() { a.loadFile(paths[0]) })
	}()
}

func (a *App) saveFile() {
	text := a.Text
	name := "文本.txt"
	if a.filePath != "" {
		name = filepath.Base(a.filePath)
	}
	go func() {
		p := a.Platform.SaveFile("存储文本", name)
		if p == "" {
			return
		}
		err := os.WriteFile(p, []byte(text), 0o644)
		a.Platform.Update(func() {
			if err != nil {
				a.status = "存储失败：" + err.Error()
				return
			}
			a.filePath = p
			a.status = "已存储到 " + p
		})
	}()
}

// ---------------------------------------------------------------- commands

// editorCommand is an editor command offered in the palette.
type editorCommand struct {
	name, title, keys string
	cmd               editor.Command
}

var editorCommands = []editorCommand{
	{"find", "查找", "⌘F", editor.Find},
	{"replace", "查找并替换", "⌥⌘F", editor.FindReplace},
	{"goto", "转到行…", "⌘L", editor.GoToLine},
	{"dup", "复制当前行", "⇧⌥↓", editor.DuplicateLinesDown},
	{"up", "上移行", "⌥↑", editor.MoveLinesUp},
	{"down", "下移行", "⌥↓", editor.MoveLinesDown},
	{"del", "删除行", "⇧⌘K", editor.DeleteLine},
	{"sort", "排序行", "", editor.SortLines},
	{"sorti", "排序行（忽略大小写）", "", editor.SortLinesCaseInsensitive},
	{"rev", "反转行顺序", "", editor.ReverseLines},
	{"uniq", "删除重复行", "", editor.UniqueLines},
	{"join", "合并行", "⌃J", editor.JoinLines},
	{"upper", "转为大写", "", editor.Uppercase},
	{"lower", "转为小写", "", editor.Lowercase},
	{"title", "转为标题式大小写", "", editor.TitleCase},
	{"trim", "删除行尾空白", "", editor.TrimTrailingWhitespace},
	{"selnext", "选中下一个相同内容（多光标）", "⌘D", editor.SelectNext},
	{"selall", "选中全部相同内容（多光标）", "⇧⌘L", editor.SelectAllMatches},
	{"above", "在上方添加光标", "⌥⌘↑", editor.AddCursorAbove},
	{"below", "在下方添加光标", "⌥⌘↓", editor.AddCursorBelow},
	{"split", "把选区拆成每行一个光标", "⇧⌘L", editor.SplitSelectionIntoLines},
	{"selline", "选中当前行", "", editor.SelectLine},
}

func editorCommandNamed(name string) *editorCommand {
	for i := range editorCommands {
		if editorCommands[i].name == name {
			return &editorCommands[i]
		}
	}
	return nil
}

var _ = strings.TrimSpace
var _ = engine.RuneLen
