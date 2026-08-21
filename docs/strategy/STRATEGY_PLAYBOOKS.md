# Predict-A-Trade — Strategy Playbooks

**Version:** v1.8.0 — Trade Management Audit + Broker Stop Validation + Cost-Aware Break-Even  
**Date:** 19 August 2026

---

## Four Strategy Products

Each strategy is an independent engine with distinct configuration. PTB enriches but does NOT replace strategy logic. All four evaluate independently every eligible cycle.

### STANDARD_SCALPING
- **Timeframes:** M1/M5 decision, M15/M30 context
- **Threshold:** 65 (MinConfluence)
- **Min RR:** 1.2
- **Cooldown:** 15 minutes
- **ATR SL Multiplier:** 1.0
- **Conflict Threshold:** > 20 → WAIT
- **Min ADX:** 20
- **Key Indicators:** EMA9/21, VWAP, BOS, candle displacement/rejection, MACD/OsMA, RSI, ADX, liquidity sweeps, MTF

### ULTRA_SCALPING
- **Timeframes:** M1 decision, M5 context
- **Threshold:** 85 (highest)
- **Min RR:** 1.0
- **Cooldown:** 15 minutes
- **ATR SL Multiplier:** 1.0
- **Conflict Threshold:** > 15 (strictest)
- **Min ADX:** 22
- **Key Indicators:** M1 displacement, VWAP, liquidity sweep+reclaim, OsMa, Stochastic, MTF, ADX, Bollinger

### STANDARD_SWING
- **Timeframes:** M15/M30/H1 decision, H4/D1 context
- **Threshold:** 55
- **Min RR:** 1.8
- **Cooldown:** 120 minutes
- **ATR SL Multiplier:** 1.5
- **Conflict Threshold:** > 25
- **Min ADX:** 18
- **Key Indicators:** EMA21/50, SMA200, HTF BOS/CHoCH, OB, FVG, ADX, MACD, RSI, breakout, MTF, liquidity, regime

### TREND_SWING
- **Timeframes:** H1/H4 decision, D1/W1 context
- **Threshold:** 50
- **Min RR:** 2.5
- **Cooldown:** 360 minutes
- **ATR SL Multiplier:** 2.0
- **Conflict Threshold:** > 30 (most tolerant)
- **Min ADX:** 22 (mandatory)
- **Key Indicators:** SMA200, EMA50, EMA21, ADX/+DI/-DI, HH/HL or LH/LL, BOS, OB, FVG, MACD, CCI, pullback, MTF, VWAP

---

## Candidate Threshold System (Advisory Signals)

Each strategy has two thresholds: a **candidate threshold** and a **trade threshold**. This separates directional opportunity detection from execution qualification.

| Strategy | Candidate Threshold | Trade Threshold | Max Theoretical Score |
|----------|--------------------:|----------------:|---------------------:|
| STANDARD_SCALPING | 40 | 65 | ~80 |
| ULTRA_SCALPING | 40 | 65 | ~78 |
| STANDARD_SWING | 35 | 55 | ~92 |
| TREND_SWING | 30 | 50 | ~75 |

### Score → Direction Mapping

| Score Range | Result | Direction | Signal Class | Executable? |
|------------|--------|-----------|--------------|-------------|
| score < candidate_threshold | NO-TRADE | — | — | No |
| candidate_threshold ≤ score < trade_threshold | Advisory | BUY_CANDIDATE / SELL_CANDIDATE | ADVISORY | No |
| score ≥ trade_threshold + all gates pass | Qualified | BUY / SELL | EXECUTABLE | Yes (if licensed) |
| score ≥ trade_threshold + gate veto | Blocked | BUY/SELL (preserved) + BLOCKED grade | ADVISORY | No |

### Why Candidates Are Not Executable

BUY_CANDIDATE and SELL_CANDIDATE are **advisory signals** that show a directional setup is forming but has not reached sufficient conviction for execution. This is a deliberate safety design:

1. **No gate evaluation**: Hard gates are not run on candidates (they would fail on insufficient score).
2. **Research grade**: Candidates receive `Grade: RESEARCH` — they are informational, not actionable.
3. **No cooldown trigger**: Candidates do not trigger strategy cooldowns.
4. **Full geometry computed**: Entry, SL, TP1/TP2/TP3 are computed for the candidate so the user can see the potential trade setup.

### Regime-Aware Thresholds

Thresholds are adjusted based on market regime. In RANGE regime, thresholds may be lower because the evidence budget is smaller (fewer trend-confirmation indicators fire). See `strategy/regime_thresholds.go` for the full regime-to-threshold mapping.

### TP/SL Geometry (v1.4.0)

Entry, Stop Loss, and Take Profit levels are computed using ATR-based multipliers:

| Strategy | SL (×ATR) | TP1 (×ATR) | TP2 (×ATR) | TP3 (×ATR) | MinRR |
|----------|----------|-----------|-----------|-----------|-------|
| STANDARD_SCALPING | 1.0 | 1.0 | 1.5 | 2.0 | 1.2 |
| ULTRA_SCALPING | 0.5 | 0.5 | 0.75 | 1.0 | 1.0 |
| STANDARD_SWING | 1.5 | 1.5 | 2.5 | 3.5 | 1.8 |
| TREND_SWING | 2.0 | 2.0 | 4.0 | 6.0 | 2.5 |

**v1.4.0 Fix**: TP levels are now ATR-based (same basis as SL), not MinRR-inflated. This prevents the issue where TP1 was 2.5x further than SL, causing trades to hit SL before reaching TP1. The MinRR gate validates R:R and rejects insufficient signals — TP is not artificially inflated.

### MQL EA Strategy Selection (v1.05)

Both MT4 and MT5 EAs include input parameters to select which strategies and directions to receive:
- 4 strategy toggles (all enabled by default)
- 4 direction filters (BUY, SELL, BUY_CANDIDATE, SELL_CANDIDATE — all enabled by default)
- Signal counters on chart panel (received, displayed, filtered)

### Signal Delivery to MT4/MT5

Signals are delivered via: Go Engine → WebSocket → Windows Agent → PAT_signals.txt → MT4/MT5 EA. The Go engine broadcasts directional signals to both the frontend dashboard and the Windows Agent simultaneously.

### Calibration Probability

Until a calibration model is VALIDATED or PROMOTED, the calibrated probability (PROB) is NULL/zero. The UI shows "Pending" in the PROB column. Raw score is always available in the Score column. See `docs/SIGNAL_TYPES_AND_PROBABILITY.md` for details.

- **Mandatory Gates:** Trending/Breakout regime, EMA100/EMA200 macro trend, ADX > 22

---

## PTB Integration

PTB provides context intelligence to all four strategies through the shared `MarketState.PTB` field. In SHADOW mode:
- PTB calculates and persists analysis
- PTB does NOT alter strategy scores, directions, entries, SLs, TPs, or risk gates
- Strategies can read PTB context but it has zero score impact

When PTB is activated (after validation):
- Position size multiplier is advisory (risk gates remain authoritative)
- Stop distance multiplier is advisory (existing SL logic remains canonical)
- PTB cannot bypass any hard gate

### PTB Synthesis Outputs

| Output | Values |
|--------|--------|
| Bias | STRONG_LONG, LONG, NEUTRAL, SHORT, STRONG_SHORT, STAND_ASIDE |
| Action | ENTER, WAIT, AVOID, EXIT |
| Setup Quality | A+, A, B, C, D, F |
| Position Size Multiplier | A+→1.0, A→0.8, B→0.6, C→0.4, D→0.2, F→0.0 |
| Stop Distance Multiplier | High manipulation→1.5, Normal→1.0, Low vol→0.8 |

---

## Advanced Risk Integration (v1.1.0)

The advanced risk layer operates above the strategy engines and below the hard risk gates:

```
Strategy Engine → Advanced Risk Layer → Hard Risk Gates → Signal
```

### Loss Recovery Manager

After consecutive losses or daily loss limit, the recovery manager transitions to RECOVERY mode:
- **NORMAL:** Full risk, normal thresholds
- **RECOVERY:** Reduced risk (0.5x), higher confluence/grade/confidence required
- **HALTED:** No trading until halt expires
- **DAILY_LIMIT:** No trading until new trading day

State is isolated per account+strategy+symbol. Restart-safe via state persistence.

### Adaptation Manager

Dynamically adjusts strategy parameters based on market phase:
- Can only make the system more conservative (never increases risk above hard limits)
- Operates on a deep copy of weights — never mutates base config
- Risk multiplier clamped to global hard max

| Phase | Risk Multiplier | Stop Multiplier | Confluence Bonus |
|-------|-----------------|-----------------|------------------|
| TRENDING | 1.0 | 1.0 | 0 |
| RANGING | 0.7 | 1.0 | +5 |
| HIGH_VOLATILITY | 0.5 | 1.5 | +10 |
| LOW_VOLATILITY | 0.8 | 0.8 | 0 |
| MANIPULATIVE | 0.3 | 1.5 | +15 |
| UNCERTAIN | 0.6 | 1.0 | +5 |

### ML Adaptation (Research Only)

When enabled with a trained model, ML provides adaptive parameter adjustments with comprehensive fallback chain. All outputs are clamped to safe bounds.

### RL Strategy Optimizer (Research Only)

RL can operate in 4 modes:
- **disabled:** No RL influence
- **shadow:** RL evaluates but does not affect signals
- **filter_only:** RL can only veto (NO_TRADE) — cannot create trades
- **live_approved:** RL can influence signals (requires explicit authorization + validation)

### Sentiment Engine

Async background sentiment analysis — never blocks the signal hot path. Falls back to neutral when unavailable or stale.

---

## Backtesting (v1.2.0)

The backtesting framework reproduces the production strategy/PTB/risk gate logic through faithful Python adapters:

- **Live/backtest parity:** Same decision logic (PTB evidence, confluence scoring, risk gates, adaptation)
- **No-lookahead guarantees:** Multi-timeframe alignment enforces information causality
- **Realistic execution:** Spread, slippage, commission, latency, partial fills, rejections
- **Conservative same-bar SL/TP:** Assumes worst case when order ambiguous
- **Walk-forward analysis:** Train/test isolation with final untouched holdout
- **Monte Carlo robustness:** Percentile distributions, probability of loss/drawdown
- **Deterministic:** Same data + config + seed = same results

```bash
cd research && python3 -m patresearch.backtesting.cli run --strategy STANDARD_SCALPING --seed 42
```

See: [Backtesting Guide](../BACKTESTING.md)

---

## Signal States

All four strategies produce one of:

| State | Meaning |
|-------|---------|
| **BUY** | Valid long trade |
| **SELL** | Valid short trade |
| **WAIT** | Setup exists, MTF conflict prevents entry |
| **NO-TRADE** | Evaluation complete, no valid setup |
| **BLOCKED** | Valid candidate, hard gate vetoed (cooldown, duplicate, spread, risk) |
| **ERROR** | Missing data, system degraded |

---

## Hard Risk Gates (12 gates, short-circuit)

Gates are evaluated in order. The first failing gate produces BLOCKED:

1. Cooldown gate
2. Duplicate signal gate
3. Spread gate
4. Exposure gate
5. Max risk gate
6. Min RR gate
7. Margin/headroom gate
8. News/session gate
9. Recovery state gate
10. Adaptation risk gate
11. RL filter gate (when enabled)
12. License/entitlement gate

All gates are deterministic, freshness/version stamped, and fail closed.


## External Data Providers (v1.3.0)

| Provider | API | Status | Strategy Impact |
|----------|-----|--------|-----------------|
| COT | Financial Modeling Prep | Adapter implemented, HTTP 402 on free tier | Optional pillar cot_etf_flow (weight=0 default, weight=15 for TREND_SWING) |
| DXY | Twelve Data (6 currencies) | Adapter implemented, DXY computation verified | Mandatory for STANDARD_SWING (macro_dxy_yield, weight 20) & TREND_SWING (macro_real_yield_dxy, weight 20) |

### DXY Computation Formula
DXY = 50.14348112 × EUR/USD^(-0.576) × USD/JPY^(0.136) × GBP/USD^(-0.119) × USD/CAD^(0.091) × USD/SEK^(0.042) × USD/CHF^(0.036)

### Fail-Safe Behavior
- DXY unavailable → STANDARD_SWING and TREND_SWING fail to NO-TRADE (mandatory pillar not satisfied)
- COT unavailable → cot_etf_flow contributes 0 weight (optional pillar skipped)
- Neither DXY nor COT ever fabricate data
