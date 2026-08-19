# Predict-A-Trade XAUUSD

## System Overview

Production XAUUSD signal generation system with four independent strategy engines, deterministic mathematical scoring, hard risk gates, MT4/MT5 integration, SaaS control plane, and a Professional Trader Brain (PTB) shared intelligence layer with 20 advanced market analysis modules.

## Four Signal Strategies

| Strategy | Timeframes | Threshold | Min RR | Cooldown |
|----------|-----------|-----------|--------|----------|
| STANDARD_SCALPING | M1/M5 + M15/M30 | 65 | 1.2 | 15m |
| ULTRA_SCALPING | M1 + M5 | 85 | 1.0 | 15m |
| STANDARD_SWING | M15/M30/H1 + H4/D1 | 55 | 1.8 | 120m |
| TREND_SWING | H1/H4 + D1/W1 | 50 | 2.5 | 360m |

All four evaluate independently every eligible cycle. No master strategy copies results.

## Signal Decision States

```
BUY      — Valid long trade
SELL     — Valid short trade
WAIT     — Setup exists but entry confirmation not complete (MTF conflict)
NO-TRADE — Evaluation completed, no valid trade
BLOCKED  — Valid candidate but hard gate prevented execution (spread, risk, cooldown, etc.)
ERROR    — Evaluation could not be completed (missing data, system degraded)
```

## Architecture

```
MT5 Master Node → Windows Agent → WebSocket
    ↓
Realtime Engine (Go)
    ├── Market data ingestion → tick/candle aggregation
    ├── Feature engines → indicators, structure, liquidity, FVG, regime, MTF, session
    ├── PTB shared intelligence layer (20 modules, all SHADOW)
    │   ├── MTF bias, volatility regime, market phase
    │   ├── Liquidity void, wick fill, session imbalance
    │   ├── Stop hunt proxy, manipulation proxy, engineered liquidity
    │   ├── Gold correlation engine (DXY/silver/yields — awaiting live feed)
    │   ├── Synthesis engine → bias, confidence, confluence, setup quality
    │   └── Data authenticity guard → rejects non-live data
    ├── Four strategy engines (independent evaluation)
    ├── Signal engine → 12 hard gates (short-circuit)
    ├── Calibration → sigmoid probability (clamped 0-100)
    └── Persistence → TimescaleDB + WebSocket broadcast
        ↓
MT4/MT5 client delivery + Dashboard
```

- **Go Real-Time Engine**: Market data → features → PTB → strategy → gates → signals → WS/DB
- **NestJS Control Plane**: IAM, billing, licensing, referrals, commissions, payouts
- **Next.js Frontend**: Dashboard, admin, public site (pending vNext API freeze)
- **Windows Agent**: MT4/MT5 tick data bridge via WebSocket
- **PostgreSQL/TimescaleDB**: Signal persistence, audit trail, PTB analysis history
- **Valkey**: Hot cache for dashboard reads

## Live Master Node Data Requirement

All production signal generation requires `source_type = LIVE_MASTER_NODE`. Test, mock, demo, fixture, synthetic, and placeholder data sources are rejected by the DataAuthenticityGuard. No production signal can be generated from non-live data.

## Professional Trader Brain (PTB)

The PTB is a shared intelligence layer that enriches — but does NOT replace — the existing four strategy engines. All modules start in **SHADOW** mode with zero production score impact until validated.

### Synthesis Engine

The core PTB function combines all evidence into a unified assessment:

| Output | Values |
|--------|--------|
| Bias | STRONG_LONG, LONG, NEUTRAL, SHORT, STRONG_SHORT, STAND_ASIDE |
| Action | ENTER, WAIT, AVOID, EXIT |
| Setup Quality | A+, A, B, C, D, F |
| Position Size Multiplier | A+→1.0, A→0.8, B→0.6, C→0.4, D→0.2, F→0.0 |
| Stop Distance Multiplier | High manipulation→1.5, Normal→1.0, Low vol→0.8 |

### Advanced Modules

| Module | Status | Score Impact |
|--------|--------|-------------|
| Liquidity Void / Displacement | SHADOW | 0 |
| Wick Fill / Wickology | SHADOW | 0 |
| Session Imbalance | SHADOW | 0 |
| Candle Range Projector | SHADOW | 0 |
| Time At Mode | SHADOW | 0 |
| Engineered Liquidity Proxy | SHADOW | 0 |
| Market Phase | SHADOW | 0 |
| Relative Tick Volume Flow | SHADOW | 0 |
| Price Delivery | SHADOW | 0 |
| Stop Hunt Proxy | SHADOW | 0 |
| Institutional Footprint | UNSUPPORTED | 0 |
| Time Cycle Analytics | SHADOW | 0 |
| Algo Activity Proxy | SHADOW | 0 |
| Complete Liquidity Map | SHADOW | 0 |
| Manipulation / Dislocation Index | SHADOW | 0 |
| MTF Bias Engine | SHADOW | 0 |
| Volatility Regime Engine | SHADOW | 0 |
| S/R Quality Engine | SHADOW | 0 |
| Microstructure Engine | SHADOW | 0 |
| Statistical Performance Engine | SHADOW | 0 |
| Data Quality Engine | ACTIVE | 0 (informational) |
| Synthesis Engine | SHADOW | 0 |
| Gold Correlation Engine | SHADOW (awaiting live feed) | 0 |
| Gold Role Classification | SHADOW (returns UNKNOWN without data) | 0 |
| ML Pattern Layer | DISABLED/RESEARCH | 0 |

### Enhanced Regime Classification

```
STRONG_TREND_UP, STRONG_TREND_DOWN, WEAK_TREND_UP, WEAK_TREND_DOWN,
RANGE_BOUND, HIGH_VOLATILITY, LOW_VOLATILITY, TRANSITIONING, MANIPULATION
```

### Gold Correlation Engine

Computes rolling Pearson correlations between gold and DXY, silver, and US10Y yields. All external feeds default to UNAVAILABLE unless connected through the Master Node. No fabricated correlations — measures the actual relationship.

### Gold Role Classification

Determines what is driving XAUUSD: CURRENCY, SAFE_HAVEN, MONETARY_ASSET, COMMODITY, INFLATION_HEDGE, or UNKNOWN. Returns UNKNOWN when macro data is unavailable — no forced classification.

## Data Source Limitations

- **Broker tick volume** (not real centralized exchange volume) — used as proxy, clearly labeled
- **Volume Profile / Cumulative Delta**: UNSUPPORTED — cleanly disabled, does not break decisions
- **Institutional Footprint**: UNSUPPORTED — broker ticks cannot provide DOM/Level2/Time&Sales
- **DXY / Silver / Yields**: Correlation engine ready, awaiting live Master Node feed
- **COT Report**: Not connected — cleanly handled
- **Real Yields / Macro**: Research only — no production feed

## ML Status

```
ML STATUS = DISABLED / RESEARCH
```

The production trading pipeline is fully deterministic. No trained model is loaded. No AI/ML/LLM inference occurs in the runtime path.

## Risk Management

PTB position size multiplier and stop distance multiplier are **advisory only**. The existing risk gates remain authoritative:
- Max risk per trade, max daily loss, exposure limits
- Spread gates, drawdown rules, margin checks
- Subscription/license/entitlement checks
- 12 hard gates in short-circuit order

PTB cannot bypass or weaken any hard gate.

## Database

- **PostgreSQL + TimescaleDB**: 13 migrations, hypertables for time-series
- **PTB Analysis History**: `trading.ptb_analysis_history` (migration 013)
- **Signal Performance**: `trading.signal_performance` (migration 013)
- **Feature Flags**: `trading.ptb_feature_flags` (migration 012)
- **Evidence Snapshots**: `trading.ptb_evidence_snapshots` (migration 012)
- **Data Provenance**: `trading.data_provenance_log` (migration 012)

## Monitoring

Prometheus metrics for PTB:
- `pat_ptb_analysis_total` — analysis count by action
- `pat_ptb_analysis_latency_ms` — evaluation latency histogram
- `pat_ptb_setup_quality_total` — grade distribution
- `pat_ptb_regime_total` — regime distribution
- `pat_ptb_manipulation_index` — current manipulation index
- `pat_ptb_confluence_score` — current confluence score
- `pat_ptb_component_failure_total` — component failures
- `pat_ptb_stale_input_total` — stale input occurrences

## Configuration

| Setting | Default | Env Var |
|---------|---------|---------|
| PTB Enabled | true | `PTB_ENABLED` |
| Shadow Mode | true | `PTB_SHADOW_MODE` |
| Min Confidence | 65.0 | — |
| Min Confluence | 70.0 | — |
| Manipulation High Risk | 70.0 | — |
| Correlation Short Window | 20 | — |
| Correlation Medium Window | 50 | — |
| Correlation Long Window | 100 | — |

## Testing

```bash
# Go realtime engine (278 tests)
cd realtime && go build ./... && go vet ./... && go test ./...

# NestJS control plane (75 tests)
cd control && npm test

# Python research (98 tests)
cd research && pytest
```

**Total: 490 tests, 0 failures**

## 12 Hard Gates (Preserved — Fail Closed)

The signal pipeline enforces 12 deterministic hard gates in short-circuit order.
The first hard veto terminates evaluation. **None may be weakened, bypassed, or removed.**

| # | Gate | Purpose | Init State |
|---|------|---------|-----------|
| 1 | data_quality | Feed freshness and tick quality | PASS (live feed) |
| 2 | session | Strategy session suitability | PASS (live feed) |
| 3 | news | News blackout windows | PASS (live feed) |
| 4 | spread | Max spread absolute + spread/ATR | PASS (live feed) |
| 5 | slippage | Expected slippage limit | PASS (live feed) |
| 6 | total_cost | Cost-to-target ratio | PASS (live feed) |
| 7 | exposure | Aggregate XAUUSD exposure | UNKNOWN → PASS (broker telemetry) |
| 8 | margin | Margin headroom | UNKNOWN → PASS (broker telemetry) |
| 9 | rr_net_expectancy | Minimum R:R | PASS (live feed) |
| 10 | entitlement | Strategy entitlement | UNKNOWN → PASS (control plane) |
| 11 | license | License validity | UNKNOWN → PASS (control plane) |
| 12 | execution_permission | Execution permit / system mode | UNKNOWN → PASS (agent connect) |

**Fail-closed behavior:** Gates 7-12 start as UNKNOWN and deny all execution until
authoritative data arrives. When the Windows Agent connects (TICK/HEARTBEAT/MASTER_INIT),
the execution permit gate is hydrated to PASS. When a MARKET_SNAPSHOT with account_info
arrives, the exposure and margin gates are hydrated from live broker data. On agent
disconnect, gates expire and fail closed automatically. No hardcoded `true` values.
No optimistic defaults. No transient allow on restart.

## Entitlement & Execution Architecture

```
authenticated user
→ backend subscription lookup
→ active plan
→ entitled strategy
→ valid license
→ authorized bound device
→ active terminal
→ verified broker/account state
→ signal strategy eligibility
→ risk approval (12 hard gates)
→ execution approval
```

Backend/server-side enforcement is mandatory. Frontend renders derived state only.
The Go realtime engine derives entitlement/license/execution-permit flags from the
authoritative gate registry cached state — never from hardcoded values.

## Provider Modes

| Mode | Allowed In | Description |
|------|-----------|-------------|
| agent | production + dev | Live MT5 data from Windows Agent |
| simulated | dev/test ONLY | Synthetic ticks for development |
| replay | dev/test | Historical replay from fixture |

Production config validation rejects `PROVIDER_MODE=simulated` when `NODE_ENV=production`.

## COT (Commitment of Traders) Provider

The COT provider fetches weekly Commitment of Traders data from Financial Modeling Prep (FMP) API.

| Setting | Default | Env Var |
|---------|---------|---------|
| COT Enabled | false |  |
| FMP API Key | (not set) |  |
| COT Symbol | GC (Gold futures) |  |

**Fail-safe behavior:** If  is not set or the API returns 402 (restricted),
COT is marked as  — it NEVER fabricates data. Signal generation continues;
the  optional pillar contributes 0 weight. COT does not block signal generation.

**Note:** The current FMP subscription may not include COT endpoints (HTTP 402).
Upgrade the FMP plan or use an alternative COT data source if needed.

## DXY (US Dollar Index) Provider

The DXY provider computes the ICE US Dollar Index from 6 component currency pairs
fetched from the Twelve Data API.

| Setting | Default | Env Var |
|---------|---------|---------|
| DXY Enabled | false | `DXY_ENABLED` |
| Twelve Data API Key | (not set) | `TWELVEDATA_API_KEY` |

**Computation:** DXY = 50.14348112 × EUR/USD^(-0.576) × USD/JPY^(0.136) ×
GBP/USD^(-0.119) × USD/CAD^(0.091) × USD/SEK^(0.042) × USD/CHF^(0.036)

**Strategy impact:** STANDARD_SWING and TREND_SWING have mandatory DXY pillars
(weight 20). If DXY is unavailable, those strategies correctly fail to NO-TRADE.

**Fail-safe behavior:** If `TWELVEDATA_API_KEY` is not set, rate-limited (429),
or any component currency is unavailable, DXY is marked as `UNAVAILABLE` —
it NEVER fabricates data. The CorrelationEngine returns `NO_DXY_FEED`.

**Rate limiting:** Twelve Data free tier allows 8 API credits/minute. The DXY
provider makes 6 calls per refresh (one per currency pair) every 5 minutes —
well within the rate limit.

## Production Environment Requirements

- **DATABASE_URL**: Must be supplied via production secret. Insecure hardcoded
  passwords (e.g. `pat_local_dev_only`) cause startup failure in production.
- **JWT_SECRET**: Must be ≥32 chars, supplied via production secret file.
  Known placeholder secrets cause startup failure in production.
- **PROVIDER_MODE**: Must be `agent` in production.
- **Valkey/Redis**: Required for hot cache and cooldown state.
- **SMTP**: ✅ VERIFIED — `mail.predictatrade.com:587` (STARTTLS), user `no-reply@predictatrade.com`.
  Password reset and verification emails working. SMTP connection tested successfully.
- **TLS certificates**: Must be provisioned externally (Nginx).
- **COT/DXY feeds**: Optional — fail safely when unavailable.

## Current Status

**Production Readiness:** CONDITIONAL GO

See [PRODUCTION_FULL_AUDIT_REPORT.md](PRODUCTION_FULL_AUDIT_REPORT.md) for the full forensic audit.

```
CONDITIONAL GO

Software Blockers: 0 (all P1/P2 resolved)
Runtime Validation Required: Live MT4/MT5 terminal
External Configuration: TLS certificates, COT subscription upgrade
SMTP: ✅ Verified working (mail.predictatrade.com:587)
DXY: ✅ Verified (Twelve Data API, 6-currency ICE formula)
JWT Secret: ✅ Generated and stored (jwt_secret.txt, gitignored)
Database URL: ✅ Stored in secret file (database_url.txt, gitignored)
```

### Blocker Resolution Summary

| ID | Severity | Resolution | Final Status |
|----|----------|-------------|--------------|
| P1-001 | P1 | Replaced hardcoded `true` gate values with authoritative gate-registry state | RESOLVED |
| P1-002 | P1 | Changed canonical production env from `simulated` to `agent`; added config validation | RESOLVED |
| P2-001 | P2 | Strengthened JWT secret validation; rejects placeholders + short secrets in production | RESOLVED |
| P2-002 | P2 | Removed hardcoded DB password from production env files; added config validation | RESOLVED |
| P2-003 | P2 | Conservative gate seeding — safety-critical gates start UNKNOWN (fail closed) | RESOLVED |

## Advanced Risk, Adaptation, Hedging, ML/RL, Sentiment

The system includes an advanced intelligence layer on top of the existing four-strategy engine:

### Loss Recovery / Capital Protection
- State machine: NORMAL → RECOVERY → HALTED / DAILY_LIMIT
- Anti-martingale, anti-revenge-trading
- State isolated per account+strategy
- Restart-safe persistence
- See: [Advanced Risk Documentation](docs/ADVANCED_RISK_ADAPTATION_INTELLIGENCE.md)

### Rule-Based Adaptation
- Market phase classification (TRENDING, RANGING, HIGH_VOLATILITY, etc.)
- Dynamic parameter adjustment (stop distance, risk multiplier, confluence)
- Can only make system MORE conservative — never increases risk above hard limits
- Deep copy of weights — never mutates base config

### Controlled Hedging
- **DISABLED BY DEFAULT**
- Broker capability aware (hedging vs netting)
- Exposure capped, no martingale escalation
- Grid/options hedging OFF by default
- Full lifecycle audit trail

### ML-Based Adaptation
- Training runs OFFLINE in Python research plane
- Only inference in Go production hot path
- Comprehensive fallback to rule-based adaptation
- Data leakage protection (chronological split, walk-forward)
- Model registry with versioning

### RL Strategy Optimizer
- Modes: disabled → shadow → filter_only → live_approved
- Unvalidated model CANNOT directly control live execution
- Reward includes drawdown penalty, transaction costs, overtrading penalty
- Walk-forward/OOS validation required for live approval

### Real-Time Sentiment Engine
- Async background refresh — never blocks signal hot path
- Provider abstraction (GDELT, Reuters, Fed, Reddit, Twitter/X)
- Timeout, retry/backoff, stale-data detection
- Neutral fallback when unavailable
- Cached snapshot consumed by signal engine

### Daily Maintenance
- UTC-based daily reset
- Idempotent (multi-instance safe)
- Resets daily loss counters

### Database (Migration 014)
- `trading.recovery_states` — loss recovery state machine
- `trading.trade_results` — closed trade outcomes
- `trading.blocked_signals` — blocked signal audit
- `trading.adaptation_history` — adaptation decisions
- `trading.hedge_positions` — active hedge lifecycle
- `trading.hedge_history` — closed hedge audit
- `trading.rl_training_history` — RL training runs
- `trading.sentiment_snapshots` — cached sentiment state
- `trading.sentiment_items` — sentiment data points

### Testing (Updated)

```bash
# Go realtime engine (243 tests)
cd realtime && go build ./... && go vet ./... && go test ./...

# NestJS control plane (75 tests)
cd control && npm test

# Python research (26 tests)
cd research && pytest

# Next.js frontend (39 tests)
cd frontend && npm test
```

**Total: 490 tests, 0 failures**

## Backtesting Framework

The system includes a production-grade event-driven backtesting framework that reproduces the production strategy/PTB/risk gate logic through faithful Python adapters.

### Key Features

- **Live/backtest parity**: Same decision logic (PTB evidence, confluence scoring, risk gates, adaptation)
- **No-lookahead guarantees**: Multi-timeframe alignment enforces information causality
- **Realistic execution**: Spread, slippage, commission, latency, partial fills, rejections
- **Conservative same-bar SL/TP**: Assumes worst case when order ambiguous
- **Exit management**: Trailing stop, break-even, time exit
- **Walk-forward analysis**: Train/test isolation with final untouched holdout
- **Monte Carlo robustness**: Percentile distributions, probability of loss/drawdown
- **Parameter sensitivity**: ±% perturbation analysis without altering production
- **Deterministic**: Same data + config + seed = same results
- **Reproducible**: Run manifests with full provenance

### CLI

```bash
# Run a backtest
cd research && python3 -m patresearch.backtesting.cli run --strategy STANDARD_SCALPING --seed 42

# Walk-forward analysis
cd research && python3 -m patresearch.backtesting.cli walk-forward --strategy STANDARD_SCALPING

# Monte Carlo
cd research && python3 -m patresearch.backtesting.cli monte-carlo --runs 1000
```

See: [Backtesting Guide](docs/BACKTESTING.md)

### Testing (Updated)

```bash
# Go realtime engine (243 tests)
cd realtime && go build ./... && go vet ./... && go test ./...

# Python research (98 tests — includes backtesting)
cd research && python3 -m pytest tests/

# NestJS control plane (75 tests)
cd control && npm test

# Next.js frontend (39 tests)
cd frontend && npm test
```

**Total: 490 tests, 0 failures**
