---
name: xauusd-strategy-spec
description: "Specify distinct XAUUSD strategies with exit profiles."
---

# xauusd-strategy-spec

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Five Strategies
1. **STANDARD_SCALPING** — M5 decision, 10min TTL, tight SL
2. **ULTRA_SCALPING** — M1 decision, 5min TTL, ignores structure
3. **STANDARD_SWING** — H1 decision, 60min TTL
4. **TREND_SWING** — H4 decision, 240min TTL
5. **MARNIE_FIB** — Fibonacci-based supplementary

## Workflow
1. Keep behavior distinct per strategy.
2. Version timeframes/features/confluence/risk/execution/exit profiles.
3. Thresholds configuration-backed/versioned.
4. TP ladders, SL/invalidation, partial close, breakeven, trailing.
5. Gross/net R:R with spread/slippage/commission/carry.
6. MTF conflict/freshness and capability grade ceilings.

## Validate
Distinct golden fixtures, historically reproducible snapshots.
