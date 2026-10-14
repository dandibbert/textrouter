package engine

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Settings are user preferences.
type Settings struct {
	Hotkey        string   `json:"hotkey"`        // global shortcut opening Clipboard Space
	Theme         string   `json:"theme"`         // "system" | "light" | "dark"
	CopyOnDone    bool     `json:"copyOnDone"`    // "完成" copies the result and hides the window
	Pinned        []string `json:"pinned"`        // cleaner names / "chain:<id>" pinned in Clipboard Space
	ShowUnmeasure bool     `json:"showUnmeasured"` // list unverified cleaners in the palette
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
		Settings: Settings{Hotkey: "CmdOrCtrl+Shift+V", Theme: "system", CopyOnDone: true, ShowUnmeasure: true,
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
		if json.Unmarshal(data, &l) == nil {
			lib = &l
		}
	}
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
