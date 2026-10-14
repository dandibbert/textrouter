package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings are user preferences.
type Settings struct {
	Hotkey        string   `json:"hotkey"`         // global shortcut opening Clipboard Space
	Theme         string   `json:"theme"`          // "system" | "light" | "dark"
	CopyOnDone    bool     `json:"copyOnDone"`     // "完成" copies the result and hides the window
	Pinned        []string `json:"pinned"`         // cleaner names / "chain:<id>" pinned in Clipboard Space
	ShowUnmeasure bool     `json:"showUnmeasured"` // list unverified cleaners in the palette
	View          View     `json:"view"`
	// NoAutoReadClipboard turns off reading the clipboard into the 剪贴板
	// document on every activation (on by default, hence the negation).
	NoAutoReadClipboard bool `json:"noAutoReadClipboard,omitempty"` // editor view options, restored on launch
}

// View is how the editor looks, remembered across launches.
type View struct {
	LineNumbers bool    `json:"lineNumbers"`
	Wrap        bool    `json:"wrap"`
	Invisibles  bool    `json:"invisibles"`
	Mono        bool    `json:"mono"`      // the options bar's monospace toggle
	Zoom        float64 `json:"zoom"`      // 1 = 100%
	Inspector   string  `json:"inspector"` // "", "cleaners" or "find"
	FindRegex   bool    `json:"findRegex"` // the find panel's regex toggle
	WindowW     int     `json:"windowWidth"`
	WindowH     int     `json:"windowHeight"`
	EditorFont  string  `json:"editorFont"` // family; "" = Inter
	CJKFont     string  `json:"cjkFont"`    // CJK fallback family; "" = system (PingFang SC)
	MonoFont    string  `json:"monoFont"`   // family for the monospace toggle; "" = SF Mono
	UIFont      string  `json:"uiFont"`     // interface family; "" = Inter
	FontSize    float64 `json:"fontSize"`   // DIPs at 100% zoom
	LineHeight  float64 `json:"lineHeight"` // multiple of the font size
}

// DefaultView is the view of a first launch.
func DefaultView() View {
	return View{LineNumbers: true, Wrap: true, Zoom: 1, FontSize: 15, LineHeight: 1.6, WindowW: 1120, WindowH: 720}
}

// Normalize fills values a missing or older file leaves out.
func (v *View) Normalize() {
	d := DefaultView()
	if v.Zoom <= 0 {
		v.Zoom = 1
	}
	if v.FontSize < 8 || v.FontSize > 48 {
		v.FontSize = d.FontSize
	}
	if v.LineHeight < 1 || v.LineHeight > 3 {
		v.LineHeight = d.LineHeight
	}
	if v.WindowW < 600 || v.WindowH < 400 {
		v.WindowW, v.WindowH = d.WindowW, d.WindowH
	}
}

// Library is everything TextRouter persists, as one JSON file.
type Library struct {
	Version  int       `json:"version"`
	Settings Settings  `json:"settings"`
	Chains   []Chain   `json:"chains"`
	RuleSets []RuleSet `json:"ruleSets"`
	path     string
}

// DefaultLibrary is what a first launch starts with.
func DefaultLibrary() *Library {
	return &Library{
		Version: 1,
		Settings: Settings{Hotkey: "CmdOrCtrl+Shift+V", Theme: "system", CopyOnDone: true, ShowUnmeasure: true, View: DefaultView(),
			Pinned: []string{"Scrub", "Remove Extra Spaces", "Make Paragraphs", "Straighten Quotes", "Trim Lines", "Remove Duplicate Lines"}},
		Chains: []Chain{{ID: "demo-email", Name: "示例：整理邮件引用", Steps: []Step{
			{Kind: KindCleaner, Cleaner: "Remove Forwarding (>) Characters"},
			{Kind: KindCleaner, Cleaner: "Make Paragraphs"},
			{Kind: KindCleaner, Cleaner: "Remove Extra Spaces"},
		}}, {ID: "demo-pangu", Name: "示例：中英文之间加空格", Steps: []Step{
			{Kind: KindFind, Find: FindOptions{Find: `(\p{Han})([A-Za-z0-9])`, Replace: "$1 $2", Regex: true, MatchCase: true}},
			{Kind: KindFind, Find: FindOptions{Find: `([A-Za-z0-9])(\p{Han})`, Replace: "$1 $2", Regex: true, MatchCase: true}},
		}}},
		RuleSets: []RuleSet{{ID: "demo-rules", Name: "示例：英式拼写转美式", Rules: []Rule{
			{Find: "colour", Replace: "color"}, {Find: "favourite", Replace: "favorite"}, {Find: "centre", Replace: "center"},
			{Find: `(\w+)ise\b`, Replace: "$1ize", Regex: true, MatchCase: true, Note: "organise → organize"},
		}}},
	}
}

// LibraryPath is <UserConfigDir>/TextRouter/library.json.
func LibraryPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "TextRouter", "library.json")
}

// LoadLibrary reads the library, falling back to defaults.
func LoadLibrary(path string) *Library {
	lib := DefaultLibrary()
	if data, err := os.ReadFile(path); err == nil {
		var l Library
		// A file from before view settings existed keeps the defaults.
		l.Settings.View = DefaultView()
		if json.Unmarshal(data, &l) == nil {
			lib = &l
		}
	}
	lib.Settings.View.Normalize()
	lib.path = path
	return lib
}

// Save writes the library atomically.
func (l *Library) Save() error {
	if l.path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(l.path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(l, "", "  ")
	if err != nil {
		return err
	}
	tmp := l.path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, l.path)
}

// Path returns where the library is stored.
func (l *Library) Path() string { return l.path }

// ChainByID finds a chain.
func (l *Library) ChainByID(id string) *Chain {
	for i := range l.Chains {
		if l.Chains[i].ID == id {
			return &l.Chains[i]
		}
	}
	return nil
}

// Resolver lets chains call each other.
func (l *Library) Resolver() ChainResolver { return l.ChainByID }
