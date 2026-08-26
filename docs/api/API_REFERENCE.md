# REST API & WebSocket Reference
## v1.16.0 — 26 August 2026

Two backends: NestJS Control Plane (:13080), Go Realtime Engine (:13081).

### NestJS — REST (/api/v1)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /auth/register | Public | Register, returns JWT |
| POST | /auth/login | Public | Login, returns JWT |
| POST | /auth/verify-otp | JWT | Verify MFA OTP |
| POST | /auth/refresh | Refresh | New access token |
| GET | /auth/me | JWT | Current user |
| POST | /auth/logout | JWT | Invalidate session |
| POST | /auth/mfa/setup | JWT | Enable MFA |
| POST | /auth/forgot | Public | Request password reset |
| GET | /users/me | JWT | Current profile |
| PATCH | /users/me | JWT | Update profile |
| GET | /plans | Public | Active plans |
| GET | /subscriptions | JWT | Current subscription |
| POST | /subscriptions | JWT | Create/upgrade |
| GET | /devices | JWT | Bound devices |
| POST | /devices/activate | JWT | Activate device |

### Go Realtime — REST (:13081)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /health | None | Health check |
| GET | /api/v1/signals | JWT | Current signals |
| GET | /api/v1/signals/history | JWT | History (paginated) |
| GET | /api/v1/market/xauusd | None | Latest tick |
| GET | /api/v1/engine/status | JWT | Engine health |

### WebSocket — (:13081)

| Path | Auth | Description |
|------|------|-------------|
| /ws/v1 | JWT | User signal stream |
| /ws/v1/agent | Agent token | Windows agent |

Events: signal, signal_update, tick, engine_status, market_alert
