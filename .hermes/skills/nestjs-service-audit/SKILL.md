---
name: nestjs-service-audit
description: "Audit NestJS services for DI, guards, and tx safety."
---

# nestjs-service-audit

Use when auditing NestJS control plane architecture.

## Modules
auth, users, admin, backtest, billing, licensing, device-auth, referrals, commissions, payouts, audit

## Checklist
1. DI: correct provider scopes
2. Guards: AuthGuard, RolesGuard, ThrottlerGuard
3. Interceptors: transaction wrapper on financial mutations
4. Pipes: Zod validation, transform enabled
5. Exception filters: no stack leaks
6. Circular deps: forwardRef usage tracked
7. ConfigService, not process.env directly
8. EventEmitter: audit events for financial ops
9. WS gateways: auth, room limits, connection limits

## Commands
cd control && npm run test && npm run test:e2e
