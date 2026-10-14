package engine

import "testing"

func TestICUSyntax(t *testing.T) {
	cases := []struct{ find, in, want string }{
		{`\p{Han}+`, "ab中文c", "ab<中文>c"},
		{`\p{Script=Han}+`, "ab中文c", "ab<中文>c"},
		{`\p{sc=Hiragana}+`, "あいう漢", "<あいう>漢"},
		{`\pL+`, "ab1", "<ab>1"},
		{`[[:alpha:]]+`, "abc 12", "<abc> 12"},
		{`[[:digit:][:space:]]+`, "ab1 2c", "ab<1 2>c"},
		{`\p{Alphabetic}+`, "ab1", "<ab>1"},
		{`\d++`, "123a", "<123>a"},
		{`a*+a`, "aaa", "aaa"},
		{`(?:ab)++`, "ababx", "<abab>x"},
		{`\h+`, "a \t\u3000b", "a<" + " \t\u3000" + ">b"},
		{`\R`, "a\r\nb\nc", "a<\r\n>b<\n>c"},
		{`a.b`, "a\rb a-b", "a\rb <a-b>"},
		{`\N{DIGIT ONE}`, "x1", "x<1>"},
		{`\Qa.b\E+`, "a.bb a.b", "<a.bb> <a.b>"},
		{`(?w)\bfoo\b`, "foo food", "<foo> food"},
		{`(?iw)FOO`, "foo", "<foo>"},
		{`\x{4e2d}+`, "中中x", "<中中>x"},
		{`[\x{4e00}-\x{9fa5}]+`, "ab中文", "ab<中文>"},
		{`(?<=元)\d`, "元5", "元<5>"},
		{`(?<y>\d{4})-\k<y>`, "2024-2024", "<2024-2024>"},
		{`[a-c[x-z]]+`, "abxyq", "<abxy>q"},
		{`\x{41}+`, "AAAb", "<AAA>b"},
	}
	for _, c := range cases {
		o := FindOptions{Find: c.find, Replace: "<$0>", Regex: true, MatchCase: true}
		got, _, err := o.Apply(c.in)
		if err != nil {
			t.Errorf("%q: %v", c.find, err)
			continue
		}
		if got != c.want {
			t.Errorf("%q on %q = %q, want %q", c.find, c.in, got, c.want)
		}
	}
}

func TestTextEscapesAndGranularity(t *testing.T) {
	cases := []struct {
		o        FindOptions
		in, want string
	}{
		{FindOptions{Find: `\t`, Replace: `\s`, MatchCase: true}, "a\tb", "a b"},
		{FindOptions{Find: `a\nb`, Replace: `\x{4e2d}`, MatchCase: true}, "a\nb", "中"},
		{FindOptions{Find: `\\`, Replace: `/`, MatchCase: true}, `a\b`, "a/b"},
		{FindOptions{Find: `.`, Replace: `$1`, MatchCase: true}, "a.b", "a$1b"},
		{FindOptions{Find: `cat`, Replace: `X`, MatchCase: true, Granularity: GranWholeWord}, "cat cats bobcat", "X cats bobcat"},
		{FindOptions{Find: `cat`, Replace: `X`, MatchCase: true, Granularity: GranStartsWith}, "cat cats bobcat", "X Xs bobcat"},
		{FindOptions{Find: `cat`, Replace: `X`, MatchCase: true, Granularity: GranEndsWith}, "cat cats bobcat", "X cats bobX"},
		{FindOptions{Find: `(\w+)`, Replace: `$1\t`, Regex: true, MatchCase: true}, "ab", "ab\t"},
		{FindOptions{Find: `^a`, Replace: `X`, Regex: true, MatchCase: true, Granularity: GranWholeWord, RegexOptions: "m"}, "a\nab", "X\nXb"},
		{FindOptions{Find: `a # letter a`, Replace: `X`, Regex: true, MatchCase: true, RegexOptions: "x"}, "ba", "bX"},
	}
	for _, c := range cases {
		got, _, err := c.o.Apply(c.in)
		if err != nil || got != c.want {
			t.Errorf("%+v on %q = %q (%v), want %q", c.o, c.in, got, err, c.want)
		}
	}
}
