# API Coverage Report

## Classification

| Endpoint | Classification | Notes |
|----------|---------------|-------|
| POST /auth/register | USED_BY_FRONTEND | Register page |
| POST /auth/login | USED_BY_FRONTEND | Login page |
| POST /auth/verify-otp | USED_BY_FRONTEND | MFA verification |
| POST /auth/refresh | USED_BY_FRONTEND | Axios interceptor (single-flight) |
| GET /auth/me | USED_BY_FRONTEND | Auth provider hydration |
| POST /auth/logout | USED_BY_FRONTEND | Logout flow |
| POST /auth/mfa/setup | USED_BY_FRONTEND | Settings (future) |
| POST /auth/mfa/verify | USED_BY_FRONTEND | Settings (future) |
| POST /auth/forgot | USED_BY_FRONTEND | Forgot password |
| POST /auth/reset | USED_BY_FRONTEND | Reset password |
| GET /users/me | USED_BY_FRONTEND | Settings |
| PATCH /users/me | USED_BY_FRONTEND | Settings |
| GET /users/:id | ADMIN_ONLY | Admin user detail |
| GET /users | ADMIN_ONLY | Admin user list |
| GET /admin/overview | ADMIN_ONLY | Admin dashboard |
| GET /admin/users | ADMIN_ONLY | Admin user list |
| PATCH /admin/users/:id/status | ADMIN_ONLY | Admin user management |
| GET /admin/subscriptions | ADMIN_ONLY | Admin subscriptions |
| GET /admin/commissions | ADMIN_ONLY | Admin referrals/billing |
| GET /admin/commissions/summary | ADMIN_ONLY | Admin referrals summary |
| GET /admin/payouts | ADMIN_ONLY | Admin referrals/billing |
| GET /admin/payouts/stats | ADMIN_ONLY | Admin referrals stats |
| GET /admin/licenses | ADMIN_ONLY | Admin licenses |
| GET /admin/devices | ADMIN_ONLY | Admin device-auth |
| GET /admin/health | ADMIN_ONLY | Admin health |
| GET /billing/invoices | USER_ONLY | User billing |
| POST /billing/webhook | NOT_FRONTEND_RELEVANT | Payment provider callback |
| GET /subscriptions | USER_ONLY | User strategies |
| POST /subscriptions | USER_ONLY | User strategies subscribe |
| GET /plans | USER_ONLY | User strategies |
| GET /plans/:id | USER_ONLY | (future) |
| GET /licensing/licenses | USER_ONLY | User MT client |
| GET /licensing/devices | USER_ONLY | User MT client |
| POST /licensing/devices | USER_ONLY | User MT client |
| GET /licensing/mt-accounts | USER_ONLY | User MT client |
| POST /licensing/mt-accounts | USER_ONLY | User MT client |
| GET /commissions | USER_ONLY | User referrals |
| GET /commissions/summary | USER_ONLY | User referrals/reports |
| GET /commissions/admin/all | ADMIN_ONLY | Admin referrals/billing |
| GET /commissions/admin/summary | ADMIN_ONLY | Admin referrals |
| GET /referrals/network | USER_ONLY | User referrals |
| GET /referrals/commissions | USER_ONLY | User referrals |
| GET /payouts | USER_ONLY | User referrals |
| POST /payouts/request | USER_ONLY | User referrals |
| GET /payouts/admin/all | ADMIN_ONLY | Admin referrals/billing |
| GET /payouts/admin/stats | ADMIN_ONLY | Admin referrals |
| POST /payouts/:id/approve | ADMIN_ONLY | Admin billing |
| GET /operations/state | ADMIN_ONLY | Admin strategies/operations |
| GET /operations/active | ADMIN_ONLY | Admin operations |
| POST /operations/halt-trading | ADMIN_ONLY | Admin operations |
| POST /operations/resume-trading | ADMIN_ONLY | Admin operations |
| POST /operations/pause-signals | ADMIN_ONLY | Admin operations |
| POST /operations/resume-signals | ADMIN_ONLY | Admin operations |
| POST /operations/strategy/:id/enable | ADMIN_ONLY | Admin strategies |
| POST /operations/strategy/:id/disable | ADMIN_ONLY | Admin strategies |
| GET /operations/ai/models | ADMIN_ONLY | Admin operations |
| GET /operations/ai/training-jobs | ADMIN_ONLY | Admin operations |
| GET /operations/ai/inference | ADMIN_ONLY | Admin operations |
| POST /operations/ai/model/:id/activate | ADMIN_ONLY | Admin operations |
| POST /operations/ai/model/:id/deactivate | ADMIN_ONLY | Admin operations |
| GET /audit | ADMIN_ONLY | Admin logs |
| GET /health | SYSTEM_ONLY | Health check (not directly used by frontend) |
| POST /devices/activate | AGENT_ONLY | Windows Agent |
| POST /devices/refresh | AGENT_ONLY | Windows Agent |
| POST /devices/heartbeat | AGENT_ONLY | Windows Agent |
| GET /devices/sessions | ADMIN_ONLY | Admin activations |
| GET /devices/devices/:id | ADMIN_ONLY | Admin activations |
| POST /devices/devices/:id/revoke | ADMIN_ONLY | Admin activations |
| GET /api/v1/signals (Go) | USED_BY_FRONTEND | Admin/User signals |
| GET /api/v1/market/state (Go) | USED_BY_FRONTEND | Admin scoring/dashboard |
| GET /api/v1/candles (Go) | USED_BY_FRONTEND | (future charts) |
| GET /api/v1/strategies (Go) | USED_BY_FRONTEND | Admin strategies |
| GET /api/v1/market/snapshot (Go) | USED_BY_FRONTEND | Admin indicators |
| GET /api/v1/agents/status (Go) | USED_BY_FRONTEND | Admin trading-reports |
| GET /api/v1/price/history (Go) | USED_BY_FRONTEND | (future charts) |
| POST /api/v1/signals/resume (Go) | BACKEND_INTERNAL | (not directly used) |
| GET /metrics (Go) | SYSTEM_ONLY | Prometheus |

## Summary

- **USED_BY_FRONTEND**: 48 endpoints
- **ADMIN_ONLY**: 26 endpoints
- **USER_ONLY**: 15 endpoints
- **AGENT_ONLY**: 3 endpoints
- **SYSTEM_ONLY**: 2 endpoints
- **NOT_FRONTEND_RELEVANT**: 1 endpoint
- **BACKEND_INTERNAL**: 1 endpoint

**Zero unexplained relevant endpoints.**
