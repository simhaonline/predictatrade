# Advanced Risk, Adaptation, Hedging, ML/RL, and Sentiment

## Overview

This document covers the advanced intelligence layer implemented on top of the existing four-strategy signal engine. All features are designed with safety-first principles:

- **No martingale, no doubling after losses**
- **No automatic reverse trades**
- **Recovery reduces risk, never increases it**
- **Adaptation can only make the system more conservative**
- **Hedging is disabled by default**
- **ML/RL are research-only by default**
- **Sentiment is async-cached, never blocks the signal hot path**

## Loss Recovery / Capital Protection Manager

**Location:** `realtime/internal/recovery/manager.go`

### State Machine

```
NORMAL → RECOVERY → HALTED
  ↑         ↓
  ←←←←←←←←←←
  (after configured wins)

NORMAL → DAILY_LIMIT (new day resets)
```

### States

| State | Behavior |
|-------|----------|
| NORMAL | Full risk, normal thresholds |
| RECOVERY | Reduced risk (0.5x), higher confluence/grade/confidence required |
| HALTED | No trading until halt expires |
| DAILY_LIMIT | No trading until new trading day |

### Critical PnL Correctness

The daily loss circuit breaker uses CORRECT sign logic:

```
daily_pnl_percent <= -max_daily_loss_percent
```

**NOT** `abs(daily_pnl) >= max_daily_loss` — a profitable day must never trigger a loss circuit breaker.

### Configuration

| Setting | Default | Env Var |
|---------|---------|---------|
| Max Daily Loss % | 2.0 | MAX_DAILY_LOSS_PERCENT |
| Max Daily Loss Count | 3 | MAX_DAILY_LOSS_COUNT |
| Max Consecutive Losses | 2 | MAX_CONSECUTIVE_LOSSES |
| Recovery Size Multiplier | 0.50 | — |
| Recovery Min Confluence | 80 | — |
| Recovery Min Setup Quality | A | — |
| Recovery Min Confidence | 75 | — |
| Recovery Max Trades | 3 | — |
| Recovery Exit After Wins | 2 | — |
| Normal Cooldown (min) | 5 | — |
| Recovery Cooldown (min) | 30 | — |
| Halt Cooldown (min) | 60 | — |

### State Isolation

State is isolated per `account_id + strategy_id + symbol`. One client's losses do not halt another client's trading.

### Restart Safety

State can be exported via `AllStates()` and restored via `RestoreStates()`. Halt state survives restart.

### Duplicate Close Event Handling

Each broker close event has a unique `close_event_id`. Duplicate events are deduplicated and ignored.

## Rule-Based Adaptation Manager

**Location:** `realtime/internal/adaptation/manager.go`

### Market Phases

| Phase | Condition | Behavior |
|-------|-----------|----------|
| TRENDING | Strong directional regime | Normal risk, boost trend/structure weights |
| RANGING | Range/mean-reversion regime | Reduce risk 0.7x, higher confluence, boost S/R |
| HIGH_VOLATILITY | High/extreme volatility | Reduce risk 0.5x, widen stops 1.5x, +10 confluence |
| LOW_VOLATILITY | Low volatility | Tighter stops 0.8x, slight risk reduction |
| MANIPULATIVE | Manipulation > 70 | Maximum caution, risk 0.3x, +15 confluence |
| UNCERTAIN | Unknown/unstable | Conservative fallback, risk 0.6x |

### Safety

- Adaptation operates on a **deep copy** of weights — never mutates base config
- Risk multiplier is **clamped** to `MaxRiskMultiplier` and `GlobalHardMaxRisk`
- The most restrictive valid limit wins
- NaN/Inf values are rejected and replaced with safe fallback
- Weights are normalized after adjustment

## Controlled Hedge Manager

**Location:** `realtime/internal/hedging/manager.go`

### Default: DISABLED

Hedging must be explicitly enabled via `HEDGING_ENABLED=true`.

### Pre-Execution Checks

Before any hedge:
1. Feature is enabled
2. Broker/account supports hedging
3. Account is not netting mode
4. License permits trading
5. Original position is still open
6. Hedge does not already exist
7. Aggregate exposure within limit
8. Loss is within thresholds
9. Market data is fresh
10. Manipulation/volatility acceptable

### No Martingale

Hedge size is capped at `HedgeSizeCap` (default 0.5 = 50% of original). Hedge size never exceeds original size.

### Grid Hedging: OFF by default

Grid hedging requires explicit `GRID_HEDGING_ENABLED=true`. No exponential volume increase.

### Options Hedging: OFF by default

Options hedging requires explicit `OPTIONS_HEDGING_ENABLED=true` and a compatible options data provider.

## ML-Based Adaptation

**Location:** `realtime/internal/ml/adaptation.go` (Go inference)
**Location:** `research/src/patresearch/ml_training.py` (Python training)

### Architecture Separation

- **Training:** Runs OFFLINE in the Python research plane
- **Inference:** Only inference runs in the Go production hot path

### Fallback Chain

If any of these conditions occur, the system falls back to rule-based adaptation:
1. Model absent
2. Model stale
3. Model incompatible
4. Inference error
5. Features missing
6. Confidence inadequate
7. Version invalid

The live signal engine continues safely. Never fails open into higher risk.

### Data Leakage Protection

- Chronological split (train → validation → test)
- Walk-forward validation
- Feature timestamps must precede prediction outcomes
- Minimum 100 training samples (configurable)

### Model Registry

Persists: model name, version, trained_at, dataset period, sample count, feature schema, validation metrics, artifact path, active status, checksum.

## RL Strategy Optimizer

**Location:** `realtime/internal/rl/optimizer.go` (Go inference/filter)
**Location:** `research/src/patresearch/rl_training.py` (Python training)

### Production Rollout States

```
disabled → shadow → filter_only → live_approved
```

| Mode | Behavior |
|------|----------|
| disabled | Returns NO_TRADE always |
| shadow | Observes but cannot block or execute |
| filter_only | May veto (NO_TRADE) but cannot create trades |
| live_approved | May influence live trading (requires explicit authorization) |

### Critical Rule

`live_approved` must **never** automatically become active merely because a model file exists. It requires explicit operator authorization and validation metrics meeting thresholds.

### Reward Function

Reward accounts for more than raw PnL:
- Realized PnL
- Drawdown penalty
- Transaction costs
- Spread/slippage costs
- Overtrading penalty
- Risk exposure penalty
- Holding cost

### Validation

- Historical replay
- Walk-forward validation
- Out-of-sample period
- Max drawdown, Sharpe/Sortino, profit factor, win rate, expectancy, trade count

## Real-Time Sentiment Engine

**Location:** `realtime/internal/sentiment/engine.go`

### Architecture

```
sentiment collector (background goroutine)
      ↓
background refresh (async, with timeout/retry/backoff)
      ↓
cache/state (in-memory snapshot)
      ↓
signal engine consumes latest valid snapshot (no blocking)
```

### Provider Abstraction

Each provider implements the `Provider` interface. One unavailable provider does not break the system.

### Provenance

Each sentiment item retains: source, provider, timestamp, fetched_at, headline_id, score, confidence, age, category.

### Real-Time Safety

- **No synchronous external HTTP calls** in the signal hot path
- `GetSnapshot()` returns cached data only — never blocks
- Timeout, retry/backoff, rate-limit handling, stale-data detection
- Bounded cache, error logging, provider health tracking

### Fallback

If sentiment is unavailable or stale:
```
neutral/no-impact fallback (influence = 0)
```

### PTB Influence

Sentiment may modify bias and confluence within bounded limits. It can make the system more conservative but never more aggressive.

## Daily Maintenance

**Location:** `realtime/internal/maintenance/scheduler.go`

### Daily Reset

- Resets daily loss counters at the configured UTC time (default: midnight UTC)
- Uses locking to prevent duplicate execution in multi-instance deployments
- Idempotent: running twice on the same day is a no-op

## Database Tables

**Migration:** `database/migrations/014_advanced_risk_adaptation_intelligence.sql`

| Table | Purpose |
|-------|---------|
| trading.recovery_states | Loss recovery state machine per account+strategy |
| trading.trade_results | Closed trade outcome persistence |
| trading.blocked_signals | Audit trail of signals blocked by recovery/risk |
| trading.adaptation_history | Rule-based and ML-based adaptation decisions |
| trading.hedge_positions | Active hedge lifecycle tracking |
| trading.hedge_history | Closed hedge audit trail with net PnL |
| trading.rl_training_history | RL model training runs with OOS validation |
| trading.sentiment_snapshots | Cached real-time sentiment state |
| trading.sentiment_items | Individual sentiment data points with provenance |

## Prometheus Metrics

| Metric | Type | Description |
|--------|------|-------------|
| pat_trading_state | Gauge | Recovery state per account+strategy |
| pat_daily_pnl_percent | Gauge | Daily PnL percentage |
| pat_consecutive_losses | Gauge | Current consecutive loss count |
| pat_recovery_entries_total | Counter | Recovery mode entries |
| pat_recovery_blocks_total | Counter | Signals blocked by recovery |
| pat_daily_limit_hits_total | Counter | Daily loss limit hits |
| pat_adaptation_phase | Gauge | Current adaptation phase |
| pat_adaptation_changes_total | Counter | Adaptation changes |
| pat_active_hedges | Gauge | Active hedge count |
| pat_hedges_opened_total | Counter | Hedges opened |
| pat_hedges_closed_total | Counter | Hedges closed |
| pat_hedge_pnl | Gauge | Aggregate hedge PnL |
| pat_ml_model_loaded | Gauge | ML model loaded (1/0) |
| pat_ml_prediction_confidence | Gauge | ML prediction confidence |
| pat_ml_fallback_total | Counter | ML fallback occurrences |
| pat_rl_mode | Gauge | RL mode (0-3) |
| pat_rl_shadow_decisions_total | Counter | RL shadow decisions |
| pat_rl_filter_blocks_total | Counter | RL filter blocks |
| pat_sentiment_score | Gauge | Overall sentiment score |
| pat_sentiment_confidence | Gauge | Sentiment confidence |
| pat_sentiment_age_seconds | Gauge | Sentiment snapshot age |
| pat_sentiment_provider_errors_total | Counter | Provider errors by provider |

## Signal Pipeline Wiring

```
Market Data
    ↓
Indicators → Features
    ↓
PTB (shared intelligence, SHADOW mode)
    ↓
Adaptation Manager (rule-based, adjusts parameters)
    ↓
Four Strategy Engines (STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING)
    ↓
Confluence Scoring (deterministic)
    ↓
Signal Engine (DecideWithAdvanced)
    ├── 12 Hard Gates (short-circuit, existing)
    ├── Recovery Gate (loss recovery state machine)
    ├── Adaptation (applies adjusted parameters)
    ├── ML Prediction (inference with fallback)
    ├── RL Filter (may veto in filter mode)
    ├── Sentiment Influence (cached, async)
    └── Size Multiplier (from recovery state)
    ↓
Signal Persistence → TimescaleDB
    ↓
WebSocket Broadcast → MT4/MT5 → Dashboard
    ↓
Trade Close Feedback → Recovery Manager (updates state)
```
