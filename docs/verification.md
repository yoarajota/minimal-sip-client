

<!-- verification:start -->

# Verification

_1 pass · inputs digest 48d03413c8b727f4 · machine-generated, do not hand-edit._

Every entry in [docs/05-evidence.md](05-evidence.md) declares how it would be checked; `python3 tools/verify_evidence.py` executes those declarations. An entry marked `unrunnable` needs something this machine did not have — the reason is in `.sota/verification.json`.

| Entry | Kind | Outcome | Date | Checks |
| :--- | :--- | :--- | :--- | :--- |
| E-006 | benchmark | pass | 2026-09-16 | exit-zero |

Reproduce: `python3 tools/verify_evidence.py`

<!-- verification:end -->
