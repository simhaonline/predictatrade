---
name: xauusd-quant-validation
description: Validate strategies/models with leakage-safe data, realistic costs, walk-forward/OOS, calibration, confidence intervals, Monte Carlo, parity, drift and claim evidence.
---

# xauusd-quant-validation

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Freeze dataset/provenance, strategy/features/target/exit/broker-cost versions.
2. Prevent look-ahead/target leakage/post-outcome filtering.
3. Model bid/ask, spread, slippage, commission, latency, missed/partial fills and carry where relevant.
4. Run golden feature tests, backtest, walk-forward and locked OOS slices.
5. Validate probability with Brier/ECE/reliability plus Wilson/sample sufficiency.
6. Run bootstrap/Monte Carlo/stress and baselines/ablations.
7. Verify Go/Python feature/decision parity.
8. Produce paper/shadow promotion and drift evidence.

## Rules
Raw score ≠ probability; models cannot self-promote; unsupported performance claims are rejected.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
