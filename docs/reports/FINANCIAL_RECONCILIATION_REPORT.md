# Financial Reconciliation Report — 2026-08-23

## Reconciliation Chain
```
Provider event → Payment → Invoice → Subscription → Entitlement
→ Signal access → Referral attribution → Commission → Ledger → Payout
```

## Current State
- **Payments**: billing.payments table exists, no live payments processed
- **Invoices**: billing.invoices table exists, no live invoices generated
- **Subscriptions**: billing.subscriptions table exists, seed data only
- **Commissions**: referral.commission_ledger exists, commission-engine.spec.ts passes
- **Payouts**: referral.payouts table exists, no live payouts processed

## Reconciliation Checks
| Check | Status | Evidence |
|-------|--------|----------|
| Payment → Invoice link | SCHEMA OK | invoice_id FK exists |
| Invoice → Subscription link | SCHEMA OK | subscription_id FK exists |
| Subscription → Entitlement | CODE OK | entitlement-policy.ts exists |
| Commission → Payment link | SCHEMA OK | source_subscription_id, purchase_id |
| Commission idempotency | CODE OK | idempotency_key column exists |
| Payout → Commission link | SCHEMA OK | payout_items reference commissions |
| Ledger reconstruction | CODE OK | commission_ledger is append-only |

## External Blockers
- No live payment provider connected (Stripe/Adyen sandbox required)
- No real commissions generated (requires active subscriptions)
- No real payouts (requires available balance + payout provider)
