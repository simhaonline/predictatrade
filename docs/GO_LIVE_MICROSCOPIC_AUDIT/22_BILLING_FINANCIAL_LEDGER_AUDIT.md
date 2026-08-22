# 22 — Billing / Financial Ledger Audit

## Checkout & payment

No PSP SDK in control (`package.json` clean of stripe/paypal/razor). `POST /api/v1/billing/webhook` is **unauthenticated** and its handler logs + acks (`billing.service.ts:19-22`). No code writes invoices/payments/subscriptions transitions.

## Production-data proof of bypass

```
billing.payments:        1 row — SUCCEEDED amount=49.00 provider=STRIPE ext=in_local_invoice_1787381953
billing.payment_events:  event_type=invoice.paid, signature_verified=false,
                         raw_payload={"id":"evt_local_invoice_1787381953",...}   ← fabricated locally
→ billing.subscriptions: the ONLY ACTIVE subscription (ELITE) chains to this payment
```

An unverified, locally-crafted webhook-shaped row activated a paid plan ⇒ §49 "never mark active because browser returned" is violated in spirit and letter; §100 "payment verification bypass" confirmed.

## Ledger architecture (schema)

`referral.commission_ledger` immutable + additive adjustments; wallets documented as reconcilable cache; status machine PENDING→CLEARED→AVAILABLE→PAID/REVERSED; dedup partial-unique index; v3 events with external_reference idempotency. **All correct-by-design, zero writers.**

## Invoices

1 invoice tied to fake payment; unique-number generation exists in schema triggers/read-models migration 025; PDF path UNVERIFIED (no generator found).

## Refund/chargeback/dunning

Absent entirely (no endpoints, no writers).

## Verdict

BILLING = NOT VERIFIED / FAILING. Revenue cannot be collected legitimately; existing financial rows include at least one fabricated event; ledger integrity is untested because unused.
