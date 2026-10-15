package engine

import (
	"encoding/json"
	"os"
	"regexp"
	"testing"
)

// TestChainExamples runs, unchanged, the 15 action lists executed on the
// real TextSoap engine.
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
		got, _ := RunActions(NodesToActions(c.Actions), c.Input, nil)
		if got != c.Output {
			t.Errorf("%s: got %q want %q", c.ID, got, c.Output)
			continue
		}
		pass++
	}
	t.Logf("chain examples: %d/%d", pass, len(cases))
}

func a(id string, p Params, kids ...Action) Action { return act(id, p, kids...) }

func TestListsAndProcessList(t *testing.T) {
	list := a("runtime.definelist", Params{"name": "L", "columns": 2, "rows": []any{[]any{"cat", "dog"}, []any{"red", "blue"}}})
	cases := []struct {
		name    string
		actions []Action
		in, out string
	}{
		{"find with list", []Action{
			a("find", Params{"find": "$v{A}", "replace": "$v{B}", "type": 0, "list_name": "L"}),
			list, // lists may be defined after their use
		}, "a red cat", "a blue dog"},
		{"process list", []Action{list,
			a("runtime.processlist", Params{"list_name": "L"},
				a("find", Params{"find": `\b$v{A}\b`, "replace": "[$v{B}]", "type": 1})),
		}, "cat cats red", "[dog] cats [blue]"},
		{"escaped var", []Action{list,
			a("find", Params{"find": "x", "replace": `\$v{A}`, "type": 0, "list_name": "L"}),
		}, "x", "$v{A}"},
		{"batch id", []Action{list,
			a("find.batch", Params{"find": "$v{A}", "replace": "$v{B}", "list_name": "L"}),
		}, "red.cat", "blue.dog"},
		{"tsv list text", []Action{
			a("runtime.definelist", Params{"name": "T", "columns": 2, "text": "a\tb\nc\td"}),
			a("find", Params{"find": "$v{A}", "replace": "$v{B}", "type": 0, "list_name": "T"}),
		}, "ac", "bd"},
	}
	for _, c := range cases {
		got, _ := RunActions(c.actions, c.in, nil)
		if got != c.out {
			t.Errorf("%s: got %q want %q", c.name, got, c.out)
		}
	}
}

func TestConditionalsMacrosLines(t *testing.T) {
	cases := []struct {
		name    string
		actions []Action
		in, out string
	}{
		{"if regex + group", []Action{a("if.match.text.regex", Params{"find": `<b>(.*?)</b>`, "group": 1},
			a("applycleaner", Params{"cleaner": "Uppercase"}))}, "<b>hi</b> there", "<b>HI</b> there"},
		{"if text invert", []Action{a("if.match.text", Params{"find": "x", "type": 0, "invert": true},
			a("applycleaner", Params{"cleaner": "Uppercase"}))}, "axbxc", "AxBxC"},
		{"skip/limit", []Action{a("if.match.text.regex", Params{"find": `\d`, "skip": 1, "limit": 2},
			a("text.tag", Params{"prefix": "[", "suffix": "]"}))}, "1234", "1[2][3]4"},
		{"modify nodes", []Action{a("if.match.text.regex", Params{"find": `\d`},
			a("modify.results.limit", Params{"count": 1}), a("text.delete", nil))}, "1234", "234"},
		{"for each line skips blanks", []Action{a("if.match.text.processlines", nil,
			a("text.insert", Params{"text": "- ", "position": 0}))}, "a\n\nb", "- a\n\n- b"},
		{"macro", []Action{
			a("runtime.applymacro", Params{"name": "M"}),
			a("runtime.macro", Params{"name": "M"}, a("text.tag", Params{"prefix": "(", "suffix": ")"})),
		}, "x", "(x)"},
		{"repeat", []Action{a("runtime.repeat", Params{"count": 3}, a("text.insert", Params{"text": "!", "position": 1}))}, "a", "a!!!"},
		{"sort numeric desc", []Action{a("line.sort", Params{"numeric": true, "descending": true})}, "2\n10\n1", "10\n2\n1\n"},
		{"sort ignore case", []Action{a("line.sort", nil)}, "b\nA\nc", "A\nb\nc\n"},
		{"number lines", []Action{a("line.number", Params{"format": `000\t`, "lines": 1})}, "a\n\nb\n", "001\ta\n\n002\tb\n"},
		{"add/remove line text", []Action{a("line.addto", Params{"text": ";", "position": 1}), a("line.removefrom", Params{"text": "#"})}, "#a\nb", "a;\nb;"},
		{"extract group", []Action{a("text.extract", Params{"find": `(\w+)@`, "group": 1})}, "a@x b@y", "a\nb"},
		{"substring", []Action{a("text.substring", Params{"mode": 2, "start": 2, "length": 3})}, "abcdef", "bcd"},
		{"quote level 2", []Action{a("text.quote", Params{"level": 2, "wrap": false})}, "a\nb", ">> a\n>> b"},
		{"url", []Action{a("if.match.url", nil, a("text.tag", Params{"prefix": "<", "suffix": ">"}))}, "see https://x.org/a, ok", "see <https://x.org/a>, ok"},
		{"disabled group", []Action{{ID: "runtime.group", Enabled: false, Children: []Action{a("text.delete", nil)}}}, "keep", "keep"},
		{"style is a no-op", []Action{a("style.text.font", nil)}, "keep", "keep"},
	}
	for _, c := range cases {
		got, _ := RunActions(c.actions, c.in, nil)
		if got != c.out {
			t.Errorf("%s: got %q want %q", c.name, got, c.out)
		}
	}
}

func TestNestedCustomCleaner(t *testing.T) {
	lib := &Library{Prefs: DefaultPrefs()}
	lib.PutCleaner(CustomCleaner{Name: "Inner", Actions: []Action{a("text.insert", Params{"text": ">", "position": 0})}})
	outer := lib.PutCleaner(CustomCleaner{Name: "Outer", Actions: []Action{a("applycleaner", Params{"cleaner": "Inner"}), a("applycleaner", Params{"cleaner": "Uppercase"})}})
	got, _ := outer.Run("a", lib.Env(nil))
	if got != ">A" {
		t.Errorf("got %q", got)
	}
	loop := lib.PutCleaner(CustomCleaner{Name: "Loop", Actions: []Action{a("applycleaner", Params{"cleaner": "Loop"})}})
	if got, _ := loop.Run("a", lib.Env(nil)); got != "a" {
		t.Errorf("self-recursion changed text: %q", got)
	}
	if lib.PutCleaner(CustomCleaner{Name: "Uppercase"}).Name == "Uppercase" {
		t.Error("custom cleaner may not shadow a built-in name")
	}
}

func TestTextSoapRoundTrip(t *testing.T) {
	in := []CustomCleaner{{Name: "Demo", Actions: []Action{
		a("find", Params{"find": "a.", "replace": "b", "type": 1, "match_case": true}),
		a("runtime.group", nil, a("applycleaner", Params{"cleaner": "Uppercase"})),
		a("runtime.definelist", Params{"name": "L", "columns": 2, "rows": []any{[]any{"x", "y"}}}),
	}}}
	data, err := ExportTextSoap(in)
	if err != nil {
		t.Fatal(err)
	}
	if !regexp.MustCompile(`<key>fileVersion</key>\s*<integer>6</integer>`).Match(data) {
		t.Errorf("fileVersion 6 missing:\n%s", data)
	}
	res, err := ImportTextSoap(data)
	if err != nil || len(res.Cleaners) != 1 {
		t.Fatal(res, err)
	}
	got, _ := res.Cleaners[0].Run("xa1", nil)
	if got != "XB" {
		t.Errorf("got %q", got)
	}
	if l := ListFromParams(res.Cleaners[0].Actions[2].Params); len(l.Rows) != 1 || l.Rows[0][1] != "y" {
		t.Errorf("list did not round trip: %+v", l)
	}
}

func TestFlatConditionalsNest(t *testing.T) {
	nodes := []any{
		map[string]any{"identifier": "cleaner.actions"},
		map[string]any{"identifier": "if.match.text", "enabled": true, "parameters": map[string]any{"find": "b", "type": 0}},
		map[string]any{"identifier": "applycleaner", "enabled": true, "parameters": map[string]any{"cleaner": "Uppercase"}},
		map[string]any{"identifier": "if.end", "enabled": true},
		map[string]any{"identifier": "endlist"},
	}
	acts := NodesToActions(nodes)
	if len(acts) != 1 || len(acts[0].Children) != 1 {
		t.Fatalf("got %+v", acts)
	}
	if got, _ := RunActions(acts, "abc", nil); got != "aBc" {
		t.Errorf("got %q", got)
	}
}

func TestBuiltinGroups(t *testing.T) {
	gs := BuiltinGroups()
	byID := map[string]Group{}
	for _, g := range gs {
		byID[g.ID] = g
	}
	if len(byID["html"].Items) < 20 || len(byID["library"].Items) < 180 || len(byID["basics"].Items) < 10 {
		t.Errorf("groups too small: html %d library %d basics %d", len(byID["html"].Items), len(byID["library"].Items), len(byID["basics"].Items))
	}
	for _, g := range gs {
		for _, it := range g.Items {
			if CleanerByName(it.Name) == nil {
				t.Errorf("group %s lists unknown cleaner %q", g.ID, it.Name)
			}
		}
	}
}
