

<!-- verification:start -->

# Verification

_8 pass · inputs digest f3f07c7bfe7825f3 · machine-generated, do not hand-edit._

Every entry in [docs/05-evidence.md](05-evidence.md) declares how it would be checked; `python3 tools/verify_evidence.py` executes those declarations. An entry marked `unrunnable` needs something this machine did not have — the reason is in `.sota/verification.json`.

| Entry | Kind | Outcome | Date | Checks |
| :--- | :--- | :--- | :--- | :--- |
| E-001 | survey | pass | 2026-09-17 | exit-zero, output-contains |
| E-002 | test | pass | 2026-09-17 | exit-zero, output-contains |
| E-003 | test | pass | 2026-09-17 | exit-zero, output-contains |
| E-004 | test | pass | 2026-09-17 | exit-zero, output-contains |
| E-005 | survey | pass | 2026-09-17 | exit-zero, output-contains |
| E-006 | benchmark | pass | 2026-09-17 | exit-zero, output-contains, computed-from, computed-from, computed-from |
| E-007 | survey | pass | 2026-09-17 | exit-zero, output-contains |
| E-008 | benchmark | pass | 2026-09-17 | precondition, exit-zero, output-contains, output-contains, computed-from |

Reproduce: `python3 tools/verify_evidence.py`

<!-- verification:end -->
