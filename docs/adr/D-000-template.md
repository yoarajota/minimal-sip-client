# D-000 — Template: <decision in one line>

**Status:** proposed | accepted | superseded by D-###
**Date:** 2026-08-14
**Phase:** P3

## Context

What forces are at play? Which scenarios (`S-###`) and drivers make this decision necessary?
Keep it factual — the constraint, not the conclusion.

## Decision

What was decided, stated as an imperative: "We use X for Y."

## Options rejected

| Option | Why rejected |
| :--- | :--- |
| TODO | TODO |

An ADR with no rejected options records a preference, not a decision. If there was genuinely no
alternative, say why the space was empty.

## Tradeoff

What this costs. Name the ISO/IEC 25010 characteristic that gets worse, and link the tradeoff
point `TP-###` in `docs/04-tradeoffs.md` if one was recorded.

## Complexity justification

Required whenever this decision adds a service, queue, layer, runtime, or dependency:

- **Adds:** TODO
- **Justified by:** S-###
- **Ledger entry:** recorded in `.sota/quality-gates.yaml` → `complexity_ledger`

If no scenario requires it, the decision is to remove it instead.

## Consequences

What becomes easier, what becomes harder, and what must be revisited if the assumption behind
this decision stops holding (cross-reference the matching `NR-###`).
