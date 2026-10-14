package engine

import "testing"

func TestExplainPattern(t *testing.T) {
	parts := ExplainPattern(`^(\d+)-[a-z]{2,}?$`, true)
	if len(parts) < 6 {
		t.Fatalf("parts %+v", parts)
	}
	for _, p := range parts {
		if p.Kind == "error" {
			t.Errorf("unexpected error part %+v", p)
		}
	}
	if ExplainPattern("(a", true)[len(ExplainPattern("(a", true))-1].Kind != "error" {
		t.Error("unbalanced paren not reported")
	}
	tp := ExplainTemplate(`$u$1 \$ $x`, true)
	if tp[len(tp)-1].Kind != "error" {
		t.Errorf("template %+v", tp)
	}
}

func TestPresetsCompile(t *testing.T) {
	for _, p := range Presets() {
		o := FindOptions{Find: p.Pattern, Replace: p.Replace, Regex: true, MatchCase: true, RegexOptions: "m"}
		if _, err := o.Compile(); err != nil {
			t.Errorf("%s: %v", p.Name, err)
		}
	}
	o := FindOptions{Find: Presets()[7].Pattern, Replace: "$1 $2", Regex: true, MatchCase: true}
	if got, _, _ := o.Apply("中文English"); got != "中 文English" && got != "中文 English" {
		t.Errorf("pangu got %q", got)
	}
}
