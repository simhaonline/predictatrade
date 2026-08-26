---
name: broker-execution-qualification
description: "Qualify broker execution per strategy class."
---

# broker-execution-qualification

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Workflow
1. Read terminal/broker symbol specification; never assume universal XAUUSD economics.
2. Validate digits, tick/point/value, contract, volume, stops/freeze, fill/order modes, sessions/rollover, swaps.
3. Validate price-unit conversion, generalized sizing, margin and stop-out.
4. Model market/limit/stop including missed/partial fills.
5. Measure signal->Agent->EA->broker latency/jitter plus spread/slippage/rejection by strategy/session/regime.
6. Assess VPS/locality and approve/reject broker-strategy class independently.

## Safety
No live qualification trades without separate explicit authorization.

## Output Contract
SOW sections, files examined/changed, tests/checks + results, unresolved risks, next action.
