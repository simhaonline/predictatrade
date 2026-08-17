---
name: xauusd-market-data
description: Implement/review XAUUSD/GC data capabilities, provenance, quality, history, futures roll/basis, session/calendar and macro feeds without fabricated flow.
---

# xauusd-market-data

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Inventory providers, datasets, licensing/retention and capabilities.
2. Separate spot bid/ask, broker tick-volume proxy, GC trades/top/depth, macro and slow context.
3. Preserve event/source/receipt timestamps, sequence, provenance and quality.
4. Implement stale/divergence/skew/gap/outlier/duplicate/out-of-order/failover handling.
5. Reconcile historical ticks/candles; never overwrite disagreement silently.
6. Handle GC contract selection, roll, continuous series and spot-futures basis.
7. Use timezone/DST-aware sessions/fixes/holidays/rollover.
8. Missing required capability degrades or NO-TRADE.

## Forbidden
Broker tick volume as centralized volume; invented CVD/DOM/aggressor/depth; roll gaps as market displacement.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
