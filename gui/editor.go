package gui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
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
	zoom        float32 // 1 = 100%
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

// scopePill shows what the next action will apply to.
func (a *App) scopePill(c *ui.Context, k tokens) {
	sels := a.selections()
	if len(sels) == 0 {
		ui.Text(c, "作用于全文").FontSize(11.5).TextColor(Muted).SingleLine().Tooltip("选中文字后，动作只作用于选区")
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
	ui.Row(c).Gap(5).Height(24).Padding(0, 8).Radius(radiusPill).Background(Accent).Border(strokeW, Ink).AlignItems(ui.Center).Children(func() {
		ui.Text(c, label).FontSize(11.5).FontWeight(700).TextColor(Ink).SingleLine()
	}).Tooltip("动作、查找替换和批量替换只作用于选中的文字")
}

func (a *App) renderGoToLine(c *ui.Context, bar ui.Element, g *editor.GoToLineBar) {
	k := tok(c)
	hard(bar.Row().Gap(8).Padding(8).Radius(radiusCtl).Background(Paper).Border(strokeW, Ink).AlignItems(ui.Center).Width(320), shadowSm)
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
	a.ensureDocs()
	for i, d := range a.docs {
		if d.path == path {
			a.switchDoc(i)
			return
		}
	}
	a.docs = append(a.docs, &document{name: filepath.Base(path), path: path, text: string(data)})
	a.switchDoc(len(a.docs) - 1)
	a.status = "已打开 " + filepath.Base(path)
}

// OpenPath opens a file the system handed over (dropped on the Dock icon,
// "Open With", or the command line) as a document tab. The activation that
// comes with it then leaves the tab alone instead of going back to the
// clipboard.
func (a *App) OpenPath(path string) {
	a.loadFile(path)
	a.openedAt = time.Now()
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
	if d := a.doc(); d.path != "" {
		name = filepath.Base(d.path)
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
			if d := a.doc(); !d.clip {
				d.path, d.name = p, filepath.Base(p)
			}
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
