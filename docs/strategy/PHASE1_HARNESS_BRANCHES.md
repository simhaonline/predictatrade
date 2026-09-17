# Phase 1 Harness Branch Matrix — Phase 0.95 Task C
Date: 2026-09-17 · Status: ALL FOUR BRANCHES PROVEN (synthetic + live) ·
Harness: scripts/phase1_run.py · Synthetic data quarantined in docs/synthetic/ (labeled SYNTHETIC_DRYRUN)

**PROMOTE is a RECOMMENDATION, not an action.** Nothing in the harness promotes,
modifies parameters, or touches live behavior. Promotion requires the flag-gated
shadow → canary → promote sequence in PHASE1_CALIBRATION_PROTOCOL.md §5 plus
operator approval.

## Statistical-method fix discovered during branch proof

The original "paired bootstrap" (resampling both samples independently and
re-centering) has zero power for a mean-difference effect — the resampled delta
centers exactly at the observed difference, so p≈0.5 for ANY effect. Replaced
with a **one-sided permutation test** (pool, shuffle labels 10k×, fraction of
permuted deltas ≥ observed). Distribution-free under exchangeability, exact
power, deterministic (seed 7). Effect-size gate (≥0.10R) unchanged.

## Branch matrix (mode × tested × result)

| Branch | Data | gate | n | effect | p | BH-FDR | Decision | Exit |
|---|---|---|---|---|---|---|---|---|
| BLOCKED→REFUSE | live EXECUTION channel | BLOCKED (47/300) | — | — | — | — | REFUSED, refuse message on stderr | **2** ✓ |
| BLOCKED→REFUSE (shadow) | live SHADOW (MARNIE_FIB) | BLOCKED (3) | 3 | — | — | — | REFUSED | 0 (mixed run) |
| HOLD | synthetic SYN_A | PASS | 320 | +0.0116R | 0.3744 | False | HOLD | 0 |
| PROMOTE | synthetic | PASS | 320 | +1.1856R | 0.0001 | True | **PROMOTE_RECOMMENDATION** | 0 |
| ROLLBACK | synthetic | PASS | 320 | −1.1927R | 0.0001 | True | ROLLBACK_RECOMMENDATION | 0 |

Golden unit tests (test_phase1_run.py) ALL PASS: BLOCKED→REFUSED; placebo→HOLD;
significant positive→PROMOTE_RECOMMENDATION; BH-FDR correction; channel separation.

## Quarantine contract

- Synthetic files live ONLY in `docs/synthetic/`, each JSON begins
  `{"label": "SYNTHETIC_DRYRUN", ...}` — `load_synthetic()` REFUSES any file
  without the exact label.
- Synthetic runs read no real tables; results carry run_label=SYNTHETIC_DRYRUN
  and are never written to trading.prediction_outcomes or any calibration input.
- The earlier shadow dry-run (Phase 0.9) remains labeled DRY-RUN/SHADOW.

## How to run

```
python3 scripts/phase1_run.py --synthetic-file docs/synthetic/phase1_branch_promote.json   # PROMOTE proof
python3 scripts/phase1_run.py --synthetic-file docs/synthetic/phase1_branch_hold.json      # HOLD proof
python3 scripts/phase1_run.py --synthetic-file docs/synthetic/phase1_branch_rollback.json  # ROLLBACK proof
python3 scripts/phase1_run.py                                                             # real gate (currently REFUSES, exit 2)
```