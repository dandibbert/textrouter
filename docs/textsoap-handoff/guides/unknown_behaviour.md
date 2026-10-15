# What is NOT known (and how to find out)

Written so a reimplementer never has to guess. Each item says what is missing
and how to close it.

## 1. Cleaner behaviour without measurements

**38 of 194** built-ins have measured behaviour. The rest
are described in `cleaners/builtin_inventory.json` but never executed against a
known input. Examples of unmeasured cleaners:

- `Remove Whitespace Before Paragraphs` (`#REMOVE_WHITESPACE_AT_BEGINNING`)
- `Capitalize Sentences with Lowercasing` (`#CAPITALIZE_SENTENCES_ALT`)
- `Capitalize Common Tech Names` (`#CAPITALIZE_COMMON_TECH_NAMES`)
- `Capitalize Lines` (`#CAPITALIZE_LINES`)
- `Capitalize Lines with Lowercasing` (`#CAPITALIZE_LINES_ALT`)
- `Randomcase` (`#RANDOMCASE`)
- `Fixup macOS Name` (`#MACOS_NAME_FIX`)
- `Capitalize Apple Names` (`#APPLE_NAME_FIX`)
- `Reverse All Characters` (`#REVERSE_CHARACTERS`)
- `Reverse Word Order` (`#REVERSE_WORDORDER`)
- `Reverse Characters in Each Word` (`#REVERSE_CHARS_INEACHWORD`)
- `Strip Diacritic Marks` (`#STRIP_DIACRITICS`)
- ...and all others not marked **(measured)**

**How to close:** on a Mac, add cases to a JSON file and run
`measure/measure.py cases.json`. The rig drives the real engine and prints
exact outputs. This is a measurement task, not a reverse-engineering task.

## 2. Action parameter schemas

60 actions are inventoried; only `find` has a fully known parameter schema.
`actions/action_registry.json` records which keys have been observed in real
saved data versus inferred. Notably **unverified**: `find.batch`'s list
selector, `style.text.*` value encodings, `runtime.javascript` context.

**How to close:** write a cleaner containing the action, save it in the app,
then read the library plist back (`actions/chain_format.md` shows the shape).
Serialization is the ground truth for parameter names.

## 3. Composition order of multi-step cleaners

`Scrub` is documented as "calls the cleaners in order", but the exact order and
whether it skips steps is not confirmed. Same for `MyScrub` (user-configurable,
falls back to Scrub).

**How to close:** differential testing — feed one input that each candidate
step would change differently, and compare against `Scrub`'s real output.

## 4. Container / conditional semantics

`if.match.*`, `runtime.repeat`, `runtime.processlist`,
`if.match.text.processlines` are documented at the user level but their
evaluation model (what `text` is inside the block, whether nested finds see
rewritten text) is **not verified**.

**How to close:** build tiny chains in the app, save, execute, read back.

## 5. Clipboard Space UI

The workflow is documented (`docs/vendor_help/clipboard_workspace.md`) but no
behaviour was measured — window lifetime, what happens on empty clipboard,
timing. This is normal UI work; no oracle needed.

## 6. Things deliberately not claimed

- No claim that any implementation here is source-identical to TextSoap.
- No claim about `MMUtilities`/`BaseAction` internals — those were mapped for
  navigation during analysis and are **not** required to match behaviour.
