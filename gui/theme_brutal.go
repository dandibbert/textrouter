package gui

import (
	"time"

	"github.com/egoist/mygo/ui"
)

// Design tokens of the 新粗野 (neo-brutalist) look, from
// docs/design/clipboard-redesign.md §1. Nothing outside this file writes a
// colour, shadow offset, radius or curve by hand.

// Colours.
var (
	Ground      = ui.Hex("#F3EFE4") // window
	Paper       = ui.Hex("#FFFFFF") // cards, inputs, menus
	Ink         = ui.Hex("#000000") // text, every stroke, every hard shadow
	Muted       = ui.Hex("#3F3F3F") // secondary text
	Faint       = ui.Hex("#7A7A7A") // placeholders, disabled, undone steps
	Accent      = ui.Hex("#FFD23F") // primary button, hover row, current item, scroll thumb
	AccentHover = ui.Hex("#FFDB5E")
	AccentSoft  = ui.Hex("#FFF1B8") // pill hover, selected row
	AccentWash  = ui.Hex("#FFF7D6") // the row of the current match
	AccentDeep  = ui.Hex("#FFB800") // scroll thumb hover
	Focus       = ui.Hex("#3A5BFF") // keyboard focus ring, jump ripple
	Del         = ui.Hex("#FFB4A3") // text about to go (with a 2px strike)
	Ins         = ui.Hex("#A6F0BF") // text about to come
	Hit         = ui.Hex("#FFE47A") // matched but not replaced
	GutterBg    = ui.Hex("#FBF7EC")
	MarkDel     = ui.Hex("#FF6B4A")
	MarkAdd     = ui.Hex("#2FBF71")
	GutterNum   = ui.Hex("#9A9A9A") // line numbers at rest
	SkipLine    = ui.Hex("#C4C4C4") // underline of an unchecked match
	Danger      = ui.Hex("#C62828") // error text on Del

	TileSpace = ui.Hex("#8FB6FF") // whitespace cleaners
	TileQuote = ui.Hex("#FF9F6B") // quote and wrapping cleaners
	TileLine  = ui.Hex("#7EE0B5") // line and paragraph cleaners
	TileCase  = ui.Hex("#F7A6E0") // case cleaners
	TileChain = ui.Hex("#FFD23F") // action chains

	TrafficRed    = ui.Hex("#FF5F57")
	TrafficYellow = ui.Hex("#FEBC2E")
	TrafficGreen  = ui.Hex("#28C840")

	White = ui.Hex("#FFFFFF") // text on Ink
)

// Shapes. Strokes are 2px Ink; shadows are hard (no blur), down and to the
// right in Ink.
const (
	strokeW   = 2
	keycapW   = 1.5
	keycapBot = 3

	shadowSm = 3 // small controls
	shadowMd = 5 // cards
	shadowLg = 6 // floating menus

	radiusCtl   = 6   // buttons, inputs, rows
	radiusSeg   = 8   // segmented control frame
	radiusThumb = 5   // its slider, cleaner tiles
	radiusPill  = 999 // capsules and count badges
	radiusKey   = 4
)

// Layout.
const (
	titleBarH  = 54
	statusBarH = 46
	rightColW  = 348
	bodyGap    = 18
	padTop     = 16
	padLeft    = 16
	padRight   = 20
	padBottom  = 20
	tickColW   = 8
	scrollW    = 10 // thumb width; the track is 6 wider = 16px
	gutterDigW = 7
	gutterPad  = 18
)

// Type.
const (
	fontSans = "Space Grotesk"
	fontMono = "Space Mono"
	fontCJK  = "Noto Sans SC, PingFang SC, Hiragino Sans GB, Noto Sans CJK SC, Microsoft YaHei"

	sizeBody   = 13
	sizeButton = 12.5
	sizeSmall  = 11
	lineBody   = 25 // px
)

// Curves (§6).
var (
	EaseSpring = ui.CubicBezier(.34, 1.56, .64, 1) // press, pop in, slider
	EaseOut    = ui.CubicBezier(.22, 1, .36, 1)    // enter, expand, scroll
	EaseIn     = ui.CubicBezier(.4, 0, 1, 1)       // exit, collapse
)

// Durations (§6).
const (
	dPress     = 60 * time.Millisecond
	dRelease   = 160 * time.Millisecond
	dRow       = 160 * time.Millisecond
	dPreview   = 320 * time.Millisecond
	dApplyGone = 500 * time.Millisecond
	dApplyNew  = 1300 * time.Millisecond
	dJump      = 550 * time.Millisecond
	dTab       = 340 * time.Millisecond
	dPanel     = 300 * time.Millisecond
	dListIn    = 340 * time.Millisecond
	dListStep  = 24 * time.Millisecond
	dMenuIn    = 240 * time.Millisecond
	dMenuOut   = 140 * time.Millisecond
	dMenuStep  = 22 * time.Millisecond
	dFlip      = 220 * time.Millisecond
	dPop       = 280 * time.Millisecond
	dExpand    = 280 * time.Millisecond
	dExpStep   = 30 * time.Millisecond
	dShake     = 380 * time.Millisecond
	dCheck     = 200 * time.Millisecond
	dDraw      = 340 * time.Millisecond
	dToastIn   = 340 * time.Millisecond
	dToastOut  = 220 * time.Millisecond
	dToastHold = 2 * time.Second
	dBar       = 300 * time.Millisecond
	dTickStep  = 15 * time.Millisecond
	dTickMax   = 300 * time.Millisecond
	dTickSlide = 300 * time.Millisecond
	caretBlink = 1060 * time.Millisecond
)

// hard puts the hard shadow of n px on e.
func hard(e ui.Element, n float32) ui.Element {
	if n <= 0 {
		return e
	}
	return e.Shadow(n, n, 0, 0, Ink)
}

// boxed gives e the 2px Ink stroke.
func boxed(e ui.Element) ui.Element { return e.Border(strokeW, Ink) }

// focusRing draws the keyboard focus ring: 3px Focus, 2px outside the box.
func focusRing(e ui.Element) ui.Element {
	if e.FocusVisible() {
		// Shadows paint in the order given: the ring first, the gap over it.
		e.Shadow(0, 0, 0, 5, Focus).Shadow(0, 0, 0, 2, Ground)
	}
	return e
}

// tileColor is the icon tile colour of a cleaner category.
func tileColor(category string) ui.Color {
	switch category {
	case "空白":
		return TileSpace
	case "引用与换行宽度", "字符与编码":
		return TileQuote
	case "换行与段落", "行操作":
		return TileLine
	case "大小写":
		return TileCase
	}
	return AccentSoft
}

// dashed draws e's 2px stroke as dashes (empty states, the rule explanation).
func dashed(e ui.Element) ui.Element {
	return e.DrawOver(func(p *ui.Painter, r ui.Rect) { p.StrokeDashed(r, Ink, radiusSeg, strokeW) })
}
