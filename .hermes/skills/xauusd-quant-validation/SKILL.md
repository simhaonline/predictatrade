---
name: xauusd-quant-validation
description: "Validate strategies with leakage-safe backtesting."
---

# xauusd-quant-validation

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Workflow
1. Freeze dataset/provenance, strategy/features/target/exit/broker-cost versions.
2. Prevent look-ahead/target leakage/post-outcome filtering.
3. Model bid/ask, spread, slippage, commission, latency, missed/partial fills, carry.
4. Run golden feature tests, backtest, walk-forward and locked OOS slices.
5. Validate probability with Brier/ECE/reliability plus Wilson/sample sufficiency.
6. Run bootstrap/Monte Carlo/stress and baselines/ablations.
7. Verify Go/Python feature/decision parity.
8. Produce paper/shadow promotion and drift evidence.

## Rules
Raw score != probability; models cannot self-promote; unsupported performance claims rejected.
