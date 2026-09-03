# Predict-A-Trade — Full Pipeline, Maths & Logic in Plain Words

> docs.md — rebuilt 2026-09-03. Everything below is read directly from the live
> engine source code (`realtime/`), not from memory. Formulas are the real ones
> the engine computes; the "Easy words" lines explain them simply.

---

## THE BIG PICTURE

```
Master EA (MT5, Windows) ──ticks via HTTPS──► Go Engine (pat-realtime)
                                                 │
        ┌────────────────────────────────────────┤  (one process, one clock)
        ▼                                        ▼
   CANDLES + FEATURES                    STRATEGIES (7)
   (indicators compute)                  (evidence → score → grade)
                                                 │
                                                 ▼
                                     CALIBRATION (score → honest %)
                                                 ▼
                                     23 RISK GATES (veto wall)
                                                 ▼
                                     CAPITAL TIERS + LOT SIZE
                                                 ▼
                                     SIGNAL ROW → DELIVERY QUEUE
                                                 │
Client EAs (MT4/MT5, Windows) ◄──HTTPS poll──┘
   (own account, own money, EA-side safety gates, ACK back)
```

- **One brain**: the Go engine. The Signal Engine is not a separate program —
  it runs *inside* the Core Engine on the same tick, same clock. No sync to break.
- **Money never transits us**: we send numbers (entry/SL/TP); the client EA
  places the order on their own broker account.
- **NO-TRADE is a normal answer.** Most evaluations end with NO-TRADE on purpose.

---

## PART 1 — DATA LAYER

### 1.1 Tick ingest
- Master EA POSTs every tick to `/ingest/agent` (nginx → engine).
- Engine validates: symbol, sane price, **server receive time is the truth**
  (the tick's own timestamp is kept for reference only — a client PC clock can drift).
- Every tick saved to `market.ticks` (27M+ rows) with `gateway_receipt_time`.

### 1.2 Candle building
- Ticks are grouped into candles per timeframe: M1, M5, M15, M30, H1, H4, D1, W1.
- OHLCV = first price, highest, lowest, last price, tick count.
- A candle only becomes "closed" when the next period starts — the engine never
  trades on half-formed candles.

**Easy words**: the Master terminal is the microphone; the engine turns the raw
sound into clean bars everyone can measure.

---

## PART 2 — THE INDICATORS (exact maths, exact periods)

All live in `realtime/internal/features/`. Computed in float64 for speed
(~0.3 ms per bar across the whole set) with decimal precision where money math
happens. Each engine keeps a rolling window (max ~500 bars) — nothing scans
infinite history, so latency is constant.

### 2.1 EMA — Exponential Moving Average (9, 21, 50, 100, 200)
```
first value = SMA(period)
EMA = close × k + EMA_prev × (1 − k),   k = 2 / (period + 1)
```
**Easy words**: an average of recent closes that leans harder on the newest
price. The k for EMA9 is 0.2 (20% of the value is the latest bar), for EMA200
it is 0.0099 (1% — very smooth). EMAs rising in order 9>21>50>100>200 = strong
bullish staircase.

### 2.2 SMA — Simple Moving Average (50, 100, 200)
```
SMA = sum(last N closes) / N
```
**Easy words**: the plain average. Used as slow reference lines, no tricks.

### 2.3 MACD (12/26/9) + OsMA
```
MACD main  = EMA(12) − EMA(26)
MACD signal = EMA(9) of MACD main
Histogram/OsMA = MACD main − signal
Bullish cross = main crossed above signal this bar
```
**Easy words**: the gap between fast and slow momentum. When the gap starts
growing after being negative, momentum is turning up.

### 2.4 RSI — Wilder's RSI (period 14)
```
avg gain / avg loss smoothed with Wilder's method (α = 1/14)
RS = avgGain / avgLoss
RSI = 100 − 100 / (1 + RS)      → ranges 0..100
```
**Easy words**: how stretched the price is. >70 stretched up, <30 stretched
down, 45–55 neutral. The engine only trusts RSI mid-zone moves (e.g. RSI_BULLISH_MID),
because >70 in a strong trend is strength, not a reversal — see Regime below.

### 2.5 ATR — Average True Range (Wilder, 14)
```
TrueRange = max(high−low, |high−prevClose|, |low−prevClose|)
ATR = Wilder-smoothed TR (α = 1/14)
```
**Easy words**: how much gold normally moves per bar. EVERY stop-loss and
take-profit in the whole system is sized in multiples of ATR — ATR is the
ruler. If ATR is small (quiet market), stops are small; if big, stops widen.

### 2.6 ADX / +DI / −DI (Wilder)
```
+DM = high − prevHigh (if > prevLow − low, else 0); −DM likewise
+DI = 100 × smoothed(+DM) / ATR ; −DI = 100 × smoothed(−DM) / ATR
DX = 100 × |+DI − −DI| / (+DI + −DI) ; ADX = Wilder-smoothed DX
```
**Easy words**: a trend *strength* meter (no direction). ADX>25 = real trend,
ADX<20 = range/chop. The +DI/−DI pair gives the direction (whoever is bigger).

### 2.7 Bollinger Bands (20, 2σ)
```
middle = SMA(20); upper = middle + 2×σ; lower = middle − 2×σ
width = 4σ / middle ; %B = (price − lower)/(upper − lower)  ∈ [0,1]
```
**Easy words**: a breathing band around price. %B tells where price sits
inside the band (1 = ceiling, 0 = floor). Squeeze = market loading a spring.

### 2.8 Stochastic (14/3/3)
```
%K = 100 × (close − lowestLow14) / (highestHigh14 − lowestLow14)
%D = SMA(3) of %K
```
**Easy words**: where price sits inside the last 14 bars' range. 80/20 are the
extremes.

### 2.9 StochRSI (RSI 14 → stoch 14 → K/D smoothing)
```
StochRSI = (RSI − min(RSI,14)) / (max(RSI,14) − min(RSI,14))
K = smoothed; D = SMA of K
```
**Easy words**: momentum of the momentum — turns *before* plain RSI. Used in
the playbook confirmation checklist.

### 2.10 CCI (20)
```
TP = (H+L+C)/3 ; CCI = (TP − SMA20(TP)) / (0.015 × meanDeviation(TP))
```
**Easy words**: how far price is from its normal level, normalized so ±100 is
"unusual". Positive+rising = bull fuel (one of the 4 confirmation points).

### 2.11 Parabolic SAR
Standard step/acceleration algorithm; flips below price in uptrends, above in
downtrends. **Easy words**: a trailing dot that sits under price while it's
climbing. The dot's side is confirmation point #2 in the playbook.

### 2.12 Ichimoku (9/26/52)
```
Tenkan = (max9 + min9)/2 ; Kijun = (max26 + min26)/2
Senkou A = (Tenkan+Kijun)/2 shifted 26 ; Senkou B = (max52+min52)/2 shifted 26
Cloud = area between A and B ; above cloud = bullish bias
```
**Easy words**: a map of support/resistance projected into the future. Price
above the whole cloud = long-only zone for swing confirmation.

### 2.13 OBV — On Balance Volume
```
volume added on up-closes, subtracted on down-closes, cumulative
```
**Easy words**: is money flowing in or out. Rising OBV with rising price =
healthy trend.

### 2.14 VWAP (session, daily reset)
```
typical price = (H+L+C)/3
VWAP = Σ(TP × volume) / Σ(volume)   — resets at the first candle of each day
bands ≈ VWAP ± rolling average range
```
**Easy words**: the day's "fair price" weighted by activity. Above VWAP = the
day's buyers are winning. One of the heavier evidence pillars (weight 12).

### 2.15 Market Structure (swing highs/lows, BOS, CHoCH)
- Swing high = a bar whose high is the highest among **2 bars on each side**
  (fractal, confirmed only 2 bars later — no looking into the future).
- Lookback window 50 bars.
- **BOS** (Break of Structure): price closes beyond the last confirmed swing —
  trend continuing. Heaviest single evidence (weight 18 × 0.14 ≈ 2.5 points).
- **CHoCH** (Change of Character): first BOS in the *opposite* direction —
  possible reversal.

**Easy words**: the engine draws the mountain ridges and valleys, then watches
which ridge gets climbed. Climbing higher ridges = buyers own the map.

### 2.16 Liquidity Pools & Sweeps
- Pool = a cluster of equal highs/lows (stop-loss parking lots).
- Sweep = wick pierces the pool but the candle **closes back inside**.
  Sell-side sweep (lows raided, close above) → bullish evidence (weight 12).

**Easy words**: big players push price into the crowd's stops to fill their own
orders, then reverse. A sweep-and-reclaim is a footprint of that move.

### 2.17 FVG — Fair Value Gaps / Order Blocks / Breakers
```
Bullish FVG: candle[i−1].high < candle[i+1].low   (a 3-candle price void)
Bearish FVG: candle[i−1].low  > candle[i+1].high
```
**Easy words**: a price jump so fast it left a hole. Price often returns to
"fill" the hole — those holes act as magnets and support. Lookback 100 bars.

### 2.18 Fibonacci (from confirmed swings)
```
retracement levels = 0.236, 0.382, 0.500, 0.618, 0.786 of the last swing
golden zone = 0.618–0.786
extensions = 1.272, 1.618, 2.618 (profit targets)
```
**Easy words**: split any swing by the golden ratio. Pullbacks that stop in the
0.618–0.786 pocket are statistically the best continuation entries.

### 2.19 Pivots (daily & weekly, classic)
```
P  = (H + L + C) / 3        (previous completed day / week)
R1 = 2P − L ; S1 = 2P − H   (+ R2/R3, S2/S3 in classic ladder)
```
**Easy words**: yesterday's balance point. Traders worldwide watch the same
levels, so they become self-fulfilling walls.

### 2.20 Regime Engine — what kind of market is this?
Priority rules (evaluated top to bottom), with hysteresis (a new regime must
hold N candles and stay a minimum time before switching):

| Priority | Condition | Regime |
|---|---|---|
| 1 | ADX>25 AND EMA9>21>50 | TRENDING_BULLISH (conf 0.8) |
| 1 | ADX>25 AND EMA9<21<50 | TRENDING_BEARISH (conf 0.8) |
| 2 | RSI>70 AND NOT trending | MEAN_REVERSION (0.7) |
| 2 | RSI<30 AND NOT trending | MEAN_REVERSION (0.7) |
| 3 | ADX<20 | RANGE (0.6) |
| 4 | ATR/price > 0.2% | HIGH_VOLATILITY (0.5) |
| 5 | EMA aligned but ADX weak | TRENDING (0.55) |
| 6 | nothing | RANGE (0.4) |

**Easy words**: a weather station. Trend strategies only trade in "trend"
weather; Fib strategies love "range" weather. RSI>70 does NOT mean "sell" —
in a real trend it means "strong". This priority order is deliberate.

### 2.21 MTF Alignment — multi-timeframe agreement
```
each TF votes +1 (bull candle), −1 (bear candle), 0 (doji)
weights: M1=0.5, M5=1.0, M15=1.5, M30=1.0, H1=2.0, H4=1.5
score = 100 × Σ(weight × vote) / Σ(weight)     ∈ [−100, +100]
```
**Easy words**: all the clocks must roughly agree. H1 (weight 2.0) counts
double M15. Positive score = higher frames lean bullish.

### 2.22 Sessions & News
- TOKYO 00:00–09:00 UTC, LONDON 08:00–17:00, NEW_YORK 13:00–22:00,
  OVERLAP = London ∩ New York (the deep-liquidity window).
- **News risk**: an economic-calendar provider flags red-news windows →
  `BLOCKED` state. Strategies veto (`NT_NEWS_RISK_BLOCKED`) — no trading into
  a news bomb.
- Session ORB (opening-range breakout) and Pullback detection run in SHADOW.

### 2.23 Candle Intelligence
Pinbar detection (rejection wick ≥ threshold of range), displacement candles
(big-body expansion = institutions moving), rejection wicks, body/range
quality ratios. These feed the CANDLE pillar (weights 10–15).

### 2.24 PTB — Professional Trader Brain (22 shadow modules)
LiquidityVoid, WickFill, SessionImbalance, CandleRangeProjector, TimeAtMode,
EngineeredLiquidity, MarketPhase, RelativeVolumeFlow, PriceDelivery,
StopHuntProxy, InstitutionalFootprint, TimeCycle, AlgoActivity,
CompleteLiquidityMap, ManipulationProxy, MTFBias, VolatilityRegime, SRQuality,
Microstructure, StatisticalPerf, DataQuality, Correlation.
**ALL default SHADOW — zero effect on scores** until validated and switched on.
They are measurement instruments being tested against live outcomes.

### 2.25 Devil Liquidity engine
Lifecycle: Detection → Qualification → Tracking → Approach → Touch → Sweep →
Reclaim/Reject → Reversal Confirmation → Outcome. Models how stops get hunted
around a marked level and whether the reclaim confirms a real reversal.

### 2.26 IGS — Institutional Gold Signal (composite, shadow)
```
IGS = CentralBankFlow + ETFFlow + COTPositioning + RealYieldRegime
    + USDRegime + COMEXPositioning + OptionsGamma + PhysicalDemand
    + InstitutionalResearchSentiment
```
Tier S (heaviest): real yields, USD regime, central banks. Tier A: COT/COMEX,
ETF flows, options gamma. Confirmation-only layer — never acts alone.

### 2.27 Astro engine (feeds ATEN)
Self-contained Vedic (sidereal) + Western (tropical) planetary engine — no
external ephemeris library; planet longitudes via validated analytical series
(±0.5° accuracy over 2015–2030, enough for daily trading granularity).
Outputs: Nakshatra bias, Hora bias, Vimshottari dasha (L1+L2), western aspects
to the Gold natal chart, eclipse/contamination flags.

---

## PART 3 — THE STRATEGIES (7) — exact rules

### 3.0 The scoring machine (how ANY strategy turns evidence into a decision)

Every strategy produces a list of **evidence pieces**. Each piece has:
- **pillar** (family: TREND, STRUCTURE, MOMENTUM, CANDLE, LIQUIDITY, VWAP, MTF, SESSION_ORB…)
- **direction** (BUY / SELL)
- **weight** (how much this pillar counts for this strategy)
- **normalized value** (0–1 strength of the observation)

```
contribution = normalizedValue × weight / 100
LongScore  = Σ contributions voting BUY   × 100   (0–100 scale)
ShortScore = Σ contributions voting SELL  × 100
```

**Conflict penalty**: if the bigger side is contradicted by higher timeframes
(e.g. M1 says BUY but H1 candle is bearish) → subtract 3 points per conflicting
frame from the dominant side. Evidence that fights itself costs points.

**Hard HTF veto**: BUY blocked if price < H1 close; SELL blocked if price >
H1 close (regardless of score).

**Thresholds** (per strategy × per regime, overridable via PAT_CANDIDATE_THRESHOLD
/ PAT_TRADE_THRESHOLD; live defaults candidate 10 / trade 25 for many rows):
```
score ≥ trade threshold      → EXECUTABLE-candidate (goes to gates)
candidate ≤ score < trade    → CANDIDATE (directional but shadow-grade)
score < candidate threshold  → NO-TRADE
```

**Grades** (quality_grade.go):
```
REJECT  : regime not allowed, or RR ≤ 0, or expectancy < −0.5R
A+      : score ≥ 70 AND RR(TP2) ≥ 2.0 AND structure confirmed AND expectancy > +0.3R
A       : score ≥ 55 AND RR(TP1) ≥ 1.3 AND expectancy ≥ 0
B       : score ≥ 30, not candidate → shadow only
C       : candidate with score ≥ 20 → shadow only
```
Only A-/A+ grades can be delivered as executable; B/C are recorded for research
and never delivered.

**Family caps**: per pillar family (e.g. all MOMENTUM pieces together) there is
a cap so one family can't dominate the score — confluence must be *broad*.

Main evidence pieces and their point values (legacy matrix, weight × norm × 100):

| Pillar | Observation | Points |
|---|---|---|
| STRUCTURE | BOS (break of structure) | ~2.5 |
| TREND | EMA9 above/below EMA21 | ~1.8 |
| TREND | ADX bullish/bearish | ~0.7 |
| VWAP | price above/below VWAP | ~1.0 |
| CANDLE | displacement (big body) | ~1.5 |
| CANDLE | rejection wick | ~1.0 |
| CANDLE | pinbar (quality-scaled) | ~1.0 |
| LIQUIDITY | sweep + reclaim | ~1.0 |
| MOMENTUM | MACD cross | ~0.6 |
| MOMENTUM | OsMA flip | ~0.4 |
| MOMENTUM | RSI mid-zone push | ~0.4 |
| MTF | alignment (scales with score) | ≤0.5 |
| SESSION_ORB | breakout of opening range | ~0.8 |
| STRUCTURE | pullback to zone | ~1.5 |

So a typical A-grade signal is not ONE magic indicator — it's 6–10 independent
measurements all pointing the same way, summing past the bar.

### 3.1 ULTRA_SCALPING — 1-minute micro scalps
- **Idea**: catch 3–10 point bursts during active sessions; each scalp must
  clear broker cost by design (micro-TP math in Part 5).
- **Timeframes**: decides on M1, context M5/M15. All regimes. Spread ≤ 1.5 pips,
  slippage ≤ 5 pts. Min ATR 3.0 (engine) — dead-flat market = no trade.
- **Geometry** (engine overrides): SL = 0.5×ATR (~3–4 pts), TP1 = 0.5×ATR,
  TP2 = 0.8×ATR, TP3 = 1.2×ATR, expiry 5 min, cooldown 5 min.
- **Micro-TP**: partial-close 50% of the position at 0.5×ATR.
- Signature evidence: EMA9/21 stack + MACD + displacement candle + BOS + VWAP side.

### 3.2 STANDARD_SCALPING — M1/M5 momentum bursts
- **Timeframes**: decides on M1+M5, context M15/M30. Spread ≤ 2.5, slippage ≤ 10.
- **Geometry**: SL = 0.8×ATR (~6–7 pts), TPs 1.0/1.5/2.5×ATR, expiry 10 min,
  cooldown 15 min. Min ATR 2.0. Micro-TP 0.8×ATR, close 40%.

### 3.3 STANDARD_SWING — M15–H1 structured swings
- **Timeframes**: decides on M15/M30/H1, context H4/D1. Spread ≤ 4, slippage ≤ 20.
- **Geometry**: SL = 1.0×ATR (engine) / 2.0×ATR (legacy), TPs 2.0/3.5/5.0×ATR
  (engine), expiry 60 min, cooldown 120 min. Structural SL option: behind the
  last swing low/high (structure > ATR when larger).
- Note: SL ≥ 3× full spread enforced (MinSLSpreadMult) — cost can never eat the stop.

### 3.4 TREND_SWING — H1/H4/D1 continuation, biggest stops
- **Timeframes**: decides on H1/H4, context D1/W1. Only TREND/BREAKOUT/HIGH_VOL
  regimes. Expiry 240 min, cooldown 360 min. Spread ≤ 5, slippage ≤ 30.
- **Special**: during RANGE weather it still computes *transition evidence* —
  "is this range breaking out?" — with its own lower thresholds (20 candidate /
  35 trade). Transition candidates are tracked but delivered only when they
  graduate.
- **Geometry**: SL = 1.5×ATR (engine), TPs 3.0/5.0/8.0×ATR, Ichimoku cloud
  confirmation for swing TF trend.

### 3.5 MARNIE_FIB — golden-zone retracement reversals
- **Idea**: price retraces into the 0.618–0.786 golden zone of a confirmed
  swing → enter with the swing direction; near 0.382/0.5 → weaker, candidate
  only; beyond 1.0 → treat as continuation, use extensions as targets.
- **Timeframes**: decides on M15/H1, context H4/D1. Min ADX 15 (works in ranges).
- **Geometry**: SL beyond the 0.786 (or 1.0) level with ATR buffer
  (SL = 1.5×ATR base), TPs at extensions 1.272/1.618/2.618 mapped to
  2.0/3.5/5.5×ATR. Expiry 120 min, cooldown 180. Micro-TP 0.7×ATR, 40%.
- Score boost when current Fib level confluences with a *previous* Fib level.

### 3.6 ATEN — astro-confluence engine (Aetherial Technical Engine Node)
- **Idea**: planetary/timing confluence as a *bias layer*, not a momentum signal.
```
Vedic composite  = Nakshatra bias × 0.3 + Hora bias × 0.2
                 + Vimshottari dasha (L1+L2) × 0.5
Western aspects  = 28% of the final composite (Gold natal chart aspects)
Signal gate      : |composite| ≥ 25  → directional bias
Eclipse / contamination → MANDATORY_NO_TRADE (hard override)
```
- **Geometry** (engine): SL = 1.0×ATR, TPs 1.0/2.0/3.0×ATR, expiry 60 min.
- **Easy words**: the slow celestial tide — used to *tilt* the other engines'
  conclusions, with a hard off-switch during eclipse windows.

### 3.7 ARCANIST — institutional killzone model (strictest strategy)
Nine ordered checks; ANY failure = NO-TRADE with a reason code:
1. **Hard stop after 17:00 UTC** (thin liquidity).
2. **News risk** must not be BLOCKED.
3. **Session gate**: Tokyo/London/Overlap/NY only.
4. **HTF bias**: Weekly + Daily must agree on direction.
5. **POIs**: fresh H4/H1 order blocks aligned with bias must exist.
6. **Asian range** (00:00–06:00 UTC on M15) must be measurable.
7. **Judas sweep**: liquidity sweep of the Asian range in the bias direction
   (the fake move) must have happened.
8. **BOS** on the execution TF (M15) must confirm the bias direction after the sweep.
9. **Entry at the nearest fresh order block**; SL = gold-pip-aware 20–25 pips
   ($2.00–$2.50; 1 gold pip = $0.10); TP1 = opposite edge of the Asian range;
   TP2 = 3× the SL distance (min RR to entry-range-level 3.0); score must be ≥ 70.

**Easy words**: wait for the morning fake-out, trade the real move that follows
it, only when the weekly and daily agree, and only in liquid hours.

### 3.8 The Playbook Confirmation Gate (last-mile checklist, all strategies)
Before ANY signal becomes executable it must pass this 4-point checklist
(features/confirmation_gate.go):

**Trigger candle rules (§9.2)**:
- candle body ≥ 50% of its range (no dojis)
- BUY trigger must close above EMA9 AND EMA21 AND the prior bar's high
- SELL trigger must close below EMA9 AND EMA21 AND the prior bar's low
- opposite wick < 60% of range (no distribution wick on a buy)

**4-point score** (needs ≥ 3, per-strategy minimums in playbook §9.3):
1. EMA9 vs EMA21 on the right side (+1)
2. Parabolic SAR on the right side (+1)
3. CCI positive & rising (or negative & falling for sells) (+1)
4. StochRSI K>D & rising (or K<D & falling) (+1)

**Spread sanity**:
- SL distance must be ≥ 5 × full spread
- trigger candle range must be ≥ 3 × full spread

**Easy words**: the composite score can say "go", but four independent quick
instruments must also agree *right now* — the final handshake before delivery.

### 3.9 Shadow evaluation
When a strategy rejects ONLY because of regime mismatch, the engine still
records what the score would have been (SHADOW_ONLY=true, EXECUTABLE=false).
Shadow signals are **never delivered, never executed** — they exist so the
calibration layer can learn from counterfactual outcomes ("what if we had?").

---

## PART 4 — SCORE → HONEST PROBABILITY (calibration maths)

Raw scores are opinion; clients need calibrated probabilities.

```
x = clamp(score, 0, 100) / 100
probability = 1 / (1 + e^−(a·x + b))       (sigmoid, per-strategy a/b)
```
- a/b are learned from **real resolved outcomes** (live calibrator periodically
  retrains from closed trades + shadow outcomes; JSON models reload into the engine).
- Only models with status VALIDATED or PROMOTED are used. Unvalidated seed
  models are NOT applied — showing a made-up confidence is forbidden.
- Same maths runs offline in the backtest engine, so backtest %s and live %s
  use identical formulas.

**Win-rate model used inside the EV gate** (cheap closed-form approximation):
```
winRate = 0.5 + (score − 55)/100 × 0.5
        + 0.05 (trend/breakout) | +0.02 (range/mean-rev) | −0.03 (high vol)
clamped to [0.35, 0.82]
```

---

## PART 5 — THE 23 RISK GATES (veto wall)

Every executable candidate passes all of these. One veto = no trade, with a
reason code stored in the DB (that's what you see as "DETECTED but not executed").

| # | Gate | Veto condition (plain words) |
|---|---|---|
| 1 | data_quality | candles/stale/NaN — data not trustworthy |
| 2 | session | outside the strategy's allowed sessions |
| 3 | news | red-news window (BLOCKED) |
| 4 | spread | current spread > strategy max (1.5–5 pips) |
| 5 | slippage | recent slippage > strategy max (5–30 pts) |
| 6 | total_cost | round-trip cost > 30% of TP1 distance (scalps, strict) |
| 7 | exposure | too many open exposure units for the symbol |
| 8 | margin | order would exceed margin budget (≤ 30% free margin) |
| 9 | rr_net_expectancy | net RR < 0.5 after costs (POOR_STRUCTURAL_RR) |
| 10 | entitlement | client plan doesn't include this strategy |
| 11 | license | license inactive/expired |
| 12 | execution_permission | automation not permitted on the account |
| 13 | stop_hunt_filter | SL sits in an obvious stop-hunt zone |
| 14 | min_atr | ATR below the strategy floor (dead market) |
| 15 | wrong_side_sl | SL on the wrong side of entry for the direction |
| 16 | risk_oversize | risk per trade over the account cap |
| 17 | position_caps | too many positions / same-direction stacking |
| 18 | daily_loss | daily drawdown limit hit (trading pauses) |
| 19 | profit_target | daily/weekly profit target hit (lock-in mode) |
| 20 | martingale_ban | doubling-down pattern detected → refused |
| 21 | edge_validation | live results deviate from backtest edge beyond tolerance |
| 22 | broker_symbol_validation | symbol spec mismatch vs broker |
| 23 | profitability | expectancy = winRate×netWin − (1−winRate)×netLoss ≤ 0 |

**Cost model behind gates 6/9/23** (config defaults):
```
round-trip cost = spread + slippage (0.10 pts) + commission (0.06 pts)
netWin  = TP distance − cost
netLoss = SL distance + cost
netRR   = netWin / netLoss          → must be ≥ 0.5
EV per 1R = winRate × netWin − (1 − winRate) × netLoss   → must be > 0
```
(v1.24 note: the profit_target gate's equity anchors now re-anchor on ANY
sustained ≥50% equity jump with zero open positions — account switches can
no longer masquerade as +8748% profits.)

---

## PART 6 — CAPITAL TIERS & LOT SIZE

### 6.1 Who may receive the signal (tier maths)
```
min-lot risk = SL distance in points × $1 per point (per 0.01 lot)
MICRO    tier: ref $100,  cap 2% = $2    → fits SL ≤ 2 pts
STANDARD tier: ref $500,  cap 2% = $10   → fits SL ≤ 10 pts
PRO      tier: ref $5,000, cap 2% = $100 → fits SL ≤ 100 pts
```
A signal carries its computed EligibleTiers at generation. Delivery enqueues the
signal ONLY to devices whose entitlement tier is in that list (fail-closed).
**Easy words**: a $100 account must never be handed a trade whose stop can cost
$22. The maths decides, not a human mood.

### 6.2 Lot size
```
riskBudget = equity × 1.5%              (MAX_RISK_PER_TRADE_PCT)
lot = riskBudget / (SL points × $ per point per lot)
lot = floor to 0.01 steps, ≥ 0.01, and margin ≤ 30% of free margin
```
Micro-TP partial close sizes (per strategy): ULTRA 50% at 0.5×ATR,
SCALP 40% at 0.8×ATR, SWING 35% at 1.2×ATR, TREND 30% at 1.8×ATR,
MARNIE 40% at 0.7×ATR.

### 6.3 Micro profit-taking must pay the bill
For every scalp: `micro-TP value ≥ round-trip cost × buffer`. A trade whose
first profit-taking can't cover spread + slippage + commission is vetoed
(`MICRO_TP_UNPROFITABLE`) — that's the whole "micro TP must cover the broker
fee + slippage" rule, enforced in maths, not hope.

---

## PART 7 — DELIVERY (store-then-deliver, fail-closed)

1. Signal row is written to `trading.signals` FIRST (permanent record).
2. `executable=true` signals are enqueued to `licensing.edge_signal_queue` for
   every eligible device; `executable=false` are stored for research only.
3. Client EA polls `GET /edge/poll` every ~2s over HTTPS with its device token;
   server hands the next PENDING item; EA executes on its own account; server
   marks ACKED with latency + ack payload.
4. **EA-side safety** (client devices have their own gates):
   - price-drift gates per tier (ULTRA 15 / SCALP 25 / SWING 60 / TREND+others
     80 points) — if price moved too far from entry before the EA acts, skip
   - TTL expiry per strategy (5–240 min) — stale signals auto-dropped
   - own spread check, own margin check, server-command envelope handling,
     license token self-heal.
5. ACK loop closes the chain: `signal_deliveries` records ack latency; the
   watchdog alerts if a device stops polling for 3 minutes.

**Delivery latency measured live**: 88% of ACKs < 10s; slow tail = devices
that were offline and ACK on reconnect (expected).

---

## PART 8 — THE LEARNING LOOP

1. Closed trades + shadow outcomes are persisted with full feature snapshots.
2. Live calibrator periodically retrains sigmoid a/b per strategy from resolved
   outcomes; writes calibration JSON; engine hot-loads VALIDATED models.
3. Edge-validation gate compares live vs backtest expectancy; drift beyond
   tolerance vetoes new signals (protects clients from a decayed edge).
4. Backtest engine (0.3 ms/bar float64, 2.1M bars in ~5.5 min, parity-proven
   vs live) is the same maths — walk-forward runs feed threshold calibration.

---

## PART 9 — BEHAVIOUR YOU WILL SEE (and why)

| Symptom | Real cause |
|---|---|
| Many DETECTED signals, few EXECUTABLE | the 23-gate wall + score thresholds — by design (fail-closed) |
| `profit_target_hit` vetoes | daily/weekly lock-in; v1.24 fixed the account-switch misread |
| Occasional `ingest TICK failed: HTTP 502` | only during engine deploys (stateful singleton); nginx now retries once; scheduled in low-liquidity windows |
| Tick timestamps ~15 min behind | the MASTER terminal's PC clock; server receipt time is fresh (0.2s) — sync the Windows clock |
| MICRO/STANDARD clients get fewer signals | wide-stop signals are PRO-tier by the min-lot-risk maths — protection, not a bug |
| `Signal queued` in logs but no trade on device | check the device's entitlement tier vs the signal's EligibleTiers |
| B/C-grade signals in DB, never delivered | shadow/research records — never delivered by design |

## PART 10 — THE 5 PLANES (who decides nothing where)

| Plane | Tech | Decides | Never does |
|---|---|---|---|
| Real-time | Go engine | signals, risk, geometry, delivery | billing, referrals, payouts |
| Research | Python | backtests, calibration, ML | live tick path |
| Control | NestJS | IAM, billing, licensing, queue API | trade decisions |
| Presentation | Next.js | renders server truth | any authority |
| Edge | MT4/MT5 EA | executes on client account, local safety gates | primary intelligence, server credentials |

## PART 11 — SAFETY PRINCIPLES (the 6 invariants)

1. **Fail-closed**: any missing data, gate error, or unknown state = NO-TRADE.
2. **Store-then-deliver**: the signal row exists before any client sees it.
3. **Money never transits us**: EAs trade client accounts directly.
4. **NO-TRADE is a first-class result**: frequency is never forced.
5. **Calibrated honesty**: probabilities only from VALIDATED models; no
   fabricated confidence anywhere.
6. **Everything is auditable**: every veto has a reason code; every delivery
   has an ACK trail; every score has its evidence list.