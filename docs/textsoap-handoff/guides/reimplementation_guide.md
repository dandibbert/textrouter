# Reimplementation guide

You are building a text-cleaning tool that should behave like TextSoap for the
cases that matter. You do not need TextSoap, and you do not need to match its
internals. The acceptance standard is: **same input → same output**
(1482 measured cases ship with this pack).

## Recommended build order

1. **Core engine + the measured 38.** Implement the cleaners in
   `cleaners/reference_implementation.py` (they are already written in plain
   Python and already score 943/1053 cases (89.6%) against the captured truth).
   Port them to your language, then run them against
   `cleaners/behaviour_truth.json`. Keep `judge.py` as the regression gate.

2. **The action/chain layer.** This is what makes the tool *customizable*, and
   it is the part users actually miss when it is absent. The format is verified
   (`actions/chain_format.md`); the action set is inventoried
   (`actions/action_reference.md`). Implement at minimum: `find`,
   `find.regex`, `find.batch`, `find.delete`, `text.insert`, `text.delete`,
   `text.tag`, `line.addto`, `line.removefrom`, `line.sort`, `runtime.group`.

3. **Chains first, conditionals later.** `if.match.*` / `runtime.*` containers
   are powerful but their schemas are the least verified part of this pack.

4. **Regex semantics.** Read `regex/regex_syntax.md` (vendor) and
   `regex/regex_measured.md` (measured). The replacement tokens are the trap.

5. **Clipboard loop.** TextSoap's real value is the loop: take clipboard text,
   run cleaner(s), put it back. `docs/vendor_help/clipboard_workspace.md`
   describes the intended UX. This needs no reverse engineering.

6. **Batch.** `docs/vendor_help/batch_cleaning.md`. The app's own `find.batch`
   action ("Find and Replace with List") is the in-app version of batch replace
   and is a good model for a clearer UI.

## Traps found the hard way

- **Load-once library.** TextSoap reads its cleaner library at launch. If you
  mimic that format, do not forget a reload path; the app itself has none.
- **`type` in a find action is overloaded**: it appears as both `1` and `true`
  in real saved data (`type: 1` = regex mode). Treat it as a small enum, not a
  bool.
- **`granularity`**: measured to produce identical output for 0, 1, 2 and 3 on
  every case tried, including with multiline on. Do **not** implement it as a
  per-line switch; treat it as unmeasured (see `regex/regex_measured.md`).
- **Case sensitivity is `match_case` only.** The `i` / `ignorecase` letters in
  `regex_options` were measured to do nothing at all. Only lowercase `m` and
  `s` in that field change behaviour. Getting this backwards is the easiest way
  to ship a find/replace that disagrees with the real app.
- **`$E` / `$e` substitute as empty** and are not upper-casing tokens; the
  case-token family is `$u`, `$U`, `$l`, `$L`, all of which apply per match.
- **`height` / `show_details` are pure UI** and must never affect behaviour.

## Where the uncertainty is

`guides/unknown_behaviour.md` lists exactly what is not known. Do not guess
and do not silently approximate: if a cleaner is unmeasured, either measure it
(on a Mac, using `measure/`) or document the divergence.
