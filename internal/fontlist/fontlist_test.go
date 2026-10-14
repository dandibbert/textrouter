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
	got := Scan("../../gui/fonts")
	if len(got) != 1 || !strings.HasPrefix(got[0].Name, "Inter") {
		t.Fatalf("scan: %+v", got)
	}
}
