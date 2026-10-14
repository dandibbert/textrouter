package engine

import (
	"fmt"
	"sort"
	"strings"
	"unicode"
)

// Step kinds. Ids follow TextSoap's action ids where one exists.
const (
	KindCleaner    = "applycleaner"
	KindFind       = "find"
	KindDelete     = "find.delete"
	KindBatch      = "find.batch"
	KindInsert     = "text.insert"
	KindTag        = "text.tag"
	KindLineAdd    = "line.addto"
	KindLineRemove = "line.removefrom"
	KindLineSort   = "line.sort"
	KindExtract    = "text.extract"
	KindWrap       = "text.wrap"
	KindQuote      = "text.quote"
	KindTitleCase  = "text.titlecase"
	KindSubstring  = "text.substring"
	KindGroup      = "runtime.group"
	KindEachLine   = "if.match.text.processlines"
	KindIfMatch    = "if.match.text.regex"
	KindRepeat     = "runtime.repeat"
	KindNote       = "runtime.usernote"
	KindChain      = "applychain"
)

// Step is one action of a chain. Fields are shared between kinds to keep
// the JSON and the editor simple; ActionInfo documents which ones a kind uses.
type Step struct {
	Kind     string      `json:"kind"`
	Disabled bool        `json:"disabled,omitempty"`
	Cleaner  string      `json:"cleaner,omitempty"`
	Find     FindOptions `json:"find,omitzero"`
	Text     string      `json:"text,omitempty"`
	Text2    string      `json:"text2,omitempty"`
	Number   int         `json:"number,omitempty"`
	Number2  int         `json:"number2,omitempty"`
	Flag     bool        `json:"flag,omitempty"`  // e.g. at end / descending
	Flag2    bool        `json:"flag2,omitempty"` // e.g. unique / ignore case
	Rules    []Rule      `json:"rules,omitempty"`
	Children []Step      `json:"children,omitempty"`
}

// Chain is a named, user-defined sequence of steps (a TextSoap "custom cleaner").
type Chain struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Note  string `json:"note,omitempty"`
	Steps []Step `json:"steps"`
}

// StepResult records what one top-level step did, for live previews.
type StepResult struct {
	Output string
	Count  int // replacements / lines changed, -1 when not meaningful
	Err    error
}

// ChainResolver lets applychain steps call other chains by id.
type ChainResolver func(id string) *Chain

// Run executes the chain and reports the text after every top-level step.
func (c *Chain) Run(text string, resolve ChainResolver) (string, []StepResult) {
	var trace []StepResult
	for _, s := range c.Steps {
		out, n, err := s.Apply(text, resolve, 0)
		if err == nil {
			text = out
		}
		trace = append(trace, StepResult{Output: text, Count: n, Err: err})
	}
	return text, trace
}

// Apply runs one step. Disabled and unknown steps leave text unchanged.
func (s Step) Apply(text string, resolve ChainResolver, depth int) (string, int, error) {
	if s.Disabled {
		return text, -1, nil
	}
	if depth > 16 {
		return text, -1, fmt.Errorf("嵌套过深")
	}
	runChildren := func(t string) (string, error) {
		for _, ch := range s.Children {
			out, _, err := ch.Apply(t, resolve, depth+1)
			if err != nil {
				return t, err
			}
			t = out
		}
		return t, nil
	}
	switch s.Kind {
	case KindCleaner:
		out := ApplyCleaner(s.Cleaner, text)
		return out, -1, nil
	case KindFind:
		return s.Find.Apply(text)
	case KindDelete:
		f := s.Find
		f.Replace = ""
		return f.Apply(text)
	case KindBatch:
		out, counts, err := ApplyRules(text, s.Rules)
		n := 0
		for _, c := range counts {
			n += c
		}
		return out, n, err
	case KindInsert:
		if s.Flag {
			return text + s.Text, -1, nil
		}
		return s.Text + text, -1, nil
	case KindTag:
		return s.Text + text + s.Text2, -1, nil
	case KindLineAdd:
		return mapLines(text, func(l string) string {
			if s.Flag {
				return l + s.Text
			}
			return s.Text + l
		}), -1, nil
	case KindLineRemove:
		return mapLines(text, func(l string) string {
			if s.Text != "" {
				if s.Flag {
					return strings.TrimSuffix(l, s.Text)
				}
				return strings.TrimPrefix(l, s.Text)
			}
			r := []rune(l)
			n := min(max(s.Number, 0), len(r))
			if s.Flag {
				return string(r[:len(r)-n])
			}
			return string(r[n:])
		}), -1, nil
	case KindLineSort:
		return SortLines(text, s.Flag, s.Flag2, s.Number == 1), -1, nil
	case KindExtract:
		f := s.Find
		ms, err := f.FindAll(text, 0)
		if err != nil {
			return text, 0, err
		}
		sep := s.Text
		if sep == "" {
			sep = "\n"
		}
		var parts []string
		for _, m := range ms {
			if f.Replace != "" {
				parts = append(parts, m.Result)
			} else {
				parts = append(parts, m.Groups[0].Text)
			}
		}
		return strings.Join(parts, sep), len(ms), nil
	case KindWrap:
		w := s.Number
		if w <= 0 {
			w = 70
		}
		return WrapAt(text, w), -1, nil
	case KindQuote:
		w := s.Number
		if w <= 0 {
			w = 70
		}
		p := s.Text
		if p == "" {
			p = "> "
		}
		return QuoteText(text, w, p), -1, nil
	case KindTitleCase:
		return TitleCase(text, splitList(s.Text), splitList(s.Text2)), -1, nil
	case KindSubstring:
		r := []rune(text)
		st := min(max(s.Number, 0), len(r))
		end := len(r)
		if s.Number2 > 0 {
			end = min(st+s.Number2, len(r))
		}
		return string(r[st:end]), -1, nil
	case KindGroup:
		out, err := runChildren(text)
		return out, -1, err
	case KindRepeat:
		n := max(s.Number, 1)
		for range min(n, 100) {
			out, err := runChildren(text)
			if err != nil {
				return text, -1, err
			}
			text = out
		}
		return text, -1, nil
	case KindEachLine:
		ls := strings.Split(text, "\n")
		for i, l := range ls {
			out, err := runChildren(l)
			if err != nil {
				return text, -1, err
			}
			ls[i] = out
		}
		return strings.Join(ls, "\n"), len(ls), nil
	case KindIfMatch:
		f := s.Find
		ms, err := f.FindAll(text, 0)
		if err != nil {
			return text, 0, err
		}
		for i := range ms {
			out, err := runChildren(ms[i].Groups[0].Text)
			if err != nil {
				return text, 0, err
			}
			ms[i].Result = out
		}
		return ApplyMatches(text, ms), len(ms), nil
	case KindChain:
		if resolve == nil {
			return text, -1, nil
		}
		if ch := resolve(s.Text); ch != nil {
			for _, st := range ch.Steps {
				out, _, err := st.Apply(text, resolve, depth+1)
				if err != nil {
					return text, -1, err
				}
				text = out
			}
		}
		return text, -1, nil
	}
	return text, -1, nil // KindNote and unknown actions: no-op (as TextSoap)
}

// SortLines sorts lines. The measured TextSoap default (ascending) ends the
// result with a newline.
func SortLines(text string, desc, ignoreCase, unique bool) string {
	ls := strings.Split(strings.TrimSuffix(text, "\n"), "\n")
	key := func(s string) string {
		if ignoreCase {
			return strings.ToLower(s)
		}
		return s
	}
	sort.SliceStable(ls, func(i, j int) bool {
		if desc {
			return key(ls[i]) > key(ls[j])
		}
		return key(ls[i]) < key(ls[j])
	})
	if unique {
		ls = strings.Split(removeDuplicateLines(strings.Join(ls, "\n")), "\n")
	}
	return strings.Join(ls, "\n") + "\n"
}

func splitList(s string) []string {
	return strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == '，' || unicode.IsSpace(r) })
}

// ActionInfo describes a step kind for the editor and the palette.
type ActionInfo struct {
	Kind, Title, Desc string
	Container         bool
	Fields            []string // which Step fields the editor shows
}

// Actions lists the step kinds the chain editor offers.
func Actions() []ActionInfo {
	return []ActionInfo{
		{KindCleaner, "应用清理器", "运行一个内置清理器", false, []string{"cleaner"}},
		{KindFind, "查找替换", "按文字或正则查找并替换", false, []string{"find", "replace", "regex"}},
		{KindDelete, "查找并删除", "删除所有匹配的文字", false, []string{"find", "regex"}},
		{KindBatch, "批量替换（规则表）", "按规则表从上到下依次替换", false, []string{"rules"}},
		{KindExtract, "提取匹配", "只保留匹配到的内容，每条一行", false, []string{"find", "regex", "sep"}},
		{KindInsert, "插入文字", "在开头或末尾插入文字", false, []string{"text", "atEnd"}},
		{KindTag, "包裹文字", "在前后加上前缀和后缀，如 <b> 和 </b>", false, []string{"text", "text2"}},
		{KindLineAdd, "给每行添加文字", "在每行开头或末尾加文字", false, []string{"text", "atEnd"}},
		{KindLineRemove, "从每行删除", "删掉每行开头/末尾的指定文字或 N 个字符", false, []string{"text", "number", "atEnd"}},
		{KindLineSort, "行排序", "升序/降序，可忽略大小写、去重", false, []string{"desc", "ignoreCase", "unique"}},
		{KindWrap, "按宽度折行", "合并段落后按指定宽度折行", false, []string{"width"}},
		{KindQuote, "引用文本", "折行并在每行前加引用符", false, []string{"width", "prefix"}},
		{KindTitleCase, "标题式大小写（扩展）", "可指定额外小词和例外词", false, []string{"small", "exceptions"}},
		{KindSubstring, "截取字符", "从第 N 个字符起截取若干字符", false, []string{"start", "length"}},
		{KindEachLine, "逐行处理", "对每一行分别执行子步骤", true, nil},
		{KindIfMatch, "对匹配内容处理", "只对匹配到的片段执行子步骤", true, []string{"find", "regex"}},
		{KindGroup, "分组", "把若干步骤放在一起，便于整体开关", true, nil},
		{KindRepeat, "重复", "把子步骤重复执行 N 次", true, []string{"times"}},
		{KindChain, "调用其他动作链", "运行另一条已保存的动作链", false, []string{"chain"}},
		{KindNote, "备注", "只是说明，不改变文本", false, []string{"text"}},
	}
}

// ActionByKind looks up ActionInfo.
func ActionByKind(kind string) ActionInfo {
	for _, a := range Actions() {
		if a.Kind == kind {
			return a
		}
	}
	return ActionInfo{Kind: kind, Title: kind, Desc: "未知动作（不会改变文本）"}
}

// Summary is a one-line human description of a step.
func (s Step) Summary() string {
	a := ActionByKind(s.Kind)
	switch s.Kind {
	case KindCleaner:
		if c := CleanerByName(s.Cleaner); c != nil {
			return c.ZhName
		}
		return s.Cleaner
	case KindFind:
		return fmt.Sprintf("「%s」→「%s」", s.Find.Find, s.Find.Replace)
	case KindDelete, KindExtract, KindIfMatch:
		return a.Title + "「" + s.Find.Find + "」"
	case KindBatch:
		return fmt.Sprintf("批量替换 · %d 条规则", len(s.Rules))
	case KindInsert, KindLineAdd:
		return a.Title + "「" + s.Text + "」"
	case KindTag:
		return s.Text + "…" + s.Text2
	}
	return a.Title
}
