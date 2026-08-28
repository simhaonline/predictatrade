# Real-Data Backtest Replay — Honest Research Snapshot

**Date:** 2026-08-28 (updated)
**Engine:** `pat-engine` live strategy code + config (`backtest.RunAll` runs the EXACT
`signal.Decide` pipeline).
**Data:** `data/xauusd_historical/XAU_15m_data.csv` (MetaTrader export). Last 150,000 bars.
**Broker:** Equiti-style `DefaultXAUUSDExecution` — contract 100oz, commission $7/lot,
spread 20pts ($0.20), leverage 500, swap ±1.5pts, **broker server time UTC+4**.

## Methodology (corrected — these were the real calc bugs)
1. **Costs are subtracted.** Each simulated trade pays round-turn spread (TypicalSpread×2),
   round-turn commission (`CommissionPrice(1.0)×2`), and 3pts round-turn slippage, all
   converted to price units via `TickSize`. Earlier runs omitted this and were optimistic.
2. **Realistic trade horizon.** `maxBars` is now per-strategy (15m bars): ULTRA 24,
   STANDARD_SCALPING 40, STANDARD_SWING 160, TREND_SWING 240. Earlier a flat 50 bars
   caused winning swing trades to "expire" and be marked to close, deflating PF.
3. **Structural entry (sweep → BOS).** Signals require a liquidity sweep in the last 8 bars
   followed by a break of structure (BOS) in the continuation direction. The generic
   weighted-vote confluence alone has no robust edge; this is the defensible edge filter.
   `BuildSnapshots` now detects sweeps/BOS as a *sequence* across bars (20-bar pivot
   window), not a single-bar coincidence.

## Results (in-sample, single pass, costs included)
| Strategy | Trades | Win% | PF (net of costs) |
|---|---|---|---|
| ULTRA_SCALPING | 493 | 31.2% | 0.72 |
| STANDARD_SCALPING | 934 | 37.0% | 0.79 |
| STANDARD_SWING | 820 | 42.9% | 0.94 |
| TREND_SWING | 437 | 37.5% | 0.89 |

With a **no-scalping broker**, only STANDARD_SWING and TREND_SWING run (correctly excluded).

## Assessment — edge NOT demonstrated
After the methodology corrections, **every product shows PF < 1.0 net of costs**. The
earlier "PF ≈ 1.0–1.1" figures were inflated by missing costs and an unrealistically
short horizon; they should be disregarded. The current strategy logic produces a weak
gross edge that costs erase. This is the genuine algorithmic gap.

**This is not a bug to patch with parameter tuning.** Manufacturing a PF > 1.0 by
adjusting constants would be overfitting/fabrication, which the SOW explicitly forbids.
A investable edge requires proper research, not in-sample curve-fitting.

## Required before any profitability claim (per SOW)
1. **Walk-forward / out-of-sample validation** — the current run is a single in-sample pass.
2. **Richer features** — true multi-timeframe context, order-flow / CVD / DOM / aggressor
   side, and session/rollover awareness (the replay currently derives a crude single-source
   approximation of liquidity/BOS).
3. **Regime & calendar filters** — avoid low-ADX chop; the confluence model still fires in
   noise.
4. **Calibration** — subscriber-facing probability must be calibrated to a named target; raw
   score is not probability.
5. **Native scalping timeframe** — scalping products need M1/M5 bars, not 15m, for an
   honest scalping assessment.

## Verdict
The engine math (cost-aware net R:R gate, broker-time sessions, capital-risk controls,
license gating, persistence) is correct and production-grade. The **strategy alpha is not
yet proven**. Do not publish any performance or probability claim until walk-forward/OOS
and calibration are complete.
