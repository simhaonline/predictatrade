# Predict-A-Trade XAUUSD — Scope of Work
## v1.16.0 — Consolidated (26 August 2026)

### 1. PROJECT IDENTITY
- Product: Predict-A-Trade XAUUSD Signal Generation & Analytics Platform
- Primary instrument: XAUUSD (Gold/US Dollar)
- Architecture: Multi-plane (Go realtime, NestJS control, Next.js frontend, Python research, MQL edge)
- Deployment: Docker Compose (all services containerized)
- Repository: github.com/simhaonline/predictatrade

### 2. PLANE BOUNDARIES (mandatory)

| Plane | Location | Responsibility | Must NOT become |
|-------|----------|----------------|-----------------|
| Go Real-Time | realtime/ | Market data, features, strategies, signals, hard gates, delivery, reconciliation | Synchronous billing/referral dependency |
| NestJS Control | control/ | IAM, MFA, RBAC, subscriptions, billing, entitlements, licensing, devices, referrals, commissions, payouts, admin | Tick-to-signal hot path |
| Next.js Presentation | frontend/ | Public site, user dashboard, admin console, charts, commercial UI | Authority for risk, entitlement, finance or probability |
| Python Research | research/ | Historical data, backtesting, validation, calibration, ML/RL research | Mandatory live-tick dependency |
| Windows/MQL Edge | windows-agent/, mql/ | Broker adapter, heartbeat, signed signal handling, execution guards | Primary intelligence or server credentials |

### 3. STRATEGY ENGINES (5 engines, v1.16.0)

| Engine | ID | Timeframes | Min Score | Expiry | Status |
|--------|----|-----------|:---------:|:------:|:------:|
| Standard Scalping | STANDARD_SCALPING | M1/M5 | 65 | 10 min | LIVE |
| Ultra Scalping | ULTRA_SCALPING | M1 | 60 | 5 min | LIVE |
| Standard Swing | STANDARD_SWING | M15/H1 | 68 | 30 min | LIVE |
| Trend Swing | TREND_SWING | H1/H4 | 70 | 60 min | LIVE |
| MARNIE_FIB | MARNIE_FIB | H1 | 70 | 60 min | SHADOW |

### 4. EVIDENCE SCORING ARCHITECTURE

13 evidence pillars with family caps:
TREND(0.35), MOMENTUM(0.30), STRUCTURE(0.25), LIQUIDITY(0.20),
SMC(0.20), MTF(0.20), CANDLE(0.20), REGIME(0.15), VWAP(0.15),
VOLATILITY(0.15), ML(0.25), SENTIMENT(0.25), SESSION_ORB(0.15)

42 technical indicators, 35 LIVE, 7 warming up.

### 5. RISK GATES (16 gates, ordered)

1. EXECUTION_PERMISSION → 2. BROKER_SYMBOL → 3-4. CAPITAL_PROTECTION
5. SPREAD → 6. NEWS → 7. SLIPPAGE → 8-9. POSITION_LIMIT
10. COOLDOWN → 11. STOP_HUNT → 12. MARGIN → 13-14. OVERTRADE
15. REGIME → 16. PROFIT_TARGET

### 6. P2 FEATURES (ACTIVE, v1.16.0)

- P2-001: Session ORB — Asian/London/NY opening ranges, breakout detection
- P2-002: Pin Bar geometry — body/wick ratios, rejection direction, quality scoring
- P2-003: Pullback detection — depth %, ATR-normalized retracement, continuation confirmation
- P2-004: Trade Group ID — multi-position signal split tracking
- P2-005: SLO targets — availability, latency, error budgets documented

### 7. CURRENT PRODUCTION STATUS

| Check | Status |
|-------|:------:|
| Go tests (28 packages) | PASS |
| Frontend tests (70) | PASS |
| Python tests (127) | PASS |
| TypeScript check | PASS |
| 16 risk gates wired | PASS |
| SL enforcement server-side | ACTIVE |
| Broker symbol validation | ACTIVE (P0) |
| Price precision rounding | ACTIVE (P1) |
| MQL EAs compiled | ⚠ Operator action |
| Production API keys | ⚠ Operator action |
| Backup/restore tested | ⚠ Not verified |

### 8. EXTERNAL DEPENDENCIES

| Provider | Purpose | Required? | Status |
|----------|---------|:---------:|:------:|
| TwelveData | XAUUSD candles + DXY | YES | LIVE |
| FMP | COT + macro data | YES | LIVE |
| Ollama (local) | AI sentiment | NO | LIVE |
| MT5 Broker | Live trading | CONDITIONAL | Configured |
| FRED | Real yields | NO | Opt-in |
| NOWPayments | Crypto payments | CONDITIONAL | Configured |
| Stripe | Card payments | CONDITIONAL | Configured |

### 9. VERSION HISTORY

| Version | Date | Key Change |
|---------|------|------------|
| v1.16.0 | 2026-08-26 | P2 features active, production readiness audit |
| v1.15.0 | 2026-08-25 | Server-side SL enforcement, legal compliance |
| v1.14.0 | 2026-08-25 | DXY macro health fix, calibration DB |
| v1.13.0 | 2026-08-25 | CI/CD all 6 jobs passing |
| v1.12.0 | 2026-08-25 | Legal compliance, consent checkboxes |
| v1.11.0 | 2026-08-24 | Live dashboard neural shell |