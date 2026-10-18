package gui

import (
	"fmt"

	"github.com/egoist/mygo/ui"
)

// The app's menus are drawn in the app (a popover under the button) instead of
// as native menus, so that they share the look of everything else: a white
// card, a 2px stroke, a 6px hard shadow, yellow rows under the pointer.

// pm is the menu being built: the same Item / Separator calls as ui.Menu.
type pm struct {
	c    *ui.Context
	open *bool
	n    int
}

// pmItem is one menu row; Chosen reports that it was clicked in this frame.
type pmItem struct{ chosen bool }

func (i pmItem) Chosen() bool { return i.chosen }

// Item adds a row.
func (m *pm) Item(label string) pmItem {
	m.n++
	c := m.c
	b := ui.ButtonBase(c.Key("pmi-"+label)).FillWidth().Height(32).Padding(0, 10).Radius(radiusCtl).AlignItems(ui.Center).Justify(ui.Start).Label(label)
	bg, bd := ui.Transparent, ui.Transparent
	if b.Hovered() {
		bg, bd = Accent, Ink
	}
	b.Background(bg).Border(strokeW, bd)
	focusRing(b)
	b.Children(func() { ui.Text(c, label).FontSize(sizeButton).FontWeight(500).TextColor(Ink).SingleLine() })
	if b.Clicked() {
		*m.open = false
		return pmItem{chosen: true}
	}
	return pmItem{}
}

// Separator adds a 2px rule.
func (m *pm) Separator() {
	m.n++
	ui.Box(m.c.Key(fmt.Sprint("pms", m.n))).FillWidth().Height(strokeW).Background(Ink).Margin(4, 2)
}

// menuFlag is the open state of the menu with this key.
func (a *App) menuFlag(key string) *bool {
	if a.menus == nil {
		a.menus = map[string]*bool{}
	}
	f, ok := a.menus[key]
	if !ok {
		f = new(bool)
		a.menus[key] = f
	}
	return f
}

// popMenu opens a menu under anchor when it is clicked; build fills it with
// Item and Separator, as for ui.Menu. Escape or a click outside closes it.
func (a *App) popMenu(c *ui.Context, key string, anchor ui.Element, build func(m *pm)) {
	a.menuAt(c, key, anchor, true, build)
}

// ctxMenu is a menu that the caller opens (by setting menuFlag(key), as a
// right click does) and that clicks on the anchor do not toggle.
func (a *App) ctxMenu(c *ui.Context, key string, anchor ui.Element, build func(m *pm)) {
	a.menuAt(c, key, anchor, false, build)
}

func (a *App) menuAt(c *ui.Context, key string, anchor ui.Element, toggle bool, build func(m *pm)) {
	open := a.menuFlag(key)
	was := *open
	ui.PopoverBase(c, anchor, open, func(panel ui.Element) {
		panel.Padding(6).MinWidth(220).Background(Paper).Radius(radiusSeg).Border(strokeW, Ink)
		hard(panel, shadowLg)
		panel.Children(func() { build(&pm{c: c, open: open}) })
	})
	if toggle && anchor.Clicked() {
		*open = !was
	}
}
