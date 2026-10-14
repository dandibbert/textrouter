package gui

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

type chainsState struct {
	selID    string
	expanded map[string]bool // step path -> preview open
	pending  func()
	showDiff bool
}

func (a *App) currentChain() *engine.Chain {
	if ch := a.Lib.ChainByID(a.chains.selID); ch != nil {
		return ch
	}
	return nil
}

type chainRun struct {
	out   string
	trace []engine.StepResult
}

var chainMemo memo

func (a *App) runChain(ch *engine.Chain) chainRun {
	js, _ := json.Marshal(ch.Steps)
	libjs, _ := json.Marshal(a.Lib.Chains)
	return chainMemo.get(hashOf("chain", a.Text, string(js), string(libjs)), func() any {
		out, tr := ch.Run(a.Text, a.Lib.Resolver())
		return chainRun{out, tr}
	}).(chainRun)
}

func (a *App) chainsView(c *ui.Context) {
	t := c.Theme()
	s := &a.chains
	if s.expanded == nil {
		s.expanded = map[string]bool{}
	}
	ch := a.currentChain()
	if ch == nil && len(a.Lib.Chains) > 0 {
		ch = &a.Lib.Chains[0]
		s.selID = ch.ID
	}
	header(c, "动作链", "把多个动作串起来，一键完成。每一步下面都能展开看「做完这一步后」的文本。", func() {
		ui.MenuButton(c, "导入/导出", func(m *ui.Menu) {
			if m.Item("导入 TextSoap 清理器库（.textsoap9Lib / plist）…").Chosen() {
				a.importTextSoap()
			}
			if m.Item("导入 TextRouter JSON…").Chosen() {
				a.importChainsJSON()
			}
			m.Separator()
			if ch != nil && m.Item("导出当前链为 TextRouter JSON…").Chosen() {
				a.exportChainJSON(ch)
			}
			if m.Item("导出全部为 TextSoap 格式（fileVersion 6）…").Chosen() {
				a.exportTextSoap()
			}
		})
		if ch != nil {
			run := a.runChain(ch)
			if ui.PrimaryButton(c, "应用到剪贴板空间").Disabled(run.out == a.Text).Clicked() {
				out := run.out
				a.Transform(ch.Name, func(string) string { return out })
			}
		}
	})
	ui.Row(c).Grow(1).PaddingX(20).PaddingY(4).Gap(16).MinHeight(0).Children(func() {
		// Chain list.
		ui.Column(c).Width(210).FillHeight().Gap(6).Children(func() {
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					for i := range a.Lib.Chains {
						cc := &a.Lib.Chains[i]
						row := ui.ButtonBase(c.Key("ch:"+cc.ID)).Justify(ui.Start).AlignItems(ui.Stretch).Padding(8, 10).Radius(6).FillWidth().Label(cc.Name)
						if cc.ID == s.selID {
							row.Background(t.Selection)
						}
						row.Children(func() {
							ui.Column(c).Gap(1).FillWidth().Children(func() {
								ui.Text(c, cc.Name).FontSize(13).MaxLines(1)
								ui.Textf(c, "%d 步", len(cc.Steps)).FontSize(11).TextColor(t.TextMuted)
							})
						})
						if row.Clicked() {
							s.selID = cc.ID
						}
					}
				})
			})
			ui.Row(c).Gap(6).Children(func() {
				if ui.Button(c, "＋ 新建").Grow(1).Clicked() {
					a.Lib.Chains = append(a.Lib.Chains, engine.Chain{ID: engine.NewID(), Name: fmt.Sprintf("动作链 %d", len(a.Lib.Chains)+1)})
					s.selID = a.Lib.Chains[len(a.Lib.Chains)-1].ID
					a.save()
				}
				if ch != nil && ui.Button(c, "删除").Clicked() {
					id := ch.ID
					for i := range a.Lib.Chains {
						if a.Lib.Chains[i].ID == id {
							a.Lib.Chains = append(a.Lib.Chains[:i], a.Lib.Chains[i+1:]...)
							break
						}
					}
					s.selID = ""
					a.save()
					ch = nil
				}
			})
		})
		if ch == nil {
			ui.Column(c).Grow(1).Center().Children(func() { muted(c, "新建一条动作链开始。") })
			return
		}
		run := a.runChain(ch)
		ui.Column(c).Grow(1).FillHeight().Gap(8).MinWidth(0).Children(func() {
			ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
				if ui.TextInput(c, &ch.Name).Label("动作链名称").FontSize(15).Grow(1).Changed() {
					a.save()
				}
				pinned := a.isPinned("chain:" + ch.ID)
				if ui.Toggle(c, &pinned, "固定到剪贴板空间").Changed() {
					a.togglePin("chain:" + ch.ID)
				}
			})
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(8).Padding(0, 6, 0, 0).Children(func() {
					s.pending = nil
					a.stepList(c, &ch.Steps, "", a.Text, run.trace)
					if s.pending != nil {
						s.pending()
						s.pending = nil
						a.save()
					}
					a.addStepMenu(c, &ch.Steps, "＋ 添加步骤")
					ui.Divider(c)
					ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
						ui.Text(c, "最终结果").Bold()
						diffBadge(c, a.Text, run.out)
						ui.Spacer(c)
						ui.Toggle(c, &s.showDiff, "显示改动")
					})
					ui.Column(c).Background(t.Surface).Radius(8).Border(1, t.Border).Padding(12).Children(func() {
						if s.showDiff {
							diffView(c, a.Text, run.out)
						} else {
							ui.Text(c, clip(run.out)).Selectable().Font("monospace").FontSize(13)
						}
					})
				})
			})
		})
	})
}

func clip(s string) string {
	if r := []rune(s); len(r) > maxPreview {
		return string(r[:maxPreview]) + "\n…（预览已截断）"
	}
	return s
}

// stepList draws steps; trace (top level only) gives each step's output.
func (a *App) stepList(c *ui.Context, steps *[]engine.Step, path string, input string, trace []engine.StepResult) {
	prev := input
	for i := range *steps {
		var res *engine.StepResult
		if trace != nil && i < len(trace) {
			res = &trace[i]
		}
		a.stepCard(c, steps, i, path+fmt.Sprint("/", i), prev, res)
		if res != nil {
			prev = res.Output
		}
	}
	if len(*steps) == 0 {
		muted(c, "还没有步骤。")
	}
}

func (a *App) stepCard(c *ui.Context, steps *[]engine.Step, i int, path string, before string, res *engine.StepResult) {
	t := c.Theme()
	s := &(*steps)[i]
	info := engine.ActionByKind(s.Kind)
	pend := &a.chains.pending
	cc := c.Key("step" + path)
	ui.Column(cc).Gap(8).Padding(10, 12).Radius(10).Background(t.Surface).Border(1, t.Border).Opacity(map[bool]float32{true: 0.55, false: 1}[s.Disabled]).Children(func() {
		ui.Row(c).Gap(8).AlignItems(ui.Center).Children(func() {
			ui.Textf(c, "%d", i+1).FontSize(11).Bold().TextColor(t.AccentText).Background(t.Accent).Radius(9).Padding(1, 6)
			ui.Column(c).Grow(1).Gap(1).MinWidth(0).Children(func() {
				ui.Text(c, info.Title).FontSize(13).Bold()
				ui.Text(c, s.Summary()).FontSize(11).TextColor(t.TextMuted).MaxLines(1)
			})
			if res != nil {
				if res.Err != nil {
					ui.Badge(c, "错误")
				} else {
					diffBadge(c, before, res.Output)
				}
			}
			on := !s.Disabled
			if ui.Switch(c, &on).Label("启用步骤 " + path).Changed() {
				s.Disabled = !on
				a.save()
			}
			ui.Button(c, "⋯").Label("步骤操作 " + path).Menu(func(m *ui.Menu) {
				if m.Item("上移").Chosen() && i > 0 {
					*pend = func() { (*steps)[i-1], (*steps)[i] = (*steps)[i], (*steps)[i-1] }
				}
				if m.Item("下移").Chosen() && i < len(*steps)-1 {
					*pend = func() { (*steps)[i+1], (*steps)[i] = (*steps)[i], (*steps)[i+1] }
				}
				if m.Item("复制一份").Chosen() {
					cp := *s
					*pend = func() { *steps = append((*steps)[:i+1], append([]engine.Step{cp}, (*steps)[i+1:]...)...) }
				}
				m.Separator()
				if m.Item("删除").Chosen() {
					*pend = func() { *steps = append((*steps)[:i], (*steps)[i+1:]...) }
				}
			})
		})
		a.stepFields(c, s, info)
		if info.Container {
			ui.Column(c).Padding(0, 0, 0, 16).Gap(6).BorderWidth(0, 0, 0, 2).BorderColor(t.Border).Children(func() {
				a.stepList(c, &s.Children, path, "", nil)
				a.addStepMenu(c, &s.Children, "＋ 添加子步骤")
			})
		}
		if res != nil {
			if res.Err != nil {
				ui.Text(c, "⚠ "+res.Err.Error()).FontSize(12).TextColor(t.Danger)
			}
			open := a.chains.expanded[path]
			if ui.Collapsible(c, "这一步之后的文本", &open, func() {
				ui.Column(c).Background(t.Background).Radius(6).Padding(8).Children(func() {
					diffView(c, clip(before), clip(res.Output))
				})
			}).Changed() || open != a.chains.expanded[path] {
				a.chains.expanded[path] = open
			}
		}
	})
}

// stepFields shows the editor for the fields a step kind uses.
func (a *App) stepFields(c *ui.Context, s *engine.Step, info engine.ActionInfo) {
	changed := false
	ch := func(e ui.Element) {
		if e.Changed() {
			changed = true
		}
	}
	has := func(f string) bool {
		for _, x := range info.Fields {
			if x == f {
				return true
			}
		}
		return false
	}
	ui.Column(c).Gap(6).Children(func() {
		if has("cleaner") {
			var names []string
			sel := ""
			for _, cl := range engine.Cleaners() {
				if !cl.Implemented {
					continue
				}
				label := cl.ZhName
				if !cl.Measured {
					label += "（未验证）"
				}
				names = append(names, label)
				if cl.Name == s.Cleaner {
					sel = label
				}
			}
			if ui.Combobox(c, &sel, names).Label("清理器").FillWidth().Changed() {
				for _, cl := range engine.Cleaners() {
					if strings.TrimSuffix(sel, "（未验证）") == cl.ZhName {
						s.Cleaner = cl.Name
						changed = true
					}
				}
			}
		}
		if has("find") {
			ui.Row(c).Gap(6).AlignItems(ui.Center).Children(func() {
				ch(ui.TextInput(c, &s.Find.Find).Font("monospace").Placeholder("查找").Label("查找").Grow(1).Basis(0))
				if has("replace") {
					ui.Text(c, "→")
					ch(ui.TextInput(c, &s.Find.Replace).Font("monospace").Placeholder("替换为（留空=删除）").Label("替换为").Grow(1).Basis(0))
				}
			})
			ui.Row(c).Gap(10).Children(func() {
				ch(ui.Checkbox(c, &s.Find.Regex, "正则"))
				ch(ui.Checkbox(c, &s.Find.MatchCase, "区分大小写"))
				if s.Find.Regex {
					ml := strings.Contains(s.Find.RegexOptions, "m")
					if ui.Checkbox(c, &ml, "多行").Changed() {
						s.Find.RegexOptions = toggleLetter(s.Find.RegexOptions, 'm', ml)
						changed = true
					}
					sl := strings.Contains(s.Find.RegexOptions, "s")
					if ui.Checkbox(c, &sl, ". 匹配换行").Changed() {
						s.Find.RegexOptions = toggleLetter(s.Find.RegexOptions, 's', sl)
						changed = true
					}
				}
			})
		}
		if has("rules") {
			muted(c, fmt.Sprintf("%d 条规则。可从「批量替换」的规则集载入：", len(s.Rules)))
			ui.Row(c).Gap(6).Wrap().Children(func() {
				for _, rs := range a.Lib.RuleSets {
					if ui.Button(c.Key("load:"+rs.ID), "载入「"+rs.Name+"」").Clicked() {
						s.Rules = append([]engine.Rule(nil), rs.Rules...)
						changed = true
					}
				}
			})
		}
		if has("text") {
			ph := "文字"
			if s.Kind == engine.KindTag {
				ph = "前缀，如 <b>"
			}
			ch(ui.TextInput(c, &s.Text).Placeholder(ph).Label("文字").FillWidth())
		}
		if has("text2") {
			ch(ui.TextInput(c, &s.Text2).Placeholder("后缀，如 </b>").Label("后缀").FillWidth())
		}
		if has("prefix") {
			ch(ui.TextInput(c, &s.Text).Placeholder("引用前缀，默认 \"> \"").Label("引用前缀").FillWidth())
		}
		if has("sep") {
			ch(ui.TextInput(c, &s.Text).Placeholder("分隔符，默认换行").Label("分隔符").FillWidth())
		}
		if has("small") {
			ch(ui.TextInput(c, &s.Text).Placeholder("额外小写词，逗号分隔").Label("小词").FillWidth())
			ch(ui.TextInput(c, &s.Text2).Placeholder("保持原样的词，逗号分隔").Label("例外").FillWidth())
		}
		if has("chain") {
			var names []string
			sel := ""
			for _, cc := range a.Lib.Chains {
				names = append(names, cc.Name)
				if cc.ID == s.Text {
					sel = cc.Name
				}
			}
			if ui.Select(c, &sel, names).Label("动作链").Changed() {
				for _, cc := range a.Lib.Chains {
					if cc.Name == sel {
						s.Text = cc.ID
						changed = true
					}
				}
			}
		}
		ui.Row(c).Gap(10).AlignItems(ui.Center).Wrap().Children(func() {
			num := func(label string, v *int, lo, hi float64) {
				f := float64(*v)
				if ui.Stepper(c, &f, lo, hi, 1).Label(label).Changed() {
					*v = int(f)
					changed = true
				}
				ui.Textf(c, "%s：%d", label, *v).FontSize(12)
			}
			if has("number") {
				num("字符数（无文字时）", &s.Number, 0, 1000)
			}
			if has("width") {
				if s.Number == 0 {
					s.Number = 70
				}
				num("宽度", &s.Number, 10, 500)
			}
			if has("times") {
				num("次数", &s.Number, 1, 100)
			}
			if has("start") {
				num("起点", &s.Number, 0, 100000)
			}
			if has("length") {
				num("长度（0=到末尾）", &s.Number2, 0, 100000)
			}
			if has("atEnd") {
				ch(ui.Checkbox(c, &s.Flag, "在末尾（否则开头）"))
			}
			if has("desc") {
				ch(ui.Checkbox(c, &s.Flag, "降序"))
			}
			if has("ignoreCase") {
				ch(ui.Checkbox(c, &s.Flag2, "忽略大小写"))
			}
			if has("unique") {
				u := s.Number == 1
				if ui.Checkbox(c, &u, "去重").Changed() {
					s.Number = map[bool]int{true: 1, false: 0}[u]
					changed = true
				}
			}
		})
	})
	if changed {
		a.save()
	}
}

func toggleLetter(s string, l rune, on bool) string {
	s = strings.ReplaceAll(s, string(l), "")
	if on {
		s += string(l)
	}
	return s
}

func (a *App) addStepMenu(c *ui.Context, steps *[]engine.Step, label string) {
	ui.MenuButton(c, label, func(m *ui.Menu) {
		for _, info := range engine.Actions() {
			if m.Item(info.Title).Chosen() {
				st := engine.Step{Kind: info.Kind}
				switch info.Kind {
				case engine.KindCleaner:
					st.Cleaner = "Remove Extra Spaces"
				case engine.KindFind, engine.KindDelete, engine.KindExtract, engine.KindIfMatch:
					st.Find.MatchCase = true
				case engine.KindRepeat:
					st.Number = 2
				}
				*steps = append(*steps, st)
				a.save()
			}
		}
	})
}

func (a *App) isPinned(id string) bool {
	for _, p := range a.Lib.Settings.Pinned {
		if p == id {
			return true
		}
	}
	return false
}

func (a *App) togglePin(id string) {
	ps := a.Lib.Settings.Pinned
	for i, p := range ps {
		if p == id {
			a.Lib.Settings.Pinned = append(ps[:i:i], ps[i+1:]...)
			a.save()
			return
		}
	}
	a.Lib.Settings.Pinned = append(ps, id)
	a.save()
}

func (a *App) importTextSoap() {
	go func() {
		paths := a.Platform.OpenFiles("导入 TextSoap 清理器库", nil, false, false)
		if len(paths) == 0 {
			return
		}
		data, err := os.ReadFile(paths[0])
		var chains []engine.Chain
		if err == nil {
			chains, err = engine.ImportTextSoapLibrary(data)
		}
		a.Platform.Update(func() {
			if err != nil {
				a.status = "导入失败：" + err.Error()
				return
			}
			a.Lib.Chains = append(a.Lib.Chains, chains...)
			a.status = fmt.Sprintf("已从 TextSoap 导入 %d 条动作链（未识别的动作会保留为“不改变文本”的步骤）", len(chains))
			a.save()
		})
	}()
}

func (a *App) importChainsJSON() {
	go func() {
		paths := a.Platform.OpenFiles("导入动作链 JSON", []string{"json"}, false, false)
		if len(paths) == 0 {
			return
		}
		data, err := os.ReadFile(paths[0])
		a.Platform.Update(func() {
			if err != nil {
				a.status = err.Error()
				return
			}
			var many []engine.Chain
			if json.Unmarshal(data, &many) != nil {
				var one engine.Chain
				if err := json.Unmarshal(data, &one); err != nil {
					a.status = "不是有效的动作链 JSON"
					return
				}
				many = []engine.Chain{one}
			}
			for i := range many {
				many[i].ID = engine.NewID()
			}
			a.Lib.Chains = append(a.Lib.Chains, many...)
			a.status = fmt.Sprintf("已导入 %d 条动作链", len(many))
			a.save()
		})
	}()
}

func (a *App) exportChainJSON(ch *engine.Chain) {
	data, _ := json.MarshalIndent(ch, "", "  ")
	go func() {
		if p := a.Platform.SaveFile("导出动作链", ch.Name+".json"); p != "" {
			err := os.WriteFile(p, data, 0o644)
			a.Platform.Update(func() { a.status = errOr(err, "已导出到 "+p) })
		}
	}()
}

func (a *App) exportTextSoap() {
	data, err := engine.ExportTextSoapLibrary(a.Lib.Chains)
	if err != nil {
		a.status = err.Error()
		return
	}
	go func() {
		if p := a.Platform.SaveFile("导出为 TextSoap 格式", "TextRouter.plist"); p != "" {
			err := os.WriteFile(p, data, 0o644)
			a.Platform.Update(func() { a.status = errOr(err, "已导出到 "+p) })
		}
	}()
}

func errOr(err error, ok string) string {
	if err != nil {
		return err.Error()
	}
	return ok
}
