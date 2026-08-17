---
name: database-migrations
description: Design/review PostgreSQL 17, TimescaleDB, pgvector, PgBouncer and Valkey contracts with canonical migrations, exact decimals and backup/rollback safety.
---

# database-migrations

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Inspect schema/migration history first.
2. Use canonical forward migrations; never rewrite applied history.
3. Enforce constraints, ownership, idempotency and exact numeric financial types.
4. Validate Timescale/pgvector/index/query/PgBouncer impacts; Valkey is cache/hot state, not sole durable truth.
5. Use least-privilege roles and backup-aware migration/rollback or forward-fix plans.

## Validate
Representative migration tests, query plans, restore path and financial/audit invariants.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
