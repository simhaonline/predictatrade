---
name: xauusd-strategy-spec
description: Specify four distinct versioned XAUUSD strategies with timeframes/features/confluence/risk/execution/prediction/exit profiles and net-expectancy logic.
---

# xauusd-strategy-spec

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Keep STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING and TREND_SWING behavior distinct.
2. Bind each version to prediction target, timeframe/session/feature/confluence/risk/execution/calibration/exit profiles.
3. Keep thresholds configuration-backed/versioned.
4. Use numeric feature/indicator/candle/structure/liquidity definitions.
5. Define TP ladders, SL/invalidation, partial close, breakeven and trailing.
6. Compute gross/net R:R with spread/slippage/commission/carry.
7. Encode MTF conflict/freshness and capability grade ceilings.
8. Frequency is never a quota; research patterns stay research until validated.

## Validate
Distinct golden fixtures and historically reproducible strategy snapshots.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
