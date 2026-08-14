# Agent contract — Minimal SIP client: what RFC 3261 subset is sufficient to register and hold a call with a real PBX?

This repository is a concept project (`C-004`, archetype `implementation`).
It answers exactly one question:

> What subset of RFC 3261 is necessary and sufficient for a from-scratch SIP client to register, establish, and tear down a two-way RTP call against a mainstream PBX?

## Content rules that bind every document here

- **No unevidenced claim.** Every capability, number, or comparison in `README.md` or `docs/`
  carries an `E-###` that resolves to an entry with a reproduction command, an environment, and
  an observed result. Adjectives without an ID get deleted, not softened.
- **No hand-written scores.** The `computed` block in `.sota/readiness.yaml` is machine-written
  and never hand-edited.
- **Readiness is descriptive.** Claim the highest level whose evidence reproduces *today*. If
  something broke, lower the level and say so.
- **Baseline or nothing.** "Faster" requires the named baseline, the metric, the conditions, and
  the `E-###`.
- **Complexity must be purchased.** Anything structural traces to a scenario `S-###` in
  `.sota/quality-gates.yaml` and is recorded in `complexity_ledger`. Otherwise remove it.
- **One question.** A second interesting idea goes to the shared backlog, not into this
  repository.
- **Real citations only.** Sources must have been fetched and read, not recalled.
- **Honest limitations.** `README.md § Limitations` names the conditions under which this
  degrades, what is untested, and what would move it up a readiness level.

## Session handoff

Before running low on context, append an `L-###` entry to `docs/03-log.md` with
`Disposition: open` describing what is half-finished. That file is the state that survives
a session boundary.

## Commit messages

History is part of the public record — it is read like the README. Commits follow
[Conventional Commits 1.0.0](https://www.conventionalcommits.org/en/v1.0.0/):

```
<type>[optional scope][!]: <description>

[optional body]

[optional footer(s)]
```

- `feat:` a new capability in the public surface; `fix:` a bug fix; `docs:` README,
  theory, evidence entries, ADRs, and the artifact narrative/page; `perf:` benchmarks
  and anything touching measured performance; `test:` the test suite; `refactor:`
  behaviour-preserving restructuring; `build:` build tooling; `ci:` CI configuration;
  `chore:` everything else. Other types may be used when they fit better.
- A scope is optional and names the part: `feat(core):`, `fix(pagination):`.
- Breaking changes to the public surface get `!` after type/scope, or a
  `BREAKING CHANGE:` footer.
- Description in imperative mood, lowercase, no trailing period:
  `fix: guard page number against empty cursor`.

Overlay rules for this repository:

- Cite the stable ID when the change has one — the message points at the record,
  it does not re-narrate it: `docs: add E-004 depth sweep`, `test: cover E-003
  failure modes`, `docs: D-001 query building decision`.
- One concern per commit. A commit that fixes a bug *and* rewrites the README is two commits.
- No mentions of assistants, AI, or tooling — the history is the project's own.
- No process-state messages ("finished phase X") — commit content, not progress.
- Never rewrite published history to hide a reversal; the history itself shows it.

## Stable IDs used in this repository

| Prefix | Meaning | Lives in |
| :--- | :--- | :--- |
| `C-###` | Concept | `.sota/concept.yaml` |
| `H-###` | Hypothesis | `.sota/concept.yaml` |
| `E-###` | Evidence | `docs/05-evidence.md` |
| `L-###` | Working-log entry | `docs/03-log.md` (append-only) |
| `SRC-###` | Source, with an `Access:` level | `docs/01-theory.md` |
| `S-###` | Quality-attribute scenario | `.sota/quality-gates.yaml` |
| `D-###` | Architecture decision record | `docs/adr/D-###-*.md` |
| `SP-###` / `TP-###` | Sensitivity / tradeoff point | `docs/04-tradeoffs.md` |
| `R-###` / `NR-###` | Risk / non-risk | `docs/04-tradeoffs.md` |
| `X-###` | ISO 5055 exemption | `.sota/quality-gates.yaml` |

IDs are never reused or renumbered. Retired IDs stay in place marked `retired`.
