# SOS.md — Admin & User Dashboard Audit (Predict-A-Trade)

**Date:** 2026-08-23
**Scope:** Frontend dashboard pages under
- `/srv/predictatrade/xauusd/frontend/src/app/(admin)/admin/*`
- `/srv/predictatrade/xauusd/frontend/src/app/(user)/dashboard/*`

**Purpose:** Identify everything PENDING / NON-FUNCTIONAL / PLACEHOLDER / BROKEN so the team can fix it. This is an audit artifact, not a fix.

---

## 1. Methodology

- Every `page.tsx` in both trees was read in full.
- Called endpoints were extracted from `useQuery` / `customInstance` / `lib/*` and matched against NestJS `control/src/modules/**/*.controller.ts` route decorators.
- Status is evidence-based. The codebase is generally honest: missing backends render explicit "backend pending", "intentionally empty", "saved locally only", or "Degraded" banners rather than fabricated data. **Two genuine defects (404 / missing route) were found and verified.**

## 2. Status Legend

| Status | Meaning |
|---|---|
| WORKING | Renders real data; actions wired to existing backend endpoints. |
| PARTIAL | Some functions work; others are honest gaps (empty tab, local-only, disabled). |
| PENDING-BACKEND | A core action/route is missing in the backend; page errors or silently fails. |
| PLACEHOLDER | Pure schema/sample UI, no backend wiring at all. |
| HONEST-EMPTY | Intentionally empty panel with a clear "no backend" note (acceptable). |

---

## 3. ADMIN PAGES (35 pages)

| Page | Status | Gaps / Pending to fix |
|---|---|---|
| `admin/dashboard` | WORKING | — |
| `admin/users` | WORKING | — |
| `admin/licenses` | PARTIAL | **List is live**, but create/suspend/revoke/renew/reset/force-logout/activation-history call `/licensing/licenses/:id/*` and `/licensing/licenses/:id/activations` — **none of these routes exist**. License lifecycle is non-functional (honestly degrades with toast). |
| `admin/subscriptions` | PARTIAL | `payments`, `refunds`, `chargebacks`, `coupons`, `provider` tabs are HONEST-EMPTY (no backend endpoints). |
| `admin/billing` | PARTIAL | `refunds`, `chargebacks`, `coupons` tabs HONEST-EMPTY. Invoice tab works (`GET /billing/invoices`). |
| `admin/plans-entitlements` | PARTIAL | `GET /plans` + `GET /subscriptions/entitlements` live. Plan **Edit/Save** → `POST /plans/:id` does not exist (read-only). |
| `admin/activations` | **PENDING-BACKEND (BUG)** | List live. **Revoke calls `/devices/devices/${id}/revoke` → 404** (should be `/licensing/devices/:id/revoke`). Admin security control silently fails. |
| `admin/ai-providers` | PARTIAL | List/toggle work (`/operations/ai/models`). "Add Provider" form disabled — no create endpoint. |
| `admin/backtesting` | WORKING | — |
| `admin/backup-dr` | PLACEHOLDER | No backend at all. Status cards "placeholder only"; restore-test form disabled. |
| `admin/broker-qualification` | PLACEHOLDER | No backend. Schema-only table; add-run form disabled. |
| `admin/commission-control-center` | PARTIAL | Summary live (`GET .../commissions/admin/summary`). **Save Rules** → `alert("pending backend")`; rule-write not persisted. |
| `admin/commission-operations` | PARTIAL | Ledger live. Hold/Release/Reverse/Adjust disabled → "pending backend". |
| `admin/device-auth` | PARTIAL (revoke broken) | List live. **Revoke uses `revokeDevice()` from `admin-api.ts` which calls `/devices/devices/:id/revoke` → 404**. Reset/Force-Upgrade/Disable-Signal → `/licensing/devices/:id/{reset,force-upgrade,disable-signal}` → **routes do not exist**. |
| `admin/feature-flags` | PLACEHOLDER | No backend. Static sample; toggles disabled. |
| `admin/finance-referral-reports` | WORKING | Derived/estimate values labeled. |
| `admin/health` | WORKING | — |
| `admin/indicator-monitor` | WORKING | Observability only; honest. |
| `admin/indicators` | WORKING | Real snapshot. |
| `admin/logs` | WORKING | `GET /audit` works; search/filter client-side only (honest note). |
| `admin/macro-news` | PLACEHOLDER | No backend. Six panels placeholder; blackout form disabled. |
| `admin/market-data` | PARTIAL | Live snapshot real. Divergence/Tick-Rate/Latency/Candle/Backfill monitoring panels "Pending Backend". |
| `admin/mt-accounts` | WORKING | `GET/POST /licensing/mt-accounts` exist. |
| `admin/operations` | WORKING | Halt/Resume/Pause/Resume all functional. |
| `admin/payout-operations` | PARTIAL | List/stats/Approve work. Reject/Process/Reconcile/Retry/Cancel disabled (pending backend). |
| `admin/referrals` | WORKING | `GET /referrals/network` live; downline tree renders. |
| `admin/regime-diagnostics` | **PENDING-BACKEND (BUG)** | Calls `GET /admin/regime-diagnostics` → **no such route in `admin.controller.ts`**. Page errors on load. |
| `admin/releases` | PLACEHOLDER | No backend. Schema-only; publish form disabled. |
| `admin/risk-center` | PARTIAL | Halt/Resume/Pause/Resume signals work. Kill-switch toggles, numeric limits, session/news blackout, "Save Risk Config" → local-only preview (pending backend). |
| `admin/scoring-board` | WORKING | Real `/market/state` + `/signals`. |
| `admin/settings` | PARTIAL | Profile/MFA endpoints exist. Password "Change" mis-wired to `/auth/mfa/verify` (honest error). Notifications tab empty ("future update"). |
| `admin/settings/accessibility` | WORKING | — |
| `admin/signals` | WORKING | — |
| `admin/strategies` | WORKING | `POST /operations/strategy/:id/enable|disable` exist. |
| `admin/trading-reports` | WORKING | — |

---

## 4. USER DASHBOARD PAGES (16 pages)

| Page | Status | Gaps / Pending to fix |
|---|---|---|
| `dashboard/activity-log` | WORKING | `GET /audit/client` works. |
| `dashboard/backtest` | WORKING | `/backtest/*` exist. |
| `dashboard/billing` | WORKING | Auto-renew toggle local-only (honest note). |
| `dashboard/license` | WORKING | — |
| `dashboard/live` | WORKING | — |
| `dashboard/mt4-mt5-client` | WORKING | Download links static `/downloads/*.exe|.mq4|.mq5` — may 404 if not hosted (minor). |
| `dashboard/notifications` | PENDING-BACKEND | `localStorage` only; no persistence endpoint (honest banner). |
| `dashboard/payouts` | WORKING | `GET /payouts` + `POST /payouts/request` exist. |
| `dashboard/referrals` | WORKING | All endpoints exist. |
| `dashboard/security` | PARTIAL | MFA + trusted devices work. Sessions/Login-History tabs intentionally restricted (honest). |
| `dashboard/settings` | PARTIAL | Password change mis-wired (`updateMyProfile` → honest "not supported" toast). |
| `dashboard/settings/accessibility` | WORKING | — |
| `dashboard/signals` | WORKING | — |
| `dashboard/strategies` | PARTIAL | Reads live entitlements; toggle writes are local-only ("pending backend support"). |
| `dashboard/support` | HONEST-EMPTY | `mailto:` only; no ticket backend (honest). |
| `dashboard/trading-reports` | WORKING | Correctly filters Master-Node accounts (tenant isolation good). |

---

## 5. CRITICAL / VERIFIED DEFECTS (fix first)

### BUG-1 — Device Revoke returns 404 (SECURITY)
- `frontend/.../app/(admin)/admin/activations/page.tsx:43` calls `POST /devices/devices/${id}/revoke`.
- `frontend/src/lib/admin-api.ts:159` `revokeDevice()` calls `POST /devices/devices/${id}/revoke`.
- **Actual backend route:** `licensing.controller.ts:57` → `POST /licensing/devices/:id/revoke` (verified exists & working).
- **Impact:** Admin device revocation silently fails (404). The correct path already exists in `lib/admin-commercial-api.ts:65`.
- **Fix:** Change both call sites to `/licensing/devices/${id}/revoke` (or reuse `admin-commercial-api.revokeDevice`).

### BUG-2 — Regime Diagnostics route missing (REALTIME/DIAGNOSTICS)
- `frontend/src/lib/admin-api.ts:194` calls `GET /admin/regime-diagnostics`.
- `admin.controller.ts` has **no** `regime-diagnostics` route (only `overview`, `users`, `subscriptions`, `commissions`, `payouts`, `licenses`, `devices`, `activations`, `trading-reports`, `health`). The service computes regime stats (`admin.service.ts:589-668`) but no controller exposes it.
- **Impact:** `admin/regime-diagnostics/page.tsx` errors on load.
- **Fix:** Add `@Get('regime-diagnostics')` to `admin.controller.ts` returning the existing `regimeStats` / `by_regime` data.

---

## 6. PRIORITIZED FIX LIST

**P0 — Functional defects (verified, break a control):**
1. BUG-1: Fix device revoke 404 path (`activations` + `admin-api.revokeDevice`).
2. BUG-2: Add `/admin/regime-diagnostics` route.

**P1 — Financial lifecycle backends (honestly labeled, but unbuilt):**
3. License write lifecycle: create/suspend/revoke/renew/reset/force-logout + activation-history (`/licensing/licenses/:id/*`).
4. Device security actions: reset / force-upgrade / disable-signal (`/licensing/devices/:id/*`).
5. Commission lifecycle ops: hold/release/reverse/adjust (`commission-operations`).
6. Payout lifecycle ops: reject/process/reconcile/retry/cancel (`payout-operations`).
7. Commission rule-write persistence (`commission-control-center` Save Rules).
8. Risk config persistence: kill-switches, numeric limits, session/news blackout (`risk-center` Save).

**P2 — Persistence / minor:**
9. `user/notifications` backend preference store (currently localStorage).
10. `user/strategies` + `user/billing` auto-renew server-side persistence.
11. `plans-entitlements` plan edit/write endpoint.
12. Subscriptions tabs: payments/refunds/chargebacks/coupons/provider endpoints (currently honest-empty).
13. `admin/settings` password-change endpoint (currently mis-wired to MFA verify).

**P3 — Placeholder features (track as unbuilt, non-blocking):**
14. `backup-dr`, `broker-qualification`, `feature-flags`, `macro-news`, `releases` — wire real backends.
15. `market-data` monitoring sub-metrics (divergence/tick-rate/latency/candle/backfill).
16. `admin/ai-providers` create provider endpoint.

---

## 7. Consolided Counts

| Status | Admin | User | Total |
|---|---|---|---|
| WORKING | 16 | 11 | 27 |
| PARTIAL | 12 | 3 | 15 |
| PLACEHOLDER | 5 | 0 | 5 |
| PENDING-BACKEND (incl. 2 bugs) | 2 | 2 | 4 |
| HONEST-EMPTY (sub-state of PARTIAL) | — | — | (inside PARTIAL) |

**Totals:** 51 pages audited — 27 WORKING, 15 PARTIAL, 5 PLACEHOLDER, 4 PENDING-BACKEND.

## 8. Notes / Positives
- No fabricated financial/trading data detected. Gaps are explicitly labeled ("backend pending", "intentionally empty", "saved locally only", "Degraded").
- Tenant isolation good (user trading-reports filters Master-Node accounts).
- Core realtime/operations (halt/resume/pause signals, strategies, signals, market snapshot, health) is functional.

---

*Generated by audit. Not committed automatically — review then apply fixes per priority above.*
