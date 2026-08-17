---
name: api-contracts
description: Design/review versioned OpenAPI/WebSocket/event contracts with auth, entitlement, idempotency, errors, backward compatibility and resumable delivery.
---

# api-contracts

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Identify authoritative owner for every field.
2. Version API/schema and define typed requests/responses/errors.
3. Enforce identity/tenant/role/plan/strategy/license/device/account permissions.
4. Require idempotency for financial/execution mutations.
5. Event envelope includes IDs, sequence, type/version, timestamps, provenance, quality and payload.
6. Define priority/backpressure; P0/P1 are never silently dropped.
7. Define snapshot + delta + reconnect/resume/resync and deprecation policy.

## Validate
Cross-plane contract tests, duplicate mutation safety and compatibility/version-error behavior.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
