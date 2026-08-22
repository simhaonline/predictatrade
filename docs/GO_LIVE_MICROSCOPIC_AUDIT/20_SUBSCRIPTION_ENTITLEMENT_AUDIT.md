# 20 — Subscription / Entitlement Audit

## Plans (from live DB `control.plans`, 5 rows)

FREE $0 · BASIC hidden/legacy/billing-disabled · STANDARD $99/mo ($990/yr) · PRO $299/mo ($2990/yr) · ELITE $699/mo ($6990/yr).
Entitlement JSON per plan (strategy flags, max_active_slots 1/2/4, signal.monthly_delivery_limit FREE=5, history.days 7+, api.access=false even ELITE). Annual savings computed server-side. Feature flags SUBSCRIPTION_V3/FREE_PLAN/ANNUAL/REFERRAL_V3 all FALSE in DB and unread by code.

## Entitlement matrix (as configured)

| FEATURE | FREE | STANDARD | PRO | ELITE |
|---|---|---|---|---|
| Strategies | STD_SCALP only | scalping+swing | +more slots | 4 slots |
| Max active slots | 1 | 2 | —(2) | 4 |
| Monthly delivery limit | 5 | higher tiers | higher | highest |
| History days | 7 | more | more | most |
| api.access | false | false | false | false |

## Enforcement reality

| Layer | Status | Evidence |
|---|---|---|
| Strategy selection at subscription create | ENFORCED server-side | `entitlement-policy.ts:24-48` named rejections |
| REST signal delivery | **NOT ENFORCED** | anonymous curl to `/api/v1/signals` returns full payloads (reproduced) |
| WebSocket delivery | **BROKEN-BOTH-WAYS** | entitlement filter exists but never populated ⇒ no WS signals for anyone; agents get unfiltered broadcast |
| Quota counters | **DOES NOT EXIST** | `signal_delivery_ledger` 0 rows; no writer anywhere; limits decorative |
| Backtest/API-access tiers | NOT ENFORCED | endpoints ignore entitlements (`api.access:false` yet backtest API open to any JWT) |
| Guest preview | CSS-only for content | data delivered then blurred (24) |

## Activation integrity

Subscription create sets status INCOMPLETE unless FREE. No webhook processor/cron/admin endpoint transitions INCOMPLETE→ACTIVE; `UPDATE billing.subscriptions` appears NOWHERE in control or Go code. The single ACTIVE row was created via the unverified fake Stripe event path (F-005). Upgrade/downgrade/expiry/dunning/proration: absent. Cache invalidation on plan change: n/a (no cache).

**Verdict: SUBSCRIPTIONS = NOT VERIFIED.** A paying product cannot be sold legitimately, and free/paid boundaries are unenforced on the delivery surface that matters.
