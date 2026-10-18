package gui

import (
	"bytes"
	"embed"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/egoist/mygo/ui"
)

// The app icon: five designs bundled, the second the default (it is also the
// icon of the built .app, resources/icon.png). The choice changes the Dock
// icon at once and is kept in a small file beside the library.

//go:embed appicons/*.png
var appIconFS embed.FS

// AppIcon is one of the bundled icons.
type AppIcon struct{ ID, Name string }

// AppIcons lists the bundled icons in the designer's order.
var AppIcons = []AppIcon{
	{"neon-routes", "Neon Routes"},
	{"mint-minimal", "Mint Minimal"},
	{"terminal-pixels", "Terminal Pixels"},
	{"tactile-wood", "Tactile Wood"},
	{"cosmic-flow", "Cosmic Flow"},
}

// DefaultAppIcon is the icon used until another is chosen.
const DefaultAppIcon = "mint-minimal"

// AppIconPNG returns the 512px PNG of an icon, the default for unknown ids.
func AppIconPNG(id string) []byte {
	data, err := appIconFS.ReadFile("appicons/" + id + ".png")
	if err != nil {
		data, _ = appIconFS.ReadFile("appicons/" + DefaultAppIcon + ".png")
	}
	return data
}

func appIconFile(libPath string) string {
	return filepath.Join(filepath.Dir(libPath), "appicon")
}

// SavedAppIcon is the id of the icon chosen earlier, from the file beside
// the library at libPath.
func SavedAppIcon(libPath string) string {
	b, err := os.ReadFile(appIconFile(libPath))
	if err != nil {
		return DefaultAppIcon
	}
	id := strings.TrimSpace(string(b))
	for _, ic := range AppIcons {
		if ic.ID == id {
			return id
		}
	}
	return DefaultAppIcon
}

// chooseAppIcon switches the Dock icon and remembers the choice.
func (a *App) chooseAppIcon(id string) {
	if err := os.WriteFile(appIconFile(a.Lib.Path()), []byte(id), 0o644); err != nil {
		a.status = "保存图标选择失败：" + err.Error()
	}
	a.appIcon = id
	a.Platform.SetAppIcon(AppIconPNG(id))
}

func (a *App) currentAppIcon() string {
	if a.appIcon == "" {
		a.appIcon = SavedAppIcon(a.Lib.Path())
	}
	return a.appIcon
}

var (
	iconBmpMu sync.Mutex
	iconBmps  = map[string]*ui.Bitmap{}
)

// appIconBitmap decodes a bundled icon for the settings page.
func appIconBitmap(id string) *ui.Bitmap {
	iconBmpMu.Lock()
	defer iconBmpMu.Unlock()
	if b, ok := iconBmps[id]; ok {
		return b
	}
	img, err := png.Decode(bytes.NewReader(AppIconPNG(id)))
	if err != nil {
		return nil
	}
	b := ui.NewBitmap(img)
	iconBmps[id] = b
	return b
}

// appIconPicker is the row of icon tiles in the settings.
func (a *App) appIconPicker(c *ui.Context) {
	cur := a.currentAppIcon()
	ui.Row(c).Gap(sp3).Wrap().Padding(sp3, 0).Children(func() {
		for _, ic := range AppIcons {
			sel := ic.ID == cur
			b := ui.ButtonBase(c.Key("appicon-" + ic.ID)).Padding(sp2).Gap(sp1).Radius(radiusCtl).AlignItems(ui.Center).Label("应用图标 " + ic.Name).Tooltip(ic.Name).Selected(sel)
			bg := Paper
			switch {
			case sel:
				bg = Accent
			case b.Hovered():
				bg = AccentSoft
			}
			b.Background(b.AnimateColor("bg", bg, ui.Animation{Duration: 120e6}))
			boxed(b)
			lift(b, shadowSm)
			b.Children(func() {
				if bm := appIconBitmap(ic.ID); bm != nil {
					ui.Image(c, bm).Size(64, 64)
				}
				ui.Text(c, ic.Name).FontSize(11).FontWeight(map[bool]int{true: 700, false: 500}[sel]).TextColor(Ink).SingleLine()
			})
			if b.Clicked() {
				a.chooseAppIcon(ic.ID)
			}
		}
	})
}
