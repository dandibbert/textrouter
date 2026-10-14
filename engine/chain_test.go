package engine

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

// TestChainExamples runs the 15 chains executed on the real TextSoap engine.
func TestChainExamples(t *testing.T) {
	data, err := os.ReadFile("testdata/chain_examples.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		ID      string `json:"id"`
		Input   string `json:"input"`
		Output  string `json:"output"`
		Actions []any  `json:"actions"`
	}
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	pass := 0
	for _, c := range cases {
		ch := Chain{Steps: NodesToSteps(c.Actions)}
		got, _ := ch.Run(c.Input, nil)
		if got != c.Output {
			t.Errorf("%s: got %q want %q", c.ID, got, c.Output)
			continue
		}
		pass++
	}
	t.Logf("chain examples: %d/%d", pass, len(cases))
}

func TestBatchRulesAndCSV(t *testing.T) {
	rules := []Rule{
		{Find: "colour", Replace: "color", MatchCase: false},
		{Find: `(\d+)元`, Replace: "¥$1", Regex: true, MatchCase: true},
		{Find: "x", Replace: "y", Disabled: true},
	}
	out, counts, err := ApplyRules("Colour 5元 x", rules)
	if err != nil || out != "color ¥5 x" || counts[0] != 1 || counts[1] != 1 || counts[2] != 0 {
		t.Fatalf("got %q %v %v", out, counts, err)
	}
	for _, sep := range []rune{',', '\t'} {
		back, err := ParseRules(FormatRules(rules, sep))
		if err != nil || len(back) != 3 || back[1].Find != `(\d+)元` || !back[1].Regex || back[0].MatchCase {
			t.Errorf("roundtrip %q: %+v %v", sep, back, err)
		}
	}
}

func TestTextSoapRoundTrip(t *testing.T) {
	in := []Chain{{Name: "Demo", Steps: []Step{
		{Kind: KindFind, Find: FindOptions{Find: "a.", Replace: "b", Regex: true, MatchCase: true}},
		{Kind: KindCleaner, Cleaner: "Uppercase"},
	}}}
	data, err := ExportTextSoapLibrary(in)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`<key>fileVersion</key>\s*<integer>6</integer>`).Match(data) {
		t.Errorf("fileVersion 6 missing:\n%s", data)
	}
	out, err := ImportTextSoapLibrary(data)
	if err != nil || len(out) != 1 {
		t.Fatal(out, err)
	}
	got, _ := out[0].Run("xa1", nil)
	if got != "XB" {
		t.Errorf("got %q", got)
	}
}

func TestChainTraceAndContainers(t *testing.T) {
	ch := Chain{Steps: []Step{
		{Kind: KindEachLine, Children: []Step{{Kind: KindCleaner, Cleaner: "Capitalize Words"}}},
		{Kind: KindIfMatch, Find: FindOptions{Find: `\d+`, Regex: true, MatchCase: true}, Children: []Step{{Kind: KindTag, Text: "[", Text2: "]"}}},
		{Kind: KindLineSort, Flag: true},
		{Kind: KindNote, Text: "noop"},
	}}
	got, trace := ch.Run("hello 1\nworld 22", nil)
	if got != "World [22]\nHello [1]\n" || len(trace) != 4 {
		t.Errorf("got %q trace %d", got, len(trace))
	}
}
