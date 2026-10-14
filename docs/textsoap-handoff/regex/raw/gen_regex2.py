#!/usr/bin/env python3
"""Focused follow-up: the options-string format, case-off behaviour, lazy
quantifiers, and replacement edge cases."""
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


# (label, input, pattern, replace, options)
CASES = [
    # options-format probing on a multiline anchor
    ("opt-empty-nl",     "a\nb\na", r"^a", "X", ""),
    ("opt-m",            "a\nb\na", r"^a", "X", "m"),
    ("opt-i,m",          "a\nb\na", r"^a", "X", "i,m"),
    ("opt-comma-m",      "a\nb\na", r"^a", "X", "m,"),
    ("opt-ICU",          "a\nb\na", r"^a", "X", "RegexOptions: m"),
    # dotall probing
    ("dot-empty",        "a\nb", r"a.b", "Z", ""),
    ("dot-s",            "a\nb", r"a.b", "Z", "s"),
    ("dot-i,s",          "a\nb", r"a.b", "Z", "i,s"),
    # case-sensitivity: is ignorecase on by default at this layer?
    ("case-upper-pat",   "abc", r"ABC", "Y", ""),
    ("case-lower-pat",   "ABC", r"abc", "Y", ""),
    # $l / $E with uppercase source
    ("l-nextchar",       "ABC", r"(A)(BC)", "$1$l$2", ""),
    ("E-restore",        "abc", r"(abc)", "$U$1$E def", ""),
    ("u-nextchar",       "abc", r"(a)(bc)", "$u$1$2", ""),
    # lazy vs greedy
    ("greedy",           "<a><b>", r"<.+>", "G", ""),
    ("lazy",             "<a><b>", r"<.+?>", "L", ""),
    # group ref ambiguity
    ("grp-1-only",       "abcdef", r"(a)(b)", "$1", ""),
    ("grp-10",           "abcdef", r"(a)(b)(c)(d)(e)(f)(g)(h)(i)(j)", "$10", ""),
    ("grp-braced",       "abcdef", r"(a)(b)", "${1}", ""),
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
