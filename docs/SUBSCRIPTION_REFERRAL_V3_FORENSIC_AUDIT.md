# Subscription / Referral v3 Forensic Audit

Date: 2026-08-22
Scope: `prompt.md` commercial subscription, entitlement, signal distribution, and referral/commission requirements.

## Current architecture

```text
Payment webhook (BillingController)
  -> BillingService.handleWebhook (currently acknowledgment-only)

PlansController -> PlansService -> control.plans + control.plan_entitlements
SubscriptionsController -> SubscriptionsService -> billing.subscriptions
CommissionsController -> CommissionsService -> referral.commission_ledger
ReferralsController -> ReferralsService -> referral relationships/ledger

Go realtime engine -> cached hard safety gates (realtime/internal/gates)
                         [no commercial distribution authorization found in audit]
```

## Component inventory and disposition

| Component | Current implementation | Desired v3 behavior | Status |
|---|---|---|---|
| `control.plans` | Migration 003 stores code, monthly price, strategy slots, allowed strategies, status | Add annual/visibility/legacy/billing metadata while preserving IDs and history | EXTEND |
| `control.plan_versions` | Versioned monthly/strategy configuration exists | Preserve and extend for annual price where needed | EXTEND |
| `control.plan_entitlements` | JSONB entitlement values keyed by plan | Reuse as the data-driven capability source | KEEP/EXTEND |
| `billing.subscriptions` | Durable subscription table; historical columns are from migration 003 | Add safe v3 event/interval/legacy compatibility fields | EXTEND |
| `billing.subscription_events` | Only subscription-scoped event and metadata columns | Add explicit commercial event fields and idempotency | EXTEND |
| `billing.payments` / `payment_events` | Durable payment and provider-event uniqueness | Reuse; commission from validated eligible amount | KEEP/EXTEND |
| `BillingService.handleWebhook` | Logs and returns `{received:true}`; does not validate or mutate state | Provider-specific verification and idempotent state transition | MODIFY (not safely completed without provider contract/credentials) |
| `referral.referral_relationships` | Five-level relationship storage and cycle checks in service helper | Preserve sponsor attribution and five-level capability | KEEP/EXTEND |
| `referral.commission_rules` | Effective-dated plan/level rates; existing seeds include legacy L4/L5 | Add effective-dated v3 rules with L1-L3 only | EXTEND |
| `referral.purchase_commission_rules` | Configuration-driven multiplier/depth table | Add v3 event rules and retain legacy purchase rules | EXTEND |
| `referral.commission_ledger` | Immutable snapshots and dedup index exist | Add event/eligible-amount/idempotency snapshots without rewriting history | EXTEND |
| `trading.signal_deliveries` | Per-device execution delivery table | Add separate user commercial delivery ledger; do not alter Go signal generation | NEW |
| Strategy authorization | `realtime/internal/gates` protects execution entitlement/license gates; no user strategy selector found | Backend validator for selected strategies and plan capabilities | NEW (control-plane library) |
| WebSocket/notification authorization | Existing realtime/license paths audited; no plan-level user distribution enforcement identified | Enforce before user distribution in the actual delivery adapter | MISSING / BLOCKED pending adapter location |
| User billing UI | Invoice history only | Plans & Subscription plus management actions | MODIFY |
| User referral UI | Ledger summary/list; status query uses legacy `CONFIRMED` | Authoritative v3 summary and empty states | MODIFY |
| Admin commercial UI | Existing admin pages/routes present; no v3 configuration evidence found | Add audited plan/rule/event visibility | EXTEND |
| Migration runner | Ordered SQL files, migration history, no down migration | Additive, idempotent forward migration | KEEP/EXTEND |

## Existing data and compatibility observations

- The repository is on `main` with a clean worktree at audit time.
- Migration 003 uses exact `DECIMAL(18,8)` monetary fields and persists plan, subscription, payment, invoice, and payment-event history.
- Migration 004 already supports five-level referrals, effective-dated commission rules, purchase rules, immutable commission snapshots, adjustments, wallets, payouts, and fraud flags.
- Existing commission seed rates are not the requested v3 rates. They must not be overwritten because they may represent historical v1/v2 behavior.
- Existing `CommissionEngine` is unit-tested but is configured by callers and currently permits L4/L5 whenever callers seed them; v3 must use separate effective-dated configuration.
- The current subscription service references `billing_cycle`, `current_period_start`, and `current_period_end`, while migration 003 defines `billing_period_start` and `billing_period_end`. This is a runtime defect to repair with the compatible schema names, not a schema rewrite.
- Free registration and provider-specific webhook semantics cannot be proven from the repository alone; no payment provider adapter or signed-provider verification implementation was found in the audited module.

## Change-impact map

Commercial configuration affects control-plane reads/writes, subscription transitions, distribution authorization, referral event classification, and dashboards. It must not import or modify the Go strategy, scoring, probability, risk, or signal-generation path. Historical billing/referral rows remain immutable; v3 records are effective-dated and additive.

## Verification commands identified

- Control plane: `npm test`, `npm run build`, `npm run lint` in `control/`.
- Realtime: `go test ./...` in `realtime/`.
- Frontend: `npm test` / `npm run build` in `frontend/` (scripts must be checked before execution).
- Database: `./scripts/migrate.sh status` and `./scripts/migrate.sh test` when PostgreSQL is available.
- Compose: `docker compose config` and service-specific builds, subject to Docker daemon/runtime availability.

## Genuine blockers recorded by audit

1. A signed payment-provider webhook contract/credentials are not present in the repository, so payment activation cannot be truthfully claimed.
2. A user-facing signal distribution adapter (separate from execution delivery) was not located, so WebSocket/email/push authorization cannot be proven end-to-end from this audit.
3. Production database counts and migration state require access to the running PostgreSQL instance; no production mutation is performed by this change.
