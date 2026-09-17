

<!-- verification:start -->

# Verification

_1 fail · inputs digest 4a5bdcb8b239ef27 · machine-generated, do not hand-edit._

Every entry in [docs/05-evidence.md](05-evidence.md) declares how it would be checked; `python3 tools/verify_evidence.py` executes those declarations. An entry marked `unrunnable` needs something this machine did not have — the reason is in `.sota/verification.json`.

| Entry | Kind | Outcome | Date | Checks |
| :--- | :--- | :--- | :--- | :--- |
| E-008 | benchmark | fail | 2026-09-17 | exit-zero, output-contains, output-contains, computed-from |

Reproduce: `python3 tools/verify_evidence.py`

<!-- verification:end -->
