package engine

import (
	"fmt"
	"strings"
)

// Part is one piece of a pattern or template with a plain-language
// (Chinese) explanation.
type Part struct {
	Text string
	Desc string
	Kind string // "literal", "class", "group", "quant", "anchor", "alt", "token", "error"
}

var escDesc = map[rune]string{
	'd': "任意一个数字 0-9", 'D': "任意一个非数字字符", 'w': "一个字母、数字或下划线（含中文）", 'W': "一个非“单词”字符",
	's': "一个空白（空格、制表符、换行）", 'S': "一个非空白字符", 'b': "单词边界（不占字符）", 'B': "非单词边界",
	'n': "换行符", 't': "制表符", 'r': "回车符", 'A': "整段文字的开头", 'z': "整段文字的结尾", 'Z': "结尾（允许最后一个换行）",
}

// ExplainPattern breaks a regex into parts with explanations. It is a
// best-effort teaching aid, not a full parser.
func ExplainPattern(p string, regex bool) []Part {
	if !regex {
		if p == "" {
			return nil
		}
		return []Part{{Text: p, Desc: "按字面查找这段文字（不是正则，. * ? 等符号都只代表自己）", Kind: "literal"}}
	}
	r := []rune(p)
	var out []Part
	var lit []rune
	flush := func() {
		if len(lit) > 0 {
			out = append(out, Part{Text: string(lit), Desc: "字面文字「" + string(lit) + "」", Kind: "literal"})
			lit = nil
		}
	}
	depth := 0
	for i := 0; i < len(r); i++ {
		c := r[i]
		switch c {
		case '\\':
			if i+1 >= len(r) {
				flush()
				out = append(out, Part{Text: "\\", Desc: "结尾多了一个反斜杠", Kind: "error"})
				continue
			}
			n := r[i+1]
			if d, ok := escDesc[n]; ok {
				flush()
				out = append(out, Part{Text: string(r[i : i+2]), Desc: d, Kind: "class"})
				i++
				continue
			}
			if (n == 'p' || n == 'P') && i+2 < len(r) && r[i+2] == '{' {
				end := indexFrom(r, '}', i+2)
				if end > 0 {
					flush()
					name := string(r[i+3 : end])
					d := "属于 Unicode 类别 " + name + " 的字符"
					switch name {
					case "Han", "IsCJKUnifiedIdeographs":
						d = "一个汉字"
					case "L":
						d = "任意一个字母（含中文）"
					case "N":
						d = "任意一个数字字符"
					case "P":
						d = "一个标点符号"
					}
					if n == 'P' {
						d = "不是：" + d
					}
					out = append(out, Part{Text: string(r[i : end+1]), Desc: d, Kind: "class"})
					i = end
					continue
				}
			}
			if n >= '1' && n <= '9' {
				flush()
				out = append(out, Part{Text: string(r[i : i+2]), Desc: fmt.Sprintf("再次出现第 %c 组匹配到的同样内容", n), Kind: "token"})
				i++
				continue
			}
			lit = append(lit, n)
			i++
		case '.':
			flush()
			out = append(out, Part{Text: ".", Desc: "任意一个字符（默认不含换行；开启“. 匹配换行”后包含）", Kind: "class"})
		case '^':
			flush()
			out = append(out, Part{Text: "^", Desc: "开头（开启“多行”时为每行开头，否则只是整段开头）", Kind: "anchor"})
		case '$':
			flush()
			out = append(out, Part{Text: "$", Desc: "结尾（开启“多行”时为每行结尾，否则只是整段结尾）", Kind: "anchor"})
		case '|':
			flush()
			out = append(out, Part{Text: "|", Desc: "或者：左边或右边任一种都算匹配", Kind: "alt"})
		case '[':
			flush()
			end := classEnd(r, i)
			if end < 0 {
				out = append(out, Part{Text: string(r[i:]), Desc: "方括号没有闭合", Kind: "error"})
				i = len(r)
				continue
			}
			body := string(r[i+1 : end])
			d := "下列字符中的任意一个：" + body
			if strings.HasPrefix(body, "^") {
				d = "除了这些以外的任意一个字符：" + body[1:]
			}
			out = append(out, Part{Text: string(r[i : end+1]), Desc: d, Kind: "class"})
			i = end
		case '(':
			flush()
			depth++
			rest := string(r[i:])
			switch {
			case strings.HasPrefix(rest, "(?:"):
				out = append(out, Part{Text: "(?:", Desc: "分组开始（不编号，仅用于组合）", Kind: "group"})
				i += 2
			case strings.HasPrefix(rest, "(?="):
				out = append(out, Part{Text: "(?=", Desc: "向后看：后面必须紧跟着…（不占字符）", Kind: "group"})
				i += 2
			case strings.HasPrefix(rest, "(?!"):
				out = append(out, Part{Text: "(?!", Desc: "向后看：后面不能是…", Kind: "group"})
				i += 2
			case strings.HasPrefix(rest, "(?<="):
				out = append(out, Part{Text: "(?<=", Desc: "向前看：前面必须是…", Kind: "group"})
				i += 3
			case strings.HasPrefix(rest, "(?<!"):
				out = append(out, Part{Text: "(?<!", Desc: "向前看：前面不能是…", Kind: "group"})
				i += 3
			case strings.HasPrefix(rest, "(?<") || strings.HasPrefix(rest, "(?P<"):
				end := indexFrom(r, '>', i)
				out = append(out, Part{Text: string(r[i : end+1]), Desc: "命名分组开始", Kind: "group"})
				i = end
			case strings.HasPrefix(rest, "(?"):
				end := indexFrom(r, ')', i)
				if end < 0 {
					end = len(r) - 1
				}
				flags := string(r[i+2 : end])
				d := "内联选项 " + flags
				if strings.Contains(flags, "i") {
					d = "从这里起忽略大小写"
				}
				if strings.Contains(flags, "m") {
					d += "；^ $ 按行匹配"
				}
				if strings.Contains(flags, "s") {
					d += "；. 也匹配换行"
				}
				out = append(out, Part{Text: string(r[i : end+1]), Desc: d, Kind: "group"})
				depth--
				i = end
			default:
				n := countGroupsBefore(r, i) + 1
				out = append(out, Part{Text: "(", Desc: fmt.Sprintf("第 %d 组开始（替换里用 $%d 引用）", n, n), Kind: "group"})
			}
		case ')':
			flush()
			depth--
			out = append(out, Part{Text: ")", Desc: "分组结束", Kind: "group"})
		case '*', '+', '?', '{':
			flush()
			q, d := quantifier(r, i)
			if q == "" {
				lit = append(lit, c)
				continue
			}
			out = append(out, Part{Text: q, Desc: d, Kind: "quant"})
			i += len([]rune(q)) - 1
		default:
			lit = append(lit, c)
		}
	}
	flush()
	if depth != 0 {
		out = append(out, Part{Text: "", Desc: "括号数量不匹配", Kind: "error"})
	}
	return out
}

func indexFrom(r []rune, c rune, from int) int {
	for i := from; i < len(r); i++ {
		if r[i] == c {
			return i
		}
	}
	return -1
}

func classEnd(r []rune, start int) int {
	i := start + 1
	if i < len(r) && r[i] == '^' {
		i++
	}
	if i < len(r) && r[i] == ']' {
		i++
	}
	for ; i < len(r); i++ {
		if r[i] == '\\' {
			i++
			continue
		}
		if r[i] == ']' {
			return i
		}
	}
	return -1
}

func countGroupsBefore(r []rune, pos int) int {
	n := 0
	inClass := false
	for i := 0; i < pos; i++ {
		switch {
		case r[i] == '\\':
			i++
		case r[i] == '[':
			inClass = true
		case r[i] == ']':
			inClass = false
		case r[i] == '(' && !inClass && !(i+1 < len(r) && r[i+1] == '?'):
			n++
		}
	}
	return n
}

func quantifier(r []rune, i int) (string, string) {
	var q, d string
	switch r[i] {
	case '*':
		q, d = "*", "前一项重复 0 次或多次"
	case '+':
		q, d = "+", "前一项重复 1 次或多次"
	case '?':
		q, d = "?", "前一项可有可无"
	case '{':
		end := indexFrom(r, '}', i)
		if end < 0 {
			return "", ""
		}
		body := string(r[i+1 : end])
		a, b, comma := strings.Cut(body, ",")
		switch {
		case !comma:
			d = "前一项正好重复 " + a + " 次"
		case b == "":
			d = "前一项至少重复 " + a + " 次"
		default:
			d = "前一项重复 " + a + " 到 " + b + " 次"
		}
		for _, c := range strings.ReplaceAll(body, ",", "") {
			if c < '0' || c > '9' {
				return "", ""
			}
		}
		q = string(r[i : end+1])
		i = end
	}
	if i+1 < len(r) && r[i+1] == '?' {
		q += "?"
		d += "（尽量少）"
	} else if q != "?" {
		d += "（尽量多）"
	}
	return q, d
}

// ExplainTemplate explains a replacement string token by token, using the
// measured TextSoap rules.
func ExplainTemplate(s string, regex bool) []Part {
	if !regex {
		if s == "" {
			return []Part{{Text: "", Desc: "替换为空 = 删除匹配内容", Kind: "literal"}}
		}
		return []Part{{Text: s, Desc: "原样替换成这段文字", Kind: "literal"}}
	}
	if s == "" {
		return []Part{{Text: "", Desc: "替换为空 = 删除匹配内容", Kind: "literal"}}
	}
	var out []Part
	for _, t := range ParseTemplate(s) {
		switch t.Kind {
		case TokLiteral:
			out = append(out, Part{Text: t.Text, Desc: "原样插入「" + t.Text + "」", Kind: "literal"})
		case TokGroup:
			d := fmt.Sprintf("插入第 %d 组匹配到的内容", t.Group)
			if t.Group == 0 {
				d = "插入整个匹配到的内容"
			}
			out = append(out, Part{Text: t.Text, Desc: d, Kind: "token"})
		case TokUpperAll:
			out = append(out, Part{Text: t.Text, Desc: "之后的内容全部转大写", Kind: "token"})
		case TokLowerAll:
			out = append(out, Part{Text: t.Text, Desc: "之后的内容全部转小写", Kind: "token"})
		case TokUpperNext:
			out = append(out, Part{Text: t.Text, Desc: "下一个字符转大写", Kind: "token"})
		case TokLowerNext:
			out = append(out, Part{Text: t.Text, Desc: "下一个字符转小写", Kind: "token"})
		case TokEnd:
			out = append(out, Part{Text: t.Text, Desc: "结束大小写转换（本身不插入任何内容）", Kind: "token"})
		case TokDropped:
			d := "“" + t.Text + "”不是有效记号，会被直接丢掉。想要美元符号请写 \\$"
			if t.Text == "$" {
				d = "单独的 $ 会被丢掉。想要美元符号请写 \\$"
			}
			out = append(out, Part{Text: t.Text, Desc: d, Kind: "error"})
		}
	}
	return out
}

// Preset is a ready-made pattern for beginners.
type Preset struct {
	Name, Pattern, Replace, Desc string
}

// Presets lists common patterns (all regex).
func Presets() []Preset {
	return []Preset{
		{"邮箱地址", `[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Za-z]{2,}`, "", "匹配 name@example.com 这类地址"},
		{"网址", `https?://[^\s<>"]+`, "", "以 http:// 或 https:// 开头直到空白"},
		{"整数", `-?\d+`, "", "可带负号的整数"},
		{"小数/数字", `-?\d+(?:\.\d+)?`, "", "整数或小数"},
		{"中国大陆手机号", `1[3-9]\d{9}`, "", "11 位手机号"},
		{"日期 YYYY-MM-DD", `(\d{4})-(\d{1,2})-(\d{1,2})`, "$1年$2月$3日", "把 2026-10-14 改成 2026年10月14日"},
		{"汉字", `\p{Han}+`, "", "连续的汉字"},
		{"中英文之间加空格", `(\p{Han})([A-Za-z0-9])`, "$1 $2", "盘古之白：汉字后紧跟英文/数字时加空格（反方向再加一条规则）"},
		{"英文/数字后接汉字加空格", `([A-Za-z0-9])(\p{Han})`, "$1 $2", "与上一条配合使用"},
		{"连续空格", ` {2,}`, " ", "两个以上空格变成一个"},
		{"行首行尾空白", `^[ \t]+|[ \t]+$`, "", "删除每行开头和结尾的空格/制表符（需开启多行）"},
		{"空行", `^[ \t]*\n`, "", "删除空白行（需开启多行）"},
		{"HTML 标签", `<[^>]+>`, "", "匹配 <b>、</p> 等标签"},
		{"重复的单词", `\b(\w+)\s+\1\b`, "$1", "the the → the"},
		{"IPv4 地址", `\b(?:\d{1,3}\.){3}\d{1,3}\b`, "", "如 192.168.1.1"},
		{"snake_case 转 camelCase", `_([a-z])`, "$u$1", "下划线后的字母转大写并去掉下划线"},
		{"每行首字母大写", `^(\w)`, "$U$1", "需开启多行"},
		{"括号里的内容", `[(（]([^)）]*)[)）]`, "", "中英文括号都可以；$1 是括号内文字"},
		// Added for the clipboard workspace's pattern menu; keep these last.
		{"数字", `\d+`, "", "连续的数字"},
		{"价格", `(\d+)元`, "¥$1", "5元 → ¥5；$1 是数字"},
	}
}
