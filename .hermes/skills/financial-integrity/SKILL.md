---
name: financial-integrity
description: "Ensure exact-decimal ledgers and audit trails."
---

# financial-integrity

## Scope
Subscription billing, referral commissions, payout processing, ledger reconciliation.

## Rules
1. All money values stored as NUMERIC(38,18) — never float/double.
2. Use decimal.js in NestJS, shopspring/decimal in Go.
3. Every financial mutation is transactional and idempotent (idempotency_key).
4. Never rewrite financial history — use compensating/reversal records.
5. Financial ops isolated from Go trading hot path.
6. Frontend renders backend ledger outcomes; never recodes finance.

## Known Issues (full-audit.md)
- C1 (CRITICAL): NOWPayments IPN HMAC wrong — payments never settle
- C3 (CRITICAL): Payout double-spend via CLEARED row reservation bug
- C2 (CRITICAL): finance.ledger_entries not in migrations (075 fixes it)
- M1 (MED): Commission credit not transactional
- M2 (MED): No subscription cancel/refund state machine
