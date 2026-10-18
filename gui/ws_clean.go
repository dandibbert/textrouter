package gui

import (
	"fmt"
	"strings"
	"time"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

// applyFx is the "apply" animation: for 500ms the preview of the change
// stays while what was deleted fades out, then the real text shows with what
// is new boxed in black, its fill fading over 1.3s.
type applyFx struct {
	pv *preview
	at time.Time
}

// searchField is the stroked field with a leading icon and a trailing
// keycap, that lifts and grows its shadow while it has the focus.
func searchField(c *ui.Context, key string, value *string, placeholder, label string, keycap string) ui.Element {
	row := ui.Row(c.Key(key)).Height(38).Padding(0, 10).Gap(8).AlignItems(ui.Center).Background(Paper).Radius(radiusCtl)
	boxed(row)
	focused := row.FocusWithin()
	t := row.AnimateWith("lift", map[bool]float32{true: -2, false: 0}[focused], dRelease, EaseSpring)
	row.Translate(t, t)
	hard(row, shadowSm-t*1)
	row.Children(func() {
		ui.Icon(c, ic("search")).FontSize(15).TextColor(Ink)
		ui.TextInputBase(c, value).Placeholder(placeholder).Label(label).Grow(1).MinWidth(0).FontSize(sizeBody).TextColor(Ink)
		if keycap != "" {
			kbd(c, keycap)
		}
	})
	return row
}

// glyphOf is the one or two characters on a cleaner's tile.
func glyphOf(cl *engine.Cleaner) string {
	switch cl.Category {
	case "空白":
		return "␣"
	case "引用与换行宽度":
		return "”"
	case "换行与段落":
		return "¶"
	case "行操作":
		return "≡"
	case "大小写":
		return "Aa"
	case "日期":
		return "日"
	case "字符与编码":
		return "é"
	}
	r := []rune(cl.ZhName)
	if len(r) == 0 {
		return "·"
	}
	return string(r[:1])
}

type cleanRow struct {
	id, name, glyph string
	tile            ui.Color
	chain           bool
}

type cleanSection struct {
	title, sub string
	rows       []cleanRow
}

func (a *App) cleanRowFor(id string) (cleanRow, bool) {
	switch {
	case strings.HasPrefix(id, "chain:"):
		ch := a.Lib.ChainByID(strings.TrimPrefix(id, "chain:"))
		if ch == nil {
			return cleanRow{}, false
		}
		return cleanRow{id: id, name: ch.Name, tile: TileChain, chain: true}, true
	case strings.HasPrefix(id, "rules:"):
		label, fn := a.resolveAction(id)
		if fn == nil {
			return cleanRow{}, false
		}
		return cleanRow{id: id, name: label, tile: TileChain, chain: true}, true
	}
	cl := engine.CleanerByName(id)
	if cl == nil || !cl.Implemented {
		return cleanRow{}, false
	}
	return cleanRow{id: id, name: cl.ZhName, glyph: glyphOf(cl), tile: tileColor(cl.Category)}, true
}

// cleanSections lists what the cleaner tab shows: the pinned cleaners and the
// action chains, or the search results among all of them.
func (a *App) cleanSections() []cleanSection {
	q := strings.ToLower(strings.TrimSpace(a.ws.query))
	if q != "" {
		var rows []cleanRow
		for i := range a.Lib.Chains {
			if strings.Contains(strings.ToLower(a.Lib.Chains[i].Name), q) {
				if r, ok := a.cleanRowFor("chain:" + a.Lib.Chains[i].ID); ok {
					rows = append(rows, r)
				}
			}
		}
		for i := range a.Lib.RuleSets {
			if strings.Contains(strings.ToLower(a.Lib.RuleSets[i].Name), q) {
				if r, ok := a.cleanRowFor("rules:" + a.Lib.RuleSets[i].ID); ok {
					rows = append(rows, r)
				}
			}
		}
		for _, cl := range engine.Cleaners() {
			if !cl.Implemented || !strings.Contains(strings.ToLower(cl.ZhName+" "+cl.Name), q) {
				continue
			}
			if r, ok := a.cleanRowFor(cl.Name); ok {
				rows = append(rows, r)
			}
		}
		if len(rows) == 0 {
			return nil
		}
		return []cleanSection{{"搜索结果", fmt.Sprintf("%d 个", len(rows)), rows}}
	}
	var pinned, chains []cleanRow
	for _, id := range a.Lib.Settings.Pinned {
		if strings.HasPrefix(id, "chain:") {
			continue
		}
		if r, ok := a.cleanRowFor(id); ok {
			pinned = append(pinned, r)
		}
	}
	for i := range a.Lib.Chains {
		if r, ok := a.cleanRowFor("chain:" + a.Lib.Chains[i].ID); ok {
			chains = append(chains, r)
		}
	}
	for i := range a.Lib.RuleSets {
		if r, ok := a.cleanRowFor("rules:" + a.Lib.RuleSets[i].ID); ok {
			chains = append(chains, r)
		}
	}
	return []cleanSection{
		{"常用", "数字 = 对当前文本的改动", pinned},
		{"动作链", "多步组合", chains},
	}
}

// applyAction applies a cleaner, chain or rule set to the selection or the
// whole text, then plays the apply animation. A cleaner that would change
// nothing does nothing.
func (a *App) applyAction(id string) {
	label, fn := a.resolveAction(id)
	if fn == nil {
		return
	}
	if len(a.selections()) == 0 && !a.actionResult(id).changed() {
		return
	}
	var act *engine.Step
	switch {
	case strings.HasPrefix(id, "chain:"):
		act = &engine.Step{Kind: engine.KindChain, Text: strings.TrimPrefix(id, "chain:")}
	case strings.HasPrefix(id, "rules:"):
		for i := range a.Lib.RuleSets {
			if a.Lib.RuleSets[i].ID == strings.TrimPrefix(id, "rules:") {
				act = &engine.Step{Kind: engine.KindBatch, Rules: a.Lib.RuleSets[i].Rules}
			}
		}
	default:
		act = &engine.Step{Kind: engine.KindCleaner, Cleaner: id}
	}
	a.transformStep(label, act, fn)
	a.ws.hov = ""
}

// cleanTab is the 清理 tab: search, then the cleaners, then the footer.
func (a *App) cleanTab(c *ui.Context, k tokens) {
	w := &a.ws
	searchField(c, "q", &w.query, "搜索清理器、动作链…", "搜索动作", "⌘K")
	secs := a.cleanSections()
	total := 0
	for _, s := range secs {
		total += len(s.rows)
	}
	listHov := false
	w.rowHov, w.headHov = "", false
	scroll := ui.Scroll(c.Key("cleanlist")).Grow(1).MinHeight(0).Margin(6, -4, 0, -4).Padding(0, 4, 8, 4)
	scroll.Children(func() {
		idx := 0
		for _, sec := range secs {
			if len(sec.rows) == 0 && sec.title != "常用" {
				continue
			}
			hdr := ui.Row(c.Key("sec"+sec.title)).FillWidth().Padding(14, 8, 6, 8).AlignItems(ui.End)
			hdr.Children(func() {
				eyebrow(c, sec.title)
				ui.Spacer(c)
				ui.Text(c, sec.sub).FontSize(sizeSmall).TextColor(Muted).SingleLine()
			})
			if hdr.Hovered() {
				w.headHov = true
			}
			if len(sec.rows) == 0 {
				hint(c, "悬停清理器后点 ☆，加入常用").Padding(0, 8)
			}
			for _, r := range sec.rows {
				a.cleanRow(c, k, r, idx)
				idx++
			}
		}
		if total == 0 && strings.TrimSpace(w.query) != "" {
			q := strings.TrimSpace(w.query)
			dashed(ui.Column(c.Key("noresult")).Margin(18, 4).Padding(16).Gap(10).AlignItems(ui.Start).Radius(radiusSeg).Children(func() {
				ui.Text(c, "没有叫「"+q+"」的动作").FontSize(13).FontWeight(700).TextColor(Ink)
				ui.Text(c, "可以直接把它当成要查找的文字。").FontSize(12).TextColor(Muted)
				if ghost(c, "q2find", "用「"+cut(q, 12)+"」新建查找替换 →", "", false).Clicked() {
					a.searchToFind(q)
				}
			}))
		}
	})
	if scroll.Hovered() {
		listHov = true
	}
	switch {
	case w.rowHov != "":
		// The row under the pointer is the hot one, wherever the last one was.
		w.frameHov = w.rowHov
	case listHov && !w.headHov:
		// In the gap between two rows: keep what was showing.
		w.frameHov = w.hov
	default:
		// Off the list, or on a heading: the text is put back.
		w.frameHov = ""
	}
	nb := ghost(c, "newcleaner", "新建清理器", "plus", false).FillWidth().Margin(8, 0, 10, 0).
		Tooltip("把已有的清理器、查找替换按顺序组合成自己的清理器")
	if nb.Clicked() {
		a.newChain()
	}
	_, impl, _ := engine.CleanerStats()
	ui.Row(c.Key("cleanfoot")).FillWidth().Padding(10, 6, 0, 6).AlignItems(ui.Center).BorderWidth(strokeW, 0, 0, 0).BorderColor(Ink).Children(func() {
		l := ui.ButtonBase(c.Key("alllib")).Label("全部清理器").Tooltip("打开清理器库")
		l.Children(func() {
			ui.Text(c, fmt.Sprintf("全部 %d 个清理器 →", impl)).FontSize(sizeButton).FontWeight(700).TextColor(Ink).Underline().SingleLine()
		})
		focusRing(l)
		if l.Clicked() {
			a.OpenTool(PageCleaners)
		}
		ui.Spacer(c)
		ui.Text(c, "悬停后点 ☆ 加入常用").FontSize(sizeButton).TextColor(Muted).SingleLine()
	})
}

// newChain makes an empty action chain (a custom cleaner) and opens the chain
// editor on it, where cleaners and other steps are added one by one.
func (a *App) newChain() {
	a.Lib.Chains = append(a.Lib.Chains, engine.Chain{ID: engine.NewID(), Name: fmt.Sprintf("动作链 %d", len(a.Lib.Chains)+1)})
	a.chains.selID = a.Lib.Chains[len(a.Lib.Chains)-1].ID
	a.save()
	a.OpenTool(PageChains)
}

// searchToFind turns a search that matched nothing into a plain find.
func (a *App) searchToFind(q string) {
	a.setTab(1)
	a.find.modeIdx = 0
	a.find.opts.Find, a.find.opts.Replace = q, ""
	a.ws.query = ""
	a.ws.find.reset()
}

// cleanRow is one action: tile, name, badge and star. Hovering it previews it
// in the text; clicking applies it.
func (a *App) cleanRow(c *ui.Context, k tokens, r cleanRow, idx int) {
	w := &a.ws
	res := a.actionResult(r.id)
	hot := w.hov == r.id
	pinned := a.isPinned(r.id)

	in := anim(c, w.tabAt.Add(min(time.Duration(idx)*dListStep, 12*dListStep)), dListIn, EaseOut)
	row := ui.Row(c.Key("row:"+r.id)).FillWidth().Height(42).Padding(0, 4, 0, 0).Gap(2).AlignItems(ui.Center).Radius(radiusCtl).Margin(0, 0, 2, 0)
	bg, bd := ui.Transparent, ui.Transparent
	if hot {
		bg, bd = Accent, Ink
	}
	t := row.AnimateWith("lift", map[bool]float32{true: -2, false: 0}[hot], dRow, EaseSpring)
	row.Background(row.AnimateColor("bg", bg, ui.Animation{Duration: 100e6})).Border(strokeW, row.AnimateColor("bd", bd, ui.Animation{Duration: 100e6}))
	row.Translate(t, t+6*(1-in)).Opacity(in)
	hard(row, max(0, -t*1.5))
	if row.Hovered() {
		w.rowHov = r.id
	}
	row.Children(func() {
		main := ui.ButtonBase(c.Key("apply:"+r.id)).Grow(1).MinWidth(0).FillHeight().Padding(0, 6, 0, 8).Gap(10).AlignItems(ui.Center).Justify(ui.Start).Label(r.name)
		focusRing(main)
		main.Children(func() {
			tile := ui.Box(c).Size(26, 26).Radius(radiusThumb).Background(r.tile).Border(strokeW, Ink).Center()
			tile.Children(func() {
				if r.chain {
					ui.Icon(c, ic("workflow")).FontSize(13).TextColor(Ink)
				} else {
					ui.Text(c, r.glyph).Font(monoFont).FontSize(12).FontWeight(700).TextColor(Ink).SingleLine()
				}
			})
			nameCol := Ink
			if !res.changed() {
				nameCol = Faint
			}
			ui.Text(c, r.name).FontSize(sizeBody).FontWeight(500).TextColor(nameCol).SingleLine().Grow(1).MinWidth(0)
			d := res.delta
			switch {
			case d == "":
				ui.Text(c, "无变化").Font(monoFont).FontSize(11).TextColor(Faint).SingleLine()
			case hot:
				ui.Text(c, d).Font(monoFont).FontSize(11).TextColor(Ink).Background(Paper).Border(strokeW, Ink).Radius(radiusPill).Padding(0, 7).SingleLine()
			default:
				ui.Text(c, d).Font(monoFont).FontSize(11).TextColor(White).Background(Ink).Border(strokeW, Ink).Radius(radiusPill).Padding(0, 7).SingleLine()
			}
		})
		if main.Clicked() {
			a.applyAction(r.id)
		}
		tip := "加入常用"
		if pinned {
			tip = "从常用移除"
		}
		star := ui.ButtonBase(c.Key("star:"+r.id)).Size(26, 26).Radius(radiusThumb).Center().Label(tip).Tooltip(tip)
		so := star.AnimateWith("show", map[bool]float32{true: 1, false: 0}[hot], 240*time.Millisecond, EaseSpring)
		star.Opacity(min(1, so)).Rotate(-30*(1-so)).Scale(0.5+0.5*so, 0.5+0.5*so)
		focusRing(star)
		star.Children(func() {
			icon := "star"
			if pinned {
				icon = "star-filled"
			}
			ui.Icon(c, ic(icon)).FontSize(15).TextColor(Ink)
		})
		if star.Clicked() {
			a.togglePin(r.id)
		}
	})
}

// startApplyFx begins the apply animation from before to after.
func (w *workspace) startApplyFx(before, after string) {
	if engine.RuneLen(before)+engine.RuneLen(after) > 200000 {
		return
	}
	pv := diffPreview("fx", cachedDiff(before, after))
	w.applyFx = &applyFx{pv: pv, at: time.Now()}
	w.pvShown = ""
}
