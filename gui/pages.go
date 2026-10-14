package gui

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

// ---------------------------------------------------------------- cleaners

type cleanersState struct {
	query    string
	category string
	selected string
	onlyMeas bool
}

func (a *App) cleanersView(c *ui.Context) {
	k := tok(c)
	s := &a.clean
	total, impl, meas := engine.CleanerStats()
	pageHeader(c, "清理器库", fmt.Sprintf("%d 个内置清理器 · 已实现 %d · 已实测 %d", total, impl, meas), nil)
	var list []*engine.Cleaner
	q := strings.ToLower(s.query)
	for _, cl := range engine.Cleaners() {
		if s.category != "全部" && s.category != "" && cl.Category != s.category {
			continue
		}
		if s.onlyMeas && !cl.Measured {
			continue
		}
		if q != "" && !strings.Contains(strings.ToLower(cl.ZhName+" "+cl.Name+" "+cl.Desc), q) {
			continue
		}
		list = append(list, cl)
	}
	sel := engine.CleanerByName(s.selected)
	ui.Row(c).Grow(1).Padding(0, sp5, sp4, sp5).Gap(sp4).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
		panel(c).Width(340).Padding(sp2).Gap(sp2).Children(func() {
			ui.Row(c).Gap(6).Padding(sp1).AlignItems(ui.Center).Children(func() {
				sf := ui.Row(c).Grow(1).Height(30).Padding(0, 8).Gap(6).Radius(rMd).Background(k.field).AlignItems(ui.Center)
				sf.Children(func() {
					ui.Icon(c, ic("search")).FontSize(13).TextColor(k.faint)
					ui.TextInputBase(c, &s.query).Placeholder("搜索清理器").Label("搜索清理器").Grow(1).FontSize(12.5).TextColor(k.text)
				})
				cats := append([]string{"全部"}, engine.Categories()...)
				ui.Select(c, &s.category, cats).Label("分类").Width(104).Height(30).FontSize(12.5)
			})
			ui.Row(c).Padding(0, sp2).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "%d 个", len(list)).FontSize(11.5).TextColor(k.faint).Grow(1)
				flag(c, "meas", "已实测", "只看已实测", &s.onlyMeas)
			})
			ui.List(c, nil, len(list), func(i int) {
				cl := list[i]
				row := ui.ButtonBase(c.Key("cl:"+cl.Token)).Justify(ui.Start).AlignItems(ui.Center).Height(46).Padding(0, sp2).Radius(rMd).Gap(sp2).FillWidth().Label(cl.ZhName)
				bg := ui.Transparent
				if sel == cl {
					bg = k.accentSoft
				} else if row.Hovered() {
					bg = k.hover
				}
				row.Background(bg)
				row.Children(func() {
					ui.Icon(c, ic(actionIcon(cl.Name))).FontSize(14).TextColor(map[bool]ui.Color{true: k.accent, false: k.faint}[sel == cl])
					ui.Column(c).Grow(1).Gap(1).MinWidth(0).Children(func() {
						ui.Text(c, cl.ZhName).FontSize(13).FontWeight(500).TextColor(k.text).SingleLine()
						ui.Text(c, cl.Name).FontSize(11).TextColor(k.faint).SingleLine()
					})
					cleanerBadgeView(c, cl)
				})
				if row.Clicked() {
					s.selected = cl.Name
				}
			}).Grow(1)
		})
		ui.Column(c).Grow(1).FillHeight().Gap(sp3).MinWidth(0).Children(func() {
			if sel == nil {
				ui.Column(c).Grow(1).Center().Gap(sp2).Children(func() {
					ui.Icon(c, ic("library-big")).FontSize(28).TextColor(k.faint)
					hint(c, "选择左侧的清理器，预览它对剪贴板空间文本的效果。")
				})
				return
			}
			out := sel.Fn(a.Text)
			panel(c).Gap(sp2).Children(func() {
				ui.Row(c).Gap(sp3).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(36, 36).Radius(10).Background(k.accentSoft).Center().Children(func() {
						ui.Icon(c, ic(actionIcon(sel.Name))).FontSize(18).TextColor(k.accent)
					})
					ui.Column(c).Grow(1).Gap(2).MinWidth(0).Children(func() {
						ui.Text(c, sel.ZhName).FontSize(17).FontWeight(650).TextColor(k.text)
						ui.Text(c, sel.Name+" · "+sel.Category).FontSize(12).TextColor(k.faint).SingleLine()
					})
					pinned := a.isPinned(sel.Name)
					label, icon := "固定", "pin"
					if pinned {
						label, icon = "取消固定", "pin-off"
					}
					if ghost(c, "pincl", label, icon, false).Tooltip("固定到剪贴板空间").Clicked() {
						a.togglePin(sel.Name)
					}
					if primary(c, "applycl", "应用", "play", "", out == a.Text).Clicked() && out != a.Text {
						a.Transform(sel.ZhName, sel.Fn)
					}
				})
				if sel.Desc != "" {
					ui.Text(c, sel.Desc).FontSize(12.5).TextColor(k.muted).LineHeight(1.5).Wrap()
				}
				msg, tn, icon := "已实测：与 TextSoap 9 在 39 个测试输入上的输出一致。", toneGreen, "circle-check"
				switch {
				case !sel.Implemented:
					msg, tn, icon = "此清理器作用于富文本格式或系统服务，在纯文本中不会改变内容。", toneAmber, "info"
				case !sel.Measured:
					msg, tn, icon = "未验证：按 TextSoap 的说明实现，未与 TextSoap 实测输出比对，细节可能不同。", toneAmber, "triangle-alert"
				}
				fg, bg := k.green, k.greenSoft
				if tn == toneAmber {
					fg, bg = k.amber, k.amberSoft
				}
				ui.Row(c).Gap(sp2).Padding(sp2, sp3).Radius(rMd).Background(bg).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, ic(icon)).FontSize(14).TextColor(fg)
					ui.Text(c, msg).FontSize(12).FontWeight(500).TextColor(fg).Grow(1).Wrap()
				})
			})
			canvasCard(c).Grow(1).MinHeight(0).Clip().Children(func() {
				ui.Row(c).Height(44).Padding(0, sp5).Gap(sp2).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(k.hair).Children(func() {
					ui.Text(c, "效果预览").FontSize(13).FontWeight(600).TextColor(k.text)
					diffBadge(c, a.Text, out)
				})
				ui.Scroll(c).Grow(1).Padding(sp5, sp6).Children(func() {
					diffView(c, clip(a.Text), clip(out))
				})
			})
		})
	})
}

func cleanerBadgeView(c *ui.Context, cl *engine.Cleaner) {
	switch {
	case !cl.Implemented:
		badge(c, "富文本", toneNeutral)
	case cl.Measured:
		badge(c, "已实测", toneGreen)
	default:
		badge(c, "未验证", toneAmber)
	}
}

func cleanerBadge(cl *engine.Cleaner) string {
	switch {
	case !cl.Implemented:
		return "富文本"
	case cl.Measured:
		return "已实测"
	}
	return "未验证"
}

// ---------------------------------------------------------------- files

type filesState struct {
	paths   []string
	proc    string
	backup  bool
	results []engine.FileResult
	applied bool
}

func (a *App) processors() ([]string, map[string]func(string) string) {
	var names []string
	fns := map[string]func(string) string{}
	for i := range a.Lib.Chains {
		label, fn := a.resolveAction("chain:" + a.Lib.Chains[i].ID)
		names = append(names, label)
		fns[label] = fn
	}
	for i := range a.Lib.RuleSets {
		label, fn := a.resolveAction("rules:" + a.Lib.RuleSets[i].ID)
		names = append(names, label)
		fns[label] = fn
	}
	for _, cl := range engine.Cleaners() {
		if cl.Implemented {
			names = append(names, cl.ZhName)
			fns[cl.ZhName] = cl.Fn
		}
	}
	return names, fns
}

func (a *App) filesView(c *ui.Context) {
	k := tok(c)
	s := &a.files
	names, fns := a.processors()
	if s.proc == "" && len(names) > 0 {
		s.proc = names[0]
	}
	pageHeader(c, "批量处理文件", "先预览会改动哪些文件，再确认写入", nil)
	ui.Column(c).Grow(1).Padding(0, sp5, sp4, sp5).Gap(sp3).MinHeight(0).Children(func() {
		step := func(n int, title, sub string, body func()) {
			ui.Row(c).Gap(sp3).Padding(sp3, 0).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(24, 24).Radius(12).Background(k.accentSoft).Center().Children(func() {
					ui.Textf(c, "%d", n).FontSize(11.5).FontWeight(700).TextColor(k.accent)
				})
				ui.Column(c).Width(170).Gap(1).Children(func() {
					ui.Text(c, title).FontSize(13).FontWeight(600).TextColor(k.text)
					ui.Text(c, sub).FontSize(11.5).TextColor(k.faint)
				})
				ui.Row(c).Grow(1).Gap(sp2).AlignItems(ui.Center).Children(body)
			})
		}
		panel(c).Gap(0).Padding(sp1, sp4).Children(func() {
			step(1, "选择文件", fmt.Sprintf("已选 %d 个文件", len(s.paths)), func() {
				if ghost(c, "pickf", "选择文件…", "file-text", false).Clicked() {
					a.pickFiles(false)
				}
				if ghost(c, "pickd", "选择文件夹…", "folder-open", false).Clicked() {
					a.pickFiles(true)
				}
			})
			ui.Box(c).Height(1).FillWidth().Background(k.hair)
			step(2, "用什么处理", "动作链、规则集或清理器", func() {
				if ui.Combobox(c, &s.proc, names).Label("处理方式").Width(320).Changed() {
					s.results = nil
				}
			})
			ui.Box(c).Height(1).FillWidth().Background(k.hair)
			step(3, "预览并写入", "写入前可以备份", func() {
				if ghost(c, "preview", "预览", "eye", len(s.paths) == 0).Clicked() && len(s.paths) > 0 {
					s.results, s.applied = engine.ProcessFiles(s.paths, fns[s.proc], false, false), false
				}
				ui.Checkbox(c, &s.backup, "备份为 .bak")
				ui.Spacer(c)
				if primary(c, "write", "写入文件", "download", "", len(s.results) == 0 || s.applied).Clicked() && len(s.results) > 0 && !s.applied {
					s.results, s.applied = engine.ProcessFiles(s.paths, fns[s.proc], true, s.backup), true
					a.status = "已写入文件"
				}
			})
		})
		changed := 0
		for _, r := range s.results {
			if r.Changed {
				changed++
			}
		}
		canvasCard(c).Grow(1).MinHeight(0).Clip().Children(func() {
			ui.Row(c).Height(44).Padding(0, sp5).Gap(sp2).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(k.hair).Children(func() {
				ui.Text(c, "文件").FontSize(13).FontWeight(600).TextColor(k.text)
				if s.results != nil {
					verb := "将改动"
					if s.applied {
						verb = "已改动"
					}
					badge(c, fmt.Sprintf("%s %d / %d", verb, changed, len(s.results)), toneAccent)
				}
			})
			if len(s.results) == 0 {
				ui.Column(c).Grow(1).Center().Gap(sp2).Children(func() {
					ui.Box(c).Size(56, 56).Radius(16).Background(k.field).Center().Children(func() {
						ui.Icon(c, ic("files")).FontSize(24).TextColor(k.faint)
					})
					ui.Text(c, "还没有预览").FontSize(13.5).FontWeight(600).TextColor(k.text)
					hint(c, "选择文件和处理方式后点「预览」，这里会列出每个文件的改动。")
				})
				return
			}
			ui.List(c, nil, len(s.results), func(i int) {
				r := s.results[i]
				ui.Row(c).Height(38).Gap(sp3).Padding(0, sp5).AlignItems(ui.Center).BorderWidth(map[bool]float32{true: 0, false: 1}[i == 0], 0, 0, 0).BorderColor(k.hair).Children(func() {
					icon, col := "file-text", k.faint
					if r.Changed {
						icon, col = "circle-check", k.accent
					}
					if r.Err != nil {
						icon, col = "triangle-alert", k.red
					}
					ui.Icon(c, ic(icon)).FontSize(14).TextColor(col)
					ui.Text(c, filepath.Base(r.Path)).FontSize(13).TextColor(k.text).Grow(1).SingleLine().Tooltip(r.Path)
					if r.Err != nil {
						ui.Text(c, r.Err.Error()).FontSize(11.5).TextColor(k.red)
					} else {
						ui.Textf(c, "%d → %d 字符", r.Before, r.After).FontSize(11.5).TextColor(k.faint).FontFeatures("tnum")
					}
				})
			}).Grow(1)
		})
	})
}

func (a *App) pickFiles(dirs bool) {
	go func() {
		paths := a.Platform.OpenFiles("选择要处理的文件", nil, true, dirs)
		if len(paths) == 0 {
			return
		}
		exp := engine.ExpandPaths(paths)
		a.Platform.Update(func() {
			a.files.paths, a.files.results = exp, nil
		})
	}()
}

// ---------------------------------------------------------------- settings

type settingsState struct {
	hotkey    string
	themeIdx  int
	inited    bool
	splitClip float32
}

func (a *App) settingsView(c *ui.Context) {
	k := tok(c)
	s := &a.setting
	st := &a.Lib.Settings
	if !s.inited {
		s.hotkey = st.Hotkey
		s.themeIdx = map[string]int{"system": 0, "light": 1, "dark": 2}[st.Theme]
		s.inited = true
	}
	pageHeader(c, "设置", "", nil)
	row := func(title, sub string, last bool, ctrl func()) {
		r := ui.Row(c).Gap(sp4).Padding(sp3, 0).AlignItems(ui.Center)
		if !last {
			r.BorderWidth(0, 0, 1, 0).BorderColor(k.hair)
		}
		r.Children(func() {
			ui.Column(c).Grow(1).Gap(2).MinWidth(0).Children(func() {
				ui.Text(c, title).FontSize(13).FontWeight(500).TextColor(k.text)
				if sub != "" {
					ui.Text(c, sub).FontSize(11.5).TextColor(k.faint).LineHeight(1.4).Wrap()
				}
			})
			ctrl()
		})
	}
	group := func(title string, body func()) {
		eyebrow(c, title).Padding(sp2, sp1, 0, sp1)
		panel(c).Gap(0).Padding(sp1, sp4).Children(body)
	}
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Padding(0, sp5, sp6, sp5).Gap(sp2).MaxWidth(720).FillWidth().Children(func() {
			group("通用", func() {
				row("外观", "跟随系统，或固定为浅色 / 深色", false, func() {
					if segmented(c, "theme", &s.themeIdx, "跟随系统", "浅色", "深色") {
						st.Theme = []string{"system", "light", "dark"}[s.themeIdx]
						a.Platform.SetTheme(st.Theme)
						a.save()
					}
				})
				row("打开剪贴板空间的全局快捷键", "格式如 CmdOrCtrl+Shift+V、Alt+Space。按下后窗口弹出并读取当前剪贴板。", true, func() {
					field(c, ui.TextInput(c, &s.hotkey).Label("全局快捷键").Width(200).Font(monoFont))
					if ghost(c, "hk", "应用", "", false).Clicked() {
						if err := a.Platform.SetHotkey(s.hotkey); err != nil {
							a.status = "快捷键设置失败：" + err.Error()
						} else {
							st.Hotkey = s.hotkey
							a.status = "快捷键已更新为 " + s.hotkey
							a.save()
						}
					}
				})
			})
			group("行为", func() {
				row("「复制并完成」后自动收起窗口", "焦点回到之前的应用，直接 ⌘V 粘贴", false, func() {
					if ui.Switch(c, &st.CopyOnDone).Label("「复制并完成」后自动收起窗口").Changed() {
						a.save()
					}
				})
				row("在动作面板中显示未验证的清理器", "未与 TextSoap 实测输出比对的清理器会带「未验证」标记", true, func() {
					if ui.Switch(c, &st.ShowUnmeasure).Label("在动作面板中显示未验证的清理器").Changed() {
						a.save()
					}
				})
			})
			group("数据", func() {
				row("资料库位置", a.Lib.Path(), false, func() {})
				row("TextSoap 兼容", "导入 .textsoap9Lib 清理器库，或导出为 TextSoap 可读的格式", true, func() {
					if ghost(c, "imp", "导入…", "upload", false).Clicked() {
						a.importTextSoap()
					}
					if ghost(c, "exp", "导出…", "download", false).Clicked() {
						a.exportTextSoap()
					}
				})
			})
			group("关于", func() {
				total, impl, meas := engine.CleanerStats()
				row("TextRouter 0.2", fmt.Sprintf("清理器 %d（已实现 %d，已实测 %d）· 查找替换语义按 TextSoap 9 的 51 条实测用例实现。", total, impl, meas), true, func() {
					ui.Image(c, appIconBmp()).Size(32, 32).Radius(8)
				})
			})
		})
	})
}
