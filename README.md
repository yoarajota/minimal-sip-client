# Minimal SIP client: what RFC 3261 subset is sufficient to register and hold a call with a real PBX?

> What subset of RFC 3261 is necessary and sufficient for a from-scratch SIP client to register, establish, and tear down a two-way RTP call against a mainstream PBX?

**Concept** `C-004` · **archetype** `implementation`.

<!-- Replace this paragraph with the summary from .sota/concept.yaml: what the concept is,
     in plain language, for a reader who has never heard of it. No adjectives about how good
     it is — the scorecard below carries that. -->

## Claim

<!-- The hypothesis from .sota/concept.yaml, verbatim, with its verdict and the evidence that
     decided it. Every number here carries an E-### ID. -->

**H-001** — TODO. → *verdict: untested*

## Baseline

Compared against **TODO** (version TODO) because TODO.
Reproduce the comparison: `TODO` — see [E-001](docs/05-evidence.md#e-001).

<!-- scorecard:start -->
<!-- Auto-generated. Do not hand-edit. -->
<!-- scorecard:end -->

## Try it

```bash
# setup — must work from a clean checkout
TODO

# the headline result in one command
TODO
```

## How it works

<!-- The mechanism, briefly. Link to docs/01-theory.md for the sourced version and to
     docs/04-tradeoffs.md for the architecture and its tradeoffs. -->

## Evidence

| ID | Claim | Command | Result |
| :--- | :--- | :--- | :--- |
| E-001 | TODO | `TODO` | TODO |

Full ledger: [docs/05-evidence.md](docs/05-evidence.md).

## Limitations

<!-- Mandatory and specific. State the conditions under which this degrades or is
     the wrong choice, the readiness ceiling it sits under and why, and what the adversarial
     pass in docs/04-tradeoffs.md found. "May not suit every use case" is not a limitation. -->

- TODO — the condition under which the baseline wins.
- TODO — what is untested.
- TODO — what would have to be true to move up a readiness level.

### What we tried that didn't work

<!-- Filled at P6 from the `dead-end` entries in docs/03-log.md. Often the most useful part of
     the repository — it is the part nobody else publishes. -->

- TODO

## Documents

| Document | Contents |
| :--- | :--- |
| [docs/01-theory.md](docs/01-theory.md) | What the literature establishes, with the `SRC-###` source ledger. |
| [docs/03-log.md](docs/03-log.md) | Append-only working log: what was tried, including what failed. |
| [docs/04-tradeoffs.md](docs/04-tradeoffs.md) | ATAM-lite: drivers, scenarios, sensitivity and tradeoff points, risks. |
| [docs/05-evidence.md](docs/05-evidence.md) | Every claim, its command, environment, and observed result. |
| [docs/adr/](docs/adr/) | Decisions and the options rejected. |
| [.sota/](.sota/) | Machine-readable readiness and quality data. |

## License

TODO
