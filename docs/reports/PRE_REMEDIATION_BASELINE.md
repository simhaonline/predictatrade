# Pre-Remediation Baseline — 2026-08-23

## System State
- Git: main @ c8e399d
- Docker: 10 containers (all healthy)
- Go: 1.25.0, all tests pass (29 suites)
- NestJS: builds clean
- Frontend: Next.js 16.3, builds with webpack

## P0 Findings

| ID | Finding | Severity | Status | Evidence |
|----|---------|----------|--------|----------|
| P0-1 | opencode.json exposed OpenRouter API key | CRITICAL | FIXED | Key replaced with "REMOVED", file gitignored |
| P0-2 | WebSocket accepts caller-supplied userId (impersonation) | CRITICAL | FIXED | Now extracts identity from JWT token |
| P0-3 | /api/v1/signals returns data without auth | HIGH | VERIFIED | curl without token returns HTTP 200 |
| P0-4 | /api/v1/market/state returns data without auth | HIGH | VERIFIED | Public market data — acceptable for live dashboard |
| P0-5 | Entitlement gates exist but signal delivery not per-user | HIGH | PARTIAL | Gates check entitlement but WS broadcast is global |

## P1 Findings

| ID | Finding | Severity | Status |
|----|---------|----------|--------|
| P1-1 | Billing webhook has no signature verification | HIGH | BLOCKED — requires payment provider credentials |
| P1-2 | Commission engine exists with tests | MEDIUM | VERIFIED — commission-engine.spec.ts passes |
| P1-3 | Payout service exists | MEDIUM | VERIFIED — code present, no live payouts |
| P1-4 | Referral attribution by code (not email) | MEDIUM | FIXED — auth.service.ts looks up by referral_codes table |

## Architecture
```
MT4/MT5 → Windows Agent → Go Realtime Engine → TimescaleDB + Valkey
→ HTTP/WebSocket → NestJS Control Plane → Next.js Dashboards
```

## Test Results
- Go: 29 suites, 0 failures
- NestJS: builds clean, has spec files for commission/entitlement/admin
- Frontend: builds with webpack
