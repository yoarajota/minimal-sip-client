# D-001 — Public surface of the minimal client: a synchronous per-dialog UAC

**Status:** accepted
**Date:** 2026-08-14
**Phase:** P3

## Context

The P2 proof of concept (`poc/main.go`) is a flat ~500-line program: one file, hardcoded
flow, no interfaces, no error handling beyond fatal prints. G3 requires a real implementation
someone else could use — interfaces, validated inputs, handled errors, tests — and the
hypothesis needs a measurement instrument: the message-trace matrix that maps each implemented
behaviour to the RFC 3261 section forcing it (S-003). The design must stay *minimal*: the
concept is about how small the load-bearing subset is, so the surface must not grow beyond
what the scenario suite (S-001) forces.

## Decision

The component is a Go package `internal/sip` exposing one synchronous `Client` that runs the
scenario suite's UAC behaviours, recording a trace as it goes:

```go
package sip

// Config is the minimal set of facts a client needs to operate.
type Config struct {
    Server   string // PBX outbound address, "host:port"
    Domain   string // AOR domain used in To/From and the REGISTER Request-URI
    User     string // AOR user part
    Password string // digest password
    RTPPort  int    // local RTP port, advertised in the SDP offer
}

func New(cfg Config) (*Client, error)

// Register performs REGISTER -> (401 challenge) -> REGISTER+Authorization -> 200.
func (c *Client) Register(ctx context.Context) error

// Call establishes a two-way RTP call to target and returns the live call.
func (c *Client) Call(ctx context.Context, target string) (*Call, error)

type Call struct { /* dialog state, RTP stream, trace */ }

func (c *Call) Hold(ctx context.Context) error   // re-INVITE with a=sendonly
func (c *Call) Resume(ctx context.Context) error // re-INVITE with a=sendrecv
func (c *Call) Hangup(ctx context.Context) error // BYE -> 200
func (c *Call) Media() (sent, recv int)          // RTP packet counts
func (c *Call) Trace() []TraceEntry              // message-trace matrix rows
```

`TraceEntry` is `{Step, Method, RFCRefs []string, Detail string}` — one row per observed
behaviour, the raw material for the matrix in `docs/matrix.md`.

Internal structure mirrors the RFC, not the PoC: `message.go` (parse/build, §7/§8),
`transaction.go` (timers, §17), `digest.go` (§22.4), `sdp.go` (RFC 4566/3264),
`rtp.go` (RFC 3550), `dialog.go` (§12/§13/§15). Each file carries the section that forces it
into the trace.

## Options rejected

| Option | Why rejected |
| :--- | :--- |
| Keep the PoC's flat single-file structure | The PoC is deliberately throwaway (workflow 04 § PoC); G3 needs interfaces, validated inputs, and three test layers. Extending the PoC would leak its no-error-handling posture into the real implementation. |
| Bind an existing stack (PJSIP via cgo, gosip, opensips) | The concept *is* "a from-scratch client's load-bearing subset". A library would make the measurement meaningless — the subset would be the library's, not the client's (R4: no strawman, and this is the inverse — no borrowed machinery either). |
| Event-driven async engine (PJSIP-style reactor) | The scenario suite is strictly sequential (S-001: one call at a time). A synchronous per-dialog client is smaller, and the trace order maps 1:1 to the message flow — the matrix is easier to audit. An async engine would be unpaid complexity (R5). |
| Implement UAS behaviour too (answer inbound INVITEs) | One question (P8): the suite's far end is the PBX. A second endpoint (two UAs calling each other) is a second scenario for a different question — parked in the backlog if ever wanted. |

## Tradeoff

Synchronous, single-dialog design trades concurrency (multiple simultaneous calls, ISO/IEC
25010 `flexibility`) for analyzability and simplicity — which are the concept's point. A reader
auditing the matrix sees exactly one linear flow per call. Linked: `TP-###` if recorded at P4.

## Complexity justification

- **Adds:** the `internal/sip` package itself, the trace recorder, and the per-file structure
  mirroring RFC sections.
- **Justified by:** S-001 (the suite requires register/call/hold/resume/teardown), S-003 (the
  trace matrix is the analyzability evidence), and the concept's own falsifier (the matrix is
  the measurement).
- **Ledger entry:** to be recorded in `.sota/quality-gates.yaml` → `complexity_ledger` when the
  package lands.

## Consequences

The README's "How it works" and "Try it" sections can now document a real public interface
(G3 requirement). Tests at three layers (conformance / failure-mode / property) target the
package boundaries, and each P1 failure-mode row maps to a test. The trace's `RFCRefs` are
where the MUST enumeration for the matrix happens — counted during implementation, recorded in
`docs/matrix.md`, decided at P5 against the 590-statement denominator from E-001.
