---
name: api-contracts
description: "Design versioned API/WebSocket event contracts."
---

# api-contracts

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Service Ports
- Go Realtime: 13081 (HTTP + WebSocket)
- NestJS Control: 13080 (REST API)
- Next.js Frontend: 13082 (SSR + static)
- Backtest Service: 8088
- Live Terminal: 13090
- Status Page: 13083
- Nginx: 80/443 (reverse proxy)

## Workflow
1. Identify authoritative owner for every field.
2. Version API/schema and define typed requests/responses/errors.
3. Enforce identity/tenant/role/plan/strategy/license/device/account permissions.
4. Idempotency for financial/execution mutations.
5. Event envelope: IDs, sequence, type/version, timestamps, provenance, quality, payload.
6. Priority/backpressure: P0/P1 never silently dropped.
7. Snapshot + delta + reconnect/resume/resync. Deprecation policy.
