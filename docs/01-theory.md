# Theory — Minimal SIP client: what RFC 3261 subset is sufficient to register and hold a call with a real PBX?

Produced at P1. Establishes what is already known, from primary sources only. Every source
below was fetched and read; nothing is cited from memory.

## 1. The mechanism

A SIP client (UAC) is, mechanically, the smallest agent that can run the four procedures the
scenario suite demands, each forced by a specific RFC 3261 section:

1. **Registration** (§10.2). A REGISTER request to the registrar's domain: Request-URI with no
   userinfo, To = address-of-record, From = same AOR with a fresh `tag`, Call-ID, CSeq
   (incremented per REGISTER with the same Call-ID), Contact binding with an expiry, plus the
   six headers mandatory in *every* request (§8.1.1): To, From, CSeq, Call-ID, Max-Forwards
   (value 70), Via (with a unique `branch`). A registrar that requires authentication answers
   401 with `WWW-Authenticate: Digest …`; the client re-sends REGISTER with an `Authorization`
   header computed from the nonce (§22.4). A 200 OK confirms the binding (§10.2.1, RFC 3665
   §2.1). Bindings are soft state: the client must refresh before expiry (§10.2.4).

2. **Call establishment** (§13.1, §8.1.1.8). An INVITE carries the six mandatory headers plus
   Contact and an SDP offer (`Content-Type: application/sdp`). The offer/answer model (§13.2.1,
   RFC 3264) puts the offer in the INVITE and the answer in the final 2xx (or the offer in 2xx
   and the answer in the ACK — a compliant UAC must support both exchanges). Provisional 1xx
   responses (180 Ringing) precede the final response. For a 2xx, the **UAC core** generates the
   ACK (same CSeq number, method ACK) and the dialog is confirmed; the ACK is passed directly to
   the transport, and re-sent on 2xx retransmissions (§13.2.2.4). For a non-2xx final response,
   the **transaction layer** generates the ACK and the dialog never starts (§17.1.1.2).

3. **Hold / resume** (RFC 3264 §5.1). A re-INVITE inside the dialog whose SDP marks the audio
   stream `a=sendonly` (hold) or `a=sendrecv` (resume). Direction attributes are the entire
   mechanism — no SIP-level hold method exists; hold is a media-direction negotiation.

4. **Teardown** (§15.1). A BYE built as any in-dialog request, sent through a new non-INVITE
   client transaction; media stops when the BYE is passed to the transaction. A BYE for an
   unknown dialog is answered 481; a BYE without the dialog tags is rejected. The UAS answers
   200 and the session is gone.

The **transaction layer** (§17) is the reliability machinery. Defaults: T1 = 500 ms (RTT
estimate), T2 = 4 s, T4 = 5 s, and the terminal timeout is 64×T1 = 32 s. INVITE transactions
retransmit on Timer A (T1, doubling) until a 1xx stops retransmission, and time out on Timer B
(64×T1). Non-INVITE transactions (REGISTER, BYE, ACK-for-non-2xx) retransmit on Timer E (T1,
doubling, capped at T2) and time out on Timer F (64×T1). Responses are matched to transactions
by the top-Via `branch` parameter plus the CSeq method (§17.1.3). A transaction timeout is
treated as 408; a transport failure as 503 (§8.1.3.1).

**Transport** (§18.1.1): UDP to port 5060 by default; a request within 200 bytes of the path
MTU or larger than 1300 bytes (unknown MTU) MUST go over a congestion-controlled transport such
as TCP. The Via `sent-by` names where responses go; responses return to the request's source
address and port.

**Media** (RFC 3550, RFC 4566): RTP fixed header is V=2, P, X, CC, M, 7-bit payload type,
16-bit sequence number, 32-bit timestamp, 32-bit SSRC (§5.1). The SDP description carries the
session (v, o, s, c, t) and media (m=audio <port> RTP/AVP <fmt>, a=rtpmap) lines; PCMU is static
payload type 0, so no rtpmap is strictly required for it. The RTP/AVP profile (RFC 3551) assigns
PT 0 to PCMU.

## 2. Conditions under which it holds

- The far end is a **mainstream, well-behaved PBX** — pinned as Asterisk 20 with the `chan_pjsip`
  driver (`res_pjsip`, configured in `pjsip.conf` with `[transport]`, `[endpoint]`, `[auth]`,
  `[aor]` objects; digest `auth_type` `password`/`md5_cred`) [SRC-008]. It sends standard,
  well-formed SIP, standard SDP, standard digest challenges.
- The client talks **directly to the PBX** — no proxy chains, no forking, no DNS SRV discovery
  beyond a configured outbound address, no presence/SUBSCRIBE/NOTIFY, no session timers
  (RFC 4028), no PRACK (100rel), no ICE/STUN/TURN, no S/MIME, no TLS beyond what the PBX
  requires for transport.
- **One media stream, one codec** (PCMU). RTCP is not required to hold a call in practice,
  though RFC 3550 specifies it for sendonly/recvonly/inactive streams too (RFC 3264 §5.1);
  whether Asterisk enforces RTCP is an open question (§ 7).
- The claim is about the **load-bearing subset**: the set of RFC 3261 normative MUST statements
  the scenario suite actually forces the client to implement, traced by the message-trace
  matrix. It is not a claim about robustness against hostile input (RFC 4475), performance
  parity with PJSIP, or feature completeness.

## Known failure modes

| Failure mode | Trigger condition | Source |
| :--- | :--- | :--- |
| Authentication loop or rejected registration | Registrar requires digest auth; nonce/stale/qop handling wrong; Authorization built from the wrong URI | [SRC-002] §2.1, [SRC-001] §22.4 |
| Registration silently expires | Binding soft state; client never refreshes before the registrar's chosen expiry | [SRC-001] §10.2.4, §10.3 |
| Call never rings out (transaction timeout) | No final response within 64×T1; client must treat as 408 and not leave the call half-open | [SRC-001] §17.1.1.2, §8.1.3.1 |
| Duplicate-response storm / ACK not quenched | `branch` or CSeq matching wrong; a retransmitted 2xx re-triggers dialog logic instead of an ACK | [SRC-001] §17.1.3, §13.2.2.4 |
| One-way or no media | RTP source address/port differs from the SDP `c=`/`m=` answer (NAT, or port asymmetry); PBX drops it | [SRC-001] §18.1.1, [SRC-008] |
| BYE rejected 481 | BYE sent without the dialog's To tag, or to a dialog that never confirmed | [SRC-001] §15.1.2 |
| Hold glare | Simultaneous re-INVITEs (both sides hold at once); offer/answer rules limit who may offer | [SRC-005] §5.1 |
| Parser fragility | The suite's messages are well-formed, but any real PBX sends optional headers and folding; over-strict parsing rejects valid messages, over-loose parsing mis-routes | [SRC-006] |
| `Content-Length`/framing mismatch | UDP datagram framing vs. the declared Content-Length disagree; message boundary lost | [SRC-001] §7.5, §18.3 |

## 4. The incumbent

**PJSIP 2.17** (pinned; `pjproject` tag 2.17, released 2026-04-22). A full multimedia stack
implementing SIP, SDP, RTP, STUN, TURN, and ICE, portable from 20 MHz embedded MIPS to
desktops; Teluu's own materials put a voice-call application "from as little as 150 KB" using
the lower-level libraries [SRC-007]. It is the honest baseline because it is the mainstream
answer to "make a SIP client work", and it completes 100% of the scenario suite by construction
— it implements the entire RFC 3261 surface.

Where it falls short as an *answer to this question*: because it implements everything, it
cannot show which parts are load-bearing. The interesting result is not "a minimal client
works" — it is the size of the minimal subset that still completes the same suite. That requires
the full stack as the point of comparison, which is exactly what PJSIP provides.

**Fairness / tuning note for P5.** The comparison is not about speed — it is "smaller
subset, same suite". PJSIP will be configured as minimally as the concept: one endpoint, one
codec
(PCMU), no ICE, no presence, no TLS unless the suite requires it, and it is pinned via the
Go-free `make`/container workflow the repository documents. The same tuning effort goes into
both sides. Asterisk 20 is the single PBX for both, pinned by container image.

## 5. Hypothesis

**H-001** — Under the scenario suite (register, two-way RTP call, hold/resume, teardown against
a pinned Asterisk 20 instance), a from-scratch SIP client requires fewer than half of RFC 3261's
normative MUST requirements to complete the same suite that PJSIP 2.17 completes in full, at the
cost of not supporting SIP features outside the suite (forking, presence, subscription, NAT
traversal, S/MIME).

*Falsified if:* the smallest subset that completes the suite still implements a majority (≥ 50%)
of RFC 3261's normative MUST requirements — then a "minimal" client is not meaningfully smaller
than a full stack, and SIP's complexity is intrinsic rather than incidental.

*Measured by:* the message-trace matrix, built during P2–P3. Every behaviour the client
implements is traced to the exact RFC 3261 section that forces it; each mapped section's
normative MUST statements are counted against the denominator established below. The raw count
of `MUST` occurrences in RFC 3261 is **590**; the count is concentrated in §16 proxy behaviour
(134), §8 UAC/UAS (75), §17 transactions (69), §12 dialogs (56), §10 registration (38), §19 URI
(29), §13 INVITE (27), §18 transport (26), §22 authentication (23), §20 headers (20), §7
messages (16), §23 S/MIME (16) [SRC-001, counted from the RFC text]. A client never exercises
the §16 (proxy) and §23 (S/MIME) MUSTs, so the load-bearing share of the *client-relevant*
normative body is what the matrix will measure, and the falsifier's 50% line is drawn against
the full 590.

## 6. Prior implementations

| Implementation | Maturity | What it does differently |
| :--- | :--- | :--- |
| PJSIP (pjproject 2.17) | production, the baseline | full SIP+SDP+RTP+STUN/TURN/ICE surface; cannot show which parts are load-bearing |
| eXosip / libosip2 | production | SIP signalling library, smaller than PJSIP but still a general stack; no minimality measurement |
| Sofia-SIP | production | full-stack library (Nokia heritage); same story |
| Linphone | production | full stack in an application; same story |
| "SIP from scratch" tutorials / toy stacks | hobby | register or call against a lab, but no suite completion and no trace to RFC sections — the missing piece this project supplies |

What this project adds: a **measurement** nobody publishes — the size of the load-bearing
RFC 3261 subset, traced section-by-section, completing a real scenario suite against a real
PBX. The prior implementations prove the full stack is buildable; none of them answer how much
of the spec a working client actually needs.

## 7. Open questions

- **Counting methodology.** "Normative MUST requirement" is not defined by RFC 3261. The raw
  occurrence count (590) over-counts (examples, prose restatements) and the P2/P3 matrix must
  define the unit (distinct normative MUST statements in the message-trace matrix) and state it
  in the evidence. Decided at P3 with the matrix, not retrofitted at P5.
- **Does Asterisk 20 enforce RTCP?** RFC 3550/3264 require RTCP even for sendonly streams;
  whether `chan_pjsip` drops a call whose media is RTP-without-RTCP determines whether RTCP is
  load-bearing for the suite. To be tested at P2 against the pinned container.
- **Is the digest nonce reuse observable?** Asterisk's nonce policy (stale/expiry) affects
  whether refresh REGISTERs need a fresh challenge round-trip; observable only against the real
  PBX.
- **UDP retransmission timers on a LAN.** On a same-host container network, retransmissions
  rarely fire; whether Timer E/F behaviour is load-bearing or merely correct-by-construction
  needs fault injection (P4), which the suite's S-002 scenario already names.
- **RFC 4475 scope.** Full parser robustness against all torture messages is large; the suite
  only needs the client to *send* correct messages and *accept* the PBX's well-formed ones. The
  boundary of what robustness is load-bearing is a scoping decision to record at P2.

## Sources

A parsed ledger, not a bibliography. Gate G1 requires **≥ 5 reachable sources with distinct
URLs, of which ≥ 3 are `Access: full-text`**.

Heading format must be exactly `### SRC-###  —  <citation>`. Every entry needs a URL, an
`Access:` line, and an `Establishes:` line.

`Access` values:

- `full-text` — you read the document. Only these count toward the ≥ 3 requirement.
- `abstract-only` — paywalled; you read the abstract and nothing more.
- `secondary` — a work quoting the primary; **must name the primary** it stands in for.
- `unreachable` — could not be read. Does not count toward G1, and any claim resting on it
  belongs in § 7 Open questions instead.

Marking a source `full-text` that you only skimmed is the one dishonesty no tool here can
detect. It is also the one that destroys the value of everything else in the repository.

---

### SRC-001 — Rosenberg et al., *SIP: Session Initiation Protocol*, RFC 3261, IETF 2002

- **URL:** https://www.rfc-editor.org/rfc/rfc3261.txt
- **Access:** full-text
- **Establishes:** the complete UAC behaviour for REGISTER (§10), INVITE/offer-answer (§13),
  BYE (§15), the six mandatory headers in every request and Contact in INVITE (§8.1.1, §20
  tables 2–3), transaction timers T1=500ms/T2=4s/T4=5s and Timers A/B/E/F/K with 64×T1=32s
  timeout (§17), UDP-5060/TCP-1300-byte transport rules and Via sent-by (§18), digest
  authentication (§22), and the response-code classes (§21). Section concentration of the 590
  normative `MUST` occurrences counted directly from this text (§16:134, §8:75, §17:69, §12:56,
  §10:38, §19:29, §13:27, §18:26, §22:23, §20:20, §7:16, §23:16).

### SRC-002 — Johnston et al., *Session Initiation Protocol (SIP) Basic Call Flow Examples*, RFC 3665, IETF 2003

- **URL:** https://www.rfc-editor.org/rfc/rfc3665.txt
- **Access:** full-text
- **Establishes:** the canonical minimal flows the scenario suite is built from: registration
  with 401 digest challenge → REGISTER with Authorization → 200 OK (§2.1), and session
  establishment INVITE(SDP) → 180 → 200(SDP) → ACK → two-way RTP → BYE → 200 (§3.1), including
  the exact message bodies (headers, SDP, CSeq handling, independent CSeq counts per dialog
  side).

### SRC-003 — Schulzrinne et al., *RTP: A Transport Protocol for Real-Time Applications*, RFC 3550, IETF 2003

- **URL:** https://www.rfc-editor.org/rfc/rfc3550.txt
- **Access:** full-text
- **Establishes:** the RTP fixed header layout — V=2, P, X, CC, M, 7-bit payload type, 16-bit
  sequence number, 32-bit timestamp, 32-bit SSRC (§5.1) — and the SSRC/sequence/timestamp
  semantics a minimal sender must implement to produce and consume a media stream.

### SRC-004 — Handley et al., *SDP: Session Description Protocol*, RFC 4566, IETF 2006

- **URL:** https://www.rfc-editor.org/rfc/rfc4566.txt
- **Access:** full-text
- **Establishes:** the SDP session/media description lines (v/o/s/c/t, m=audio <port> RTP/AVP,
  a=rtpmap) and the direction attributes — `a=sendonly`, `a=recvonly`, `a=sendrecv`,
  `a=inactive` — that carry hold/resume, including that sendrecv is the default when no
  direction attribute is present.

### SRC-005 — Rosenberg & Schulzrinne, *An Offer/Answer Model with SDP*, RFC 3264, IETF 2002

- **URL:** https://www.rfc-editor.org/rfc/rfc3264.txt
- **Access:** full-text
- **Establishes:** the offer/answer rules that bound a client's re-INVITE behaviour: a stream
  offered `a=sendonly` (hold) must be answered `recvonly`/`inactive` and vice versa (§5.1), the
  port-zero rejection rule, and the constraint that an answerer cannot re-offer until the
  transaction completes — the basis for the hold-glare failure mode.

### SRC-006 — Sparks et al., *SIP Torture Test Messages*, RFC 4475, IETF 2006

- **URL:** https://www.rfc-editor.org/rfc/rfc4475.txt
- **Access:** full-text
- **Establishes:** the size of full parser robustness — valid-but-tortuous messages, invalid
  messages that must be rejected cleanly, and transaction/application-layer semantics tests
  (§3.1–3.3). Used in this theory only to scope what the suite does *not* require: the suite's
  PBX sends well-formed messages, so RFC 4475 robustness is not load-bearing for the hypothesis
  (recorded as an open question, § 7).

### SRC-007 — Teluu Ltd., *About PJSIP*, pjsip.org

- **URL:** https://www.pjsip.org/about.htm
- **Access:** full-text
- **Establishes:** the incumbent's own characterization: a free/open-source multimedia stack
  implementing SIP, SDP, RTP, STUN, TURN and ICE, "portable and suitable for almost any type of
  systems", with a voice-call application "from as little as 150 KB" using the lower-level
  libraries — the full-stack surface the concept claims is larger than the load-bearing subset.

### SRC-008 — Asterisk Project, *res_pjsip: SIP Resource* configuration reference, Asterisk 20 documentation

- **URL:** https://docs.asterisk.org/Asterisk_20_Documentation/API_Documentation/Module_Configuration/res_pjsip/
- **Access:** full-text
- **Establishes:** the PBX side of the suite: `chan_pjsip`/`res_pjsip` is Asterisk's SIP channel
  driver, configured in `pjsip.conf` with `[transport]`, `[endpoint]`, `[auth]` and `[aor]`
  objects; `auth_type` values `password`/`md5_cred` implement digest authentication; endpoint
  options (`outbound_auth`, `aors`, `context`, `rtp_timeout`, media port ranges) define what a
  registering client must satisfy.
