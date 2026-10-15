#!/usr/bin/env python3
"""Validate the regex find/replace engine semantics: ICU flags and the
$U/$u/$L/$l/$E case-change replacement commands."""
import sys


def lit(s):
    out = []
    for ch in s:
        o = ord(ch)
        if ch == '"': out.append('\\"')
        elif ch == '\\': out.append('\\\\')
        elif ch == '\n': out.append('\\n')
        elif ch == '\t': out.append('\\t')
        elif ch == '\r': out.append('\\r')
        elif o < 32 or o > 126: out.append('\\u%04x' % o)
        else: out.append(ch)
    return ''.join(out)


BASE = "The big brown fox jumped over the blue cow"
MULTI = "a\nb\na\nb"

# (label, input, pattern, replace, options)
CASES = [
    ("doc-example",      BASE, r"(b\w+)", "$u$1", "i"),
    ("upper-all",        BASE, r"(b\w+)", "$U$1", "i"),
    ("upper-then-E",     BASE, r"(b\w+)", "$U$1$E-tail", "i"),
    ("lower-all",        BASE, r"(B\w+)", "$L$1", ""),
    ("nextchar-lower",   BASE, r"(b)(\w+)", "$1$l$2", ""),
    ("group0",           BASE, r"\w+", "[$0]", ""),
    ("group1-only",      BASE, r"(\w+) (\w+)", "$2 $1", ""),
    ("no-matchcase-i",   "ABC abc", r"abc", "X", "i"),
    ("matchcase-off",    "ABC abc", r"abc", "X", ""),
    ("multiline-m",      "a\nb\na", r"^a", "X", "m"),
    ("multiline-off",    "a\nb\na", r"^a", "X", ""),
    ("dotall-s",         "a\nb", r"a.b", "Z", "s"),
    ("dotall-off",       "a\nb", r"a.b", "Z", ""),
    ("comma-flags",      "a\nb", r"^a.b", "Z", "s,m"),
    ("backslash-escape", BASE, r"(b\w+)", "\\$1", "i"),
    ("literal-dollar",   BASE, r"(b\w+)", "$$1", "i"),
    ("hex-escape-find",  "a\tb", r"a\x{09}b", "TAB", ""),
    ("unicode-prop",     "a1b2", r"\p{Nd}", "N", ""),
    ("word-boundary",    BASE, r"\bb\w+", "W", ""),
]


def main():
    L = ['expr -l objc -- id $t = (id)[(id)objc_getClass("TSText") performSelector:(SEL)sel_registerName("new")]']
    for label, inp, pat, rep, opt in CASES:
        L.append('expr -l objc -O -- @"@@@ %s"' % lit(label))
        L.append('expr -l objc -O -- (void)[$t performSelector:(SEL)sel_registerName("setString:") withObject:@"%s"]' % lit(inp))
        L.append('expr -l objc -O -- (long)((long(*)(id,SEL,id,id,id))objc_msgSend)((id)$t, (SEL)sel_registerName("regexFind:replace:options:"), @"%s", @"%s", @"%s")' % (lit(pat), lit(rep), lit(opt)))
        L.append('expr -l objc -O -- (id)[(id)[$t performSelector:(SEL)sel_registerName("string")] performSelector:(SEL)sel_registerName("stringByReplacingOccurrencesOfString:withString:") withObject:@"\\n" withObject:@"<LF>"]')
    L += ['detach', 'quit']
    open(sys.argv[1], 'w').write('\n'.join(L) + '\n')
    print("cases:", len(CASES))


main()
