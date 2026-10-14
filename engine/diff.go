package engine

// Seg is a piece of a diff: Op is '=', '-' (only in old) or '+' (only in new).
type Seg struct {
	Op   byte
	Text string
}

// Diff computes a character-level diff (LCS on runes, with common prefix
// and suffix trimmed). Large inputs fall back to one replace segment.
func Diff(a, b string) []Seg {
	ra, rb := []rune(a), []rune(b)
	p := 0
	for p < len(ra) && p < len(rb) && ra[p] == rb[p] {
		p++
	}
	s := 0
	for s < len(ra)-p && s < len(rb)-p && ra[len(ra)-1-s] == rb[len(rb)-1-s] {
		s++
	}
	var out []Seg
	add := func(op byte, r []rune) {
		if len(r) == 0 {
			return
		}
		if n := len(out); n > 0 && out[n-1].Op == op {
			out[n-1].Text += string(r)
			return
		}
		out = append(out, Seg{op, string(r)})
	}
	add('=', ra[:p])
	ma, mb := ra[p:len(ra)-s], rb[p:len(rb)-s]
	if len(ma)*len(mb) > 1_500_000 {
		add('-', ma)
		add('+', mb)
	} else {
		n, m := len(ma), len(mb)
		dp := make([][]int32, n+1)
		for i := range dp {
			dp[i] = make([]int32, m+1)
		}
		for i := n - 1; i >= 0; i-- {
			for j := m - 1; j >= 0; j-- {
				if ma[i] == mb[j] {
					dp[i][j] = dp[i+1][j+1] + 1
				} else {
					dp[i][j] = max(dp[i+1][j], dp[i][j+1])
				}
			}
		}
		i, j := 0, 0
		for i < n && j < m {
			switch {
			case ma[i] == mb[j]:
				add('=', ma[i:i+1])
				i, j = i+1, j+1
			case dp[i+1][j] >= dp[i][j+1]:
				add('-', ma[i:i+1])
				i++
			default:
				add('+', mb[j:j+1])
				j++
			}
		}
		add('-', ma[i:])
		add('+', mb[j:])
	}
	add('=', ra[len(ra)-s:])
	return out
}

// DiffStats counts inserted and deleted runes.
func DiffStats(segs []Seg) (ins, del int) {
	for _, s := range segs {
		switch s.Op {
		case '+':
			ins += RuneLen(s.Text)
		case '-':
			del += RuneLen(s.Text)
		}
	}
	return
}
