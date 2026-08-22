# Predict-A-Trade — Codex Advice

## Overall opinion

The architecture is salvageable, but production billing, payouts, and user-scoped signal access should remain disabled until the critical security and commercial controls are proven.

## Recommended order of work

1. Rotate the exposed provider API key immediately. Remove it from `opencode.json`, scan repository history, and use injected secrets only.

2. Fix authorization before additional UI work:
   - derive WebSocket identity from authenticated tokens, never a `userId` query parameter;
   - enforce entitlements in APIs and WebSocket serialization;
   - test Free, Standard, Pro and Elite response leakage.

3. Implement payment processing as an idempotent state machine:
   - verify webhook signatures;
   - persist provider event IDs;
   - handle duplicate and out-of-order events;
   - support renewal, downgrade, expiry, refund and chargeback transitions.

4. Make PostgreSQL authoritative for quotas and finance. Valkey should accelerate reads, but never determine financial truth. Quota consumption must use atomic durable records.

5. Independently reconcile the full financial chain:

   `provider payment → payment row → subscription → entitlement → referral → commission ledger → payout`

6. Add deterministic tests for concurrent quota consumption, duplicate webhooks, duplicate commissions, refunds, chargebacks, self-referrals, circular referrals and payout retries.

7. Do not rewrite the whole system. Extend the existing NestJS modules, migrations, Go entitlement gates and ledger structures with compatibility-preserving changes.

8. Reconcile stale or deleted documentation and repository state before release so operational decisions use one dated source of truth.

## Immediate release position

```text
Production billing: NO
Real commission payouts: NO
Live automated trading: NO
Architecture rewrite: NO
P0 security/commercial remediation: YES
```

The next safe milestone is completion of P0 security and commercial remediation with evidence from isolated tests or a provider sandbox. Only then should broader release-gate validation begin.
