# Frontend Route / API Matrix

## Auth Routes

| Route | Role | Menu | REST API | Mutations | WebSocket | Permission | Loading | Empty | Error | Status |
|-------|------|------|----------|-----------|-----------|------------|---------|-------|-------|--------|
| /login | Public | N/A | POST /auth/login | None | None | None | Spinner | N/A | Error msg | ✅ |
| /register | Public | N/A | POST /auth/register | None | None | None | Spinner | N/A | Error msg | ✅ |
| /forgot-password | Public | N/A | POST /auth/forgot | None | None | None | Spinner | Success msg | Error msg | ✅ |
| /reset-password | Public | N/A | POST /auth/reset | None | None | None | Spinner | Success msg | Error msg | ✅ |
| /verify-otp | Public | N/A | POST /auth/verify-otp | None | None | None | Spinner | N/A | Error msg | ✅ |

## Admin Routes

| Route | Role | Menu | REST API | Mutations | WebSocket | Permission | Loading | Empty | Error | Status |
|-------|------|------|----------|-----------|-----------|------------|---------|-------|-------|--------|
| /admin/dashboard | Admin | Live Dashboard | GET /admin/overview, GET /admin/health | None | WS /ws/v1 | ADMIN | Skeletons | N/A | Retry btn | ✅ |
| /admin/signals | Admin | Signals | GET /api/v1/signals (Go) | None | WS /ws/v1 | ADMIN | DataTable loading | Empty state | Retry btn | ✅ |
| /admin/indicators | Admin | Indicators | GET /api/v1/market/snapshot (Go) | None | None | ADMIN | Loading text | Inventory shown | Retry btn | ✅ |
| /admin/strategies | Admin | Strategies | GET /operations/state, POST /operations/strategy/:id/enable, POST /operations/strategy/:id/disable | Enable/Disable | None | ADMIN | Loading text | N/A | Error msg | ✅ |
| /admin/scoring-board | Admin | Scoring Board | GET /api/v1/market/state (Go) | None | None | ADMIN | Loading | N/A | Retry btn | ✅ |
| /admin/activations | Admin | Activations | GET /devices/sessions, POST /devices/devices/:id/revoke | Revoke | None | ADMIN | DataTable loading | Empty state | Retry btn | ✅ |
| /admin/licenses | Admin | Licenses | GET /admin/licenses | None | None | ADMIN | DataTable loading | Empty state | Retry btn | ✅ |
| /admin/users | Admin | Users | GET /admin/users, PATCH /admin/users/:id/status | Suspend/Activate | None | ADMIN | DataTable loading | Empty state | Retry btn | ✅ |
| /admin/subscriptions | Admin | Subscriptions | GET /admin/subscriptions | None | None | ADMIN | DataTable loading | Empty state | Retry btn | ✅ |
| /admin/billing | Admin | Billing | GET /admin/subscriptions, GET /admin/commissions, GET /admin/payouts, POST /payouts/:id/approve | Approve payout | None | ADMIN | DataTable loading | Empty state | Retry btn | ✅ |
| /admin/referrals | Admin | Referrals | GET /admin/commissions, GET /admin/commissions/summary, GET /admin/payouts, GET /admin/payouts/stats | None | None | ADMIN | DataTable loading | Empty state | Retry btn | ✅ |
| /admin/device-auth | Admin | Device Auth | GET /admin/devices | None | None | ADMIN | DataTable loading | Empty state | Retry btn | ✅ |
| /admin/trading-reports | Admin | Trading Reports | GET /admin/overview, GET /api/v1/agents/status (Go) | None | None | ADMIN | Loading | N/A | N/A | ✅ |
| /admin/backtesting | Admin | Backtesting | GET /api/v1/market/state (Go) | None | None | ADMIN | Loading | N/A | N/A | ✅ |
| /admin/logs | Admin | Logs | GET /audit | None | None | ADMIN | DataTable loading | Empty state | Retry btn | ✅ |
| /admin/operations | Admin | Operations | GET /operations/state, GET /operations/active, POST /operations/* | Halt/Resume/Pause | None | ADMIN | Loading | N/A | Error msg | ✅ |
| /admin/health | Admin | Health | GET /admin/health, GET /health (NestJS) | None | None | ADMIN | Loading | N/A | Retry btn | ✅ |
| /admin/settings | Admin | Settings | (none) | None | None | ADMIN | N/A | N/A | N/A | ✅ |
| /admin/settings/accessibility | Admin | (sub) | (none) | None | None | ADMIN | N/A | N/A | N/A | ✅ |

## User Routes

| Route | Role | Menu | REST API | Mutations | WebSocket | Permission | Loading | Empty | Error | Status |
|-------|------|------|----------|-----------|-----------|------------|---------|-------|-------|--------|
| /dashboard/live | User | Live Dashboard | (none) | None | WS /ws/v1 | USER | Loading | No signals msg | N/A | ✅ |
| /dashboard/signals | User | Signals | GET /api/v1/signals (Go) | None | WS /ws/v1 | USER | DataTable loading | Empty state | Retry btn | ✅ |
| /dashboard/mt4-mt5-client | User | MT4/MT5 Client | GET /licensing/devices, GET /licensing/mt-accounts | None | None | USER | Loading | Empty state | N/A | ✅ |
| /dashboard/strategies | User | Strategies | GET /plans, GET /subscriptions, POST /subscriptions | Subscribe | None | USER | Loading | No plans msg | Error msg | ✅ |
| /dashboard/trading-reports | User | Trading Reports | GET /subscriptions, GET /commissions/summary, GET /billing/invoices | None | None | USER | Loading | N/A | N/A | ✅ |
| /dashboard/backtest | User | Backtest | (none, static info) | None | None | USER | N/A | N/A | N/A | ✅ |
| /dashboard/referrals | User | Referrals | GET /commissions, GET /commissions/summary | None | None | USER | DataTable loading | Empty state | Retry btn | ✅ |
| /dashboard/billing | User | Billing | GET /billing/invoices | None | None | USER | DataTable loading | Empty state | Retry btn | ✅ |
| /dashboard/settings | User | Settings | (none) | None | None | USER | N/A | N/A | N/A | ✅ |
| /dashboard/settings/accessibility | User | (sub) | (none) | None | None | USER | N/A | N/A | N/A | ✅ |

## Error Routes

| Route | Description | Status |
|-------|-------------|--------|
| /not-found (404) | Page not found | ✅ |
| /forbidden (403) | Access denied | ✅ |
| error.tsx | Route-level error boundary | ✅ |
| global-error.tsx | Global error boundary | ✅ |
