# Predict-A-Trade — Final Traceability Matrix

**Version:** v1.15.0 — Server-Side SL Enforcement + Legal Compliance + CI/CD  
**Date:** 25 August 2026  
**Previous:** v1.10.1 (21 August 2026)

---

## v1.0.0 — Foundation + PTB

| SOW Section | Requirement | Implementation Files | API Endpoints | Frontend Routes | Tests | Status |
|---|---|---|---|---|---|---|
| 3.1 | Go Real-Time Trading Plane | `realtime/cmd/`, `realtime/internal/` | Go HTTP/WS gateway | N/A | Go tests (gates, strategy, math) | COMPLETE |
| 3.2 | Python Research Plane | `research/src/patresearch/` (incl. `quantitative_strategy_engine.py`) | N/A | N/A | Python tests (127) | COMPLETE |
| 3.3 | NestJS Control Plane | `control/src/` | All `/api/v1/*` routes | N/A | Backend tests | COMPLETE |
| 3.4 | Next.js Presentation Plane | `frontend/src/` | N/A | All routes | Frontend tests | COMPLETE |
| 5 | NestJS Module List | `control/src/modules/` | Auth, Users, Plans, Subscriptions, Billing, Referrals, Commissions, Payouts, Licensing, Audit, Health, Admin, Operations, Device-Auth | N/A | Backend tests | COMPLETE |
| 6 | Market Data Architecture | `realtime/internal/marketdata/` | Go HTTP API | Chart, Market Pulse | Go tests | COMPLETE |
| 10 | Multi-Timeframe Engine | `realtime/internal/features/mtf.go` | Go HTTP API | Dashboard | Go tests | COMPLETE |
| 11 | Market Regime Engine | `realtime/internal/features/regime.go` | Go HTTP API | Dashboard | Go tests | COMPLETE |
| 12 | Quantitative Market Engines | `realtime/internal/features/` | Go HTTP API | Dashboard | Go tests | COMPLETE |
| 12A | Four Strategy Playbooks | `realtime/internal/strategy/strategies.go` | Go HTTP API | Strategies | Go strategy tests | COMPLETE |
| 12C | Strategy Confluence Engine | `realtime/internal/strategy/confluence.go` | Go HTTP API | N/A | Go confluence tests | COMPLETE |
| 13 | Macro Intelligence | `realtime/internal/features/` (partial) | Go HTTP API | Dashboard | Go tests | PARTIAL |
| 14 | News Blackout Engine | `realtime/internal/gates/` | Go HTTP API | N/A | Go gate tests | COMPLETE |
| 15 | Prediction Contract | `realtime/internal/signal/engine.go` | Go HTTP API | Signals | Go tests | COMPLETE |
| 16-17 | Signal Scoring & Grades | `realtime/internal/signal/engine.go` | Go HTTP API | Signals | Go tests | COMPLETE |
| 19 | Signal Lifecycle | `realtime/internal/signal/engine.go` | Go HTTP API | Signals, Positions | Go tests | COMPLETE |
| 25 | Risk Engine | `realtime/internal/gates/` | Go HTTP API | Risk | Go gate tests | COMPLETE |
| 25B | Broker-Aware Price Units | `realtime/internal/types/types.go` | Go HTTP API | N/A | Go tests | COMPLETE |
| 34 | IAM | `control/src/modules/auth/`, `control/src/modules/users/`, migrations 002, 007 | `/auth/*`, `/users/*`, `/admin/users` | Login, Register, Security | Backend auth tests | COMPLETE |
| 35 | RBAC | `control/src/common/guards/`, migrations 002 | AdminGuard on admin endpoints | Admin routes | Backend tests | COMPLETE |
| 36 | Organizations/Tenants | migration 002 (schema exists) | N/A | N/A | N/A | PARTIAL (schema only) |
| 61 | API Credentials | migration 002 (schema exists) | N/A | N/A | N/A | PARTIAL (schema only) |
| 62-68 | Plans & Pricing | `control/src/modules/plans/` | `/plans`, `/admin/plans` | Subscription | Backend tests | COMPLETE |
| 63-68 | Subscriptions | `control/src/modules/subscriptions/` | `/subscriptions`, `/admin/subscriptions` | Subscription | Backend tests | COMPLETE |
| 69 | Commission Engine | `control/src/modules/commissions/commission-engine.ts` | `/commissions`, `/admin/commissions` | Commissions | Commission engine tests | COMPLETE |
| 69 | Referral System | `control/src/modules/referrals/` | `/referrals/*` | Referral | Backend tests | COMPLETE |
| 69 | Payouts | `control/src/modules/payouts/` | `/payouts/*`, `/admin/payouts` | Payouts | Backend tests | COMPLETE |
| 72 | Session Security | `control/src/modules/auth/auth.service.ts`, migrations 006, 007 | `/auth/refresh`, `/auth/logout` | Login, Security | Backend auth tests | COMPLETE |
| 80 | Security Headers | `control/src/main.ts`, `infra/nginx/` | All endpoints | All routes | N/A | COMPLETE |
| 82 | Observability | `realtime/internal/observability/`, `infra/prometheus/` | `/health`, `/metrics` | Infrastructure | N/A | COMPLETE |
| 131 | Windows Agent | `windows-agent/` | Agent WS | MT Setup | Go build | COMPLETE |
| 132 | MT4/MT5 EAs | `mql/mt4/`, `mql/mt5/` | N/A | N/A | N/A | COMPLETE |
| 152-184 | Admin Dashboard | `frontend/src/app/(admin)/` (20+ pages) | `/admin/*` (12 endpoints) | Admin | Frontend tests | COMPLETE |
| 185-200 | Visual System | `frontend/src/styles/`, design tokens | N/A | All auth/dashboard | Frontend build | COMPLETE |
| - | Domain Routing | `infra/nginx/`, env files | All domains | All routes | N/A | COMPLETE |
| - | CI/CD | `.github/workflows/ci.yml` | N/A | N/A | CI config | COMPLETE |
| - | Docker Compose | `docker-compose.yml` | N/A | N/A | N/A | COMPLETE |
| - | Systemd Services | `infra/systemd/` (4 units) | N/A | N/A | N/A | COMPLETE |

## v1.0.0 — PTB Intelligence (Stage 4)

| Requirement | Implementation Files | Tests | Status |
|---|---|---|---|
| PTB shared intelligence layer (20+ modules) | `realtime/internal/ptb/` (9 files) | Go tests | COMPLETE |
| PTB Synthesis Engine | `realtime/internal/ptb/synthesis.go` | Go tests | COMPLETE |
| Position Size Multiplier (advisory) | `realtime/internal/ptb/synthesis.go` | Go tests | COMPLETE |
| Dynamic Stop Distance Multiplier | `realtime/internal/ptb/synthesis.go` | Go tests | COMPLETE |
| Gold Correlation Engine | `realtime/internal/ptb/` | Go tests | COMPLETE (awaiting feed) |
| Gold Role Classification | `realtime/internal/ptb/` | Go tests | COMPLETE |
| Enhanced Regime (9 states) | `realtime/internal/ptb/` | Go tests | COMPLETE |
| Liquidity Targeting | `realtime/internal/ptb/` | Go tests | COMPLETE |
| Machine-readable Reason Codes | `realtime/internal/ptb/` | Go tests | COMPLETE |
| Deterministic Market Narrative | `realtime/internal/ptb/` | Go tests | COMPLETE |
| Centralized PTB Configuration | `realtime/internal/ptb/config.go` | Go tests | COMPLETE |
| Data Authenticity Guard | `realtime/internal/ptb/` | Go tests | COMPLETE |
| Signal State Separation (6 states) | `realtime/internal/types/types.go` | Go tests | COMPLETE |
| Feature Flag System | `realtime/internal/ptb/` | Go tests | COMPLETE |
| Database Migration 012 (PTB tables) | `database/migrations/012_ptb_intelligence_tables.sql` | N/A | COMPLETE |
| Database Migration 013 (PTB performance) | `database/migrations/013_ptb_synthesis_performance.sql` | N/A | COMPLETE |
| 10 Prometheus PTB Metrics | `realtime/internal/observability/` | N/A | COMPLETE |
| 252 tests (v1.0.0 baseline) | — | 252 | COMPLETE |

## v1.1.0 — Advanced Risk, Adaptation, Hedging, ML/RL, Sentiment

| Requirement | Implementation Files | Tests | Status |
|---|---|---|---|
| Loss Recovery Manager | `realtime/internal/recovery/manager.go` | 16 tests | COMPLETE |
| Daily Circuit Breaker | `realtime/internal/recovery/manager.go` | 2 tests | COMPLETE |
| Recovery Mode (state machine) | `realtime/internal/recovery/manager.go` | 4 tests | COMPLETE |
| Rule-Based Adaptation Manager | `realtime/internal/adaptation/manager.go` | 8 tests | COMPLETE |
| Controlled Hedge Manager | `realtime/internal/hedging/manager.go` | 10 tests | COMPLETE |
| Grid Hedging (OFF) | `realtime/internal/hedging/manager.go` | 1 test | COMPLETE |
| Options Hedging (OFF) | `realtime/internal/hedging/manager.go` | 1 test | COMPLETE |
| ML Adaptation Manager | `realtime/internal/ml/adaptation.go`, `ml_training.py` | 9+4 tests | COMPLETE |
| ML Model Registry | `realtime/internal/ml/adaptation.go` | 2 tests | COMPLETE |
| RL Strategy Optimizer | `realtime/internal/rl/optimizer.go`, `rl_training.py` | 8+5 tests | COMPLETE |
| RL Shadow/Filter/Live Modes | `realtime/internal/rl/optimizer.go` | 4 tests | COMPLETE |
| Real-Time Sentiment Engine | `realtime/internal/sentiment/engine.go` | 9 tests | COMPLETE |
| Sentiment Cache | `realtime/internal/sentiment/engine.go` | 2 tests | COMPLETE |
| Daily Maintenance Scheduler | `realtime/internal/maintenance/scheduler.go` | 3 tests | COMPLETE |
| Database Migration 014 | `database/migrations/014_advanced_risk_adaptation_intelligence.sql` | N/A | COMPLETE |
| 376 tests (v1.1.0) | — | 376 | COMPLETE |

## v1.2.0 — Backtesting Framework

| Requirement | Implementation Files | Tests | Status |
|---|---|---|---|
| Historical Data Layer | `research/.../backtesting/data/loader.py` | 5 tests | COMPLETE |
| Data Quality Validation | `research/.../backtesting/data/quality.py` | 5 tests | COMPLETE |
| Multi-TF Alignment (no-lookahead) | `research/.../backtesting/data/alignment.py` | 3 tests | COMPLETE |
| Session/Calendar Engine | `research/.../backtesting/data/session_calendar.py` | 4 tests | COMPLETE |
| Event-Driven Engine | `research/.../backtesting/engine/core.py` | 0 tests | COMPLETE |
| Execution Simulator | `research/.../backtesting/engine/execution.py` | 10 tests | COMPLETE |
| Portfolio Engine | `research/.../backtesting/engine/portfolio.py` | 12 tests | COMPLETE |
| PTB Strategy Adapter | `research/.../backtesting/strategy/ptb_strategy.py` | 5 tests | COMPLETE |
| Precomputed PTB | `research/.../backtesting/strategy/precomputed_ptb_strategy.py` | 1 test | COMPLETE |
| RL Standalone + Filter | `research/.../backtesting/strategy/rl_strategy.py` | 5 tests | COMPLETE |
| Walk-Forward Analysis | `research/.../backtesting/analytics/walk_forward.py` | 3 tests | COMPLETE |
| Monte Carlo | `research/.../backtesting/analytics/monte_carlo.py` | 3 tests | COMPLETE |
| Parameter Sensitivity | `research/.../backtesting/analytics/sensitivity.py` | 2 tests | COMPLETE |
| Performance Metrics | `research/.../backtesting/analytics/metrics.py` | 12 tests | COMPLETE |
| Report Generator + Run Manifest | `research/.../backtesting/reporting/report.py` | 2 tests | COMPLETE |
| CLI | `research/.../backtesting/cli.py` | 0 tests | COMPLETE |
| Risk Gate Parity | `research/.../backtesting/engine/core.py` | 5 tests | COMPLETE |
| Integration Tests | `research/tests/backtesting/test_integration.py` | 8 tests | COMPLETE |
| Golden Replay Test | `research/tests/backtesting/test_integration.py` | 2 tests | COMPLETE |
| No-Lookahead Test | `research/tests/backtesting/test_integration.py` | 1 test | COMPLETE |
| Database Migration 015 | `database/migrations/015_backtesting_tables.sql` | N/A | COMPLETE |
| 448 tests (v1.2.0 total) | — | 448 | COMPLETE |

## v1.2.0 — Additional Features

| Requirement | Implementation Files | Tests | Status |
|---|---|---|---|
| Signal Cooldown | `realtime/internal/signal/cooldown.go` | Go tests | COMPLETE |
| Signal Delivery | `realtime/internal/signal/delivery.go` | Go tests | COMPLETE |
| Advanced Signal Integration | `realtime/internal/signal/advanced.go` | Go tests | COMPLETE |
| Golden Integration Tests | `realtime/internal/strategy/golden_integration_test.go` | Go tests | COMPLETE |
| NestJS Operations Module | `control/src/modules/operations/` | Backend tests | COMPLETE |
| NestJS Device-Auth Module | `control/src/modules/device-auth/` | Backend tests | COMPLETE |
| NestJS Health Module | `control/src/modules/health/` | Backend tests | COMPLETE |
| NestJS Admin Module | `control/src/modules/admin/` | Backend tests | COMPLETE |
| Correlation ID Interceptor | `control/src/common/interceptors/` | Backend tests | COMPLETE |
| Mail Service | `control/src/common/mail/` | Backend tests | COMPLETE |
| Frontend API Client | `frontend/src/lib/api.ts` | Frontend tests | COMPLETE |
| Frontend Auth | `frontend/src/lib/auth.ts` | Frontend tests | COMPLETE |
| Frontend Middleware | `frontend/src/middleware.ts` | Frontend tests | COMPLETE |
| Frontend Trading Components | `frontend/src/components/trading/` (6 components) | Frontend tests | COMPLETE |
| Master Node MT4 EA | `mql/mt4/PredictATrade_MasterNode_MT4.mq4` | N/A | COMPLETE |
| Master Node MT5 EA | `mql/mt5/PredictATrade_MasterNode_MT5.mq5` | N/A | COMPLETE |
| Database Migrations 006-011 | `database/migrations/006-011` | N/A | COMPLETE |
| Environment Files | `infra/env/` (6 files) | N/A | COMPLETE |
| Nginx Configs | `infra/nginx/` (5 sites) | N/A | COMPLETE |
| Systemd Units | `infra/systemd/` (4 units) | N/A | COMPLETE |
| Windows Validation Scripts | `scripts/windows/` (7 scripts) | N/A | COMPLETE |
| Backup Scripts | `scripts/backup/` | N/A | COMPLETE |

## Summary

| Metric | Value |
|--------|-------|
| Total SOW requirements traced | 65+ |
| Complete | 60+ |
| Partial | 2 (Organizations/Tenants schema-only, Macro Intelligence partial) |
| Blocked | 0 |
| Total tests | 448 (243 Go + 98 Python + 68 NestJS + 39 Frontend) |
| Database migrations | 15 |
| Database tables | 165 |
| Frontend pages | 40+ |
| API endpoints | 60+ |
| PTB modules | 20+ |
| Advanced risk modules | 7 |
| Backtesting modules | 27 |

## v1.4.0 — Color Palette + Signal Delivery + TP/SL Geometry Fix

| SOW Section | Requirement | Implementation Files | API Endpoints | Frontend Routes | Tests | Status |
|---|---|---|---|---|---|---|
| UI/Command Center | Approved color palette | `frontend/src/styles/globals.css`, `frontend/tailwind.config.ts` | N/A | All pages | TypeScript compile | COMPLETE |
| UI/Command Center | Semantic trading color tokens | `frontend/src/app/**/*.tsx` (80+ replacements) | N/A | All pages | TypeScript compile | COMPLETE |
| Real-Time Trading Plane | Signal delivery to Windows Agents | `realtime/internal/gateway/agent_ws.go`, `realtime/cmd/realtime-engine/main.go` | N/A | N/A | Go build + tests | COMPLETE |
| Strategy | TP/SL ATR-based geometry | `realtime/internal/strategy/strategies.go` | N/A | N/A | Go build + tests | COMPLETE |
| Strategy | Minimum SL distance enforcement | `realtime/internal/strategy/strategies.go` | N/A | N/A | Go build + tests | COMPLETE |
| Windows/MQL Edge | MQL EA v1.05 strategy selection | `mql/mt4/PredictATrade_MT4.mq4`, `mql/mt5/PredictATrade_MT5.mq5` | N/A | N/A | N/A | COMPLETE |
| Windows/MQL Edge | MQL EA v1.05 direction filter | `mql/mt4/PredictATrade_MT4.mq4`, `mql/mt5/PredictATrade_MT5.mq5` | N/A | N/A | N/A | COMPLETE |
| Observability | Regime diagnostics route | nginx config, Go engine | `GET /api/v1/admin/regime-diagnostics` | `/admin/regime-diagnostics` | curl test | COMPLETE |
| Entitlement | License/entitlement gate hydration | `realtime/cmd/realtime-engine/main.go` | N/A | N/A | Go build + tests | COMPLETE |
| Market-Data Truth | Session gate overlap fix | `realtime/internal/features/session.go` | N/A | N/A | Go test | COMPLETE |
| API/Events | Canonical idempotency handling | `realtime/internal/marketdata/persistence.go` | N/A | N/A | Go build | COMPLETE |

### v1.4.0 Summary

| Metric | Value |
|--------|-------|
| Total tests | 490 (243 Go + 98 Python + 68 NestJS + 39 Frontend + 42 additional) |
| New migrations | 0 |
| New API endpoints | 1 (nginx proxy: `/api/v1/admin/regime-diagnostics`) |
| New frontend pages | 0 |
| Files changed | 29 (10 docs + 3 Go + 2 MQL + 1 CSS + 1 tailwind config + 12 TSX/frontend) |


---

## v1.10.0–v1.10.1 — News Breakout + OCO + Economic Calendar + Notifications + Cross-Check

| SOW Section | Requirement | Implementation Files | Tests | Migrations | Status | Evidence |
|---|---|---|---|---|---|---|
| 14 | News Blackout Engine (fail-safe) | `realtime/pkg/news/risk_engine.go`, `realtime/pkg/news/provider.go`, `realtime/pkg/news/fmp_provider.go` | 12 tests | 022 (economic_events, news_provider_health, news_risk_decisions) | **WIRED** (v1.10.1) | RiskEngine wired into session engine via NewsRiskProvider interface; fail-safe adapter returns NONE when disabled, DATA_UNAVAILABLE when provider fails |
| 14 | NewsGate EXTREME/DATA_UNAVAILABLE | `realtime/internal/gates/implementations.go` | gates_test.go | N/A | **FIXED** (v1.10.1) | Now blocks on HIGH, EXTREME, DATA_UNAVAILABLE, BLOCKED per ShouldBlock() |
| SOW News | News Breakout Engine | `realtime/internal/breakout/breakout.go` | 11 tests | 022 (breakout_plans) | IMPLEMENTED (OFF) | Disabled by default; 15+ eligibility gates; all risk gates enforced |
| SOW News | OCO State Machine | `realtime/internal/oco/group.go` | 11 tests | 022 (oco_groups) | IMPLEMENTED (OFF) | 11-state machine; idempotent trigger; race reconciliation; broker restart handling |
| SOW Notifications | External Notification Adapters | `realtime/pkg/notifications/` (email, telegram, whatsapp, push) | 12 tests | 022 (notification_deliveries) | SOFTWARE_READY | Async queue, retry, dead-letter; missing creds = NOT_CONFIGURED; disabled by default |
| NestJS | Admin Health (Valkey fix) | `control/src/modules/admin/admin.service.ts` | admin.service.spec.ts | N/A | **FIXED** (v1.10.1) | Valkey TCP check no longer nested inside Go catch block; defensive defaults in commissionSummary/payoutStats |
| DB | Migration 022 applied | `database/migrations/022_news_breakout_oco_notifications.sql` | N/A | 022 | **APPLIED** (v1.10.1) | 6 new tables + 2 additive columns created in production DB |
| DB | Migration history tracking | `audit.migration_history` | N/A | N/A | **CREATED** (v1.10.1) | 25 migrations recorded as COMPLETED; migrate.sh works correctly |

### v1.10.1 Cross-Check Summary

| Metric | Value |
|--------|-------|
| Total tests | 333 (29 Go packages + 127 Python + 107 NestJS + 70 Frontend) |
| Bugs found & fixed | 7 (brace bug, defensive defaults, mock fix, NewsGate, RiskEngine wiring, migration 022, migration history) |
| New migrations applied | 1 (022) |
| Services restarted | 2 (realtime-engine, control) |
| Audit result | PASS — 0 failed, 0 warned |

## v1.11.0–v1.15.0 — 25 August 2026 — SL Enforcement + Legal + CI/CD + Dashboard Polish

### Server-Side SL Enforcement (v1.15.0)

| # | SOW Requirement | Implementation Files | Tests | Migrations | API/UI | Observability | Status | Evidence |
|---|----------------|---------------------|-------|------------|--------|---------------|--------|----------|
| 1 | EXECUTION_ACK SL verification | `realtime/internal/marketdata/agent_provider.go`, `realtime/cmd/realtime-engine/main.go` | 30/30 Go packages pass | N/A | N/A | observability.Log + audit.client_events | **PASS** | Server verifies SL>0 + SL matches server value (±0.5pts) |
| 2 | Position SL monitoring | `realtime/cmd/realtime-engine/main.go` (checkPositionSLs) | Go tests pass | N/A | N/A | Log + audit.client_events | **PASS** | Scans broker snapshot for PAT positions with SL=0 |
| 3 | CLOSE_POSITION command | `realtime/internal/gateway/agent_ws.go`, `windows-agent/internal/agent.go`, `mql/mt5/PredictATrade_MT5.mq5`, `mql/mt4/PredictATrade_MT4.mq4` | Go + Windows Agent build pass | N/A | WS command | Log | **PASS** | Server→Agent→EA closes position by ticket/magic |
| 4 | EMERGENCY_STOP command | Same as #3 | Same | N/A | WS command | Log | **PASS** | Server→Agent→EA closes ALL + halts |
| 5 | KILL_SWITCH command | Same as #3 | Same | N/A | WS command | Log | **PASS** | Server→Agent→EA closes all + ExpertRemove + disconnect |
| 6 | Agent suspension (3-strike) | `realtime/cmd/realtime-engine/main.go` (recordSLViolation) | Go tests pass | N/A | N/A | audit.client_events | **PASS** | 3 violations → DisconnectAgent, other agents unaffected |
| 7 | MQL EA v1.09 command handlers | `mql/mt5/PredictATrade_MT5.mq5`, `mql/mt4/PredictATrade_MT4.mq4` | N/A (MQL) | N/A | EA IPC | Print logs | **PASS** | HandleClosePosition, HandleEmergencyStop, HandleKillSwitch |
| 8 | Position SL in snapshot | `mql/mt5/PredictATrade_MT5.mq5` (PAT_BuildPositionDetails), `mql/mt4/PredictATrade_MT4.mq4` | N/A | N/A | MARKET_SNAPSHOT JSON | N/A | **PASS** | Per-position ticket/magic/SL/TP/volume in snapshot |
| 9 | Signal delivery NOT blocked | `realtime/cmd/realtime-engine/main.go` (broadcastSignalToAll) | Go tests pass | N/A | N/A | N/A | **PASS** | isAgentSuspended removed from broadcast path |

### DXY→macroHealth Fix (v1.14.0)

| # | SOW Requirement | Implementation Files | Tests | Migrations | API/UI | Observability | Status | Evidence |
|---|----------------|---------------------|-------|------------|--------|---------------|--------|----------|
| 10 | DXY health wiring | `realtime/cmd/realtime-engine/main.go` (StartRefreshLoop callback) | Go tests pass | N/A | N/A | ML/Sentiment enabled | **PASS** | macroHealth.OnDXYFetchSuccess() called on DXY refresh |
| 11 | Calibration DB tables | `database/migrations/072_calibration_tables.sql` | N/A | 072 | N/A | N/A | **PASS** | calibration.model_versions, predictions, outcomes |

### Legal Compliance (v1.12.0)

| # | SOW Requirement | Implementation Files | Tests | Migrations | API/UI | Observability | Status | Evidence |
|---|----------------|---------------------|-------|------------|--------|---------------|--------|----------|
| 12 | Terms of Service | `frontend/src/app/terms/page.tsx` | N/A | N/A | /terms page | N/A | **PASS** | 18 sections, market-standard |
| 13 | Privacy Policy | `frontend/src/app/privacy/page.tsx` | N/A | N/A | /privacy page | N/A | **PASS** | 16 sections, PDPL/GDPR compliant |
| 14 | Data Processing Agreement | `frontend/src/app/data-processing-agreement/page.tsx` | N/A | N/A | /data-processing-agreement page | N/A | **PASS** | 14 sections, technical+organizational measures |
| 15 | Consent tracking (backend) | `control/src/modules/auth/dto/auth.dto.ts`, `control/src/modules/auth/auth.service.ts` | NestJS tests pass | 071 | POST /auth/register | audit.client_events | **PASS** | 6 consent fields validated + logged |
| 16 | Signup form with consents | `frontend/src/app/(auth)/register/page.tsx` | Frontend lint pass | N/A | /register page | N/A | **PASS** | 3 required + 3 optional checkboxes |
| 17 | Login form improvement | `frontend/src/app/(auth)/login/page.tsx` | Frontend lint pass | N/A | /login page | N/A | **PASS** | Remember me, icons, success banner |

### CI/CD (v1.13.0)

| # | SOW Requirement | Implementation Files | Tests | Migrations | API/UI | Observability | Status | Evidence |
|---|----------------|---------------------|-------|------------|--------|---------------|--------|----------|
| 18 | All 6 CI jobs pass | `.github/workflows/ci.yml` | 6/6 GitHub Actions green | N/A | N/A | CI dashboard | **PASS** | Go, NestJS, Frontend, Python, Windows Agent, Security |
| 19 | Go test race fix | `realtime/pkg/notifications/notifications_test.go` | go test -race pass | N/A | N/A | N/A | **PASS** | sync.Mutex on mockProvider.sendCount |
| 20 | Go config test fix | `realtime/internal/config/config_capital_test.go` | go test pass (no DBURL) | N/A | N/A | N/A | **PASS** | helperDefault() with dummy DBURL |
| 21 | Frontend peer-dep fix | `frontend/package.json`, `frontend/package-lock.json` | npm ci pass | N/A | N/A | N/A | **PASS** | @testing-library/react v16 (React 19) |
| 22 | Security scan precision | `.github/workflows/ci.yml` | Scan pass (no FP) | N/A | N/A | N/A | **PASS** | Precise regex for actual secrets only |

### Signal Engine Audit (v1.14.0)

| # | SOW Requirement | Implementation Files | Tests | Migrations | API/UI | Observability | Status | Evidence |
|---|----------------|---------------------|-------|------------|--------|---------------|--------|----------|
| 23 | 5 strategy engines verified | `realtime/internal/strategy/`, `realtime/cmd/realtime-engine/main.go` | 30/30 Go packages | N/A | /api/v1/engines/status | Prometheus metrics | **PASS** | All 5 LIVE after timeframe closes |
| 24 | 13 indicator pillars | `realtime/internal/features/`, `realtime/internal/strategy/` | Go tests pass | N/A | Signal evidence JSON | N/A | **PASS** | EMA, ADX, VWAP, MACD, OsMA, RSI, Stoch, CCI, MTF, SMC, cross-market |
| 25 | ML/Sentiment re-enabled | `realtime/cmd/realtime-engine/main.go` | Go tests pass | N/A | N/A | Ollama connected | **PASS** | ML weight 0.15 + sentiment 0.05 active |
