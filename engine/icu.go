package engine

import (
	"fmt"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/unicode/runenames"
)

// TextSoap matches with ICU (NSRegularExpression). The matcher here is
// regexp2 (.NET syntax), so ICU-only syntax is rewritten before compiling.
// translateICU handles what users actually write: \h \H \v \V \R \X \N{…}
// \Q…\E \pL \p{Script=…} [[:alpha:]] possessive quantifiers, the w flag,
// and ICU's "." which never matches any line terminator (not only \n).

const (
	icuLineTerm = `\n\x0B\f\r\x{85}\x{2028}\x{2029}`
	icuHSpace   = `\t\x{20}\x{A0}\x{1680}\x{180E}\x{2000}-\x{200A}\x{202F}\x{205F}\x{3000}`
)

var posixClasses = map[string]string{
	"alpha":  `\p{L}\p{Nl}\p{Mn}\p{Mc}`,
	"alnum":  `\p{L}\p{Nl}\p{Mn}\p{Mc}\p{Nd}`,
	"digit":  `\p{Nd}`,
	"upper":  `\p{Lu}`,
	"lower":  `\p{Ll}`,
	"space":  `\s`,
	"blank":  icuHSpace,
	"punct":  `\p{P}`,
	"cntrl":  `\p{Cc}`,
	"xdigit": `0-9A-Fa-f`,
	"word":   `\w`,
	"graph":  `^\s\p{Cc}\p{Cn}\p{Cs}`,
	"print":  `^\p{Cc}\p{Cn}\p{Cs}`,
}

var propertyAliases = map[string]string{
	"alphabetic": "alpha", "uppercase": "upper", "lowercase": "lower",
	"white_space": "space", "whitespace": "space",
}

// translateICU rewrites an ICU pattern into regexp2 syntax. Named groups
// become plain groups so that numbering follows ICU (in order of their
// opening parenthesis; .NET numbers unnamed groups first); names[i] is the
// name of group i, if any.
func translateICU(pat string, dotAll bool) (string, []string, error) {
	r := []rune(pat)
	var out strings.Builder
	type frame struct{ start int }
	var groups []frame
	names := []string{""}
	byName := map[string]int{}
	lastAtom := -1
	// An inline (?s) anywhere means "." is managed by the pattern itself.
	if strings.Contains(pat, "(?s") || hasInlineFlag(pat, 's') {
		dotAll = true
	}
	emitAtom := func(s string) {
		lastAtom = out.Len()
		out.WriteString(s)
	}
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch c {
		case '\\':
			if i+2 < len(r) && r[i+1] == 'k' && (r[i+2] == '<' || r[i+2] == '{') {
				closer := '>'
				if r[i+2] == '{' {
					closer = '}'
				}
				if end := indexRune(r, i+3, closer); end > 0 {
					name := string(r[i+3 : end])
					if idx, ok := byName[name]; ok {
						emitAtom(fmt.Sprintf(`\k<%d>`, idx))
						i = end
						continue
					}
					return "", nil, fmt.Errorf("没有名为 %s 的分组", name)
				}
			}
			s, n, err := icuEscape(r, i, false)
			if err != nil {
				return "", nil, err
			}
			i += n - 1
			if s == "" {
				continue
			}
			emitAtom(s)
		case '[':
			s, n, err := icuClass(r, i)
			if err != nil {
				return "", nil, err
			}
			i += n - 1
			emitAtom(s)
		case '(':
			groups = append(groups, frame{out.Len()})
			if i+1 >= len(r) || r[i+1] != '?' {
				names = append(names, "")
			} else if i+2 < len(r) && (r[i+2] == '<' || r[i+2] == '\'') && i+3 < len(r) && r[i+3] != '=' && r[i+3] != '!' {
				closer := '>'
				if r[i+2] == '\'' {
					closer = '\''
				}
				if end := indexRune(r, i+3, closer); end > 0 {
					name := string(r[i+3 : end])
					names = append(names, name)
					byName[name] = len(names) - 1
					out.WriteRune('(')
					i = end
					continue
				}
			}
			if i+1 < len(r) && r[i+1] == '?' {
				j := i + 2
				k := j
				for k < len(r) && strings.ContainsRune("imsxwU-", r[k]) {
					k++
				}
				if k > j && k < len(r) && (r[k] == ')' || r[k] == ':') {
					flags := strings.Map(func(x rune) rune {
						if x == 'w' || x == 'U' {
							return -1
						}
						return x
					}, string(r[j:k]))
					if r[k] == ')' {
						groups = groups[:len(groups)-1]
						if flags != "" && flags != "-" {
							out.WriteString("(?" + flags + ")")
						}
						i = k
						lastAtom = -1
						continue
					}
					if flags == "" || flags == "-" {
						out.WriteString("(?:")
					} else {
						out.WriteString("(?" + flags + ":")
					}
					i = k
					continue
				}
			}
			out.WriteRune('(')
		case ')':
			out.WriteRune(')')
			if len(groups) > 0 {
				lastAtom = groups[len(groups)-1].start
				groups = groups[:len(groups)-1]
			}
		case '.':
			if dotAll {
				emitAtom(".")
			} else {
				emitAtom("[^" + icuLineTerm + "]")
			}
		case '*', '+', '?', '{':
			q := string(c)
			j := i + 1
			if c == '{' {
				k := i + 1
				for k < len(r) && (r[k] >= '0' && r[k] <= '9' || r[k] == ',') {
					k++
				}
				if k >= len(r) || r[k] != '}' || k == i+1 {
					emitAtom(`\{`)
					continue
				}
				q = string(r[i : k+1])
				j = k + 1
			}
			if j < len(r) && r[j] == '+' && lastAtom >= 0 {
				atom := out.String()[lastAtom:]
				s := out.String()[:lastAtom]
				out.Reset()
				out.WriteString(s)
				out.WriteString("(?>" + atom + q + ")")
				i = j
				lastAtom = -1
				continue
			}
			out.WriteString(q)
			if j < len(r) && r[j] == '?' {
				out.WriteRune('?')
				j++
			}
			i = j - 1
			lastAtom = -1
		default:
			emitAtom(string(c))
		}
	}
	return out.String(), names, nil
}

func hasInlineFlag(pat string, flag rune) bool {
	for i := 0; i+2 < len(pat); i++ {
		if pat[i] == '(' && pat[i+1] == '?' {
			for j := i + 2; j < len(pat); j++ {
				ch := rune(pat[j])
				if ch == flag {
					return true
				}
				if ch == '-' || !strings.ContainsRune("imsxwU", ch) {
					break
				}
			}
		}
	}
	return false
}

// icuEscape translates the escape at r[i] ('\\'). inClass selects the form
// usable inside [...] (no surrounding brackets). It returns the text and the
// number of runes consumed.
func icuEscape(r []rune, i int, inClass bool) (string, int, error) {
	if i+1 >= len(r) {
		return `\\`, 1, nil
	}
	c := r[i+1]
	wrap := func(set string, neg bool) string {
		if inClass {
			if neg {
				return "" // handled by caller for negated sets inside classes
			}
			return set
		}
		if neg {
			return "[^" + set + "]"
		}
		return "[" + set + "]"
	}
	switch c {
	case 'h':
		return wrap(icuHSpace, false), 2, nil
	case 'H':
		if inClass {
			return `\P{Zs}`, 2, nil
		}
		return wrap(icuHSpace, true), 2, nil
	case 'v':
		return wrap(icuLineTerm, false), 2, nil
	case 'V':
		if inClass {
			return `\P{Zl}`, 2, nil
		}
		return wrap(icuLineTerm, true), 2, nil
	case 'R':
		if inClass {
			return icuLineTerm, 2, nil
		}
		return `(?:\r\n|[` + icuLineTerm + `])`, 2, nil
	case 'X':
		if inClass {
			return `X`, 2, nil
		}
		return `(?:\r\n|\P{M}\p{M}*)`, 2, nil
	case 'Q':
		end := -1
		for k := i + 2; k+1 < len(r); k++ {
			if r[k] == '\\' && r[k+1] == 'E' {
				end = k
				break
			}
		}
		lit := r[i+2:]
		n := len(r) - i
		if end >= 0 {
			lit = r[i+2 : end]
			n = end + 2 - i
		}
		var b strings.Builder
		for _, x := range lit {
			b.WriteString(escapeRune(x))
		}
		return b.String(), n, nil
	case 'E':
		return "", 2, nil
	case 'N':
		if i+2 < len(r) && r[i+2] == '{' {
			end := indexRune(r, i+3, '}')
			if end < 0 {
				return "", 0, fmt.Errorf("\\N{ 缺少 }")
			}
			name := string(r[i+3 : end])
			ch, ok := runeByName(name)
			if !ok {
				return "", 0, fmt.Errorf("未知的 Unicode 字符名：%s", name)
			}
			return escapeRune(ch), end + 1 - i, nil
		}
		return `[^\n]`, 2, nil
	case 'x':
		if i+2 < len(r) && r[i+2] == '{' {
			if end := indexRune(r, i+3, '}'); end > 0 {
				return string(r[i : end+1]), end + 1 - i, nil
			}
		}
		return `\x`, 2, nil
	case 'k':
		if i+2 < len(r) && (r[i+2] == '<' || r[i+2] == '{') {
			closer := '>'
			if r[i+2] == '{' {
				closer = '}'
			}
			if end := indexRune(r, i+3, closer); end > 0 {
				return `\k<` + string(r[i+3:end]) + `>`, end + 1 - i, nil
			}
		}
		return `\k`, 2, nil
	case 'U':
		if i+10 <= len(r) {
			hex := string(r[i+2 : i+10])
			var v rune
			if _, err := fmt.Sscanf(hex, "%08x", &v); err == nil {
				return escapeRune(v), 10, nil
			}
		}
		return "U", 2, nil
	case 'p', 'P':
		neg := c == 'P'
		var name string
		n := 0
		if i+2 < len(r) && r[i+2] == '{' {
			end := indexRune(r, i+3, '}')
			if end < 0 {
				return "", 0, fmt.Errorf("\\p{ 缺少 }")
			}
			name = string(r[i+3 : end])
			n = end + 1 - i
		} else if i+2 < len(r) {
			name = string(r[i+2])
			n = 3
		} else {
			return `\` + string(c), 2, nil
		}
		return icuProperty(name, neg, inClass), n, nil
	}
	return string(r[i : i+2]), 2, nil
}

func icuProperty(name string, neg, inClass bool) string {
	if strings.HasPrefix(name, "^") {
		name, neg = name[1:], !neg
	}
	key, val, hasEq := strings.Cut(name, "=")
	if hasEq {
		k := strings.ToLower(strings.ReplaceAll(key, " ", ""))
		switch k {
		case "script", "sc", "scx", "script_extensions":
			name = val
		case "gc", "generalcategory", "general_category":
			name = val
		default:
			name = val
		}
	}
	n := strings.ReplaceAll(name, " ", "_")
	if strings.HasPrefix(n, "Is") && len(n) > 2 && unicode.Scripts[n[2:]] != nil {
		n = n[2:]
	}
	lower := strings.ToLower(n)
	if a, ok := propertyAliases[lower]; ok {
		lower = a
	}
	if set, ok := posixClasses[lower]; ok && unicode.Categories[n] == nil {
		if strings.HasPrefix(set, "^") {
			if neg {
				set = set[1:]
			} else if inClass {
				return `\S`
			} else {
				return "[" + set + "]"
			}
		} else if neg {
			if inClass {
				return set
			}
			return "[^" + set + "]"
		}
		if inClass {
			return set
		}
		return "[" + set + "]"
	}
	for s := range unicode.Scripts {
		if strings.EqualFold(s, n) {
			n = s
			break
		}
	}
	p := `\p{`
	if neg {
		p = `\P{`
	}
	return p + n + "}"
}

// icuClass translates a bracket expression starting at r[i] == '['.
func icuClass(r []rune, i int) (string, int, error) {
	var b strings.Builder
	b.WriteRune('[')
	j := i + 1
	if j < len(r) && r[j] == '^' {
		b.WriteRune('^')
		j++
	}
	if j < len(r) && r[j] == ']' {
		b.WriteString(`\]`)
		j++
	}
	for j < len(r) {
		c := r[j]
		switch {
		case c == ']':
			b.WriteRune(']')
			return b.String(), j + 1 - i, nil
		case c == '\\':
			s, n, err := icuEscape(r, j, true)
			if err != nil {
				return "", 0, err
			}
			b.WriteString(s)
			j += n
		case c == '[' && j+1 < len(r) && r[j+1] == ':':
			end := -1
			for k := j + 2; k+1 < len(r); k++ {
				if r[k] == ':' && r[k+1] == ']' {
					end = k
					break
				}
			}
			if end < 0 {
				b.WriteString(`\[`)
				j++
				continue
			}
			name := string(r[j+2 : end])
			neg := strings.HasPrefix(name, "^")
			b.WriteString(icuProperty(strings.TrimPrefix(name, "^"), neg, true))
			j = end + 2
		case c == '[':
			// ICU set union: [a-z[0-9]] — flatten the nested set.
			s, n, err := icuClass(r, j)
			if err != nil {
				return "", 0, err
			}
			inner := s[1 : len(s)-1]
			if strings.HasPrefix(inner, "^") {
				b.WriteString(`\[`)
				j++
				continue
			}
			b.WriteString(inner)
			j += n
		default:
			b.WriteRune(c)
			j++
		}
	}
	return "", 0, fmt.Errorf("缺少 ]")
}

func indexRune(r []rune, from int, c rune) int {
	for k := from; k < len(r); k++ {
		if r[k] == c {
			return k
		}
	}
	return -1
}

func escapeRune(c rune) string {
	if strings.ContainsRune(`\.+*?()|[]{}^$#-`, c) || unicode.IsSpace(c) {
		return fmt.Sprintf(`\x{%X}`, c)
	}
	return string(c)
}

var (
	runeNamesOnce sync.Once
	runeNames     map[string]rune
)

func runeByName(name string) (rune, bool) {
	runeNamesOnce.Do(func() {
		runeNames = make(map[string]rune, 40000)
		for c := rune(0); c <= 0x10FFFF; c++ {
			if c >= 0xD800 && c <= 0xDFFF {
				continue
			}
			if n := runenames.Name(c); n != "" && !strings.HasPrefix(n, "<") {
				runeNames[n] = c
			}
		}
	})
	c, ok := runeNames[strings.ToUpper(strings.TrimSpace(name))]
	return c, ok
}
