package engine

import (
	"fmt"
	"html"
	"math/rand/v2"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode"

	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

type impl struct {
	zh string
	fn func(string) string
}

const nbsp = "\u00a0"

var builtinImpls = map[string]impl{}

func reg(tok, zh string, fn func(string) string) { builtinImpls[tok] = impl{zh, fn} }

// unimpl registers a cleaner that only makes sense for rich text or OS
// services; it keeps the Chinese name but passes text through.
func unimpl(tok, zh string) { builtinImpls[tok] = impl{zh, nil} }

var (
	reSpaces      = regexp.MustCompile(` +`)
	reLeadSpaces  = regexp.MustCompile(`(?m)^ +`)
	reWSBeforeNL  = regexp.MustCompile(`[ \t]+\n`)
	reNL2Plus     = regexp.MustCompile(`\n{2,}`)
	reNL3Plus     = regexp.MustCompile(`\n{3,}`)
	reTags        = regexp.MustCompile(`(?s)<[^>]*>`)
	reURL         = regexp.MustCompile(`(?i)\b(?:https?://|ftp://|www\.)[^\s<>"'）)】]+`)
	reLineNumber  = regexp.MustCompile(`(?m)^[ \t]*\d+[.):]?[ \t]+`)
	reJammed      = regexp.MustCompile(`([a-z])([.!?,;:])([A-Za-z])`)
	reDateYMD     = regexp.MustCompile(`\b(\d{4})[-/.](\d{1,2})[-/.](\d{1,2})\b`)
	reDateMDY     = regexp.MustCompile(`\b(\d{1,2})[-/.](\d{1,2})[-/.](\d{4})\b`)
	reNumEntity   = regexp.MustCompile(`&#(x[0-9A-Fa-f]+|\d+);`)
	reHexPairs    = regexp.MustCompile(`[0-9A-Fa-f]{2}`)
	reParaSplit   = regexp.MustCompile(`\n\s*\n`)
)

// ---------------------------------------------------------------- whitespace

func removeExtraSpaces(s string) string {
	s = strings.ReplaceAll(s, nbsp, " ")
	s = reSpaces.ReplaceAllString(s, " ")
	s = strings.ReplaceAll(s, " \n", "\n")
	return reLeadSpaces.ReplaceAllString(s, "")
}

func trimSpaceChars(l string) string { return strings.Trim(l, " ") }

func mapLines(s string, f func(string) string) string {
	ls := strings.Split(s, "\n")
	for i, l := range ls {
		ls[i] = f(l)
	}
	return strings.Join(ls, "\n")
}

func trimLines(s string) string {
	return mapLines(s, trimSpaceChars)
}

func isHSpace(r rune) bool { return r == ' ' || r == '\t' || r == '\u00a0' }

func stripRuns(s, ch string, n int) string {
	re := regexp.MustCompile(regexp.QuoteMeta(ch) + "{" + fmt.Sprint(n) + ",}")
	return re.ReplaceAllString(s, ch)
}

func removeExtraReturns(s string) string {
	s = reNL2Plus.ReplaceAllString(s, "\n\n")
	if s == "\n" {
		return ""
	}
	return s
}

// makeParagraphs: single returns become spaces, runs of 2+ stay.
func makeParagraphs(s string) string {
	s = reWSBeforeNL.ReplaceAllString(s, "\n")
	r := []rune(s)
	var b strings.Builder
	for i, c := range r {
		if c == '\n' && (i == 0 || r[i-1] != '\n') && (i+1 >= len(r) || r[i+1] != '\n') && !(i+1 < len(r) && r[i+1] == '-') {
			if len(r) == 1 {
				b.WriteRune(c)
				continue
			}
			b.WriteByte(' ')
			continue
		}
		b.WriteRune(c)
	}
	return b.String()
}

func removeControl(s string) string {
	return strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			return -1
		}
		return r
	}, s)
}

// ---------------------------------------------------------------- case

const sentenceEnd = ".!?"

func capitalizeSentences(s string) string {
	var b strings.Builder
	start, lineStart := true, true
	for _, ch := range s {
		if start && unicode.IsLetter(ch) {
			b.WriteRune(unicode.ToUpper(ch))
			start, lineStart = false, false
			continue
		}
		b.WriteRune(ch)
		switch {
		case ch == '\n':
			start, lineStart = true, true
		case strings.ContainsRune(sentenceEnd, ch):
			start, lineStart = true, false
		case unicode.IsSpace(ch):
			if lineStart {
				start = false // measured: leading blanks of the text cancel
			}
		case !strings.ContainsRune("\"'(\u201c\u2018", ch):
			start, lineStart = false, false
		}
	}
	return b.String()
}

func isWord(r rune) bool { return r == '_' || unicode.IsLetter(r) || unicode.IsDigit(r) }

func capitalizeWordsKeep(s string) string {
	var b strings.Builder
	prev := ' '
	for _, c := range s {
		if isWord(c) && !isWord(prev) {
			b.WriteRune(unicode.ToTitle(c))
		} else {
			b.WriteRune(c)
		}
		prev = c
	}
	return b.String()
}

func capitalizeWords(s string) string { return capitalizeWordsKeep(strings.ToLower(s)) }

var titleSmall = map[string]bool{"a": true, "an": true, "and": true, "as": true, "at": true, "but": true, "by": true,
	"en": true, "for": true, "if": true, "in": true, "of": true, "on": true, "or": true, "the": true, "to": true, "via": true}

func hasInternalCap(w string) bool {
	r := []rune(w)
	for _, c := range r[min(1, len(r)):] {
		if unicode.IsUpper(c) {
			return true
		}
	}
	return false
}

// TitleCase capitalises words; extraSmall are additional always-lowercase
// words and exceptions are left untouched (text.titlecase action).
func TitleCase(s string, extraSmall, exceptions []string) string {
	small := map[string]bool{}
	for k := range titleSmall {
		small[k] = true
	}
	for _, w := range extraSmall {
		small[strings.ToLower(w)] = true
	}
	exc := map[string]bool{}
	for _, w := range exceptions {
		exc[w] = true
	}
	return mapLines(s, func(line string) string {
		words := strings.Fields(line)
		for i, w := range words {
			prevOpen := i > 0 && !endsAlnum(words[i-1])
			switch {
			case exc[w] || isCamel(w):
			case small[strings.ToLower(w)] && i != 0 && i != len(words)-1 && !prevOpen:
				words[i] = strings.ToLower(w)
			default:
				if allUpper(w) {
					w = strings.ToLower(w)
				}
				r := []rune(w)
				for k, c := range r {
					if unicode.IsLetter(c) || unicode.IsDigit(c) {
						r[k] = unicode.ToUpper(c)
						break
					}
				}
				words[i] = string(r)
			}
		}
		return strings.Join(words, " ")
	})
}

// isCamel: a lowercase letter directly followed by an uppercase one
// (iPhone, TextSoap, macOS) — such words keep their capitalisation.
func isCamel(w string) bool {
	r := []rune(w)
	for i := 1; i < len(r); i++ {
		if unicode.IsLower(r[i-1]) && unicode.IsUpper(r[i]) {
			return true
		}
	}
	return false
}

func allUpper(w string) bool {
	has := false
	for _, c := range w {
		if unicode.IsLower(c) {
			return false
		}
		if unicode.IsUpper(c) {
			has = true
		}
	}
	return has
}

func endsAlnum(w string) bool {
	r := []rune(w)
	c := r[len(r)-1]
	return unicode.IsLetter(c) || unicode.IsDigit(c)
}

func capitalizeLines(lower bool) func(string) string {
	return func(s string) string {
		return mapLines(s, func(l string) string {
			if lower {
				l = strings.ToLower(l)
			}
			r := []rune(l)
			for i, c := range r {
				if unicode.IsLetter(c) {
					r[i] = unicode.ToUpper(c)
					break
				}
			}
			return string(r)
		})
	}
}

func replaceWords(s string, words []string) string {
	for _, w := range words {
		re := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(w) + `\b`)
		s = re.ReplaceAllString(s, w)
	}
	return s
}

var techNames = []string{"JavaScript", "TypeScript", "GitHub", "GitLab", "PostgreSQL", "MySQL", "SQLite", "MongoDB", "macOS", "iOS", "iPadOS",
	"watchOS", "tvOS", "visionOS", "Xcode", "WordPress", "YouTube", "LinkedIn", "PayPal", "OpenAI", "ChatGPT", "Wi-Fi", "Bluetooth",
	"USB", "HTML", "CSS", "JSON", "XML", "API", "URL", "HTTP", "HTTPS", "PHP", "Node.js", "npm", "Linux", "Windows", "Android", "Google", "Microsoft"}
var appleNames = []string{"iPhone", "iPad", "iPod", "iMac", "iCloud", "iTunes", "iMessage", "iWork", "iMovie", "iBooks", "AirPods", "AirPlay",
	"AirDrop", "MacBook", "FaceTime", "HomePod", "Apple Watch", "Apple TV", "macOS", "iOS", "iPadOS", "watchOS", "tvOS", "visionOS"}

// ---------------------------------------------------------------- typography

func straightenQuotes(s string) string {
	return strings.NewReplacer("\u201c", `"`, "\u201d", `"`, "\u2018", "'", "\u2019", "'").Replace(s)
}

func smartenQuotes(s string) string {
	var b strings.Builder
	prev := ' '
	for _, ch := range s {
		switch ch {
		case '"':
			if unicode.IsSpace(prev) {
				b.WriteRune('\u201c')
			} else {
				b.WriteRune('\u201d')
			}
		case '\'':
			if unicode.IsSpace(prev) {
				b.WriteRune('\u2018')
			} else {
				b.WriteRune('\u2019')
			}
		default:
			b.WriteRune(ch)
		}
		prev = ch
	}
	return b.String()
}

func stripDiacritics(s string) string {
	t := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	out, _, err := transform.String(t, s)
	if err != nil {
		return s
	}
	return out
}

func internetFriendly(s string) string {
	s = stripDiacritics(s)
	s = straightenQuotes(s)
	return strings.NewReplacer(nbsp, " ", "\u2022", "-", "\u00a9", "(C)", "\u00ae", "(R)", "\u2122", "(TM)", "\u2026", "...", "\u2014", "--", "\u2013", " - ").Replace(s)
}

func removeHighASCII(s string) string {
	s = straightenQuotes(stripDiacritics(s))
	s = strings.NewReplacer(nbsp, " ", "\u00a9", "(C)", "\u2026", "...", "\u2014", "-", "\u2013", "-").Replace(s)
	return strings.Map(func(r rune) rune {
		if r >= 128 && r <= 255 {
			return -1
		}
		return r
	}, s)
}

func convertToASCII(s string) string {
	return removeHighASCII(internetFriendly(s))
}

// ---------------------------------------------------------------- quoting / wrapping

var reFwd = regexp.MustCompile(`^(?:[ \t]*>)+[ \t]*`)

func removeForwardingRaw(s string) string {
	return mapLines(s, func(l string) string { return reFwd.ReplaceAllString(l, "") })
}

// removeForwarding (measured): NBSP become spaces and leading spaces go too.
func removeForwarding(s string) string {
	s = strings.ReplaceAll(s, nbsp, " ")
	return mapLines(s, func(l string) string { return strings.TrimLeft(reFwd.ReplaceAllString(l, ""), " ") })
}

func convertMIME(s string) string {
	dec := func(prefix string) func(string) string {
		re := regexp.MustCompile(prefix + `([0-9A-Fa-f]{2})`)
		return func(s string) string {
			return re.ReplaceAllStringFunc(s, func(m string) string {
				var v int
				fmt.Sscanf(m[1:], "%x", &v)
				return string(rune(v))
			})
		}
	}
	return dec("%")(dec("=")(s))
}

func removeDuplicateLines(s string) string {
	if s == "\n" {
		return "" // measured
	}
	seen := map[string]bool{}
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l == "" || !seen[l] {
			seen[l] = true
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

func greedyWrap(words []string, width int) []string {
	var out, buf []string
	cur := 0
	for _, w := range words {
		n := len([]rune(w))
		add := n
		if len(buf) > 0 {
			add++
		}
		if len(buf) > 0 && cur+add > width {
			out = append(out, strings.Join(buf, " "))
			buf, cur = []string{w}, n
		} else {
			buf = append(buf, w)
			cur += add
		}
	}
	return append(out, strings.Join(buf, " "))
}

// WrapAt fixes paragraphs and greedily wraps them at width characters.
// Lines that already fit keep their spacing (measured).
func WrapAt(s string, width int) string {
	s = makeParagraphs(s)
	s = strings.TrimRightFunc(s, isHSpace)
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if len([]rune(l)) > width {
			out = append(out, greedyWrap(strings.Fields(l), width)...)
		} else {
			out = append(out, l)
		}
	}
	return strings.Join(out, "\n")
}

// QuoteText rewraps and prefixes every line with " > ".
func QuoteText(s string, width int, prefix string) string {
	s = makeParagraphs(s)
	s = strings.TrimRightFunc(s, isHSpace)
	ls := strings.Split(WrapAt(s, width), "\n")
	for i, l := range ls {
		if i == len(ls)-1 && l == "" && i > 0 {
			continue // a final empty line is not quoted (measured)
		}
		ls[i] = prefix + l
	}
	return strings.Join(ls, "\n")
}

func increaseQuote(s string) string { return mapLines(s, func(l string) string { return ">" + l }) }

// ---------------------------------------------------------------- html

func extractHTML(s string) string {
	s = reTags.ReplaceAllString(s, "")
	s = html.UnescapeString(s)
	s = strings.ReplaceAll(s, "\t", "")
	s = removeExtraSpaces(s)
	s = regexp.MustCompile(` +\n`).ReplaceAllString(s, "\n")
	s = reNL3Plus.ReplaceAllString(s, "\n\n")
	if s == "\n" {
		return ""
	}
	return s
}

func wrap(pre, post string) func(string) string { return func(s string) string { return pre + s + post } }

func textToNumericEntities(all bool) func(string) string {
	return func(s string) string {
		var b strings.Builder
		for _, r := range s {
			if all || r > 127 || r == '<' || r == '>' || r == '&' || r == '"' {
				fmt.Fprintf(&b, "&#%d;", r)
			} else {
				b.WriteRune(r)
			}
		}
		return b.String()
	}
}

func textToNamedEntities(s string) string {
	s = html.EscapeString(s)
	return strings.NewReplacer("\u00a0", "&nbsp;", "\u00a9", "&copy;", "\u00ae", "&reg;", "\u2122", "&trade;", "\u2026", "&hellip;",
		"\u2014", "&mdash;", "\u2013", "&ndash;", "\u201c", "&ldquo;", "\u201d", "&rdquo;", "\u2018", "&lsquo;", "\u2019", "&rsquo;",
		"\u00e9", "&eacute;", "\u00e8", "&egrave;", "\u00e0", "&agrave;", "\u00fc", "&uuml;", "\u00f6", "&ouml;", "\u00e4", "&auml;").Replace(s)
}

func htmlList(tag string) func(string) string {
	return func(s string) string {
		var b strings.Builder
		b.WriteString("<" + tag + ">\n")
		for _, l := range strings.Split(s, "\n") {
			if strings.TrimSpace(l) != "" {
				b.WriteString("  <li>" + l + "</li>\n")
			}
		}
		b.WriteString("</" + tag + ">")
		return b.String()
	}
}

// ---------------------------------------------------------------- lines

func lines(s string) []string { return strings.Split(s, "\n") }

func sortLines(desc bool) func(string) string {
	return func(s string) string {
		ls := lines(strings.TrimSuffix(s, "\n"))
		sort.SliceStable(ls, func(i, j int) bool {
			if desc {
				return ls[i] > ls[j]
			}
			return ls[i] < ls[j]
		})
		return strings.Join(ls, "\n") + "\n"
	}
}

func reverseLines(s string) string {
	ls := lines(s)
	for i, j := 0, len(ls)-1; i < j; i, j = i+1, j-1 {
		ls[i], ls[j] = ls[j], ls[i]
	}
	return strings.Join(ls, "\n")
}

func shuffleLines(s string) string {
	ls := lines(s)
	rand.Shuffle(len(ls), func(i, j int) { ls[i], ls[j] = ls[j], ls[i] })
	return strings.Join(ls, "\n")
}

// ---------------------------------------------------------------- misc

func reverseRunes(s string) string {
	r := []rune(s)
	for i, j := 0, len(r)-1; i < j; i, j = i+1, j-1 {
		r[i], r[j] = r[j], r[i]
	}
	return string(r)
}

var reWordRun = regexp.MustCompile(`\S+`)

func reverseWordOrder(s string) string {
	return mapLines(s, func(l string) string {
		w := strings.Fields(l)
		for i, j := 0, len(w)-1; i < j; i, j = i+1, j-1 {
			w[i], w[j] = w[j], w[i]
		}
		return strings.Join(w, " ")
	})
}

func rot13(s string) string {
	return strings.Map(func(r rune) rune {
		switch {
		case r >= 'a' && r <= 'z':
			return 'a' + (r-'a'+13)%26
		case r >= 'A' && r <= 'Z':
			return 'A' + (r-'A'+13)%26
		}
		return r
	}, s)
}

func randomCase(s string) string {
	return strings.Map(func(r rune) rune {
		if rand.IntN(2) == 0 {
			return unicode.ToUpper(r)
		}
		return unicode.ToLower(r)
	}, s)
}

func extractURLs(appendMode bool) func(string) string {
	return func(s string) string {
		urls := strings.Join(reURL.FindAllString(s, -1), "\n")
		if appendMode {
			if urls == "" {
				return s
			}
			return s + "\n\n" + urls
		}
		return urls
	}
}

func unicodeNames(s string) string {
	var b strings.Builder
	for _, r := range s {
		if r < 128 {
			b.WriteRune(r)
		} else {
			fmt.Fprintf(&b, "{U+%04X}", r)
		}
	}
	return b.String()
}

func dateFmt(layoutFor func(y, m, d string) string) func(string) string {
	return func(s string) string {
		s = reDateYMD.ReplaceAllStringFunc(s, func(x string) string {
			p := reDateYMD.FindStringSubmatch(x)
			return layoutFor(p[1], pad2(p[2]), pad2(p[3]))
		})
		return s
	}
}

func pad2(s string) string {
	if len(s) == 1 {
		return "0" + s
	}
	return s
}

func dateLong(style string) func(string) string {
	return func(s string) string {
		return reDateYMD.ReplaceAllStringFunc(s, func(x string) string {
			p := reDateYMD.FindStringSubmatch(x)
			t, err := time.Parse("2006-01-02", p[1]+"-"+pad2(p[2])+"-"+pad2(p[3]))
			if err != nil {
				return x
			}
			switch style {
			case "short":
				return t.Format("1/2/06")
			case "medium":
				return t.Format("Jan 2, 2006")
			case "long":
				return t.Format("January 2, 2006")
			}
			return t.Format("Monday, January 2, 2006")
		})
	}
}

func init() {
	reg("#SCRUB", "一键清理 Scrub", scrub)
	reg("#MYSCRUB", "我的清理 MyScrub", scrub)
	reg("#REMOVE_EXTRA_SPACES", "删除多余空格", removeExtraSpaces)
	reg("#REMOVE_EXTRA_SPACES_ORIG", "删除多余空格（旧版）", func(s string) string { return reSpaces.ReplaceAllString(s, " ") })
	reg("#REMOVE_WHITESPACE_AT_BEGINNING", "删除段首空白", func(s string) string {
		return mapLines(s, func(l string) string { return strings.TrimLeftFunc(l, isHSpace) })
	})
	reg("#REMOVE_FORWARDING_CHARS", "删除转发符号 (>)", removeForwarding)
	reg("#STRIP_QUOTING_CHARS", "去除引用符号 (>)", removeForwarding)
	reg("#CONVERT_MIME", "解码 MIME 字符（%20 / =20）", convertMIME)
	reg("#MAKE_PARAGRAPHS", "合并成段落", makeParagraphs)
	reg("#REMOVE_ALL_TABS", "删除所有制表符", func(s string) string { return strings.ReplaceAll(s, "\t", "") })
	reg("#REMOVE_ALL_SPACES", "删除所有空格", func(s string) string { return strings.ReplaceAll(s, " ", "") })
	reg("#REMOVE_ALL_WHITESPACE", "删除所有空白", func(s string) string {
		return strings.Map(func(r rune) rune {
			if unicode.IsSpace(r) {
				return -1
			}
			return r
		}, s)
	})
	reg("#REMOVE_CONTROL_CHARS", "删除控制字符", removeControl)
	reg("#REMOVE_EXTRA_RETURNS", "删除多余换行", removeExtraReturns)
	reg("#UPPERCASE", "全部大写", strings.ToUpper)
	reg("#LOWERCASE", "全部小写", strings.ToLower)
	reg("#CONVERT_TO_UPPERCASE", "转大写（旧）", strings.ToUpper)
	reg("#CONVERT_TO_LOWERCASE", "转小写（旧）", strings.ToLower)
	reg("#CAPITALIZE_SENTENCES", "句首大写", capitalizeSentences)
	reg("#CAPITALIZE_SENTENCES_ALT", "句首大写（其余小写）", func(s string) string { return capitalizeSentences(strings.ToLower(s)) })
	reg("#TITLE_CASE1", "标题式大小写", func(s string) string { return TitleCase(s, nil, nil) })
	reg("#CAPITALIZE_WORDS", "每词首字母大写", capitalizeWords)
	reg("#CAPITALIZE_COMMON_TECH_NAMES", "规范常见技术名词", func(s string) string { return replaceWords(s, techNames) })
	reg("#CAPITALIZE_LINES", "行首大写", capitalizeLines(false))
	reg("#CAPITALIZE_LINES_ALT", "行首大写（其余小写）", capitalizeLines(true))
	reg("#RANDOMCASE", "随机大小写", randomCase)
	reg("#MACOS_NAME_FIX", "规范 macOS 写法", func(s string) string {
		return regexp.MustCompile(`(?i)\bmac ?os ?x?\b`).ReplaceAllStringFunc(s, func(m string) string {
			if strings.HasSuffix(strings.ToLower(m), "x") && !strings.HasSuffix(strings.ToLower(m), "os") {
				return "macOS"
			}
			return "macOS"
		})
	})
	reg("#APPLE_NAME_FIX", "规范苹果产品名", func(s string) string { return replaceWords(s, appleNames) })
	reg("#REVERSE_CHARACTERS", "反转全部字符", reverseRunes)
	reg("#REVERSE_WORDORDER", "反转词序", reverseWordOrder)
	reg("#REVERSE_CHARS_INEACHWORD", "反转每个词的字符", func(s string) string { return reWordRun.ReplaceAllStringFunc(s, reverseRunes) })
	reg("#STRIP_DIACRITICS", "去除变音符号", stripDiacritics)
	reg("#MAKE_UNICODENAMES", "非 ASCII 显示为码点", unicodeNames)
	reg("#CONVERT_TO_ASCII", "转换为 ASCII", convertToASCII)
	reg("#CONVERT_BULLETS", "项目符号转 ASCII", func(s string) string { return strings.ReplaceAll(s, "\u2022", "-") })
	reg("#STRAIGHTEN_QUOTES", "弯引号转直引号", straightenQuotes)
	reg("#SMARTEN_QUOTES", "直引号转弯引号", smartenQuotes)
	reg("#HTML_SMARTEN_QUOTES", "HTML 中直引号转弯引号", func(s string) string {
		var b strings.Builder
		last := 0
		for _, loc := range reTags.FindAllStringIndex(s, -1) {
			b.WriteString(smartenQuotes(s[last:loc[0]]))
			b.WriteString(s[loc[0]:loc[1]])
			last = loc[1]
		}
		b.WriteString(smartenQuotes(s[last:]))
		return b.String()
	})
	reg("#EXPAND_TABS", "制表符转 4 个空格", func(s string) string { return strings.ReplaceAll(s, "\t", "    ") })
	reg("#REWRAP_TEXT", "重新折行（70）", func(s string) string { return WrapAt(s, 70) })
	reg("#WRAP_AT_50", "按 50 字符折行", func(s string) string { return WrapAt(s, 50) })
	reg("#WRAP_AT_60", "按 60 字符折行", func(s string) string { return WrapAt(s, 60) })
	reg("#WRAP_AT_70", "按 70 字符折行", func(s string) string { return WrapAt(s, 70) })
	reg("#QUOTE_TEXT", "引用文本 ( > )", func(s string) string { return QuoteText(s, 70, " > ") })
	reg("#QUOTE_L1", "引用一级", func(s string) string { return QuoteText(s, 70, "> ") })
	reg("#QUOTE_L2", "引用二级", func(s string) string { return QuoteText(s, 70, ">> ") })
	reg("#QUOTE_L3", "引用三级", func(s string) string { return QuoteText(s, 70, ">>> ") })
	reg("#QUOTE_L4", "引用四级", func(s string) string { return QuoteText(s, 70, ">>>> ") })
	reg("#QUOTE_BASED_FIRST_LINE", "按首行引用级别引用", func(s string) string {
		first := strings.SplitN(s, "\n", 2)[0]
		pre := reFwd.FindString(first)
		return QuoteText(removeForwarding(s), 70, pre)
	})
	reg("#INCREASE_QUOTE_LEVEL", "提高引用级别", increaseQuote)
	reg("#REMOVE_FORWARDED_TEXT", "删除被引用的行", func(s string) string {
		var out []string
		for _, l := range lines(s) {
			if !strings.HasPrefix(strings.TrimLeft(l, " \t"), ">") {
				out = append(out, l)
			}
		}
		return strings.Join(out, "\n")
	})
	reg("#NETTEXT", "互联网友好文本", internetFriendly)
	reg("#REMOVE_HIASCII", "删除高位 ASCII 字符", removeHighASCII)
	reg("#1CR_TO_2CR", "单换行变双换行", func(s string) string {
		return regexp.MustCompile(`\n+`).ReplaceAllStringFunc(s, func(m string) string {
			if m == "\n" {
				return "\n\n"
			}
			return m
		})
	})
	reg("#1CR_TO_SPACE", "单换行变空格", makeParagraphs)
	for n := 2; n <= 4; n++ {
		n := n
		reg(fmt.Sprintf("#STRIP_GT_%dCR", n-1), fmt.Sprintf("%d 个以上换行合并为 1 个", n), func(s string) string { return stripRuns(s, "\n", n) })
		reg(fmt.Sprintf("#STRIP_GT_%dSPACE", n-1), fmt.Sprintf("%d 个以上空格合并为 1 个", n), func(s string) string { return stripRuns(s, " ", n) })
		reg(fmt.Sprintf("#STRIP_GT_%dTAB", n-1), fmt.Sprintf("%d 个以上制表符合并为 1 个", n), func(s string) string { return stripRuns(s, "\t", n) })
	}
	reg("#MULTIPLE_CR_TO_1CR", "多个换行变 1 个", func(s string) string {
		s = regexp.MustCompile(`\n+`).ReplaceAllString(s, "\n")
		if s == "\n" {
			return ""
		}
		return s
	})
	reg("#MULTIPLE_CR_TO_2CR", "多个换行变 2 个", removeExtraReturns)
	reg("#ELLIPSIS_TO_3PERIODS", "省略号 … 转 ...", func(s string) string { return strings.ReplaceAll(s, "\u2026", "...") })
	reg("#EM_DASH_TO_2HYPHENS", "破折号 — 转 --", func(s string) string { return strings.ReplaceAll(s, "\u2014", "--") })
	reg("#EN_DASH_TO_SPACE_DASH_SPACE", "短破折号 – 转 \" - \"", func(s string) string { return strings.ReplaceAll(s, "\u2013", " - ") })
	reg("#3PERIODS_TO_ELLIPSIS", "... 转省略号 …", func(s string) string { return strings.ReplaceAll(s, "...", "\u2026") })
	reg("#2HYPHENS_TO_EM_DASH", "-- 转破折号 —", func(s string) string { return strings.ReplaceAll(s, "--", "\u2014") })
	reg("#SPACE_DASH_SPACE_TO_EN_DASH", "\" - \" 转短破折号 –", func(s string) string { return strings.ReplaceAll(s, " - ", " \u2013 ") })
	reg("#ENABLE_LIGATURES", "启用连字", func(s string) string { return strings.NewReplacer("ffi", "\ufb03", "ffl", "\ufb04", "ff", "\ufb00", "fi", "\ufb01", "fl", "\ufb02").Replace(s) })
	reg("#DISABLE_LIGATURES", "拆开连字", func(s string) string {
		return strings.NewReplacer("\ufb03", "ffi", "\ufb04", "ffl", "\ufb00", "ff", "\ufb01", "fi", "\ufb02", "fl").Replace(s)
	})
	reg("#LINE_ENDING_LF", "换行符：Unix (LF)", func(s string) string { return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n") })
	reg("#LINE_ENDING_CR", "换行符：经典 Mac (CR)", func(s string) string {
		return strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\r"), "\n", "\r")
	})
	reg("#LINE_ENDING_CRLF", "换行符：Windows (CRLF)", func(s string) string {
		s = strings.ReplaceAll(strings.ReplaceAll(s, "\r\n", "\n"), "\r", "\n")
		return strings.ReplaceAll(s, "\n", "\r\n")
	})
	reg("#TRIM_BEGINNING_OF_LINES", "去掉行首空格", func(s string) string { return mapLines(s, func(l string) string { return strings.TrimLeft(l, " ") }) })
	reg("#TRIM_END_OF_LINES", "去掉行尾空格", func(s string) string { return mapLines(s, func(l string) string { return strings.TrimRight(l, " ") }) })
	reg("#TRIM_LINES", "去掉行首尾空白", trimLines)
	reg("#CONVERT_NON_BREAKING_SPACES", "不换行空格转普通空格", func(s string) string { return strings.ReplaceAll(s, nbsp, " ") })
	reg("#URL_EXTRACT_BY_REPLACING", "提取网址（替换原文）", extractURLs(false))
	reg("#URL_EXTRACT_BY_APPENDING", "提取网址（追加到末尾）", extractURLs(true))
	reg("#NOTHING", "什么都不做", func(s string) string { return s })
	reg("#LINE_SORT_ASCENDING", "行排序（升序）", sortLines(false))
	reg("#LINE_SORT_DESCENDING", "行排序（降序）", sortLines(true))
	reg("#LINE_REVERSE", "反转行顺序", reverseLines)
	reg("#LINE_RANDOMIZE", "随机打乱行", shuffleLines)
	reg("#REMOVE_DUPLICATE_LINES", "删除重复行", removeDuplicateLines)
	reg("#REMOVE_LINENUMBERS", "删除行号", func(s string) string { return reLineNumber.ReplaceAllString(s, "") })
	for _, t := range []string{"#LINESPACE_10", "#LINESPACE_12", "#LINESPACE_15", "#LINESPACE_20", "#PARAGRAPH_SINGLESPACE", "#PARAGRAPH_ALIGN_LEFT",
		"#PARAGRAPH_ALIGN_RIGHT", "#PARAGRAPH_ALIGN_CENTER", "#PARAGRAPH_ALIGN_FULL", "#PARAGRAPH_CLEARTABSTOPS", "#PARAGRAPH_RESETTABSTOPS",
		"#REMOVE_STYLE_FROM_TEXT", "#REMOVE_FORECOLOR_FROM_TEXT", "#REMOVE_BACKCOLOR_FROM_TEXT", "#REMOVE_HYPERLINKING_FROM_TEXT",
		"#URL_MAKECLICKABLE", "#NOTIFY_STATS", "#NOTIFY_STATS_REMOVE", "#MARKDOWN_TEXT_RTF", "#MARKDOWN_TEXT_PREVIEW",
		"#MARKDOWN_TEXT_MARKED2", "#HTML_TEXT_TO_RTF", "#MIXED_MARKDOWN", "#MARKDOWN_TEXT", "#MARKDOWN_TEXT_APPEND", "#HTML_TO_MARKDOWN_TEXT"} {
		unimpl(t, "")
	}
	reg("#MAKE_PLAIN_TEXT", "转为纯文本", func(s string) string { return s })
	reg("#NORMALIZE_DATE_SHORTFORM", "日期：短格式", dateLong("short"))
	reg("#NORMALIZE_DATE_MEDIUMFORM", "日期：中格式", dateLong("medium"))
	reg("#NORMALIZE_DATE_LONGFORM", "日期：长格式", dateLong("long"))
	reg("#NORMALIZE_DATE_FULLFORM", "日期：完整格式", dateLong("full"))
	reg("#DATEFORMAT_US", "日期：MM-DD-YYYY", dateFmt(func(y, m, d string) string { return m + "-" + d + "-" + y }))
	reg("#DATEFORMAT_UK", "日期：DD-MM-YYYY", dateFmt(func(y, m, d string) string { return d + "-" + m + "-" + y }))
	reg("#DATEFORMAT_YEARFIRST", "日期：YYYY-MM-DD", func(s string) string {
		return reDateMDY.ReplaceAllStringFunc(dateFmt(func(y, m, d string) string { return y + "-" + m + "-" + d })(s), func(x string) string {
			p := reDateMDY.FindStringSubmatch(x)
			return p[3] + "-" + pad2(p[1]) + "-" + pad2(p[2])
		})
	})
	reg("#DATENOW", "展开 DateNow 为今天日期", func(s string) string { return strings.ReplaceAll(s, "DateNow", time.Now().Format("2006-01-02")) })
	reg("#ENCRYPT_WITH_ROT13", "ROT13 编码", rot13)
	reg("#STRING_TO_HEX", "文本转十六进制", func(s string) string { return fmt.Sprintf("%X", []byte(s)) })
	reg("#HEX_TO_STRING", "十六进制转文本", func(s string) string {
		clean := regexp.MustCompile(`[^0-9A-Fa-f]`).ReplaceAllString(s, "")
		var b []byte
		for _, p := range reHexPairs.FindAllString(clean, -1) {
			var v byte
			fmt.Sscanf(p, "%02x", &v)
			b = append(b, v)
		}
		return string(b)
	})
	reg("#REPLACE_AT_SIGN", "@ 替换为 (AT)", func(s string) string { return strings.ReplaceAll(s, "@", "(AT)") })
	reg("#FIX_JAMMED_WORDS", "修复粘连单词（标点后补空格）", func(s string) string { return reJammed.ReplaceAllString(s, "$1$2 $3") })
	reg("#EXTRACT_TEXT_FROM_HTML", "从 HTML 源码提取文本", extractHTML)
	reg("#HTML_ENTITY_TO_TEXT", "HTML 实体转文本", html.UnescapeString)
	reg("#TEXT_TO_NAMED_HTML_ENTITY", "文本转命名 HTML 实体", textToNamedEntities)
	reg("#TEXT_TO_NAMED_HTML_ENTITY_ALT", "文本转命名 HTML 实体（备选）", textToNamedEntities)
	reg("#TEXT_TO_NUMERIC_HTML_ENTITY", "文本转数字 HTML 实体", textToNumericEntities(false))
	reg("#ALL_TEXT_TO_NUMERIC_HTML_ENTITIES", "全部文本转数字实体", textToNumericEntities(true))
	reg("#HTML_CREATE_OL", "HTML：生成有序列表", htmlList("ol"))
	reg("#HTML_CREATE_UL", "HTML：生成无序列表", htmlList("ul"))
	for _, t := range []struct{ tok, zh, tag string }{
		{"ITALIC", "斜体", "i"}, {"STRONG", "Strong", "strong"}, {"EMPHASIZED", "强调", "em"}, {"UNDERLINE", "下划线", "u"},
		{"PREFORMATTED", "预格式", "pre"}, {"DIV", "div", "div"}, {"PARAGRAPH", "段落", "p"}, {"SPAN", "span", "span"},
		{"BLOCKQUOTE", "引用块", "blockquote"}, {"CITE", "引文", "cite"}, {"H1", "h1", "h1"}, {"H2", "h2", "h2"}, {"H3", "h3", "h3"},
		{"H4", "h4", "h4"}, {"H5", "h5", "h5"}, {"H6", "h6", "h6"}, {"UL", "无序列表", "ul"}, {"OL", "有序列表", "ol"}, {"LI", "列表项", "li"},
		{"TABLE", "表格", "table"}, {"TABLEROW", "表格行", "tr"}, {"TABLECELL", "单元格", "td"},
	} {
		reg("#HTML_TAG_"+t.tok, "HTML 标签："+t.zh, wrap("<"+t.tag+">", "</"+t.tag+">"))
	}
	// Measured: on whole plain text (no selection) TextSoap leaves the text
	// unchanged for these three tag cleaners. We keep that behaviour.
	reg("#HTML_TAG_BOLD", "HTML 标签：粗体（实测：整段纯文本不变）", func(s string) string { return s })
	reg("#BBCODE_TAG_BOLD", "BBCode：粗体（实测：整段纯文本不变）", func(s string) string { return s })
	reg("#PLIST_TAG_STRING", "Plist 标签：string（实测：整段纯文本不变）", func(s string) string { return s })
	reg("#HTML_TAG_BREAK", "HTML 标签：换行 <br>", func(s string) string { return s + "<br>" })
	reg("#HTML_TAG_HORIZONTAL_RULE", "HTML 标签：分隔线 <hr>", func(s string) string { return s + "<hr>" })
	reg("#HTML_TAG_LINK", "HTML 标签：链接", func(s string) string { return `<a href="` + s + `">` + s + `</a>` })
	for _, t := range []struct{ tok, zh, tag string }{{"ARRAY", "array", "array"}, {"DICTIONARY", "dict", "dict"}, {"KEY", "key", "key"},
		{"INTEGER", "integer", "integer"}, {"FLOAT", "real", "real"}} {
		reg("#PLIST_TAG_"+t.tok, "Plist 标签："+t.zh, wrap("<"+t.tag+">", "</"+t.tag+">"))
	}
	reg("#PLIST_TAG_TRUE", "Plist 标签：true", func(s string) string { return s + "<true/>" })
	reg("#PLIST_TAG_FALSE", "Plist 标签：false", func(s string) string { return s + "<false/>" })
	for _, t := range []struct{ tok, zh, tag string }{{"ITALIC", "斜体", "i"}, {"UNDERLINE", "下划线", "u"}, {"STRIKETHROUGH", "删除线", "s"},
		{"CENTERTEXT", "居中", "center"}, {"QUOTE", "引用", "quote"}, {"IMAGE", "图片", "img"}, {"CODE", "代码", "code"}, {"UL", "列表", "list"},
		{"OL", "有序列表", "list=1"}, {"TABLE", "表格", "table"}, {"TABLEROW", "表格行", "tr"}, {"TABLEHEADERCELL", "表头", "th"},
		{"TABLECELL", "单元格", "td"}, {"YOUTUBE", "YouTube", "youtube"}, {"GOOGLEVIDEO", "Google Video", "googlevideo"}} {
		end := strings.SplitN(t.tag, "=", 2)[0]
		reg("#BBCODE_TAG_"+t.tok, "BBCode："+t.zh, wrap("["+t.tag+"]", "[/"+end+"]"))
	}
	reg("#BBCODE_TAG_LI", "BBCode：列表项", func(s string) string { return "[*]" + s })
}

// scrub reproduces the measured Scrub: control chars, MIME, forwarding,
// quotes, then paragraphs and spaces; runs of 4+ returns collapse to one.
func scrub(s string) string {
	if s == "\n" {
		return s
	}
	s = removeControl(s)
	s = convertMIME(s)
	s = removeForwardingRaw(s)
	s = straightenQuotes(s)
	s = makeParagraphs(s)
	s = removeExtraSpaces(s)
	s = stripRuns(s, "\n", 4)
	return s
}
