package engine

import (
	"encoding/json"
	"os"
	"sort"
	"testing"
)

type truthCase struct {
	ID      string `json:"id"`
	Cleaner string `json:"cleaner"`
	Input   string `json:"input"`
	Output  string `json:"output"`
}

// knownDivergent lists measured cases we knowingly do not match yet.
// TestCleanerTruth fails if the pass rate drops below the recorded floor.
const truthFloor = 0.999

func TestCleanerTruth(t *testing.T) {
	data, err := os.ReadFile("testdata/behaviour_truth.json")
	if err != nil {
		t.Fatal(err)
	}
	var cases []truthCase
	if err := json.Unmarshal(data, &cases); err != nil {
		t.Fatal(err)
	}
	type stat struct{ pass, total int }
	per := map[string]*stat{}
	pass := 0
	for _, c := range cases {
		cl := CleanerByName(c.Cleaner)
		if cl == nil {
			t.Errorf("missing cleaner %q", c.Cleaner)
			continue
		}
		s := per[c.Cleaner]
		if s == nil {
			s = &stat{}
			per[c.Cleaner] = s
		}
		s.total++
		if got := cl.Fn(c.Input); got == c.Output {
			s.pass++
			pass++
		} else if testing.Verbose() {
			t.Logf("DIFF %s: in %q got %q want %q", c.ID, c.Input, got, c.Output)
		}
	}
	var names []string
	for n := range per {
		names = append(names, n)
	}
	sort.Strings(names)
	for _, n := range names {
		if per[n].pass != per[n].total {
			t.Logf("%-36s %d/%d", n, per[n].pass, per[n].total)
		}
	}
	rate := float64(pass) / float64(len(cases))
	t.Logf("cleaner truth: %d/%d (%.1f%%)", pass, len(cases), rate*100)
	if rate < truthFloor {
		t.Errorf("pass rate %.3f below floor %.2f", rate, truthFloor)
	}
}

func TestEveryCleanerRuns(t *testing.T) {
	total, impl, measured := CleanerStats()
	if total != 186 && total < 180 {
		t.Errorf("only %d cleaners", total)
	}
	for _, c := range Cleaners() {
		_ = c.Fn("Hello  world.\n\nsecond “line” 2026-10-14 https://x.y")
		_ = c.Fn("")
	}
	t.Logf("cleaners: %d total, %d implemented, %d measured", total, impl, measured)
}
