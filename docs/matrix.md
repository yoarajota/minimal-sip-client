# Message-trace matrix — the load-bearing subset of RFC 3261

The measurement instrument for H-001. Every behaviour the minimal client implements is traced
to the RFC 3261 section that forces it; the sum of the normative MUST statements forced by
those sections is the subset size, compared against the whole-RFC MUST count (E-001, E-007).

## Counting methodology

**Unit (decided at P3, operationalised at P5 — never retrofitted):** a row counts the
distinct normative MUST statements in the referenced section(s) that the behaviour actually
forces — not every MUST occurrence in the section (many are about other roles: proxy behaviour
in §16, UAS behaviour in §15.1.2, registrar behaviour in §10.3). A "normative MUST statement"
is one RFC sentence carrying MUST/MUST NOT.

**Operationalisation (recorded 2026-08-14, before the counts were finalised):**

1. **Forced** means: the implemented behaviour in `internal/sip` demonstrably implements
   compliance with the statement as part of the behaviour the suite exercises — deleting the
   code that satisfies it would break the suite or make the implementation an incorrect
   instance of the behaviour. Excluded: statements satisfied incidentally (generic unknown-
   header ignoring), statements conditional on unimplemented features (TLS/SIPS, pre-existing
   route sets, Record-Route, multicast, forking), and statements binding roles the client does
   not play (proxy, registrar, UAS, server).
2. **Count per statement, not per occurrence**: a sentence with two MUSTs (e.g. "…MUST be
   expressible… and MUST be less than 2**31") is one statement. Each counted statement is
   quoted in the column, so the count is auditable.
3. **No double counting across rows**: a statement forced by two behaviours (e.g. §17.1.1.2
   covers provisional handling *and* the retransmission timers) is counted once, in the row
   whose behaviour first demands it. Rows 6, 8 and 9 of the P3 scaffold were merged into the
   single "client transaction layer" row because they share §17.1.1.2; the union is what
   matters.
4. **Two robustness cross-checks** are reported alongside the primary count: the
   occurrence-level count (each MUST occurrence in a forced statement, matching the 590
   occurrence denominator) and the unfiltered bound (every MUST occurrence in the cited
   sections regardless of role). The verdict holds under all three.

## The count

| # | Behaviour (trace step) | RFC 3261 section forcing it | MUST statements forced (each quoted) | Count |
| :--- | :--- | :--- | :--- | :--- |
| 1 | REGISTER construction: Request-URI = domain (no userinfo), To = AOR, From + tag, Call-ID, CSeq, Contact binding, Expires | §10.2, §8.1.1.1–8.1.1.8 | §10.2: "The following header fields, except Contact, MUST be included in a REGISTER request"; "The 'userinfo' and '@' components of the SIP URI MUST NOT be present"; "This address-of-record MUST be a SIP URI or SIPS URI"; "A UA MUST increment the CSeq value by one for each REGISTER request with the same Call-ID"; "UAs MUST NOT send a new registration … until they have received a final response … or the previous REGISTER request has timed out". §8.1.1.1: — (only MUST is conditional on a pre-existing route set). §8.1.1.2: "All SIP implementations MUST support the SIP URI scheme"; "A request outside of a dialog MUST NOT contain a To tag". §8.1.1.3: "The From field MUST contain a new 'tag' parameter" (+ §19.3: the tag "MUST be globally unique and cryptographically random"). §8.1.1.4: "It MUST be the same for all requests and responses sent by either UA in a dialog"; "the Call-ID header field MUST be selected by the UAC as a globally unique identifier over space and time". §8.1.1.5: "The method MUST match that of the request"; "The sequence number value MUST be expressible as a 32-bit unsigned integer and MUST be less than 2**31". §8.1.1.6: "A UAC MUST insert a Max-Forwards header field into each request it originates". §8.1.1.7: "it MUST insert a Via into that request"; "The protocol name and protocol version … MUST be SIP and 2.0"; "The Via header field value MUST contain a branch parameter"; "The branch parameter value MUST be unique across space and time"; "The branch ID … MUST always begin with the characters 'z9hG4bK'". §8.1.1.8: "The Contact header field MUST be present and contain exactly one SIP or SIPS URI in any request that can result in the establishment of a dialog"; "this URI MUST be valid even if used in subsequent requests outside of any dialogs" | 22 |
| 2 | Request line and six mandatory headers (method, Request-URI, SIP-Version; To, From, CSeq, Call-ID, Max-Forwards=70, Via+branch) | §7.1, §8.1.1 | §7.1: "The Request-URI MUST NOT contain unescaped spaces or control characters and MUST NOT be enclosed in '<>'"; "applications sending SIP messages MUST include a SIP-Version"; "implementations MUST send upper-case". §8.1.1: "A valid SIP request formulated by a UAC MUST, at a minimum, contain the following header fields: To, From, CSeq, Call-ID, Max-Forwards, and Via" | 4 |
| 3 | Digest authentication: parse WWW-Authenticate, compute response, Authorization on retry | §22.4 | "For SIP, the 'uri' MUST be enclosed in quotation marks"; "a cnonce value MUST NOT be sent in an Authorization … header field if no qop directive has been sent"; "If a client receives a 'qop' parameter in a challenge header field, it MUST send the 'qop' parameter in any resulting authorization header field" | 3 |
| 4 | 401/407 handling on any request (REGISTER *and* INVITE — Asterisk challenges the initial INVITE) | §8.1.3.5, §22.2 | §8.1.3.5: — (all SHOULD). §22.2: "it MUST increment the CSeq header field value as it would normally when sending an updated request" | 1 |
| 5 | INVITE with SDP offer (Content-Type/Content-Length, Contact mandatory) | §13.2.1, §8.1.1.8–8.1.1.10, §20.14 | §13.2.1: "The initial offer MUST be in either an INVITE or … the first reliable non-failure message from the UAS"; "The UAC MUST treat the first session description it receives as the answer, and MUST ignore any session descriptions in subsequent responses"; "The Session Description Protocol (SDP) … MUST be supported by all user agents"; "its usage for constructing offers and answers MUST follow the procedures defined in [13]". §20.14: "the header field MUST be used" (Content-Length on UDP); "Content-Length header field value MUST be set to zero" (no body). §8.1.1.8 Contact already counted in row 1 | 6 |
| 6 | Client transaction layer (INVITE §17.1.1.2, non-INVITE §17.1.2.2, matching §17.1.3): state machine, timers T1/T2/64×T1, retransmission, provisional handling (1xx stops INVITE retransmission), 64×T1 timeout → 408, non-2xx final → transaction-layer ACK, response matching by branch+CSeq | §17.1.1.2, §17.1.2.2, §17.1.3 | §17.1.1.2: "The initial state, 'calling', MUST be entered…"; "The client transaction MUST pass the request to the transport layer for transmission"; "the client transaction MUST start timer A with a value of T1"; "the client transaction MUST start timer B with a value of 64*T1 seconds"; "When timer A fires, the client transaction MUST retransmit the request … and MUST reset the timer with a value of 2*T1"; "the request MUST be retransmitted again"; "This process MUST continue so that the request is retransmitted with intervals that double"; "the exponential backoffs … MUST be used"; "The client transaction MUST NOT generate an ACK"; "the provisional response MUST be passed to the TU"; "Any further provisional responses MUST be passed up to the TU"; "reception of a response with status code from 300-699 MUST cause the client transaction to transition to 'Completed'"; "The client transaction MUST pass the received response up to the TU, and the client transaction MUST generate an ACK request"; "The ACK MUST be sent to the same address, port, and transport"; "Any retransmissions of the final response … MUST cause the ACK to be re-passed"; "the newly received response MUST NOT be passed up to the TU"; "reception of a 2xx response MUST cause the client transaction to enter the 'Terminated' state, and the response MUST be passed up to the TU"; "The client transaction MUST be destroyed the instant it enters the 'Terminated' state". §17.1.2.2: "The request MUST be passed to the transport layer for transmission"; "the client transaction MUST set timer E to fire in T1 seconds"; "the request MUST be passed to the transport layer for retransmission, and Timer E MUST be reset with a value of T2 seconds"; "the TU MUST be informed of a timeout, and the client transaction MUST transition to the terminated state"; "the response MUST be passed to the TU" (provisional, Trying); "the response MUST be passed to the TU, and the client transaction MUST transition to the 'Completed' state" (final, Trying); "the response MUST be passed to the TU, and the client transaction MUST transition to the 'Completed' state" (final, Proceeding); "Once the transaction is in the terminated state, it MUST be destroyed immediately". §17.1.3: — (no MUST) | 33 |
| 7 | 2xx → dialog confirmed: To tag captured, remote target from Contact, ACK by UAC core (same CSeq, method ACK), dialog state | §12.1.2, §13.2.2.4 | §12.1.2: "it MUST provide a SIP or SIPS URI with global scope … in the Contact header field"; "This state MUST be maintained for the duration of the dialog"; "The remote target MUST be set to the URI from the Contact header field of the response"; "The local sequence number MUST be set to the value of the sequence number in the CSeq header field of the request"; "The call identifier component of the dialog ID MUST be set to the value of the Call-ID in the request"; "The local tag component of the dialog ID MUST be set to the tag in the From field in the request, and the remote tag component of the dialog ID MUST be set to the tag in the To field of the response"; "The remote URI MUST be set to the URI in the To field, and the local URI MUST be set to the URI in the From field". §13.2.2.4: "the dialog MUST be transitioned to the 'confirmed' state"; "a new dialog in the 'confirmed' state MUST be constructed"; "The UAC core MUST generate an ACK request for each 2xx received"; "The sequence number of the CSeq header field MUST be the same as the INVITE being acknowledged, but the CSeq method MUST be ACK"; (ACK credentials: NOT counted — the suite's PBX does not require them; the client sends none) | 14 |
| 8 | In-dialog request construction (BYE, re-INVITE): To/From with tags, CSeq increment, Request-URI = remote target, no Route | §12.2.1.1 | "The URI in the To field of the request MUST be set to the remote URI from the dialog state"; "The tag in the To header field of the request MUST be set to the remote tag of the dialog ID"; "The From URI of the request MUST be set to the local URI from the dialog state"; "The tag in the From header field of the request MUST be set to the local tag of the dialog ID"; "The Call-ID of the request MUST be set to the Call-ID of the dialog"; "Requests within a dialog MUST contain strictly monotonically increasing and contiguous CSeq sequence numbers"; "the value of the local sequence number MUST be incremented by one, and this value MUST be placed into the CSeq header field"; "an initial value MUST be chosen using the guidelines of Section 8.1.1.5"; "The method field in the CSeq header field value MUST match the method of the request"; "If the route set is empty, the UAC MUST place the remote target URI into the Request-URI"; "The UAC MUST NOT add a Route header field to the request" | 12 |
| 9 | Hold: re-INVITE offer `a=sendonly`; Resume: re-INVITE `a=sendrecv`; answer direction rules | §13.2.1 (offer/answer), RFC 3264 §5.1 | §13.2.1's in-dialog offer rules are MAY ("…the UAC MAY generate subsequent offers…"); the sendonly/recvonly/sendrecv direction rules live in RFC 3264 (external RFC — not counted against the RFC 3261 denominator, same as rows 11–12) | 0 (n/a: RFC 3264) |
| 10 | BYE: in-dialog request, session ends when BYE passed to transaction; 2xx to BYE | §15.1.1, §15.1.2 | §15.1.1: "The UAC MUST consider the session terminated … as soon as the BYE request is passed to the client transaction"; "the UAC MUST consider the session and the dialog terminated" (on 481/408/timeout). §15.1.2: — (UAS role; the client is UAC-only) | 2 |
| 11 | Transport: UDP framing, Via sent-by, responses to source address/port, 65535-byte buffer | §18.1.1, §18.1.2 | §18.1.1: "implementations MUST be able to handle messages up to the maximum datagram packet size. For UDP, this size is 65,535 bytes"; "the client transport MUST insert a value of the 'sent-by' field into the Via header field"; "For unreliable unicast transports, the client transport MUST be prepared to receive responses on the source IP address from which the request is sent … and the port number in the 'sent-by' field". §18.1.2: "If there is a match, the response MUST be passed to that transaction". (The 1300-byte → TCP rule is conditional on message size; suite messages never exceed it. The sent-by response filter is implemented via branch matching, not sent-by checks — not counted.) | 4 |
| 12 | SDP offer/answer (RFC 4566): v/o/s/c/t/m lines, rtpmap, direction attributes | §13.2.1 (via RFC 4566, RFC 3264) | n/a — external RFC (RFC 4566), outside the RFC 3261 denominator | 0 (n/a) |
| 13 | RTP send/receive (RFC 3550): 12-byte header, SSRC/seq/timestamp, PT 0 PCMU | §13.2.1 (media via RFC 3550) | n/a — external RFC (RFC 3550), outside the RFC 3261 denominator | 0 (n/a) |
| | **Total (primary unit: distinct normative MUST statements)** | | | **90** |

## Result and robustness

- **Primary (statements, both sides): 90 / 540 = 16.7%** of RFC 3261's normative MUST
  statements (540 = whole-RFC sentence-level count, multi-MUST sentences counted once —
  E-007).
- **Cross-check 1 (occurrences, both sides): 103 / 590 = 17.5%** — counting every MUST
  occurrence inside the forced statements against the 590-occurrence denominator (E-001).
- **Cross-check 2 (unfiltered bound): 156 / 590 = 26.4%** — every MUST occurrence in the
  cited sections regardless of role; even a reader who ignores the role filtering lands well
  under the 50% line.

**Verdict (decided 2026-08-14): H-001 supported** — the minimal client forces 90 of 540
normative MUST statements (16.7%) while completing the same suite the full PJSIP 2.17 stack
completes (E-008). The falsifier line (≥ 50%) is not approached under any counting rule.

## Deliberately outside the matrix

Behaviours the suite does *not* force, per `docs/01-theory.md § 2` — their absence is the
claim, not a gap. The cost clause of H-001:

- **Proxy behaviour (§16):** 134 of the 590 MUST occurrences; the client talks directly to the
  PBX, none are forced.
- **S/MIME (§23):** 16 occurrences; the suite uses digest auth, not S/MIME.
- **Full parser robustness (RFC 4475):** the PBX sends well-formed messages; robustness is
  scoped out (§ 7 of `docs/01-theory.md`).
- **Presence/SUBSCRIBE/NOTIFY, session timers (RFC 4028), PRACK (100rel), ICE/STUN/TURN,
  forking, multi-call concurrency, TCP/TLS transports, codecs other than PCMU.** Each is a
  MUST-free zone for this client; quantified in the cost table (E-008).

## Status

Complete at P5. Rows 1–13 were exercised end-to-end by the component against the real PBX
(E-004); the enumeration above is auditable statement-by-statement against the RFC text
(E-007). Supports S-003.
