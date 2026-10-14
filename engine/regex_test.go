package engine

import (
	"encoding/json"
	"os"
	"testing"
)

type regexCase struct {
	Probe        string `json:"probe"`
	Input        string `json:"input"`
	Find         string `json:"find"`
	Replace      string `json:"replace"`
	MatchCase    *bool  `json:"match_case"`
	RegexOptions string `json:"regex_options"`
	Type         *int   `json:"type"`
	Output       string `json:"output"`
}

// TestRegexCases runs the 51 measured TextSoap find/replace cases.
func TestRegexCases(t *testing.T) {
	data, err := os.ReadFile("testdata/regex_cases.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []regexCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	pass := 0
	for _, c := range cases {
		o := FindOptions{Find: c.Find, Replace: c.Replace, Regex: true, MatchCase: true, RegexOptions: c.RegexOptions}
		if c.MatchCase != nil {
			o.MatchCase = *c.MatchCase
		}
		if c.Type != nil {
			o.Regex = *c.Type != 0
		}
		got, _, err := o.Apply(c.Input)
		if err != nil || got != c.Output {
			t.Errorf("%s: got %q want %q (err %v)", c.Probe, got, c.Output, err)
			continue
		}
		pass++
	}
	t.Logf("regex cases: %d/%d pass", pass, len(cases))
}
