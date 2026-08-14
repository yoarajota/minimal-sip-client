# Evidence ledger — Minimal SIP client: what RFC 3261 subset is sufficient to register and hold a call with a real PBX?

Every claim in this repository resolves to an entry here. An entry without a
reproduction command is a note, not evidence, and fails validation.

**Required in every entry:** a fenced command block, an `Environment:` line, and a `Result:`
line. Headings must be exactly `### E-###  —  <title>` so the tooling can parse them.

IDs are never reused. Evidence that stops reproducing is marked `Status: broken` and the
readiness level that depended on it comes down.

---

### E-001 — TODO short title

**Claim.** TODO — the specific statement this evidence supports, and where it appears
(README, `docs/01-theory.md § 5`, scenario `S-001`, …).

**Environment:** OS / CPU / RAM / runtime version / dependency versions / dataset version.
Pin everything that could move the number.

```bash
# runnable from the repository root on a clean checkout after the documented setup
TODO
```

**Result:** TODO — the observed output. For measurements, report variance across ≥ 5 runs
(min / median / max, or mean ± sd). A single run is not a measurement.

**Status:** reproducing
**Supports:** H-001, S-001, TRL 3 for `core`
**Recorded:** 2026-08-14

---

<!-- Template for further entries:

### E-002 — title

**Claim.**
**Environment:**

```bash
```

**Result:**
**Status:** reproducing | broken
**Supports:**
**Recorded:**

-->

## Benchmark methodology

Filled at P5. Applies to every entry tagged as a benchmark.

- **What is measured:** TODO
- **Runs:** TODO (≥ 5), **warm-up:** TODO
- **Held constant:** TODO
- **Baseline configuration:** TODO — the same tuning effort was spent on the baseline as on
  the concept; if not, say so, because it invalidates the comparison.
- **Known measurement bias:** TODO
