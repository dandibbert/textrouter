package main

import (
	"path/filepath"
	"testing"
)

func TestBatchOutput(t *testing.T) {
	src := filepath.Join("/data", "notes", "a.txt")
	cases := []struct {
		s    engineBatch
		want string
	}{
		{engineBatch{DestFolder: "results"}, "/data/notes/results/a.txt"},
		{engineBatch{DestFolder: "/out"}, "/out/a.txt"},
		{engineBatch{Prefix: "clean-", Suffix: "_v2"}, "/data/notes/clean-a_v2.txt"},
		{engineBatch{NewExt: ".md"}, "/data/notes/a.md"},
		{engineBatch{DestFolder: "out", NewExt: "csv"}, "/data/notes/out/a.csv"},
	}
	for _, c := range cases {
		if got := c.s.output(src); got != filepath.FromSlash(c.want) {
			t.Errorf("%+v: got %s, want %s", c.s, got, c.want)
		}
	}
}

func TestBatchWants(t *testing.T) {
	def := engineBatch{Extensions: "rtf, .adoc"}
	only := engineBatch{ExtMode: "only", Extensions: "rtf"}
	for path, want := range map[string]bool{"a.txt": true, "a.MD": true, "a.rtf": true, "a.adoc": true, "a.png": false} {
		if got := def.wants(path); got != want {
			t.Errorf("default %s: got %v", path, got)
		}
	}
	if only.wants("a.txt") || !only.wants("b.RTF") {
		t.Error("only mode should pick just the listed extensions")
	}
}

func TestDecodeText(t *testing.T) {
	cases := map[string][]byte{
		"BOM":      {0xEF, 0xBB, 0xBF, 'h', 'i'},
		"UTF-16LE": {0xFF, 0xFE, 'h', 0, 'i', 0},
		"UTF-16BE": {0xFE, 0xFF, 0, 'h', 0, 'i'},
		"UTF-8":    []byte("hi"),
	}
	for name, b := range cases {
		if got := decodeText(b); got != "hi" {
			t.Errorf("%s: got %q", name, got)
		}
	}
	if got := decodeText([]byte{0xC4, 0xE3, 0xBA, 0xC3}); got != "你好" {
		t.Errorf("GB18030: got %q", got)
	}
	if got := decodeText([]byte{0xFF, 0xFE, 0x3D, 0xD8, 0x00, 0xDE}); got != "😀" {
		t.Errorf("surrogate pair: got %q", got)
	}
}
