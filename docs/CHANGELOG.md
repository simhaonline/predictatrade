# Changelog

## Current Version: v1.3.1 (19 August 2026) — Signal Display Fix + Documentation Update

### Version Summary

| Version | Date | Key Changes | Tests |
|---------|------|-------------|-------|
| v1.0.0 | 2026-08-18 | Stage 4 PTB: 20+ intelligence modules, 4 strategies, 12 gates, golden tests | 252 |
| v1.1.0 | 2026-08-18 | Advanced Risk: loss recovery, adaptation, hedging, ML/RL, sentiment, maintenance | 376 |
| v1.2.0 | 2026-08-18 | Backtesting Framework: event-driven engine, execution sim, walk-forward, Monte Carlo | 448 |
| v1.3.0 | 2026-08-18 | Production Remediation: gate fixes, COT/DXY adapters, SMTP, JWT/DB secrets, agent wiring | 490 |
| v1.3.1 | 2026-08-19 | Signal display fix: BUY_CANDIDATE/SELL_CANDIDATE filters, PROB "Pending" label, candidate CalibratedProbability | 490 |

### v1.3.1 Changes (19 August 2026)

#### Fixed
- **SIGNAL GENERATION STOPPED — 3 root causes fixed**:
  1. **Duplicate key constraint (`idx_signals_canonical_idempotency`)**: When the same candle was processed multiple times (agent sends multiple snapshots), the second insert failed. Now gracefully handles canonical idempotency duplicates (returns nil = already saved, not an error).
  2. **Entitlement/License gates stuck at UNKNOWN**: v1.3.0 introduced conservative gate seeding (UNKNOWN = fail-closed), but no hydration mechanism existed for entitlement and license gates. Added background goroutine `hydrateEntitlementLicenseGates()` that queries `licensing.licenses` (ACTIVE) and `billing.subscriptions` (ACTIVE) every 10 seconds and hydrates gates to PASS when active records exist.
  3. **Session gate rejecting overlap sessions**: `IsSessionAllowed()` only accepted `OVERLAP` but the MT5 agent reports `LONDON_NEWYORK_OVERLAP`. Added all overlap session variants and fallback pattern matching for unknown session names containing known session keywords.
- **Admin signals page**: Added BUY_CANDIDATE and SELL_CANDIDATE to direction filters (previously only BUY, SELL, NO-TRADE)
- **Admin signals page**: Added color coding for BUY_CANDIDATE (amber) and SELL_CANDIDATE (orange) directions
- **Admin signals page**: PROB column now shows "Pending" instead of "—" when calibration is unverified (with explanatory text)
- **Admin signals page**: Added legend explaining BUY vs BUY_CANDIDATE vs NO-TRADE and probability calibration requirement
- **User signals page**: Added color coding for BUY_CANDIDATE (amber) and SELL_CANDIDATE (orange) directions
- **User signals page**: PROB column now shows "Pending" with tooltip instead of "—" when calibration is unverified
- **User signals page**: Added legend explaining signal direction types
- **Go engine**: `CalibratedProbability` field now set on advisory/candidate (BUY_CANDIDATE/SELL_CANDIDATE) signals — was missing, causing PROB to always be zero for candidates
- **StatusBadge component**: Added signal lifecycle statuses (DETECTED, CONFIRMED, CANDIDATE, BLOCKED, INVALIDATED, TRIGGERED, FILLED, STOPPED, CLOSED)

#### Added
- `docs/SIGNAL_TYPES_AND_PROBABILITY.md` — comprehensive reference for signal direction types, candidate thresholds, calibration probability, and frontend display behavior

#### Documentation Updated
- `docs/CHANGELOG.md` — v1.3.1 entry
- `docs/IMPLEMENTATION_STATUS.md` — signal display status
- `docs/SIGNAL_TYPES_AND_PROBABILITY.md` — new comprehensive signal reference
- `docs/strategy/STRATEGY_PLAYBOOKS.md` — candidate threshold documentation
- `docs/api/API_REFERENCE.md` — signal direction types
- `docs/guides/USER_GUIDE.md` — signal types explanation
- `docs/guides/ADMIN_GUIDE.md` — admin signal panel explanation
- `docs/INDEX.md` — new document index entry

### Technical Details

**BUY vs BUY_CANDIDATE distinction:**
- `BUY` = score ≥ trade threshold + all 12 hard gates passed → EXECUTABLE
- `BUY_CANDIDATE` = candidate threshold ≤ score < trade threshold → ADVISORY (not executable)
- This is a safety design: candidates show directional conviction forming but not yet qualified

**PROB "Pending" behavior:**
- Per SOW §16, §36: calibrated probability must be NULL/zero until calibration model is VALIDATED or PROMOTED
- Default seeded models have Status: "UNVERIFIED" — this is correct
- UI now shows "Pending" instead of "—" to make the state clear
- Raw score is always shown in the separate Score column



### Production Audit Remediation (2026-08-18)
- All 5 P1/P2 blockers resolved (P1-001, P1-002, P2-001, P2-002, P2-003)
- COT provider adapter implemented (Financial Modeling Prep API)
- DXY provider adapter implemented (Twelve Data API, 6-component ICE formula)
- SMTP configured and verified (mail.predictatrade.com:587, STARTTLS)
- JWT secret loaded from gitignored secret file (openssl rand -base64 32)
- DATABASE_URL loaded from gitignored secret file (database_url.txt)
- Windows Agent error code separation (AUTH_TOKEN_EXPIRED, LICENSE_EXPIRED, etc.)
- Agent connectivity → gate hydration wiring (SetBrokerAccountHydrateFn, SetAgentConnectFn)
- WebSocket entitlement fail-closed (empty entitlements = no signal delivery)
- Conservative gate seeding (safety-critical gates start UNKNOWN → fail closed)
- 490 tests, 0 failures
- Decision: CONDITIONAL GO (software blockers: 0)

---

## Previous Versions



### Version Summary

| Version | Date | Key Changes | Tests |
|---------|------|-------------|-------|
| v1.0.0 | 2026-08-18 | Stage 4 PTB: 20+ intelligence modules, 4 strategies, 12 gates, golden tests | 252 |
| v1.1.0 | 2026-08-18 | Advanced Risk: loss recovery, adaptation, hedging, ML/RL, sentiment, maintenance | 376 |
| v1.2.0 | 2026-08-18 | Backtesting Framework: event-driven engine, execution sim, walk-forward, Monte Carlo | 448 |

### Production Audit (2026-08-18)
- Full forensic audit completed: CONDITIONAL GO
- 2 P1 critical, 3 P2 high — ALL RESOLVED in v1.3.0
- Decision: CONDITIONAL GO (software blockers: 0, external config only)
- See: `PRODUCTION_FULL_AUDIT_REPORT.md`

---

## v1.0.0 — Stage 4 PTB (18 August 2026)

### Added
- Professional Trader Brain (PTB) shared intelligence layer with 20 advanced modules
- PTB Synthesis Engine: confluence scoring, bias determination, setup quality (A+ through F)
- Position Size Multiplier (advisory only — risk gates remain authoritative)
- Dynamic Stop Distance Multiplier (context-aware: manipulation, volatility)
- Gold Correlation Engine (Pearson correlation for DXY, silver, yields — awaiting live feed)
- Gold Role Classification (CURRENCY, SAFE_HAVEN, MONETARY_ASSET, etc. — returns UNKNOWN without data)
- Enhanced Regime Classification (9 states: STRONG_TREND_UP through MANIPULATION)
- Liquidity Targeting (directional targets from swing highs/lows)
- Machine-readable Reason Codes (MTF_BULLISH_ALIGNMENT, HIGH_MANIPULATION_RISK, etc.)
- Deterministic Market Narrative generation
- Centralized PTB Configuration (`ptb/config.go` with env-var overrides)
- 10 Prometheus observability metrics for PTB
- Data Authenticity Guard (rejects non-LIVE_MASTER_NODE sources)
- Signal State Separation: BUY, SELL, WAIT, NO-TRADE, BLOCKED, ERROR
- Feature Flag System (OFF, SHADOW, ACTIVE, DISABLED, UNSUPPORTED, RESEARCH)
- Database Migration 012: PTB feature flags, evidence snapshots, data provenance log
- Database Migration 013: PTB analysis history, signal performance feedback
- REST API gap fix: reason_codes, evidence, gate_results now retrieved
- 252 tests (up from 207 baseline)

### Fixed (Stage 2 carry-forward)
- Calibration sigmoid saturation (score >100 clamped to [0,100])
- StandardScalping missing applyFamilyCaps (double-counting prevention)
- Liquidity sweep direction case mismatch (dead wiring fixed)
- MTF evidence contribution overscaling (normalized /100)
- Stochastic signal line (3-period SMA of %K instead of = StochMain)

### All modules default to SHADOW mode with zero production score impact.

## v1.0.0 — Stage 2 Forensic Verification (18 August 2026)

### Fixed
- 5 confirmed defects (2 P0, 2 P1, 1 P2)
- 23 golden tests for indicator math, score math, signal states, risk math
- 209.7/99.1% calibration saturation root-caused and fixed
- Four strategy engines verified independently correct
- AI vs deterministic classification completed (all deterministic)
- Billing/referral isolation from trading math proven

## v1.1.0 — Advanced Risk, Adaptation, Hedging, ML/RL, Sentiment (18 August 2026)

### Added — Loss Recovery / Capital Protection
- Loss Recovery Manager with state machine (NORMAL → RECOVERY → HALTED / DAILY_LIMIT)
- Anti-martingale, anti-revenge-trading enforcement
- Correct daily loss PnL sign logic (daily_pnl_percent <= -max_daily_loss_percent, never abs())
- State isolation per account+strategy+symbol
- Duplicate broker close event deduplication
- Restart-safe state persistence (export/restore)
- 16 focused tests covering all state transitions and edge cases

### Added — Rule-Based Adaptation
- Adaptation Manager with 6 market phases (TRENDING, RANGING, HIGH_VOLATILITY, LOW_VOLATILITY, MANIPULATIVE, UNCERTAIN)
- Dynamic parameter adjustment (stop distance, risk multiplier, confluence, confidence, weights)
- Risk clamping hierarchy (adaptive → strategy → account → global hard max)
- Deep copy of weights — never mutates base config
- NaN/Inf rejection, weight normalization
- 8 tests covering all phases, clamping, fallback

### Added — Controlled Hedging
- Controlled Hedge Manager (DISABLED BY DEFAULT)
- Pre-execution checks: broker capability, netting mode, license, exposure, thresholds
- Partial hedging with size cap (never exceeds original)
- No martingale escalation
- Grid hedging OFF by default, options hedging OFF by default
- Auto-close on expiry, trailing stop support
- Full lifecycle audit trail (original_trade_id ↔ hedge_trade_id correlation)
- 10 tests covering disabled state, thresholds, dedup, netting, exposure, expiry

### Added — ML-Based Adaptation
- ML Adaptation Manager with comprehensive fallback chain
- Model Registry with versioning, staleness detection, sample count validation
- Inference bounds clamping (stop 0.5-2.0, size 0.1-1.0, confluence 50-100)
- Python training pipeline with chronological split, walk-forward validation
- Data leakage protection (feature timestamps must precede outcomes)
- 9 Go tests + 4 Python tests

### Added — RL Strategy Optimizer
- RL Manager with 4 deployment modes (disabled, shadow, filter_only, live_approved)
- Filter mode can only veto (NO_TRADE) — cannot create trades
- Live approval requires explicit authorization + validation metrics
- Python RL training environment with multi-component reward function
- Reward includes: PnL, drawdown penalty, transaction cost, spread, slippage, overtrading, risk exposure, holding cost
- 8 Go tests + 5 Python tests

### Added — Real-Time Sentiment Engine
- Async background refresh — NEVER blocks signal hot path
- Provider abstraction interface (one unavailable provider doesn't break system)
- Timeout, retry/backoff, rate-limit handling, stale-data detection
- Provider health tracking with error counts
- Neutral/no-impact fallback when unavailable or stale
- Source weighting, confidence threshold
- 9 tests covering success, timeout, rate limit, partial failure, no provider, stale, neutral, weighting, non-blocking

### Added — Daily Maintenance
- UTC-based daily reset scheduler
- Idempotent (prevents duplicate execution in multi-instance deployments)
- Resets daily loss counters, clears DAILY_LIMIT state
- 3 tests

### Added — Database Migration 014
- trading.recovery_states — loss recovery state machine
- trading.trade_results — closed trade outcomes
- trading.blocked_signals — blocked signal audit trail
- trading.adaptation_history — adaptation decisions
- trading.hedge_positions — active hedge lifecycle
- trading.hedge_history — closed hedge audit trail
- trading.rl_training_history — RL training runs with OOS metrics
- trading.sentiment_snapshots — cached sentiment state
- trading.sentiment_items — sentiment data points with provenance

### Added — Signal Pipeline Integration
- AdvancedManagers struct holding all advanced intelligence managers
- DecideWithAdvanced method wiring recovery, adaptation, ML, RL, sentiment into signal engine
- Advanced gates are ADDITIONAL to existing 12 hard gates — never weaken them
- Integration tests for all four strategies with advanced gates
- RL filter veto test, recovery block test, NO-TRADE preservation test
- Sentiment influence test, size multiplier test, adaptation test, ML fallback test

### Added — Prometheus Metrics
- 20 new low-cardinality metrics for recovery, adaptation, hedging, ML, RL, sentiment
- No trade ID, client ID, user ID, license, or raw error text in labels

### Added — Configuration
- AdvancedConfig in config.go with env-var loading
- .env.example with all new environment variables
- All high-risk features DISABLED by default

### Added — Documentation
- docs/ADVANCED_RISK_ADAPTATION_INTELLIGENCE.md — comprehensive feature documentation
- docs/IMPLEMENTATION_STATUS.md — feature matrix
- README.md updated with advanced features section
- CHANGELOG.md updated

### Test Results
- Go: 243 tests PASS (up from 168)
- Python: 26 tests PASS (up from 16)
- NestJS: 68 tests PASS (unchanged)
- Frontend: 39 tests PASS (unchanged)
- Total: 376 tests, 0 failures

## v1.2.0 — Backtesting Framework (18 August 2026)

### Added — Backtesting Framework
- Event-driven backtest engine with deterministic event ordering
- Historical data layer: loader, quality validator, multi-timeframe alignment
- Data quality validation: timestamps, UTC, ordering, duplicates, OHLC, outliers, gaps
- Multi-timeframe alignment with NO future leakage (verified by automated tests)
- Session/calendar engine: Tokyo, London, NY, overlaps, kill zones, DST-aware
- Execution simulator: spread (fixed/dynamic), slippage (fixed/percentage/ATR), commission, latency, partial fills, rejections
- Direction correctness: LONG=ask, SHORT=bid, exit side correctness
- Conservative same-bar SL/TP policy (SL first when ambiguous)
- Portfolio engine: positions, equity, realized/unrealized PnL, drawdown, MAE/MFE
- Exit management: trailing stop (ATR-based), break-even, time exit, end-of-test close
- PTB strategy adapter: reproduces Go production evidence/confluence/risk logic
- Precomputed PTB feature pipeline with parity validation
- Precomputed PTB replay strategy
- RL standalone strategy (NO_TRADE/LONG/SHORT/CLOSE)
- RL confirmation filter (PTB candidate → RL confirm/veto)
- RL feature schema safety verification (names, order, dtype, NaN/Inf)
- Risk-gate parity: spread, session, R:R, exposure, daily loss, recovery mode
- Walk-forward analysis with train/test isolation and final untouched holdout
- Monte Carlo robustness: percentile distributions, probability of loss/drawdown
- Parameter sensitivity analysis (±%, does not alter production)
- Performance metrics: return, CAGR, Sharpe, Sortino, drawdown, Calmar, win rate, profit factor, expectancy, R:R, VaR, consecutive wins/losses
- XAUUSD segmentation: by session, regime, direction, strategy
- Report generator: summary.json, trades.csv, equity.csv, metrics.json, configuration.json, data_quality.json, run_manifest.json
- Run manifest with full reproducibility metadata
- CLI: run, validate-data, precompute, walk-forward, monte-carlo, sensitivity, list, show

### Added — Database Migration 015
- trading.backtest_runs, trading.backtest_trades, trading.backtest_fold_results, trading.backtest_artifacts, trading.backtest_parameter_sets

### Added — Tests (72 new)
- Data loading, UTC, quality, alignment, no-lookahead (15 tests)
- Execution: spread, slippage, commission, direction, rejection, partial fill (10 tests)
- Portfolio: long/short PnL, SL/TP, conservative, trailing, break-even, time exit, equity (12 tests)
- Metrics: return, Sharpe, Sortino, win rate, profit factor, edge cases (12 tests)
- Robustness: walk-forward isolation, Monte Carlo determinism, sensitivity (8 tests)
- Integration: full pipeline, NO_TRADE, blocked, data quality, all strategies (15 tests)

### Test Results
- Go: 243 tests PASS (unchanged)
- Python: 98 tests PASS (up from 26)
- NestJS: 68 tests PASS (unchanged)
- Frontend: 39 tests PASS (unchanged)
- Total: 448 tests, 0 failures

## [2026-08-19] — Screenshot-Driven Bug Audit & Repair

### Fixed
- **Go engine agents/status stale cache (AUD-001):** `/api/v1/agents/status` endpoint was returning stale Valkey cache data (`agents_connected:0`) instead of live `agentHub.AgentCount()`. This caused the admin dashboard to show "Master Node OFFLINE" and "No Windows Agent connected" while the MT5 terminal showed the agent as CONNECTED. Fixed to always use live in-memory agent connection state.
- **OperationsService missing active_strategies (AUD-002):** `getTradingState()` returned `disabled_strategies` but not `active_strategies` or `last_updated`, causing all strategies to show "Inactive" on the dashboard and "0 active" on the operations page. Now computes active strategies from the 4 canonical strategy set minus disabled ones.
- **Stale RESUME_* operations (AUD-003):** `resumeTrading()`, `resumeSignals()`, and `enableStrategy()` created operations with status `ACTIVE` that never transitioned to terminal state, accumulating as duplicate "Active Operations". Now uses status `COMPLETED` for instantaneous actions. Migration 016 cleaned up 2 existing stale records.
- **Valkey health hardcoded UNKNOWN (AUD-004):** `AdminService.systemHealth()` hardcoded Valkey/Redis status as `UNKNOWN`. Replaced with a real TCP socket connection check.
- **Trading Reports agents count (AUD-005):** Frontend used `Array.isArray(agents)` on an object response, always showing 0 connected agents. Fixed to read `agents_connected` field.

### Infrastructure
- Stopped and disabled duplicate `pat-*` systemd services that conflicted with canonical `predictatrade-*` services.

### Migrations
- `016_fix_stale_operations.sql`: Added `completed_at` column to `control.platform_operations`; cleaned up stale ACTIVE RESUME_*/ENABLE_* operations.

### Tests
- Added `operations.service.spec.ts` (10 tests) covering `getTradingState` active_strategies/last_updated, stale operation cleanup, and RESUME_*/ENABLE_* COMPLETED status.
- Added Valkey health regression tests in `admin.service.spec.ts` (2 tests) verifying status is never UNKNOWN.
- Added `http_agents_test.go` (2 tests) verifying live agent count precedence over cache and zero-agent state.

## [2026-08-19] — Admin Data Relationship & Audit Forensic Repair

### Fixed
- **Licenses page empty (root cause: no records + field mismatch):** Reconciled license `ee710bf6` for user@simhaonline.com; fixed API field mapping (license_key→key, issued_at→activated_at, added plan_name/user_email via LEFT JOINs).
- **Subscriptions page empty (root cause: no records + field mismatch):** Reconciled Elite/ACTIVE subscription; fixed API field mapping (billing_period_start→current_period_start, added license_key via LEFT JOIN).
- **Device Auth page empty (root cause: no device records):** Reconciled device record from Go engine agent data; added terminal types (MT4/MT5) and license info to API response.
- **Activations page empty (root cause: wrong endpoint):** Changed from `/devices/sessions` (empty) to new `/admin/activations` endpoint querying `device_activations`.
- **Logs & Audit empty (root cause: no audit persistence):** Wired audit logging into AuthService (LOGIN_SUCCESS/FAILED), AdminService (USER_SUSPENDED/REACTIVATED), OperationsService (HALT/RESUME), and assignLicense (LICENSE_ASSIGNED). Added metadata sanitization.
- **User Onboarding missing license mapping:** Added `getUserDetail` API with full relationship map; added `assignLicense` API; enhanced Users page with subscription/license/device/activation display and license assignment UI.

### Added
- `AdminService.listAllActivations()` — queries device_activations with user/license/device JOINs
- `AdminService.assignLicense()` — creates license for user with audit event
- `AdminService.getUserDetail()` — returns user with subscription, licenses, devices, activations
- `AdminController` endpoints: GET /admin/activations, GET /admin/users/:id/detail, POST /admin/users/:id/assign-license
- Audit event logging in auth, admin, and operations services
- Frontend: license assignment UI, terminal type display, user detail drawer with relationship map

### Migrations
- `017_reconcile_production_data.sql`: Reconciled subscription, license, device, device_activations, license_event, and audit_event records for user@simhaonline.com

### Tests
- 6 new regression tests in `admin.service.spec.ts`: listAllLicenses, listAllSubscriptions, listAllDevices, listAllActivations, getUserDetail — all verify known production records appear
