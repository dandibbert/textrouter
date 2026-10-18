package gui

import (
	"bytes"
	"embed"
	"fmt"
	"image"
	"image/png"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// Spacing scale and the legacy names the pages use. Colours, shadows,
// strokes and curves come from theme_brutal.go.
const (
	sp1 = 4
	sp2 = 8
	sp3 = 12
	sp4 = 16
	sp5 = 20
	sp6 = 24
	sp8 = 32

	rSm = radiusCtl
	rMd = radiusCtl
	rLg = radiusCtl
	rXl = radiusCtl

	sidebarW = 224
	uiFont   = fontSans + ", " + fontCJK

	editorFont = fontMono + ", " + fontCJK
	monoFont   = fontMono + ", " + fontCJK + ", monospace"
)

// tokens are the semantic colours the pages ask for, all drawn from the
// brutal palette. The look has one appearance: the system's light or dark
// setting does not change it.
type tokens struct {
	dark                         bool
	window, sidebar, canvas      ui.Color
	field, fieldHover            ui.Color
	hair, hairStrong             ui.Color
	text, muted, faint           ui.Color
	hover, press, pill           ui.Color
	accent, accentHi, accentSoft ui.Color
	accentText                   ui.Color
	green, greenSoft             ui.Color
	red, redSoft                 ui.Color
	amber, amberSoft             ui.Color
	shadow, shadowNear           ui.Color
}

func tok(c *ui.Context) tokens {
	return tokens{
		window: Ground, sidebar: Ground, canvas: Paper,
		field: Paper, fieldHover: AccentWash,
		hair: Ink, hairStrong: Ink,
		text: Ink, muted: Muted, faint: Faint,
		hover: AccentSoft, press: Accent, pill: AccentSoft,
		// Emphasis is Ink on a yellow fill: accent is the colour of text and
		// icons, Accent itself is used directly for fills.
		accent: Ink, accentHi: Ink, accentSoft: AccentSoft, accentText: Ink,
		green: Ink, greenSoft: Ins,
		red: Danger, redSoft: Del,
		amber: Ink, amberSoft: Hit,
		shadow: Ink, shadowNear: Ink,
	}
}

// applyTheme sets the widget theme from the tokens so that MyGo's own
// controls (selects, menus, checkboxes, toasts) match.
func applyTheme(c *ui.Context) tokens {
	k := tok(c)
	t := *c.Theme()
	t.Dark = false
	t.Background = Ground
	t.Surface = Paper
	t.SurfaceHover = AccentSoft
	t.SurfacePressed = Accent
	t.Border = Ink
	t.Text, t.TextMuted = Ink, Muted
	t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = Accent, AccentHover, AccentDeep, Ink
	t.Danger, t.Warning, t.Success = Danger, Ink, Ink
	t.Selection = Accent
	t.Focus = Focus
	t.Radius = radiusCtl
	t.FontSize = sizeBody
	t.Font = uiFontFamily
	t.Inverse, t.InverseText = Ink, White
	t.Scrollbar, t.ScrollbarWidth = Accent, scrollW
	t.ScrollbarOutline, t.ScrollbarHover, t.ScrollbarDrag, t.ScrollbarMinThumb = Ink, AccentDeep, Ink, 44
	c.SetTheme(&t)
	return k
}

//go:embed icons/*.svg
var iconFS embed.FS

//go:embed fonts/Inter.ttf
var interTTF []byte

//go:embed fonts/SpaceGrotesk.ttf
var spaceGroteskTTF []byte

//go:embed fonts/SpaceMono-Regular.ttf
var spaceMonoTTF []byte

//go:embed fonts/SpaceMono-Bold.ttf
var spaceMonoBoldTTF []byte

var (
	iconMu    sync.Mutex
	iconCache = map[string]*ui.SVG{}
	fontOnce  sync.Once
)

// registerFonts bundles the OFL fonts of the design: Space Grotesk for the
// interface, Space Mono for the text, and Inter, still offered in settings.
func registerFonts() {
	fontOnce.Do(func() {
		_ = ui.RegisterFont(interTTF, "Inter")
		_ = ui.RegisterFont(spaceGroteskTTF, fontSans)
		_ = ui.RegisterFont(spaceMonoTTF, fontMono)
		_ = ui.RegisterFont(spaceMonoBoldTTF, fontMono)
	})
}

// ic returns a bundled Lucide icon (ISC licence) by name.
func ic(name string) *ui.SVG {
	iconMu.Lock()
	defer iconMu.Unlock()
	if s, ok := iconCache[name]; ok {
		return s
	}
	data, err := iconFS.ReadFile("icons/" + name + ".svg")
	if err != nil {
		data, _ = iconFS.ReadFile("icons/sparkles.svg")
	}
	s := ui.MustParseSVG(data)
	iconCache[name] = s
	return s
}

// ---------------------------------------------------------------- atoms

var quick = ui.Animation{Duration: dRow, Ease: EaseSpring}

// lift gives a bordered button its press behaviour (§6 "按钮按下 / 松开"):
// resting at 3px of hard shadow, hover lifts it 1px and grows the shadow to
// 4px, pressing moves it 3px and drops the shadow to 0 in 60ms, and letting
// go springs back in 160ms.
func lift(b ui.Element, base float32) ui.Element {
	if b.IsDisabled() {
		return b.Opacity(0.4).Shadow(1, 1, 0, 0, Ink)
	}
	target, dur := float32(0), dRelease
	switch {
	case b.Pressed():
		target, dur = 3, dPress
	case b.Hovered():
		target = -1
	}
	t := b.AnimateWith("lift", target, dur, EaseSpring)
	b.Translate(t, t)
	hard(b, max(0, base-t))
	return focusRing(b)
}

// pressable gives a flat ButtonBase the shared hover/press/focus treatment:
// a soft yellow wash on hover, solid yellow while pressed.
func pressable(c *ui.Context, b ui.Element, k tokens, idle ui.Color) ui.Element {
	bg := idle
	switch {
	case b.IsDisabled():
	case b.Pressed():
		bg = Accent
	case b.Hovered():
		bg = AccentSoft
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6}))
	if b.IsDisabled() {
		b.Opacity(0.4)
	}
	return focusRing(b)
}

// iconButton is a square, borderless button that shows a white box and a
// stroke on hover; it has a tooltip.
func iconButton(c *ui.Context, key, icon, label, tip string) ui.Element {
	b := ui.ButtonBase(c.Key(key)).Size(34, 34).Radius(radiusCtl).Center().Label(label).Tooltip(tip)
	bg, bd := ui.Transparent, ui.Transparent
	if !b.IsDisabled() && b.Hovered() {
		bg, bd = Paper, Ink
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6})).Border(strokeW, bd)
	s := b.AnimateTo("s", map[bool]float32{true: 0.9, false: 1}[b.Pressed()], quick)
	b.Scale(s, s)
	if b.IsDisabled() {
		b.Opacity(0.3)
	}
	focusRing(b)
	b.Children(func() { ui.Icon(c, ic(icon)).FontSize(16).TextColor(Ink) })
	return b
}

// chip is a compact bordered button with an icon and a label (pinned actions).
func chip(c *ui.Context, key, icon, label, tip string) ui.Element {
	b := ui.ButtonBase(c.Key(key)).Height(30).Padding(0, 11, 0, 9).Radius(radiusCtl).Gap(6).AlignItems(ui.Center).Label(label).Tooltip(tip)
	b.Background(Paper)
	boxed(b)
	lift(b, shadowSm)
	b.Children(func() {
		ui.Icon(c, ic(icon)).FontSize(14).TextColor(Ink)
		ui.Text(c, label).FontSize(sizeButton).FontWeight(500).SingleLine().TextColor(Ink)
	})
	return b
}

// primary is the one yellow button of a screen, with an optional shortcut.
func primary(c *ui.Context, key, label, icon, shortcut string, disabled bool) ui.Element {
	b := ui.ButtonBase(c.Key(key)).Height(34).Padding(0, 14, 0, 12).Radius(radiusCtl).Gap(7).AlignItems(ui.Center).Label(label).Disabled(disabled)
	if icon == "" {
		b.Padding(0, 14)
	}
	bg := Accent
	if !disabled && b.Hovered() {
		bg = AccentHover
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6}))
	boxed(b)
	lift(b, shadowSm)
	b.Children(func() {
		if icon != "" {
			ui.Icon(c, ic(icon)).FontSize(15).TextColor(Ink)
		}
		ui.Text(c, label).FontSize(13).FontWeight(700).SingleLine().TextColor(Ink)
		if shortcut != "" {
			kbd(c, shortcut)
		}
	})
	return b
}

// ghost is a secondary text button: white, stroked, with a hard shadow.
func ghost(c *ui.Context, key, label, icon string, disabled bool) ui.Element {
	b := ui.ButtonBase(c.Key(key)).Height(32).Padding(0, 12).Radius(radiusCtl).Gap(6).AlignItems(ui.Center).Label(label).Disabled(disabled)
	if icon != "" {
		b.Padding(0, 12, 0, 10)
	}
	bg := Paper
	if !disabled && b.Hovered() {
		bg = AccentWash
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6}))
	boxed(b)
	lift(b, shadowSm)
	b.Children(func() {
		if icon != "" {
			ui.Icon(c, ic(icon)).FontSize(14).TextColor(Ink)
		}
		ui.Text(c, label).FontSize(sizeButton).FontWeight(500).SingleLine().TextColor(Ink)
	})
	return b
}

// kbd draws a keycap: 1.5px stroke, 3px bottom edge, 10px mono type.
func kbd(c *ui.Context, s string) ui.Element {
	return ui.Text(c, s).Font(monoFont).FontSize(10).TextColor(Ink).SingleLine().
		Padding(0, 5).Radius(radiusKey).Background(Paper).
		BorderWidth(keycapW, keycapW, keycapBot, keycapW).BorderColor(Ink)
}

type tone int

const (
	toneNeutral tone = iota
	toneAccent
	toneGreen
	toneAmber
	toneRed
)

// badge is a small stroked capsule.
func badge(c *ui.Context, s string, tn tone) ui.Element {
	bg := Paper
	switch tn {
	case toneAccent:
		bg = Accent
	case toneGreen:
		bg = Ins
	case toneAmber:
		bg = Hit
	case toneRed:
		bg = Del
	}
	return ui.Text(c, s).FontSize(sizeSmall).FontWeight(700).Font(monoFont).TextColor(Ink).Background(bg).
		Radius(radiusPill).Border(strokeW, Ink).Padding(0, 7).SingleLine().FontFeatures("tnum")
}

// eyebrow is a small uppercase section label.
func eyebrow(c *ui.Context, s string) ui.Element {
	return ui.Text(c, strings.ToUpper(s)).FontSize(sizeSmall).FontWeight(700).TextColor(Ink).LetterSpacing(0.88)
}

// panel is a raised card: white, stroked, with a 5px hard shadow.
func panel(c *ui.Context) ui.Element {
	return hard(boxed(ui.Column(c).Background(Paper).Radius(radiusCtl).Padding(sp4).Gap(sp3)), shadowMd)
}

// canvasCard is the large editor surface.
func canvasCard(c *ui.Context) ui.Element {
	return hard(boxed(ui.Column(c).Background(Paper).Radius(radiusCtl)), shadowMd)
}

// field restyles a MyGo text input to the stroked, hard-shadowed look; its
// shadow grows and it lifts 2px while focused.
func field(c *ui.Context, e ui.Element) ui.Element {
	e.Background(Paper).Radius(radiusCtl).Padding(7, 10).FontSize(sizeBody)
	boxed(e)
	if e.Focused() {
		e.Translate(-2, -2)
		hard(e, shadowMd)
	} else {
		hard(e, shadowSm)
	}
	return focusRing(e)
}

// pageHeader is the toolbar row at the top of every page; it doubles as
// the window's drag region under the hidden title bar.
func pageHeader(c *ui.Context, title, subtitle string, actions func()) {
	ui.Row(c).Height(titleBarH).Padding(0, sp5, 0, sp3).Gap(sp2).AlignItems(ui.Center).
		BorderWidth(0, 0, strokeW, 0).BorderColor(Ink).Background(Ground).DragWindow().Children(func() {
		if headerLead != nil {
			headerLead()
		}
		ui.Text(c, title).FontSize(16).FontWeight(700).TextColor(Ink).SingleLine().LetterSpacing(-0.3)
		if subtitle != "" {
			ui.Text(c, subtitle).FontSize(12).TextColor(Muted).SingleLine().MinWidth(0).Shrink(1).Margin(0, 0, 0, sp1)
		}
		ui.Spacer(c)
		if actions != nil {
			actions()
		}
	})
}

// headerLead draws what precedes a page title (sidebar toggle, room for
// the window controls); set by the App each frame.
var headerLead func()

// hint is a muted helper line.
func hint(c *ui.Context, s string) ui.Element {
	return ui.Text(c, s).FontSize(12).TextColor(Muted).LineHeight(1.45)
}

func cut(s string, n int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return string(r)
}

//go:embed assets/icon64.png
var appIconPNG []byte

var appIconBmp = sync.OnceValue(func() *ui.Bitmap {
	img, err := png.Decode(bytes.NewReader(appIconPNG))
	if err != nil {
		return ui.NewBitmap(image.NewRGBA(image.Rect(0, 0, 1, 1)))
	}
	return ui.NewBitmap(img)
})

// segmented is a stroked switch: the chosen label sits on a yellow box.
func segmented(c *ui.Context, key string, sel *int, labels ...string) bool {
	changed := false
	hard(boxed(ui.Row(c.Key(key)).Padding(3).Gap(2).Radius(radiusSeg).Background(Paper)), shadowSm).Children(func() {
		for i, l := range labels {
			b := ui.ButtonBase(c.Key(fmt.Sprint(key, i))).Height(28).Padding(0, 12).Radius(radiusThumb).Center().Label(l).Selected(*sel == i)
			bg, bd := ui.Transparent, ui.Transparent
			switch {
			case *sel == i:
				bg, bd = Accent, Ink
			case b.Hovered():
				bg = AccentSoft
			}
			b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6})).Border(strokeW, bd)
			focusRing(b)
			b.Children(func() { ui.Text(c, l).FontSize(12.5).FontWeight(700).TextColor(Ink).SingleLine() })
			if b.Clicked() && *sel != i {
				*sel, changed = i, true
			}
		}
	})
	return changed
}

// flag is a capsule toggle (Aa, .*, ^$): white at rest, soft yellow on hover,
// Ink with yellow type when on.
func flag(c *ui.Context, key, text, tip string, on *bool) bool {
	b := ui.ButtonBase(c.Key(key)).Height(28).MinWidth(32).Padding(0, 10).Radius(radiusPill).Center().Label(tip).Tooltip(tip).Selected(*on)
	bg, col := Paper, Ink
	switch {
	case *on:
		bg, col = Ink, Accent
	case b.Hovered():
		bg = AccentSoft
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 140e6})).Border(strokeW, Ink)
	s := b.AnimateWith("press", map[bool]float32{true: 0.93, false: 1}[b.Pressed()], dRelease, EaseSpring)
	b.Scale(s, s)
	focusRing(b)
	b.Children(func() { ui.Text(c, text).FontSize(12).FontWeight(700).Font(monoFont).TextColor(col).SingleLine() })
	if b.Clicked() {
		*on = !*on
		return true
	}
	return false
}

// tokenColor maps an explanation part kind to its colour: type is Ink
// throughout, only errors stand out.
func tokenColor(k tokens, kind string) ui.Color {
	if kind == "error" {
		return Danger
	}
	return Ink
}
