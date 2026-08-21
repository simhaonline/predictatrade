# Trading Mathematics Specification

## Version: v1.8.0 — Trade Management Audit + Broker Stop Validation + Cost-Aware Break-Even

## Four Strategy Scoring Formula

All four strategies use the same `scoreDirection` formula:

```
longScore  = Σ(e.Contribution for e in evidence where Direction = BUY)  × 100
shortScore = Σ(e.Contribution for e in evidence where Direction = SELL) × 100

if longScore > shortScore:
    longScore -= conflictPenalty
    if longScore > MinConfluence: direction = BUY
    else: direction = NO_TRADE (INSUFFICIENT_SCORE)
else:
    shortScore -= conflictPenalty
    if shortScore > MinConfluence: direction = SELL
    else: direction = NO_TRADE (INSUFFICIENT_SCORE)

rawScore = max(longScore, shortScore)
```

## Family Caps

```
TREND ≤ 0.25, MOMENTUM ≤ 0.20, VWAP ≤ 0.10, STRUCTURE ≤ 0.20,
LIQUIDITY ≤ 0.15, SSMC ≤ 0.15, MTF ≤ 0.15, CANDLE ≤ 0.15, REGIME ≤ 0.10
```

## MTF Evidence Normalization

```
MTF score range: [-100, +100]
Contribution = (mtfScore / 100.0) × factor
```



## Candidate Microprofit Geometry (v1.6.0)

Candidate signals (BUY_CANDIDATE/SELL_CANDIDATE) use tighter ATR multipliers
than qualified signals to capture microprofit from weaker directional edges.

```
Candidate SL  = ATR × candidate_sl_multiplier
Candidate TP1 = Entry ± (ATR × candidate_tp1_multiplier)
Candidate TP2 = Entry ± (ATR × candidate_tp2_multiplier)
Candidate TP3 = Entry ± (ATR × candidate_tp3_multiplier)
```

| Strategy | SL Mult | TP1 Mult | TP2 Mult | TP3 Mult |
|----------|---------|----------|----------|----------|
| Ultra Scalping | 1.0 | 1.0 | 2.0 | 3.0 |
| Standard Scalping | 1.0 | 1.5 | 2.5 | 4.0 |
| Standard Swing | 1.5 | 1.5 | 3.0 | 5.0 |
| Trend Swing | 2.0 | 2.0 | 3.5 | 5.0 |

## Vectorized Python Implementations (v1.5.0)

All indicator formulas above have vectorized pandas/numpy implementations in
`research/src/patresearch/quantitative_strategy_engine.py` (`QuantitativeStrategyEngine`).
These are used for batch historical computation and are parity-verified against
the scalar `reference_math.py` functions (SOW Section 137). The vectorized engine
does **not** replace the Go production indicator implementations — it provides a
fast research-plane counterpart for large-scale backtesting and feature studies.

| Indicator | Vectorized Method | Scalar Reference |
|-----------|------------------|-----------------|
| SMA | `compute_sma(df, period)` | — |
| EMA | `compute_ema(df, period)` | `reference_math.ema()` |
| RSI (Wilder) | `compute_rsi(df, period)` | `reference_math.rsi()` |
| ATR (Wilder) | `compute_atr(df, period)` | `reference_math.atr()` |
| ADX + DI | `compute_adx(df, period)` | — |
| MACD | `compute_macd(df, 12, 26, 9)` | — |
| Bollinger Bands | `compute_bollinger_bands(df, 20, 2.0)` | — |

## Calibration

```
clampedScore = clamp(rawScore, 0, 100)
probability = sigmoid(a × clampedScore/100 + b)
```

Default model parameters:
| Strategy | a | b |
|----------|---|---|
| STANDARD_SCALPING | 2.5 | -0.5 |
| ULTRA_SCALPING | 3.0 | -0.8 |
| STANDARD_SWING | 2.0 | -0.3 |
| TREND_SWING | 1.8 | -0.2 |

## Entry / SL / TP

```
Entry = CurrentPrice (mid)
BUY:  SL = structuralLow - λ_SL×ATR - 0.5×Spread
SELL: SL = structuralHigh + λ_SL×ATR + 0.5×Spread
TP1 = Entry ± max(MinRR × |Entry-SL|, 1.5×ATR)
TP2 = Entry ± TP1_dist × 1.5
TP3 = Entry ± TP1_dist × 2.0
```

## R:R

```
RR1 = |TP1-Entry| / |Entry-SL|
RR2 = |TP2-Entry| / |Entry-SL|
RR3 = |TP3-Entry| / |Entry-SL|
NetRR = (target_dist - cost) / (stop_dist + cost)
```

---

## PTB Synthesis Engine Mathematics

### Confluence Score

```
weights = {
    mtf: 0.15, regime: 0.15, volatility: 0.10,
    structure: 0.20, liquidity: 0.15, session: 0.10,
    manipulation: 0.10, macro: 0.05
}

confluence = Σ(weight[comp] × componentScore[comp]) / Σ(weight[comp])
```

Only available components contribute. Unavailable components are skipped (not zeroed).

### Bias Determination

```
longScore = Σ(weight[comp] × score[comp]) for components where direction = "LONG"
shortScore = Σ(weight[comp] × score[comp]) for components where direction = "SHORT"
netScore = (longScore - shortScore) / totalWeight

if netScore > 50: bias = STRONG_LONG
elif netScore > 15: bias = LONG
elif netScore < -50: bias = STRONG_SHORT
elif netScore < -15: bias = SHORT
else: bias = NEUTRAL

biasStrength = |netScore| / 100
```

### Setup Quality Grading

```
score = (confluence + confidence) / 2

if score ≥ 90: grade = A+
elif score ≥ 80: grade = A
elif score ≥ 70: grade = B
elif score ≥ 60: grade = C
elif score ≥ 50: grade = D
else: grade = F
```

### Position Size Multiplier

```
A+ → 1.00
A  → 0.80
B  → 0.60
C  → 0.40
D  → 0.20
F  → 0.00
```

Advisory only — existing risk manager remains authoritative.

### Stop Distance Multiplier

```
if manipulationIndex > 70: multiplier = 1.5
elif volatilityState == "LOW": multiplier = 0.8
else: multiplier = 1.0
```

### Action Determination

```
if manipulationIndex > 70: action = AVOID
elif confidence < 65: action = WAIT
elif bias in (STRONG_LONG, LONG, STRONG_SHORT, SHORT): action = ENTER
else: action = WAIT
```

### Enhanced Regime Classification

```
if manipulationIndex > 70: regime = MANIPULATION
elif ADX > 30 AND bullish: regime = STRONG_TREND_UP
elif ADX > 30 AND bearish: regime = STRONG_TREND_DOWN
elif ADX > 20 AND bullish: regime = WEAK_TREND_UP
elif ADX > 20 AND bearish: regime = WEAK_TREND_DOWN
elif ADX < 18: regime = RANGE_BOUND
elif volatility == EXTREME or HIGH: regime = HIGH_VOLATILITY
elif volatility == LOW: regime = LOW_VOLATILITY
else: regime = TRANSITIONING
```

### Manipulation / Dislocation Index

```
dislocationIndex = 0
+ 0.2 if wick_rejection
+ 0.3 if false_breakout (structure_break AND rejection)
+ 0.2 if sweep_detected
+ 0.2 if displacement

clamped to [0, 1.0]
manipulationIndex = dislocationIndex × 100
```

---

## Gold Correlation Engine Mathematics

### Pearson Correlation

```
n = min(len(x), len(y))
meanX = Σx[i] / n
meanY = Σy[i] / n

numerator = Σ((x[i] - meanX) × (y[i] - meanY))
denomX = Σ((x[i] - meanX)²)
denomY = Σ((y[i] - meanY)²)

if denomX == 0 or denomY == 0:
    return 0  (constant series — undefined)
if n < 2:
    return 0  (insufficient samples)

correlation = numerator / sqrt(denomX × denomY)
```

### Correlation Direction

```
if correlation < -0.3: direction = INVERSE
elif correlation > 0.3: direction = DIRECT
else: direction = NEUTRAL
```

### Gold/Silver Ratio

```
goldSilverRatio = goldPrice / silverPrice
```

### Freshness Check

```
if dataAge > 300 seconds: quality = STALE
if dataAge > 600 seconds: quality = UNAVAILABLE (do not use)
```

---

## Indicator Formulas

### EMA
```
multiplier = 2 / (period + 1)
EMA[t] = price[t] × multiplier + EMA[t-1] × (1 - multiplier)
```
Periods: 9, 21, 50, 100, 200

### RSI (Wilder)
```
avgGain = Σ(positive changes) / period
avgLoss = Σ(|negative changes|) / period
RS = avgGain / avgLoss
RSI = 100 - 100/(1+RS)
```
If avgLoss = 0: RSI = 100

### ATR (Wilder)
```
TR = max(H-L, |H-prevC|, |L-prevC|)
ATR = Σ(TR) / period
```

### MACD (12/26/9)
```
MACD_line = EMA(12) - EMA(26)
Signal = EMA(MACD_line_history, 9)
Histogram = MACD_line - Signal
```

### ADX (Wilder)
```
+DM = max(high[i]-high[i-1], 0) if > -DM
-DM = max(low[i-1]-low[i], 0) if > +DM
+DI = (+DM/ATR) × 100
-DI = (-DM/ATR) × 100
DX = |+DI-(-DI)| / (+DI+(-DI)) × 100
ADX = average(DX)
```

### Stochastic (14/3/3)
```
%K = 100 × (Close - LowestLow) / (HighestHigh - LowestLow)
%K_signal = SMA(%K_history, 3)
```

### Bollinger Bands (20/2)
```
Middle = SMA(20)
stdDev = sqrt(Σ(close-Middle)² / 20)
Upper = Middle + 2×stdDev
Lower = Middle - 2×stdDev
Width = (Upper-Lower) / Middle
```

### CCI (20)
```
TP = (H+L+C)/3
CCI = (TP - SMA(TP,20)) / (0.015 × MeanDeviation)
```

### MTF Alignment
```
state[TF] = +1 if close>open, -1 if close<open, 0 if equal
MTFScore = 100 × Σ(weight[TF] × state[TF]) / Σ(weight[TF])
```
Weights: M1=0.5, M5=1.0, M15=1.5, M30=1.0, H1=2.0, H4=2.5, D1=3.0

### Volatility Regime
```
atrPct = ATR / price
if atrPct > 0.005: EXTREME
elif atrPct > 0.003: HIGH
elif atrPct > 0.001: NORMAL
elif atrPct > 0.0005: LOW
else: EXTREME_LOW
```

### Liquidity Void
```
size_atr = bodySize / ATR
strength = min(size_atr / 3.0, 1.0)
```

### Session Imbalance
```
normalized = |CurrentPrice - VWAP| / ATR
if diff > 0 AND normalized > 0.3: BULLISH
if diff < 0 AND normalized > 0.3: BEARISH
else: BALANCED
```


---

## TP/SL Geometry (v1.4.0)

### ATR-Based Level Computation

Entry, Stop Loss, and Take Profit levels are computed using ATR (Average True Range) multipliers:

**BUY:**
```
Entry = Ask price
SL    = Entry - (ATRMultiplierSL × ATR)
TP1   = Entry + (ATRMultiplierTP1 × ATR)
TP2   = Entry + (ATRMultiplierTP2 × ATR)
TP3   = Entry + (ATRMultiplierTP3 × ATR)
```

**SELL:**
```
Entry = Bid price
SL    = Entry + (ATRMultiplierSL × ATR)
TP1   = Entry - (ATRMultiplierTP1 × ATR)
TP2   = Entry - (ATRMultiplierTP2 × ATR)
TP3   = Entry - (ATRMultiplierTP3 × ATR)
```

### Strategy ATR Multipliers

| Strategy | SL | TP1 | TP2 | TP3 | MinRR |
|----------|-----|-----|-----|-----|-------|
| STANDARD_SCALPING | 1.0×ATR | 1.0×ATR | 1.5×ATR | 2.0×ATR | 1.2 |
| ULTRA_SCALPING | 0.5×ATR | 0.5×ATR | 0.75×ATR | 1.0×ATR | 1.0 |
| STANDARD_SWING | 1.5×ATR | 1.5×ATR | 2.5×ATR | 3.5×ATR | 1.8 |
| TREND_SWING | 2.0×ATR | 2.0×ATR | 4.0×ATR | 6.0×ATR | 2.5 |

### Minimum SL Distance

SL distance is enforced to be at least `ATRMultiplierSL × ATR` from entry. When structural levels (swing lows/highs) are close to entry, the ATR-based minimum prevents SL from being too tight.

### v1.4.0 Fix

**Before**: TP1 = max(MinRR × SL_distance, 1.5×ATR) — made TP1 2.5x further than SL.
**After**: TP1 = ATRMultiplierTP1 × ATR — balanced with SL (R:R ≈ 1:1 for TP1).
The MinRR gate validates R:R and rejects insufficient signals — TP is not inflated.
