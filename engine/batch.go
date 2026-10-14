package engine

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"strings"
)

// Rule is one row of a batch replace table.
type Rule struct {
	Disabled  bool   `json:"disabled,omitempty"`
	Find      string `json:"find"`
	Replace   string `json:"replace"`
	Regex     bool   `json:"regex,omitempty"`
	MatchCase bool   `json:"matchCase"`
	Note      string `json:"note,omitempty"`
}

// Options converts the rule to FindOptions. Regex rules get multiline ^/$
// so that "line start/end" works the way beginners expect in a table.
func (r Rule) Options() FindOptions {
	o := FindOptions{Find: r.Find, Replace: r.Replace, Regex: r.Regex, MatchCase: r.MatchCase}
	if r.Regex {
		o.RegexOptions = "m"
	}
	return o
}

// RuleSet is a saved batch-replace table.
type RuleSet struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Rules []Rule `json:"rules"`
}

// ApplyRules runs the rules top to bottom; each rule sees the output of the
// previous one. counts[i] is how many replacements rule i made. A rule with
// an invalid pattern is skipped and reported in the error.
func ApplyRules(text string, rules []Rule) (string, []int, error) {
	counts := make([]int, len(rules))
	var errs []string
	for i, r := range rules {
		if r.Disabled || r.Find == "" {
			continue
		}
		out, n, err := r.Options().Apply(text)
		if err != nil {
			errs = append(errs, fmt.Sprintf("第 %d 条：%v", i+1, err))
			continue
		}
		text, counts[i] = out, n
	}
	if len(errs) > 0 {
		return text, counts, fmt.Errorf("%s", strings.Join(errs, "；"))
	}
	return text, counts, nil
}

// ParseRules reads CSV or TSV (detected by the first line). Columns:
// find, replace, [regex], [match case], [note]. A header row (find/查找)
// is skipped. Boolean cells accept 1/true/yes/是/y/✓.
func ParseRules(data []byte) ([]Rule, error) {
	data = bytes.TrimPrefix(data, []byte("\ufeff"))
	first, _, _ := bytes.Cut(data, []byte("\n"))
	rd := csv.NewReader(bytes.NewReader(data))
	if bytes.Count(first, []byte("\t")) > 0 {
		rd.Comma = '\t'
		rd.LazyQuotes = true
	}
	rd.FieldsPerRecord = -1
	recs, err := rd.ReadAll()
	if err != nil {
		return nil, err
	}
	var out []Rule
	for i, rec := range recs {
		if len(rec) == 0 || (len(rec) == 1 && rec[0] == "") {
			continue
		}
		if i == 0 {
			h := strings.ToLower(strings.TrimSpace(rec[0]))
			if h == "find" || h == "查找" || h == "search" {
				continue
			}
		}
		r := Rule{Find: rec[0], MatchCase: true}
		if len(rec) > 1 {
			r.Replace = rec[1]
		}
		if len(rec) > 2 {
			r.Regex = truthy(rec[2])
		}
		if len(rec) > 3 && strings.TrimSpace(rec[3]) != "" {
			r.MatchCase = truthy(rec[3])
		}
		if len(rec) > 4 {
			r.Note = rec[4]
		}
		out = append(out, r)
	}
	return out, nil
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "y", "是", "✓", "x", "regex", "正则":
		return true
	}
	return false
}

// FormatRules writes rules as CSV (sep ',') or TSV (sep '\t') with a header.
func FormatRules(rules []Rule, sep rune) []byte {
	var b bytes.Buffer
	w := csv.NewWriter(&b)
	w.Comma = sep
	_ = w.Write([]string{"find", "replace", "regex", "match_case", "note"})
	for _, r := range rules {
		_ = w.Write([]string{r.Find, r.Replace, boolCell(r.Regex), boolCell(r.MatchCase), r.Note})
	}
	w.Flush()
	return b.Bytes()
}

func boolCell(b bool) string {
	if b {
		return "1"
	}
	return "0"
}
