# REST API & WebSocket Reference
## v1.16.0 — 26 August 2026

Two backends share the API surface:
| Service | Base URL | Port | Stack |
|---------|----------|:----:|-------|
| NestJS Control Plane | /api/v1 | 13080 | NestJS |
| Go Realtime Engine | / | 13081 | Go |

All mutations require JWT. Admin requires AdminGuard. WebSocket requires JWT or agent token.

---

## NestJS Control Plane — REST

### Auth (/auth)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /auth/register | Public | Register, returns JWT + refresh |
| POST | /auth/login | Public | Login, returns JWT + refresh |
| POST | /auth/verify-otp | JWT | Verify MFA OTP |
| POST | /auth/refresh | Refresh Token | Refresh access token |
| GET | /auth/me | JWT | Current user |
| POST | /auth/logout | JWT | Invalidate session |
| POST | /auth/mfa/setup | JWT | Enable MFA — returns TOTP secret |
| POST | /auth/forgot | Public | Request password reset |
| POST | /auth/reset | Public | Reset password with token |

### Users (/users)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /users/me | JWT | Current profile |
| PATCH | /users/me | JWT | Update profile |
| GET | /users/:id | JWT | User by ID (self or admin) |
| GET | /users | Admin | List users (paginated) |

### Plans (/plans)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /plans | Public | Active plans |
| GET | /plans/:id | Public | Plan details |

### Subscriptions (/subscriptions)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /subscriptions | JWT | Current subscription |
| POST | /subscriptions | JWT | Create/upgrade |
| DELETE | /subscriptions | JWT | Cancel |

### Devices (/devices)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /devices | JWT | Bound devices |
| POST | /devices/activate | JWT | Activate device |
| DELETE | /devices/:id | JWT | Remove device |

### Backtests (/backtests) — Admin
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /backtests | Admin | List backtests |
| POST | /backtests | Admin | Start backtest |
| GET | /backtests/:id | Admin | Results |

---

## Go Realtime Engine — REST + WS

### REST (port 13081)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /health | None | Health check |
| GET | /api/v1/signals | JWT | Current signals |
| GET | /api/v1/signals/history | JWT | History (paginated) |
| GET | /api/v1/signals/:id | JWT | Signal detail |
| GET | /api/v1/market/xauusd | None | Latest XAUUSD tick |
| GET | /api/v1/market/indicators | JWT | Live indicators |
| GET | /api/v1/engine/status | JWT | Engine health + liveness |

### WebSocket (port 13081)
| Path | Auth | Description |
|------|------|-------------|
| /ws/v1 | JWT | User signal stream |
| /ws/v1/agent | Agent token | Windows agent |

#### Server → Client Events
| Event | Payload | Description |
|-------|---------|-------------|
| signal | Signal | New trading signal |
| signal_update | Signal | Signal status change |
| tick | Tick | Real-time price |
| engine_status | Status | Engine health change |
| market_alert | Alert | Market event |

#### Client → Server Events
| Event | Payload | Description |
|-------|---------|-------------|
| subscribe | {symbols, strategies} | Filter subscription |
| ping | {} | Keep-alive |
