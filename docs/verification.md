

<!-- verification:start -->

# Verification

_8 pass · inputs digest f3bf152d336c5825 · machine-generated, do not hand-edit._

Every entry in [docs/05-evidence.md](05-evidence.md) declares how it would be checked; `python3 tools/verify_evidence.py` executes those declarations. An entry marked `unrunnable` needs something this machine did not have — the reason is in `.sota/verification.json`.

| Entry | Kind | Outcome | Date | Checks |
| :--- | :--- | :--- | :--- | :--- |
| E-001 | survey | pass | 2026-09-16 | exit-zero, output-contains |
| E-002 | test | pass | 2026-09-16 | exit-zero, output-contains |
| E-003 | test | pass | 2026-09-16 | exit-zero, output-contains |
| E-004 | test | pass | 2026-09-16 | exit-zero, output-contains |
| E-005 | survey | pass | 2026-09-16 | exit-zero, output-contains |
| E-006 | benchmark | pass | 2026-09-16 | exit-zero |
| E-007 | survey | pass | 2026-09-16 | exit-zero, output-contains |
| E-008 | benchmark | pass | 2026-09-16 | exit-zero |

Reproduce: `python3 tools/verify_evidence.py`

<!-- verification:end -->
