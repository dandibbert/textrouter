package engine

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"sort"
	"strings"
	"time"

	"howett.net/plist"
)

// This file reads and writes TextSoap 9 cleaner records (the library format
// described in the handoff pack's chain_format.md). Only the parts whose
// schema is verified are mapped faithfully: `find` nodes and containers.
// Other node parameters are best guesses; unknown nodes import as no-op
// steps, exactly as TextSoap treats unknown actions.

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	}
	return fmt.Sprint(v)
}

func asInt(v any, def int) int {
	switch t := v.(type) {
	case bool:
		if t {
			return 1
		}
		return 0
	case int:
		return t
	case int64:
		return int(t)
	case uint64:
		return int(t)
	case float64:
		return int(t)
	case string:
		var n int
		if _, err := fmt.Sscan(t, &n); err == nil {
			return n
		}
	}
	return def
}

func asBool(v any, def bool) bool {
	if v == nil {
		return def
	}
	return asInt(v, 0) != 0
}

func firstOf(p map[string]any, keys ...string) any {
	for _, k := range keys {
		if v, ok := p[k]; ok {
			return v
		}
	}
	return nil
}

// NodeToStep converts one serialized TextSoap action node.
func NodeToStep(node map[string]any) Step {
	id := asString(node["identifier"])
	p, _ := node["parameters"].(map[string]any)
	if p == nil {
		p = map[string]any{}
	}
	s := Step{Kind: id, Disabled: !asBool(node["enabled"], true)}
	findOpts := func() FindOptions {
		// Measured: a missing `type` behaves like regex (type=1).
		return FindOptions{
			Find:         asString(p["find"]),
			Replace:      asString(p["replace"]),
			Regex:        asInt(p["type"], 1) != 0,
			MatchCase:    asBool(p["match_case"], true),
			RegexOptions: asString(p["regex_options"]),
		}
	}
	switch id {
	case "find", "find.regex", "find.delete":
		s.Find = findOpts()
		if id == "find.regex" {
			s.Find.Regex = true
			s.Kind = KindFind
		}
	case "find.batch", "find.batch.regex":
		s.Kind = KindBatch
		s.Text = asString(p["list_name"])
	case "applycleaner", "applycleaner.static":
		s.Kind = KindCleaner
		s.Cleaner = asString(firstOf(p, "cleaner", "cleaner_name", "name", "value"))
	case "text.insert":
		s.Text = asString(firstOf(p, "text", "value", "string"))
		s.Flag = strings.Contains(strings.ToLower(asString(p["position"])), "end")
	case "text.tag":
		s.Text = asString(firstOf(p, "prefix", "before", "start"))
		s.Text2 = asString(firstOf(p, "suffix", "after", "end"))
	case "line.addto", "line.removefrom":
		s.Text = asString(firstOf(p, "text", "value", "string"))
	case "line.sort":
		s.Flag = asBool(firstOf(p, "descending", "order"), false)
	case "text.wrap", "text.quote":
		s.Number = asInt(firstOf(p, "width", "length", "max_length"), 0)
	case "text.extract":
		s.Find = findOpts()
		s.Find.Regex = true
	case "runtime.group", "runtime.repeat", "if.match.text.processlines", "if.match.text.regex":
		if id == "if.match.text.regex" {
			s.Find = findOpts()
			s.Find.Regex = true
		}
		s.Number = asInt(firstOf(p, "count", "times"), 1)
		for _, key := range []string{"actionList", "children", "actions"} {
			if list, ok := firstOf(node, key).([]any); ok {
				s.Children = NodesToSteps(list)
				break
			}
			if list, ok := p[key].([]any); ok {
				s.Children = NodesToSteps(list)
				break
			}
		}
	case "runtime.usernote":
		s.Kind = KindNote
		s.Text = asString(firstOf(p, "note", "text"))
	}
	return s
}

// NodesToSteps converts an actionList, dropping the structural
// cleaner.actions / endlist bracket nodes.
func NodesToSteps(list []any) []Step {
	var out []Step
	for _, n := range list {
		m, ok := n.(map[string]any)
		if !ok {
			continue
		}
		switch asString(m["identifier"]) {
		case "cleaner.actions", "endlist", "if.end":
			continue
		}
		out = append(out, NodeToStep(m))
	}
	return out
}

// ImportTextSoapLibrary reads a TextSoap library plist (binary or XML) or a
// single cleaner record and returns its cleaners as chains.
func ImportTextSoapLibrary(data []byte) ([]Chain, error) {
	var root any
	if _, err := plist.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("不是 TextSoap 的 plist 文件：%w", err)
	}
	var records []map[string]any
	collect := func(m map[string]any) {
		if _, ok := m["actionList"]; ok {
			records = append(records, m)
		}
	}
	switch r := root.(type) {
	case map[string]any:
		collect(r)
		if contents, ok := r["contents"].(map[string]any); ok {
			if cl, ok := contents["cleaners"].(map[string]any); ok {
				for _, v := range cl {
					if m, ok := v.(map[string]any); ok {
						collect(m)
					}
				}
			}
		}
	case []any:
		for _, v := range r {
			if m, ok := v.(map[string]any); ok {
				collect(m)
			}
		}
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("文件里没有找到清理器记录")
	}
	var out []Chain
	for _, rec := range records {
		list, _ := rec["actionList"].([]any)
		out = append(out, Chain{ID: NewID(), Name: asString(rec["name"]), Steps: NodesToSteps(list)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// StepToNode writes a step back in TextSoap's node shape.
func StepToNode(s Step) map[string]any {
	params := map[string]any{}
	id := s.Kind
	switch s.Kind {
	case KindFind, KindDelete:
		t := 0
		if s.Find.Regex {
			t = 1 // always explicit: a missing type is a trap
		}
		params = map[string]any{"find": s.Find.Find, "replace": s.Find.Replace, "type": t, "granularity": 0,
			"match_case": s.Find.MatchCase, "regex_options": s.Find.RegexOptions, "list_name": ""}
		if s.Kind == KindDelete {
			params["replace"] = ""
		}
	case KindCleaner:
		params["cleaner"] = s.Cleaner
	case KindInsert, KindLineAdd, KindLineRemove, KindNote:
		params["text"] = s.Text
	case KindTag:
		params["prefix"], params["suffix"] = s.Text, s.Text2
	case KindWrap, KindQuote:
		params["width"] = s.Number
	}
	n := map[string]any{"identifier": id, "parameters": params, "enabled": !s.Disabled,
		"show_children": true, "show_details": true, "height": 0.0}
	if len(s.Children) > 0 {
		var kids []any
		for _, c := range s.Children {
			kids = append(kids, StepToNode(c))
		}
		n["actionList"] = kids
	}
	return n
}

// ExportTextSoapLibrary writes chains as a TextSoap library XML plist with
// fileVersion 6 records (2 would be silently ignored by TextSoap).
func ExportTextSoapLibrary(chains []Chain) ([]byte, error) {
	cleaners := map[string]any{}
	for _, c := range chains {
		uuid := strings.ToUpper(uuid4())
		list := []any{map[string]any{"identifier": "cleaner.actions", "parameters": map[string]any{}, "enabled": true,
			"show_children": true, "show_details": true, "height": 0.0}}
		for _, s := range c.Steps {
			list = append(list, StepToNode(s))
		}
		list = append(list, map[string]any{"identifier": "endlist", "parameters": map[string]any{}, "enabled": true,
			"show_children": true, "show_details": true, "height": 0.0})
		cleaners[uuid] = map[string]any{
			"name": c.Name, "uuid": uuid, "datatype": "cleaner", "actionList": list,
			"properties":   map[string]any{"author": "TextRouter", "comment": c.Note, "version": "1"},
			"lastModified": time.Now(), "min-required": "9.0", "version": 1, "fileVersion": 6, "revision": 1,
		}
	}
	root := map[string]any{"version": 1, "contents": map[string]any{"cleaners": cleaners, "groups": map[string]any{}}}
	var b bytes.Buffer
	enc := plist.NewEncoderForFormat(&b, plist.XMLFormat)
	enc.Indent("\t")
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func uuid4() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

// NewID returns a short random id for chains and rule sets.
func NewID() string { return uuid4()[:8] }
