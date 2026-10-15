# Regular expression syntax (vendor reference)

Converted from TextSoap's own help page `regex_syntax.html`. This is the
vendor's description of the regex dialect the app uses.

---

Overview

- Welcome
- Clipboard Workspace
- A Quick Tour
- Cleaner Groups Summary
- What’s New in TextSoap 9

MyLibrary

- Cleaner Editor
- My Cleaner Actions
- Custom Group Editor

Tutorials

- Your First Cleaner
- Removing parameters from a URL
- Using Lists in Your Cleaners
- Matching Typefaces, Attributes

Cleaner Descriptions

- Date-related Cleaners
- HTML Cleaners
- Plist Cleaners
- Other Cleaner Descriptions

Extras

- Preferences
- Batch File Cleaning

Reference

- Glossary
- Regular Expression Syntax
- Files and Folders

Regular Expression Syntax

TextSoap uses the ICU (International Components for Unicode) Regular Expression syntax with some custom extensions (for case changing).

For your convenience, the regular expression syntax from the ICU documentation is included below. When in doubt, you should refer to the official ICU User Guide - Regular Expressions documentation page.

See Also

ICU User Guide - Regular Expressions

Unicode Technical Standard #18 - Unicode Regular Expressions

Metacharacters

|

| Character
| Description

| \a
| Match a BELL, \u0007

| \A
| Match at the beginning of the input. Differs from ^ in that \A will not match after a new-line within the input.

| \b, outside of a [Set]
| Match if the current position is a word boundary. Boundaries occur at the transitions between word \w and non-word \W characters, with combining marks ignored.
See also: Overview of Text Boundary Analysis

| \b, within a [Set]
| Match a BACKSPACE, \u0008.

| \B
| Match if the current position is not a word boundary.

| \cx
| Match a Control-x character.

| \d
| Match any character with the Unicode General Category of Nd (Number, Decimal Digit).

| \D
| Match any character that is not a decimal digit.

| \e
| Match an ESCAPE, \u001B.

| \E
| Terminates a \Q…\E quoted sequence.

| \f
| Match a FORM FEED, \u000C.

| \G
| Match if the current position is at the end of the previous match.

| \n
| Match a LINE FEED, \u000A.

| \N{Unicode Character Name}
| Match the named Unicode Character.

| \p{Unicode Property Name}
| Match any character with the specified Unicode Property.

| \P{Unicode Property Name}
| Match any character not having the specified Unicode Property.

| \Q
| Quotes all following characters until \E.

| \r
| Match a CARRIAGE RETURN, \u000D.

| \s
| Match a white space character. White space is defined as [\t\n\f\r\p{Z}].

| \S
| Match a non-white space character.

| \t
| Match a HORIZONTAL TABULATION, \u0009.

| \uhhhh
| Match the character with the hex value hhhh.

| \Uhhhhhhhh
| Match the character with the hex value hhhhhhhh. Exactly eight hex digits must be provided, even though the largest Unicode code point is \U0010ffff.

| \w
| Match a word character. Word characters are [\p{Ll}\p{Lu}\p{Lt}\p{Lo}\p{Nd}].

| \W
| Match a non-word character.

| \x{h…}
| Match the character with hex value hhhh. From one to six hex digits may be supplied.

| \xhh
| Match the character with two digit hex value hh.

| \X
| Match a Grapheme Cluster.

| \Z
| Match if the current position is at the end of input, but before the final line terminator, if one exists.

| \z
| Match if the current position is at the end of input.

| \n
| Back Reference. Match whatever the nth capturing group matched. n must be a number ≥ 1 and ≤ total number of capture groups in the pattern. Note: Octal escapes, such as \012, are not supported.

| [pattern]
| Match any one character from the set. See ICU Regular Expression Character Classes for a full description of what may appear in the pattern.

| .
| Match any character.

| ^
| Match at the beginning of a line.

| $
| Match at the end of a line.

| \
| Quotes the following character. Characters that must be quoted to be treated as literals are: * ? + [ ( ) { } ^ $ | \ . /

Operators

|

| Operator
| Description

| |
| Alternation. A | B matches either A or B.

| *
| Match zero or more times. Match as many times as possible.

| +
| Match one or more times. Match as many times as possible.

| ?
| Match zero or one times. Prefer one.

| {n}
| Match exactly n times.

| {n,}
| Match at least n times. Match as many times as possible.

| {n,m}
| Match between n and m times. Match as many times as possible, but not more than m.

| *?
| Match zero or more times. Match as few times as possible.

| +?
| Match one or more times. Match as few times as possible.

| ??
| Match zero or one times. Prefer zero.

| {n}?
| Match exactly n times.

| {n,}?
| Match at least n times, but no more than required for an overall pattern match.

| {n,m}?
| Match between n and m times. Match as few times as possible, but not less than n.

| *+
| Match zero or more times. Match as many times as possible when first encountered, do not retry with fewer even if overall match fails. Possessive match.

| ++
| Match one or more times. Possessive match.

| ?+
| Match zero or one times. Possessive match.

| {n}+
| Match exactly n times. Possessive match.

| {n,}+
| Match at least n times. Possessive match.

| {n,m}+
| Match between n and m times. Possessive match.

| (…)
| Capturing parentheses. Range of input that matched the parenthesized subexpression is available after the match.

| (?:…)
| Non-capturing parentheses. Groups the included pattern, but does not provide capturing of matching text. Somewhat more efficient than capturing parentheses.

| (?>…)
| Atomic-match parentheses. First match of the parenthesized subexpression is the only one tried; if it does not lead to an overall pattern match, back up the search for a match to a position before the (?> .

| (?#…)
| Free-format comment (?#comment).

| (?=…)
| Look-ahead assertion. True if the parenthesized pattern matches at the current input position, but does not advance the input position.

| (?!…)
| Negative look-ahead assertion. True if the parenthesized pattern does not match at the current input position. Does not advance the input position.

| (?<=…)
| Look-behind assertion. True if the parenthesized pattern matches text preceding the current input position, with the last character of the match being the input character just before the current position. Does not alter the input position. The length of possible strings matched by the look-behind pattern must not be unbounded (no * or + operators).

| (?<!…)
| Negative Look-behind assertion. True if the parenthesized pattern does not match text preceding the current input position, with the last character of the match being the input character just before the current position. Does not alter the input position. The length of possible strings matched by the look-behind pattern must not be unbounded (no * or + operators).

| (?ismwx-ismwx:…)
| Flag settings. Evaluate the parenthesized expression with the specified flags enabled or -disabled.

| (?ismwx-ismwx)
| Flag settings. Change the flag settings. Changes apply to the portion of the pattern following the setting. For example, (?i) changes to a case insensitive match. See also: Regular Expression Options

See Also

ICU User Guide - Regular Expressions

Regular Expression Options

ICU Regular Expression Character Classes

The following was originally from ICU User Guide - UnicodeSet, but has been adapted to fit the needs of this documentation. Specifically, the ICU UnicodeSet documentation describes an ICU C++ object— UnicodeSet. The term UnicodeSet was effectively replaced with Character Class, which is more appropriate in the context of regular expressions. As always, you should refer to the original, official documentation when in doubt.

See Also

ICU User Guide - UnicodeSet

UTS #18 Unicode Regular Expressions - Subtraction and Intersection

UTS #18 Unicode Regular Expressions - Properties

Overview

A character class is a regular expression pattern that represents a set of Unicode characters or character strings. The following table contains some example character class patterns:

|

| Pattern
| Description

| [a-z]
| The lower case letters a through z

| [abc123]
| The six characters a, b, c, 1, 2, and 3

| [\p{Letter}]
| All characters with the Unicode General Category of Letter.

String Values

In addition to being a set of Unicode code point characters, a character class may also contain string values. Conceptually, a character class is always a set of strings, not a set of characters. Historically, regular expressions have treated […] character classes as being composed of single characters only, which is equivalent to a string that contains only a single character.

Character Class Patterns

Patterns are a series of characters bounded by square brackets that contain lists of characters and Unicode property sets. Lists are a sequence of characters that may have ranges indicated by a - between two characters, as in a-z. The sequence specifies the range of all characters from the left to the right, in Unicode order. For example, [a c d-f m] is equivalent to [a c d e f m]. Whitespace can be freely used for clarity as [a c d-f m] means the same as [acd-fm].

Unicode property sets are specified by a Unicode property, such as [:Letter:]. ICU version 2.0 supports General Category, Script, and Numeric Value properties (ICU will support additional properties in the future). For a list of the property names, see the end of this section. The syntax for specifying the property names is an extension of either POSIX or Perl syntax with the addition of =value. For example, you can match letters by using the POSIX syntax [:Letter:], or by using the Perl syntax \p{Letter}. The type can be omitted for the Category and Script properties, but is required for other properties.

The following table lists the standard and negated forms for specifying Unicode properties in both POSIX or Perl syntax. The negated form specifies a character class that includes everything but the specified property. For example, [:^Letter:] matches all characters that are not [:Letter:].

|

| Syntax Style
| Standard
| Negated

| POSIX
| [:type=value:]
| [:^type=value:]

| Perl
| \p{type=value}
| \P{type=value}

See Also

UTS #18 Unicode Regular Expressions - Properties

Character classes can then be modified using standard set operations— Union, Inverse, Difference, and Intersection.

- To union two sets, simply concatenate them. For example, [[:letter:] [:number:]]

- To intersect two sets, use the & operator. For example, [[:letter:] & [a-z]]

- To take the set-difference of two sets, use the - operator. For example, [[:letter:] - [a-z]]

- To invert a set, place a ^ immediately after the opening [. For example, [^a-z]. In any other location, the ^ does not have a special meaning.

The binary operators & and - have equal precedence and bind left-to-right. Thus [[:letter:]-[a-z]-[\u0100-\u01FF]] is equivalent to [[[:letter:]-[a-z]]-[\u0100-\u01FF]]. Another example is the set [[ace][bdf] - [abc][def]] is not the empty set, but instead the set [def]. This only really matters for the difference operation, as the intersection operation is commutative.

Another caveat with the & and - operators is that they operate between sets. That is, they must be immediately preceded and immediately followed by a set. For example, the pattern [[:Lu:]-A] is illegal, since it is interpreted as the set [:Lu:] followed by the incomplete range -A. To specify the set of uppercase letters except for A, enclose the A in a set: [[:Lu:]-[A]].

|

| Pattern
| Description

| [a]
| The set containing a.

| [a-z]
| The set containing a through z and all letters in between, in Unicode order.

| [^a-z]
| The set containing all characters but a through z, that is, U+0000 through a–1 and z+1 through U+FFFF.

| [[pat1][pat2]]
| The union of sets specified by pat1 and pat2.

| [[pat1]&[pat2]]
| The intersection of sets specified by pat1 and pat2.

| [[pat1]-[pat2]]
| The asymmetric difference of sets specified by pat1 and pat2.

| [:Lu:]
| The set of characters belonging to the given Unicode category. In this case, Unicode uppercase letters. The long form for this is [:UppercaseLetter:].

| [:L:]
| The set of characters belonging to all Unicode categories starting with L, that is, [[:Lu:][:Ll:][:Lt:][:Lm:][:Lo:]]. The long form for this is [:Letter:].

See Also

UTS #18 Unicode Regular Expressions - Subtraction and Intersection

String Values in Character Classes

String values are enclosed in {curly brackets}. For example:

|

| Pattern
| Description

| [abc{def}]
| A set containing four members, the single characters a, b, and c and the string def

| [{abc}{def}]
| A set containing two members, the string abc and the string def.

| [{a}{b}{c}][abc]
| These two sets are equivalent. Each contains three items, the three individual characters a, b, and c. A {string} containing a single character is equivalent to that same character specified in any other way.

Character Quoting and Escaping in ICU Character Class Patterns

Single Quote

Two single quotes represent a single quote, either inside or outside single quotes. Text within single quotes is not interpreted in any way, except for two adjacent single quotes. It is taken as literal text— special characters become non-special. These quoting conventions for ICU character classes differ from those of Perl or Java. In those environments, single quotes have no special meaning, and are treated like any other literal character.

Backslash Escapes

Outside of single quotes, certain backslashed characters have special meaning:

|

| Pattern
| Description

| \uhhhh
| Exactly 4 hex digits; h in [0–9A-Fa-f]

| \Uhhhhhhhh
| Exactly 8 hex digits

| \xhh
| 1–2 hex digits

| \ooo
| 1–3 octal digits; o in [0–7]

| \a
| U+0007 BELL

| \b
| U+0008 BACKSPACE

| \t
| U+0009 HORIZONTAL TAB

| \n
| U+000A LINE FEED

| \v
| U+000B VERTICAL TAB

| \f
| U+000C FORM FEED

| \r
| U+000D CARRIAGE RETURN

| \
| U+005C BACKSLASH

Anything else following a backslash is mapped to itself, except in an environment where it is defined to have some special meaning. For example, \p{Lu} is the set of uppercase letters. Any character formed as the result of a backslash escape loses any special meaning and is treated as a literal. In particular, note that \u and \U escapes create literal characters.

Whitespace

Whitespace, as defined by the ICU API, is ignored unless it is quoted or backslashed.

Property Values

The following property value styles are recognized:

|

| Style
| Description

| Short
| Omits the =type argument. Used to prevent ambiguity and only allowed with the Category and Script properties.

| Medium
| Uses an abbreviated type and value.

| Long
| Uses a full type and value.

If the type or value is omitted, then the = equals sign is also omitted. The short style is only used for Category and Script properties because these properties are very common and their omission is unambiguous.

In actual practice, you can mix type names and values that are omitted, abbreviated, or full. For example, if Category=Unassigned you could use what is in the table explicitly, \p{gc=Unassigned}, \p{Category=Cn}, or \p{Unassigned}.

When these are processed, case and whitespace are ignored so you may use them for clarity, if desired. For example, \p{Category = Uppercase Letter} or \p{Category = uppercase letter}.

For a list of properties supported by ICU, see ICU User Guide - Unicode Properties.

See Also

ICU User Guide - Unicode Properties

UTS #18 Unicode Regular Expressions - Properties

Unicode Properties

The following tables list some of the commonly used Unicode Properties, which can be matched in a regular expression with \p{Property}. The tables were created from the Unicode 5.2 Unicode Character Database, which is the version used by ICU that ships with Mac OS X 10.6.

Category

| Category

| L | Letter

| LC | CasedLetter

| Lu | UppercaseLetter

| Ll | LowercaseLetter

| Lt | TitlecaseLetter

| Lm | ModifierLetter

| Lo | OtherLetter

|

| P | Punctuation

| Pc | ConnectorPunctuation

| Pd | DashPunctuation

| Ps | OpenPunctuation

| Pe | ClosePunctuation

| Pi | InitialPunctuation

| Pf | FinalPunctuation

| Po | OtherPunctuation

|

| N | Number

| Nd | DecimalNumber

| Nl | LetterNumber

| No | OtherNumber

|

| M | Mark

| Mn | NonspacingMark

| Mc | SpacingMark

| Me | EnclosingMark

|

| S | Symbol

| Sm | MathSymbol

| Sc | CurrencySymbol

| Sk | ModifierSymbol

| So | OtherSymbol

|

| Z | Separator

| Zs | SpaceSeparator

| Zl | LineSeparator

| Zp | ParagraphSeparator

|

| C | Other

| Cc | Control

| Cf | Format

| Cs | Surrogate

| Co | PrivateUse

| Cn | Unassigned

Script

| Arabic | Armenian | Balinese

| Bengali | Bopomofo | Braille

| Buginese | Buhid | Canadian_Aboriginal

| Carian | Cham | Cherokee

| Common | Coptic | Cuneiform

| Cypriot | Cyrillic | Deseret

| Devanagari | Ethiopic | Georgian

| Glagolitic | Gothic | Greek

| Gujarati | Gurmukhi | Han

| Hangul | Hanunoo | Hebrew

| Hiragana | Inherited | Kannada

| Katakana | Kayah_Li | Kharoshthi

| Khmer | Lao | Latin

| Lepcha | Limbu | Linear_B

| Lycian | Lydian | Malayalam

| Mongolian | Myanmar | New_Tai_Lue

| Nko | Ogham | Ol_Chiki

| Old_Italic | Old_Persian | Oriya

| Osmanya | Phags_Pa | Phoenician

| Rejang | Runic | Saurashtra

| Shavian | Sinhala | Sundanese

| Syloti_Nagri | Syriac | Tagalog

| Tagbanwa | Tai_Le | Tamil

| Telugu | Thaana | Thai

| Tibetan | Tifinagh | Ugaritic

| Unknown | Vai | Yi

Extended Property Class

| ASCII_Hex_Digit | Alphabetic

| Bidi_Control | Dash

| Default_​Ignorable_​Code_​Point | Deprecated

| Diacritic | Extender

| Grapheme_Base | Grapheme_Extend

| Grapheme_Link | Hex_Digit

| Hyphen | IDS_​Binary_​Operator

| IDS_Trinary_Operator | ID_Continue

| ID_Start | Ideographic

| Join_Control | ​​Logical_Order_Exception

| Lowercase | Math

| Noncharacter_Code_Point | Other_Alphabetic

| Other_Default_Ignorable_Code_Point | Other_Grapheme_Extend

| Other_ID_Continue | Other_ID_Start

| Other_Lowercase | Other_Math

| Other_Uppercase | Pattern_Syntax

| Pattern_White_Space | Quotation_Mark

| Radical | STerm

| Soft_Dotted | Terminal_Punctuation

| Unified_Ideograph | Uppercase

| Variation_Selector | White_Space

| XID_Continue | XID_Start

Unicode Character Database

Unicode properties are defined in the Unicode Character Database, or UCD. From time to time the UCD is revised and updated. The properties available, and the definition of the characters they match, depend on the UCD that ICU was built with.

Note: In general, the ICU and UCD versions change with each major operating system release.

See Also

UTS #18 Unicode Regular Expressions - Properties

UTS #18 Unicode Regular Expressions - Compatibility Properties

Unicode Character Database

The Unicode Standard - Unicode 5.2

Versions of the Unicode Standard

ICU Replacement Text Syntax

Replacement Text Syntax

|

| Character
| Description

| $n
| The text of capture group n will be substituted for $n. n must be ≥ 0 and not greater than the number of capture groups. A $ not followed by a digit has no special meaning, and will appear in the substitution text as itself, a $.

| \
| Treat the character following the backslash as a literal, suppressing any special meaning. Backslash escaping in substitution text is only required for $ and \, but may proceed any character. The backslash itself will not be copied to the substitution text.

Case Change During Replacement

TextSoap includes some additional processing commands which allow you to dynamically change the case of your replacement. This is similar to PCRE (which are typically preceded with \, ie. \U, \u, \L, \l, \E). In TextSoap, these commands are preceded with a $ (dollar sign).

|

| Cmd
| Action

| $U
| All future text is uppercased

| $u
| Next character is uppercased

| $L
| All future text is lowercased

| $l
| Next character is lowercased

| $E
| All future text case is preserved