# Cleaner / action-chain on-disk format

**Status: verified.** This format was read back from, written to, and executed by
the real TextSoap engine. A cleaner you write in this shape loads and runs.

TextSoap stores the whole library as ONE binary property list (bplist) at
`~/Library/Application Support/TextSoap/textsoap9.textsoap9data/main0090.textsoap9Lib`.
It is loaded **once at app launch** — editing it while the app runs has no
effect until relaunch. Unknown cleaner names are safe no-ops.

## Top level

```
{
  "version": <int>,
  "contents": {
     "cleaners": { "<UUID>": <cleaner record>, ... },
     "groups":   { ... }
  }
}
```

## Cleaner record

```
{
  "name":         "My Cleaner",          # this is the string applyCleaner: takes
  "uuid":         "<UUID uppercase>",
  "datatype":     "cleaner",
  "actionList":   [ <action node>, ... ], # ordered; execution order
  "properties":   {"author": ..., "comment": ..., "version": ...},
  "lastModified": <date>,
  "min-required": "9.0",
  "version":      <int>,
  "fileVersion":  <int>,
  "revision":     <int>
}
```

## Action node

Every node — the container, each action, and the terminator — shares one shape:

```
{
  "identifier":    "find",        # action id, or "cleaner.actions" / "endlist"
  "parameters":    { ... },       # action-specific; see action_registry.json
  "enabled":       bool,
  "show_children": bool,
  "show_details":  bool,
  "height":        float          # UI-only, carries no behaviour
}
```

`actionList` for a real cleaner is bracketed by two structural nodes:

```
[ {"identifier": "cleaner.actions", ...},   # opens the list
  <actions...>,
  {"identifier": "endlist", ...} ]          # closes it
```

Containers (`runtime.group`, `if.match.*`, `runtime.repeat`, ...) nest further
action nodes in their own `actionList`/children.

## Executing a chain

The engine entry point is the same as for built-ins:

```objc
id t = [TSText new];
[t setString:@"input"];
[t applyCleaner:@"My Cleaner"];      // by name, from the loaded library
NSString *out = [t string];
```

This is exactly how the pack's chain cases were measured, so a chain that
matches `behaviour_truth.json`'s chain entries is observationally equivalent on
those inputs.

## Traps when generating this format yourself

Measured the hard way; getting either wrong produces a silent no-op, not an
error.

1. **`fileVersion` must be `6`.** Records written with `fileVersion: 2` are
   accepted by the parser and then **ignored by the engine** — `applyCleaner:`
   succeeds and the text passes through unchanged. If every generated cleaner
   returns its input verbatim, check this field first. Real cleaners saved by
   the app use `6`.
2. **`type` on a `find` node is `0` (literal) or `1` (regex).** A hand-built
   node that omits `type` defaults to literal `0`, so a regex `find` silently
   matches nothing and the input passes through. Always set it explicitly.
3. **A cleaner is only visible to the engine after the library is loaded at
   launch.** Writing the plist into a running app's container has no effect.
   Launch a fresh instance (or write the library before launching).
