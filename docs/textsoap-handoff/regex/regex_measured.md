# Measured regex / replacement behaviour

51 cases executed by the real TextSoap 9 engine in an isolated
sandbox. Every row below is emitted from `regex_cases.json`, which carries the
engine's actual `output`; nothing here is hand-written or copied.

Cases are declared in `specs/probe_cases.py` (`REGEX_PROBES`) and produced by
`scripts/measure_verified.py`.

## How the cases were run

Each case becomes a real cleaner containing one serialized `find` action,
executed with `-[TSText applyCleaner:]` by name — the same path the app itself
uses. `regex_options`, `match_case`, `type` and `granularity` are that find
action's own parameters.

```objc
[t setString:input];
[t applyCleaner:@"ZZ_R<n>"];   // n indexes this case
NSString *out = [t string];
```

The rig that does this is in `measure/` and is runnable against the sandbox
library shipped in the same folder.

## What the measurements say

- `match_case = false` is the **only** way to get case-insensitive matching.  
  evidence `case-match_case-false`: `ABC abc` with find `abc` → `Y` gives `Y Y`
- `match_case = true` is the serialized default and is case-sensitive.  
  evidence `case-match_case-true`: `ABC abc` with find `abc` → `Y` gives `ABC Y`
- The option letter `i` does **nothing**. Do not implement it as ignore-case.  
  evidence `case-i-letter-inert`: `ABC abc` with find `abc` → `Y` gives `ABC Y`
- An inline `(?i)` **does** work, unlike the options field.  
  evidence `case-inline-i-flag`: `ABC abc` with find `(?i)abc` → `Y` gives `Y Y`
- With no options, `^` matches only at the very start of the text.  
  evidence `opt6-empty`: `a\nb\na` with find `^a` → `X` gives `X\nb\na`
- `m` makes `^`/`$` match at line boundaries.  
  evidence `opt6-m`: `a\nb\na` with find `^a` → `X` gives `X\nb\nX`
- Uppercase option letters are inert; only lowercase `m` and `s` do anything.  
  evidence `opt6-M-uppercase-inert`: `a\nb\na` with find `^a` → `X` gives `X\nb\na`
- `s` makes `.` match a newline.  
  evidence `opt6-s`: `a\nb` with find `a.b` → `Z` gives `Z`
- An unknown option string is silently ignored, not an error.  
  evidence `opt6-zzz-inert`: `ABC abc` with find `abc` → `Y` gives `ABC Y`
- There is no separate `match_lines` parameter key; only `regex_options` matters.  
  evidence `opt6-match_lines-key-inert`: `a\nb\na` with find `^a` → `X` gives `X\nb\na`
- `type = 0` means literal text.  
  evidence `type-0-literal`: `a.b` with find `.` → `Z` gives `aZb`
- `type = 1` means regular expression.  
  evidence `type-1-regex`: `a.b` with find `.` → `Z` gives `ZZZ`
- Omitting `type` behaves like `type = 1`.  
  evidence `type-1-is-default`: `a.b` with find `.` → `Z` gives `ZZZ`
- `granularity` does not disable multiline matching: 0, 1, 2 and 3  
  evidence `gran-0`: `a\nb\na` with find `^a` → `X` gives `X\nb\nX`
- `$0` is the whole match and may be repeated.  
  evidence `tok-whole-match`: `AbCd` with find `(AbCd)` → `$0$0` gives `AbCdAbCd`
- `$n` is capture group n, and groups may be reordered.  
  evidence `tok-backref`: `ab12` with find `([a-z]+)([0-9]+)` → `$2-$1` gives `12-ab`
- `$u` uppercases only the FIRST character of the substitution.  
  evidence `tok-case-u-mixed`: `AbCd` with find `(AbCd)` → `$u$1` gives `AbCd`
- `$U` uppercases the whole substitution.  
  evidence `tok-case-U-mixed`: `AbCd` with find `(AbCd)` → `$U$1` gives `ABCD`
- `$l` lowercases only the first character.  
  evidence `tok-case-l-mixed`: `AbCd` with find `(AbCd)` → `$l$1` gives `abCd`
- `$L` lowercases the whole substitution.  
  evidence `tok-case-L-mixed`: `AbCd` with find `(AbCd)` → `$L$1` gives `abcd`
- The case tokens apply to `$0` as well.  
  evidence `tok-case-applies-to-zero`: `AbCd` with find `(AbCd)` → `$U$0` gives `ABCD`
- `$e` is consumed and substitutes as **empty**.  
  evidence `tok-e-inert`: `abc` with find `b` → `<$e>` gives `a<>c`
- `$E` behaves exactly like `$e`; it is NOT an upper-casing token.  
  evidence `tok-E-inert`: `abc` with find `b` → `<$E>` gives `a<>c`
- `$E` before a group does not alter that group.  
  evidence `tok-E-before-group`: `AbCd` with find `(AbCd)` → `$E$1` gives `AbCd`
- `\U` does not uppercase; it writes a literal `U`.  
  evidence `tok-uppercase-letter-unsupported`: `the quick brown fox` with find `(quick)` → `\\U$1` gives `the Uquick brown fox`
- `$$` substitutes as **empty**; it is not a literal `$`.  
  evidence `tok-literal-dollar`: `cost 5` with find `5` → `$$` gives `cost `
- `\$` is how you write a literal `$`.  
  evidence `tok-backslash-dollar`: `cost 5` with find `5` → `\\$` gives `cost $`
- A group that did not participate substitutes as empty.  
  evidence `tok-missing-group`: `ac` with find `(a)(z)?(c)` → `[$2]` gives `[]`
- An out-of-range group number substitutes as empty.  
  evidence `tok-outofrange-group`: `abc` with find `b` → `$9` gives `ac`
- `$01` parses as group 0 followed by a literal `1`, not as group 01.  
  evidence `tok-number-extra-digit`: `AbCd` with find `(AbCd)` → `$01` gives `AbCd1`
- The `${1}` brace form is NOT supported: the `$` is dropped and the rest is literal.  
  evidence `tok-brace-syntax-unsupported`: `AbCd` with find `(AbCd)` → `${1}` gives `1}`
- An unknown `$word` is eaten as a token and substitutes as empty, keeping the rest of the text.  
  evidence `tok-unknown-token`: `abc` with find `a` → `$UNKNOWNTOKEN` gives `NKNOWNTOKENbc`
- When nothing matches, the input passes through untouched.  
  evidence `no-match`: `hello` with find `zzz` → `!` gives `hello`

> Each bullet above is asserted at pack-build time against the measured data.
> If a measurement changes, the build fails rather than shipping a stale claim.

## Case tables

### Case sensitivity

| case | input | find | replace | other params | measured output |
|---|---|---|---|---|---|
| `case-match_case-false` | `ABC abc` | `abc` | `Y` | `match_case=False` | `Y Y` |
| `case-match_case-true` | `ABC abc` | `abc` | `Y` | `match_case=True` | `ABC Y` |
| `case-i-letter-inert` | `ABC abc` | `abc` | `Y` | `regex_options=i` | `ABC Y` |
| `case-I-letter-inert` | `ABC abc` | `abc` | `Y` | `regex_options=I` | `ABC Y` |
| `case-ignorecase-word-inert` | `ABC abc` | `abc` | `Y` | `regex_options=ignorecase` | `ABC Y` |
| `case-i-with-match_case-true` | `ABC abc` | `abc` | `Y` | `regex_options=i` `match_case=True` | `ABC Y` |
| `case-inline-i-flag` | `ABC abc` | `(?i)abc` | `Y` | — | `Y Y` |

### The options field

| case | input | find | replace | other params | measured output |
|---|---|---|---|---|---|
| `opt6-empty` | `a\nb\na` | `^a` | `X` | — | `X\nb\na` |
| `opt6-m` | `a\nb\na` | `^a` | `X` | `regex_options=m` | `X\nb\nX` |
| `opt6-M-uppercase-inert` | `a\nb\na` | `^a` | `X` | `regex_options=M` | `X\nb\na` |
| `opt6-lm` | `a\nb\na` | `^a` | `X` | `regex_options=lm` | `X\nb\nX` |
| `opt6-matchlines-word-inert` | `a\nb\na` | `^a` | `X` | `regex_options=matchlines` | `X\nb\nX` |
| `opt6-s` | `a\nb` | `a.b` | `Z` | `regex_options=s` | `Z` |
| `opt6-S-uppercase-inert` | `a\nb` | `a.b` | `Z` | `regex_options=S` | `a\nb` |
| `opt6-dotall-word-inert` | `a\nb` | `a.b` | `Z` | `regex_options=dotall` | `a\nb` |
| `opt6-zzz-inert` | `ABC abc` | `abc` | `Y` | `regex_options=zzz` | `ABC Y` |
| `opt6-match_lines-key-inert` | `a\nb\na` | `^a` | `X` | — | `X\nb\na` |
| `opt6-dot_matches_all-key-inert` | `a\nb` | `a.b` | `Z` | — | `a\nb` |
| `opt6-inline-m` | `a\nb\na` | `(?m)^a` | `X` | — | `X\nb\nX` |

### type and granularity

| case | input | find | replace | other params | measured output |
|---|---|---|---|---|---|
| `type-0-literal` | `a.b` | `.` | `Z` | `type=0` | `aZb` |
| `type-1-regex` | `a.b` | `.` | `Z` | `type=1` | `ZZZ` |
| `type-1-is-default` | `a.b` | `.` | `Z` | — | `ZZZ` |
| `gran-0` | `a\nb\na` | `^a` | `X` | `regex_options=m` `granularity=0` | `X\nb\nX` |
| `gran-1` | `a\nb\na` | `^a` | `X` | `regex_options=m` `granularity=1` | `X\nb\nX` |
| `gran-2` | `a\nb\na` | `^a` | `X` | `regex_options=m` `granularity=2` | `X\nb\nX` |
| `gran-3` | `a\nb\na` | `^a` | `X` | `regex_options=m` `granularity=3` | `X\nb\nX` |

### Replacement template tokens

| case | input | find | replace | other params | measured output |
|---|---|---|---|---|---|
| `tok-backref` | `ab12` | `([a-z]+)([0-9]+)` | `$2-$1` | — | `12-ab` |
| `tok-whole-match` | `AbCd` | `(AbCd)` | `$0$0` | — | `AbCdAbCd` |
| `tok-group-repeat` | `AbCd` | `(AbCd)` | `$1$1` | — | `AbCdAbCd` |
| `tok-case-u-mixed` | `AbCd` | `(AbCd)` | `$u$1` | — | `AbCd` |
| `tok-case-U-mixed` | `AbCd` | `(AbCd)` | `$U$1` | — | `ABCD` |
| `tok-case-l-mixed` | `AbCd` | `(AbCd)` | `$l$1` | — | `abCd` |
| `tok-case-L-mixed` | `AbCd` | `(AbCd)` | `$L$1` | — | `abcd` |
| `tok-case-mid` | `snake_case` | `_([a-z])` | `$u$1` | — | `snakeCase` |
| `tok-case-applies-to-zero` | `AbCd` | `(AbCd)` | `$U$0` | — | `ABCD` |
| `tok-case-on-other-group` | `AbCd` | `(Ab)(Cd)` | `$U$2` | — | `CD` |
| `tok-case-l-on-other-group` | `AbCd` | `(Ab)(Cd)` | `$l$2` | — | `cd` |
| `tok-uppercase-letter-unsupported` | `the quick brown fox` | `(quick)` | `\\U$1` | — | `the Uquick brown fox` |
| `tok-E-inert` | `abc` | `b` | `<$E>` | — | `a<>c` |
| `tok-e-inert` | `abc` | `b` | `<$e>` | — | `a<>c` |
| `tok-E-before-group` | `AbCd` | `(AbCd)` | `$E$1` | — | `AbCd` |
| `tok-literal-dollar` | `cost 5` | `5` | `$$` | — | `cost ` |
| `tok-backslash-dollar` | `cost 5` | `5` | `\\$` | — | `cost $` |
| `tok-bare-dollar` | `a b` | ` ` | `$` | — | `ab` |
| `tok-missing-group` | `ac` | `(a)(z)?(c)` | `[$2]` | — | `[]` |
| `tok-outofrange-group` | `abc` | `b` | `$9` | — | `ac` |
| `tok-number-extra-digit` | `AbCd` | `(AbCd)` | `$01` | — | `AbCd1` |
| `tok-brace-syntax-unsupported` | `AbCd` | `(AbCd)` | `${1}` | — | `1}` |
| `tok-empty-template` | `keep drop` | `drop` | `` | — | `keep ` |
| `tok-unknown-token` | `abc` | `a` | `$UNKNOWNTOKEN` | — | `NKNOWNTOKENbc` |

### No-match passthrough

| case | input | find | replace | other params | measured output |
|---|---|---|---|---|---|
| `no-match` | `hello` | `zzz` | `!` | — | `hello` |

## Known limits

- No case exercises a capture that spans a newline (e.g. `(?s)` capturing across `\n`).
- `granularity` produced identical output for every value tried; treat it as
  unmeasured rather than understood.
- The replacement **count** is not observed on this path, only the resulting text.

