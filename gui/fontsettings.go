package gui

import (
	"fmt"
	"strings"

	"github.com/dandibbert/textrouter/internal/fontlist"
	"github.com/egoist/mygo/ui"
)

// fontUI is the state of the font settings.
type fontUI struct {
	open     string // which picker is open: "editor", "cjk", "mono", "ui"
	query    string
	monoOnly bool
}

const fontSample = "TextRouter 文本清理 —「中文」与 English 混排，0123456789。\nThe quick brown fox jumps over the lazy dog. 天地玄黄，宇宙洪荒。"

// fontSettings is the 字体 group of the settings.
func (a *App) fontSettings(c *ui.Context, k tokens, row func(title, sub string, last bool, ctrl func()), group func(string, func())) {
	fontlist.Load(c.Invalidate)
	fams, ready := fontlist.Ready()
	v := &a.Lib.Settings.View
	group("编辑器字体", func() {
		// Live preview at the top, as it will look in the editor.
		size := float32(v.FontSize)
		ui.Column(c.Key("font-preview")).Gap(sp2).Padding(sp3, sp4).Margin(sp3, 0, sp2, 0).Radius(rMd).Background(k.canvas).Border(1, k.hair).Children(func() {
			ui.Row(c).Gap(sp2).AlignItems(ui.Center).Children(func() {
				eyebrow(c, "实时预览")
				ui.Box(c).Grow(1)
				ui.Textf(c, "%s · %g pt · 行高 %.1f", orDefault(v.EditorFont, "Space Mono（默认）"), v.FontSize, v.LineHeight).FontSize(11).TextColor(k.faint)
			})
			ui.Text(c, fontSample).Font(a.editorFamily()).FontSize(size).LineHeight(float32(v.LineHeight)).TextColor(k.text).Wrap()
			ui.Text(c, "func main() { fmt.Println(\"等宽 Mono 0O1lI\") }").Font(a.monoFamily()).FontSize(size - 1.5).LineHeight(float32(v.LineHeight)).TextColor(k.muted)
		})
		a.fontRow(c, k, row, "editor", "正文字体", "编辑器显示文本用的字体", &v.EditorFont, "Space Mono（默认）", fams, ready, func(f fontlist.Family) bool { return true })
		a.fontRow(c, k, row, "cjk", "中文后备字体", "正文字体里没有的汉字用这个字体显示", &v.CJKFont, "苹方（系统默认）", fams, ready, func(f fontlist.Family) bool { return f.CJK })
		row("字号", "100% 缩放时的大小；⌘+ / ⌘- 只调缩放", false, func() {
			if iconButton(c, "fs-", "minus", "减小字号", "").Clicked() {
				v.FontSize = max(v.FontSize-1, 9)
				a.save()
			}
			ui.Textf(c, "%g", v.FontSize).FontSize(13).FontFeatures("tnum").Width(32).TextAlign(ui.Center).TextColor(k.text)
			if iconButton(c, "fs+", "plus", "增大字号", "").Clicked() {
				v.FontSize = min(v.FontSize+1, 40)
				a.save()
			}
		})
		row("行高", "行与行之间的距离，字号的倍数", false, func() {
			opts := []float64{1.3, 1.45, 1.6, 1.8, 2.0}
			idx := 2
			for i, o := range opts {
				if abs64(o-v.LineHeight) < 0.01 {
					idx = i
				}
			}
			if segmented(c, "lh", &idx, "1.3", "1.45", "1.6", "1.8", "2.0") {
				v.LineHeight = opts[idx]
				a.save()
			}
		})
		a.fontRow(c, k, row, "ui", "界面字体（可选）", "按钮、菜单和面板的字体", &v.UIFont, "Space Grotesk（默认）", fams, ready, func(f fontlist.Family) bool { return !f.Mono })
	})
}

func abs64(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// fontRow is one font setting: the current family, drawn in itself, and a
// searchable list of installed families that opens under it.
func (a *App) fontRow(c *ui.Context, k tokens, row func(string, string, bool, func()), id, title, sub string, val *string, def string, fams []fontlist.Family, ready bool, keep func(fontlist.Family) bool) {
	fu := &a.fontUI
	row(title, sub, id == "ui", func() {
		b := ui.ButtonBase(c.Key("fp-"+id)).Height(30).MinWidth(200).Padding(0, 10, 0, 12).Radius(rMd).Gap(6).AlignItems(ui.Center).
			Background(k.field).Border(1, k.hairStrong).Label(title)
		b.Children(func() {
			name := orDefault(*val, def)
			fam := orDefault(*val, defaultEditorFamily)
			ui.Text(c, name).Font(fam + ", " + defaultCJKFamily).FontSize(13).TextColor(k.text).Grow(1)
			ui.Icon(c, ic(map[bool]string{true: "chevron-down", false: "chevron-right"}[fu.open == id])).FontSize(12).TextColor(k.faint)
		})
		if b.Clicked() {
			if fu.open == id {
				fu.open = ""
			} else {
				fu.open, fu.query = id, ""
			}
		}
	})
	if fu.open != id {
		return
	}
	ui.Column(c.Key("fl-"+id)).Gap(sp2).Padding(sp2, 0, sp3, 0).Children(func() {
		ui.Row(c).Gap(sp2).AlignItems(ui.Center).Children(func() {
			field(c, ui.TextInput(c, &fu.query).Placeholder("搜索已安装的字体…").Label("搜索字体").Grow(1).AutoFocus())
			if id == "editor" {
				ui.Switch(c, &fu.monoOnly).Label("只看等宽")
				ui.Text(c, "只看等宽").FontSize(12).TextColor(k.muted)
			}
		})
		if !ready {
			ui.Text(c, "正在读取已安装的字体…").FontSize(12).TextColor(k.faint)
			return
		}
		q := strings.ToLower(strings.TrimSpace(fu.query))
		var list []fontlist.Family
		for _, f := range fams {
			if !keep(f) || (id == "editor" && fu.monoOnly && !f.Mono) {
				continue
			}
			if q != "" && !strings.Contains(strings.ToLower(f.Name), q) && !strings.Contains(f.LocalName, fu.query) {
				continue
			}
			list = append(list, f)
		}
		ui.Text(c, fmt.Sprintf("%d 个字体 · 每一项都用它自己的字体显示", len(list))).FontSize(11).TextColor(k.faint)
		ui.Scroll(c.Key("fls-"+id)).Height(232).Radius(rMd).Background(k.canvas).Border(1, k.hair).Children(func() {
			ui.Column(c).Padding(4, 4).Children(func() {
				pick := func(key, name, shown, family, note string) {
					sel := *val == name
					r := ui.ButtonBase(c.Key("fi-"+id+key)).Height(34).Padding(0, 10).Radius(rSm).Gap(8).AlignItems(ui.Center).FillWidth().Label(shown)
					pressable(c, r, k, map[bool]ui.Color{true: k.accentSoft, false: ui.Transparent}[sel])
					r.Children(func() {
						ui.Text(c, shown).Font(family).FontSize(15).TextColor(k.text).MaxLines(1)
						if note != "" {
							ui.Text(c, note).FontSize(11).TextColor(k.faint).MaxLines(1)
						}
						ui.Box(c).Grow(1)
						if sel {
							ui.Icon(c, ic("check")).FontSize(14).TextColor(k.accent)
						}
					})
					if r.Clicked() {
						*val = name
						a.save()
					}
				}
				pick("-default", "", def, defaultEditorFamily+", "+defaultCJKFamily, "")
				for i, f := range list {
					if i >= 250 {
						ui.Text(c, "…输入名字以查找更多").FontSize(11).TextColor(k.faint).Padding(6, 10)
						break
					}
					var notes []string
					if f.LocalName != "" {
						notes = append(notes, f.LocalName)
					}
					if f.Mono {
						notes = append(notes, "等宽")
					}
					if f.CJK {
						notes = append(notes, "中文")
					}
					shown := f.Name
					family := f.Name + ", " + defaultCJKFamily
					if id == "cjk" {
						shown = f.Name + "  中文字体示例"
					}
					pick(f.Name, f.Name, shown, family, strings.Join(notes, " · "))
				}
			})
		})
	})
}
