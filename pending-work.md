# Pending Work — `prompt.md` Compliance

Audit date: 2026-08-22  
Scope: read-only audit of the 3,182-line `prompt.md` against the current repository.  
Current status: **BLOCKED for full acceptance; PARTIAL implementation**.

## P0 — Security and authorization blockers

- [ ] Bind dashboard WebSocket identity to the authenticated session/token. The current gateway accepts a caller-supplied `userId` query parameter.
  - Evidence: `realtime/internal/gateway/websocket.go`
  - Acceptance: unauthenticated or impersonated clients cannot receive user-scoped events; identity is server-derived and tested.
- [ ] Enforce effective entitlements on every user signal, history, analytics, notification, export, and report API.
  - Acceptance: unauthorized fields and records are absent from responses, not merely hidden in the frontend.
- [ ] Implement authenticated WebSocket entitlement loading, topic filtering, entitlement-change refresh, disconnect/reconnect behavior, and downgrade revocation.
  - Acceptance: per-plan and per-user authorization tests pass for all four plans.
- [ ] Implement and test server-side signal-field serialization for Free, Standard, Pro, and Elite.
  - Acceptance: restricted fields such as TP2, TP3, advanced evidence, and internal metadata never reach unauthorized clients.

## P0 — Payment and commercial integrity blockers

- [ ] Implement a real provider adapter with signature verification, idempotency, and validated payment activation.
- [ ] Implement upgrade, downgrade, renewal, expiry, grace-period, refund, chargeback, and cancellation transitions.
- [ ] Connect validated commercial events to effective entitlement recalculation, cache invalidation, and `subscription_changed` delivery.
  - Acceptance: provider-backed integration tests prove the complete lifecycle without manual admin activation.
- [ ] Implement the production signal-distribution adapter that consumes `trading.signal_delivery_ledger`.
- [ ] Implement concurrency-safe Free quota allocation/consumption with PostgreSQL as authority and Valkey as optimization only.
  - Acceptance: concurrent delivery cannot exceed the configured monthly quota; Valkey restart does not reset usage.

## P1 — Required entitlement and admin functionality

- [ ] Add the authoritative dashboard capability manifest endpoint/service, including modules, strategies, fields, limits, history, notifications, and analytics.
- [ ] Add admin plan-entitlement configuration for modules, strategies, fields, history, quotas, devices, API, exports, and notifications.
- [ ] Add admin per-user entitlement inspection and audited grant, revoke, temporary override, quota adjustment/reset, suspension, and restoration controls.
- [ ] Implement entitlement precedence exactly as specified: security suspension, safety switch, subscription state, plan, admin override, strategy selection, quota, preference.
- [ ] Implement automatic downgrade strategy conflict resolution with deterministic fallback and tests.
- [ ] Ensure entitlement/config changes propagate without restart, with versioning, cache invalidation, and affected-user notification.

## P1 — Required dashboard experiences

- [ ] Build the dedicated Free Signals experience; Free users must not receive the normal daily signal feed or paid signal counts.
- [ ] Make user navigation and home modules capability-driven rather than plan-name-driven frontend assumptions.
- [ ] Add locked-state previews that contain no restricted data.
- [ ] Enforce user ownership and retention windows for history, exports, analytics, notifications, reports, devices, and referrals.
- [ ] Complete mobile, responsive, light/dark/system, accessibility, loading, stale, degraded, and error-state acceptance coverage.

## P1 — Required audit and specification artifacts

- [ ] Create `docs/USER_DASHBOARD_V3_FORENSIC_AUDIT.md` with file/component/API/backend/database ownership and KEEP/EXTEND/REPAIR/REBUILD/DEPRECATE/MISSING classifications.
- [ ] Create `docs/USER_DASHBOARD_ACCESS_MATRIX.md` as the authoritative Free/Standard/Pro/Elite entitlement matrix.
- [ ] Add prompt-to-file traceability for all applicable sections and acceptance criteria.
- [ ] Reconcile stale claims in `docs/SUBSCRIPTION_FINAL_IMPLEMENTATION_REPORT.md` and `docs/SUBSCRIPTION_TEST_REPORT.md` with current verified results.

## P2 — Verification and release evidence

- [ ] Add Free, Standard, Pro, and Elite persona tests.
- [ ] Add upgrade, downgrade, admin override, stale-cache, direct-URL, response-leak, WebSocket, quota-concurrency, and browser-network tests.
- [ ] Verify migration constraints, indexes, rollback/forward-fix procedures, and production reconciliation evidence.
- [ ] Complete security review, dependency/SAST/secret checks, observability coverage, backup/restore, DR, and release-gate evidence.
- [ ] Re-run the complete acceptance suite and issue a final PASS/PARTIAL/BLOCKED report.

## Already verified

- Frontend tests: 84 passed.
- Frontend TypeScript: passed.
- Frontend production build: passed.
- Frontend ESLint: 0 errors, 14 warnings.
- Realtime Go tests: passed.
- Docker frontend, control, and realtime services: healthy at audit time.
- Subscription migrations 024 and 025 exist and are additive/effective-dated.
- No production payment, subscription, commission, payout, or live trading mutation was performed.

## Current blockers

Full `prompt.md` acceptance cannot be declared until the P0 security, payment, signal-distribution, and quota items are implemented and tested. Existing plan/referral configuration and dashboard presentation work must remain preserved while those items are completed.
