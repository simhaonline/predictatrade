# Master MT4/MT5 Multi-Timeframe Trading Playbook

**Comprehensive consolidated document** covering the full framework from beginning to end:

- indicator list and exact purpose
- indicator calculation methods
- how and why indicators conflict
- full conflict-priority logic
- candle reading and realtime execution
- MT4/MT5 exact Buy / Sell / SL / TP / R:R template
- bullish and bearish walkthroughs
- side-by-side comparison sheet
- candle-by-candle decision tree
- broker/account structure layer
- account-adjusted rules table for all 4 strategies

---

## Important Notes

- This is a **rule framework**, not a profit guarantee.
- Use **closed candles for confirmation**.
- Use **live candles for information**, not blind entry.
- Backtest and demo test before live use.
- For MT4/MT5, always remember:
  - **Buy opens at Ask, closes at Bid**
  - **Sell opens at Bid, closes at Ask**
- Turn on the **Ask line** and **spread display** on your platform.

---

# 1) System Philosophy

This system works only when each tool has a job.

## The three-layer model

### 1. Signal quality
Driven by:
- EMA 9
- EMA 21
- EMA 50
- EMA 100
- EMA 200
- SMA 50
- SMA 100
- EMA Cross 9/21
- SAR Direction
- StochRSI
- CCI 20
- candle structure

### 2. Execution quality
Driven by:
- spread
- commission
- slippage
- order type
- platform behavior
- session liquidity

### 3. Holding-cost suitability
Driven by:
- swap
- swap-free / Islamic structure
- administrative overnight costs
- position duration

## Core truth

> Indicators give **signal quality**.  
> Candles give **timing quality**.  
> Broker/account structure gives **execution quality**.

---

# 2) Indicator Set and Practical Jobs

| Indicator | Main Job | Do Not Use It As |
|---|---|---|
| EMA 200 | Major trend / market regime | Fast entry trigger |
| EMA 100 | Intermediate trend filter | Micro entry signal |
| EMA 50 | Active trend support/resistance | Standalone entry |
| EMA 21 | Pullback zone / short trend support | Long-term regime |
| EMA 9 | Immediate momentum | Trend filter |
| EMA Cross 9/21 | Trigger timing | Full trend bias |
| SMA 50 | Broader structure confirmation | Fast trigger |
| SMA 100 | Broader structure confirmation | Fast trigger |
| SAR | Continuation/trailing confirmation | Reliable signal in chop |
| StochRSI | Reset / overextension timing | Trend direction |
| CCI 20 | Momentum quality | Major bias alone |
| Candles | Final proof of participation | Standalone without context |

---

# 3) Indicator Calculation Methods

## 3.1 EMA Formula
For any period `n`:

`EMA_t = alpha * Close_t + (1 - alpha) * EMA_(t-1)`

where:

`alpha = 2 / (n + 1)`

### Smoothing values
- EMA 9 = `2 / 10 = 0.20`
- EMA 21 = `2 / 22 ≈ 0.0909`
- EMA 50 = `2 / 51 ≈ 0.0392`
- EMA 100 = `2 / 101 ≈ 0.0198`
- EMA 200 = `2 / 201 ≈ 0.0100`

### Meaning
- lower period = faster reaction
- higher period = slower reaction
- EMA weights recent price more heavily

---

## 3.2 SMA Formula
For any period `n`:

`SMA_t = (Close_t + Close_(t-1) + ... + Close_(t-n+1)) / n`

### Meaning
- simple average of last `n` closes
- smoother than EMA
- slower than EMA

---

## 3.3 EMA Cross 9/21
This is the relationship between EMA 9 and EMA 21.

### Bullish cross
`EMA9 crosses above EMA21`

### Bearish cross
`EMA9 crosses below EMA21`

Equivalent form:

`D_t = EMA9_t - EMA21_t`

- `D_t` from negative to positive = bullish cross
- `D_t` from positive to negative = bearish cross

### Meaning
- short-term momentum shift
- trigger, not regime

---

## 3.4 Parabolic SAR Formula
In an uptrend:

`SAR_next = SAR_current + AF * (EP - SAR_current)`

Where:
- `AF` = acceleration factor
- `EP` = extreme point (highest high in uptrend)

In a downtrend, EP becomes the lowest low.

### Common settings
- Step = `0.02`
- Max = `0.20`

### Meaning
- direction confirmation
- trailing stop logic
- noisy in sideways markets

---

## 3.5 RSI and StochRSI
### RSI
`RSI = 100 - 100 / (1 + RS)`

where:

`RS = Average Gain / Average Loss`

### StochRSI
`StochRSI = (RSI - Lowest(RSI,n)) / (Highest(RSI,n) - Lowest(RSI,n))`

### Common setting
- RSI length = 14
- Stoch length = 14
- smoothing = 3, 3
- key levels = 20 / 80 or 0.2 / 0.8

### Meaning
- very sensitive momentum reset tool
- useful for timing pullback completion
- not a standalone direction tool

---

## 3.6 CCI 20
### Typical Price
`TP = (High + Low + Close) / 3`

### CCI formula
`CCI = (TP - SMA(TP,20)) / (0.015 * MeanDeviation(TP,20))`

### Meaning
- measures distance from average relative to normal deviation
- above `+100` = strong positive momentum
- below `-100` = strong negative momentum
- above `0` = bullish side
- below `0` = bearish side

---

# 4) Why Indicators Conflict

Indicators conflict because they measure different things on different speeds.

## 4.1 Fast vs slow conflict
- EMA 9 / 21 react quickly
- EMA 50 / 100 / 200 react slowly
- SMA 50 / 100 are smoother and slower still

Example:
- EMA 9 crosses above EMA 21
- price still below EMA 200

Meaning:
- short-term bounce
- larger trend still bearish

---

## 4.2 Trend vs stretch conflict
Example:
- price above EMA 50 / 100 / 200 = bullish trend
- StochRSI overbought

Meaning:
- trend still up
- entry may be late
- not automatically a short

---

## 4.3 Duplicate information conflict
Examples:
- EMA 50 and SMA 50 overlap in purpose
- EMA 100 and SMA 100 overlap in purpose
- EMA cross and SAR both act like trigger tools

Use duplicates as context, not extra votes.

---

## 4.4 Multi-timeframe conflict
Example:
- 4H bullish
- 15m bearish
- 1m bullish reversal

Meaning:
- big trend up
- small pullback down
- micro entry back up

This is normal multi-timeframe behavior.

---

# 5) Anti-Conflict Hierarchy

## Full priority order
When indicators disagree, trust them in this order:

1. **Higher timeframe trend**
2. **EMA 200**
3. **EMA 100 / EMA 50 structure**
4. **Price location**
5. **Candle close quality**
6. **Broker/account execution fit**
7. **EMA 21**
8. **EMA 9/21 cross**
9. **SAR**
10. **CCI 20**
11. **StochRSI**
12. **SMA 50 / 100** as slow context only

## Master rule

> A lower-priority indicator must never overrule a higher-priority one.

Examples:
- StochRSI cannot overrule EMA 200.
- SAR cannot overrule healthy higher-timeframe structure.
- EMA 9/21 cross cannot overrule a flat MA cluster.

---

# 6) Full Conflict Priority Table

## 6.1 Regime conflicts

| Situation | Real Meaning | Priority Winner | Action |
|---|---|---|---|
| Above EMA 200, StochRSI overbought | Trend up, entry late | EMA 200 | Wait for pullback, do not short blindly |
| Below EMA 200, StochRSI oversold | Trend down, move stretched | EMA 200 | Wait for bounce, do not buy blindly |
| Above EMA 200, EMA 9 crosses below EMA 21 | Pullback inside uptrend | EMA 200 | Treat as pullback |
| Below EMA 200, EMA 9 crosses above EMA 21 | Bounce inside downtrend | EMA 200 | Treat as rally |
| Price chopping around flat EMA 200 | No clean regime | None | Stand aside |

## 6.2 Trend structure conflicts

| Situation | Real Meaning | Priority Winner | Action |
|---|---|---|---|
| EMA 50 bullish, EMA 100/200 still bearish | Early transition | EMA 100/200 | Aggressive scalp only, not strong swing bias |
| EMA 50 bearish, EMA 100/200 still bullish | Deeper pullback or transition | EMA 100/200 | Wait for structure break before strong short bias |
| EMA 50 and SMA 50 disagree slightly | Normal speed difference | EMA 50 for fast read | Use SMA as context only |
| EMA 100 and SMA 100 disagree slightly | Early vs slow confirmation | EMA 100 for fast read | Use SMA as context |
| EMA 50/100/200 flat or tangled | No trend | None | Skip trend setups |

## 6.3 Setup location conflicts

| Situation | Real Meaning | Priority Winner | Action |
|---|---|---|---|
| Bull trend, price far above EMA 21/50 | Extended | Location | Do not chase |
| Bear trend, price far below EMA 21/50 | Extended | Location | Do not chase |
| Above EMA 200 but below setup EMA 50 | Weak near-term structure | Location | Wait for reclaim or strong candle |
| Below EMA 200 but above setup EMA 50 | Bounce ongoing | Location | Wait for rejection |
| Price trapped between EMA 50/100/200 | Compression | Structure | Skip until clear breakout |

## 6.4 Trigger conflicts

| Situation | Real Meaning | Priority Winner | Action |
|---|---|---|---|
| EMA 9/21 bullish cross, weak close | Momentum not proven | Candle close | Wait |
| EMA 9/21 bearish cross, weak close | Momentum not proven | Candle close | Wait |
| SAR flips in flat MAs | Whipsaw | Trend structure | Ignore SAR |
| EMA cross and SAR agree after huge impulse candle | Late trigger | Location | Avoid chase |

## 6.5 Oscillator conflicts

| Situation | Real Meaning | Priority Winner | Action |
|---|---|---|---|
| CCI bullish, StochRSI bearish | Momentum and stretch disagree | Trend + candle | Wait for price confirmation |
| CCI bearish, StochRSI bullish | Bounce may be weak | Trend + candle | Wait for structure confirmation |
| Strong bullish trend, StochRSI overbought, CCI strong | Trend continuation | Trend + CCI | Do not short only for overbought |
| Strong bearish trend, StochRSI oversold, CCI weak | Trend continuation | Trend + CCI | Do not buy only for oversold |
| CCI > 0 but price below EMA 50/100/200 | Small bounce inside weak market | Structure | Aggressive scalp only |
| CCI < 0 but price above EMA 50/100/200 | Small pullback in healthy uptrend | Structure | Possible pullback only |

## 6.6 Multi-timeframe conflicts

| Situation | Real Meaning | Winner | Action |
|---|---|---|---|
| Higher TF bullish, lower TF bearish | Pullback | Higher TF | Look for long re-entry |
| Higher TF bearish, lower TF bullish | Bounce | Higher TF | Look for short re-entry |
| Higher TF sideways, lower TF trending | Local move only | Higher TF | Trade smaller or skip |
| Higher TF and lower TF aligned bullish | Continuation quality | Both | Best long setups |
| Higher TF and lower TF aligned bearish | Continuation quality | Both | Best short setups |

## 6.7 Candle conflicts

| Situation | Real Meaning | Winner | Action |
|---|---|---|---|
| Indicators bullish, candle closes weak with long upper wick | Buyers not convincing | Candle | Skip or wait |
| Indicators bearish, candle closes weak with long lower wick | Sellers not convincing | Candle | Skip or wait |
| Price rejects EMA zone before oscillators fully turn | Price leads oscillator | Candle + location | Accept only if structure is strong |
| Oscillators aligned, candle is doji/spinning top | No commitment | Candle | No trade |
| Signal appears directly into major support/resistance | Bad location | Location | Skip |

---

# 7) Candle Reading Framework

## 7.1 Live candle vs closed candle

### Use a live candle for
- reading momentum
- judging wick rejection
- seeing whether buyers or sellers are stepping in
- preparing for a setup

### Use a closed candle for
- actual trigger confirmation
- EMA reclaim confirmation
- breakout confirmation
- trade execution

## Golden rule

> A live candle can **suggest** a trade. A closed candle should **confirm** the trade.

---

## 7.2 Candle anatomy

### Body size
- large body = conviction
- small body = indecision or absorption

### Wicks
- long lower wick at EMA 21 / 50 or support = buyer defense
- long upper wick at EMA 21 / 50 or resistance = seller defense

### Closing position
- close near high = bullish strength
- close near low = bearish strength
- close in middle = weak confirmation

### Relation to moving averages
- reclaim above EMA 9/21 after pullback = momentum restart
- rejection below EMA 9/21 after bounce = bearish restart

---

## 7.3 Strong vs weak realtime candles

### Strong bullish live candle
- pushes up quickly from open
- holds above midpoint during development
- small upper wick
- reclaims or holds above EMA 9 / EMA 21
- attacks or closes above previous high

### Strong bearish live candle
- pushes down quickly from open
- holds below midpoint during development
- small lower wick
- stays below EMA 9 / EMA 21
- attacks or closes below previous low

### Weak candle signs
- long rejection wick against intended direction
- cannot hold above/below EMA 9/21
- closes in middle of range
- breakout fails quickly

---

## 7.4 Realtime reading steps

1. **Location**: is price at EMA 21 / 50 or major S/R?
2. **Body behavior**: is the candle expanding or fading?
3. **Wick behavior**: are wicks rejecting the zone?
4. **EMA behavior**: is price reclaiming or rejecting EMA 9/21?
5. **Structure behavior**: did price break prior high/low and hold?
6. **Close quality**: did it close strong, not just wick through?
7. **Follow-through**: did the next candle accept the move?

---

## 7.5 Realtime mistakes to avoid

- entering mid-candle because it looks strong
- confusing a wick with a breakout
- using StochRSI before price confirms
- taking the third or fourth expansion candle
- ignoring higher timeframe direction
- treating every EMA touch as automatic support or resistance

---

# 8) Broker / Account Structure Layer

The account type does **not** change EMA, SAR, StochRSI, or CCI calculations.  
It changes whether a valid signal is still worth taking after costs.

## 8.1 Core retail accounts referenced

### Classic Account
- no minimum deposit
- $0 commission
- standard spreads
- MT5 only
- best fit: beginners, swing more than scalping

### Standard Account
- low/minimal deposit requirement depending on entity
- $0 commission
- average spreads around 1.4 pips
- MT4 / MT5 / WebTrader
- best fit: beginner to intermediate, swing and selective scalping

### Premier Account
- minimum deposit around $3,000
- raw spreads from 0.0 pips
- commission about $3.5 per side per lot
- best fit: active traders, serious scalpers

### Micro Account
- micro lot sizing
- flexible leverage
- $0 commission
- best fit: low-risk live testing and psychology practice

## 8.2 Specialized variations

### Islamic (Swap-Free)
- no overnight interest / rollover / swap
- may involve alternative administrative charges
- useful mainly for swing and trend swing

### Demo
- virtual funds
- live-market simulation
- excellent for signal practice, not full fill realism

### Copy Trading
- follower or strategy provider structure
- outside this manual manual-candle system

## 8.3 Corporate / institutional
- corporate account: legal-entity market access
- institutional / prime: professional liquidity and HFT configurations
- outside normal retail manual rules

---

## 8.4 Execution-model categories

| Category | Practical Meaning | Pricing Model | Best For |
|---|---|---|---|
| ECN / Raw Spread | Direct market-style access | Raw spread + commission | Scalpers, active traders |
| STP | Straight-through routing style | Marked-up spread or spread-only | Simple-cost day traders |
| Standard / Dealing-Desk Style | Retail spread-focused structure | Wider / fixed / standard spread | Beginners, small capital |
| Micro / Cent | Smaller contract sizes | Usually spread-focused | Safe live practice |
| Islamic / Swap-Free | Overlay on another account | No swap, possible admin fee | Overnight-compliant trading |

## Important note
Do not trust marketing labels alone. Measure:
- current spread
- average spread during your trading session
- commission per round turn
- slippage behavior
- minimum stop distance
- overnight charges

---

## 8.5 Effective cost formulas

### Effective Trading Cost
`Effective Cost = Spread + Commission in pips + Expected Slippage`

### All-in cost for held trades
`All-in Cost = Effective Cost + Overnight Cost Equivalent`

### Commission pip equivalent
For many FX majors, if 1 standard lot pip value is about `$10`:
- round-turn commission `$7` ≈ `0.7 pip`

Formula:
`Commission in Pips = Round-turn Commission / Pip Value per Lot`

---

## 8.6 Strategy sensitivity to cost

| Strategy | Cost Sensitivity | Why |
|---|---|---|
| Ultra Scalping | Very high | Targets and stops are small |
| Standard Scalping | High | Cost still meaningfully reduces edge |
| Standard Swing | Medium | Targets larger, cost smaller fraction |
| Trend Swing | Lower on spread, higher on holding cost | Overnight cost matters more |

---

## 8.7 Account suitability by strategy

| Strategy | Classic | Standard | Premier | Micro | Islamic Overlay |
|---|---|---|---|---|---|
| Ultra Scalping | Poor | Poor to borderline | Best | Practice only | Mostly irrelevant |
| Standard Scalping | Borderline | Acceptable selectively | Best | Practice only | Usually not needed |
| Standard Swing | Good | Good | Excellent | Good for testing | Very useful if holding overnight |
| Trend Swing | Good | Good | Excellent | Good for testing | Very useful if holding overnight |

---

## 8.8 Account-based trade filters

### Max effective cost as % of 1R

| Strategy | Max Effective Cost as % of 1R |
|---|---:|
| Ultra Scalping | 20% |
| Standard Scalping | 15% |
| Standard Swing | 8% |
| Trend Swing | 5% |

### Trigger range vs spread filter
Do not take the trade if:

`Trigger Candle Range < 3 x Current Spread`

For very tight scalps, prefer `4 x spread`.

### Stop-loss vs spread filter
Do not take the trade if:

`SL Distance < 5 x Spread`

### Session filter
For scalping, prefer:
- London
- London / New York overlap

Avoid:
- rollover
- low-liquidity random hours
- spread spikes
- major news if your strategy is not news-based

### Overnight cost filter
For swing / trend swing:
- check swaps or admin charges
- swap-free is helpful, but not automatically cost-free

---

# 9) Universal MT4/MT5 Rules Template

## 9.1 Indicator settings
- EMA 9
- EMA 21
- EMA 50
- EMA 100
- EMA 200
- SMA 50
- SMA 100
- SAR: Step `0.02`, Max `0.20`
- StochRSI: `14, 14, 3, 3` with levels `20 / 80`
- CCI: `20`

---

## 9.2 Universal trigger candle definition

### Bullish trigger candle
All must be true:
- Close > Open
- body >= 50% of candle range
- Close > EMA 9
- Close > EMA 21
- Close > previous candle high

### Bearish trigger candle
All must be true:
- Close < Open
- body >= 50% of candle range
- Close < EMA 9
- Close < EMA 21
- Close < previous candle low

---

## 9.3 Confirmation score

### Buy score (+1 each)
- EMA 9 > EMA 21
- SAR bullish
- CCI > 0 and rising
- StochRSI K > D and rising

### Sell score (+1 each)
- EMA 9 < EMA 21
- SAR bearish
- CCI < 0 and falling
- StochRSI K < D and falling

### Minimum score required

| Strategy | Minimum Score |
|---|---:|
| Ultra Scalping | 3/4 |
| Standard Scalping | 3/4 |
| Standard Swing | 2/4 |
| Trend Swing | 2/4 |

---

## 9.4 Entry logic
- Buy Stop = trigger high + buffer
- Sell Stop = trigger low - buffer
- Buffer = `max(1 pip/tick, current spread)`

## 9.5 Risk definition
- Buy: `R = Entry - StopLoss`
- Sell: `R = StopLoss - Entry`

## 9.6 Hard no-trade filters
Do not take any trade if:
- EMA 50 / 100 / 200 are flat or tangled
- price is chopping around EMA 200
- EMA 9/21 flips repeatedly
- SAR flips repeatedly
- trigger candle is weak
- entry is directly into major support/resistance
- there is not enough room to TP2
- lower timeframe fights strong higher timeframe bias
- cost-to-risk filter fails

---

# 10) Strategy Timeframe Map

| Strategy | Bias TF | Setup TF | Entry TF |
|---|---|---|---|
| Ultra Scalping | 15m | 5m | 1m |
| Standard Scalping | 1H | 15m | 5m |
| Standard Swing | Daily | 4H | 1H |
| Trend Swing | Weekly | Daily | 4H |

---

# 11) Strategy 1: Ultra Scalping

## 11.1 Purpose
Very fast entries aligned with immediate trend.

## 11.2 Core market structure
### Buy bias on 15m
- close above EMA 200
- EMA 50 > EMA 100
- EMA 9 above EMA 21 or reclaiming strongly
- SAR mostly below price
- candles making higher highs / higher lows

### Sell bias on 15m
Reverse the above.

## 11.3 Setup on 5m
### Long setup
- pullback into EMA 21 or EMA 50
- pullback does not close below 5m EMA 100
- pullback candles smaller than prior impulse candles
- StochRSI low and turning up
- CCI recovering

### Short setup
- bounce into EMA 21 or EMA 50
- bounce does not close above 5m EMA 100
- bounce candles smaller than prior bearish impulse candles
- StochRSI high and turning down
- CCI rolling over

## 11.4 Entry on 1m
Enter only if trigger candle is valid and score >= 3/4.

### Buy
- Buy Stop above 1m trigger high
- SL below recent 1m swing low or last 3 closed 1m candles
- TP1 = 1R (close 50%, move to BE)
- TP2 = 1.5R (close 30%)
- TP3 = 2R or nearest 5m swing high (close 20%)

### Sell
- Sell Stop below 1m trigger low
- SL above recent 1m swing high or last 3 closed 1m candles
- TP1 = 1R (close 50%, move to BE)
- TP2 = 1.5R (close 30%)
- TP3 = 2R or nearest 5m swing low (close 20%)

## 11.5 Cancel rules
- pending order not triggered within 3 entry candles
- price closes back through EMA 21 before trigger
- setup timeframe breaks pullback/bounce invalidation

## 11.6 Realtime candle reading
### Good long sequence
- 15m trend up
- 5m red pullback candles get smaller
- lower wick appears at EMA 21 / 50
- 1m first strong green reclaim candle closes above EMA 9 / 21 and prior high

### Good short sequence
- 15m trend down
- 5m green bounce candles get smaller
- upper wick appears at EMA 21 / 50
- 1m first strong red rejection candle closes below EMA 9 / 21 and prior low

## 11.7 Account-adjusted rules table

| Account / Model | Suitability | Cost Rule | Session Rule | Execution Adjustment |
|---|---|---|---|---|
| Premier / Raw / ECN-like | Best | Effective cost <= 20% of 1R | London or overlap preferred | Full strategy allowed |
| Standard / STP-like | Borderline | Only trade if spread is stable and trigger range >= 4x spread | Only liquid sessions | Take only A-grade setups |
| Classic | Poor | Avoid unless spread is unusually tight and setup is exceptional | MT5 only, liquid sessions | Prefer not to ultra scalp |
| Micro | Practice only | Use for execution rehearsal, not edge optimization | Any practice session | Small size only |
| Islamic Overlay | Neutral | No major effect for same-day trades | N/A | Usually irrelevant for ultra scalp |

---

# 12) Strategy 2: Standard Scalping

## 12.1 Purpose
Cleaner, less noisy intraday continuation entries.

## 12.2 Bias on 1H
### Buy bias
- close above EMA 200
- EMA 50 > EMA 100
- EMA 100 above EMA 200 or clearly rising
- close above SMA 50 and SMA 100
- price not inside MA cluster

### Sell bias
Reverse the above.

## 12.3 Setup on 15m
### Long setup
- pullback to EMA 21 or EMA 50
- no strong close below 15m EMA 100
- CCI turns up from below 0 or below -100
- StochRSI turns up from low zone

### Short setup
- bounce to EMA 21 or EMA 50
- no strong close above 15m EMA 100
- CCI turns down from above 0 or above +100
- StochRSI turns down from high zone

## 12.4 Entry on 5m
Valid trigger candle + score >= 3/4.

### Buy
- Buy Stop above 5m trigger high
- SL below lowest low of last 3 closed 5m candles or 15m pullback low
- TP1 = 1R (close 40%, move to BE)
- TP2 = 2R (close 30%)
- TP3 = 3R or nearest 15m swing high (close 30%)

### Sell
- Sell Stop below 5m trigger low
- SL above highest high of last 3 closed 5m candles or 15m bounce high
- TP1 = 1R (close 40%, move to BE)
- TP2 = 2R (close 30%)
- TP3 = 3R or nearest 15m swing low (close 30%)

## 12.5 Cancel rules
- pending order not triggered within 4 entry candles
- 5m closes back through EMA 21 before trigger
- 15m invalidates setup structure

## 12.6 Realtime candle reading
### Long
- 1H trend up
- 15m pullback weakens into EMA 21/50
- 5m bullish engulfing / breakout candle closes above EMA 9 / 21 and prior high

### Short
- 1H trend down
- 15m bounce weakens into EMA 21/50
- 5m bearish engulfing / breakdown candle closes below EMA 9 / 21 and prior low

## 12.7 Account-adjusted rules table

| Account / Model | Suitability | Cost Rule | Session Rule | Execution Adjustment |
|---|---|---|---|---|
| Premier / Raw / ECN-like | Best | Effective cost <= 15% of 1R | London or overlap preferred | Full strategy allowed |
| Standard / STP-like | Acceptable selectively | Only clean setups with room to TP2 and spread not inflated | Liquid sessions only | Prefer 5m trigger with strong body close |
| Classic | Borderline | Better for selective scalp, not frequent tight targets | MT5 only, liquid sessions | Take only strongest trend continuations |
| Micro | Practice only | Good for training fills and management | Practice use | Small size, not performance evaluation |
| Islamic Overlay | Minor importance | Only matters if trade held overnight | N/A | Usually unnecessary |

---

# 13) Strategy 3: Standard Swing

## 13.1 Purpose
Hold for hours to days using structured pullbacks.

## 13.2 Bias on Daily
### Buy bias
- close above EMA 200
- EMA 50 > EMA 100
- EMA 100 above EMA 200 or rising
- close above SMA 50 and SMA 100
- higher highs / higher lows

### Sell bias
Reverse the above.

## 13.3 Setup on 4H
### Long setup
- pullback into EMA 21 or EMA 50
- deeper touch into EMA 100 allowed only if Daily remains bullish
- pullback candles lose force
- lower wick rejection or support hold visible
- CCI turns up from negative area
- StochRSI turns up from low zone

### Short setup
- bounce into EMA 21 or EMA 50
- deeper touch into EMA 100 allowed only if Daily remains bearish
- bounce candles lose force
- upper wick rejection visible
- CCI turns down from positive area
- StochRSI turns down from high zone

## 13.4 Entry on 1H
Valid trigger candle + score >= 2/4.

### Buy
- Buy Stop above 1H trigger high
- SL below 1H swing low or 4H pullback low
- TP1 = 1R (close 33%, move to BE)
- TP2 = 2R (close 33%)
- TP3 = 4R or nearest Daily swing high (close 34%)

### Sell
- Sell Stop below 1H trigger low
- SL above 1H swing high or 4H bounce high
- TP1 = 1R (close 33%, move to BE)
- TP2 = 2R (close 33%)
- TP3 = 4R or nearest Daily swing low (close 34%)

## 13.5 Cancel rules
- pending order not triggered within 6 entry candles
- 1H closes back through EMA 21 before trigger
- 4H invalidates setup structure strongly

## 13.6 Realtime candle reading
### Long
- Daily trend intact above EMA 200
- 4H red pullback candles shrink and form lower wicks near EMA 21 / 50
- 1H first strong green reclaim candle closes above EMA 9 / 21 and minor structure high

### Short
- Daily trend intact below EMA 200
- 4H green bounce candles shrink and form upper wicks near EMA 21 / 50
- 1H first strong red rejection candle closes below EMA 9 / 21 and minor structure low

## 13.7 Account-adjusted rules table

| Account / Model | Suitability | Cost Rule | Holding Rule | Execution Adjustment |
|---|---|---|---|---|
| Premier / Raw / ECN-like | Excellent | Effective cost <= 8% of 1R | Fine for active swing | Full strategy allowed |
| Standard / STP-like | Good | Spread acceptable because R is larger | Good for normal swing holding | Standard default choice |
| Classic | Good | Fine if spreads are normal and you accept MT5 only | Better for swing than scalp | Good beginner swing account |
| Micro | Good for testing | Useful for real-money process training | Small-size live rehearsal | Not ideal for scaling performance |
| Islamic Overlay | Very useful | Check admin fee equivalents | Best if positions held overnight | Preferred for compliance/overnight control |

---

# 14) Strategy 4: Trend Swing

## 14.1 Purpose
Capture larger continuation moves lasting days to weeks.

## 14.2 Bias on Weekly
### Buy bias
- close above EMA 200
- EMA 50 > EMA 100
- EMA 100 above EMA 200 or rising
- Weekly structure bullish
- close above SMA 50 / SMA 100

### Sell bias
Reverse the above.

## 14.3 Setup on Daily
### Long setup
- pullback into EMA 21, EMA 50, or occasionally EMA 100
- pullback loses force
- no major bearish structure break
- CCI resets then turns up
- StochRSI resets then turns up

### Short setup
- bounce into EMA 21, EMA 50, or occasionally EMA 100
- bounce loses force
- no major bullish structure break
- CCI resets then turns down
- StochRSI resets then turns down

## 14.4 Entry on 4H
Valid trigger candle + score >= 2/4.

### Buy
- Buy Stop above 4H trigger high
- SL below 4H swing low or Daily pullback low
- TP1 = 1R (close 25%, move to BE)
- TP2 = 3R (close 25%)
- TP3 = trail remaining 50% using Daily EMA 21 or Daily SAR

### Sell
- Sell Stop below 4H trigger low
- SL above 4H swing high or Daily bounce high
- TP1 = 1R (close 25%, move to BE)
- TP2 = 3R (close 25%)
- TP3 = trail remaining 50% using Daily EMA 21 or Daily SAR

## 14.5 Trail exit rules
### Buy exit
Exit remaining if:
- Daily closes below Daily EMA 21
- and Daily SAR flips bearish

### Sell exit
Exit remaining if:
- Daily closes above Daily EMA 21
- and Daily SAR flips bullish

## 14.6 Cancel rules
- pending order not triggered within 8 entry candles
- 4H loses EMA 21 before trigger
- Daily invalidates setup structure decisively

## 14.7 Realtime candle reading
### Long
- Weekly trend up
- Daily pullback drifts down with shrinking red bodies and lower wick defense
- 4H first strong bullish reclaim candle closes above EMA 9 / 21 and mini-structure high

### Short
- Weekly trend down
- Daily bounce drifts up with shrinking green bodies and upper wick rejection
- 4H first strong bearish rejection candle closes below EMA 9 / 21 and mini-structure low

## 14.8 Account-adjusted rules table

| Account / Model | Suitability | Cost Rule | Holding Rule | Execution Adjustment |
|---|---|---|---|---|
| Premier / Raw / ECN-like | Excellent | Effective cost <= 5% of 1R | Fine, but check overnight costs | Strong active option |
| Standard / STP-like | Good | Spread matters less than overnight cost | Good for normal trend swing | Good default if not scalping |
| Classic | Good | Suitable because targets are large | MT5 only | Acceptable for trend holds |
| Micro | Good for process training | Small size helps hold longer without stress | Practice only | Not ideal for scaling capital |
| Islamic Overlay | Very useful | Focus on all-in overnight cost | Preferred when trades last days/weeks | Strong fit for trend swing |

---

# 15) Bullish Walkthroughs

## 15.1 Ultra Scalping Long Walkthrough

### Story sequence
1. 15m closes strongly above EMA 200 and EMA 21/50.
2. 5m pulls back into EMA 21 with smaller red candles.
3. Lower wick appears and pullback stops expanding.
4. 1m shows a small failed red candle.
5. Next 1m candle reclaims EMA 9.
6. First strong 1m green candle closes above EMA 9, EMA 21, and previous high.
7. Buy Stop above trigger high.
8. Follow-through candle breaks high and holds.

### Key read
- trend up
- pullback weakens
- first real bullish reclaim candle appears

## 15.2 Standard Swing Long Walkthrough

### Story sequence
1. Daily remains above EMA 200 with bullish structure intact.
2. 4H pulls into EMA 21/50 with shrinking red candles.
3. Lower wick rejection appears on 4H.
4. 1H prints weak indecision first.
5. Then a strong bullish engulfing closes above EMA 9 / 21 and minor structure high.
6. Buy Stop above trigger high.
7. Next candle confirms.

### Key read
- higher timeframe trend intact
- setup timeframe pullback controlled
- entry timeframe first strong reclaim candle

---

# 16) Bearish Walkthroughs

## 16.1 Ultra Scalping Short Walkthrough

### Story sequence
1. 15m closes below EMA 200 with EMA 50 < EMA 100.
2. 5m bounces into EMA 21 with smaller green candles.
3. Upper wick appears and bounce loses energy.
4. 1m shows a small failed green attempt.
5. Next 1m candle turns indecisive near EMA 9/21.
6. First strong 1m red candle closes below EMA 9, EMA 21, and prior low.
7. Sell Stop below trigger low.
8. Follow-through candle breaks low and holds.

### Key read
- trend down
- bounce weakens
- first real bearish rejection candle appears

## 16.2 Standard Swing Short Walkthrough

### Story sequence
1. Daily remains below EMA 200 with bearish structure intact.
2. 4H bounces into EMA 21/50 with shrinking green candles.
3. Upper wick rejection appears on 4H.
4. 1H prints weak indecision first.
5. Then a strong bearish engulfing closes below EMA 9 / 21 and minor structure low.
6. Sell Stop below trigger low.
7. Next candle confirms.

### Key read
- higher timeframe trend intact down
- setup timeframe bounce controlled and weak
- entry timeframe first strong bearish reclaim candle

---

# 17) Annotated Fake Chart Tables

## 17.1 Ultra Scalping Long Example

| Candle | TF | Price Action | Indicator Read | Meaning | Action |
|---|---|---|---|---|---|
| A | 15m | Strong green candle closes near high | Above EMA 200, EMA 9 > 21, EMA 50 > 100 | Trend up | Long bias only |
| B | 15m | Small red pause candle | Holds above EMA 21 | Pullback only | Drop to 5m |
| 1 | 5m | Red pullback into EMA 21 | StochRSI resetting, CCI soft | Pullback starting | Wait |
| 2 | 5m | Smaller red candle with lower wick | Still above EMA 50 | Sellers weakening | Watch |
| 3 | 5m | Hammer/doji at EMA 21 | Closes back above EMA 21 | Setup support holding | Go to 1m |
| a | 1m | Small red test candle | Around EMA 9/21 | No long yet | Wait |
| b | 1m | Small green reclaim candle | Reclaims EMA 9 | Early strength only | Wait |
| c | 1m | Strong green trigger candle breaks prior high | Close > EMA 9/21, CCI up, StochRSI up | Real trigger | Prepare Buy Stop |
| d | 1m | Next candle breaks trigger high | Follow-through | Entry confirmed | Enter long |

## 17.2 Ultra Scalping Short Example

| Candle | TF | Price Action | Indicator Read | Meaning | Action |
|---|---|---|---|---|---|
| A | 15m | Strong red candle closes near low | Below EMA 200, EMA 9 < 21, EMA 50 < 100 | Trend down | Short bias only |
| B | 15m | Small green pause candle | Still below EMA 21 | Bounce only | Drop to 5m |
| 1 | 5m | Green bounce into EMA 21 | StochRSI rising, CCI recovering | Bounce forming | Wait |
| 2 | 5m | Smaller green candle, upper wick appears | Remains below EMA 50 | Buyers weakening | Watch |
| 3 | 5m | Doji / rejection candle | Closes back below EMA 21 | Setup resistance holding | Go to 1m |
| a | 1m | Small green test candle | Around EMA 9/21 | No short yet | Wait |
| b | 1m | Small indecision candle | Mixed around EMA 9/21 | No proof | Wait |
| c | 1m | Strong red trigger candle breaks prior low | Close < EMA 9/21, CCI down, StochRSI down | Real trigger | Prepare Sell Stop |
| d | 1m | Next candle breaks trigger low | Follow-through | Entry confirmed | Enter short |

## 17.3 Standard Swing Long Example

| Candle | TF | Price Action | Indicator Read | Meaning | Action |
|---|---|---|---|---|---|
| A | Daily | Trend candle bullish | Above EMA 200, EMA 50 > 100 | Structure bullish | Long bias |
| B | Daily | Small red pullback | Into EMA 21 | Corrective move | Watch 4H |
| 1 | 4H | Red pullback candle | Into EMA 21 | First retracement | Wait |
| 2 | 4H | Smaller red candle with lower wick | CCI flattening, StochRSI low | Sellers losing force | Stay patient |
| 3 | 4H | Probe lower, long lower wick | Rejection near EMA 50 | Support reacting | Setup improves |
| a | 1H | Small red test candle | Near EMA 21 | Sellers weak | Wait |
| b | 1H | Small green reclaim candle | Back above EMA 9 | Buyers testing control | Watch |
| c | 1H | Strong bullish engulfing | Close > EMA 9/21, CCI up | Proper trigger | Prepare Buy Stop |
| d | 1H | Breaks trigger high | Follow-through | Entry confirmed | Enter long |

## 17.4 Standard Swing Short Example

| Candle | TF | Price Action | Indicator Read | Meaning | Action |
|---|---|---|---|---|---|
| A | Daily | Trend candle bearish | Below EMA 200, EMA 50 < 100 | Structure bearish | Short bias |
| B | Daily | Small green bounce | Into EMA 21 | Corrective move | Watch 4H |
| 1 | 4H | Green bounce candle | Into EMA 21 | First retracement | Wait |
| 2 | 4H | Smaller green candle with upper wick | CCI flattening, StochRSI high | Buyers losing force | Stay patient |
| 3 | 4H | Probe higher, long upper wick | Rejection near EMA 50 | Resistance reacting | Setup improves |
| a | 1H | Small green test candle | Near EMA 21 | Buyers weak | Wait |
| b | 1H | Small red reclaim candle | Back below EMA 9 | Sellers testing control | Watch |
| c | 1H | Strong bearish engulfing | Close < EMA 9/21, CCI down | Proper trigger | Prepare Sell Stop |
| d | 1H | Breaks trigger low | Follow-through | Entry confirmed | Enter short |

---

# 18) Live Market Narration Examples

## 18.1 Ultra Scalping Long Narration
- Price is above EMA 200 on 15m.
- EMA 50 is above EMA 100.
- The 5m pullback is entering EMA 21.
- The second 5m red candle is smaller.
- Good, pullback is losing energy.
- Now a lower wick appears at support.
- On 1m, the first green candle is too weak. Wait.
- The next green candle breaks the prior high.
- It closes above EMA 9 and EMA 21.
- CCI turns up, StochRSI turns up.
- This is the first real bullish reclaim candle.
- Wait for close.
- Candle closes strong.
- Place Buy Stop above high.
- Next candle breaks the high and holds.
- Entry active.

## 18.2 Ultra Scalping Short Narration
- Price is below EMA 200 on 15m.
- EMA 50 is below EMA 100.
- The 5m bounce is entering EMA 21.
- The second 5m green candle is smaller.
- Good, bounce is weakening.
- An upper wick appears near EMA 21.
- On 1m, the first red candle is weak. Wait.
- The next red candle breaks the prior low.
- It closes below EMA 9 and EMA 21.
- CCI turns down, StochRSI turns down.
- This is the first real bearish rejection candle.
- Wait for close.
- Candle closes strong.
- Place Sell Stop below low.
- Next candle breaks the low and holds.
- Entry active.

## 18.3 Standard Swing Long Narration
- Daily is above EMA 200 and structure remains bullish.
- 4H pulls into EMA 21 / 50.
- Red bodies shrink and lower wicks appear.
- Good, pullback is weakening.
- On 1H, the first green attempt is not enough.
- Wait.
- Now a strong bullish engulfing closes above EMA 9 / 21 and prior high.
- CCI turns up, StochRSI turns up.
- Wait for close.
- Candle closes strong.
- Buy Stop above high.
- Next candle breaks and holds.
- Entry active.

## 18.4 Standard Swing Short Narration
- Daily is below EMA 200 and structure remains bearish.
- 4H bounces into EMA 21 / 50.
- Green bodies shrink and upper wicks appear.
- Good, bounce is weakening.
- On 1H, the first red attempt is not enough.
- Wait.
- Now a strong bearish engulfing closes below EMA 9 / 21 and prior low.
- CCI turns down, StochRSI turns down.
- Wait for close.
- Candle closes strong.
- Sell Stop below low.
- Next candle breaks and holds.
- Entry active.

---

# 19) Side-by-Side Bullish vs Bearish Comparison Sheet

## 19.1 Bias comparison

| Topic | Bullish Read | Bearish Read |
|---|---|---|
| EMA 200 | Price above EMA 200 | Price below EMA 200 |
| EMA 50 / 100 | EMA 50 above EMA 100 | EMA 50 below EMA 100 |
| EMA 9 / 21 | EMA 9 above EMA 21 | EMA 9 below EMA 21 |
| SAR | Mostly below price | Mostly above price |
| Structure | Higher highs / higher lows | Lower highs / lower lows |
| Preference | Longs only | Shorts only |

## 19.2 Setup comparison

| Topic | Bullish Setup | Bearish Setup |
|---|---|---|
| Pullback location | Into EMA 21 / EMA 50 support | Into EMA 21 / EMA 50 resistance |
| Opposite candles | Small red candles | Small green candles |
| Wick clue | Lower wicks appear | Upper wicks appear |
| Momentum behavior | Selling weakens | Buying weakens |
| CCI | Negative/flat then turns up | Positive/flat then turns down |
| StochRSI | Low/reset then turns up | High/reset then turns down |

## 19.3 Trigger comparison

| Topic | Bullish Trigger | Bearish Trigger |
|---|---|---|
| Body | Strong green | Strong red |
| Close position | Near high | Near low |
| EMA relation | Above EMA 9 and EMA 21 | Below EMA 9 and EMA 21 |
| Structure break | Breaks prior high | Breaks prior low |
| Wick quality | Small upper wick | Small lower wick |
| Meaning | Buyers regained control | Sellers regained control |

## 19.4 Follow-through comparison

| Topic | Bullish | Bearish |
|---|---|---|
| Next candle | Breaks trigger high and holds | Breaks trigger low and holds |
| Warning sign | Immediate dump back below EMA 21 | Immediate squeeze back above EMA 21 |
| Better entry | First strong reclaim candle | First strong rejection / breakdown candle |
| Worse entry | Third or fourth stretched green candle | Third or fourth stretched red candle |

---

# 20) Printable Candle-by-Candle Decision Tree

## Step 1: Bias check
### If price above EMA 200 and EMA 50 > EMA 100
- Bias = Bullish
- Continue to bullish setup check

### If price below EMA 200 and EMA 50 < EMA 100
- Bias = Bearish
- Continue to bearish setup check

### If price around EMA 200 or MAs flat/tangled
- Cancel / no trade

---

## Step 2A: Bullish setup check
- Is price pulling into EMA 21 or EMA 50?
- Are pullback candles small and weakening?
- Are lower wicks appearing?
- Did a large bearish candle break structure strongly?
  - Yes = cancel / wait for new setup
- Is price far above EMA 21 / 50 and already extended?
  - Yes = wait, do not chase

## Step 2B: Bearish setup check
- Is price bouncing into EMA 21 or EMA 50?
- Are bounce candles small and weakening?
- Are upper wicks appearing?
- Did a large bullish candle break structure strongly?
  - Yes = cancel / wait for new setup
- Is price far below EMA 21 / 50 and already extended?
  - Yes = wait, do not chase

---

## Step 3A: Bullish trigger check
### If live candle
- reclaims EMA 9 / EMA 21
- breaks previous candle high
- holds above midpoint
- keeps a strong green body

Then:
- wait for close

### If closed candle
- above EMA 9
- above EMA 21
- above previous high
- near its high

Then:
- enter / place Buy Stop

### If candle
- leaves long upper wick
- closes back under EMA 21
- fails to break structure

Then:
- cancel / wait for next candle

## Step 3B: Bearish trigger check
### If live candle
- rejects EMA 9 / EMA 21
- breaks previous candle low
- holds below midpoint
- keeps a strong red body

Then:
- wait for close

### If closed candle
- below EMA 9
- below EMA 21
- below previous low
- near its low

Then:
- enter / place Sell Stop

### If candle
- leaves long lower wick
- closes back above EMA 21
- fails to break structure

Then:
- cancel / wait for next candle

---

## Step 4: Confirmation score check
### Bullish
+1 each for:
- EMA 9 > EMA 21
- SAR bullish
- CCI > 0 and rising
- StochRSI turning up

### Bearish
+1 each for:
- EMA 9 < EMA 21
- SAR bearish
- CCI < 0 and falling
- StochRSI turning down

### If score below required minimum
- wait / cancel

### If score meets minimum
- continue

---

## Step 5: Account / cost check
- Does effective cost pass the strategy threshold?
- Is trigger range at least 3x spread? Prefer 4x for tight scalps.
- Is SL at least 5x spread?
- Is spread stable for this session?
- For held trades, is overnight cost acceptable?

If any answer is no:
- cancel

---

## Step 6: Room-to-target check
- Is there enough room to TP2 before major support/resistance?
  - No = cancel
  - Yes = continue

---

## Step 7: Entry follow-through
- Place stop order beyond trigger candle
- If next candle triggers and holds direction = trade active
- If next candle violently rejects through EMA 21 = warning
- If order is not triggered within allowed bar count = cancel pending order

---

# 21) One-Look Manual Checklist

## BUY Checklist
- [ ] Higher timeframe close above EMA 200
- [ ] EMA 50 > EMA 100
- [ ] EMA 50 slope positive
- [ ] No MA cluster/chop
- [ ] Setup timeframe pulled back to EMA 21 or EMA 50
- [ ] No strong bearish structure break
- [ ] Entry timeframe bullish trigger candle valid
- [ ] Confirmation score >= minimum
- [ ] Cost filter passes
- [ ] Enough room to TP2
- [ ] Spread acceptable
- [ ] Place Buy Stop above trigger high
- [ ] Set SL below structure low
- [ ] Set TP1 / TP2 / TP3 by strategy
- [ ] Move SL to BE after TP1
- [ ] Cancel if not triggered in allowed bars

## SELL Checklist
- [ ] Higher timeframe close below EMA 200
- [ ] EMA 50 < EMA 100
- [ ] EMA 50 slope negative
- [ ] No MA cluster/chop
- [ ] Setup timeframe bounced to EMA 21 or EMA 50
- [ ] No strong bullish structure break
- [ ] Entry timeframe bearish trigger candle valid
- [ ] Confirmation score >= minimum
- [ ] Cost filter passes
- [ ] Enough room to TP2
- [ ] Spread acceptable
- [ ] Place Sell Stop below trigger low
- [ ] Set SL above structure high
- [ ] Set TP1 / TP2 / TP3 by strategy
- [ ] Move SL to BE after TP1
- [ ] Cancel if not triggered in allowed bars

---

# 22) Account-Adjusted Rules Table for All 4 Strategies

## 22.1 Summary matrix

| Strategy | Best Account Type | Acceptable Account Type | Practice Account | Swap-Free Need |
|---|---|---|---|---|
| Ultra Scalping | Premier / Raw / ECN-like | Standard only selectively | Micro / Demo | Low |
| Standard Scalping | Premier / Raw / ECN-like | Standard / Classic selectively | Micro / Demo | Low to minor |
| Standard Swing | Standard / Premier | Classic / STP-like | Micro / Demo | Medium to high if overnight |
| Trend Swing | Standard / Premier / Islamic | Classic / STP-like | Micro / Demo | High if held for days/weeks |

## 22.2 Detailed strategy account table

| Strategy | Account Type | Suitable? | Spread / Cost Tolerance | Preferred Session | Overnight Rule | Notes |
|---|---|---:|---|---|---|---|
| Ultra Scalping | Premier / Raw / ECN-like | Yes | Effective cost <= 20% of 1R | London / Overlap | Usually irrelevant | Best fit for tight targets |
| Ultra Scalping | Standard / STP-like | Limited | Only if spread stable and trigger range >= 4x spread | London / Overlap only | Usually irrelevant | Take A-grade setups only |
| Ultra Scalping | Classic | Weak | Avoid unless spreads are unusually favorable | Liquid session only | Irrelevant | MT5 only; generally not ideal |
| Ultra Scalping | Micro | Practice | Use for process, not performance | Any practice session | Irrelevant | Good for psychology and execution drills |
| Standard Scalping | Premier / Raw / ECN-like | Yes | Effective cost <= 15% of 1R | London / Overlap | Low importance | Best choice |
| Standard Scalping | Standard / STP-like | Yes, selectively | Strong trigger close, enough room to TP2 | Liquid sessions | Low importance | Avoid news spikes |
| Standard Scalping | Classic | Borderline | Use only high-quality continuation setups | Liquid sessions | Low importance | Better if not ultra-tight targets |
| Standard Scalping | Micro | Practice | Good for testing management | Practice use | Low importance | Not for serious performance judgment |
| Standard Swing | Premier / Raw / ECN-like | Yes | Effective cost <= 8% of 1R | Any healthy session | Important if multi-day | Excellent all-around |
| Standard Swing | Standard / STP-like | Yes | Spread usually acceptable due to larger R | Any healthy session | Check swap/admin | Good default retail choice |
| Standard Swing | Classic | Yes | Fine if spreads normal | Any healthy session | Check swap/admin | Beginner-friendly swing use |
| Standard Swing | Micro | Yes for testing | Small size, live process training | Any session | Check holding charges | Good learning bridge |
| Standard Swing | Islamic Overlay | Strong fit | Focus on all-in overnight cost | Any | Preferred for overnight compliance | Verify admin fee structure |
| Trend Swing | Premier / Raw / ECN-like | Yes | Effective cost <= 5% of 1R | Any healthy session | Overnight cost matters | Excellent if active and funded |
| Trend Swing | Standard / STP-like | Yes | Spread less important than hold cost | Any | Check swap/admin carefully | Good practical retail choice |
| Trend Swing | Classic | Yes | Acceptable because targets large | Any | Check swap/admin | MT5 only |
| Trend Swing | Micro | Testing only | Good for long-hold discipline practice | Any | Check holding charges | Not for serious scaling |
| Trend Swing | Islamic Overlay | Strong fit | Focus on overnight efficiency | Any | Preferred for multi-day holds | Important for Sharia compliance |

---

# 23) Final Core Principles

## Principle 1
**The moving averages choose direction.**

## Principle 2
**Price location decides whether the setup is attractive or late.**

## Principle 3
**Realtime candles show intent.**

## Principle 4
**Closed candles confirm entries.**

## Principle 5
**Oscillators improve timing; they do not lead the whole decision.**

## Principle 6
**Broker/account structure decides whether a valid signal is still worth trading after cost.**

## Final summary sentence

> **Bias gives permission. Setup gives location. Realtime candles show intent. Closed candles confirm entry. Indicators support the decision. Account structure decides whether the trade is worth taking after costs.**
