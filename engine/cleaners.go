package engine

import (
	_ "embed"
	"encoding/json"
	"sort"
	"strings"
	"sync"
)

//go:embed data/builtin_inventory.json
var inventoryJSON []byte

// Cleaner is one built-in text transform.
type Cleaner struct {
	Token    string // TextSoap token, e.g. "#SCRUB"
	Name     string // TextSoap English name; also the id used by chains
	ZhName   string // Chinese display name
	Category string // Chinese category
	Desc     string // vendor description (English)
	// Measured is true when the cleaner is one of the 38 with measured
	// TextSoap truth (behaviour_truth.json). Others are UNVERIFIED: they
	// follow the vendor description and were never checked against the app.
	Measured bool
	// Implemented is false for cleaners that only make sense for rich text
	// or need OS services (rulers, fonts, notifications...). They pass text
	// through unchanged.
	Implemented bool
	Fn          func(string) string `json:"-"`
}

var (
	cleanerOnce sync.Once
	cleaners    []*Cleaner
	cleanerByID map[string]*Cleaner
)

type invEntry struct {
	Token       string `json:"token"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Kind        string `json:"kind"`
}

var measuredNames = map[string]bool{}

func init() {
	for _, n := range []string{"Scrub", "MyScrub", "Remove Extra Spaces", "Remove Forwarding (>) Characters",
		"Convert MIME-encoded Characters", "Make Paragraphs", "Remove Extra Returns", "Remove All Tabs",
		"Remove Control Characters", "Uppercase", "Lowercase", "Capitalize Sentences", "Capitalize Words",
		"Capitalize with Title Case", "Straighten Quotes", "Smarten Quotes", "Expand Tabs", "Rewrap Text",
		"Wrap At 50", "Wrap At 60", "Wrap At 70", "Quote Text", "Internet Friendly Text",
		"Remove All High Ascii Characters", "Trim Lines", "Trim Beginning of Lines", "Trim End of Lines",
		"Strip 2 or more Spaces", "Remove All Spaces", "Remove All Whitespace", "Remove Duplicate Lines",
		"Multiple Returns to 1 Return", "Convert Non-Breaking Spaces", "Convert Bullets to ASCII",
		"Extract Text from HTML Source", "HTML Tag: Bold", "BBCode: Bold", "PLIST Tag: String"} {
		measuredNames[n] = true
	}
}

func loadCleaners() {
	var inv struct {
		Cleaners []invEntry `json:"cleaners"`
	}
	_ = json.Unmarshal(inventoryJSON, &inv)
	cleanerByID = map[string]*Cleaner{}
	category := "常用"
	for _, e := range inv.Cleaners {
		if strings.HasPrefix(e.Token, "#LABEL:") {
			category = labelZh(strings.TrimPrefix(e.Token, "#LABEL:"))
			continue
		}
		c := &Cleaner{Token: e.Token, Name: e.Name, Desc: e.Description, Measured: measuredNames[e.Name]}
		c.Category = guessCategory(e.Token, category)
		if impl, ok := builtinImpls[e.Token]; ok {
			c.Fn, c.ZhName, c.Implemented = impl.fn, impl.zh, impl.fn != nil
		}
		if c.ZhName == "" {
			c.ZhName = e.Name
		}
		if c.Fn == nil {
			c.Fn = func(s string) string { return s }
		}
		cleaners = append(cleaners, c)
		cleanerByID[c.Name] = c
		cleanerByID[c.Token] = c
	}
}

// Cleaners returns every built-in cleaner in TextSoap's order.
func Cleaners() []*Cleaner {
	cleanerOnce.Do(loadCleaners)
	return cleaners
}

// CleanerByName finds a cleaner by English name or #TOKEN.
func CleanerByName(name string) *Cleaner {
	cleanerOnce.Do(loadCleaners)
	return cleanerByID[name]
}

// ApplyCleaner runs a cleaner by name; unknown names pass text through, as
// in TextSoap.
func ApplyCleaner(name, text string) string {
	if c := CleanerByName(name); c != nil {
		return c.Fn(text)
	}
	return text
}

// Categories lists the categories in first-seen order.
func Categories() []string {
	seen := map[string]bool{}
	var out []string
	for _, c := range Cleaners() {
		if !seen[c.Category] {
			seen[c.Category] = true
			out = append(out, c.Category)
		}
	}
	return out
}

// CleanerStats counts measured / implemented cleaners.
func CleanerStats() (total, implemented, measured int) {
	for _, c := range Cleaners() {
		total++
		if c.Implemented {
			implemented++
		}
		if c.Measured {
			measured++
		}
	}
	return
}

func labelZh(l string) string {
	switch l {
	case "Markdown Cleaners":
		return "Markdown"
	case "HTML Cleaners":
		return "HTML"
	case "Plist Cleaners":
		return "Plist"
	case "BBCode Cleaners":
		return "BBCode"
	case "Deprecated":
		return "已弃用"
	}
	return l
}

func guessCategory(tok, label string) string {
	if label != "常用" {
		return label
	}
	t := tok
	switch {
	case t == "#SCRUB" || t == "#MYSCRUB" || t == "#NOTHING":
		return "综合清理"
	case strings.Contains(t, "CASE") || strings.Contains(t, "CAPITALIZE") || strings.Contains(t, "NAME_FIX") || t == "#TITLE_CASE1":
		return "大小写"
	case strings.Contains(t, "SPACE") || strings.Contains(t, "TAB") || strings.Contains(t, "WHITESPACE") || strings.Contains(t, "TRIM"):
		return "空白"
	case strings.Contains(t, "CR") || strings.Contains(t, "RETURN") || strings.Contains(t, "PARAGRAPH") && !strings.Contains(t, "PARAGRAPH_") || strings.Contains(t, "LINE_ENDING"):
		return "换行与段落"
	case strings.Contains(t, "QUOTE") || strings.Contains(t, "FORWARD") || strings.Contains(t, "WRAP"):
		return "引用与换行宽度"
	case strings.HasPrefix(t, "#LINE") || strings.Contains(t, "DUPLICATE") || strings.Contains(t, "LINENUMBERS"):
		return "行操作"
	case strings.Contains(t, "DATE"):
		return "日期"
	case strings.Contains(t, "LINESPACE") || strings.HasPrefix(t, "#PARAGRAPH_") || strings.Contains(t, "STYLE") || strings.Contains(t, "COLOR") || strings.Contains(t, "HYPERLINK") || strings.Contains(t, "PLAIN") || strings.Contains(t, "LIGATURES"):
		return "富文本（纯文本下无效）"
	case t == "#SCRUB" || t == "#MYSCRUB" || t == "#NOTHING":
		return "综合清理"
	}
	return "字符与编码"
}

// SortedCleanerNames is a helper for tests and palettes.
func SortedCleanerNames() []string {
	var out []string
	for _, c := range Cleaners() {
		out = append(out, c.Name)
	}
	sort.Strings(out)
	return out
}
