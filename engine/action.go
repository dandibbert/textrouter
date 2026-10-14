package engine

import (
	"context"
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

// Action is one node of a custom cleaner, in TextSoap's own shape
// (identifier + parameters + enabled + nested actionList). Parameter keys
// follow TextSoap where they are known (find, replace, type, match_case,
// granularity, regex_options, list_name); the others are TextRouter's and
// are listed in ActionCatalog.
type Action struct {
	ID       string   `json:"identifier"`
	Params   Params   `json:"parameters"`
	Enabled  bool     `json:"enabled"`
	Comment  string   `json:"comment,omitempty"`
	Children []Action `json:"actionList,omitempty"`
}

// Params are an action's parameters.
type Params map[string]any

// Str returns a string parameter.
func (p Params) Str(k, def string) string {
	if v, ok := p[k]; ok && v != nil {
		return asString(v)
	}
	return def
}

// Int returns an integer parameter.
func (p Params) Int(k string, def int) int {
	if v, ok := p[k]; ok && v != nil {
		return asInt(v, def)
	}
	return def
}

// Bool returns a boolean parameter.
func (p Params) Bool(k string, def bool) bool { return asBool(p[k], def) }

// CustomCleaner is a user-defined cleaner: a named list of actions.
type CustomCleaner struct {
	UUID     string    `json:"uuid"`
	Name     string    `json:"name"`
	Author   string    `json:"author,omitempty"`
	Comment  string    `json:"comment,omitempty"`
	Actions  []Action  `json:"actions"`
	Modified time.Time `json:"modified"`
}

// List is the table of a Define List action: up to 4 columns, A–D.
type List struct {
	Name    string
	Columns int
	Rows    [][]string
}

// Clipboard is what clipboard actions use.
type Clipboard interface {
	ReadText() string
	WriteText(string)
}

// Env carries what a running cleaner can reach.
type Env struct {
	// Custom finds a custom cleaner by name (for Apply Cleaner).
	Custom func(name string) *CustomCleaner
	// MyScrub is the cleaner name #MYSCRUB stands for ("" = Scrub).
	MyScrub   string
	Clipboard Clipboard
	// AllowScripts enables Run Shell Script.
	AllowScripts bool
	Warnings     []string

	lists  map[string]List
	macros map[string][]Action
	vars   []string
	depth  int
}

func (e *Env) warn(f string, a ...any) {
	msg := fmt.Sprintf(f, a...)
	for _, w := range e.Warnings {
		if w == msg {
			return
		}
	}
	e.Warnings = append(e.Warnings, msg)
}

// StepTrace is what one top-level action did.
type StepTrace struct {
	Output string `json:"output"`
	Count  int    `json:"count"`
	Error  string `json:"error,omitempty"`
}

// Run executes a custom cleaner. Lists and macros may be defined anywhere in
// the cleaner, so they are collected before the first action runs.
func (c *CustomCleaner) Run(text string, env *Env) (string, []StepTrace) {
	return RunActions(c.Actions, text, env)
}

// RunActions executes a list of actions and traces the top-level ones.
func RunActions(actions []Action, text string, env *Env) (string, []StepTrace) {
	if env == nil {
		env = &Env{}
	}
	if env.lists == nil {
		env.lists = map[string]List{}
		env.macros = map[string][]Action{}
	}
	collectDefinitions(actions, env)
	var trace []StepTrace
	for _, a := range actions {
		out, n, err := env.apply(a, text)
		st := StepTrace{Count: n}
		if err != nil {
			st.Error = err.Error()
		} else {
			text = out
		}
		st.Output = text
		trace = append(trace, st)
	}
	return text, trace
}

func collectDefinitions(actions []Action, env *Env) {
	for _, a := range actions {
		switch a.ID {
		case "runtime.definelist":
			l := ListFromParams(a.Params)
			env.lists[l.Name] = l
		case "runtime.macro":
			env.macros[a.Params.Str("name", "")] = a.Children
			continue
		case "runtime.macrolist":
			for _, g := range a.Children {
				env.macros[g.Params.Str("name", "")] = g.Children
			}
			continue
		}
		collectDefinitions(a.Children, env)
	}
}

// ListFromParams reads a Define List action: name, columns, rows (a list of
// rows, each a list of cells) or text (tab-delimited lines).
func ListFromParams(p Params) List {
	l := List{Name: p.Str("name", ""), Columns: min(max(p.Int("columns", 2), 1), 4)}
	switch rows := p["rows"].(type) {
	case []any:
		for _, r := range rows {
			var row []string
			if cells, ok := r.([]any); ok {
				for _, c := range cells {
					row = append(row, asString(c))
				}
			}
			l.Rows = append(l.Rows, row)
		}
	case [][]string:
		l.Rows = rows
	}
	if l.Rows == nil && p.Str("text", "") != "" {
		l.Rows = ParseListText(p.Str("text", ""))
	}
	return l
}

// ParseListText reads tab-delimited lines into rows (columns past 4 are
// ignored, as in TextSoap's text editor for lists).
func ParseListText(s string) [][]string {
	var rows [][]string
	for _, line := range strings.Split(strings.ReplaceAll(s, "\r\n", "\n"), "\n") {
		if line == "" {
			continue
		}
		cells := strings.Split(line, "\t")
		if len(cells) > 4 {
			cells = cells[:4]
		}
		rows = append(rows, cells)
	}
	return rows
}

var listVar = regexp.MustCompile(`(\\?)\$v\{(?:col:)?([A-Da-d])\}`)

// substVars replaces $v{A}…$v{D} (also $v{col:A}) with the current row's
// cells. A backslash before $v{…} keeps it as text.
func (e *Env) substVars(s string) string {
	if e.vars == nil || !strings.Contains(s, "$v{") {
		return s
	}
	return listVar.ReplaceAllStringFunc(s, func(m string) string {
		sub := listVar.FindStringSubmatch(m)
		if sub[1] != "" {
			return m[1:]
		}
		i := int(unicode.ToUpper(rune(sub[2][0])) - 'A')
		if i < len(e.vars) {
			return e.vars[i]
		}
		return ""
	})
}

// FindOptionsFrom reads TextSoap's find parameters. Measured defaults:
// type missing = regex, match_case missing = true.
func FindOptionsFrom(p Params, forceRegex bool) FindOptions {
	o := FindOptions{
		Find:         p.Str("find", ""),
		Replace:      p.Str("replace", ""),
		Regex:        p.Int("type", 1) != 0,
		MatchCase:    p.Bool("match_case", true),
		RegexOptions: p.Str("regex_options", ""),
		Granularity:  p.Int("granularity", 0),
	}
	if forceRegex {
		o.Regex = true
	}
	return o
}

func (e *Env) findOpts(p Params, forceRegex bool) FindOptions {
	o := FindOptionsFrom(p, forceRegex)
	o.Find, o.Replace = e.substVars(o.Find), e.substVars(o.Replace)
	return o
}

func (e *Env) runChildren(children []Action, text string) (string, error) {
	for _, ch := range children {
		out, _, err := e.apply(ch, text)
		if err != nil {
			return text, err
		}
		text = out
	}
	return text, nil
}

// apply runs one action. Disabled and unknown actions leave text unchanged
// (measured for unknown ids).
func (e *Env) apply(a Action, text string) (string, int, error) {
	if !a.Enabled {
		return text, -1, nil
	}
	if e.depth > 32 {
		return text, -1, fmt.Errorf("嵌套过深（可能是清理器互相调用）")
	}
	e.depth++
	defer func() { e.depth-- }()
	p := a.Params
	if p == nil {
		p = Params{}
	}
	switch a.ID {
	case "find", "find.regex", "find.delete", "find.batch", "find.batch.regex":
		forceRegex := a.ID == "find.regex" || a.ID == "find.batch.regex"
		if a.ID == "find.batch" && p["type"] == nil {
			p = withDefault(p, "type", 0)
		}
		replaceWith := func(o FindOptions) FindOptions {
			if a.ID == "find.delete" {
				o.Replace = ""
			}
			return o
		}
		if name := p.Str("list_name", ""); name != "" {
			l, ok := e.lists[name]
			if !ok {
				return text, 0, fmt.Errorf("找不到列表「%s」", name)
			}
			total := 0
			saved := e.vars
			for _, row := range l.Rows {
				e.vars = row
				o := replaceWith(e.findOpts(p, forceRegex))
				if o.Find == "" {
					continue
				}
				out, n, err := o.Apply(text)
				if err != nil {
					e.vars = saved
					return text, total, err
				}
				text, total = out, total+n
			}
			e.vars = saved
			return text, total, nil
		}
		return replaceWith(e.findOpts(p, forceRegex)).Apply(text)

	case "applycleaner", "applycleaner.static":
		name := p.Str("cleaner", "")
		return e.applyCleaner(name, text), -1, nil

	case "text.insert":
		s := UnescapeText(e.substVars(p.Str("text", "")))
		switch p.Int("position", 0) {
		case 1:
			return text + s, -1, nil
		case 2:
			return s, -1, nil
		}
		return s + text, -1, nil
	case "text.delete":
		return "", -1, nil
	case "text.tag":
		return UnescapeText(e.substVars(p.Str("prefix", ""))) + text + UnescapeText(e.substVars(p.Str("suffix", ""))), -1, nil
	case "text.extract":
		o := FindOptions{Find: e.substVars(p.Str("find", "")), Regex: true, MatchCase: p.Bool("match_case", true), RegexOptions: p.Str("regex_options", "m")}
		ms, err := o.FindAll(text, 0)
		if err != nil {
			return text, 0, err
		}
		g := p.Int("group", 0)
		var parts []string
		for _, m := range ms {
			if g < len(m.Groups) && m.Groups[g].Ok {
				parts = append(parts, m.Groups[g].Text)
			}
		}
		res := strings.Join(parts, UnescapeText(p.Str("separator", `\n`)))
		if p.Bool("append", false) {
			if res == "" {
				return text, len(ms), nil
			}
			return strings.TrimRight(text, "\n") + "\n" + res, len(ms), nil
		}
		return res, len(ms), nil
	case "text.substring":
		r := []rune(text)
		n := max(p.Int("count", 1), 0)
		switch p.Int("mode", 0) {
		case 1:
			return string(r[max(len(r)-n, 0):]), -1, nil
		case 2:
			st := min(max(p.Int("start", 1)-1, 0), len(r))
			end := min(st+max(p.Int("length", 1), 0), len(r))
			return string(r[st:end]), -1, nil
		}
		return string(r[:min(n, len(r))]), -1, nil
	case "text.wrap":
		return WrapAt(text, max(p.Int("width", 70), 1)), -1, nil
	case "text.quote":
		pre := UnescapeText(p.Str("prefix", "")) + strings.Repeat(p.Str("mark", ">"), min(max(p.Int("level", 1), 0), 10)) + UnescapeText(p.Str("suffix", " "))
		if p.Bool("wrap", true) {
			return QuoteText(text, max(p.Int("width", 70), 10), pre), -1, nil
		}
		return mapLines(text, func(l string) string { return pre + l }), -1, nil
	case "text.titlecase":
		return TitleCase(text, splitList(p.Str("small_words", "")), splitList(p.Str("special_words", ""))), -1, nil
	case "text.copytoclip":
		if e.Clipboard != nil {
			e.Clipboard.WriteText(text)
		}
		return text, -1, nil
	case "text.appendtoclip":
		if e.Clipboard != nil {
			e.Clipboard.WriteText(e.Clipboard.ReadText() + UnescapeText(p.Str("separator", "")) + text)
		}
		return text, -1, nil
	case "text.pastefromclip":
		if e.Clipboard != nil {
			return e.Clipboard.ReadText(), -1, nil
		}
		return text, -1, nil

	case "line.addto":
		s := UnescapeText(e.substVars(p.Str("text", "")))
		end := p.Int("position", 0) == 1
		return mapContentLines(text, func(l string) string {
			if end {
				return l + s
			}
			return s + l
		}), -1, nil
	case "line.removefrom":
		s := UnescapeText(e.substVars(p.Str("text", "")))
		end := p.Int("position", 0) == 1
		return mapContentLines(text, func(l string) string {
			if end {
				return strings.TrimSuffix(l, s)
			}
			return strings.TrimPrefix(l, s)
		}), -1, nil
	case "line.sort":
		return e.sortLines(text, p)
	case "line.number":
		return numberLines(text, p), -1, nil
	case "line.loop", "if.match.text.processlines":
		skipBlank := a.ID == "if.match.text.processlines"
		ls := strings.Split(text, "\n")
		n := 0
		for i, l := range ls {
			if skipBlank && strings.TrimSpace(l) == "" {
				continue
			}
			out, err := e.runChildren(a.Children, l)
			if err != nil {
				return text, n, err
			}
			ls[i] = out
			n++
		}
		return strings.Join(ls, "\n"), n, nil

	case "if.match.text", "if.match.text.regex":
		o := e.findOpts(p, a.ID == "if.match.text.regex")
		if a.ID == "if.match.text" && p["type"] == nil {
			o.Regex = false
		}
		ms, err := o.FindAll(text, 0)
		if err != nil {
			return text, 0, err
		}
		g := p.Int("group", 0)
		var rs []span
		for _, m := range ms {
			if g < len(m.Groups) && m.Groups[g].Ok {
				rs = append(rs, span{m.Groups[g].Start, m.Groups[g].End})
			}
		}
		return e.applyToSpans(text, rs, modifiers(a), a.Children)
	case "if.match.url":
		return e.applyToPattern(text, urlPattern, a)
	case "if.match.date":
		return e.applyToPattern(text, datePattern, a)

	case "runtime.group":
		out, err := e.runChildren(a.Children, text)
		return out, -1, err
	case "runtime.repeat":
		n := min(max(p.Int("count", 1), 0), 1000)
		for range n {
			out, err := e.runChildren(a.Children, text)
			if err != nil {
				return text, -1, err
			}
			if out == text && p.Bool("until_unchanged", false) {
				break
			}
			text = out
		}
		return text, -1, nil
	case "runtime.processlist":
		name := p.Str("list_name", "")
		l, ok := e.lists[name]
		if !ok {
			return text, 0, fmt.Errorf("找不到列表「%s」", name)
		}
		saved := e.vars
		defer func() { e.vars = saved }()
		for _, row := range l.Rows {
			e.vars = row
			out, err := e.runChildren(a.Children, text)
			if err != nil {
				return text, -1, err
			}
			text = out
		}
		return text, len(l.Rows), nil
	case "runtime.applymacro":
		name := p.Str("name", "")
		m, ok := e.macros[name]
		if !ok {
			return text, -1, fmt.Errorf("找不到宏「%s」", name)
		}
		out, err := e.runChildren(m, text)
		return out, -1, err
	case "runtime.shellscript":
		if !e.AllowScripts {
			e.warn("「运行 Shell 脚本」默认关闭，可在设置中开启")
			return text, -1, nil
		}
		return runShell(p.Str("script", ""), text)
	case "runtime.definelist", "runtime.macro", "runtime.macrolist", "runtime.usernote",
		"modify.results.invert", "modify.results.limit", "modify.results.skip", "if.end":
		return text, -1, nil
	}
	if info, ok := actionIndex()[a.ID]; ok && info.Support != SupportFull {
		e.warn("「%s」在纯文本中不起作用，已跳过", info.Title)
	}
	return text, -1, nil
}

func withDefault(p Params, k string, v any) Params {
	q := Params{}
	for kk, vv := range p {
		q[kk] = vv
	}
	q[k] = v
	return q
}

func (e *Env) applyCleaner(name, text string) string {
	if name == "MyScrub" || name == "#MYSCRUB" {
		name = e.MyScrub
		if name == "" || name == "MyScrub" {
			name = "Scrub"
		}
	}
	if c := CleanerByName(name); c != nil {
		return c.Fn(text)
	}
	if e.Custom != nil {
		if cc := e.Custom(name); cc != nil {
			out, err := e.runChildren(cc.Actions, text)
			if err == nil {
				return out
			}
		}
	}
	return text
}

type span struct{ start, end int }

type resultMods struct {
	invert      bool
	skip, limit int
}

func modifiers(a Action) resultMods {
	m := resultMods{invert: a.Params.Bool("invert", false), skip: a.Params.Int("skip", 0), limit: a.Params.Int("limit", 0)}
	for _, ch := range a.Children {
		switch ch.ID {
		case "modify.results.invert":
			m.invert = m.invert || ch.Enabled
		case "modify.results.skip":
			if ch.Enabled {
				m.skip = ch.Params.Int("count", ch.Params.Int("skip", 0))
			}
		case "modify.results.limit":
			if ch.Enabled {
				m.limit = ch.Params.Int("count", ch.Params.Int("limit", 0))
			}
		}
	}
	return m
}

// applyToSpans runs children on each span (rune offsets) and splices the
// results back. Skip/limit pick spans; invert switches to the text between
// them.
func (e *Env) applyToSpans(text string, rs []span, m resultMods, children []Action) (string, int, error) {
	if m.skip > 0 {
		rs = rs[min(m.skip, len(rs)):]
	}
	if m.limit > 0 && len(rs) > m.limit {
		rs = rs[:m.limit]
	}
	r := []rune(text)
	if m.invert {
		var inv []span
		last := 0
		for _, s := range rs {
			if s.start > last {
				inv = append(inv, span{last, s.start})
			}
			last = max(last, s.end)
		}
		if last < len(r) {
			inv = append(inv, span{last, len(r)})
		}
		rs = inv
	}
	var b strings.Builder
	last := 0
	n := 0
	for _, s := range rs {
		if s.start < last {
			continue
		}
		b.WriteString(string(r[last:s.start]))
		out, err := e.runChildren(children, string(r[s.start:s.end]))
		if err != nil {
			return text, n, err
		}
		b.WriteString(out)
		last = s.end
		n++
	}
	b.WriteString(string(r[last:]))
	return b.String(), n, nil
}

var (
	urlPattern  = `(?i)\b(?:https?://|ftp://|mailto:|www\.)[^\s<>"'（）「」，。]+[^\s<>"'（）「」，。.,;:!?)\]]`
	datePattern = `\b(?:\d{4}[-/.]\d{1,2}[-/.]\d{1,2}|\d{1,2}[-/.]\d{1,2}[-/.]\d{2,4}|(?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]*\.? \d{1,2},? \d{4}|\d{1,2} (?:Jan|Feb|Mar|Apr|May|Jun|Jul|Aug|Sep|Oct|Nov|Dec)[a-z]* \d{4})\b|\d{4}年\d{1,2}月\d{1,2}日`
)

func (e *Env) applyToPattern(text, pat string, a Action) (string, int, error) {
	o := FindOptions{Find: pat, Regex: true, MatchCase: true}
	ms, err := o.FindAll(text, 0)
	if err != nil {
		return text, 0, err
	}
	var rs []span
	for _, m := range ms {
		rs = append(rs, span{m.Start, m.End})
	}
	return e.applyToSpans(text, rs, modifiers(a), a.Children)
}

// mapContentLines maps every line, leaving a final empty line (after a
// trailing newline) alone.
func mapContentLines(s string, f func(string) string) string {
	trail := strings.HasSuffix(s, "\n")
	body := strings.TrimSuffix(s, "\n")
	ls := strings.Split(body, "\n")
	for i := range ls {
		ls[i] = f(ls[i])
	}
	out := strings.Join(ls, "\n")
	if trail {
		out += "\n"
	}
	return out
}

// sortLines implements Sort Lines. Measured: with default parameters the
// result is ascending and ends with a newline.
func (e *Env) sortLines(text string, p Params) (string, int, error) {
	ls := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	var opts []collate.Option
	if !p.Bool("match_case", false) {
		opts = append(opts, collate.IgnoreCase)
	}
	if p.Bool("ignore_diacritics", false) {
		opts = append(opts, collate.IgnoreDiacritics)
	}
	if p.Bool("numeric", false) {
		opts = append(opts, collate.Numeric)
	}
	col := collate.New(language.Und, opts...)
	keyOf := func(l string) string { return l }
	if p.Bool("ignore_whitespace", false) {
		prev := keyOf
		keyOf = func(l string) string { return strings.TrimLeftFunc(prev(l), unicode.IsSpace) }
	}
	if p.Bool("use_expression", false) && p.Str("find", "") != "" {
		o := FindOptions{Find: p.Str("find", ""), Replace: p.Str("replace", "$0"), Regex: true, MatchCase: true}
		if _, err := o.Compile(); err != nil {
			return text, -1, err
		}
		prev := keyOf
		keyOf = func(l string) string {
			k := prev(l)
			ms, _ := o.FindAll(k, 1)
			if len(ms) == 0 {
				return k
			}
			return ms[0].Result
		}
	}
	keys := make([]string, len(ls))
	for i, l := range ls {
		keys[i] = keyOf(l)
	}
	idx := make([]int, len(ls))
	for i := range idx {
		idx[i] = i
	}
	desc := p.Bool("descending", false)
	sort.SliceStable(idx, func(i, j int) bool {
		c := col.CompareString(keys[idx[i]], keys[idx[j]])
		if desc {
			return c > 0
		}
		return c < 0
	})
	out := make([]string, len(ls))
	for i, k := range idx {
		out[i] = ls[k]
	}
	return strings.Join(out, "\n") + "\n", -1, nil
}

// numberLines implements Number Lines. format: a run of 0s is the number,
// padded to the run's length; \t and \s are tab and space.
func numberLines(text string, p Params) string {
	n := p.Int("start", 1)
	inc := p.Int("increment", 1)
	nonEmpty := p.Int("lines", 0) == 1
	format := p.Str("format", `0.\s`)
	zeros := regexp.MustCompile(`0+`)
	loc := zeros.FindStringIndex(format)
	return mapContentLines(text, func(l string) string {
		if nonEmpty && strings.TrimSpace(l) == "" {
			return l
		}
		var label string
		if loc == nil {
			label = strconv.Itoa(n) + UnescapeText(format)
		} else {
			num := fmt.Sprintf("%0*d", loc[1]-loc[0], n)
			label = UnescapeText(format[:loc[0]]) + num + UnescapeText(format[loc[1]:])
		}
		n += inc
		return label + l
	})
}

func runShell(script, text string) (string, int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "/bin/sh", "-c", script)
	cmd.Stdin = strings.NewReader(text)
	out, err := cmd.Output()
	if err != nil {
		return text, -1, fmt.Errorf("脚本出错：%v", err)
	}
	return string(out), -1, nil
}

func splitList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '，' || unicode.IsSpace(r) })
}
