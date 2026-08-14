# Message-trace matrix — the load-bearing subset of RFC 3261

The measurement instrument for H-001. Every behaviour the minimal client implements is traced
to the RFC 3261 section that forces it; the sum of the normative `MUST` statements forced by
those sections is the subset size, compared against the 590-occurrence denominator (E-001).

**Counting methodology (open question § 7 of `docs/01-theory.md`):** a row counts the distinct
normative MUST statements in the referenced section(s) that the behaviour actually forces —
not every MUST occurrence in the section (many are about other roles, e.g. proxy behaviour in
§16). The unit is decided here, at P3, while each behaviour is implemented, so the count is
never retrofitted at P5. The `MUSTs forced` column is filled in as `internal/sip` lands; rows
are the PoC's observed behaviours (E-002) plus what the suite demands.

| # | Behaviour (trace step) | RFC 3261 section forcing it | MUSTs forced | Evidence |
| :--- | :--- | :--- | :--- | :--- |
| 1 | REGISTER construction: Request-URI = domain (no userinfo), To = AOR, From + tag, Call-ID, CSeq, Contact binding, Expires | §10.2, §8.1.1.1–8.1.1.8 | (counted at P3) | E-002 |
| 2 | Six mandatory headers in every request (To, From, CSeq, Call-ID, Max-Forwards=70, Via+branch) | §8.1.1 | (counted at P3) | E-002 |
| 3 | Digest authentication: parse WWW-Authenticate, compute response, Authorization on retry | §22.4 | (counted at P3) | E-002 |
| 4 | 401/407 handling on any request (REGISTER *and* INVITE — Asterisk challenges the initial INVITE) | §8.1.3.5, §22.2 | (counted at P3) | E-002 |
| 5 | INVITE with SDP offer (Content-Type/Content-Length, Contact mandatory) | §13.2.1, §8.1.1.8–8.1.1.10 | (counted at P3) | E-002 |
| 6 | Provisional response handling (1xx stops INVITE retransmission) | §17.1.1.2 | (counted at P3) | E-002 |
| 7 | 2xx → dialog confirmed: To tag captured, remote target from Contact, ACK by UAC core (same CSeq, method ACK) | §12.1.2, §13.2.2.4 | (counted at P3) | E-002 |
| 8 | Non-2xx final → transaction-layer ACK (implemented: verified for 401; full for all 300–699) | §17.1.1.2 | (counted at P3) | E-002 |
| 9 | Transaction timers: T1=500ms doubling (INVITE), Timer E cap T2=4s (non-INVITE), 64×T1=32s timeout, response matching by branch+CSeq | §17.1.1.2, §17.1.2.2, §17.1.3 | (counted at P3) | E-002 |
| 10 | In-dialog request construction: To tag, From tag, CSeq increment, Request-URI = remote target (Contact of 2xx) | §12.2.1.1 | (counted at P3) | E-002 |
| 11 | Hold: re-INVITE offer `a=sendonly`; Resume: re-INVITE `a=sendrecv`; answer direction rules | §13.2.1 (offer/answer), RFC 3264 §5.1 | (counted at P3) | E-002 |
| 12 | BYE: in-dialog request, session ends when BYE passed to transaction; 2xx to BYE | §15.1.1, §15.1.2 | (counted at P3) | E-002 |
| 13 | Transport: UDP framing, Via sent-by, responses to source address/port | §18.1.1, §18.1.2 | (counted at P3) | E-002 |
| 14 | SDP offer/answer (RFC 4566): v/o/s/c/t/m lines, rtpmap, direction attributes | §13.2.1 (via RFC 4566, RFC 3264) | n/a (non-RFC-3261) | E-002 |
| 15 | RTP send/receive (RFC 3550): 12-byte header, SSRC/seq/timestamp, PT 0 PCMU | §13.2.1 (media via RFC 3550) | n/a (non-RFC-3261) | E-002 |

## Deliberately outside the matrix

Behaviours the suite does *not* force, per `docs/01-theory.md § 2` — their absence is the
claim, not a gap:

- **Proxy behaviour (§16):** the client talks directly to the PBX; 134 of the 590 MUST
  occurrences live in §16 and none are load-bearing here.
- **S/MIME (§23):** 16 occurrences; the suite uses digest auth, not S/MIME.
- **Full parser robustness (RFC 4475):** the PBX sends well-formed messages; RFC 4475
  robustness is scoped out (§ 7 open question).
- **Presence/SUBSCRIBE/NOTIFY, session timers (RFC 4028), PRACK (100rel), ICE/STUN/TURN,
  forking, multi-call concurrency.**

## Status

Rows 1–13, 15 were exercised end-to-end by the P2 PoC (E-002). The `MUSTs forced` enumeration
per row is completed during the P3 `internal/sip` implementation (D-001), where each behaviour
is written against its forcing section and the distinct normative MUST statements are counted.
The final H-001 verdict (supported / falsified) is decided at P5 against the 50% line.
