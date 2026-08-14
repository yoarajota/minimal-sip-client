# Theory — Minimal SIP client: what RFC 3261 subset is sufficient to register and hold a call with a real PBX?

Produced at P1. Establishes what is already known, from primary sources only. Every source
below was fetched and read; nothing is cited from memory.

## 1. The mechanism

<!-- How the concept works, in enough detail that a reader could implement a naive version.
     Not history, not marketing. If there is a formal statement (an invariant, a bound, a
     complexity class), state it. -->

TODO

## 2. Conditions under which it holds

<!-- Every mechanism has an operating envelope. What must be true for the claimed behaviour to
     appear? Load, data distribution, concurrency model, hardware, network assumptions. -->

TODO

## 3. Known failure modes

<!-- Required by gate G1. What the literature says goes wrong, and under what conditions.
     These become the property/edge tests at P3 and the adversarial pass at P4. -->

| Failure mode | Trigger condition | Source |
| :--- | :--- | :--- |
| TODO | TODO | [n] |

## 4. The incumbent

<!-- The baseline named in .sota/concept.yaml. What it does, why it is the honest comparison,
     and where the literature says it falls short. -->

TODO

## 5. Hypothesis

**H-001** — Under TODO conditions, Minimal SIP client: what RFC 3261 subset is sufficient to register and hold a call with a real PBX? achieves TODO metric TODO comparison against
TODO baseline, at the cost of TODO tradeoff.

*Falsified if:* TODO — the concrete observation that would kill it.
*Measured by:* TODO — the benchmark that will decide it at P5.

## 6. Prior implementations

<!-- What already exists. If a mature implementation exists, say so plainly and state what this
     project adds — understanding, a measurement, a different tradeoff. "Nothing" is an
     acceptable answer and is worth knowing before P2. -->

| Implementation | Maturity | What it does differently |
| :--- | :--- | :--- |
| TODO | TODO | TODO |

## 7. Open questions

<!-- Anything the literature does not settle, and anything a source could not be verified for.
     Unverifiable claims go here as questions, never into the sections above. -->

- TODO

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

### SRC-001 — TODO author, *TODO title*, TODO venue and year

- **URL:** https://TODO
- **Access:** TODO
- **Establishes:** TODO — what this source actually settled, specifically. Not "background on
  the topic".

<!-- Repeat for each source. Minimum five reachable, minimum three full-text.

### SRC-002 — citation

- **URL:**
- **Access:**
- **Establishes:**

-->
