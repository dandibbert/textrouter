# TextSoap 9 — portable behaviour pack

Handoff pack for reimplementing TextSoap's text-cleaning behaviour **without
access to TextSoap**. Everything needed is either in this folder or is a
measurement you can reproduce on macOS.

Source app: `/Applications/textsoap9.app`, TextSoap 9.7.6 (build 3278),
bundle id `com.unmarked.textsoap9`, Universal (x86_64 + arm64).
Pack built: 2026-10-14 07:19 UTC

## What this is, and what it is not

This pack describes **behaviour**, not code. It is explicitly *not* a
requirement to match TextSoap's internals, and the goal of a reimplementation is
**observable equivalence**: same input, same output.

Honest coverage at the time of writing:

| area | state |
|---|---|
| Built-in cleaner inventory (names, descriptions) | **194/194 inventoried** (188 with prose descriptions) |
| Cleaners with **measured** behaviour (differential truth) | **38 of 194** — 1482 measured cases |
| Cleaners with a **candidate implementation** in this pack | **27** — 943/1053 cases (89.6%) agreement on comparable cases |
| Action model (60 user-composable actions) | inventoried with id/title/description; schema known for 1/60 |
| Action-chain serialization format | **verified** (15 chain cases executed on the live engine) |
| Find/Replace regex semantics | 51 measured cases |
| Clipboard Space UI behaviour | described from vendor docs; **not** measured |

The number that matters: **38 of 194 cleaners have
measured behaviour.** The remaining ones are not black magic — most are
one-line transforms described in their own description text — but they are
*unverified*, and this pack marks them so rather than pretending.

## Read this first: five measured facts that are easy to get wrong

Each of these was measured against the real engine and contradicts the obvious
guess. They are repeated in the relevant guides, because getting one wrong
produces silent divergence rather than an error.

1. **Case sensitivity is `match_case` only.** The `i` / `ignorecase` letters in
   the `regex_options` field do **nothing** (only lowercase `m` and `s` in that
   field have any effect). Implementing `i` as ignore-case is wrong.
2. **`$E` / `$e` substitute as empty.** The case tokens are `$u` `$U` `$l` `$L`,
   and they apply **per match**. `$$` is empty, not a literal `$`; the literal
   dollar is `\$`. The `${1}` brace form is not supported.
3. **`granularity` is not a per-line switch.** Values 0–3 produced identical
   output on every case tried, including with multiline on. Treat it as
   unmeasured.
4. **Generated cleaner records must use `fileVersion: 6`.** With `2` the engine
   silently ignores them and passes text through unchanged. Likewise a `find`
   node without an explicit `type` defaults to literal.
5. **`find:replace:options:` and `regexFind:replace:options:` do not reproduce
   user-visible behaviour.** Only `applyCleaner:` with a serialized `find`
   action does. See `regex/regex_measured.md`.

## Layout

```
OVERVIEW.md                     ← you are here
cleaners/
  builtin_inventory.json        all 194 built-ins as data (token/name/desc)
  builtin_inventory.md          the same, readable, grouped
  behaviour_truth.json          MEASURED outputs: 38 cleaners × 39 inputs
  corpus.json                   the 39 test inputs
  reference_implementation.py   candidate implementations (27 cleaners)
  judge.py                      scores a candidate against behaviour_truth.json
  spec_notes.md                 per-cleaner spec, doc quote + measured notes
actions/
  action_reference.md           60 composable actions, readable
  action_registry.json          the same as data (ids, titles, groups, classes)
  chain_format.md               the on-disk cleaner/chain format, verified
  chain_examples.json           real, executed chain examples
regex/
  regex_syntax.md               vendor regex reference (verbatim, converted)
  regex_measured.md             measured replacement-token behaviour
  regex_cases.json              measured regex cases
docs/
  vendor_help/*.md              the app's own help book, converted to text
guides/
  reimplementation_guide.md     how to use this pack; recommended build order
  unknown_behaviour.md          what is NOT known and how to measure it
measure/
  oracle.py                     rig: drive real TextSoap over lldb
  run_oracle.py                 regenerate behaviour_truth.json
  measure.py                    run new cases through the rig
  README.md                     how to use the rig (macOS only)
```

## How to start

Read `guides/reimplementation_guide.md` first — it tells you which order to
build in and where the traps are. `cleaners/behaviour_truth.json` is your test
suite: it is real captured output and it will not lie to you.

If you are on Linux with no Mac available: everything under `cleaners/`,
`actions/`, `regex/`, `docs/` is plain data and prose. You can implement and
test against `behaviour_truth.json` for the 38 measured cleaners immediately.
The `measure/` rig only helps when you want to confirm a cleaner that has no
truth yet; the Mac is the oracle, not a dependency.
