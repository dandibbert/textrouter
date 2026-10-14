package engine

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"howett.net/plist"
)

// This file reads and writes TextSoap 9 cleaner records (the library format
// in docs/textsoap-handoff/actions/chain_format.md). Actions are TextSoap's
// own nodes, so they round-trip unchanged; unknown actions are kept and,
// as in TextSoap, do nothing when run.

func asString(v any) string {
	switch t := v.(type) {
	case string:
		return t
	case nil:
		return ""
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprint(int64(t))
		}
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

var structural = map[string]bool{"cleaner.actions": true, "endlist": true}

// NodeToAction converts one serialized TextSoap action node.
func NodeToAction(node map[string]any) Action {
	a := Action{ID: asString(node["identifier"]), Enabled: asBool(node["enabled"], true), Params: Params{}}
	if p, ok := node["parameters"].(map[string]any); ok {
		for k, v := range p {
			a.Params[k] = normalizePlist(v)
		}
	}
	a.Comment = asString(node["comment"])
	for _, key := range []string{"actionList", "children", "actions"} {
		if list, ok := node[key].([]any); ok {
			a.Children = NodesToActions(list)
			break
		}
	}
	return a
}

func normalizePlist(v any) any {
	switch t := v.(type) {
	case uint64:
		return int(t)
	case int64:
		return int(t)
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = normalizePlist(t[i])
		}
		return out
	case map[string]any:
		out := map[string]any{}
		for k, x := range t {
			out[k] = normalizePlist(x)
		}
		return out
	}
	return v
}

// NodesToActions converts an actionList, dropping the cleaner.actions /
// endlist bracket nodes. Old-style flat conditionals (actions up to an
// if.end) are nested into their conditional.
func NodesToActions(list []any) []Action {
	var flat []Action
	for _, n := range list {
		m, ok := n.(map[string]any)
		if !ok {
			continue
		}
		if structural[asString(m["identifier"])] {
			continue
		}
		flat = append(flat, NodeToAction(m))
	}
	return nestFlatConditionals(flat)
}

func nestFlatConditionals(flat []Action) []Action {
	var out []Action
	for i := 0; i < len(flat); i++ {
		a := flat[i]
		info, _ := ActionByID(a.ID)
		if strings.HasPrefix(a.ID, "if.match") && info.Container && len(a.Children) == 0 {
			depth := 0
			for j := i + 1; j < len(flat); j++ {
				if strings.HasPrefix(flat[j].ID, "if.match") && len(flat[j].Children) == 0 {
					depth++
				}
				if flat[j].ID == "if.end" {
					if depth == 0 {
						a.Children = nestFlatConditionals(flat[i+1 : j])
						i = j
						break
					}
					depth--
				}
			}
		}
		if a.ID == "if.end" {
			continue
		}
		out = append(out, a)
	}
	return out
}

// ImportResult is what a TextSoap file contained.
type ImportResult struct {
	Cleaners []CustomCleaner `json:"cleaners"`
	Skipped  int             `json:"skipped"`
}

// ImportTextSoap reads a TextSoap library or cleaner file (binary or XML
// plist), or TextRouter's own JSON export.
func ImportTextSoap(data []byte) (ImportResult, error) {
	if bytes.HasPrefix(bytes.TrimSpace(data), []byte("{")) {
		return importJSON(data)
	}
	var root any
	if _, err := plist.Unmarshal(data, &root); err != nil {
		return ImportResult{}, fmt.Errorf("不是 TextSoap 的清理器文件：%w", err)
	}
	var records []map[string]any
	var walk func(v any)
	walk = func(v any) {
		switch t := v.(type) {
		case map[string]any:
			if _, ok := t["actionList"]; ok && t["name"] != nil {
				records = append(records, t)
				return
			}
			for _, k := range sortedKeys(t) {
				walk(t[k])
			}
		case []any:
			for _, x := range t {
				walk(x)
			}
		}
	}
	walk(root)
	if len(records) == 0 {
		return ImportResult{}, fmt.Errorf("文件里没有找到清理器")
	}
	var res ImportResult
	for _, rec := range records {
		list, _ := rec["actionList"].([]any)
		props, _ := rec["properties"].(map[string]any)
		c := CustomCleaner{
			UUID:    strings.ToUpper(asString(rec["uuid"])),
			Name:    asString(rec["name"]),
			Actions: NodesToActions(list),
		}
		if props != nil {
			c.Author, c.Comment = asString(props["author"]), asString(props["comment"])
		}
		if t, ok := rec["lastModified"].(time.Time); ok {
			c.Modified = t
		}
		if c.UUID == "" {
			c.UUID = NewUUID()
		}
		res.Cleaners = append(res.Cleaners, c)
	}
	sort.SliceStable(res.Cleaners, func(i, j int) bool { return res.Cleaners[i].Name < res.Cleaners[j].Name })
	return res, nil
}

func importJSON(data []byte) (ImportResult, error) {
	var wrap struct {
		Cleaners []CustomCleaner `json:"cleaners"`
	}
	if err := json.Unmarshal(data, &wrap); err == nil && len(wrap.Cleaners) > 0 {
		return ImportResult{Cleaners: wrap.Cleaners}, nil
	}
	var one CustomCleaner
	if err := json.Unmarshal(data, &one); err != nil || one.Name == "" {
		return ImportResult{}, fmt.Errorf("无法识别的文件")
	}
	return ImportResult{Cleaners: []CustomCleaner{one}}, nil
}

func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// ActionToNode writes an action in TextSoap's node shape.
func ActionToNode(a Action) map[string]any {
	params := map[string]any{}
	for k, v := range a.Params {
		params[k] = plistSafe(v)
	}
	if strings.HasPrefix(a.ID, "find") || strings.HasPrefix(a.ID, "if.match.text") {
		// A find node without type is silently literal in hand-built files.
		if _, ok := params["type"]; !ok {
			params["type"] = 1
		}
	}
	n := map[string]any{"identifier": a.ID, "parameters": params, "enabled": a.Enabled,
		"show_children": true, "show_details": true, "height": 0.0}
	if a.Comment != "" {
		n["comment"] = a.Comment
	}
	if len(a.Children) > 0 {
		kids := make([]any, 0, len(a.Children))
		for _, c := range a.Children {
			kids = append(kids, ActionToNode(c))
		}
		n["actionList"] = kids
	}
	return n
}

func plistSafe(v any) any {
	switch t := v.(type) {
	case nil:
		return ""
	case float64:
		if t == float64(int64(t)) {
			return int64(t)
		}
	case []any:
		out := make([]any, len(t))
		for i := range t {
			out[i] = plistSafe(t[i])
		}
		return out
	case []string:
		out := make([]any, len(t))
		for i := range t {
			out[i] = t[i]
		}
		return out
	case [][]string:
		out := make([]any, len(t))
		for i := range t {
			out[i] = plistSafe(t[i])
		}
		return out
	}
	return v
}

func bracket(id string) map[string]any {
	return map[string]any{"identifier": id, "parameters": map[string]any{}, "enabled": true,
		"show_children": true, "show_details": true, "height": 0.0}
}

// CleanerRecord is a TextSoap cleaner record (fileVersion 6: TextSoap
// silently ignores records with 2).
func CleanerRecord(c CustomCleaner) map[string]any {
	list := []any{bracket("cleaner.actions")}
	for _, a := range c.Actions {
		list = append(list, ActionToNode(a))
	}
	list = append(list, bracket("endlist"))
	mod := c.Modified
	if mod.IsZero() {
		mod = time.Now()
	}
	return map[string]any{
		"name": c.Name, "uuid": strings.ToUpper(c.UUID), "datatype": "cleaner", "actionList": list,
		"properties":   map[string]any{"author": c.Author, "comment": c.Comment, "version": "1"},
		"lastModified": mod, "min-required": "9.0", "version": 1, "fileVersion": 6, "revision": 1,
	}
}

// ExportTextSoap writes cleaners as a TextSoap library (XML plist).
func ExportTextSoap(cleaners []CustomCleaner) ([]byte, error) {
	recs := map[string]any{}
	for _, c := range cleaners {
		if c.UUID == "" {
			c.UUID = NewUUID()
		}
		recs[strings.ToUpper(c.UUID)] = CleanerRecord(c)
	}
	root := map[string]any{"version": 1, "contents": map[string]any{"cleaners": recs, "groups": map[string]any{}}}
	var b bytes.Buffer
	enc := plist.NewEncoderForFormat(&b, plist.XMLFormat)
	enc.Indent("\t")
	if err := enc.Encode(root); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// NewUUID returns an uppercase random UUID, as TextSoap writes them.
func NewUUID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	b[6] = b[6]&0x0f | 0x40
	b[8] = b[8]&0x3f | 0x80
	return strings.ToUpper(fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]))
}
