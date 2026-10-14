package engine

import "strings"

// TextSoap's built-in groups (vendor help, "Cleaner Groups Summary"). The
// help names the groups and their purpose but not every member, so members
// follow the descriptions. "standard" and "custom" also list the user's own
// cleaners; the UI appends them.

type builtinGroup struct {
	id, name string
	tokens   []string
}

var builtinGroupDefs = []builtinGroup{
	{"standard", "标准", nil},
	{"basics", "基础", []string{"#SCRUB", "#MYSCRUB", "#REMOVE_EXTRA_SPACES", "#REMOVE_FORWARDING_CHARS", "#MAKE_PARAGRAPHS",
		"#REMOVE_EXTRA_RETURNS", "#TRIM_LINES", "#UPPERCASE", "#LOWERCASE", "#CAPITALIZE_SENTENCES", "#TITLE_CASE1",
		"#CAPITALIZE_WORDS", "#STRAIGHTEN_QUOTES", "#SMARTEN_QUOTES", "#REWRAP_TEXT", "#QUOTE_TEXT", "#CONVERT_NON_BREAKING_SPACES",
		"#REMOVE_DUPLICATE_LINES", "#LINE_SORT_ASCENDING", "#EXTRACT_TEXT_FROM_HTML"}},
	{"email", "邮件", []string{"#SCRUB", "#REMOVE_FORWARDING_CHARS", "#STRIP_QUOTING_CHARS", "#REMOVE_FORWARDED_TEXT", "#CONVERT_MIME",
		"#MAKE_PARAGRAPHS", "#REWRAP_TEXT", "#QUOTE_TEXT", "#QUOTE_L1", "#QUOTE_L2", "#QUOTE_L3", "#QUOTE_L4", "#QUOTE_BASED_FIRST_LINE",
		"#INCREASE_QUOTE_LEVEL", "#WRAP_AT_50", "#WRAP_AT_60", "#WRAP_AT_70", "#NETTEXT", "#REPLACE_AT_SIGN"}},
	{"typography", "排版", []string{"#STRAIGHTEN_QUOTES", "#SMARTEN_QUOTES", "#3PERIODS_TO_ELLIPSIS", "#ELLIPSIS_TO_3PERIODS",
		"#2HYPHENS_TO_EM_DASH", "#EM_DASH_TO_2HYPHENS", "#SPACE_DASH_SPACE_TO_EN_DASH", "#EN_DASH_TO_SPACE_DASH_SPACE",
		"#CONVERT_BULLETS", "#CONVERT_NON_BREAKING_SPACES", "#FIX_JAMMED_WORDS", "#ENABLE_LIGATURES", "#DISABLE_LIGATURES"}},
	{"case", "大小写转换", []string{"#UPPERCASE", "#LOWERCASE", "#CAPITALIZE_SENTENCES", "#CAPITALIZE_SENTENCES_ALT", "#TITLE_CASE1",
		"#CAPITALIZE_WORDS", "#CAPITALIZE_LINES", "#CAPITALIZE_LINES_ALT", "#CAPITALIZE_COMMON_TECH_NAMES", "#MACOS_NAME_FIX",
		"#APPLE_NAME_FIX", "#RANDOMCASE"}},
	{"quoting", "文本引用", []string{"#QUOTE_TEXT", "#QUOTE_L1", "#QUOTE_L2", "#QUOTE_L3", "#QUOTE_L4", "#QUOTE_BASED_FIRST_LINE",
		"#INCREASE_QUOTE_LEVEL", "#REMOVE_FORWARDING_CHARS", "#STRIP_QUOTING_CHARS", "#REMOVE_FORWARDED_TEXT"}},
	{"dates", "日期", []string{"#NORMALIZE_DATE_SHORTFORM", "#NORMALIZE_DATE_MEDIUMFORM", "#NORMALIZE_DATE_LONGFORM",
		"#NORMALIZE_DATE_FULLFORM", "#DATEFORMAT_US", "#DATEFORMAT_UK", "#DATEFORMAT_YEARFIRST", "#DATENOW"}},
	{"extraneous", "去除多余", []string{"#REMOVE_EXTRA_SPACES", "#REMOVE_WHITESPACE_AT_BEGINNING", "#REMOVE_ALL_TABS", "#REMOVE_ALL_SPACES",
		"#REMOVE_ALL_WHITESPACE", "#REMOVE_CONTROL_CHARS", "#REMOVE_EXTRA_RETURNS", "#REMOVE_HIASCII", "#TRIM_BEGINNING_OF_LINES",
		"#TRIM_END_OF_LINES", "#TRIM_LINES", "#STRIP_GT_1CR", "#STRIP_GT_2CR", "#STRIP_GT_3CR", "#STRIP_GT_1SPACE", "#STRIP_GT_2SPACE",
		"#STRIP_GT_3SPACE", "#STRIP_GT_1TAB", "#STRIP_GT_2TAB", "#STRIP_GT_3TAB", "#MULTIPLE_CR_TO_1CR", "#MULTIPLE_CR_TO_2CR",
		"#REMOVE_DUPLICATE_LINES", "#REMOVE_LINENUMBERS"}},
	{"rulers", "段落标尺", []string{"#LINESPACE_10", "#LINESPACE_12", "#LINESPACE_15", "#LINESPACE_20", "#PARAGRAPH_SINGLESPACE",
		"#PARAGRAPH_ALIGN_LEFT", "#PARAGRAPH_ALIGN_RIGHT", "#PARAGRAPH_ALIGN_CENTER", "#PARAGRAPH_ALIGN_FULL",
		"#PARAGRAPH_CLEARTABSTOPS", "#PARAGRAPH_RESETTABSTOPS"}},
	{"markdown", "Markdown", nil},
	{"html", "HTML", nil},
	{"bbcode", "BBCode", nil},
	{"plist", "Plist", nil},
	{"library", "库（全部）", nil},
	{"custom", "自定义", nil},
}

// BuiltinGroups returns the built-in groups with their cleaner names.
func BuiltinGroups() []Group {
	sections := map[string][]string{}
	var all []string
	for _, c := range Cleaners() {
		if c.Section == "deprecated" {
			continue
		}
		all = append(all, c.Name)
		if c.Section != "" {
			sections[c.Section] = append(sections[c.Section], c.Name)
		}
	}
	var out []Group
	for _, d := range builtinGroupDefs {
		g := Group{ID: d.id, Name: d.name, BuiltIn: true}
		names := []string{}
		switch d.id {
		case "standard":
			for _, t := range builtinGroupDefs[1].tokens {
				names = append(names, nameOf(t))
			}
		case "markdown", "html", "bbcode", "plist":
			names = sections[strings.ToLower(d.id)]
		case "library":
			names = all
		default:
			for _, t := range d.tokens {
				names = append(names, nameOf(t))
			}
		}
		for _, n := range names {
			if n != "" {
				g.Items = append(g.Items, GroupItem{Kind: "cleaner", Name: n})
			}
		}
		out = append(out, g)
	}
	return out
}

func nameOf(token string) string {
	if c := CleanerByName(token); c != nil {
		return c.Name
	}
	return ""
}
