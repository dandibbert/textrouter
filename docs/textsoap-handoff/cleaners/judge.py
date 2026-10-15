#!/usr/bin/env python3
"""Judge a candidate implementation against the live app's captured behaviour.

  python3 cleaners/judge.py                     # pretty report
  python3 cleaners/judge.py --json              # machine-readable
  python3 cleaners/judge.py --cleaner "Scrub"   # one cleaner
  python3 cleaners/judge.py --show 5            # print first N mismatches per cleaner

Reads cleaners/behaviour_truth.json (produced by measure/run_oracle.py, which drives the
real TextSoap engine over lldb) and reference_implementation.py, and reports per-cleaner
agreement. A cleaner with no implementation is reported as `unimplemented`,
never as a pass, so partial coverage cannot masquerade as equivalence.
"""
from __future__ import annotations

import argparse
import json
import pathlib
import sys
from collections import defaultdict

HERE = pathlib.Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

import reference_implementation as reference  # noqa: E402


def load():
    truth = json.load(open(HERE / "behaviour_truth.json"))
    corpus = {c["id"]: c["input"] for c in json.load(open(HERE / "corpus.json"))}
    return truth, corpus


def judge(truth, corpus, only=None, show=0):
    per = defaultdict(lambda: {"pass": 0, "fail": 0, "unimpl": 0, "mismatch": []})
    for row in truth:
        cid, case = row["id"].split("|", 1)
        if only and cid != only:
            continue
        rec = per[cid]
        if cid not in reference.IMPLEMENTED:
            rec["unimpl"] += 1
            continue
        if "output" not in row:
            rec["fail"] += 1
            continue
        try:
            got = reference.clean(cid, corpus[case])
        except NotImplementedError:
            rec["unimpl"] += 1
            continue
        if got == row["output"]:
            rec["pass"] += 1
        else:
            rec["fail"] += 1
            if len(rec["mismatch"]) < max(show, 3):
                rec["mismatch"].append((case, row["output"], got))
    return per


def main():
    ap = argparse.ArgumentParser()
    ap.add_argument("--json", action="store_true")
    ap.add_argument("--cleaner")
    ap.add_argument("--show", type=int, default=0)
    a = ap.parse_args()

    truth, corpus = load()
    per = judge(truth, corpus, a.cleaner, a.show)

    if a.json:
        json.dump({k: {kk: vv for kk, vv in v.items() if kk != "mismatch"}
                   for k, v in per.items()}, sys.stdout, indent=1)
        return 0

    tot = defaultdict(int)
    print(f"{'cleaner':34} {'pass':>5} {'fail':>5} {'unimpl':>7}   status")
    print("-" * 74)
    for name in sorted(per):
        r = per[name]
        for k in ("pass", "fail", "unimpl"):
            tot[k] += r[k]
        n = r["pass"] + r["fail"]
        if r["unimpl"] and not n:
            status = "UNIMPLEMENTED"
        elif r["fail"] == 0 and n:
            status = "OK"
        else:
            status = f"{100*r['pass']/n:.0f}%"
        print(f"{name:34} {r['pass']:>5} {r['fail']:>5} {r['unimpl']:>7}   {status}")
        if a.show:
            for case, want, got in r["mismatch"][:a.show]:
                print(f"      [{case}]")
                print(f"        app: {want!r}")
                print(f"        ref: {got!r}")
    print("-" * 74)
    n = tot["pass"] + tot["fail"]
    print(f"{'TOTAL':34} {tot['pass']:>5} {tot['fail']:>5} {tot['unimpl']:>7}"
          f"   {100*tot['pass']/n:.1f}% of comparable cases" if n else "no comparable cases")
    comparable = sum(r["pass"] + r["fail"] for r in per.values())
    print(f"\ncleaners judged:    {sum(1 for r in per.values() if r['pass']+r['fail'])}")
    print(f"cleaners unimplemented: {sum(1 for r in per.values() if r['unimpl'] and not r['pass']+r['fail'])}")
    print(f"comparable cases:   {comparable}")
    return 0 if tot["fail"] == 0 else 1


if __name__ == "__main__":
    sys.exit(main())
