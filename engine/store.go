package engine

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Prefs mirror TextSoap's Preferences window, plus what the textsoapAgent
// companion provides (global shortcut, menu bar icon).
type Prefs struct {
	// General
	Appearance           string `json:"appearance"` // system | light | dark
	QuitWhenLastClosed   bool   `json:"quitWhenLastClosed"`
	ApplyToAllNoSel      bool   `json:"applyToAllWhenNoSelection"`
	MyScrub              string `json:"myScrub"`
	DefaultGroup         string `json:"defaultGroup"`
	GroupOrder           string `json:"groupOrder"` // builtin-custom | custom-builtin | custom
	CompactDisplay       bool   `json:"compactDisplay"`
	ClipboardWorkspace   string `json:"clipboardWorkspace"` // plain | disabled
	CopyBackOnClose      bool   `json:"copyBackOnClose"`
	NewFindUsesSearchDef bool   `json:"newFindUsesSearchDefaults"`
	// Text editing
	ShowInvisibles   bool   `json:"showInvisibles"`
	ShowLineNumbers  bool   `json:"showLineNumbers"`
	ShowWordCount    bool   `json:"showWordCount"`
	HighlightLine    bool   `json:"highlightCurrentLine"`
	WrapText         bool   `json:"wrapText"`
	Zoom             int    `json:"zoom"`
	FontFamily       string `json:"fontFamily"`
	FontSize         int    `json:"fontSize"`
	SearchRegex      bool   `json:"searchRegex"`
	SearchMatchCase  bool   `json:"searchMatchCase"`
	SearchMatchLines bool   `json:"searchMatchLines"`
	SearchLive       bool   `json:"searchLive"`
	// Agent
	GlobalHotkey  string `json:"globalHotkey"`
	MenuBarIcon   bool   `json:"menuBarIcon"`
	AllowScripts  bool   `json:"allowShellScripts"`
	SidebarWidth  int    `json:"sidebarWidth"`
	SidebarHidden bool   `json:"sidebarHidden"`
}

// DefaultPrefs are TextSoap's defaults where it documents them.
func DefaultPrefs() Prefs {
	return Prefs{
		Appearance: "system", ApplyToAllNoSel: true, MyScrub: "Scrub", DefaultGroup: "standard",
		GroupOrder: "builtin-custom", ClipboardWorkspace: "plain", CopyBackOnClose: true, NewFindUsesSearchDef: true,
		ShowLineNumbers: true, ShowWordCount: true, HighlightLine: true, WrapText: true, Zoom: 100, FontSize: 14,
		SearchMatchLines: true, SearchLive: true, GlobalHotkey: "CmdOrCtrl+Alt+V", MenuBarIcon: true, SidebarWidth: 320,
	}
}

// GroupItem is a cleaner, a label or a divider in a group.
type GroupItem struct {
	Kind  string `json:"kind"` // cleaner | label | divider
	Name  string `json:"name,omitempty"`
	Color string `json:"color,omitempty"`
}

// Group is a collection of cleaners.
type Group struct {
	ID      string      `json:"id"`
	Name    string      `json:"name"`
	BuiltIn bool        `json:"builtIn,omitempty"`
	Items   []GroupItem `json:"items"`
}

// BatchSettings are the last Batch File Cleaning settings.
type BatchSettings struct {
	Cleaner     string `json:"cleaner"`
	ExtMode     string `json:"extMode"` // additional | only
	Extensions  string `json:"extensions"`
	DestFolder  string `json:"destFolder"`
	Prefix      string `json:"prefix"`
	Suffix      string `json:"suffix"`
	NewExt      string `json:"newExt"`
	Encoding    string `json:"encoding"`
	IncludeSubs bool   `json:"includeSubfolders"`
}

// DefaultBatch matches TextSoap's "Default Settings".
func DefaultBatch() BatchSettings {
	return BatchSettings{Cleaner: "Scrub", ExtMode: "additional", DestFolder: "results", Encoding: "auto"}
}

// SearchEntry is one remembered interactive search.
type SearchEntry struct {
	Find         string `json:"find"`
	Replace      string `json:"replace"`
	Regex        bool   `json:"regex"`
	MatchCase    bool   `json:"matchCase"`
	RegexOptions string `json:"regexOptions"`
	Granularity  int    `json:"granularity"`
}

// Library is everything TextRouter keeps, in one JSON file (TextSoap's
// MyLibrary + preferences).
type Library struct {
	Version  int             `json:"version"`
	Cleaners []CustomCleaner `json:"cleaners"`
	Groups   []Group         `json:"groups"`
	Prefs    Prefs           `json:"prefs"`
	History  []SearchEntry   `json:"searchHistory"`
	Batch    BatchSettings   `json:"batch"`

	mu   sync.Mutex
	path string
}

// LibraryPath is <UserConfigDir>/TextRouter/library.json.
func LibraryPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = os.TempDir()
	}
	return filepath.Join(dir, "TextRouter", "library.json")
}

// LoadLibrary reads the library; a missing or unreadable file gives the
// defaults (with the example cleaners of a first launch).
func LoadLibrary(path string) *Library {
	lib := &Library{Version: 2, Prefs: DefaultPrefs(), Batch: DefaultBatch()}
	data, err := os.ReadFile(path)
	if err == nil {
		var probe struct {
			Version int `json:"version"`
		}
		_ = json.Unmarshal(data, &probe)
		if probe.Version >= 2 {
			_ = json.Unmarshal(data, lib)
		} else {
			lib.Cleaners = exampleCleaners()
		}
	} else {
		lib.Cleaners = exampleCleaners()
	}
	if lib.Prefs.Zoom == 0 {
		lib.Prefs.Zoom = 100
	}
	if lib.Prefs.FontSize == 0 {
		lib.Prefs.FontSize = 14
	}
	lib.path = path
	return lib
}

// Save writes the library atomically.
func (l *Library) Save() error {
	l.mu.Lock()
	defer l.mu.Unlock()
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

// CleanerByName finds a custom cleaner by name.
func (l *Library) CleanerByName(name string) *CustomCleaner {
	for i := range l.Cleaners {
		if l.Cleaners[i].Name == name {
			return &l.Cleaners[i]
		}
	}
	return nil
}

// CleanerByUUID finds a custom cleaner by uuid.
func (l *Library) CleanerByUUID(id string) *CustomCleaner {
	for i := range l.Cleaners {
		if strings.EqualFold(l.Cleaners[i].UUID, id) {
			return &l.Cleaners[i]
		}
	}
	return nil
}

// PutCleaner inserts or replaces a cleaner (by uuid) and keeps names unique.
func (l *Library) PutCleaner(c CustomCleaner) CustomCleaner {
	if c.UUID == "" {
		c.UUID = NewUUID()
	}
	if strings.TrimSpace(c.Name) == "" {
		c.Name = "未命名清理器"
	}
	c.Name = l.uniqueName(c.Name, c.UUID)
	c.Modified = time.Now()
	if old := l.CleanerByUUID(c.UUID); old != nil {
		if old.Name != c.Name {
			l.renameInGroups(old.Name, c.Name)
		}
		*old = c
		return c
	}
	l.Cleaners = append(l.Cleaners, c)
	return c
}

func (l *Library) uniqueName(name, uuid string) string {
	taken := func(n string) bool {
		if CleanerByName(n) != nil {
			return true
		}
		for _, c := range l.Cleaners {
			if c.Name == n && !strings.EqualFold(c.UUID, uuid) {
				return true
			}
		}
		return false
	}
	if !taken(name) {
		return name
	}
	for i := 2; ; i++ {
		n := fmt.Sprintf("%s %d", name, i)
		if !taken(n) {
			return n
		}
	}
}

func (l *Library) renameInGroups(from, to string) {
	for gi := range l.Groups {
		for ii := range l.Groups[gi].Items {
			if it := &l.Groups[gi].Items[ii]; it.Kind == "cleaner" && it.Name == from {
				it.Name = to
			}
		}
	}
}

// DeleteCleaner removes a custom cleaner.
func (l *Library) DeleteCleaner(uuid string) {
	for i := range l.Cleaners {
		if strings.EqualFold(l.Cleaners[i].UUID, uuid) {
			l.Cleaners = append(l.Cleaners[:i], l.Cleaners[i+1:]...)
			return
		}
	}
}

// PutGroup inserts or replaces a custom group.
func (l *Library) PutGroup(g Group) Group {
	if g.ID == "" {
		g.ID = NewUUID()
	}
	g.BuiltIn = false
	for i := range l.Groups {
		if l.Groups[i].ID == g.ID {
			l.Groups[i] = g
			return g
		}
	}
	l.Groups = append(l.Groups, g)
	return g
}

// DeleteGroup removes a custom group.
func (l *Library) DeleteGroup(id string) {
	for i := range l.Groups {
		if l.Groups[i].ID == id {
			l.Groups = append(l.Groups[:i], l.Groups[i+1:]...)
			return
		}
	}
}

// RememberSearch keeps the last 30 distinct searches, newest first.
func (l *Library) RememberSearch(e SearchEntry) {
	if e.Find == "" {
		return
	}
	out := []SearchEntry{e}
	for _, h := range l.History {
		if h.Find == e.Find && h.Replace == e.Replace && h.Regex == e.Regex {
			continue
		}
		out = append(out, h)
	}
	if len(out) > 30 {
		out = out[:30]
	}
	l.History = out
}

// Env returns a run environment bound to the library.
func (l *Library) Env(clip Clipboard) *Env {
	return &Env{
		Custom:       l.CleanerByName,
		MyScrub:      l.Prefs.MyScrub,
		Clipboard:    clip,
		AllowScripts: l.Prefs.AllowScripts,
	}
}

// Apply runs a built-in or custom cleaner by name.
func (l *Library) Apply(name, text string, clip Clipboard) (string, []string) {
	env := l.Env(clip)
	if c := l.CleanerByName(name); c != nil && CleanerByName(name) == nil {
		out, _ := c.Run(text, env)
		return out, env.Warnings
	}
	return env.applyCleaner(name, text), env.Warnings
}

func act(id string, p Params, children ...Action) Action {
	return Action{ID: id, Params: p, Enabled: true, Children: children}
}

func exampleCleaners() []CustomCleaner {
	return []CustomCleaner{
		{UUID: NewUUID(), Name: "示例：整理邮件引用", Comment: "去掉转发符号，合并段落，删除多余空格。", Actions: []Action{
			act("applycleaner", Params{"cleaner": "Remove Forwarding (>) Characters"}),
			act("applycleaner", Params{"cleaner": "Make Paragraphs"}),
			act("applycleaner", Params{"cleaner": "Remove Extra Spaces"}),
		}},
		{UUID: NewUUID(), Name: "示例：中英文之间加空格", Actions: []Action{
			act("find", Params{"find": `(\p{Han})([A-Za-z0-9])`, "replace": "$1 $2", "type": 1, "match_case": true, "regex_options": "m", "granularity": 0, "list_name": ""}),
			act("find", Params{"find": `([A-Za-z0-9])(\p{Han})`, "replace": "$1 $2", "type": 1, "match_case": true, "regex_options": "m", "granularity": 0, "list_name": ""}),
		}},
		{UUID: NewUUID(), Name: "示例：按列表批量替换", Comment: "列表的每一行是一组「查找 → 替换」，A 列查找，B 列替换为。", Actions: []Action{
			act("runtime.definelist", Params{"name": "英式转美式", "columns": 2, "rows": []any{
				[]any{"colour", "color"}, []any{"favourite", "favorite"}, []any{"centre", "center"}, []any{"organise", "organize"},
			}}),
			act("find", Params{"find": "$v{A}", "replace": "$v{B}", "type": 0, "match_case": false, "granularity": 2, "regex_options": "", "list_name": "英式转美式"}),
		}},
	}
}
