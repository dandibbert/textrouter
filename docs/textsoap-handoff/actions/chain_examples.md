# Measured chain examples

15 cleaners (real `find`/container action chains) executed by the TextSoap 9 engine in an isolated sandbox. Every entry carries the engine's actual `output`; nothing here is hand-written.

Cases are declared in `specs/probe_cases.py` (`CHAIN_PROBES`) and produced by `scripts/measure_verified.py`.

- `ch-find-basic` -- literal find/replace through a cleaner
- `ch-find-regex` -- regex capture backreference
- `ch-find-delete` -- replace with an empty template == delete
- `ch-chain-two-finds` -- two find actions in sequence: order matters
- `ch-chain-two-finds-reversed` -- the same two actions reversed: compare with the previous case
- `ch-chain-three` -- three chained finds, each seeing the previous output
- `ch-find-no-match` -- no match: untouched
- `ch-find-dollarish` -- chain-level check of the $$ template
- `ch-find-case-u` -- the $u capitalisation behaviour, end to end
- `ch-group-empty` -- an empty container must be a no-op
- `ch-unknown-action` -- CONTROL: an unknown action id must not corrupt the text
- `ch-line-sort` -- line.sort default parameters, measured rather than guessed
- `ch-text-wrap` -- text.wrap: the width key is undocumented; the output reveals the default
- `ch-remove-extra-spaces` -- a documented-cleaner action used directly
- `ch-two-actions-independent` -- two non-overlapping replaces
