# Predict-A-Trade

# Final Full Scope of Work — Production-Grade XAUUSD Intelligence, Signals, Execution, Live Command Center, Referral Growth & Commission Platform

**Project:** Predict-A-Trade  
**Platform:** XAUUSD Intelligent Prediction, Signal Distribution, Execution, Subscription, Referral and Growth Intelligence Platform  
**Version:** v1.0.0 — Final Standalone Integrated Full-Stack Production Specification  
**Primary Asset:** XAUUSD / Gold Spot  
**Primary Web Frontend:** Next.js  
**Control/API Backend:** NestJS  
**Real-Time Trading Engine:** Go  
**Research / ML / AI / Quantum-Inspired Research:** Python  
**Terminal Integration:** MQL4 / MQL5 + Go Windows Agent  
**Primary Database:** PostgreSQL 17  
**Time-Series:** TimescaleDB  
**Vector Search:** pgvector  
**Connection Pooling:** PgBouncer  
**Hot State / Cache:** Valkey  
**Broker Display/Candle Clock:** UTC+03:00 unless a validated broker profile explicitly proves otherwise  
**Internal Time Truth:** UTC  
**Production Target:** Secure self-hosted production environment  
**Commercial Model:** Three monthly subscription tiers with one-time setup fees, granular four-strategy entitlements, configurable five-level referral commissions, immutable commission ledger and controlled payouts  
**Revision Date:** 17 August 2026  
**Document Status:** Canonical standalone Scope of Work; no companion addendum is required to understand or implement the scope.

**Integrated UI/UX Baseline v1.0.0 (17 August 2026):** Part IV — Sections 185–200 — Visual System, Live Chart and User/Admin Dashboard Specification is part of this canonical baseline, not a later addendum. Sections 1–200 are simultaneously authoritative. Part IV adds presentation-plane requirements and, only where explicitly listed in Section 200.1, refines presentation-layer details of Sections 31, 32, 33, 32A, 127, 152 and 167–184. No trading, risk, financial, licensing, entitlement, security or compliance requirement is weakened.  

**Specification Authority:** This single document contains the complete implementation scope from Sections 1 through 200 (Parts I–IV). Earlier SOWs, gap analyses, quantum/realtime supplements and Live Command Center additions have been consolidated into this document. Existing production behavior shall be preserved unless this specification explicitly requires a compatible migration.

> **Non-regression authority:** Hard safety, security, financial-integrity, broker, data-quality, entitlement, compliance, execution and risk controls always win over convenience or presentation requirements. Where two safety limits both apply, the stricter fail-closed rule governs.

> **Accuracy terminology rule:** “Accuracy” shall never be used without a named metric and denominator. Prediction outcome quality, probability calibration, execution correctness, signal-delivery reliability and commercial-ledger correctness are distinct metrics.

> **Critical interpretation rule:** Numeric thresholds in XAUUSD strategy sections are initial versioned baseline parameters, not universal truths or guaranteed trading edges. They must remain configuration-backed, reproducible, realistically cost-tested, walk-forward/OOS validated, paper/shadow observed, calibrated, approved, versioned and reversible before live activation.

> **Entitlement is not strategy logic:** `STANDARD_SCALPING`, `ULTRA_SCALPING`, `STANDARD_SWING` and `TREND_SWING` are separately versioned strategy products. Commercial entitlement grants access; it does not define the trading logic.

> **Data capability rule:** Never fabricate order-flow, CVD, DOM, aggressor-side, iceberg, global resting-liquidity or depth features when the subscribed source does not provide the required data. Missing capability becomes explicit quality state and may lower grade or force `NO-TRADE`.

> **Dashboard truth rule:** Every live price, indicator, market-state label, liquidity object, evidence node, signal, risk state, execution event, position value, referral node, commission amount and payout state must trace to an authoritative source, deterministic calculation, approved calibrated model, broker/terminal event or immutable commercial ledger record. Live mode shall contain no fabricated activity.

---

# 1. Executive Objective

Build Predict-A-Trade into a complete production-grade XAUUSD intelligence platform capable of:

- ingesting and validating real-time XAUUSD and macro market data;
- generating deterministic BUY / SELL / NO-TRADE predictions;
- estimating calibrated probability rather than presenting arbitrary confidence scores as probability;
- validating signals through quantitative, macro, AI and optional experimental intelligence pillars;
- applying hard risk controls;
- distributing signals in real time to web users and licensed Windows MT4/MT5 clients;
- optionally executing approved signals through MT4/MT5;
- managing users, administrators, subscriptions, plans, strategy entitlements, licenses, devices and trading accounts;
- operating a configurable multi-level referral, commission, affiliate-wallet and payout system driven only by validated subscription payments;
- monitoring the entire signal-to-broker lifecycle;
- storing complete historical and audit records;
- supporting replay, backtesting, calibration and research;
- providing production Admin and User dashboards;
- providing secure Windows client installation, activation and update infrastructure;
- providing auditable and measurable production reliability.

The complete lifecycle shall be:

```
MARKET DATA
    ↓
NORMALIZATION
    ↓
DATA QUALITY
    ↓
TIME-SERIES / FEATURE STATE
    ↓
MARKET REGIME
    ↓
MULTI-TIMEFRAME ANALYSIS
    ↓
STRUCTURE / LIQUIDITY / VOLUME / VOLATILITY
    ↓
DXY / YIELDS / NEWS / MACRO
    ↓
EXISTING PREDICT-A-TRADE PILLARS
    ↓
QUANTITATIVE SCORING
    ↓
PREDICTION CANDIDATE
    ↓
OPTIONAL AI / EXPERIMENTAL VERIFICATION
    ↓
CALIBRATION
    ↓
HARD RISK ENGINE
    ↓
MASTER DECISION
    ↓
BUY / SELL / NO-TRADE
    ↓
REAL-TIME SIGNAL DISTRIBUTION
    ├── USER DASHBOARD
    ├── ADMIN DASHBOARD
    └── LICENSED WINDOWS MT4/MT5 CLIENT
              ↓
       OPTIONAL EXECUTION
              ↓
            BROKER
              ↓
        RECONCILIATION
              ↓
           OUTCOME
              ↓
       ANALYTICS / RESEARCH

```

**NO-TRADE must remain a first-class and acceptable outcome.**

The goal is not maximum trade frequency.

The goal is **maximum empirically measurable risk-adjusted decision quality with safe execution**.

---

# 2. Non-Negotiable Engineering Principles

The rebuilt platform shall remain:

- deterministic first;
- AI-assisted rather than AI-dependent;
- broker-independent;
- low latency;
- fault tolerant;
- explainable;
- auditable;
- replayable;
- backtestable;
- calibratable;
- secure by default;
- license-aware;
- multi-user capable;
- API-first;
- observable;
- recoverable;
- backward compatible wherever possible;
- financially precise for all subscription, commission and payout calculations;
- referral/commission logic isolated from the real-time trading path;
- commissionable only from validated eligible product/subscription revenue, never from recruitment alone.

Hard risk rules shall always have veto authority.

A valid signal shall not automatically imply an executable order.

AI shall never override:

```
DATA QUALITY FAILURE
NEWS BLACKOUT
MAXIMUM SPREAD
MAXIMUM SLIPPAGE
MAXIMUM EXPOSURE
ACCOUNT RESTRICTIONS
LICENSE RESTRICTIONS
EXECUTION PERMISSIONS
BROKER RESTRICTIONS
EMERGENCY STOP

```

---

# 3. Mandatory Four-Plane Architecture

The platform shall be separated into four major planes.

## 3.1 Real-Time Trading Plane — Golang

Responsible for:

```
Market-data ingestion
Normalization
Tick processing
Candle aggregation
Feature updates
Regime detection
Signal scoring
Signal lifecycle
Risk validation
Execution authorization
Real-time signal streaming
MT4/MT5 execution gateway
Execution reconciliation
Critical health state

```

This remains the authoritative low-latency production path.

## 3.2 Intelligence / Research Plane — Python

Responsible for:

```
Historical research
Backtesting
Walk-forward testing
ML
Calibration
NLP
News intelligence
AI orchestration
Vision
Pattern similarity
Feature research
Model evaluation
Dataset generation
Offline optimization

```

Python must not become mandatory for every real-time tick decision.

## 3.3 SaaS / Control Plane — NestJS

Responsible for:

```
Authentication
Authorization
Users
Administrators
Organizations / tenants
Roles / permissions
Subscriptions
Plans
Entitlements
Licensing
Windows device registration
MT4/MT5 account registration
Client downloads
Client versions
Configuration
Feature flags
Audit
Notifications
Support
Administrative operations
Billing integration
Subscription lifecycle
Referral attribution
Commission engine
Affiliate wallet / payouts
Legal acknowledgements
Public control APIs
Dashboard APIs

```

**NestJS must not replace the Go signal engine.**

## 3.4 Presentation Plane — Next.js

Provide:

```
Public website
User Portal
Admin Portal
Authentication UI
Signal Terminal
Historical Analytics
License Management
Downloads
Configuration
Support
Operational Monitoring

```

---

# 4. Production Component Topology

```
                     ┌─────────────────────────┐
                     │     PUBLIC WEBSITE      │
                     │        Next.js          │
                     └─────────────────────────┘

                     ┌─────────────────────────┐
                     │      USER PORTAL        │
                     │        Next.js          │
                     └───────────┬─────────────┘
                                 │
                     ┌───────────▼─────────────┐
                     │      ADMIN PORTAL       │
                     │        Next.js          │
                     └───────────┬─────────────┘
                                 │
                                 ▼
                     ┌─────────────────────────┐
                     │ NESTJS CONTROL PLANE    │
                     │ IAM / License / Admin   │
                     │ Tenant / Audit / Plans  │
                     └─────┬───────────┬───────┘
                           │           │
                  PostgreSQL           Valkey
                           │
                TimescaleDB + pgvector
                           │
        ┌──────────────────┴───────────────────┐
        │                                      │
        ▼                                      ▼
┌───────────────────┐                 ┌───────────────────┐
│ GO REAL-TIME      │                 │ PYTHON RESEARCH   │
│ ENGINE            │                 │ / AI              │
├───────────────────┤                 ├───────────────────┤
│ Market Data       │                 │ ML                │
│ Features          │                 │ Backtest          │
│ Regime            │                 │ Calibration       │
│ Structure         │                 │ NLP               │
│ Liquidity         │                 │ AI                │
│ Signals           │                 │ Vision            │
│ Risk              │                 │ Research          │
└─────────┬─────────┘                 └───────────────────┘
          │
          ▼
┌───────────────────────┐
│ REAL-TIME GATEWAY     │
│ HTTPS / WebSocket     │
└───────────┬───────────┘
            │
    Signed entitlement
    + authenticated client
            │
            ▼
┌───────────────────────┐
│ WINDOWS AGENT — GO    │
├───────────────────────┤
│ Licensing             │
│ Device Identity       │
│ Signal Stream         │
│ Heartbeat             │
│ Client Updates        │
│ Local Audit           │
│ MT4/MT5 IPC           │
└───────┬────────┬──────┘
        │        │
        ▼        ▼
     MT4 EA    MT5 EA
        │        │
        └────┬───┘
             ▼
           BROKER

```

---

# 5. Technology Responsibilities

## Golang

Use Golang for latency-sensitive and execution-sensitive workloads.

## Python

Use Python for research and asynchronous intelligence workloads.

## NestJS

Use NestJS for the application/control plane.

Recommended modules:

```
AuthModule
UsersModule
OrganizationsModule
RolesModule
LicensesModule
DevicesModule
MTAccountsModule
PlansModule
EntitlementsModule
SignalsQueryModule
AnalyticsModule
DownloadsModule
UpdatesModule
NotificationsModule
SupportModule
AuditModule
ConfigModule
FeatureFlagsModule
AdminModule
BillingModule
SubscriptionsModule
ReferralsModule
CommissionsModule
PayoutsModule
AffiliateRiskModule
FinanceReportingModule
HealthModule

```

## Next.js

Use the App Router and server-side data access where appropriate.

Create reusable:

```
Design System
Authentication Shell
Role-aware Navigation
Tables
Data Grids
Charts
Real-time Signal Cards
Trading Terminal Components
Audit Viewer
Status Components
License Components
MT Setup Wizard

```

## MQL4 / MQL5

Keep terminal code lightweight.

No primary predictive intelligence shall reside inside the EA.

---

# 6. Market Data Architecture

Support at minimum:

```
XAUUSD
DXY
US2Y
US5Y
US10Y
US30Y
Real yields where available
Economic calendar
Macro news
Relevant geopolitical news

```

XAUUSD data shall include:

```
Bid
Ask
Mid
Spread
Tick volume
Timestamp
Broker timestamp
Source timestamp
Gateway timestamp
Receipt timestamp
OHLC
Symbol specification
Contract size
Tick size
Tick value
Digits

```

---


# 6A. XAUUSD Flow, Futures, Cross-Market and Data-Capability Stack

The market-data plane shall distinguish **spot price truth**, **broker-local volume proxies**, **centralized futures trade flow**, **order-book depth**, and **slow macro/positioning data**. They are not interchangeable.

## 6A.1 Minimum Data Capabilities

Define a capability registry:

```text
SPOT_BID_ASK
SPOT_BROKER_TICK_VOLUME
GC_TRADES
GC_TOP_OF_BOOK
GC_MARKET_BY_PRICE
GC_MARKET_BY_ORDER
DXY
NOMINAL_YIELDS
REAL_YIELDS
ECONOMIC_CALENDAR
NEWS
COT
ETF_FLOWS
CENTRAL_BANK_FLOW
```

Each provider/feed record shall identify:

```text
provider_id
dataset_id
instrument
capabilities[]
source_timezone
timestamp_precision
sequence_support
historical_depth
redistribution_rights
retention_rights
quality_sla
enabled
```

A strategy must declare `required_capabilities[]` and `optional_capabilities[]`.

## 6A.2 Spot XAUUSD Volume Semantics

Spot XAUUSD is an OTC/decentralized market. Broker tick volume is therefore a **broker-local activity proxy**, not a centralized total market-volume measure.

Store it explicitly as:

```text
BROKER_TICK_VOLUME_PROXY
```

It may contribute to a validated feature but must never be presented as centralized XAUUSD volume.

## 6A.3 GC Futures Flow

When licensed data is available, ingest COMEX Gold futures data to create centralized exchange-flow evidence such as:

```text
GC trade volume
GC bid/ask
GC aggressor-side delta when supported
CVD
session CVD
rolling CVD
delta divergence
volume profile
POC / VAH / VAL
trade imbalance
depth imbalance when supported
absorption/exhaustion candidates
```

Do not infer true aggressor-side delta from a feed that does not support a defensible classification method. If a derived proxy is used, label it `ESTIMATED` and record the derivation/version.

## 6A.4 Futures Contract Selection, Roll and Basis

The engine shall not treat a futures symbol as a timeless continuous instrument.

Implement:

```text
futures_contracts
futures_roll_calendar
active_contract_selection
continuous_series_method
roll_adjustment
spot_futures_basis
basis_quality
```

Active-contract selection shall be driven by approved rules using expiry/volume/open-interest/liquidity rather than a hard-coded ticker.

Every feature derived from a continuous futures series shall preserve:

```text
source_contract
roll_state
adjustment_method
basis
quality_flag
```

Roll discontinuities must not be mistaken for market displacement or structural breaks.

## 6A.5 Lead/Lag Is Measured, Not Assumed

Track time-varying:

```text
gc_spot_lead_lag_ms
gc_spot_return_correlation
gc_spot_basis_zscore
```

No strategy may assume that futures always lead spot. Lead/lag is an empirical feature with confidence/quality metadata.

## 6A.6 Optional Depth / Order-Book Features

DOM, stacked imbalance, heatmap, replenishment and order-book anomaly features require an appropriate depth feed.

Capability modes:

```text
NONE
TOP_OF_BOOK
MARKET_BY_PRICE
MARKET_BY_ORDER
```

A strategy requiring depth shall degrade or become `NO-TRADE` when the necessary capability is unavailable.

Do not label behavior as illegal “spoofing” from an algorithmic pattern alone. Use neutral states such as:

```text
ORDER_BOOK_ANOMALY
RAPID_PULLING
REPLENISHMENT
ABSORPTION_CANDIDATE
```

unless separately verified.

## 6A.7 Slow-Moving Context Feeds

Support optional:

```text
COT positioning
GLD / approved ETF holdings/flows
central-bank gold demand data
physical-premium data where licensed
risk/geopolitical indices
```

These are primarily swing/trend context features and must not be used as millisecond execution triggers.

## 6A.8 Flow Data Quality

Every flow feature stores:

```text
feature_id
source
capability
observed_at
event_time
value
quality
latency_ms
derivation_version
contract
roll_state
```

Quality states:

```text
AUTHORITATIVE
DERIVED
ESTIMATED
DEGRADED
STALE
UNAVAILABLE
INVALID
```

# 7. Redundant Market Data and Feed Quality

Production shall support:

```
PRIMARY_FEED
SECONDARY_FEED
BROKER_FEED

```

Implement:

- stale-feed detection;
- divergence detection;
- timestamp skew detection;
- reconnect handling;
- feed failover;
- bad-tick rejection;
- outlier detection;
- duplicate tick detection;
- out-of-order detection;
- gap detection;
- market-hours awareness.

Failover must not silently change price sources without recording an event.

---

# 8. Historical Candle Architecture

Do not rely on accumulated live ticks alone to construct all dashboard history.

The platform must provide authoritative historical candles for:

```
M1
M5
M15
M30
H1
H4
D1
W1
MN1

```

Implement:

```
Historical backfill
Gap repair
Incremental refresh
Candle reconciliation
Source tracking
Quality flags

```

The dashboard must distinguish:

```
COMPLETE
PARTIAL
ESTIMATED
STALE
INVALID

```

candle states.

---


# 8A. Tick-Level Historical Truth, Provenance and Reconciliation

Ultra-scalping research cannot be validated from OHLC candles alone.

Maintain historical datasets at the highest licensed resolution available:

```text
spot bid/ask ticks
broker tick-volume events
GC trades/quotes/depth where licensed
M1 authoritative candles
higher-timeframe candles
economic-event releases
macro series
broker execution/cost observations
```

The initial research objective may target history from 2015 onward where lawful/licensed data quality permits, but **date coverage is a data-availability target, not permission to invent or interpolate missing tick truth**.

Every historical dataset shall preserve:

```text
provider
dataset version
download/import timestamp
source timezone
raw checksum
normalization version
gap report
quality report
license/retention metadata
```

Backfilled and live-built candles must be reconciled. If sources disagree, preserve the source-specific values and the reconciliation result rather than silently overwriting history.

For tick-level backtests, store or reconstruct bid and ask separately where possible. Mid-price-only research must be explicitly marked as unsuitable for final ultra-scalping cost validation.

# 9. Clock and Time Synchronization

All internal timestamps shall use UTC.

Production servers and Windows clients must maintain synchronized clocks.

Record distinct timestamps for:

```
Source event
Gateway receipt
Signal creation
Risk decision
Signal publication
Windows receipt
EA receipt
Broker submission
Broker acknowledgement
Fill
Exit

```

Use monotonic timers for internal latency measurement where appropriate.

---

# 10. Multi-Timeframe Engine

Support:

```
D1
H4
H1
M30
M15
M5
M1
Tick
Seconds where available

```

Recommended hierarchy:

```
D1   → Macro Bias
H4   → Major Structure
H1   → Intraday Bias
M30  → Session Structure
M15  → Setup
M5   → Confirmation
M1   → Entry
Tick → Execution Timing

```

---


# 10A. Strategy-Specific Multi-Timeframe Alignment

The generic timeframe hierarchy is not sufficient. Each strategy shall use a versioned `strategy_timeframe_profile`.

Minimum fields:

```text
strategy_id
version
context_timeframes[]
setup_timeframes[]
confirmation_timeframes[]
entry_timeframes[]
execution_resolution
required_freshness_ms_by_timeframe
alignment_weights
conflict_policy
invalidation_timeframe
```

Calculate a reproducible:

```text
mtf_alignment_score ∈ [-100, +100]
```

with an evidence breakdown rather than an opaque number.

A timeframe is not considered valid merely because a cached candle exists. Continuously monitor:

```text
timeframe_sync_lag_ms
last_complete_candle_at
expected_next_candle_at
gap_count
source_quality
```

If an entry timeframe required by a strategy is stale/invalid, the strategy must not trade even if higher timeframes remain healthy.

## 10A.1 Initial Strategy Timeframe Profiles

These are versioned baselines subject to validation.

### STANDARD_SCALPING

```text
Context:      H1 + M30
Setup:        M15 + M5
Entry:        M5
Timing:       M1 / tick when available
Typical hold: 1–15 minutes
Initial TTL:  15 minutes
```

### ULTRA_SCALPING

```text
Context:      M15 + M5
Setup:        M5 + M1
Entry:        M1
Timing:       tick / sub-minute if authoritative
Typical hold: 10 seconds–3 minutes
Initial TTL:  3 minutes
```

### STANDARD_SWING

```text
Context:      D1 + H4
Setup:        H1 + M15
Entry:        H1 / M15
Typical hold: 4–48 hours
Initial TTL:  4 hours
```

### TREND_SWING

```text
Context:      W1 + D1
Setup:        H4 + H1
Entry:        H4 / H1
Typical hold: 1–10 days
Initial TTL:  24 hours
```

## 10A.2 Conflict Policy

Examples of conflict shall be encoded, tested and strategy-specific.

A lower-timeframe bullish trigger against a strongly bearish required higher-timeframe bias must not silently pass. The strategy definition shall specify one of:

```text
HARD_REJECT
REQUIRE_REVERSAL_CONFIRMATION
DOWNGRADE_GRADE
IGNORE_NON_REQUIRED_TF
```

Default for production candidates: unresolved required-timeframe conflict → `NO-TRADE`.

# 11. Market Regime Engine

Classify:

```
TRENDING_BULLISH
TRENDING_BEARISH
RANGE
BREAKOUT
MEAN_REVERSION
HIGH_VOLATILITY
LOW_VOLATILITY
LIQUIDITY_EVENT
NEWS_EVENT
UNSTABLE
NO_TRADE

```

Inputs shall include:

```
ATR
ATR percentile
ADX
EMA slope
Realized volatility
Bollinger width
Structure
Displacement
Volume
Spread
Session
Liquidity
News
Macro

```

---


# 11A. Gold Volatility and Transaction-Cost Regime

Do not model XAUUSD volatility with a single fixed ATR threshold.

Maintain normalized and absolute measures:

```text
ATR by timeframe
ATR percentile by session/regime
realized volatility
range percentile
Bollinger-width percentile
spread absolute
spread percentile
spread / ATR ratio
slippage percentile
gap/jump frequency
```

Initial example bands from research may be seeded in configuration, but production classification shall be driven by historical percentiles and broker/session context.

Per strategy, define:

```text
min_volatility
max_volatility
max_spread_absolute
max_spread_to_atr
max_expected_slippage
max_total_cost_to_target
```

The **stricter** applicable cost/volatility gate wins.

Ultra-scalping shall be disabled when market movement is too small to cover estimated execution cost or so unstable that its validated execution envelope is exceeded.

Trend-swing shall not manufacture a trend signal in a statistically dead range merely because slow indicators are aligned.

# 12. Quantitative Market Engines

Maintain production implementations for:

## Market Structure

```
HH
HL
LH
LL
BOS
CHoCH
MSS
Internal Structure
External Structure
Displacement
Failure

```

## Liquidity

```
Equal Highs
Equal Lows
Previous Day H/L
Previous Week H/L
Session H/L
Liquidity Sweep
Liquidity Grab
Stop Run
Failed Breakout
Sweep Rejection
Sweep Displacement

```

## FVG

```
Bullish
Bearish
Fresh
Partially filled
Mitigated
Invalidated

```

## Order Blocks

```
Bullish
Bearish
Breaker
Fresh
Mitigated
Invalidated

```

## VWAP

```
Daily
Session
Anchored
Deviation bands
Slope
Reclaim
Rejection
Cross

```

## Volume Profile

```
POC
VAH
VAL
HVN
LVN
Acceptance
Rejection
Value Migration

```

## Technical

```
EMA
SMA
RSI
Divergence
MACD
Stochastic
CCI
ADX
ATR
Bollinger Bands
Realized Volatility

```

Indicators provide evidence.

**No single indicator shall independently authorize a trade.**

---


# 12A. Four Canonical XAUUSD Strategy Playbooks

The four commercial strategies are separate quantitative products. They must not be aliases pointing to one generic signal.

Canonical IDs:

```text
STANDARD_SCALPING
ULTRA_SCALPING
STANDARD_SWING
TREND_SWING
```

Every deployed strategy maps to an immutable/versioned definition:

```text
strategy_definition_id
strategy_id
version
status
prediction_target_id
timeframe_profile_id
session_profile_id
feature_profile_id
confluence_profile_id
risk_profile_id
execution_profile_id
calibration_profile_id
created_at
approved_at
approved_by
code_commit
```

Statuses:

```text
DRAFT
RESEARCH
BACKTESTED
OOS_VALIDATED
PAPER
SHADOW
APPROVED
ACTIVE
SUSPENDED
DEPRECATED
ROLLED_BACK
```

No strategy definition may self-promote.

## 12A.1 STANDARD_SCALPING — Initial Baseline

Purpose: selective intraday scalp, not high-frequency trading.

```text
Typical holding horizon:    1–15 minutes
Initial frequency target:   5–15 candidates/trades per week, not a quota
Primary entry timeframe:    M5
Execution timing:           M1/tick when healthy
HTF context:                H1 + M30
Preferred market windows:   validated London/NY high-liquidity windows
Initial minimum gross R:R:  1.20
Initial max spread:         $0.35, also subject to relative spread/ATR and cost gates
Initial signal TTL:         15 minutes
Initial Tier-1 news block:  30 minutes before/after, configurable
Initial stop model:         structure + volatility buffer around 1.0 ATR(M5)
Initial targets:            nearby intraday liquidity; TP1/TP2 by default
```

Mandatory concept classes for an `ACTIVE` definition:

```text
healthy data
allowed session
valid regime
HTF/MTF context
liquidity event or approved alternative setup
structure confirmation
positive cost-adjusted expectancy gate
hard risk pass
```

## 12A.2 ULTRA_SCALPING — Initial Baseline

Purpose: very short-duration, cost-sensitive microstructure trading. It is **not** HFT and must be disabled if the available data/execution path cannot support it.

```text
Typical holding horizon:    10 seconds–3 minutes
Initial frequency target:   15–40 candidates/trades per week, not a quota
Primary entry timeframe:    M1
Execution timing:           tick/sub-minute if authoritative
HTF context:                M15 + M5
Preferred windows:          validated London/NY liquidity windows
Initial minimum gross R:R:  1.00 only when calibrated probability and net expectancy justify it
Initial max spread:         $0.25 hard seed, plus stricter relative/cost gates
Initial signal TTL:         3 minutes
Initial Tier-1 news block:  60 minutes before/after
Tier-2 treatment:           configurable; default conservative
Initial stop model:         tight structure + volatility buffer around 0.7 ATR(M1)
Initial targets:            nearest micro-liquidity/imbalance; TP1 primary
```

Additional production requirements:

```text
realistic bid/ask cost model
slippage distribution
execution latency distribution
GC trade-flow capability for highest grades when strategy version requires it
strict duplicate/TTL protection
broker-specific stop/freeze validation
```

If required flow capability is absent, the strategy definition shall either:

```text
NO_TRADE
or
DEGRADE to a separately validated proxy-only variant with an explicit grade ceiling
```

Never silently substitute broker tick volume for exchange flow.

## 12A.3 STANDARD_SWING — Initial Baseline

```text
Typical holding horizon:    4–48 hours
Initial frequency target:   2–5 trades per week, not a quota
Primary entry timeframe:    M15/H1
HTF context:                H4 + D1
Allowed periods:            broad liquid market periods; reject broker rollover/illiquid state
Initial minimum gross R:R:  1.80 baseline; validate 1.80 vs stricter alternatives such as 2.00
Initial max spread:         $0.45 plus relative/cost gates
Initial signal TTL:         4 hours
Initial Tier-1 new-entry block: 15 minutes before/after, configurable
Initial stop model:         structure + volatility buffer around 1.5 ATR(H1)
Targets:                    HTF liquidity / prior day high-low / structure / profile; TP1–TP3
```

Macro, real-yield and higher-timeframe structure carry more weight than for scalp strategies.

## 12A.4 TREND_SWING — Initial Baseline

```text
Typical holding horizon:    1–10 days
Initial frequency target:   2–6 trades per month, not a quota
Primary entry timeframe:    H1/H4
HTF context:                D1 + W1
Entry restriction:          no new entry into validated Friday-close/illiquid risk window
Initial minimum gross R:R:  2.50
Initial max spread:         $0.50 plus relative/cost gates
Initial signal TTL:         24 hours
Initial stop model:         weekly/HTF structure + volatility buffer around 2.0 ATR(H4)
Targets:                    weekly/monthly liquidity, major structure, HTF volume profile
```

Slow context may include:

```text
real yields
DXY regime
COT
ETF/approved fund flow
central-bank demand
carry/swap cost
```

These data are context—not proof of direction.

## 12A.5 Strategy Frequency Is Not a Quota

The frequency ranges above are product/research expectations only.

The engine must never force a trade to meet a count.

```text
NO-TRADE > frequency target
```

when evidence or execution quality is insufficient.


# 12B. Versioned Indicator / Feature Parameter Registry

All strategy-critical feature parameters shall be data/configuration driven.

Key:

```text
symbol + strategy + timeframe + regime + broker_class + feature_version
```

Examples:

```text
EMA periods
RSI period
RSI divergence pivot strength
MACD parameters
stochastic parameters
CCI parameters
ADX period/threshold
ATR period
Bollinger period/deviation
VWAP anchor/bands
swing pivot strength
equal-high/low tolerance
BOS close/wick policy
CHoCH/MSS policy
FVG minimum size/fill threshold
OB validity/mitigation threshold
displacement threshold
liquidity freshness
CVD divergence window
volume-profile value-area percentage
```

Do not encode a statement such as “RSI 7 is correct for ultra scalping” as an immutable rule. A value such as 7 may be a seed candidate in the `ULTRA_SCALPING` research profile and must compete against alternatives under the validation policy.

Parameter changes create a new profile version. They do not rewrite historical signal snapshots.


# 12C. Strategy Confluence Engine

Each strategy defines:

```text
mandatory_pillars[]
optional_pillars[]
weights
minimum_score
minimum_long_short_separation
minimum_confluence_count
maximum_missing_optional_weight
grade_ceiling_by_data_capability
```

Scoring shall be deterministic from the stored feature snapshot. Scores are **not probabilities**.

## 12C.1 Initial Seed Weight Matrices

These matrices are implementation baselines to be validated and versioned.

### STANDARD_SCALPING

| Evidence Group | Seed Weight |
|---|---:|
| Liquidity sweep/rejection or approved liquidity trigger | 25 |
| Structure + MTF alignment | 20 |
| FVG/OB/IFVG/location + VWAP | 15 |
| Flow/volume evidence | 20 |
| Regime/volatility | 10 |
| Macro/news alignment | 10 |

Seed candidate threshold: `75/100`.  
Seed long-short separation: `20`.  
Liquidity context is mandatory unless a separately validated setup family explicitly replaces it.

### ULTRA_SCALPING

| Evidence Group | Seed Weight |
|---|---:|
| Flow/microstructure | 30 |
| Liquidity event | 25 |
| Structure/MTF | 15 |
| Imbalance/FVG/VWAP location | 15 |
| Execution-cost quality | 10 |
| Macro/news state | 5 |

Seed candidate threshold: `85/100`.  
Seed long-short separation: `25`.  
Execution-cost quality is mandatory. Flow capability is mandatory for any strategy version that claims exchange-flow confirmation.

### STANDARD_SWING

| Evidence Group | Seed Weight |
|---|---:|
| D1/H4 structure | 20 |
| HTF liquidity/location | 15 |
| Macro / DXY / real yield | 20 |
| MTF trend alignment | 15 |
| Volume profile / flow | 10 |
| Regime / volatility | 10 |
| Risk/reward / carry / execution cost | 10 |

Seed candidate threshold: `70/100`.  
Seed long-short separation: `15`.

### TREND_SWING

| Evidence Group | Seed Weight |
|---|---:|
| W1/D1/H4 trend and structure | 25 |
| Macro / real yield / DXY | 20 |
| COT / ETF / slow flow | 15 |
| MTF alignment | 15 |
| Major liquidity / HTF profile | 10 |
| Trend persistence / volatility | 10 |
| Carry / execution cost | 5 |

Seed candidate threshold: `75/100`.  
Seed long-short separation: `15`.

## 12C.2 Hard Gates Override Score

A 95/100 score must still produce `NO-TRADE` if any hard gate fails:

```text
DATA_QUALITY
SESSION
NEWS
SPREAD
SLIPPAGE
TOTAL_COST
RISK
BROKER
MARGIN
LICENSE
ENTITLEMENT
EXECUTION_PERMISSION
EMERGENCY_STOP
```

## 12C.3 Evidence Contribution Contract

Store for each signal:

```text
pillar
feature
raw_value
normalized_value
direction
weight
contribution
mandatory
quality
source
version
reason_code
```

This allows exact explainability and replay.


# 12D. XAUUSD Liquidity, SMC/ICT-Style Feature Engine

These concepts are quantitative features to test—not market doctrine.

Maintain explicit buy-side and sell-side liquidity pools.

Pool types may include:

```text
BSL
SSL
EQUAL_HIGHS
EQUAL_LOWS
PREVIOUS_DAY_HIGH
PREVIOUS_DAY_LOW
PREVIOUS_WEEK_HIGH
PREVIOUS_WEEK_LOW
PREVIOUS_MONTH_HIGH
PREVIOUS_MONTH_LOW
SESSION_HIGH
SESSION_LOW
ASIA_RANGE_HIGH
ASIA_RANGE_LOW
MIDNIGHT_OPEN
SWING_HIGH
SWING_LOW
VOLUME_PROFILE_LEVEL
```

Tables:

```text
liquidity_pools
sweep_events
structure_events
fvg_zones
order_blocks
breaker_blocks
liquidity_voids
session_levels
```

`liquidity_pools` minimum fields:

```text
pool_id
symbol
type
side
price
price_tolerance
strength
timeframe
session
created_at
last_touched_at
swept_at
mitigated_at
invalidated_at
status
source_snapshot_id
feature_version
```

`sweep_events` minimum fields:

```text
sweep_id
pool_id
event_time
extreme_price
penetration_distance
close_back_distance
rejection_wick_ratio
displacement_after
volume_or_flow_confirmation
bos_after
quality
```

## 12D.1 Quantified Definitions

Do not rely on visual adjectives.

Thresholds shall preferably scale with:

```text
tick_size
ATR
realized volatility
session percentile
spread
```

Examples:

```text
equal_level_tolerance = max(N ticks, K × ATR)
displacement = body_ratio + range/ATR + optional flow z-score
sweep = penetration beyond pool + close/rejection policy + follow-through window
```

`N` and `K` are versioned parameters.

## 12D.2 BOS / CHoCH / MSS

Support both wick and close/body observations but make the strategy policy explicit:

```text
WICK_BREAK
CLOSE_BREAK
BODY_CONFIRMATION
```

The engine shall record all raw events and let the strategy profile decide which constitute confirmation.

## 12D.3 FVG / IFVG / Liquidity Void

Represent gap state continuously:

```text
FRESH
PARTIAL
MITIGATED
INVERTED
INVALIDATED
```

Store `fill_ratio` rather than hiding the threshold. A strategy profile selects its acceptable maximum fill ratio.

## 12D.4 Order Block / Breaker Block

An order block must have an objective creation/validation rule linked to:

```text
swing context
displacement
structure break
mitigation
invalidation
age/freshness
```

A breaker is a separately typed state, not merely an order block with a different label.

## 12D.5 Premium / Discount and OTE-Like Zones

Support configurable range-relative levels such as:

```text
0.50
0.62
0.79
```

as research features. They must not independently authorize a trade.

## 12D.6 AMD / Power-of-Three / Judas-Swing Research Features

Accumulation-Manipulation-Distribution, Power-of-Three, Judas-swing and similar concepts may be implemented only as:

```text
RESEARCH_ONLY
SHADOW
or approved SIGNAL_CONTRIBUTOR
```

until reproducible event definitions and independent validation exist.


# 12E. Order-Flow, Volume-Profile and VWAP Rules

When data capability exists, calculate:

```text
CVD by bar/session
price-vs-CVD divergence
delta percentile
stacked imbalance
aggressive/passive imbalance
absorption candidate
exhaustion candidate
session volume profile
daily volume profile
POC
VAH
VAL
HVN
LVN
VWAP
anchored VWAP
VWAP standard-deviation bands
```

Examples of strategy evidence to validate:

```text
price sweeps BSL + CVD fails to confirm + bearish displacement
price trades beyond VWAP 2σ + reclaims VWAP band
acceptance above VAH + positive delta + aligned structure
POC migration in trend direction
```

These are feature hypotheses. The engine shall measure their conditional hit rate by:

```text
strategy
session
regime
volatility bucket
news state
broker/cost bucket
```

No feature receives a permanent positive weight merely because it is popular in discretionary trading.

# 12F. Research-Derived Candle Patterns and Strategy Variant Registry

The following concepts are valid **research candidates** derived from the supporting XAUUSD execution research. They must be implemented as versioned, testable strategy variants rather than as discretionary prose or guaranteed edges.

Maintain a `strategy_variant_definition` registry with at minimum:

```text
variant_id
strategy_id
version
name
status
required_capabilities[]
required_features[]
entry_rule_expression
invalidation_rule_expression
target_rule_expression
session_profile_id
risk_profile_id
execution_profile_id
prediction_target_id
minimum_data_quality
grade_ceiling
created_at
approved_at
approved_by
code_commit
```

Initial research candidates include:

```text
ULTRA_SCALPING:
- DOM_CONSUMPTION_BURST
- CVD_VWAP_DIVERGENCE
- MICRO_LIQUIDITY_SWEEP_CHOCH

STANDARD_SCALPING:
- SESSION_HIGH_LOW_SWEEP_REVERSAL
- M5_FVG_REVERSION
- SESSION_MOMENTUM_TRANSITION
- OVERLAP_BREAKOUT_EXTENSION
- VWAP_FVG_PULLBACK

STANDARD_SWING:
- H4_PROFILE_H1_STRUCTURE_SHIFT
- HTF_FVG_ENGULFING
- WEEKLY_RANGE_EXPANSION_RESEARCH

TREND_SWING:
- D1_W1_MACRO_TREND_PULLBACK
- HTF_FVG_ENGULFING
- WEEKLY_ORDER_BLOCK_CONTINUATION
```

No variant may be activated merely because it appears in research material.

## 12F.1 Candle Pattern Engine

Add deterministic candle-pattern features where useful:

```text
BULLISH_ENGULFING
BEARISH_ENGULFING
REJECTION_WICK
FULL_BODY_DISPLACEMENT
INSIDE_BAR
OUTSIDE_BAR
PIN_BAR_CANDIDATE
GAP_OR_IMBALANCE_CONTEXT
```

Each pattern must have numeric definitions based on body/range/wick ratios, ATR/tick normalization, and timeframe context. Visual labels alone are insufficient.

A pattern is evidence, not authorization.

## 12F.2 200-EMA and Long-Horizon Trend Filter

Support a configurable 200-period EMA candidate for swing/trend research.

Never encode:

```text
price above D1 EMA200 -> shorts forbidden forever
price below D1 EMA200 -> longs forbidden forever
```

Instead store and validate:

```text
ema_period
ema_timeframe
price_distance_from_ema
ema_slope
ema_slope_normalized
regime
strategy
conditional expectancy
```

The active strategy profile decides whether EMA200 is mandatory, optional, weighted, or ignored.

## 12F.3 Seed Target Ranges Are Research Metadata

Any pip/price target ranges from supporting research shall be retained only as seed metadata for experiment design.

Production targets must be generated from the approved strategy target model using:

```text
market structure
liquidity objective
ATR / realized volatility
spread
expected slippage
commission
carry where applicable
prediction horizon
broker specification
minimum net expectancy
```

A fixed target range must never override a better structural/risk decision.


# 13. Macro Intelligence

Implement independent components for:

```
DXY
Treasury yields
Real yields
Economic calendar
Macro releases
Central-bank events
Geopolitical events
Session context

```

Macro intelligence returns structured states rather than arbitrary prose.

Example:

```
{
  "dxy_bias": "BEARISH",
  "us10y_bias": "BEARISH",
  "news_risk": "LOW",
  "macro_gold_bias": "BULLISH"
}

```

---


# 13A. Gold-Specific Macro Scoring and Event Surprise

The macro engine shall emit structured components, not a single opaque “gold bullish” sentence.

Potential components:

```text
dxy_score
real_yield_score
nominal_yield_score
risk_appetite_score
event_surprise_score
fed_policy_score
cot_score
etf_flow_score
central_bank_flow_score
geopolitical_risk_score
```

Normalize each component to a documented range and preserve its source, freshness and confidence.

An initial aggregate may use a range such as `-40..+40`, but the component weights and any filter threshold must be versioned and empirically validated.

## 13A.1 Correlation Is Dynamic

Measure rolling relationships between XAUUSD and:

```text
DXY
US real yields
nominal Treasury yields
GC futures
XAGUSD
selected risk assets
```

Store:

```text
window
correlation
confidence/sample_size
stability
regime
```

Do not hard-code an assumed correlation such as “DXY is always -0.85.”

## 13A.2 Economic Surprise

For releases with forecast/actual data calculate a reproducible surprise feature, for example:

```text
raw_surprise = actual - consensus
normalized_surprise = raw_surprise / historical_surprise_scale
```

Account for:

```text
units
revisions
directionality
release history
provider corrections
```

NLP may explain the release but shall not replace authoritative numeric values.

## 13A.3 Slow Context

COT is weekly positioning data and is suitable as swing/trend context, not an ultra-scalping trigger.

ETF holdings and central-bank demand are slow-moving context with their own freshness thresholds.

# 14. News Blackout Engine

Support configurable:

```
pre_event_minutes
event_duration
post_event_minutes
event_severity
strategy
account
risk_mode

```

Risk modes:

```
NO_TRADE
REDUCED_RISK
NORMAL

```

High-impact news must default to safe behavior unless a dedicated news strategy has independently passed production validation.

---


# 14A. XAUUSD Session, Fix, Holiday and Killzone Engine

Session logic shall be authoritative, timezone-aware and DST-aware.

Never encode London/US market windows permanently as fixed UTC offsets.

Use IANA timezone identifiers and official/provider calendars.

Maintain:

```text
session_definitions
market_calendars
holiday_calendars
gold_fix_windows
liquidity_windows
broker_trading_sessions
rollover_windows
```

A session record shall include:

```text
session_id
name
timezone
local_start
local_end
calendar_id
dst_policy
holiday_policy
allowed_strategies[]
volatility_profile
liquidity_profile
spread_multiplier
```

## 14A.1 LBMA Fix Windows

Support the LBMA Gold Price benchmark windows as London-local events, including the twice-daily benchmark times. Convert to UTC at runtime using the correct London timezone/DST rule.

Treat them as liquidity/event windows, not proof of reversal.

## 14A.2 CME/COMEX Gold Availability vs US Liquidity Windows

Separate:

```text
exchange trading availability
daily maintenance/break
US data-release window
COMEX/US liquidity-event windows
NY cash-session context
```

Do not model the entire GC trading day as a single hard-coded “COMEX 13:30–20:00 GMT” interval.

## 14A.3 Research Killzones

London-open, New-York-open, London-close, “Silver Bullet,” Power-of-Three and similar windows may be represented as versioned research/session features.

They become production filters only after:

```text
precise timezone definition
DST handling
historical event study
OOS validation
cost-adjusted validation
approval
```

## 14A.4 Day-of-Week Claims

Statements such as “gold trends Tuesday–Thursday” or “Monday/Friday mean-revert” are hypotheses, not hard rules.

Implement day-of-week as a categorical feature and validate it. Do not turn it into a production veto without evidence.

## 14A.5 Rollover / Illiquidity

Broker-specific rollover/trading-break windows shall come from the broker profile/terminal specification.

A conservative global fallback may exist but must be clearly marked `FALLBACK`, and broker-discovered session data takes priority.

## 14A.6 Initial Strategy News Defaults

Seed configuration:

```text
ULTRA_SCALPING:
  Tier 1: block new signals ±60m
  Tier 2: conservative block/reduced-risk according to validated profile

STANDARD_SCALPING:
  Tier 1: block new signals ±30m

STANDARD_SWING:
  Tier 1: block new entries ±15m

TREND_SWING:
  Tier 1: block new entries/require explicit hold-risk policy
```

Existing positions are not forcibly closed merely because a news window begins unless the active, validated position-risk policy explicitly requires it.

Pre-event spread/liquidity deterioration can trigger an earlier `NO-TRADE` based on relative spread/volatility and feed-quality gates rather than a universal fixed number.

# 15. Prediction Contract

Every prediction must explicitly define **what is being predicted**.

Do not use ambiguous “accuracy.”

Each prediction shall include:

```
prediction_id
signal_id
symbol
direction
prediction_horizon
timeframe
entry_model
target_definition
invalidation_definition
raw_score
calibrated_probability
expected_move
expected_rr
confidence_band
model_version
strategy_version
feature_snapshot_id
created_at
expires_at

```

Examples of independently measurable targets:

```
P(TP1 before SL)
P(TP2 before SL)
P(TP3 before SL)
P(close > entry after N minutes)
P(close < entry after N minutes)
Expected MFE
Expected MAE
Expected return over horizon

```

This definition becomes the basis of all accuracy claims.

---


# 15A. Strategy-Specific Prediction Targets

Calibration is meaningless unless each strategy has a precise label.

Each active strategy shall define one or more `prediction_target` records.

### STANDARD_SCALPING

Primary candidate:

```text
P(TP1 before SL within signal/position horizon)
```

Secondary:

```text
expected net R after estimated spread/slippage/commission
MFE
MAE
```

### ULTRA_SCALPING

Primary:

```text
P(TP1 before SL within ultra horizon)
```

Must include cost-sensitive outcome labeling using bid/ask and latency assumptions.

### STANDARD_SWING

Independently measure:

```text
P(TP1 before SL)
P(TP2 before SL)
P(TP3 before SL)
expected return within 48h horizon
MFE / MAE
```

### TREND_SWING

Independently measure:

```text
P(target before invalidation within configured multi-day horizon)
expected net return
MFE / MAE
time-to-target
carry/swap effect
```

A model calibrated for one target/horizon must not be presented as calibrated for another.

# 16. Signal Scoring

Maintain:

```
LONG_SCORE
SHORT_SCORE

```

Normalized to:

```
0–100

```

A score is **not automatically a probability**.

Store separately:

```
raw_score
calibrated_probability

```

Final decision must consider:

```
Score
Score separation
Regime
Structure
Liquidity
Volatility
Volume
VWAP
Macro
News
Data quality
Spread
Slippage
Risk/reward
Execution availability
License / account permissions

```

---

# 17. Signal Quality Grades

Support:

```
A+
A
B
C
NO-TRADE

```

The eligible minimum grade must be configurable.

Do not assume a grade implies a fixed probability without calibration.

---


# 17A. Grade and Probability Governance

Grades are presentation/policy labels derived **after** calibration and sample-sufficiency checks.

Do not hard-code:

```text
A+ = fixed probability
A  = fixed probability
B  = fixed probability
```

Instead maintain a versioned grade policy:

```text
grade_policy_id
strategy_id
prediction_target_id
calibration_version
minimum_sample_size
probability_bin
confidence_interval_policy
minimum_expectancy
maximum_drawdown_condition
grade
effective_from
```

Before sufficient validation, a signal shall use:

```text
RESEARCH
UNRATED
or SHADOW
```

rather than a misleading A+/A probability grade.

The gap-analysis example probability ranges may be used as research hypotheses only and must not be published or enabled until observed calibration supports them.

# 18. NO-TRADE Reasons

Standardized machine-readable reasons shall include:

```
INSUFFICIENT_SCORE
CONFLICTING_TIMEFRAMES
HIGH_NEWS_RISK
EXTREME_VOLATILITY
LOW_LIQUIDITY
HIGH_SPREAD
POOR_RR
UNCLEAR_STRUCTURE
DXY_CONFLICT
YIELD_CONFLICT
STALE_DATA
FEED_DEGRADED
AI_DISAGREEMENT
SIGNAL_EXPIRED
SESSION_UNSUITABLE
EXECUTION_UNAVAILABLE
BROKER_UNAVAILABLE
RISK_LIMIT_REACHED
LICENSE_RESTRICTED
ACCOUNT_NOT_AUTHORIZED
SYSTEM_DEGRADED

```

---

# 19. Signal Lifecycle

```
DETECTED
VALIDATING
CANDIDATE
AI_VERIFYING
CALIBRATING
RISK_CHECK
CONFIRMED
ACTIVE
TRIGGERED
EXPIRED
INVALIDATED
CANCELLED
ORDER_SENT
ACKNOWLEDGED
FILLED
PARTIALLY_FILLED
TP1
TP2
TP3
STOPPED
CLOSED

```

Every transition shall be timestamped.

---

# 20. AI Verification

AI receives structured data.

It must not be asked to infer numeric indicator values from screenshots when authoritative structured values already exist.

Return structured output such as:

```
{
  "decision": "SUPPORT",
  "confidence": 0.74,
  "risk_flags": [],
  "reason_codes": [
    "STRUCTURE_CONFIRMED",
    "MACRO_SUPPORTIVE"
  ]
}

```

Possible policies on AI failure:

```
ALLOW_QUANT_ONLY
DEGRADE_CONFIDENCE
REJECT_SIGNAL

```

---

# 21. Model Governance

Every production model shall have:

```
model_id
model_version
training_dataset
training_period
feature_version
code_commit
parameters
validation_report
approval_status
approved_by
approved_at
deployment_date
rollback_version

```

Support:

```
DRAFT
VALIDATING
SHADOW
APPROVED
ACTIVE
DEPRECATED
ROLLED_BACK

```

No model shall self-promote into production.

---

# 22. Drift Monitoring

Monitor:

```
Feature drift
Prediction drift
Calibration drift
Win-rate drift
Expectancy drift
Regime drift
Slippage drift
Broker-quality drift
Latency drift

```

Drift must trigger review rather than uncontrolled automatic re-optimization.

---

# 23. Experimental / Alternative Intelligence Pillars

Any Vedic, Astro/KP, celestial-cycle or other non-mainstream intelligence component already used by Predict-A-Trade shall remain:

- clearly isolated;
- feature flagged;
- auditable;
- independently scored;
- independently backtested;
- independently calibrated;
- unable to override hard risk rules;
- unable to claim statistical effectiveness without reproducible validation.

Support:

```
DISABLED
RESEARCH_ONLY
SHADOW
SIGNAL_CONTRIBUTOR

```

Promotion from one mode to another requires empirical validation and administrative approval.

---

# 24. Master Decision Hierarchy

```
1. DATA QUALITY
2. MARKET STATE
3. MARKET REGIME
4. STRUCTURE
5. LIQUIDITY
6. VOLUME / VWAP / VOLATILITY
7. MACRO
8. MULTI-TIMEFRAME ALIGNMENT
9. EXISTING PILLARS
10. QUANTITATIVE SCORE
11. AI VERIFICATION
12. CALIBRATION
13. HARD RISK
14. EXECUTION CONDITIONS
15. ENTITLEMENTS / LICENSE
16. MASTER DECISION
17. BUY / SELL / NO-TRADE

```

---

# 25. Risk Engine

Before execution validate:

```
risk_per_trade
position_size
maximum_exposure
maximum_daily_loss
maximum_drawdown
maximum_positions
maximum_symbol_exposure
maximum_spread
maximum_slippage
maximum_trade_frequency
cooldown
session
news
volatility
account permissions
broker restrictions
license entitlements
execution mode

```

Mandatory rule failure:

```
EXECUTION = DENIED

```

and where applicable:

```
FINAL DECISION = NO-TRADE

```

---


# 25A. Strategy Risk Profiles and Cost-Aware Hard Vetoes

Every strategy version links to a versioned `risk_profile_id`.

Minimum fields:

```text
risk_per_trade_policy
maximum_daily_loss
maximum_drawdown
maximum_concurrent_xau_exposure
maximum_positions
maximum_new_trades_per_day
loss_cooldown
minimum_gross_rr
minimum_net_expectancy
maximum_spread_absolute
maximum_spread_to_atr
maximum_expected_slippage
maximum_total_cost_to_target
minimum_margin_headroom
allowed_sessions
allowed_regimes
news_policy
weekend_policy
carry_policy
correlation_policy
```

## 25A.1 Initial Seed Profiles

These are conservative starting configurations for research/shadow and require approval before live use.

| Strategy | Seed Min Gross R:R | Seed Max Spread | Seed New Trades/Day | Seed Loss Cooldown |
|---|---:|---:|---:|---:|
| STANDARD_SCALPING | 1.20 | $0.35 | 3 | 30m |
| ULTRA_SCALPING | 1.00 only with validated positive net expectancy | $0.25 | 5 | 30m |
| STANDARD_SWING | 1.80 | $0.45 | 2 | strategy-configured |
| TREND_SWING | 2.50 | $0.50 | 1 | strategy-configured |

The table is not a profitability promise. Validation may tighten these values.

## 25A.2 Transaction-Cost Gate

Estimate before order authorization:

```text
spread_cost
commission
expected_slippage
financing/carry when relevant
latency_cost_estimate
total_expected_cost
```

Require:

```text
expected_edge_after_costs > approved minimum
```

and:

```text
total_cost_to_target_ratio <= approved maximum
```

## 25A.3 Aggregated XAUUSD Exposure

Risk is account-level, not strategy-label-level.

Two strategies must not bypass exposure limits by opening equivalent XAUUSD direction risk independently.

Aggregate:

```text
symbol exposure
direction exposure
notional
margin
stop-risk
correlated synthetic exposure where configured
```

before authorizing a new order.

## 25A.4 Margin and Broker Specification

Use live/broker profile values for:

```text
contract size
tick value
margin requirement
currency conversion
minimum lot
lot step
stops/freeze
```

If broker metadata is missing or inconsistent, automated execution is denied.

# 25B. Broker-Aware Price Units, Position Sizing, Margin and Stop-Out Safety

Gold CFD quoting conventions vary by broker. The production system must never rely on informal “pip” language for risk-critical math.

## 25B.1 Canonical Internal Units

Use canonical internal quantities:

```text
price_distance
point_size
tick_size
tick_value
contract_size
lots
notional
account_currency_value
```

If the UI displays pips/points, the label and conversion must come from the broker execution profile.

Store:

```text
broker_point_size
broker_tick_size
display_point_definition
display_pip_definition
conversion_version
```

Never hard-code globally:

```text
1 pip = $0.10
1 point = $0.01
1 lot = 100 oz
```

Such conventions may be true for a particular broker profile but are not platform-wide truths.

## 25B.2 Generalized Risk-Based Position Sizing

Position sizing shall use broker-reported or validated symbol economics.

Preferred generalized calculation:

```text
risk_budget_account_ccy =
    risk_basis * approved_risk_fraction

loss_per_lot_account_ccy =
    stop_distance_in_ticks
    * tick_value_per_lot_in_account_ccy
    + expected_entry_cost_per_lot
    + expected_exit_cost_per_lot
    + relevant_financing_buffer

raw_lots =
    risk_budget_account_ccy / loss_per_lot_account_ccy

approved_lots =
    floor_to_lot_step(
        min(raw_lots, broker_max_lot, risk_max_lot)
    )
```

Where tick value is unavailable or unreliable, calculate from validated contract specifications and currency conversion, then cross-check against terminal-reported values.

Reject automatic execution when the two methods materially disagree.

## 25B.3 Margin and Stop-Out

Before order authorization calculate or retrieve:

```text
required_margin
free_margin_after_order
margin_level_after_order
broker_margin_call_level
broker_stop_out_level
stress_spread
stress_slippage
stress_margin_level
```

Require an approved `minimum_margin_headroom`.

High leverage must **not** be described as inherently protective. It may reduce initial margin on some broker/account structures but also enables larger exposure; risk remains controlled by position sizing, aggregate exposure, broker margin rules and hard loss limits.

## 25B.4 Carry / Swap

For positions that may cross rollover, include:

```text
swap_long
swap_short
swap_method
triple_swap_rule
holding_horizon
expected_carry_cost
```

Swap-free or alternative account structures may be supported as broker/account metadata, but the platform must not assume a “swap-free” label means zero economic cost; validate the broker’s actual fee structure.


# 26. Execution Modes

Support separate entitlements and runtime modes:

```
SIGNAL_ONLY
MANUAL_EXECUTION
ASSISTED_EXECUTION
AUTO_EXECUTION
PAPER
SHADOW
EMERGENCY_STOP

```

Receiving a signal must never automatically imply permission for autonomous execution.

---

# 26A. Order-Type Policy, Fill Risk and Execution Locality

## 26A.1 Order Types

Support strategy-specific authorization for:

```text
MARKET
LIMIT
STOP
STOP_LIMIT where supported
```

The execution profile shall define:

```text
allowed_order_types[]
preferred_order_type
max_order_age
max_entry_deviation
max_slippage
max_queue_wait
partial_fill_policy
cancel_replace_policy
fallback_policy
```

Do not state that limit orders “guarantee exact entry” or “eliminate slippage.” A limit order constrains the worst acceptable price but can remain unfilled, partially fill, lose queue priority, or behave differently under gaps/venue rules.

The backtest/replay engine must model missed fills and partial fills where relevant.

## 26A.2 Market-Order Safety

Market orders require:

```text
fresh quote
spread gate
slippage estimate
latency gate
market state gate
broker fill-mode compatibility
maximum deviation policy where supported
hard SL/TP attachment or immediate protective-order workflow
```

Protective risk orders must be attached atomically when the broker/API supports it; otherwise the system must use the safest broker-supported sequence and measure the temporary unprotected interval.

## 26A.3 Execution Locality / VPS Qualification

Ultra-short strategies require an execution path qualified by measurement, not by geographic marketing claims.

Maintain an `execution_locality_profile`:

```text
client_region
terminal_host_region
broker_server
broker_server_region_if_known
network_rtt_p50
network_rtt_p95
network_jitter_p95
packet_loss
agent_to_terminal_latency_p95
terminal_to_broker_ack_p50
terminal_to_broker_ack_p95
signal_to_order_p95
order_to_fill_p95
last_measured_at
quality_state
```

A VPS or colocated host may be recommended when it materially improves measured execution quality, but no location or ping value guarantees a fill price.

For `ULTRA_SCALPING`, the active execution profile shall define maximum permitted latency/jitter/slippage envelopes. If the measured path is outside the validated envelope:

```text
EXECUTION_DENIED
EXECUTION_QUALITY_INSUFFICIENT
```

or downgrade to signal-only according to policy.

## 26A.4 Broker/Account Class Qualification

Do not require a marketing label such as “Raw” or “Zero Spread” as a universal rule.

Instead qualify accounts using measured economics:

```text
commission_round_turn
spread_p50
spread_p95
slippage_p50
slippage_p95
rejection_rate
fill_rate
partial_fill_rate
latency
swap/carry
minimum_stops
execution mode
```

A broker/account is eligible for a strategy only when the complete validated cost/execution envelope passes.


# 27. Entry, Stop and Take-Profit Engine

Entry:

```
MARKET
LIMIT
BREAKOUT
RETEST
PULLBACK
LIQUIDITY_SWEEP
FVG_RETEST
ORDER_BLOCK_RETEST
VWAP_RECLAIM

```

Stop:

```
STRUCTURE
ATR
LIQUIDITY
SWING
FVG
ORDER_BLOCK
HYBRID

```

Targets:

```
TP1
TP2
TP3

```

Targets can derive from:

```
Liquidity
Previous H/L
VWAP
Volume Profile
Fibonacci
Session levels
Structure

```

---


# 27A. Strategy Definition, Entitlement and Execution Separation

Three distinct objects shall exist:

```text
commercial_entitlement
strategy_definition
execution_permission
```

They shall never be collapsed into one boolean.

Example:

```text
User has ULTRA_SCALPING entitlement
+ ULTRA_SCALPING v3 is ACTIVE
+ account has AUTO_EXECUTION permission
+ live risk gate passes
= eligible for automated order authorization
```

Failure of any term prevents auto-execution.

A strategy definition contains:

```text
entry models
timeframe profile
session profile
feature profile
confluence profile
risk profile
execution profile
prediction target
calibration profile
```

Changing a subscription does not mutate historical strategy behavior. Changing a strategy version does not alter historical entitlements.

# 28. Real-Time Signal Distribution

Provide:

```
HTTPS REST
Secure WebSocket

```

Recommended channels:

```
/ws/xauusd
/ws/xauusd/signals
/ws/xauusd/market-state
/ws/xauusd/structure
/ws/xauusd/news
/ws/xauusd/execution

```

Connections must enforce:

```
Authentication
Entitlement
License state
Device state where applicable
Subscription
Connection limits
Rate limits

```

---

# 29. WebSocket Delivery Semantics

Every event should include:

```
event_id
sequence
type
created_at
server_time
payload_version

```

Support:

```
Heartbeats
Resume cursor
Reconnect
Missed-event replay
Duplicate suppression
Backpressure handling
Maximum queue depth
Slow-client detection

```

Signals must not silently disappear during brief reconnects.

Expired signals must never be replayed as executable signals.

---

# 30. NestJS Control Plane

NestJS becomes the authoritative application/control plane for:

```text
Identity
Roles
Organizations
Plans
Subscriptions
Billing
Strategy Entitlements
Licenses
Devices
Referral Attribution
Referral Relationships
Commission Rules
Commission Ledger
Affiliate Wallets
Payouts
Affiliate Risk / Fraud Review
Downloads
Client Releases
User Preferences
Admin Operations
Audit
Support
Legal Acceptance
```

It may aggregate read-only signal information for dashboards.

It must **not sit synchronously in every market tick → signal decision path**.

The referral/commission subsystem belongs to the SaaS/control plane and must react only to validated billing events. It shall never become a dependency of the Go real-time trading engine.

Recommended NestJS modules include:

```text
PlansModule
SubscriptionsModule
BillingModule
EntitlementsModule
LicensesModule
ReferralsModule
CommissionsModule
PayoutsModule
AffiliateRiskModule
FinanceReportingModule
AuditModule
NotificationsModule
```

Financial state transitions shall be transactional, idempotent, auditable and replay-safe.

---

# 31. Next.js Application Architecture

Provide three distinct logical experiences.

## Public Site

```
Marketing
Technology
Methodology
Verified performance
Pricing / access
Legal
Risk disclosure
Documentation
Login

```

## User Application

Authenticated customer experience.

## Administrative Application

Restricted operational command center.

Shared components may live in a common design system.

Administrative privilege boundaries must remain enforced server-side even when UI components are shared.

---

# 32. User Dashboard

The User Dashboard shall contain all existing trading, licensing and account functions plus the commercial/referral experience below.

## Overview

```text
Current XAUUSD Price
Session
Regime
Current Signal
Signal Grade
Calibrated Probability
News Risk
MT Connection
License Status
Subscription Status
Current Plan
Active Strategy Entitlements
Open Position Summary
```

## Live Signal Terminal

Display only strategies currently granted by the user's effective plan/license entitlements:

```text
BUY / SELL / NO-TRADE
Created Time
TTL countdown
Entry
Entry Zone
SL
TP1
TP2
TP3
R:R
Timeframe
Strategy
Grade
Score
Calibrated Probability
Regime
Session
News Risk
Invalidation
```

The UI must not merely hide unauthorized strategies. Server-side APIs and real-time gateways must enforce the same entitlement.

## Explainability

Show:

```text
Positive Evidence
Negative Evidence
Long Score
Short Score
Reason Codes
Pillar Contributions
AI Verification
Risk Decision
```

## Chart

Provide synchronized chart overlays for:

```text
Candles
Entry
SL
TP
Liquidity
FVG
Order Blocks
VWAP
Session levels
Signal markers
```

## Signal History

Filters:

```text
Date
Direction
Grade
Strategy
Timeframe
Outcome
Session
Regime
```

History access must respect the user's subscription/entitlement history and any commercial retention policy.

## Performance

Display:

```text
Total Signals
Executed Signals
Win Rate
Loss Rate
Profit Factor
Expectancy
Average R
Drawdown
MFE
MAE
TP1 / TP2 / TP3 Rate
Signal Latency
Execution Latency
```

Clearly distinguish platform signal performance from individual user/account trading performance.

## MT4/MT5

Provide:

```text
Download Agent
Download MT4 EA
Download MT5 EA
Setup Wizard
License Activation
Device Status
Broker
Account
Terminal Version
Agent Version
Connection Test
Heartbeat
Latency
```

## Subscription & Plan

Display:

```text
Plan Name
Setup Fee
Monthly Fee
Billing Interval
Subscription Status
Current Billing Period
Next Billing Date
Auto-Renew State
Payment History
Invoices / Receipts
Selected Strategies
Available Strategy Slots
Upgrade / Downgrade Options
Cancellation State
```

Initial production plans are defined in Section 69.

## License

Display:

```text
Plan
License ID
Status
Start Date
Expiry
Features
Allowed Devices
Active Devices
Allowed MT Accounts
Active Accounts
MT4 entitlement
MT5 entitlement
Execution entitlement
Strategy entitlements
```

## Referral / Affiliate Center

Every eligible user shall have a dedicated referral dashboard showing:

```text
Referral Code
Referral Link
Attribution Status
Direct Referrals
Network Size
Active Referred Subscribers
L1 / L2 / L3 / L4 / L5 Network Counts
First-Payment Conversions
Second-Payment Retention
Recurring-Payment Retention
Referral Revenue
Total Commission Earned
Pending Commission
Cleared Commission
Available Commission
Paid Commission
Reversed Commission
Earnings by Level
Earnings by Plan
Earnings by Purchase/Billing-Cycle Type
Recent Commission Ledger
Payout History
```

The dashboard must clearly explain the critical second-payment rule: the second eligible monthly subscription payment pays referral commission to **Level 1 only**.

The user must never be able to change sponsor relationships, commission rates, commission amounts, payment status or payout approval state through client-side parameters.

## Payouts

Provide:

```text
Available Balance
Minimum Withdrawal Threshold
Payout Method
Payout Eligibility State
Withdrawal Request
Withdrawal History
Requested
Under Review
Approved
Processing
Paid
Failed
Cancelled
```

Sensitive payout details must be tokenized/masked and never exposed unnecessarily.

## Security

```text
Password
MFA
Sessions
Trusted Devices
Login History
Revoke Session
```

## Notifications

User-configurable:

```text
A+ Signals
A Signals
Signal Expiry
TP
SL
News Blackout
Connection Failure
License Expiry
Subscription Renewal
Payment Failed
New Referral
Referral Converted
Commission Pending
Commission Available
Commission Reversed
Payout Status
Affiliate Hold
```

## Support

```text
Documentation
FAQ
Setup Guide
Billing Help
Referral Program Terms
Commission Explanation
Payout Help
Diagnostics
Support Ticket
Client Log Upload
```

---


# 32A. Production XAUUSD Chart and Replay Specification

The chart is a visualization of server-authoritative state. It must not recalculate production signal truth in the browser.

All overlays derive from:

```text
signal_snapshot
market_state snapshot
structure/liquidity state
approved indicator state
```

## 32A.1 Required Trading Views

Provide synchronized, switchable multi-timeframe charts appropriate to the selected strategy.

Support:

```text
linked crosshair/time
candles
bid/ask or spread indicator where useful
entry zone
SL
TP1/TP2/TP3
R:R
TTL
signal grade/probability
liquidity pools
sweep markers
BOS/CHoCH/MSS
FVG/IFVG
OB/breaker
VWAP/bands
session boxes
prior day/week/month levels
midnight open
POC/VAH/VAL
```

Optional, capability-gated:

```text
footprint
delta
CVD
depth/heatmap
```

## 32A.2 Persistent State

Zones/levels shall have state:

```text
FRESH
TOUCHED
PARTIAL
MITIGATED
SWEPT
INVALIDATED
EXPIRED
```

The state shown in UI must match the state used by the Go signal engine.

## 32A.3 Renderer Decision

During repository audit, evaluate:

```text
existing chart library
TradingView/licensing restrictions
custom canvas/WebGL solution
latency
historical replay needs
mobile behavior
data redistribution rights
```

Do not replace a working chart stack merely for novelty.

## 32A.4 Replay

Provide historical signal/bar replay for authorized users/admins/researchers:

```text
time-controlled playback
same strategy/config version
same feature snapshot where available
no future leakage
signal lifecycle replay
execution/cost replay
```

Replay is educational/research functionality; it must clearly distinguish simulated results from live results.

## 32A.5 Audit Export

Authorized users/admins may export a redacted signal snapshot/chart image for audit/support. The export must carry signal/config/version identifiers and must not expose secrets or downline/private data.

# 33. Admin Dashboard

The Admin Dashboard shall become the production operational and commercial command center.

## Overview

Display:

```text
Platform Health
Active Users
Active Subscribers
MRR
Setup Fees Collected
Online Windows Agents
MT4 Connections
MT5 Connections
Active Licenses
Signals Today
NO-TRADE Rate
Open Positions
Feed Health
AI Health
Database Health
Latency
Critical Alerts
Referral Revenue
Pending Commission Liability
Available Commission Liability
Payouts Pending
Fraud Holds
```

## Clients / Users

Admin shall:

```text
Create
Invite
View
Suspend
Reactivate
Reset MFA
End Sessions
Assign Role
Assign Plan
Review Subscription
Review Strategy Entitlements
Review Activity
Review MT Accounts
Review Devices
Review Sponsor / Referral Relationship
Review Commission History
Place / Release Affiliate Hold
```

Sponsor changes after attribution require privileged authorization, a mandatory reason, and immutable audit history.

## Plans & Entitlements

Admin shall manage the three production plans and any future plans through configuration, not hardcoded UI logic.

Controls include:

```text
Plan Code
Plan Name
One-Time Setup Fee
Monthly Price
Currency
Billing Interval
Allowed Strategy Set
Maximum Active Strategy Slots
Plan Status
Effective From / Until
Upgrade Rules
Downgrade Rules
Cancellation Rules
Grace Period
Entitlements
```

All pricing and entitlement changes must be versioned and effective-dated.

## Subscriptions & Billing

Admin shall view:

```text
Subscription ID
Customer
Plan
Status
Billing Period
Renewal Date
Invoices
Payments
Refunds
Chargebacks
Coupons / Credits
Payment Provider References
Purchase/Billing-Cycle Number
Commissionable Amount
```

Admin must not be able to mark a payment successful by merely editing client-visible state. Payment truth comes from the validated billing/payment subsystem.

## Referral Network

Admin shall provide:

```text
Affiliate Search
Referral Codes
Referral Attribution
Sponsor Chain
Downline Tree up to L5
Network Statistics
Clicks
Registrations
Verified Users
First Payments
Second Payments
Recurring Payments
Conversion Funnels
Referral Revenue
Suspicious Clusters
```

Provide graph/tree visualization with pagination and depth guards for large networks.

## Commission Control Center

Admin shall be able to configure future rules for:

```text
Plan-Level L1-L5 Base Rates
Purchase/Billing-Cycle Multipliers
Maximum Eligible Referral Level per Purchase Type
Commissionable Amount Policy
Holding Period
Minimum Withdrawal
Payout Schedule
Commission Caps
Affiliate Holds
Refund / Chargeback Handling
Effective Dates
```

Initial rules are locked to Section 69 until a privileged, audited change is approved.

Every rule change shall preserve historical rate snapshots; editing a rule must never rewrite historical commission ledger entries.

## Commission Operations

Admin can:

```text
Search Commission Ledger
Filter by User / Source User / Plan / Level / Billing Cycle
Inspect Rule Snapshot
Inspect Payment Source
Hold Commission
Release Commission
Reverse Commission
Create Audited Adjustment
Export Reconciliation Report
```

Manual adjustments must be additive ledger records, not silent edits.

## Payout Operations

Admin can:

```text
Review Withdrawal Requests
Run Risk Review
Approve
Reject with Reason
Process
Record Provider Reference
Reconcile Paid Payout
Retry Failed Payout
Cancel
Export Payout Report
```

High-risk payout actions require MFA/re-authentication and RBAC separation.

## Licenses

Admin shall:

```text
Create license
Assign owner
Set plan
Set expiry
Set grace period
Set maximum devices
Set maximum MT accounts
Enable MT4
Enable MT5
Enable signal-only
Enable assisted execution
Enable auto execution
Suspend
Revoke
Renew
Reset activation
Force logout
Review activation history
```

License strategy entitlements must be derived from the effective subscription/plan unless an explicitly audited override exists.

## Devices

Show:

```text
Device ID
Owner
Windows version
Agent version
First seen
Last seen
Last IP
Connection status
License
MT terminals
Security state
```

Allow:

```text
Revoke Device
Reset Device
Force Upgrade
Disable Signal Access
```

## MT Accounts

Display:

```text
Broker
Broker Server
Platform
Account Reference
Symbol Mapping
Trading Permission
Connection
Last Heartbeat
Open Positions
```

Do not store broker account passwords.

## Signals

Allow administrators to inspect:

```text
Full Signal Snapshot
Reason Codes
Feature Snapshot
Pillar Scores
AI Responses
Risk Decision
Recipients
Execution Commands
Outcome
Entitlement Decision
```

## Risk Center

Provide:

```text
Global Kill Switch
Strategy Kill Switch
Account Kill Switch
Broker Kill Switch
Symbol Kill Switch
Maximum Exposure
Maximum Spread
Maximum Slippage
Maximum Drawdown
Daily Loss
Trading Sessions
News Blackout
```

Destructive actions require re-authentication.

## Strategies

Provide:

```text
Version
State
Backtest
Walk Forward
Shadow Results
Production Status
Feature Flags
Rollout Percentage
Plan Availability
```

## AI Providers

Manage:

```text
Provider
Model
Health
Latency
Usage
Timeout
Fallback
Enabled State
```

Secrets must remain masked.

## Market Data

Display:

```text
Primary Feed
Secondary Feed
Divergence
Tick Rate
Latency
Last Tick
Candle Health
Missing Data
Backfill Status
```

## Macro / News

Display:

```text
DXY
Yields
Calendar
News Feed
Blackouts
Provider Health
```

## Infrastructure

Display:

```text
Services
Database
PgBouncer
Valkey
Storage
CPU
RAM
Disk
Connections
Queues
WebSockets
Windows Agents
Billing Event Consumer
Commission Worker
Payout Worker
```

## Finance / Referral Reports

Provide:

```text
Sales by Plan
MRR / Churn
Setup Fee Revenue
Referral-Attributed Revenue
Commission by Plan
Commission by Level
Commission by Billing Cycle Type
Pending / Cleared / Available / Paid / Reversed
Network Payout Ratio
Payout Reconciliation
Refund / Chargeback Impact
Affiliate Retention
Second-Payment Conversion
Recurring-Payment Conversion
```

## Audit

Search:

```text
Actor
Action
Entity
Timestamp
IP
Request ID
Old State
New State
Reason
```

Include pricing, referral, sponsor, commission, payout, refund and affiliate-hold actions.

## Releases

Admin can manage:

```text
Windows Agent Releases
MT4 EA Releases
MT5 EA Releases
Minimum Supported Version
Recommended Version
Mandatory Upgrade
Stable / Beta Channels
Rollback
Checksums
Signatures
```

---

# 34. Authentication

Support:

```
Email / username
Strong password authentication
MFA
Recovery codes
Session management
Password reset
Email verification
Login throttling
Account lock protection
Risk-based security events

```

Preferred MFA:

```
TOTP
WebAuthn / Passkeys where supported

```

Administrative accounts must require MFA.

---

# 35. Authorization

Implement RBAC.

Suggested roles:

```
SUPER_ADMIN
ADMIN
RISK_MANAGER
TRADING_OPERATOR
SUPPORT
ANALYST
AUDITOR
USER

```

Permissions must be granular.

Examples:

```
license:create
license:revoke
risk:modify
execution:stop
strategy:activate
user:suspend
audit:read
signal:read
device:revoke

```

Never rely solely on hidden UI controls for authorization.

---

# 36. Multi-Tenancy

Even if initially operated as a single organization, the schema shall support future tenancy.

Tenant-scoped objects include:

```
Users
Licenses
Devices
Plans
MT Accounts
Notifications
Support Tickets
API Keys
Preferences
Reports

```

Use application-layer authorization plus PostgreSQL row-level security where appropriate as defense in depth.

---

# 37. License Architecture

Licensing is mandatory for Windows MT4/MT5 clients.

Each license shall contain:

```
license_id
tenant_id
user_id
plan_id
status
issued_at
valid_from
expires_at
grace_until
max_devices
max_mt_accounts
allowed_features
allowed_execution_modes
created_by
revoked_at
revocation_reason

```

Status:

```
PENDING
ACTIVE
SUSPENDED
EXPIRED
REVOKED
GRACE

```

---

# 38. Entitlement Architecture

Do not hardcode plan names into trading, signal delivery or execution logic.

Use granular entitlements such as:

```text
signals.realtime
signals.history
signals.explainability
signals.news
signals.macro

strategy.standard_scalping
strategy.ultra_scalping
strategy.standard_swing
strategy.trend_swing
strategy.max_active_slots

client.windows
client.mt4
client.mt5

execution.manual
execution.assisted
execution.auto

devices.max
accounts.max

analytics.basic
analytics.advanced

notifications.realtime
api.access
```

Plans map to entitlements.

Subscriptions establish the commercial right to a plan.

Licenses materialize those rights for devices/accounts.

The signal and execution gateways evaluate the effective entitlement set on every protected request/stream.

Initial strategy entitlement rules:

| Plan | Strategy Entitlement |
|---|---|
| STANDARD | Exactly 1 active strategy, selected from Standard Scalping or Standard Swing |
| PRO | Any 2 active strategies selected from Standard Scalping, Ultra Scalping, Standard Swing, Trend Swing |
| ELITE | All 4 strategies active |

A strategy that is not entitled must not be distributed over REST, WebSocket, Windows Agent, MT4 or MT5.

Plan changes must propagate to license/entitlement leases without requiring destructive recreation of user or device records.

---

# 39. License Activation

Recommended activation flow:

```
1. User signs into User Dashboard
2. User receives/downloads licensed Windows Agent
3. Agent generates cryptographic device identity
4. Agent connects over TLS
5. User authenticates / enters activation code
6. Server validates license
7. Device public identity is registered
8. License-device binding is created
9. Server issues short-lived signed entitlement lease
10. Agent connects to real-time signal gateway
11. Go gateway verifies signed entitlement
12. Agent begins receiving permitted signals

```

Never depend exclusively on a changeable hardware fingerprint.

---

# 40. Cryptographic Device Identity

The Windows Agent shall generate a device key pair during enrollment.

Private key should be protected using appropriate Windows secure-storage facilities.

Where available, stronger hardware-backed storage may be used.

Server stores:

```
device_id
public_key
key_algorithm
created_at
last_rotated_at
revoked_at

```

Use challenge-response when appropriate.

---

# 41. Signed Entitlement Lease

To avoid making NestJS a single point of failure for real-time delivery:

```
NestJS License Service
        ↓
Signed Short-Lived Entitlement
        ↓
Windows Agent
        ↓
Go Signal Gateway

```

The Go gateway shall verify the signed entitlement locally using the issuer's public key.

This avoids querying the licensing database for every signal.

Entitlement should contain:

```
subject
license_id
device_id
plan
features
execution_modes
issued_at
expires_at
token_id
issuer
audience

```

Revocation shall be propagated rapidly.

Short token lifetime limits stale authorization.

---

# 42. Device Limits

Enforce:

```
Maximum registered devices
Maximum simultaneously connected devices
Maximum MT accounts
Maximum concurrent terminals

```

Admin can:

```
Revoke
Reset
Transfer
Suspend

```

device activations.

---

# 43. MT Account Binding

A license may optionally bind to approved MT4/MT5 accounts.

Store only required metadata:

```
account_reference
broker
broker_server
platform
currency
symbol_mapping
authorized
first_seen
last_seen

```

Do not store users' broker passwords in Predict-A-Trade.

---

# 44. Windows Trading Agent

Create a production Windows Service, preferably Golang.

Responsibilities:

```
Authentication
License activation
Device identity
Entitlement refresh
Secure WebSocket
Signal reception
Signature validation
TTL validation
Replay protection
Local signal cache
MT4 communication
MT5 communication
Terminal discovery
Heartbeat
Broker health
Local audit
Diagnostics
Update checking
Kill switch

```

Windows Service must automatically restart according to safe recovery policy.

---

# 45. MT4 / MT5 EA

Provide:

```
PredictATrade_MT4.mq4
PredictATrade_MT5.mq5

```

Responsibilities:

```
Receive signal from local Windows Agent
Display signal
Validate local session
Validate terminal/account state
Prevent duplicates
Check symbol mapping
Check trading permission
Place authorized orders
Apply SL
Apply TP
Modify position
Partial close
Full close
Report acknowledgement
Report broker errors
Heartbeat
Emergency stop

```

The EA does not directly connect to PostgreSQL or AI providers.

---

# 46. MT4/MT5 Signal-Only Mode

Users whose licenses contain only:

```
signals.realtime
client.mt4 / client.mt5

```

may receive:

```
BUY
SELL
NO-TRADE
Entry
SL
TP1
TP2
TP3
TTL
Grade
Probability
Explanation

```

but the EA shall not automatically submit orders.

---

# 47. MT4/MT5 EA Panel

Display a compact panel:

```
Predict-A-Trade
License: ACTIVE
Connection: ONLINE
Broker: Connected
Mode: SIGNAL ONLY / AUTO
Signal: BUY
Grade: A+
TTL: 04:32
Entry
SL
TP1
TP2
TP3
Latency
Last heartbeat

```

Avoid oversized fonts or obstructing the trading chart.

---

# 48. Local Windows-to-EA Communication

Use a local-only authenticated IPC mechanism.

Possible implementations:

```
Named Pipe
Loopback TCP
Loopback HTTP
ZeroMQ on loopback

```

Requirements:

- bind only to local machine;
- authenticate the local session;
- reject external interfaces;
- no permanent server secret inside MQL;
- no database credentials;
- no AI credentials;
- no master private keys.

---

# 49. Secure Signal Protocol

Every signal command shall include:

```
message_version
command_id
signal_id
license_id
device_id
account_id
symbol
direction
order_type
volume
entry
stop_loss
take_profit
strategy
issued_at
expires_at
nonce
signature

```

Reject:

```
Expired
Duplicate
Invalid signature
Wrong device
Wrong account
Wrong license
Unauthorized feature
Invalid volume
Invalid symbol
Risk violation

```

---

# 50. Idempotency

Execution is always idempotent.

The same:

```
command_id

```

must execute at most once.

Maintain:

```
command_id
signal_id
device_id
account_id
execution_status
broker_ticket
received_at
executed_at

```

---

# 51. License Failure Safety

If license becomes:

```
SUSPENDED
EXPIRED
REVOKED

```

the system shall:

```
Stop new signal entitlement as configured
Stop new automated orders
Preserve audit records
Leave existing broker-side SL/TP active
Notify user
Notify admin

```

Do not forcibly close live positions solely because a subscription expires unless an explicitly configured and separately authorized safety policy requires it.

---

# 52. Windows Installer

Provide a signed installer package.

Recommended deliverables:

```
Predict-A-Trade-Agent-x64.msi
MT4 EA
MT5 EA
Installation Guide
Release Notes
Checksums

```

The installer shall:

```
Install Windows Service
Set least-privilege permissions
Create required local directories
Configure logs
Install/update safely
Support repair
Support uninstall

```

---

# 53. Code Signing and Client Supply Chain

Production Windows binaries must be Authenticode signed.

Release metadata shall include:

```
version
channel
published_at
minimum_server_version
minimum_client_version
sha256
signature
release_notes
mandatory

```

Never distribute unsigned replacement binaries as routine production updates.

---

# 54. Automatic Updates

Support:

```
STABLE
BETA

```

channels.

Agent shall:

```
Check signed update manifest
Verify signature
Download over TLS
Verify SHA-256
Stage update
Stop service safely
Install
Restart
Health check
Rollback if required

```

Admin can define minimum supported versions.

---

# 55. PostgreSQL 17 Architecture

PostgreSQL remains the system of record.

Use separate logical schemas where appropriate:

```text
iam
control
licensing
billing
referral
finance
trading
market
research
audit
support
```

Create separate database roles for:

```text
migration
nest_control
billing_worker
commission_worker
payout_worker
go_realtime
python_research
readonly_analytics
audit_reader
backup
```

Do not allow every service to connect as database owner.

Financial records and trading records may share the same PostgreSQL cluster but must use clear schema/role boundaries. The referral/commission subsystem shall not hold locks or execute synchronous work inside the real-time signal transaction path.

All monetary fields must use exact decimal types. Binary floating-point is prohibited for subscription balances, commission rates, commission amounts, wallet balances and payouts.

---

# 56. PgBouncer

Use PgBouncer for application connection pooling.

NestJS, Go and Python workloads must have appropriately separated pools.

Monitor:

```
client connections
server connections
wait time
pool exhaustion
transaction rate
errors

```

Avoid excessive per-service PostgreSQL connection counts.

---

# 57. TimescaleDB

Use TimescaleDB hypertables for high-volume time-series workloads.

Recommended:

```
ticks
candles
features
market_states
signal_events
execution_events
position_events
latency_events
broker_metrics
client_heartbeats

```

Use continuous aggregates where they materially reduce dashboard and analytics cost.

Implement explicit:

```
Retention
Compression / columnstore policy
Chunk policy
Index strategy
Aggregate refresh policy

```

Do not delete raw historical data merely because a dashboard aggregate exists unless a defined retention policy permits it.

---


# 57A. Additional XAUUSD / Flow Hypertables

Add, where required by the actual schema design:

```text
flow_features
spot_futures_basis
liquidity_events
structure_events
session_state_events
data_quality_events
strategy_feature_snapshots
```

Suggested `flow_features` dimensions:

```text
time
symbol
source_instrument
futures_contract
feature_name
value
quality
strategy_relevance
source_sequence
latency_ms
derivation_version
```

Do not store high-cardinality unbounded labels as Prometheus metrics; detailed event dimensions belong in Timescale/log storage.

# 58. pgvector

Use pgvector for research/intelligence workloads such as:

```
News embeddings
Historical setup embeddings
Signal-context embeddings
Research-document embeddings
Pattern representations
AI-context retrieval

```

Vector similarity is evidence.

It shall not independently authorize an order.

Store:

```
embedding_model
embedding_version
source_id
created_at
metadata

```

so vectors can be reproduced or migrated later.

---

# 59. Valkey

Use Valkey for short-lived state:

```
Latest price
Latest features
Current regime
Current market state
Active signals
Risk state
Session state
License entitlement cache
Heartbeats
Rate limits
Distributed locks
Short-lived sessions where applicable

```

PostgreSQL remains authoritative for persistent business state.

---

# 60. Canonical Database Migration Policy

There shall be **one authoritative migration sequence**.

No service may independently use automatic production schema synchronization.

Requirements:

```
Versioned migrations
Forward migration
Rollback strategy where feasible
Pre-production validation
Backup before destructive changes
Schema compatibility checks
Migration audit

```

Go, NestJS and Python must consume the same canonical schema contract.

---

# 61. Core IAM Tables

Minimum:

```
users
organizations
memberships
roles
permissions
role_permissions
sessions
mfa_methods
recovery_codes
login_events
api_credentials

```

---

# 62. Licensing, Billing & Referral Tables

Minimum licensing tables:

```text
plans
plan_prices
plan_entitlements
licenses
license_entitlements
devices
device_keys
license_devices
activations
entitlement_leases
license_events
client_releases
download_events
mt_accounts
mt_connections
```

Minimum billing tables:

```text
subscriptions
subscription_events
invoices
invoice_items
payments
payment_events
refunds
chargebacks
coupons
credits
```

Minimum referral/commission tables:

```text
affiliate_profiles
referral_codes
referral_attributions
referral_relationships
referral_events
commission_rules
purchase_commission_rules
commission_ledger
commission_adjustments
affiliate_wallets
payout_methods
payouts
payout_items
affiliate_risk_flags
commission_caps
```

Recommended constraints include:

```text
UNIQUE(referral_codes.code)
UNIQUE(referral_relationships.child_user_id)
CHECK(referral_relationships.parent_user_id <> child_user_id)
UNIQUE(payment_events.provider, provider_event_id)
UNIQUE(commission_ledger.purchase_id, recipient_user_id, level, commission_rule_snapshot_id)
```

The tree must remain acyclic. Cycle prevention must be enforced server-side, not only in the UI.

Historical commission ledger entries must snapshot the exact plan, base rate, multiplier, eligible depth, commissionable amount policy and rule version used at creation time.

---

# 63. Signal / Trading Tables

Maintain:

```
signals
signal_events
signal_snapshots
signal_recipients
predictions
prediction_outcomes
risk_decisions
execution_commands
execution_events
positions
trades
broker_events
manual_interventions

```

---


# 63A. Strategy and Quantitative Configuration Tables

Add canonical tables or equivalent normalized structures:

```text
strategy_definitions
strategy_versions
strategy_timeframe_profiles
strategy_session_profiles
indicator_parameter_profiles
feature_definitions
confluence_profiles
confluence_weights
risk_profiles
execution_profiles
prediction_targets
calibration_profiles
grade_policies
validation_policies

market_calendars
session_definitions
gold_fix_windows

futures_contracts
futures_roll_rules
data_provider_capabilities

liquidity_pools
sweep_events
structure_events
fvg_zones
order_blocks
breaker_blocks
liquidity_voids

backtest_runs
walk_forward_runs
validation_reports
calibration_reports
promotion_approvals
```

Every signal snapshot shall reference the exact applicable versions.

Database constraints must prevent an `ACTIVE` strategy version from referencing missing/inactive required profiles.

# 64. Signal Snapshot

A signal must contain enough information to reproduce the decision later.

Capture:

```
Market Data
Features
Indicators
Regime
Structure
Liquidity
Volume
VWAP
Volatility
DXY
Yields
News
Session
Pillar Scores
Model Versions
AI Outputs
Risk Rules
Configuration Version
Strategy Version
Final Decision

```

---

# 65. Audit Architecture

Audit events must be append-oriented and resistant to silent modification.

Record:

```text
event_id
actor_type
actor_id
tenant_id
action
entity_type
entity_id
request_id
timestamp
source_ip
user_agent
old_value
new_value
reason
correlation_id
```

High-risk events:

```text
License revoke
User suspend
Role change
Risk change
Kill switch
Strategy activation
Execution enable
AI provider change
Secret rotation
Device reset
Client release
Plan price change
Plan entitlement change
Referral sponsor change
Commission rule change
Purchase multiplier change
Commission hold/release
Commission reversal
Manual commission adjustment
Payout approval/rejection
Payout method change
Refund/chargeback handling
Affiliate suspension
```

Financially relevant audit records must correlate to payment, commission ledger and payout identifiers.

Historical financial records are immutable except through explicit reversal/adjustment entries.

---

# 66. Configuration

Do not hardcode strategy-critical or commercial-critical values.

Configuration hierarchy:

```text
GLOBAL
TENANT
PLAN
STRATEGY
SYMBOL
BROKER
ACCOUNT
REFERRAL_PROGRAM
PAYOUT_POLICY
```

Trading examples:

```text
minimum_score
minimum_score_difference
minimum_rr
minimum_grade
max_spread
max_slippage
news_blackout
signal_ttl
cooldown
risk_per_trade
maximum_positions
allowed_sessions
allowed_regimes
ai_timeout
ai_required
execution_mode
```

Commercial examples:

```text
setup_fee
monthly_price
billing_interval
plan_strategy_entitlements
max_active_strategy_slots
referral_depth
l1_l5_base_rates
first_purchase_multiplier
second_purchase_multiplier
recurring_purchase_multiplier
max_level_first
max_level_second
max_level_recurring
commissionable_setup_fee
commission_holding_period
minimum_withdrawal
payout_schedule
commission_caps
attribution_window_days
```

Every production configuration change must be versioned, effective-dated where applicable, and auditable.

A change to pricing or commission rules applies prospectively unless an explicit migration is approved. It must never silently rewrite historical invoices, ledger entries or payouts.

---


# 66A. Strategy Configuration Safety

Strategy configuration changes are high-risk changes.

Requirements:

```text
schema validation
type/range validation
cross-field validation
dry-run evaluation
diff preview
reason
actor
approval workflow for production
effective_at
rollback target
immutable audit
```

Production strategy configuration must not be edited directly from an arbitrary browser JSON payload.

No automated optimizer, ML job, AI model or Codex agent may write directly into `ACTIVE` production strategy parameters.

Optimization output lands in:

```text
CANDIDATE
```

and requires validation + human/admin approval before promotion.

# 67. Feature Flags

At minimum:

```
xauusd_scalping
order_flow
dxy
treasury_yields
news_filter
ai_verification
vision_ai
ml_classifier
experimental_astro
paper_trading
shadow_trading
live_signals
auto_execution
mt4
mt5
windows_agent
license_enforcement
new_user_dashboard
new_admin_dashboard

```

Feature flags shall support controlled rollout.

---


# 67A. Additional XAUUSD Feature Flags

Add:

```text
gc_futures_flow
gc_depth
spot_tick_volume_proxy
liquidity_pool_engine
smc_features
ifvg
breaker_blocks
amd_research
silver_bullet_research
gold_macro_scoring
cot_context
etf_flow_context
strategy_standard_scalping
strategy_ultra_scalping
strategy_standard_swing
strategy_trend_swing
chart_footprint
chart_heatmap
signal_replay
```

Experimental flags default to disabled or research/shadow mode until validated.

# 68. Notifications

Create a notification abstraction.

Possible channels:

```text
In-App
Email
Browser Push
Desktop Client
Configured external integrations
```

Notification categories:

```text
Signal
Execution
Risk
News
License
Security
System
Billing
Referral
Commission
Payout
Support
```

Commercial notification events include:

```text
Subscription Activated
Renewal Upcoming
Payment Succeeded
Payment Failed
Plan Changed
Referral Registered
Referral First Payment
Referral Second Payment
Referral Recurring Payment
Commission Created
Commission Cleared
Commission Available
Commission Reversed
Withdrawal Requested
Withdrawal Approved
Withdrawal Paid
Withdrawal Failed
Affiliate Hold
```

Delivery failures must not affect signal generation, billing truth, commission calculation or payout accounting.

---

# 68A. Dynamic Market/Execution Intelligence Alerts

Add asynchronous monitoring jobs that can update operational state and alert users/admins without becoming mandatory tick-path dependencies.

Track where licensed/available:

```text
high-impact economic calendar changes
event schedule revisions
macro data corrections
DXY / yield feed health
CME/GC feed health
broker spread distribution shifts
broker commission configuration changes
swap/carry changes
broker trading-session changes
execution latency deterioration
slippage/rejection deterioration
client minimum-version risk
```

Outputs:

```text
INFORMATIONAL
WATCH
DEGRADED
RISK_REVIEW_REQUIRED
STRATEGY_SUSPEND_RECOMMENDED
```

Only approved deterministic policy may automatically suspend a strategy. Monitoring jobs may not autonomously optimize or activate strategy parameters.

User-facing notifications must distinguish:

```text
market information
signal notification
risk warning
license/subscription notice
execution-quality warning
```

Admin dashboards shall show the source, timestamp, freshness, affected strategy/broker, and remediation status for every alert.


# 69. Subscription, Billing, Plans & Multi-Level Referral Commission System

This section defines the production commercial model for Predict-A-Trade.

The platform shall operate three primary monthly subscription plans with one-time setup fees. Plan pricing, strategy entitlements, referral commission rates, billing-cycle multipliers and payout rules must be configuration-driven and effective-dated.

## 69.1 Canonical Production Plans

| Plan Code | Setup Fee | Monthly Fee | Strategy Access |
|---|---:|---:|---|
| `STANDARD` | $19 one-time | $99/month | Exactly **1** active strategy: **Standard Scalping OR Standard Swing** |
| `PRO` | $29 one-time | $499/month | Any **2** active strategies from **Standard Scalping, Ultra Scalping, Standard Swing, Trend Swing** |
| `ELITE` | $39 one-time | $999/month | **All 4** strategies: Standard Scalping + Ultra Scalping + Standard Swing + Trend Swing |

If the existing database or code uses `BASIC` for the $99 plan, implement a backward-compatible migration/alias from `BASIC` to `STANDARD`; do not break existing subscribers.

The commercial difference defined here is strategy access. Device limits, MT account limits, execution mode, analytics, API access and other entitlements remain granular configuration and must not be guessed from the price tier.

## 69.2 Strategy Selection Rules

For `STANDARD`:

```text
max_active_strategy_slots = 1
allowed = [STANDARD_SCALPING, STANDARD_SWING]
```

For `PRO`:

```text
max_active_strategy_slots = 2
allowed = [
  STANDARD_SCALPING,
  ULTRA_SCALPING,
  STANDARD_SWING,
  TREND_SWING
]
```

For `ELITE`:

```text
max_active_strategy_slots = 4
required/default = all four
```

User strategy selection must be stored server-side and included in the effective entitlement lease.

A strategy change policy shall be configurable. Recommended production default: user-requested strategy changes become effective at the next billing period; administrators may perform an immediate override only with audit logging.

## 69.3 Subscription States

Support:

```text
INCOMPLETE
TRIAL
ACTIVE
PAST_DUE
GRACE
SUSPENDED
CANCEL_AT_PERIOD_END
CANCELLED
EXPIRED
```

Billing state and license state are related but not identical.

A payment callback shall never directly bypass entitlement, licensing, risk or execution controls.

## 69.4 One-Time Setup Fee Policy

The setup fee is a separate one-time charge:

```text
STANDARD = $19
PRO      = $29
ELITE    = $39
```

**Initial production referral policy:** setup fees are non-commissionable.

Represent this as configuration:

```text
commissionable_setup_fee = false
```

This avoids mixing onboarding/service costs with recurring referral economics. The architecture may support changing this rule prospectively later, but historical transactions must remain unchanged.

## 69.5 Commissionable Subscription Amount

Referral commissions shall be calculated from `commissionable_amount`, not blindly from the displayed catalog price.

Initial production definition:

```text
commissionable_amount =
  eligible recurring subscription subtotal
  - eligible discounts/coupons/credits
```

Exclude by default:

```text
one-time setup fee
taxes
refunded amounts
chargebacked amounts
non-subscription add-ons
manual non-eligible credits
```

Payment-processing fees and other accounting treatments must be configurable and approved by finance/accounting policy.

## 69.6 Referral Eligibility Principle

Referral commission is earned only from validated eligible customer subscription payments.

No commission is generated merely for:

```text
link clicks
registrations
account verification
recruitment
network size
unpaid invoices
failed payments
free trials
```

This distinction must be explicit in product copy, referral terms, dashboards and reporting.

## 69.7 Referral Depth

The initial production referral depth is five levels:

```text
Level 1 = direct sponsor
Level 2 = sponsor's sponsor
Level 3
Level 4
Level 5
```

A user can have only one direct sponsor.

If a sponsor does not exist at a level, no commission is generated for that level. Unallocated commission is not automatically compressed upward or redistributed unless a future rule explicitly introduces compression.

## 69.8 Base Commission Matrix

Initial base rates:

| Referral Level | STANDARD $99 | PRO $499 | ELITE $999 |
|---|---:|---:|---:|
| Level 1 | 10% | 15% | 20% |
| Level 2 | 4% | 5% | 6% |
| Level 3 | 2% | 3% | 4% |
| Level 4 | 1% | 2% | 2% |
| Level 5 | 0.5% | 1% | 1% |
| **Maximum Base Network Rate** | **17.5%** | **26%** | **33%** |

These are base rates. The effective rate depends on the customer's eligible subscription payment number/type.

## 69.9 Monthly Subscription Payment Number Model

Because the commercial model is monthly, the purchase-number concept is mapped to successful eligible subscription billing cycles.

```text
Eligible Subscription Payment #1
    -> FIRST_PURCHASE
    -> L1-L5 eligible
    -> 100% multiplier

Eligible Subscription Payment #2
    -> SECOND_PURCHASE
    -> L1 only
    -> 75% multiplier

Eligible Subscription Payment #3 and every later eligible monthly payment
    -> RECURRING_PURCHASE
    -> L1-L5 eligible
    -> 50% multiplier
```

The one-time setup fee does **not** increment this counter.

The counter is based only on validated eligible recurring subscription payments.

## 69.10 Critical Second-Payment Rule

**Purchase/payment #2 pays referral commission only to the buyer's direct sponsor (Level 1).**

Levels 2-5 receive **zero** commission on the second eligible monthly subscription payment, regardless of their normal base rates.

Configuration:

```text
SECOND_PURCHASE:
  multiplier = 0.75
  max_referral_level = 1
```

This rule must be enforced in the commission engine, database rule configuration, test suite, dashboards, examples and reporting.

## 69.11 Third and Subsequent Monthly Payments

For eligible monthly subscription payment #3 and later:

```text
multiplier = 0.50
max_referral_level = 5
```

All available sponsor levels L1-L5 may receive 50% of their plan-specific base rate.

## 69.12 Effective Rate Formula

For an eligible sponsor level:

```text
effective_commission_rate =
    base_commission_rate
    × purchase_multiplier
```

Then:

```text
commission_amount =
    commissionable_amount
    × effective_commission_rate
```

If the referral level exceeds `max_referral_level`, effective rate is zero and no commission ledger record should normally be created for that ineligible level.

## 69.13 Complete Effective Commission Matrix

### STANDARD — $99 Monthly

| Level | Base | 1st Payment | 2nd Payment | 3rd+ Payment |
|---|---:|---:|---:|---:|
| L1 | 10% | 10% | 7.5% | 5% |
| L2 | 4% | 4% | 0% | 2% |
| L3 | 2% | 2% | 0% | 1% |
| L4 | 1% | 1% | 0% | 0.5% |
| L5 | 0.5% | 0.5% | 0% | 0.25% |
| **Network Total** | **17.5%** | **17.5%** | **7.5%** | **8.75%** |

### PRO — $499 Monthly

| Level | Base | 1st Payment | 2nd Payment | 3rd+ Payment |
|---|---:|---:|---:|---:|
| L1 | 15% | 15% | 11.25% | 7.5% |
| L2 | 5% | 5% | 0% | 2.5% |
| L3 | 3% | 3% | 0% | 1.5% |
| L4 | 2% | 2% | 0% | 1% |
| L5 | 1% | 1% | 0% | 0.5% |
| **Network Total** | **26%** | **26%** | **11.25%** | **13%** |

### ELITE — $999 Monthly

| Level | Base | 1st Payment | 2nd Payment | 3rd+ Payment |
|---|---:|---:|---:|---:|
| L1 | 20% | 20% | 15% | 10% |
| L2 | 6% | 6% | 0% | 3% |
| L3 | 4% | 4% | 0% | 2% |
| L4 | 2% | 2% | 0% | 1% |
| L5 | 1% | 1% | 0% | 0.5% |
| **Network Total** | **33%** | **33%** | **15%** | **16.5%** |

## 69.14 Example Commission Amounts at Full Monthly Price

Using full monthly price as the commissionable amount and financial rounding per ledger entry:

### STANDARD — First Payment $99

```text
L1 10%   = $9.90
L2 4%    = $3.96
L3 2%    = $1.98
L4 1%    = $0.99
L5 0.5%  = $0.50
```

### STANDARD — Second Payment $99

```text
L1 7.5%  = $7.43
L2-L5    = $0.00
```

### PRO — First Payment $499

```text
L1 15% = $74.85
L2 5%  = $24.95
L3 3%  = $14.97
L4 2%  = $9.98
L5 1%  = $4.99
```

### PRO — Second Payment $499

```text
L1 11.25% = $56.14
L2-L5     = $0.00
```

### ELITE — First Payment $999

```text
L1 20% = $199.80
L2 6%  = $59.94
L3 4%  = $39.96
L4 2%  = $19.98
L5 1%  = $9.99
```

### ELITE — Second Payment $999

```text
L1 15% = $149.85
L2-L5  = $0.00
```

## 69.15 Upgrades, Downgrades and Re-Subscriptions

Plan changes must not be exploitable to reset the referral payment sequence.

Initial production rules:

```text
Upgrade does not reset purchase/payment number.
Downgrade does not reset purchase/payment number.
Strategy selection change does not reset purchase/payment number.
License/device reset does not reset purchase/payment number.
Cancellation does not automatically erase purchase/payment history.
Re-subscription by the same verified customer continues the historical eligible-payment sequence unless an audited business rule explicitly resets it.
```

Mid-cycle proration, upgrade charges and downgrade credits shall use an explicit `ADJUSTMENT` or `PRORATION` transaction type.

**Recommended initial policy:** prorated adjustment charges do not create referral commission unless specifically configured.

## 69.16 Referral Code and Attribution

Every eligible affiliate receives a unique referral code and URL.

Example:

```text
https://predictatrade.com/register?ref=ABC123
```

Initial attribution policy:

```text
First valid referral attribution wins.
Attribution window is configurable.
Recommended initial window = 30 days.
```

Once the sponsor relationship is established, it is not changed automatically.

Manual sponsor changes require:

```text
privileged admin permission
re-authentication/MFA
mandatory reason
cycle validation
immutable audit record
```

## 69.17 Referral Registration Flow

```text
Referral Link Click
    ↓
Referral Code Validated
    ↓
Attribution Stored
    ↓
Visitor Registers
    ↓
Sponsor Relationship Created
    ↓
Account Verification
    ↓
Subscription Checkout
    ↓
Payment Provider
    ↓
Signed Webhook Verified
    ↓
Payment Validated / Idempotency Checked
    ↓
Subscription Activated or Renewed
    ↓
Payment Number Determined
    ↓
Plan + Commission Rules Loaded
    ↓
Eligible Sponsor Chain Loaded
    ↓
Commission Ledger Records Created
    ↓
PENDING
```

## 69.18 Payment/Commission Trust Boundary

The client must never submit or control:

```text
payment success
commissionable amount
commission rate
purchase multiplier
referral level
commission amount
payout approval
```

The referral engine reacts only to a validated server-side billing event.

Recommended event flow:

```text
PAYMENT_PROVIDER
    ↓ signed webhook
BILLING VALIDATION
    ↓ durable payment event
SUBSCRIPTION STATE
    ↓ outbox/event
COMMISSION ENGINE
    ↓
COMMISSION LEDGER
    ↓
AFFILIATE BALANCE
    ↓
PAYOUT SYSTEM
```

The commission system is a sidecar business subsystem and must remain isolated from the Go market-data/signal/execution critical path.

## 69.19 Idempotency and Exactly-Once Financial Effect

Payment providers may deliver duplicate or reordered events.

Requirements:

```text
unique provider_event_id
unique payment transaction identifier
idempotent webhook processing
durable inbox/outbox pattern or equivalent
transactional commission creation
duplicate commission constraint
replay-safe refund/chargeback handling
```

A duplicate payment webhook must never create a second subscription renewal or duplicate commission.

## 69.20 Commission Ledger

Every commission is recorded individually.

Minimum ledger fields:

```text
id
recipient_user_id
source_user_id
source_subscription_id
purchase_id / payment_id
invoice_id
plan_id
plan_version
purchase_number
purchase_type
level
base_commission_rate
purchase_multiplier
effective_commission_rate
commissionable_amount
commission_amount
currency
status
commission_rule_id
commission_rule_snapshot
payment_event_id
created_at
cleared_at
available_at
paid_at
reversed_at
correlation_id
```

Do not rewrite ledger history when rules change.

Corrections use reversal or adjustment records.

## 69.21 Commission Status Lifecycle

Normal lifecycle:

```text
PENDING
    ↓
CLEARED
    ↓
AVAILABLE
    ↓
PAID
```

Exception states:

```text
CANCELLED
REVERSED
CHARGEBACK
FRAUD_HOLD
```

A configurable holding period shall apply before commission becomes withdrawable.

## 69.22 Refunds, Partial Refunds and Chargebacks

Refunded or chargebacked subscription revenue must not remain as withdrawable commission.

Requirements:

```text
full refund -> reverse related commission
partial refund -> proportional or policy-defined adjustment
chargeback -> hold/reverse related commission
already-paid commission -> negative adjustment/recovery ledger entry
```

Never delete the original commission record.

All reversals must reference the original ledger item and payment/refund/chargeback event.

## 69.23 Affiliate Wallet / Balance Model

Do not treat a mutable single balance column as the accounting source of truth.

Use ledger-derived balance buckets:

```text
pending
cleared
available
paid
reversed
on_hold
```

A cached wallet summary may be maintained for performance, but it must be reconcilable to the underlying commission and payout ledgers.

## 69.24 Payout System

Recommended flow:

```text
Commission Generated
    ↓
PENDING
    ↓ holding period
CLEARED
    ↓
AVAILABLE
    ↓
Withdrawal Request
    ↓
Eligibility / Risk / Compliance Check
    ↓
APPROVED
    ↓
PROCESSING
    ↓
PAID
```

Payout statuses:

```text
REQUESTED
UNDER_REVIEW
APPROVED
PROCESSING
PAID
FAILED
CANCELLED
```

Minimum withdrawal threshold, supported payout methods, payout schedule and manual/automatic approval must be configuration-driven.

The architecture must be ready to require identity/tax/payout verification where applicable without hardcoding one jurisdiction's policy.

## 69.25 Commission Caps

Support optional configurable caps:

```text
maximum commission per transaction
maximum daily commission
maximum monthly commission
maximum affiliate available balance
maximum network payout per customer/payment
```

Caps must create explicit reason codes and audit events.

## 69.26 Anti-Abuse and Fraud Controls

Minimum controls:

```text
self-referral prevention
duplicate-account detection
circular referral prevention
sponsor-chain cycle detection
fake purchase prevention
payment validation
refund/chargeback abuse detection
device/account correlation signals
payment-method correlation signals
IP/network anomaly signals
high-velocity signup/purchase detection
commission farming detection
payout-risk scoring
manual review / hold workflow
```

Automated signals should create review/hold workflows rather than silently confiscating balances.

No user may alter a sponsor relationship through a client-side request.

## 69.27 Financial Precision

Use exact decimal arithmetic.

Recommended database types:

```text
DECIMAL(18,8) for rates/amount internals
currency-specific rounding at posting/payout boundary
```

Do not use binary floating point for financial calculations.

Persist both the unrounded calculation basis where appropriate and the posted rounded amount.

Rounding policy must be deterministic and tested.

## 69.28 Commission Rule Versioning

Commission rules shall support:

```text
effective_from
effective_until
rule_version
active
approved_by
approved_at
```

Historical commissions retain the rule snapshot active when the payment was processed.

Changing a future rate must not alter old commission records.

## 69.29 Required User Referral Analytics

Track:

```text
Referral Link Clicks
Registrations
Verified Users
First Subscription Payments
Second Subscription Payments
Third+ Subscription Payments
Conversion Rate
Second-Payment Retention
Recurring Retention
Revenue Generated
Commission Generated
Commission Paid
Earnings by Level
Earnings by Plan
```

Analytics events are not themselves financial truth; the commission ledger remains authoritative.

## 69.30 Required Admin Commercial Reporting

Provide:

```text
Sales by Plan
Setup Fee Revenue
Monthly Subscription Revenue
MRR
Active Subscribers
New Subscribers
Upgrades
Downgrades
Cancellations
Churn
Failed Renewals
Referral-Attributed Revenue
Direct vs Network Revenue
First/Second/Recurring Payment Counts
Second-Payment Conversion
Recurring Retention
Commission Expense by Plan
Commission Expense by Level
Pending Liability
Available Liability
Paid Commission
Reversed Commission
Payout Reconciliation
Refund/Chargeback Impact
Affiliate Concentration
Fraud Holds
```

## 69.31 API Requirements

User/referral:

```text
GET  /api/v1/referrals/me
GET  /api/v1/referrals/link
GET  /api/v1/referrals/network
GET  /api/v1/referrals/statistics

GET  /api/v1/commissions
GET  /api/v1/commissions/summary
GET  /api/v1/commissions/levels
GET  /api/v1/commissions/purchases
GET  /api/v1/commissions/{id}

GET  /api/v1/payouts
POST /api/v1/payouts/request
GET  /api/v1/payouts/{id}
```

Subscription:

```text
GET  /api/v1/subscription
GET  /api/v1/plans
POST /api/v1/subscription/checkout
POST /api/v1/subscription/change-plan
POST /api/v1/subscription/change-strategies
POST /api/v1/subscription/cancel
GET  /api/v1/billing/invoices
GET  /api/v1/billing/payments
```

Admin:

```text
GET    /api/v1/admin/plans
POST   /api/v1/admin/plans
PATCH  /api/v1/admin/plans/{id}

GET    /api/v1/admin/referrals
GET    /api/v1/admin/referrals/{userId}
POST   /api/v1/admin/referrals/{userId}/change-sponsor

GET    /api/v1/admin/commission-rules
POST   /api/v1/admin/commission-rules
PATCH  /api/v1/admin/commission-rules/{id}

GET    /api/v1/admin/purchase-rules
PATCH  /api/v1/admin/purchase-rules/{id}

GET    /api/v1/admin/commissions
POST   /api/v1/admin/commissions/{id}/hold
POST   /api/v1/admin/commissions/{id}/release
POST   /api/v1/admin/commissions/{id}/reverse
POST   /api/v1/admin/commissions/adjustments

GET    /api/v1/admin/payouts
POST   /api/v1/admin/payouts/{id}/approve
POST   /api/v1/admin/payouts/{id}/reject
POST   /api/v1/admin/payouts/{id}/process
```

Payment webhooks are provider-specific server endpoints and must require signature verification, timestamp/replay checks and idempotency.

## 69.32 Security Requirements

Mandatory:

```text
server-side commission calculation
signed payment webhooks
strict RBAC
MFA/re-authentication for high-risk admin/payout actions
rate limiting
CSRF protection where applicable
secure referral attribution
database constraints
idempotency
immutable audit
fraud detection
secret management
masked payout data
least-privilege database roles
```

Clients must never be trusted to submit their own commission amount.

## 69.33 Commercial and Referral Acceptance Criteria

The subsystem is not production-ready until all are proven:

1. All three setup/monthly prices are represented correctly and effective-dated.
2. STANDARD allows exactly one of Standard Scalping or Standard Swing.
3. PRO allows any two of the four strategies.
4. ELITE grants all four strategies.
5. Strategy entitlement is enforced server-side in REST, WebSocket and Windows/MT clients.
6. Every eligible user can receive a unique referral code.
7. First-valid attribution works within the configured attribution window.
8. Sponsor relationships remain acyclic.
9. Self-referral and obvious duplicate-account abuse are blocked/flagged.
10. The first eligible monthly payment uses 100% multiplier and may pay L1-L5.
11. The second eligible monthly payment uses 75% multiplier and pays **L1 only**.
12. The second eligible payment never creates L2-L5 commission.
13. The third and later eligible monthly payments use 50% multiplier and may pay L1-L5.
14. The setup fee is non-commissionable under the initial production configuration.
15. Upgrades/downgrades do not reset the payment sequence.
16. Duplicate payment events cannot create duplicate subscription or commission effects.
17. Commission rate, multiplier, eligible depth and amount are snapshotted.
18. Refunds and chargebacks reverse/adjust commissions without deleting history.
19. Financial calculations use exact decimal arithmetic.
20. Users can inspect referral network, earnings by level, earnings by plan and earnings by payment type.
21. Admins can inspect and configure future commission rules without rewriting history.
22. Payout requests cannot exceed available balance.
23. Payout approval and completion are auditable and reconciled.
24. Affiliate holds prevent withdrawal but do not silently destroy ledger history.
25. Billing/referral failure cannot block or degrade the Go real-time trading engine.
26. All plan, billing, referral, commission and payout changes are covered by automated tests and audit events.

## 69.34 Initial Production Configuration Lock

Initial release configuration:

```text
PLANS
STANDARD = $19 setup + $99/month
PRO      = $29 setup + $499/month
ELITE    = $39 setup + $999/month

SETUP FEE COMMISSIONABLE
false

REFERRAL DEPTH
5

BASE RATES
STANDARD = [10%, 4%, 2%, 1%, 0.5%]
PRO      = [15%, 5%, 3%, 2%, 1%]
ELITE    = [20%, 6%, 4%, 2%, 1%]

PAYMENT RULES
#1  = 100% multiplier, max level 5
#2  = 75% multiplier, max level 1
#3+ = 50% multiplier, max level 5

ATTRIBUTION
first valid referral wins
window configurable; initial target 30 days

HISTORICAL RULES
immutable snapshots
```

Any change to these values must be privileged, effective-dated, reasoned and audited.

---

# 70. Support System

User Portal:

```
Create Ticket
Reply
Attachment
Priority
Category
Status

```

Admin:

```
Assign
Escalate
Close
Internal Note
View Diagnostics

```

Windows Agent should be able to create a redacted diagnostic bundle without exposing private keys or broker passwords.

---

# 71. Security Baseline

Target a documented application-security verification baseline such as OWASP ASVS.

Implement:

```
TLS
HSTS
Secure cookies
CSP
CSRF protection
Input validation
Output encoding
Rate limiting
Authentication
MFA
RBAC
Row-level authorization
Secret management
Replay protection
Command signing
Audit logging
Security headers
Dependency scanning

```

Administrative interfaces require stronger controls than public pages.

---

# 72. Token Security

Use short-lived access tokens or secure server-side sessions.

Refresh credentials shall be:

```
Rotatable
Revocable
Device-aware where appropriate
Protected from browser JavaScript where possible

```

Do not store unrestricted long-lived bearer credentials in browser localStorage.

---

# 73. Service-to-Service Authentication

Internal services must authenticate each other.

Possible architecture:

```
mTLS
Signed service JWT
Private service network

```

Public credentials shall not double as service credentials.

---

# 74. Secret Management

Secrets include:

```
AI provider keys
Database passwords
Signing keys
Email credentials
Payment credentials
Market-data credentials
Webhook secrets

```

Requirements:

```
Encrypted storage
Restricted filesystem permissions
No source-control secrets
Rotation
Versioning
Access auditing

```

Frontend bundles must never receive them.

---

# 75. Private-Key Architecture

Separate keys for:

```
User authentication
License signing
Execution-command signing
Update-manifest signing
Windows code signing

```

Do not use one master key for everything.

Support key rotation using key IDs.

---

# 76. Data Privacy

Classify:

```text
Public
Internal
Confidential
Secret
```

Sensitive fields must have explicit retention policies.

Minimize collection of:

```text
IP history
Device identifiers
Broker account information
Diagnostic logs
Referral attribution identifiers
Affiliate risk signals
Payout identity/tax data
Payout destination details
```

Only collect data required for operations, security, licensing, billing, referral attribution, payout, fraud prevention or legal/accounting obligations.

Referral dashboards must not expose unnecessary personal information about downline users. Network views should use privacy-preserving identifiers and aggregate statistics where possible.

Payout destination and verification data must be encrypted/tokenized or delegated to an approved payment provider where feasible.

Data exports and deletion workflows must account for financial/audit retention requirements and must not destroy records that must remain for accounting, fraud investigation or immutable ledger integrity.

---

# 77. Legal and Risk Acknowledgements

Version and retain acceptance of applicable documents such as:

```text
Terms of Service
Privacy Policy
Risk Disclosure
Execution Agreement
Automated Trading Disclosure
Experimental Feature Disclosure
Subscription / Billing Terms
Cancellation / Refund Policy
Referral / Affiliate Program Terms
Commission Rules
Payout Terms
Anti-Abuse / Fraud Policy
```

Store:

```text
document_type
document_version
user_id
accepted_at
source_ip
user_agent
```

Material updates may require re-acceptance.

Referral/affiliate terms must clearly state that:

- commission is generated only from eligible validated subscription revenue;
- no commission is earned merely for recruitment, registration or network size;
- the second eligible monthly payment pays Level 1 only under the initial production rule;
- commissions may remain pending during a configurable holding period;
- refunds, chargebacks, fraud findings and accounting corrections can reverse or adjust commissions;
- payout eligibility may depend on account, identity, tax, sanctions or other checks required by the operating jurisdiction/payment provider;
- historical ledger and audit records may be retained even after account closure where required for legitimate accounting/security purposes.

Jurisdiction-specific legal, tax, consumer, affiliate-marketing and financial-services review must be completed before enabling the program in a target market.

---

# 78. Marketing Claim Governance

No public trading-performance claim such as:

```text
94% accuracy
Guaranteed returns
Guaranteed signals
Guaranteed profitability
```

may be published merely from an internal score.

Performance claims require an approved evidence package containing:

```text
Metric definition
Dataset
Period
Sample size
Instrument
Strategy
Trade costs
Slippage assumptions
Out-of-sample status
Confidence interval where relevant
Validation method
Report version
```

Public web metrics must be traceable to the corresponding approved report.

Referral/affiliate marketing must also prohibit misleading earnings claims such as:

```text
guaranteed referral income
guaranteed monthly commission
risk-free income
income based only on recruiting users
```

Affiliate dashboards and marketing materials must distinguish:

```text
revenue generated
commission generated
commission pending
commission available
commission paid
```

Example or historical affiliate earnings must not be presented as a guaranteed future result.

---

# 79. Data Vendor Rights

Before redistributing market/news data to users or MT clients, confirm permitted usage under the relevant provider agreement.

Track:

```
Provider
Dataset
License type
Redistribution allowed
Display restrictions
Retention restrictions
Attribution requirements

```

---

# 80. API Architecture

## Control Plane

Examples:

```text
POST /api/v1/auth/login
POST /api/v1/auth/logout
POST /api/v1/auth/mfa/verify

GET  /api/v1/user/profile
GET  /api/v1/user/licenses
GET  /api/v1/user/devices
GET  /api/v1/user/mt-accounts
GET  /api/v1/user/signals
GET  /api/v1/user/performance

GET  /api/v1/plans
GET  /api/v1/subscription
POST /api/v1/subscription/checkout
POST /api/v1/subscription/change-plan
POST /api/v1/subscription/change-strategies
POST /api/v1/subscription/cancel
GET  /api/v1/billing/invoices
GET  /api/v1/billing/payments

GET  /api/v1/referrals/me
GET  /api/v1/referrals/link
GET  /api/v1/referrals/network
GET  /api/v1/referrals/statistics

GET  /api/v1/commissions
GET  /api/v1/commissions/summary
GET  /api/v1/commissions/levels
GET  /api/v1/commissions/purchases

GET  /api/v1/payouts
POST /api/v1/payouts/request

POST /api/v1/licenses/activate
POST /api/v1/licenses/refresh
POST /api/v1/devices/register

GET  /api/v1/downloads
GET  /api/v1/releases/latest
```

## Admin

```text
GET    /api/v1/admin/users
GET    /api/v1/admin/licenses
POST   /api/v1/admin/licenses
PATCH  /api/v1/admin/licenses/{id}
POST   /api/v1/admin/licenses/{id}/suspend
POST   /api/v1/admin/licenses/{id}/revoke

GET    /api/v1/admin/plans
POST   /api/v1/admin/plans
PATCH  /api/v1/admin/plans/{id}

GET    /api/v1/admin/subscriptions
GET    /api/v1/admin/payments
GET    /api/v1/admin/referrals
GET    /api/v1/admin/commissions
GET    /api/v1/admin/payouts
GET    /api/v1/admin/commission-rules
GET    /api/v1/admin/purchase-rules

GET    /api/v1/admin/devices
POST   /api/v1/admin/devices/{id}/revoke

GET    /api/v1/admin/signals
GET    /api/v1/admin/execution
GET    /api/v1/admin/risk
GET    /api/v1/admin/audit
GET    /api/v1/admin/health
```

High-risk commercial mutations shall use explicit endpoints, idempotency keys where appropriate, reason fields, authorization, re-authentication and audit.

## Billing Webhooks

Provider webhook endpoints must be server-only and enforce:

```text
signature verification
timestamp/replay protection
event idempotency
provider/account scoping
schema validation
durable event recording
```

## Go Real-Time

Retain:

```text
/api/v1/xauusd/status
/api/v1/xauusd/market-state
/api/v1/xauusd/regime
/api/v1/xauusd/signals
/api/v1/xauusd/performance
```

Internal execution endpoints remain non-public.

The Go real-time API must consume effective entitlement decisions but must never call the referral/commission engine in the market tick-to-signal critical path.

---

# 81. API Contracts

Use explicit API versions.

Generate/OpenAPI-document control APIs.

All externally consumed messages require:

```
schema version
backward compatibility policy
deprecation policy

```

Breaking client changes require versioned rollout.

---

# 82. Real-Time Authentication

User Dashboard and Windows clients must authenticate before joining real-time channels.

Gateway validates:

```
Identity
Account status
License
Entitlement
Device where applicable
Channel permission
Connection limits
Token expiry

```

Authentication must not require a database query for every market event.

---

# 83. Execution Reconciliation

Continuously reconcile:

```
Predict-A-Trade
vs
Windows Agent
vs
MT Terminal
vs
Broker

```

Detect:

```
Missing Position
Unexpected Position
Wrong Volume
Wrong SL
Wrong TP
Duplicate
Orphan Trade
Manual Intervention

```

---

# 84. Manual Intervention

Record:

```
Who
When
What changed
Old value
New value
Account
Position

```

Policies:

```
ALLOW
WARN
RE-ALIGN
BLOCK_NEW_AUTOMATION

```

---

# 85. Emergency Controls

Kill switches:

```
GLOBAL
STRATEGY
SYMBOL
BROKER
ACCOUNT
LICENSE
DEVICE
MT4
MT5
WINDOWS_AGENT

```

Default emergency action:

```
STOP NEW ORDERS

```

Optional:

```
CLOSE POSITIONS

```

Close-all must require separate explicit authorization.

---

# 86. Observability

Use unified metrics, logs and distributed tracing.

Include correlation identifiers across:

```
HTTP Request
WebSocket Connection
Signal
Prediction
License
Device
Execution Command
Broker Ticket

```

Recommended telemetry standard:

```
OpenTelemetry
Prometheus
Grafana
Structured JSON Logs

```

---

# 87. Critical Metrics

Track:

```
market_data_latency
feature_processing_latency
signal_generation_latency
risk_validation_latency
ai_verification_latency
signal_delivery_latency
websocket_connections
websocket_reconnects
license_validation_latency
agent_heartbeats
execution_latency
fill_rate
reject_rate
slippage
database_latency
pgbouncer_wait
valkey_latency
api_latency
api_errors

```

---


# 87A. Gold-Specific Trading Observability

Add dashboards/metrics for:

```text
xauusd_spread_p50_by_session
xauusd_spread_p95_by_session
spread_to_atr_ratio
market_data_gap_rate
timeframe_sync_lag_ms
sweep_detection_latency_ms
liquidity_pool_count_by_state
liquidity_pool_hit_rate
structure_event_rate
cvd_divergence_event_rate
flow_feature_quality
gc_spot_lead_lag_ms
gc_spot_basis
regime_classification_confidence
signal_confluence_score_distribution
signal_score_separation_distribution
no_trade_reason_breakdown
strategy_candidate_rate
strategy_activation_rate
strategy_expiry_rate
strategy_outcome_by_session
strategy_outcome_by_regime
slippage_by_strategy_broker_session
cost_to_target_ratio
calibration_error_by_strategy
```

Metrics used for financial/trading claims must come from reproducible analytics, not mutable dashboard counters.

# 88. Product Metrics

Track separately:

```text
Active users
Active subscriptions
MRR
Setup fee revenue
New subscriptions
Renewals
Failed renewals
Upgrades
Downgrades
Cancellations
Churn
Active licenses
Active devices
Online agents
Connected MT4
Connected MT5
Signals delivered
Signals opened/viewed
Signal expiry rate
Notification delivery
Client version adoption
Activation failures
Referral link clicks
Referred registrations
Referral-attributed subscriptions
First-payment conversions
Second-payment conversions
Recurring retention
Referral-attributed revenue
Commission generated
Commission pending
Commission available
Commission paid
Commission reversed
Payout success/failure
Fraud holds
```

Do not mix product/business usage metrics with predictive performance metrics.

Do not present gross revenue, referral-attributed revenue, commission expense or affiliate payout as trading performance.

---

# 89. Service-Level Objectives

Define measurable targets rather than vague “fast” requirements.

Initial engineering targets:

```
Critical quantitative processing:
sub-100 ms where infrastructure/data permit

Control API:
p95 target < 300 ms for ordinary cached/read requests

Real-time signal fan-out:
measured end-to-end and continuously reported

Availability:
explicit service-specific SLOs

Data freshness:
explicit threshold per feed

```

External feed, Internet and broker latency must be reported separately from internal processing latency.

---

# 90. High Availability

Support:

```
Process supervision
Automatic restart
Market-feed reconnect
Database reconnect
Valkey reconnect
WebSocket reconnect
Windows Agent reconnect
MT reconnect

```

Use active/standby where appropriate.

Do not introduce unnecessary distributed complexity merely to claim microservices.

---

# 91. Graceful Degradation

Examples:

```
AI fails:
→ Quant-only or reject according to policy

Secondary macro feed fails:
→ Flag degraded state

Database temporary outage:
→ Real-time engine uses controlled hot state/buffer

License API temporary outage:
→ Existing valid signed lease continues until expiry

Web dashboard fails:
→ Real-time engine continues

News feed fails:
→ configured NO-TRADE / degraded policy

```

---

# 92. Backup

Back up:

```
PostgreSQL
TimescaleDB
Configuration
Encryption metadata
Audit
Release metadata
Critical operational files

```

Implement:

```
Encrypted backups
Retention
Off-host copy
Point-in-time recovery
Restore verification

```

---

# 93. Disaster Recovery

Define explicit:

```
RPO
RTO
Backup location
Restore procedure
Failover procedure
DNS procedure
Credential recovery
Signing-key recovery

```

Perform scheduled restore drills.

A backup that has never been restored is not sufficient production validation.

---

# 94. Security Incident Response

Create procedures for:

```
Credential leak
Admin compromise
Signing-key compromise
License abuse
Client compromise
Database breach
Unauthorized execution
Malicious client
Market-data manipulation

```

Support rapid:

```
Token revocation
Device revocation
License revocation
Key rotation
Execution stop
Client minimum-version enforcement

```

---

# 95. CI/CD

Every build pipeline shall run:

```
Formatting
Linting
Unit Tests
Integration Tests
Security Tests
Dependency Audit
Secret Scan
SAST
Build
Artifact Hashing

```

Production release shall require passing gates.

---

# 96. Supply-Chain Security

Generate an SBOM for production releases where practical.

Pin critical dependency versions.

Do not execute arbitrary installation scripts in production pipelines without review.

Record:

```
Git commit
Build ID
Artifact checksum
Build timestamp
Release version

```

---

# 97. Environment Separation

Maintain:

```
LOCAL
DEVELOPMENT
TEST
STAGING
PRODUCTION

```

Production:

- must have independent credentials;
- must not use test licenses;
- must not use test signing keys;
- must not share writable databases with staging.

---

# 98. Testing — Quantitative Engine

Test:

```
Indicators
Structure
Liquidity
FVG
Order Blocks
VWAP
Volume Profile
Regime
Macro
Scoring
Calibration
TTL
Risk
Entry
SL
TP
Position sizing
NO-TRADE

```

---


# 98A. Required Strategy / Feature Test Matrix

For each of the four strategies, add deterministic tests for:

```text
required timeframe freshness
session allow/deny
holiday/DST transition
news tier behavior
broker rollover behavior
score threshold
long-short separation
mandatory pillar failure
data-capability downgrade
spread absolute gate
spread/ATR gate
slippage gate
R:R gate
net expectancy gate
max trade frequency
loss cooldown
aggregate XAU exposure
TTL expiration
duplicate prevention
risk veto precedence
entitlement rejection
execution permission rejection
```

Feature-level golden tests shall cover:

```text
BOS / CHoCH / MSS
equal highs/lows
BSL/SSL pool creation
sweep/rejection
FVG fill state
IFVG transition
OB/breaker transition
displacement
VWAP/bands
POC/VAH/VAL
CVD and divergence when data-capable
futures roll/basis
economic surprise
macro freshness
```

Use fixed fixtures/golden datasets so refactors cannot silently alter feature semantics.

# 99. Testing — Dashboard / NestJS

Test:

```
Authentication
MFA
Authorization
RBAC
Tenant isolation
API validation
Pagination
Filtering
WebSocket authorization
Admin actions
User actions
Audit creation
Configuration
Feature flags

```

---

# 100. Testing — Licensing

Test:

```
New activation
Expired license
Suspended license
Revoked license
Device limit
MT account limit
Wrong device
Wrong account
Duplicate activation
Entitlement refresh
Revocation propagation
Control-plane outage
Clock skew
Invalid signature
Expired lease

```

---

# 101. Testing — Windows Client

Validate:

```
Windows 10/11 supported builds
Install
Repair
Upgrade
Rollback
Uninstall
Service restart
Network disconnect
Reconnect
Sleep/resume
Clock correction
Firewall restriction
Agent crash
Corrupt configuration

```

---

# 102. Testing — MT4/MT5

Validate:

```
Signal reception
BUY
SELL
SL
TP
Modify SL
Modify TP
Partial close
Full close
Duplicate command
Expired signal
Invalid symbol
Invalid volume
Market closed
Broker rejection
Terminal disconnect
Reconnect
Manual intervention
Emergency stop

```

---

# 103. Broker Compatibility Matrix

Maintain explicit testing for each supported broker/account specification.

Record:

```
Broker
Server
XAUUSD symbol
Digits
Contract size
Tick value
Minimum lot
Lot step
Stops level
Fill mode
Typical spread
MT4 / MT5
Validation date

```

No universal XAUUSD contract assumption.

---


# 103A. XAUUSD Broker Execution Profile

Extend the compatibility matrix into a machine-readable broker/symbol profile.

Required fields:

```text
broker
server
platform
canonical_symbol = XAUUSD
broker_symbol
aliases/suffix
base_currency
quote_currency
account_currency
digits
point
tick_size
tick_value
tick_value_currency
contract_size
minimum_lot
maximum_lot
lot_step
stops_level
freeze_level
fill_modes[]
market_sessions[]
maintenance_breaks[]
swap_long
swap_short
swap_calculation_method
triple_swap_day_or_rule
margin_calculation
leverage/margin tiers
commission model
typical spread
spread p95
slippage distribution
last_observed_at
last_validated_at
```

Do not assume:

```text
XAUUSD is always named XAUUSD
contract size is always 100 oz
lot step is always 0.01
Wednesday is universally the same carry treatment
every broker has the same daily break
IOC/FOK/RETURN support is identical
tick value is identical across account currencies
```

The Windows Agent/EA shall read live terminal symbol properties and compare them to the approved profile.

Material mismatch:

```text
EXECUTION_DENIED
BROKER_PROFILE_MISMATCH
```

until reconciled.

Backtests shall use broker-specific spread, commission, slippage, stop/freeze, trading-session and carry assumptions for any result advertised as executable on that broker class.

# 103B. Broker Execution Qualification Test Suite

For every broker/server/account class intended for automated execution, run a repeatable qualification suite.

Minimum tests:

```text
symbol discovery and alias mapping
point/tick/pip display normalization
tick-value validation
contract-size validation
lot min/max/step
stops/freeze levels
fill modes
market-session discovery
maintenance-break discovery
commission calculation
spread p50/p95/p99 by session
slippage distribution by order type/session
order acknowledgement latency
fill latency
partial-fill behavior
rejection behavior
cancel/replace behavior
SL/TP attachment behavior
margin calculation
stop-out/margin-call metadata
swap/carry calculation
disconnect/reconnect
terminal restart
agent restart
duplicate-order protection
clock-skew handling
```

Qualification result:

```text
APPROVED_SIGNAL_ONLY
APPROVED_MANUAL
APPROVED_ASSISTED
APPROVED_AUTO_STANDARD
APPROVED_AUTO_ULTRA
REJECTED
```

Approval is strategy-specific and effective-dated.

A broker may pass standard-swing execution and fail ultra-scalping execution.

No UI label such as “compatible” may imply that all strategies and execution modes are approved.


# 104. Performance / Load Testing

Test:

```
Tick bursts
Signal bursts
Thousands of WebSockets
License refresh bursts
Admin queries
Historical analytics
Timescale queries
Vector searches
Client reconnect storms

```

Define measurable thresholds before production launch.

---

# 105. Failure / Chaos Testing

Simulate:

```
Market feed failure
Database failure
PgBouncer failure
Valkey failure
AI timeout
News outage
Gateway restart
Network partition
Windows disconnect
MT disconnect
Broker rejection
Clock skew
Invalid messages
Duplicate events
Disk pressure

```

Safe degradation is part of acceptance.

---

# 106. Prediction Validation

Mandatory reports:

```
Backtest
Walk Forward
Out-of-Sample
Paper Trading
Shadow Trading
Limited Live
Calibration
Regime
Session
MFE / MAE
Slippage
Execution Quality
Drawdown
Latency

```

Compare models against explicit baselines.

Do not approve production solely because headline win rate is high.

---


# 106A. XAUUSD Backtest, Walk-Forward and Cost Model

The validation engine shall be event-driven enough for the strategy horizon being tested.

## 106A.1 Cost Model

Include, where applicable:

```text
historical bid/ask spread
commission
slippage distribution
latency
partial fills
order rejection
stop/freeze level
minimum lot/step
market gaps
swap/carry
currency conversion
broker trading hours
```

Ultra-scalping final validation requires tick/bid-ask data or an equivalently defensible execution dataset. OHLC-only positive results are insufficient for production approval.

## 106A.2 Market Mechanics

Model:

```text
GC contract rolls
spot/futures basis
DST
holidays
news events
broker maintenance
session boundaries
candle completeness
feed gaps
```

## 106A.3 Anti-Leakage

Prevent:

```text
look-ahead
future candle access
future-revised macro leakage
label leakage
survivorship bias where relevant
using final daily data before its release time
```

For overlapping labels/time-series ML, use appropriate temporal splitting, purging/embargo where required, and document the method.

## 106A.4 Required Slices

Report by:

```text
strategy
strategy version
session
hour
day of week
regime
volatility quartile
news state
DXY regime
real-yield regime
spread/cost quartile
broker profile
grade
prediction target
```

## 106A.5 Metrics

Trading:

```text
sample size
win/loss
profit factor
expectancy
average R
median R
max drawdown
MFE
MAE
turnover
time in market
TP1/TP2/TP3 rate
stop rate
expiry rate
```

Execution:

```text
fill rate
reject rate
spread
slippage
latency
cost-to-gross-target
```

Calibration:

```text
Brier score
log loss where appropriate
ECE/calibration error
reliability curve
confidence interval
```

Statistical robustness:

```text
bootstrap/confidence intervals
parameter sensitivity
regime stability
multiple-testing awareness
baseline comparison
```

## 106A.6 Baselines

Compare each strategy to defensible baselines such as:

```text
NO-TRADE / zero exposure
simple trend baseline
simple mean-reversion baseline
previous production strategy version
```

Do not approve a complex model solely because its raw hit rate exceeds 50%.

## 106A.7 OOS Lock

Maintain a locked OOS period/data partition that is not repeatedly optimized against.

A suggestion such as “post-2023 OOS” can be a research split, but the final split must be documented based on available dataset chronology and must not be changed simply to improve results.

# 107. Calibration

Where statistically appropriate evaluate:

```
Platt Scaling
Isotonic Regression
Calibration Curves
Brier Score
Reliability Diagrams

```

Measure calibration by:

```
Strategy
Signal Grade
Timeframe
Session
Regime
Prediction Horizon

```

---


# 107A. Strategy Promotion / Demotion Policy

Replace arbitrary gates such as “200 live ticks” with statistically meaningful **completed prediction/trade outcomes**.

Every strategy version shall have a `validation_policy` defining:

```text
minimum_completed_labels
minimum_regime_coverage
minimum_session_coverage
minimum_calibration_quality
minimum_cost_adjusted_expectancy
maximum_drawdown
minimum_execution_quality
maximum_error/incident rate
required shadow duration
required reviewer roles
```

Exact values are strategy-specific, effective-dated and approved.

Promotion sequence:

```text
DRAFT
→ RESEARCH
→ BACKTESTED
→ WALK_FORWARD_VALIDATED
→ OOS_VALIDATED
→ PAPER
→ SHADOW
→ APPROVED
→ ACTIVE
```

Any promotion requires an immutable approval record.

Automatic demotion/safety suspension may occur for:

```text
critical data-quality failure
calibration drift
expectancy deterioration
excess drawdown
execution-quality deterioration
broker mismatch
security incident
model/config integrity failure
```

Automatic demotion may stop new signals/orders; automatic **re-optimization and re-promotion are forbidden**.

# 108. Performance Reporting Integrity

Every report shall capture:

```
Code version
Strategy version
Data version
Start/end dates
Trading costs
Spread assumptions
Slippage assumptions
Latency assumptions
Sample size
In-sample / OOS status

```

Results must be reproducible.

---

# 109. User Dashboard Acceptance Criteria

Complete only when a licensed/subscribed user can:

- securely authenticate;
- enable MFA;
- see real-time XAUUSD;
- see BUY / SELL / NO-TRADE;
- see TTL;
- see entry/SL/TP;
- understand positive and negative evidence;
- inspect signal history;
- inspect performance;
- download signed Windows/MT packages;
- activate a license;
- register a device;
- connect MT4;
- connect MT5;
- run a connection test;
- view device/license status;
- view current subscription, plan price and billing state;
- see exactly the strategy entitlements allowed by the plan;
- change permitted strategy selections through server-side entitlement validation;
- inspect invoices/payment history;
- access referral code/link;
- inspect referral network up to the allowed presentation depth;
- view commission by level, plan and billing-cycle type;
- view pending, cleared, available, paid and reversed commission;
- request a payout only from available balance;
- inspect payout status/history;
- understand the L1-only second-payment rule;
- manage sessions;
- receive configured notifications;
- access support.

Unauthorized strategies and financial mutations must remain inaccessible even if the client/UI is tampered with.

---

# 110. Admin Dashboard Acceptance Criteria

Complete only when authorized administrators can:

- administer users;
- administer roles;
- administer licenses;
- administer devices;
- administer MT accounts;
- manage plans and entitlements;
- manage setup fees and monthly pricing with effective dates;
- inspect subscriptions, invoices, payments, refunds and chargebacks;
- inspect referral attribution and sponsor chains;
- manage future L1-L5 commission rates;
- manage purchase/billing-cycle multipliers and eligible depth;
- confirm the second-payment rule is L1-only;
- inspect complete commission ledgers and rule snapshots;
- hold, release, reverse and adjust commissions through audited operations;
- review and process payout requests;
- view commercial/affiliate reconciliation reports;
- inspect and act on referral/fraud flags;
- inspect signals;
- inspect full decision snapshots;
- monitor execution;
- monitor risk;
- trigger appropriate kill switches;
- manage feature flags;
- manage client releases;
- inspect market-data health;
- inspect AI/provider health;
- inspect infrastructure;
- inspect security events;
- inspect audit records;
- view backup/DR state;
- perform operations without exposing secrets.

Pricing, sponsor, commission and payout changes require server-side RBAC and immutable audit records.

---

# 111. Licensing, Entitlement & Commercial Acceptance Criteria

Licensing is complete only when:

```text
License activation works
Device binding works
Device limit works
Account limit works
MT4 entitlement works
MT5 entitlement works
Signal-only entitlement works
Auto-execution entitlement works
Strategy entitlement works
Expiry works
Renewal works
Suspension works
Revocation works
Reset works
Signed lease works
Revocation propagates
Expired lease is rejected
Client cannot bypass entitlement
Admin audit is complete
```

Commercial entitlement integration is complete only when:

```text
STANDARD grants exactly one allowed standard strategy
PRO grants any two strategies
ELITE grants all four strategies
Subscription activation grants the correct plan entitlements
Past-due/grace/suspension behavior is deterministic
Plan upgrades/downgrades propagate safely
Unauthorized WebSocket strategy streams are rejected
Unauthorized Windows/MT strategy delivery is rejected
Billing failure never bypasses explicit grace/license policy
Referral commission state cannot grant trading privileges
Referral/commission subsystem failure does not block the Go trading path
```

Referral/commission acceptance must additionally satisfy all criteria in Section 69.33.

---

# 112. Production Security Acceptance

Before production:

```
No default passwords
No source-control secrets
MFA for admins
TLS enforced
Secure headers
CSP configured
RBAC tested
Tenant isolation tested
RLS tested where used
Rate limits tested
Command signatures tested
Replay protection tested
Client signatures validated
Audit logging validated
Backup restore tested

```

---

# 113. Production Rollout

Required progression:

```
Stage 0
Historical Research

Stage 1
Backtest

Stage 2
Walk-Forward / OOS

Stage 3
Historical Replay

Stage 4
Paper Trading

Stage 5
Shadow Trading

Stage 6
User Signal Distribution

Stage 7
Licensed MT Signal-Only

Stage 8
Limited Assisted Execution

Stage 9
Limited Automated Execution

Stage 10
Expanded Production

```

Each transition requires explicit approval and evidence.

---

# 114. Repository Audit Before Coding

Before implementing this SOW, inspect the complete repository.

Map:

```
Frontend
Admin Portal
User Portal
Backend
Go Services
Python Services
Database
Migrations
WebSockets
Authentication
Existing License Logic
Existing MT Integration
Existing Signals
Existing Pillars
Master Engine
Risk
News
AI
MCP
Deployment
Monitoring
Tests

```

Also inspect the existing administrative portal before creating another competing admin implementation.

---

# 115. Required Architecture Deliverables Before Modification

Produce:

```
Repository Map
Dependency Map
Service Map
Data Flow
Signal Flow
Control Plane Flow
License Flow
Authentication Flow
User Dashboard Map
Admin Dashboard Map
Database Map
Timescale Map
pgvector Map
Valkey Map
WebSocket Map
MT4/MT5 Map
Windows Agent Map
Risk Map
Deployment Map
Security Boundary Map

```

Classify each component:

```
REUSE
EXTEND
ADAPT
REPLACE WITH JUSTIFICATION
NEW
DEPRECATE

```

---

# 116. No-Damage Rules

Mandatory:

```
DO NOT rewrite existing pillars unnecessarily.

DO NOT create a second competing Master Engine.

DO NOT duplicate working market-data pipelines.

DO NOT replace the Go real-time path with NestJS.

DO NOT place NestJS synchronously in the tick-to-signal critical path.

DO NOT delete existing functionality without migration.

DO NOT break existing APIs without versioning.

DO NOT perform destructive database changes without backup/migration.

DO NOT expose private credentials to Next.js.

DO NOT put AI keys inside MT4/MT5.

DO NOT put database credentials inside Windows clients.

DO NOT put signing private keys inside clients.

DO NOT trust client-side authorization.

DO NOT trust a license key string alone.

DO NOT assume all brokers use identical XAUUSD specifications.

DO NOT allow AI to override hard risk.

DO NOT mutate live strategy parameters automatically.

DO NOT execute expired signals.

DO NOT execute duplicate commands.

DO NOT claim arbitrary score values are probabilities.

DO NOT publish unverified accuracy claims.

DO NOT activate live autonomous execution before staged validation.

```

---

# 117. Implementation Phases

## Phase 1 — Full Repository Reverse Engineering

Audit existing architecture and identify reusable components. Do not begin commercial/referral rewrites until current billing, plan, license, user, database and dashboard behavior is mapped.

## Phase 2 — Canonical Architecture

Freeze boundaries between:

```text
Go
Python
NestJS
Next.js
Windows
MQL
Database
Billing
Referral / Commission
Payout
```

## Phase 3 — Database Foundation

Implement/validate:

```text
PostgreSQL 17
TimescaleDB
pgvector
PgBouncer
Valkey
Canonical migrations
Roles
Schemas
Financial decimal policy
```

## Phase 4 — IAM / NestJS

Implement/validate:

```text
Users
Roles
MFA
Sessions
Organizations
Audit
```

## Phase 5 — Plans, Pricing & Subscription Billing

Implement:

```text
STANDARD $19 + $99/month
PRO $29 + $499/month
ELITE $39 + $999/month
Plan versioning
Strategy entitlements
Subscription lifecycle
Invoices
Payments
Signed webhooks
Idempotency
Refunds
Chargebacks
```

## Phase 6 — Multi-Level Referral & Commission Engine

Implement:

```text
Referral codes
Referral attribution
Five-level sponsor tree
Commission rule database
Purchase/payment numbering
100% / 75%-L1-only / 50% rules
Commission ledger
Wallet summaries
Refund/chargeback reversal
Affiliate risk
```

## Phase 7 — Payout System

Implement:

```text
Available balance
Withdrawal requests
Risk/compliance review state
Approval
Processing
Provider reference
Reconciliation
Failure/retry
Audit
```

## Phase 8 — Licensing

Implement:

```text
Plans
Entitlements
Licenses
Devices
Device keys
Activation
Signed leases
Revocation
Strategy entitlement propagation
```

## Phase 9 — User Dashboard

Implement the full Next.js User Portal including subscription, strategy selection, referral center, commission ledger and payout views.

## Phase 10 — Admin Dashboard

Extend/rebuild the existing Admin Portal without duplicating working functionality. Add plans, billing, referral network, commission controls, payout operations and finance reporting.

## Phase 11 — Historical Data

Implement:

```text
Backfill
Gap detection
Candle completeness
Timescale retention/aggregation
```

## Phase 11A — Freeze Four Strategy Specifications Before Real-Time Coding

Before modifying the production signal engine, create and approve versioned playbooks for:

```text
STANDARD_SCALPING
ULTRA_SCALPING
STANDARD_SWING
TREND_SWING
```

Each playbook must freeze its initial:

```text
prediction targets
timeframes
session policy
required data capabilities
feature definitions
confluence profile
risk profile
execution profile
backtest assumptions
calibration policy
promotion policy
```

Do not begin production activation of Phase 12 until these definitions exist in code/config + tests.

## Phase 12 — Real-Time Quantitative Engine

Implement and validate the XAUUSD market engines using the frozen/versioned strategy definitions.

Required sub-phases:

```text
12.1 Data capability registry
12.2 GC futures/roll/basis integration where licensed
12.3 Session/calendar engine
12.4 Timeframe health/alignment
12.5 Structure engine
12.6 Liquidity pool/sweep engine
12.7 FVG/IFVG/OB/breaker/liquidity-void engine
12.8 VWAP/volume-profile engine
12.9 Flow/CVD engine where data-capable
12.10 Gold macro engine
12.11 Strategy confluence engine
12.12 Per-strategy cost/risk gates
12.13 Deterministic signal snapshots
12.14 Replay/golden-test verification
```

## Phase 13 — Prediction / Calibration

Implement formal prediction targets and calibrated probability.

## Phase 14 — AI / Research

Complete controlled Python intelligence integrations.

## Phase 15 — Signal Distribution

Secure web/user signal streaming with plan/strategy entitlement enforcement.

## Phase 16 — Windows Agent

Implement activation, secure stream, entitlement refresh, IPC and lifecycle.

## Phase 17 — MT4 / MT5

Implement and validate both EAs, including strategy entitlement rejection.

## Phase 18 — Signed Client Distribution

Implement:

```text
Installer
Signing
Release channels
Updates
Rollback
```

## Phase 19 — Observability / Security

Implement monitoring, tracing, alerts and security controls for both trading and commercial planes.

## Phase 20 — Commercial / Referral Validation

Run:

```text
Billing webhook tests
Duplicate event tests
Plan entitlement tests
Commission matrix tests
Second-payment L1-only tests
Referral cycle/self-referral tests
Refund tests
Chargeback tests
Wallet reconciliation
Payout reconciliation
Financial precision tests
Abuse/fraud tests
Load tests
Failure recovery tests
```

## Phase 21 — Trading Validation

Run:

```text
Backtest
Walk Forward
OOS
Paper
Shadow
Load
Failure
Security
```

## Phase 22 — Licensed Signal Production

Enable real subscribed clients without automated execution.

## Phase 23 — Limited Live Execution

Enable individually approved accounts only.

## Phase 24 — General Production

Expand only after measurable trading, security, billing, commission, payout and reconciliation acceptance gates are met.

---

# 118. Required Deliverables

Deliver:

```text
Architecture Documentation
Repository Audit
Repository AGENTS.md / nested instruction files where needed
Codex project subagent definitions
Codex project Skills
Codex plugin/MCP usage policy
XAUUSD four-strategy playbooks
Strategy Definition Registry
Indicator/Feature Parameter Registry
Confluence Profiles
Risk/Execution Profiles
Gold Session/Calendar Specification
Liquidity/SMC Feature Specification
GC Futures/Flow/Basis Specification
XAUUSD Backtest & Cost Model Specification
Strategy Promotion/Calibration Specification
Database ERD
API Specification
WebSocket Specification
Signal Schema
Prediction Schema

Commercial Plan Specification
Pricing / Plan Versioning Specification
Subscription Lifecycle Specification
Billing Webhook Specification
Payment Idempotency Specification
Strategy Entitlement Matrix

Referral Program Specification
Referral Attribution Specification
Five-Level Referral Tree
Commission Rule Specification
Commission Calculation Engine
Commission Ledger Specification
Affiliate Wallet / Balance Model
Refund / Chargeback Commission Policy
Payout System Specification
Affiliate Risk / Anti-Abuse Specification
Finance / Commission Reconciliation Report

License Specification
Device Protocol
Windows Agent
MT4 EA
MT5 EA
User Dashboard
Admin Dashboard
Referral User Dashboard
Admin Commission Control Center
Payout Operations UI

Authentication
RBAC
MFA
Licensing
Entitlements
Historical Data Pipeline
TimescaleDB Architecture
pgvector Architecture
PgBouncer Configuration
Valkey Architecture
Monitoring
Alerting
Backup
DR Documentation
Security Documentation
Installer
Update System

Admin Manual
User Manual
Affiliate / Referral Manual
Billing & Subscription Guide
Commission & Payout Operations Runbook
Finance Reconciliation Runbook
MT4 Setup Guide
MT5 Setup Guide
Operations Runbook
Incident Runbook
Release Runbook
Validation Reports
```

All deliverables must reflect the same canonical production plan and commission configuration; no contradictory plan/rate tables may exist across code, docs or UI.

---

# 119. Definition of Production Ready

Predict-A-Trade shall not be considered production ready merely because it generates BUY/SELL signals or successfully charges a subscription.

Production readiness requires:

```text
Correctness
Determinism
Calibration
Data quality
Security
Licensing
Authentication
Authorization
Plan/Strategy Entitlement Correctness
Subscription Billing Correctness
Payment Idempotency
Referral Attribution Integrity
Commission Calculation Correctness
Financial Precision
Ledger Auditability
Refund/Chargeback Reconciliation
Payout Reconciliation
Anti-Abuse Controls
Explainability
Risk control
Execution safety
Replayability
Client integrity
Monitoring
Backups
Disaster recovery
Operational documentation
Validated user experience
Validated admin operations
Validated finance/affiliate operations
```

A trading subsystem incident must not corrupt billing/commission state.

A billing/referral/commission incident must not block the Go real-time market/signal path.

No production launch is complete until finance can reconcile:

```text
validated customer payments
subscription state
commissionable revenue
commission ledger
available affiliate balances
payouts
refunds
chargebacks
adjustments
```

to an explainable, repeatable result.

---

# 120. Final Production Objective

The completed trading platform shall operate as:

```text
OBSERVE
    ↓
VALIDATE
    ↓
UNDERSTAND
    ↓
CLASSIFY
    ↓
PREDICT
    ↓
SCORE
    ↓
CALIBRATE
    ↓
VERIFY
    ↓
CONTROL RISK
    ↓
DECIDE
    ↓
DISTRIBUTE
    ↓
ENTITLE
    ↓
LICENSE
    ↓
EXECUTE IF AUTHORIZED
    ↓
RECONCILE
    ↓
MONITOR
    ↓
MEASURE
    ↓
AUDIT
    ↓
RESEARCH
    ↓
IMPROVE THROUGH CONTROLLED RELEASES
```

Its commercial/customer lifecycle shall operate independently but consistently as:

```text
REGISTER / ATTRIBUTE REFERRAL
    ↓
SELECT PLAN
    ↓
PAY SETUP + MONTHLY SUBSCRIPTION
    ↓
VALIDATE PAYMENT
    ↓
ACTIVATE SUBSCRIPTION
    ↓
GRANT STRATEGY ENTITLEMENTS
    ↓
ISSUE / REFRESH LICENSE
    ↓
DELIVER AUTHORIZED FEATURES
```

When a validated eligible payment has a sponsor chain, the financial sidecar lifecycle shall be:

```text
VALIDATED ELIGIBLE SUBSCRIPTION PAYMENT
    ↓
DETERMINE PAYMENT NUMBER
    ↓
LOAD PLAN BASE RATES
    ↓
APPLY PURCHASE MULTIPLIER
    ↓
APPLY ELIGIBLE REFERRAL DEPTH
    ↓
CREATE IMMUTABLE COMMISSION LEDGER
    ↓
PENDING
    ↓
CLEARED
    ↓
AVAILABLE
    ↓
PAYOUT
    ↓
RECONCILE
    ↓
AUDIT
```

The critical initial referral rule remains:

```text
1st eligible monthly payment -> L1-L5 at 100%
2nd eligible monthly payment -> L1 only at 75%
3rd+ eligible monthly payment -> L1-L5 at 50%
```

Predict-A-Trade shall become a **complete production-grade XAUUSD Prediction Intelligence and Subscription Platform**, not an indicator collection, not an uncontrolled trading bot, and not an unaudited commission script.

Its core characteristics shall remain:

```text
Deterministic
Real-Time
Low-Latency
Broker-Independent
Multi-Timeframe
Risk-Aware
AI-Assisted
Calibrated
Explainable
Auditable
Replayable
License-Controlled
Subscription-Aware
Strategy-Entitlement-Aware
Referral-Aware
Financially Precise
Payout-Reconcilable
Secure
Fault-Tolerant
Operationally Observable
Production-Ready
```

The ultimate objective is not to promise a fixed percentage of winning trades.

The objective is to **continuously measure, validate and improve statistically defensible XAUUSD decision quality while protecting users and the platform through hard data-quality, licensing, subscription, financial, security, risk, commission and execution controls.**

---

---


# 121. Codex Engineering Execution Contract

This SOW is intended to be executable by Codex against an **existing repository**.

Codex shall not begin with a greenfield rewrite.

## 121.1 Mandatory First Action

Before code changes:

1. Read the repository root instructions.
2. Recursively inspect the project architecture.
3. Identify actual build/test/lint/migration/deployment commands.
4. Produce the architecture deliverables required by Sections 114–115.
5. Classify each relevant component as:

```text
REUSE
EXTEND
ADAPT
REPLACE_WITH_JUSTIFICATION
NEW
DEPRECATE
```

6. Find the existing Master Engine, trading path, admin portal, user portal, billing/license logic, database migrations, MT integration and monitoring before creating anything that overlaps.
7. Create a change plan mapping SOW requirement → existing component → proposed files → tests → migration/rollback.
8. Only then implement.

## 121.2 Repository Instruction Hierarchy

Codex shall use a root `AGENTS.md` as the project constitution.

After the repository audit, create nested `AGENTS.md` files **only in real repository directories that need specialized rules**.

Typical logical scopes may include:

```text
Go real-time engine
Python research
NestJS control plane
Next.js web
database/migrations
Windows Agent
MT4
MT5
operations/infra
```

Do not invent directory paths before the audit.

Root `AGENTS.md` must contain:

```text
architecture boundaries
no-damage rules
security rules
financial precision rules
trading/risk invariants
approved build/lint/test commands
migration rules
secret-handling rules
generated-file rules
definition of done
```

Nested rules may refine but never weaken root safety/security/risk rules.

## 121.3 Worktree / Change Isolation

For large concurrent work, use isolated branches/worktrees where supported.

Never allow two writing agents to edit the same bounded component simultaneously without explicit coordination.

Prefer parallel subagents for:

```text
repository exploration
read-only audits
test review
security review
documentation research
independent validation
```

Serialize changes that touch:

```text
canonical migrations
shared schemas
Master Engine
risk engine
public API contracts
billing ledger
commission ledger
entitlement protocol
```

## 121.4 No Autonomous Production Mutation

During implementation, Codex must not:

```text
connect to or trade a live broker account
enable live automated execution
rotate production signing keys
change production DNS
run destructive production migrations
modify real commission balances
issue real payouts
use production payment credentials
silently change ACTIVE strategy parameters
```

without a separate explicit production authorization workflow outside normal coding.

Use local/test/staging fixtures and scoped credentials.


---


# 122. Required Codex Skills

Create project skills under the repository's Codex skill directory (normally `.agents/skills/<skill-name>/SKILL.md`) after auditing existing skills and reusing compatible ones.

Each skill shall have a concise activation description, workflow, validation checklist, and references/scripts only when useful.

The project requires these capabilities:

## 122.1 `repo-audit`

Purpose:

```text
reverse-engineer repository
map services/dependencies/data flows
locate duplicate/legacy systems
classify REUSE/EXTEND/ADAPT/REPLACE/NEW/DEPRECATE
produce change-impact map
```

## 122.2 `architecture-guardrails`

Enforce:

```text
Go authoritative real-time path
Python research/asynchronous intelligence
NestJS control plane
Next.js presentation
Windows/MQL edge execution
no synchronous referral/billing dependency in tick path
```

## 122.3 `xauusd-market-data`

Cover:

```text
spot bid/ask
historical candles/ticks
data quality
GC futures capability
contract roll
basis
session/calendar
macro feeds
provider provenance/licensing
```

## 122.4 `xauusd-strategy-spec`

Create/validate:

```text
four strategy definitions
timeframe profiles
feature parameters
confluence profiles
risk profiles
execution profiles
prediction targets
versioning
```

## 122.5 `xauusd-quant-validation`

Cover:

```text
golden feature tests
backtesting
walk-forward
OOS
paper/shadow
cost model
calibration
drift
promotion evidence
performance-claim integrity
```

## 122.6 `trading-risk-safety`

Enforce:

```text
hard veto precedence
aggregate exposure
spread/slippage/cost gates
margin checks
news/session safety
kill switches
TTL/idempotency
execution permission
```

## 122.7 `mt4-mt5-windows`

Cover:

```text
Windows service
device/license protocol
signed signals
local IPC
MT4/MT5 EA
broker symbol mapping
terminal specifications
idempotent execution
installer/update/signing
```

## 122.8 `control-plane-saas`

Cover:

```text
NestJS IAM
MFA/RBAC
subscriptions
billing/webhooks
licenses
entitlements
referrals
commissions
payouts
audit
```

## 122.9 `frontend-trading-ui`

Cover:

```text
Next.js user/admin portals
server-side authorization
real-time state
XAUUSD chart
explainability
billing/referral UI
accessibility/responsiveness
honest empty/degraded states
```

## 122.10 `database-migrations`

Cover:

```text
PostgreSQL 17
TimescaleDB
pgvector
PgBouncer
Valkey contracts
canonical migrations
least-privilege roles
financial decimals
rollback/backup safety
```

## 122.11 `api-contracts`

Cover:

```text
OpenAPI
WebSocket schemas
versioning
idempotency
error contracts
entitlement enforcement
backward compatibility
```

## 122.12 `security-supply-chain`

Cover:

```text
OWASP ASVS baseline
threat modeling
secret scanning
SAST/dependency audit
SBOM
signing
update manifests
key separation/rotation
client supply chain
```

## 122.13 `observability-sre`

Cover:

```text
OpenTelemetry
Prometheus
Grafana
structured logs
SLOs
alerts
HA
backup/restore
DR
chaos/failure tests
```

## 122.14 `release-gates`

Enforce:

```text
research → backtest → OOS → replay → paper → shadow → signals → limited execution
evidence checklist
approval records
rollback readiness
```

## 122.15 `docs-runbooks`

Maintain:

```text
architecture docs
ADRs
API docs
strategy playbooks
admin/user manuals
MT setup guides
incident/runbook
release/rollback
finance reconciliation
validation reports
```

Skills are reusable procedure packs, not permission to bypass the SOW.


---


## 122.16 `broker-execution-qualification`

Cover:

```text
broker symbol economics
pip/point/tick normalization
generalized position sizing
margin/stop-out checks
order-type semantics
missed/partial fill modeling
latency/jitter measurement
VPS/locality qualification
spread/slippage/rejection statistics
strategy-specific broker approval
```

This skill must challenge fixed broker assumptions and require measured evidence.


# 123. Required Codex Subagents

Create project-scoped subagent definitions under `.codex/agents/` where supported. Reuse existing equivalent agents instead of duplicating them.

## 123.1 `repo_explorer`

Read-mostly.

Responsibilities:

```text
full repository map
dependency/service map
find existing implementations
identify dead/duplicate code
collect build/test commands
```

No architecture-changing writes.

## 123.2 `platform_architect`

Read-mostly / design authority.

Responsibilities:

```text
four-plane boundary review
ADR creation
integration design
schema/API boundary review
migration sequencing
conflict resolution
```

## 123.3 `go_realtime_engineer`

Write scope limited to the Go real-time/data/signal/risk domains assigned by the parent.

Responsibilities:

```text
ingestion
normalization
feature engines
strategy scoring
risk
signal lifecycle
real-time gateway
reconciliation
```

Must not implement billing/referral business truth inside Go tick path.

## 123.4 `python_quant_researcher`

Responsibilities:

```text
dataset tooling
backtest
walk-forward/OOS
calibration
ML/NLP research
feature studies
drift analytics
validation reports
```

May generate candidate parameters but cannot activate production settings.

## 123.5 `nestjs_control_engineer`

Responsibilities:

```text
IAM
subscriptions
billing
referrals/commission
payout
license/control APIs
audit
admin operations
```

Financial operations must remain exact-decimal, transactional and idempotent.

## 123.6 `nextjs_frontend_engineer`

Responsibilities:

```text
public site
user portal
admin portal
real-time terminal
charts
license/MT setup
billing/referral/payout UI
accessibility/responsive UX
```

No client-only authorization.

## 123.7 `database_engineer`

Responsibilities:

```text
schema/ERD
migrations
Timescale policies
indexes
constraints
roles
PgBouncer
Valkey contracts
query performance
backup-aware migration plan
```

Canonical migration sequence only.

## 123.8 `windows_mql_engineer`

Responsibilities:

```text
Go Windows Service
signed update path
IPC
MT4 EA
MT5 EA
broker compatibility
terminal execution guards
```

No server/database/private signing credentials in clients.

## 123.9 `quant_validator`

Independent reviewer.

Responsibilities:

```text
challenge leakage
challenge cost assumptions
verify strategy distinctions
verify calibration
verify sample sufficiency
verify baseline comparisons
reject unsupported performance claims
```

Prefer read-only access to production code/settings.

## 123.10 `security_reviewer`

Independent read-mostly reviewer.

Responsibilities:

```text
threat model
auth/session review
RBAC/tenant isolation
secret/key review
client protocol review
payment/referral abuse review
supply-chain review
security acceptance
```

## 123.11 `qa_reliability_engineer`

Responsibilities:

```text
unit/integration/E2E
golden tests
load
chaos
reconnect/replay
Windows/MT matrix
migration tests
backup/restore verification
```

## 123.12 `release_manager`

Read-mostly final gate.

Responsibilities:

```text
collect test evidence
check migrations/backups
check security report
check quant validation
check commercial reconciliation
check release artifacts/checksums
produce GO/NO-GO report
```

The release manager cannot waive hard risk/security/financial correctness failures.

## 123.13 `broker_execution_validator`

Independent read-mostly/test-focused reviewer.

Responsibilities:

```text
validate broker profile against terminal data
validate price-unit conversions
validate tick value / contract size
validate position sizing
validate margin and stop-out behavior
validate order-type behavior
measure execution latency and slippage
challenge “raw/zero spread” marketing assumptions
approve/reject broker-strategy execution class
```

Must not place live trades unless a separately authorized controlled qualification environment explicitly permits it.


## 123.14 Coordination Rule

The parent Codex session owns the integrated plan.

Subagents return:

```text
findings
files examined
files changed
tests run
unresolved risks
recommended next action
```

The parent resolves cross-domain conflicts before merge.


---


# 124. Required Plugins, Connectors and MCP Policy

Plugins/tools extend Codex capability; they do not become architectural dependencies of the production trading path.

## 124.1 Required Where Applicable

### GitHub

Use the GitHub integration/plugin when the repository is hosted on GitHub for:

```text
repository context
issues
pull requests
CI status
review workflow
release traceability
```

If the project uses another SCM, use its equivalent integration. Lack of GitHub must not block local implementation.

### Codex Security

Use the Codex Security plugin/integration for independent codebase security scanning/review before production release, in addition to normal SAST/dependency/secret scanning.

## 124.2 Strongly Recommended

### Documentation / Primary-Source MCP or Connector

Use only authoritative primary documentation for version-sensitive engineering decisions such as:

```text
Next.js
NestJS
Go
Python
PostgreSQL
TimescaleDB
pgvector
Valkey
MetaTrader/MQL
market-data providers
payment providers
```

Never copy a web snippet into production without checking version/applicability.

### Browser / Playwright Tooling

Use Playwright or equivalent browser automation for:

```text
authentication flows
user portal
admin portal
entitlement visibility
chart interactions
responsive UI
billing/referral workflows in test mode
```

This can be local tooling or an available MCP/plugin; it is not required to be a production service.

### Sentry

Use only if Sentry is part of the project observability stack. The canonical observability requirement remains OpenTelemetry + Prometheus + Grafana + structured logs. Do not add a second telemetry stack merely because a plugin exists.

## 124.3 Optional Workflow Integrations

Project-management/chat integrations may be used for issue/release coordination but are not necessary to satisfy this SOW.

## 124.4 Forbidden Tooling Practice

Do not give Codex/tool plugins unrestricted access to:

```text
production broker execution
production payment mutation
production payout mutation
production signing private keys
database superuser
live secret vault export
```

Use least-privilege test/staging credentials.

Any connector capable of consequential production writes must be separately authorized and audited.


---


# 125. Codex Build Loop and Definition of Done

For every implementation slice Codex shall follow:

```text
READ
→ MAP REQUIREMENT
→ DESIGN MINIMUM COMPATIBLE CHANGE
→ WRITE/UPDATE TEST
→ IMPLEMENT
→ LINT/FORMAT
→ UNIT TEST
→ INTEGRATION TEST
→ SECURITY/BOUNDARY CHECK
→ DOCUMENT
→ REPORT EVIDENCE
```

## 125.1 Do Not Stop at “Code Compiles”

A feature is complete only when applicable evidence exists for:

```text
implementation
migration
backward compatibility
unit tests
integration tests
E2E tests
security
observability
documentation
rollback
acceptance criteria
```

Trading logic additionally requires:

```text
deterministic fixtures
historical replay
cost-aware backtest
walk-forward/OOS
calibration
paper/shadow
```

before live activation.

## 125.2 Required Status Report After Each Phase

Codex shall report:

```text
Phase
Requirements completed
Files created/changed
Database migrations
Tests executed + result
Security checks
Performance checks
Quant validation status
Backward-compatibility impact
Known limitations
Deferred items with reason
Rollback path
Next phase
```

Never write “complete” when an acceptance criterion remains unverified.

## 125.3 Stopping Condition

The coding task is not complete because the repository builds.

The final output must include a traceability matrix:

```text
SOW requirement
implementation file(s)
test(s)
migration(s)
dashboard/API surface
observability
status
evidence
```

All P0 gap closures and production safety gates must be `PASS` before claiming production readiness.


---


# 126. Final Integrated Acceptance — XAUUSD Gap Closure

The gap closure is complete only when all of the following are proven:

1. The four strategies produce **distinct versioned behavior**, not four entitlement labels over one generic signal.
2. Every strategy has explicit prediction targets, timeframes, sessions, feature/confluence profile, risk profile, execution profile and calibration policy.
3. All numeric strategy thresholds are configuration-backed, versioned, auditable and historically reproducible.
4. Required timeframe health is continuously enforced.
5. BSL/SSL liquidity pools and sweep events are machine-defined and testable.
6. BOS/CHoCH/MSS, FVG/IFVG, OB/breaker, displacement and liquidity-void semantics have deterministic fixtures.
7. Spot tick volume is never mislabeled as centralized volume.
8. GC futures flow, where used, handles contract selection, roll, basis, provenance and data capability correctly.
9. CVD/depth/order-flow features are never fabricated from incapable feeds.
10. Sessions/fixes/holidays use timezone/DST-aware calendars.
11. Research concepts such as killzones/Power-of-Three/day-of-week tendencies remain experimental until validated.
12. Macro scoring separates DXY, real yields, events, positioning and slow-flow context with freshness/quality.
13. News policy is per strategy and event tier.
14. Spread, slippage, cost-to-target, margin and broker-spec gates are hard execution controls.
15. XAUUSD exposure is aggregated across strategies at account/risk level.
16. Broker symbols, contract sizes, lot steps, stop/freeze levels, fill modes, swaps and trading sessions are broker-profile driven.
17. Ultra-scalping production validation uses tick/bid-ask execution-quality evidence.
18. Backtests include realistic costs and prevent look-ahead/leakage.
19. Walk-forward/OOS reports are sliced by strategy, regime, session and volatility.
20. Probability is calibrated to an explicit target; raw score is never marketed as probability.
21. Grades are calibration-policy outputs and remain unrated until sample sufficiency exists.
22. Promotion is evidence-based and human-approved; no model/optimizer self-promotes.
23. Chart overlays come from the same server-side state used by the decision engine.
24. User/admin/Windows/MT delivery enforces strategy entitlement server-side.
25. Signal protocol remains signed, TTL-bound, replay-safe and idempotent.
26. Hard risk can veto AI, quant score, entitlement and execution.
27. Commercial billing/referral/commission failures cannot block the real-time Go trading path.
28. Trading failures cannot corrupt financial ledgers.
29. Gold-specific telemetry, drift and NO-TRADE reasons are observable.
30. Codex repository instructions, skills, subagents and security/release gates exist and are used.
31. CI/CD, security, backup/restore, DR, client signing and release rollback gates pass.
32. No performance/accuracy/profitability claim is published without its approved evidence package.

33. Pip/point/tick display and all risk math are broker-profile normalized; no universal XAUUSD pip or 100-ounce assumption remains in execution-critical code.
34. Position sizing uses validated tick value/contract economics and includes expected execution costs.
35. Margin headroom and broker stop-out behavior are validated before auto-execution.
36. Order-type policy models missed/partial fills and does not claim limit orders guarantee fills.
37. Ultra-scalping broker eligibility includes measured latency/jitter/slippage/rejection evidence and can fail independently of swing eligibility.
38. Supporting-research variants such as EMA200, engulfing, CVD/VWAP, overlap breakouts and Judas-swing behavior are versioned research/strategy features rather than hard-coded doctrine.
39. Backend and frontend are fully wired through real API/WebSocket contracts with no fake live data or client-only authorization.
40. Full-stack E2E journeys for authentication, entitlement, signals, licensing, billing/referral, payout and admin operations pass in test/staging.

Only after these conditions and all earlier Sections 1–120 acceptance criteria are satisfied may Predict-A-Trade be described as **production-ready**.

---

# 127. Full Backend + Frontend Completion Contract

This SOW is a **full-stack implementation contract**. Codex must not treat the trading engine, database, control API, or frontend as optional follow-up work.

## 127.1 Backend Completion

Backend completion requires the integrated implementation and verification of:

```text
Go:
- market ingestion
- normalization
- candle aggregation
- feature engines
- regime
- four distinct strategy engines
- scoring
- calibration consumption
- hard risk
- signal lifecycle
- real-time gateway
- execution authorization
- reconciliation
- health/metrics

Python:
- historical import/backfill tooling
- dataset provenance
- feature research
- backtesting
- walk-forward/OOS
- calibration
- model evaluation
- NLP/news intelligence where configured
- drift analytics
- reproducible validation reports

NestJS:
- authentication/MFA/session lifecycle
- users/roles/organizations
- plans/pricing
- subscriptions/billing
- payments/webhooks
- strategy entitlements
- licensing/devices/MT accounts
- downloads/releases
- referrals/commissions/wallets/payouts
- support/notifications
- audit/config/feature flags
- admin operations
- dashboard/query APIs
- health endpoints

Data:
- PostgreSQL schemas/roles
- TimescaleDB hypertables/policies
- pgvector where justified
- PgBouncer
- Valkey contracts
- canonical migrations
- backup/restore-aware schema evolution
```

All externally reachable backend endpoints must have:

```text
schema validation
authentication/authorization where required
rate limits
consistent error contract
OpenAPI documentation
audit where consequential
idempotency where mutating/financial
tests
metrics/logging/tracing
```

## 127.2 Frontend Completion

Frontend completion requires production Next.js implementations for:

```text
Public:
- landing/product
- plans/pricing
- strategy descriptions without performance guarantees
- legal/risk disclosures
- authentication entry
- download/public release information where appropriate

User:
- overview
- real-time signal terminal
- XAUUSD chart
- strategy selector constrained by entitlement
- signal explainability
- signal history
- performance/validation views
- market/data quality
- MT4/MT5 setup wizard
- license/device/MT-account management
- subscription/billing
- invoices/payments
- referral center
- commission analytics
- payout requests/history
- notifications
- security/session/MFA
- support tickets
- downloads/releases

Admin:
- operations overview
- users/roles
- plans/pricing/entitlements
- subscriptions/payments/refunds/chargebacks
- referrals/sponsor chains
- commission rules/ledger
- payout operations
- licenses/devices/MT accounts
- strategy registry/versions
- risk center/kill switches
- signals/decision snapshots
- execution/reconciliation
- market data/providers
- broker execution qualification
- macro/news
- AI/providers/models
- feature flags/configuration
- notifications/support
- infrastructure/health
- observability links/status
- security events
- audit
- backups/DR status
- client releases
- finance/reconciliation reports
```

## 127.3 Frontend Quality Requirements

Every page must implement as applicable:

```text
server-side authorization
responsive desktop/tablet/mobile layout
keyboard navigation
accessible labels and focus
WCAG-oriented contrast
loading state
empty state
degraded-data state
error state
permission-denied state
subscription/entitlement state
reconnect state for live channels
optimistic UI only where safe
no fabricated chart/signal data
timezone-aware display
consistent numeric formatting
exact decimal formatting for money
```

No protected page may rely on hiding buttons as its authorization mechanism.

## 127.4 API-to-UI Traceability

Every production dashboard widget must identify its source:

```text
REST endpoint
WebSocket channel/event
database/report source
freshness
error/degraded behavior
permission/entitlement requirement
```

Do not ship dead widgets, placeholder cards, fake KPIs, static “live” charts or hard-coded success counters.

## 127.5 Full-Stack E2E Journeys

At minimum automate the following end-to-end journeys in test/staging:

```text
register/login/MFA/logout
password/session recovery flow
view plans
subscribe with test provider
change strategy selection within plan
reject unauthorized strategy
receive real-time entitled signal
display signal + chart + evidence
activate license
register device
bind MT account
download signed client artifact
support reconnect behavior
referral attribution
commission creation from eligible payment
second-payment L1-only rule
payout request from available balance
admin approve/reject payout
admin suspend license
admin revoke device
admin inspect signal decision snapshot
admin trigger non-production kill switch/test control
refund/chargeback reconciliation
frontend permission-denied behavior
frontend degraded market-data behavior
```

## 127.6 No Partial-Completion Claim

Codex must not declare the project complete when:

```text
backend exists but frontend is incomplete
frontend uses mocks because APIs are incomplete
APIs exist but are not wired to UI
UI exists but entitlement/RBAC is client-only
migrations are missing
tests are missing
WebSocket reconnect/replay is untested
financial reconciliation is unverified
strategy validation is unfinished
broker execution qualification is unfinished for auto-execution
```

When an external dependency is unavailable (licensed market data, payment credentials, code-signing certificate, broker test account, etc.), implement the adapter/interface, configuration, validation, tests with fixtures, and an honest disabled/degraded state. Record the dependency as an external activation blocker; never fabricate successful integration.

---

# 128. Supporting-Research Normalization Rules

The supporting XAUUSD execution research is incorporated into this SOW under the following production interpretation:

1. DOM/order-book signals require actual depth capability; broker tick volume is not a substitute.
2. CVD requires defensible aggressor-side trade classification or a clearly labeled estimated derivation.
3. CVD + Session VWAP, liquidity-sweep + CHoCH, FVG mitigation, volume profile, EMA200, engulfing candles, weekly-cycle ideas and “Judas Swing” are versioned hypotheses/features, not guaranteed edges.
4. Fixed Dubai/GMT+4 “golden windows” are presentation examples only. Production uses timezone/DST-aware session definitions and converts to user timezone dynamically.
5. News windows are strategy/event-tier policies. Open positions are not universally flattened unless the approved policy requires it.
6. Cost examples such as specific spread/commission/slippage values are seed research examples only. Production uses measured broker/account/session distributions.
7. A 100-ounce contract is not universal. Risk math is broker-profile driven.
8. Pip/point language is never used as the sole internal risk unit.
9. High leverage is not an edge and is not a risk control.
10. Limit orders constrain price but do not guarantee fills.
11. VPS proximity can reduce latency but cannot guarantee execution price; execution quality is measured.
12. Swap-free account labels are not assumed to mean zero economic carrying cost.
13. Day-of-week patterns are categorical research features until independently validated.
14. Trend/macro relationships such as DXY/real-yield direction are dynamic features, not permanent deterministic rules.
15. Any strategy target, stop, frequency, session or indicator value from research is a versioned seed parameter subject to the validation and promotion policy of this SOW.

---

# 129. Codex Full-Project Execution Directive

Codex shall execute this SOW as a complete repository engineering program, not as a design-only exercise.

Required behavior:

```text
1. Audit the existing repository first.
2. Preserve working behavior and data.
3. Reuse/extend before replacing.
4. Create/update AGENTS.md and project-scoped skills/subagents after the audit.
5. Build database foundation and contracts first.
6. Build backend services and migrations.
7. Build Next.js public/user/admin frontends against real contracts.
8. Build/complete real-time Go + research Python integration.
9. Build/complete licensing, Windows Agent and MT4/MT5 integration.
10. Wire observability/security/backup/release controls.
11. Run unit, integration, E2E, replay, load, failure and security tests.
12. Resolve failures rather than merely documenting them when they are implementable.
13. Never mutate live trading, live payments, real payouts or production signing keys without explicit authorization.
14. Continue through all implementation phases until every implementable P0/P1 requirement has evidence.
15. Produce a final SOW-to-code traceability matrix and GO/NO-GO report.
```

The repository is not complete because a subset compiles. Backend, frontend, contracts, database, tests, security, observability and documentation must all be mutually wired.---

# 130. Prediction Accuracy, Calibration, Performance Evidence and Delivery SLO Governance

This section operationalizes the accuracy/calibration requirements already present in Sections 15, 15A, 16, 17A, 78, 89, 106, 107 and 108.

## 130.1 “Accuracy” Must Be a Named, Reproducible Metric

The platform shall not use a generic `accuracy` field as a proxy for trading quality.

Every reported predictive metric must identify:

```text
strategy_id
strategy_definition_id
prediction_target_id
calibration_version
dataset_id
dataset_version
broker/execution profile where applicable
evaluation period
completed label count
cost model version
exit_profile_id
OOS / walk-forward / paper / shadow status
metric definition
point estimate
confidence interval where applicable
report_id
```

Valid distinct concepts include:

```text
P(TP1 before SL within horizon)
P(TP2 before SL within horizon)
P(TP3 before SL within horizon)
expected net return within horizon
expected R
TP1 / TP2 / TP3 hit rate
Brier score
ECE
profit factor
maximum drawdown
MFE / MAE
delivery success rate
service availability
execution reconciliation success
```

A raw confluence/quant score is not a probability.

## 130.2 No Universal “98% Prediction Win Rate” Requirement

No strategy, plan, subscription or user tier shall be coded, advertised, graded or activated on the assumption that 98% of trades must win.

A prediction-quality claim such as “98% accurate” is prohibited unless it has an approved evidence package under Section 130.6 and Section 78 and is legally approved for the applicable jurisdiction.

All three commercial plans shall receive the same integrity of measurement and operational delivery for the strategies to which they are entitled, but they shall not be assigned an invented common win-rate target.

## 130.3 Mandatory Calibration

Every probability-bearing signal must map to an explicit prediction target.

Examples:

```text
STANDARD_SCALPING:
  P(TP1 before SL within configured scalp horizon)
  P(TP2 before SL within configured scalp horizon)

ULTRA_SCALPING:
  P(TP1 before SL within configured ultra horizon)
  with bid/ask, latency and cost-sensitive labeling

STANDARD_SWING:
  P(TP1 before SL)
  P(TP2 before SL)
  P(TP3 before SL)
  expected return within configured horizon

TREND_SWING:
  P(target before invalidation within configured multi-day horizon)
  expected net return
  time-to-target
  carry/swap-aware outcome
```

A model calibrated for one target, horizon, broker cost model or exit profile must never be presented as calibrated for another.

Approved calibration methods may include:

```text
Platt/logistic calibration
isotonic calibration
other statistically justified calibration methods
```

Calibration quality must be evaluated with:

```text
Brier score
Expected Calibration Error (ECE)
reliability diagrams
probability-bin sample counts
baseline comparison
confidence intervals where meaningful
```

No universal Brier/ECE threshold is treated as production truth. Thresholds are strategy/target-specific validation policy, compared against suitable baselines and historical evidence.

## 130.4 Sample Sufficiency and Confidence Intervals

Observed hit/win rates must include sample size and uncertainty.

For binomial outcome rates, use Wilson confidence intervals or an approved statistically equivalent method.

Seed statistical examples from the gap analysis may be used for sanity checks:

```text
98 wins / 100 completed labels
  observed = 98%
  Wilson interval is materially wider than a ±1% marketing claim

Approximate sample size for a 98% proportion at ±1% margin
  around 753 completed labels under the stated normal-approximation assumptions
```

These examples are not promotion thresholds by themselves.

Required sample sufficiency shall be defined by the versioned grade/promotion policy and shall consider:

```text
strategy
prediction target
regime
session
volatility
broker/execution profile
exit profile
label independence/autocorrelation
multiple-testing burden
OOS status
```

## 130.5 Grade Governance

No subscriber-facing `A+`, `A`, `B`, or other quality grade may be rendered unless all are true:

```text
approved grade_policy row exists
minimum completed-label requirement is met
calibration report exists
applicable confidence-interval rule passes
minimum net expectancy rule passes
drawdown/risk condition passes
strategy_definition is eligible for that environment
```

Before sufficiency:

```text
RESEARCH
UNRATED
SHADOW
```

must be used instead.

## 130.6 Performance Claim Evidence Package

Every public or subscriber-facing performance number must link to an immutable approved evidence package.

Add:

```text
performance_claim_id
metric_name
metric_definition
strategy_id
strategy_definition_id
prediction_target_id
exit_profile_id
dataset_id
period_start
period_end
sample_size
cost_model_version
slippage_assumption/version
broker_profile scope
OOS_status
confidence_interval_method
point_estimate
lower_bound
upper_bound
report_id
report_version
approved_by
approved_at
expires_at
public_visibility
```

Every UI KPI capable of being interpreted as trading performance must expose:

```text
performance_claim_source = report_id / performance_claim_id
verification_state = VERIFIED | UNVERIFIED | EXPIRED | REVOKED
```

Rules:

1. `UNVERIFIED`, `EXPIRED` or `REVOKED` performance claims are hidden from public marketing surfaces.
2. Subscriber dashboards may show unverified research metrics only when clearly labeled as research and never as a guarantee.
3. Editing a strategy, cost model, exit profile, broker profile, dataset or label definition invalidates incompatible claim packages.
4. Claim approval is a human-controlled action with audit trail.

## 130.7 Operational Signal-Delivery SLO

A legitimate operational target may be expressed separately from predictive performance.

Define:

```text
expected_delivery =
  a published signal for a recipient that is entitled,
  has an active valid delivery session/lease,
  and is within the signal delivery contract

successful_delivery =
  expected_delivery that is received and acknowledged
  within the strategy signal TTL / configured delivery deadline

signal_delivery_success_rate =
  successful_delivery / expected_delivery
```

Initial monthly SLO seed:

```text
signal_delivery_success_rate >= 98%
```

This is an operational SLO, not a prediction win-rate promise.

Track client-offline and invalid-license cases separately rather than silently deleting them from all reliability reporting:

```text
eligible_online_delivery_success_rate
offline_recipient_count
license_rejected_count
entitlement_rejected_count
expired_before_delivery_count
replay_recovery_count
duplicate_suppressed_count
late_delivery_count
```

Delivery must remain:

```text
entitlement-aware
signed
TTL-bound
deduplicated
replay-safe
reconnect-safe
acknowledged
auditable
```

## 130.8 Required Accuracy/Calibration UI

User dashboard:

```text
calibrated probability for the exact target
sample sufficiency state
grade state
TP1/TP2/TP3 historical hit rates where approved
net expectancy where approved
performance evidence/report link
research/unrated labels
no generic “98% accuracy” banner
```

Admin dashboard:

```text
calibration reliability charts
Brier/ECE
Wilson intervals
sample counts
promotion readiness
claim approval/revocation
report expiration
delivery SLO
late/failed delivery reasons
```

---

# 131. Non-Blocking, Fail-Closed Gate Architecture

“Non-blocking” means the real-time decision loop never waits on network, database, broker or model I/O. It does not mean safety checks may be skipped.

## 131.1 Pure Cached-State Gate Contract

A real-time gate is a deterministic pure evaluation over already-available state:

```text
gate(input_snapshot, gate_state_snapshot) -> PASS | VETO | DEGRADED | UNKNOWN
```

Mandatory gates shall never perform synchronous:

```text
PostgreSQL queries
remote HTTP calls
AI calls
broker RPC
payment/licensing API calls
filesystem blocking work
DNS
external calendar fetches
```

inside the tick/candidate decision path.

If a mandatory gate's state is missing, stale, invalid, incompatible or exceeds its evaluation budget:

```text
NO-TRADE
```

with a machine-readable reason.

## 131.2 Two-Layer Gate State

Layer 1 — in-process immutable/lock-free snapshot:

- maintained by dedicated goroutines;
- refreshed on the applicable tick batch, bar close, event change or lease refresh;
- read directly by the decision loop;
- generation/version stamped.

Layer 2 — Valkey hot state:

- used for cross-process/shared hot state and recovery;
- keyed by symbol/timeframe/strategy/account scope as appropriate;
- never required as a synchronous lookup for each decision when the local mirror is healthy.

PostgreSQL is persistence/audit/configuration truth, not per-tick gate lookup storage.

Canonical shape:

```text
gate_state = {
  gate_id,
  scope,
  value,
  state,
  evaluated_at,
  event_time,
  freshness_ms,
  valid_until,
  source_version,
  config_version,
  quality,
  generation
}
```

## 131.3 Gate Registry and Seed Latency Budgets

All budgets are versioned SLO seeds and must be benchmarked on the production target.

| Gate | Initial class/budget | Refresh/evaluation trigger | Mandatory failure behavior |
|---|---:|---|---|
| Data quality / feed state | fast, <1 ms | tick batch/feed event | fail closed |
| Session/calendar/holiday | fast, <1 ms | schedule transition | fail closed |
| News blackout | fast, <1 ms | event/calendar update | fail closed |
| Spread / spread-to-ATR | fast, <1 ms | tick batch | fail closed |
| Slippage / execution-cost envelope | fast, <1 ms | tick batch | fail closed |
| TTL / duplicate / cooldown | fast, <1 ms | candidate | fail closed |
| R:R / net-expectancy | fast, <1 ms | candidate | fail closed |
| Regime / structure / liquidity / confluence | mid, <5 ms | bar close/feature update | fail closed |
| Aggregate XAU exposure / margin | mid, <5 ms | account/candidate update | fail closed |
| Entitlement / license / execution permission | mid, <5 ms local evaluation | lease/account state update | fail closed |
| Macro / DXY / yields | background | source update | stale policy → veto/degrade |
| AI verification | background | candidate policy | strategy policy: quant-only or reject |

A production benchmark may tighten or relax a seed only through versioned policy and evidence.

## 131.4 Short-Circuit Veto Ordering

Default cheapest/highest-safety ordering:

```text
data_quality
→ session
→ news
→ spread
→ slippage
→ total_cost
→ exposure
→ margin
→ R:R / net_expectancy
→ entitlement
→ license
→ execution_permission
```

The first hard veto terminates gate evaluation for that candidate and records all state required to reproduce the decision.

Expensive evidence scoring runs only when prerequisite hard gates are healthy.

## 131.5 Gate Watchdog

A dedicated watchdog shall measure:

```text
gate_eval_latency_ms p50/p95/p99/max
gate_state_age_ms
gate_budget_exceeded_count
gate_unknown_count
gate_degraded_count
gate_veto_count
gate_pass_count
no_trade_count_by_gate
```

If a mandatory gate exceeds policy:

```text
GATE_DEGRADED
```

is raised and that gate fails closed without pausing other services.

## 131.6 Async Execution Boundary

The real-time candidate/tick loop must never wait for broker execution.

Use:

```text
execution_intent
transactional/outbox event
idempotency key
async Windows/MT submission
broker acknowledgement stream
reconciliation loop
```

Timeout state:

```text
EXECUTION_TIMEOUT
```

does not imply “not filled.” Reconciliation determines broker truth.

No retry may duplicate an already-submitted order.

## 131.7 Gate Freshness and Version Compatibility

Each strategy definition declares:

```text
required_gate_ids[]
required_gate_freshness[]
allowed_degraded_modes[]
gate_policy_version
```

A gate snapshot generated from an incompatible strategy/config/feature version is invalid even if recent in wall-clock time.

---

# 132. Extended Versioned Indicator / Feature Registry

This section extends Sections 12 and 12B. All indicators are evidence contributors, never solo trade authorization.

Each registry row shall include:

```text
indicator_id
implementation_version
formula_version
parameter_schema
default_seed_parameters
required_capabilities[]
timeframe applicability
strategy applicability
warmup requirement
missing-data policy
quality-state rules
normalization policy
online implementation
research implementation
parity_tolerance
approved_status
```

## 132.1 Required Additional Indicator Families

| Indicator | Canonical formula/definition sketch | Initial seed | Primary role |
|---|---|---|---|
| SuperTrend | `HL2 ± multiplier × ATR`; trend flips by approved band-cross rules | ATR 10, multiplier 3 | trend state / trail research |
| Keltner Channels | `EMA(n) ± multiplier × ATR` | 20, 2 | envelope/squeeze |
| Donchian Channels | rolling highest high / lowest low | 20 scalp; 55 swing seed | breakout/structure |
| Parabolic SAR | standard SAR recurrence with acceleration factor | step .02, max .2 | trend/trailing research |
| OBV | cumulative signed volume by close direction | versioned | volume-flow proxy |
| MFI | typical-price money-flow ratio transformed to 0–100 | 14 | volume-weighted momentum |
| Williams %R | `-100 × (HH-C)/(HH-LL)` | 14 | momentum extremes |
| Force Index | EMA of `(C-Cprev) × volume` | 13 | momentum/flow proxy |
| Chandelier Exit | long: `HH(n) - mult×ATR`; short mirror | 22, 3 | trailing-stop candidate |
| Heiken Ashi | canonical smoothed OHLC transform | standard | display/trend research |
| Classic Pivot Points | `P=(H+L+C)/3`, standard R/S levels | D/W/M | reference/target levels |
| Fibonacci Retracement/Extension | 0.382/.5/.618/.786; 1.272/1.618/2.618 seed ratios | anchored to deterministic swing | target/confluence research |
| Rolling Z-score | `(x-rolling_mean)/rolling_std` | 20–100 seed | normalization |
| Session-relative stats | session range position/percentiles | session-calibrated | session context |
| Ichimoku | canonical five-line calculation | 9/26/52 | swing/trend research |
| Hurst exponent | approved estimator over long window | >=256 observations seed | persistence research |

## 132.2 Volume Semantics

For OTC spot XAUUSD:

- broker tick volume remains `BROKER_TICK_VOLUME_PROXY`;
- OBV/MFI/Force Index derived from broker tick volume must carry `ESTIMATED`/proxy semantics;
- such features must not be labeled centralized market flow.

When authoritative exchange/futures volume is used, the source/capability/contract/roll state must be stored.

## 132.3 Classic Pivot Point Reference

For prior approved period `H/L/C`:

```text
P  = (H + L + C) / 3
R1 = 2P - L
S1 = 2P - H
R2 = P + (H - L)
S2 = P - (H - L)
R3 = H + 2(P - L)
S3 = L - 2(H - P)
```

Session/period boundaries must be timezone/calendar-defined, never inferred from local server midnight.

## 132.4 Fibonacci Anchor Governance

Fibonacci levels are invalid unless the anchor is deterministic and reproducible.

Store:

```text
anchor_method
swing_low_id
swing_high_id
anchor_timeframe
direction
ratios
feature_version
```

No human-drawn discretionary Fibonacci level may become silent production state.

---

# 133. Numeric Candle-Pattern Registry

Section 12F.1 pattern names shall be backed by deterministic numeric definitions.

For candle:

```text
body  = abs(C - O)
range = H - L
upper_wick = H - max(O, C)
lower_wick = min(O, C) - L
```

Requirements:

- reject/mark invalid when `range <= 0`;
- normalize minimum significance with ATR/tick-size/session-volatility policy;
- use versioned thresholds;
- require context where stated;
- pattern output is evidence, not trade authorization.

## 133.1 Initial Pattern Seeds

| Pattern | Initial numeric definition | Context |
|---|---|---|
| BULLISH_ENGULFING | previous bearish; current bullish; `C >= O_prev`; `O <= C_prev`; `body_cur >= 1.1×body_prev`; `body_cur >= 0.5×ATR` | liquidity/structure context |
| BEARISH_ENGULFING | mirrored bullish definition | liquidity/structure context |
| HAMMER | `lower_wick >= 2×body`; `upper_wick <= 0.3×body`; `body <= 0.4×range` | after down-leg / swing low |
| SHOOTING_STAR | `upper_wick >= 2×body`; `lower_wick <= 0.3×body`; `body <= 0.4×range` | after up-leg / swing high |
| DOJI | `body <= 0.1×range` | context-dependent |
| INSIDE_BAR | `H <= H_prev` and `L >= L_prev` | context-dependent |
| OUTSIDE_BAR | `H > H_prev` and `L < L_prev` | context-dependent |
| PIN_BAR_CANDIDATE | dominant wick `>=2×body`, opposite wick `<=0.3×body` | pool/level |
| MARUBOZU | `body >=0.9×range` | context-dependent |
| MORNING_STAR | bearish → small-body → bullish; final close above midpoint of first candle body | support/structure |
| EVENING_STAR | mirror of morning star | resistance/structure |
| THREE_WHITE_SOLDIERS | three bullish candles, rising closes, opens within approved prior-body band | context-dependent |
| HARAMI | current candle body/range containment policy inside prior body | context-dependent |
| TWEEZER_TOP/BOTTOM | two highs/lows equal within versioned `equal_level_tolerance` | level/liquidity context |

## 133.2 Pattern Output Contract

```text
pattern_event_id
pattern_id
pattern_version
symbol
timeframe
bar_ids[]
detected_at
numeric_evidence
atr_normalization
context_flags
quality
source_candle_versions
```

Golden fixtures are mandatory for every pattern and mirrored long/short case.

---

# 134. Canonical Quantitative Math and Statistical Reference Library

Codex shall implement a dependency-light reference library and production-tested equivalents.

Required reference deliverables:

```text
research/reference_math.py
Go production math package(s)
unit tests
golden cross-language fixtures
property/fuzz tests where appropriate
precision policy
formula/version documentation
```

The addendum referenced a companion `reference_math.py`; because it is not part of the supplied SOW files, its behavior is specified here as a required deliverable.

## 134.1 Gross and Net R:R

For price distances:

```text
gross_RR_TPi =
  abs(TPi - Entry) / abs(Entry - SL)
```

A cost-adjusted seed representation:

```text
net_RR_TPi =
  (target_distance_price - expected_round_trip_cost_price)
  / (stop_distance_price + expected_round_trip_cost_price)
```

Production implementation must use broker-normalized tick/point economics from Sections 25B and 103A rather than assuming cost can always be represented as a simple raw price delta.

## 134.2 Expectancy and Profit Factor

```text
E_R = P(win) × AvgWinR - P(loss) × AvgLossR

E_currency =
  expectation calculated from the actual exit ladder,
  position size, costs, partial fills, slippage and currency conversion

ProfitFactor =
  gross realized/expected winning amount
  /
  abs(gross realized/expected losing amount)
```

Seed sanity example:

```text
p_win = .55
AvgWinR = 1.5
AvgLossR = 1.0
E_R = .375R
```

No strategy activates merely because the point estimate is positive; promotion policy and uncertainty still apply.

## 134.3 Sharpe and Sortino

For a clearly defined per-trade/per-period return series:

```text
Sharpe = (mean(return) - reference_rate_if_applicable) / std(return)
Sortino = (mean(return) - target_return) / downside_deviation
```

Annualization, if used, must state the sampling assumption. Do not apply an arbitrary annualization factor to irregular/autocorrelated trade returns without documentation.

## 134.4 Kelly — Research Only

Seed:

```text
b = AvgWinR / AvgLossR
q = 1 - p
f_star = p - q/b
```

Kelly is research diagnostics only.

Live position sizing remains the approved fixed-fraction/risk model of Section 25B.2.

If Kelly is studied, use uncertainty-aware inputs such as a conservative probability bound and impose strict caps.

## 134.5 Risk of Ruin

The simple heuristic:

```text
((1-p)/p)^N
```

may be used only as a clearly labeled research sanity check under its narrow equal-stake/equal-payoff/i.i.d.-style assumptions.

It is **not** a universal production formula.

In particular, `p <= 0.5` does not by itself imply certain ruin when payoff asymmetry/positive expectancy exists.

Production risk-of-ruin and drawdown evaluation must use the empirical cost-adjusted R distribution, dependency-aware stress scenarios and Monte Carlo/bootstrap analysis.

## 134.6 Calibration Metrics

```text
Brier = mean((p - outcome)^2)

ECE =
  Σ_bin (n_bin / N) × abs(observed_frequency_bin - mean_forecast_bin)
```

Store calibration bin edges, bin counts and method version.

## 134.7 Wilson Interval

Implement a tested Wilson score interval for binary hit-rate reporting.

The exact implementation shall have fixtures for:

```text
0/n
n/n
small n
98/100
196/200
large n
```

## 134.8 Approximate Sample-Size Planning

An approved power-analysis implementation shall exist.

The addendum seed formula may be retained for planning:

```text
n ≈ (z_alpha + z_beta)^2
    × [p1(1-p1) + p0(1-p0)]
    / (p1 - p0)^2
```

but final validation may require a more appropriate test/design when labels are dependent, stratified, sequential or multiply tested.

Seed sanity targets from the addendum:

```text
55% vs 50%: order of ~1.5k completed labels
60% vs 50%: order of ~385 completed labels
98% proportion ±1%: order of ~753 labels under simple assumptions
```

These numbers are planning examples, not strategy acceptance rules.

## 134.9 Monte Carlo / Bootstrap Drawdown

For each strategy and combined portfolio:

- resample the empirical net-R outcome sequence;
- use block/bootstrap methods when autocorrelation exists;
- initial simulation count seed: at least 10,000 paths;
- report median, p90, p95, p99 and worst observed simulated drawdown;
- preserve random seed/version for reproducibility;
- stress costs/slippage and losing streak clustering.

Risk limits must be based on an approved policy, not the single most optimistic path.

## 134.10 MTF Alignment

Seed deterministic score:

```text
mtf_alignment_score =
  100 × Σ(weight_tf × state_tf) / Σ(weight_tf)

state_tf ∈ {-1, 0, +1}
```

Only valid/fresh timeframes participate.

A missing/stale **required** timeframe invalidates the strategy evaluation rather than silently renormalizing it away.

## 134.11 Cost-to-Target

```text
cost_to_target =
  total_expected_round_trip_cost_price
  /
  abs(TP1 - Entry)
```

Initial research maximum seed may be `0.25` where already specified, but strategy/broker/session-specific validated policy governs.

---

# 135. Versioned Exit Profile — SL, TP1/TP2/TP3, Partial Close, Breakeven and Trailing

Section 27 shall be implemented with a first-class immutable/versioned `exit_profile`.

## 135.1 Exit Profile Contract

```text
exit_profile_id
strategy_id
version
entry_reference_policy
stop_model
stop_atr_multiplier
structure_stop_policy
stop_buffer_policy
minimum_stop_distance_policy
maximum_stop_distance_policy
tp1_selection_policy
tp2_selection_policy
tp3_selection_policy
tp1_fraction
tp2_fraction
tp3_fraction
breakeven_trigger
breakeven_buffer_policy
trailing_trigger
trailing_model
time_stop_policy
news_open-position_policy
partial_fill_policy
rounding_policy
broker_constraint_policy
status
approved_by
approved_at
code_commit
```

Fractions must sum to the intended close quantity within exact tolerance.

## 135.2 Stop-Loss Construction

Seed conceptual rule:

```text
raw_atr_distance =
  k_strategy × ATR(entry_timeframe)

minimum_broker_distance =
  max(
    broker stop-level constraint,
    applicable freeze/modify constraint,
    safety tick buffer
  )

candidate_distance =
  clamp(
    raw_atr_distance,
    configured minimum,
    configured maximum
  )
```

If a deterministic structure stop exists, combine it according to the versioned policy; do not always choose the farther or nearer stop without strategy definition.

For a long, a policy may choose a stop below:

```text
structure_low - buffer
```

For a short:

```text
structure_high + buffer
```

The final stop must pass:

```text
risk budget
lot-size calculation
broker minimum distance
broker freeze/modify semantics
tick-size rounding
margin/stop-out safety
maximum strategy stop distance
cost/expectancy gate
```

Invalid stop geometry → `EXECUTION_DENIED` / `NO-TRADE`, never silent widening.

## 135.3 Target-Selection Sources

Initial strategy mapping:

| Strategy | TP1 | TP2 | TP3 |
|---|---|---|---|
| STANDARD_SCALPING | nearest valid BSL/SSL, equal H/L, session H/L or approved intraday liquidity | next validated liquidity/prior level/VWAP band | optional profile POC/VAH/VAL or versioned objective |
| ULTRA_SCALPING | nearest micro-liquidity/imbalance target supported by data capability | optional next micro target | normally disabled unless separately validated |
| STANDARD_SWING | PDH/PDL, prior-week H/L or approved structural level | VAH/VAL/order block | structure objective/Fibonacci extension if validated |
| TREND_SWING | major HTF liquidity | HTF profile/order block | weekly/monthly/versioned trend objective |

Every chosen TP stores its source/evidence.

## 135.4 Initial Exit-Ladder Seeds

These are validation seeds, not immutable live rules.

| Strategy | TP1 action | TP2 action | TP3 action | Breakeven seed | Trailing seed |
|---|---|---|---|---|---|
| STANDARD_SCALPING | close 50% | close remaining 50% | disabled by default | after confirmed TP1 fill | optional tight versioned trail |
| ULTRA_SCALPING | close 100%, or separately validated 60/40 variant | optional | disabled | normally none unless validated | tight chandelier/ATR variant research |
| STANDARD_SWING | close 40% | close 30% | close 30% | after confirmed TP1 fill | ATR trail after TP2 |
| TREND_SWING | close 30% | close 30% | close 40% | optional after TP1 | chandelier/ATR after TP2 |

A strategy variant using a different ladder is a different versioned exit profile and must be backtested/calibrated as such.

## 135.5 Breakeven

Move-to-breakeven must trigger from a confirmed broker fill/event, not merely from chart price touching TP1.

Long seed:

```text
new_SL = entry + cost_neutral_buffer
```

Short mirror:

```text
new_SL = entry - cost_neutral_buffer
```

Buffer policy may use:

```text
expected round-trip cost
0.1 × ATR seed
broker tick buffer
```

The chosen policy must be validated and broker-rounded.

“Breakeven” presented to the user must be cost-aware; entry price alone may still realize a net loss after costs.

## 135.6 Trailing

Supported versioned models may include:

```text
ATR trail
Chandelier trail
structure swing trail
VWAP/level trail
fixed tick trail only if broker-normalized and validated
```

Trailing updates must respect:

```text
freeze level
stop level
minimum modification interval
broker rate limits
price rounding
idempotency
partial fills
terminal reconnect
```

## 135.7 Target Distance Constraints

A TP must satisfy broker and economic viability.

Conceptually:

```text
target_ticks >= broker_minimum
target_value > validated expected execution cost
cost_to_target <= approved maximum
target_distance <= strategy horizon/volatility envelope where policy requires
```

No hard-coded universal “gold pip” distance.

## 135.8 Partial Close and Fill Semantics

Partial-close instructions require:

```text
close_intent_id
position_id
target_id
requested_fraction
requested_volume
rounded_volume
broker_min_lot
broker_lot_step
submitted_at
acknowledged_at
filled_volume
remaining_volume
idempotency_key
reconciliation_state
```

The system must handle:

```text
partial target fills
partial close rejection
minimum remaining lot
broker volume rounding
disconnect between TP fills
duplicate terminal events
manual user close
broker-side stop/TP changes
```

## 135.9 Calibration and Backtest Consequence

The exit ladder changes the outcome distribution.

Therefore backtests and calibration shall run with the **same**:

```text
exit_profile_id
partial-close fractions
breakeven logic
trailing logic
time stop
cost model
slippage model
fill model
```

used by the evaluated production candidate.

Do not calibrate only `P(TP1 before SL)` and then claim that probability describes the P&L of a multi-target partial-close ladder.

---

# 136. R:R, “R” Unit and Net-Expectancy Governance

## 136.1 Define R with Broker-Normalized Economics

Do not define monetary risk as raw price distance × tick value without tick-size normalization.

Canonical pre-cost stop risk:

```text
stop_ticks =
  abs(Entry - SL) / tick_size

gross_R_currency =
  stop_ticks
  × tick_value_per_lot_in_account_currency
  × lots
```

Where tick value is direction/account-currency dependent, use the broker-profile conversion model.

Also store expected cost separately:

```text
expected_round_trip_cost_currency
net_risk_currency
```

## 136.2 Gross vs Net R:R

Display and decision logic must distinguish:

```text
gross_RR_TP1
gross_RR_TP2
gross_RR_TP3

net_RR_TP1
net_RR_TP2
net_RR_TP3
```

Gross R:R may be used for strategy research constraints.

Final execution authorization uses cost-aware net economics and approved expectancy policy.

A setup that passes gross R:R but fails net R:R/cost-to-target shall be rejected.

## 136.3 Primary Display R:R

For subscriber UI:

- show each TP's R:R where that TP exists;
- identify the primary TP explicitly;
- for scalps, TP1 may be primary;
- show the exit-ladder percentages;
- show “gross” and “estimated net” labels distinctly;
- never hide costs inside an unexplained R:R number.

## 136.4 Expectancy Is the Binding Economic Constraint

Do not optimize R:R upward in isolation.

The strategy promotion objective must consider:

```text
calibrated hit probabilities
partial-close fractions
average win/loss
costs
slippage
missed fills
drawdown
tail losses
regime/session stability
```

A higher target that reduces hit probability may have worse net expectancy.

---

# 137. Cross-Language Determinism, Online/Offline Feature Parity and Replay Fidelity

## 137.1 Golden Feature Parity Harness

Go is the authoritative live path and Python is the research path; they must not silently implement different mathematics.

Create shared canonical fixtures covering:

```text
raw ticks
bid/ask
candles
session/calendar state
broker specification
feature inputs
macro/news snapshots
strategy configuration
expected features
expected gate outcomes
expected signal candidate
expected exit geometry
```

CI must execute the same fixtures through Go and Python.

Comparison classes:

```text
BIT_EXACT
DECIMAL_EXACT
TOLERANCE_BOUNDED
ORDERING_EQUIVALENT
STATE_MACHINE_EQUIVALENT
```

Each feature declares its allowed comparison class/tolerance.

## 137.2 Deterministic Decision Reproduction

The same canonical snapshot plus the same:

```text
strategy_definition_id
feature versions
gate policy
risk profile
execution profile
exit profile
broker profile
calibration profile
```

must reproduce the same decision and reason codes.

Any unavoidable nondeterminism must be isolated, seeded and recorded.

## 137.3 Online/Offline Feature Parity

Research must consume the same feature definitions and normalization semantics as live production.

Do not train/calibrate using a feature that live Go computes differently.

Every stored signal snapshot shall be sufficient to reproduce:

```text
raw/normalized inputs
feature values
quality states
gate states
score contributions
calibrated probability lookup
risk decision
exit targets
final decision
```

## 137.4 Replay Parity

Replay tests must compare:

```text
historical event order
feature state
gate state
candidate state
signal decision
NO-TRADE reason
SL/TP geometry
execution-intent generation
```

against recorded production/shadow fixtures.

## 137.5 Drift on Parity

Feature-parity regressions are release blockers.

Create:

```text
feature_parity_run_id
fixture_set_version
go_commit
python_commit
config_version
mismatch_count
max_numeric_error
status
artifact/report
```

---

# 138. UAE Regulatory, Marketing and Commercial Activation Gate

This section is an engineering/compliance release gate, not a legal conclusion.

Before offering paid trading signals, marketing investment/trading recommendations, enabling assisted/automatic execution, or enabling referral payouts, obtain an up-to-date written regulatory-perimeter assessment for the actual:

```text
legal entity
place of establishment
customer jurisdictions
product/instrument
signal/advice presentation
degree of personalization
execution functionality
custody/payment flow
marketing channels
referral/affiliate structure
```

## 138.1 Jurisdictional Perimeter

The assessment must determine which authority/regime applies, including as relevant:

```text
UAE federal/mainland securities regulation
DIFC / DFSA
ADGM / FSRA
customer-country restrictions
```

Do not infer regulatory status solely from a Dubai/UAE company/free-zone registration.

## 138.2 Financial Recommendation / Promotion Controls

Implement a legal/compliance approval state for:

```text
public strategy descriptions
performance claims
social/media promotions
affiliate creatives
email campaigns
dashboard claims
automated-execution marketing
```

No unapproved marketing claim is publishable.

## 138.3 “98% Accuracy” and Guarantee Ban

Unless an applicable regulator/legal approval and evidence package explicitly permits a narrowly defined statement:

```text
guaranteed returns
guaranteed profit
98% prediction accuracy
risk-free
no-loss
certain win
```

shall be blocked from production marketing content.

The platform may publish approved operational SLOs such as delivery reliability only when clearly labeled as operational, not predictive.

## 138.4 Referral / Five-Level Commission Activation

The five-level referral design remains governed by Section 69, but payout activation additionally requires:

```text
legal review
payment-provider/bank acceptance
KYC/KYB requirements where applicable
sanctions screening policy
tax/reporting assessment
anti-abuse controls
terms acceptance
affiliate marketing policy
prohibited-jurisdiction rules
```

Referral revenue remains based on eligible validated subscription revenue, never recruitment alone.

## 138.5 Compliance Artifacts

Store/version:

```text
legal_review_id
jurisdiction
scope
counsel/provider
issued_at
expires/review_at
conditions
prohibited_features
approved_features
document checksum/reference
approval status
```

External legal documents need not be stored in the trading database if policy forbids it; an immutable reference/checksum and approval metadata are sufficient.

## 138.6 Timezone

UAE/local presentation may use Asia/Dubai, but session/news/fix engines remain UTC internally and timezone/DST-aware for the market venue being modeled.

Never hard-code a fixed GMT+4 “golden window” as market truth.

---

# 139. Gate, Calibration, Exit and Delivery Observability

Extend Sections 86, 87, 87A, 88 and 89.

## 139.1 Gate Metrics

Required dimensions:

```text
gate_id
strategy_id
symbol
timeframe
broker_profile
environment
result
reason
```

Metrics:

```text
gate_eval_latency_ms
gate_state_age_ms
gate_budget_exceeded_count
gate_pass_count
gate_veto_count
gate_unknown_count
gate_degraded_count
no_trade_reason_count
```

Avoid unbounded high-cardinality labels such as raw user ID in Prometheus.

## 139.2 Calibration Metrics

```text
calibration_brier
calibration_ece
calibration_bin_count
calibration_sample_count
probability_drift
observed_hit_rate
wilson_lower
wilson_upper
grade_policy_state
promotion_readiness
```

Detailed per-user/per-signal data belongs in logs/analytics storage, not high-cardinality metric labels.

## 139.3 Exit Metrics

```text
tp1_hit_rate
tp2_hit_rate
tp3_hit_rate
sl_hit_rate
breakeven_move_count
breakeven_stop_count
trailing_activation_count
partial_close_reject_count
partial_close_reconcile_error_count
mfe
mae
time_to_target
realized_R
net_R
```

All sliced in analytical reports by approved dimensions.

## 139.4 Delivery Metrics

```text
signal_publish_count
expected_delivery_count
delivery_ack_count
delivery_within_ttl_count
late_delivery_count
delivery_failure_count
offline_recipient_count
replay_recovery_count
duplicate_suppressed_count
websocket_reconnect_count
```

## 139.5 Dashboards and Alerts

Create Grafana/admin views for:

```text
gate health
NO-TRADE gate breakdown
calibration health
strategy sample sufficiency
exit-ladder outcomes
delivery SLO
broker execution health
feature-parity status
claim verification state
```

Alert on sustained:

```text
mandatory gate degradation
gate budget exceedance
feature parity failure
delivery SLO burn
calibration drift
claim evidence expiration
reconciliation backlog
```

---

# 140. Integrated Data Model, API and UI Additions

## 140.1 Database Additions

Implement through canonical migrations; names may adapt to existing schema after repository audit.

Required logical entities:

```text
gate_definitions
gate_policy_versions
gate_evaluation_events
indicator_definitions / feature_definitions
candle_pattern_definitions
exit_profiles
performance_claims
performance_claim_evidence
calibration_reports
calibration_bins or report artifacts
feature_parity_runs
feature_fixture_sets
signal_delivery_receipts
delivery_slo_rollups
exit_action_events
partial_close_events
compliance_approvals / legal_review_references
```

Do not persist every ephemeral per-tick gate snapshot to PostgreSQL if volume is unjustified. Persist decision-relevant snapshots/events sufficient for audit and replay; keep hot current state in memory/Valkey.

## 140.2 Signal Snapshot Extension

Section 64 signal snapshots shall additionally capture:

```text
gate_policy_version
gate_results[]
indicator/feature versions
candle_pattern flags + versions
exit_profile_id
SL source/evidence
TP1 source/evidence
TP2 source/evidence
TP3 source/evidence
gross_RR values
net_RR estimates
expected_cost
calibration_report/version
performance evidence state if shown
```

## 140.3 Control APIs

Add or extend authenticated APIs for:

```text
GET  /strategies/:id/calibration
GET  /strategies/:id/performance-evidence
GET  /signals/:id/decision-snapshot
GET  /signals/:id/exit-profile
GET  /operations/gates
GET  /operations/gates/:gateId
GET  /operations/delivery-slo
GET  /admin/calibration/reports
POST /admin/performance-claims/:id/approve
POST /admin/performance-claims/:id/revoke
GET  /admin/feature-parity/runs
GET  /admin/exit-profiles
POST /admin/exit-profiles
POST /admin/exit-profiles/:id/approve
GET  /admin/compliance/approvals
```

Actual route conventions shall follow the repository's canonical API versioning.

All consequential writes require:

```text
RBAC
MFA/step-up where policy requires
audit
schema validation
idempotency where appropriate
optimistic/version conflict protection
```

## 140.4 User UI Additions

Live Signal Terminal shall show when applicable:

```text
strategy
grade state
calibrated probability + exact target
Entry
SL
TP1/TP2/TP3
exit percentages
gross R:R by target
estimated net R:R by target
cost estimate
signal TTL
data quality
NO-TRADE reason
evidence summary
performance report reference
```

Do not show a target that the active exit profile does not use.

## 140.5 Admin UI Additions

Add:

```text
Gate Registry / Health
Gate Latency / Budget
NO-TRADE Gate Analytics
Indicator/Pattern Version Registry
Exit Profile Registry
Calibration Center
Sample Sufficiency
Performance Claim Approval
Feature-Parity CI Results
Delivery SLO
Compliance Approval State
```

No admin UI setting may directly mutate live production strategy behavior without the existing configuration safety, approval, audit and rollback controls.

---

# 141. Testing, Validation and Release Gates for Sections 130–140

## 141.1 Determinism Tests

Required:

```text
same input → same output
Go/Python golden parity
long/short mirror fixtures
timezone/DST fixtures
broker digit/tick-size variants
missing-data fixtures
stale-state fixtures
config-version mismatch fixtures
```

## 141.2 Gate Tests

Test every gate for:

```text
PASS
VETO
UNKNOWN
STALE
DEGRADED
budget exceeded
wrong generation/version
missing dependency
hot-state recovery
```

Prove no mandatory gate executes blocking I/O in the decision path.

Use profiling/tracing/static review as evidence.

## 141.3 Indicator and Candle Tests

For each new indicator/pattern:

```text
known-vector unit test
boundary test
warmup test
NaN/zero-range test
missing-bar test
gap test
Go/Python parity test
version-migration fixture
```

## 141.4 Math Tests

Required:

```text
R:R
broker-normalized R
expectancy
profit factor
Sharpe
Sortino
Kelly research helper
Wilson interval
Brier
ECE
sample-size planner
Monte Carlo reproducibility
MTF score
cost-to-target
```

Use independent expected vectors for critical formulas.

## 141.5 Exit-Ladder Backtests

Backtest engine must simulate:

```text
bid/ask
spread
commission
slippage
latency model
order type
partial fills
TP fractions
BE trigger after confirmed modeled fill
trailing model
stop/freeze constraints
time stops
news policy
swap/carry where relevant
```

Backtests that ignore the active exit ladder are invalid for production calibration/promotion.

## 141.6 Delivery Reliability Tests

Test:

```text
recipient entitlement changes
lease expiration
disconnect/reconnect
replay
duplicate publication
duplicate ACK
late ACK
signal expiration
client offline
server restart
Valkey loss/recovery
gateway failover
Windows agent restart
EA restart
```

Verify SLO numerator/denominator correctness.

## 141.7 Performance Claim Tests

Test:

```text
missing evidence → hidden/unverified
expired evidence → hidden/revoked per policy
strategy version changes → incompatible claim invalidated
cost-model change → revalidation required
admin approval audit
public API cannot publish unapproved claim
```

## 141.8 Compliance Release Gate

Before paid/public activation in a jurisdiction, CI/CD/release evidence shall require the configured compliance approval state.

The build may remain technically deployable while restricted features are disabled.

External approval absence is an activation blocker, not permission to fabricate approval.

---

# 142. Integrated Implementation Phases, Deliverables, Skills and Subagent Requirements

These amendments are mandatory additions to Sections 117–125 and 127–129.

## 142.1 Integrated Phase 11A — Freeze Exit and Gate Policies

Before production real-time strategy work, each strategy definition must also freeze:

```text
exit_profile_id
gate_policy_version
indicator/feature versions
candle-pattern versions used
net-expectancy policy
claim/grade policy
```

## 142.2 Integrated Phase 12 — Gate Runtime

Real-Time Quantitative Engine phase must implement:

```text
local immutable gate snapshot
Valkey hot-state contract
gate registry
short-circuit veto ordering
latency watchdog
fail-closed degradation
gate metrics
```

before live signal activation.

## 142.3 Integrated Phase 13 — Reference Math and Calibration

Prediction/Calibration phase must deliver:

```text
reference_math.py or canonical equivalent
Go/Python parity fixtures
Wilson intervals
Brier/ECE
calibration reports
sample sufficiency
claim evidence linkage
```

## 142.4 Integrated Phase 19 — Observability

Add the complete Section 139 metrics/dashboard/alert set.

## 142.5 Integrated Phase 21 — Exit-Aware Trading Validation

Trading validation must use the exact exit profile, cost model and fill assumptions intended for promotion.

## 142.6 Integrated Phase 22/23 Activation — Compliance

Paid signal production and live/assisted execution require the applicable Section 138 compliance approval state, in addition to all technical gates.

## 142.7 Required Deliverables

Add to Section 118:

```text
Accuracy/Metric Definition Specification
Calibration Evidence Package Specification
Performance Claim Governance Specification
Operational Delivery SLO Specification
Non-Blocking Gate Architecture Specification
Gate Registry and Latency Budget
Extended Indicator Registry
Numeric Candle-Pattern Registry
Canonical Quantitative Math Library + Tests
Exit Profile / TP Ladder Specification
R:R and Net-Expectancy Specification
Feature-Parity Harness
Delivery Receipt / SLO Reporting
Compliance Activation Checklist
Gate/Calibration/Delivery Grafana Dashboards
Gap-Addendum Coverage Matrix
```

## 142.8 Codex Skill Requirements

Do not create duplicate skills where existing Sections 122.x skills can be extended.

`xauusd-strategy-spec` must additionally cover:

```text
extended indicators
numeric candle patterns
exit profiles
TP ladders
breakeven
trailing
R:R/net expectancy
```

`xauusd-quant-validation` must additionally cover:

```text
Wilson intervals
Brier/ECE
sample sufficiency
Monte Carlo
claim evidence
exit-aware backtests
Go/Python feature parity
```

`trading-risk-safety` must additionally cover:

```text
net R:R
cost-to-target
exit geometry
partial-close safety
gate fail-closed semantics
```

`observability-sre` must additionally cover:

```text
gate metrics
delivery SLO
calibration health
feature-parity health
claim-expiration alerts
```

`release-gates` must additionally require:

```text
feature parity PASS
exit-aware backtest PASS
gate non-blocking evidence
delivery reliability evidence
compliance activation state
performance claim evidence validation
```

## 142.9 Codex Subagent Requirements

`go_realtime_engineer`:

```text
implement cached gate runtime
implement production math equivalents
implement exit-state transitions
emit deterministic snapshots
```

`python_quant_researcher`:

```text
implement reference math
calibration/statistical reports
exit-aware backtests
feature parity vectors
Monte Carlo/stress analysis
```

`quant_validator`:

```text
independently validate confidence intervals
sample sufficiency
calibration
net expectancy
exit ladder
unsupported claims
```

`qa_reliability_engineer`:

```text
cross-language golden fixtures
gate latency/fail-closed tests
delivery SLO fault tests
partial-close/reconciliation tests
```

`release_manager`:

```text
verify performance-claim evidence
verify parity report
verify gate health
verify compliance activation blockers
```

---

# 143. Final Integrated Gap-Closure Acceptance and Traceability

The canonical SOW is complete only when Sections 1–143 are satisfied as applicable.

## 143.1 Integrated Coverage Matrix

| Gap Analysis / Addendum Requirement | Final Canonical Location |
|---|---|
| Accurate signal generation | 15/15A/16/17A/106/107 + 130 + 137 |
| Deterministic same-input/same-output behavior | 98A + 137 + 141 |
| Calibrated probability | 15A/107 + 130 + 134 |
| “98% prediction accuracy” governance | 78 + 130 + 138 |
| 98% operational delivery SLO | 89 + 130.7 + 139.4 + 141.6 |
| Non-blocking gates | 131 |
| Gate latency budgets | 131.3 |
| Gate watchdog/degradation | 131.5 + 139 |
| Async execution boundary | 50/83 + 131.6 |
| Extended indicators | 132 |
| Pivot Points | 132.3 |
| Fibonacci | 132.4 |
| Numeric candle patterns | 133 |
| R:R formulas | 134.1 + 136 |
| Expectancy / PF | 134.2 + 136.4 |
| Sharpe / Sortino | 134.3 |
| Kelly research-only | 134.4 |
| Risk-of-ruin handling | 134.5 |
| Brier / ECE | 134.6 |
| Wilson CI / sample planning | 134.7–134.8 |
| Monte Carlo drawdown | 134.9 |
| MTF alignment math | 10A + 134.10 |
| Cost-to-target | 25A.2 + 134.11 |
| SL construction | 27 + 135.2 |
| TP1/TP2/TP3 selection | 27 + 135.3 |
| Partial close | 135.4 + 135.8 |
| Breakeven | 135.5 |
| Trailing | 135.6 |
| Exit ladder in calibration/backtest | 135.9 + 141.5 |
| Gross vs net R:R | 136 |
| UAE regulatory perimeter | 77/78 + 138 |
| Cross-language feature parity | 137 + 141 |
| Online/offline feature parity | 137.3 |
| Gate observability | 139 |
| Data/API/UI integration | 140 |
| Implementation/Codex amendments | 142 |

## 143.2 Additional Production-Ready Acceptance Criteria

Append to Section 126:

41. Every mandatory decision gate is precomputed/cached, deterministic, freshness/version stamped and fail-closed.
42. No per-candidate mandatory gate performs synchronous external I/O.
43. Gate p50/p95/p99 latency and budget-exceeded behavior are observable.
44. Go and Python feature/decision fixtures pass the declared parity contract.
45. Extended indicators are versioned and never solo-authorize a trade.
46. Candle-pattern definitions are numeric, context-aware, versioned and fixture-tested.
47. The reference quantitative math library and production equivalents pass independent fixtures.
48. Subscriber probability is tied to an explicit prediction target and active exit profile.
49. No grade is shown before sample/calibration policy sufficiency.
50. No public performance KPI exists without an approved evidence source.
51. “98%” is never used as an invented prediction win-rate requirement.
52. The operational delivery SLO has a precise denominator, ACK/TTL semantics and offline/reject breakdowns.
53. Every strategy has a versioned exit profile.
54. SL/TP geometry is broker-normalized and cost/risk constrained.
55. TP1/TP2/TP3 partial-close actions are idempotent and reconciled with broker truth.
56. Breakeven is triggered from confirmed modeled/real fill state and is cost-aware.
57. Trailing behavior is versioned and broker-constraint aware.
58. Backtests include the exact exit ladder, breakeven and trailing behavior being promoted.
59. Gross and net R:R are separate; final execution considers net economics.
60. Risk-of-ruin decisions do not rely on an invalid universal win-rate-only shortcut.
61. Compliance/legal/payment-provider activation blockers are represented honestly and can disable commercial/live features without fabricating approval.
62. Gate/calibration/exit/delivery/feature-parity telemetry and dashboards are operational.
63. The final SOW-to-code traceability report includes Sections 130–143.
64. Codex does not declare completion until all implementable P0/P1 requirements in Sections 1–143 have evidence.

## 143.3 Final Codex Directive

Sections 130–143 have the same implementation authority as Sections 1–129.

Codex shall:

```text
1. audit before modifying;
2. map existing implementation to Sections 1–143;
3. reuse/extend before replace;
4. implement missing schema/contracts first;
5. implement cached fail-closed gate architecture;
6. implement extended indicator/pattern registries;
7. implement reference math + Go/Python parity;
8. implement versioned exit profiles and exit-aware backtests;
9. implement calibration/performance evidence and delivery SLO;
10. wire real backend APIs/WebSockets to all required frontend surfaces;
11. add observability/security/compliance activation controls;
12. run unit/integration/golden/replay/E2E/load/chaos/security tests;
13. fix implementable failures rather than documenting them only;
14. keep live trading, live payouts, live payment mutation and production signing keys disabled unless explicitly authorized;
15. produce the final Sections 1–143 traceability matrix and GO/NO-GO report.
```

The final implementation is not complete because a model predicts, a backend compiles, a dashboard renders, or a payment succeeds. It is complete only when trading intelligence, risk, execution, delivery, SaaS control, licensing, commercial ledgers, UI, security, observability, statistical validation, compliance activation state and operational recovery are mutually consistent and evidenced.

---

# 144. Quantum-Inspired Intelligence Plane

## 144.1 Objective

Implement a mathematically explicit, versioned, deterministic and independently validated quantum-inspired evidence layer for:

```text
market-state representation
regime mixtures
context/order effects
path interference hypotheses
liquidity-transition modeling
uncertainty/entropy
first-passage and target-before-stop estimates
feature/variant selection research
portfolio/strategy allocation research
```

It shall not replace:

```text
market-data truth
classical structure/liquidity calculations
hard gates
risk engine
calibration
broker qualification
execution reconciliation
commercial entitlements
```

## 144.2 Plane placement

```text
PYTHON RESEARCH PLANE
  - fit and validate candidate quantum-inspired models
  - generate immutable candidate parameters
  - conduct ablation, walk-forward, OOS and calibration
  - produce signed/versioned artifacts and reports

GO REAL-TIME PLANE
  - load approved small fixed-size model artifact
  - compute deterministic CPU inference
  - expose optional capped evidence contribution
  - emit replayable state and metrics

NESTJS CONTROL PLANE
  - registry, approval, rollout, feature flag and audit
  - no per-tick calls

NEXT.JS PRESENTATION PLANE
  - show honest, approved evidence and health
  - never calculate production quantum state in the browser
```

## 144.3 No-blocking rule

The Go decision loop may read only an in-process approved model snapshot and current feature snapshot. Model loading, database access, artifact verification and configuration refresh occur asynchronously. Missing, corrupt, stale or incompatible quantum state contributes zero and produces a quality reason; it must not pause the strategy.

## 144.4 Model status

```text
DRAFT
RESEARCH
BACKTESTED
WALK_FORWARD_VALIDATED
OOS_VALIDATED
PAPER
SHADOW
APPROVED_OPTIONAL
ACTIVE_OPTIONAL
SUSPENDED
DEPRECATED
ROLLED_BACK
```

No model self-promotes.

---

# 145. Quantum-Inspired Mathematical Foundation

All variables are dimensionless after approved normalization unless a unit is explicitly stated. `i² = -1`. The use of complex vectors is a modeling device, not evidence of physical quantum behavior.

## 145.1 Hilbert-space state

For an approved finite dimension `d`, define:

```math
\mathcal{H}=\mathbb{C}^{d}
```

A pure market-state representation is:

```math
|\psi_t\rangle = [\alpha_{1,t},\ldots,\alpha_{d,t}]^T,
\qquad
\langle\psi_t|\psi_t\rangle = \sum_{k=1}^{d}|\alpha_{k,t}|^2=1.
```

Complex amplitudes may be expressed as:

```math
\alpha_{k,t}=\sqrt{p_{k,t}}e^{i\phi_{k,t}}.
```

`p_k` and `φ_k` are learned/defined model parameters or deterministic transforms. They are not physical wave amplitudes.

Global phase shall not change any output:

```math
|\psi\rangle \sim e^{i\gamma}|\psi\rangle.
```

A global-phase invariance test is mandatory.

## 145.2 Inner product, norm and similarity

```math
\langle\phi|\psi\rangle=\sum_k \phi_k^*\psi_k,
\qquad
\|\psi\|_2=\sqrt{\langle\psi|\psi\rangle}.
```

Pure-state fidelity:

```math
F(|\phi\rangle,|\psi\rangle)=|\langle\phi|\psi\rangle|^2.
```

A fidelity/kernel output is a similarity feature only; it cannot independently authorize an order.

## 145.3 Robust feature normalization

For raw feature `x_i` and a training-only reference distribution:

```math
z_i=\frac{x_i-\operatorname{median}(x_i)}
{\max(1.4826\operatorname{MAD}(x_i),\epsilon)}
```

Then bound the value:

```math
\tilde z_i=\tanh(\operatorname{clip}(z_i,-z_{max},z_{max})).
```

Normalization parameters are fitted on training data only, stored with the model artifact, and used identically in Python and Go. Missing/invalid values follow the declared feature policy and quality mask; they are never silently converted into valid zeros.

## 145.4 Deterministic state encoding

### Direct complex encoding

For learned/versioned matrices `W_R`, `W_I` and offsets `b_R`, `b_I`:

```math
v_R=W_R\tilde z+b_R,
\qquad
v_I=W_I\tilde z+b_I,
```

```math
v=v_R+i v_I,
\qquad
|\psi\rangle=\frac{v}{\max(\|v\|_2,\epsilon)}.
```

### Angle encoding

For bounded features:

```math
\theta_i=\frac{\pi}{2}(1+\tilde z_i),
```

```math
|q_i\rangle=
\cos(\theta_i/2)|0\rangle+
 e^{i\varphi_i}\sin(\theta_i/2)|1\rangle.
```

An unrestricted tensor product grows as `2^n` and is prohibited in the production path unless its dimension and latency are explicitly bounded. Production shall prefer a small direct state, block-factorized state, low-rank state, or fixed-bond tensor representation.

## 145.5 Density matrix and mixed regimes

Pure state:

```math
\rho=|\psi\rangle\langle\psi|.
```

Regime mixture:

```math
\rho_t=\sum_{r=1}^{R}\pi_{r,t}|\psi_{r,t}\rangle\langle\psi_{r,t}|,
\qquad
\pi_{r,t}\ge0,
\qquad
\sum_r\pi_{r,t}=1.
```

Required invariants:

```math
\rho=\rho^\dagger,
\qquad
\rho\succeq0,
\qquad
\operatorname{Tr}(\rho)=1.
```

A controlled noise seed may be represented by:

```math
\rho'=(1-\eta)\rho+\eta I/d,
\qquad 0\le\eta\le1.
```

`η` is a versioned parameter, not a live optimizer output.

## 145.6 Measurement and Born-style outputs

Define positive semidefinite measurement operators:

```math
E_B,E_S,E_N\succeq0,
\qquad
E_B+E_S+E_N=I.
```

Raw outcomes:

```math
q_B=\operatorname{Tr}(\rho E_B),
\quad
q_S=\operatorname{Tr}(\rho E_S),
\quad
q_N=\operatorname{Tr}(\rho E_N).
```

Required:

```math
0\le q_k\le1,
\qquad
q_B+q_S+q_N=1
```

within declared numerical tolerance.

A directional observable may be:

```math
A=E_B-E_S,
\qquad
m=\operatorname{Tr}(\rho A)\in[-1,1].
```

**Critical rule:** `q_B`, `q_S`, `q_N` are raw quantum-inspired model outputs. They are not subscriber-facing trading probabilities until calibrated against a precise target, horizon, broker/cost model and `exit_profile_id`.

## 145.7 Uncertainty, entropy and purity

Von Neumann entropy:

```math
S(\rho)=-\operatorname{Tr}(\rho\log\rho)
       =-\sum_j\lambda_j\log\lambda_j.
```

Normalized entropy:

```math
H_N=\frac{S(\rho)}{\log d}\in[0,1].
```

Purity:

```math
\mathcal{P}=\operatorname{Tr}(\rho^2),
\qquad
1/d\le\mathcal{P}\le1.
```

High entropy/low purity may reduce optional contribution or produce an abstention suggestion. It cannot bypass the classical NO-TRADE policy.

## 145.8 Observables, commutators and incompatibility diagnostic

For Hermitian observables `A` and `B`:

```math
[A,B]=AB-BA.
```

Variance:

```math
(\Delta A)^2=\operatorname{Tr}(\rho A^2)-\operatorname{Tr}(\rho A)^2.
```

Robertson diagnostic:

```math
\Delta A\Delta B\ge\frac12|\operatorname{Tr}(\rho[A,B])|.
```

This may quantify model-order/context sensitivity. It must never be described as a physical uncertainty principle governing gold prices.

## 145.9 Unitary context evolution

For a Hermitian context operator/Hamiltonian `H_c`:

```math
U_c(\Delta\tau)=e^{-iH_c\Delta\tau},
\qquad
U_c^\dagger U_c=I,
```

```math
\rho' = U_c\rho U_c^\dagger.
```

`τ` is model/event time. It is not physical time and does not replace UTC timestamps.

One deterministic construction is:

```math
H_t=H_0+\sum_j g_j(\tilde z_t)G_j,
```

where every `G_j` is Hermitian and all functions/parameters are versioned.

## 145.10 Open-system/decoherence update

Continuous research form:

```math
\frac{d\rho}{d\tau}
=-i[H,\rho]
+\sum_j\gamma_j
\left(
 L_j\rho L_j^\dagger
 -\frac12\{L_j^\dagger L_j,\rho\}
\right).
```

For live deterministic inference, prefer a tested completely positive trace-preserving discrete channel:

```math
\rho_{t+1}=\sum_jK_j\rho_tK_j^\dagger,
\qquad
\sum_jK_j^\dagger K_j=I.
```

This can model controlled forgetting, regime noise or stale-evidence decay. It is not a physical market law.

## 145.11 Measurement/update rule

For observed event `y` with operator `M_y`:

```math
p(y)=\operatorname{Tr}(M_y\rho M_y^\dagger),
```

```math
\rho_y=
\frac{M_y\rho M_y^\dagger}
{\max(p(y),\epsilon)}.
```

Event order must match actual event time and sequence. No future event may update a historical state.

## 145.12 Interference model

For alternative evidence paths with amplitudes:

```math
A_k=\sqrt{w_k}e^{i\phi_k},
```

```math
P\propto\left|\sum_k A_k\right|^2
=\sum_k w_k
+2\sum_{j<k}\sqrt{w_jw_k}\cos(\phi_j-\phi_k).
```

Interference terms model context interaction such as:

```text
liquidity sweep + structure shift
structure + GC flow
VWAP reclaim + volatility regime
macro direction + HTF structure
```

Phases must be fitted only under the approved validation policy, regularized, versioned and ablation-tested. Arbitrary hand-tuned phases are prohibited in active models.

## 145.13 Sequential/context-order probability

For projectors/effects `P_A` then `P_B`, a sequence-sensitive research probability is:

```math
p(B\mid A)=
\frac{\operatorname{Tr}(P_BP_A\rho P_AP_B)}
{\max(\operatorname{Tr}(P_A\rho),\epsilon)}.
```

This may model genuine event order such as “sweep, then CHoCH, then retest.” It must not be used to reorder data after the fact.

## 145.14 Composite states and dependency diagnostics

For a composite representation:

```math
\mathcal H=\mathcal H_{price}\otimes
\mathcal H_{flow}\otimes
\mathcal H_{macro}.
```

Reduced state:

```math
\rho_A=\operatorname{Tr}_B(\rho_{AB}).
```

Mutual-information diagnostic:

```math
I(A:B)=S(\rho_A)+S(\rho_B)-S(\rho_{AB}).
```

Terms such as “entanglement” must remain internal mathematical terminology and must not be marketed as physical entanglement of markets.

## 145.15 State distance and drift

Mixed-state fidelity:

```math
F(\rho,\sigma)=
\left[
\operatorname{Tr}\sqrt{\sqrt\rho\sigma\sqrt\rho}
\right]^2.
```

Trace distance:

```math
D(\rho,\sigma)=\frac12\|\rho-\sigma\|_1.
```

These may support regime-change and model-drift diagnostics. Thresholds require OOS validation.

---

# 146. Quantum Walk, Path Ensemble, Kernel and Optimization Research

## 146.1 Discrete quantum-inspired walk

Represent a liquidity/price-level graph with position state and directional coin state:

```math
|\Psi_{n+1}\rangle=S(C_n\otimes I)|\Psi_n\rangle.
```

Where:

- `C_n` is a versioned context-dependent unitary/contractive transform;
- `S` shifts state between adjacent or graph-connected liquidity levels;
- graph nodes may represent broker-normalized price bins, BSL/SSL pools, VWAP bands, POC/VAH/VAL or approved structure levels.

Output distribution:

```math
p_n(x)=\sum_c|\langle x,c|\Psi_n\rangle|^2.
```

Research outputs may include target-before-stop mass, first-passage asymmetry and probability mass near invalidation. Production use requires fixed graph construction, no future levels and realistic spread/cost handling.

## 146.2 Open/decoherent walk

A stochastic/decoherent channel may be applied:

```math
\rho_{n+1}=\sum_jK_jU_n\rho_nU_n^\dagger K_j^\dagger.
```

This is preferred over unstable long coherent walks on noisy market data.

## 146.3 Path-ensemble model

For a path `x_0,...,x_T`, a dimensionless Euclidean action seed is:

```math
S_E[x]=\sum_{t=0}^{T-1}
\left[
\frac{(\Delta x_t-\mu_t\Delta t)^2}
{2\sigma_t^2\Delta t+\epsilon}
+V(x_t,t)\Delta t
\right].
```

Weight:

```math
w[x]=\exp(-S_E[x]/\lambda).
```

Target-before-stop estimate:

```math
\hat p_{TP<SL}=
\frac{\sum_x w[x]\,\mathbf1\{TP\text{ hit before }SL\}}
{\sum_x w[x]}.
```

`V` may encode approved liquidity barriers, spread/cost penalties and session/news constraints. This is a quantum-inspired path ensemble evaluated offline or within a strictly bounded precomputed approximation. It does not justify a real-time Monte Carlo loop on each tick.

## 146.4 Quantum-inspired kernel

For feature map `|ψ(x)⟩`:

```math
K(x,z)=|\langle\psi(x)|\psi(z)\rangle|^2.
```

For density states, use an approved fidelity-based kernel. Kernel models are trained and validated offline. Live Go inference must use a bounded representation such as fixed support vectors, random-feature approximation or compiled low-rank form with measured latency.

## 146.5 Variational model

A parameterized state may be written:

```math
|\psi(x;\theta)\rangle=U(x;\theta)|0\rangle,
```

and output:

```math
m_k(x;\theta)=
\langle0|U^\dagger O_kU|0\rangle.
```

A parameter-shift gradient for compatible parameterization is:

```math
\frac{\partial m}{\partial\theta_j}
=\frac12
\left[m(\theta_j+\pi/2)-m(\theta_j-\pi/2)\right].
```

Training remains offline. No online gradient update may mutate an active model.

## 146.6 QUBO for research optimization

Binary candidate selection `x_i∈{0,1}` may use:

```math
\min_x\quad x^TQx+c^Tx+
\lambda_f(\sum_i x_i-K)^2+
\lambda_r\,\text{RiskPenalty}(x)+
\lambda_l\,\text{LatencyPenalty}(x).
```

Applications:

```text
feature subset proposal
strategy-variant proposal
candidate session/regime selection
portfolio allocation research
latency-aware model compression
```

Optimization runs on classical deterministic solvers/simulated annealing under controlled seeds. Output status is `CANDIDATE`; it cannot write ACTIVE configuration.

## 146.7 Portfolio/strategy allocation research

For candidate weights `w`:

```math
\min_w\quad
-w^T\mu+\lambda w^T\Sigma w
+\kappa\,\text{Turnover}(w)
+\xi\,\text{TailRisk}(w)
```

subject to aggregate XAUUSD exposure, margin and entitlement constraints. The production risk engine remains authoritative; allocation research cannot bypass Section 25/25A/25B.

---

# 147. Quantum-Inspired Inference Contract

## 147.1 Input snapshot

```text
quantum_input_snapshot_id
symbol
strategy_definition_id
prediction_target_id
exit_profile_id
broker_profile_id
feature_snapshot_id
feature_map_version
normalization_version
market_event_time_utc
broker_time_utc_plus_3
session_state
regime_state
quality_mask
input_vector
```

## 147.2 Deterministic inference sequence

```text
1. Verify model artifact signature/hash and compatibility.
2. Verify feature version, strategy, target, exit and broker profile.
3. Verify required input freshness/quality.
4. Apply stored robust normalization.
5. Encode the bounded state.
6. Construct pure/mixed density state.
7. Apply only approved context/channel transitions.
8. Measure BUY/SELL/NO-TRADE raw outputs.
9. Calculate entropy, purity and quality.
10. Map raw output to a bounded optional contribution.
11. Pass the fused candidate through the existing calibration and hard-risk process.
12. Store the candidate-level reproducibility snapshot/hash.
```

## 147.3 Output contract

```text
quantum_evaluation_id
model_id
model_version
artifact_hash
strategy_definition_id
prediction_target_id
exit_profile_id
feature_snapshot_id
raw_buy
raw_sell
raw_no_trade
directional_expectation
entropy
purity
state_quality
optional_contribution_long
optional_contribution_short
inference_latency_us
reason_codes[]
evaluated_at_utc
evaluated_at_broker_utc_plus_3
```

## 147.4 Numerical policy

- use deterministic `float64`/`complex128` semantics unless a separately validated lower-precision path exists;
- fixed matrix dimensions and preallocated buffers in Go;
- reject NaN/Inf;
- Hermitian symmetrization may use `(ρ+ρ†)/2` only as a documented numerical correction;
- trace renormalization is allowed only inside tolerance;
- a materially negative eigenvalue or invalid probability is `QUANTUM_NUMERIC_INVALID` and contributes zero;
- record tolerances in the model artifact;
- no nondeterministic parallel reduction in the authoritative path.

---

# 148. Fusion with Existing Predict-A-Trade Pillars

## 148.1 Optional contribution only

The quantum-inspired layer is an optional evidence group. It has no authority over:

```text
DATA_QUALITY
SESSION
NEWS
SPREAD
SLIPPAGE
TOTAL_COST
RISK
EXPOSURE
MARGIN
BROKER_PROFILE
LICENSE
ENTITLEMENT
EXECUTION_PERMISSION
EMERGENCY_STOP
```

## 148.2 Contribution formula

A seed bounded contribution is:

```math
q_{dir}=q_B-q_S,
```

```math
q_{quality}=Q_{data}(1-H_N)\,Q_{parity}\,Q_{freshness},
```

```math
c_Q=\operatorname{clip}
\left(w_Q q_{dir}q_{quality},-c_{max},c_{max}\right).
```

For long/short:

```math
S_L'=\operatorname{clip}(S_L+c_Q,0,100),
```

```math
S_S'=\operatorname{clip}(S_S-c_Q,0,100).
```

All factors, cap and sign conventions are versioned.

Initial governance seed:

```text
maximum absolute quantum contribution: <= 10 score points
maximum configured confluence share: <= 10%
missing/stale/invalid quantum state: zero contribution
hard-gate failure: NO-TRADE regardless of contribution
classical mandatory pillars: still required
```

A strategy-specific policy may use a smaller cap. A higher cap requires a new validation/promotion evidence package.

## 148.3 No probability averaging

Do not average a classical calibrated probability and a Born-style output.

The fused model output must be recalibrated for the exact:

```text
strategy definition
prediction target
horizon
exit profile
broker/cost profile
session/regime scope
```

## 148.4 Expected-utility decision

For action `a∈{BUY,SELL,ABSTAIN}` and outcome `y`:

```math
EU(a|x)=\sum_y \hat p(y|x)u(a,y)-C(a|x).
```

Where `C` includes spread, commission, expected slippage, latency cost and carry when applicable. Trading remains authorized only if:

```text
all hard gates PASS
mandatory classical confluence PASS
net expectancy policy PASS
EU(best trade) > approved abstention threshold
risk engine PASS
```

## 148.5 Explanation

User-facing explanation shall use plain evidence statements, for example:

```text
Optional state model supports bullish continuation.
Optional state model is uncertain; no contribution applied.
Optional state model conflicts with classical evidence; score reduced within its approved cap.
```

Do not show mystical terminology. Admin/research views may expose state entropy, purity, interference terms, model version, ablation and raw outputs.

---

# 149. Strategy-Specific Quantum-Inspired Profiles

Each profile is a research seed and requires the existing promotion sequence.

## 149.1 STANDARD_SCALPING

```text
Context: H1 + M30
Setup: M15 + M5
Entry: M5
Timing: M1/tick
Suggested state dimension seed: 8–16
Evaluation trigger: approved candidate/feature update, not remote per tick
Quantum hypotheses:
- sweep → structure shift → retest sequence
- liquidity + VWAP + flow context interaction
- regime-mixture uncertainty
Initial optional cap: <= 8 score points
```

It cannot replace the mandatory liquidity/structure/cost gates.

## 149.2 ULTRA_SCALPING

```text
Context: M15 + M5
Setup: M5 + M1
Timing: tick/sub-minute when authoritative
Suggested state dimension seed: 4–8, fixed-size
Evaluation trigger: candidate event with precomputed features
Initial quantum inference budget seed: p99 <= 250 microseconds on qualified host
Initial optional cap: <= 6 score points
```

Requirements:

- no Python, QPU, database or network dependency;
- no heap allocation in the hot evaluation function after warmup where practical;
- no use of unavailable GC/depth features;
- missing flow capability is handled by the existing capability policy;
- quantum output cannot rescue a setup that fails cost-to-target, latency, spread or slippage gates.

## 149.3 STANDARD_SWING

```text
Context: D1 + H4
Setup: H1 + M15
Suggested state dimension seed: 16–32
Evaluation trigger: feature/bar/macro state change
Hypotheses:
- mixed macro/structure regime
- path mass to TP1/TP2/TP3 before invalidation
- contextual DXY/real-yield/HTF-liquidity interaction
Initial optional cap: <= 8 score points
```

## 149.4 TREND_SWING

```text
Context: W1 + D1 + H4
Suggested state dimension seed: 16–64 with bounded low-rank implementation
Evaluation trigger: H1/H4/macro state change
Hypotheses:
- trend-regime persistence/decoherence
- macro/slow-flow/HTF-location mixture
- multi-day target-before-invalidation path ensemble
Initial optional cap: <= 8 score points
```

Carry/swap remains part of the ordinary net-expectancy and risk calculations.

## 149.5 Distinct artifacts

The four strategies may not share one generic artifact under different labels unless independent validation proves that the artifact plus strategy-specific measurement/profile produces genuinely distinct versioned behavior. Every active pairing stores its own approval and calibration linkage.

---

# 150. UTC+03:00 Broker-Time Standard

## 150.1 Authoritative clocks

Every event stores a UTC instant. Broker display and broker-aligned candle time use fixed offset `+03:00`.

```math
t_{broker}=t_{UTC}+3\text{ hours}
```

```math
t_{UTC}=t_{broker}-3\text{ hours}
```

Required representation:

```text
event_time_utc            = 2026-08-17T10:15:30.123456Z
broker_time               = 2026-08-17T13:15:30.123456+03:00
broker_utc_offset_minutes = 180
broker_time_profile_id
source_time
gateway_receipt_time_utc
server_publish_time_utc
```

Never store an ambiguous timestamp such as `2026-08-17 13:15:30` without offset/profile metadata.

## 150.2 Fixed offset versus venue timezone

`UTC+03:00` is the broker clock, not a replacement for venue timezones.

Use IANA timezones for:

```text
Europe/London — LBMA and London sessions/fixes
America/New_York — New York/US data context
America/Chicago — CME/COMEX venue context where applicable
Asia/Tokyo and other approved session contexts
```

Venue events are converted:

```text
venue-local time -> UTC using IANA/DST -> broker display +03:00
```

London and New York DST changes must therefore change their displayed broker-time windows. The broker offset itself remains `+03:00` under this requirement.

## 150.3 Broker-aligned candle buckets

For UTC epoch time `t`, broker offset `O=10,800,000 ms` and candle duration `Δ`:

```math
bucketStartUTC=
\Delta\left\lfloor\frac{t+O}{\Delta}\right\rfloor-O.
```

Examples for fixed broker `UTC+03:00`:

```text
Broker D1 00:00 begins at 21:00 UTC on the previous UTC date.
Broker H4 boundaries 00/04/08/12/16/20 map to UTC 21/01/05/09/13/17.
Broker week Monday 00:00 begins Sunday 21:00 UTC.
```

Each strategy declares one candle alignment profile:

```text
BROKER_ALIGNED_UTC_PLUS_3
UTC_ALIGNED
VENUE_ALIGNED
SOURCE_NATIVE
```

The live chart, historical candles, feature engine and replay must use the same selected alignment. Silent mixing is prohibited.

## 150.4 Signal timestamps and TTL

Every signal shows:

```text
created_at_utc
created_at_broker_time (+03:00)
expires_at_utc
expires_at_broker_time (+03:00)
remaining_ttl_ms from monotonic/local receipt calculation
source_event_time_utc
```

TTL and idempotency use UTC instants plus monotonic timers where applicable. They must not depend on a formatted wall-clock string.

## 150.5 Clock synchronization

Seed operational targets, to be validated per environment:

```text
server clock offset alert: >25 ms
server clock critical: >100 ms
Windows Agent clock warning: >100 ms
Windows Agent clock critical for ultra execution: strategy-profile value, seed >250 ms
```

Use NTP/chrony or approved equivalent and monotonic latency timers. A critical skew causes execution denial for affected strategies.

## 150.6 Broker-time mismatch

The Windows Agent/EA shall compare terminal server time to expected `UTC+03:00` behavior. Material persistent mismatch yields:

```text
BROKER_TIME_PROFILE_MISMATCH
EXECUTION_DENIED
```

until the broker profile is reconciled. No code may simply add three hours twice.

## 150.7 Dashboard timezone behavior

Default trading display:

```text
Broker Time (UTC+03:00)
```

Optional secondary views:

```text
UTC
User Local Time
Venue Local Time
```

Every displayed time includes a label/offset. Signal history filters use UTC boundaries internally and clearly state the display timezone.

---

# 151. Low-Latency Real-Time Architecture

## 151.1 Critical path

```text
Feed socket
→ timestamp/sequence capture
→ normalization/bad-tick checks
→ in-process ring/bounded queue
→ candle/feature incremental update
→ immutable market snapshot
→ cheap hard-gate precheck
→ strategy candidate
→ optional quantum-inspired CPU evaluation
→ confluence/calibration
→ hard risk
→ signal state transition
→ priority publication
```

The path must not synchronously wait for:

```text
PostgreSQL
TimescaleDB
Valkey network lookup when local state is healthy
NestJS
Next.js
Python
AI provider
QPU/quantum cloud
billing/referral services
remote broker acknowledgement
```

## 151.2 Hot-state design

Use:

- single-writer or partitioned ownership by symbol/stream where practical;
- immutable generation-stamped snapshots;
- preallocated buffers and bounded queues;
- lock-free/read-copy-update patterns where justified and tested;
- local mirrors of gate, license lease and account-risk state;
- asynchronous persistence and recovery through approved outbox/journal mechanisms;
- separate priority classes for market deltas, signals, risk, execution and control events.

Market-display updates may be coalesced under pressure. Signal lifecycle, risk, execution and audit events must not be silently dropped.

## 151.3 Seed internal latency budgets

These are engineering targets, not guarantees of Internet or broker latency. They require benchmark confirmation on the production host.

| Stage | Initial seed target |
|---|---:|
| receipt → normalized tick | p99 <= 0.5 ms |
| incremental feature update | p99 <= 2 ms |
| hard cached gates | as Section 131, generally <1 ms each |
| optional quantum inference | p99 <= 0.25 ms ultra; <=1 ms other profiles |
| candidate + scoring | p99 <= 5 ms |
| risk decision | p99 <= 2 ms |
| signal serialization/publication enqueue | p99 <= 2 ms |
| total internal receipt → publish | p99 <= 10 ms for qualified ultra host; <=25 ms general seed |

If a target cannot be met, the measured value is reported and the affected strategy is degraded or denied according to its execution profile. External source, network, Windows, terminal and broker latencies remain separately measured.

## 151.4 Allocation and pause control

For ultra-sensitive functions:

```text
fixed-size matrices
preallocated scratch memory
bounded channels
no reflection in hot path
no unbounded JSON work in decision loop
no synchronous logging
no per-candidate artifact parsing
profiled garbage-collection behavior
```

A lower-level optimization is permitted only after correctness and parity tests.

## 151.5 Multi-node publication

Cross-node fan-out may use an approved durable or resumable stream mechanism. Valkey Streams may be used for short-lived replay/shared fan-out when consistent with the existing architecture. The authoritative signal record persists asynchronously in PostgreSQL. No cross-node bus may become a reason to duplicate execution commands.

## 151.6 Backpressure

```text
price/tick visualization: latest-value or bounded coalescing allowed
candle update: sequence-aware coalescing allowed
signal and lifecycle: durable/replayable, no silent loss
execution/risk: highest priority, idempotent and reconciled
slow client: disconnect with resumable cursor after policy threshold
```

---

# 152. Live XAUUSD Chart and Dashboard Streaming

## 152.1 Server-authoritative state

The browser renders server-authoritative candles and overlays. It does not recompute production indicators, quantum state, structure, liquidity, SL/TP, score, probability or risk decisions.

## 152.2 Data flow

```text
Initial chart request
→ REST snapshot: historical candles + active overlay snapshot + stream cursor
→ authenticated Go WebSocket connection
→ incremental candle/tick/overlay/signal events
→ Web Worker decode/sequence check
→ Canvas/WebGL renderer
→ requestAnimationFrame paint
```

Next.js may provide authentication/bootstrap pages, but high-frequency market streaming should connect to the authorized real-time gateway rather than route every tick through a React server render.

## 152.3 Live event contract

```text
event_id
stream_id
sequence
type
schema_version
symbol
timeframe
alignment_profile_id
event_time_utc
broker_time_utc_plus_3
server_publish_time_utc
payload
checksum/signature where required
```

Event types include:

```text
PRICE_DELTA
CANDLE_OPEN
CANDLE_UPDATE
CANDLE_CLOSE
SPREAD_UPDATE
MARKET_STATE
OVERLAY_SNAPSHOT
OVERLAY_DELTA
SIGNAL_STATE
RISK_STATE
NEWS_STATE
HEARTBEAT
RESYNC_REQUIRED
```

## 152.4 Resume and gap handling

Client sends:

```text
stream_id
last_acknowledged_sequence
last_snapshot_id
```

Gateway either replays valid missed events or sends a fresh snapshot. An expired signal is replayed only as historical/non-executable state. Sequence gaps display `RESYNCING`; the chart must not pretend to be live.

## 152.5 Render performance

- decode and sequence checking in a Web Worker where supported;
- render with Canvas/WebGL for dense data;
- avoid putting every tick into global React state;
- batch visual price updates to the next animation frame, normally 16–50 ms;
- signals/risk alerts bypass ordinary visual coalescing;
- virtualize long tables;
- cap visible history and use level-of-detail aggregation;
- preserve linked crosshair/time across strategy timeframes;
- show last-event age and connection quality.

Seed healthy-client targets:

```text
market event → browser receipt p95: measured, target <=100 ms on qualified network
browser receipt → paint p95: <=50 ms
combined live tick-to-paint p95 target: <=150 ms on qualified network
sustained chart frame rate target: >=55 FPS on approved desktop profile
reconnect/resume target: <=2 seconds when network/service permits
```

These are operational targets, not broker-fill guarantees.

## 152.6 Required chart displays

All Section 32A requirements remain, plus:

```text
Broker Time UTC+03:00 axis label
UTC tooltip value
bid/ask/spread state
last source/gateway/publish/render latency
quantum optional-contributor status when approved
quantum state quality/uncertainty in admin/research view
NO-TRADE reason marker
exit-ladder fractions
confirmed TP/partial-close/breakeven/trailing events
candle alignment profile
feed quality and replay/resync state
```

## 152.7 Historical/live continuity

The last historical candle and first live candle must reconcile by:

```text
symbol
source
alignment profile
open time UTC
broker open time +03:00
price precision
sequence/generation
quality state
```

A mismatch causes snapshot repair/resync, not a visual splice.

---

# 153. Quantum and Real-Time Data Model Requirements

Canonical migrations shall adapt names to the audited repository while preserving these logical entities.

## 153.1 Quantum model registry

```text
quantum_model_definitions
quantum_model_versions
quantum_feature_maps
quantum_normalization_profiles
quantum_measurement_profiles
quantum_channel_profiles
quantum_strategy_bindings
quantum_model_artifacts
quantum_evaluations
quantum_validation_reports
quantum_ablation_reports
quantum_drift_reports
quantum_candidate_optimizations
```

Minimum model-version fields:

```text
model_id
version
strategy_id
strategy_definition_id
prediction_target_id
exit_profile_id
broker_profile_scope
feature_map_version
normalization_version
state_dimension
rank_or_bond_limit
measurement_profile_id
channel_profile_id
artifact_uri/reference
artifact_sha256
signature/key_id
code_commit
training_dataset_id
training_period
status
approved_by
approved_at
effective_from
rolled_back_to
```

## 153.2 Candidate-level evaluation storage

Do not persist a full density matrix for every market tick. Persist for signal candidates/shadow samples as policy requires:

```text
quantum_evaluation_id
feature_snapshot_id
model_version
input_hash
state_summary or compressed approved state
raw outputs
entropy
purity
quality
contribution
latency
reason codes
timestamps UTC and +03:00
```

Retention is explicit.

## 153.3 Time alignment

Add or confirm:

```text
time_profiles
candle_alignment_profiles
broker_time_validation_events
clock_skew_events
```

## 153.4 Live-chart delivery

Add or confirm:

```text
chart_snapshot_metadata
stream_cursors
signal_delivery_receipts
client_render_telemetry aggregates
stream_gap_events
resync_events
```

Avoid high-volume per-frame PostgreSQL writes. Aggregate non-audit rendering telemetry.

---

# 154. Quantum and Real-Time API / WebSocket Requirements

Actual route prefixes follow the existing `/api/v1` contract.

## 154.1 User/read APIs

```text
GET /xauusd/chart/snapshot
GET /xauusd/chart/candles
GET /signals/:id/decision-snapshot
GET /signals/:id/optional-model-evidence
GET /time/broker-profile
GET /time/session-conversions
```

User optional-model evidence is exposed only when approved for the strategy and phrased as evidence, not certainty.

## 154.2 Admin/research APIs

```text
GET  /admin/quantum/models
GET  /admin/quantum/models/:id
POST /admin/quantum/models/:id/approve-optional
POST /admin/quantum/models/:id/suspend
POST /admin/quantum/models/:id/rollback
GET  /admin/quantum/validation-reports
GET  /admin/quantum/ablation-reports
GET  /admin/quantum/drift
GET  /admin/time/broker-profile-health
GET  /admin/realtime/latency
GET  /admin/realtime/stream-health
```

Consequential writes require RBAC, step-up/MFA where configured, reason, version conflict protection and immutable audit.

## 154.3 WebSocket channels

Extend existing channels rather than duplicate them:

```text
/ws/xauusd/ticks or approved price-delta channel
/ws/xauusd/candles
/ws/xauusd/chart-overlays
/ws/xauusd/signals
/ws/xauusd/market-state
/ws/xauusd/execution
/ws/xauusd/health
```

Quantum state is normally embedded as a small optional evidence object in the authoritative signal/market-state event. Do not create a high-frequency public “quantum” stream that leaks model internals or adds avoidable load.

---

# 155. Calibration, Validation and Statistical Governance

## 155.1 Born output is not calibrated probability

Raw measurement output must be calibrated through the Section 130/107 process. The calibration dataset must use the exact target and exit profile.

## 155.2 Incremental-value test

Every candidate report compares at minimum:

```text
classical production/baseline model
classical model + same number of non-quantum interaction features
quantum-inspired optional model
fused classical + quantum-inspired model
NO-TRADE/zero-exposure baseline
```

Required evaluation includes:

```text
net expectancy after costs
Brier score
ECE/reliability
log loss where appropriate
AUC only as secondary discrimination metric
profit factor
max drawdown
MFE/MAE
turnover
fill/reject/slippage assumptions
latency cost
session/regime stability
confidence intervals
```

## 155.3 Multiple testing and overfitting

Record the number of:

```text
feature maps
state dimensions
phase specifications
channels
measurement profiles
hyperparameter sets
sessions/regimes
strategy variants
```

tested. Use an approved correction/deflated-performance or holdout policy. Repeated tuning against locked OOS data is prohibited.

## 155.4 Ablation

Ablate:

```text
complex phase/interference
mixed-regime state
channel/decoherence
quantum walk/path feature
kernel component
optional contribution
```

If a simpler real-valued/classical model performs equivalently within uncertainty, prefer the simpler model unless the quantum-inspired representation has a documented operational advantage.

## 155.5 Promotion to OPTIONAL_CONTRIBUTOR

Requires all existing promotion gates plus:

```text
Go/Python parity PASS
mathematical invariant tests PASS
latency budget PASS
incremental net-value evidence PASS
calibration PASS
no material tail-risk deterioration
ablation report approved
explanation reviewed
security/artifact-signing PASS
paper/shadow duration and labels sufficient
human approval
rollback tested
```

Failure returns zero quantum contribution; it must not stop classical strategy processing unless an integrity/security failure requires strategy suspension for another reason.

---

# 156. Go/Python Determinism and Math Test Suite

## 156.1 Required invariants

Test:

```text
state norm = 1
rho Hermitian
rho trace = 1
rho positive semidefinite within tolerance
measurement effects positive semidefinite
measurement effects sum to identity
probabilities in [0,1]
probabilities sum to 1
unitary U†U = I
Kraus completeness
channel trace preservation
global phase invariance
no NaN/Inf
same input/version → same output
```

## 156.2 Golden fixtures

Fixtures cover:

```text
pure bullish state
pure bearish state
maximum mixed/no-trade state
high entropy
low entropy
conflicting evidence/interference
missing capability
stale features
UTC/+03:00 boundary
London/NY DST transition
broker D1/H4/week boundary
all four strategies
broker tick-size/digit variants
```

## 156.3 Property/fuzz tests

Generate bounded valid states/channels and verify invariants. Fuzz invalid matrices, malformed artifacts, extreme feature values and sequence disorder. Fail closed on invalid inputs.

## 156.4 Replay

A stored candidate with identical model/feature/strategy/calibration/exit/broker versions must reproduce:

```text
raw outputs
entropy/purity
optional contribution
fused scores
calibrated probability lookup
NO-TRADE/trade decision
reason codes
SL/TP geometry
```

---

# 157. Risk and Execution Safety

## 157.1 Prohibited quantum authority

Quantum-inspired output may not directly:

```text
change risk-per-trade
increase leverage
increase lot size
widen SL
remove SL
bypass spread/slippage/cost gate
bypass news/session gate
bypass margin/stop-out checks
bypass broker mismatch
bypass entitlement/license
submit an order
retry an execution
change referral/billing state
```

## 157.2 Position sizing

All sizing remains Section 25B broker-normalized math. Quantum-inspired confidence is not a direct lot multiplier in v1.0.0. Any future probability-sensitive sizing is a separate strategy/risk-profile version, conservatively capped and independently validated.

## 157.3 Exit geometry

Quantum path/first-passage output may be recorded as research evidence for target selection, but the active versioned `exit_profile` and classical risk/structure engine determine SL/TP/partial close/breakeven/trailing. No target exists unless it satisfies Sections 135–136.

## 157.4 Security failure

Invalid signature/hash or incompatible artifact:

```text
QUANTUM_ARTIFACT_INVALID
quantum contribution = 0
alert admin
retain classical operation if safe
```

If artifact compromise indicates broader supply-chain compromise, normal kill-switch/incident policy applies.

---

# 158. Quantum and Real-Time Observability / SLO Requirements

## 158.1 Quantum metrics

```text
quantum_inference_latency_us p50/p95/p99/max
quantum_evaluation_count
quantum_zero_contribution_count
quantum_invalid_state_count
quantum_artifact_mismatch_count
quantum_trace_error
quantum_min_eigenvalue
quantum_probability_sum_error
quantum_entropy
quantum_purity
quantum_directional_expectation
quantum_contribution_distribution
quantum_classical_disagreement
quantum_incremental_expectancy
quantum_calibration_brier
quantum_calibration_ece
quantum_drift_fidelity
quantum_drift_trace_distance
```

Detailed signal/model identifiers belong in structured logs/analytics, not high-cardinality Prometheus labels.

## 158.2 Time metrics

```text
server_clock_offset_ms
agent_clock_offset_ms
broker_time_offset_observed_minutes
broker_time_profile_mismatch_count
candle_alignment_mismatch_count
session_conversion_error_count
```

## 158.3 Live-chart metrics

```text
chart_snapshot_latency_ms
chart_ws_publish_to_ack_ms
chart_sequence_gap_count
chart_resync_count
chart_late_event_count
client_tick_to_paint_ms aggregate
client_render_fps aggregate
slow_client_disconnect_count
market_delta_coalesced_count
signal_event_drop_count = 0 target
```

## 158.4 Alerts

Alert on sustained:

```text
invalid quantum math state
parity regression
artifact mismatch
quantum latency budget burn
broker UTC+03:00 mismatch
clock skew critical
chart sequence gaps/resync storm
signal delivery SLO burn
internal receipt-to-publish SLO burn
```

---

# 159. Frontend and Admin Requirements

## 159.1 User signal card

Show:

```text
BUY / SELL / NO-TRADE
Strategy
Broker Time UTC+03:00
UTC time in tooltip/details
Created / Expires / TTL
Entry / SL / TP1 / TP2 / TP3
Exit percentages
Gross and estimated net R:R
Calibrated probability + exact target
Grade/sample sufficiency
Data/feed quality
Connection/live latency
Optional model evidence status, if approved
NO-TRADE reason
```

Never show “quantum probability” as if it were calibrated outcome probability.

## 159.2 Admin quantum center

Provide:

```text
model registry and status
strategy/target/exit binding
artifact hash/signature
feature map and dimension
latency benchmark
math invariant test report
Go/Python parity report
backtest/walk-forward/OOS report
calibration/reliability report
ablation/baseline comparison
paper/shadow outcomes
drift/entropy/purity
optional contribution distribution
approval/rollback/audit
```

## 159.3 Live-chart status

Chart always shows one of:

```text
LIVE
DELAYED
RESYNCING
STALE
OFFLINE
REPLAY
```

with last update age. Do not display a static chart as live.

---

# 160. Security, Compliance and Claim Controls

## 160.1 Artifact security

Model artifacts require:

```text
immutable version
SHA-256
signature/key ID
approved source/build
schema version
compatibility manifest
rollback reference
```

The Go engine loads only approved artifacts and verifies them before activation.

## 160.2 Access control

Research parameters, density states, phase parameters and feature maps are confidential/internal unless deliberately published. User APIs expose only approved explanations. Admin mutations follow RBAC/MFA/audit requirements.

## 160.3 Claims

Quantum-inspired features do not weaken Sections 78, 130 and 138. Public/subscriber claims require the same approved evidence package and must identify the precise metric. “Quantum” is not evidence.

## 160.4 UAE activation

Paid signals, assisted/auto execution and referral payouts remain blocked until the applicable Section 138 regulatory/legal/payment-provider approval is valid. Adding quantum-inspired terminology requires marketing/legal review because it may amplify misleading impressions.

---

# 161. Quantum and Real-Time Implementation Phases

These phases are mandatory within the integrated Section 117/142 implementation sequence.

## 161.1 Q0 — Repository and claim audit

```text
locate any existing “quantum” code/marketing
locate live chart/data paths
locate timezone assumptions
locate per-tick allocations/I/O
map existing feature parity harness
classify REUSE/EXTEND/ADAPT/REPLACE/NEW/DEPRECATE
```

## 161.2 Q1 — Time and candle alignment foundation

```text
create/validate UTC+03:00 broker profile
add explicit timestamp fields
implement bucket formula
add DST/venue conversion tests
wire UI labels
reject broker profile mismatch
```

## 161.3 Q2 — Low-latency baseline before quantum

```text
benchmark existing receipt-to-publish path
remove synchronous hot-path I/O
implement immutable snapshots/bounded queues
add latency tracing
prove chart sequence/resume
```

Quantum work must not hide pre-existing latency problems.

## 161.4 Q3 — Reference math

```text
Python reference implementation
Go production implementation
shared golden fixtures
invariant/property/fuzz tests
artifact schema
```

## 161.5 Q4 — Research models

```text
state encoding
mixed regimes
measurement
interference/context
bounded walk/path research
kernel/QUBO candidates
classical baselines
```

## 161.6 Q5 — Exit-aware validation

Use exact strategy, broker, cost and exit profile. Run:

```text
walk-forward
locked OOS
ablation
multiple-testing control
calibration
Monte Carlo/bootstrap drawdown
latency/cost sensitivity
```

## 161.7 Q6 — Go shadow inference

```text
signed artifact loading
fixed-size deterministic inference
zero live contribution
latency benchmark
parity/replay
drift/health dashboard
```

## 161.8 Q7 — Optional contributor rollout

Only after approval:

```text
small capped weight
paper/shadow comparison
canary rollout
strategy-specific enablement
instant rollback/zero contribution
```

## 161.9 Q8 — Live chart completion

```text
REST snapshot
sequenced WS deltas
Web Worker + Canvas/WebGL
UTC/+03:00 display
reconnect/replay/resync
performance and accessibility tests
```

---

# 162. Quantum and Real-Time Required Deliverables

The Section 118 deliverables include all of the following:

```text
Quantum-Inspired Mathematical Reference
Quantum Model and Artifact Schema
Quantum Feature-Map Registry
Quantum Measurement/Channel Registry
Four Strategy Quantum Profile Seeds
Classical Baseline and Ablation Specification
Quantum Go/Python Golden Fixture Set
Quantum Numerical-Invariant Test Report
Quantum Latency Benchmark Report
Quantum Calibration and Incremental-Value Report
Quantum Drift/Observability Dashboard
UTC+03:00 Broker Time Specification
Broker Candle Alignment Test Pack
Venue/DST-to-Broker Conversion Test Pack
Low-Latency Critical-Path Architecture
Receipt-to-Publish Benchmark
Live Chart REST/WebSocket Contract
Chart Sequence/Resume/Resync Test Report
Browser Tick-to-Paint Benchmark
v1.0.0 SOW-to-Code Traceability Matrix
v1.0.0 GO/NO-GO Report
```

---

# 163. Quantum and Real-Time Acceptance Criteria

v1.0.0 is not production-ready until all applicable criteria pass.

1. All Sections 1–143 remain implemented or honestly tracked; no earlier requirement is removed.
2. “Quantum” is explicitly quantum-inspired classical computation, not a guarantee or physical-market claim.
3. No QPU, Python, database or remote service is required in the tick-to-signal path.
4. Every quantum model is linked to an exact strategy, prediction target, exit profile, feature set and broker/cost scope.
5. State encoding and normalization are versioned and fitted without leakage.
6. Pure-state normalization passes.
7. Density matrices are Hermitian, PSD and unit trace within tolerance.
8. Measurement operators are valid and outputs sum to one within tolerance.
9. Unitary/channel invariants pass.
10. Global phase does not change output.
11. Go and Python golden parity passes.
12. Same snapshot/version reproduces the same optional contribution and decision.
13. Invalid/stale/missing quantum state produces zero contribution without blocking safe classical operation.
14. Quantum contribution is capped at the approved strategy-specific limit.
15. Quantum output never bypasses a hard gate, risk, broker, margin, license, entitlement or emergency stop.
16. Born-style outputs are not displayed as calibrated probabilities.
17. The fused model is calibrated for the exact target and exit profile.
18. Incremental-value evidence includes simpler classical interaction baselines.
19. Multiple testing, ablation and locked OOS controls are documented.
20. No live activation occurs without paper/shadow evidence, human approval and tested rollback.
21. Broker time is represented as `+03:00` with UTC retained as the instant truth.
22. D1, H4 and weekly broker-aligned candle boundaries pass exact fixtures.
23. London/New York/CME/LBMA venue events use IANA/DST conversion before display in `+03:00`.
24. Every signal carries UTC and broker `+03:00` creation/expiry times.
25. TTL uses UTC/monotonic timing and is not affected by UI timezone.
26. Persistent broker-time mismatch denies affected auto execution.
27. No mandatory gate performs synchronous external I/O.
28. Internal receipt-to-publish latency is measured by stage and strategy.
29. Ultra quantum inference meets its approved local CPU budget or contributes zero/degrades.
30. Dashboard historical snapshot and live stream reconcile by sequence and candle alignment.
31. WebSocket resume/replay cannot resurrect an expired signal as executable.
32. Chart displays LIVE/DELAYED/RESYNCING/STALE/OFFLINE/REPLAY honestly.
33. The browser does not recalculate authoritative signal truth.
34. Market visualization may coalesce, but signal/risk/execution events are not silently dropped.
35. Tick-to-paint and reconnect benchmarks are reported separately from broker execution latency.
36. Quantum model artifacts are hashed, signed, versioned and rollback-capable.
37. Quantum marketing/performance claims obey evidence and compliance approval rules.
38. Security, load, chaos, replay, parity, chart and timezone tests pass.
39. User/admin APIs and UI enforce server-side authorization and entitlements.
40. The final traceability matrix includes Sections 1–163 and identifies all external activation blockers.

---

# 164. v1.0.0 Integrated Traceability Map

| v1.0.0 requirement | Existing integration point |
|---|---|
| Quantum optional contributor | Sections 12C, 23, 24, 130, 131 |
| Quantum raw output calibration | Sections 15/15A, 17A, 107, 130 |
| Quantum math reference and parity | Sections 134, 137, 141 |
| Quantum target/exit binding | Sections 15A, 135–136 |
| Quantum cost-aware validation | Sections 25A, 106A, 134, 141 |
| Quantum model governance | Sections 21–22, 66A, 107A |
| No-blocking inference | Section 131 |
| Artifact security | Sections 74–75, 95–96 |
| UTC instant truth | Section 9 |
| Broker UTC+03:00 display/alignment | Sections 9, 14A, 32A, 103A plus Section 150 |
| Venue DST correctness | Sections 14A, 98A, 106A |
| Low-latency Go path | Sections 3.1, 5, 89, 127 plus Section 151 |
| Live server-authoritative chart | Sections 32A, 127.2–127.4 plus Section 152 |
| Delivery ACK/resume/SLO | Sections 29, 130.7, 139.4 |
| Hard-risk supremacy | Sections 2, 24–26, 131, 157 |
| Broker execution qualification | Sections 103A–103B |
| UAE/commercial activation | Sections 77–78, 138 |
| Full-stack acceptance | Sections 126–129, 143, 163 |

---

# 165. Practical Signal Decision Order

The final live decision order is:

```text
1. Receive and sequence XAUUSD/related market events.
2. Store UTC event instant; derive broker time UTC+03:00.
3. Validate feed, timestamp, broker profile and candle alignment.
4. Incrementally update authoritative classical features.
5. Read immutable cached gate/account/entitlement snapshots.
6. Short-circuit on any hard veto → NO-TRADE.
7. Evaluate the applicable classical strategy candidate.
8. If an approved compatible optional quantum artifact is healthy,
   compute its bounded CPU contribution; otherwise contribution = 0.
9. Recompute fused score/separation without removing mandatory pillars.
10. Apply the exact target/exit-profile calibration.
11. Apply net R:R, cost-to-target, expected-utility and risk rules.
12. Produce BUY / SELL / NO-TRADE with reason codes.
13. Publish the signed, TTL-bound, sequenced signal and chart overlay.
14. Display UTC+03:00 broker time by default.
15. If execution is entitled and authorized, enqueue an idempotent intent.
16. Reconcile Windows/MT/broker truth asynchronously.
17. Record outcomes for exit-aware calibration, drift and controlled research.
```

---

# 166. Final Objective

Predict-A-Trade v1.0.0 shall use quantum-inspired mathematics only where it improves reproducible, cost-adjusted and calibrated decision quality beyond simpler baselines.

The design objective is:

```text
CLASSICAL MARKET TRUTH
+ DETERMINISTIC HARD GATES
+ VERSIONED OPTIONAL QUANTUM-INSPIRED EVIDENCE
+ EXACT TARGET/EXIT CALIBRATION
+ BROKER-NORMALIZED RISK
+ UTC INSTANT TRUTH / UTC+03:00 BROKER DISPLAY
+ LOW-LATENCY GO INFERENCE
+ SERVER-AUTHORITATIVE LIVE CHART
+ SIGNED/ENTITLED/REPLAY-SAFE DELIVERY
= BUY / SELL / NO-TRADE
```

The platform shall prefer `NO-TRADE` over unsupported certainty, prefer a simpler validated classical model over an unproven complex model, and prefer measured operational quality over marketing language.

No quantum-inspired feature, score, model, plan or dashboard changes the fundamental objective established by Sections 1–143: continuously measure and improve statistically defensible XAUUSD decision quality while protecting users and the platform through hard data-quality, licensing, financial, security, risk, execution and compliance controls.

---

# Part III — Live Intelligence Command Center and Growth Intelligence

# 167. Live XAUUSD Intelligence Command Center

## 167.1 Objective

Build a production-grade, visually advanced, server-authoritative **XAUUSD Live Intelligence Command Center** for subscribers and administrators. The dashboard shall make the platform's existing market-data, indicator, structure, liquidity, macro/news, signal, risk, execution, position, subscription, referral and commission systems visible in one coherent real-time experience without inventing data, recalculating production truth in the browser, or weakening any safety/compliance requirement.

The command center shall answer five questions continuously:

```text
1. WHAT IS GOLD DOING NOW?
2. WHY IS IT DOING IT?
3. WHERE IS PRICE LIKELY TO INTERACT NEXT?
4. IS A TRADE CURRENTLY ELIGIBLE AND EXECUTABLE?
5. WHAT IS HAPPENING TO THE ACTIVE TRADE AND THE SUBSCRIBER'S REFERRAL BUSINESS?
```

The design goal is a modern institutional market-intelligence console, not a decorative retail trading dashboard. Motion, particles, network lines, heatmaps, counters and animations are permitted only when they are driven by actual backend state or events.

## 167.2 Dashboard operating modes

The user interface shall provide four primary modes:

```text
MARKET
TRADING
GROWTH
COMMAND_CENTER
```

### MARKET

Focus on XAUUSD price, market regime, multi-timeframe evidence, indicators, structure, liquidity, volatility, sessions, macro/news, calibrated prediction evidence and feed health.

### TRADING

Focus on signal lifecycle, hard gates, cost/risk eligibility, execution, open position, exit ladder, P&L, MFE/MAE, broker reconciliation and closed-trade outcomes.

### GROWTH

Focus on referral-network growth, subscription conversions, active network, recurring revenue contribution, commission ledger, pending/approved/available/paid amounts, payouts and plan performance.

### COMMAND_CENTER

Combine the highest-value MARKET, TRADING and GROWTH panels into a full-screen desktop/4K trading-room view.

## 167.3 Server-authoritative rule

The browser shall render authoritative data from the Go/NestJS/backend systems. It shall not calculate or invent:

```text
production indicators
market structure
liquidity scores
signal scores
calibrated probability
risk decisions
execution eligibility
commission amounts
referral qualification
payout balances
subscription state
profit/loss truth
```

UI-side derived values are limited to harmless formatting, visual interpolation, countdown presentation from server timestamps, chart geometry, viewport transforms and explicitly non-authoritative presentation calculations.

## 167.4 No-fake-activity rule

The live dashboard shall not use random or synthetic activity to appear busy. Prohibited subscriber-facing behavior includes:

```text
fake ticks
fake fills
fake orders
fake P&L
fake referral registrations
fake commissions
fake payouts
fake liquidity
fake AI-thinking events
random network nodes presented as users
random market-flow particles presented as real flow
invented confidence/probability
invented time-to-target
```

A quiet market or inactive referral network shall be shown honestly as quiet. Approved demo/replay/sandbox data must be unmistakably labeled `DEMO`, `REPLAY` or `SANDBOX` and must never be visually confused with live state.

---

# 168. Dashboard Information Provenance and Truth Classification

## 168.1 Mandatory provenance classes

Every market-intelligence object exposed to the UI shall be classifiable as one of:

```text
OBSERVED
DERIVED
INFERRED
CALIBRATED
COMMERCIAL_LEDGER
```

### OBSERVED

Direct source/broker/terminal facts, including:

```text
bid
ask
last/source price where applicable
spread
tick sequence
tick volume/source volume where supported
broker/terminal timestamps
broker execution acknowledgements
fills
partial fills
position/account truth
MT5 depth/DOM only when the connected broker/source actually supplies it
```

### DERIVED

Deterministic calculations from approved observed data, including:

```text
candles
ATR
RSI
MACD
ADX
moving averages
VWAP
volume/tick-volume profile
session range
volatility regime
market structure
FVG/order-block candidates
transaction-cost state
MFE/MAE
```

### INFERRED

Model/heuristic interpretations such as:

```text
probable liquidity pools
stop-cluster hypotheses
liquidity magnet score
smart-money-style sweep interpretation
regime classification
AI/optional-model evidence
```

Because XAUUSD spot/CFD has no single global centralized order book, inferred liquidity must never be described as known global resting orders unless the source actually provides venue-specific depth and the UI names that venue/source.

### CALIBRATED

Subscriber-facing probabilities or grades that have passed the target/horizon/broker/cost/exit-profile calibration requirements already defined in this SOW.

### COMMERCIAL_LEDGER

Referral, subscription, commission and payout values derived from the authoritative commercial records and immutable ledger, not from frontend arithmetic.

## 168.2 Provenance payload

Where appropriate, dashboard objects shall carry or resolve to:

```text
provenance_class
source_id/feed_id/broker_profile_id
source_event_time_utc
broker_time_utc_plus_3
calculation_version/model_version
quality_state
freshness_ms
reason_codes[]
stream_sequence
```

The UI shall make provenance/quality inspectable without overwhelming normal subscribers.

---

# 169. Global Market Header and Market-State Bar

## 169.1 Always-visible header

The command center shall provide a persistent compact header containing authoritative live state such as:

```text
XAUUSD symbol
bid
ask
mid/reference price where approved
absolute change
percentage change for the selected reference window
current broker time UTC+03:00
current session
connection status
feed quality
last update age
spread
spread regime
ATR/current volatility measure
ADX/trend strength where enabled
tick/flow imbalance when source capability supports it
market regime
signal direction/NO-TRADE state
calibrated confidence/probability only where approved
MT4/MT5/Windows Agent connection state
end-to-end data latency summary
```

The exact change baseline, price convention and timeframe must be labeled and versioned; the UI must not silently mix bid, ask, mid and last conventions.

## 169.2 Market-state vocabulary

The system may expose controlled, versioned state labels such as:

```text
RANGE
COMPRESSION
EXPANSION
BULLISH_EXPANSION
BEARISH_EXPANSION
VOLATILITY_SPIKE
LIQUIDITY_SWEEP
STRUCTURE_SHIFT
TREND_CONTINUATION
REVERSAL_CANDIDATE
NEWS_RISK
COST_STRESS
NO_TRADE
```

Each label must resolve to machine-readable reason codes and source evidence.

## 169.3 Execution state banner

One global state shall always be visible for the selected strategy:

```text
SCANNING
SETUP_FORMING
CONFIRMATION_PENDING
SIGNAL_READY
RISK_CHECK
EXECUTABLE
EXECUTION_PENDING
ORDER_SENT
LIVE_POSITION
MANAGING
EXITING
CLOSED
REJECTED
NO_TRADE
DEGRADED
```

`REJECTED` and `NO_TRADE` must show concise reason(s), for example spread, cost-to-target, stale feed, news veto, risk veto, entitlement or broker mismatch.

---

# 170. Multi-Timeframe Market Pulse and Indicator Intelligence

## 170.1 Multi-timeframe radar

For every timeframe available to the selected strategy and permitted by its data capability, show a compact row containing:

```text
timeframe
trend direction
trend strength
momentum state
structure state
liquidity bias/location
volatility state
VWAP/location state
indicator consensus score if approved
data quality
freshness
```

The dashboard shall respect the four strategy profiles already defined by the SOW. It shall not imply that every timeframe has equal importance. Strategy-specific context/setup/entry/timing timeframes shall be visually distinguished.

## 170.2 Indicator intelligence cards

Indicators shall be presented as **value + interpretation + direction + strength + contribution + freshness**, not merely raw numbers.

Examples of fields to display when enabled by the active profile:

```text
RSI value/state/slope/divergence evidence
MACD line/signal/histogram/crossover/momentum state
ADX value/trend-strength state
ATR value/percentile/volatility regime
EMA/SMA alignment and distance
VWAP/session-VWAP distance and reclaim/rejection state
Bollinger/Keltner state where configured
stochastic/CCI/ROC or other Section 132 indicators when enabled
numeric candle-pattern evidence from Section 133
```

The UI shall read the versioned indicator outputs already used by the authoritative engine. It must not maintain a separate browser implementation that could diverge from production decisions.

## 170.3 Consensus summary

The market pulse may show an aggregate label such as:

```text
STRONG_BULLISH
BULLISH
NEUTRAL
BEARISH
STRONG_BEARISH
CONFLICTED
INSUFFICIENT_DATA
```

The aggregate must be traceable to the approved strategy scoring/confluence profile and must not be presented as a probability unless it is calibrated as one.

---

# 171. Live XAUUSD Chart and Intelligence Overlays

## 171.1 Chart role

The central chart remains the primary visual anchor and extends Section 152. It shall support a professional TradingView-like experience while preserving server-authoritative market truth.

## 171.2 Required selectable overlays

When the underlying engines/data support them, provide toggles for:

```text
bid/ask/current spread
EMA/SMA sets
VWAP/session VWAP
Asian/London/New York session high/low
previous day high/low
previous week high/low
swing highs/lows
BOS
CHoCH
support/resistance
FVG
order blocks/mitigation areas
supply/demand zones
equal highs/equal lows
liquidity pools
liquidity sweeps
rejection/displacement markers
volume or tick-volume profile
POC/VAH/VAL when valid
entry zone
invalidation
SL
TP1/TP2/TP3
partial-close events
breakeven move
trailing-stop state
signal lifecycle markers
news/macro risk windows
predicted/calibrated scenario corridor when approved
```

## 171.3 Overlay density control

The user shall be able to toggle grouped layers such as:

```text
STRUCTURE
LIQUIDITY
FVG_OB
VWAP
SESSIONS
INDICATORS
SIGNALS
TRADES
RISK
NEWS
PREDICTION
```

The platform shall prevent unreadable default clutter and remember user layout preferences per device/account where permitted.

## 171.4 Scenario corridor

If the prediction/calibration subsystem supports scenario probabilities, the chart may render a non-deterministic scenario corridor such as bullish continuation, range and bearish reversal. It shall never render fabricated future candles or imply certainty.

Every scenario must identify:

```text
target definition
horizon
calibration profile
sample sufficiency
updated_at
valid-until/TTL where applicable
```

---

# 172. Liquidity Intelligence and Flow Visualization

## 172.1 Liquidity command panel

Create a dedicated high-visual-value liquidity panel that displays the current price relative to relevant approved levels such as:

```text
buy-side liquidity hypotheses
sell-side liquidity hypotheses
previous day/week/session highs and lows
equal highs/equal lows
swing liquidity
FVG boundaries
order-block/mitigation zones
VWAP bands
POC/VAH/VAL when valid
session liquidity
recent sweep events
rejection/displacement events
```

## 172.2 Liquidity target rows

For each active level, where the engine can support it, display:

```text
price/zone
side/type
provenance class
strength/score
number of supporting factors
distance in price
normalized distance in ATR or approved unit
freshness
last interaction/sweep time
status: ACTIVE / TOUCHED / SWEPT / INVALIDATED / STALE
```

A field such as `probability of reach` or `estimated time to reach` may be displayed **only** if a separately validated/calibrated target-horizon model exists. Otherwise omit it or show `NOT CALIBRATED`; do not convert heuristic scores into pseudo-probabilities.

## 172.3 Liquidity heatmap

Provide an optional vertical price heatmap whose intensity reflects the approved liquidity score or actual venue depth when genuinely available. Hover/inspection shall identify why a zone is intense, for example:

```text
previous-day high
multiple equal highs
M15 supply/OB
three prior rejections
profile node
recent sweep
```

For inferred liquidity, the legend shall explicitly say `INFERRED LIQUIDITY`, not `ORDER BOOK`.

## 172.4 Flow animation

Animated flow lines/particles may connect current price, evidence nodes and target liquidity zones. Their direction, rate and intensity must be computed from actual state changes or approved flow measures. When no valid flow measure exists, render a static relationship map rather than synthetic motion.

---

# 173. AI, Strategy and Evidence Consensus Network

## 173.1 Network visualization

Create a real-time evidence network centered on XAUUSD/selected strategy. Nodes may represent existing evidence groups such as:

```text
TREND
MOMENTUM
STRUCTURE
LIQUIDITY
VOLATILITY
VWAP_PROFILE
MACRO
NEWS
PATTERN
CLASSICAL_MODEL
OPTIONAL_QUANTUM_MODEL
RISK
EXECUTION_QUALITY
DATA_QUALITY
```

Each node shall expose:

```text
state/direction
score or contribution when applicable
quality/freshness
pass/fail/neutral state
reason codes
model/profile version where relevant
```

## 173.2 Center resolution

The center node may resolve to:

```text
BUY
SELL
NO_TRADE
WAIT
```

and may show a calibrated probability only if the exact target/horizon/exit/broker/calibration linkage is valid.

## 173.3 Optional quantum presentation

The optional quantum-inspired model remains governed by Sections 144–160. Subscriber presentation shall use plain language such as `optional state model supports bullish continuation` or `optional state model uncertain — zero contribution`. Raw internal matrices, phases or mystical terminology are not part of the normal user dashboard.

## 173.4 Conflict visualization

When evidence conflicts, the network shall visibly show disagreement rather than collapse it into a misleading single green/red state. The user may inspect which mandatory pillar, gate or cost rule is preventing execution.

---

# 174. Signal Intelligence, Risk and Trade Readiness

## 174.1 Signal card expansion

Extend Section 159.1 so the selected signal/set-up card can display, as applicable:

```text
direction: BUY / SELL / NO_TRADE / WAIT
strategy product
setup type
created_at / expires_at / TTL
entry price or zone
SL
TP1 / TP2 / TP3
partial-close percentages
breakeven/trailing policy
invalidation level/condition
gross R:R
estimated net R:R
expected transaction cost
spread
expected slippage
calibrated probability with exact target/horizon
grade
sample sufficiency
market regime
session
multi-timeframe confluence
key supporting evidence
key conflicting evidence
news/macro risk state
data/feed quality
risk state
execution eligibility
license/entitlement state
broker qualification state
```

## 174.2 Trade-readiness meter

A visual readiness panel may show component scores/states such as:

```text
trend
momentum
structure
liquidity
volatility
cost
risk
data quality
execution quality
```

The final readiness state must map to the authoritative gate/scoring profile and use controlled labels such as `WATCH`, `FORMING`, `READY`, `EXECUTABLE`, `BLOCKED`. Decorative percentage bars must not create a new unvalidated scoring system.

## 174.3 What-is-happening narrative

The dashboard may provide a concise server-generated narrative explaining the current market thesis and invalidation. It must be grounded in structured authoritative evidence and must not be allowed to override or mutate the signal. If an LLM is used for phrasing, it is an asynchronous presentation layer only; deterministic structured facts remain the source of truth.

---

# 175. Live Execution Pipeline and Broker Quality

## 175.1 Execution timeline

For entitled users with relevant permissions, display a real-time event timeline from candidate through execution and reconciliation. Events may include:

```text
SIGNAL_GENERATED
SIGNAL_VALIDATED
DATA_GATE_PASS/FAIL
SESSION_GATE_PASS/FAIL
NEWS_GATE_PASS/FAIL
SPREAD_GATE_PASS/FAIL
SLIPPAGE_GATE_PASS/FAIL
TOTAL_COST_GATE_PASS/FAIL
RISK_GATE_PASS/FAIL
MARGIN_GATE_PASS/FAIL
ENTITLEMENT_GATE_PASS/FAIL
EXECUTION_ELIGIBLE
INTENT_CREATED
ORDER_SENT
BROKER_ACK
PARTIAL_FILL
FILLED
REJECTED
CANCELLED
RECONCILED
```

Each event shall include authoritative timestamps and identifiers sufficient for traceability.

## 175.2 Execution-quality panel

Where data exists, display:

```text
signal/reference price
requested price
fill price
spread at decision
spread at fill
expected slippage
realized slippage
internal decision latency
network/agent latency
broker acknowledgement latency
fill latency
total decision-to-fill latency
commission/fees
estimated all-in transaction cost
execution-quality score if approved and versioned
reject/requote reason
```

The UI shall not label Internet/broker latency as internal engine latency or vice versa.

## 175.3 Subscriber visibility

Sensitive broker/account fields shall be access-controlled. Users see their own execution truth. Administrators see authorized operational detail. Aggregated platform execution-quality analytics must not leak another subscriber's account or personally identifiable information.

---

# 176. Live Position and Trade-Management Intelligence

## 176.1 Active position panel

When a position is open, the trading panel shall automatically prioritize:

```text
symbol
side
strategy
volume/lot size when permitted
entry price
current bid/ask/reference price
unrealized P&L
realized P&L for partial exits
current R
SL/current protective stop
TP1/TP2/TP3
partial-close completion
breakeven state
trailing state
time in trade
MFE
MAE
spread/current cost state
current risk usage
liquidity target distances
market-structure deterioration
signal/invalidation state
broker/terminal reconciliation state
```

## 176.2 Target progress

Progress bars or distance meters for TP1/TP2/TP3 shall be calculated from actual geometry and current authoritative price, not from fake completion percentages. If the position has multiple partial fills or exit legs, progress shall reflect the canonical position/exit model.

## 176.3 Event timeline

Show material trade-management events such as:

```text
position opened
partial fill
TP1 hit/confirmed
partial close confirmed
stop moved to breakeven
trail activated/updated
TP2/TP3 hit
manual intervention
risk action
stop hit
position closed
reconciliation correction
```

---

# 177. Market Event Timeline and Session Intelligence

## 177.1 Market event tape

Provide a replayable chronological tape of significant events, including applicable:

```text
session open/close
session high/low formation
liquidity build-up/sweep
BOS/CHoCH
FVG creation/fill
order-block interaction
VWAP reclaim/rejection
volatility regime transition
news-risk transition
signal candidate
signal issue/expiry
risk rejection
execution events
TP/SL/exit-management events
```

Events shall be generated from authoritative event records, not reconstructed from frontend state after the fact.

## 177.2 Session flow panel

Display Asia, London and New York session state using correct IANA/DST logic while preserving broker display time `UTC+03:00`. Show, where supported:

```text
active session
session start/end in broker time
session range
session high/low
session volatility
session bias/regime
time to next major session/overlap
overlap state
LBMA/CME/approved venue event context
```

---

# 178. Referral Growth and Commission Command Center

## 178.1 Objective

Add a real-time **Growth Intelligence** dashboard integrated with the existing plans, subscriptions, five-level referral structure, commission policy, payout rules and ledgers retained by Section 69. This extension is visualization and analytics over the canonical commercial engine; it does not redefine commission percentages, payment-stage rules, eligibility, refund/clawback rules, or payout compliance.

The canonical first-payment, second-payment-L1-only and third-plus-payment commission policy remains authoritative. Dashboard code shall query policy/versioned ledger results and shall never hard-code alternate commission math.

## 178.2 Subscriber referral summary

Display authorized subscriber metrics such as:

```text
direct referrals
network referrals by level L1-L5
new referrals today
new referrals 7D/30D
verified/qualified referrals when the business rules define them
active paid subscribers in network
inactive/cancelled subscribers
conversion rate
activation rate
renewal rate
churn rate
network growth rate
network recurring revenue contribution where permitted
lifetime qualified sales where permitted
```

Every metric must define its denominator, date range, status filter and timezone.

## 178.3 Commission summary

Display ledger-backed amounts such as:

```text
lifetime earned
current-period earned
pending
held
approved
available
paid
reversed/clawed back
refund-adjusted amount
next eligible payout amount
next payout date/window when known
currency
```

`pending`, `available` and `paid` must have distinct accounting meanings and must reconcile to ledger entries.

## 178.4 Five-level network visualization

Provide an interactive network graph based only on real referral relationships:

```text
YOU
 ├─ LEVEL 1 nodes
 │   └─ LEVEL 2 nodes
 │       └─ LEVEL 3 nodes
 │           └─ LEVEL 4 nodes
 │               └─ LEVEL 5 nodes
```

The graph shall support:

```text
collapse/expand by branch
aggregate large branches
filter by level
filter by active/inactive/qualified state
filter by plan
filter by date range
show network counts and value summaries
show commission-generating event links
```

Subscriber views shall mask personal information according to privacy policy. The graph must not expose arbitrary names, emails, phone numbers, KYC details or payment information of downstream users.

## 178.5 Referral activity stream

A live stream may show privacy-safe events such as:

```text
REFERRAL_CLICK_RECORDED where exposed by policy
REFERRAL_REGISTERED
REFERRAL_QUALIFIED
SUBSCRIPTION_STARTED
SUBSCRIPTION_RENEWED
PLAN_UPGRADED
PLAN_DOWNGRADED
SUBSCRIPTION_CANCELLED
COMMISSION_CREATED
COMMISSION_APPROVED
COMMISSION_AVAILABLE
COMMISSION_REVERSED
PAYOUT_CREATED
PAYOUT_PAID
PAYOUT_FAILED
```

The activity feed shall use masked identity/region/level/plan information only as authorized.

## 178.6 Commission-flow visualization

Provide a flow diagram showing how an actual eligible payment maps into the canonical commission ledger across applicable referral levels. The diagram shall display:

```text
payment/subscription event
payment sequence/category per Section 69
eligible commission levels
policy version
commission amount per applicable level
withholding/hold status where applicable
resulting subscriber commission
ledger status
```

No animation may imply a commission was earned before the authoritative ledger creates the corresponding entry.

## 178.7 Plan performance

Show plan-specific business metrics by reading the canonical plan/subscription catalog rather than hard-coding plan names/prices in the frontend. Applicable metrics include:

```text
active subscribers by plan
new subscriptions by plan
renewals by plan
upgrades/downgrades
cancellations
network MRR/recurring revenue contribution where permitted
commission generated by plan
conversion by plan
```

## 178.8 Forecasts

Projected recurring commission or network revenue may be displayed only as clearly labeled `FORECAST`/`PROJECTED`, with the model/assumptions/date range visible. Forecasts are not wallet balances and cannot be withdrawn.

## 178.9 Rank/status presentation

If the canonical commercial configuration defines partner/rank tiers, the UI may show current rank, progress and remaining criteria. If no such policy exists, the frontend shall not invent Bronze/Silver/Gold/etc. ranks. Rank configuration must be versioned and administered through the commercial control plane.

---

# 179. Command Center Event Architecture

## 179.1 Two event domains

The command center shall expose two logically separate event families:

```text
TRADING_INTELLIGENCE_STREAM
GROWTH_INTELLIGENCE_STREAM
```

Commercial/referral activity must never become a dependency of the Go real-time trading critical path.

## 179.2 Trading event types

Extend Section 152 with typed events such as:

```text
PRICE_DELTA
CANDLE_OPEN
CANDLE_UPDATE
CANDLE_CLOSE
SPREAD_UPDATE
INDICATOR_UPDATE
MULTITF_STATE
MARKET_REGIME_CHANGED
STRUCTURE_UPDATE
LIQUIDITY_UPDATE
LIQUIDITY_SWEEP
PROFILE_UPDATE
SESSION_STATE
MACRO_STATE
NEWS_STATE
SIGNAL_CANDIDATE
SIGNAL_STATE
RISK_STATE
EXECUTION_STATE
POSITION_STATE
EXIT_STATE
BROKER_HEALTH
FEED_HEALTH
HEARTBEAT
RESYNC_REQUIRED
```

## 179.3 Growth event types

Use durable, ledger-safe commercial events such as:

```text
REFERRAL_METRICS_UPDATED
REFERRAL_REGISTERED
REFERRAL_QUALIFIED
SUBSCRIPTION_STATE_CHANGED
PAYMENT_RECORDED
COMMISSION_CREATED
COMMISSION_STATE_CHANGED
PAYOUT_STATE_CHANGED
NETWORK_METRICS_UPDATED
PLAN_METRICS_UPDATED
```

Commercial events may be delivered through NestJS/WebSocket/SSE or the repository's approved event mechanism. They do not require tick-level latency.

## 179.4 Event envelope

All live dashboard events shall follow a versioned envelope compatible with Section 152 and include as applicable:

```text
event_id
stream_id
sequence
type
schema_version
tenant_id/user_scope where applicable
symbol/timeframe for market events
entity_id for commercial events
event_time_utc
broker_time_utc_plus_3 where relevant
server_publish_time_utc
provenance_class
quality_state
payload
```

## 179.5 Priority classes

```text
P0: risk, execution, emergency, position protection
P1: signal lifecycle, entitlement-critical state
P2: market/candle/indicator/structure/liquidity state
P3: referral/commission/payout state
P4: presentation-only aggregates/telemetry
```

Backpressure may coalesce P2 market visualization and P4 telemetry where safe. P0/P1 events must not be silently discarded. Commission/payout ledger truth remains durable even if the live UI event is delayed.

---

# 180. Command Center Data Model, API and WebSocket Requirements

## 180.1 Data model additions

Adapt names to the audited repository while preserving logical entities such as:

```text
dashboard_layout_preferences
dashboard_widget_preferences
market_state_snapshots
multitf_state_snapshots
indicator_state_snapshots
structure_events
liquidity_zones
liquidity_events
liquidity_score_versions
market_regime_events
session_state_events
signal_explanation_snapshots
execution_quality_events
position_state_events
trade_management_events
command_center_stream_cursors
command_center_client_telemetry_aggregates
referral_network_metric_snapshots
referral_activity_events
commission_dashboard_snapshots
plan_performance_snapshots
```

Do not duplicate canonical subscriptions, referral edges, commission ledger or payout ledger when those entities already exist. Dashboard tables are materialized/read models or aggregates only where justified.

## 180.2 User/read APIs

Extend existing `/api/v1` conventions with repository-appropriate routes equivalent to:

```text
GET /xauusd/command-center/snapshot
GET /xauusd/market-pulse
GET /xauusd/indicators/state
GET /xauusd/structure/state
GET /xauusd/liquidity/state
GET /xauusd/liquidity/heatmap
GET /xauusd/evidence-consensus
GET /xauusd/trade-readiness
GET /xauusd/execution/current
GET /xauusd/positions/current
GET /xauusd/events/timeline
GET /xauusd/sessions/state

GET /growth/summary
GET /growth/network
GET /growth/activity
GET /growth/commissions/summary
GET /growth/commissions/ledger
GET /growth/payouts/summary
GET /growth/plans/performance
```

Routes must enforce tenant/user ownership, plan entitlement, privacy and regulatory activation rules.

## 180.3 Admin APIs

Provide authorized admin equivalents for:

```text
market-command-center health
stream health/gaps/resync
indicator/structure/liquidity versions
execution-quality analytics
subscriber dashboard feature flags
commercial/referral metrics health
commission reconciliation health
payout reconciliation health
plan-performance aggregates
privacy masking policy
rank policy if enabled
```

## 180.4 WebSocket/SSE channels

Extend rather than duplicate existing streams. Logical channels may include:

```text
/ws/xauusd/market
/ws/xauusd/indicators
/ws/xauusd/structure
/ws/xauusd/liquidity
/ws/xauusd/signals
/ws/xauusd/execution
/ws/xauusd/positions
/ws/xauusd/timeline
/ws/growth/network
/ws/growth/commissions
/ws/growth/activity
```

A consolidated multiplexed authenticated stream is acceptable and may be preferable if the existing gateway architecture already supports channel subscriptions and priority routing.

## 180.5 Snapshot-first protocol

Every dashboard mode shall support:

```text
1. authenticated snapshot
2. snapshot_id + stream cursor
3. sequenced deltas
4. gap detection
5. replay or fresh resync
6. stale/offline state if recovery fails
```

The UI must not join stale aggregates to fresh market ticks as though they are synchronized.

---

# 181. Frontend UX, Rendering and Visual-System Requirements

## 181.1 Desktop-first institutional layout

The primary Command Center shall be optimized for professional desktop displays, including 1440p and 4K. It shall also degrade responsively to laptop/tablet and a simplified mobile layout.

Recommended information hierarchy:

```text
TOP: market header + execution state + health
LEFT: multi-timeframe market pulse / indicators
CENTER: live chart
RIGHT: signal / trade readiness / position
MIDDLE-WIDE: liquidity intelligence and evidence network
LOWER: execution stream / position management / event timeline
BOTTOM OR GROWTH MODE: referral network / commission flow / growth metrics
```

## 181.2 Full-screen Trading Room mode

Provide a one-click full-screen `TRADING_ROOM`/`COMMAND_CENTER` presentation that removes ordinary navigation and emphasizes continuous market, signal, execution, position and growth state for large monitors.

## 181.3 Motion design

Permitted event-driven motion includes:

```text
price pulse on authoritative tick
new-candle transition
liquidity-sweep flash
structure-change pulse
signal state transition
risk pass/fail transition
order/fill animation
TP/partial-close/BE/trail event
referral-registration pulse
commission-created flow
payout-state transition
```

Motion shall be subtle, performant, and disabled/reduced when the user selects reduced motion or the device cannot maintain target performance.

## 181.4 Visual semantics

Use a restrained institutional palette. Suggested semantic roles, subject to existing brand colors:

```text
gold: XAUUSD/primary market emphasis
positive/green: approved bullish/positive/paid states
negative/red: bearish/risk/rejected/loss states
blue/cyan: neutral analytics/data/AI state
amber/orange: liquidity/warning/macro/event risk
neutral graphite/obsidian: base surfaces
```

Color alone shall never be the sole carrier of meaning; icons/text/patterns are required for accessibility.

## 181.5 Performance

Preserve Section 152 targets. Additionally:

```text
avoid per-tick global React state
use Web Worker(s) for decode/sequence/aggregation where appropriate
use Canvas/WebGL for dense chart/heatmap/network rendering
virtualize large execution/referral/ledger tables
incrementally update graph nodes instead of full rerender
pause/minimize offscreen expensive animation
use requestAnimationFrame for visual batching
```

Growth-mode events may target human-visible near-real-time delivery rather than tick latency; their ledger truth and ordering are more important than animation speed.

## 181.6 Accessibility and responsive behavior

Mandatory:

```text
keyboard navigation
WCAG-appropriate contrast
screen-reader labels for critical states
reduced-motion support
non-color pass/fail indicators
responsive panel stacking
readable numeric formatting
locale/currency/timezone-aware display without changing stored truth
```

---

# 182. Security, Privacy, Compliance and Financial-Integrity Rules for Dashboard

## 182.1 Commercial ledger supremacy

Commission, payout and referral earnings displayed to subscribers must come from the authoritative commercial/ledger service. Frontend totals shall never become financial truth.

All balances must reconcile:

```text
opening balance
+ approved credits
- reversals/clawbacks
- payouts/debits
= closing balance
```

according to the canonical ledger semantics.

## 182.2 Privacy

Referral-network visualization shall default to privacy-preserving representations. Downline PII is not a visualization feature. Access to detailed user data is controlled by the existing RBAC/privacy/compliance rules.

## 182.3 Compliance activation

Section 138 and Section 160.4 remain authoritative. The UI may be built and tested before regulatory/payment-provider activation, but paid signals, assisted/auto execution and referral payouts must remain blocked wherever the canonical activation gates require it.

## 182.4 Claims and financial presentation

The dashboard shall distinguish:

```text
actual realized P&L
unrealized P&L
backtest performance
paper/shadow performance
forecast/projected commission
pending commission
available commission
paid commission
```

No forecast, backtest, model score or pending commission may be styled as guaranteed money.

## 182.5 Entitlements

Dashboard widgets and strategy-specific details shall respect subscription entitlements. A subscriber sees only the strategy products/features licensed to the account. Locked features may be advertised only in a non-misleading manner and must not leak restricted signal data through WebSocket or API payloads.

---

# 183. Observability, Testing and Acceptance for Live Command Center

## 183.1 Command-center metrics

Add or confirm metrics such as:

```text
command_center_snapshot_latency_ms
command_center_ws_connections
command_center_event_publish_latency_ms
command_center_sequence_gap_count
command_center_resync_count
command_center_stale_widget_count
command_center_render_fps aggregate
command_center_tick_to_paint_ms aggregate
indicator_state_age_ms
structure_state_age_ms
liquidity_state_age_ms
execution_event_delivery_latency_ms
position_state_age_ms
```

## 183.2 Growth metrics

```text
growth_snapshot_latency_ms
growth_event_delivery_latency_ms
referral_metric_refresh_age_ms
commission_event_lag_ms
commission_reconciliation_error_count
payout_reconciliation_error_count
network_graph_node_count aggregate
network_graph_render_latency_ms aggregate
```

Avoid high-cardinality user IDs as Prometheus labels.

## 183.3 Required tests

Add automated tests covering:

```text
snapshot + stream continuity
sequence gap/resync
stale/offline labeling
server-authoritative indicator parity
multi-timeframe profile mapping
liquidity provenance labels
inferred-vs-observed distinction
no fake liquidity/order-book claims
signal/SL/TP/exit parity
risk/execution state transitions
execution timestamp/latency attribution
position MFE/MAE/P&L parity
session/DST/broker-time display
entitlement leakage prevention
referral L1-L5 graph correctness
privacy masking
commission ledger reconciliation
payment-sequence commission-policy rendering
commission reversal/refund display
payout state transitions
forecast-vs-earned labeling
responsive layout
keyboard/reduced-motion accessibility
55 FPS desktop target under qualified load
WebSocket reconnect/resume
load/soak/chaos behavior
```

## 183.4 Acceptance criteria

The Live Intelligence Command Center is accepted only when all of the following are true:

1. Market, trading and growth views are implemented and use real backend data.
2. The central XAUUSD chart is server-authoritative and reconciles historical/live state.
3. Market Pulse shows strategy-specific multi-timeframe state and freshness.
4. Indicator cards display authoritative value/state/contribution, not a duplicate browser indicator engine.
5. Liquidity visualization labels observed/derived/inferred truth correctly.
6. Inferred spot/CFD liquidity is never marketed as a global order book.
7. Evidence consensus shows conflicts and mandatory vetoes honestly.
8. Signal card shows entry, invalidation, SL, TP ladder, net/gross R:R, calibrated target probability where valid, quality and TTL.
9. Trade-readiness state maps to the canonical gates/scoring profile.
10. Execution pipeline shows actual lifecycle events and measured latency/slippage/cost data.
11. Active position view reconciles broker/terminal/backend truth and exit events.
12. Market event timeline is generated from stored/replayable authoritative events.
13. Session display follows IANA/DST conversion and broker `UTC+03:00` requirements.
14. Growth dashboard reads the canonical five-level referral and commission systems.
15. Referral graph contains only real relationships and protects downline privacy.
16. Commission totals reconcile exactly to authoritative ledger entries.
17. First-payment, second-payment-L1-only and third-plus-payment policies are rendered from canonical policy/ledger outcomes rather than recoded in the frontend.
18. Pending, approved, available, paid, reversed and forecast values are visually distinct.
19. No fake market, execution, referral, commission or AI activity is used in live mode.
20. Replay/demo mode is unmistakably labeled and cannot submit live orders or financial mutations.
21. Entitlements are enforced server-side for every dashboard API and stream.
22. P0/P1 trading events are not silently dropped under backpressure.
23. Commercial events never block the Go trading critical path.
24. Snapshot/resume/resync works after network interruption.
25. The dashboard meets existing live-chart performance targets on the qualified desktop profile.
26. 4K full-screen Trading Room/Command Center mode is functional.
27. Accessibility/reduced-motion requirements pass.
28. Observability exposes market, stream, render, execution and growth health.
29. Security/privacy/compliance tests pass.
30. All new code preserves Sections 1–166 and introduces no regression to existing strategy, risk, execution, licensing, referral or accounting behavior.

---

# 184. Live Command Center Implementation, Deliverables and Final Objective

## 184.1 Implementation phases

The following implementation sequence is mandatory after repository audit and before final production promotion, integrated with the Section 117/142 plan:

```text
L0 — Audit existing dashboard, API, WebSocket, indicator, structure, liquidity,
     execution, position, referral and ledger paths. Mark REUSE/EXTEND/NEW.

L1 — Define canonical dashboard read models, provenance classes, event schemas,
     sequence/resume behavior and entitlement matrix.

L2 — Build server-authoritative Market Pulse, indicator state, chart overlays,
     structure/liquidity snapshots and event timeline.

L3 — Build Liquidity Intelligence heatmap/flow and Evidence Consensus network.

L4 — Build Signal Intelligence, Trade Readiness, execution timeline,
     execution-quality panel and live position management UI.

L5 — Build Growth Intelligence: five-level network, activity stream,
     commission flow, ledger summary, payout state and plan performance.

L6 — Build consolidated authenticated snapshot + multiplexed live event delivery,
     Web Worker/Canvas/WebGL rendering and full-screen Command Center mode.

L7 — Add privacy, entitlement, compliance, accessibility, observability,
     load/chaos/replay tests and performance optimization.

L8 — Run end-to-end acceptance using real/sandbox/replay fixtures with
     zero fabricated live data, perform security review and production rollout.
```

## 184.2 Required deliverables

The canonical deliverables include all of the following:

```text
Live XAUUSD Intelligence Command Center
Market / Trading / Growth / Command Center modes
Full-screen Trading Room layout
Market Pulse and multi-timeframe radar
Indicator Intelligence panels
Advanced server-authoritative XAUUSD chart overlays
Liquidity Intelligence map and heatmap
Evidence/AI consensus network
Signal Intelligence and Trade Readiness panel
Live execution pipeline and execution-quality panel
Live position/trade-management panel
Market event and session timeline
Five-level referral network visualization
Referral growth analytics
Commission flow and ledger dashboard
Payout dashboard
Plan-performance analytics
Snapshot/event schemas and API contracts
WebSocket/SSE resumable live delivery
Dashboard database/read-model migrations
Frontend performance/accessibility tests
Commercial ledger reconciliation tests
Security/privacy/entitlement tests
Operational dashboards and alerts
Updated architecture diagrams and runbooks
Updated user/admin documentation
Traceability matrix covering Sections 1–184
```

## 184.3 Final integrated objective

Predict-A-Trade shall present a subscriber-facing system where every important visual state is connected to real platform truth:

```text
OBSERVED MARKET DATA
+ DERIVED INDICATORS / STRUCTURE / LIQUIDITY
+ CALIBRATED MARKET INTELLIGENCE
+ HARD RISK / COST / ENTITLEMENT GATES
+ TRANSPARENT EXECUTION / POSITION TRUTH
+ SERVER-AUTHORITATIVE LIVE VISUALIZATION
+ FIVE-LEVEL REFERRAL NETWORK TRUTH
+ IMMUTABLE COMMISSION / PAYOUT LEDGER TRUTH
= LIVE XAUUSD + TRADING + GROWTH INTELLIGENCE COMMAND CENTER
```

The Command Center may look visually dramatic, but its credibility depends on the opposite of theatrical data: **every number, node, pulse, heat region, execution event, referral event and commission movement must be traceable to an authoritative source, model, policy or ledger entry.**

No dashboard feature introduced by Sections 167–184 may weaken the existing non-regression, scientific-integrity, risk, execution, financial-integrity, privacy, compliance, entitlement or fail-closed rules in Sections 1–166.



---


# Part IV — Visual System, Live Chart and User/Admin Dashboard Specification (v1.0.0 Baseline)

> **Integrated baseline authority:** Part IV is an inseparable part of the v1.0.0 canonical SOW. Sections 1–200 are simultaneously authoritative. Part IV defines presentation-plane requirements (design tokens, light/dark mode, navigation, live chart, user trading dashboard, admin operations dashboard, component library and testing) and — only where explicitly stated in Section 200.1 — refines presentation-layer details of Sections 31, 32, 33, 32A, 127, 152 and 167–184. No trading, risk, financial, licensing, entitlement, security or compliance requirement is weakened. Where any conflict exists between a presentation requirement and a non-presentation safety requirement, the non-regression authority in Section 2 governs.

---

# 185. Presentation Design Directive and References

## 185.1 Purpose

Define the complete visual system, live chart and dashboard layouts for the Predict-A-Trade presentation plane (Next.js) so that the platform presents:

- a **trader-focused, institutional trading-room dashboard** for subscribers (User application), following the referenced Instagram-reel trading dashboard composition as the primary dashboard type;
- a **non-trader operations console** for administrators (Admin application), with business/commercial/operational focus and **no trading-terminal detail**;
- a **CoinMarketCap/CoinGecko-style market-data experience** for public market pages and the live XAUUSD chart (price header card, timeframe toolbar, candlestick chart, stats strip, data tables with sparklines);
- full **Light and Dark mode** with a token-based color system (CSS custom properties + JS theme provider) seeded from the attached design image(s).

## 185.2 Design References

| # | Reference | Use | Availability |
|---|---|---|---|
| R1 | Attached design image(s) (`image.png`, 1919×943 desktop screenshot) | Color code (dark theme), light/dark mode, menu style — token extraction in Section 186 | Provided; palette extracted programmatically (Section 186.3) |
| R2 | Instagram reel `DcFFr2htpy8` (trading dashboard) | Primary dashboard composition for the User trading room (Section 192) | Not accessible to automated tooling (HTTP 403/login wall); requires human visual confirmation — see Section 200.4 |
| R3 | CoinMarketCap — coin page & derivatives market pages (`coinmarketcap.com/charts/...`) | Left-sidebar grouped navigation, price header card, timeframe toolbar, candlestick chart, stats strip, dark navy palette conventions | Public |
| R4 | CoinGecko (`coingecko.com`) | Market summary strip (KPI cards), data tables with sortable columns + sparklines, gainers/losers, "Customize" table controls | Public |

## 185.3 Scope and Non-Regression

1. Part IV covers the presentation plane only: design tokens, theming, typography, layout grid, navigation, global header/ticker, live chart, user dashboard, admin dashboard, command-center restyle, component library, data-visualization rules, accessibility/i18n, frontend performance, tests and acceptance.
2. **Nothing in Part IV permits fabricated data.** All charts, cards, tables and animations must render server-authoritative, provenance-labelled data per Sections 167–168 and 182. Demo/replay/sandbox data must carry the mandatory `DEMO`, `REPLAY` or `SANDBOX` labels.
3. **Entitlement is not a UI concern alone.** Every menu item, widget, chart timeframe and signal detail shown must also be enforced server-side (Section 38, 127.3). Hidden UI is never an authorization control.
4. The four-plane architecture (Section 3), the Go authoritative real-time path, the non-blocking gate architecture (Section 131) and the commercial/ledger truth (Section 69) are unaffected.

## 185.4 Two Distinct Audiences (mandatory separation)

| | USER application | ADMIN application |
|---|---|---|
| Audience | Subscriber / trader | Platform operator / staff (SUPER_ADMIN, ADMIN, RISK_MANAGER, TRADING_OPERATOR, SUPPORT, ANALYST, AUDITOR) |
| Primary focus | Live trading intelligence: price, chart, signals, risk state, positions, exit ladder, performance, referral growth | Business operations: users, subscriptions, billing, referrals, commissions, payouts, licenses, risk controls, market-data health, infrastructure, audit, reports |
| Trading detail level | Full trading-room detail (Section 192) | **Deliberately limited**: no live signal terminal, no open positions/P&L tables, no entry/SL/TP panels, no live candlestick chart with trade geometry. Trading surfaces are restricted to health, status, aggregates and controls (Section 193) |
| Menu style | Trader navigation (Section 189.3) | Operations navigation (Section 189.4) |

---

# 186. Design Token System (CSS/JS)

## 186.1 Single Source of Truth

1. Create `design-tokens.json` as the **single source of truth** for the entire visual system.
2. Generate from it, in CI, at minimum:
   - CSS custom properties: `:root` (base/always-on tokens), `[data-theme="light"]`, `[data-theme="dark"]`;
   - a JS theme provider module (React context) exporting typed tokens for inline/canvas/WebGL use (charts, heatmaps, network graphs, gradients);
   - optional Tailwind/utility-map theme extension if the project uses Tailwind.
3. **No component, chart or inline style may hard-code a color, font size or spacing value** that has a token. Enforce with a lint rule (stylelint/eslint) and a repo CI check that scans `hex|rgb\(|hsl\(` occurrences outside token definitions, golden fixtures and approved exceptions (e.g., data-driven series colors from broker profiles).

## 186.2 Token Namespaces

```text
--color-*      brand & semantic colors
--surface-*    background layers (base, raised, overlay, sidebar, header, popover)
--text-*       text roles (primary, secondary, muted, on-accent, on-status)
--border-*     border/divider roles
--accent-*     interactive accent (primary action, hover, active, focus ring)
--status-*     semantic states (up, down, warning, danger, info, neutral, disabled)
--chart-*      chart-specific colors (candle-up/down, volume, grid, crosshair, per-indicator)
--spacing-*    4px-based scale
--radius-*     shape scale
--shadow-*     elevation (light uses shadows; dark may use surface-lift + borders)
--motion-*     duration/easing tokens
--font-*       family/weights
```

## 186.3 Dark Palette (extracted from the attached image)

The following values were **programmatically extracted** from `image.png` (1919×943) — dominant/region colors:

| Token | Value (detected) | Status |
|---|---:|---|
| `--surface-base` (page background) | `#121B2B` | CONFIRMED (89% of image) |
| `--surface-raised` (main panel) | `#172030` | CONFIRMED |
| `--surface-elevated` (cards/right panel) | `#1C2637` | CONFIRMED |
| `--surface-sidebar` | `#282F3E` | CONFIRMED (menu region average `#272E3C`) |
| `--surface-hover` | `#354152` | CONFIRMED (palette) |
| `--border-subtle` | `#2B3547` | CONFIRMED (palette) |
| `--text-primary` | `#F9F9FA` | CONFIRMED |
| `--text-secondary` | `#B7BBC4` | CONFIRMED |
| `--text-muted` | `#9297A0` | CONFIRMED |
| `--accent-primary` | `#145CFA` | CONFIRMED (bright blue) |
| `--accent-hover` | `#4F98EC` | CONFIRMED |
| `--status-up` (buy/profit) | `#06D66F` | CONFIRMED |
| `--status-down` (sell/loss) | `#E86161` | CONFIRMED |
| `--status-warning` (amber/liquidity/news) | `#E9A265` | CONFIRMED |
| `--color-brand-gold` (XAUUSD emphasis) | `#E9A265` v1.0.0 baseline; future brand changes require a versioned token update | BASELINE_APPROVED |
| `--chart-grid` | derived from `--border-subtle` at ~35% opacity | DERIVED |
| `--chart-volume` | up/down at ~40% opacity | DERIVED |

If future brand assets or owner-approved screenshots introduce revised hex values, update the token sheet (Section 187.5) through a versioned visual-system change. v1.0.0 implementation is not blocked by future palette changes; token values remain configuration, not business logic.

## 186.4 Light Palette (v1.0.0 approved baseline)

The following values are the approved v1.0.0 light-mode baseline. Future brand refinements may replace token values through a versioned design-system update without changing semantic meaning:

| Token | Value (derived) | Status |
|---|---:|---|
| `--surface-base` | `#FFFFFF` | DERIVED |
| `--surface-raised` | `#F5F7FA` | DERIVED |
| `--surface-elevated` | `#FFFFFF` | DERIVED |
| `--surface-sidebar` | `#FFFFFF` (with `#E4E7EC` border) | DERIVED |
| `--surface-hover` | `#EDF0F5` | DERIVED |
| `--border-subtle` | `#E4E7EC` | DERIVED |
| `--text-primary` | `#0F1114` | DERIVED |
| `--text-secondary` | `#5B616E` | DERIVED |
| `--text-muted` | `#8A919E` | DERIVED |
| `--accent-primary` | `#145CFA` (same brand blue) | DERIVED |
| `--accent-hover` | `#4F98EC` | DERIVED |
| `--status-up` | `#16C784` (darker green for AA contrast on white) | DERIVED |
| `--status-down` | `#EA3943` (darker red for AA contrast on white) | DERIVED |
| `--status-warning` | `#B9761F` | DERIVED |
| `--color-brand-gold` | `#E9A265` v1.0.0 baseline | BASELINE_APPROVED |

Rules: same token namespace in both themes; identical layout in both themes; semantic meaning of status colors must be identical across themes (only the numeric value may differ).

## 186.5 Semantic Status Mapping (never color-only)

| Semantic token | Meaning (with text/icon companion) |
|---|---|
| `--status-up` | BUY signal, bullish bias, profit, positive change, paid/approved positive states |
| `--status-down` | SELL signal, bearish bias, loss, negative change, rejected/negative states |
| `--status-warning` | liquidity zones, news/macro risk, holds, pending review, degraded |
| `--status-danger` | hard-gate veto, NO-TRADE, risk limit, kill-switch state, error, fraud hold |
| `--status-info` | neutral analytics, AI/optional-model state, information |
| `--status-neutral` | disabled, unknown, unrated |

Every status-colored element must carry a text label, icon or pattern in addition to color (WCAG — Section 197).

## 186.6 Chart Palette (theme-aware)

| Element | Dark seed | Light seed |
|---|---|---|
| Candle up / area-up / volume-up | `#06D66F` | `#16C784` |
| Candle down / area-down / volume-down | `#E86161` | `#EA3943` |
| Grid lines | `--border-subtle` 35% | `--border-subtle` 60% |
| Crosshair line | `#9297A0` | `#5B616E` |
| Crosshair OHLC readout bg | `#1C2637` | `#FFFFFF` |
| VWAP | `#4F98EC` | `#145CFA` |
| EMA/SMA sets | versioned token set (e.g., `#E9A265`, `#06D66F`-alt, `#C792EA`) | same hues darkened |
| Liquidity pools BSL/SSL | amber/blue fills at 12–18% opacity | amber/blue at 10–15% |
| FVG zones | up/down colors at 10% fill | same at 8% |
| Order blocks | dashed outline in warning color | same |
| Sweep markers | warning color glyph | same |
| Session boxes | `--status-info` at 6% fill | same |
| Signal markers (BUY/SELL/NO-TRADE) | up/down/neutral glyphs with direction arrows | same |
| Entry/SL/TP lines | entry `--text-primary`, SL `--status-down`, TP `--status-up`, dashed | same |
| Scenario corridor (approved only) | gradient fill of primary color at 8% | same |

## 186.7 Contrast and Accessibility Compliance

1. Text tokens must meet WCAG 2.1 AA (4.5:1 normal, 3:1 large) in **both** themes; verify automatically in CI (axe-core + contrast audit job over token pairs).
2. Critical trading states (signal direction, NO-TRADE, TTL expiry, connection state) must meet AA for their text labels and use non-color indicators.
3. Focus rings use `--accent-primary` with a visible offset in both themes.
4. Any token change runs the contrast audit; failing tokens block the release.

## 186.8 Token Usage Rules

1. Theme tokens are never overridden by component-scoped magic values.
2. Canvas/WebGL charts read colors from the theme provider at render time; a theme change triggers a redraw from cached data (no refetch) — Section 187.4.
3. Broker-profile or data-driven colors (e.g., per-strategy accent) must be drawn from a controlled registry that resolves to tokens or approved data values, never ad-hoc literals.

---

# 187. Light & Dark Mode Architecture

## 187.1 Mechanism

1. Theming root: `<html data-theme="dark|light">`, persisted as a user preference, applied via CSS custom properties.
2. All components consume tokens only. Charts/heatmaps/networks receive the active theme via the JS theme provider.
3. Default theme for trading surfaces: **dark** (matches the attached design and institutional conventions). Public marketing pages: dark default with light option. Admin: user-preference with dark default.

## 187.2 Resolution Order

```text
1. Explicit user choice (stored, e.g., localStorage "pat-theme")
2. System preference (prefers-color-scheme) when user has not chosen
3. Brand/tenant default (config-driven; admin may set the default for unauthenticated surfaces)
```

A theme switcher (sun/moon toggle, Section 195) is available in the top bar on all public and authenticated surfaces. The choice must be applied consistently across all tabs of the same browser profile (storage event sync).

## 187.3 No-Flash / No-FOUC

1. An inline `<script>` in the document head resolves the theme from storage/system before first paint (executed before hydration).
2. SSR renders both-theme-safe markup (no theme-dependent server markup); only tokens switch client-side.
3. E2E test asserts: no theme flash on load, theme persists across reload, theme changes apply to all panels and the chart within one frame batch.

## 187.4 Theme-Aware Charts and Visualizations

1. Chart/heatmap/network canvases redraw from the **same cached data** on theme change; target: redraw completes < 100 ms on the qualified desktop profile.
2. No data refetch on theme change; no duplicate WebSocket subscription.
3. Legend, crosshair readout and tooltips are DOM overlays using tokens (not canvas-hardcoded colors) where feasible.

## 187.5 Token Confirmation Sheet (living document)

Maintain `docs/design/token-confirmation-sheet.md` listing every token with: token name, value, source (IMAGE-EXTRACTED / DERIVED / BRAND-ASSET / USER-CONFIRMED / V1_BASELINE), status (CONFIRMED / BASELINE_APPROVED / OVERRIDDEN_WITH_REASON), and date. The release gate (Section 199) requires every token used by shipped surfaces to have one of those resolved states; no unresolved visual token may remain at production rollout.

---

# 188. Typography, Numeric Formatting, Spacing and Layout Grid

## 188.1 Typography

1. Font stack: system-first with optional brand font, e.g. `Inter, -apple-system, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif`; monospace variant for raw tape/order-flow numbers (`JetBrains Mono, "Cascadia Code", Consolas, monospace`).
2. **All prices, amounts, scores and counts use `font-variant-numeric: tabular-nums`** so digits do not jump during live updates (critical for a trading UI).
3. Type scale tokens: 11 (micro/labels), 12 (table), 13 (body-s), 14 (body), 16 (h4/card title), 20 (h3), 24 (h2), 32 (page/price-hero). Weights: 400/500/600/700. Line heights per scale; 44px minimum touch targets for interactive controls on mobile.

## 188.2 Numeric and Money Formatting (presentation layer)

1. **Price decimals** come from the broker profile (Section 103A `digits`/`point`), never a hard-coded global; display convention labeled (bid/ask/mid) per Section 169.1.
2. **Money** (subscriptions, commissions, payouts, P&L in account currency): exact-decimal formatting, 2 decimals (USD seed), thousands separators, `$` prefix; signed values with explicit `+`/`−`.
3. **Percentages**: 1–2 decimals; change chips show sign, value and labeled baseline window (e.g., "24h", "session") — the baseline is never implied (Section 169.1).
4. **Timestamps**: always carry a timezone label; default broker time `UTC+03:00` with UTC in tooltip (Sections 150.7, 152.6); countdowns from server timestamps using monotonic client delta with a documented skew tolerance (seed ±10 s, config).
5. Locale-aware display never changes stored truth (Section 181.6); RTL-ready — Section 197.3.

## 188.3 Spacing, Radius, Borders, Elevation

1. Spacing scale (4 px base): `--spacing-1:4 … --spacing-12:48`; components use tokens only.
2. Radius: `--radius-sm:4px` (chips), `--radius-md:8px` (cards/inputs), `--radius-lg:12px` (panels/dialogs), `--radius-full` (pills/avatars).
3. Dark theme elevation uses **surface-lift + border** (base → raised → elevated), light theme uses border + subtle shadow; define `--shadow-1..4` per theme.
4. Density: default comfortable density for admin/ops; user trading surfaces may offer a compact-density toggle stored per account (config + preference).

## 188.4 Responsive Layout Grid

1. Base grid: 12 columns, 24 px gutters (desktop), fluid on mobile.
2. Breakpoints (min-width): `XS 375 · S 640 · M 768 · T 1024 · L 1280 · XL 1440 · XXL 1920 · 4K ≥2560`.
3. Dashboard panels define span rules per breakpoint; Section 192.8/193 define stacking order.
4. Full-screen Trading Room (Section 194.2) uses the full viewport with an optional 4K-friendly scale factor; no horizontal scroll on 1440p/4K in the primary layouts.

---

# 189. Navigation and Menu Style (CoinMarketCap-style)

## 189.1 Application Shell

```text
┌──────────────────────────────────────────────────────────────┐
│ TOP BAR: logo · search · global shortcuts · theme · notif · user │
├──────────────────────────────────────────────────────────────┤
│ GLOBAL TICKER STRIP (XAUUSD …) — collapsible (Section 190.1)   │
├──────────┬───────────────────────────────────────────────────┤
│ SIDEBAR  │  CONTENT AREA (page)                              │
│ (nav,    │                                                   │
│  240px   │                                                   │
│  → 64px) │                                                   │
│          │                                                   │
└──────────┴───────────────────────────────────────────────────┘
```

1. **Sidebar**: left, collapsible `240px → 64px` (icon-only with tooltips), grouped sections with uppercase section headers, icon + label rows, active state = accent-tinted background + 3 px accent left indicator, badge counts (e.g., payout queue, support tickets), collapsed tooltip behavior, keyboard navigation (`Alt+1..9` shortcuts configurable).
2. **Top bar**: logo/wordmark, global search (users, signals, tickets — RBAC-scoped), quick links, theme toggle, notifications bell (Section 68 categories), user menu (profile, settings, logout).
3. **Ticker strip**: collapsible strip below the top bar on user trading surfaces (Section 190.1); admin may render the compact variant (Section 190.3).
4. Mobile: sidebar becomes a drawer (hamburger) for admin; user app uses a bottom tab bar (5 primary destinations) + "More" sheet; ticker strip collapses to a single row.

## 189.2 Sidebar Design Details

- Section grouping mirrors the CoinMarketCap pattern (groups: e.g., "TRADING", "ACCOUNT", "GROWTH" for users; "OPERATIONS", "COMMERCE", "SYSTEM" for admin).
- Each item: `id`, `label`, `icon`, `route`, `permission` (RBAC), `badgeSource` (e.g., payouts-pending), `entitlement` (for user items).
- Server provides the permitted menu set per session (Section 189.6); the client renders only permitted items, but enforcement is server-side.

## 189.3 USER Application Menu (trader-focused)

```text
TRADING
  Overview                  (market header + chart + signal summary)
  Live Chart                (full chart page, Section 191)
  Signals                   (live signal terminal + history)
  Positions & Trades        (open positions, exit ladder, closed trades)
  Performance               (signal & account performance, Section 32)
  Market Pulse              (multi-timeframe radar, indicators, liquidity, news/macro)
ACCOUNT
  MT4 / MT5                 (agent download, setup wizard, devices, connections)
  Subscription & Plan       (plan, billing, invoices, strategy slots)
  License & Devices         (licenses, devices, MT accounts)
  Referral & Growth         (referral network, commissions, growth analytics)
  Payouts                   (balance, withdraw, history)
  Notifications & Alerts    (preferences, in-app log)
  Security                  (password, MFA, sessions, login history)
  Support                   (docs, FAQ, tickets)
```

## 189.4 ADMIN Application Menu (non-trader operations)

```text
OPERATIONS
  Dashboard                 (business KPIs + platform health, Section 193.2)
  Users                     (accounts, roles, suspensions, sponsors)
  Subscriptions & Billing   (subscriptions, invoices, payments, refunds, chargebacks)
  Plans & Pricing           (plans, setup/monthly prices, entitlements, effective dates)
  Referral Network          (tree/graph, attribution, suspicious clusters)
  Commission Control        (rules, ledger, holds/releases/reversals, adjustments)
  Payouts                   (requests, review, approve/process/reconcile)
LICENSING
  Licenses                  (create/assign/suspend/revoke/renew)
  Devices & MT Accounts     (device registry, revocations, MT bindings)
  Client Releases           (agent/EA releases, channels, checksums, rollback)
SYSTEM
  Strategies                (registry/versions/status/rollout — no trade detail)
  Risk Controls             (kill switches, exposure/spread limits, news blackout)
  Market Data Health        (feeds, latency, gaps, backfill — no trading detail)
  AI Providers              (models, health, latency, fallback)
  Infrastructure            (services, DB, Valkey, PgBouncer, queues, storage)
  Audit & Security          (audit log, security events, secrets health)
  Finance Reports           (sales, MRR, commissions, reconciliation)
  Support Queue             (tickets, diagnostics, internal notes)
```

**Admin exclusions (mandatory):** no "Live Chart" with trade geometry, no live signal terminal, no open positions/P&L list, no entry/SL/TP panels, no MT EA execution panels. Admin trading surfaces are limited to: strategy status registry, risk controls, market-data health, and aggregate signal/execution health metrics (e.g., NO-TRADE rate, delivery SLO) — never per-trade detail (Section 193.1).

## 189.5 Role-Based Rendering + Server Enforcement

1. The menu is rendered from a server-provided permission/entitlement list (no client-side role guessing).
2. Direct URL access, API calls and WebSocket channels enforce the same permissions (Sections 35, 38, 127.3, 180).
3. Nav items show a "locked" (non-misleading) state for features the account may see but lacks entitlement (Section 182.5) — no signal data leakage in locked state payloads.

---

# 190. Global Market Header and Ticker Strip

## 190.1 Global Ticker Strip (User trading surfaces)

Compact always-visible strip (height ~40 px, collapsible) containing, from Section 169.1:

```text
XAUUSD  bid  ask  (mid)  ▲/+0.42% (24h)  ·  Spread $0.21  ·  Regime: RANGE  ·
Session: LONDON (07:03 remaining)  ·  Feed: PRIMARY OK  ·  Broker 13:45:02 +03:00  ·
Data age 0.8s  ·  Connection: LIVE
```

- Price convention (bid/ask/mid) labeled; change baseline labeled; regime chip links to evidence (reason codes).
- States: `LIVE / DELAYED / RESYNCING / STALE / OFFLINE` chip (Section 159.3); DEMO/REPLAY watermark when applicable.
- On low bandwidth/mobile, the strip collapses to price + change + connection chip.

## 190.2 Price Header Card (CoinMarketCap-style, market/chart pages)

```text
┌───────────────────────────────────────────────────────────────┐
│ XAUUSD  Gold Spot (broker-normalized symbol per §103A)         │
│ $2,431.18  ▲ +0.42% (+$10.22)  [24h window — labeled]          │
│ High $2,442.10 · Low $2,420.35 · Spread $0.21 · ATR(M15) $3.1  │
│ Session range · Volume: BROKER_TICK_VOLUME_PROXY (labeled)     │
└───────────────────────────────────────────────────────────────┘
```

1. Big price uses `--font-size-32`, tabular-nums, weight 600.
2. Change chip: green/red per `--status-up/down`, sign, baseline label; never color-only.
3. Stats strip (High/Low/Spread/ATR/Session/Volume proxy) uses the Section 6A.2 proxy label — centralized volume is never implied.
4. Live update cadence: price text updated on tick-batch coalescing (Section 151.6), not per tick render.

## 190.3 ADMIN Header Variant (compact)

Admin top area shows: platform status chip (operational/degraded), feed health, market state label (e.g., RANGE/NEWS_RISK — from §169.2 vocabulary), broker clock, and system health (DB/Valkey/queues). **No bid/ask ticking, no signal direction, no trade detail.** Admin gets business KPIs, not market ticking.

---

# 191. Live Chart — CoinMarketCap/CoinGecko-Style Specification

## 191.1 Chart Page / Panel Composition

```text
┌──────────────────────────────────────────────────────────────┐
│ PRICE HEADER CARD (Section 190.2)                             │
├──────────────────────────────────────────────────────────────┤
│ TIMEFRAME TOOLBAR: [M1][M5][M15][M30][H1][H4][D1][W1][MN1]    │
│   + chart type toggle (Candles|Line|Area) + overlay groups     │
│   + fullscreen · export · LIVE/DELAYED chip                    │
├──────────────────────────────────────────────────────────────┤
│ CHART CARD                                                    │
│   candles/line/area  ·  crosshair OHLC readout                │
│   volume subchart   ·  overlays (Section 191.5)               │
│   status line: alignment profile, source, last event age      │
├──────────────────────────────────────────────────────────────┤
│ STATS STRIP: session H/L · PDH/PDL · spread · ATR · VWAP dist │
└──────────────────────────────────────────────────────────────┘
```

## 191.2 Timeframes and Entitlement

1. Timeframe toolbar from `M1, M5, M15, M30, H1, H4, D1, W1, MN1` (Section 8); tick/sub-minute views only for entitled strategies with authoritative data (Section 10A).
2. Timeframes shown/available per active strategy entitlement and plan (`analytics.basic` vs `analytics.advanced` may gate depth/history); locked timeframes show a non-misleading lock state (Section 182.5).
3. Timeframe switch = server snapshot + cursor re-anchor (Section 152.4); no client-side aggregation of candle truth.

## 191.3 Chart Types

1. **Candlestick (primary)** — server-authoritative candles with COMPLETE/PARTIAL/ESTIMATED/STALE/INVALID quality state visible on hover (Section 8).
2. **Line/Area toggle** — derived client-side from the same server candles for display only (harmless presentation calculation, Section 167.3); area gradient uses theme tokens.
3. **Volume subchart** — `BROKER_TICK_VOLUME_PROXY` labeled; GC flow/depth subcharts only when capability exists (Sections 6A, 32A.1) — otherwise hidden, never fabricated.
4. Optional capability-gated: footprint/delta/CVD/depth heatmap (Section 32A.1) with capability labels.

## 191.4 Crosshair and Readout

1. Crosshair with OHLC readout (precision per broker profile), volume, spread and timestamp (broker time + UTC tooltip).
2. Readout values use tabular-nums; theme-aware.
3. Linked crosshair/time across synchronized timeframes (Section 32A.1).

## 191.5 Overlays (server-authoritative)

1. All overlays come from server state (Sections 32A.1, 170–172): entry zone, SL, TP1/TP2/TP3, exit ladder, liquidity pools/sweeps, BOS/CHoCH/MSS, FVG/IFVG, OB/breaker, VWAP/bands, session boxes, prior day/week/month levels, midnight open, POC/VAH/VAL, signal markers, news/macro risk windows, approved scenario corridor.
2. Grouped toggles (STRUCTURE, LIQUIDITY, FVG_OB, VWAP, SESSIONS, INDICATORS, SIGNALS, TRADES, RISK, NEWS, PREDICTION — Section 171.3); default sets per strategy profile; user preference persisted per account.
3. Indicator overlays (EMA/SMA sets, VWAP, Bollinger, SuperTrend, etc. from Sections 132/170) render **server-computed values only**; the browser never computes production indicators (Sections 152.1, 170.2).

## 191.6 Data Contract and Status

1. Snapshot-first protocol per Section 152.4/180.5: REST snapshot → `snapshot_id` + cursor → sequenced deltas → gap detection → replay/resync.
2. Chart status chip: `LIVE / DELAYED / RESYNCING / STALE / OFFLINE / REPLAY` (Section 159.3) with last-update age; a static chart is never displayed as live.
3. Expired signals render only as historical/non-executable markers (Section 29).
4. Historical/live candle reconciliation per Section 152.7; mismatch → snapshot repair/resync, never a visual splice.

## 191.7 Interactions

1. Zoom/pan/scroll (mouse wheel, drag, touch pinch), keyboard arrows for time navigation.
2. Fullscreen mode (Section 181.2/194.2); ESC exits.
3. Audit-safe screenshot/export per Section 32A.5 (carries signal/config/version identifiers; no secrets, no downline data).
4. Chart settings persistence (overlays, type, density) per account.

## 191.8 Lite vs Pro Chart

`analytics.basic` (Lite): candles, core timeframes, volume, basic overlays (EMAs, VWAP, sessions). `analytics.advanced` (Pro): extended indicators, footprint/depth where capability-gated, replay access per Section 32A.4, extended history. Plan mapping is configuration-driven (Section 38).

## 191.9 Performance

1. Section 152.5 targets remain binding: receipt→paint p95 ≤150 ms qualified network, browser→paint p95 ≤50 ms, ≥55 FPS, reconnect/resume ≤2 s.
2. Web Worker decode/sequence check; Canvas/WebGL rendering; no per-tick React state; rAF batching; virtualized tables; capped history with LOD aggregation.
3. Theme change redraw ≤100 ms without refetch (Section 187.4).

## 191.10 Empty/Degraded/Replay States

1. No-data/failed-fetch: skeleton → "NO DATA — feed degraded" panel with retry; **never fabricated candles**.
2. Replay mode: `REPLAY` chip + timeline controls + simulated-vs-live disclaimer (Sections 32A.4, 167.4); replay cannot submit orders (Section 183.4 #20).

---

# 192. USER Dashboard — Trader-Focused Layout (Instagram-Reel style)

## 192.1 Concept

The User dashboard is a **trading room**, not a generic admin panel: large live chart center-stage, right-side signal/trade panel, live market intelligence surrounding it, and event-driven motion (price pulses, sweep flashes, signal transitions — Section 181.3). Reference composition: the Instagram-reel trading dashboard (R2) as primary dashboard type; CoinMarketCap-style market data conventions (R3) for charts/stats.

## 192.2 Main Grid (desktop, 12-col)

```text
┌───────────────────────────────────────────────────────────────┐
│ GLOBAL TICKER STRIP (§190.1)                                   │
├──────────┬────────────────────────────────────────┬───────────┤
│          │  LIVE CHART (§191)                      │  SIGNAL   │
│  MARKET  │  candles + overlays + volume            │  CARD     │
│  PULSE   │  (span 7-8)                             │  (§192.3) │
│  (§192.5)│                                        │  + TRADE  │
│  (span 2)│                                        │  PANEL    │
│          │                                        │  (§192.4) │
│          │                                        │  (span 3) │
├──────────┴────────────────────────────────────────┴───────────┤
│ MARKET PULSE ROW: multi-TF radar · indicators · liquidity · news │
├───────────────────────────────────────────────────────────────┤
│ POSITIONS & TAPES: open positions · execution timeline · event  │
│ tape (spans)                                                    │
└───────────────────────────────────────────────────────────────┘
```

1. **Center**: Live Chart (Section 191) — the visual anchor.
2. **Right panel (span 3)**: Signal Card on top (current/latest signal for the active strategy), Trade Management panel below (position, exit ladder, P&L, MFE/MAE, management actions when authorized).
3. **Left (span 2)**: compact Market Pulse / multi-timeframe radar (Section 192.5).
4. **Bottom**: positions table (virtualized), execution timeline (Section 175.1), market event tape (Section 177.1), session panel.
5. **Mode switch** (top of content): `MARKET | TRADING | GROWTH | COMMAND_CENTER` (Section 167.2) — MARKET hides trade panel; TRADING is the default full layout; GROWTH swaps bottom panels for referral/commission panels (Section 178); COMMAND_CENTER = full-screen 4K (Section 194).

## 192.3 Signal Card (right panel, from Sections 159.1/174.1)

```text
┌───────────────────────────────┐
│ STRATEGY: STANDARD_SCALPING   │  GRADE A  [sample ok]
│ BUY  ▲   p(TP1<SL)=0.62 (cal) │
│ TTL 09:41  · created/expires   │
│ Entry 2430.10 (zone)          │
│ SL 2426.80  TP1 2433.40 (50%) │
│ TP2 2435.00 (50%)             │
│ Gross R:R 1.42 · Net est 1.21 │
│ Cost est $0.42  spread $0.21  │
│ Invalidation: < 2425.50 close │
│ [Evidence] [Risk decision]    │
└───────────────────────────────┘
```

1. Direction banner (BUY/SELL/NO-TRADE/WAIT) with arrow + color + icon (non-color indicator).
2. Calibrated probability shown **only** with exact target/horizon label and sample-sufficiency state (Sections 15A, 130.3–130.5); otherwise `UNRATED`.
3. TTL countdown from server timestamps (monotonic client delta); expiry state visible.
4. Exit ladder percentages (TP fractions), gross vs net R:R distinctly labeled (Section 136.3); cost estimate and current spread.
5. Evidence expander: positive/negative evidence, pillar contributions, reason codes, AI verification, risk decision (Sections 32, 174.1).
6. NO-TRADE card: reason chip(s) + gate breakdown (Section 174).

## 192.4 Trade Management Panel

1. Open position: side, strategy, lots (permitted), entry, current bid/ask, unrealized/realized P&L, current R, SL/TP ladder, BE/trailing state, time in trade, MFE/MAE (Section 176.1).
2. Target progress meters computed from actual geometry and authoritative price (Section 176.2).
3. Management actions (close partial/full, move BE, trail) **rendered only when the account holds the corresponding execution permission** (Section 27A); actions call server endpoints with idempotency keys (Section 50); confirmation dialogs; audit.
4. Execution-quality summary per Section 175.2 when entitled.
5. Signal-only accounts see a read-only panel with the SIGNAL ONLY badge (Section 46).

## 192.5 Market Pulse / Indicator Cards (Section 170)

1. Multi-timeframe radar rows (M1…D1/W1 per strategy profile): direction, strength, momentum, structure, liquidity, volatility, VWAP location, data quality, freshness — context/setup/entry timeframes visually distinguished (Section 170.1).
2. Indicator cards = value + interpretation + direction + strength + contribution + freshness (Section 170.2) — server values only.
3. Consensus chip (STRONG_BULLISH…CONFLICTED/INSUFFICIENT_DATA) traceable to the confluence profile; never presented as probability unless calibrated (Section 170.3).

## 192.6 Quick Views (tabs)

`Overview | Chart | Signals | History | Performance | Positions | Growth`

- Overview = Section 192.2 composition; Chart = full-page chart (Section 191); Signals = live terminal + filterable history (Section 32); History = signal history with filters (date, direction, grade, strategy, TF, outcome, session, regime) + entitlement/history retention respected (Section 32); Performance = signal vs account performance, TP1/2/3 rates, latency metrics (Section 32); Positions = position/trade list + closed-trade outcomes; Growth = referral/commission/payout dashboard (Section 178).
- Tables follow CoinGecko style: sortable columns, filters, sparklines (7-day signal flow, equity), virtualized rows, exact-decimal money (Section 188.2).

## 192.7 Responsive Stacking (user)

```text
Desktop ≥1280:  sidebar + ticker + 12-col grid (192.2)
Tablet 768–1279: chart full width; right panel stacks below chart;
                 market pulse becomes horizontal scroll row
Mobile <768:     bottom tab bar; order: ticker(1 row) → price card →
                 chart → signal card → positions → market pulse;
                 trade panel via tab; full-screen chart gesture
```

## 192.8 Event-Driven Motion (reel-style, but honest)

1. Permitted motion strictly from real events (Section 181.3): price pulse on authoritative tick, new-candle transition, sweep flash, structure-change pulse, signal state transition, risk pass/fail, order/fill animation, TP/partial-close/BE/trail events, referral-registration pulse, commission flow, payout transition.
2. No synthetic motion when markets are quiet (Section 167.4); reduced-motion support (Section 197).
3. Motion tokens: durations 120–400 ms, easing curves per token set; performance guard — disable when device cannot hold ≥55 FPS or reduced-motion is set.

---

# 193. ADMIN Dashboard — Non-Trader Operations Layout

## 193.1 Design Principle

The Admin application is an **operations and business console**. Administrators are operators, not traders: they manage users, commerce, licensing, risk controls and platform health. Therefore the admin UI must **not** include: live signal terminal, open positions/P&L tables, entry/SL/TP/exit-ladder panels, live candlestick chart with trade geometry, MT EA execution panels, or per-trade execution detail. Trading information shown to admins is limited to **health, status, aggregates and controls** (e.g., NO-TRADE rate trend, gate health, delivery SLO, strategy status registry, feed health, risk-control switches). This is also enforced server-side: admin APIs never return trading-level detail not required for operations (Section 193.6).

## 193.2 Overview Page (operations dashboard)

```text
┌──────────────────────────────────────────────────────────────┐
│ STATUS STRIP: platform · feed · DB · Valkey · queues · agents  │
├──────────────┬──────────────┬──────────────┬─────────────────┤
│ MRR $xx      │ Active Subs  │ Churn %      │ Pending Comm    │
│ (+Δ vs prior)│ (Δ)          │ (Δ)          │ Available Comm  │
│ Setup Fees   │ New Subs 7d  │ Payouts Pending │ Fraud Holds  │
├──────────────┴──────────────┴──────────────┴─────────────────┤
│ CHARTS (business analytics, Section 193.4):                  │
│  Revenue by plan (bar) · Commission by level (bar) ·         │
│  Subscriber growth (area) · Churn trend (line) ·             │
│  Referral funnel (funnel) · Payout volume (bar)              │
├──────────────────────────────────────────────────────────────┤
│ ALERT FEED: risk/infra/commercial alerts (Section 68A) with  │
│  source, freshness, affected entity, remediation status      │
└──────────────────────────────────────────────────────────────┘
```

1. KPI cards: label, value, delta vs prior period, sparkline; refresh via business metrics API (Section 88/69.30), not dashboard counters — metrics truth from reproducible analytics (Section 87A).
2. Alert feed with severity colors + text icons (never color-only), acknowledgement, remediation links.
3. No live price ticking required; admin pages may use periodic refresh (seed 30–60 s) with manual refresh.

## 193.3 Module Page Layouts (summary)

| Module | Layout |
|---|---|
| Users | Filterable table (search, status, plan, sponsor) + detail drawer (profile, subscription, licenses, devices, sponsor chain, commission history, activity) + actions (suspend/reactivate, reset MFA, end sessions, role change) with reason dialogs |
| Subscriptions & Billing | Subscription list + detail (billing period, invoices, payments, refunds/chargebacks, provider refs) + payment truth read-only (Section 33) |
| Plans & Pricing | Plan cards/table with effective-dated price & entitlement editing, version history, dry-run preview (Section 66A) |
| Referral Network | Tree/graph visualization with pagination + depth guards (Section 33); sponsor-change flow with MFA + reason; suspicious-cluster flags |
| Commission Control | Rules table (effective-dated, snapshots), ledger explorer (filters by user/plan/level/cycle), hold/release/reverse/adjust actions with reason + audit |
| Payouts | Queue with status workflow (REQUESTED→UNDER_REVIEW→APPROVED→PROCESSING→PAID/FAILED/CANCELLED), risk review panel, MFA step-up for approvals, provider reference entry, reconciliation export |
| Licenses / Devices / MT | Registry tables + actions (create/suspend/revoke/reset/force logout), activation history |
| Strategies | Registry/status table (version, state, rollout %, flags) — **no strategy trade detail**; activation/rollback with approval workflow |
| Risk Controls | Kill-switch panel (global/strategy/symbol/broker/account/device), exposure & spread/slippage limits, news blackout config — destructive actions require re-auth (Section 33) |
| Market Data Health | Feed cards (primary/secondary, divergence, latency, tick rate, gaps, backfill status) — no trade detail |
| Releases / Audit / Infrastructure / Reports | Standard management layouts per Section 33 |

## 193.4 Charts on Admin (analytics mode only)

1. Chart types limited to business analytics: bar, line, area, funnel, heatmap (e.g., commission by plan × month), table+sparkline. **No candlestick component is used on admin pages.**
2. Reuse the shared chart library in analytics mode (same performance rules); data from business/analytics APIs (Sections 88, 69.30), aggregated — never per-user PII in aggregate charts (Section 175.3/182.2).
3. All admin charts carry their source label, refresh time and, where applicable, report/evidence references (Sections 130.6, 87A).

## 193.5 Approval and Destructive-Action UX

1. Consequential actions (payout approval, sponsor change, commission reversal, kill switch, license revoke, price change) require: confirm dialog with summary, mandatory reason field, MFA/step-up where policy requires (Sections 33, 65, 69.32), and post-action audit toast/link.
2. Every mutation endpoint is idempotency-keyed where appropriate (Section 80).

## 193.6 Server-Side Enforcement

1. Admin APIs enforce RBAC per role (Section 35) and never return trading-level or PII payloads beyond the caller's permission (defense in depth with RLS where used, Section 36).
2. The UI hides nothing by client logic alone; E2E tests verify that direct URL/API access to excluded admin surfaces is denied (Section 199).

---

# 194. Command Center Modes under the New Visual System

## 194.1 Mode Mapping

| Mode (Section 167.2) | Composition under Part IV |
|---|---|
| `MARKET` | Ticker + Market Pulse (left) + Live Chart (center) + evidence/liquidity panels; signal card minimal |
| `TRADING` | Default user layout (Section 192.2): chart + signal card + trade panel + positions/timeline |
| `GROWTH` | Ticker + Growth panels: referral network graph, commission flow, ledger summary, payout state, plan performance (Section 178), activity stream |
| `COMMAND_CENTER` | Full-screen 4K composite: Market Pulse + Live Chart + Signal/Trade panel + Liquidity/Evidence network + Event tape + Growth mini-panels (referral count, commission available) — per Section 181.2 |

Mode switch is persisted per account; each mode is a distinct route with its own server snapshot + stream subscriptions (Section 180.5).

## 194.2 Full-Screen Trading Room

1. One-click entry (`F11`-equivalent button / menu item), hides sidebar/top bar/nav; ESC exits.
2. 4K-friendly: optional density scale, no horizontal scroll, panel reflow for ≥2560 px.
3. Preserves all server-authoritative rules, entitlements and status chips; not a separate data path (same API/WS channels).

## 194.3 Theme-Aware Advanced Visuals

1. **Liquidity heatmap** (Section 172.3): vertical price heatmap, intensity from approved liquidity scores/real depth; legend `INFERRED LIQUIDITY` vs `ORDER BOOK` (never implied); theme tokens; hover explains zone evidence.
2. **Evidence consensus network** (Section 173): nodes = evidence groups with state/direction/quality/reason; conflict shown visibly (Section 173.4); optional quantum node per Sections 144–160 presentation rules (plain-language labels only).
3. **Referral network graph** (Section 178.4): real relationships only; collapse/expand, filters, privacy masking (no downline PII — Section 182.2); commission-event links.
4. Flow/particle animations only from real state changes (Sections 172.4, 178.6); static map when no valid flow measure exists.

---

# 195. Component Library and Reusable UI Inventory

All components are theme-token-driven, server-data-connected, with defined empty/degraded/error/permission states and RTL support. Minimum inventory:

**Layout & shell**
`AppShell` (sidebar+topbar+ticker+content), `Sidebar` (collapsible, grouped, badges), `TopBar`, `TickerStrip` (§190.1), `PageHeader`, `Breadcrumbs`, `ModeSwitcher` (MARKET/TRADING/GROWTH/COMMAND_CENTER), `FullscreenRoom`.

**Data display**
`KPIStat` (label/value/delta/sparkline), `StatStrip`, `PriceChip`, `ChangeChip` (labeled baseline), `Badge`, `StatusChip` (LIVE/DELAYED/…, grade, regime), `DataTable` (virtualized, sortable, filterable, CoinGecko-style), `Sparkline`, `ProgressBar`, `Gauge` (trade-readiness, §174.2), `Tabs`, `Timeline`, `TreeGraph` (referral), `Heatmap` (liquidity), `NetworkGraph` (evidence), `Funnel` (referral), `AreaLineBar` (analytics charts).

**Trading components (user app only)**
`PriceTicker`, `SignalCard` (§192.3), `TradeReadinessMeter` (§174.2), `OrderPanel`/`ExitLadder` (§192.4), `PositionRow`, `TTLCountdown`, `ExecutionTimeline` (§175.1), `MarketPulseRadar` (§192.5), `IndicatorCard` (§170.2), `LiquidityPanel`, `EventTape` (§177.1), `SessionPanel`, `ReplayBar` (REPLAY mode).

**Chart**
`CandleChart` (Canvas/WebGL, §191), `ChartToolbar`, `OverlayGroupToggles`, `CrosshairReadout`, `ChartStatusChip`.

**Commercial (user + admin)**
`ReferralTree` (§178.4), `CommissionFlowDiagram` (§178.6), `LedgerTable`, `PayoutStatus`, `PlanPerformance`, `CommissionSummary`.

**Feedback & states**
`Toast`, `Skeleton`, `EmptyState`, `DegradedState`, `ErrorState`, `PermissionDeniedState`, `LockedFeatureCard`, `DemoWatermark`, `ConfirmDialog` (reason+step-up), `Drawer`, `Dialog`, `CommandPalette` (global search), `ThemeToggle`.

**Controls**
`SearchBox`, `FilterBar`, `Toggle`, `SegmentedControl`, `Dropdown`, `DateRangePicker` (timezone-aware), `CopyButton`, `Pagination`.

Each component ships with: token usage (no literals), API source mapping (REST/WS/db — Section 127.4), accessibility behavior, empty/degraded states, and unit/E2E tests.

---

# 196. Data Visualization Rules (no fake data)

1. Every visual object renders server-authoritative data with provenance class where applicable (Section 168.1); the UI adds only formatting, geometry and viewport transforms (Section 167.3).
2. Charts never contain fabricated candles/ticks/liquidity/order flow; capability-gated features are hidden or labeled `NOT CALIBRATED`/capability-missing (Sections 172.2, 6A).
3. Demo/replay/sandbox visuals carry the mandatory `DEMO`/`REPLAY`/`SANDBOX` watermark and cannot mutate live state (Section 167.4, 183.4 #20).
4. Animations are event-driven only; static when no events; reduced-motion honored (Section 197).
5. Forecasts/projections are labeled `FORECAST` with model/assumptions/date range and are never styled as balances (Section 178.8, 182.4).
6. Color is never the sole carrier of meaning (icons, text, patterns required) (Sections 181.4, 186.5).

---

# 197. Accessibility, i18n, RTL, Responsive

## 197.1 Accessibility (extends Section 181.6)

1. Full keyboard navigation for all dashboard flows; visible focus (Section 186.7); skip-to-content links.
2. Screen-reader labels for critical states: price changes, signal direction, TTL, connection state, NO-TRADE reasons, alerts; live regions for signal/execution announcements (aria-live polite, critical states assertive).
3. Reduced-motion: `prefers-reduced-motion` disables all non-essential animation (Section 192.8).
4. Contrast per Section 186.7; non-color indicators per Section 186.5.
5. WCAG-oriented: target AA, Level AA for interactive components; automated axe checks in CI + quarterly manual audit.

## 197.2 i18n and Localization

1. Architecture supports locale packs (en seed; Arabic (ar) as the primary secondary target given UAE operations — Section 138.6).
2. Locale-aware: numbers, dates, times, currencies (display only — stored truth unchanged), pluralization, date ranges.
3. Timezone display per Section 150.7 (broker time default + UTC + user-local options), never affecting TTL/UTC truth.

## 197.3 RTL Readiness

1. Layouts use logical CSS properties (margin-inline-start, inset-inline-end, etc.) so the sidebar, charts axes and timelines mirror correctly under `dir="rtl"`.
2. Charts: time axis direction handling defined for RTL (axis order flips; crosshair and readouts remain correct).
3. All components tested in RTL mode with at least the Arabic locale fixture.

## 197.4 Responsive Behavior

1. Breakpoints and stacking per Sections 188.4, 192.7 (user), 193 (admin).
2. Panel priority: trading surfaces (chart, signal, positions) above all else on small screens for users; admin keeps tables/actions accessible, never sacrificing action affordances.
3. Touch targets ≥44 px on mobile; horizontal scroll only inside data tables (never the page shell).

---

# 198. Frontend Performance and Rendering Rules

1. **Worker-first streaming**: Web Worker decode/sequence/aggregation (Sections 152.5, 181.5); main thread handles input and paint only.
2. **Canvas/WebGL** for dense chart/heatmap/network; virtualized tables; rAF batching; offscreen/background tabs pause heavy animation.
3. **No per-tick React state**; price updates via refs/imperative handles; signals/risk events bypass coalescing (Section 151.6).
4. **Bundle budget**: per-route budget (seed: user dashboard main route ≤ 400 KB gz), dynamic import for chart/network heavy modules, code-split by route; CI enforces budgets.
5. **Image/media**: next/image optimization; no unoptimized screenshots in production surfaces; chart export uses canvas rendering (not DOM screenshot).
6. **Caching**: SWR/React Query-style caching for REST snapshots; WS deltas immutable; theme switch does not refetch (Section 187.4).
7. Targets (extend Section 152.5): page TTI ≤ 3 s on qualified desktop (seeded), table render (10k rows virtualized) ≤ 200 ms, theme switch ≤ 100 ms, command center ≥ 55 FPS.
8. Monitoring: web-vitals + custom chart/stream metrics per Section 183.1; alert on sustained regression.

---

# 199. Testing, Acceptance Criteria and Deliverables for the Visual System

## 199.1 Required Tests

1. **Token/theme tests**: token parity between `design-tokens.json`, CSS and JS provider (CI); contrast audit both themes; no hard-coded color lint; theme-switch E2E (no flash, persistence, tab sync, chart redraw ≤100 ms without refetch).
2. **Layout tests**: snapshot/visual regression at all breakpoints (XS…4K); no horizontal scroll in primary layouts; admin/user nav separation; sidebar collapse; bottom-tab behavior on mobile.
3. **Menu/RBAC tests**: server-provided menu sets; direct URL access to admin-excluded and user-unauthorized routes denied; locked-feature states contain no leaked payload data.
4. **Chart tests**: timeframe switch (snapshot+cursor), overlay toggles, crosshair readout precision, zoom/pan/fullscreen, resync/gap/replay labeling, LIVE/DELAYED/… chips, export redaction, capability-gated subcharts hidden without capability, no fabricated candles in degraded state.
5. **User dashboard tests**: signal card fields per §192.3, exit ladder percentages, TTL countdown (server-time delta), trade-readiness mapping to gates, position panel parity (MFE/MAE/P&L), execution timeline, event tape from stored events, growth panels per Section 178.
6. **Admin dashboard tests**: KPI values from business APIs (reconciliation to reports), payout approval workflow with MFA/reason, commission ledger explorer, kill-switch flow with re-auth, no trading-terminal components present in admin bundle routes (static check), admin market-data health only.
7. **E2E journeys added** (extend Section 127.5): register→theme choose→subscribe→see entitled chart/timeframes; unentitled strategy locked; admin approve payout; admin trigger kill switch; user chart reconnect/resume; replay mode watermark; mobile stacking.
8. **Performance/a11y**: §198 targets; axe CI; keyboard walkthrough tests; reduced-motion tests.
9. **Chaos/soak**: chart stream gap storms, slow clients, theme toggle under load, 4K command center at ≥55 FPS.

## 199.2 Acceptance Criteria (additions; PASS/FAIL with evidence)

1. Both themes render all surfaces without layout breakage or contrast failure.
2. Token sheet is CONFIRMED (or waived with documented reason) before production rollout.
3. User app is a trader-focused dashboard per Section 192; admin app contains no trading-terminal detail per Section 193.
4. Live chart is server-authoritative, sequence-safe and honors capability/entitlement gates.
5. All dashboard numbers trace to APIs/reports; no fake data; demo/replay labeled.
6. Navigation is server-permission-driven; excluded surfaces denied server-side.
7. Performance targets (§198) pass on the qualified desktop profile.
8. Accessibility/i18n/RTL requirements (§197) pass.
9. No regression in Sections 1–184 trading, financial or security behavior (Section 200.1).

## 199.3 Deliverables

```text
design-tokens.json (+ generated CSS/JS/Tailwind maps)
Theme provider + no-flash bootstrap
Component library (Section 195 inventory) with tests
Live chart implementation (Section 191)
User trading dashboard (Section 192)
Admin operations dashboard (Section 193)
Command Center modes + full-screen room (Section 194)
Token confirmation sheet (Section 187.5)
Style guide / visual system documentation
Accessibility & i18n/RTL test reports
Visual regression suite + performance report
Updated traceability matrix covering Sections 1–200
```

---

# 200. Final Integration Register, Versioning and v1.0.0 Baseline Decisions

## 200.1 Precise Presentation-Layer Integration Map

| Section | Effect of Part IV |
|---|---|
| §31 (Next.js architecture) | Extended: design system and three experiences restyled per §§186–193 |
| §32 (User Dashboard) | Layout and components per §192; all §32 functional content remains authoritative |
| §32A (Chart/Replay) | Chart presentation per §191; functional requirements (overlays, states, replay, export) unchanged |
| §33 (Admin Dashboard) | Layout/menu per §193; all §33 functional modules remain; trading-detail presentation restricted per §193.1 |
| §34–35, §38 | Unchanged authority; navigation consumes permissions/entitlements per §189.5 |
| §127.2–127.4 | Extended by §§191–199 component/surface list and API-to-UI traceability |
| §152 (Live chart streaming) | Data contract unchanged; visual composition per §191 |
| §167–184 (Command Center) | Visual system/layouts/modes per §§186–195; functional requirements unchanged |
| §181 (UX/visual semantics) | §181.4 palette implemented through §186 tokens; §181.3/181.5/181.6 extended by §§192.8/198/197 |
| §183 (acceptance) | Extended by §199.2 acceptance additions |
| §138 (UAE/compliance) | Unchanged; Arabic/RTL readiness per §197.2–197.3 |
| §69 (commercial) | Unchanged; money formatting per §188.2 |

## 200.2 Canonical Versioning

1. This document is the **Predict-A-Trade v1.0.0 canonical production baseline** covering Sections 1–200.
2. Historical internal labels from earlier drafting passes are non-authoritative and have been normalized to **v1.0.0**.
3. The final build report traceability matrix must cover Sections 1–200 plus the appendices in this file.
4. Any future functional change requires a new semantic document version and must preserve non-regression authority unless the owner explicitly approves a controlled breaking revision.

## 200.3 External Visual References

1. External inspiration links/images are references only. Codex does **not** require access to any external social-media reel to implement v1.0.0 because the required layout, behavior, components, truth rules and acceptance criteria are fully specified in Sections 167–199.
2. CoinMarketCap/CoinGecko-style references define interaction/layout conventions only; no third-party branding, copyrighted assets or copied proprietary code shall be embedded.
3. Future screenshots or brand assets may refine design tokens through a versioned design-system update but do not block the v1.0.0 build.

## 200.4 v1.0.0 Visual Baseline Decision Sheet

| # | Item | v1.0.0 Decision |
|---|---|---|
| 1 | Dark palette tokens (§186.3) | APPROVED BASELINE |
| 2 | Light palette (§186.4) | APPROVED BASELINE |
| 3 | Brand gold accent | `#E9A265` APPROVED BASELINE; configuration-backed |
| 4 | Menu style | Left sidebar + top bar + ticker strip — APPROVED BASELINE |
| 5 | User trading-room composition (§192) | APPROVED BASELINE from the canonical SOW |
| 6 | Admin no-trading-detail boundary (§193) | MANDATORY |
| 7 | Default theme | Dark for authenticated trading surfaces; user choice persists |
| 8 | Fonts | System-first sans stack + approved monospace fallback; optional future brand font |

No unresolved visual-design decision remains in the v1.0.0 implementation contract. Future visual refinements are normal versioned product changes and are not blockers to implementation, test or production-readiness evidence.

---

# Canonical Codex Build Authority

Codex shall treat this entire file as one indivisible implementation contract. It must audit the existing repository first, preserve working behavior, create a requirement-to-code traceability matrix, and implement the complete backend, frontend, realtime trading plane, research plane, data layer, Windows/MT4/MT5 integration, licensing, billing, referral/commission/payout system, Live Intelligence Command Center, **Visual System, Live Chart, User trading dashboard and Admin operations console (Part IV, Sections 185–200)**, tests, observability, security controls, documentation and release gates required by Sections 1–200. All requirements in Part IV are mandatory, including the design-token system, light/dark mode, navigation, live chart, user/admin dashboard layouts, component library and their acceptance criteria, unless an item is explicitly marked optional or is blocked by an external legal/provider approval that cannot be resolved in code. No unresolved visual-design confirmation remains in v1.0.0.

Codex must not stop because code compiles, a UI renders, tests partially pass, or a feature is stubbed. Completion requires evidence against the acceptance criteria and definition-of-production-ready requirements in this SOW. Production-mutating operations remain subject to the explicit safety/approval rules in this document.

---

# Appendix A — v1.0.0 Production Implementation Profile

This appendix makes the deployment and repository-execution assumptions explicit so Codex does not introduce unnecessary infrastructure or a competing greenfield platform.

## A.1 Existing-Repository-First Rule

The implementation target is the existing Predict-A-Trade repository. Codex must:

```text
AUDIT FIRST
PRESERVE WORKING BEHAVIOR
REUSE BEFORE REPLACE
MIGRATE COMPATIBLY
TEST BEFORE PROMOTION
NEVER CLAIM COMPLETE WITHOUT EVIDENCE
```

Repository-discovered commands, paths and service names take precedence over invented names. If current repository structure conflicts with a proposed example path in this SOW, adapt the implementation to the actual repository while preserving the logical boundary and acceptance requirement.

## A.2 Production Host Baseline

v1.0.0 production operations shall target:

```text
Ubuntu 24.04 LTS
Nginx reverse proxy / TLS edge
systemd-supervised production application services
PostgreSQL 17 + TimescaleDB + pgvector + PgBouncer
Valkey
Go real-time engine
NestJS control plane
Next.js presentation plane
Python research / AI workers
Go Windows Agent + MQL4/MQL5 terminal components
```

Containerization may be used for local development, CI, integration testing, disposable verification and staging where useful. It must not force a Kubernetes migration or replace a working host/systemd production deployment merely for architectural fashion. Kubernetes is outside v1.0.0 unless explicitly approved in a later scope revision.

## A.3 Environment Separation

Maintain explicit:

```text
development
test
staging
paper
shadow
production
```

boundaries with separate credentials, signing material, provider modes, financial/test data and broker permissions. Test/staging must not share live execution credentials or mutable production financial credentials.

## A.4 Infrastructure Reproducibility

Codex shall document and version all reproducible host/service configuration it legitimately owns, including:

```text
service units
Nginx application config
environment-variable contracts
database migrations
backup/restore procedures
monitoring/alert configuration
release manifests
Windows client release manifests
checksums/signature metadata
runbooks
```

Secrets themselves must never be committed.

## A.5 Production Mutation Boundary

Implementation may create code, migrations, test fixtures, documentation and deployment artifacts inside the repository. It must not autonomously:

```text
enable live automated trading
place live broker orders
run destructive production migrations
change production DNS
rotate production signing keys
alter real subscription/commission/payout balances
publish unsupported trading-performance claims
```

Those actions remain explicit operator-controlled release steps.

---

# Appendix B — Codex CLI Execution Contract

## B.1 Required One-Command Behavior

The Codex run must consume this entire file as the implementation contract and execute against the existing repository. It must:

1. read root and nested `AGENTS.md` instructions;
2. audit the repository before changes;
3. build a requirement-to-code/test/migration/observability traceability matrix;
4. classify existing components as `REUSE`, `EXTEND`, `ADAPT`, `REPLACE_WITH_JUSTIFICATION`, `NEW` or `DEPRECATE`;
5. implement all mandatory backend, frontend, realtime, research, database, Windows, MT4/MT5, licensing, subscription, referral, commission, payout, dashboard, security, observability and release requirements;
6. use primary authoritative documentation for version-sensitive decisions;
7. run applicable lint, unit, integration, E2E, security, load/replay, deterministic parity, migration, backup/restore and quantitative validation gates;
8. preserve `NO-TRADE`, hard-risk vetoes, server-side entitlements, immutable financial-ledger truth and no-fake-live-data rules;
9. never self-enable live execution or mutate real production financial state;
10. continue until every applicable acceptance criterion is evidenced as `PASS` or reported as a genuine external blocker with exact reason, owner/action needed and safe partial state;
11. write a final evidence report rather than merely stating that the code builds.

## B.2 Canonical CLI Command

Assuming this file is saved in the Predict-A-Trade project root as `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md`, run:

```bash
cd /srv/predict-a-trade/xauusd && { printf '%s\n\n' 'Execute this canonical Predict-A-Trade v1.0.0 SOW end-to-end against the existing repository. Audit first, preserve working behavior, implement every mandatory requirement, use authoritative primary documentation for version-sensitive decisions, run all required tests/security/quantitative/replay/release gates, never autonomously mutate live trading or real financial state, and do not stop at compile/UI success: finish with requirement-to-code traceability and evidence for every applicable acceptance criterion; report only genuine external blockers.'; cat Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md; } | codex exec --sandbox workspace-write -c 'sandbox_workspace_write.network_access=true' --search -o CODEX_FINAL_REPORT_v1.0.0.md -
```

`--sandbox workspace-write` is the required default for this SOW. Do not replace it with `--dangerously-bypass-approvals-and-sandbox` merely to avoid permission discipline.

## B.3 Completion Output

The final Codex report must include:

```text
overall status: PASS / PARTIAL / BLOCKED
requirements completed
requirements not completed
traceability matrix
files created/changed
migrations
tests and exact results
security findings
performance/load evidence
quantitative validation evidence
broker/execution validation status
commercial-ledger reconciliation status
Windows/MT4/MT5 validation status
UI/E2E/accessibility status
backup/restore/DR status
known limitations
external blockers
rollback path
production GO/NO-GO recommendation
```

`PASS` is forbidden while any mandatory, applicable P0 safety/security/financial-integrity requirement is unverified or failed.


---

# Appendix C — Implementation Status (2026-08-17 Codex Run)

This appendix annotates the implementation status of SOW requirements after the 2026-08-17 production-readiness completion run. It does not modify the requirements above — it only records what has been implemented and verified.

## Status Markers

- `IMPLEMENTED` — Code exists and is wired
- `VERIFIED` — Code exists, wired, and tested (unit/integration)
- `IMPROVED` — Existing implementation was enhanced during this run
- `EXTERNAL_DEPENDENCY` — Code is complete but final verification requires an unavailable external service

## Section Status

| Section | Requirement | Status | Notes |
|---------|-------------|--------|-------|
| §1 | Executive Objective | VERIFIED | All four planes implemented and tested |
| §3 | Four-Plane Architecture | VERIFIED | Go, NestJS, Next.js, Python with correct boundaries |
| §5 | NestJS Modules | VERIFIED | 12 modules including new Admin module |
| §6 | Market Data Architecture | IMPLEMENTED | Go engine with simulated provider; live requires credentials (EXTERNAL_DEPENDENCY) |
| §10 | Multi-Timeframe Engine | VERIFIED | M1-D1 aggregation in Go |
| §11 | Market Regime Engine | VERIFIED | Trending/Range/Breakout/MeanReversion |
| §12 | Quantitative Market Engines | VERIFIED | Structure, Liquidity, FVG, VWAP, Indicators, MTF, Session |
| §12A | Four Canonical Strategies | VERIFIED | STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING |
| §12C | Strategy Confluence Engine | VERIFIED | Deterministic scoring, mandatory pillar check, 4 distinct profiles |
| §13 | Macro Intelligence | IMPLEMENTED | Types defined; full ingestion requires external data |
| §14 | News Blackout Engine | VERIFIED | News gate with veto on HIGH/BLOCKED |
| §15-16 | Prediction Contract & Scoring | VERIFIED | Full lifecycle, calibrated probability |
| §17 | Signal Quality Grades | VERIFIED | A+/A/B/C/NO-TRADE/RESEARCH/SHADOW |
| §18 | NO-TRADE Reasons | VERIFIED | 25+ standardized reason codes |
| §19 | Signal Lifecycle | VERIFIED | 20+ lifecycle states |
| §22 | Realtime Events | VERIFIED | Versioned envelopes, sequence, P0/P1 |
| §24 | Master Decision | VERIFIED | Confluence→Direction→Gates→BUY/SELL/NO-TRADE |
| §25 | Risk Engine | VERIFIED | 12 gates, fail-closed, short-circuit |
| §25B | Broker-Aware Pricing | VERIFIED | BrokerProfile with all required fields |
| §31-34 | Auth/IAM | VERIFIED | Register, Login, JWT, MFA, Refresh rotation, Password reset |
| §35 | RBAC | IMPROVED | JWT includes role; AdminGuard on all admin endpoints |
| §44 | Windows Agent | VERIFIED | Device key, WS connect, heartbeat, signal reception |
| §48 | MT4/MT5 IPC | EXTERNAL_DEPENDENCY | Named pipe framework implemented; requires Windows terminal |
| §55 | DB Schemas | VERIFIED | 11 schemas, 7 migrations, unique constraints |
| §62-68 | Plans & Pricing | VERIFIED | Plans with entitlements, JwtAuthGuard |
| §63-68 | Subscriptions | VERIFIED | User + admin endpoints |
| §69 | Commission Engine | VERIFIED | 5-level chain, L1-only second payment, 16 tests |
| §69 | Referral System | VERIFIED | Network, commissions, cycle prevention |
| §69 | Payouts | VERIFIED | User + admin, approve flow |
| §72 | Session Security | IMPROVED | 384-bit tokens, SHA-256, rotation, reuse detection, user status check |
| §80 | Security Headers | IMPROVED | helmet, HSTS, CSP, Cache-Control on auth, CSRF origin validation |
| §131 | Hard Risk Gates | VERIFIED | 12 gates, fail-closed, freshness stamped |
| §134 | Quant Math | VERIFIED | Go/Python parity verified |
| §137 | Backtesting | VERIFIED | Cost-aware backtester, walk-forward, locked OOS |
| §149 | Observability | VERIFIED | 15+ Prometheus metrics, structured JSON logging |
| §152-184 | Admin Dashboard | IMPROVED | Real API data replacing hardcoded values; 12 admin endpoints |
| §185-200 | UI/UX Visual System | VERIFIED | Design tokens, dark/light themes, app shell, loading/empty/error states |
| §192 | User Dashboard | IMPROVED | All 12+ pages wired to real APIs (previously stub templates) |
| §193 | Admin Dashboard | IMPROVED | Real statistics from database, no fabricated data |
| §194 | Command Center Modes | IMPLEMENTED | MARKET, TRADING, GROWTH, COMMAND_CENTER modes |

## Admin/User Dashboard Separation (SOW §185.4, §157.2)

### User/Subscriber Dashboard
- **Route:** `/dashboard/*`
- **Data sources:** User-scoped API endpoints (`/subscriptions`, `/referrals`, `/commissions`, `/payouts`, `/licensing`)
- **RBAC:** `JwtAuthGuard` only — any authenticated user can access
- **Content:** Trading intelligence, signals, market data, subscription, referral, license, MT setup, security

### Admin/Super Admin Dashboard
- **Route:** `/admin/*`
- **Data sources:** Admin API endpoints (`/admin/*`) protected by `AdminGuard`
- **RBAC:** `JwtAuthGuard` + `AdminGuard` (requires `role: "ADMIN"` in JWT)
- **Content:** System overview, user management, subscription oversight, commission control, payout management, license/device management, audit
- **Separation:** Different layout, different navigation menu, different API namespace, backend-enforced RBAC

## Test Summary (2026-08-17)

| Suite | Count | Status |
|-------|-------|--------|
| Go unit tests | 278 tests | PASS |
| Python tests | 98 | PASS |
| NestJS tests | 75 (auth, commission, device-auth, security-validation) | PASS |
| Next.js tests | 39 | PASS |
| **Total** | **490 tests** | **ALL PASS** |

## Build Summary (2026-08-17)

| Build | Status |
|-------|--------|
| Go realtime engine | PASS |
| Windows Agent | PASS |
| NestJS control plane | PASS |
| Next.js frontend (24/24 pages) | PASS |
| Backend typecheck | PASS (0 errors) |
| Frontend typecheck | PASS (0 errors) |
| Backend lint | PASS (0 errors, 4 warnings) |
| Frontend lint | PASS (0 errors, 3 warnings) |

---

## Implementation Status (Audit Date: 2026-08-18)

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Four strategies | ✅ IMPLEMENTED | All four Evaluate() called in main.go:486 |
| 12 hard gates | ✅ IMPLEMENTED | All 12 registered in main.go:723-734 |
| PTB intelligence (20+ modules) | ✅ IMPLEMENTED (SHADOW) | ptb/modules.go, zero score impact |
| Signal persistence | ✅ IMPLEMENTED | persister.SaveSignal in main.go |
| Candidate persistence | ✅ IMPLEMENTED | persister.SaveCandidate in main.go |
| Risk decision persistence | ✅ IMPLEMENTED | persister.SaveRiskDecision in main.go |
| Cooldown management | ✅ IMPLEMENTED | signal/cooldown.go |
| Duplicate prevention | ✅ IMPLEMENTED | signal/delivery.go |
| Calibration | ✅ IMPLEMENTED | calibration/consumer.go |
| WebSocket broadcast | ✅ IMPLEMENTED | gateway/websocket.go |
| Agent WebSocket | ✅ IMPLEMENTED | gateway/agent_ws.go |
| Loss recovery | ✅ IMPLEMENTED | recovery/manager.go (16 tests) |
| Adaptation | ✅ IMPLEMENTED | adaptation/manager.go (8 tests) |
| Hedging | ✅ IMPLEMENTED (disabled) | hedging/manager.go (10 tests) |
| ML adaptation | ✅ IMPLEMENTED (disabled) | ml/adaptation.go (9 tests) |
| RL optimizer | ✅ IMPLEMENTED (disabled) | rl/optimizer.go (8 tests) |
| Sentiment | ✅ IMPLEMENTED (disabled) | sentiment/engine.go (9 tests) |
| Daily maintenance | ✅ IMPLEMENTED | maintenance/scheduler.go (3 tests) |
| Entitlement enforcement | ✅ IMPLEMENTED | Gates derive from authoritative state via ResolveEntitlementState() — no hardcoded true |
| Auth + MFA | ✅ IMPLEMENTED | control/auth/ (68 tests) |
| Licensing + device auth | ✅ IMPLEMENTED | control/licensing/ |
| Subscriptions | ✅ IMPLEMENTED | control/subscriptions/ |
| Referrals + commissions | ✅ IMPLEMENTED | control/referrals/, control/commissions/ |
| Payouts | ✅ IMPLEMENTED | control/payouts/ |
| Frontend | ✅ IMPLEMENTED | Next.js build passes, 39 tests, 40+ pages |
| MT4 EA | ✅ IMPLEMENTED | mql/mt4/ (not runtime validated) |
| MT5 EA | ✅ IMPLEMENTED | mql/mt5/ (not runtime validated) |
| Windows Agent | ✅ IMPLEMENTED | windows-agent/ (not runtime validated) |
| Nginx + TLS | ✅ IMPLEMENTED | infra/nginx/ (5 site configs) |
| Systemd | ✅ IMPLEMENTED | infra/systemd/ (4 units) |
| Prometheus + Grafana | ✅ IMPLEMENTED | infra/prometheus/, infra/grafana/ |
| Backup scripts | ✅ IMPLEMENTED | scripts/backup/ |
| Backtesting framework | ✅ IMPLEMENTED | research/backtesting/ (72 tests) |
| Database (165 tables) | ✅ IMPLEMENTED | 15 migrations, all additive |
| COT report | ✅ IMPLEMENTED | cot_provider.go — FMP API adapter, 8 tests, fails safe on 402 |
| DXY correlation | ✅ IMPLEMENTED | dxy_provider.go — Twelve Data API, ICE formula, 6 tests |
| Volume Profile/CVD | 📡 UNSUPPORTED BY DATA SOURCE | Broker tick volume only |
| Live MT4/MT5 validation | 🖥 RUNTIME VALIDATION REQUIRED | Code exists, not tested with live terminal |

**Overall: 490 tests, 0 failures. Production Readiness: CONDITIONAL GO**

### v1.3.0 Remediation Summary (2026-08-18)
| Blocker | Resolution |
|---------|-----------|
| P1-001 Hardcoded gate values | ResolveEntitlementState from gate registry |
| P1-002 Simulated provider in prod | PROVIDER_MODE=agent + config validation |
| P2-001 Placeholder JWT secret | Generated via openssl, stored in gitignored file |
| P2-002 Hardcoded DB password | Stored in gitignored database_url.txt, loaded by Go+NestJS |
| P2-003 Permissive gate seeding | SeedConservativeGateStates - safety gates start UNKNOWN |
| Agent to Gate hydration | SetBrokerAccountHydrateFn / SetAgentConnectFn callbacks |
| COT provider | cot_provider.go - Financial Modeling Prep API adapter |
| DXY provider | dxy_provider.go - Twelve Data API, ICE DXY from 6 currencies |
| SMTP | mail.predictatrade.com:587 - verified working |
| Error codes | Windows Agent distinguishes AUTH/LICENSE/SYSTEM/RISK errors |
| WS entitlement | isEntitled fail-closed - empty = no signal delivery

### v1.3.0 Remediation Summary (2026-08-18)
| Blocker | Resolution |
|---------|-----------|
| P1-001 Hardcoded gate values |  from gate registry |
| P1-002 Simulated provider in prod |  + config validation |
| P2-001 Placeholder JWT secret | Generated via openssl, stored in gitignored file |
| P2-002 Hardcoded DB password | Stored in gitignored database_url.txt, loaded by Go+NestJS |
| P2-003 Permissive gate seeding |  — safety gates start UNKNOWN |
| Agent → Gate hydration |  /  callbacks |
| COT provider |  — Financial Modeling Prep API adapter |
| DXY provider |  — Twelve Data API, ICE DXY from 6 currencies |
| SMTP | mail.predictatrade.com:587 — verified working |
| Error codes | Windows Agent distinguishes AUTH/LICENSE/SYSTEM/RISK errors |
| WS entitlement |  fail-closed — empty = no signal delivery |

All repository-controlled software blockers (P1-001, P1-002, P2-001, P2-002, P2-003) are RESOLVED.
Remaining: external configuration (SMTP/TLS/COT/DXY) and runtime validation (live MT4/MT5).
