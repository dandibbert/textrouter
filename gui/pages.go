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
	t := c.Theme()
	s := &a.clean
	total, impl, meas := engine.CleanerStats()
	header(c, "清理器库", fmt.Sprintf("共 %d 个内置清理器（兼容 TextSoap 9），已实现 %d 个，其中 %d 个与 TextSoap 实测结果逐条比对一致；其余标为「未验证」。", total, impl, meas), nil)
	ui.Row(c).PaddingX(20).Gap(8).AlignItems(ui.Center).Children(func() {
		ui.SearchField(c, &s.query).Placeholder("搜索清理器").Label("搜索清理器").Width(260)
		cats := append([]string{"全部"}, engine.Categories()...)
		ui.Select(c, &s.category, cats).Label("分类").Width(200)
		ui.Checkbox(c, &s.onlyMeas, "只看已实测")
	})
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
	ui.Row(c).Grow(1).Padding(10, 20, 12, 20).Gap(16).MinHeight(0).Children(func() {
		ui.List(c, nil, len(list), func(i int) {
			cl := list[i]
			row := ui.ButtonBase(c.Key("cl:"+cl.Token)).Justify(ui.Start).AlignItems(ui.Stretch).Padding(8, 10).Radius(6).FillWidth().Label(cl.ZhName)
			if sel == cl {
				row.Background(t.Selection)
			}
			row.Children(func() {
				ui.Row(c).Gap(8).AlignItems(ui.Center).FillWidth().Children(func() {
					ui.Column(c).Grow(1).Gap(1).MinWidth(0).Children(func() {
						ui.Text(c, cl.ZhName).FontSize(13).MaxLines(1)
						ui.Text(c, cl.Name).FontSize(11).TextColor(t.TextMuted).MaxLines(1)
					})
					ui.Badge(c, cleanerBadge(cl))
				})
			})
			if row.Clicked() {
				s.selected = cl.Name
			}
		}).Width(330).FillHeight()
		ui.Column(c).Grow(1).FillHeight().Gap(8).MinWidth(0).Children(func() {
			if sel == nil {
				muted(c, "选择左侧的清理器，可预览它对剪贴板空间文本的效果。")
				return
			}
			ui.Text(c, sel.ZhName).FontSize(17).Bold()
			muted(c, sel.Name+" · "+sel.Token+" · "+sel.Category)
			if sel.Desc != "" {
				ui.Text(c, sel.Desc).FontSize(12).Wrap()
			}
			switch {
			case !sel.Implemented:
				ui.Text(c, "此清理器作用于富文本格式或系统服务，在纯文本中不会改变内容。").FontSize(12).TextColor(t.Warning)
			case !sel.Measured:
				ui.Text(c, "未验证：按 TextSoap 的说明实现，未与 TextSoap 实测输出比对，细节可能不同。").FontSize(12).TextColor(t.Warning)
			default:
				ui.Text(c, "已实测：与 TextSoap 9 在 39 个测试输入上的输出一致。").FontSize(12).TextColor(t.Success)
			}
			out := sel.Fn(a.Text)
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				diffBadge(c, a.Text, out)
				ui.Spacer(c)
				pinned := a.isPinned(sel.Name)
				if ui.Toggle(c, &pinned, "固定到剪贴板空间").Changed() {
					a.togglePin(sel.Name)
				}
				if ui.PrimaryButton(c, "应用").Disabled(out == a.Text).Clicked() {
					a.Transform(sel.ZhName, sel.Fn)
				}
			})
			ui.Scroll(c).Grow(1).Background(t.Surface).Radius(8).Border(1, t.Border).Padding(12).Children(func() {
				diffView(c, clip(a.Text), clip(out))
			})
		})
	})
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
	t := c.Theme()
	s := &a.files
	names, fns := a.processors()
	if s.proc == "" && len(names) > 0 {
		s.proc = names[0]
	}
	header(c, "批量处理文件", "对多个文本文件运行动作链、批量规则或清理器。先预览会改动哪些文件，再确认写入。", nil)
	ui.Column(c).Grow(1).Padding(4, 20, 12, 20).Gap(10).MinHeight(0).Children(func() {
		card(c).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "1. 选择文件").Bold().Width(110)
				if ui.Button(c, "选择文件…").Clicked() {
					a.pickFiles(false)
				}
				if ui.Button(c, "选择文件夹…").Clicked() {
					a.pickFiles(true)
				}
				muted(c, fmt.Sprintf("已选 %d 个文件", len(s.paths)))
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "2. 用什么处理").Bold().Width(110)
				if ui.Combobox(c, &s.proc, names).Label("处理方式").Width(320).Changed() {
					s.results = nil
				}
			})
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				ui.Text(c, "3. 预览并写入").Bold().Width(110)
				if ui.Button(c, "预览").Disabled(len(s.paths) == 0).Clicked() {
					s.results, s.applied = engine.ProcessFiles(s.paths, fns[s.proc], false, false), false
				}
				ui.Checkbox(c, &s.backup, "写入前备份为 .bak")
				if ui.PrimaryButton(c, "写入文件").Disabled(len(s.results) == 0 || s.applied).Clicked() {
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
		if s.results != nil {
			verb := "将改动"
			if s.applied {
				verb = "已改动"
			}
			ui.Textf(c, "%s %d / %d 个文件", verb, changed, len(s.results)).Bold()
		}
		ui.List(c, nil, len(s.results), func(i int) {
			r := s.results[i]
			ui.Row(c).Gap(8).Padding(4, 8).Children(func() {
				mark := "·"
				if r.Changed {
					mark = "●"
				}
				if r.Err != nil {
					mark = "⚠"
				}
				ui.Text(c, mark).Width(16).TextColor(t.Accent)
				ui.Text(c, filepath.Base(r.Path)).Grow(1).MaxLines(1).Tooltip(r.Path)
				if r.Err != nil {
					ui.Text(c, r.Err.Error()).FontSize(11).TextColor(t.Danger)
				} else {
					ui.Textf(c, "%d → %d 字符", r.Before, r.After).FontSize(11).TextColor(t.TextMuted)
				}
			})
		}).Grow(1)
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
	s := &a.setting
	st := &a.Lib.Settings
	if !s.inited {
		s.hotkey = st.Hotkey
		s.themeIdx = map[string]int{"system": 0, "light": 1, "dark": 2}[st.Theme]
		s.inited = true
	}
	header(c, "设置", "", nil)
	ui.Scroll(c).Grow(1).Children(func() {
		ui.Column(c).Padding(4, 20, 20, 20).Gap(12).MaxWidth(680).Children(func() {
			card(c).Children(func() {
				ui.Text(c, "外观").Bold()
				if ui.Segmented(c, &s.themeIdx, "跟随系统", "浅色", "深色").Label("外观").Changed() {
					st.Theme = []string{"system", "light", "dark"}[s.themeIdx]
					a.Platform.SetTheme(st.Theme)
					a.save()
				}
			})
			card(c).Children(func() {
				ui.Text(c, "打开剪贴板空间的全局快捷键").Bold()
				muted(c, "格式如 CmdOrCtrl+Shift+V、Alt+Space。按下后窗口弹出并读取当前剪贴板。")
				ui.Row(c).Gap(8).Children(func() {
					ui.TextInput(c, &s.hotkey).Label("全局快捷键").Width(240).Font("monospace")
					if ui.Button(c, "应用").Clicked() {
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
			card(c).Children(func() {
				ui.Text(c, "行为").Bold()
				if ui.Checkbox(c, &st.CopyOnDone, "「复制并完成」后自动收起窗口").Changed() {
					a.save()
				}
				if ui.Checkbox(c, &st.ShowUnmeasure, "在动作面板中显示未验证的清理器").Changed() {
					a.save()
				}
			})
			card(c).Children(func() {
				ui.Text(c, "数据").Bold()
				muted(c, "动作链、规则集和设置保存在：")
				ui.Text(c, a.Lib.Path()).Font("monospace").FontSize(12).Selectable()
				ui.Row(c).Gap(8).Children(func() {
					if ui.Button(c, "导入 TextSoap 清理器库…").Clicked() {
						a.importTextSoap()
					}
					if ui.Button(c, "导出为 TextSoap 格式…").Clicked() {
						a.exportTextSoap()
					}
				})
			})
			card(c).Children(func() {
				ui.Text(c, "关于").Bold()
				total, impl, meas := engine.CleanerStats()
				muted(c, fmt.Sprintf("TextRouter · 清理器 %d（已实现 %d，已实测 %d）· 查找替换语义按 TextSoap 9 的 51 条实测用例实现。", total, impl, meas))
			})
		})
	})
}
