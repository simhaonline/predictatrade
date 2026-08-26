---
name: xauusd-market-data
description: "Implement XAUUSD data with provenance and quality."
---

# xauusd-market-data

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Data Sources
- Twelve Data API (primary: XAUUSD spot, DXY)
- FMP API (COT positioning)
- Broker MT5 tick feed (Windows Agent)
- Ollama (sentiment), ONNX (ML inference)

## Workflow
1. Inventory providers, datasets, licensing/retention and capabilities.
2. Separate spot bid/ask, broker tick-volume proxy, GC trades, macro and slow context.
3. Preserve event/source/receipt timestamps, sequence, provenance and quality.
4. Stale/divergence/skew/gap/outlier/duplicate/failover handling.
5. Reconcile historical ticks/candles; never overwrite disagreement silently.
6. Handle futures roll, basis, continuous series.
7. Timezone/DST-aware sessions/fixes/holidays/rollover.
8. Missing capability degrades or NO-TRADE.

## Forbidden
Broker tick volume as centralized volume; invented CVD/DOM/aggressor/depth; roll gaps as displacement.
