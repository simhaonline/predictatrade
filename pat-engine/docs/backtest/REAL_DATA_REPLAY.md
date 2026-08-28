# Real-Data Backtest Replay — Honest Research Snapshot

**Date:** 2026-08-28
**Engine:** `pat-engine` live strategy code + config (no separate research path — `backtest.RunAll` runs the EXACT `signal.Decide` pipeline).
**Data:** `data/xauusd_historical/XAU_15m_data.csv` (MetaTrader export, semicolon-delimited). Last 150,000 bars used (capped for repeatability).
**Broker:** Equiti-style `DefaultXAUUSDExecution` — contract 100oz, commission $7/lot, spread 0.20, leverage 500, swap ±1.5pts, **broker server time UTC+4**.

## Results (in-sample, single pass)

| Strategy | Trades | Win% | PF (gross) |
|---|---|---|---|
| ULTRA_SCALPING | 4282 | 33.6% | 1.07 |
| STANDARD_SCALPING | 9008 | 37.9% | 1.02 |
| STANDARD_SWING | 13875 | 41.1% | 1.03 |
| TREND_SWING | 49811 | 42.3% | 1.10 |

With a **no-scalping broker** (`AllowsScalping=false`), both scalping products are
correctly excluded (`BROKER_SCALPING_NOT_ALLOWED`) and only the two swing/trend
products remain.

## What is verified
- The **net R:R hard gate** (spread + commission + swap, expressed in price units via
  `ContractSize`) is live and filters every candidate; only trades with net R:R ≥
  `MinNetRR` (1.3 prod) proceed. This is the cost-aware discipline the prior design
  lacked.
- Broker scalping restriction is enforced before any signal is built.
- All four versioned strategy products produce directional signals on real XAUUSD data
  through the identical code path used live.

## Honest caveats (per SOW — these are NOT performance/probability claims)
1. **In-sample, single pass.** No walk-forward / out-of-sample split, no cross-validation.
2. **Exit fills are idealized.** `Simulate` hits SL/TP on the raw price level; it does
   not subtract per-trade spread/slippage on the exit, so PF is mildly optimistic.
   The entry gate already accounts for costs, which is why PFs are modest.
3. **State reconstruction is partial.** `BuildSnapshots` derives indicators, candle
   displacement, a lightweight BOS and liquidity sweeps, and an approximate session
   label from the bar timestamp. It does NOT model multi-timeframe context, order
   flow / CVD / DOM, or news risk — those inputs are absent from the historical file.
4. **Raw score ≠ probability.** Per SOW, subscriber-facing probability requires named
   calibration against a prediction target; none is published here.
5. **Timeframe mismatch.** Strategies are evaluated on a uniform 15m bar feed as a test
   bed; scalping products are designed for M1/M5 decision contexts.

## Required before any external claim
Walk-forward + OOS, spread/slippage realism on exits, calibration of the published
probability, and a replay on the native scalping timeframe (M1/M5) for the scalping
products.
