---
name: trading-risk-safety
description: Enforce hard-veto data/news/session/spread/slippage/cost/margin/exposure/broker/license/entitlement/TTL/idempotency/emergency safety.
---

# trading-risk-safety

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Enumerate mandatory gates per strategy/account/broker.
2. Gates are deterministic, cached/precomputed where required, freshness/version stamped and fail closed.
3. No synchronous external I/O in mandatory candidate gate evaluation.
4. Validate aggregate exposure, net R:R, cost-to-target, margin, stop-out and exit geometry.
5. Validate news/session/rollover, TTL/replay/idempotency and kill switches.
6. Test partial close/breakeven/trailing/reconciliation and emit machine-readable NO-TRADE/reject reasons.

## Validate
Gate p50/p95/p99, watchdog/degraded behavior and fail-closed semantics.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
