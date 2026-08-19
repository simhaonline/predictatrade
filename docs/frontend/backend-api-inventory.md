# Backend API Inventory

## NestJS Control Plane (port 13080, prefix /api/v1)

| Module | Method | Route | Roles | Request DTO | Response | Frontend Consumer | Implemented |
|--------|--------|-------|-------|-------------|----------|-------------------|------------|
| Auth | POST | /auth/register | Public | RegisterDto | { id, email, accessToken } | Register page | ✅ |
| Auth | POST | /auth/login | Public | LoginDto | { accessToken, user } or { mfaRequired, challengeId } | Login page | ✅ |
| Auth | POST | /auth/verify-otp | Public | VerifyOtpDto | { accessToken } | Verify-OTP page | ✅ |
| Auth | POST | /auth/refresh | Cookie | (none, cookie) | { accessToken } | Axios interceptor | ✅ |
| Auth | GET | /auth/me | Authenticated | (none) | User profile | Auth provider | ✅ |
| Auth | POST | /auth/logout | Authenticated | (none) | { success: true } | Auth provider | ✅ |
| Auth | POST | /auth/mfa/setup | Authenticated | (none) | { secret, otpauth } | Settings (future) | ✅ |
| Auth | POST | /auth/mfa/verify | Authenticated | MfaSetupDto | { verified } | Settings (future) | ✅ |
| Auth | POST | /auth/forgot | Public | ForgotPasswordDto | { sent } | Forgot password | ✅ |
| Auth | POST | /auth/reset | Public | ResetPasswordDto | { success } | Reset password | ✅ |
| Users | GET | /users/me | Authenticated | (none) | User profile | Settings | ✅ |
| Users | PATCH | /users/me | Authenticated | UpdateUserDto | Updated user | Settings | ✅ |
| Users | GET | /users/:id | Admin | (none) | User profile | Admin users | ✅ |
| Users | GET | /users | Admin | page, limit | { items, total, page, limit } | Admin users | ✅ |
| Admin | GET | /admin/overview | Admin | (none) | Overview stats | Admin dashboard | ✅ |
| Admin | GET | /admin/users | Admin | page, limit | { items, total } | Admin users | ✅ |
| Admin | PATCH | /admin/users/:id/status | Admin | status | Updated user | Admin users | ✅ |
| Admin | GET | /admin/subscriptions | Admin | page, limit | { items, total } | Admin subscriptions | ✅ |
| Admin | GET | /admin/commissions | Admin | page, limit | { items, total } | Admin referrals/billing | ✅ |
| Admin | GET | /admin/commissions/summary | Admin | (none) | Summary object | Admin referrals | ✅ |
| Admin | GET | /admin/payouts | Admin | page, limit | { items, total } | Admin referrals/billing | ✅ |
| Admin | GET | /admin/payouts/stats | Admin | (none) | Stats object | Admin referrals | ✅ |
| Admin | GET | /admin/licenses | Admin | page, limit | { items, total } | Admin licenses | ✅ |
| Admin | GET | /admin/devices | Admin | page, limit | { items, total } | Admin device-auth | ✅ |
| Admin | GET | /admin/health | Admin | (none) | { services: [...] } | Admin health | ✅ |
| Billing | GET | /billing/invoices | Authenticated | (none) | Invoice[] | User billing | ✅ |
| Billing | POST | /billing/webhook | Public | body, headers | { received, eventType } | NOT_FRONTEND | N/A |
| Subscriptions | GET | /subscriptions | Authenticated | (none) | Subscription[] | User strategies | ✅ |
| Subscriptions | POST | /subscriptions | Authenticated | CreateSubscriptionDto | Subscription | User strategies | ✅ |
| Plans | GET | /plans | Authenticated | (none) | Plan[] | User strategies | ✅ |
| Plans | GET | /plans/:id | Authenticated | (none) | Plan | (future) | ✅ |
| Licensing | GET | /licensing/licenses | Authenticated | (none) | License[] | User MT client | ✅ |
| Licensing | GET | /licensing/devices | Authenticated | (none) | Device[] | User MT client | ✅ |
| Licensing | POST | /licensing/devices | Authenticated | body | Device | User MT client | ✅ |
| Licensing | GET | /licensing/mt-accounts | Authenticated | (none) | MTAccount[] | User MT client | ✅ |
| Licensing | POST | /licensing/mt-accounts | Authenticated | body | MTAccount | User MT client | ✅ |
| Commissions | GET | /commissions | Authenticated | (none) | Commission[] | User referrals | ✅ |
| Commissions | GET | /commissions/summary | Authenticated | (none) | Summary | User referrals/reports | ✅ |
| Commissions | GET | /commissions/admin/all | Admin | page, limit | { items, total } | Admin referrals/billing | ✅ |
| Commissions | GET | /commissions/admin/summary | Admin | (none) | Global summary | Admin referrals | ✅ |
| Referrals | GET | /referrals/network | Authenticated | (none) | Referral[] | User referrals | ✅ |
| Referrals | GET | /referrals/commissions | Authenticated | (none) | Commission[] | User referrals | ✅ |
| Payouts | GET | /payouts | Authenticated | (none) | Payout[] | User referrals | ✅ |
| Payouts | POST | /payouts/request | Authenticated | RequestPayoutDto | Payout | User referrals | ✅ |
| Payouts | GET | /payouts/admin/all | Admin | page, limit | { items, total } | Admin referrals/billing | ✅ |
| Payouts | GET | /payouts/admin/stats | Admin | (none) | Stats | Admin referrals | ✅ |
| Payouts | POST | /payouts/:id/approve | Admin | (none) | Approved payout | Admin billing | ✅ |
| Operations | GET | /operations/state | Admin | (none) | TradingState | Admin strategies/operations | ✅ |
| Operations | GET | /operations/active | Admin | (none) | Operation[] | Admin operations | ✅ |
| Operations | POST | /operations/halt-trading | Admin | { reason } | Operation | Admin operations | ✅ |
| Operations | POST | /operations/resume-trading | Admin | { reason } | Operation | Admin operations | ✅ |
| Operations | POST | /operations/pause-signals | Admin | { reason } | Operation | Admin operations | ✅ |
| Operations | POST | /operations/resume-signals | Admin | { reason } | Operation | Admin operations | ✅ |
| Operations | POST | /operations/strategy/:id/enable | Admin | { reason } | Operation | Admin strategies | ✅ |
| Operations | POST | /operations/strategy/:id/disable | Admin | { reason } | Operation | Admin strategies | ✅ |
| Operations | GET | /operations/ai/models | Admin | (none) | Model[] | Admin operations | ✅ |
| Operations | GET | /operations/ai/training-jobs | Admin | (none) | Job[] | Admin operations | ✅ |
| Operations | GET | /operations/ai/inference | Admin | limit | Inference[] | Admin operations | ✅ |
| Operations | POST | /operations/ai/model/:id/activate | Admin | (none) | { success } | Admin operations | ✅ |
| Operations | POST | /operations/ai/model/:id/deactivate | Admin | (none) | { success } | Admin operations | ✅ |
| Audit | GET | /audit | Admin | page, limit | { items, total } | Admin logs | ✅ |
| Health | GET | /health | Public | (none) | { status, database } | Admin health | ✅ |
| Devices | POST | /devices/activate | Public | body | Device session | Windows Agent | N/A |
| Devices | POST | /devices/refresh | Public | body | New tokens | Windows Agent | N/A |
| Devices | POST | /devices/heartbeat | Public | body | Session state | Windows Agent | N/A |
| Devices | GET | /devices/sessions | Admin | (none) | Session[] | Admin activations | ✅ |
| Devices | GET | /devices/devices/:id | Admin | (none) | Device details | Admin activations | ✅ |
| Devices | POST | /devices/devices/:id/revoke | Admin | { reason } | Revoked | Admin activations | ✅ |

## Go Realtime Engine (port 8080)

| Method | Route | Roles | Frontend Consumer | Implemented |
|--------|-------|-------|-------------------|------------|
| GET | /health | Public | Admin health | ✅ |
| GET | /ready | Public | Admin health | ✅ |
| GET | /metrics | System | Prometheus (not frontend) | N/A |
| WS | /ws | Public | WebSocket manager | ✅ |
| WS | /ws/v1 | Public | WebSocket manager | ✅ |
| WS | /ws/v1/agent | Agent | Windows Agent | N/A |
| WS | /ws/agent | Agent | Windows Agent | N/A |
| GET | /api/v1/signals | Public | Admin/User signals | ✅ |
| GET | /api/v1/market/state | Public | Admin scoring/dashboard | ✅ |
| GET | /api/v1/candles | Public | (future charts) | ✅ |
| GET | /api/v1/strategies | Public | Admin strategies | ✅ |
| GET | /api/v1/market/snapshot | Public | Admin indicators | ✅ |
| GET | /api/v1/agents/status | Public | Admin trading-reports | ✅ |
| GET | /api/v1/price/history | Public | (future charts) | ✅ |
| POST | /api/v1/signals/resume | Public | (future) | ✅ |
