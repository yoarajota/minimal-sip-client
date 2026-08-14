#!/usr/bin/env python3
"""Reproduces the RFC 3261 MUST counts behind E-007 / docs/matrix.md.

Fetches RFC 3261 (rfc-editor.org), then prints:
  1. the whole-RFC occurrence count (the 590 of E-001),
  2. the whole-RFC sentence-level statement count (multi-MUST sentences once),
  3. per-cited-section raw occurrence counts (the unfiltered bound),
  4. the totals behind the three ratios in docs/matrix.md.

The judgment of which statements are *forced* (role-filtering, the primary 87)
is recorded auditable-statement-by-statement in docs/matrix.md; this script
reproduces the counts it sits on. Run from the repository root.
"""
import re
import sys
import urllib.request
from pathlib import Path

RFC = "https://www.rfc-editor.org/rfc/rfc3261.txt"
# The sections the matrix cites, deduplicated (see docs/matrix.md rows).
CITED = [
    "7.1", "8.1.1", "8.1.1.1", "8.1.1.2", "8.1.1.3", "8.1.1.4", "8.1.1.5", "8.1.1.6",
    "8.1.1.7", "8.1.1.8", "8.1.1.9", "8.1.1.10", "8.1.3.5", "10.2", "12.1.2",
    "12.2.1.1", "13.2.1", "13.2.2.4", "15.1.1", "15.1.2", "17.1.1.2", "17.1.2.2",
    "17.1.3", "18.1.1", "18.1.2", "22.2", "22.4", "19.3", "20.14",
]


def sections(text: str) -> dict:
    lines = text.replace("\f", "\n").split("\n")
    sec_re = re.compile(r"^(\d+(?:\.\d+)*)\s+(\S.*)$")
    out, current = {}, None
    for i, ln in enumerate(lines):
        stripped = ln.rstrip()
        m = sec_re.match(stripped)
        nxt = lines[i + 1].rstrip() if i + 1 < len(lines) else ""
        nxt_is_header = bool(sec_re.match(nxt)) if nxt else False
        is_header = (m and len(stripped) < 95 and "..." not in stripped
                     and (nxt == "" or nxt_is_header))
        if is_header:
            current = m.group(1)
            out[current] = []
        elif current:
            out[current].append(ln)
    return out


def main() -> int:
    cache = Path("/tmp/rfc3261-bench.txt")
    if not cache.exists():
        print(f"fetching {RFC}")
        cache.write_bytes(urllib.request.urlopen(RFC).read())  # noqa: S310
    text = cache.read_text()

    occurrences = len(re.findall(r"\bMUST\b", text))
    flat = re.sub(r"\s+", " ", text)
    start = flat.find("Abstract")
    body = flat[start:] if start > 0 else flat
    statements = sum(1 for s in re.split(r"(?<=\.)\s+", body) if re.search(r"\bMUST\b", s))

    secs = sections(text)
    per = {s: sum(ln.count("MUST") for ln in secs.get(s, [])) for s in CITED}
    cited_raw = sum(per.values())

    print(f"whole-RFC MUST occurrences:        {occurrences}")
    print(f"whole-RFC MUST statements:          {statements}")
    print(f"cited-section raw occurrences:     {cited_raw}  (unfiltered bound)")
    print()
    for s in CITED:
        print(f"  {s:>10}: {per[s]}")
    print()
    print("matrix ratios (see docs/matrix.md):")
    print(f"  primary statements   90 / {statements} = {90/statements*100:.1f}%")
    print(f"  occurrences          103 / {occurrences} = {103/occurrences*100:.1f}%")
    print(f"  unfiltered bound     {cited_raw} / {occurrences} = {cited_raw/occurrences*100:.1f}%")
    return 0


if __name__ == "__main__":
    sys.exit(main())
