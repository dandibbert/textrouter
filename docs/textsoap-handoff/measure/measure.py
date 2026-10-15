#!/usr/bin/env python3
"""Measure new cases against the LIVE TextSoap engine (macOS only).

  python3 measure.py cases.json          # -> prints cases with "output"
  python3 measure.py --cleaners A B      # -> every corpus input x those cleaners

cases.json: [{"id":..., "cleaner":"...", "input":"..."}]  or chain form:
            [{"id":..., "actions":[...], "input":"..."}]
Chain cases run inside a private sandbox instance with its own library, so your
own TextSoap library is never modified.
"""
import json, sys, pathlib
sys.path.insert(0, str(pathlib.Path(__file__).resolve().parent))
import oracle

def main():
    if len(sys.argv) > 1 and sys.argv[1] == "--cleaners":
        corpus = json.load(open(pathlib.Path(__file__).parent.parent / "cleaners/corpus.json"))
        names = sys.argv[2:]
        cases = [{"id": f"{n}|{c['id']}", "cleaner": n, "input": c["input"]}
                 for n in names for c in corpus]
    else:
        cases = json.load(open(sys.argv[1]))
    json.dump(oracle.run_cases(cases), sys.stdout, ensure_ascii=False, indent=1)

if __name__ == "__main__":
    main()
