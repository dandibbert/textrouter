package gui

import (
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo/ui"
)

type updateSpyPlatform struct {
	NopPlatform
	info           UpdateInfo
	checks         int
	autoCheckSet   []bool
	autoDownload   []bool
	savedHotkeys   []string
}

func (p *updateSpyPlatform) UpdateInfo() UpdateInfo { return p.info }
func (p *updateSpyPlatform) CheckForUpdates()       { p.checks++ }
func (p *updateSpyPlatform) SetAutoCheckUpdates(on bool) {
	p.info.AutoCheck = on
	p.autoCheckSet = append(p.autoCheckSet, on)
}
func (p *updateSpyPlatform) SetAutoDownloadUpdates(on bool) {
	p.info.AutoDownload = on
	p.autoDownload = append(p.autoDownload, on)
}
func (p *updateSpyPlatform) SetHotkey(acc string) error {
	p.savedHotkeys = append(p.savedHotkeys, acc)
	return nil
}

func TestDefaultAppVersionMatchesMyGoJSON(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("..", "mygo.json"))
	if err != nil {
		t.Fatalf("read mygo.json: %v", err)
	}
	var cfg struct {
		Version string `json:"version"`
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatalf("parse mygo.json: %v", err)
	}
	if cfg.Version == "" || cfg.Version != DefaultAppVersion {
		t.Fatalf("DefaultAppVersion = %q, mygo.json version = %q", DefaultAppVersion, cfg.Version)
	}
}

func TestSettingsActiveUpdateControlsAndPreferences(t *testing.T) {
	spy := &updateSpyPlatform{
		info: UpdateInfo{
			Version:      "0.2.7",
			Enabled:      true,
			AutoCheck:    true,
			AutoDownload: false,
			LastCheck:    time.Now().Add(-15 * time.Minute),
		},
	}
	lib := engine.LoadLibrary(filepath.Join(t.TempDir(), "library.json"))
	a := New(lib, spy)
	tt := ui.NewTester(a.View, 1180, 760)
	a.OpenTool(PageSettings)
	settle(tt, 3)

	// Switch to 更新与关于 via the left navigation rail.
	if err := tt.Click("更新与关于"); err != nil {
		t.Fatalf("click nav: %v", err)
	}
	settle(tt, 2)

	if !tt.HasText("检查并下载更新…") {
		t.Fatalf("missing active update button: %q", tt.Texts())
	}
	if !tt.HasText("TextRouter 0.2.7") {
		t.Fatalf("expected real version in About, got: %q", tt.Texts())
	}
	for _, tx := range tt.Texts() {
		if tx == "TextRouter 0.2" {
			t.Fatalf("stale hardcoded version 'TextRouter 0.2' still present")
		}
	}

	if err := tt.Click("检查并下载更新…"); err != nil {
		t.Fatalf("click check updates: %v", err)
	}
	if spy.checks != 1 {
		t.Fatalf("expected CheckForUpdates to be called once, got %d", spy.checks)
	}

	if err := tt.Click("切换自动检查更新"); err != nil {
		t.Fatalf("toggle auto check: %v", err)
	}
	if err := tt.Click("切换自动下载并安装更新"); err != nil {
		t.Fatalf("toggle auto download: %v", err)
	}
	if len(spy.autoCheckSet) != 1 || spy.autoCheckSet[0] != false {
		t.Fatalf("autoCheckSet = %v, want [false]", spy.autoCheckSet)
	}
	if len(spy.autoDownload) != 1 || spy.autoDownload[0] != true {
		t.Fatalf("autoDownload = %v, want [true]", spy.autoDownload)
	}

	// Sidebar quick update button works too.
	if err := tt.Click("检查更新…"); err != nil {
		t.Fatalf("click sidebar check updates: %v", err)
	}
	if spy.checks != 2 {
		t.Fatalf("expected 2 CheckForUpdates calls, got %d", spy.checks)
	}

	// Tools popover menu also exposes 检查更新….
	a.Page = PageClipboard
	settle(tt, 2)
	if err := tt.Click("工具"); err != nil {
		t.Fatalf("open tools menu: %v", err)
	}
	settle(tt, 2)
	if !tt.HasText("检查更新…") {
		t.Fatalf("tools menu missing 检查更新…: %q", tt.Texts())
	}
	if err := tt.Click("检查更新…"); err != nil {
		t.Fatalf("click tools menu update: %v", err)
	}
	if spy.checks != 3 {
		t.Fatalf("expected 3 CheckForUpdates calls, got %d", spy.checks)
	}

	// Command palette (⌘K) exposes 检查更新… as well.
	a.openPalette()
	a.paletteQuery = "检查更新"
	items := a.paletteItems()
	if len(items) == 0 || items[0].id != "sys:check-updates" {
		t.Fatalf("palette missing check-updates: %+v", items)
	}
	a.runPaletteItem(items[0])
	if spy.checks != 4 {
		t.Fatalf("expected 4 CheckForUpdates calls, got %d", spy.checks)
	}
}

// TestSettingsNoRightSideBlankVoid guards against the regression where the
// Settings column was left-aligned with MaxWidth(720), leaving a large empty
// void on the right side of the sheet.
func TestSettingsNoRightSideBlankVoid(t *testing.T) {
	for _, size := range [][2]int{{820, 520}, {1024, 700}, {1180, 760}, {1440, 900}} {
		w, h := size[0], size[1]
		t.Run(fmt.Sprintf("%dx%d", w, h), func(t *testing.T) {
			lib := engine.LoadLibrary(filepath.Join(t.TempDir(), "library.json"))
			a := New(lib, nil)
			tt := ui.NewTester(a.View, w, h)
			a.OpenTool(PageSettings)
			settle(tt, 3)

			// Sheet spans [36, w-36]. Rightmost control in first card ("深色" in segmented)
			// and "应用" button should extend near the right side of the sheet.
			sheetRight := float32(w - 36)
			darkR, ok := tt.Find("深色")
			if !ok {
				t.Fatalf("missing 深色 control")
			}
			applyR, ok := tt.Find("应用")
			if !ok {
				t.Fatalf("missing 应用 button")
			}
			rightmost := max(darkR.X+darkR.W, applyR.X+applyR.W)
			unusedRight := sheetRight - rightmost
			// Account for sheet padding (20) + scrollbar gutter (16) + right scroll pad (8) + panel pad (16) = ~60px.
			if unusedRight > 120 {
				t.Fatalf("excessive empty space on right of Settings (%gx%g): rightmost=%.1f sheetRight=%.1f gap=%.1f",
					float32(w), float32(h), rightmost, sheetRight, unusedRight)
			}
			if rightmost > sheetRight+0.5 {
				t.Fatalf("settings controls overflowed sheet right edge: rightmost=%.1f sheetRight=%.1f", rightmost, sheetRight)
			}

			// Verify left nav and right controls never overlap.
			navR, ok := tt.Find("全部设置")
			if !ok {
				t.Fatalf("missing 全部设置 nav item")
			}
			titleR, ok := tt.Find("外观")
			if !ok {
				t.Fatalf("missing 外观 label")
			}
			if navR.X+navR.W > titleR.X+0.5 {
				t.Fatalf("settings nav (%+v) overlaps content (%+v)", navR, titleR)
			}
		})
	}
}

// TestAppIconsSingleRowAndInBounds verifies the 5 app icon tiles sit in a
// balanced single row at normal desktop widths and stay inside the sheet.
func TestAppIconsSingleRowAndInBounds(t *testing.T) {
	for _, w := range []int{1060, 1180, 1440} {
		t.Run(fmt.Sprint("w=", w), func(t *testing.T) {
			lib := engine.LoadLibrary(filepath.Join(t.TempDir(), "library.json"))
			a := New(lib, nil)
			a.setting.section = "icon"
			tt := ui.NewTester(a.View, w, 760)
			a.OpenTool(PageSettings)
			settle(tt, 3)

			var prev ui.Rect
			var firstY float32
			for i, icn := range AppIcons {
				r, ok := tt.Find("应用图标 " + icn.Name)
				if !ok {
					t.Fatalf("missing app icon tile %q", icn.Name)
				}
				if r.X+r.W > float32(w-36)+0.5 {
					t.Fatalf("icon tile %q overflows sheet: %+v", icn.Name, r)
				}
				if i == 0 {
					firstY = r.Y
				} else {
					if abs32(r.Y-firstY) > 2 {
						t.Fatalf("icon tile %q wrapped onto second row at w=%d: y=%.1f vs firstY=%.1f", icn.Name, w, r.Y, firstY)
					}
					if prev.X+prev.W > r.X+0.5 {
						t.Fatalf("icon tiles overlap: prev=%+v cur=%+v", prev, r)
					}
				}
				prev = r
			}
		})
	}
}

// TestAllSheetsTopBreathingRoomAndNoOverflow ensures every tool sheet keeps
// clean vertical separation below pageHeader and stays within bounds even at
// the minimum window size (820x520).
func TestAllSheetsTopBreathingRoomAndNoOverflow(t *testing.T) {
	sheets := []struct {
		page        string
		firstAnchor string
	}{
		{PageSettings, "全部设置"},
		{PageCleaners, "搜索清理器"},
		{PageChains, "我的动作链"},
		{PageBatch, "导入 / 导出"},
		{PageFiles, "选择文件…"},
	}
	for _, size := range [][2]int{{820, 520}, {1180, 760}} {
		w, h := size[0], size[1]
		for _, sh := range sheets {
			t.Run(fmt.Sprintf("%s/%dx%d", sh.page, w, h), func(t *testing.T) {
				lib := engine.LoadLibrary(filepath.Join(t.TempDir(), "library.json"))
				a := New(lib, nil)
				a.Text = "My favourite colour is grey.\nHello  world"
				tt := ui.NewTester(a.View, w, h)
				a.OpenTool(sh.page)
				settle(tt, 3)

				closeR, ok := tt.Find("关闭")
				if !ok {
					t.Fatalf("missing sheet close button")
				}
				headerBottom := float32(36 + titleBarH)
				if closeR.Y+closeR.H > headerBottom+0.5 {
					t.Fatalf("close button outside header: %+v", closeR)
				}
				r, ok := tt.Find(sh.firstAnchor)
				if !ok {
					t.Fatalf("missing %q in %q", sh.firstAnchor, tt.Texts())
				}
				if r.Y < headerBottom+8 {
					t.Fatalf("%s content (%q at y=%.1f) sits too close to headerBottom=%.1f", sh.page, sh.firstAnchor, r.Y, headerBottom)
				}
				if r.X < 36-0.5 || r.X+r.W > float32(w-36)+0.5 {
					t.Fatalf("%s %q out of sheet horizontal bounds [36, %d]: %+v", sh.page, sh.firstAnchor, w-36, r)
				}
			})
		}
	}
}

// TestBatchColumnsDoNotOverlap guards against crowded/touching 选项 and 命中
// columns in the batch replacement table.
func TestBatchColumnsDoNotOverlap(t *testing.T) {
	for _, w := range []int{820, 1180} {
		t.Run(fmt.Sprint("w=", w), func(t *testing.T) {
			lib := engine.LoadLibrary(filepath.Join(t.TempDir(), "library.json"))
			a := New(lib, nil)
			a.Text = "My favourite colour is centre grey — organise it."
			tt := ui.NewTester(a.View, w, 700)
			a.OpenTool(PageBatch)
			settle(tt, 3)

			reR, ok1 := tt.Find("按正则表达式匹配")
			csR, ok2 := tt.Find("区分大小写")
			hitR, ok3 := tt.Find("1 处")
			delR, ok4 := tt.Find("删除第 1 条")
			if !ok1 || !ok2 || !ok3 || !ok4 {
				t.Fatalf("missing batch row controls: re=%v cs=%v hit=%v del=%v", ok1, ok2, ok3, ok4)
			}
			if reR.X+reR.W > csR.X+0.5 {
				t.Fatalf(".* (%+v) overlaps Aa (%+v)", reR, csR)
			}
			if csR.X+csR.W > hitR.X-2 {
				t.Fatalf("Aa (%+v) crowds or overlaps 命中 badge (%+v)", csR, hitR)
			}
			if hitR.X+hitR.W > delR.X+0.5 {
				t.Fatalf("命中 badge (%+v) overlaps delete button (%+v)", hitR, delR)
			}
		})
	}
}

// TestChainsStepNumbersAndIconsAreVisible guards against the black-on-black
// contrast bug where timeline step numbers and selected chain icon boxes drew
// Ink on Ink.
func TestChainsStepNumbersAndIconsAreVisible(t *testing.T) {
	lib := engine.LoadLibrary(filepath.Join(t.TempDir(), "library.json"))
	a := New(lib, nil)
	a.Text = "> quoted   line"
	tt := ui.NewTester(a.View, 1180, 760)
	a.OpenTool(PageChains)
	settle(tt, 4)

	img := tt.Image()
	step1R, ok := tt.Find("1")
	if !ok {
		t.Fatalf("missing step number 1")
	}
	var brightPx, darkPx int
	for y := int(step1R.Y) - 6; y <= int(step1R.Y+step1R.H)+6; y++ {
		for x := int(step1R.X) - 6; x <= int(step1R.X+step1R.W)+6; x++ {
			if !inBounds(img.Bounds(), x, y) {
				continue
			}
			r, g, b, _ := color.RGBAModel.Convert(img.At(x, y)).RGBA()
			lum := (r>>8)*299 + (g>>8)*587 + (b>>8)*114
			if lum > 150000 {
				brightPx++
			} else if lum < 40000 {
				darkPx++
			}
		}
	}
	if brightPx < 20 || darkPx < 10 {
		t.Fatalf("step 1 bubble lacks contrast (black-on-black regression?): brightPx=%d darkPx=%d rect=%+v", brightPx, darkPx, step1R)
	}
}

func inBounds(b image.Rectangle, x, y int) bool { return image.Pt(x, y).In(b) }

// TestTitleBarFileTabSaveButtonAndTabsDoNotOverlap extends the title bar
// overlap regression check to cover file tabs with the 存储 button and crumbs.
func TestTitleBarFileTabSaveButtonAndTabsDoNotOverlap(t *testing.T) {
	for _, w := range []int{820, 960, 1180} {
		t.Run(fmt.Sprint("w=", w), func(t *testing.T) {
			lib := engine.LoadLibrary(filepath.Join(t.TempDir(), "library.json"))
			a := New(lib, nil)
			a.Text = "clip"
			tt := ui.NewTester(a.View, w, 700)
			tt.SetTitleBar(ui.TitleBar{Height: 42, Left: 78})
			dir := t.TempDir()
			for i := 0; i < 4; i++ {
				p := filepath.Join(dir, fmt.Sprintf("long-document-name-%d.txt", i))
				if err := os.WriteFile(p, []byte("hello   world"), 0o644); err != nil {
					t.Fatal(err)
				}
				a.loadFile(p)
			}
			a.applyAction("Remove Extra Spaces")
			a.pushStep("新动作链 1", &engine.Step{Kind: engine.KindCleaner, Cleaner: "Uppercase"})
			settle(tt, 4)

			ordered := []string{"打开文件", "工具", "撤销", "重做", "保存文件"}
			var prev ui.Rect
			var prevName string
			for i, name := range ordered {
				r, ok := tt.Find(name)
				if !ok {
					t.Fatalf("missing %q in %q", name, tt.Texts())
				}
				if r.X < 0 || r.X+r.W > float32(w)+0.5 {
					t.Fatalf("%q out of window [0, %d]: %+v", name, w, r)
				}
				if i > 0 && prev.X+prev.W > r.X+0.5 {
					t.Fatalf("%s (%+v) overlaps %s (%+v)", prevName, prev, name, r)
				}
				prev, prevName = r, name
			}
		})
	}
}

// TestPaletteNoDuplicateDeadFindEntries ensures the command palette does not
// list disabled editor-internal find commands alongside 查找替换.
func TestPaletteNoDuplicateDeadFindEntries(t *testing.T) {
	lib := engine.LoadLibrary(filepath.Join(t.TempDir(), "library.json"))
	a := New(lib, nil)
	a.paletteQuery = "查找"
	for _, it := range a.paletteItems() {
		if strings.HasPrefix(it.id, "cmd:find") || strings.HasPrefix(it.id, "cmd:replace") {
			t.Fatalf("dead internal editor command in palette: %+v", it)
		}
	}
}
