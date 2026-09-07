# Predict-A-Trade — Indicator & Signal Reference (indicator.md)

Authoritative reference for how the realtime engine reads candles, computes
every indicator, turns them into evidence, scores that evidence into a
direction, applies trade geometry and hard gates, and emits a signal.

Everything here is derived from the production code in this repo:

| Layer | Code |
|---|---|
| Core indicator maths (RSI/ATR/ADX/EMA) | `realtime/pkg/math/math.go`, `realtime/pkg/math/wilder.go` |
| Live indicator engine | `realtime/internal/features/indicators.go` |
| Structure / liquidity / FVG | `realtime/internal/features/structure.go`, `liquidity.go`, `fvg.go` |
| Candle intelligence | `realtime/internal/features/candle.go` |
| SAR / Ichimoku / StochRSI / Fibonacci / Pivots / ORB / Pullback | `realtime/internal/features/sar.go`, `ichimoku.go`, `stochrsi.go`, `fibonacci.go`, `pivots.go`, `session_orb.go`, `pullback.go` |
| Regime classification | `realtime/internal/features/regime.go` |
| Multi-timeframe | `realtime/internal/features/mtf.go` |
| VWAP | `realtime/internal/features/vwap.go` |
| Strategies (evidence scoring) | `realtime/internal/strategy/strategies.go` |
| Signal decision + gates | `realtime/internal/signal/engine.go`, `realtime/internal/gates/` |
| Candle pipeline | `realtime/cmd/realtime-engine/main.go` (processCandle) |

---

## 1. The pipeline (candle → signal)

```
MT4/MT5 Master EA (ticks + candles, market.ticks / market.candles, source column)
        │
        ▼
Aggregator (bucket bars: M1 M5 M15 M30 H1 H4 D1)
        │  bar_closed event
        ▼
processCandle()                                    [cmd/realtime-engine/main.go]
  ├─ per-timeframe feature Registry (M1 state never touches H4 state)
  ├─ IndicatorEngine.Process(candle)     → EMA/SMA/MACD/RSI/ATR/ADX/BB/Stoch/CCI/OBV
  ├─ StructureEngine / LiquidityEngine / FVGEngine / VWAP / SAR / Ichimoku / StochRSI
  ├─ Fibonacci / Pivot / SessionORB / Pullback engines
  ├─ RegimeEngine (classification with hysteresis)
  ├─ MTFEngine (cross-timeframe alignment score)
  ├─ ML inference (ONNX) + LLM market-context score (Ollama)
        │
        ▼
Strategy.Evaluate(state)    [4 strategies: ULTRA/STANDARD scalping, STANDARD/TREND swing]
  ├─ regime + session + news checks (early NO-TRADE exits)
  ├─ build evidence list (pillar, feature, direction BUY/SELL, weight, contribution)
  ├─ family caps (limit double-counting of correlated indicators)
  ├─ score direction: LongScore / ShortScore (+ dominance + conflict penalty)
  ├─ regime-specific candidate/trade thresholds
  ├─ HTF trend filter veto (H1 close)
  └─ BuildTradeGeometry → Entry / SL / TP1 / TP2 / TP3 (ATR-based, % profiles override)
        │
        ▼
signal Engine.Decide(input)
  ├─ market-closed short-circuit → single NO-TRADE (no rows, no signals)
  ├─ non-directional result → persisted with reason codes
  ├─ directional result → hard gates (short-circuit, fail-closed)
  │     DataQuality → Session → News → Spread → Slippage → TotalCost → MinATR
  │     → StopHuntFilter → Exposure → Margin → RR/Net-Expectancy → Profitability
  │     → Entitlement → License → ExecutionPermit → capital gates (wrong-side SL,
  │     risk oversize, position caps, daily loss, profit target, martingale ban, edge)
  └─ all pass → SIGNAL (BUY/SELL candidate, Executable)  |  veto → BLOCKED (never delivered)
        │
        ▼
WebSocket → dashboard + Windows Agent → MT4/MT5 Client EA (only when Executable=true)
```

Key invariants:

- Every indicator output is **UNAVAILABLE, never fabricated** when data is
  insufficient or invalid (zero high/low MT5 gap candles are skipped, not zero-filled).
- **NO-TRADE is a first-class result.** Nothing is forced; gates fail closed.
- Decision timeframes are per-strategy (`DecisionTFs`): M1/M5 for scalpers,
  M15/M30/H1 for standard swing, H1/H4 for trend swing — a swing engine never
  re-fires on every M1 bar, and per-strategy+bar idempotency prevents duplicates.
- Locally computed ATR (from candles) is authoritative; agent-snapshot ATR is
  only a fallback (agent feed has intermittently reported ATR ≈ price).

---

## 2. Reading a candle

A candle is `OHLC` + tick volume + timeframe + timestamp (bucket-open time;
bar close = open + timeframe duration).

Before any math, the engine validates geometry:
`High >= max(Open, Close, Low)` and `Low <= min(Open, Close, High)`.
An invalid candle returns zero features — it is treated as UNAVAILABLE, never
as neutral evidence.

Per-candle structural quantities used everywhere downstream:

```
body          = |Close − Open|
range         = High − Low
upperWick     = High − max(Open, Close)
lowerWick     = min(Open, Close) − Low
bodyRatio     = body / range
atrNormalized = range / ATR14
```

Volume is **tick volume** (broker data). True volume profile and cumulative
delta are explicitly UNAVAILABLE and never fabricated.

---

## 3. Indicator reference (formulas as implemented)

### 3.1 Trend

**EMA 9 / 21 / 50 / 100 / 200** (`fEMAWindow`)
```
alpha = 2 / (period + 1)
seed  = first value of window
ema_t = value_t * alpha + ema_{t-1} * (1 - alpha)
```
Computed over a rolling 200-bar window per timeframe. Warmup: `n >= period`.
Also tracked: `EMACross921` — a true crossover event (prev EMA9 ≤ EMA21 and
now EMA9 > EMA21), not just alignment.

**SMA 50 / 100 / 200** — running mean of the last N closes.

**MACD (12, 26, 9)**
```
MACD line = EMA12 − EMA26
Signal    = EMA9 over stored MACD-line history (≥9 samples)
Histogram / OsMA = MACD line − Signal
```
Bullish when MACD line > Signal; OsMA > 0 confirms momentum.

**ADX 14 with +DI / −DI** (Wilder, full method) — trend strength filter:
```
+DM_t = H_t − H_{t-1}  if > 0 and > −(L_t − L_{t-1}), else 0
−DM_t = L_{t-1} − L_t  if > 0 and >  (H_t − H_{t-1}), else 0
TR_t  = max(|H−L|, |H−C_prev|, |L−C_prev|)
+DI = 100·Wilder(+DM)/Wilder(TR), −DI = 100·Wilder(−DM)/Wilder(TR)
DX  = 100·|+DI − −DI| / (+DI + −DI);  ADX = Wilder(DX, 14)
```
Requires ≥28 bars. Zero-high/low candles are skipped. ADX is used as a
**filter, never an entry signal alone** (ADX>25 trending, ADX<20 ranging).

### 3.2 Momentum

**RSI 14** (Wilder)
```
change = C_t − C_{t-1};  gain = max(change,0);  loss = max(−change,0)
first avg = SMA(gain/loss, 14);  then avg = (prev*13 + value) / 14
RS  = avgGain / avgLoss
RSI = 100 − 100/(1+RS)
```
Flat market (both averages 0) → RSI = 50; no losses → 100.
Strategy use: scalping reads mid-range RSI (50–70 bull / 30–50 bear zone);
regime engine flags RSI>70 / RSI<30 as mean-reversion only when ADX<25.

**Stochastic 14/3/3**
```
%K = 100 * (Close − lowestLow(14)) / (highestHigh(14) − lowestLow(14))
Signal = 3-period SMA of %K
```

**StochRSI (separate engine: RSI14, stoch14, K3, D3)**
```
StochRSI_raw = (RSI − minRSI(14)) / (maxRSI(14) − minRSI(14))
K = SMA(raw, 3);  D = SMA(K, 3)
```

**CCI 20**
```
TP = (H + L + C) / 3
CCI = (TP_last − SMA(TP,20)) / (0.015 × meanDeviation(TP,20))
```

### 3.3 Volatility

**ATR 14** (Wilder) — the single most load-bearing indicator:
```
TR_t = max(H−L, |H−C_prev|, |L−C_prev|)      (skip zero H/L candles)
ATR_1 = mean(TR, 14);  ATR_t = (ATR_{t-1}·13 + TR_t) / 14
```
Used for: SL/TP geometry, spread-vs-ATR cost gates, stop-hunt distance checks,
candle displacement/compression/expansion classification, volatility regime.

**Bollinger Bands 20/2**
```
Middle = SMA20;  StdDev over 20 closes
Upper = Middle + 2σ;  Lower = Middle − 2σ
Width = 4σ / Middle        (z-scored over a 50-bar rolling window)
```

### 3.4 Volume / flow

**OBV** — over the rolling window: add volume on up closes, subtract on down
closes. Z-scored over 50 bars (min 20 samples).

**Session VWAP** — anchored at first candle of the UTC day:
```
typical = (H + L + C) / 3
VWAP = Σ(typical × volume) / Σ(volume)
bands ≈ VWAP ± (High − Low) of the current bar
```
MT5-snapshot VWAP is preferred; locally computed VWAP fills in when absent.

**Tick-volume z-score** — 50-bar rolling normalisation of tick volume.

### 3.5 Discrete indicator engines

**Parabolic SAR (0.02 / 0.20)** — Wilder acceleration factor; reverses when
price touches SAR; outputs value, side (IsLong), reversal flag, warmup flag.

**Ichimoku (9, 26, 52, 26)** — Tenkan/Kijun midpoints, Senkou A/B displaced
forward 26 bars, Chikou displaced back 26; classifies price above/below/in
cloud. No look-ahead: displaced future values are never treated as known.

**Fibonacci retracement** — anchored to **confirmed** structural swings only
(never arbitrary windows): levels 0.236 / 0.382 / 0.500 / 0.618 / 0.786 over
the swing-high↔swing-low range; anchors changing invalidates old levels.

**Pivot points (classic, daily + weekly)** — previous **completed** period
OHLC only, UTC boundaries:
```
P  = (H + L + C) / 3
R1 = 2P − L,  S1 = 2P − H
R2 = P + (H−L),  S2 = P − (H−L)
R3 = H + 2(P−L),  S3 = L − 2(H−P)
```

**Session ORB** — Asian / London / NY session high-low ranges tracked in
broker time; breakout direction vs the current session's opening range, plus
compression (tight range < 0.5 of reference = pre-breakout squeeze).

**Pullback detection** — trend anchor from confirmed swings; pullback depth %,
ATR-normalised retracement, continuation confirmation, quality score
(quality ≥ 0.3 required before it may contribute evidence).

---

## 4. Candle-pattern intelligence (CandleEngine)

All ATR-relative thresholds keep the patterns timeframe-normalised:

| Feature | Rule (as coded) |
|---|---|
| Doji | body < 10% of range |
| Pin bar | dominant wick > 60% of range |
| Rejection | wick (against close side) > 50% of range |
| Displacement | body > 2 × ATR (strong one-sided move) |
| Compression | range < 0.5 × ATR (squeeze) |
| Expansion | range > 2 × ATR (volatility burst) |
| Engulfing | current body engulfs previous body and engulfs it directionally |
| Inside bar | current range within previous range |
| Outside bar | current range engulfs previous range |
| Breakout | close beyond previous bar's high/low |
| Consecutive closes | running bull/bear streak counters |

Pin-bar quality (0–1) = 0.4×(1 − body ratio) + 0.4×(wick dominance) +
0.2×(ATR normalisation); rejection direction BUY (long lower wick) / SELL
(long upper wick). Only quality ≥ 0.5 earns evidence.

---

## 5. Market-structure & SMC features

**StructureEngine (fractal swings, BOS, CHoCH, MSS)**
- Swing high/low = fractal with **2 confirmed bars on each side** (no
  look-ahead — a swing is only confirmed after its right side exists).
- **BOS** (Break of Structure): close beyond the most recent confirmed swing
  → trend continuation event, direction bull/bear.
- **CHoCH / MSS** (Change of Character / Market Structure Shift): first break
  against the prevailing trend → early reversal flag.

**LiquidityEngine (pools & sweeps)**
- Liquidity pool: two swing points within ±0.10 of each other (equal
  highs/lows) → resting-stop cluster, strength 3.
- Sweep: wick pierces the pool but the close returns (high ≥ pool & close <
  pool = sell-side swept → bullish reversal fuel; low ≤ pool & close > pool =
  buy-side swept → bearish). Swept pools are marked and not re-used.
- CVD / DOM / order-flow: UNAVAILABLE (broker tick data) — never fabricated.

**FVGEngine (Fair Value Gaps & Order Blocks)**
- Bullish FVG: `candle[i-1].High < candle[i+1].Low` (3-candle imbalance);
  bearish FVG mirrored. Tracked until filled (price trades back through).
- Order block: last opposite-colour candle before the displacement move
  (bearish candle before a bullish break = bullish OB; vice versa).
- Zones feed SMC evidence only while unmitigated / unfilled.

---

## 6. Context engines

**RegimeEngine (v2, hysteresis)** — `classifyRaw` priority order:

1. ADX > 25 + EMA9>EMA21>EMA50 → TRENDING_BULLISH (conf 0.8)
2. ADX > 25 + EMA9<EMA21<EMA50 → TRENDING_BEARISH (0.8)
3. RSI > 70 / < 30 **and not trending** → MEAN_REVERSION (0.7)
4. ADX < 20 → RANGE (0.6)
5. ATR > 0.2% of price → HIGH_VOLATILITY (0.5)
6. EMA alignment only → TRENDING (0.55)
7. default → RANGE (0.4)

Anti-flicker: min hold 5 min, 3 consecutive confirming candles to transition,
confidence decay 0.92/candle when conditions lapse, forced re-evaluation below
0.25 confidence. Volatility state = HIGH when ATR/close > 0.2%.

**MTFEngine** — per-timeframe candle direction (close>open=+1, <open=−1),
weighted: M1 0.5, M5 1.0, M15 1.5, M30 1.0, H1 2.0, H4 1.5 →
`score = 100 × Σ(w·state)/Σw` ∈ [−100, +100].

**SessionEngine** — TOKYO / LONDON / NEW_YORK / OVERLAP / SYDNEY in broker
time, plus weekend/market-closed state and news-risk level (NONE / MEDIUM /
HIGH / BLOCKED).

---

## 7. Evidence scoring (how indicators become a direction)

Every indicator/feature reading becomes an **evidence contribution**:
`(pillar, feature, direction BUY|SELL, weight, contribution, quality)`.
Strategies add evidence only when a comparison is genuinely directional —
equal values carry **no** evidence (no fabricated tie-breaks).

Contribution weights by strategy (weight, normalised contribution) —
StandardScalping example:

| Pillar | Feature examples | Weight | Contrib |
|---|---|---|---|
| TREND | EMA9 vs EMA21 | 15 | 0.12 |
| STRUCTURE | BOS event | 18 | 0.14 |
| VWAP | price above/below session VWAP | 12 | 0.08 |
| CANDLE | bullish/bearish displacement | 15 | 0.10 |
| CANDLE | rejection wick | 12 | 0.08 |
| MOMENTUM | MACD vs Signal | 10 | 0.06 |
| MOMENTUM | OsMA sign | 8 | 0.05 |
| MOMENTUM | RSI mid-range (50–70 / 30–50) | 8 | 0.05 |
| TREND | ADX>MinADX with +DI>−DI | 10 | 0.07 |
| LIQUIDITY | sell-side sweep (bullish) / buy-side (bearish) | 12 | 0.08 |
| MTF | alignment score sign | 10 | 0.05×|score|/100 |
| STRUCTURE | above/below daily pivot | 8 | 0.04 |
| P2 add-ons | pullback / ORB breakout / pin bar | 8–10 | 0.03–0.10 |

**Family caps** (prevent correlated indicators from stacking):
TREND 0.35, MOMENTUM 0.30, VOLATILITY 0.15, VWAP 0.15, STRUCTURE 0.25,
LIQUIDITY 0.20, SMC 0.20, MTF 0.20, CANDLE 0.20, REGIME 0.15, ML 0.25,
SENTIMENT 0.25, SESSION_ORB 0.15. Any family above its cap is scaled down.

**AI layers (fail-open):**
- ML (ONNX): confidence > 30 → directional ±0.15 → ±2.25 added to RawScore.
- LLM market context (Ollama, once per bar, honestly labelled — not news
  sentiment): −1..+1 → ×5 added to RawScore. Never required for a signal.

**Direction decision** (`scoreDirectionWithThresholds`):
```
LongScore  = Σ BUY contributions  × 100
ShortScore = Σ SELL contributions × 100
dominance  = |LongScore − ShortScore|   (must exceed MinDominanceMargin,
            else NO-TRADE: conflicting direction)
score ≥ tradeThreshold     → BUY/SELL (gate-evaluated)
candidate ≤ score < trade  → BUY_CANDIDATE / SELL_CANDIDATE (advisory)
score < candidateThreshold → NO_TRADE (insufficient score)
```
Regime-specific thresholds override the strategy default (fallback:
candidate = 0.6 × trade). Conflict penalties before the threshold check:
M1-vs-H1 or H4 contradiction +3 each, RANGE-regime trend trade +10,
countertrend without CHoCH/MSS +20, spread/ATR > 0.5 +10; penalty > 40 → WAIT.

**Strategy score thresholds** (confluence profiles, seed baselines):
- STANDARD_SCALPING: min score 75, min separation 20 (v1.26 live 65/40)
- ULTRA_SCALPING: 85 / 25
- STANDARD_SWING: 70 / 15 (live MinConfluence 55)
- TREND_SWING: 75 / 15 (live MinConfluence 50)

**HTF trend filter (hard veto):** block BUY if price is below the H1 close,
block SELL if above (`NTHTFBearishVeto` / `NTHTFBullishVeto`).

---

## 8. Trade geometry (Entry / SL / TP)

```
entry = current price (bid/ask-aware, broker-digits rounded)
atr   = ATR14 × VolatilityScale (2.0 — feed-compensation, per-symbol override)
      raised so SL ≥ MinSLSpreadMult (3.0) × spread when necessary
```
Percentage exit profiles from the DB (StrategyExitSpec) take **priority**;
ATR multipliers are the fallback:

| Strategy | SL×ATR | TP1×ATR | TP2×ATR | TP3×ATR | MinRR | MaxSpread(pips) | Expiry |
|---|---|---|---|---|---|---|---|
| ULTRA_SCALPING | 1.0 | 1.5 | 2.5 | 4.0 | 2.0 | 1.5 | 3 min |
| STANDARD_SCALPING | 0.8 | 1.2 | 2.0 | 3.5 | 1.0 | 2.5 | 10 min |
| STANDARD_SWING | 2.0 | 3.0 | 5.0 | 8.0 | 2.0 | 4.0 | 60 min |
| TREND_SWING | 2.5 | 4.0 | 6.5 | 10.0 | 2.5 | 5.0 | 240 min |

Safety rails: every distance capped at 5% of entry; SL side enforced
(`enforceSLDirection`) — a stop on the wrong side of entry is corrected
defensively; TrendSwing additionally requires the macro filter
EMA100 > EMA200 (bull) / < (bear) before any trade.

`GrossRR(TP1) = |TP1 − Entry| / |Entry − SL|`, `NetRR` subtracts round-trip
cost from the target and adds it to the stop. Micro profit-taking: MicroTP +
PartialClosePct for micro scalps that must clear broker fee + slippage.

---

## 9. Hard gates (short-circuit, fail-closed)

Evaluated in this order; the **first VETO ends evaluation**:

1. **DataQuality** — candle/tick freshness, validity, required features present
2. **Session** — session allowed for the strategy
3. **News** — news-risk level (HIGH/BLOCKED vetoes)
4. **Spread** — spread vs strategy max (absolute + spread/ATR ratio)
5. **Slippage** — strategy max slippage points
6. **TotalCost** — round-trip cost vs cost-to-target ratio
7. **MinATR** — absolute ATR floor (default 2.0 for XAUUSD)
8. **StopHuntFilter** — distance to structural low/high ≥ 1.5 × ATR
9. **Exposure** — current vs max exposure
10. **Margin** — free-margin headroom
11. **RR / Net-Expectancy** — GrossRR ≥ strategy MinRR, net expectancy > 0
12. **Profitability** — per-strategy live-edge validation (proven-negative
    edge → NO_TRADE fail-closed; the "capital protection" log lines)
13. **Entitlement / License / ExecutionPermit** — subscription + licence state
14. **Capital gates (R1–R7):** WrongSideSL, RiskOversize (≤1.5%/trade, can
    size the lot down instead of vetoing), PositionCaps, DailyLoss (2%/4%/5%),
    ProfitTarget (5%/12%), MartingaleBan, EdgeValidation
    (PF ≥ 1.2, expectancy ≥ 0.2R, ≥ 50 samples)

Gate states are cached snapshots with freshness stamps (no synchronous I/O in
the decision path); DEGRADED results are advisory for candidates, fail-closed
for executable trades. A vetoed signal keeps its thesis direction for
diagnostics but is `Executable=false` — it is never delivered to the EA.

---

## 10. Output signal classes

| Class | Meaning |
|---|---|
| BUY / SELL (executable) | all evidence + all gates passed; delivered to agent |
| BUY_CANDIDATE / SELL_CANDIDATE | directional but below trade threshold or degraded-only gates; advisory, shown on dashboard |
| NO_TRADE | insufficient score, regime/session mismatch, conflicting direction, insufficient dominance, ATR not ready, market closed… |
| WAIT / BLOCKED / ERROR | conflict penalty > 40; gate veto; internal error |
| TREND_TRANSITION candidate | TrendSwing in RANGE: transition evidence (ADX expansion, EMA slope, BB/ATR expansion, BOS, MTF) above candidate threshold (20) but below trade threshold (35) — non-executable advisory |

Every signal carries: strategy, direction, grade, RawScore/LongScore/ShortScore,
entry/SL/TP1-3, gross RR, suggested (safe) lot, regime, session, news risk,
full evidence list with per-pillar contributions, gate evaluations, expiry
(15 min unless strategy expiry), and reason codes for audit.

---

## 11. Per-strategy indicator emphasis (what each strategy reads)

**ULTRA_SCALPING (M1 decision, M5/M15 context)** — flow microstructure,
liquidity events, execution cost quality; EMA9/21, VWAP, spread/ATR, tick
momentum. Strictest spread/cost gates (0.25 abs, 0.40×ATR), 5 trades/day cap.

**STANDARD_SCALPING (M1/M5, M15/M30 context)** — EMA9/21 alignment, VWAP side,
short-term BOS, displacement + rejection candles, MACD/OsMA, RSI mid-range,
ADX>20 side, liquidity sweeps, MTF, daily pivot; mandatory: liquidity +
structure pillars; range-adaptive evidence in RANGE; pullback/ORB/pin-bar P2
evidence; min 3 confluences.

**STANDARD_SWING (M15/M30/H1, H4/D1 context)** — EMA21/50 alignment,
SMA200 side, H1/H4 BOS + CHoCH, unmitigated order blocks and unfilled FVGs,
ADX, MACD, RSI vs 50, breakout closes, MTF, HTF sweeps; mandatory: D1/H4
structure; RR ≥ 1.8, 2 trades/day, 120-min cooldown.

**TREND_SWING (H1/H4, D1/W1 context)** — SMA200 major trend (highest weight),
EMA50, EMA21/50, ADX strong-trend gate, macro EMA100/200 mandatory, MTF,
macro (DXY/yield) and COT/ETF-flow pillars; in RANGE it only emits
transition candidates; MinRR 2.5, 1 trade/day, 240-min cooldown.

---

## 12. Indicator availability policy

Computed locally from candles: EMA(9/21/50/100/200), SMA(50/100/200), MACD,
OsMA, RSI(14), Stochastic(14/3/3), StochRSI(14/14/3/3), CCI(20), ADX/DI(14),
ATR(14), Bollinger(20/2), OBV, session VWAP, Parabolic SAR, Ichimoku,
Fibonacci, pivots, ORB ranges, pullback metrics, structure (BOS/CHoCH/MSS),
liquidity pools/sweeps, FVG/OB.

**Never fabricated when unavailable:** Volume Profile, Cumulative Delta,
DOM/order-flow (broker tick volume only), StochRSI before warmup, Senkou spans
before displacement history. Missing optional pillars are tracked as
`missingOptionalWeight` and can veto via `NTSystemDegraded`; missing mandatory
pillars always veto (`NTUnclearStructure`).

---

## 13. Persistence & observability

Per closed bar (async, 3 s timeout): candle → `market.candles`, regime →
`market.regime_history`, key indicators (ATR14, RSI14, EMA9) →
`market.indicator_history` with `source=local_compute,
quality=AUTHORITATIVE`. Every strategy evaluation is audited as a pipeline
execution with per-step timings; every decision carries evidence + gate
evaluations for replay.

---

*Maintainer note: keep this file in sync with the code. Source of truth is
always `realtime/internal/features/*` and `realtime/internal/strategy/*` —
when indicator formulas, weights, thresholds, or gates change, update this
document in the same commit.*