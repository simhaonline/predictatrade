# REST API & WebSocket Reference
## v1.17.2 — 28 August 2026

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
| GET | /subscriptions/entitlements | JWT | Selected strategies + features |

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
| GET | /health | None | Health check — returns `{"status":"ok"}` |
| GET | /api/v1/signals | JWT | Current signals (limit 50 default, max 200). Plan-filtered: admin sees all, authenticated sees entitled strategies, unauthenticated sees ADVISORY-only |
| GET | /api/v1/signals?limit=N | JWT | Signals with explicit limit (1-200) |
| GET | /api/v1/trades | JWT | REAL executed trades from trading.trade_results. Strategy-filterable via `?strategy=STANDARD_SCALPING` |
| GET | /api/v1/market/snapshot | None | Latest XAUUSD tick + indicators snapshot |
| GET | /api/v1/market/indicators | JWT | Live indicators |
| GET | /api/v1/engine/status | JWT | Engine health + liveness |
| GET | /api/v1/agents/status | Admin | Connected agent count, version, last heartbeat |
| GET | /api/v1/system-health | Admin | Full system health including engine, market feeds, agents |

### License Validation (v1.16.x)
| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /api/v1/license/validate | Agent Token | Proactive server-side license validation. No agent changes required |

### WebSocket (port 13081 exec + 13091 data)
| Path | Auth | Description |
|------|------|-------------|
| /ws/v1 | JWT | User signal stream |
| /ws/v1/agent | Agent token | Windows Client Agent (execution) connection |
| /ws/v1/data | Agent token | Windows Master Node (data-only) connection — port 13091 |

#### Server → Client Events
| Event | Payload | Description |
|-------|---------|-------------|
| signal | Signal | New trading signal (includes TP1/TP2/TP3, Regime, Session, QualityGrade, ExpectancyR, SuggestedLot, RiskDollars, RiskPctOfEquity) |
| signal_update | Signal | Signal status change |
| tick | Tick | Real-time price |
| engine_status | Status | Engine health change |
| market_alert | Alert | Market event |

#### Client → Server Events
| Event | Payload | Description |
|-------|---------|-------------|
| subscribe | {symbols, strategies} | Filter subscription |
| ping | {} | Keep-alive |

### Signal Object (Go Engine Response)
| Field | Type | Description |
|-------|------|-------------|
| ID | UUID | Unique signal identifier |
| Direction | string | BUY, SELL, BUY_CANDIDATE, SELL_CANDIDATE, NO-TRADE |
| StrategyID | string | STANDARD_SCALPING, ULTRA_SCALPING, etc. |
| RawScore | decimal | Raw composite score |
| CalibratedProbability | decimal | Win probability (0-1; "Pending" if not validated) |
| EntryPrice | decimal | Entry zone mid-point |
| StopLoss | decimal | Stop loss level |
| TP1 | decimal | Take profit level 1 |
| TP2 | decimal | Take profit level 2 |
| TP3 | decimal | Take profit level 3 |
| GrossRRTP1/2/3 | decimal | Gross R:R per TP level |
| Regime | string | TRENDING_BULLISH, TRENDING_BEARISH, RANGE, etc. |
| Session | string | TOKYO, LONDON, NEW_YORK, OVERLAP |
| QualityGrade | string | A+, A, B, REJECTED |
| ExpectancyR | decimal | Expected value per unit risk |
| ExpectancyScore | float | 0-100 quality score |
| SuggestedLot | decimal | Engine-recommended lot (risk-capped) |
| RiskDollars | decimal | Risk at stop distance, USD |
| RiskPctOfEquity | decimal | Risk as % of account equity |
| SLDistancePoints | decimal | Stop distance in points |
| Evidence | array | Evidence contributions with pillar/feature/direction |
| ReasonCodes | array | NO-TRADE reason codes |
| Executable | boolean | Whether signal is EXECUTABLE (passed all gates) |
| SignalClass | string | ADVISORY or EXECUTABLE |
| CreatedAt | ISO 8601 | Signal creation timestamp (UTC) |
