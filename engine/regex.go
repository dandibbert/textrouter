package engine

import (
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/dlclark/regexp2"
)

// FindOptions describes one find/replace operation. The semantics follow the
// measured TextSoap 9 behaviour (see README "实测语义"):
//
//   - Regex false means literal text (TextSoap type=0).
//   - MatchCase false is the ONLY way to ignore case. The letters i/I in
//     RegexOptions are inert, as in TextSoap; an inline (?i) works.
//   - Only lowercase m (multiline ^/$) and s (dot matches newline) in
//     RegexOptions have an effect. Everything else is silently ignored.
//   - Replacement tokens: $0-$9 (single digit), $u $U $l $L case changes,
//     $E/$e end case change (substitute empty), "\x" literal x, any other
//     "$x" pair is dropped, a bare trailing "$" is dropped. No ${1}.
type FindOptions struct {
	Find         string `json:"find"`
	Replace      string `json:"replace"`
	Regex        bool   `json:"regex"`
	MatchCase    bool   `json:"matchCase"`
	RegexOptions string `json:"regexOptions,omitempty"`
}

// Multiline reports whether the m option letter is set.
func (o FindOptions) Multiline() bool { return strings.ContainsRune(o.RegexOptions, 'm') }

// DotAll reports whether the s option letter is set.
func (o FindOptions) DotAll() bool { return strings.ContainsRune(o.RegexOptions, 's') }

// Match is one match with its groups, in rune offsets of the input.
type Match struct {
	Start, End int
	Groups     []Group // index 0 is the whole match
	Result     string  // what this match is replaced by
}

// Group is one capture group of a match. Ok is false when it did not take part.
type Group struct {
	Start, End int
	Text       string
	Ok         bool
}

// Compile builds the matcher. Literal patterns are escaped.
func (o FindOptions) Compile() (*regexp2.Regexp, error) {
	pat := o.Find
	if !o.Regex {
		pat = regexp2.Escape(pat)
	}
	var opt regexp2.RegexOptions
	if !o.MatchCase {
		opt |= regexp2.IgnoreCase
	}
	if o.Regex && o.Multiline() {
		opt |= regexp2.Multiline
	}
	if o.Regex && o.DotAll() {
		opt |= regexp2.Singleline
	}
	re, err := regexp2.Compile(pat, opt)
	if err != nil {
		return nil, err
	}
	re.MatchTimeout = 2e9
	return re, nil
}

// FindAll returns every match (non-overlapping, left to right) with its
// computed replacement. limit <= 0 means no limit.
func (o FindOptions) FindAll(text string, limit int) ([]Match, error) {
	if o.Find == "" {
		return nil, nil
	}
	re, err := o.Compile()
	if err != nil {
		return nil, err
	}
	tmpl := ParseTemplate(o.Replace)
	if !o.Regex {
		tmpl = Template{{Kind: TokLiteral, Text: o.Replace}}
	}
	var out []Match
	m, err := re.FindStringMatch(text)
	for m != nil && err == nil {
		mm := Match{Start: m.Index, End: m.Index + m.Length}
		for _, g := range m.Groups() {
			gr := Group{Ok: len(g.Captures) > 0}
			if gr.Ok {
				gr.Start, gr.End, gr.Text = g.Index, g.Index+g.Length, g.String()
			}
			mm.Groups = append(mm.Groups, gr)
		}
		mm.Result = tmpl.Expand(mm.Groups)
		out = append(out, mm)
		if limit > 0 && len(out) >= limit {
			break
		}
		m, err = re.FindNextMatch(m)
	}
	return out, err
}

// Apply applies the find/replace and returns the new text and the number
// of replacements.
func (o FindOptions) Apply(text string) (string, int, error) {
	ms, err := o.FindAll(text, 0)
	if err != nil || len(ms) == 0 {
		return text, 0, err
	}
	return ApplyMatches(text, ms), len(ms), nil
}

// ApplyMatches splices match results into text (rune offsets).
func ApplyMatches(text string, ms []Match) string {
	r := []rune(text)
	var b strings.Builder
	last := 0
	for _, m := range ms {
		if m.Start < last {
			continue
		}
		b.WriteString(string(r[last:m.Start]))
		b.WriteString(m.Result)
		last = m.End
	}
	b.WriteString(string(r[last:]))
	return b.String()
}

// TokenKind is the kind of a replacement template token.
type TokenKind int

const (
	TokLiteral   TokenKind = iota
	TokGroup               // $n
	TokUpperAll            // $U
	TokUpperNext           // $u
	TokLowerAll            // $L
	TokLowerNext           // $l
	TokEnd                 // $E / $e
	TokDropped             // an unsupported $x pair, substitutes nothing
)

// Token is one piece of a parsed replacement template.
type Token struct {
	Kind  TokenKind
	Text  string // literal text, or the raw source for other kinds
	Group int
}

// Template is a parsed replacement string.
type Template []Token

// ParseTemplate parses a replacement following the measured rules.
func ParseTemplate(s string) Template {
	var t Template
	var lit strings.Builder
	flush := func() {
		if lit.Len() > 0 {
			t = append(t, Token{Kind: TokLiteral, Text: lit.String()})
			lit.Reset()
		}
	}
	r := []rune(s)
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch {
		case c == '\\' && i+1 < len(r):
			i++
			lit.WriteRune(r[i])
		case c == '$':
			flush()
			if i+1 >= len(r) {
				t = append(t, Token{Kind: TokDropped, Text: "$"})
				continue
			}
			i++
			n := r[i]
			raw := "$" + string(n)
			switch {
			case n >= '0' && n <= '9':
				t = append(t, Token{Kind: TokGroup, Group: int(n - '0'), Text: raw})
			case n == 'U':
				t = append(t, Token{Kind: TokUpperAll, Text: raw})
			case n == 'u':
				t = append(t, Token{Kind: TokUpperNext, Text: raw})
			case n == 'L':
				t = append(t, Token{Kind: TokLowerAll, Text: raw})
			case n == 'l':
				t = append(t, Token{Kind: TokLowerNext, Text: raw})
			case n == 'E' || n == 'e':
				t = append(t, Token{Kind: TokEnd, Text: raw})
			default:
				t = append(t, Token{Kind: TokDropped, Text: raw})
			}
		default:
			lit.WriteRune(c)
		}
	}
	flush()
	return t
}

// Expand builds the substitution for one match.
func (t Template) Expand(groups []Group) string {
	var b strings.Builder
	mode := TokEnd // no case change
	next := TokEnd // pending one-character change
	write := func(s string) {
		for _, c := range s {
			switch mode {
			case TokUpperAll:
				c = unicode.ToUpper(c)
			case TokLowerAll:
				c = unicode.ToLower(c)
			}
			switch next {
			case TokUpperNext:
				c = unicode.ToUpper(c)
				next = TokEnd
			case TokLowerNext:
				c = unicode.ToLower(c)
				next = TokEnd
			}
			b.WriteRune(c)
		}
	}
	for _, tok := range t {
		switch tok.Kind {
		case TokLiteral:
			write(tok.Text)
		case TokGroup:
			if tok.Group < len(groups) && groups[tok.Group].Ok {
				write(groups[tok.Group].Text)
			}
		case TokUpperAll, TokLowerAll:
			mode = tok.Kind
		case TokUpperNext, TokLowerNext:
			next = tok.Kind
		case TokEnd:
			mode, next = TokEnd, TokEnd
		}
	}
	return b.String()
}

// RuneLen is a helper for UIs that work in runes.
func RuneLen(s string) int { return utf8.RuneCountInString(s) }
