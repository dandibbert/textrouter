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

// Design tokens. Everything visual in the app comes from here: a 4-pt
// spacing scale, three radii, one accent, a neutral ramp for light and
// dark, and two type families (UI chrome and the editor).

const (
	sp1 = 4
	sp2 = 8
	sp3 = 12
	sp4 = 16
	sp5 = 20
	sp6 = 24
	sp8 = 32

	rSm = 6
	rMd = 8
	rLg = 12
	rXl = 14

	sidebarW   = 224
	titleBarH  = 52
	uiFont     = "Inter, PingFang SC, Hiragino Sans GB, Noto Sans CJK SC, Microsoft YaHei"
	editorFont = "Inter, PingFang SC, Hiragino Sans GB, Noto Sans CJK SC"
	monoFont   = "SF Mono, JetBrains Mono, Menlo, Noto Sans Mono CJK SC, monospace"
)

// tokens are the semantic colors of the current appearance.
type tokens struct {
	dark                         bool
	window, sidebar, canvas      ui.Color // window bg, sidebar fill (no vibrancy), raised card
	field, fieldHover            ui.Color // inputs
	hair, hairStrong             ui.Color // 1px separators
	text, muted, faint           ui.Color
	hover, press, pill           ui.Color // neutral interaction fills
	accent, accentHi, accentSoft ui.Color
	accentText                   ui.Color
	green, greenSoft             ui.Color
	red, redSoft                 ui.Color
	amber, amberSoft             ui.Color
	shadow, shadowNear           ui.Color
}

var (
	accentL = ui.Hex("#5E5CE6") // indigo, close to macOS systemIndigo
	accentD = ui.Hex("#7D7AFF")
)

func tok(c *ui.Context) tokens {
	if c.Theme().Dark {
		return tokens{
			dark:   true,
			window: ui.Hex("#18181B"), sidebar: ui.Hex("#1F1F23"), canvas: ui.Hex("#222226"),
			field: ui.RGBA(255, 255, 255, 0.05), fieldHover: ui.RGBA(255, 255, 255, 0.08),
			hair: ui.RGBA(255, 255, 255, 0.07), hairStrong: ui.RGBA(255, 255, 255, 0.12),
			text: ui.Hex("#ECECEE"), muted: ui.Hex("#9B9BA3"), faint: ui.Hex("#6B6B73"),
			hover: ui.RGBA(255, 255, 255, 0.06), press: ui.RGBA(255, 255, 255, 0.10), pill: ui.RGBA(255, 255, 255, 0.09),
			accent: accentD, accentHi: ui.Hex("#8F8CFF"), accentSoft: accentD.Alpha(0.18), accentText: ui.Hex("#FFFFFF"),
			green: ui.Hex("#4ADE80"), greenSoft: ui.RGBA(74, 222, 128, 0.16),
			red: ui.Hex("#F87171"), redSoft: ui.RGBA(248, 113, 113, 0.16),
			amber: ui.Hex("#FBBF24"), amberSoft: ui.RGBA(251, 191, 36, 0.18),
			shadow: ui.RGBA(0, 0, 0, 0.45), shadowNear: ui.RGBA(0, 0, 0, 0.35),
		}
	}
	return tokens{
		window: ui.Hex("#F6F6F7"), sidebar: ui.Hex("#EDEDF0"), canvas: ui.Hex("#FFFFFF"),
		field: ui.RGBA(0, 0, 0, 0.035), fieldHover: ui.RGBA(0, 0, 0, 0.055),
		hair: ui.RGBA(0, 0, 0, 0.07), hairStrong: ui.RGBA(0, 0, 0, 0.11),
		text: ui.Hex("#1D1D1F"), muted: ui.Hex("#6E6E76"), faint: ui.Hex("#A1A1A8"),
		hover: ui.RGBA(0, 0, 0, 0.045), press: ui.RGBA(0, 0, 0, 0.08), pill: ui.RGBA(0, 0, 0, 0.065),
		accent: accentL, accentHi: ui.Hex("#6D6BF0"), accentSoft: accentL.Alpha(0.11), accentText: ui.Hex("#FFFFFF"),
		green: ui.Hex("#15803D"), greenSoft: ui.RGBA(34, 197, 94, 0.16),
		red: ui.Hex("#DC2626"), redSoft: ui.RGBA(239, 68, 68, 0.13),
		amber: ui.Hex("#B45309"), amberSoft: ui.RGBA(245, 158, 11, 0.18),
		shadow: ui.RGBA(15, 15, 30, 0.08), shadowNear: ui.RGBA(15, 15, 30, 0.05),
	}
}

// applyTheme sets the widget theme from the tokens so that MyGo's own
// controls (selects, menus, checkboxes, toasts) match.
func applyTheme(c *ui.Context) tokens {
	k := tok(c)
	t := *c.Theme()
	t.Background = k.window
	t.Surface = k.canvas
	t.SurfaceHover = k.canvas.Mix(k.text, 0.04)
	t.SurfacePressed = k.canvas.Mix(k.text, 0.08)
	t.Border = k.hairStrong
	t.Text, t.TextMuted = k.text, k.muted
	t.Accent, t.AccentHover, t.AccentPressed, t.AccentText = k.accent, k.accentHi, k.accent.Mix(ui.Hex("#000000"), 0.12), k.accentText
	t.Danger, t.Warning, t.Success = k.red, k.amber, k.green
	t.Selection = k.accent.Alpha(0.24)
	t.Focus = k.accent.Alpha(0.55)
	t.Radius = rMd
	t.FontSize = 13
	t.Font = uiFont
	t.ScrollbarWidth = 6
	if k.dark {
		t.Inverse, t.InverseText = ui.Hex("#2C2C31"), ui.Hex("#F4F4F5")
	} else {
		t.Inverse, t.InverseText = ui.Hex("#1D1D1F"), ui.Hex("#F5F5F7")
	}
	c.SetTheme(&t)
	return k
}

//go:embed icons/*.svg
var iconFS embed.FS

//go:embed fonts/Inter.ttf
var interTTF []byte

var (
	iconMu    sync.Mutex
	iconCache = map[string]*ui.SVG{}
	fontOnce  sync.Once
)

func registerFonts() {
	fontOnce.Do(func() { _ = ui.RegisterFont(interTTF, "Inter") })
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

var quick = ui.Animation{Spring: ui.SpringOptions{Duration: 220e6, Bounce: 0.15}}

// pressable gives a ButtonBase the shared hover/press/focus treatment.
func pressable(c *ui.Context, b ui.Element, k tokens, idle ui.Color) ui.Element {
	bg := idle
	switch {
	case b.Pressed():
		bg = k.press
	case b.Hovered():
		bg = k.hover
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6}))
	s := b.AnimateTo("s", map[bool]float32{true: 0.96, false: 1}[b.Pressed()], quick)
	b.Scale(s, s)
	if b.FocusVisible() {
		b.Shadow(0, 0, 0, 3, k.accent.Alpha(0.35))
	}
	return b
}

// iconButton is a square, borderless toolbar button with a tooltip.
func iconButton(c *ui.Context, key, icon, label, tip string) ui.Element {
	k := tok(c)
	b := ui.ButtonBase(c.Key(key)).Size(30, 30).Radius(rMd).Center().Label(label).Tooltip(tip)
	pressable(c, b, k, ui.Transparent)
	b.Children(func() { ui.Icon(c, ic(icon)).FontSize(16).TextColor(k.muted) })
	return b
}

// chip is a compact pill with an icon and a label (pinned actions).
func chip(c *ui.Context, key, icon, label, tip string) ui.Element {
	k := tok(c)
	b := ui.ButtonBase(c.Key(key)).Height(30).Padding(0, 11, 0, 9).Radius(rMd).Gap(6).AlignItems(ui.Center).Label(label).Tooltip(tip)
	pressable(c, b, k, k.field)
	b.Children(func() {
		ui.Icon(c, ic(icon)).FontSize(14).TextColor(k.muted)
		ui.Text(c, label).FontSize(12.5).FontWeight(500).SingleLine().TextColor(k.text)
	})
	return b
}

// primary is the one accent button of a screen, with an optional shortcut.
func primary(c *ui.Context, key, label, icon, shortcut string, disabled bool) ui.Element {
	k := tok(c)
	b := ui.ButtonBase(c.Key(key)).Height(32).Padding(0, 14, 0, 12).Radius(rMd).Gap(7).AlignItems(ui.Center).Label(label).Disabled(disabled)
	if icon == "" {
		b.Padding(0, 14)
	}
	if disabled {
		b.Background(k.pill).Children(func() {
			if icon != "" {
				ui.Icon(c, ic(icon)).FontSize(15).TextColor(k.faint)
			}
			ui.Text(c, label).FontSize(13).FontWeight(600).SingleLine().TextColor(k.faint)
		})
		return b
	}
	bg := k.accent
	if b.Hovered() && !disabled {
		bg = k.accentHi
	}
	if b.Pressed() {
		bg = k.accent.Mix(ui.Hex("#000000"), 0.12)
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6})).
		Shadow(0, 1, 2, 0, k.accent.Alpha(0.35)).Shadow(0, 4, 12, -2, k.accent.Alpha(0.35)).
		InsetShadow(0, 1, 0, 0, ui.RGBA(255, 255, 255, 0.18))
	s := b.AnimateTo("s", map[bool]float32{true: 0.97, false: 1}[b.Pressed()], quick)
	b.Scale(s, s)
	if disabled {
		b.Opacity(0.45)
	}
	if b.FocusVisible() {
		b.Shadow(0, 0, 0, 3, k.accent.Alpha(0.35))
	}
	b.Children(func() {
		if icon != "" {
			ui.Icon(c, ic(icon)).FontSize(15).TextColor(k.accentText)
		}
		ui.Text(c, label).FontSize(13).FontWeight(600).SingleLine().TextColor(k.accentText)
		if shortcut != "" {
			ui.Text(c, shortcut).FontSize(11.5).FontWeight(500).TextColor(ui.RGBA(255, 255, 255, 0.72)).SingleLine()
		}
	})
	return b
}

// ghost is a secondary text button: quiet fill, hover brighten.
func ghost(c *ui.Context, key, label, icon string, disabled bool) ui.Element {
	k := tok(c)
	b := ui.ButtonBase(c.Key(key)).Height(30).Padding(0, 12).Radius(rMd).Gap(6).AlignItems(ui.Center).Label(label).Disabled(disabled)
	if icon != "" {
		b.Padding(0, 12, 0, 10)
	}
	pressable(c, b, k, k.field)
	if disabled {
		b.Opacity(0.45)
	}
	b.Children(func() {
		if icon != "" {
			ui.Icon(c, ic(icon)).FontSize(14).TextColor(k.muted)
		}
		ui.Text(c, label).FontSize(12.5).FontWeight(500).SingleLine().TextColor(k.text)
	})
	return b
}

// kbd draws a keycap hint.
func kbd(c *ui.Context, s string) ui.Element {
	k := tok(c)
	return ui.Text(c, s).FontSize(11).FontWeight(500).TextColor(k.muted).SingleLine().
		Padding(1, 5).Radius(rSm-1).Background(k.field).Border(1, k.hair)
}

type tone int

const (
	toneNeutral tone = iota
	toneAccent
	toneGreen
	toneAmber
	toneRed
)

// badge is a small rounded label.
func badge(c *ui.Context, s string, tn tone) ui.Element {
	k := tok(c)
	fg, bg := k.muted, k.pill
	switch tn {
	case toneAccent:
		fg, bg = k.accent, k.accentSoft
	case toneGreen:
		fg, bg = k.green, k.greenSoft
	case toneAmber:
		fg, bg = k.amber, k.amberSoft
	case toneRed:
		fg, bg = k.red, k.redSoft
	}
	return ui.Text(c, s).FontSize(11).FontWeight(600).TextColor(fg).Background(bg).Radius(999).Padding(1.5, 7).SingleLine().FontFeatures("tnum")
}

// eyebrow is a small uppercase section label.
func eyebrow(c *ui.Context, s string) ui.Element {
	k := tok(c)
	return ui.Text(c, s).FontSize(11).FontWeight(600).TextColor(k.faint).LetterSpacing(0.3)
}

// panel is a raised card.
func panel(c *ui.Context) ui.Element {
	k := tok(c)
	p := ui.Column(c).Background(k.canvas).Radius(rLg).Padding(sp4).Gap(sp3)
	if k.dark {
		p.Border(1, k.hair).Shadow(0, 8, 24, -6, k.shadow)
	} else {
		p.Shadow(0, 0, 0, 1, k.hair).Shadow(0, 1, 2, 0, k.shadowNear).Shadow(0, 8, 24, -8, k.shadow)
	}
	return p
}

// canvasCard is the large borderless editor surface.
func canvasCard(c *ui.Context) ui.Element {
	k := tok(c)
	p := ui.Column(c).Background(k.canvas).Radius(rXl)
	if k.dark {
		p.Border(1, k.hair).Shadow(0, 12, 32, -8, k.shadow)
	} else {
		p.Shadow(0, 0, 0, 1, k.hair).Shadow(0, 1, 3, 0, k.shadowNear).Shadow(0, 14, 36, -12, k.shadow)
	}
	return p
}

// field restyles a MyGo text input to the app's quiet filled look.
func field(c *ui.Context, e ui.Element) ui.Element {
	k := tok(c)
	bg := k.field
	if e.Hovered() {
		bg = k.fieldHover
	}
	e.Background(bg).Radius(rMd).Border(1, ui.Transparent).Padding(7, 10).FontSize(13)
	if e.Focused() {
		e.Background(k.canvas).Border(1, k.accent.Alpha(0.7)).Shadow(0, 0, 0, 3, k.accent.Alpha(0.16))
	}
	return e
}

// pageHeader is the toolbar row at the top of every page; it doubles as
// the window's drag region under the hidden title bar.
func pageHeader(c *ui.Context, title, subtitle string, actions func()) {
	k := tok(c)
	ui.Row(c).Height(titleBarH).Padding(0, sp5, 0, sp6).Gap(sp2).AlignItems(ui.Center).DragWindow().Children(func() {
		ui.Text(c, title).FontSize(15).FontWeight(650).TextColor(k.text).SingleLine()
		if subtitle != "" {
			ui.Text(c, subtitle).FontSize(12.5).TextColor(k.faint).SingleLine().MinWidth(0).Shrink(1).Margin(0, 0, 0, sp1)
		}
		ui.Spacer(c)
		if actions != nil {
			actions()
		}
	})
}

// hint is a muted helper line.
func hint(c *ui.Context, s string) ui.Element {
	return ui.Text(c, s).FontSize(12).TextColor(tok(c).muted).LineHeight(1.45)
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

// segmented is a compact pill switch.
func segmented(c *ui.Context, key string, sel *int, labels ...string) bool {
	k := tok(c)
	changed := false
	ui.Row(c.Key(key)).Padding(2).Gap(2).Radius(rMd).Background(k.field).Children(func() {
		for i, l := range labels {
			b := ui.ButtonBase(c.Key(fmt.Sprint(key, i))).Height(24).Padding(0, 10).Radius(rSm).Center().Label(l).Selected(*sel == i)
			bg := ui.Transparent
			if *sel == i {
				bg = k.canvas
				if k.dark {
					bg = ui.RGBA(255, 255, 255, 0.12)
				}
				b.Shadow(0, 1, 2, 0, k.shadowNear).Shadow(0, 0, 0, 1, k.hair)
			} else if b.Hovered() {
				bg = k.hover
			}
			b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6}))
			col, w := k.muted, 500
			if *sel == i {
				col, w = k.text, 600
			}
			b.Children(func() { ui.Text(c, l).FontSize(12).FontWeight(w).TextColor(col).SingleLine() })
			if b.Clicked() && *sel != i {
				*sel, changed = i, true
			}
		}
	})
	return changed
}

// flag is a small toggle chip used inside fields (Aa, .*, ^$).
func flag(c *ui.Context, key, text, tip string, on *bool) bool {
	k := tok(c)
	b := ui.ButtonBase(c.Key(key)).Height(22).MinWidth(26).Padding(0, 6).Radius(5).Center().Label(tip).Tooltip(tip).Selected(*on)
	bg, col := ui.Transparent, k.faint
	if *on {
		bg, col = k.accentSoft, k.accent
	} else if b.Hovered() {
		bg = k.hover
	}
	b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 100e6}))
	b.Children(func() { ui.Text(c, text).FontSize(11.5).FontWeight(600).Font(monoFont).TextColor(col).SingleLine() })
	if b.Clicked() {
		*on = !*on
		return true
	}
	return false
}

// tokenColor maps an explanation part kind to its syntax color.
func tokenColor(k tokens, kind string) ui.Color {
	switch kind {
	case "group", "alt":
		return k.accent
	case "class":
		return k.green
	case "quant":
		return k.amber
	case "anchor":
		if k.dark {
			return ui.Hex("#F472B6")
		}
		return ui.Hex("#C026D3")
	case "token":
		if k.dark {
			return ui.Hex("#38BDF8")
		}
		return ui.Hex("#0369A1")
	case "error":
		return k.red
	}
	return k.text
}
