# Predict-A-Trade — Signal Generation Pipeline & Engine Report

**Version:** v1.0.0  
**Date:** 2026-08-24  
**Scope:** XAUUSD-only signal generation from tick to delivery  

---

## Table of Contents

1. [Architecture Overview](#1-architecture-overview)
2. [Signal Generation Pipeline (End-to-End)](#2-signal-generation-pipeline-end-to-end)
3. [Indicator Engine — 42+ Indicators](#3-indicator-engine--42-indicators)
4. [Feature Registry & Market State](#4-feature-registry--market-state)
5. [Strategy Engine — 5 Strategies](#5-strategy-engine--5-strategies)
6. [Scoring System & Thresholds](#6-scoring-system--thresholds)
7. [Hard Gates — 14 Gates](#7-hard-gates--14-gates)
8. [Cross-Market Confluence Engine](#8-cross-market-confluence-engine)
9. [Signal Persistence & Delivery](#9-signal-persistence--delivery)
10. [Outcome Resolver & Validation](#10-outcome-resolver--validation)
11. [File Map](#11-file-map)

---

## 1. Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────┐
│                        SIGNAL GENERATION PIPELINE                    │
│                                                                      │
│  MT5 Master Node (tick feed)                                        │
│       │                                                              │
│       ▼                                                              │
│  Go Realtime Engine (realtime/)                                     │
│       │                                                              │
│       ├── Candle Aggregation (M1/M5/M15/H1/H4/D1/W1)               │
│       │       │                                                      │
│       │       ▼                                                      │
│       ├── Feature Registry → MarketState                             │
│       │       ├── IndicatorEngine (42+ indicators)                  │
│       │       ├── StructureEngine (BOS/CHoCH/swings)               │
│       │       ├── LiquidityEngine (pools/sweeps)                    │
│       │       ├── FVGEngine (fair value gaps/OBs)                   │
│       │       ├── VWAPEngine (session/rolling VWAP)                 │
│       │       ├── RegimeEngine (trend/range/volatility)             │
│       │       ├── MTFEngine (multi-timeframe alignment)             │
│       │       ├── SessionEngine (sessions/news/exchanges)           │
│       │       ├── SAREngine (Parabolic SAR)                         │
│       │       ├── IchimokuEngine (cloud system)                     │
│       │       ├── StochRSIEngine (stochastic RSI)                   │
│       │       ├── FibonacciEngine (retracement levels)              │
│       │       ├── PivotEngine (daily/weekly pivots)                 │
│       │       ├── CandleEngine (candle intelligence)                │
│       │       ├── MarnieFibEngine (Marnie Fib retracement/ext)      │
│       │       └── RollingStats (Z-scores for OBV/vol/BB width)      │
│       │       │                                                      │
│       │       ▼                                                      │
│       ├── Strategy Engine (5 strategies evaluate MarketState)       │
│       │       ├── STANDARD_SCALPING  → StrategyResult               │
│       │       ├── ULTRA_SCALPING     → StrategyResult               │
│       │       ├── STANDARD_SWING     → StrategyResult               │
│       │       ├── TREND_SWING        → StrategyResult               │
│       │       └── MARNIE_FIB         → StrategyResult               │
│       │       │                                                      │
│       │       ▼                                                      │
│       ├── Scoring: scoreDirectionWithThresholds                     │
│       │       ├── Long/Short score aggregation                      │
│       │       ├── Family caps (anti-double-count)                   │
│       │       ├── PHI threshold function                            │
│       │       ├── Dominance check (min margin)                      │
│       │       └── Candidate vs Trade classification                 │
│       │       │                                                      │
│       │       ▼                                                      │
│       ├── Cross-Market Confluence (SHADOW mode)                     │
│       │       ├── 7 drivers: DXY, EURUSD, COT, BTC, Oil, Yield, VIX│
│       │       ├── Bounded score adjustment (-15..+10)              │
│       │       └── Never blocks signal generation                    │
│       │       │                                                      │
│       │       ▼                                                      │
│       ├── Hard Gates (14 deterministic gates)                       │
│       │       ├── DataQuality, Session, News, Spread               │
│       │       ├── Slippage, TotalCost, Exposure, Margin            │
│       │       ├── RRNetExpectancy, Entitlement, License             │
│       │       ├── ExecutionPermit, StopHuntFilter, MinATR           │
│       │       └── Fail-closed: any gate fail → NO-TRADE            │
│       │       │                                                      │
│       │       ▼                                                      │
│       ├── Signal Engine                                             │
│       │       ├── Cooldown manager (per-strategy)                   │
│       │       ├── Duplicate checker (idempotency)                   │
│       │       ├── Calibration (probability mapping)                 │
│       │       ├── Exit profile (SL/TP geometry)                     │
│       │       └── Signal persistence (PostgreSQL/TimescaleDB)       │
│       │       │                                                      │
│       │       ▼                                                      │
│       ├── Delivery Layer                                            │
│       │       ├── WebSocket Hub → Frontend dashboard                │
│       │       ├── Agent Hub → Windows Agent → MT4/MT5 EA           │
│       │       └── NTFy push notifications                           │
│       │                                                              │
│       └── Outcome Resolver (background, SHADOW validation)          │
│               ├── Tracks TP1/TP2/TP3/SL/Expiry                     │
│               └── Resolves XAUUSD shadow snapshots                  │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Signal Generation Pipeline (End-to-End)

### Stage 1: Tick Ingestion
- **Source:** MT5 Master Node EA → Windows Agent → Go Engine WebSocket
- **Data:** Bid, Ask, Timestamp (UTC)
- **Quality:** `QualityAuthoritative` when from live MT5; degrades on staleness
- **File:** `realtime/cmd/realtime-engine/main.go` (agent WebSocket handler)

### Stage 2: Candle Aggregation
- Timeframes: M1, M5, M15, H1, H4, D1, W1
- Each candle: Open, High, Low, Close, Volume (tick volume), Time, IsClosed
- Candles stored in `MarketState.Candles[timeframe]`

### Stage 3: Feature Evaluation (Registry.Evaluate)
- **Input:** Completed candle + all timeframe candles + last tick
- **Output:** `MarketState` struct containing ALL features
- **Process:** Each engine processes the candle independently, results are assembled
- **File:** `realtime/internal/features/registry.go`

```go
func (r *Registry) Evaluate(candle *types.Candle, allCandles map[types.Timeframe]*types.Candle, lastTick *types.Tick) *MarketState
```

### Stage 4: Strategy Evaluation
- All 5 strategies evaluate the same `MarketState`
- Each strategy produces `StrategyResult` with direction, scores, evidence, geometry
- **File:** `realtime/internal/strategy/strategies.go`, `realtime/internal/strategy/marnie_fib.go`

### Stage 5: Scoring & Classification
- `scoreDirectionWithThresholds()` aggregates evidence → long/short scores
- PHI threshold function prevents single-family dominance
- Dominance check ensures sufficient directional margin
- Classification: `BUY` / `SELL` (trade threshold) vs `BUY_CANDIDATE` / `SELL_CANDIDATE` (candidate threshold) vs `NO-TRADE`

### Stage 6: Cross-Market Confluence (SHADOW)
- Reads cached driver snapshots (no external I/O on hot path)
- Produces bounded score adjustment
- Currently SHADOW mode: no production signal impact
- **File:** `realtime/internal/crossmarket/engine.go`

### Stage 7: Hard Gate Evaluation
- 14 deterministic gates evaluate the signal
- Any gate failure → signal rejected or downgraded
- Gates are fail-closed (UNKNOWN = reject)
- **File:** `realtime/internal/gates/`

### Stage 8: Signal Persistence
- Signal saved to PostgreSQL/TimescaleDB
- Includes all evidence, gate results, timestamps, geometry
- **File:** `realtime/internal/signal/`

### Stage 9: Signal Delivery
- WebSocket Hub broadcasts to frontend dashboard
- Agent Hub sends to Windows Agent → MT4/MT5 EA
- NTFy push notifications for qualifying signals
- **File:** `realtime/cmd/realtime-engine/main.go` (`broadcastSignalToAll`)

---

## 3. Indicator Engine — 42+ Indicators

### 3.1 Trend Indicators

| # | Indicator | Period | Formula | Source | File |
|---|-----------|--------|---------|--------|------|
| 1 | EMA9 | 9 | Exponential MA, α=2/(9+1) | Local | `indicators.go` |
| 2 | EMA21 | 21 | Exponential MA, α=2/(21+1) | Local | `indicators.go` |
| 3 | EMA50 | 50 | Exponential MA, α=2/(50+1) | Local | `indicators.go` |
| 4 | EMA100 | 100 | Exponential MA, α=2/(100+1) | Local | `indicators.go` |
| 5 | EMA200 | 200 | Exponential MA, α=2/(200+1) | Local | `indicators.go` |
| 6 | EMACross921 | — | Bullish cross event: prev EMA9 ≤ EMA21 AND curr EMA9 > EMA21 | Local | `indicators.go` |
| 7 | SMA50 | 50 | Simple Moving Average | Local | `indicators.go` |
| 8 | SMA100 | 100 | Simple Moving Average | Local | `indicators.go` |
| 9 | SMA200 | 200 | Simple Moving Average | Local | `indicators.go` |
| 10 | MACD Main | 12,26 | EMA12 - EMA26 | Local | `indicators.go` |
| 11 | MACD Signal | 9 | EMA9 of MACD line history | Local | `indicators.go` |
| 12 | MACD Histogram | — | MACD Main - MACD Signal | Local | `indicators.go` |
| 13 | MACD Bull Cross | — | MACD crossed above Signal line | Local | `indicators.go` |
| 14 | MACD Bear Cross | — | MACD crossed below Signal line | Local | `indicators.go` |
| 15 | ADX | 14 | Wilder's ADX with +DI/-DI | Local | `indicators.go` |
| 16 | ADX Plus DI | 14 | Wilder's +DI | Local | `indicators.go` |
| 17 | ADX Minus DI | 14 | Wilder's -DI | Local | `indicators.go` |
| 18 | Parabolic SAR | AF=0.02, Max=0.20 | Parabolic Stop and Reverse | Local | `sar.go` |
| 19 | Parabolic SAR Long | — | Is SAR in long mode | Local | `sar.go` |
| 20 | Ichimoku Tenkan | 9 | (9-period H + 9-period L) / 2 | Local | `ichimoku.go` |
| 21 | Ichimoku Kijun | 26 | (26-period H + 26-period L) / 2 | Local | `ichimoku.go` |
| 22 | Ichimoku Senkou A | — | (Tenkan + Kijun) / 2, displaced +26 | Local | `ichimoku.go` |
| 23 | Ichimoku Senkou B | 52 | (52-period H + 52-period L) / 2, displaced +26 | Local | `ichimoku.go` |
| 24 | Ichimoku Chikou | 26 | Close displaced -26 | Local | `ichimoku.go` |
| 25 | Ichimoku Cloud Top | — | max(SenkouA, SenkouB) | Local | `ichimoku.go` |
| 26 | Ichimoku Cloud Bot | — | min(SenkouA, SenkouB) | Local | `ichimoku.go` |
| 27 | Ichimoku Above Cloud | — | Close > CloudTop | Local | `ichimoku.go` |
| 28 | Ichimoku Below Cloud | — | Close < CloudBot | Local | `ichimoku.go` |
| 29 | Ichimoku In Cloud | — | CloudBot ≤ Close ≤ CloudTop | Local | `ichimoku.go` |

### 3.2 Momentum Indicators

| # | Indicator | Period | Formula | Source | File |
|---|-----------|--------|---------|--------|------|
| 30 | RSI | 14 | Wilder's RSI | Local | `indicators.go` |
| 31 | Stochastic %K | 14 | 100*(C-LowestLow)/(HighestHigh-LowestLow) | Local | `indicators.go` |
| 32 | Stochastic %D | 3 | 3-period SMA of %K | Local | `indicators.go` |
| 33 | StochRSI Raw | 14 | (RSI - min(RSI,14)) / (max(RSI,14) - min(RSI,14)) | Local | `stochrsi.go` |
| 34 | StochRSI K | 3 | 3-period SMA of StochRSI Raw | Local | `stochrsi.go` |
| 35 | StochRSI D | 3 | 3-period SMA of StochRSI K | Local | `stochrsi.go` |
| 36 | CCI | 20 | (TP - SMA(TP)) / (0.015 * MeanDeviation) | Local | `indicators.go` |

### 3.3 Volatility Indicators

| # | Indicator | Period | Formula | Source | File |
|---|-----------|--------|---------|--------|------|
| 37 | ATR | 14 | Wilder's ATR: TR = max(H-L, |H-prevC|, |L-prevC|) | Local | `indicators.go` |
| 38 | Bollinger Upper | 20, 2σ | SMA20 + 2*StdDev | Local | `indicators.go` |
| 39 | Bollinger Lower | 20, 2σ | SMA20 - 2*StdDev | Local | `indicators.go` |
| 40 | Bollinger Middle | 20 | SMA20 | Local | `indicators.go` |
| 41 | Bollinger Width | — | (Upper - Lower) / Middle | Local | `indicators.go` |
| 42 | Bollinger Bull Rev | — | Close crossed back above lower band | Local | `indicators.go` |
| 43 | Bollinger Bear Rev | — | Close crossed back below upper band | Local | `indicators.go` |

### 3.4 Volume Indicators

| # | Indicator | Period | Formula | Source | Provenance | File |
|---|-----------|--------|---------|--------|------------|------|
| 44 | OBV | — | Cumulative: +Vol on up-close, -Vol on down-close | Local | TICK_VOLUME | `indicators.go` |
| 45 | OBV Z-Score | 50 | Rolling Z-score of OBV (min 20 samples) | Local | TICK_VOLUME | `registry.go` |
| 46 | Tick Volume Z-Score | 50 | Rolling Z-score of tick volume | Local | TICK_VOLUME | `registry.go` |
| 47 | BB Width Z-Score | 50 | Rolling Z-score of Bollinger width | Local | — | `registry.go` |
| 48 | Session VWAP | — | ∑(TypicalPrice × Volume) / ∑Volume, session-anchored | Local/MT5 | TICK_VOLUME | `vwap.go` |
| 49 | VWAP Upper Band | — | VWAP + band | Local/MT5 | — | `vwap.go` |
| 50 | VWAP Lower Band | — | VWAP - band | Local/MT5 | — | `vwap.go` |
| 51 | Rolling VWAP | — | Rolling VWAP | Local/MT5 | — | `vwap.go` |

> **UNAVAILABLE indicators (never fabricated):**
> - Volume Profile (requires real volume; broker provides tick volume only)
> - Cumulative Delta / DOM / Order Flow (requires centralized exchange data)
> - Aggressor-side data

### 3.5 Structure & Liquidity Features

| Feature | Description | File |
|---------|-------------|------|
| Swing Highs/Lows | Confirmed pivot points from StructureEngine | `structure.go` |
| BOS (Break of Structure) | Trend continuation event | `structure.go` |
| CHoCH (Change of Character) | Trend reversal event | `structure.go` |
| MSS (Market Structure Shift) | Major structure shift | `structure.go` |
| Current Trend | "bullish" / "bearish" / "" | `structure.go` |
| Liquidity Pools | Equal highs/lows as liquidity targets | `liquidity.go` |
| Liquidity Sweeps | Detect when price wicks past a pool | `liquidity.go` |
| FVG (Fair Value Gaps) | Imbalance zones | `fvg.go` |
| Order Blocks | Last opposing candle before displacement | `fvg.go` |
| Breakers | Failed order blocks that become support/resistance | `fvg.go` |

### 3.6 Fibonacci & Pivot Features

| Feature | Description | File |
|---------|-------------|------|
| Fibonacci Retracement | 0.236, 0.382, 0.500, 0.618, 0.786 from confirmed swings | `fibonacci.go` |
| Marnie Fib Retracement | 0.236, 0.382, 0.5, 0.618, 0.786, 1.0 | `marnie_fib.go` |
| Marnie Fib Extension | 1.272, 1.618, 2.618 | `marnie_fib.go` |
| Marnie Golden Zone | 0.618 - 0.786 (high-probability reversal area) | `marnie_fib.go` |
| Marnie Confluence Score | 0-100 based on price proximity to golden zone | `marnie_fib.go` |
| Daily Pivots | P, R1-R3, S1-S3 from previous completed day OHLC | `pivots.go` |
| Weekly Pivots | P, R1-R3, S1-S3 from previous completed week OHLC | `pivots.go` |

### 3.7 Regime & Session Features

| Feature | Description | File |
|---------|-------------|------|
| Regime | TRENDING_BULLISH, TRENDING_BEARISH, RANGE, MEAN_REVERSION, BREAKOUT, HIGH_VOLATILITY | `regime.go` |
| MTF Score | Multi-timeframe alignment score (0-100) | `mtf.go` |
| MTF States | Per-timeframe state map | `mtf.go` |
| Current Session | LONDON, NEW_YORK, OVERLAP, TOKYO, SYDNEY, OFF_HOURS | `session.go` |
| Is Overlap | London + NY overlap flag | `session.go` |
| Is Weekend | Saturday/Sunday flag | `session.go` |
| News Risk | NONE, LOW, MEDIUM, HIGH from economic calendar | `session.go` |

### 3.8 Candle Intelligence

| Feature | Description | File |
|---------|-------------|------|
| Body Size, Upper/Lower Wick | Candle anatomy | `candle.go` |
| Body/Range Ratio | Body as fraction of total range | `candle.go` |
| Pattern Detection | Doji, PinBar, Engulfing, InsideBar, OutsideBar | `candle.go` |
| Displacement/Rejection/Breakout | Price action signals | `candle.go` |
| Compression/Expansion | Volatility regime candle patterns | `candle.go` |
| ATR Normalized | Candle range normalized by ATR | `candle.go` |
| Consecutive Bull/Bear | Streak counting | `candle.go` |

---

## 4. Feature Registry & Market State

### Registry (`realtime/internal/features/registry.go`)

The `Registry` orchestrates all feature engines. On each completed candle:

```go
func (r *Registry) Evaluate(candle, allCandles, lastTick) *MarketState {
    // 1. Structure (swings, BOS, CHoCH)
    structure := r.structureEngine.Process(candle)
    
    // 2. Liquidity (pools, sweeps)
    liquidity := r.liquidityEngine.Process(candle, structure.SwingHighs)
    
    // 3. FVG / Order Blocks
    fvg := r.fvgEngine.Process(candle)
    
    // 4. VWAP
    vwap := r.vwapEngine.Process(candle)
    
    // 5. Core Indicators (42+)
    indicators := r.indicatorEngine.Process(candle)
    
    // 6. Regime classification
    regime := r.regimeEngine.Process(candle, indicators)
    
    // 7. MTF alignment
    mtf := r.mtfEngine.Process(allCandles)
    
    // 8. Session & news
    session := r.sessionEngine.Process(candle.Time)
    
    // 9. SAR, Ichimoku, StochRSI
    sar := r.sarEngine.Process(candle)
    ichimoku := r.ichimokuEngine.Process(candle)
    stochRSI := r.stochRSIEngine.Process(candle)
    
    // 10. Wire into IndicatorFeatures
    indicators.ParabolicSAR = sar.Value
    indicators.IchimokuTenkan = ichimoku.Tenkan
    // ... (all new indicators wired)
    
    // 11. Rolling Z-scores
    indicators.OBVZScore = r.obvZScore.ZScoreDecimal(indicators.OBV)
    indicators.TickVolumeZScore = ...
    indicators.BBWidthZScore = ...
    
    // 12. Fibonacci from confirmed structure
    fib := r.fibonacciEngine.Process(candle, structure)
    
    // 13. Daily/Weekly pivots
    pivots := r.pivotEngine.Process(candle)
    
    // 14. Candle intelligence
    candleIntel := r.candleEngine.Process(candle, indicators.ATR)
    
    // 15. Feature readiness map
    readiness := r.buildFeatureReadiness(...)
    
    // 16. Assemble MarketState
    return &MarketState{
        Symbol, Timestamp, LastTick, CurrentPrice,
        Bid, Ask, Spread, Mid,
        Candles, Structure, Liquidity, FVG, VWAP,
        Indicators, Regime, MTF, Session, Candle,
        Fibonacci, Pivots, FeatureReadiness, Quality,
    }
}
```

### MarketState Struct

The `MarketState` is the single source of truth passed to all strategies:

| Field | Type | Description |
|-------|------|-------------|
| Symbol | string | "XAUUSD" |
| Timestamp | time.Time | Candle timestamp (UTC) |
| LastTick | *Tick | Most recent bid/ask |
| CurrentPrice | decimal | Mid price |
| Bid/Ask/Spread/Mid | decimal | Live quote |
| Candles | map[Timeframe]*Candle | All timeframe candles |
| Structure | StructureFeatures | Swings, BOS, CHoCH, trend |
| Liquidity | LiquidityFeatures | Pools, sweeps (CVD/DOM UNAVAILABLE) |
| FVG | FVGFeatures | Fair value gaps, order blocks |
| VWAP | VWAPFeatures | Session/rolling VWAP + bands |
| Indicators | IndicatorFeatures | 42+ indicators |
| Regime | RegimeFeatures | Current regime + transition |
| MTF | MTFFeatures | Multi-timeframe alignment |
| Session | SessionFeatures | Session/overlap/news |
| Candle | CandleIntelligence | Candle anatomy + patterns |
| Fibonacci | FibonacciFeatures | Retracement levels |
| Pivots | PivotFeatures | Daily/weekly pivots |
| FeatureReadiness | map[string]FeatureReadiness | Per-feature health |
| Quality | QualityState | Data quality level |

---

## 5. Strategy Engine — 5 Strategies

All strategies implement the `Strategy` interface:

```go
type Strategy interface {
    ID() types.StrategyID
    Evaluate(state *features.MarketState) StrategyResult
}
```

### 5.1 STANDARD_SCALPING

| Property | Value |
|----------|-------|
| Decision TFs | M5, M15 |
| Context TFs | M15, H1 |
| Min Confluence | 45 (configurable per regime) |
| ATR Multiplier SL | 1.2 |
| ATR Multiplier TP1/TP2/TP3 | 1.8 / 3.0 / 5.0 |
| Max Spread | 3.5 pips |
| Max Slippage | 15 points |
| Min ADX | 20 |
| Min RR | 1.5 |
| Expiry | 30 min |
| Cooldown | 45 min |
| Accepted Regimes | Trending, Breakout |
| Accepted Sessions | London, NY, Overlap |
| **Focus** | Fast scalps on momentum + structure alignment in trending markets |

**Evidence pillars used:** TREND (EMA, MACD, ADX), MOMENTUM (RSI, Stoch), STRUCTURE (BOS), MTF (alignment), VWAP (deviation), CANDLE (displacement), VOLATILITY (ATR, BB)

### 5.2 ULTRA_SCALPING

| Property | Value |
|----------|-------|
| Decision TFs | M1, M5 |
| Context TFs | M5, M15 |
| Min Confluence | 50 (configurable per regime) |
| ATR Multiplier SL | 1.0 |
| ATR Multiplier TP1/TP2/TP3 | 1.5 / 2.5 / 4.0 |
| Max Spread | 2.5 pips |
| Max Slippage | 10 points |
| Min ADX | 25 |
| Min RR | 1.5 |
| Expiry | 15 min |
| Cooldown | 30 min |
| Accepted Regimes | Trending, Breakout, MeanReversion |
| Accepted Sessions | London, NY, Overlap |
| **Focus** | Ultra-fast scalps on M1/M5 with tight spread requirements |

### 5.3 STANDARD_SWING

| Property | Value |
|----------|-------|
| Decision TFs | M15, H1 |
| Context TFs | H1, H4, D1 |
| Min Confluence | 40-55 (regime-dependent) |
| ATR Multiplier SL | 1.5 |
| ATR Multiplier TP1/TP2/TP3 | 2.5 / 4.5 / 7.5 |
| Max Spread | 5.0 pips |
| Max Slippage | 25 points |
| Min ADX | 18 |
| Min RR | 2.0 |
| Expiry | 180 min |
| Cooldown | 240 min |
| Accepted Regimes | Trending, Breakout, Range, MeanReversion |
| Accepted Sessions | London, NY, Overlap, Tokyo |
| **Focus** | Swing trades with wider stops, range-adaptive evidence in RANGE regime |

### 5.4 TREND_SWING

| Property | Value |
|----------|-------|
| Decision TFs | H1, H4 |
| Context TFs | D1, W1 |
| Min Confluence | 50 (configurable per regime) |
| ATR Multiplier SL | 2.0 |
| ATR Multiplier TP1/TP2/TP3 | 3.5 / 6.0 / 10.0 |
| Max Spread | 6.0 pips |
| Max Slippage | 30 points |
| Min ADX | 22 |
| Min RR | 2.5 |
| Expiry | 720 min (12 hours) |
| Cooldown | 720 min |
| Accepted Regimes | Trending, Breakout only |
| Accepted Sessions | London, NY, Overlap, Tokyo, Sydney |
| **Focus** | Position trades on H1/H4 with D1/W1 context; trending/breakout only |

### 5.5 MARNIE_FIB (NEW)

| Property | Value |
|----------|-------|
| Decision TFs | M15, H1 |
| Context TFs | H4, D1 |
| Min Confluence | 40 (regime-dependent: 25-45) |
| ATR Multiplier SL | 1.5 |
| ATR Multiplier TP1/TP2/TP3 | 2.0 / 3.5 / 5.5 |
| Max Spread | 4.0 pips |
| Max Slippage | 20 points |
| Min ADX | 15 (Fib works in ranging markets) |
| Min RR | 2.0 |
| Expiry | 120 min |
| Cooldown | 180 min |
| Accepted Regimes | Range, MeanReversion, Trending, Breakout, HighVol |
| Accepted Sessions | London, NY, Overlap, Tokyo, Sydney |
| **Focus** | Fibonacci retracement reversal entries in golden zone (0.618-0.786) |

**Marnie Fib Engine** (`realtime/internal/features/marnie_fib.go`):

```
Retracement Ratios: 0.236, 0.382, 0.5, 0.618, 0.786, 1.0
Extension Ratios:   1.272, 1.618, 2.618
Golden Zone:        0.618 - 0.786 (high-probability reversal area)
```

**Scoring logic:**
- Price IN golden zone (0.618-0.786) → confluence score = 100, high-weight evidence
- Price NEAR golden zone (confluence > 50) → moderate evidence
- Price DISTANT (confluence > 25) → low evidence
- At exact Fib level (< 1% of range) → additional evidence
- Multi-level confluence (alignment with previous Fib levels) → +10 score boost
- TP targets override with Fib extension levels (1.272, 1.618, 2.618) when better

**Current state:** Returns `FIB_NO_SWING_ANCHORS` when structure engine has not yet populated swing highs/lows. Once structure data is available, the strategy generates directional signals.

---

## 6. Scoring System & Thresholds

### 6.1 Evidence Aggregation

Each strategy collects `EvidenceContribution` entries from multiple pillars:

```go
type EvidenceContribution struct {
    Pillar          string          // TREND, MOMENTUM, VOLATILITY, etc.
    Feature         string          // EMA9_ABOVE_EMA21, RSI_OVERSOLD, etc.
    Direction       Direction       // BUY or SELL
    Weight          decimal         // pillar weight
    Contribution    decimal         // normalized contribution (0-1)
    NormalizedValue decimal         // same as Contribution (used by confluence engine)
    Quality         QualityState
    Source          string
    Version         string
    ReasonCode      string
}
```

### 6.2 Family Caps (Anti-Double-Counting)

Correlated indicators within the same family are capped:

| Family | Max Total Contribution |
|--------|----------------------|
| TREND | 0.25 |
| MOMENTUM | 0.20 |
| VOLATILITY | 0.10 |
| VWAP | 0.10 |
| STRUCTURE | 0.20 |
| LIQUIDITY | 0.15 |
| SMC | 0.15 |
| MTF | 0.15 |
| CANDLE | 0.15 |
| REGIME | 0.10 |
| ML | 0.25 |
| SENTIMENT | 0.25 |
| FIBONACCI | (no explicit cap — counted separately) |

If a family's total exceeds its cap, all contributions in that family are proportionally scaled down.

### 6.3 PHI Threshold Function

Prevents single-family dominance — requires MOST families positive:

```
PHI(x) = 0                           if x < 0.65
PHI(x) = 0.65 + 0.35*(x-0.65)/0.35   if 0.65 ≤ x ≤ 1.0
PHI(x) = 1.0                          if x > 1.0
```

### 6.4 scoreDirectionWithThresholds

```go
func scoreDirectionWithThresholds(evidence, candidateThreshold, tradeThreshold, conflictPenalty) {
    // 1. Sum BUY contributions → longScore
    // 2. Sum SELL contributions → shortScore
    // 3. Scale to 0-100 (multiply by 100)
    // 4. Apply conflict penalty to dominant side
    // 5. Dominance check: |longScore - shortScore| must exceed MinDominanceMargin
    // 6. If dominant score >= candidateThreshold AND dominance OK:
    //      → return BUY or SELL (directional)
    //    Else:
    //      → return NO-TRADE with reason
}
```

### 6.5 Regime-Specific Thresholds

**File:** `realtime/internal/strategy/regime_thresholds.go`

Thresholds are mathematically justified per regime based on maximum achievable evidence budget:

| Strategy | Regime | Candidate | Trade | Max Score | Reason |
|----------|--------|-----------|-------|-----------|--------|
| STANDARD_SCALPING | TRENDING_BULL | 40 | 65 | ~80 | All trend evidence aligns |
| STANDARD_SCALPING | TRENDING_BEAR | 40 | 65 | ~80 | All trend evidence aligns |
| STANDARD_SCALPING | BREAKOUT | 40 | 65 | ~75 | BOS + displacement |
| STANDARD_SCALPING | RANGE | 30 | 45 | ~47 | Evidence split + family caps |
| STANDARD_SCALPING | MEAN_REVERSION | 30 | 45 | ~47 | Same as RANGE |
| STANDARD_SCALPING | HIGH_VOL | 30 | 50 | ~55 | Reduced evidence quality |
| ULTRA_SCALPING | TRENDING_BULL | 40 | 65 | ~78 | |
| ULTRA_SCALPING | TRENDING_BEAR | 40 | 65 | ~78 | |
| ULTRA_SCALPING | BREAKOUT | 40 | 65 | ~72 | |
| ULTRA_SCALPING | MEAN_REVERSION | 35 | 50 | ~52 | |
| ULTRA_SCALPING | RANGE | 35 | 50 | ~52 | |
| ULTRA_SCALPING | HIGH_VOL | 35 | 55 | ~58 | |
| STANDARD_SWING | TRENDING_BULL | 35 | 55 | ~92 | Wide evidence budget |
| STANDARD_SWING | TRENDING_BEAR | 35 | 55 | ~92 | |
| STANDARD_SWING | BREAKOUT | 35 | 55 | ~85 | |
| STANDARD_SWING | RANGE | 25 | 40 | ~60 | Range-adaptive evidence |
| STANDARD_SWING | MEAN_REVERSION | 25 | 40 | ~60 | |
| STANDARD_SWING | HIGH_VOL | 30 | 50 | ~65 | |
| TREND_SWING | TRENDING_BULL | 30 | 50 | ~75 | |
| TREND_SWING | TRENDING_BEAR | 30 | 50 | ~75 | |
| TREND_SWING | BREAKOUT | 30 | 50 | ~70 | |
| MARNIE_FIB | RANGE | 25 | 40 | ~55 | Fib retracement reversals |
| MARNIE_FIB | MEAN_REVERSION | 25 | 40 | ~55 | |
| MARNIE_FIB | TRENDING_BULL | 30 | 45 | ~65 | Fib pullback entries |
| MARNIE_FIB | TRENDING_BEAR | 30 | 45 | ~65 | |
| MARNIE_FIB | BREAKOUT | 30 | 45 | ~60 | Fib extension targets |
| MARNIE_FIB | HIGH_VOL | 25 | 40 | ~50 | Wider Fib zones |

### 6.6 Signal Classification

| Score Range | Classification | Signal Direction | Action |
|-------------|---------------|-----------------|--------|
| score ≥ TradeThreshold | EXECUTABLE | BUY / SELL | Passes to hard gates |
| CandidateThreshold ≤ score < TradeThreshold | ADVISORY | BUY_CANDIDATE / SELL_CANDIDATE | Published as advisory |
| score < CandidateThreshold | — | NO-TRADE | Not published |

---

## 7. Hard Gates — 14 Gates

**File:** `realtime/internal/types/types.go` (GateID definitions), `realtime/internal/gates/`

All gates are deterministic, freshness/version stamped, and **fail-closed** (UNKNOWN state = reject).

| # | Gate ID | Description | Fail Condition |
|---|---------|-------------|----------------|
| 1 | `data_quality` | Verifies data freshness and quality state | Quality < Authoritative or stale tick |
| 2 | `session` | Validates trading session (London/NY/Overlap) | Outside accepted sessions or weekend |
| 3 | `news` | Economic calendar event risk check | High-impact news within blackout window |
| 4 | `spread` | Current spread within strategy max | Spread > MaxSpreadPips |
| 5 | `slippage` | Expected slippage within strategy max | Estimated slippage > MaxSlippagePoints |
| 6 | `total_cost` | Spread + slippage + commission < threshold | Total cost exceeds cost-to-target ratio |
| 7 | `exposure` | Aggregate XAUUSD exposure within limits | Too many open positions or net exposure exceeded |
| 8 | `margin` | Sufficient margin headroom | Margin insufficient for position size |
| 9 | `rr_net_expectancy` | Net R:R after costs meets minimum | NetRR < MinRR after cost adjustment |
| 10 | `entitlement` | User plan includes this strategy | Strategy not in user's plan |
| 11 | `license` | Valid active license | License expired or invalid |
| 12 | `execution_permission` | Auto-execution permitted | Manual mode or execution disabled |
| 13 | `stop_hunt_filter` | Stop loss not in obvious liquidity pool | SL placed at equal highs/lows (stop hunt risk) |
| 14 | `min_atr` | ATR above minimum for meaningful movement | ATR too low (no volatility = no trade) |

### Gate Evaluation Flow

```
Strategy produces directional signal (BUY/SELL or BUY_CANDIDATE/SELL_CANDIDATE)
    │
    ├── For EXECUTABLE signals: ALL 14 gates must PASS
    │       Any gate FAIL → signal downgraded to ADVISORY or rejected
    │
    ├── For ADVISORY signals: gates are informational only
    │       Gate results attached but don't block publication
    │
    └── Gate results stored in Signal.GateResults[] for audit
```

### Gate Seeding

Gates are seeded conservatively (fail-closed) at startup:
```go
gateRegistry := gates.NewRegistry()
registerGates(gateRegistry, cfg)
gates.SeedConservativeGateStates(gateRegistry)
go refreshGateStates(gateRegistry, stateMgr, agentProvider)
```

Entitlement and license gates are hydrated from the control plane database.

---

## 8. Cross-Market Confluence Engine

**Files:** `realtime/internal/crossmarket/`

### 8.1 Purpose

The Cross-Market Confluence Engine is a **CONFIRMATION LAYER**, not a trading trigger. It:
- Consumes external market data (DXY, EURUSD, yields, VIX, COT, BTC, Oil)
- Normalizes each driver to a bounded impact score (-100..+100)
- Measures agreement/conflict between drivers
- Produces a bounded confluence score that **ADJUSTS** existing signal confidence
- **Never generates BUY/SELL by itself**
- **Never blocks signal generation** — hard gates remain authoritative

### 8.2 Mode

Currently in **SHADOW** mode (`CROSS_MARKET_MODE=shadow`):
- Computes confluence scores and persists shadow snapshots
- Does NOT modify production signal scores
- Used for validation and calibration data collection

### 8.3 Drivers

| Driver | Weight | Tier | Status | Normalization |
|--------|--------|------|--------|---------------|
| DXY (Dollar Index) | 25.0 | HIGH | CONNECTED | Inverse correlation: USD up → Gold down |
| EURUSD | 10.0 | MEDIUM | CONNECTED | Anti-double-counted with DXY (×0.4 weight) |
| Real Yields | 25.0 | HIGH | CONNECTED | FRED API: real yield up → Gold down |
| Fed Context | 15.0 | HIGH | NOT_CONFIGURED | Disabled — no feed configured |
| VIX | 10.0 | MEDIUM | CONNECTED | UVXY proxy: VIX up → Gold up (risk-off) |
| COT | 8.0 | MEDIUM | CONNECTED | CFTC commitments: net long → bullish |
| BTC | 7.0 | LOW | CONNECTED | Risk sentiment proxy |
| Oil | 5.0 | LOW | CONNECTED | Inflation proxy |

### 8.4 Confluence Scoring

```go
func (e *Engine) Evaluate(signalDirection, eventRisk) ConfluenceResult {
    // 1. Collect active driver snapshots with freshness decay
    // 2. Calculate effective weight = baseWeight × confidence × freshness
    // 3. Apply collinearity control (DXY/EURUSD anti-double-count)
    // 4. Compute weighted confluence score (-100..+100)
    // 5. Calculate agreement/conflict ratios
    // 6. Classify safe-haven regime (NORMAL, RISK_OFF, SAFE_HAVEN_GOLD, etc.)
    // 7. Detect correlation regime (NORMAL, WEAK, INVERSE, BREAKDOWN)
    // 8. Detect divergence severity (NONE, LOW, MODERATE, HIGH, EXTREME)
    // 9. Compute bounded score adjustment:
    //      adjustment = clamp(score × confidence × adjustment_factor, MaxPenalty, MaxBonus)
    //      MaxBonus = +10.0, MaxPenalty = -15.0
}
```

### 8.5 Safe-Haven Regimes

| Regime | Description |
|--------|-------------|
| NORMAL | Normal market conditions |
| RISK_ON | Risk appetite high (stocks up, gold down) |
| RISK_OFF | Risk aversion (stocks down, gold up) |
| SAFE_HAVEN_GOLD | Gold-specific safe haven demand |
| SAFE_HAVEN_USD | USD-specific safe haven demand |
| DUAL_SAFE_HAVEN | Both gold and USD bid (rare) |
| LIQUIDITY_STRESS | Liquidity crisis (everything selling off) |
| MIXED | Conflicting signals |
| UNKNOWN | Insufficient data |

---

## 9. Signal Persistence & Delivery

### 9.1 Signal Lifecycle Timestamps

Each signal tracks detailed lifecycle stages:

| Timestamp | Description |
|-----------|-------------|
| MarketTime | Source candle time (UTC) |
| MarketBarOpenTime | Candle open time |
| MarketBarCloseTime | Candle close time |
| DetectedAt | Strategy evaluation processing time |
| CandidateDetectedAt | Candidate threshold crossed |
| QualifiedAt | Trade threshold crossed + gates passed |
| PublishedAt | Signal published to delivery layer |
| DeliveryQueuedAt | Queued for MT4/MT5 delivery |
| DeliveredAt | Delivered to Windows Agent |
| AcknowledgedAt | MT4/MT5 EA acknowledged |
| ExecutionSubmittedAt | Order submitted to broker |
| BrokerFillAt | Broker fill timestamp |

### 9.2 Signal Structure (Key Fields)

```go
type Signal struct {
    ID                  string
    Symbol              string          // "XAUUSD"
    StrategyID          StrategyID      // e.g., "STANDARD_SCALPING"
    Direction           Direction       // BUY, SELL, BUY_CANDIDATE, etc.
    Grade               SignalGrade     // A, B, C, D
    RawScore            decimal         // 0-100
    LongScore           decimal
    ShortScore          decimal
    CalibratedProbability decimal       // 0-1 (calibrated, not raw score)
    EntryPrice          decimal
    StopLoss            decimal
    TP1/TP2/TP3         decimal
    GrossRR/NetRR       decimal         // R:R ratios
    ExpectedCost        decimal
    Regime              Regime
    Session             string
    NewsRisk            string
    Timeframe           Timeframe
    TTL                 time.Duration
    Status              SignalStatus
    ReasonCodes         []NoTradeReason
    Evidence            []EvidenceContribution
    GateResults         []GateEvaluation
    SignalClass         string          // ADVISORY, EXECUTABLE
    ShadowOnly          bool            // true if cross-market shadow
    Executable          bool            // true if passes all gates
    // ... (versioning, exit lifecycle, transition analysis)
}
```

### 9.3 Cooldown & Deduplication

- **CooldownManager:** Per-strategy cooldown prevents rapid re-fire after a signal
- **DuplicateChecker:** Idempotency — prevents duplicate signals for same candle/strategy/direction
- **File:** `realtime/internal/signal/`

### 9.4 Delivery Channels

```
Signal (post-gates)
    │
    ├── WebSocket Hub → Frontend Dashboard (Next.js)
    │       Users see signal in real-time on XAUUSD Command Center
    │
    ├── Agent Hub → Windows Agent → MT4/MT5 EA
    │       Signal sent via WebSocket to connected Windows Agents
    │       Agent forwards to MT4/MT5 EA for execution
    │
    └── NTFy Push Notification
            Self-hosted NTFy server for mobile alerts
```

**Delivery function:**
```go
func broadcastSignalToAll(wsHub *WebSocketHub, agentHub *AgentHub, signal *Signal) {
    wsHub.BroadcastSignal(signal)           // Frontend
    
    if signal.Direction != DirectionNoTrade {
        payload, _ := json.Marshal(signal)
        agentHub.Broadcast(payload)         // Windows Agents (MT4/MT5)
        // Log delivery
    }
}
```

---

## 10. Outcome Resolver & Validation

**File:** `realtime/internal/crossmarket/outcome_resolver.go`

### 10.1 Purpose

The Outcome Resolver monitors XAUUSD price movement and resolves shadow signal outcomes for cross-market validation. It tracks whether TP1/TP2/TP3/SL/Expiry was hit for each unresolved shadow snapshot.

### 10.2 Process

```
Background goroutine (periodic, not per-tick):
    1. Get current XAUUSD bid/ask
    2. Fetch unresolved shadow snapshots from DB
    3. For each snapshot:
       a. Check expiry → EXPIRED
       b. Check TP1 hit → TP1_HIT
       c. Check TP2 hit → TP2_HIT
       d. Check TP3 hit → TP3_HIT
       e. Check SL hit → SL_HIT
    4. Update outcome in database
    5. Record outcome event
```

### 10.3 Validation Framework

- Shadow snapshots stored in `trading.cross_market_shadow_snapshots`
- Validation results tracked in `trading.cross_market_validation_results`
- Only XAUUSD price determines outcomes (reference assets never determine trade outcomes)
- Currently: 0/30 usable validation days (insufficient data for statistical significance)

---

## 11. File Map

### Core Pipeline

| File | Purpose |
|------|---------|
| `realtime/cmd/realtime-engine/main.go` | Main pipeline wiring (~2400 lines) |
| `realtime/internal/features/registry.go` | Feature engine orchestration |
| `realtime/internal/features/state.go` | MarketState struct + StateManager |
| `realtime/internal/features/indicators.go` | 42+ core indicators |
| `realtime/internal/features/structure.go` | Market structure (BOS/CHoCH/swings) |
| `realtime/internal/features/liquidity.go` | Liquidity pools/sweeps |
| `realtime/internal/features/fvg.go` | Fair value gaps / order blocks |
| `realtime/internal/features/vwap.go` | Session/rolling VWAP |
| `realtime/internal/features/regime.go` | Regime classification |
| `realtime/internal/features/mtf.go` | Multi-timeframe alignment |
| `realtime/internal/features/session.go` | Session/news/exchange hours |
| `realtime/internal/features/sar.go` | Parabolic SAR |
| `realtime/internal/features/ichimoku.go` | Ichimoku Cloud |
| `realtime/internal/features/stochrsi.go` | Stochastic RSI |
| `realtime/internal/features/fibonacci.go` | Fibonacci retracement |
| `realtime/internal/features/pivots.go` | Daily/weekly pivots |
| `realtime/internal/features/candle.go` | Candle intelligence |
| `realtime/internal/features/marnie_fib.go` | Marnie Fib engine |
| `realtime/internal/features/history_bootstrap.go` | History warmup requirements |

### Strategy Engine

| File | Purpose |
|------|---------|
| `realtime/internal/strategy/strategies.go` | 4 core strategies + AllStrategies() + scoring |
| `realtime/internal/strategy/marnie_fib.go` | Marnie Fib strategy |
| `realtime/internal/strategy/regime_thresholds.go` | Per-strategy/regime thresholds |

### Cross-Market

| File | Purpose |
|------|---------|
| `realtime/internal/crossmarket/types.go` | Driver definitions, Config, ConfluenceResult |
| `realtime/internal/crossmarket/engine.go` | Confluence scoring engine |
| `realtime/internal/crossmarket/outcome_resolver.go` | Shadow outcome resolver |
| `realtime/internal/crossmarket/normalize.go` | Driver normalization functions |

### Gates & Signal

| File | Purpose |
|------|---------|
| `realtime/internal/gates/` | Hard gate implementations |
| `realtime/internal/signal/` | Signal engine, cooldown, dedup |
| `realtime/internal/types/types.go` | Signal struct, GateID, Direction, StrategyID |

### Gateway

| File | Purpose |
|------|---------|
| `realtime/internal/gateway/http.go` | HTTP API routes |
| `realtime/internal/gateway/agent_ws.go` | WebSocket agent handler |

### Configuration

| File | Purpose |
|------|---------|
| `infra/env/realtime.env` | All environment configuration |

---

## Summary

The Predict-A-Trade signal generation pipeline processes XAUUSD market data through:

1. **42+ technical indicators** computed locally from candle history (no fabrication of unavailable data)
2. **5 distinct strategies** with genuinely different logic, timeframes, and evidence pillars
3. **Regime-specific thresholds** mathematically justified per strategy+regime combination
4. **14 deterministic hard gates** that fail-closed for safety
5. **Cross-market confluence** with 7 connected drivers in SHADOW mode (no production impact)
6. **Full lifecycle tracking** from market time to broker fill with 12+ timestamp stages
7. **Cooldown + deduplication** for signal quality control
8. **Multi-channel delivery** (WebSocket frontend, Windows Agent MT4/MT5, NTFy push)

**Key safety principles:**
- `NO-TRADE` is a first-class valid result
- Raw score is NOT probability (calibration required)
- Unavailable data degrades quality or causes NO-TRADE (never fabricated)
- Cross-market never blocks signal generation
- Hard gates are authoritative and fail-closed
- All financial truth in control plane, isolated from trading hot path
