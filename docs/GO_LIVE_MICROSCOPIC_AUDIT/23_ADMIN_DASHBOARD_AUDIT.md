# 23 — Admin Dashboard Audit

Pages (all under `/admin/*`, Next.js client components polling REST): dashboard, users, subscriptions, billing, referrals, licenses, logs, health, operations, strategies, scoring-board, signals, regime-diagnostics, indicator-monitor, device-auth, trading-reports.

## Route→API→RBAC matrix (verified)

| Page | APIs | Server guard |
|---|---|---|
| /admin/dashboard | /admin/overview, /operations/state, /signals, /market/state, /agents/status, /health + WS | overview/state/ops: AdminGuard ✅; Go endpoints: **none** |
| /admin/users | /admin/users CRUD-lite, detail, assign-license | AdminGuard ✅ |
| /admin/billing | /admin/subscriptions, /admin/commissions(+summary), /admin/payouts(+stats), POST approve | AdminGuard ✅; approve is broken backend (21) |
| /admin/logs | /audit | 401 anonymous reproduced ✅ |
| /admin/regime-diagnostics | Go `/api/v1/admin/regime-diagnostics` | **NO GUARD on Go side** ❌ |
| /admin/device-auth | /agents/status, /market/snapshot, /admin/devices | mixed |

## Data honesty

- Financial tables render server ledger strings verbatim (no recomputation) ✅.
- Loading/error/empty states solid via DataTable component ✅.
- Defects: `engineAlive` tautology ⇒ RT engine always "Operational" on any HTTP 200 (`admin/dashboard/page.tsx:108`); decorative hardcoded "12 Hard Gates" panel with no data binding (`scoring-board/page.tsx:162-173`); RawScore=prob×100 fabrication for WS rows; MFA challengeId dropped between login page and verify-otp page.
- Payout approval = single unconfirmed click relying on (broken) backend idempotency.

## Admin capabilities vs §45

Plan status/quota/strategy flags are DB-seeded but there is **no admin UI/API to edit entitlements or feature flags** — configuration changes require SQL. Every admin mutation audited via audit.audit_events where implemented (user status, license assign, ops halt/resume) — but audit insert failures are swallowed.

**Status: PARTIAL** — genuine data + guards on NestJS plane; Go-plane admin surfaces unguarded; several cosmetic-but-misleading panels.
