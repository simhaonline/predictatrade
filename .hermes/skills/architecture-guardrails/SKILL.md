---
name: architecture-guardrails
description: "Enforce Go/Python/NestJS/Next.js plane boundaries."
---

# architecture-guardrails

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Five Planes
1. **Go — Real-Time Trading**: authoritative market, feature, strategy, signal, risk, execution, delivery. No billing/referral/commercial deps in hot path.
2. **Python — Research**: datasets, backtesting, walk-forward/OOS, calibration, ML. Not mandatory per tick.
3. **NestJS — Control**: IAM/MFA/RBAC, subscriptions, billing, licensing, referrals, commissions, payouts, audit.
4. **Next.js — Presentation**: server-authoritative truth. Never risk/strategy/entitlement/finance authority.
5. **Windows/MQL Edge**: lightweight execution adapters/guards.

## Workflow
1. Map change to its authoritative plane.
2. Reject circular/synchronous dependencies violating hot-path isolation.
3. Prefer versioned APIs/events/read models to duplicated truth.

## Validate
Go remains realtime authority. Python not per-tick. Commercial calls never block trading. Browser never truth authority.
