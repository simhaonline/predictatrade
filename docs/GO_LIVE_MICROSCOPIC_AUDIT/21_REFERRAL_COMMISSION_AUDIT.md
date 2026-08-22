# 21 — Referral / Commission Audit

## Attribution (implemented)

Deterministic code `PAT-<userId>` auto-provisioned at register; registration inserts exactly one level-1 `referral.referral_relationships` row (self-referral blocked by DB CHECK at level 1). Frontend builds `predictatrade.com/register?ref=` links.

## Multi-level design vs implementation

Schema supports levels 1–5 (CHECK + UNIQUE(child,level)); CommissionEngine expects sponsorChain[L1..L5] — **but no code materializes ancestors beyond level 1** ⇒ multi-level is schema-complete, pipeline-incomplete.

## Commission math (engine unreachable — quoted for record)

`effectiveRate = baseRate × purchaseRule.multiplier`; `amount = commissionableAmount × effectiveRate` to 2dp HALF_UP; level cap = min(rule.maxLevel, chain, 5); FIRST×1.0 / SECOND×0.75 L1-only / RECURRING×0.50 rules; rule snapshots embedded per ledger row. **Spec-vs-seed mismatch**: tests validate STANDARD [10,4,2,1,0.5]% while DB seeds [10,3,1,0,0]% and maxLevel 5-vs-3 divergences — money math under test ≠ money math configured.

DB truth: **commission_ledger = 0 rows**; engine class imported only by its spec file. Refund/chargeback classification exists in-engine with no invoker.

## Payouts — broken

```
INSERT INTO referral.payouts (..., amount, ...) → column does not exist (requested_amount)
status 'PENDING' violates state machine ('REQUESTED')
method/destination silently discarded; no balance check; no idempotency key; no transaction
approvePayout flips status without approved_by/approved_amount/fee/net/ledger debit/payout_items
getStats() SUM(amount) would also 42703
```

Anti-abuse tables (`affiliate_risk_flags`: SELF_REFERRAL/CIRCULAR/COMMISSION_FARMING/IP_CORRELATION) have zero code references.

## Verdict

REFERRALS = NOT VERIFIED (cannot pay a referrer today). Positive: ledger-first immutable design + dedup indexes are correct-by-construction once writers exist; exact-decimal preserved end-to-end.
