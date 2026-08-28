# XAUUSD (Gold) Scalping Playbook — Logic, Factors, Numbers
*Built 28 Aug 2026. Spot gold ≈ $4,600/oz; Aug 2026 has been the strongest month in years (a
$4,324–$4,524 day on 18 Aug). High-volatility regime changes the rules — see §6.*

> Not financial advice. This is the operating logic used by short-timeframe gold traders and how to
> measure whether it works for you. Most retail scalpers lose; the loss is almost always cost + size,
> not the chart.

---

## 1. The core logic (read this first)

Scalping gold is **not** "find a great entry." It is a cost-and-selectivity problem:

```
Daily P&L  =  N × [ p·AvgWin − (1−p)·AvgLoss − Cost ]
              ↑        ↑          ↑           ↑         ↑
        trades taken  win rate   $ won     $ lost   spread+comm+slippage
```

Only three of those variables are under your control: **N**, **AvgLoss/Cost**, and **when you take
the trade**. Round-turn cost on gold is roughly **$19 per 1.0 lot** (12-point spread + $7/lot
commission) = **$0.19/oz you must beat before you are flat**. So:

| Target you're chasing | $/oz | stop $/oz | Cost as % of gross |
|---|---|---|---|
| 30 "pips" | 0.50 | 0.35 | **38%** |
| 100 pts | 1.00 | 0.60 | 19% |
| 150 pts | 1.50 | 1.00 | 12.7% |
| 250 pts | 2.50 | 1.50 | 7.6% |
| 400 pts | 4.00 | 2.50 | 4.8% |

**Consequence #1:** sub-$1.00 targets on gold are structurally negative-expectancy for 99% of
people. Target $1.20–$3.00 per ounce and stop *lowering* your timeframe instead of lowering your target.

**Consequence #2:** the profit ceiling is set by **R:R and selectivity**, not by lot size. See §8.

---

## 2. Factor 1 — Cost structure (the biggest lever, most ignored)

- Raw/ECN account only: 10–18-point spreads + $4–7/lot commission beats a "zero commission"
  standard account at 35–50 points. Over 100 trades that's ~$2,000–3,000 per standard lot of pure leakage.
- Measure **your** realised cost: (avg spread + commission + measured slippage) ÷ avg target.
  Keep it under 10–12%. If it's 20%, you need a 60% win rate just to be flat — you will not get it.
- Slippage rules: never market-buy *into* a widening spread after a 30-cent impulse; use
  buy-limit into the micro-pullback, and stop-*limit* (not stop-market) on breakout entries.
- Widened spread = flat rule: spread > 2× its session median → no new trades (rolls, news minutes, thin books).

## 3. Factor 2 — Timing (volatility is not evenly distributed)

Times for **late Aug 2026** in Dubai local (GST, UTC+4). Both shift ±1h at DST changes; re-check quarterly.

| Window | Dubai | Why it matters |
|---|---|---|
| Asia (quiet) | 04:00–12:00 | $20–40 ranges, wide-ish spreads. Range fades only; no trend trades |
| London pre-open drift | 12:00–13:00 | Positions into Asia high/low; spread tightens |
| **London open** | **13:00–15:00** | Highest pips/hour of the day, best pattern reliability |
| US data (08:30 ET) | 16:30 (17:30 winter) | CPI/NFP/PPI — 100–300-cent impulses in 60s |
| London/NY overlap | 18:00–21:00 | Deepest liquidity, tightest spreads, cleanest trends (2nd best window) |
| NY PM close | 21:00–00:00 | Volume and range decay → flat by ~22:30–23:00 |
| FOMC (14:00 ET) | 22:00 (23:00 winter) | Stand aside ±20 min |

**Rule:** ~70–80% of a good scalper's month comes from two windows (13:00–15:00 and 18:00–21:00 Dubai).
Trading outside them converts a break-even edge into a loss via spread and chop.

## 4. Factor 3 — Volatility regime (sizing + targets scale with it)

- Read **M15 ATR(14)** and **Daily ATR**. Normal: day range $60–100. High-impact days: $150–300.
- Volatility filter: skip setups when M15 ATR < 0.8× its 50-day average → that's whipsaw land.
- Target and stop are **derived from ATR, never fixed in pips**:
  `stop = 1.2–1.6 × M5 ATR (or structural low + 0.10 buffer)`, `target = 1.5–2.5 × stop`.
- Higher ATR = wider stop = **smaller lots** so $ risk stays constant. That single discipline is what
  lets people survive gold's 30-second, $3 spikes.

## 5. Factor 4 — Level structure (where the moves actually start)

Mark once, before the session, on a clean chart:
**PDH / PDL / PDC** → **Asia high/low** → **London open ± 30 min high/low** → **current-day VWAP**
→ **prior-day VAH/VAL/POC** → round numbers (at $4,600: **$50 majors, $10 minors**; $500s are the
institutional magnets) → the **daily open** and **London 10:00 GMT fix**.

Logic behind gold's best intraday moves:
1. Liquidity pools sit just beyond obvious highs/lows (stop clusters + breakout orders).
2. Gold repeatedly **sweeps** the pool, then **reclaims** the level — that reversal, not the breakout,
   is the high-R:R scalp.
3. First touch of a level → usually holds (fade). Third/fourth touch → usually breaks (go with it).
4. Breakouts that fail within 3 candles reverse to the opposite side of the range — take the return leg.

## 6. Factor 5 — Macro/cross-asset direction filter

Gold is a USD-denominated, zero-coupon, real-rate asset. For intraday *bias* (not reasons), watch:
- **DXY** and **US 2y/10y yields** (real yields matter most): rising real yields = headwind.
- Fed path / CME FedWatch, and any **Treasury refunding or bond-market intervention** headlines —
  these drove August 2026's "debasement trade" bid alongside ETF and central-bank inflows.
- 30-min **correlation check** before a series of longs: if DXY is falling and yields falling, longs
  have a tailwind; if both rising, take shorts or nothing.
- Geopolitical spikes are fast and mean-reverting → take the impulse, don't marry the position.

## 7. Factor 6 — Setups worth repeating (one or two, not six)

| # | Setup | Condition | Entry / Stop / Target |
|---|---|---|---|
| 1 | **Sweep & reclaim** | Asia or PDH/PDL level + prior day | Sweep of level, close back inside on M1/M5 / beyond sweep wick 0.10 / VWAP → opposite extreme (1.5–3R) |
| 2 | **London open breakout** | Range ≥ $25 in prior 2h | Re-test of broken opening-range high/low / opposite side of range / 2× Asia-range width |
| 3 | **VWAP trend pullback** | Trend day: price > VWAP, VWAP rising | First pullback to VWAP + bullish M1 close / below pullback low / 2R, trail on M5 20-EMA |
| 4 | **Fail-to-continue reversal** | After a news impulse (17:30/22:00 Dubai) | 3–5 min after print, fade failed new extreme / above impulse high / 1.5–2R |
| 5 | **Asia range fade** | Only 04:00–11:30, no news | Tag of range extreme + rejection wick / 0.15 beyond extreme / mid-range to opposite edge |

Kill-list (no trade): spread blown, ATR dead, 5-min window around a tier-1 print, third consecutive
stop-out on the same level, Friday NY PM, day after a $250 range.

## 8. Factor 7 — Risk, sizing, exits (where "maximum profit" actually lives)

**Sizing:** `lots = (equity × risk%) ÷ (stop $/oz × 100)`.
$10k, 1% risk, $1.00 stop → **1.0 lot** ($460k notional, ~$4.6k margin at 1:100).
0.5% for prop-firm accounts; ≥1% on gold is how accounts die, because one news candle = 3–5× a normal stop.

**Break-even win rate (with $19/lot cost):** 1:1 → 59.5% · 1.5:1 → 47.6% · 2:1 → 39.7%.
So: **raise R:R, then take fewer trades.** A 65%-win, 1:1 scalper loses money; a 50%-win, 2:1 scalper prints.

**Exit logic:** half at 1R to BP (stop to BE), rest to a 2R/3R target, trail the remainder under each
M5 swing low after price tags VWAP+1 ATR. Never widen a stop; never move BE before 1R is paid.

**Daily circuit breakers:** stop at **−1.5R**, stop at **+3R**, hard stop 3.5h after session start,
max 6–8 trades/day, size halves after 2 consecutive losers, flat 2 min before a tier-1 print.

## 9. Traps that consume the entire edge

Averaging down / martingale · fixed lot size across volatility regimes · scalping a $0.50 target on a
$0.35 spread · trading news *spikes* instead of post-news reaction · widening stops to "give it room"
· revenge trading after a London stop-out · holding into NY PM or the Friday close · trusting a broker
feed whose prices lag the futures (compare to GC futures / spot 24h at all times) · adding size to
"catch up" after a losing week.

## 10. Measure like a desk, not a hobby

Weekly, from a journal (template: `journal_template.csv`) and MT4/MT5 statement:

- **Expectancy per trade** and **profit factor** (target > 1.5)
- **Cost as % of gross profit** (target < 10%)
- **MAE/MFE**: how far against did it go (is the stop 20 cents too tight?), how far in favour
  (are you exiting at 0.6R what a 2R level was worth?). This is the single highest-value chart you can draw.
- **Slippage stats** on entries/exits vs. intended price; per-session and per-time-of-day buckets
- **Setup-level stats**: at least 30 samples per setup before you touch the rules
- **R-multiples** journal (not dollars) — removes broker balance noise from your judgement

## 11. What "maximum" realistically means

A disciplined gold scalper's realistic ceiling is roughly **1–4% of equity per week** in a favourable
volatility regime, with 30–50% of days flat (no setup) and 5–8 trades on the days that qualify. Anyone
advertising "10% daily" on XAUUSD is either running invisible martingale or selling you the account they
blew up. Compounding at 2%/week ≈ 100%/year with a max drawdown you'd survive; the same account sized 4×
to force 8%/week has a near-certain blow-up inside a quarter.

---
`scalper_math.py` in this folder recomputes every number above: `python3 scalper_math.py`

## 12. Tools in this folder
- `scalper_math.py` — cost drag, break-even win rate, position size, expectancy ladder (edit the constants at the top for your broker).
- `journal_template.csv` — one row per trade; log R-multiples, not dollars, plus MAE/MFE in cents/oz.
- `scorecard.py` — `python3 scorecard.py journal_template.csv` → expectancy, profit factor,
  early-exit leakage, stop efficiency, spread drag.

Timezone cross-check for the tables above (Dubai GST = UTC+4, no DST):
London open 08:00 BST = 13:00 GST · US 08:30 ET data = 16:30 GST · FOMC 14:00 ET = 22:00 GST ·
London/NY overlap 13:00–17:00 UTC = 18:00–22:00 GST. Shift one hour when the UK/US change clocks.

# XAUUSD scalping chart — exact template (what goes on the screen)

Three windows, one instrument, two extra tabs. Nothing else. If a window is doing no
decision-making, close it — clutter is why scalpers hesitate.

```
┌───────────────────────────────┬───────────────────────────────┐
│  W1 · M15  "the map"          │  W3 · M1 "the trigger"        │
│  bias + levels only           │  entry candle + spread box    │
│  candles, VWAP, 20EMA, ATR    │  no indicators except VWAP    │
├───────────────────────────────┴───────────────────────────────┤
│  W2 · M5  "the setup" (largest window, where you live)         │
│  candles + VWAP + 20/50 EMA + Asia/London boxes + range stat  │
└───────────────────────────────────────────────────────────────┘
  TAB 4: DXY + US2y/10y side by side (30s glance, direction filter)
  TAB 5: economic calendar (today only, filter: gold / USD high+medium impact)
```

## Indicators, exact settings, and what each one decides

| Indicator | Setting | Decides | Notes |
|---|---|---|---|
| **Session VWAP** | anchored, reset 00:00 UTC (optional 2nd: anchored to London open) | long-side vs short-side bias, pullback magnet | the single best intraday anchor on gold |
| **ATR(14)** | M5 for stop size, M15 for regime | stop distance, "is today worth trading" | stop = 1.5× M5 ATR, never a fixed pip count |
| **20 EMA** | M5, close-based | whether today is a trend day | price holding VWAP + 20EMA = pullbacks are buyable |
| **50 EMA** | M15 | the intraday value zone | a reclaim of M15 50EMA after a sweep = highest-quality long |
| **Session range / HLC box** | Asia 04:00–12:00 GST, London 13:00–13:30 GST | breakout targets, fade zones | draw as a box, not a line |
| **PDH / PDL / PDC** | from prior day (use `fetch_levels.py`) | liquidity pools for sweeps | thickest lines on the chart |
| **Pivots (classic, daily)** | R1/R2/S1/S2 | secondary magnets and target selection | dashed, thin, never a reason to trade alone |
| **Volume Profile (session)** | shape 24h, POC/VAH/VAL | where price should mean-revert | only if your platform has real futures volume |
| **Relative volume** (optional) | vs 10-day same-time-of-day | filters fake breakouts | spot FX usually can't show this → use futures feed |
| **Spread widget** | live bid/ask, plus 1h median | GO / NO-GO | no new trades if spread > 2× session median |
| **Heikin-Ashi** (optional) | M15 window only | makes trend days obvious visually | NEVER take levels/slippage numbers off HA candles |

## Deliberately left off the chart
RSI/Stoch overbought-oversold (gold trends 3–4 hours without a breath; an "OB" short into a
sweep is the classic donor account) · Bollinger + MACD + Stoch stacked (three lagging
re-arrangements of the same closes) · Ichimoku, Parabolic SAR, Supertrend on M1 (too laggy to
matter at a $1.50 target) · moving-average crossovers as entries (they fire after the scalp is over) ·
indicator-driven "confluence" stacking — 5 indicators agreeing means 5 versions of the same 3 candles.

## The one visual edge that isn't an indicator
Mark **where the stops are**, then wait for them to be taken out. On gold that means: obvious
double top/bottom, Asia high/low, PDH/PDL, and $50/$100 rounds. A candle that wicks through one
of those and closes back inside, on rising range, is the highest-expectancy picture a discretionary
scalper gets on XAUUSD. Everything else on the chart is a tiebreaker.

## Instrument choice matters
Spot XAUUSD (CFD/OTC) is tick-volume only — "volume" indicators are fake there. If you want real
volume/footprint/DOM, chart **COMEX GC futures** (or an OANDA/TradingView feed mapped to it) for
levels and execution on your CFD; prices track within a few cents intraday. Then: real volume,
real session VWAP, delta/CVD, DOM, futures-only liquidity hours.

------------------------------
## 📈 Trend and Moving Averages

* EMA (Exponential Moving Average)
* How to use: Identify dynamic support/resistance levels. Trade crossovers (e.g., 9 EMA crossing 21 EMA) to enter trends. Use the 200 EMA to determine the long-term market bias.
* SMA (Simple Moving Average)
* How to use: Assess institutional trend direction using 50, 100, and 200 periods. Price above the SMA indicates a bullish environment; price below indicates a bearish environment.

------------------------------
## 📊 Momentum and Oscillators

* MACD (Moving Average Convergence Divergence)
* How to use: Trade centerline crossovers (above/below zero) for trend confirmation. Trade signal line crossovers for early entry. Spot momentum shifts via bullish/bearish divergences.
* RSI (Relative Strength Index)
* How to use: Identify overbought conditions above 70 and oversold conditions below 30. Look for centerline (50) breaks to confirm trend strength. Spot trend reversals using price-to-oscillator divergences.
* Stochastic Oscillator
* How to use: Identify rapid cyclical turning points. Buy when the %K line crosses above the %D line below the 20 oversold level. Sell when it crosses below above the 80 overbought level.
* CCI (Commodity Channel Index)
* How to use: Detect new trends or extreme cyclical conditions. Readings above +100 signal strong bullish momentum. Readings below -100 signal strong bearish momentum.
* StochRSI (Stochastic RSI)
* How to use: Apply as a hyper-sensitive momentum gauge. Identify extreme overbought/oversold states much faster than traditional RSI to time precise entries in sideways markets.

------------------------------
## 🔎 Volatility and Strength

* ADX (Average Directional Index)
* How to use: Measure overall trend strength regardless of direction. A reading above 25 indicates a strong trending market. A reading below 20 indicates a weak, range-bound market.
* +DI / -DI (Directional Indicators)
* How to use: Determine specific trend direction. A bullish signal triggers when +DI crosses above -DI. A bearish signal triggers when -DI crosses above +DI.
* Bollinger Bands
* How to use: Trade volatility expansions and contractions. Buy near the lower band during ranges and sell near the upper band. Watch for a "Bollinger Band Squeeze" to anticipate explosive breakouts.
* ATR (Average True Range)
* How to use: Gauge current market volatility. Multiply the ATR value (e.g., 1.5x or 2x ATR) to set dynamic trailing stop-losses and calculate position sizes based on market noise.

------------------------------
## 💡 Volume and Complex Systems

* OBV (On-Balance Volume)
* How to use: Confirm price trends using volume flow. Rising OBV validates a price uptrend. Falling OBV validates a price downtrend. Divergences between OBV and price signal smart-money accumulation or distribution.
* Parabolic SAR (Stop and Reverse)
* How to use: Define trailing stop-loss placements. Dots trailing below the price indicate an uptrend. Dots flipping above the price signal a trend reversal and act as an exit trigger.
* Ichimoku Cloud
* How to use: Evaluate support, resistance, trend direction, and momentum simultaneously. Buy when the price breaks above the Cloud (Kumo) while the Tenkan-sen crosses above the Kijun-sen. Sell when the price drops below the Cloud.

## 📥 Entry Rules

* Long (Buy):
1. Price must be trading completely above the 200 SMA.
   2. The ADX must be above 25, confirming a strong trending market.
   3. The +DI line must be above the -DI line.
   4. Enter the trade immediately when the 9 EMA crosses above the 21 EMA.
* Short (Sell):
1. Price must be trading completely below the 200 SMA.
   2. The ADX must be above 25.
   3. The -DI line must be above the +DI line.
   4. Enter the trade immediately when the 9 EMA crosses below the 21 EMA.

## 📤 Exit Rules

* Stop Loss: Place your stop loss exactly 2 × ATR away from your entry price to protect against market noise.
* Take Profit: Exit the position manually when the 9 EMA crosses back over the 21 EMA in the opposite direction, OR when the ADX falls below 20 (signaling the trend has died).

------------------------------
## ⚡ Strategy 2: The Multi-Timeframe Momentum Scalper
Best for: Fast-paced, high-probability momentum entries (Ultra/Intraday setups).
Indicators used: 50 EMA (Trend filter), MACD (Momentum confirmation), Stochastic (Execution trigger).
## 📥 Entry Rules

* Long (Buy):
1. Price must be trading above the 50 EMA.
   2. The MACD histogram must be above 0 (positive momentum).
   3. Wait for the Stochastic lines to drop below 20 (oversold condition).
   4. Enter the trade when the Stochastic %K crosses above the %D line while still inside or just leaving the oversold zone.
* Short (Sell):
1. Price must be trading below the 50 EMA.
   2. The MACD histogram must be below 0 (negative momentum).
   3. Wait for the Stochastic lines to rise above 80 (overbought condition).
   4. Enter the trade when the Stochastic %K crosses below the %D line while inside or just leaving the overbought zone.

## 📤 Exit Rules

* Stop Loss: Place your stop loss just above the recent swing high (for shorts) or below the recent swing low (for longs).
* Take Profit: Secure profits when the Stochastic lines reach the opposite extreme (above 80 for longs, or below 20 for shorts), OR target a fixed 1:1.5 Risk-to-Reward ratio.

------------------------------
## 💥 Strategy 3: The Volatility Breakout Squeeze
Best for: Exploiting sudden market expansions after long periods of consolidation.
Indicators used: Bollinger Bands (Volatility), RSI (Directional momentum), OBV (Volume validation).
## 📥 Entry Rules

* Long (Buy):
1. The Squeeze: The Bollinger Bands must contract narrowly, sitting horizontally for at least 15–20 periods.
   2. The Breakout: A full candlestick must break and close completely above the Upper Bollinger Band.
   3. Volume Confirmation: The OBV line must be rising or breaking above its own recent resistance peak.
   4. Momentum Confirmation: The RSI must be above 50 (preferably breaking above 60).
* Short (Sell):
1. The Squeeze: The Bollinger Bands must contract narrowly and sit horizontally.
   2. The Breakout: A full candlestick must break and close completely below the Lower Bollinger Band.
   3. Volume Confirmation: The OBV line must be falling or breaking below its own recent support floor.
   4. Momentum Confirmation: The RSI must be below 50 (preferably breaking below 40).

## 📤 Exit Rules

* Stop Loss: Place your stop loss exactly at the Middle Bollinger Band (20 SMA) line at the time of entry.
* Take Profit: Trail your stop loss along the Middle Bollinger Band as it moves, or exit the trade completely when the price touches the opposite outer Bollinger Band.

------------------------------
## 🧮 The Core Mathematical Formula
To find your position size, use this formula:
$$\text{Position Size} = \frac{\text{Account Balance} \times \text{Risk Percentage}}{\text{ATR} \times \text{ATR Multiplier} \times \text{Tick/Pip Value}}$$ 
------------------------------
## 📌 Step 1: Define Your Input Variables
Your calculator requires five primary inputs from the user before every trade:

   1. Account Balance ($): The total capital currently in your trading account (e.g., $10,000).
   2. Risk Percentage (%): The maximum percentage of your account you are willing to lose on this single trade. Standard risk management is 1% to 2%.
   3. Current ATR Value: The absolute value read directly from your 14-period ATR indicator on your charting platform.
   4. ATR Multiplier: The volatility multiplier from your strategy rules (e.g., 2×ATR from Strategy 1).
   5. Asset Specifications (Pip/Tick Value): The monetary value of a single unit movement for that specific asset class (e.g., $1.00 per dollar move in stocks, or $10 per pip in Forex standard lots).

------------------------------
## 📈 Step 2: Spreadsheet Architecture (Excel / Google Sheets)
Set up your spreadsheet columns exactly like this to automate the math:

| Cell | Label | Description / Formula | Example Input |
|---|---|---|---|
| A2 | Account Balance | Total trading capital | 10000 |
| B2 | Risk Percentage | Max risk per trade (decimal or %) | 0.01 (for 1%) |
| C2 | Current ATR | 14-period ATR reading | 2.50 |
| D2 | ATR Multiplier | Volatility stop multiplier | 2 |
| E2 | Asset Point Value | Value of 1 full point move per 1 unit | 1 (for standard stocks) |
| F2 | Total Dollar Risk | =A2 * B2 | Calculates: $100 |
| G2 | Stop Loss Distance | =C2 * D2 | Calculates: 5.00 points |
| H2 | Final Position Size | =F2 / (G2 * E2) | Calculates: 20 Shares |

------------------------------
## 💡 Step 3: Practical Asset Examples
Because different asset classes treat price increments differently, your Asset Point Value (Column E) changes depending on what you trade:
## 1. Stock Trading Example

* Account: $10,000 | Risk: 1% ($100)
* Stock Price: $150.00 | ATR: $2.50 | Multiplier: 2×
* Stop Loss Distance: 2.50 × 2 = $5.00
* Position Size: $100 ÷ $5.00 = 20 Shares

## 2. Forex Trading Example (EUR/USD)

* Account: $10,000 | Risk: 1% ($100)
* ATR: 0.0025 (25 pips) | Multiplier: 2×
* Stop Loss Distance: 0.0025 × 2 = 0.0050 (50 pips)
* Position Size (Units): $100 ÷ 0.0050 = 20,000 Units (0.2 Micro Lots)

------------------------------
## ⚠️ Critical Guardrail: The Leverage Check
Always add a safety mechanism column to check your Notional Position Value against your actual account balance.

* Formula: = Position Size × Current Asset Price
* Why it matters: In low-volatility markets, the ATR will shrink, causing the formula to suggest a massive position size. If the required capital exceeds your account balance (or your allowed leverage limits), you must override the calculator and cap the trade size at your maximum allowable leverage.

------------------------------

------------------------------
## 📈 Trend and Moving Averages

* EMA (Exponential Moving Average)
* How to use: Identify dynamic support/resistance levels. Trade crossovers (e.g., 9 EMA crossing 21 EMA) to enter trends. Use the 200 EMA to determine the long-term market bias.
* SMA (Simple Moving Average)
* How to use: Assess institutional trend direction using 50, 100, and 200 periods. Price above the SMA indicates a bullish environment; price below indicates a bearish environment.

------------------------------
## 📊 Momentum and Oscillators

* MACD (Moving Average Convergence Divergence)
* How to use: Trade centerline crossovers (above/below zero) for trend confirmation. Trade signal line crossovers for early entry. Spot momentum shifts via bullish/bearish divergences.
* RSI (Relative Strength Index)
* How to use: Identify overbought conditions above 70 and oversold conditions below 30. Look for centerline (50) breaks to confirm trend strength. Spot trend reversals using price-to-oscillator divergences.
* Stochastic Oscillator
* How to use: Identify rapid cyclical turning points. Buy when the %K line crosses above the %D line below the 20 oversold level. Sell when it crosses below above the 80 overbought level.
* CCI (Commodity Channel Index)
* How to use: Detect new trends or extreme cyclical conditions. Readings above +100 signal strong bullish momentum. Readings below -100 signal strong bearish momentum.
* StochRSI (Stochastic RSI)
* How to use: Apply as a hyper-sensitive momentum gauge. Identify extreme overbought/oversold states much faster than traditional RSI to time precise entries in sideways markets.

------------------------------
## 🔎 Volatility and Strength

* ADX (Average Directional Index)
* How to use: Measure overall trend strength regardless of direction. A reading above 25 indicates a strong trending market. A reading below 20 indicates a weak, range-bound market.
* +DI / -DI (Directional Indicators)
* How to use: Determine specific trend direction. A bullish signal triggers when +DI crosses above -DI. A bearish signal triggers when -DI crosses above +DI.
* Bollinger Bands
* How to use: Trade volatility expansions and contractions. Buy near the lower band during ranges and sell near the upper band. Watch for a "Bollinger Band Squeeze" to anticipate explosive breakouts.
* ATR (Average True Range)
* How to use: Gauge current market volatility. Multiply the ATR value (e.g., 1.5x or 2x ATR) to set dynamic trailing stop-losses and calculate position sizes based on market noise.

------------------------------
## 💡 Volume and Complex Systems

* OBV (On-Balance Volume)
* How to use: Confirm price trends using volume flow. Rising OBV validates a price uptrend. Falling OBV validates a price downtrend. Divergences between OBV and price signal smart-money accumulation or distribution.
* Parabolic SAR (Stop and Reverse)
* How to use: Define trailing stop-loss placements. Dots trailing below the price indicate an uptrend. Dots flipping above the price signal a trend reversal and act as an exit trigger.
* Ichimoku Cloud
* How to use: Evaluate support, resistance, trend direction, and momentum simultaneously. Buy when the price breaks above the Cloud (Kumo) while the Tenkan-sen crosses above the Kijun-sen. Sell when the price drops below the Cloud.

## 📥 Entry Rules

* Long (Buy):
1. Price must be trading completely above the 200 SMA.
   2. The ADX must be above 25, confirming a strong trending market.
   3. The +DI line must be above the -DI line.
   4. Enter the trade immediately when the 9 EMA crosses above the 21 EMA.
* Short (Sell):
1. Price must be trading completely below the 200 SMA.
   2. The ADX must be above 25.
   3. The -DI line must be above the +DI line.
   4. Enter the trade immediately when the 9 EMA crosses below the 21 EMA.

## 📤 Exit Rules

* Stop Loss: Place your stop loss exactly 2 × ATR away from your entry price to protect against market noise.
* Take Profit: Exit the position manually when the 9 EMA crosses back over the 21 EMA in the opposite direction, OR when the ADX falls below 20 (signaling the trend has died).

------------------------------
## ⚡ Strategy 2: The Multi-Timeframe Momentum Scalper
Best for: Fast-paced, high-probability momentum entries (Ultra/Intraday setups).
Indicators used: 50 EMA (Trend filter), MACD (Momentum confirmation), Stochastic (Execution trigger).
## 📥 Entry Rules

* Long (Buy):
1. Price must be trading above the 50 EMA.
   2. The MACD histogram must be above 0 (positive momentum).
   3. Wait for the Stochastic lines to drop below 20 (oversold condition).
   4. Enter the trade when the Stochastic %K crosses above the %D line while still inside or just leaving the oversold zone.
* Short (Sell):
1. Price must be trading below the 50 EMA.
   2. The MACD histogram must be below 0 (negative momentum).
   3. Wait for the Stochastic lines to rise above 80 (overbought condition).
   4. Enter the trade when the Stochastic %K crosses below the %D line while inside or just leaving the overbought zone.

## 📤 Exit Rules

* Stop Loss: Place your stop loss just above the recent swing high (for shorts) or below the recent swing low (for longs).
* Take Profit: Secure profits when the Stochastic lines reach the opposite extreme (above 80 for longs, or below 20 for shorts), OR target a fixed 1:1.5 Risk-to-Reward ratio.

------------------------------
## 💥 Strategy 3: The Volatility Breakout Squeeze
Best for: Exploiting sudden market expansions after long periods of consolidation.
Indicators used: Bollinger Bands (Volatility), RSI (Directional momentum), OBV (Volume validation).
## 📥 Entry Rules

* Long (Buy):
1. The Squeeze: The Bollinger Bands must contract narrowly, sitting horizontally for at least 15–20 periods.
   2. The Breakout: A full candlestick must break and close completely above the Upper Bollinger Band.
   3. Volume Confirmation: The OBV line must be rising or breaking above its own recent resistance peak.
   4. Momentum Confirmation: The RSI must be above 50 (preferably breaking above 60).
* Short (Sell):
1. The Squeeze: The Bollinger Bands must contract narrowly and sit horizontally.
   2. The Breakout: A full candlestick must break and close completely below the Lower Bollinger Band.
   3. Volume Confirmation: The OBV line must be falling or breaking below its own recent support floor.
   4. Momentum Confirmation: The RSI must be below 50 (preferably breaking below 40).

## 📤 Exit Rules

* Stop Loss: Place your stop loss exactly at the Middle Bollinger Band (20 SMA) line at the time of entry.
* Take Profit: Trail your stop loss along the Middle Bollinger Band as it moves, or exit the trade completely when the price touches the opposite outer Bollinger Band.

------------------------------
## 🧮 The Core Mathematical Formula
To find your position size, use this formula:
$$\text{Position Size} = \frac{\text{Account Balance} \times \text{Risk Percentage}}{\text{ATR} \times \text{ATR Multiplier} \times \text{Tick/Pip Value}}$$ 
------------------------------
## 📌 Step 1: Define Your Input Variables
Your calculator requires five primary inputs from the user before every trade:

   1. Account Balance ($): The total capital currently in your trading account (e.g., $10,000).
   2. Risk Percentage (%): The maximum percentage of your account you are willing to lose on this single trade. Standard risk management is 1% to 2%.
   3. Current ATR Value: The absolute value read directly from your 14-period ATR indicator on your charting platform.
   4. ATR Multiplier: The volatility multiplier from your strategy rules (e.g., 2×ATR from Strategy 1).
   5. Asset Specifications (Pip/Tick Value): The monetary value of a single unit movement for that specific asset class (e.g., $1.00 per dollar move in stocks, or $10 per pip in Forex standard lots).

------------------------------
## 📈 Step 2: Spreadsheet Architecture (Excel / Google Sheets)
Set up your spreadsheet columns exactly like this to automate the math:

| Cell | Label | Description / Formula | Example Input |
|---|---|---|---|
| A2 | Account Balance | Total trading capital | 10000 |
| B2 | Risk Percentage | Max risk per trade (decimal or %) | 0.01 (for 1%) |
| C2 | Current ATR | 14-period ATR reading | 2.50 |
| D2 | ATR Multiplier | Volatility stop multiplier | 2 |
| E2 | Asset Point Value | Value of 1 full point move per 1 unit | 1 (for standard stocks) |
| F2 | Total Dollar Risk | =A2 * B2 | Calculates: $100 |
| G2 | Stop Loss Distance | =C2 * D2 | Calculates: 5.00 points |
| H2 | Final Position Size | =F2 / (G2 * E2) | Calculates: 20 Shares |

------------------------------
## 💡 Step 3: Practical Asset Examples
Because different asset classes treat price increments differently, your Asset Point Value (Column E) changes depending on what you trade:
## 1. Stock Trading Example

* Account: $10,000 | Risk: 1% ($100)
* Stock Price: $150.00 | ATR: $2.50 | Multiplier: 2×
* Stop Loss Distance: 2.50 × 2 = $5.00
* Position Size: $100 ÷ $5.00 = 20 Shares

## 2. Forex Trading Example (EUR/USD)

* Account: $10,000 | Risk: 1% ($100)
* ATR: 0.0025 (25 pips) | Multiplier: 2×
* Stop Loss Distance: 0.0025 × 2 = 0.0050 (50 pips)
* Position Size (Units): $100 ÷ 0.0050 = 20,000 Units (0.2 Micro Lots)

------------------------------
## ⚠️ Critical Guardrail: The Leverage Check
Always add a safety mechanism column to check your Notional Position Value against your actual account balance.

* Formula: = Position Size × Current Asset Price
* Why it matters: In low-volatility markets, the ATR will shrink, causing the formula to suggest a massive position size. If the required capital exceeds your account balance (or your allowed leverage limits), you must override the calculator and cap the trade size at your maximum allowable leverage.

------------------------------



