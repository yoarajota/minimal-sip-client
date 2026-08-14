# D-002 — Test strategy: configurable fake UAS for unit tests, real Asterisk for integration

**Status:** accepted
**Date:** 2026-08-14
**Phase:** P3

## Context

G3 requires three test layers (conformance, failure-mode, property) and the P1 failure-modes
table (§3) lists faults a real PBX will not reliably produce on demand: silent servers
(timeouts), dropped messages (retransmission), wrong-branch responses, malformed input. The
scenario suite's real environment is an Asterisk container (S-001), but a unit test cannot
depend on Docker being present or on the PBX misbehaving deterministically.

## Decision

Two test tiers, both recorded as evidence:

1. **Unit tier** — a configurable fake UAS (`internal/sip/fake_test.go`) that answers
   REGISTER/INVITE/BYE the way a real PBX does (401 digest challenge, 200 with SDP answer and
   To tag) and can be told to drop the first N requests, stay silent, or answer with a foreign
   branch. It drives the failure-mode tests (`transaction_test.go`: timeout→408, retransmission
   recovery, wrong-branch ignored, non-2xx ACK generation) and the property tests (parser fuzz,
   CSeq monotonicity, CSeq/body regression).
2. **Integration tier** — `internal/sip/suite_test.go` (build tag `integration`) runs the full
   scenario suite against the pinned Asterisk container (`docker-compose.yml`, `run-suite.sh`).
   It is the S-001 verification and catches what the fake cannot: real digest behaviour, real
   SDP negotiation, real RTP echo.

The fake is deliberately NOT a conformance oracle for message syntax — RFC 4475 robustness is
scoped out (§ 7 of `docs/01-theory.md`); the fake exists to exercise the client's fault
handling, not to torture-test its parser.

## Options rejected

| Option | Why rejected |
| :--- | :--- |
| Unit-test only against the fake, integration only against the PoC | The PoC (poc/) is throwaway; the component's integration behaviour would go unmeasured. The suite test exists because the component must prove the same suite the PoC proved (S-001). |
| Integration tests always (no build tag) | `go test ./...` would then require Docker for every unit run. The `integration` build tag keeps the unit tier hermetic and the integration tier explicit. |
| A shared in-memory SIP stack as the fake | Building a second SIP implementation to test the first tests nothing — the fake is a scripted responder, not an implementation. |

## Tradeoff

The fake is a fourth kind of component (a test peer) that must be kept honest — if it drifts
towards "anything goes", the failure-mode tests stop proving anything about the client. The
integration tier is the guard against that drift: every behaviour the fake scripted is also
exercised against the real PBX in the suite.

## Complexity justification

- **Adds:** the fake UAS (~150 lines, test-only) and the compose integration stack (the
  pinned Asterisk image, the client container).
- **Justified by:** S-002 (fault-injection tests need a peer that can misbehave on demand),
  S-001 (the suite needs the real PBX).
- **Ledger entry:** recorded in `.sota/quality-gates.yaml` → `complexity_ledger`.

## Consequences

`make test` is hermetic and fast; `make suite` (or `./run-suite.sh`) is the real-PBX gate that
G3 and later P5 evidence rely on. Both are recorded in `docs/05-evidence.md` (E-003, E-004).
