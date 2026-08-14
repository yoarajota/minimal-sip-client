# D-003 — Project-local quality toolchain built by `make tools`

**Status:** accepted
**Date:** 2026-08-14
**Phase:** P3

## Context

G3 requires the ISO/IEC 5055 gate (`make quality`) to run every declared weakness check and
exit non-zero on violation. Rule R10 forbids host-global tool installs as part of setup — a
reader on a clean machine must be able to reproduce the gate without installing anything
system-wide. The declared checks (`.sota/quality-gates.yaml` `iso5055.weaknesses`) map to
gofmt/go vet (toolchain), staticcheck, errcheck, gocyclo, deadcode, gitleaks and golangci-lint
(third-party).

## Decision

`make tools` installs every third-party tool into `.tools/` via `go install pkg@version` with
**pinned versions**, with `GOBIN` pointed at the repository. `make quality` depends on `tools`
and invokes the binaries from `.tools/`. `.tools/` is gitignored; the versions are pinned in
the Makefile, so a clean checkout reproduces the exact toolchain.

Pins (chosen to build under Go 1.22.2): gocyclo v0.6.0, errcheck v1.7.0,
golang.org/x/tools/cmd/deadcode v0.24.0, gitleaks v8.21.2 (via its pre-rename module path
`github.com/zricethezav/gitleaks/v8` — the module declares that path at that version),
staticcheck 2025.1.1, golangci-lint v1.60.3.

Two check-specific notes, recorded so they are not re-litigated:

- `errcheck` gets `-ignore 'net:.*,<module>/internal/sip:Close'` because Close/WriteToUDP/
  SetReadDeadline calls are deliberate best-effort (cleanup, ACKs) — each is marked
  `//nolint:errcheck` for the golangci-lint pass, and the standalone errcheck uses the ignore
  rule since it does not honour nolint.
- `gocyclo` and `gofmt` run with `-ignore 'poc/'` / path exclusion: the PoC (`poc/`, separate
  Go module) is throwaway evidence, vet/build-checked but not held to the component gate.

## Options rejected

| Option | Why rejected |
| :--- | :--- |
| Use globally installed tools (`~/go/bin`) | Violates R10's project-local requirement and fails on a clean machine. |
| `go run pkg@version` per check | Re-resolves and rebuilds on every run; slow and less deterministic than one pinned install step. |
| Commit the `.tools/` binaries | Generated artefacts must not be committed (R13); the Makefile reproduces them. |
| Run golangci-lint with its default linter set | Its default enables many style linters the minimal implementation neither needs nor wants; `.golangci.yml` enables the safety set (errcheck, govet, ineffassign, staticcheck, unused) plus `dupl` (the mapped CWE-1041 check). |

## Tradeoff

`make quality` now needs a network fetch the first time (tool builds). That is the price of
R10 honesty; after the first `make tools`, the gate is offline and fast (~13 s observed).

## Complexity justification

- **Adds:** `.tools/` (gitignored build output), the `tools` target, `.golangci.yml`.
- **Justified by:** the G3 ISO/IEC 5055 gate, which is exactly the scenario that requires a
  reproducible, project-local runner (S-001 verification path, `iso5055.runner`).
- **Ledger entry:** recorded in `.sota/quality-gates.yaml` → `complexity_ledger`.

## Consequences

A reader on a clean machine runs `make quality` and gets the same gate the author ran. The
`iso5055.last_run` record (commit + exit code) is honest because the runner really executes the
declared tools. If a tool's pinned version needs a newer Go, the `go install` step fails loudly
at `make tools`, not silently at the gate.
