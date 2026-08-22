# Plans & Referral Summary

Updated: 2026-08-22
Release: `640e0aa`

## Active customer plans

| Plan | Monthly | Annual | Referral eligibility |
|---|---:|---:|---|
| FREE | $0 | N/A | No commission |
| STANDARD | $99 | $990 | L1 10%, L2 3%, L3 1% |
| PRO | $299 | $2,990 | L1 15%, L2 4%, L3 2% |
| ELITE | $699 | $6,990 | L1 18%, L2 5%, L3 2% |

Current v3 L4/L5 rates are explicitly 0%. Historical L4/L5 ledger records and legacy BASIC records remain preserved and are not rewritten.

## Referral event multipliers

- First eligible paid subscription: 100%, levels L1-L3.
- First renewal / second eligible paid event: 75%, L1 only.
- Third and later eligible paid events: 50%, levels L1-L3.
- FREE registration is not a purchase, does not increment purchase number, and creates no commission.

## New dashboard/API behavior

- Client Plans & Subscription shows FREE, STANDARD, PRO, and ELITE with server-configured fees and annual savings.
- Client Strategy Preferences reads server-authoritative entitlements and displays locked strategies.
- Admin Subscription Management and Billing show plan code plus monthly/annual fees.
- Admin Referrals shows effective referral rates from the `/plans` API.
- Commission summaries use the real lifecycle states: PENDING, CLEARED, AVAILABLE, PAID, and REVERSED.
- `/plans` exposes annual fees, calculated annual savings, effective referral rates, and referral event rules.

## Deployment status

- Control and frontend images were rebuilt/restarted for this release.
- Migrations `024_subscription_referral_v3.sql` and `025_subscription_billing_interval.sql` are additive, applied, and recorded as `COMPLETED` in `audit.migration_history`.
- Control `/api/v1/health` returns HTTP 200 after restart. Frontend `/` returns HTTP 307 to `/preview` and serves successfully.
- Payment-provider activation/webhook verification remains an external blocker.
