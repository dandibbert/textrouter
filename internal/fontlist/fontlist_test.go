package fontlist

import (
	"os"
	"strings"
	"testing"
)

func TestParseInter(t *testing.T) {
	data, err := os.ReadFile("../../gui/fonts/Inter.ttf")
	if err != nil {
		t.Skip(err)
	}
	fs, err := Parse(data)
	if err != nil || len(fs) != 1 || !strings.HasPrefix(fs[0].Name, "Inter") || fs[0].Mono {
		t.Fatalf("%+v %v", fs, err)
	}
	// The folder also holds the design's Space Grotesk and Space Mono.
	got := Scan("../../gui/fonts")
	names := map[string]bool{}
	for _, f := range got {
		names[strings.Fields(f.Name)[0]] = true
	}
	if len(got) != 3 || !names["Inter"] || !names["Space"] {
		t.Fatalf("scan: %+v", got)
	}
}
