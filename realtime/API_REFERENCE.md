# Predict-A-Trade API Reference
## v1.16.0 — 26 August 2026

### Architecture
Two backend services:
| Service | Base URL | Port | Tech |
|---------|----------|:----:|------|
| NestJS Control Plane | /api/v1 | 13080 | NestJS |
| Go Realtime Engine | / | 13081 | Go |

All mutations require JWT. Admin requires AdminGuard. WebSocket requires JWT.

---

## NestJS Control Plane — REST

### Auth (/auth)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /auth/register | Public | Register, returns JWT + refresh |
| POST | /auth/login | Public | Login, returns JWT + refresh |
| POST | /auth/verify-otp | JWT | Verify MFA OTP |
| POST | /auth/refresh | Refresh | Refresh access token |
| GET | /auth/me | JWT | Current user |
| POST | /auth/logout | JWT | Invalidate session |
| POST | /auth/mfa/setup | JWT | Enable MFA (TOTP + backup codes) |
| POST | /auth/mfa/verify | JWT | Verify MFA setup |
| POST | /auth/forgot | Public | Password reset request |
| POST | /auth/reset | Public | Reset password with token |
| GET | /auth/consent | JWT | Get user consent status |
| PUT | /auth/consent | JWT | Update consent preferences |

### Users (/users)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /users/me | JWT | Current user profile |
| PATCH | /users/me | JWT | Update profile |
| GET | /users/:id | JWT | User by ID (self or admin) |
| GET | /users | Admin | List all users (paginated) |

### Plans (/plans)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /plans | Public | List active plans |
| GET | /plans/:id | Public | Plan details |

### Subscriptions (/subscriptions)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /subscriptions | JWT | Current subscription |
| POST | /subscriptions | JWT | Create/upgrade |
| DELETE | /subscriptions | JWT | Cancel subscription |

### Devices (/devices)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /devices | JWT | List bound devices |
| POST | /devices/activate | JWT | Activate new device |
| DELETE | /devices/:id | JWT | Remove device |

### Backtests (/backtests) — Admin
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /backtests | Admin | List backtests |
| POST | /backtests | Admin | Start backtest |
| GET | /backtests/:id | Admin | Backtest results |
| DELETE | /backtests/:id | Admin | Delete backtest |

---

## Go Realtime Engine — REST + WS

### REST Endpoints (port 13081)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /health | None | Health check |
| GET | /api/v1/signals | JWT | Current signals |
| GET | /api/v1/signals/history | JWT | Signal history (paginated) |
| GET | /api/v1/signals/:id | JWT | Single signal detail |
| GET | /api/v1/market/xauusd | None | Latest XAUUSD tick |
| GET | /api/v1/market/indicators | JWT | Live indicator values |
| GET | /api/v1/dashboard/stats | JWT | Dashboard statistics |
| GET | /api/v1/engine/status | JWT | Engine health + liveness |

### WebSocket (port 13081)
| Path | Auth | Description |
|------|------|-------------|
| /ws/v1 | JWT | User signal stream |
| /ws/v1/agent | Agent token | Windows agent connection |

### WebSocket Events
**Server → Client:**
| Event | Payload | Description |
|-------|---------|-------------|
| signal | Signal | New trading signal |
| signal_update | Signal | Signal status change |
| tick | Tick | Real-time price update |
| engine_status | Status | Engine health change |
| market_alert | Alert | Market event notification |

**Client → Server:**
| Event | Payload | Description |
|-------|---------|-------------|
| subscribe | {symbols, strategies} | Filter subscription |
| ping | {} | Keep-alive |

### Signal Object Schema
```json
{
  "id": "uuid",
  "symbol": "XAUUSD",
  "strategy_id": "STANDARD_SCALPING",
  "direction": "BUY",
  "grade": "A",
  "status": "DETECTED",
  "raw_score": "72.5",
  "long_score": "78.0",
  "short_score": "12.0",
  "entry_price": "2430.50",
  "stop_loss": "2425.00",
  "tp1": "2442.00",
  "tp2": "2453.75",
  "tp3": "2465.50",
  "regime": "TRENDING_BULLISH",
  "session": "LONDON",
  "news_risk": "LOW",
  "evidence": [...],
  "trade_group_id": "GRP-STANDARD_SCALPING-1724684400",
  "created_at": "2026-08-26T10:00:00Z",
  "expires_at": "2026-08-26T10:10:00Z"
}
```