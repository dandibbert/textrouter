#!/usr/bin/env python3
"""Differential oracle: run a cleaner over inputs in the LIVE TextSoap engine.

This is the equivalence judge for the reimplementation. It attaches lldb to the
already-running textsoap9, drives TSText (new -> setString: -> applyCleaner: ->
string) and parses the values back out. No Hopper, no REA.

  python3 oracle.py cases.json > truth.json

cases.json: [{"cleaner": "Remove Extra Spaces", "input": "a  b", "id": "sp-1"}, ...]
truth.json: same list with "output" (or "error") filled in.

Action-chain support (the half that makes TextSoap *customizable*):

a chain case carries "actions": [{"identifier": "find", "parameters": {...}}, ...]
instead of "cleaner". A whole cleaner item (both halves of a real library
record) is passed via "cleaner_item": {"name": ..., "actionList": [...]}; it is
written into an ISOLATED library under $TS_HOME, a private textsoap9 instance is
launched against it, and the chain is executed through applyCleaner: by name.
The caller's own library is never touched unless TS_ALLOW_SIDE_EFFECTS=1.
"""
import json, subprocess, sys, re, tempfile, os, shutil, plistlib, time, uuid, signal

APP = "/Applications/textsoap9.app/Contents/MacOS/textsoap9"


def run_lldb(pid, lines):
    """Run an explicit lldb command list against `pid` and return the raw log.

    The single lldb driver for the whole repository. Anything that wants to send
    its own command list (probes) or its own case list (measurements) goes
    through here; there is deliberately no second copy.
    """
    tmp = tempfile.mkdtemp(prefix="tslldb-")
    try:
        cmds = os.path.join(tmp, "c.lldb")
        open(cmds, "w").write("\n".join(lines) + "\ndetach\nquit\n")
        p = subprocess.run(["lldb", "-p", str(pid), "-s", cmds, "-o", "quit", "-b"],
                           capture_output=True, text=True, timeout=1800)
        return p.stdout + "\n" + p.stderr
    finally:
        shutil.rmtree(tmp, ignore_errors=True)


# Reading a string back through one round trip, newlines escaped so that a
# multi-line value cannot be confused with the surrounding log.
READ_STRING = ('expr -l objc -O -- (id)[(id)[$t performSelector:(SEL)sel_registerName("string")] '
               'performSelector:(SEL)sel_registerName("stringByReplacingOccurrencesOfString:withString:") '
               'withObject:@"\\n" withObject:@"<LF>"]')

SET_TEXT = ('expr -l objc -O -- (void)[$t performSelector:(SEL)sel_registerName("setString:") '
            'withObject:@"%s"]')

TEXT_NEW = ('expr -l objc -- id $t = (id)[(id)objc_getClass("TSText") '
            'performSelector:(SEL)sel_registerName("new")]')

APPLY_CLEANER = ('expr -l objc -O -- (void)[$t performSelector:(SEL)sel_registerName("applyCleaner:") '
                 'withObject:@"%s"]')


def case_script(cases, call_for, filter_state=lambda c: c.get("input", "")):
    """Build a tagged lldb script that runs one case per index.

    `call_for(i, case)` returns the lldb command that performs the action under
    test. Every case is bracketed by @@@IDX/@@@END markers so multi-line and
    empty results are recovered exactly.
    """
    lines = [TEXT_NEW]
    for i, c in enumerate(cases):
        lines.append('expr -l objc -O -- @"@@@IDX %d"' % i)
        if filter_state(c):
            lines.append(SET_TEXT % objc_lit(c.get("input", "")))
        cmd = call_for(i, c)
        if cmd:
            lines.append(cmd)
        lines.append(READ_STRING)
        lines.append('expr -l objc -O -- @"@@@END %d"' % i)
    return lines


def apply_cleaner_cases(cases, name_for):
    """The measured-correct path: one real cleaner per case, executed by name."""
    return case_script(cases, lambda i, c: APPLY_CLEANER % name_for(i, c))


def split_count(raw):
    """Split a `-O`-printed result into (count, text) when a count is printed.

    Kept here (not in the measurement script) because it is the one place that
    knows the layout of an lldb `-O` return value followed by the string.
    """
    if raw is None:
        return None, None
    head, sep, tail = raw.partition("\n\n")
    if head.strip().isdigit():
        return int(head.strip()), tail
    if head.strip() == "<nil>":
        return 0, tail
    return None, raw


def objc_lit(s: str) -> str:
    out = []
    for ch in s:
        o = ord(ch)
        if ch == '"':  out.append('\\"')
        elif ch == '\\': out.append('\\\\')
        elif ch == '\n': out.append('\\n')
        elif ch == '\t': out.append('\\t')
        elif ch == '\r': out.append('\\r')
        elif o < 32 or o > 126: out.append('\\u%04x' % o)
        else: out.append(ch)
    return ''.join(out)


HERE = os.path.dirname(os.path.abspath(__file__))
# Empty-but-valid library used as the starting point for sandbox runs.
# Built-in cleaners live in the binary, so an empty library is enough.
PRISTINE = os.path.join(HERE, "baseline_library.bplist")
LIB_REL = os.path.join("Library", "Application Support", "TextSoap",
                      "textsoap9.textsoap9data", "main0090.textsoap9Lib")
GROUP_CONTAINER = "PUYTPPSST8.com.unmarked.textsoap"
GROUP_DATA = os.path.expanduser(
    "~/Library/Group Containers/%s/Data" % GROUP_CONTAINER)


def chain_target(case):
    """Name of the cleaner to run for a chain/cleaner-item case."""
    if "cleaner_item" in case:
        return case["cleaner_item"]["name"]
    return case.get("name") or "ZZZ_chain_%s" % case.get("id", "x")


def find_node(**params):
    """The one definition of a serialized `find` action node.

    Parameter keys and defaults come from TSTextFindParams.dictionary read at
    runtime, cross-checked against the records the app itself writes into the
    user library:
        {find="", granularity=0, match_case=1, regex_options="", replace="", type=0}
    plus `list_name`, which only appears in records written from the UI.
    """
    base = {"find": "", "replace": "", "type": 1, "granularity": 0,
            "match_case": True, "regex_options": "", "list_name": ""}
    base.update(params)
    return {"identifier": "find", "parameters": base, "enabled": True,
            "show_children": True, "show_details": True, "height": 145.0}


def action_node(identifier, **params):
    """The one definition of a serialized non-`find` action node."""
    return {"identifier": identifier, "parameters": dict(params), "enabled": True,
            "show_children": True, "show_details": True, "height": 0.0}


def make_cleaner_item(case):
    """Build a real library cleaner record (the format the app itself writes)."""
    if "cleaner_item" in case:
        item = dict(case["cleaner_item"])
    else:
        item = {"name": chain_target(case),
                "actionList": [{"identifier": "cleaner.actions", "height": 0.0,
                                "enabled": True, "show_children": True,
                                "parameters": {}, "show_details": True}]
                + list(case["actions"])
                + [{"identifier": "endlist", "height": 0.0, "enabled": True,
                    "show_children": True, "parameters": {}, "show_details": True}]}
    u = str(uuid.uuid4()).upper()
    item.update({"min-required": "9.0", "version": 6, "revision": 0,
                 "fileVersion": 6, "uuid": u, "datatype": "cleaner",
                 "lastModified": time.time(),
                 "properties": item.get("properties",
                                        {"comment": "", "version": "", "author": "oracle"})})
    return u, item


class Sandbox:
    """A private textsoap9 instance with its own library, so the user's is untouched."""

    def __init__(self, items):
        self.home = tempfile.mkdtemp(prefix="tshome-")
        lib = os.path.join(self.home, LIB_REL)
        os.makedirs(os.path.dirname(lib), exist_ok=True)
        if os.path.exists(PRISTINE):
            # Start from the empty-but-valid pristine library so every run is comparable.
            shutil.copy(PRISTINE, lib)
            d = plistlib.load(open(lib, "rb"))
        else:
            d = {"version": 2, "contents": {"cleaners": {}, "groups": {}}}
        for uuid_, item in items:
            d["contents"]["cleaners"][uuid_] = item
        plistlib.dump(d, open(lib, "wb"))
        # The app resolves its library and its catalog through NSHomeDirectory(),
        # which IGNORES $HOME and keeps returning the real user's home. The only
        # variable that actually relocates it is CFFIXED_USER_HOME. Without this,
        # a "sandbox" run silently loads the user's real library (and can write to
        # it). Verified: NSHomeDirectory() returns the temp dir only with this set.
        # The group container must also be scaffolded or the app exits on launch.
        gc = os.path.join(self.home, "Library", "Group Containers", GROUP_CONTAINER, "Data")
        if os.path.isdir(GROUP_DATA):
            shutil.copytree(GROUP_DATA, gc, dirs_exist_ok=True)
        else:
            os.makedirs(gc, exist_ok=True)
        env = dict(os.environ, HOME=self.home, CFFIXED_USER_HOME=self.home)
        self.log = open(os.path.join(self.home, "app.log"), "wb")
        self.proc = subprocess.Popen([APP], env=env, stdout=self.log,
                                     stderr=subprocess.STDOUT,
                                     stdin=subprocess.DEVNULL)
        time.sleep(7)
        if self.proc.poll() is not None:
            raise RuntimeError("sandbox textsoap9 exited: %s" % self.log.name)

    def close(self):
        try:
            self.proc.send_signal(signal.SIGTERM)
            self.proc.wait(timeout=10)
        except Exception:
            try: self.proc.kill()
            except Exception: pass
        self.log.close()
        shutil.rmtree(self.home, ignore_errors=True)


def build_script(targets, inputs):
    L = ['expr -l objc -- id $t = (id)[(id)objc_getClass("TSText") performSelector:(SEL)sel_registerName("new")]']
    for i, (target, inp) in enumerate(zip(targets, inputs)):
        L.append('expr -l objc -O -- @"@@@IDX %d"' % i)
        L.append('expr -l objc -O -- (void)[$t performSelector:(SEL)sel_registerName("setString:") withObject:@"%s"]' % objc_lit(inp))
        L.append('expr -l objc -O -- (void)[$t performSelector:(SEL)sel_registerName("applyCleaner:") withObject:@"%s"]' % objc_lit(target))
        L.append('expr -l objc -O -- (id)[(id)[$t performSelector:(SEL)sel_registerName("string")] performSelector:(SEL)sel_registerName("stringByReplacingOccurrencesOfString:withString:") withObject:@"\\n" withObject:@"<LF>"]')
        L.append('expr -l objc -O -- @"@@@END %d"' % i)
    L += ['detach', 'quit']
    return '\n'.join(L) + '\n'


def live_pid():
    pid = subprocess.run(["pgrep", "-x", "textsoap9"], capture_output=True, text=True).stdout.strip().split("\n")[0]
    if not pid:
        sys.exit("textsoap9 is not running; launch it first")
    return pid


def run(targets, inputs, pid):
    """Run plain cleaners against the live app."""
    return run_lldb(pid, build_script(targets, inputs))


def parse(log, n):
    """Return outputs aligned to case index.

    Values are delimited by explicit @@@IDX / @@@END markers rather than by
    "next line" heuristics, so multi-line and empty values are captured exactly.
    """
    res = [None] * n
    idx = None
    lines = []
    for line in log.splitlines():
        if line.startswith("(lldb)"):
            continue
        m = re.search(r"@@@IDX (\d+)", line)
        if m:
            idx = int(m.group(1)); lines = []; continue
        m = re.search(r"@@@END (\d+)", line)
        if m:
            if idx is not None:
                # lldb brackets every value with one blank line before and after.
                if lines and lines[0] == "": lines = lines[1:]
                if lines and lines[-1] == "": lines = lines[:-1]
                res[idx] = "\n".join(lines).replace("<LF>", "\n")
            idx = None; lines = []; continue
        if idx is not None:
            # lldb prints this banner instead of a value when the object's
            # description is empty (e.g. an empty NSString). It means "".
            lines.append("" if line.strip() == "<object returned empty description>" else line)
    return res


def run_cases(cases):
    """Execute mixed cases: plain cleaners against the live app, chains in a sandbox."""
    plain = [c for c in cases if "actions" not in c and "cleaner_item" not in c]
    chain = [c for c in cases if "actions" in c or "cleaner_item" in c]
    for c in cases:
        c.pop("output", None); c.pop("error", None)

    if plain:
        log = run([c["cleaner"] for c in plain], [c["input"] for c in plain], live_pid())
        if os.environ.get("TS_ORACLE_LOG"):
            open(os.environ["TS_ORACLE_LOG"], "w").write(log)
        for c, v in zip(plain, parse(log, len(plain))):
            if v is None: c["error"] = "unparsed"
            else: c["output"] = v

    if chain:
        items, targets = [], []
        for c in chain:
            u, item = make_cleaner_item(c)
            items.append((u, item))
            targets.append(item["name"])
        sandbox = Sandbox(items)
        try:
            log = run(targets, [c["input"] for c in chain], sandbox.proc.pid)
            if os.environ.get("TS_ORACLE_LOG_CHAIN"):
                open(os.environ["TS_ORACLE_LOG_CHAIN"], "w").write(log)
            for c, v in zip(chain, parse(log, len(chain))):
                if v is None: c["error"] = "unparsed"
                else: c["output"] = v
        finally:
            sandbox.close()
    return cases


if __name__ == "__main__":
    cases = json.load(open(sys.argv[1]))
    json.dump(run_cases(cases), sys.stdout, ensure_ascii=False, indent=1)
