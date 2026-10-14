#!/usr/bin/env python3
"""Reference implementation of the built-in cleaners, judged by the live app.

This is the *candidate* side of the differential oracle. It is written from
TextSoap's own shipped documentation (app help book + the built-in library
`desc` fields), then corrected against `oracle_truth.json` where the
documentation was ambiguous. Every cleaner here is meant to be observationally
equivalent to TextSoap for the inputs in `corpus.json`; it is not a claim of
implementation identity.

`specs/judge.py` runs this against the captured truth and reports per-cleaner
agreement. Cleaners not listed in IMPLEMENTED raise NotImplementedError and are
reported as "not implemented" rather than silently passing.

Terminology: TextSoap runs on an NSMutableString; "\n" is what the app's `string`
accessor returns for a hard return.
"""
from __future__ import annotations

import re
import unicodedata

# ---------------------------------------------------------------------------
# helpers
# ---------------------------------------------------------------------------

NBSP = "\u00a0"

# Characters TextSoap's case cleaners treat as sentence terminators.
SENTENCE_END = ".!?"

TITLE_SMALL_WORDS = {
    "a", "an", "and", "as", "at", "but", "by", "en", "for",
    "if", "in", "of", "on", "or", "the", "to", "via",
}

# Words whose non-standard capitalization is preserved by title case
# (the help book names these explicitly).
def _has_internal_cap(word: str) -> bool:
    """iTunes / iPhone / TextSoap style: a capital after the first letter."""
    return any(c.isupper() for c in word[1:])


def _split_words(text: str):
    """Yield (separator, word) pairs, where separator is the whitespace run."""
    return re.findall(r"(\s*)(\S*)", text)


# ---------------------------------------------------------------------------
# whitespace cleaners
# ---------------------------------------------------------------------------

def remove_extra_spaces(text: str) -> str:
    """NBSP -> space, collapse space runs, strip leading spaces per line.

    Help book: "Converts non-breaking spaces to standard spaces, then removes two
    or more spaces with a single space. Leading spaces at the beginning of a line
    are also removed, as are trailing spaces."
    Observed: a trailing run collapses to ONE space, it is not deleted
    ('   lead and trail   ' -> 'lead and trail '), so "trailing spaces" in the
    docs means trailing *extra* spaces.
    """
    text = text.replace(NBSP, " ")
    text = re.sub(r" +", " ", text)
    text = re.sub(r"(?m)^ +", "", text)
    return text


def trim_lines(text: str) -> str:
    """Strip leading and trailing whitespace on every line."""
    return "\n".join(line.strip() for line in text.split("\n"))


def remove_all_spaces(text: str) -> str:
    """Delete 0x20 only; tabs and everything else are untouched."""
    return text.replace(" ", "")


def remove_all_whitespace(text: str) -> str:
    """Delete spaces and tabs."""
    return text.replace(" ", "").replace("\t", "")


def remove_all_tabs(text: str) -> str:
    return text.replace("\t", "")


def expand_tabs(text: str) -> str:
    """Help book: "Converts tabs (hex 0x09) to four (4) spaces"."""
    return text.replace("\t", "    ")


def remove_extra_returns(text: str) -> str:
    """Runs of 2+ returns collapse to exactly 2; a lone return is removed."""
    text = re.sub(r"\n{2,}", "\n\n", text)
    if text == "\n":
        return ""
    return text


def make_paragraphs(text: str) -> str:
    """Single returns become a space; runs of 2+ are paragraph breaks.

    Trailing whitespace before a return is dropped first ('a   \\nb' -> 'a b'),
    but whitespace at the very end of the text is kept.
    """
    text = re.sub(r"[ \t]+(?=\n)", "", text)
    text = re.sub(r"(?<!\n)\n(?!\n)", " ", text)
    return text


def remove_control_characters(text: str) -> str:
    """Delete C0 controls except LF and CR."""
    return "".join(c for c in text if ord(c) >= 32 or c in "\n\r")


def convert_non_breaking_spaces(text: str) -> str:
    return text.replace(NBSP, " ")


# ---------------------------------------------------------------------------
# case cleaners
# ---------------------------------------------------------------------------

def uppercase(text: str) -> str:
    return text.upper()


def lowercase(text: str) -> str:
    return text.lower()


def capitalize_sentences(text: str) -> str:
    """Uppercase the first letter of each sentence; leave the rest as-is.

    (The lowercasing variant is a separate cleaner:
    "Capitalize Sentences with Lowercasing".)
    """
    out = []
    start = True
    for ch in text:
        if start and ch.isalpha():
            out.append(ch.upper())
            start = False
        else:
            out.append(ch)
            if ch in SENTENCE_END:
                start = True
            elif not ch.isspace() and ch not in "\"'(\u201c\u2018":
                start = False
    return "".join(out)


def capitalize_words(text: str) -> str:
    """Lowercase everything, then capitalize each word."""
    text = text.lower()
    return re.sub(r"\b(\w)", lambda m: m.group(1).upper(), text)


def capitalize_title_case(text: str) -> str:
    """Title case: small words lowercase unless first/last in the line,
    non-standard capitalization preserved (iTunes, iPhone, TextSoap), and
    whitespace runs normalised to single spaces (observed: tab and NBSP
    boundaries become spaces).
    """
    lines_out = []
    for line in text.split("\n"):
        words = [w for w in re.split(r"\s+", line) if w]
        small_run = []
        result = []
        for i, w in enumerate(words):
            first, last = (i == 0), (i == len(words) - 1)
            if _has_internal_cap(w):
                result.append(w)                       # iTunes, iPhone, macOS
            elif w.lower() in TITLE_SMALL_WORDS and not first and not last:
                result.append(w.lower())
            else:
                result.append(w[:1].upper() + w[1:].lower())
        lines_out.append(" ".join(result))
    return "\n".join(lines_out)


# ---------------------------------------------------------------------------
# typography
# ---------------------------------------------------------------------------

def straighten_quotes(text: str) -> str:
    return (text.replace("\u201c", '"').replace("\u201d", '"')
                .replace("\u2018", "'").replace("\u2019", "'"))


def smarten_quotes(text: str) -> str:
    """Positional: an opening mark follows start/whitespace, else closing."""
    out = []
    prev = " "
    for ch in text:
        if ch == '"':
            out.append("\u201c" if prev.isspace() or prev == "" else "\u201d")
        elif ch == "'":
            out.append("\u2018" if prev.isspace() or prev == "" else "\u2019")
        else:
            out.append(ch)
        prev = ch
    return "".join(out)


def internet_friendly_text(text: str) -> str:
    """Strip diacritics, straighten quotes, map common symbols to ASCII."""
    text = unicodedata.normalize("NFD", text)
    text = "".join(c for c in text if unicodedata.category(c) != "Mn")
    text = straighten_quotes(text)
    for src, dst in [("\u00a9", "(C)"), ("\u00ae", "(R)"), ("\u2122", "(TM)"),
                     ("\u2026", "..."), ("\u2014", "--"), ("\u2013", "-")]:
        text = text.replace(src, dst)
    return text


def remove_high_ascii(text: str) -> str:
    """Strip 128-255, after mapping the symbols the app special-cases."""
    for src, dst in [("\u00a9", "(C)"), ("\u2026", "..."),
                     ("\u2014", "-"), ("\u2013", "-")]:
        text = text.replace(src, dst)
    return "".join(c for c in text if ord(c) < 128)


# ---------------------------------------------------------------------------
# structure / quoting
# ---------------------------------------------------------------------------

def remove_forwarding(text: str) -> str:
    """Strip one or more leading '>' marks and the spaces after them."""
    return "\n".join(re.sub(r"^(?:[ \t]*>)+[ \t]*", "", line)
                     for line in text.split("\n"))


def convert_bullets(text: str) -> str:
    return text.replace("\u2022", "-")


def convert_mime(text: str) -> str:
    """%XX and =XX are both decoded (help book names both forms)."""
    def sub(m):
        try:
            return chr(int(m.group(1), 16))
        except ValueError:
            return m.group(0)
    return re.sub(r"[%]([0-9A-Fa-f]{2})", sub,
                  re.sub(r"[=]([0-9A-Fa-f]{2})", sub, text))


def remove_duplicate_lines(text: str) -> str:
    """Drop repeated lines, keeping the first occurrence."""
    seen, out = set(), []
    for line in text.split("\n"):
        if line not in seen:
            seen.add(line)
            out.append(line)
    return "\n".join(out)


def wrap_at(text: str, width: int) -> str:
    """Make Paragraphs, then greedy-wrap each paragraph at `width`."""
    text = make_paragraphs(text)
    out = []
    for para in text.split("\n\n"):
        words, line, buf = para.split(), [], []
        cur = 0
        for w in words:
            add = len(w) if not buf else len(w) + 1
            if buf and cur + add > width:
                out.append(" ".join(buf)); buf = [w]; cur = len(w)
            else:
                buf.append(w); cur += add
        out.append(" ".join(buf))
        out.append("")
    if out and out[-1] == "":
        out.pop()
    return "\n\n".join(out)


def rewrap_text(text: str) -> str:
    return wrap_at(text, 70)


# ---------------------------------------------------------------------------
# registry
# ---------------------------------------------------------------------------

IMPLEMENTED = {
    "Remove Extra Spaces": remove_extra_spaces,
    "Trim Lines": trim_lines,
    "Remove All Spaces": remove_all_spaces,
    "Remove All Whitespace": remove_all_whitespace,
    "Remove All Tabs": remove_all_tabs,
    "Expand Tabs": expand_tabs,
    "Remove Extra Returns": remove_extra_returns,
    "Make Paragraphs": make_paragraphs,
    "Remove Control Characters": remove_control_characters,
    "Convert Non-Breaking Spaces": convert_non_breaking_spaces,
    "Uppercase": uppercase,
    "Lowercase": lowercase,
    "Capitalize Sentences": capitalize_sentences,
    "Capitalize Words": capitalize_words,
    "Capitalize with Title Case": capitalize_title_case,
    "Straighten Quotes": straighten_quotes,
    "Smarten Quotes": smarten_quotes,
    "Internet Friendly Text": internet_friendly_text,
    "Remove All High Ascii Characters": remove_high_ascii,
    "Remove Forwarding (>) Characters": remove_forwarding,
    "Convert Bullets to ASCII": convert_bullets,
    "Convert MIME-encoded Characters": convert_mime,
    "Remove Duplicate Lines": remove_duplicate_lines,
    "Rewrap Text": rewrap_text,
    "Wrap At 50": lambda t: wrap_at(t, 50),
    "Wrap At 60": lambda t: wrap_at(t, 60),
    "Wrap At 70": lambda t: wrap_at(t, 70),
}


def clean(name: str, text: str) -> str:
    if name not in IMPLEMENTED:
        raise NotImplementedError(f"cleaner not implemented: {name}")
    return IMPLEMENTED[name](text)


if __name__ == "__main__":
    import sys
    print(clean(sys.argv[1], sys.argv[2]))
