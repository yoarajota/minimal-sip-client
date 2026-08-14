# Benchmark — the H-001 measurement (P5)

One command: `./bench/run.sh`. Three legs, exit 0 only when every run of every leg passes:

| Leg | What it measures | Runs |
| :--- | :--- | :--- |
| concept | The minimal client completes the scenario suite against the real pinned Asterisk 20 (register, two-way RTP call, hold/resume, teardown); media counts and wall time per run | 5 (default; `RUNS=N` to change) |
| baseline | The PJSIP 2.17 incumbent (pjsua2, the incumbent's own high-level Python API, built from source tag 2.17) completes the same suite's protocol steps against the same Asterisk: register, call, two-way media, hold, resume re-INVITE, hangup. **Finding:** its media does not restart after hold in the headless container (`media-restart=no`) — recorded, not tuned away | 5 |
| fault | The Asterisk container is killed mid-call; the client must surface the BYE timeout (408) and exit cleanly — the R-003 mitigation | 1 |

## Environment declaration (G5, recorded 2026-08-14)

- **Runtime:** Docker Desktop 29.6.1 daemon, WSL2 backend; docker compose v5.3.0.
- **Images (all version-pinned):** `andrius/asterisk:20.7-cert11_debian-trixie` (the PBX),
  `golang:1.22-alpine` (the client), `minimal-sip-baseline:pjsua-2.17` (the baseline, built
  from `pjproject` tag 2.17 — see `baseline/Dockerfile`).
- **Topology:** client and PBX on one compose network, no host ports published (the whole
  suite is containerised per R10; nothing is installed on the host).
- **Machine:** the host is a laptop (Ubuntu 24.04 on WSL2) — thermal throttling under load is
  possible and disclosed; media counts are protocol-level (packets), not timing-sensitive, so
  the measured quantity is robust to machine speed. Wall-time figures are indicative only.
- **Nothing else ran on the host during measurement.**

## Baseline tuning (workflow step 1) — disclosed asymmetry

The baseline is PJSIP 2.17 built from source with its recommended configuration
(`./configure` defaults, `-O2`), used through its own high-level Python API (pjsua2) with the
null sound device and the codec surface pinned to PCMU (the same single codec the concept
client uses). **No extra effort was spent tuning the baseline beyond this**, and none is
possible to withhold on the concept side either — the comparison is about completing the
suite, not about speed, so tuning asymmetry does not distort the measured quantity.

**One finding, recorded rather than tuned away (E-008):** pjsua2 cannot restart a held call's
media without a sound device — after the resume re-INVITE's 200 the stream stays inactive
(`media-restart=no` in every baseline run). The re-INVITEs themselves succeed (hold sendonly
200, resume sendrecv 200). The concept client resumes media fully (102/102). This is a
headless-container limitation of the incumbent's Python API, reproducible across all 5 runs;
the benchmark reports it rather than hiding it under a looser check.

## The measured quantity and its crossover

The benchmark does **not** measure speed (the claim is not about performance). It measures
completion: can the minimal client complete the suite that the full stack completes? The
interesting number is the subset size (90 of 540 normative MUST statements — `docs/matrix.md`,
E-007), and the crossover is the 50% falsifier line, which is not approached (16.7% under the
primary unit). The absence of a measured crossover is stated: the concept's cost is not
performance but feature surface, quantified in E-008's cost table.

## Cost clause (measured)

H-001's cost is "not supporting SIP features outside the suite". E-008 quantifies the
unforced surface: 450 of 540 normative MUST statements (83.3%) are not forced, by section —
including the 134 proxy MUSTs of §16, the 16 S/MIME MUSTs of §23, the 12 message-basics MUSTs
of §7, and the forking,
presence/SUBSCRIBE, PRACK, session-timer, TCP/TLS and codec surfaces. The cost is paid in
scope, not in speed or reliability on the suite.
