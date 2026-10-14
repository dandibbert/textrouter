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
//
// Text (non-regex) fields understand \\ \t \n \s and \x{hhhh}, as the vendor
// help documents for "Find and Replace Text". Granularity applies to text
// finds only: 0 contains, 1 starts with (word start), 2 whole words, 3 ends
// with (word end). Measured: it has no effect on regex finds.
type FindOptions struct {
	Find         string `json:"find"`
	Replace      string `json:"replace"`
	Regex        bool   `json:"regex"`
	MatchCase    bool   `json:"matchCase"`
	RegexOptions string `json:"regexOptions,omitempty"`
	Granularity  int    `json:"granularity,omitempty"`
}

// Granularity values of text finds.
const (
	GranContains = iota
	GranStartsWith
	GranWholeWord
	GranEndsWith
)

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

// AllowComments reports whether the x option letter is set.
func (o FindOptions) AllowComments() bool { return strings.ContainsRune(o.RegexOptions, 'x') }

// Pattern returns the regexp2 pattern the options compile to.
func (o FindOptions) Pattern() (string, error) {
	if !o.Regex {
		pat := regexp2.Escape(UnescapeText(o.Find))
		switch o.Granularity {
		case GranStartsWith:
			pat = `(?<![\w])` + pat
		case GranWholeWord:
			pat = `(?<![\w])` + pat + `(?![\w])`
		case GranEndsWith:
			pat = pat + `(?![\w])`
		}
		return pat, nil
	}
	return translateICU(o.Find, o.DotAll())
}

// Compile builds the matcher. Literal patterns are escaped.
func (o FindOptions) Compile() (*regexp2.Regexp, error) {
	pat, err := o.Pattern()
	if err != nil {
		return nil, err
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
	if o.Regex && o.AllowComments() {
		opt |= regexp2.IgnorePatternWhitespace
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
		tmpl = Template{{Kind: TokLiteral, Text: UnescapeText(o.Replace)}}
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
			s, n := textEscape(r, i)
			lit.WriteString(s)
			i += n - 1
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

// textEscape decodes the escape at r[i] == '\\': \t tab, \n return, \s
// space, \x{hhhh} a code point; any other \c is c (measured: "\U" is "U",
// "\$" is "$"). It returns the text and the runes consumed.
func textEscape(r []rune, i int) (string, int) {
	if i+1 >= len(r) {
		return `\`, 1
	}
	switch c := r[i+1]; c {
	case 't':
		return "\t", 2
	case 'n':
		return "\n", 2
	case 's':
		return " ", 2
	case 'x':
		if i+2 < len(r) && r[i+2] == '{' {
			if end := indexRune(r, i+3, '}'); end > i+3 {
				var v rune
				ok := true
				for _, h := range r[i+3 : end] {
					d := strings.IndexRune("0123456789abcdef", unicode.ToLower(h))
					if d < 0 {
						ok = false
						break
					}
					v = v*16 + rune(d)
				}
				if ok && utf8.ValidRune(v) {
					return string(v), end + 1 - i
				}
			}
		}
		return "x", 2
	default:
		return string(c), 2
	}
}

// UnescapeText decodes the escapes text fields accept (see textEscape).
func UnescapeText(s string) string {
	if !strings.ContainsRune(s, '\\') {
		return s
	}
	r := []rune(s)
	var b strings.Builder
	for i := 0; i < len(r); i++ {
		if r[i] == '\\' {
			t, n := textEscape(r, i)
			b.WriteString(t)
			i += n - 1
			continue
		}
		b.WriteRune(r[i])
	}
	return b.String()
}

// RuneLen is a helper for UIs that work in runes.
func RuneLen(s string) int { return utf8.RuneCountInString(s) }
