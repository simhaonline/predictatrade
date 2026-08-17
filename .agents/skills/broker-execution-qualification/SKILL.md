---
name: broker-execution-qualification
description: Qualify each broker/strategy execution class using measured XAUUSD symbol economics, sizing, margin, order behavior, latency, spread/slippage/rejects and locality.
---

# broker-execution-qualification

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Read terminal/broker symbol specification; never assume universal XAUUSD economics.
2. Validate digits, tick/point/value, contract, volume, stops/freeze, fill/order modes, sessions/rollover and swaps.
3. Validate price-unit conversion, generalized sizing, margin and stop-out.
4. Model market/limit/stop including missed/partial fills.
5. Measure signal→Agent→EA→broker latency/jitter plus spread/slippage/rejection distributions by strategy/session/regime.
6. Assess VPS/locality and approve/reject broker-strategy class independently; ultra-scalping may fail while swing passes.

## Safety
No live qualification trades without separate explicit authorization.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
