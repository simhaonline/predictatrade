# API Reference

**Version:** v1.3.1 — Advanced Risk + Backtesting  
**Date:** 19 August 2026

---

## Overview

The Predict-A-Trade API spans two backend services:

| Service | Base URL | Port | Technology |
|---------|----------|------|------------|
| NestJS Control Plane | `https://api.predictatrade.com/api/v1` | 3000 | NestJS |
| Go Real-Time Engine | `https://live.predictatrade.com` | 8081 | Go |

All mutation endpoints require JWT authentication (Bearer token). Admin endpoints require `AdminGuard` (role=ADMIN). WebSocket connections require JWT via query parameter or header.

---

## NestJS Control Plane — REST API

### Authentication (`/auth`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/auth/register` | Public | Register new user, returns JWT + refresh token |
| POST | `/auth/login` | Public | Login with email/password, returns JWT + refresh token |
| POST | `/auth/verify-otp` | JWT | Verify MFA OTP code |
| POST | `/auth/refresh` | Refresh Token | Refresh access token |
| GET | `/auth/me` | JWT | Get current authenticated user |
| POST | `/auth/logout` | JWT | Invalidate session and refresh token |
| POST | `/auth/mfa/setup` | JWT | Enable MFA — returns TOTP secret + backup codes |
| POST | `/auth/mfa/verify` | JWT | Verify MFA setup with OTP code |
| POST | `/auth/forgot` | Public | Request password reset email |
| POST | `/auth/reset` | Public | Reset password with token |

### Users (`/users`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/users/me` | JWT | Get current user profile |
| PATCH | `/users/me` | JWT | Update current user profile |
| GET | `/users/:id` | JWT | Get user by ID (self or admin) |
| GET | `/users` | Admin | List all users (paginated) |

### Plans (`/plans`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/plans` | Public | List all active plans |
| GET | `/plans/:id` | Public | Get plan details |

### Subscriptions (`/subscriptions`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/subscriptions` | JWT | Get current user's subscription |
| POST | `/subscriptions` | JWT | Create/upgrade subscription |

### Billing (`/billing`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/billing/invoices` | JWT | List user's invoices |
| POST | `/billing/webhook` | Public (signed) | Payment provider webhook callback |

### Referrals (`/referrals`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/referrals/network` | JWT | Get referral network tree |
| GET | `/referrals/commissions` | JWT | Get user's commission history |

### Commissions (`/commissions`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/commissions` | JWT | Get current user's commissions |
| GET | `/commissions/summary` | JWT | Get commission summary/totals |
| GET | `/commissions/admin/all` | Admin | List all commissions (paginated) |
| GET | `/commissions/admin/summary` | Admin | Global commission summary |

### Payouts (`/payouts`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/payouts` | JWT | Get user's payout history |
| POST | `/payouts/request` | JWT | Request a payout |
| GET | `/payouts/admin/all` | Admin | List all payouts (paginated) |

### Licensing (`/licensing`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/licensing/licenses` | JWT | List user's licenses |
| GET | `/licensing/devices` | JWT | List user's registered devices |
| POST | `/licensing/devices` | JWT | Register a new device |
| GET | `/licensing/mt-accounts` | JWT | List user's MT accounts |
| POST | `/licensing/mt-accounts` | JWT | Register an MT account |

### Device Authentication (`/devices`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/devices/activate` | License Key | Activate device with license key |
| POST | `/devices/refresh` | Device Token | Refresh device session token |
| POST | `/devices/heartbeat` | Device Token | Device heartbeat (keep-alive) |
| GET | `/devices/sessions` | JWT | List device sessions |
| GET | `/devices/devices/:id` | JWT | Get device details |
| POST | `/devices/devices/:id/revoke` | JWT | Revoke a device session |

### Health (`/health`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | Public | Health check (DB connectivity, service status) |

### Audit (`/audit`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/audit` | Admin | List audit events (paginated) |

### Operations (`/operations`)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/operations/state` | Admin | Get current operational state |
| GET | `/operations/active` | Admin | Get active strategies and signals |
| POST | `/operations/halt-trading` | Admin | Emergency halt all trading |
| POST | `/operations/resume-trading` | Admin | Resume trading after halt |
| POST | `/operations/pause-signals` | Admin | Pause signal delivery |
| POST | `/operations/resume-signals` | Admin | Resume signal delivery |
| POST | `/operations/strategy/:id/enable` | Admin | Enable a specific strategy |
| POST | `/operations/strategy/:id/disable` | Admin | Disable a specific strategy |
| GET | `/operations/ai/models` | Admin | List AI/ML models in registry |
| GET | `/operations/ai/training-jobs` | Admin | List training jobs |
| GET | `/operations/ai/inference` | Admin | Get inference history |
| POST | `/operations/ai/model/:id/activate` | Admin | Activate an AI model |
| POST | `/operations/ai/model/:id/deactivate` | Admin | Deactivate an AI model |

### Admin (`/admin`)

All admin endpoints require JWT + AdminGuard.

| Method | Path | Description |
|--------|------|-------------|
| GET | `/admin/overview` | Platform overview stats |
| GET | `/admin/users` | List all users (paginated) |
| PATCH | `/admin/users/:id/status` | Update user status (active/suspended) |
| GET | `/admin/subscriptions` | List all subscriptions (paginated) |
| GET | `/admin/commissions` | List all commissions (paginated) |
| GET | `/admin/commissions/summary` | Commission summary |
| GET | `/admin/payouts` | List all payouts (paginated) |
| GET | `/admin/payouts/stats` | Payout statistics |
| GET | `/admin/licenses` | List all licenses (paginated) |
| GET | `/admin/devices` | List all devices (paginated) |
| GET | `/admin/health` | System health overview |

---

## Go Real-Time Engine — REST API

Base URL: `https://live.predictatrade.com` or `https://api.predictatrade.com` (proxied)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | Public | Health check |
| GET | `/ready` | Public | Readiness check |
| GET | `/metrics` | Public | Prometheus metrics (50+ metrics) |
| GET | `/api/v1/signals` | Public | Recent signals (50) |
| GET | `/api/v1/market/state` | Public | Current market state |
| GET | `/api/v1/candles` | Public | Recent candles (200) |
| GET | `/api/v1/strategies` | Public | Four strategy IDs and config |
| GET | `/api/v1/market/snapshot` | Public | Latest Master Node snapshot |
| GET | `/api/v1/agents/status` | Public | Windows Agent connection status |
| GET | `/api/v1/price/history` | Public | Price history from Valkey cache |
| GET | `/api/v1/signals/resume` | Public | Signal resume for reconnection |

### Signal Object (REST + WebSocket parity)

```json
{
  "id": "uuid",
  "symbol": "XAUUSD",
  "strategy_id": "STANDARD_SCALPING",
  "direction": "BUY|SELL|BUY_CANDIDATE|SELL_CANDIDATE|WAIT|NO-TRADE|BLOCKED|ERROR",
  "grade": "A+|A|B|C|NO-TRADE|WAIT|BLOCKED|ERROR|UNRATED|RESEARCH|SHADOW",
  "raw_score": "82.4",
  "long_score": "82.4",
  "short_score": "51.2",
  "calibrated_probability": "0.755",
  "entry_price": "4400.05",
  "stop_loss": "4385.05",
  "tp1": "4418.05",
  "tp2": "4427.05",
  "tp3": "4445.05",
  "regime": "STRONG_TREND_UP",
  "session": "LONDON",
  "news_risk": "NONE",
  "timeframe": "M5",
  "status": "CONFIRMED",
  "reason_codes": ["MTF_BULLISH_ALIGNMENT", "HTF_STRUCTURE_BULLISH"],
  "evidence": [{"pillar":"TREND","feature":"EMA9_ABOVE_EMA21","direction":"BUY","weight":"15","contribution":"0.12"}],
  "gate_results": [{"gate_id":"spread","result":"PASS","reason_codes":[]}],
  "created_at": "2026-08-18T16:00:00Z",
  "expires_at": "2026-08-18T16:15:00Z",
  "signal_class": "EXECUTABLE|ADVISORY",
  "candidate_threshold": 40,
  "trade_threshold": 65,
  "calibration_status": "UNVERIFIED|SHADOW|VALIDATED|PROMOTED"
}
```

### Direction Types

| Direction | Description | Executable? | Signal Class |
|-----------|-------------|-------------|--------------|
| `BUY` | Qualified long — score ≥ trade threshold + all gates passed | ✅ | EXECUTABLE |
| `SELL` | Qualified short — score ≥ trade threshold + all gates passed | ✅ | EXECUTABLE |
| `BUY_CANDIDATE` | Advisory long — candidate threshold ≤ score < trade threshold | ❌ | ADVISORY |
| `SELL_CANDIDATE` | Advisory short — candidate threshold ≤ score < trade threshold | ❌ | ADVISORY |
| `WAIT` | Insufficient features or conflicting signals — no action | ❌ | — |
| `NO-TRADE` | Score below candidate threshold or strategy NO-TRADE | ❌ | — |
| `BLOCKED` | Direction preserved (BUY/SELL) but gate veto/safety block occurred | ❌ | ADVISORY |
| `ERROR` | Processing error | ❌ | — |

### Calibrated Probability

The `calibrated_probability` field is NULL/zero until a calibration model is VALIDATED or PROMOTED (SOW §16, §36). The `calibration_status` field indicates the current state. Default seeded models have `UNVERIFIED` status. See `docs/SIGNAL_TYPES_AND_PROBABILITY.md` for details.

---

## WebSocket

### Connection

| URL | Purpose | Auth |
|-----|---------|------|
| `wss://live.predictatrade.com/ws/v1` | Dashboard/client signal stream | JWT |
| `wss://live.predictatrade.com/ws/v1/agent` | Windows MT5 Agent | Device Token |

### Event Envelope

```json
{
  "event_id": "uuid",
  "stream_id": "signals:STANDARD_SCALPING",
  "sequence": 1,
  "schema_version": "1.2.0",
  "timestamp": "2026-08-18T16:00:00Z",
  "type": "SIGNAL|MARKET_STATE|MARKET_SNAPSHOT|AGENT_STATUS",
  "priority": "P0|P1|P2",
  "payload": { ... }
}
```

### Event Types

| Type | Priority | Description |
|------|----------|-------------|
| SIGNAL | P0 (BUY/SELL), P1 (NO-TRADE/WAIT/BLOCKED/ERROR) | Strategy signal |
| MARKET_STATE | P2 | Market state update (10ms cadence) |
| MARKET_SNAPSHOT | P2 | Master Node snapshot |
| AGENT_STATUS | P2 | Agent connection status |

### Entitlement Filtering

WebSocket clients are filtered by strategy entitlement. Only entitled strategies' signals are delivered.

### Signal Resume

On reconnection, clients request signal resume via `GET /api/v1/signals/resume` to retrieve missed signals since last sequence number.

---

## PTB Analysis Output

When PTB exits shadow mode, the following fields are available via the signal payload:

```json
{
  "analysis_id": "ptb-1234567890",
  "regime": "STRONG_TREND_UP",
  "gold_role": "UNKNOWN",
  "volatility_state": "NORMAL",
  "manipulation_index": 25.0,
  "bias": "LONG",
  "bias_strength": 0.65,
  "confidence": 78.4,
  "confluence_score": 81.2,
  "setup_quality": "A",
  "action": "ENTER",
  "position_size_multiplier": 0.80,
  "stop_distance_multiplier": 1.0,
  "market_narrative": "Regime: STRONG_TREND_UP...",
  "key_drivers": ["mtf (82, LONG)", "structure (70, LONG)"],
  "risk_factors": ["MACRO_DATA_UNAVAILABLE"],
  "positive_factors": ["MTF_BULLISH_ALIGNMENT", "HTF_STRUCTURE_BULLISH"],
  "negative_factors": ["MACRO_DATA_UNAVAILABLE"],
  "reason_codes": ["MTF_BULLISH_ALIGNMENT", "HTF_STRUCTURE_BULLISH"],
  "component_scores": {
    "mtf": {"score": 82, "direction": "LONG"},
    "structure": {"score": 70, "direction": "LONG"},
    "liquidity": {"score": 70, "direction": "LONG"}
  },
  "data_quality": {"market_data": "EXCELLENT", "macro": "UNAVAILABLE"},
  "shadow_mode": true
}
```

---

## Advanced Risk API

The advanced risk layer (v1.1.0) is integrated into the signal pipeline. The following fields are available in the signal payload when advanced features are enabled:

### Loss Recovery State

```json
{
  "recovery_state": "NORMAL|RECOVERY|HALTED|DAILY_LIMIT",
  "recovery_reason": "consecutive_losses|daily_loss_limit|halt_expiry",
  "consecutive_losses": 1,
  "daily_pnl_percent": -0.5
}
```

### Adaptation Context

```json
{
  "market_phase": "TRENDING|RANGING|HIGH_VOLATILITY|LOW_VOLATILITY|MANIPULATIVE|UNCERTAIN",
  "adaptation_adjustments": {
    "risk_multiplier": 0.7,
    "stop_distance_multiplier": 1.5,
    "confluence_bonus": 10,
    "weight_adjustments": {"trend": 1.2, "sr": 0.8}
  }
}
```

### Hedging Status

```json
{
  "hedging_enabled": false,
  "active_hedges": [],
  "aggregate_exposure": 0.0
}
```

### ML/RL Status

```json
{
  "ml_adaptation_active": false,
  "rl_mode": "disabled|shadow|filter_only|live_approved",
  "model_version": null
}
```

### Sentiment

```json
{
  "sentiment_available": false,
  "sentiment_score": 0,
  "sentiment_category": "NEUTRAL",
  "sentiment_confidence": 0
}
```

---

## Backtesting CLI API

Backtesting is accessed via Python CLI (not REST):

```bash
# Run a backtest
cd research && python3 -m patresearch.backtesting.cli run --strategy STANDARD_SCALPING --seed 42

# Walk-forward analysis
cd research && python3 -m patresearch.backtesting.cli walk-forward --strategy STANDARD_SCALPING

# Monte Carlo analysis
cd research && python3 -m patresearch.backtesting.cli monte-carlo --runs 1000

# Parameter sensitivity
cd research && python3 -m patresearch.backtesting.cli sensitivity --strategy STANDARD_SCALPING
```

Backtest results are persisted to `trading.backtest_runs`, `trading.backtest_trades`, `trading.backtest_fold_results`, `trading.backtest_artifacts`, `trading.backtest_parameter_sets` (migration 015).

---

## MT4/MT5 Signal Contract

Both MT4 and MT5 receive identical JSON via shared file (FILE_COMMON):

```json
{
  "ID": "signal-uuid",
  "Direction": "BUY|SELL",
  "Grade": "A+|A|B|C",
  "StrategyID": "STANDARD_SCALPING",
  "EntryPrice": 4400.05,
  "StopLoss": 4385.05,
  "TP1": 4418.05,
  "TP2": 4427.05,
  "TP3": 4445.05
}
```

MT4/MT5 only execute `BUY` or `SELL` — all other states (WAIT, NO-TRADE, BLOCKED, ERROR) are ignored.

### Master Node EAs

Two additional EAs act as Master Node data providers:

| EA | File | Role |
|----|------|------|
| PredictATrade_MasterNode_MT4 | `mql/mt4/PredictATrade_MasterNode_MT4.mq4` | Publishes MT4 tick data to Windows Agent |
| PredictATrade_MasterNode_MT5 | `mql/mt5/PredictATrade_MasterNode_MT5.mq5` | Publishes MT5 tick data to Windows Agent |

---

## Error Responses

All API errors use a consistent machine-readable format:

```json
{
  "statusCode": 400,
  "message": "Validation failed",
  "error": "Bad Request",
  "correlationId": "uuid"
}
```

| HTTP Status | Meaning |
|-------------|---------|
| 400 | Bad Request — validation error |
| 401 | Unauthorized — missing/invalid JWT |
| 403 | Forbidden — insufficient permissions (AdminGuard) |
| 404 | Not Found — resource doesn't exist |
| 409 | Conflict — duplicate/idempotency violation |
| 429 | Too Many Requests — rate limited |
| 500 | Internal Server Error |

---

## Pagination

List endpoints accept `page` and `limit` query parameters:

```
GET /admin/users?page=2&limit=20
```

Response format:

```json
{
  "data": [...],
  "total": 150,
  "page": 2,
  "limit": 20,
  "totalPages": 8
}
```

---

## Authentication Flow

1. **Register/Login** → receive `accessToken` (JWT, 15min) + `refreshToken` (30 days)
2. **Use accessToken** in `Authorization: Bearer <token>` header for all authenticated requests
3. **Refresh** via `POST /auth/refresh` with refresh token before access token expires
4. **MFA** (optional) — setup via `POST /auth/mfa/setup`, verify each login via `POST /auth/verify-otp`
5. **Logout** via `POST /auth/logout` — invalidates session and refresh token

### Session Security

- Session token rotation (migration 006)
- Auth hardening with recovery codes and login event tracking (migration 007)
- Device activation sessions (migration 008)
- Correlation ID interceptor on all requests

---

## Rate Limiting

Rate limiting is enforced via Nginx and application-level guards. See `infra/nginx/` for configuration.
