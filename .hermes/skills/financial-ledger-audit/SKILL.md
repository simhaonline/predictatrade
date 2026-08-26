---
name: financial-ledger-audit
description: "Audit ledgers for double-entry correctness."
---

# financial-ledger-audit

Use when auditing PAT financial ledgers.

Tables: finance.ledger_entries (migration 075), billing, commissions, payouts

Checks: double-entry (debits==credits), balance integrity, no negative inventory, idempotency, NUMERIC(38,18), audit trail

Critical issues: C1 (NOWPayments HMAC), C2 (ledger table), C3 (payout double-spend)

Reconcile: billing vs Stripe/NOWPayments daily, commissions vs tiers weekly, payouts vs disbursements per cycle
