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
    ("tok-none",      "ABC abc", r"abc", "Y", ""),
    ("tok-matchcase", "ABC abc", r"abc", "Y", "matchcase"),
    ("tok-ignorecase","ABC abc", r"abc", "Y", "ignorecase"),
    ("tok-both",      "ABC abc", r"abc", "Y", "matchcase,ignorecase"),
    ("tok-matchlines","a\nb\na", r"^a", "X", "lm"),
    ("tok-linesoff",  "a\nb\na", r"^a", "X", ""),
    ("tok-dotall",    "a\nb", r"a.b", "Z", "dotall"),
    ("tok-dotall2",   "a\nb", r"a.b", "Z", "s"),
    ("tok-words",     "a-b", r"\ba", "W", "unicodewords"),
    ("tok-comments",  "ab", r"a #note\n b", "C", "comments"),
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
