---
name: trading-risk-safety
description: "Enforce hard-veto safety gates for trading."
---

# trading-risk-safety

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## 14-Gate Registry (short-circuit order)
1. DataQuality -> 2. Session -> 3. News -> 4. Spread -> 5. Slippage -> 6. TotalCost -> 7. MinATR -> 8. StopHuntFilter -> 9. Exposure -> 10. Margin -> 11. RRNExpectancy -> 12. Entitlement -> 13. License -> 14. ExecutionPermit

## Workflow
1. Enumerate mandatory gates per strategy/account/broker.
2. Gates are deterministic, cached/precomputed, freshness/version stamped, fail closed.
3. No synchronous external I/O in candidate gate evaluation.
4. Validate aggregate exposure, net R:R, cost-to-target, margin, stop-out, exit geometry.
5. Validate news/session/rollover, TTL/replay/idempotency, emergency stop and kill switch.
6. NO-TRADE is a valid first-class result.
