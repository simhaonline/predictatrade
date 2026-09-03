# Strategy Playbooks
## v1.26.0 — 03 September 2026

> v1.26 headline: STANDARD_SCALPING win-rate-first rebuild (ba83784) —
> parity backtest −45.97% PF 0.81 → +17.90% PF 1.30 (wr 57.4%, DD 7.86%).
> Cost-aware geometry (SL 0.8×ATR / TP1 1.2×ATR, mig 129), liquidity
> dead-zone block (02/19/21/23h UTC), OVEREXTENDED score-cap gate (≥62).

### Engine Inventory

| # | Engine | ID | TFs | Min Score | Expiry | Mode |
|---|--------|----|-----|:---------:|:------:|:----:|
| 1 | Standard Scalping | STANDARD_SCALPING | M1/M5 | 65 | 10m | LIVE |
| 2 | Ultra Scalping | ULTRA_SCALPING | M1 | 60 | 5m | LIVE |
| 3 | Standard Swing | STANDARD_SWING | M15/H1 | 68 | 30m | LIVE |
| 4 | Trend Swing | TREND_SWING | H1/H4 | 70 | 60m | LIVE |
| 5 | MARNIE_FIB | MARNIE_FIB | H1 | 70 | 60m | SHADOW |

### Per-Strategy Exit Specifications (v1.16.x)

Each strategy now has its own exit profile with defined TP1/TP2/TP3 levels and micro profit-taking targets. R:R ratios are computed per level from the strategy-specific stop distance.

| Engine | TP1 R:R | TP2 R:R | TP3 R:R | Micro TP | Partial Close |
|--------|:-------:|:-------:|:-------:|:--------:|:-------------:|
| Standard Scalping | 1.0x | 2.0x | 3.0x | Configurable | Configurable |
| Ultra Scalping | 0.75x | 1.5x | — | Configurable | Configurable |
| Standard Swing | 1.5x | 3.0x | — | Configurable | Configurable |
| Trend Swing | 2.0x | 4.0x | — | Configurable | Configurable |
| MARNIE_FIB | Per fib level | Per fib level | — | — | — |

### Standard Scalping (M1/M5) — v1.26 REBUILD (2026-09-03, ba83784)
- Personality: Quick scalping, high-frequency, low-exposure
- Decision TF: M1, HTF confirmation: M5
- Min ATR: 5 pips, Max spread: 2.5 pips
- Cost-aware geometry (mig 129): SL 0.8×ATR (stop_pct 0.08%), TP1 1.2×ATR
  (0.09%), TP2 2.0×ATR (0.15%), TP3 3.5×ATR (0.26%) — breakeven wr ≈44%
- Entry gate: momentum (OsMA/MACD) must not contradict direction + ADX≥20;
  liquidity dead-zones blocked (02/19/21/23h UTC); OVEREXTENDED gate at
  RawScore≥62 (mid-band momentum = entry; extreme = exit)
- Regime: TRENDING_BULLISH, TRENDING_BEARISH, RANGE, HIGH_VOLATILITY
- Backtest evidence (90d Q4-2025 parity): 61 trades, wr 57.4%, PF 1.30,
  +17.90%, maxDD 7.86% (was −45.97% PF 0.81 at TP1 2.5×ATR)

### Ultra Scalping (M1)
- Personality: Ultra-fast, lower thresholds, quick exits
- Decision TF: M1
- Min ATR: 3 pips, Max spread: 2.0 pips
- SL buffer: 1.0x ATR, TP1: 0.75x, TP2: 1.5x
- Candidate thresholds widened (v1.16.x) for broader BUY_CANDIDATE/SELL_CANDIDATE reach

### Standard Swing (M15/H1)
- Personality: Medium-term swings, structure-focused
- Decision TF: M15, HTF: H1
- Min ATR: 8 pips, Max spread: 3.0 pips
- SL buffer: 2.0x ATR, TP1: 1.5x, TP2: 3.0x

### Trend Swing (H1/H4)
- Personality: Long-term trend following, high conviction
- Decision TF: H1, HTF: H4
- Min ATR: 12 pips, Max spread: 3.5 pips
- SL buffer: 2.5x ATR, TP1: 2.0x, TP2: 4.0x

### MARNIE_FIB (H1) — SHADOW
- Personality: Fibonacci confluence trader
- Fibonacci levels: 38.2%, 50%, 61.8%, 78.6%
- Requires BOS/CHoCH + 2+ fib confluences
- Accumulating outcomes before activation
- Available on ELITE plan

### Micro Profit-Taking (v1.16.x)
Each strategy defines a `MicroTP` level — a first partial profit target before TP1. The engine computes `PartialClosePct` (fraction of position to close at MicroTP) per strategy, enabling risk-reducing partial exits while letting the remainder run to full TP targets.

### Signal Quality & Expectancy (v1.16.x)
Every signal now carries:
- **QualityGrade:** A+, A, B, or REJECTED — derived from evidence confluence, R:R quality, and gate pass/fail status
- **ExpectancyR (EV_R):** Expected value per unit risk = (P_win × AvgWinR) − (P_loss × AvgLossR) − CostR
- **ExpectancyScore:** 0-100 normalized score for sorting and comparison
- **CalibratedProbability:** Win probability from research-trained calibration model (shows "Pending" until validated per SOW §16, §36)
