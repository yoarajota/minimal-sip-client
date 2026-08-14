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

Filled at P5. Applies to every entry tagged as a benchmark.

- **What is measured:** TODO
- **Runs:** TODO (≥ 5), **warm-up:** TODO
- **Held constant:** TODO
- **Baseline configuration:** TODO — the same tuning effort was spent on the baseline as on
  the concept; if not, say so, because it invalidates the comparison.
- **Known measurement bias:** TODO
