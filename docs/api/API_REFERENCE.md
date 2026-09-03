# REST & WebSocket API Reference
## v1.27.0 — 04 September 2026

Two backends share the API surface:

| Service | Base URL | Port | Stack |
|---------|----------|:----:|-------|
| NestJS Control Plane | `/api/v1` | 13080 | NestJS 12 |
| Go Realtime Engine | `/` (direct) or `/api/v1/*` via `api.` edge | 13081 (+13091 data) | Go 1.25 |

**Machine-readable spec:** [`openapi.json`](openapi.json) (OpenAPI 3.0, 64 paths — generated from
the control plane's `@nestjs/swagger` decorators; also served from `control/openapi.json`).

**Auth:** JWT Bearer for users/admins; agent tokens for the Windows edge; HMAC signatures for
payment webhooks. Admin routes additionally pass `AdminGuard` (+ `RolesGuard`/`PermissionGuard`
where role-scoped). All responses are JSON; errors are
`{"message": string|string[], "error": string, "statusCode": int}`.

---

## 1. Auth (`/auth`) — `auth.controller.ts`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /auth/register | Public | Register; seeds consent records; returns JWT + refresh cookie |
| POST | /auth/login | Public (throttled 10/min) | Returns `{accessToken, user, mfaRequired?}`; privileged roles require MFA enrollment |
| POST | /auth/verify-otp | Public challenge | Verify TOTP for login (`challengeId`, `method`) |
| POST | /auth/mfa/setup | JWT | Enroll authenticator — returns TOTP secret + otpauth URL |
| POST | /auth/mfa/verify | JWT | Confirm enrollment with a code |
| POST | /auth/refresh | Cookie | Rotate access token (single-flight, shared across tabs) |
| GET | /auth/me | JWT | Current user (fresh role) |
| POST | /auth/logout | JWT | Invalidate session, clear refresh cookie |
| POST | /auth/forgot | Public | Request password reset (tokenized email) |
| POST | /auth/reset | Public | Reset password with token |

## 2. Users (`/users`) — `users.controller.ts`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /users/me | JWT | Current profile |
| PATCH | /users/me | JWT | Update `displayName` / `timezone` |
| GET | /users/{id} | Self or Admin | User detail |
| GET | /users | Admin | Paginated list |

## 3. Plans (`/plans`) — `plans.controller.ts`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /plans | Public | Active plans incl. `allowed_strategies`, price, caps |
| GET | /plans/{id} | Public | Plan detail |

## 4. Subscriptions (`/subscriptions`) — `subscriptions.controller.ts`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /subscriptions | JWT | Current subscription (with plan code) |
| POST | /subscriptions | JWT | Create/upgrade. Server re-validates request against
`entitlement-policy.ts` (`AT_LEAST_ONE_STRATEGY_REQUIRED`, `STRATEGY_NOT_ENTITLED`,
`plan_strategy_limit`). FREE → status ACTIVE + paid invoice; paid → INCOMPLETE until webhook |
| PATCH | /subscriptions/strategies | JWT | Update selected strategies (entitlement-validated) |
| POST | /subscriptions/{id}/pause | JWT | Pause own subscription |
| POST | /subscriptions/{id}/resume | JWT | Resume |
| POST | /subscriptions/{id}/cancel | JWT | Cancel |
| GET | /subscriptions/entitlements | JWT | `{code, allowed_strategies, selected_strategies, features, caps}` |
| GET | /subscriptions/strategies | JWT | Strategy entitlement matrix (admin console variant `/admin` scope) |

Admin (under `/admin`): `/admin/subscriptions`, `/admin/subscriptions/{payments|refunds|chargebacks|coupons|provider}`.

## 5. Billing (`/billing`) — `billing.controller.ts`

> **Payments policy (v1.17.4): USDT-only.** Stripe is disabled at the controller:
> `/billing/stripe/checkout` → `403 USDT-only`, `/billing/webhook/stripe` → 204
> (unless operator sets `PAT_ENABLE_STRIPE=true`). **Anti-scam settlement:** IPN
> requires `x-nowpayments-sig` HMAC-SHA512 (timing-safe compare), exact-key
> replay dedupe (`billing.payment_events`), transactional one-shot settlement,
> and **amount verification** — the gateway-reported paid amount must cover the
> invoice expected within `NOWPAYMENTS_UNDERPAY_TOLERANCE_PCT` (default 2%),
> otherwise the payment is marked **UNDERPAID** (audit row; subscription NOT
> activated). Users see the live state on the billing page banner
> (awaiting_payment / underpaid / confirmed / failed) via `GET
> /billing/payments`. `NOWPAYMENTS_REQUIRE_AMOUNT=strict` refuses settlement
> when the gateway omits amounts.

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /billing/invoices | JWT | Own invoices (branded PDF available via `/invoices/{id}/html`) |
| GET | /billing/invoices/generate | Admin | Invoices for a subscription (`subscriptionId` query) |
| GET | /billing/invoices/{id} | Owner/Admin | Invoice |
| GET | /billing/invoices/{id}/html | Owner/Admin | Rendered HTML invoice |
| POST | /billing/invoices/{id}/mark-paid | Admin | Manual settlement |
| GET | /billing/payments | JWT | USDT payment status (dashboard banner) — `display_status` ∈ awaiting_payment/confirmed/underpaid/failed, `amount`, gateway event, NOWPayments `hosted_url` |
| POST | /billing/stripe/checkout | JWT | **DISABLED (USDT-only)** — 403 unless `PAT_ENABLE_STRIPE=true` |
| POST | /billing/webhook | HMAC | Legacy Stripe webhook — 204 when Stripe disabled |
| POST | /billing/webhook | HMAC | Stripe webhook — raw-body HMAC verified |
| POST | /billing/nowpayments/create-invoice | JWT | Crypto invoice via NOWPayments |
| POST | /billing/webhook/nowpayments | HMAC | NOWPayments IPN callback |

## 6. Commissions (`/commissions`) — `commissions.controller.ts`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /commissions/summary | JWT | Own commission summary |
| GET | /commissions | JWT | Own ledger (filterable) |
| GET | /commissions/admin/all | Admin | Full ledger page |
| GET | /commissions/admin/summary | Admin | Totals + status breakdown |
| POST/PUT | /commissions/admin/rules… | Admin | Rule CRUD (`admin/rules`, clear/hold/release per entry) |

## 7. Payouts (`/payouts`) — `payouts.controller.ts`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /payouts | JWT | Own payouts |
| POST | /payouts/request | JWT | Request withdrawal (idempotency-keyed) |
| GET | /payouts/admin/all | Admin | Queue |
| GET | /payouts/admin/stats | Admin | Totals by status |
| POST | /payouts/{id}/approve · /reject · /process · /reconcile · /retry | Admin | State machine transitions |

## 8. Referrals (`/referrals`) — `referrals.controller.ts`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /referrals/code | JWT | Referral code + link |
| GET | /referrals/network | JWT | Downline tree + stats |
| GET | /referrals/commissions | JWT | Per-referral earnings |

## 9. Licensing (`/licensing`) — `licensing.controller.ts`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | /licensing/validate | Public/agent | Validate a license key (optionally device-bound) → `{valid, status, plan, allowed_strategies}` |
| GET/POST | /licensing/devices | Agent/Admin | List / register devices (hardware fingerprint) |
| POST | /licensing/devices/{id}/heartbeat · /revoke | Agent/Admin | Telemetry / revoke |
| GET/POST | /licensing/mt-accounts | Agent/Admin | MT account bindings |
| GET | /licensing/licenses | Admin | All licenses + status |

Device-auth suite (`/devices`): `activate`, `refresh`, `heartbeat`, `sessions`, `devices/{id}`,
`devices/{id}/revoke` — used by the Windows Agent for activation sessions.

## 10. Operations (`/operations`) — Admin — `operations.controller.ts`

| Method | Path | Description |
|--------|------|-------------|
| GET | /operations/state | Trading state (halted? signals paused?) |
| GET | /operations/active | Active platform operations |
| POST | /operations/halt-trading · /resume-trading | Full execution halt (body `{reason}` required) |
| POST | /operations/pause-signals · /resume-signals | Signal generation pause (body `{reason}` required) |
| POST | /operations/strategy/{id}/enable · /disable | Per-strategy kill switch |
| GET | /operations/ai/models · /ai/training-jobs · /ai/inference | ML registry views |
| POST | /operations/ai/model/{id}/activate · /deactivate | Model activation (cannot self-promote) |

## 11. Admin (`/admin`) — `admin.controller.ts` + `admin-extras.controller.ts`

| Method | Path | Description |
|--------|------|-------------|
| GET | /admin/overview | User/subscription/commission headline metrics |
| GET/PATCH | /admin/users · /admin/users/{id}/status | User management |
| GET | /admin/health · /admin/subscriptions · /admin/commissions(+summary) · /admin/payouts(+stats) · /admin/plans · /admin/licenses · /admin/devices · /admin/activations | Console data grids |
| GET | /admin/subscriptions/{payments|refunds|chargebacks|coupons|provider} | Billing ops |
| GET | /admin/trading-reports | Aggregated from `trading.trade_results` (real fills only) |
| GET | /admin/regime-diagnostics *(realtime)* | Regime engine state |
| GET/PUT | /admin/risk-config | Global risk configuration |
| GET | /admin/signal-accuracy | Signal quality metrics |
| GET | /admin/backup-dr · /admin/releases · /admin/broker-qualification · /admin/macro-news | Admin-extras views |
| GET/PUT | /admin/feature-flags · /feature-flags/{id} | PTB feature-flag registry (`trading.ptb_feature_flags`; mode `OFF|SHADOW|ACTIVE|DISABLED|UNSUPPORTED|RESEARCH`, `id` is a UUID) |

## 12. Audit & Compliance

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /audit | JWT | Own client events (paginated) |
| GET | /audit/client | JWT | Alias — activity log page |
| GET | /audit/events | Admin | Full audit stream |
| — | GDPR erase/anonymize/retention | Admin only | `compliance/gdpr.service.ts` (migration 088) |

## 13. Backtests (`/backtest`) — `backtest.controller.ts`

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /backtest/data | JWT | Available datasets + history (synthetic or TimescaleDB-sourced) |
| GET | /backtest/runs | JWT | Own runs |
| GET | /backtest/runs/{runId} | JWT | Run + trades |
| POST | /backtest/run | JWT | Start backtest (engine-validated config) |
| GET | /backtest/runs/{runId}/download | JWT | CSV export |

## 14. Health & Guest

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /health | None | Control plane liveness + DB check |
| GET | /api/docs | Internal | Swagger UI (production: blocked externally) |

Guest-preview suite (`/guest`): `session`, `status`, `register`, `otp/resend`, `otp/verify`,
`unsubscribe`, `unsubscribe-status` — anonymous 5-minute live preview funnel.

---

## 15. Go Realtime Engine

### REST (direct :13081, or via `api.predictatrade.com` edge)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | /health | None | `{status:"ok", broker_time, time_mode:"UTC_ALIGNED"}` |
| GET | /api/v1/signals | JWT* | Signals (limit ≤200). Admin sees all; user sees entitled strategies; anonymous sees ADVISORY-only |
| GET | /api/v1/trades | JWT | Real fills from `trading.trade_results` (`?strategy=` filter) |
| GET | /api/v1/market/snapshot | None | Tick + 42 indicators + VWAP bands + session |
| GET | /api/v1/market/state | None | Full market state incl. Regime |
| GET | /api/v1/market/indicators | JWT | Indicators only |
| GET | /api/v1/candles | JWT | Cached candles (TimescaleDB ⇢ Valkey ⇢ in-memory ladder) |
| GET | /api/v1/agents/status | Admin | Agent count, mt4/mt5 links, snapshot_count, `data_health` (NO_DATA/HEALTHY/STALE/CRITICAL), `market_closed` (true when Master EA reports closed-market liveness snapshots), `last_snapshot_at` |
| GET | /api/v1/engines/status | JWT | 5 strategy engines + liveness |
| GET | /api/v1/system-health | Admin | Engine + DB + feeds + agents rollup |
| GET | /api/v1/liquidity/** | JWT | Devil Liquidity marks/qualification |
| GET | /api/v1/cross-market/** | JWT | DXY/BTC/Oil confluence + validation |
| GET | /api/v1/devil-liquidity/marks | JWT | Mark lifecycle states |
| GET | /api/v1/strategies | None | Enabled strategy IDs |
| POST | /api/v1/license/validate | Agent | Proactive license validation (v1.16+) |
| GET | /metrics | Prometheus | Scrape endpoint (`pat_*` metric families incl. `pat_reconciliation_*`, gates, engines) |

### WebSocket

| Path | Port | Auth | Purpose |
|------|:----:|------|---------|
| /ws/v1/agent | 13081 | Agent token | Windows **Client Agent** (execution): receives SIGNAL, CLOSE_POSITION, EMERGENCY_STOP, KILL_SWITCH, LICENSE_STATUS, REQUEST_SNAPSHOT; sends EXECUTION_ACK, TRADE_RESULT, SLIPPAGE_EVENT, CAPITAL_WARNING |
| /ws/v1/data | 13091 | Agent token | Windows **Master Node** (data-only): MARKET_SNAPSHOT, MASTER_TICK, MASTER_INIT/DEINIT, LICENSE_CHECK |
| /ws/v1 | 13081 | JWT | Browser signal stream (dashboard) |
| /ws | platform edge | JWT | Browser realtime (live-terminal relay) |

**Server → client events:** `signal`, `signal_update`, `tick`, `engine_status`, `market_alert`,
`LICENSE_STATUS`, `CLOSE_POSITION`, `EMERGENCY_STOP`, `KILL_SWITCH`, `REQUEST_SNAPSHOT`; client EAs also emit a `LIVENESS` ping (connectivity-only, `market_closed:true`) that the agent forwards upstream during closed markets, `LIVENESS` (client-EA closed-market connectivity ping — agent→engine) — the status page and agents/status now also expose `market_closed`.

**Signal object** (abridged — full field list in §9 of old reference and `realtime/internal/types`):
`ID`, `Direction`, `StrategyID`, `SignalClass` (ADVISORY|EXECUTABLE), `RawScore`,
`CalibratedProbability` ("Pending" until calibration VALIDATED), `EntryPrice`, `StopLoss`,
`TP1/2/3`, `GrossRRTP1/2/3`, `Regime`, `Session`, `QualityGrade` (A+|A|B|REJECTED),
`ExpectancyR`, `ExpectancyScore`, `SuggestedLot`, `RiskDollars`, `RiskPctOfEquity`,
`SLDistancePoints`, `Evidence{}`, `ReasonCodes[]`, `Executable: bool`, `CreatedAt` (UTC).

**Account-type fields (v1.27, additive everywhere — absent = legacy EA):**
- EA → engine (`INIT`, `ACCOUNT_INFO`, `LICENSE_CHECK`, `EXECUTION_ACK` JSON payloads;
  `edge-heartbeat` body): `account_type` ∈ `Demo|Contest|Islamic|MicroCent|ECN|STP|Standard`
  (+ `account_type_verified`, `account_type_confirms` on heartbeat; `demo:true` extra tag
  on Demo payloads).
- Engine ingest: `SnapshotAccount.AccountType` (Go, `account_type,omitempty`) —
  populated from MasterNode `account_info.account_type` and fleet ACCOUNT_INFO.
- Control `POST /api/v1/devices/edge-heartbeat`: persists `account_type` +
  `account_type_verified` to `licensing.edge_device_state` and `licensing.devices`
  (fail-open; unknown keys tolerated). Reference tables: `licensing.account_types`,
  `licensing.strategy_parameters` (mig 133).

---

## 16. Rate limiting

- Global: 300 req/min/IP via `ThrottlerModule` (auth/registration endpoints have stricter
  per-route overrides).
- Login: 10/min/route; guest-preview OTP: dedicated buckets.
- Agents bypass browser limits (separate authenticated WS routes without `limit_conn`).

## 17. Versioning & compatibility

- All NestJS routes are prefixed `/api/v1` (`app.setGlobalPrefix`).
- Machine-readable errors + `X-Correlation-Id` propagation.
- Additive changes only within `v1`; breaking changes require `v2` prefix (SOW §80).
- Live OpenAPI 3.0 spec: [`openapi.json`](openapi.json) (generated from control decorators).