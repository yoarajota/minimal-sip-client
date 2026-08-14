# Working log — Minimal SIP client: what RFC 3261 subset is sufficient to register and hold a call with a real PBX?

Append-only. **Never edit an existing entry**; add a new one that supersedes it. This is the
only artefact in the repository that records work *in progress*, and it is deliberately narrow.

Open entries are printed as handoff state to the next session, so this file is how one session
tells the next what was in flight. That is its primary job — write to it before you run out of
context, not after.

## What belongs here — and what does not

Most work-in-progress knowledge already has a home. Use the table before writing an entry:

| What you have | Where it goes |
| :--- | :--- |
| A measurement, even a disappointing one — it has a command and a result | `docs/05-evidence.md` as an `E-###` with the claim it refuted |
| A choice between real alternatives | `docs/adr/D-###` — the rejected options table exists for this |
| A bound on when the concept degrades | `README.md § Limitations` |
| A second concept worth its own repository | the shared backlog |
| **No decision made and no reproducible command** — a surprise, a blind alley, an abandoned attempt | **here** |
| **Work half-finished right now** | **here, as `Disposition: open`** |

If it fits a row above this file, put it there. A log entry that should have been an ADR
weakens both.

Throwaway scripts and intermediate data are not log entries — those belong in a temp directory,
not the repository. This file records findings, not files.

## Entry format

Headings must be exactly `### L-###  —  YYYY-MM-DD  —  <short title>`, and every entry needs a
`Disposition:` line. Both are parsed by the gate checks.

| Disposition | Means |
| :--- | :--- |
| `open` | Still in flight. Printed by `next` as handoff state. **Blocks the phase gate.** |
| `dead-end` | Tried, did not work, deliberately not pursued. Terminal and legitimate. |
| `promoted: D-###` | Became an architecture decision. |
| `promoted: E-###` | Became evidence. |
| `promoted: README` | Became a stated limitation. |

Every entry must reach a terminal disposition before its phase gate passes. That is the rule
that stops this file becoming a pile — and unlike most promotion rules, it is checked.

---

<!-- No entries yet. Open entries are reported as in-flight work, so this file
     deliberately starts empty — a placeholder entry here would report fake state on day one.

     Copy this shape for the first real entry:

### L-001 — 2026-08-14 — short title

**Context:** what you were doing when this came up.
**Found:** what actually happened, specifically enough that nobody repeats it. Include the
version, flag, or condition that mattered.
**Disposition:** open | dead-end | promoted: D-### | promoted: E-### | promoted: README

-->
