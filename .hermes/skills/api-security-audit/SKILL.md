---
name: api-security-audit
description: "Audit REST/WS endpoints for auth, RBAC, rate limits."
---

# api-security-audit

Use when auditing Predict-A-Trade API endpoints.

## Endpoints
NestJS :13080, Go :13081, Next.js :13082, backtest 8088, live terminal 13090

## Checklist
1. Auth on every mutation (JWT + MFA for sensitive ops)
2. RBAC: admin vs user vs public
3. Idempotency key on financial mutations
4. Rate limiting: login/register/verify-otp/reset-password
5. Input validation: Zod (NestJS), struct tags (Go)
6. Correlation-ID across HTTP + WebSocket
7. WS auth: token in connect, origin validation
8. Error responses: structured JSON, stable codes, no stack traces
9. CORS: per-origin, no wildcard in prod
10. Tenant isolation: tenant_id from JWT

## Known Issues
H1: JWT secret dual-source, H3: backtest IDOR
F1: accessToken in JS cookie, F2: no CSP/HSTS
