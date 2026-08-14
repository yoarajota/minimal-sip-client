# Evidence ledger — Minimal SIP client: what RFC 3261 subset is sufficient to register and hold a call with a real PBX?

Every claim in this repository resolves to an entry here. An entry without a
reproduction command is a note, not evidence, and fails validation.

**Required in every entry:** a fenced command block, an `Environment:` line, and a `Result:`
line. Headings must be exactly `### E-###  —  <title>` so the tooling can parse them.

IDs are never reused. Evidence that stops reproducing is marked `Status: broken` and the
readiness level that depended on it comes down.

---

### E-001 — P1 literature survey: RFC 3261 subset mechanism

**Claim.** The theory pass read the primary sources that establish how a minimal SIP client
works and what the scenario suite requires: RFC 3261 (SIP), RFC 3665 (basic call flows),
RFC 3550 (RTP), RFC 4566 (SDP), RFC 3264 (offer/answer), RFC 4475 (torture tests), PJSIP
(incumbent) and the Asterisk res_pjsip reference. Supports the H-001 threshold by counting
the RFC 3261 normative `MUST` occurrences (590) that the message-trace matrix will measure
against. Appears in `docs/01-theory.md` and scenario S-001.

**Environment:** Linux (Ubuntu 24.04), bash 5.x, curl 8.x, network access to
rfc-editor.org, pjsip.org, docs.asterisk.org.

```bash
mkdir -p /tmp/sip-sources && cd /tmp/sip-sources
for rfc in 3261 3264 3550 4566 3665 4475; do
  curl -sf -o "rfc$rfc.txt" "https://www.rfc-editor.org/rfc/rfc$rfc.txt"
done
curl -sf -o pjsip-about.html "https://www.pjsip.org/about.htm"
curl -sf -o asterisk-respjsip.html \
  "https://docs.asterisk.org/Asterisk_20_Documentation/API_Documentation/Module_Configuration/res_pjsip/"
ls -la
printf 'MUST count in RFC 3261: %s\n' "$(grep -o 'MUST' rfc3261.txt | wc -l)"
```

**Result:** 8 sources fetched and read (6 RFCs + PJSIP about page + Asterisk res_pjsip
reference), all `Access: full-text` for the sections used. `MUST count in RFC 3261: 590`.
Key extractions: six mandatory headers in every request, Contact mandatory in INVITE (§8.1.1);
REGISTER 401→Authorization→200 flow (§10, RFC 3665 §2.1); INVITE→180→200→ACK→RTP→BYE→200
flow (RFC 3665 §3.1); transaction timers T1=500ms/T2=4s/T4=5s/64×T1=32s (§17); UDP 5060 and
the 1300-byte TCP rule (§18); hold via re-INVITE `a=sendonly` (RFC 3264 §5.1); RTP header
layout (RFC 3550 §5.1).

**Status:** reproducing
**Supports:** H-001, S-001, TRL 1 for `core`
**Recorded:** 2026-08-14

---

### E-002 — P2 proof of concept: minimal client completes the suite against a real PBX

**Claim.** A from-scratch RFC 3261 client (`poc/`, ~500 lines of Go, no SIP library)
completes the scenario suite against a real Asterisk 20 PBX in containers: REGISTER with
HTTP-digest auth, two-way RTP call, hold/resume via re-INVITE, teardown. This is the
critical function from the P1 theory — a minimal subset of RFC 3261 is sufficient to
work against a mainstream PBX. Supports H-001 and scenario S-001.

**Prediction (from P1, RFC 3665 §2.1/§3.1):** the flow would need the six mandatory headers
(To, From, CSeq, Call-ID, Max-Forwards, Via) plus Contact for INVITE (§8.1.1), the
REGISTER 401→Authorization→200 exchange (§10, §22.4), INVITE→180→200→ACK, media, and
BYE→200, with the hold offer answered `recvonly` (RFC 3264 §5.1). Observed: exactly that
flow, with one addition — Asterisk challenges the **initial INVITE** with 401 (endpoint
has `auth` configured), so digest handling on INVITE is load-bearing too.

**Environment:** Docker Desktop daemon 29.6.1 (WSL2 backend), docker compose v5.3.0.
Images: `andrius/asterisk:20.7-cert11_debian-trixie` (Asterisk certified 20.7-cert11) and
`golang:1.22-alpine`; client compiled with Go 1.22.2. Client and PBX on one compose
network; Asterisk configs in `poc/asterisk/` (pjsip.conf endpoint `alice`, digest
`userpass`; extensions.conf `100` → Answer+Echo; rtp.conf ports 10000–10050).

```bash
./poc/run.sh     # from the repository root; exit 0 = suite passed
```

**Result:** exit 0, `SUITE PASSED: register -> call -> hold -> resume -> teardown against a
real PBX`. Observed message flow:

```
REGISTER -> 401 challenge -> REGISTER+Authorization -> 200 OK
INVITE -> (401 -> INVITE+Authorization) -> 180 -> 200 OK (SDP answer) -> ACK
re-INVITE(sendonly) -> 200 OK (answer recvonly) -> ACK
re-INVITE(sendrecv) -> 200 OK (answer sendrecv) -> ACK
BYE -> 200 OK
```

RTP (PCMU, 440 Hz tone, 20 ms packets): active phase sent 152 / received 152 (echo
symmetry through Asterisk `Echo()`); during hold received 0 (client sends nothing); after
resume sent 102 / received 102. The hold offer's answer was `recvonly` and the resume
offer's answer `sendrecv`, as RFC 3264 §5.1 requires.

**Status:** reproducing
**Supports:** H-001, S-001, TRL 3 for `core`
**Recorded:** 2026-08-14

---

### E-003 — Component unit suite: conformance, failure-mode and property tests

**Claim.** The `internal/sip` component passes its three test layers against the configurable
fake UAS (D-002): conformance (message parse/build incl. folding, digest RFC 2617 vectors,
SDP directions, RTP header pack), failure modes from `docs/01-theory.md § 3` (transaction
timeout → 408, dropped-message retransmission recovery, wrong-branch response ignored,
non-2xx final → transaction-layer ACK, 401 challenge on REGISTER and INVITE, rejected call),
and property/edge (parser fuzz never panics, CSeq monotonicity, INVITE carries its SDP body).
Supports S-002 (fault-tolerance) and TRL 4 for `core`.

**Environment:** Go 1.22.2 on Linux (Ubuntu 24.04), no network, no Docker required.

```bash
make test     # from the repository root; exits 0 when the suite passes
```

**Result:** exit 0. `ok github.com/yoarajota/minimal-sip-client/internal/sip` — 20 tests and a
fuzz seed corpus pass (observed 2026-08-14). Key failure-mode behaviours verified: a silent
server yields a 408-class `TransactionError`; a dropped first request is recovered by
Timer E retransmission; a foreign-branch response is ignored; a 404 to an INVITE produces the
transaction-layer ACK with the same branch and method ACK; wrong credentials surface the 401.

**Status:** reproducing
**Supports:** H-001, S-002, TRL 4 for `core`
**Recorded:** 2026-08-14

---

### E-004 — Component integration suite against the real PBX

**Claim.** The `internal/sip` component completes the scenario suite against a real, pinned
Asterisk 20 PBX in containers — the relevant environment (TRL 5): register with digest auth,
two-way RTP call, hold/resume via re-INVITE, teardown. Supports S-001 (functional
completeness) and TRL 5 for `core`.

**Environment:** Docker Desktop daemon 29.6.1 (WSL2 backend), docker compose v5.3.0. Images:
`andrius/asterisk:20.7-cert11_debian-trixie` (Asterisk certified 20.7-cert11) and
`golang:1.22-alpine`; client compiled with Go 1.22.2. Client and PBX on one compose network;
PBX config in `poc/asterisk/` (pjsip.conf endpoint `alice`, digest `userpass`; extensions.conf
`100` → Answer+Echo; rtp.conf ports 10000–10050).

```bash
./run-suite.sh     # from the repository root; exit 0 = suite passed
```

**Result:** exit 0. `--- PASS: TestSuiteIntegration`. Observed trace:

```
register: REGISTER -> 401 -> REGISTER+Authorization -> 200
invite:   INVITE -> 180 -> 200 (SDP answer) -> ACK
hold:     re-INVITE(sendonly) -> 200 (answer recvonly) -> ACK
resume:   re-INVITE(sendrecv) -> 200 (answer sendrecv) -> ACK
bye:      BYE -> 200
```

RTP (PCMU, 440 Hz tone, 20 ms packets): active phase sent 152 / received 152 (echo symmetry
through Asterisk `Echo()`); held phase received 0; resumed phase sent 102 / received 102.

**Status:** reproducing
**Supports:** H-001, S-001, TRL 5 for `core`
**Recorded:** 2026-08-14

---

### E-005 — P4 tradeoff analysis (ATAM-lite) and integration scores

**Claim.** The P4 analysis records the architecture, its drivers, 5 sensitivity points,
2 tradeoff points, 6 risks (all with states), 3 non-risks, and the five-attack adversarial
pass; the declared system (3 components, 2 seams) scores core↔asterisk IRL 5 and
core↔client-runtime IRL 4, each `rationale` naming the blocker for the next level. The G4
gate, which requires ≥ 3 sensitivity points, ≥ 1 tradeoff point, ≥ 1 integration and a
provisioning declaration for every non-concept component, passes.

**Environment:** none (document analysis against the gate's own checks).

```bash
python3 tools/sota.py validate .    # from the project root; exit 0 = G4 passes
```

**Result:** exit 0 (observed 2026-08-14). Findings and scores are in
[docs/04-tradeoffs.md](04-tradeoffs.md) and `.sota/readiness.yaml`.

**Status:** reproducing
**Supports:** G4 gate, S-001, S-002, S-003
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

Applies to E-006, E-007, E-008 (the P5 H-001 measurement).

- **What is measured:** the scenario suite *completion* — register, two-way RTP call,
  hold/resume, teardown — by the minimal client (concept) and by the PJSIP 2.17 incumbent
  (baseline), and the subset size: how many of RFC 3261's normative MUST statements the
  suite forces. Not a speed benchmark: the claim is about subset size, not performance, and
  the crossover is the 50% falsifier line, not a latency curve.
- **Runs:** 5 per leg (concept and baseline), full warm call lifecycle per run (each run is
  itself a warm-up; registration + call establishment precede measurement), 1 fault leg.
  Variance is reported per run.
- **Held constant:** pinned images (Asterisk certified 20.7-cert11, golang 1.22-alpine,
  pjproject tag 2.17), one compose network, no host ports, same five configuration values
  on both clients, same PBX dialplan (`Echo()`).
- **Baseline configuration:** PJSIP 2.17 built from source with its recommended
  configuration (configure defaults, -O2), used via its own high-level Python API
  (pjsua.py) with `no_sound`. No extra tuning was spent on the baseline; the comparison is
  about completing the same suite, so tuning asymmetry cannot distort the measured quantity.
  This asymmetry (none) is stated per workflow step 1.
- **Known measurement bias:** the media-count axis is protocol-level (RTP packets) and
  robust to machine speed; wall-time figures are indicative only (laptop under WSL2, thermal
  throttling possible and disclosed). The forced-statement judgment in docs/matrix.md is an
  audit of the implementation against the RFC text (E-007) and is bounded above by the
  unfiltered count 152/590 = 25.8% — even a reader who rejects every role-filtering call
  stays under the 50% line, so the verdict is not sensitive to the judgment.

---

### E-006 — P5 benchmark: concept suite ×5 and fault injection

**Claim.** The minimal client completes the scenario suite against the real pinned Asterisk
20 container 5 times (each run: register → two-way RTP call → hold → resume → teardown,
media counts and wall time recorded), and under fault injection — the PBX container killed
mid-call — surfaces the hangup timeout (408-class TransactionError after the 64×T1 window)
and exits cleanly. Supports S-001, S-002, TRL 6 for `core`, and R-003's mitigation.

**Environment:** containerised (compose network, no host ports):
`andrius/asterisk:20.7-cert11_debian-trixie` + `golang:1.22-alpine`; Docker Desktop 29.6.1
(WSL2), compose v5.3.0; client built with Go 1.22.2.

```bash
./bench/run.sh    # from the repository root; exit 0 = every run of every leg passed
```

**Result:** exit 0 (observed 2026-08-14). Concept leg: 5/5 runs passed, wall time 11.51 s per run
(media counts in the run log; active-phase sent/recv symmetric through Asterisk `Echo()`). Fault
leg: the PBX container was killed mid-call — media stopped, the hangup BYE timed out with a
408-class TransactionError after the 64×T1 window (32 s), clean exit. Per-run media counts and
wall times are in the run log (bench/run.sh prints them; reproduced by re-running the command).

**Status:** reproducing
**Supports:** H-001, S-001, S-002, TRL 6 for `core`
**Recorded:** 2026-08-14

---

### E-007 — P5 message-trace matrix: the subset measurement

**Claim.** The matrix (docs/matrix.md) enumerates, auditable statement-by-statement, the
normative MUST statements the implemented client forces: 90 statements (16.7%) of the 540
whole-RFC statements, 103/590 = 17.5% on the occurrence unit, and 156/590 = 26.4% even
counting every occurrence in the cited sections without role filtering. All three sit well
under the 50% falsifier line. Supports S-003 and the H-001 verdict.

**Environment:** the extraction is reproducible offline after one fetch; the forced-statement
judgment is an audit of `internal/sip` against the RFC text (both pinned).

```bash
python3 bench/extract-musts.py    # from the repository root; reproduces the counts
```

**Result:** exit 0. Output ends with the three ratios (90/540 = 16.7%, 103/590 = 17.5%,
156/590 = 26.4%) — observed 2026-08-14. The statement-by-statement enumeration with quoted
RFC text is in docs/matrix.md.

**Status:** reproducing
**Supports:** H-001, S-003, the falsifier check
**Recorded:** 2026-08-14

---

### E-008 — P5 baseline: PJSIP 2.17 completes the same suite + the cost table

**Claim.** The PJSIP 2.17 incumbent (pjsua2, the stack's own high-level Python API, built
from source tag 2.17) completes the suite's protocol steps against the same pinned Asterisk
5 times: register (200), call (CONFIRMED), two-way media (≈100 RTP packets/3 s through the
echo path, driven by a 440 Hz tone — the same tone the concept client sends), hold
(re-INVITE sendonly → 200, media stops), resume re-INVITE (sendrecv → 200), hangup. **One
finding, recorded, not tuned away:** pjsua2 cannot restart a held call's media without a
sound device — after the resume 200 the stream stays inactive and only stray packets return
(`media-restart=no`). The concept client resumes media fully (102/102, E-004/E-006), so it
completes every suite step where the incumbent's Python API in a headless container does not.
The cost clause is measured on both units: on the statement unit, 450 of 540 (83.3%)
normative MUST statements are not forced; on the occurrence unit, 487 of 590 (82.5%).
Unforced occurrences by section (from E-001's per-section survey): §7 message basics 12, §8
58, §10 33, §12 35, §13 17, §16 proxy 134, §17 36, §18 22, §19 28, §20 18, §22 19, §23 S/MIME
16, remainder (CANCEL/OPTIONS/INFO, §21 response codes, §5–6, §9, §11, §14, §15, §24–26) 59.
Plus the suite-external features the incumbent supports and the concept does not (forking,
presence/SUBSCRIBE, PRACK/100rel, session timers, TCP/TLS, non-PCMU codecs).

**Environment:** containerised (compose network): `minimal-sip-baseline:pjsua-2.17` image
built from pjproject tag 2.17 (Dockerfile in bench/baseline/), same Asterisk and network as
the concept leg.

```bash
./bench/run.sh    # baseline leg (leg 2) runs pjsua 5 times; exit 0 = all passed
```

**Result:** exit 0 (observed 2026-08-14). Baseline leg: 5/5 runs PASS —
`register=200 call=CONFIRMED media=active(rx ~101–103) hold=ok(sendonly) resume-reinvite=200
media-restart=no(headless pjsua2 limitation) bye=ok`. The media-restart finding is
reproducible across all 5 runs and documented in bench/README.md.

**Status:** reproducing
**Supports:** H-001 (baseline side), the cost clause
**Recorded:** 2026-08-14
