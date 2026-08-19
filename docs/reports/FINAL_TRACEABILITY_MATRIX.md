# Predict-A-Trade — Final Traceability Matrix

| SOW Section | Requirement | Implementation Files | API Endpoints | Frontend Routes | Tests | Status |
|---|---|---|---|---|---|---|
| 3.1 | Go Real-Time Trading Plane | `realtime/cmd/`, `realtime/internal/` | Go HTTP/WS gateway | N/A | Go tests (gates, strategy, math) | COMPLETE |
| 3.2 | Python Research Plane | `research/src/patresearch/` | N/A | N/A | Python tests (16) | COMPLETE |
| 3.3 | NestJS Control Plane | `control/src/` | All `/api/v1/*` routes | N/A | Backend tests (37) | COMPLETE |
| 3.4 | Next.js Presentation Plane | `frontend/src/` | N/A | All routes | Frontend tests (26) | COMPLETE |
| 5 | NestJS Module List | `control/src/modules/` | Auth, Users, Plans, Subscriptions, Billing, Referrals, Commissions, Payouts, Licensing, Audit, Health, Admin | N/A | Backend tests | COMPLETE |
| 6 | Market Data Architecture | `realtime/internal/marketdata/` | Go HTTP API | Chart, Market Pulse | Go tests | COMPLETE |
| 10 | Multi-Timeframe Engine | `realtime/internal/features/mtf.go` | Go HTTP API | Dashboard | Go tests | COMPLETE |
| 11 | Market Regime Engine | `realtime/internal/features/regime.go` | Go HTTP API | Dashboard | Go tests | COMPLETE |
| 12 | Quantitative Market Engines | `realtime/internal/features/` | Go HTTP API | Dashboard | Go tests | COMPLETE |
| 12A | Four Strategy Playbooks | `realtime/internal/strategy/strategies.go` | Go HTTP API | N/A | Go strategy tests | COMPLETE |
| 12C | Strategy Confluence Engine | `realtime/internal/strategy/confluence.go` | Go HTTP API | N/A | Go confluence tests | COMPLETE |
| 13 | Macro Intelligence | `realtime/internal/features/` (partial) | Go HTTP API | Dashboard | Go tests | PARTIAL |
| 14 | News Blackout Engine | `realtime/internal/gates/` | Go HTTP API | N/A | Go gate tests | COMPLETE |
| 15 | Prediction Contract | `realtime/internal/signal/engine.go` | Go HTTP API | Signals | Go tests | COMPLETE |
| 16-17 | Signal Scoring & Grades | `realtime/internal/signal/engine.go` | Go HTTP API | Signals | Go tests | COMPLETE |
| 19 | Signal Lifecycle | `realtime/internal/signal/engine.go` | Go HTTP API | Signals, Positions | Go tests | COMPLETE |
| 25 | Risk Engine | `realtime/internal/gates/` | Go HTTP API | N/A | Go gate tests | COMPLETE |
| 25B | Broker-Aware Price Units | `realtime/internal/types/types.go` | Go HTTP API | N/A | Go tests | COMPLETE |
| 34 | IAM | `control/src/modules/auth/`, `control/src/modules/users/`, migrations 002, 007 | `/auth/*`, `/users/*`, `/admin/users` | Login, Register, Security | Backend auth tests | COMPLETE |
| 35 | RBAC | `control/src/common/guards/`, migrations 002 | AdminGuard on admin endpoints | Admin routes | Backend tests | COMPLETE |
| 36 | Organizations/Tenants | migration 002 (schema exists) | N/A | N/A | N/A | PARTIAL (schema only) |
| 61 | API Credentials | migration 002 (schema exists) | N/A | N/A | N/A | PARTIAL (schema only) |
| 62-68 | Plans & Pricing | `control/src/modules/plans/` | `/plans`, `/admin/plans` | Subscription | Backend tests | COMPLETE |
| 63-68 | Subscriptions | `control/src/modules/subscriptions/` | `/subscriptions`, `/admin/subscriptions` | Subscription | Backend tests | COMPLETE |
| 69 | Commission Engine | `control/src/modules/commissions/commission-engine.ts` | `/commissions`, `/admin/commissions` | Referral | Commission engine tests (16) | COMPLETE |
| 69 | Referral System | `control/src/modules/referrals/` | `/referrals/*` | Referral | Backend tests | COMPLETE |
| 69 | Payouts | `control/src/modules/payouts/` | `/payouts/*`, `/admin/payouts` | Payouts | Backend tests | COMPLETE |
| 72 | Session Security | `control/src/modules/auth/auth.service.ts`, migrations 006, 007 | `/auth/refresh`, `/auth/logout` | Login, Security | Backend auth tests (21) | COMPLETE |
| 80 | Security Headers | `control/src/main.ts`, `infra/nginx/` | All endpoints | All routes | N/A | COMPLETE |
| 82 | Observability | `realtime/internal/observability/`, `infra/prometheus/` | `/health` | N/A | N/A | COMPLETE |
| 131 | Windows Agent | `windows-agent/` | Agent WS | MT Setup | Go build | COMPLETE |
| 132 | MT4/MT5 EAs | `mql/mt4/`, `mql/mt5/` | N/A | N/A | N/A | COMPLETE (code) |
| 152-184 | Admin Dashboard | `frontend/src/app/(admin)/` | `/admin/*` (12 endpoints) | Admin | Frontend tests | COMPLETE |
| 185-200 | Visual System | `frontend/src/styles/`, design tokens | N/A | All auth/dashboard | Frontend build | COMPLETE |
| - | Domain Routing | `infra/nginx/`, `frontend/.env.local` | All domains | All routes | N/A | COMPLETE |
| - | CI/CD | `.github/workflows/ci.yml` | N/A | N/A | CI config | COMPLETE |
| - | Docker Compose | `docker-compose.yml` | N/A | N/A | N/A | COMPLETE |
| - | Systemd Services | `infra/systemd/` | N/A | N/A | N/A | COMPLETE |
