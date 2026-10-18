package gui

import (
	"fmt"
	"time"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

// ---------------------------------------------------------------- step trail

// step is one entry of the trail shown in the title bar: the text after it
// and, when it was a cleaner or a find/replace, the chain step that redoes it.
type step struct {
	label string
	text  string
	act   *engine.Step
	at    time.Time
}

// trail is the document's history: 原文 → step → step. pos is the step the
// text is at; steps after it were undone and can be redone.
type trail struct {
	steps []step
	pos   int
}

const maxTrail = 200

// trail returns the document's trail, starting it from the current text.
func (a *App) trail() *trail {
	if a.tr == nil {
		a.syncEditor()
		a.tr = &trail{steps: []step{{label: "原文", text: a.Text}}}
	}
	return a.tr
}

// catchUp records edits typed since the last step as a step of their own, so
// that an action after typing can be undone back to the typing.
func (a *App) catchUp() {
	t := a.trail()
	a.syncEditor()
	if a.Text != t.steps[t.pos].text {
		t.steps = append(t.steps[:t.pos+1], step{label: "编辑", text: a.Text, at: time.Now()})
		t.pos++
	}
}

// pushStep records the text as it is now as the next step.
func (a *App) pushStep(label string, act *engine.Step) {
	t := a.trail()
	t.steps = append(t.steps[:t.pos+1], step{label: label, text: a.Text, act: act, at: time.Now()})
	t.pos++
	if len(t.steps) > maxTrail {
		t.steps = t.steps[len(t.steps)-maxTrail:]
		t.pos = len(t.steps) - 1
	}
}

// goStep sets the text to step i of the trail, keeping the steps around it.
func (a *App) goStep(i int) {
	a.catchUp()
	t := a.trail()
	if i < 0 || i >= len(t.steps) || i == t.pos {
		return
	}
	t.pos = i
	a.replaceAll(t.steps[i].text)
	a.clearDiffBase()
	a.changes = nil
	a.last = nil
	a.ws.selLines, a.ws.applyFx = nil, nil
}

// CanUndo and CanRedo tell whether there is a step to go back or forward to.
func (a *App) CanUndo() bool {
	t := a.trail()
	return t.pos > 0 || a.Text != t.steps[t.pos].text
}

func (a *App) CanRedo() bool {
	t := a.trail()
	return t.pos < len(t.steps)-1 && a.Text == t.steps[t.pos].text
}

// Undo goes back one step.
func (a *App) Undo() {
	a.catchUp()
	t := a.trail()
	if t.pos == 0 {
		return
	}
	a.goStep(t.pos - 1)
	a.status = "已撤销"
}

// Redo goes forward one step.
func (a *App) Redo() {
	a.catchUp()
	t := a.trail()
	if t.pos >= len(t.steps)-1 {
		return
	}
	a.goStep(t.pos + 1)
	a.status = "已重做"
}

// saveTrailAsChain turns the cleaner and find steps up to the current one into
// a new action chain, and says so in a toast.
func (a *App) saveTrailAsChain() {
	t := a.trail()
	var steps []engine.Step
	for _, s := range t.steps[1 : t.pos+1] {
		if s.act != nil {
			steps = append(steps, *s.act)
		}
	}
	if len(steps) < 2 {
		return
	}
	name := fmt.Sprintf("新动作链 %d", len(a.Lib.Chains)+1)
	a.Lib.Chains = append(a.Lib.Chains, engine.Chain{ID: engine.NewID(), Name: name, Steps: steps})
	a.save()
	a.showToast("已存为动作链「"+name+"」", fmt.Sprintf("%d 步 · 在“动作链”分组里", len(steps)))
}

// ---------------------------------------------------------------- toast

type toastState struct {
	text, sub string
	at        time.Time
}

// showToast pops the black toast up from the window's bottom.
func (a *App) showToast(text, sub string) {
	a.ws.toast = toastState{text: text, sub: sub, at: time.Now()}
}

// toastView draws the toast: it springs up in 340ms, stays 2s and sinks away
// in 220ms.
func (a *App) toastView(c *ui.Context) {
	if a.toast != "" {
		a.showToast(a.toast, "")
		a.toast = ""
	}
	t := &a.ws.toast
	if t.text == "" {
		return
	}
	el := time.Since(t.at)
	total := dToastIn + dToastHold + dToastOut
	if el >= total {
		*t = toastState{}
		return
	}
	c.Invalidate()
	y, alpha := float32(0), float32(1)
	still := c.Preferences().ReduceMotion
	switch {
	case still:
	case el < dToastIn:
		p := EaseSpring(float32(el) / float32(dToastIn))
		y, alpha = 28*(1-p), min(1, float32(el)/float32(dToastIn)*3)
	case el > dToastIn+dToastHold:
		p := EaseIn(float32(el-dToastIn-dToastHold) / float32(dToastOut))
		y, alpha = 18*p, 1-p
	}
	ui.Overlay(c, func() {
		ui.Column(c.Key("toast-host")).Absolute().Top(0).Left(0).Right(0).Bottom(0).AlignItems(ui.Center).Justify(ui.End).
			PassThrough().Padding(0, 0, 22, 0).Children(func() {
			row := ui.Row(c.Key("toast")).Gap(12).AlignItems(ui.Center).Padding(10, 14, 10, 12).Radius(radiusSeg).Background(Ink).Border(strokeW, Ink)
			row.Shadow(4, 4, 0, 0, Accent).Translate(0, y).Opacity(alpha)
			row.Children(func() {
				ui.Box(c).Size(22, 22).Radius(11).Background(Accent).Center().Children(func() {
					ui.Icon(c, ic("check")).FontSize(12).TextColor(Ink)
				})
				ui.Text(c, t.text).FontSize(13).FontWeight(700).TextColor(White).SingleLine()
				if t.sub != "" {
					ui.Text(c, t.sub).FontSize(12).TextColor(Faint).SingleLine()
				}
			})
		})
	})
}

// ---------------------------------------------------------------- timing

// anim returns how far an event animation that began at t0 and lasts d has
// come, 0 to 1, eased; it asks for frames while it runs. With the system's
// "reduce motion" on, it is done at once.
func anim(c *ui.Context, t0 time.Time, d time.Duration, ease ui.Easing) float32 {
	if t0.IsZero() || c.Preferences().ReduceMotion {
		return 1
	}
	p := float32(time.Since(t0)) / float32(d)
	if p >= 1 {
		return 1
	}
	c.Invalidate()
	if p < 0 {
		return 0
	}
	if ease == nil {
		return p
	}
	return ease(p)
}

// workspace is the clipboard window's own state (title bar, right column and
// the text's overlays), apart from the text and editor in App.
type workspace struct {
	tab    int // 0 清理, 1 查找替换
	tabAt  time.Time
	tabDir float32

	query string // the cleaner search box
	hov   string // the action the pointer is on (its preview shows), "" none
	hovAt time.Time

	textKey uint64         // identifies the text for the caches
	cache   map[string]any // per-text results: action diffs, matches, previews
	pvShown string
	pvAt    time.Time

	pv       *preview // what the body shows instead of the text: a preview
	pvEd     pvEditor
	applyFx  *applyFx
	selLines map[int]bool

	toast    toastState
	copiedAt time.Time

	find findUI

	frameHov string // the action hovered in this frame's build
	rowHov   string // the row the pointer is on
	headHov  bool   // the pointer is on a section heading
	focusFnd bool   // ⌘F: put the caret in the find box
}
