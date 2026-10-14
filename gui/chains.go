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
	return chainMemo.get(hashOf("chain", a.scopeText(), string(js), string(libjs)), func() any {
		out, tr := ch.Run(a.scopeText(), a.Lib.Resolver())
		return chainRun{out, tr}
	}).(chainRun)
}

func (a *App) chainsView(c *ui.Context) {
	k := tok(c)
	s := &a.chains
	if s.expanded == nil {
		s.expanded = map[string]bool{}
	}
	ch := a.currentChain()
	if ch == nil && len(a.Lib.Chains) > 0 {
		ch = &a.Lib.Chains[0]
		s.selID = ch.ID
	}
	pageHeader(c, "动作链", "把多个动作串起来，一键完成", func() {
		a.scopePill(c, k)
		ui.Box(c).Width(sp2)
		ui.MenuButton(c, "导入 / 导出", func(m *ui.Menu) {
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
		}).Height(30).Radius(rMd).Background(k.field).Border(1, ui.Transparent).FontSize(12.5)
		if ch != nil {
			run := a.runChain(ch)
			if primary(c, "applychain", "应用到剪贴板空间", "play", "", run.out == a.scopeText()).Clicked() && run.out != a.scopeText() {
				a.Transform(ch.Name, a.chainFn(ch))
			}
		}
	})
	ui.Row(c).Grow(1).Padding(0, sp5, sp4, sp5).Gap(sp4).MinHeight(0).AlignItems(ui.Stretch).Children(func() {
		// Chain list.
		panel(c).Width(220).Padding(sp2).Gap(2).Children(func() {
			ui.Row(c).Padding(sp1, sp1, sp2, sp2).AlignItems(ui.Center).Children(func() {
				eyebrow(c, "我的动作链").Grow(1)
				if iconButton(c, "newchain", "plus", "新建动作链", "新建动作链").Size(24, 24).Clicked() {
					a.Lib.Chains = append(a.Lib.Chains, engine.Chain{ID: engine.NewID(), Name: fmt.Sprintf("动作链 %d", len(a.Lib.Chains)+1)})
					s.selID = a.Lib.Chains[len(a.Lib.Chains)-1].ID
					a.save()
				}
			})
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(2).Children(func() {
					for i := range a.Lib.Chains {
						cc := &a.Lib.Chains[i]
						sel := cc.ID == s.selID
						row := ui.ButtonBase(c.Key("ch:"+cc.ID)).Justify(ui.Start).AlignItems(ui.Center).Height(46).Padding(0, sp2).Radius(rMd).Gap(sp2).FillWidth().Label(cc.Name).Selected(sel)
						bg := ui.Transparent
						if sel {
							bg = k.accentSoft
						} else if row.Hovered() {
							bg = k.hover
						}
						row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 120e6}))
						row.Children(func() {
							icol, ibg := k.muted, k.field
							if sel {
								icol, ibg = k.accentText, k.accent
							}
							ui.Box(c).Size(26, 26).Radius(7).Background(ibg).Center().Children(func() {
								ui.Icon(c, ic("workflow")).FontSize(14).TextColor(icol)
							})
							ui.Column(c).Gap(1).Grow(1).MinWidth(0).Children(func() {
								ui.Text(c, cc.Name).FontSize(13).FontWeight(map[bool]int{true: 600, false: 500}[sel]).TextColor(k.text).SingleLine()
								ui.Textf(c, "%d 步", len(cc.Steps)).FontSize(11).TextColor(k.faint)
							})
							if a.isPinned("chain:" + cc.ID) {
								ui.Icon(c, ic("pin")).FontSize(12).TextColor(k.faint)
							}
						})
						if row.Clicked() {
							s.selID = cc.ID
						}
					}
				})
			})
		})
		if ch == nil {
			ui.Column(c).Grow(1).Center().Gap(sp2).Children(func() {
				ui.Icon(c, ic("workflow")).FontSize(28).TextColor(k.faint)
				hint(c, "新建一条动作链开始。")
			})
			return
		}
		run := a.runChain(ch)
		ui.Column(c).Grow(1).FillHeight().Gap(sp3).MinWidth(0).Children(func() {
			ui.Row(c).Gap(sp2).AlignItems(ui.Center).Children(func() {
				name := ui.TextInputBase(c, &ch.Name).Label("动作链名称").FontSize(18).FontWeight(650).TextColor(k.text).Grow(1).Padding(4, 6).Radius(rSm)
				if name.Hovered() || name.Focused() {
					name.Background(k.field)
				}
				if name.Changed() {
					a.save()
				}
				pinned := a.isPinned("chain:" + ch.ID)
				label, icon := "固定到剪贴板空间", "pin"
				if pinned {
					label, icon = "已固定", "pin-off"
				}
				if ghost(c, "pinchain", label, icon, false).Clicked() {
					a.togglePin("chain:" + ch.ID)
				}
				if iconButton(c, "delchain", "trash-2", "删除", "删除这条动作链").Clicked() {
					id := ch.ID
					for i := range a.Lib.Chains {
						if a.Lib.Chains[i].ID == id {
							a.Lib.Chains = append(a.Lib.Chains[:i], a.Lib.Chains[i+1:]...)
							break
						}
					}
					s.selID = ""
					a.save()
				}
			})
			if a.Lib.ChainByID(ch.ID) == nil {
				return
			}
			ui.Scroll(c).Grow(1).Children(func() {
				ui.Column(c).Gap(0).Padding(2, 6, 4, 2).Children(func() {
					s.pending = nil
					a.stepList(c, &ch.Steps, "", a.scopeText(), run.trace)
					if s.pending != nil {
						s.pending()
						s.pending = nil
						a.save()
					}
					ui.Row(c).Padding(sp2, 0, sp4, 38).Children(func() { a.addStepMenu(c, &ch.Steps, "添加步骤") })
					canvasCard(c).Children(func() {
						ui.Row(c).Height(44).Padding(0, sp3, 0, sp5).Gap(sp2).AlignItems(ui.Center).BorderWidth(0, 0, 1, 0).BorderColor(k.hair).Children(func() {
							ui.Icon(c, ic("circle-check")).FontSize(14).TextColor(k.green)
							ui.Text(c, "最终结果").FontSize(13).FontWeight(600).TextColor(k.text)
							diffBadge(c, a.scopeText(), run.out)
							ui.Spacer(c)
							idx := map[bool]int{false: 0, true: 1}[s.showDiff]
							if segmented(c, "cview", &idx, "结果", "改动对照") {
								s.showDiff = idx == 1
							}
						})
						ui.Column(c).Padding(sp5, sp6).Children(func() {
							if s.showDiff {
								diffView(c, a.scopeText(), run.out)
							} else {
								ui.Text(c, clip(run.out)).Selectable().Font(editorFont).FontSize(15).LineHeight(1.65).TextColor(k.text)
							}
						})
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
		a.stepCard(c, steps, i, path+fmt.Sprint("/", i), prev, res, i == len(*steps)-1)
		if res != nil {
			prev = res.Output
		}
	}
	if len(*steps) == 0 {
		hint(c, "还没有步骤。").Padding(sp2, 0, sp2, 38)
	}
}

func (a *App) stepCard(c *ui.Context, steps *[]engine.Step, i int, path string, before string, res *engine.StepResult, last bool) {
	k := tok(c)
	s := &(*steps)[i]
	info := engine.ActionByKind(s.Kind)
	pend := &a.chains.pending
	cc := c.Key("step" + path)
	ui.Row(cc).Gap(sp3).AlignItems(ui.Stretch).Children(func() {
		// Timeline: number bubble and connector.
		ui.Column(c).Width(26).AlignItems(ui.Center).Children(func() {
			bg, fg := k.accent, k.accentText
			if s.Disabled {
				bg, fg = k.pill, k.faint
			}
			ui.Box(c).Size(24, 24).Radius(12).Background(bg).Center().Margin(sp3, 0, 0, 0).Children(func() {
				ui.Textf(c, "%d", i+1).FontSize(11.5).FontWeight(700).TextColor(fg).FontFeatures("tnum")
			})
			if !last {
				ui.Box(c).Width(2).Grow(1).Radius(1).Background(k.hairStrong).Margin(4, 0, 0, 0)
			}
		})
		ui.Column(c).Grow(1).MinWidth(0).Padding(0, 0, sp3, 0).Children(func() {
			card := panel(c).Gap(sp3).Padding(sp3, sp3, sp3, sp4)
			if s.Disabled {
				card.Opacity(0.55)
			}
			card.Children(func() {
				ui.Row(c).Gap(sp2).AlignItems(ui.Center).Children(func() {
					ui.Icon(c, ic(stepIcon(s.Kind))).FontSize(15).TextColor(k.accent)
					ui.Column(c).Grow(1).Gap(1).MinWidth(0).Children(func() {
						ui.Text(c, info.Title).FontSize(13.5).FontWeight(600).TextColor(k.text).SingleLine()
						ui.Text(c, s.Summary()).FontSize(11.5).TextColor(k.faint).SingleLine()
					})
					if res != nil {
						if res.Err != nil {
							badge(c, "错误", toneRed)
						} else {
							diffBadge(c, before, res.Output)
						}
					}
					on := !s.Disabled
					if ui.Switch(c, &on).Label("启用步骤 " + path).Changed() {
						s.Disabled = !on
						a.save()
					}
					iconButton(c, "more", "ellipsis", "步骤操作 "+path, "更多").Size(28, 28).Menu(func(m *ui.Menu) {
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
					ui.Column(c).Padding(sp1, 0, 0, sp3).Gap(sp2).BorderWidth(0, 0, 0, 2).BorderColor(k.accentSoft).Children(func() {
						a.stepList(c, &s.Children, path, "", nil)
						a.addStepMenu(c, &s.Children, "添加子步骤")
					})
				}
				if res != nil {
					if res.Err != nil {
						ui.Text(c, res.Err.Error()).FontSize(12).TextColor(k.red)
					}
					open := a.chains.expanded[path]
					h := ui.ButtonBase(c.Key("exp")).Justify(ui.Start).AlignItems(ui.Center).Gap(6).Label("这一步之后的文本")
					h.Children(func() {
						rot := h.AnimateTo("rot", map[bool]float32{true: 90, false: 0}[open], quick)
						ui.Icon(c, ic("chevron-right")).FontSize(12).TextColor(k.faint).Rotate(rot)
						ui.Text(c, "这一步之后的文本").FontSize(12).FontWeight(500).TextColor(map[bool]ui.Color{true: k.text, false: k.muted}[h.Hovered()])
					})
					if h.Clicked() {
						open = !open
						a.chains.expanded[path] = open
					}
					if open {
						ui.Column(c).Background(k.field).Radius(rMd).Padding(sp3, sp4).Children(func() {
							diffView(c, clip(before), clip(res.Output))
						})
					}
				}
			})
		})
	})
}

// stepIcon picks an icon per step kind.
func stepIcon(kind string) string {
	switch kind {
	case engine.KindCleaner:
		return "sparkles"
	case engine.KindFind:
		return "replace-all"
	case engine.KindDelete:
		return "eraser"
	case engine.KindExtract:
		return "scissors"
	case engine.KindTag:
		return "type"
	case engine.KindRepeat:
		return "redo-2"
	case engine.KindIfMatch:
		return "text-search"
	}
	return "wand-sparkles"
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
				ch(field(c, ui.TextInput(c, &s.Find.Find)).Font(monoFont).Placeholder("查找").Label("查找").Grow(1).Basis(0))
				if has("replace") {
					ui.Icon(c, ic("arrow-right")).FontSize(13).TextColor(tok(c).faint)
					ch(field(c, ui.TextInput(c, &s.Find.Replace)).Font(monoFont).Placeholder("替换为（留空=删除）").Label("替换为").Grow(1).Basis(0))
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
			ch(field(c, ui.TextInput(c, &s.Text)).Placeholder(ph).Label("文字").FillWidth())
		}
		if has("text2") {
			ch(field(c, ui.TextInput(c, &s.Text2)).Placeholder("后缀，如 </b>").Label("后缀").FillWidth())
		}
		if has("prefix") {
			ch(field(c, ui.TextInput(c, &s.Text)).Placeholder("引用前缀，默认 \"> \"").Label("引用前缀").FillWidth())
		}
		if has("sep") {
			ch(field(c, ui.TextInput(c, &s.Text)).Placeholder("分隔符，默认换行").Label("分隔符").FillWidth())
		}
		if has("small") {
			ch(field(c, ui.TextInput(c, &s.Text)).Placeholder("额外小写词，逗号分隔").Label("小词").FillWidth())
			ch(field(c, ui.TextInput(c, &s.Text2)).Placeholder("保持原样的词，逗号分隔").Label("例外").FillWidth())
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
	k := tok(c)
	b := ui.ButtonBase(c.Key("add:"+label)).Height(30).Padding(0, 12, 0, 10).Radius(rMd).Gap(6).AlignItems(ui.Center).Label(label)
	pressable(c, b, k, ui.Transparent)
	b.Border(1, k.hairStrong).BorderStyle(ui.BorderDashed)
	b.Children(func() {
		ui.Icon(c, ic("plus")).FontSize(14).TextColor(k.accent)
		ui.Text(c, label).FontSize(12.5).FontWeight(600).TextColor(k.accent)
	})
	b.Menu(func(m *ui.Menu) {
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
