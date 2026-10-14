package gui

import (
	"hash/maphash"

	"github.com/dandibbert/textrouter/engine"
)

// The view is rebuilt every frame; diffs and chain runs are memoised so
// that previews of large texts stay smooth.

var seed = maphash.MakeSeed()

type memoEntry struct {
	key uint64
	val any
}

type memo struct {
	entries []memoEntry
}

func (m *memo) get(key uint64, compute func() any) any {
	for _, e := range m.entries {
		if e.key == key {
			return e.val
		}
	}
	v := compute()
	m.entries = append(m.entries, memoEntry{key, v})
	if len(m.entries) > 32 {
		m.entries = m.entries[1:]
	}
	return v
}

func hashOf(parts ...string) uint64 {
	var h maphash.Hash
	h.SetSeed(seed)
	for _, p := range parts {
		h.WriteString(p)
		h.WriteByte(0)
	}
	return h.Sum64()
}

var diffMemo memo

func cachedDiff(a, b string) []engine.Seg {
	return diffMemo.get(hashOf("diff", a, b), func() any { return engine.Diff(a, b) }).([]engine.Seg)
}
