package gui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

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
	if s.selected == "" && len(list) > 0 {
		s.selected = list[0].Name
	}
	sel := engine.CleanerByName(s.selected)
	ui.Row(c).Grow(1).Padding(sp4, sp5, sp4, sp5).Gap(sp4).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
		panel(c).Width(340).Padding(sp2).Gap(sp2).Children(func() {
			ui.Row(c).Gap(6).Padding(sp1).AlignItems(ui.Center).Children(func() {
				sf := ui.Row(c.Key("cl-search")).Grow(1).Height(32).Padding(0, 8).Gap(6).Radius(radiusCtl).Background(Paper).AlignItems(ui.Center)
				boxed(sf)
				if sf.FocusWithin() {
					hard(sf, shadowSm)
				}
				sf.Children(func() {
					ui.Icon(c, ic("search")).FontSize(13).TextColor(Ink)
					ui.TextInputBase(c, &s.query).Placeholder("搜索清理器").Label("搜索清理器").Grow(1).MinWidth(0).FontSize(12.5).TextColor(Ink)
				})
				cats := append([]string{"全部"}, engine.Categories()...)
				ui.Select(c, &s.category, cats).Label("分类").Width(104).Height(32).FontSize(12.5)
			})
			ui.Row(c).Padding(0, sp2).Gap(6).AlignItems(ui.Center).Children(func() {
				ui.Textf(c, "%d 个", len(list)).FontSize(11.5).TextColor(Muted).Grow(1)
				flag(c, "meas", "已实测", "只看已实测", &s.onlyMeas)
			})
			ui.List(c, nil, len(list), func(i int) {
				cl := list[i]
				active := sel == cl
				row := ui.ButtonBase(c.Key("cl:"+cl.Token)).Justify(ui.Start).AlignItems(ui.Center).Height(46).Padding(0, sp2).Radius(radiusCtl).Gap(sp2).FillWidth().Label(cl.ZhName).Selected(active)
				bg, bd := ui.Transparent, ui.Transparent
				switch {
				case active:
					bg, bd = Accent, Ink
				case row.Hovered():
					bg = AccentSoft
				}
				row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 100e6})).Border(strokeW, bd)
				focusRing(row)
				row.Children(func() {
					ui.Icon(c, ic(actionIcon(cl.Name))).FontSize(14).TextColor(map[bool]ui.Color{true: Ink, false: Muted}[active])
					ui.Column(c).Grow(1).Gap(1).MinWidth(0).Children(func() {
						ui.Text(c, cl.ZhName).FontSize(13).FontWeight(map[bool]int{true: 700, false: 500}[active]).TextColor(Ink).SingleLine()
						ui.Text(c, cl.Name).FontSize(11).TextColor(map[bool]ui.Color{true: Muted, false: Faint}[active]).SingleLine()
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
					ui.Box(c).Size(52, 52).Radius(radiusSeg).Background(AccentSoft).Border(strokeW, Ink).Center().Children(func() {
						ui.Icon(c, ic("library-big")).FontSize(24).TextColor(Ink)
					})
					hint(c, "选择左侧的清理器，预览它对当前文档的效果。")
				})
				return
			}
			out := sel.Fn(a.Text)
			panel(c).Gap(sp2).Children(func() {
				ui.Row(c).Gap(sp3).AlignItems(ui.Center).Children(func() {
					ui.Box(c).Size(38, 38).Radius(radiusThumb).Background(tileColor(sel.Category)).Border(strokeW, Ink).Center().Children(func() {
						ui.Icon(c, ic(actionIcon(sel.Name))).FontSize(18).TextColor(Ink)
					})
					ui.Column(c).Grow(1).Gap(2).MinWidth(0).Children(func() {
						ui.Text(c, sel.ZhName).FontSize(17).FontWeight(700).TextColor(Ink)
						ui.Text(c, sel.Name+" · "+sel.Category).FontSize(12).TextColor(Muted).SingleLine()
					})
					pinned := a.isPinned(sel.Name)
					label, icon := "固定", "pin"
					if pinned {
						label, icon = "取消固定", "pin-off"
					}
					if ghost(c, "pincl", label, icon, false).Tooltip("固定到常用").Clicked() {
						a.togglePin(sel.Name)
					}
					if primary(c, "applycl", "应用", "play", "", out == a.Text).Clicked() && out != a.Text {
						a.Transform(sel.ZhName, sel.Fn)
					}
				})
				if sel.Desc != "" {
					ui.Text(c, sel.Desc).FontSize(12.5).TextColor(Muted).LineHeight(1.5).Wrap()
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
				ui.Row(c).Gap(sp2).Padding(sp2, sp3).Radius(radiusCtl).Background(bg).Border(strokeW, Ink).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, ic(icon)).FontSize(14).TextColor(fg)
					ui.Text(c, msg).FontSize(12).FontWeight(500).TextColor(fg).Grow(1).Wrap()
				})
			})
			canvasCard(c).Grow(1).MinHeight(0).Clip().Children(func() {
				ui.Row(c).Height(44).Padding(0, sp5).Gap(sp2).AlignItems(ui.Center).BorderWidth(0, 0, strokeW, 0).BorderColor(Ink).Children(func() {
					ui.Text(c, "效果预览").FontSize(13).FontWeight(700).TextColor(Ink)
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
	ui.Column(c).Grow(1).Padding(sp4, sp5, sp4, sp5).Gap(sp3).MinHeight(0).Children(func() {
		step := func(n int, title, sub string, body func()) {
			ui.Row(c).Gap(sp3).Padding(sp3, 0).AlignItems(ui.Center).Children(func() {
				ui.Box(c).Size(26, 26).Radius(13).Background(Accent).Border(strokeW, Ink).Center().Children(func() {
					ui.Textf(c, "%d", n).Font(monoFont).FontSize(11.5).FontWeight(700).TextColor(Ink)
				})
				ui.Column(c).Width(176).Gap(1).Children(func() {
					ui.Text(c, title).FontSize(13).FontWeight(700).TextColor(Ink)
					ui.Text(c, sub).FontSize(11.5).TextColor(Muted)
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
				if len(s.paths) > 0 {
					if ghost(c, "clearf", "清空", "x", false).Clicked() {
						s.paths, s.results, s.applied = nil, nil, false
					}
				}
			})
			ui.Box(c).Height(strokeW).FillWidth().Background(Ink)
			step(2, "用什么处理", "动作链、规则集或清理器", func() {
				if ui.Combobox(c, &s.proc, names).Label("处理方式").Width(320).Changed() {
					s.results = nil
				}
			})
			ui.Box(c).Height(strokeW).FillWidth().Background(Ink)
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
			ui.Row(c).Height(44).Padding(0, sp5).Gap(sp2).AlignItems(ui.Center).BorderWidth(0, 0, strokeW, 0).BorderColor(Ink).Children(func() {
				ui.Text(c, "文件").FontSize(13).FontWeight(700).TextColor(Ink)
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
					ui.Box(c).Size(56, 56).Radius(radiusSeg).Background(AccentSoft).Border(strokeW, Ink).Center().Children(func() {
						ui.Icon(c, ic("files")).FontSize(24).TextColor(Ink)
					})
					if len(s.paths) > 0 {
						ui.Textf(c, "已选 %d 个文件，等待预览", len(s.paths)).FontSize(13.5).FontWeight(700).TextColor(Ink)
						hint(c, "点击上方「预览」查看每个文件将会发生的改动。")
					} else {
						ui.Text(c, "还没有预览").FontSize(13.5).FontWeight(700).TextColor(Ink)
						hint(c, "选择文件和处理方式后点「预览」，这里会列出每个文件的改动。")
					}
				})
				return
			}
			ui.List(c, nil, len(s.results), func(i int) {
				r := s.results[i]
				ui.Row(c).Height(38).Gap(sp3).Padding(0, sp5).AlignItems(ui.Center).BorderWidth(map[bool]float32{true: 0, false: 1}[i == 0], 0, 0, 0).BorderColor(k.hair).Children(func() {
					icon, col := "file-text", Faint
					if r.Changed {
						icon, col = "circle-check", Ink
					}
					if r.Err != nil {
						icon, col = "triangle-alert", Danger
					}
					ui.Icon(c, ic(icon)).FontSize(14).TextColor(col)
					ui.Text(c, filepath.Base(r.Path)).FontSize(13).TextColor(Ink).Grow(1).SingleLine().Tooltip(r.Path)
					if r.Err != nil {
						ui.Text(c, r.Err.Error()).FontSize(11.5).TextColor(Danger)
					} else {
						ui.Textf(c, "%d → %d 字符", r.Before, r.After).FontSize(11.5).TextColor(Muted).FontFeatures("tnum")
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
	section   string // "", "all", "general", "font", "icon", "data", "update"
}

type settingsNavItem struct {
	id, icon, label string
}

var settingsSections = []settingsNavItem{
	{"all", "settings-2", "全部设置"},
	{"general", "monitor", "通用与行为"},
	{"font", "type", "编辑器字体"},
	{"icon", "sparkles", "应用图标"},
	{"data", "database", "数据与兼容"},
	{"update", "download", "更新与关于"},
}

func (a *App) appVersion() string {
	if v := strings.TrimSpace(a.Platform.UpdateInfo().Version); v != "" {
		return v
	}
	return DefaultAppVersion
}

func formatLastCheck(t time.Time) string {
	if t.IsZero() {
		return "尚未检查"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "刚刚"
	case d < time.Hour:
		return fmt.Sprintf("%d 分钟前", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%d 小时前", int(d.Hours()))
	default:
		return t.Local().Format("2006-01-02 15:04")
	}
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
	if s.section == "" {
		s.section = "all"
	}
	upd := a.Platform.UpdateInfo()
	ver := a.appVersion()

	pageHeader(c, "设置", "偏好配置、编辑器字体、图标与软件更新", nil)

	row := func(title, sub string, last bool, ctrl func()) {
		r := ui.Row(c).Gap(sp4).Padding(9, 0).AlignItems(ui.Center)
		if !last {
			r.BorderWidth(0, 0, strokeW, 0).BorderColor(Ink)
		}
		r.Children(func() {
			ui.Column(c).Grow(1).Gap(2).MinWidth(0).Children(func() {
				ui.Text(c, title).FontSize(13).FontWeight(700).TextColor(Ink)
				if sub != "" {
					ui.Text(c, sub).FontSize(11.5).TextColor(Muted).LineHeight(1.38).Wrap()
				}
			})
			ctrl()
		})
	}
	group := func(title string, body func()) {
		eyebrow(c, title).Padding(sp1, sp1, 0, sp1)
		panel(c).Gap(0).Padding(2, sp4).Children(body)
	}
	show := func(sec string) bool {
		return s.section == "all" || s.section == sec
	}

	ui.Row(c).Grow(1).Padding(sp4, sp5, sp4, sp5).Gap(sp4).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
		// Left navigation rail: quick jumps + version/update summary, preventing wasted horizontal space.
		panel(c.Key("settings-nav")).Width(208).Padding(sp2).Gap(4).Children(func() {
			eyebrow(c, "设置分类").Padding(sp1, sp2, sp1, sp2)
			for _, it := range settingsSections {
				sel := s.section == it.id
				b := ui.ButtonBase(c.Key("snav-"+it.id)).Height(36).Padding(0, 10).Radius(radiusCtl).Gap(8).
					AlignItems(ui.Center).Justify(ui.Start).FillWidth().Label(it.label).Selected(sel)
				bg, bd := ui.Transparent, ui.Transparent
				switch {
				case sel:
					bg, bd = Accent, Ink
				case b.Hovered():
					bg = AccentSoft
				}
				b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 100e6})).Border(strokeW, bd)
				focusRing(b)
				b.Children(func() {
					ui.Icon(c, ic(it.icon)).FontSize(14).TextColor(Ink)
					ui.Text(c, it.label).FontSize(sizeButton).FontWeight(map[bool]int{true: 700, false: 500}[sel]).TextColor(Ink).SingleLine()
				})
				if b.Clicked() {
					s.section = it.id
				}
			}
			ui.Spacer(c)
			ui.Box(c).Height(strokeW).FillWidth().Background(Ink).Margin(sp1, 0)
			ui.Column(c.Key("snav-footer")).Padding(sp2, sp2, sp1, sp2).Gap(sp2).Children(func() {
				ui.Row(c).Gap(sp2).AlignItems(ui.Center).Children(func() {
					if bm := appIconBitmap(a.currentAppIcon()); bm != nil {
						ui.Image(c, bm).Size(32, 32).Radius(8)
					} else {
						ui.Image(c, appIconBmp()).Size(32, 32).Radius(8)
					}
					ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
						ui.Text(c, "TextRouter").FontSize(13).FontWeight(700).TextColor(Ink).SingleLine()
						ui.Text(c, "v"+ver).Font(monoFont).FontSize(11).TextColor(Muted).SingleLine()
					})
				})
				if ghost(c, "nav-check-update", "检查更新…", "download", false).FillWidth().Height(30).Clicked() {
					a.Platform.CheckForUpdates()
				}
			})
		})

		// Right scrollable content pane: fills the available sheet width cleanly with no right-side void.
		ui.Scroll(c.Key("settings-scroll")).Grow(1).MinWidth(0).FillHeight().Children(func() {
			ui.Column(c.Key("settings-body")).Padding(0, 8, sp6, 2).Gap(sp2).FillWidth().Children(func() {
				if show("general") {
					group("通用", func() {
						row("外观", "仅影响系统窗口标题栏与外框；应用界面固定为新粗野浅色主题", false, func() {
							if segmented(c, "theme", &s.themeIdx, "跟随系统", "浅色", "深色") {
								st.Theme = []string{"system", "light", "dark"}[s.themeIdx]
								a.Platform.SetTheme(st.Theme)
								a.save()
							}
						})
						row("打开剪贴板文档的全局快捷键", "格式如 CmdOrCtrl+Shift+V、Alt+Space。按下后窗口弹出并读取当前剪贴板。", true, func() {
							applyHotkey := func() {
								if err := a.Platform.SetHotkey(s.hotkey); err != nil {
									a.status = "快捷键设置失败：" + err.Error()
									a.showToast("快捷键设置失败", err.Error())
								} else {
									st.Hotkey = s.hotkey
									a.status = "快捷键已更新为 " + s.hotkey
									a.showToast("快捷键已更新", s.hotkey)
									a.save()
								}
							}
							hk := field(c, ui.TextInputBase(c, &s.hotkey).Label("全局快捷键").Width(200).Font(monoFont).TextColor(Ink))
							if hk.Submitted() {
								applyHotkey()
							}
							if ghost(c, "hk", "应用", "", false).Clicked() {
								applyHotkey()
							}
						})
					})
				}

				if show("font") {
					a.fontSettings(c, k, row, group)
				}

				if show("general") {
					group("行为", func() {
						row("激活时自动读取剪贴板", "剪贴板工作区为空时自动载入；已有内容时在标题栏提示「剪贴板有新内容 · 载入」，点击才替换（可 ⌘Z 找回）", false, func() {
							on := !st.NoAutoReadClipboard
							if ui.Switch(c, &on).Label("切换激活时自动读取剪贴板").Changed() {
								st.NoAutoReadClipboard = !on
								a.save()
							}
						})
						row("「复制并完成」后自动收起窗口", "焦点回到之前的应用，直接 ⌘V 粘贴", false, func() {
							if ui.Switch(c, &st.CopyOnDone).Label("切换复制并完成后自动收起窗口").Changed() {
								a.save()
							}
						})
						row("在动作面板中显示未验证的清理器", "未与 TextSoap 实测输出比对的清理器会带「未验证」标记", true, func() {
							if ui.Switch(c, &st.ShowUnmeasure).Label("切换在动作面板中显示未验证的清理器").Changed() {
								a.save()
							}
						})
					})
				}

				if show("data") {
					group("数据", func() {
						row("资料库位置", a.Lib.Path(), false, func() {
							if ghost(c, "copypath", "复制路径", "copy", false).Clicked() {
								if a.writeClip != nil {
									a.writeClip(a.Lib.Path())
								} else {
									c.WriteClipboard(a.Lib.Path())
								}
								a.showToast("已复制资料库路径", filepath.Base(a.Lib.Path()))
							}
						})
						row("TextSoap 兼容", "导入 .textsoap9Lib 清理器库，或导出为 TextSoap 可读的格式", true, func() {
							ui.Row(c).Gap(sp2).Children(func() {
								if ghost(c, "imp", "导入…", "upload", false).Clicked() {
									a.importTextSoap()
								}
								if ghost(c, "exp", "导出…", "download", false).Clicked() {
									a.exportTextSoap()
								}
							})
						})
					})
				}

				if show("icon") {
					group("应用图标", func() {
						row("图标", "点一个换上；Dock 里立刻生效，下次启动也记得。Finder 里的图标是安装包自带的（Mint Minimal）。", true, func() {})
						a.appIconPicker(c)
					})
				}

				if show("update") {
					group("软件更新", func() {
						sub := fmt.Sprintf("当前版本 v%s · 上次检查：%s", ver, formatLastCheck(upd.LastCheck))
						row("主动检查并下载更新", sub, false, func() {
							if primary(c, "check-updates", "检查并下载更新…", "download", "", false).Clicked() {
								a.Platform.CheckForUpdates()
							}
						})
						if !upd.Enabled {
							ui.Row(c.Key("upd-note")).Gap(sp2).Padding(sp2, sp3).Margin(sp2, 0, sp1, 0).Radius(radiusCtl).Background(Hit).Border(strokeW, Ink).AlignItems(ui.Center).Children(func() {
								ui.Icon(c, ic("info")).FontSize(14).TextColor(Ink)
								ui.Text(c, "当前为开发版或未带更新签名的构建，无法直接原地替换安装；正式发布版支持一键下载安装与自动更新。").FontSize(11.5).FontWeight(500).TextColor(Ink).Grow(1).Wrap()
							})
						}
						row("自动检查更新", "每天在后台自动检查 GitHub Releases 是否有新版本", false, func() {
							on := upd.AutoCheck
							if ui.Switch(c, &on).Label("切换自动检查更新").Changed() {
								a.Platform.SetAutoCheckUpdates(on)
							}
						})
						row("自动下载并安装更新", "后台发现新版本时自动下载校验，下次启动进入新版本", true, func() {
							on := upd.AutoDownload
							if ui.Switch(c, &on).Label("切换自动下载并安装更新").Changed() {
								a.Platform.SetAutoDownloadUpdates(on)
							}
						})
					})
					group("关于", func() {
						total, impl, meas := engine.CleanerStats()
						row("TextRouter "+ver, fmt.Sprintf("清理器 %d（已实现 %d，已实测 %d）· 查找替换语义按 TextSoap 9 的 51 条实测用例实现。", total, impl, meas), true, func() {
							if bm := appIconBitmap(a.currentAppIcon()); bm != nil {
								ui.Image(c, bm).Size(36, 36).Radius(8)
							} else {
								ui.Image(c, appIconBmp()).Size(36, 36).Radius(8)
							}
						})
					})
				}
			})
		})
	})
}
