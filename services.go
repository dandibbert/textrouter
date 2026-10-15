package main

import (
	"context"
	"errors"
	"os"
	"slices"
	"strings"
	"sync"
	"unicode/utf16"

	"github.com/dandibbert/textrouter/engine"
	"github.com/egoist/mygo"
)

// Events Go sends to pages.
var (
	// Command is a menu command for the focused document window, such as
	// "find", "findNext", "toggleSidebar" or "myScrub".
	Command = mygo.NewEvent[string]("command")
	// LoadText replaces the text of a document window (undoably), e.g. when
	// the global shortcut brings back the clipboard workspace.
	LoadText = mygo.NewEvent[string]("doc:load")
	// CloseRequested asks a document window for its final text before it
	// closes; the page answers with Document.Close.
	CloseRequested = mygo.NewEvent[bool]("doc:closeRequested")
	// DocChanged tells a document window its kind, path or name changed.
	DocChanged = mygo.NewEvent[DocInfo]("doc:changed")
	// LibraryChanged is broadcast when cleaners or groups change.
	LibraryChanged = mygo.NewEvent[bool]("library:changed")
	// PrefsChanged is broadcast when preferences change.
	PrefsChanged = mygo.NewEvent[engine.Prefs]("prefs:changed")
	// SelectCleaner asks MyLibrary to show a cleaner (by uuid).
	SelectCleaner = mygo.NewEvent[string]("library:select")
)

// state is the library shared by every window.
type state struct {
	mu  sync.RWMutex
	lib *engine.Library
}

var app = &state{}

func (s *state) read(fn func(l *engine.Library)) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fn(s.lib)
}

// write changes the library, saves it and tells every window.
func (s *state) write(fn func(l *engine.Library) error) error {
	s.mu.Lock()
	err := fn(s.lib)
	if err == nil {
		err = s.lib.Save()
	}
	s.mu.Unlock()
	LibraryChanged.Broadcast(true)
	mygo.RunOnMain(refreshTrayMenu)
	return err
}

func (s *state) prefs() engine.Prefs {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lib.Prefs
}

// systemClipboard is the clipboard that clipboard actions use.
type systemClipboard struct{}

func (systemClipboard) ReadText() string  { return mygo.Clipboard.ReadText() }
func (systemClipboard) WriteText(t string) { mygo.Clipboard.WriteText(t) }

// ---------------------------------------------------------------------------
// Soap: text processing.

// Soap applies cleaners and runs find/replace for the pages.
type Soap struct{}

// CleanerInfo is a cleaner as the cleaner list shows it.
type CleanerInfo struct {
	Name        string `json:"name"`
	Zh          string `json:"zh"`
	Category    string `json:"category"`
	Desc        string `json:"desc"`
	Section     string `json:"section,omitempty"`
	Implemented bool   `json:"implemented"`
	Measured    bool   `json:"measured"`
	Custom      bool   `json:"custom"`
	UUID        string `json:"uuid,omitempty"`
}

// Boot is everything a window needs to start.
type Boot struct {
	Platform   string                 `json:"platform"`
	Cleaners   []CleanerInfo          `json:"cleaners"`
	Groups     []engine.Group         `json:"groups"`
	Custom     []engine.CustomCleaner `json:"custom"`
	Catalog    []engine.ActionInfo    `json:"catalog"`
	Categories []string               `json:"categories"`
	Prefs      engine.Prefs           `json:"prefs"`
	History    []engine.SearchEntry   `json:"history"`
	Presets    []Preset               `json:"presets"`
	Batch      engine.BatchSettings   `json:"batch"`
	LibPath    string                 `json:"libPath"`
}

// Preset is a ready-made regular expression.
type Preset struct {
	Name    string `json:"name"`
	Pattern string `json:"pattern"`
	Replace string `json:"replace"`
	Desc    string `json:"desc"`
}

// Boot returns the cleaners, groups, action catalog and preferences.
func (Soap) Boot() Boot {
	b := Boot{Platform: platform(), Catalog: engine.ActionCatalog(), Categories: engine.ActionCategories}
	for _, c := range engine.Cleaners() {
		if c.Section == "deprecated" {
			continue
		}
		b.Cleaners = append(b.Cleaners, CleanerInfo{Name: c.Name, Zh: c.ZhName, Category: c.Category, Desc: c.Desc,
			Section: c.Section, Implemented: c.Implemented, Measured: c.Measured})
	}
	app.read(func(l *engine.Library) {
		for _, c := range l.Cleaners {
			b.Cleaners = append(b.Cleaners, CleanerInfo{Name: c.Name, Zh: c.Name, Category: "自定义", Desc: c.Comment,
				Implemented: true, Custom: true, UUID: c.UUID})
		}
		b.Custom = slices.Clone(l.Cleaners)
		b.Groups = groupsFor(l)
		b.Prefs = l.Prefs
		b.History = slices.Clone(l.History)
		b.Batch = l.Batch
		b.LibPath = l.Path()
	})
	for _, p := range engine.Presets() {
		b.Presets = append(b.Presets, Preset{p.Name, p.Pattern, p.Replace, p.Desc})
	}
	return b
}

// groupsFor lists the groups in the order Preferences ask for. "standard"
// and "custom" also list the user's cleaners.
func groupsFor(l *engine.Library) []engine.Group {
	var custom []engine.GroupItem
	for _, c := range l.Cleaners {
		custom = append(custom, engine.GroupItem{Kind: "cleaner", Name: c.Name})
	}
	var builtin []engine.Group
	for _, g := range engine.BuiltinGroups() {
		switch g.ID {
		case "standard", "custom", "library":
			if len(custom) > 0 {
				if g.ID != "custom" {
					g.Items = append(g.Items, engine.GroupItem{Kind: "divider"}, engine.GroupItem{Kind: "label", Name: "我的清理器"})
				}
				g.Items = append(g.Items, custom...)
			}
		}
		builtin = append(builtin, g)
	}
	user := slices.Clone(l.Groups)
	switch l.Prefs.GroupOrder {
	case "custom-builtin":
		return append(user, builtin...)
	case "custom":
		if len(user) > 0 {
			return user
		}
	}
	return append(builtin, user...)
}

// ApplyResult is the text a cleaner produced.
type ApplyResult struct {
	Text     string   `json:"text"`
	Changed  bool     `json:"changed"`
	Warnings []string `json:"warnings"`
}

// Apply runs a built-in or custom cleaner on text.
func (Soap) Apply(name, text string) ApplyResult {
	var out string
	var warn []string
	app.read(func(l *engine.Library) {
		if name == "#MYSCRUB" || name == "MyScrub" {
			name = l.Prefs.MyScrub
			if name == "" {
				name = "Scrub"
			}
		}
		out, warn = l.Apply(name, text, systemClipboard{})
	})
	return ApplyResult{Text: out, Changed: out != text, Warnings: warn}
}

// RunResult is what a cleaner being edited does to sample text.
type RunResult struct {
	Text     string             `json:"text"`
	Steps    []engine.StepTrace `json:"steps"`
	Warnings []string           `json:"warnings"`
}

// Run runs actions on text, for MyLibrary's preview.
func (Soap) Run(actions []engine.Action, text string) RunResult {
	var r RunResult
	app.read(func(l *engine.Library) {
		env := l.Env(systemClipboard{})
		r.Text, r.Steps = engine.RunActions(actions, text, env)
		r.Warnings = env.Warnings
	})
	return r
}

// FindRequest is an interactive search.
type FindRequest struct {
	Text         string `json:"text"`
	Find         string `json:"find"`
	Replace      string `json:"replace"`
	Regex        bool   `json:"regex"`
	MatchCase    bool   `json:"matchCase"`
	RegexOptions string `json:"regexOptions"`
	Granularity  int    `json:"granularity"`
	// Limit caps the matches returned (0 = 5000).
	Limit int `json:"limit"`
}

func (r FindRequest) options() engine.FindOptions {
	return engine.FindOptions{Find: r.Find, Replace: r.Replace, Regex: r.Regex, MatchCase: r.MatchCase,
		RegexOptions: r.RegexOptions, Granularity: r.Granularity}
}

// CaptureGroup is a capture group of a match. Offsets are UTF-16, as in JavaScript.
type CaptureGroup struct {
	Index int    `json:"index"`
	Name  string `json:"name,omitempty"`
	From  int    `json:"from"`
	To    int    `json:"to"`
	Text  string `json:"text"`
	Ok    bool   `json:"ok"`
}

// FoundMatch is one match with its replacement.
type FoundMatch struct {
	From   int     `json:"from"`
	To     int     `json:"to"`
	Text   string  `json:"text"`
	Result string  `json:"result"`
	Groups []CaptureGroup `json:"groups"`
}

// FindResult lists the matches of a search.
type FindResult struct {
	Matches   []FoundMatch `json:"matches"`
	Truncated bool         `json:"truncated"`
	Error     string       `json:"error,omitempty"`
}

// offsets converts rune offsets of s to UTF-16 offsets.
type offsets []int

func newOffsets(s string) offsets {
	o := make(offsets, 0, len(s)+1)
	n := 0
	for _, r := range s {
		o = append(o, n)
		n += utf16.RuneLen(r)
	}
	return append(o, n)
}

func (o offsets) at(i int) int {
	if i < 0 {
		return 0
	}
	if i >= len(o) {
		return o[len(o)-1]
	}
	return o[i]
}

// Find lists the matches of a search with their replacements and groups.
func (Soap) Find(req FindRequest) FindResult {
	limit := req.Limit
	if limit <= 0 {
		limit = 5000
	}
	ms, err := req.options().FindAll(req.Text, limit+1)
	if err != nil {
		return FindResult{Error: friendlyError(err)}
	}
	res := FindResult{Matches: []FoundMatch{}}
	if len(ms) > limit {
		ms, res.Truncated = ms[:limit], true
	}
	off := newOffsets(req.Text)
	for _, m := range ms {
		fm := FoundMatch{From: off.at(m.Start), To: off.at(m.End), Result: m.Result}
		for gi, g := range m.Groups {
			gr := CaptureGroup{Index: gi, Name: g.Name, Ok: g.Ok, Text: g.Text}
			if g.Ok {
				gr.From, gr.To = off.at(g.Start), off.at(g.End)
			}
			if gi == 0 {
				fm.Text = g.Text
			}
			fm.Groups = append(fm.Groups, gr)
		}
		res.Matches = append(res.Matches, fm)
	}
	return res
}

// Change replaces From..To (UTF-16 offsets) with Insert.
type Change struct {
	From   int    `json:"from"`
	To     int    `json:"to"`
	Insert string `json:"insert"`
}

// ReplaceResult is what Replace All would change.
type ReplaceResult struct {
	Changes []Change `json:"changes"`
	Error   string   `json:"error,omitempty"`
}

// ReplaceAll returns the changes that replace every match, for the editor
// to apply as one undoable step.
func (Soap) ReplaceAll(req FindRequest) ReplaceResult {
	ms, err := req.options().FindAll(req.Text, 0)
	if err != nil {
		return ReplaceResult{Error: friendlyError(err)}
	}
	off := newOffsets(req.Text)
	res := ReplaceResult{Changes: []Change{}}
	last := -1
	for _, m := range ms {
		if m.Start < last {
			continue
		}
		res.Changes = append(res.Changes, Change{From: off.at(m.Start), To: off.at(m.End), Insert: m.Result})
		last = m.End
	}
	return res
}

// Part explains a piece of a pattern or template.
type Part struct {
	Text string `json:"text"`
	Desc string `json:"desc"`
	Kind string `json:"kind"`
}

// Explain breaks a find pattern (or a replace template, when template is
// true) into parts with plain-language explanations.
func (Soap) Explain(pattern string, regex, template bool) []Part {
	var ps []engine.Part
	if template {
		ps = engine.ExplainTemplate(pattern, regex)
	} else {
		ps = engine.ExplainPattern(pattern, regex)
	}
	out := make([]Part, 0, len(ps))
	for _, p := range ps {
		out = append(out, Part{p.Text, p.Desc, p.Kind})
	}
	return out
}

func friendlyError(err error) string {
	msg := err.Error()
	msg = strings.TrimPrefix(msg, "error parsing regexp: ")
	return msg
}

// ReadClipboard returns the text on the clipboard.
func (Soap) ReadClipboard() string { return mygo.Clipboard.ReadText() }

// WriteClipboard puts text on the clipboard.
func (Soap) WriteClipboard(text string) { mygo.Clipboard.WriteText(text) }

// ---------------------------------------------------------------------------
// Library: MyLibrary and preferences.

// LibraryService edits custom cleaners, groups and preferences.
type LibraryService struct{}

// SaveCleaner inserts or replaces a custom cleaner and returns it as saved
// (with a unique name).
func (LibraryService) SaveCleaner(c engine.CustomCleaner) (engine.CustomCleaner, error) {
	var saved engine.CustomCleaner
	err := app.write(func(l *engine.Library) error {
		saved = l.PutCleaner(c)
		return nil
	})
	return saved, err
}

// NewCleaner creates an empty custom cleaner.
func (s LibraryService) NewCleaner(name string) (engine.CustomCleaner, error) {
	if name == "" {
		name = "新清理器"
	}
	return s.SaveCleaner(engine.CustomCleaner{Name: name, Actions: []engine.Action{}})
}

// DuplicateCleaner copies a custom cleaner.
func (s LibraryService) DuplicateCleaner(uuid string) (engine.CustomCleaner, error) {
	var c *engine.CustomCleaner
	app.read(func(l *engine.Library) {
		if x := l.CleanerByUUID(uuid); x != nil {
			cp := *x
			c = &cp
		}
	})
	if c == nil {
		return engine.CustomCleaner{}, errors.New("找不到这个清理器")
	}
	c.UUID = ""
	c.Name += " 副本"
	return s.SaveCleaner(*c)
}

// DeleteCleaner removes a custom cleaner and its entries in groups.
func (LibraryService) DeleteCleaner(uuid string) error {
	return app.write(func(l *engine.Library) error {
		if c := l.CleanerByUUID(uuid); c != nil {
			name := c.Name
			for gi := range l.Groups {
				l.Groups[gi].Items = slices.DeleteFunc(l.Groups[gi].Items, func(it engine.GroupItem) bool {
					return it.Kind == "cleaner" && it.Name == name
				})
			}
		}
		l.DeleteCleaner(uuid)
		return nil
	})
}

// SaveGroup inserts or replaces a custom group.
func (LibraryService) SaveGroup(g engine.Group) (engine.Group, error) {
	var saved engine.Group
	err := app.write(func(l *engine.Library) error {
		if strings.TrimSpace(g.Name) == "" {
			g.Name = "未命名分组"
		}
		saved = l.PutGroup(g)
		return nil
	})
	return saved, err
}

// DeleteGroup removes a custom group.
func (LibraryService) DeleteGroup(id string) error {
	return app.write(func(l *engine.Library) error { l.DeleteGroup(id); return nil })
}

// ImportSummary is what an import added.
type ImportSummary struct {
	Added   []string `json:"added"`
	Skipped int      `json:"skipped"`
}

// Import asks for TextSoap libraries or exported cleaners and adds them.
func (LibraryService) Import(ctx context.Context) (ImportSummary, error) {
	paths, err := mygo.Dialog.Open(mygo.OpenDialogOptions{
		Parent: mygo.CallerWindow(ctx), Title: "导入清理器", Multiple: true,
		Message: "选择 TextSoap 的 .textsoap9Lib / .textsoapCleaner 文件，或 TextRouter 导出的文件",
	})
	if err != nil || len(paths) == 0 {
		return ImportSummary{}, err
	}
	var sum ImportSummary
	var found []engine.CustomCleaner
	var firstErr error
	for _, data := range readImports(paths) {
		r, err := engine.ImportTextSoap(data)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		found = append(found, r.Cleaners...)
		sum.Skipped += r.Skipped
	}
	if len(found) == 0 {
		if firstErr == nil {
			firstErr = errors.New("这些文件里没有找到清理器")
		}
		return sum, firstErr
	}
	err = app.write(func(l *engine.Library) error {
		for _, c := range found {
			c.UUID = ""
			sum.Added = append(sum.Added, l.PutCleaner(c).Name)
		}
		return nil
	})
	return sum, err
}

// Export asks where to save the given cleaners as a TextSoap library.
func (LibraryService) Export(ctx context.Context, uuids []string) (string, error) {
	var cs []engine.CustomCleaner
	app.read(func(l *engine.Library) {
		for _, id := range uuids {
			if c := l.CleanerByUUID(id); c != nil {
				cs = append(cs, *c)
			}
		}
	})
	if len(cs) == 0 {
		return "", errors.New("没有选中的清理器")
	}
	name := "TextRouter.textsoap9Lib"
	if len(cs) == 1 {
		name = cs[0].Name + ".textsoapCleaner"
	}
	path, err := mygo.Dialog.Save(mygo.SaveDialogOptions{Parent: mygo.CallerWindow(ctx), Title: "导出清理器", DefaultPath: name})
	if err != nil || path == "" {
		return "", err
	}
	data, err := engine.ExportTextSoap(cs)
	if err != nil {
		return "", err
	}
	return path, os.WriteFile(path, data, 0o644)
}

// SetPrefs saves preferences and applies them.
func (LibraryService) SetPrefs(p engine.Prefs) (engine.Prefs, error) {
	var old engine.Prefs
	app.mu.Lock()
	old = app.lib.Prefs
	app.lib.Prefs = p
	err := app.lib.Save()
	app.mu.Unlock()
	applyPrefs(old, p)
	PrefsChanged.Broadcast(p)
	if old.GroupOrder != p.GroupOrder || old.MyScrub != p.MyScrub {
		LibraryChanged.Broadcast(true)
	}
	return p, err
}

// ResetPrefs restores the default preferences.
func (s LibraryService) ResetPrefs() (engine.Prefs, error) { return s.SetPrefs(engine.DefaultPrefs()) }

// RememberSearch adds a search to the history.
func (LibraryService) RememberSearch(e engine.SearchEntry) []engine.SearchEntry {
	var h []engine.SearchEntry
	app.mu.Lock()
	app.lib.RememberSearch(e)
	h = slices.Clone(app.lib.History)
	_ = app.lib.Save()
	app.mu.Unlock()
	return h
}

// ClearHistory forgets every search.
func (LibraryService) ClearHistory() {
	app.mu.Lock()
	app.lib.History = nil
	_ = app.lib.Save()
	app.mu.Unlock()
}

// SaveBatch remembers the Batch File Cleaning settings.
func (LibraryService) SaveBatch(b engine.BatchSettings) {
	app.mu.Lock()
	app.lib.Batch = b
	_ = app.lib.Save()
	app.mu.Unlock()
}

// RevealLibrary shows the library file in the file manager.
func (LibraryService) RevealLibrary() {
	app.read(func(l *engine.Library) { mygo.Shell.ShowItemInFolder(l.Path()) })
}
