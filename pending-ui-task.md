# Pending Backend Tasks — Upgrade Degraded UI to REAL

Generated: 2026-08-23
Purpose: Each item below is a UI feature already built (frontend) that currently renders an honest
"Degraded / endpoint-pending" state because the corresponding backend capability does not exist.
Implementing these converts those pages from PARTIAL/STUB to REAL per `ui-report.md` §0.

Convention: all endpoints require authenticated, role-checked access (tenant isolation, RBAC).
Financial endpoints must be exact-decimal, transactional, idempotent, and ledger-backed (SOW §Financial Integrity).
No endpoint may fabricate data or mutate live trading.

---

## 1. User Dashboard Backend

### 1.1 User Sessions / Login History (`/dashboard/security`)
- [ ] `GET /device-auth/sessions/me` — return current user's own active sessions (currently admin-guarded only).
- [ ] `GET /audit/login-history/me` — user-scoped login events (actor = self) for the Login History tab.
- [ ] `POST /device-auth/sessions/:id/revoke` — user-scoped revoke of own session/device.

### 1.2 Password Change (`/dashboard/security`, `/dashboard/settings`)
- [ ] `POST /users/me/password` — authenticated password change (verify current + set new). `PATCH /users/me` currently only accepts `displayName`/`timezone`.

### 1.3 Strategy Selection Persistence (`/dashboard/strategies`)
- [ ] `PATCH /subscriptions/me/strategies` — persist user-selected/enabled strategies against server entitlement.
- [ ] `GET /subscriptions/me/strategies` — return current selection (or extend `/subscriptions/entitlements`).

### 1.4 Billing Cancel / Downgrade / Auto-Renew (`/dashboard/billing`)
- [ ] `POST /subscriptions/me/cancel` — cancel with effective-date + reason.
- [ ] `POST /subscriptions/me/downgrade` — plan change (entitlement-checked).
- [ ] `PATCH /subscriptions/me` — toggle `autoRenew`; expose `nextBillingDate`, `billingCycle`.

### 1.5 Notifications Preferences (`/dashboard/notifications`)
- [ ] `GET /users/me/notifications` + `PUT /users/me/notifications` — persist category preferences (signals/trades/billing/referrals/system).

### 1.6 Support / Tickets (`/dashboard/support`)
- [ ] `POST /support/tickets` + `GET /support/tickets/me` — user ticket create/list (currently mailto-only).

---

## 2. Admin Commercial / Operations Backend

### 2.1 Plans Write API (`/admin/plans-entitlements`)
- [ ] `POST /plans`, `PATCH /plans/:id`, `DELETE /plans/:id` — currently `GET /plans`, `GET /plans/:id` only.
- [ ] Persist plan features, price, billing cycle, strategy availability, entitlement mapping.

### 2.2 Commission Rule Configuration (`/admin/commission-control-center`)
- [ ] `GET /commissions/rules` + `PUT /commissions/rules` — base rate, L1–L5 levels, plan base rates.
- [ ] Must derive commission only from the canonical eligible revenue policy (SOW §Financial Integrity).

### 2.3 Commission Operations (`/admin/commission-operations`)
- [ ] `POST /commissions/admin/:id/hold`, `/release`, `/reverse`, `/adjust` — currently read-only (`/commissions/admin/all`, `/commissions/admin/summary`).
- [ ] All ops ledger-backed with compensating/reversal records; idempotent.

### 2.4 Payout Operations Workflow (`/admin/payout-operations`)
- [ ] `POST /payouts/:id/reject`, `/process`, `/reconcile`, `/retry`, `/cancel` — currently only `POST /payouts/:id/approve` exists.
- [ ] `GET /payouts/admin/export` (server CSV) optional; client CSV already implemented as fallback.
- [ ] Status state-machine + audit trail per payout.

### 2.5 License CRUD (admin) (`/admin/licenses`)
- [ ] `POST /admin/licenses` (create), `PATCH /admin/licenses/:id` (suspend/revoke/renew/reset),
      `POST /admin/licenses/:id/force-logout`, `GET /admin/licenses/:id/activations` (history).
- [ ] Currently only `GET /admin/licenses` + user-scoped `/licensing/*` exist.

### 2.6 Device Auth Admin Actions (`/admin/device-auth`)
- [ ] `POST /admin/devices/:id/reset`, `/force-upgrade`, `/disable-signal` (revoke already exists via `/licensing/devices/:id/revoke`).

### 2.7 Risk Center Config Persistence (`/admin/risk-center`)
- [ ] `GET /operations/risk-config` + `PUT /operations/risk-config` — kill switches
      (strategy/account/broker/symbol), exposure/spread/slippage/drawdown/daily-loss limits,
      session & news-blackout config.
- [ ] Live Halt/Pause already wired; this adds durable config storage (currently local/optimistic).
- [ ] Expose the 12 hard-gate statuses via a stable endpoint if not already (reuse market-state/scoring).

### 2.8 Audit Server-Side Search (`/admin/logs`)
- [ ] `GET /audit?actor=&action=&entity=&ip=&oldState=&newState=&reason=&from=&to=` — currently only full list; UI does client-side filter.

### 2.9 Billing Detail (`/admin/billing`)
- [ ] `GET /billing/refunds`, `/billing/chargebacks`, `/billing/coupons`, provider references — currently only `/billing/invoices`.

---

## 3. Admin Infrastructure Backend

### 3.1 AI Providers Management (`/admin/ai-providers`)
- [ ] `GET/POST/PATCH/DELETE /ai/providers` (add/edit/delete provider config). Model activate/deactivate already exists.

### 3.2 Market Data Feed-Health (`/admin/market-data`)
- [ ] `GET /market/feed-health` — divergence, tick rate, latency, candle health, backfill status.
- [ ] `/market/snapshot` already REAL; this adds monitoring sub-metrics.

### 3.3 Macro / News (`/admin/macro-news`)
- [ ] `GET /macro/dxy`, `/macro/yields`, `/macro/calendar`, `/news/feed`, `/news/blackouts`, `/macro/provider-health`
      (or a single aggregated `/macro/summary`). Provider integration + timezone/DST-aware calendar required.

### 3.4 Releases Registry (`/admin/releases`)
- [ ] `GET/POST/PATCH /releases` — client release versions, checksums, signatures, rollback metadata.

### 3.5 Backup / DR Status (`/admin/backup-dr`)
- [ ] `GET /system/backup-dr` — last run, status, RPO/RTO, restore-test result (read from backup orchestration).

### 3.6 Feature Flags (`/admin/feature-flags`)
- [ ] `GET/PUT /config/feature-flags` — flag key/value/env/description with persistence + env scoping.

### 3.7 Broker Execution Qualification (`/admin/broker-qualification`)
- [ ] `GET/POST /broker-qualification` — per-broker/strategy measured economics
      (spread, slippage, margin, latency, reject rate, locality). Backed by broker-execution-qualification module.

---

## 4. Verification Gate (before marking UI REAL)
For each endpoint above:
- [ ] Controller + service + DTO + authz (RBAC/tenant) implemented.
- [ ] DB migration if new state (exact-decimal financial types; TimescaleDB for time-series).
- [ ] Unit + integration tests; financial ops idempotent + ledger-backed.
- [ ] Frontend degraded guard removed and wired to live data; honest state retained only on error.
- [ ] `npm run build` + `npm run typecheck` pass; relevant e2e updated.
