#!/usr/bin/env python3
"""Re-capture behaviour truth by running the corpus through the LIVE app (macOS).

Writes measure/oracle_truth.json -- the shipped cleaners/behaviour_truth.json
is never overwritten, so a failed run cannot silently corrupt your test
suite. Compare the two before replacing it.
"""
import json, sys, pathlib
HERE = pathlib.Path(__file__).resolve().parent
PACK = HERE.parent
sys.path.insert(0, str(HERE))
import oracle  # the differential judge

cleaners = json.load(open(PACK/"cleaners/acceptance_cleaners.json"))
corpus   = json.load(open(PACK/"cleaners/corpus.json"))
cases = [{"id": f"{c}|{k['id']}", "cleaner": c, "input": k["input"]}
         for c in cleaners for k in corpus]
print("cases:", len(cases), file=sys.stderr)
CH = 120
truth = []
for i in range(0, len(cases), CH):
    chunk = cases[i:i+CH]
    log = oracle.run([c["cleaner"] for c in chunk], [c["input"] for c in chunk], oracle.live_pid())
    vals = oracle.parse(log, len(chunk))
    for c, v in zip(chunk, vals):
        c.pop("input", None)
        if v is None: c["error"] = "unparsed"
        else: c["output"] = v
    truth += chunk
    print(f"  {i+len(chunk)}/{len(cases)}", file=sys.stderr)
json.dump(truth, open(HERE/"oracle_truth.json","w"), ensure_ascii=False, indent=1)
ok = sum(1 for t in truth if "output" in t)
print(f"captured {ok}/{len(truth)}", file=sys.stderr)
