---
name: control-plane-saas
description: "Implement NestJS IAM, billing, referrals, payouts."
---

# control-plane-saas

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Modules (control/src/modules/)
- auth/ (IAM, MFA, RBAC, JWT), users/, admin/, backtest/, billing/ (NOWPayments, Stripe)
- licensing/, device-auth/, referrals/, commissions/, payouts/, audit/

## Workflow
1. Server-side tenant/role/plan/strategy/license/device/account authorization.
2. Idempotent verified subscription/payment webhooks and versioned plans.
3. Five-level referral/qualification/commission from backend policy.
4. Exact-decimal transactional immutable ledgers and compensating reversals.
5. Isolate commercial failures from Go trading path.

## Known Issues (full-audit.md)
- C1 (CRITICAL): NOWPayments IPN HMAC mismatch - payments never settle
- C3 (CRITICAL): Payout double-spend - CLEARED never transitions to RESERVED
- H1 (HIGH): JWT secret dual-source
- H3 (HIGH): Backtest cross-tenant IDOR
