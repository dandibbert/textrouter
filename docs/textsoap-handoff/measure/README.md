# The measurement rig (macOS only)

This rig is **optional**. It exists so the captured truth can be *extended* when
you need to, using the real app as an oracle. The Linux reimplementation does
not need it: the measured results are already shipped in this pack.

## The only entry point that reproduces user-visible behaviour

```objc
id t = [TSText new];
[t setString:@"input"];
[t applyCleaner:@"Cleaner Name"];   // built-in OR a cleaner from a library
NSString *out = [t string];
```

Two other plausible-looking calls were tried and **do not** work:

| call | measured behaviour |
|---|---|
| `-[TSText find:replace:options:]` | replaces nothing, under every argument form tried |
| `-[TSText regexFind:replace:options:]` | replaces, but **ignores its third argument** |

So a `find` action must go through `applyCleaner:`, as a serialized action
inside a cleaner.

## Two traps that cost real time

1. **Synthesized cleaners must be `fileVersion: 6`.** The app silently ignores
   records written with `fileVersion: 2` and passes the text through unchanged.
   If every chain result equals its input, suspect this first.
2. **Isolation needs `CFFIXED_USER_HOME`, not `HOME`.** Overriding `HOME` does
   not change `NSHomeDirectory()`, so a "sandboxed" run silently drives the
   *real* app against the *user's* live library. Also scaffold the app's
   UUID-keyed group container, or the app will not find its library at all.

A correct sandbox run applies the sandbox's own cleaner and gets the sandbox's
expected output back; the rig in this folder does that.

## Requirements

- macOS with TextSoap 9 installed (for built-in cleaners, a running instance)
- `lldb` (ships with Xcode command line tools)
- Python 3.8+

## Use

```bash
# measure specific new cases
python3 measure.py my_cases.json

# measure every corpus input against some cleaners by name
python3 measure.py --cleaners "Scrub" "Make Paragraphs"

# re-capture the shipped truth table (slow, ~4 min)
python3 run_oracle.py
```

## Caveats learned while building this

- Attach is fragile if the app is busy; run cases in chunks (~120 at a time)
  and parse with the `@@@IDX`/`@@@END` markers, not by line heuristics.
- Multi-line and empty outputs are normal -- never infer "no output".
- Swift class names in the binary need the `textsoap9.` module prefix at runtime
  (`NSClassFromString(@"textsoap9.TS9Cleaner")`), while `TSText` is a plain ObjC
  class. The `otool` dump omits that prefix, which is why lookups return nil.
- Do **not** write into the live `~/Library/...` library of a running app you
  care about: the library is read once and a bad write can crash the app.
