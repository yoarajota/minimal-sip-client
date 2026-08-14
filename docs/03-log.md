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

### L-001 — 2026-08-14 — P3 component phase in flight

**Context:** G2 passed (poc/ completes the suite against Asterisk 20, E-002, exit 0).
Advanced to P3; the public surface is decided (D-001) and the message-trace matrix instrument
is scaffolded (docs/matrix.md). The component rewrite itself is not started.

**Found:** the P2 PoC run surfaced two load-bearing behaviours the P1 theory did not predict:
(1) Asterisk challenges the **initial INVITE** with 401 (endpoint has `auth` configured), so
digest handling is required on INVITE, not only REGISTER — matrix row 4; (2) in-dialog
requests must send the Request-URI as the bare Contact URI and the To tag outside the angle
brackets — two PoC bugs (bareURI extraction, tag placement) that cost two debugging rounds
and are fixed in poc/.

**Disposition:** promoted: D-001, D-002, D-003, E-003, E-004
