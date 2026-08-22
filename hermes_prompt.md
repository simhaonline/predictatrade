# PREDICT-A-TRADE XAUUSD

## FORENSIC INDICATOR, SCORING, PROBABILITY & SIGNAL MATHEMATICAL VERIFICATION

You are acting as a:

* Senior Quantitative Researcher
* Algorithmic Trading Auditor
* Applied Mathematician
* XAUUSD Trading Systems Engineer
* Statistical Model Validator
* Market Microstructure Specialist
* Technical Indicator Specialist
* Risk Mathematics Auditor
* AI/ML Model Validation Engineer
* Trading Signal Forensic Investigator

Your task is to perform an **independent microscopic mathematical audit of the complete Predict-A-Trade XAUUSD signal-generation engine**.

The central question is:

> **Are Predict-A-Trade BUY / SELL / NO-TRADE signals genuinely supported by the underlying market data, indicators, mathematics, scoring, probability, regime logic, risk calculations and strategy rules—or are any values incorrect, inconsistent, artificial, misleading, unreachable, stale, duplicated, fabricated or mathematically unjustified?**

This is a verification exercise.

Do NOT assume existing code is correct.

Do NOT trust displayed SCORE, PROB, Confidence, Entry, SL, TP or Regime.

Do NOT use the existing production function to verify itself.

Every important calculation must be independently reconstructed.

---

# 1. PRIMARY AUDIT OBJECTIVE

Prove independently whether the final signal:

```text
BUY
SELL
NO-TRADE
```

is correct for the exact market conditions that existed when the signal was created.

For each selected signal reconstruct:

```text
RAW MARKET DATA
↓
CANDLES
↓
INDICATORS
↓
MARKET STRUCTURE
↓
LIQUIDITY
↓
SESSION
↓
VOLATILITY
↓
REGIME
↓
STRATEGY EVIDENCE
↓
WEIGHTED SCORE
↓
PROBABILITY
↓
CONFIDENCE
↓
RISK GATES
↓
ENTRY
↓
STOP LOSS
↓
TP1
↓
TP2
↓
TP3
↓
RISK/REWARD
↓
POSITION SIZE
↓
FINAL BUY / SELL / NO-TRADE
```

Every stage must be independently verified.

---

# 2. STRICT AUDIT RULE — DO NOT MODIFY TRADING LOGIC

This is initially an audit.

Do NOT automatically:

* change indicator periods
* modify strategy thresholds
* lower signal thresholds
* increase signal frequency
* modify scoring weights
* change probability formulas
* weaken regime filters
* bypass NO-TRADE
* weaken risk rules
* change ATR multipliers
* alter Entry/SL/TP
* remove hard gates
* tune models to produce more signals

First identify the truth.

If defects are discovered:

1. document them,
2. prove them,
3. identify root cause,
4. identify correct mathematical behavior,
5. recommend repair,
6. explain expected effect.

Do not silently fix calculations before documenting the incorrect implementation.

---

# 3. CREATE AUDIT WORKSPACE

Create:

```text
docs/SIGNAL_MATH_FORENSIC_AUDIT/
```

Create:

```text
00_EXECUTIVE_SIGNAL_VERDICT.md
01_MARKET_DATA_INTEGRITY.md
02_CANDLE_MATHEMATICS.md
03_INDICATOR_FORMULA_AUDIT.md
04_MARKET_STRUCTURE_AUDIT.md
05_REGIME_MATHEMATICS.md
06_STRATEGY_EVIDENCE_MATRIX.md
07_SCORING_ENGINE_FORENSICS.md
08_PROBABILITY_CONFIDENCE_AUDIT.md
09_SIGNAL_GEOMETRY_AUDIT.md
10_RISK_REWARD_POSITION_SIZE_AUDIT.md
11_SIGNAL_REPRODUCTION_TESTS.md
12_LOOKAHEAD_DATA_LEAKAGE_AUDIT.md
13_EDGE_CASE_MATHEMATICS.md
14_SIGNAL_CORRECTNESS_REGISTER.md
15_DEFECT_AND_REMEDIATION_REGISTER.md
16_FINAL_SIGNAL_CERTIFICATION.md
```

Diagnostic scripts may be created under:

```text
tools/audit/signal_math/
```

They must remain isolated from production logic.

---

# 4. FIRST MAP THE COMPLETE SIGNAL PIPELINE

Locate exact production files/functions/classes responsible for:

* tick ingestion
* Bid/Ask
* spread
* candle creation
* timeframe aggregation
* indicator calculation
* market structure
* liquidity
* VWAP
* volume
* regime
* strategy selection
* evidence extraction
* score
* probability
* confidence
* hard gates
* candidate signal
* signal approval
* Entry
* SL
* TP
* RR
* lot sizing
* expiration
* signal persistence

Produce:

| Stage | File | Function/Class | Input | Output | Consumer |
| ----- | ---- | -------------- | ----- | ------ | -------- |

Find duplicate implementations.

Determine which implementation is actually invoked in production.

---

# 5. SOURCE-OF-TRUTH VERIFICATION

For every signal determine the true source of:

```text
ACTION
ENTRY
SL
TP1
TP2
TP3
SCORE
PROB
CONFIDENCE
REGIME
RISK %
RR
TIMEFRAME
SESSION
```

Determine whether these values originate from:

* actual quantitative calculation
* database
* strategy engine
* AI model
* LLM
* frontend transformation
* fallback
* default
* placeholder
* random/mock logic

Any production trading field generated artificially must be treated as a critical defect.

---

# 6. RAW XAUUSD DATA VALIDATION

Before validating indicators, verify the data they consume.

For selected samples validate:

```text
timestamp
bid
ask
mid
spread
open
high
low
close
volume
tick count
symbol
broker
digits
point size
tick size
```

Check:

* duplicates
* missing ticks
* missing candles
* out-of-order data
* stale quotes
* zero prices
* invalid highs/lows
* inconsistent timeframes

Fundamental OHLC rule:

```text
High >= max(Open, Close)
Low <= min(Open, Close)
High >= Low
```

If source data is wrong, downstream indicator verification is invalid.

---

# 7. CANDLE GENERATION MATHEMATICS

Independently reconstruct candles from raw data where possible.

Verify:

```text
Open  = first valid price
High  = maximum price
Low   = minimum price
Close = last valid price
```

Verify boundaries for:

* M1
* M5
* M15
* M30
* H1
* H4

and all other implemented timeframes.

Check exact timestamp boundaries.

No tick from the next candle may contaminate the previous candle.

---

# 8. CLOSED VS FORMING CANDLE

Determine whether strategies calculate signals from:

```text
closed candle
forming candle
tick-level state
```

Ensure implementation matches specification.

If a strategy expects closed candles but uses current incomplete candle values, classify this.

Check repainting risk.

---

# 9. INDEPENDENT INDICATOR IMPLEMENTATION

Do NOT verify production indicators by importing the same production function.

Create independent audit implementations using formulas directly.

Where practical compare against a reliable numerical library as an additional reference, but the audit formula must remain independently understandable.

For each indicator compare:

```text
Production Value
Independent Value
Absolute Difference
Relative Difference
Tolerance
PASS / FAIL
```

---

# 10. EMA AUDIT

Verify all implemented EMAs, including:

```text
EMA 9
EMA 21
EMA 50
EMA 100
EMA 200
```

Formula:

```text
α = 2 / (N + 1)

EMA_t =
Price_t × α
+
EMA_(t-1) × (1-α)
```

Investigate initialization:

* first price
* SMA seed
* library-specific initialization

Check whether initialization creates material divergence.

Verify price source:

```text
Close?
Typical Price?
Median?
```

Do not assume.

---

# 11. SMA AUDIT

Verify:

```text
SMA 50
SMA 100
SMA 200
```

Formula:

```text
SMA_N =
(sum of previous N selected prices) / N
```

Check exact N candles.

Detect off-by-one window errors.

---

# 12. RSI 14

Verify Wilder RSI mathematics.

Calculate:

```text
Gain
Loss
Average Gain
Average Loss

RS = Average Gain / Average Loss

RSI = 100 - 100/(1+RS)
```

Confirm whether Wilder smoothing or another method is used.

Test:

```text
RSI > 70
RSI < 30
RSI around 50
zero losses
zero gains
```

---

# 13. MACD 12/26/9

Verify:

```text
MACD Line = EMA12 - EMA26

Signal = EMA9(MACD)

Histogram = MACD - Signal
```

Check:

* crossover detection
* sign
* histogram direction
* previous/current comparison

Detect reversed bullish/bearish logic.

---

# 14. ADX 14

Independently reconstruct:

```text
True Range
+DM
-DM
ATR
+DI
-DI
DX
ADX
```

Verify Wilder smoothing.

Check interpretation of:

```text
ADX strength
+DI versus -DI
```

ADX itself is not directional.

Ensure code does not incorrectly interpret high ADX as automatically bullish.

---

# 15. ATR 14

Verify True Range:

```text
TR =
max(
High-Low,
abs(High-PreviousClose),
abs(Low-PreviousClose)
)
```

Verify ATR smoothing.

This is especially important because ATR may influence:

* SL
* TP
* volatility
* strategy thresholds
* trailing stop
* position sizing

Any ATR error can contaminate the entire signal.

---

# 16. BOLLINGER BANDS

Verify:

```text
Middle = SMA20

Upper = SMA20 + 2σ

Lower = SMA20 - 2σ
```

Determine:

* population or sample standard deviation
* period
* price source

Verify Bollinger Width formula.

---

# 17. STOCHASTIC

For configured Stochastic such as 14/3/3 verify:

```text
%K =
100 ×
(Close - LowestLow) /
(HighestHigh - LowestLow)
```

Then verify smoothing for %K and %D.

Test zero-range condition.

---

# 18. STOCHASTIC RSI

Verify actual implementation.

Do not confuse Stochastic price with Stochastic RSI.

Reconstruct:

```text
StochRSI =
(RSI - min RSI) /
(max RSI - min RSI)
```

and configured smoothing.

---

# 19. CCI 20

Verify:

```text
Typical Price =
(H + L + C) / 3
```

Then calculate mean deviation and CCI with correct constant.

---

# 20. VWAP

Verify:

```text
VWAP =
Σ(Price × Volume) /
ΣVolume
```

Determine:

* price definition
* reset period
* volume type
* session boundaries

A VWAP reset error can materially distort scalping signals.

---

# 21. OBV

Verify:

```text
if Close_t > Close_(t-1):
    OBV += Volume

if Close_t < Close_(t-1):
    OBV -= Volume
```

Check treatment of equal closes.

---

# 22. PARABOLIC SAR

Reconstruct SAR independently.

Verify:

* acceleration factor
* maximum acceleration
* reversal
* extreme point
* trend transition

---

# 23. ICHIMOKU

Verify configured values:

```text
Tenkan
Kijun
Senkou A
Senkou B
Chikou
```

Check displacement carefully.

Prevent accidental future-data leakage caused by incorrect shifting.

---

# 24. VOLUME PROFILE

Audit exact implementation rather than assuming standard behavior.

Determine:

* lookback
* binning
* price resolution
* POC
* value area
* high-volume nodes
* low-volume nodes

Check whether tick volume is being represented accurately.

---

# 25. CUMULATIVE DELTA

Determine whether genuine Bid/Ask/aggressor information exists.

If not, determine how delta is approximated.

Do NOT label an approximation as genuine order-flow delta without evidence.

---

# 26. MARKET STRUCTURE

Independently audit:

```text
Swing High
Swing Low
HH
HL
LH
LL
BOS
MSS
CHoCH
```

Define exact algorithms mathematically.

Avoid vague visual definitions.

For example define pivot confirmation precisely.

Check repainting/look-ahead implications of swing confirmation.

---

# 27. LIQUIDITY

Audit:

```text
BSL
SSL
equal highs
equal lows
liquidity sweep
stop hunt
```

Determine exact tolerances.

A sweep should have explicit mathematical conditions.

Do not accept arbitrary labels.

---

# 28. FAIR VALUE GAP

Verify exact bullish/bearish FVG rules.

Audit:

* three-candle relationship
* minimum gap
* ATR normalization if used
* mitigation
* expiry

---

# 29. FIBONACCI

Determine:

* swing anchors
* retracement levels
* extension levels
* directional orientation

Incorrect swing anchoring can invert Fibonacci evidence.

---

# 30. PIVOTS

Verify pivot equations used.

Do not assume classic pivots if code uses Fibonacci/Woodie/Camarilla.

Identify exact method.

---

# 31. SESSION CALCULATIONS

Verify session identification:

```text
Tokyo
London
New York
London/New York overlap
```

Check:

* UTC
* broker timezone
* DST
* frontend timezone

Session classification at signal timestamp must be correct.

---

# 32. INDICATOR WARM-UP

For every indicator determine minimum data requirement.

Example:

```text
EMA200
```

must not be treated as reliable from a handful of candles.

Determine:

```text
minimum bars
recommended warmup
actual implementation
```

Indicators without sufficient history must produce a safe state rather than artificial zeros.

---

# 33. ZERO VS UNKNOWN

Search for cases where:

```text
missing
NaN
None
undefined
not calculated
```

are converted to:

```text
0
```

This can create false strategy evidence.

The engine must distinguish:

```text
REAL ZERO
NOT AVAILABLE
NOT CALCULATED
INVALID
```

---

# 34. INDICATOR TIMESTAMP ALIGNMENT

Check that all indicators in a decision correspond to the same valid evaluation point.

Detect situations where:

```text
EMA = candle T
RSI = candle T-1
MACD = candle T
VWAP = another timeframe
```

unless explicitly intended.

---

# 35. MULTI-TIMEFRAME ALIGNMENT

Where strategies combine multiple timeframes, determine exactly which bars are visible at signal time.

Example:

At 10:17:

```text
M5 completed through 10:15
M15 completed through 10:15
H1 completed through 10:00
```

if using closed bars.

Do not accidentally use future H1 close.

---

# 36. LOOK-AHEAD BIAS

This section is mandatory.

Search for:

* `shift(-1)`
* negative shift
* future index
* next candle
* future high
* future low
* future close
* target reused as feature
* centered rolling window
* full-series normalization
* future swing confirmation incorrectly available

Any future information affecting a supposedly live historical decision must be classified as critical.

---

# 37. MARKET REGIME INDEPENDENT VERIFICATION

Reconstruct regime logic independently.

Determine input factors such as:

* ADX
* moving-average slope
* ATR
* Bollinger width
* structure
* volatility
* momentum
* mean deviation

For sampled points calculate regime manually/independently.

Compare:

```text
Expected Regime
Production Regime
```

---

# 38. REGIME BOUNDARY TESTS

Test directly below, at and above each threshold.

Example:

```text
ADX = 24.99
ADX = 25.00
ADX = 25.01
```

Determine whether:

```text
>
>=
<
<=
```

matches intended specification.

Boundary bugs commonly create unexplained NO-TRADE behavior.

---

# 39. STRATEGY-BY-STRATEGY AUDIT

Perform separate forensic verification for:

```text
STANDARD_SCALPING
ULTRA_SCALPING
STANDARD_SWING
TREND_SWING
```

Use actual production names if different.

For each reconstruct the exact equation and decision tree.

---

# 40. STRATEGY EVIDENCE MATRIX

Create for each strategy:

| Factor | Bullish Rule | Bearish Rule | Weight | Actual Value | Contribution |
| ------ | ------------ | ------------ | -----: | -----------: | -----------: |

Include every component.

Examples:

```text
EMA structure
RSI
MACD
ADX
VWAP
Bollinger
Stochastic
structure
liquidity
volume
session
regime
ATR
FVG
confluence
```

Use actual implementation.

---

# 41. BULLISH/BEARISH SYMMETRY

Check whether BUY and SELL logic are properly mirrored where they should be.

Detect mistakes such as:

```text
BUY:
EMA9 > EMA21

SELL accidentally also:
EMA9 > EMA21
```

instead of:

```text
EMA9 < EMA21
```

Look for inverted comparisons.

---

# 42. SCORE RECONSTRUCTION

This is one of the most important sections.

Extract exact formula.

Example conceptual structure:

```text
Score =
w1×EMA
+
w2×MACD
+
w3×RSI
+
w4×VWAP
+
w5×Structure
+
w6×Liquidity
+
...
```

Do NOT assume this equation.

Build the actual one from source.

---

# 43. SCORE COMPONENT TRACE

For every sampled signal create:

```text
Signal ID

EMA contribution:
MACD contribution:
RSI contribution:
ADX contribution:
VWAP contribution:
Structure contribution:
Liquidity contribution:
Volume contribution:
Session contribution:
Regime contribution:
Other contribution:

Raw total:
Normalization:
Penalty:
Bonus:
Final score:
```

The sum must reconcile exactly with stored/displayed score.

---

# 44. SCORE RANGE

Determine mathematically possible:

```text
minimum score
maximum score
```

Do not rely on documentation.

Prove it from weights and transformations.

Then determine whether configured thresholds are actually reachable.

---

# 45. UNREACHABLE THRESHOLD DETECTION

Example:

If maximum realistically achievable score is:

```text
23
```

but minimum signal threshold is:

```text
25
```

then strategy may be mathematically incapable of producing a signal.

Detect this systematically.

---

# 46. SCORE NORMALIZATION

If normalization exists, verify formula.

Examples might include:

```text
raw/max
sigmoid
min-max
z-score
softmax
```

Confirm denominator cannot become zero.

Check clipping.

---

# 47. PENALTIES & BONUSES

Audit:

* regime penalties
* spread penalty
* news penalty
* low-volume penalty
* volatility penalty
* confluence bonus
* session bonus

Ensure signs are correct.

A penalty must not accidentally increase score.

---

# 48. PROBABILITY FORENSIC AUDIT

Determine exactly how `PROB` is generated.

Classify as one of:

```text
CALIBRATED MODEL PROBABILITY
RAW MODEL PROBABILITY
LOGISTIC TRANSFORM
NORMALIZED SCORE
HEURISTIC
RULE-BASED
LLM-GENERATED
PLACEHOLDER
UNKNOWN
```

This classification is mandatory.

---

# 49. PROBABILITY EQUATION

If probability is mathematical, write exact equation.

Examples might involve:

```text
p = 1/(1+e^-z)
```

or another transform.

Do not assume.

Independently recompute sampled probabilities.

---

# 50. PROBABILITY CALIBRATION

If marketed/displayed as actual probability of success, test historical calibration.

For example:

Signals predicted around:

```text
50–60%
60–70%
70–80%
80–90%
90–100%
```

Calculate actual observed success frequency.

Produce:

| Predicted Bucket | Count | Mean Prediction | Actual Win Rate |
| ---------------- | ----: | --------------: | --------------: |

Use sufficient samples.

If sample size is insufficient, state:

```text
INSUFFICIENT DATA FOR CALIBRATION CLAIM
```

---

# 51. BRIER SCORE

Where binary outcome and sufficient observations exist calculate:

```text
Brier =
mean((forecast probability - actual outcome)^2)
```

Report honestly.

---

# 52. CONFIDENCE AUDIT

Determine whether confidence is:

* separate from probability
* duplicate of probability
* transformed score
* qualitative mapping
* ML confidence
* ensemble agreement
* heuristic

If PROB and Confidence are essentially the same number under different labels, report that.

---

# 53. LLM/AI NUMBER GENERATION

Search whether any LLM can generate:

```text
score
probability
confidence
entry
SL
TP
direction
```

without deterministic verification.

LLM-generated financial numbers must not be trusted simply because they look plausible.

If AI output contributes:

* capture raw AI output
* validate parser
* verify numerical constraints
* verify deterministic gates
* verify rejection of malformed data

---

# 54. HARD GATES VS SCORE

Reconstruct exact decision sequence.

Determine whether:

```text
hard gates
→ scoring
→ threshold
```

or:

```text
scoring
→ hard gates
```

or another sequence.

This matters for explaining NO-TRADE.

---

# 55. NO-TRADE FORENSICS

For every NO-TRADE sample identify exactly:

```text
FIRST FAILED CONDITION
OTHER FAILED CONDITIONS
SCORE
THRESHOLD
REGIME
MISSING DATA
HARD GATE
```

Create machine-readable reason codes.

Examples:

```text
REGIME_MISMATCH
INSUFFICIENT_SCORE
INSUFFICIENT_HISTORY
SPREAD_TOO_HIGH
VOLATILITY_BLOCK
NEWS_BLOCK
RISK_BLOCK
STALE_MARKET
MISSING_INDICATOR
```

---

# 56. EARLY RETURN AUDIT

Search strategies for early returns.

Example:

```python
if regime_not_allowed:
    return NO_TRADE
```

Determine what calculations never happen afterward.

This may explain:

```text
missing PROB
missing SCORE
Entry = 0
SL = 0
TP = 0
```

Distinguish valid lifecycle behavior from implementation defects.

---

# 57. CANDIDATE VS FINAL SIGNAL

Clearly distinguish:

```text
strategy candidate
scored candidate
approved candidate
published signal
executed trade
```

Do not mix these states.

A rejected candidate must not appear as a live trading signal.

---

# 58. DIRECTION VERIFICATION

For every BUY/SELL signal reconstruct all directional evidence.

Produce:

```text
Bullish evidence count
Bearish evidence count
Weighted bullish score
Weighted bearish score
Final direction
```

Check whether the winning direction genuinely had stronger evidence.

---

# 59. CONFLICTING INDICATORS

Determine how conflicts are handled.

Example:

```text
EMA bullish
MACD bullish
RSI overbought bearish/risk
Structure bearish
VWAP bullish
```

Verify deterministic weighting rather than arbitrary final direction.

---

# 60. ENTRY PRICE MATHEMATICS

Determine exact Entry source:

* Bid
* Ask
* Mid
* candle close
* limit level
* breakout level
* retest level
* VWAP
* structure level

For BUY/SELL verify correct Bid/Ask use considering actual execution mechanics.

---

# 61. SPREAD HANDLING

Explicitly verify spread.

If:

```text
Bid = 2500.00
Ask = 2500.30
```

a market BUY cannot realistically enter at:

```text
2500.00
```

unless pricing representation intentionally abstracts spread.

Document exact convention.

---

# 62. STOP LOSS MATHEMATICS

Determine how SL is generated:

* ATR
* structure
* swing level
* volatility
* fixed distance
* hybrid

For each sampled signal independently calculate SL.

Verify correct direction.

BUY:

```text
SL < Entry
```

SELL:

```text
SL > Entry
```

---

# 63. TAKE PROFIT MATHEMATICS

Independently calculate:

```text
TP1
TP2
TP3
```

Determine whether based on:

* RR
* ATR
* structure
* liquidity
* Fibonacci
* hybrid

Check monotonic ordering.

BUY:

```text
Entry < TP1 <= TP2 <= TP3
```

SELL:

```text
Entry > TP1 >= TP2 >= TP3
```

---

# 64. RISK-REWARD

Independently recompute.

BUY:

```text
Risk = Entry - SL
Reward = TP - Entry
RR = Reward / Risk
```

SELL:

```text
Risk = SL - Entry
Reward = Entry - TP
RR = Reward / Risk
```

Validate every displayed RR.

---

# 65. TRANSACTION-COST-ADJUSTED RR

Where appropriate compute realistic RR after:

* spread
* commission
* expected slippage

This is particularly important for Ultra Scalping.

A theoretical profitable setup may become economically invalid after transaction costs.

---

# 66. POSITION SIZING

Reconstruct exact position sizing formula.

Conceptually:

```text
RiskAmount =
AccountEquity × RiskPct

LotSize =
RiskAmount /
(SLDistance × ValuePerUnit)
```

Adapt to actual XAUUSD contract rules.

Verify:

* contract size
* tick size
* tick value
* broker digits
* account currency
* min lot
* max lot
* volume step

Never hardcode generic gold values if broker specifications differ.

---

# 67. ROUNDING

Audit all rounding:

```text
Entry
SL
TP
score
probability
lot size
RR
```

Internal calculations should not unnecessarily use display-rounded values.

---

# 68. FLOATING-POINT BOUNDARIES

Test important thresholds with:

```text
threshold - epsilon
threshold
threshold + epsilon
```

Ensure floating-point behavior doesn't create inconsistent results.

---

# 69. ATR = ZERO / MISSING DATA

Test:

* ATR zero
* missing volume
* zero standard deviation
* equal highest/lowest Stochastic range
* zero losses RSI
* zero gains RSI
* missing previous close

No divide-by-zero.

No false signal.

---

# 70. DUPLICATE CALCULATION SOURCES

Identify whether indicators are calculated separately in:

```text
Go
Python
frontend
database
```

If so compare formulas.

There must not be multiple conflicting definitions.

---

# 71. FRONTEND MUST NOT RECALCULATE TRADING TRUTH

Check whether frontend derives:

```text
PROB
SCORE
Entry
SL
TP
RR
```

independently.

Trading truth should normally originate from trusted backend calculations.

Frontend formatting must not change values.

---

# 72. DATABASE RECONCILIATION

For each sampled signal compare:

| Field       | Engine | DB |
| ----------- | -----: | -: |
| Score       |        |    |
| Probability |        |    |
| Confidence  |        |    |
| Entry       |        |    |
| SL          |        |    |
| TP1         |        |    |
| TP2         |        |    |
| TP3         |        |    |
| Regime      |        |    |

Investigate every discrepancy.

---

# 73. API RECONCILIATION

Then compare:

```text
Engine
=
Database
=
REST API
=
WebSocket
```

Allow only intentional formatting differences.

---

# 74. DASHBOARD RECONCILIATION

Finally verify:

```text
Admin Dashboard
User Dashboard
```

match authoritative backend data.

The UI must not display fabricated fallback values.

---

# 75. SELECT REPRESENTATIVE SIGNALS

Do not audit only convenient signals.

Select samples covering:

```text
BUY
SELL
NO-TRADE
```

from every strategy where available.

Target at least:

```text
10+ representative decisions per strategy
```

if sufficient historical data exists.

Include boundary cases.

---

# 76. SIGNAL FORENSIC REPORT

For each deeply audited signal produce:

```text
SIGNAL ID:
STRATEGY:
TIMESTAMP:
SYMBOL:
TIMEFRAME:

MARKET:
Bid:
Ask:
Spread:

INDICATORS:
EMA9:
EMA21:
EMA50:
EMA100:
EMA200:
RSI:
MACD:
ADX:
ATR:
VWAP:
etc.

STRUCTURE:
REGIME:
SESSION:

BULLISH EVIDENCE:
BEARISH EVIDENCE:

RAW SCORE:
PENALTIES:
BONUSES:
FINAL SCORE:
THRESHOLD:

PRODUCTION PROB:
INDEPENDENT PROB:

CONFIDENCE:

ENTRY:
SL:
TP1:
TP2:
TP3:
RR:

HARD GATES:

EXPECTED DECISION:
PRODUCTION DECISION:

FINAL RESULT:
CORRECT / INCORRECT / UNVERIFIED
```

---

# 77. EXACT REPRODUCIBILITY

Given:

```text
same market snapshot
same configuration
same strategy version
```

running signal calculation multiple times should produce the same result unless intentionally stochastic.

Test reproducibility.

If results vary, identify why.

---

# 78. CONFIGURATION SNAPSHOT

Every historical signal should ideally be traceable to the configuration that created it.

Determine whether signal can be linked to:

```text
strategy version
weight version
threshold version
model version
indicator settings
risk settings
```

Without this, historical reproduction may be impossible after configuration changes.

---

# 79. HISTORICAL REPLAY

Where infrastructure safely permits:

Replay historical data sequentially through an isolated copy/test harness.

No future data may be available during each decision.

Compare replay-generated decisions with stored signals.

---

# 80. BACKTEST VS LIVE ENGINE

Determine whether backtest and live systems use exactly the same core calculation functions.

If different implementations exist, compare them carefully.

This can create:

```text
backtest works
live doesn't
```

or vice versa.

---

# 81. SLIPPAGE FROM SIGNAL TIME

Measure where data exists:

```text
signal market price
actual execution price
difference
```

Determine whether signal quality claims account for execution reality.

---

# 82. SIGNAL EXPIRY

Verify mathematically appropriate expiry.

A signal created from a short-term market setup should not remain valid indefinitely.

Check whether expired geometry is still shown as actionable.

---

# 83. STRATEGY FREQUENCY SANITY

Calculate:

```text
candidate count
signal count
NO-TRADE count
```

for every strategy.

Report:

```text
Candidate Rate
Approval Rate
NO-TRADE Rate
BUY Rate
SELL Rate
```

Very abnormal values may indicate wiring or threshold problems.

---

# 84. DIRECTIONAL BALANCE

Check historical BUY/SELL distribution.

Example:

```text
99.9% BUY
0.1% SELL
```

may indicate:

* directional bug
* test-period bias
* regime bias
* legitimate market conditions

Investigate rather than assume.

---

# 85. SCORE DISTRIBUTION

Produce statistical distribution by strategy:

```text
minimum
maximum
mean
median
standard deviation
P10
P25
P50
P75
P90
P95
P99
```

Compare signal threshold with actual score distribution.

This helps determine whether thresholds are realistic.

---

# 86. PROBABILITY DISTRIBUTION

Perform the same analysis for probability.

Identify suspicious patterns such as:

```text
every signal = 75%
every signal = 80%
integer-only probabilities
never below 60%
```

These can indicate artificial mappings.

---

# 87. CONFIDENCE DISTRIBUTION

Check whether confidence varies meaningfully.

Compare correlation among:

```text
Score
Probability
Confidence
```

If correlation is exactly 1.0 because they are simple transformations, document this.

---

# 88. INDICATOR CORRELATION / DOUBLE COUNTING

Determine whether scoring double-counts highly related indicators.

Examples:

```text
EMA crossover
MACD
moving-average slope
```

can represent similar trend information.

Likewise:

```text
RSI
Stochastic
CCI
```

can overlap momentum information.

Document excessive double weighting.

Do not automatically change weights during audit.

---

# 89. INFORMATION GROUPING

Group evidence conceptually:

```text
Trend
Momentum
Volatility
Structure
Liquidity
Volume
Session
Macro
Risk
```

Calculate how much theoretical score each group contributes.

Detect hidden overweighting.

---

# 90. HARD-GATE DUPLICATION

Determine whether the same market characteristic is both:

```text
hard gate
+
large score penalty
```

creating accidental double punishment.

Likewise detect accidental double bonuses.

---

# 91. REGIME + STRATEGY COMPATIBILITY

Create matrix:

| Regime | Standard Scalping | Ultra Scalping | Standard Swing | Trend Swing |
| ------ | ----------------- | -------------- | -------------- | ----------- |

Use:

```text
ALLOWED
CONDITIONAL
BLOCKED
```

based on code.

Then verify every state is logically reachable.

---

# 92. STRATEGY REACHABILITY

Use mathematical/state analysis to prove each strategy can actually produce:

```text
BUY
SELL
```

under some valid inputs.

If impossible, report:

```text
UNREACHABLE STRATEGY PATH
```

with exact cause.

---

# 93. THRESHOLD SENSITIVITY — ANALYSIS ONLY

Do NOT tune production thresholds.

But calculate sensitivity around existing threshold.

Example:

```text
Threshold 20
Threshold 22
Threshold 25
Threshold 28
```

Show how signal rate would hypothetically change.

This is analysis only.

Do not use this to automatically increase signal frequency.

---

# 94. FALSE POSITIVE / FALSE NEGATIVE ANALYSIS

Where sufficient labeled historical outcomes exist evaluate:

```text
True Positive
False Positive
True Negative
False Negative
```

Use carefully defined signal success criteria.

Document criteria first.

---

# 95. SIGNAL OUTCOME DEFINITION

Determine exactly what a "winning signal" means.

Possible definitions:

```text
TP1 before SL
TP2 before SL
TP3 before SL
positive PnL
risk-adjusted positive result
```

Do not mix definitions when validating probability.

---

# 96. PATH-DEPENDENT OUTCOME

For historical evaluation use intrabar path correctly.

If both TP and SL occur inside same candle and lower-resolution data cannot determine which came first:

classify as:

```text
AMBIGUOUS
```

Do not conveniently assume TP first.

---

# 97. SPREAD-AWARE BACKTESTING

If historical backtest uses only OHLC midpoint data but live trading uses Bid/Ask:

identify difference.

Signal performance can be materially overstated otherwise.

---

# 98. LLM/AI SHOULD NOT OVERRIDE MATHEMATICS UNSAFELY

If an AI layer modifies a deterministic signal:

identify exact rule.

Example:

```text
Quant engine = NO-TRADE
LLM = BUY
Final = BUY
```

would be extremely dangerous unless strictly controlled.

Determine precedence.

---

# 99. MASTER SIGNAL EQUATION

At the end derive, in human-readable form, the real production decision algorithm for every strategy.

For example conceptually:

```text
IF
    market_data_fresh
AND regime_allowed
AND hard_gates_pass
AND evidence_score >= threshold
AND risk_valid
THEN
    determine BUY/SELL
    calculate geometry
    publish
ELSE
    NO-TRADE
```

But replace this with the actual implementation.

---

# 100. PROVE A BUY

Find a real BUY signal.

Reconstruct it independently.

Answer:

```text
Why BUY?
Why not SELL?
Why not NO-TRADE?
```

Every answer must cite exact mathematical evidence.

---

# 101. PROVE A SELL

Repeat for SELL.

Answer:

```text
Why SELL?
Why not BUY?
Why not NO-TRADE?
```

---

# 102. PROVE A NO-TRADE

Repeat for NO-TRADE.

Answer:

```text
Why NO-TRADE?
What exact condition prevented BUY?
What exact condition prevented SELL?
Was this intentional?
```

---

# 103. SIGNAL CORRECTNESS CATEGORIES

Every sampled decision must receive one:

```text
MATHEMATICALLY VERIFIED

MATHEMATICALLY VALID BUT CONFIGURATION-DEPENDENT

CORRECT NO-TRADE

INCORRECT DIRECTION

INCORRECT SCORE

INCORRECT PROBABILITY

INCORRECT REGIME

INCORRECT INDICATOR

INCORRECT GEOMETRY

INCORRECT RISK

INSUFFICIENT DATA

UNVERIFIED
```

---

# 104. P0 CONDITIONS

Treat as potential P0/NO-GO defects:

* future-data leakage
* wrong BUY/SELL direction
* incorrect SL direction
* incorrect position sizing
* incorrect risk %
* fabricated score
* fabricated probability
* fabricated indicators
* stale market data used as live
* strategy using missing values as zeros
* severe candle misalignment
* incorrect broker XAUUSD contract mathematics
* duplicate execution caused by duplicate signals

---

# 105. FINDING FORMAT

Every defect must contain:

```text
Finding ID:
Severity:
Strategy:
Signal ID:
Component:
Production Value:
Independent Value:
Difference:
Expected Formula:
Actual Formula:
File:
Function:
Root Cause:
Impact:
Does it alter direction? YES/NO
Does it alter score? YES/NO
Does it alter probability? YES/NO
Does it alter risk? YES/NO
Recommended Fix:
Regression Tests Required:
```

---

# 106. FINAL INDICATOR TABLE

Produce:

| Indicator        | Formula Verified | Independent Samples | Match | Signal Impact |
| ---------------- | ---------------- | ------------------: | ----- | ------------- |
| EMA9             |                  |                     |       |               |
| EMA21            |                  |                     |       |               |
| EMA50            |                  |                     |       |               |
| EMA100           |                  |                     |       |               |
| EMA200           |                  |                     |       |               |
| RSI14            |                  |                     |       |               |
| MACD             |                  |                     |       |               |
| ADX14            |                  |                     |       |               |
| ATR14            |                  |                     |       |               |
| Bollinger        |                  |                     |       |               |
| VWAP             |                  |                     |       |               |
| Stochastic       |                  |                     |       |               |
| Stoch RSI        |                  |                     |       |               |
| CCI              |                  |                     |       |               |
| OBV              |                  |                     |       |               |
| PSAR             |                  |                     |       |               |
| Ichimoku         |                  |                     |       |               |
| Volume Profile   |                  |                     |       |               |
| Cumulative Delta |                  |                     |       |               |

Include all implemented indicators.

---

# 107. FINAL STRATEGY TABLE

| Strategy          | BUY Verified | SELL Verified | NO-TRADE Verified | Score Correct | Probability Correct | Verdict |
| ----------------- | ------------ | ------------- | ----------------- | ------------- | ------------------- | ------- |
| Standard Scalping |              |               |                   |               |                     |         |
| Ultra Scalping    |              |               |                   |               |                     |         |
| Standard Swing    |              |               |                   |               |                     |         |
| Trend Swing       |              |               |                   |               |                     |         |

---

# 108. FINAL SIGNAL ACCURACY TABLE

For audited sample:

| Signal | Strategy | Production | Independent Decision | Score Match | PROB Match | Geometry Match | Verdict |
| ------ | -------- | ---------- | -------------------- | ----------- | ---------- | -------------- | ------- |

Do not write PASS without independent reproduction.

---

# 109. FINAL VERDICT

The final report must begin and end with:

```text
============================================================

PREDICT-A-TRADE XAUUSD
SIGNAL MATHEMATICAL CERTIFICATION

INDICATORS:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

MARKET STRUCTURE:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

REGIME:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

STRATEGY LOGIC:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

SCORING:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

PROBABILITY:
VERIFIED / HEURISTIC / FAILED / UNVERIFIED

CONFIDENCE:
VERIFIED / HEURISTIC / FAILED / UNVERIFIED

ENTRY/SL/TP:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

RISK/REWARD:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

POSITION SIZING:
VERIFIED / PARTIAL / FAILED / UNVERIFIED

LOOK-AHEAD BIAS:
NONE FOUND / FOUND / UNVERIFIED

LIVE DATA ALIGNMENT:
VERIFIED / FAILED / UNVERIFIED


FINAL SIGNAL ENGINE VERDICT:

GENUINELY MATHEMATICALLY VERIFIED
or
CONDITIONAL — DEFECTS REMAIN
or
NOT MATHEMATICALLY RELIABLE


ARE CURRENT BUY/SELL SIGNALS GENUINE?

YES / CONDITIONAL / NO


ARE SCORE VALUES GENUINE?

YES / PARTIAL / NO


IS PROB A GENUINE STATISTICAL PROBABILITY?

YES / NO — HEURISTIC ONLY / UNVERIFIED


CAN THE SIGNAL ENGINE BE TRUSTED FOR LIVE TRADING?

YES / CONDITIONAL / NO


P0 DEFECTS:
P1 DEFECTS:
P2 DEFECTS:
P3 DEFECTS:

============================================================
```

---

# 110. MOST IMPORTANT FINAL QUESTIONS

Answer each explicitly:

1. Are raw XAUUSD market inputs valid?
2. Are candles mathematically correct?
3. Are timestamps aligned?
4. Are all indicators independently correct?
5. Are any indicators using insufficient history?
6. Are missing indicators converted to zero?
7. Is there any look-ahead bias?
8. Is there any repainting risk?
9. Is multi-timeframe alignment correct?
10. Is market structure mathematically deterministic?
11. Are BOS/MSS/CHoCH genuine?
12. Are BSL/SSL/sweeps genuinely derived?
13. Is regime classification mathematically correct?
14. Are strategy-regime mappings correct?
15. Can each strategy genuinely generate BUY?
16. Can each strategy genuinely generate SELL?
17. Can each strategy correctly generate NO-TRADE?
18. Are any decision paths unreachable?
19. Are scoring weights genuine?
20. Does component sum equal final score?
21. Are thresholds mathematically reachable?
22. Are bonuses and penalties correctly signed?
23. Is SCORE independently reproducible?
24. What exactly does PROB mean?
25. Is PROB statistically calibrated?
26. Is Confidence materially different from PROB?
27. Is any LLM fabricating quantitative values?
28. Is final direction genuinely supported by weighted evidence?
29. Are Entry/SL/TP correct?
30. Is spread handled correctly?
31. Is ATR used correctly?
32. Is RR mathematically correct?
33. Is position sizing correct for broker XAUUSD specifications?
34. Are transaction costs considered adequately?
35. Are signal values preserved correctly in DB?
36. Does API reproduce DB values exactly?
37. Does WebSocket reproduce authoritative values?
38. Does Admin UI display genuine values?
39. Does User UI display genuine values?
40. Can the same historical signal be reproduced exactly?
41. Are historical signals linked to configuration/model versions?
42. Why is each selected NO-TRADE a NO-TRADE?
43. Why is each selected BUY genuinely BUY?
44. Why is each selected SELL genuinely SELL?
45. Is there any mathematical reason current signals should NOT be trusted?

---

# 111. ZERO-ASSUMPTION PRINCIPLE

Never conclude:

```text
function exists → correct
library used → correct
score shown → genuine
probability shown → calibrated
signal generated → valid
historical win → mathematically correct
```

Instead require:

```text
REAL MARKET INPUT
+
CORRECT TIMESTAMP
+
CORRECT INDICATORS
+
CORRECT STRUCTURE
+
CORRECT REGIME
+
CORRECT EVIDENCE
+
CORRECT WEIGHTS
+
CORRECT SCORE
+
VALID PROBABILITY
+
RISK GATES
+
CORRECT GEOMETRY
+
CORRECT POSITION SIZE
+
INDEPENDENT REPRODUCTION
=
GENUINELY VERIFIED SIGNAL
```

---

# 112. FINAL REQUIREMENT — CHALLENGE THE SIGNAL

For every sampled BUY or SELL, do not merely attempt to prove that production is correct.

Actively try to disprove it.

Ask:

```text
Could another indicator interpretation change the direction?

Was a candle still forming?

Was any evidence stale?

Was an indicator missing?

Was an unknown converted to zero?

Was the regime wrong?

Was a bearish factor accidentally scored bullish?

Was evidence counted twice?

Was the threshold applied incorrectly?

Was PROB derived after the fact?

Was any future candle visible?

Was the signal price executable considering spread?

Was SL/TP geometry realistic?

Would broker contract specifications change risk?

Was this calculation actually persisted?

Did the UI modify the result?
```

Only when the signal survives adversarial mathematical verification may it receive:

```text
MATHEMATICALLY VERIFIED
```

---

# 113. DO NOT REPAIR YET

Complete the forensic report first.

When finished:

1. present exact errors,
2. identify mathematical root causes,
3. rank P0–P3,
4. identify which defects can alter BUY/SELL direction,
5. identify which defects only affect display,
6. identify which defects affect risk,
7. identify which defects affect probability calibration,
8. propose exact corrective formulas,
9. propose regression tests,
10. STOP before modifying production strategy logic.

The next development phase will be separately authorized after the audit.

---

# 114. BEGIN

Start from the Predict-A-Trade XAUUSD repository root.

First:

1. read `AGENTS.md`,
2. locate actual signal pipeline,
3. locate indicator implementations,
4. locate strategy implementations,
5. locate scoring/probability calculations,
6. locate signal geometry,
7. locate persistence,
8. select representative historical signal records,
9. build independent audit calculators,
10. reproduce the calculations,
11. challenge every result,
12. issue the final mathematical certification.

Do not optimize for more trades.

Do not optimize for better-looking probabilities.

Do not optimize to make the platform receive PASS.

**Optimize only for mathematical truth, reproducibility, risk safety, and genuine signal correctness.**
