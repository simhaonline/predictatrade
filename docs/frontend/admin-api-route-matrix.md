# Admin API Route Matrix

| Route | Feature | GET APIs | Mutations | WS | Required Role | Status |
|-------|---------|----------|-----------|-----|-------------|--------|
| /admin/dashboard | Live Dashboard | /admin/overview, /operations/state, /api/v1/health, /api/v1/market/state, /api/v1/agents/status, /health | — | /ws (signals, market, agent) | ADMIN | ✅ |
| /admin/signals | Signal Panel | /api/v1/signals (Go) | — | /ws (signals) | ADMIN | ✅ |
| /admin/indicators | Indicator Panel | /api/v1/market/snapshot (Go) | — | — | ADMIN | ✅ |
| /admin/strategies | Strategy Panel | /operations/state | POST /operations/strategy/:id/enable, /disable | — | ADMIN | ✅ |
| /admin/scoring-board | Scoring Board | /api/v1/market/state (Go) | — | — | ADMIN | ✅ |
| /admin/activations | Activations | /devices/sessions | POST /devices/devices/:id/revoke | — | ADMIN | ✅ |
| /admin/licenses | License Mgmt | /admin/licenses | — | — | ADMIN | ✅ |
| /admin/users | User Onboarding | /admin/users | PATCH /admin/users/:id/status | — | ADMIN | ✅ |
| /admin/subscriptions | Subscription Mgmt | /admin/subscriptions | — | — | ADMIN | ✅ |
| /admin/billing | Billing & Payouts | /admin/subscriptions, /admin/commissions, /admin/payouts | POST /payouts/:id/approve | — | ADMIN | ✅ |
| /admin/referrals | Referral & Commissions | /admin/commissions, /admin/commissions/summary, /admin/payouts, /admin/payouts/stats | — | — | ADMIN | ✅ |
| /admin/device-auth | Device Auth | /admin/devices | — | — | ADMIN | ✅ |
| /admin/trading-reports | Trading Reports | /admin/overview, /api/v1/agents/status | — | — | ADMIN | ✅ |
| /admin/backtesting | Backtesting | /api/v1/market/state (framework info) | — | — | ADMIN | ✅ |
| /admin/logs | Logs & Audit | /audit | — | — | ADMIN | ✅ |
| /admin/operations | Platform Operations | /operations/state, /operations/active, /operations/ai/models | POST /operations/halt-trading, /resume-trading, /pause-signals, /resume-signals, /strategy/:id/* | — | ADMIN | ✅ |
| /admin/health | System Health | /admin/health, /health, /api/v1/health | — | — | ADMIN | ✅ |
| /admin/settings | Settings | /users/me | PATCH /users/me, POST /auth/mfa/* | — | ADMIN | ✅ |
