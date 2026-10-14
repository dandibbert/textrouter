#!/usr/bin/env python3
"""Run a cleaner x input matrix through the live TSText engine via lldb.

INPUTS hold REAL control characters in Python; they are re-encoded as ObjC
escape sequences (\\n, \\t, \\uXXXX) when written into the lldb command file.
"""
import sys

INPUTS = [
    ("extra-spaces",   "a    b"),
    ("forwarding",     "> quoted text"),
    ("broken-para",    "line one\nline two"),
    ("multi-return",   "a\n\n\n\nb"),
    ("tabs",           "a\t\tb"),
    ("mime",           "a%20b%2Cc"),
    ("trim",           "   lead and trail   "),
    ("smart-quotes",   "\u201chi\u201d and \u2018x\u2019"),
    ("ctrl-char",      "a\u0001b"),
    ("nbsp",           "a\u00a0b"),
    ("two-paras",      "first paragraph here\n\nsecond paragraph here"),
    ("wrap-long",      "The quick brown fox jumps over the lazy dog and keeps on running past the fence and into the woods today."),
]

CLEANERS = [
    "Scrub",
    "Remove Extra Spaces",
    "Remove Forwarding (>) Characters",
    "Convert MIME-encoded Characters",
    "Make Paragraphs",
    "Remove Extra Returns",
    "Remove All Tabs",
    "Straighten Quotes",
    "Remove Whitespace Before Paragraphs",
    "Remove Control Characters",
    "Convert Non-Breaking Spaces",
    "Trim Lines",
    "Uppercase",
    "Rewrap Text",
]


def objc_literal(s: str) -> str:
    """Encode a Python string as the contents of an ObjC @"..." literal."""
    out = []
    for ch in s:
        o = ord(ch)
        if ch == '"':
            out.append('\\"')
        elif ch == '\\':
            out.append('\\\\')
        elif ch == '\n':
            out.append('\\n')
        elif ch == '\t':
            out.append('\\t')
        elif ch == '\r':
            out.append('\\r')
        elif o < 32 or o > 126:
            out.append('\\u%04x' % o)
        else:
            out.append(ch)
    return ''.join(out)


def main():
    pairs = [(c, n, v) for c in CLEANERS for n, v in INPUTS]
    lines = ['expr -l objc -- id $t = (id)[(id)objc_getClass("TSText") performSelector:(SEL)sel_registerName("new")]']
    for cname, iname, ival in pairs:
        lines.append('expr -l objc -O -- @"@@@ %s || %s"' % (objc_literal(cname), objc_literal(iname)))
        lines.append('expr -l objc -O -- (void)[$t performSelector:(SEL)sel_registerName("setString:") withObject:@"%s"]' % objc_literal(ival))
        lines.append('expr -l objc -O -- (void)[$t performSelector:(SEL)sel_registerName("applyCleaner:") withObject:@"%s"]' % objc_literal(cname))
        lines.append('expr -l objc -O -- (id)[(id)[$t performSelector:(SEL)sel_registerName("string")] performSelector:(SEL)sel_registerName("stringByReplacingOccurrencesOfString:withString:") withObject:@"\\n" withObject:@"<LF>"]')
    lines += ['detach', 'quit']
    open(sys.argv[1], 'w').write('\n'.join(lines) + '\n')
    print("pairs:", len(pairs))


if __name__ == "__main__":
    main()
