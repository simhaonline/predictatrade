# Indicators and Features

**Version:** v1.2.0 — Advanced Risk + Backtesting  
**Date:** 18 August 2026

---

## Locally Computed Indicators (`features/indicators.go`)

| Indicator | Period | Formula | Used By |
|-----------|--------|---------|---------|
| EMA | 9, 21, 50, 100, 200 | EMA[t] = price×mult + EMA[t-1]×(1-mult) | All 4 strategies |
| SMA | 50, 100, 200 | Simple average | Swing, Trend |
| MACD | 12/26/9 | EMA12-EMA26, Signal=EMA9(MACD) | All 4 |
| ADX | 14 | Wilder's DX average | All 4 |
| +DI/-DI | 14 | +DM/ATR×100, -DM/ATR×100 | All 4 |
| RSI | 14 | Wilder's RS method | All 4 |
| Stochastic | 14/3/3 | %K = 100×(C-LL)/(HH-LL), Signal=SMA(%K,3) | Ultra |
| CCI | 20 | (TP-SMA)/(0.015×MeanDev) | Trend |
| ATR | 14 | Wilder's True Range average | All 4 |
| Bollinger | 20/2 | SMA±2×stdDev | Ultra |
| OBV | — | Cumulative tick volume | Informational |
| Parabolic SAR | 0.02/0.20 | Acceleration factor | Available, not in strategy evidence |
| Ichimoku | 9/26/52 | Tenkan/Kijun/Senkou | Available, not in strategy evidence |
| StochRSI | 14/14/3/3 | Stochastic of RSI | Available, not in strategy evidence |

## Structure Features (`features/structure.go`)
- BOS (Break of Structure)
- CHoCH (Change of Character)
- Swing highs/lows (fractal-based, 2-bar confirmation, no look-ahead)
- Current trend classification

## Liquidity Features (`features/liquidity.go`)
- Liquidity pools (equal highs/lows)
- Sweep detection (wick penetration + close back inside)
- CVD/DOM: UNSUPPORTED — never fabricated

## FVG Features (`features/fvg.go`)
- Fair Value Gaps
- Order Blocks
- Breaker Blocks

## Session Features (`features/session.go`)
- UTC-based session detection: TOKYO, LONDON, OVERLAP, NEW_YORK, SYDNEY
- Weekend detection
- News risk: NONE (simplified — no calendar feed)

## MTF Features (`features/mtf.go`)
- Per-timeframe state: +1 (bullish), -1 (bearish), 0 (neutral)
- Weighted alignment score: [-100, +100]
- Weights: M1=0.5, M5=1.0, M15=1.5, M30=1.0, H1=2.0, H4=2.5, D1=3.0

---

## PTB Advanced Modules (`ptb/modules.go`)

| Module | Status | Description |
|--------|--------|-------------|
| Liquidity Void | SHADOW | Displacement detection via body/ATR ratio |
| Wick Fill | SHADOW | Empirical wick statistics |
| Session Imbalance | SHADOW | Price vs VWAP normalized by ATR |
| Candle Range Projector | SHADOW | Current vs expected range |
| Time At Mode | SHADOW | Price occupancy (warming up) |
| Engineered Liquidity Proxy | SHADOW | Equal highs/lows detection |
| Market Phase | SHADOW | Accumulation/Expansion/Distribution |
| Relative Volume Flow | SHADOW | Tick volume proxy (NOT real volume) |
| Price Delivery | SHADOW | Session behavior analysis (warming up) |
| Stop Hunt Proxy | SHADOW | Sweep detection with quality score |
| Institutional Footprint | UNSUPPORTED | No DOM/Level2/T&S data |
| Time Cycle Analytics | SHADOW | Empirical time patterns (warming up) |
| Algo Activity Proxy | SHADOW | Tick arrival/burstiness proxy |
| Complete Liquidity Map | SHADOW | INFERRED_PRICE_STRUCTURE levels |
| Manipulation Proxy | SHADOW | Dislocation index 0-100 |
| MTF Bias Engine | SHADOW | Enhanced alignment with bias |
| Volatility Regime Engine | SHADOW | ATR/price classification |
| S/R Quality Engine | SHADOW | Level quality scoring |
| Microstructure Engine | SHADOW | Spread, tick balance, velocity |
| Statistical Performance | SHADOW | Outcome tracking |
| Data Quality Engine | ACTIVE | Measured quality from real attributes |

## PTB Synthesis Engine (`ptb/synthesis.go`)
- Confluence score (weighted component average)
- Bias determination (STRONG_LONG → STRONG_SHORT)
- Setup quality grading (A+ through F)
- Position size multiplier (advisory)
- Stop distance multiplier (context-aware)
- Action recommendation (ENTER, WAIT, AVOID)
- Reason codes (machine-readable)
- Market narrative (deterministic template)
- Enhanced regime (9 states)
- Gold role classification (UNKNOWN without data)
- Gold correlation engine (Pearson, awaiting feeds)
- Liquidity targeting (directional)
- Component score breakdown

## Enhanced Regime Classification

```
STRONG_TREND_UP, STRONG_TREND_DOWN, WEAK_TREND_UP, WEAK_TREND_DOWN,
RANGE_BOUND, HIGH_VOLATILITY, LOW_VOLATILITY, TRANSITIONING, MANIPULATION
```

## Gold Correlation Engine

Computes rolling Pearson correlations between gold and DXY, silver, and US10Y yields. All external feeds default to UNAVAILABLE unless connected through the Master Node. No fabricated correlations.

## Gold Role Classification

Determines what is driving XAUUSD: CURRENCY, SAFE_HAVEN, MONETARY_ASSET, COMMODITY, INFLATION_HEDGE, or UNKNOWN. Returns UNKNOWN when macro data is unavailable.

---

## Advanced Risk Modules (v1.1.0)

### Loss Recovery Manager (`recovery/manager.go`)
- State machine: NORMAL → RECOVERY → HALTED / DAILY_LIMIT
- Anti-martingale, anti-revenge-trading enforcement
- Correct daily loss PnL sign logic
- State isolation per account+strategy+symbol
- Duplicate broker close event deduplication
- Restart-safe state persistence

### Adaptation Manager (`adaptation/manager.go`)
- 6 market phases (TRENDING, RANGING, HIGH_VOLATILITY, LOW_VOLATILITY, MANIPULATIVE, UNCERTAIN)
- Dynamic parameter adjustment (stop distance, risk multiplier, confluence, confidence, weights)
- Risk clamping hierarchy (adaptive → strategy → account → global hard max)
- Deep copy of weights — never mutates base config
- NaN/Inf rejection, weight normalization

### Controlled Hedge Manager (`hedging/manager.go`)
- DISABLED BY DEFAULT
- Pre-execution checks: broker capability, netting mode, license, exposure, thresholds
- Partial hedging with size cap (never exceeds original)
- No martingale escalation
- Grid hedging OFF, options hedging OFF by default
- Auto-close on expiry, trailing stop support
- Full lifecycle audit trail

### ML Adaptation Manager (`ml/adaptation.go`)
- Model Registry with versioning, staleness detection, sample count validation
- Inference bounds clamping (stop 0.5-2.0, size 0.1-1.0, confluence 50-100)
- Python training pipeline with chronological split, walk-forward validation
- Data leakage protection

### RL Strategy Optimizer (`rl/optimizer.go`)
- 4 deployment modes: disabled, shadow, filter_only, live_approved
- Filter mode can only veto (NO_TRADE) — cannot create trades
- Live approval requires explicit authorization + validation metrics
- Python RL training environment with multi-component reward function

### Sentiment Engine (`sentiment/engine.go`)
- Async background refresh — NEVER blocks signal hot path
- Provider abstraction interface
- Timeout, retry/backoff, rate-limit handling, stale-data detection
- Provider health tracking with error counts
- Neutral/no-impact fallback when unavailable

### Daily Maintenance (`maintenance/scheduler.go`)
- UTC-based daily reset scheduler
- Idempotent (prevents duplicate execution in multi-instance deployments)
- Resets daily loss counters, clears DAILY_LIMIT state

---

## Backtesting Modules (v1.2.0)

### Data Layer (`research/src/patresearch/backtesting/data/`)
- `loader.py` — Historical data loading from database
- `quality.py` — Data quality validation
- `alignment.py` — Multi-timeframe alignment with no-lookahead guarantee
- `session_calendar.py` — Session and calendar engine

### Engine Layer (`research/src/patresearch/backtesting/engine/`)
- `core.py` — Event-driven backtesting core
- `execution.py` — Execution simulator (spread, slippage, commission, latency, partial fills)
- `portfolio.py` — Portfolio engine with trailing stop, break-even, time exit

### Strategy Layer (`research/src/patresearch/backtesting/strategy/`)
- `ptb_strategy.py` — PTB strategy adapter (live parity)
- `precomputed_ptb_strategy.py` — Precomputed PTB for replay
- `rl_strategy.py` — RL standalone and confirmation filter

### Analytics Layer (`research/src/patresearch/backtesting/analytics/`)
- `metrics.py` — Performance metrics (Sharpe, Sortino, profit factor, etc.)
- `walk_forward.py` — Walk-forward analysis with final holdout
- `monte_carlo.py` — Monte Carlo robustness analysis
- `sensitivity.py` — Parameter sensitivity analysis

### Support Modules
- `features/precompute.py` — PTB feature precomputation
- `reporting/report.py` — Report generator and run manifest
- `config/__init__.py` — Environment configuration
- `cli.py` — Command-line interface

---

## Unsupported Features (Honestly Labeled)

- Volume Profile: UNSUPPORTED (broker tick volume only)
- Cumulative Delta: UNSUPPORTED (no order-flow)
- Institutional Footprint: UNSUPPORTED (no DOM/Level2/T&S)
- DXY/Silver/Yields: Awaiting live Master Node feed
- COT Report: Not connected
- ML: DISABLED/RESEARCH (no trained model in production)
- Real Volume / Order Flow: Not available from current broker source
