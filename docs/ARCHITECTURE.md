# Predict-A-Trade v1.0.0 — Architecture

## Four-Plane Architecture (SOW Section 3)

### 1. Go — Real-Time Trading Plane (`/realtime`)
Authoritative market-data, feature, strategy, signal, hard-risk, execution-authorization,
realtime delivery and reconciliation path. No synchronous billing/referral/commission
dependency in tick-to-signal processing.

**Key packages:**
- `internal/types` — Canonical type definitions
- `internal/gates` — Non-blocking fail-closed gate architecture (SOW Section 131)
- `internal/strategy` — Four distinct strategy confluence engines (SOW Section 12A-12F)
- `internal/signal` — Signal lifecycle and master decision hierarchy (SOW Section 24)
- `pkg/math` — Canonical quantitative math (SOW Section 134)

### 2. Python — Intelligence/Research Plane (`/research`)
Datasets, research, backtesting, walk-forward/OOS, calibration, ML/NLP, feature studies,
drift and validation. Python must not become a mandatory dependency for every live tick.

**Key modules:**
- `reference_math.py` — Canonical math matching Go implementations (SOW Section 137)
- Tests verify Go/Python parity for all critical formulas

### 3. NestJS — SaaS/Control Plane (`/control`)
IAM/MFA/RBAC, tenants, users, subscriptions, billing/webhooks, entitlements, licensing,
devices, MT accounts, referrals, commissions, payouts, audit, config and admin operations.

**Key modules:**
- `commissions/commission-engine.ts` — Critical commission engine with L1-only second-payment rule
- `auth/auth.service.ts` — JWT authentication with bcrypt password hashing
- `referrals/referrals.service.ts` — Five-level sponsor tree with cycle prevention

### 4. Next.js — Presentation Plane (`/frontend`)
Public site, user portal, admin operations console, XAUUSD Live Command Center,
charting, licensing/downloads and growth/financial UI.

**Key features:**
- Design token system with dark/light themes (SOW Section 186)
- Server-authoritative rendering — no fabricated data
- Collapsible sidebar navigation (SOW Section 189)
- User trading dashboard (SOW Section 192)
- Admin operations dashboard (SOW Section 193)

## Database (SOW Section 55)
PostgreSQL 17 with logical schemas: iam, control, licensing, billing, referral, finance,
trading, market, research, audit, support. TimescaleDB hypertables for ticks, candles,
market states, flow features. pgvector for research embeddings. PgBouncer for pooling.
Valkey for hot state.

## Windows/MQL Edge (SOW Section 44-54)
Go Windows Agent + MQL4/MQL5 EAs remain lightweight execution adapters/guards.
No primary predictive intelligence or server/private signing credentials in EAs.

## Non-Regression Authority
Hard safety, security, financial-integrity, broker, data-quality, entitlement, compliance,
execution and risk controls always win over convenience or presentation requirements.
