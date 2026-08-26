---
name: database-migrations
description: "Design PostgreSQL/TimescaleDB migrations safely."
---

# database-migrations

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Stack
- PostgreSQL 17, TimescaleDB HA, pgvector
- Valkey 8.0 (cache/hot state, not sole durable truth)
- 50+ migration files (001-078), 156 tables

## Workflow
1. Inspect schema/migration history first.
2. Canonical forward migrations; never rewrite applied history.
3. Exact numeric financial types (NUMERIC(38,18) for money).
4. Enforce constraints, ownership, idempotency.
5. Validate Timescale/pgvector/index/query/PgBouncer impacts.
6. Least-privilege roles and backup-aware migration/rollback.

## Migrate: ./scripts/migrate.sh up | down | seed | test

## Validate
Migration tests, query plans, restore path, financial/audit invariants.
