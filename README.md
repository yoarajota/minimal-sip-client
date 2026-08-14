# Minimal SIP client: what RFC 3261 subset is sufficient to register and hold a call with a real PBX?

> What subset of RFC 3261 is necessary and sufficient for a from-scratch SIP client to register, establish, and tear down a two-way RTP call against a mainstream PBX?

**Concept** `C-004` · **archetype** `implementation`.

A from-scratch SIP client (Go, no SIP library) that implements only the RFC 3261 subset
needed to register, establish, hold, resume and tear down a two-way RTP call against a real
PBX (the pinned Asterisk 20 configuration, per R-004) — and measures how much of the
specification that subset actually is. The measurement instrument is the message-trace matrix
(`docs/matrix.md`): every implemented behaviour traced to the RFC section that forces it,
counted against the 590 normative `MUST` occurrences in RFC 3261 (E-001).

## Claim

**H-001** — Under the scenario suite (register, two-way RTP call, hold/resume, teardown
against a pinned Asterisk 20 instance), a from-scratch SIP client requires fewer than half of
RFC 3261's normative MUST requirements to complete the same suite that PJSIP 2.17 completes in
full, at the cost of not supporting SIP features outside the suite (forking, presence,
subscription, NAT traversal, S/MIME). → *verdict: supported* (decided at P5, 2026-08-14).

The suite forces **90 of 540** normative MUST statements (16.7%) — 103/590 = 17.5% on the
occurrence unit, and 156/590 = 26.4% even counting every MUST in the cited sections without
role filtering (E-007). All three are far under the 50% falsifier line, while the full PJSIP
2.17 stack completes the same suite's protocol steps (E-008): its Python API's media does
not restart after hold in the headless container (a recorded finding, E-008), whereas the
concept client resumes media fully. The cost: 450 of 540 statements (83.3%) are not
forced — proxy behaviour (§16: 134), S/MIME (§23: 16), presence/SUBSCRIBE, PRACK, session
timers, TCP/TLS, other codecs (E-008).

*Falsified if* the smallest subset that completes the suite still implements a majority
(≥ 50%) of the normative MUST requirements — not approached under any counting rule.

## Baseline

Compared against **PJSIP 2.17** (`pjproject` tag 2.17) because it is the mainstream full SIP
stack: it completes 100% of the scenario suite by construction, so it is the honest point of
comparison for "how small can the load-bearing subset be?". The comparison is not about speed —
it is about the size of the subset that still completes the same suite.

## Public interface

The component is the Go package `internal/sip` (design decision [D-001](docs/adr/D-001-public-surface.md)):

```go
cfg := sip.Config{Server: "pbx:5060", Domain: "pbx", User: "alice", Password: "secret", RTPPort: 40000}
c, err := sip.New(cfg)
ctx := context.Background()

err = c.Register(ctx)                 // REGISTER -> 401 -> REGISTER+Authorization -> 200
call, err := c.Call(ctx, "100")       // INVITE -> 180 -> 200 (SDP) -> ACK
stats := call.MediaPhase(3*time.Second, true)  // sent/recv RTP packet counts
err = call.Hold(ctx)                  // re-INVITE(a=sendonly) -> 200 (answer recvonly)
err = call.Resume(ctx)                // re-INVITE(a=sendrecv) -> 200 (answer sendrecv)
err = call.Hangup(ctx)                // BYE -> 200
for _, e := range c.Trace() { ... }   // message-trace matrix rows
```

The client is synchronous and per-dialog: one registration, one call. It implements only UAC
behaviour — the far end is the PBX.

<!-- scorecard:start -->

### Readiness scorecard

_Auto-generated. Do not hand-edit._

| Measure | Value | Meaning |
| :--- | :--- | :--- |
| **TRL** | **6** | System prototype demonstrated |
| **SRL**  | **5** | High-risk component technology development defined — seams only |
| Composite SRL | 0.534 | standard formulation (all components, diagonal-inclusive) |
| Weakest component | client-runtime (0.4259) | lowest component-level SRL |
| Weakest seam | core<->client-runtime (IRL 4) | lowest-scoring integration pair |
| Phase | P6 | |
| Hypothesis | supported | |
| Suitable for | early-adopters | |

| Component | Role | TRL | Component SRL |
| :--- | :--- | :-: | :-: |
| `core` | concept | 6 | 0.490 |
| `asterisk` | supporting | 9 | 0.685 |
| `client-runtime` | host | 5 | 0.426 |

| Scenario | Characteristic | Priority | Status |
| :--- | :--- | :--- | :--- |
| S-001 | functional-suitability | high | pass |
| S-002 | reliability | high | pass |
| S-003 | maintainability | medium | pass |

<!-- scorecard:end -->

## Try it

Requires Docker (the suite runs against a pinned Asterisk 20 container):

```bash
# setup — nothing to install; quality tools build into .tools/ on first run
make quality

# unit suite (hermetic; no Docker needed)
make test

# the headline result: the full suite against the real PBX, one command
./run-suite.sh
```

## How it works

The client is a deliberate subset of RFC 3261 (see [docs/matrix.md](docs/matrix.md) for the
full trace). Mechanics: the six mandatory headers in every request (§8.1.1); REGISTER with
HTTP-digest auth (§10, §22.4); INVITE/offer-answer with an SDP PCMU offer (§13, RFC 4566/3264);
the transaction layer with RFC timers T1=500ms / T2=4s / T4=5s and the 64×T1 timeout (§17);
UDP transport to port 5060 (§18); RTP send/receive with the 12-byte header (§RFC 3550). Hold is
a re-INVITE whose offer is `a=sendonly`, resumed with `a=sendrecv` — there is no SIP-level hold
method. Sourced and referenced in [docs/01-theory.md](docs/01-theory.md).

## Evidence

| ID | Claim | Command | Result |
| :--- | :--- | :--- | :--- |
| E-001 | P1 literature survey: RFC 3261 subset mechanism; 590 normative `MUST` occurrences | curl the sources | 8 sources read, MUST count 590 |
| E-002 | P2 PoC completes the suite against the real PBX | `./poc/run.sh` | exit 0; media 152/152 echo |
| E-003 | Component unit suite: conformance, failure-mode, property | `make test` | exit 0; 20 tests + fuzz corpus |
| E-004 | Component integration suite against the real PBX | `./run-suite.sh` | exit 0; register/call/hold/resume/teardown |
| E-006 | Benchmark: concept suite ×5 + fault injection (PBX killed mid-call) | `./bench/run.sh` | exit 0; 5/5 runs, clean 408 on killed PBX |
| E-007 | The subset measurement: 90/540 = 16.7% (103/590, 156/590 bounds) | `python3 bench/extract-musts.py` | exit 0; ratios printed |
| E-008 | Baseline: PJSIP 2.17 completes the suite's protocol steps ×5 + cost table | `./bench/run.sh` | exit 0; 5/5 PASS; media-restart finding recorded |

Full ledger: [docs/05-evidence.md](docs/05-evidence.md).

## Limitations

### Where the baseline wins

- **Breadth.** PJSIP is a production SIP stack with forking, presence/SUBSCRIBE, PRACK,
  session timers, TCP/TLS, every codec, and twenty years of field hardening. The concept
  client is 1,100 lines that completes one suite. Anyone building a general-purpose
  communications product should use PJSIP; the only claim this repository makes is about the
  size of the load-bearing subset (E-008's cost table).
- **The client is a measurement instrument first.** A competent engineer solving the *call*
  problem with PJSIP loses nothing for using SIP (that is the substitution check, A5). What
  the incumbent cannot do — and this repo exists for — is say which parts of RFC 3261 are
  load-bearing. Judge the repository on the matrix and evidence, not on the client.

### What is untested

- **Interop is proven against exactly one PBX.** The message/dialog/auth surface has only run
  against the pinned Asterisk 20.7 container; the claim's "mainstream PBX" is that
  configuration. Widening it needs a second PBX (R-004).
- **Media is one client against Asterisk's `Echo()` application.** The RTP path is real and
  two-way (send and receive through the PBX), but there is no second real endpoint; two
  phones calling each other is untested.
- **Only UDP transport and PCMU.** No TCP/TLS, no other codecs, no NAT traversal; a PBX that
  requires TLS or a codec outside PCMU will not complete the suite (R-001, R-002).
- **The incumbent's Python API cannot restart a held call's media headless** — pjsua2's
  stream stays inactive after the resume 200 (E-008, 5/5 runs). The concept client resumes
  media fully (102/102). The re-INVITEs themselves succeed in the baseline; only media
  restart fails, in the headless container.

### What would move it up a level

- **TRL 7** needs a production-like operational environment: a second real endpoint, a load
  generator, sustained multi-call traffic, and monitoring — none of which this suite has
  (readiness.yaml boundary note, E-006).
- **IRL 6** for core↔asterisk needs a second codec, concurrent calls, and fault injection at
  the real seam; **IRL 5** for core↔client-runtime needs the SIGTERM/drain path tested.
- **The comparative measurement's boundary is the suite itself.** The 16.7% subset number
  holds for *this* scenario suite and *this* pinned configuration. A suite that added
  forking, presence, or TCP would force more MUSTs — the matrix (docs/matrix.md) is the
  instrument for re-measuring. The baseline side is a functional completion proof (E-008),
  not a code-size comparison.

### What we tried that didn't work

- The P2 PoC initially sent the INVITE without its SDP body (`Content-Length: 0`) — the call
  "succeeded" but no media flowed; the integration suite (E-004) caught it, the regression test
  `TestInviteCarriesBody` now guards it.
- A re-INVITE whose CSeq collided with the initial INVITE's was answered 491 (Request Pending)
  — the dialog CSeq must be the CSeq actually used on the wire after the auth retry.

## Documents

| Document | Contents |
| :--- | :--- |
| [docs/01-theory.md](docs/01-theory.md) | What the literature establishes, with the `SRC-###` source ledger. |
| [docs/03-log.md](docs/03-log.md) | Append-only working log: what was tried, including what failed. |
| [docs/04-tradeoffs.md](docs/04-tradeoffs.md) | ATAM-lite: drivers, scenarios, sensitivity and tradeoff points, risks. |
| [docs/05-evidence.md](docs/05-evidence.md) | Every claim, its command, environment, and observed result. |
| [docs/matrix.md](docs/matrix.md) | The message-trace matrix: which RFC 3261 sections are load-bearing. |
| [docs/adr/](docs/adr/) | Decisions and the options rejected. |
| [.sota/](.sota/) | Machine-readable readiness and quality data. |

## License

MIT — chosen deliberately: this repository exists to be benchmark material, and permissive
licensing keeps it usable as such (workflow 07 step 5).
