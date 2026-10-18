package gui

import (
	"math"

	"github.com/dandibbert/textrouter/engine"
)

// Families used when the settings leave a font empty.
const (
	defaultEditorFamily = fontMono
	defaultCJKFamily    = fontCJK
	defaultMonoFamily   = fontMono
	defaultUIFamily     = fontSans
)

func orDefault(s, d string) string {
	if s == "" {
		return d
	}
	return s
}

// editorFamily is the editor's font list: the chosen font, then the CJK
// fallback, then the defaults.
func (a *App) editorFamily() string {
	v := &a.Lib.Settings.View
	return orDefault(v.EditorFont, defaultEditorFamily) + ", " + orDefault(v.CJKFont, defaultCJKFamily) + ", " + defaultCJKFamily
}

func (a *App) monoFamily() string {
	v := &a.Lib.Settings.View
	return orDefault(v.MonoFont, defaultMonoFamily) + ", " + orDefault(v.CJKFont, defaultCJKFamily) + ", monospace"
}

func (a *App) uiFamily() string {
	v := &a.Lib.Settings.View
	return orDefault(v.UIFont, defaultUIFamily) + ", " + orDefault(v.CJKFont, defaultCJKFamily)
}

// zoomBy steps the zoom by 10%.
func (a *App) zoomBy(d int) {
	z := math.Round(float64(a.view.zoom)*10+float64(d)) / 10
	a.view.zoom = float32(max(0.6, min(z, 2.5)))
}

// restoreView applies the saved view options.
func (a *App) restoreView() {
	v := a.Lib.Settings.View
	a.view = editorPrefs{lineNumbers: v.LineNumbers, wrap: v.Wrap, invisibles: v.Invisibles, mono: v.Mono, zoom: float32(v.Zoom)}
	a.inspector = v.Inspector
	if v.Inspector == "find" {
		a.ws.tab = 1
	}
	if v.FindRegex {
		a.find.modeIdx = 1
	}
}

// currentView is the view options as they are now.
func (a *App) currentView() engine.View {
	v := a.Lib.Settings.View
	v.LineNumbers, v.Wrap, v.Invisibles, v.Mono = a.view.lineNumbers, a.view.wrap, a.view.invisibles, a.view.mono
	v.Zoom = math.Round(float64(a.view.zoom)*100) / 100
	v.Inspector = a.inspector
	v.FindRegex = a.find.modeIdx == 1
	return v
}

// persistView saves the view options when they changed. Called each
// frame; writes only on change.
func (a *App) persistView() {
	if a.detached {
		return
	}
	v := a.currentView()
	if v != a.Lib.Settings.View || !a.viewSaved {
		changed := v != a.Lib.Settings.View
		a.Lib.Settings.View = v
		a.viewSaved = true
		if changed {
			a.save()
		}
	}
}

// SetWindowSize records the window's size (from the window's resize
// events) in the settings.
func (a *App) SetWindowSize(w, h int) {
	v := &a.Lib.Settings.View
	if w < 600 || h < 400 || (v.WindowW == w && v.WindowH == h) {
		return
	}
	v.WindowW, v.WindowH = w, h
	a.save()
}

// uiFontFamily is the interface font, from the settings.
var uiFontFamily = uiFont
