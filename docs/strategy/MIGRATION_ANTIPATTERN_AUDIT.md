# Migration anti-pattern follow-ups — CREATE TABLE IF NOT EXISTS on legacy tables

Lesson source: Phase 0.5 / migration 150. `trading.predictions` already existed
(legacy shape, 0 rows); `CREATE TABLE IF NOT EXISTS` silently no-op'd and the
engine wrote into a table missing the new columns until the migration was
re-run with ALTER TABLE. Guard shipped Phase 0.75: `VerifyOutcomeSchema`
fails closed at startup if any writer-expected column is missing.

Anti-pattern: `CREATE TABLE IF NOT EXISTS x (...new columns...)` when `x` may
pre-exist with an older shape. The statement succeeds, the new columns are
absent, writes degrade silently.

## Rule (for all future migrations)
- New tables: `CREATE TABLE IF NOT EXISTS` is fine ONLY when the table is
  guaranteed never to have existed before (fresh name, no legacy writer).
- Extending existing tables: ALWAYS explicit existence check + ALTER TABLE
  ADD COLUMN IF NOT EXISTS (Postgres 9.6+) — idempotent and shape-safe.
- Writers must verify schema at startup (see VerifyOutcomeSchema pattern).

## Audit result (Phase 0.75)
46 historical migrations use `CREATE TABLE IF NOT EXISTS`. None are rewritten
(no history rewrite per prompt.md). Risk is contained because:
1. Each migration ran once in sequence on this deployment (recorded in
   audit.migration_history), so within THIS database the shapes match.
2. The failure mode matters only on FRESH installs replaying migrations —
   mitigated by the startup schema guard for the outcome pipeline and by the
   DB round-trip tests for the other writers.

## Follow-ups (to schedule, NOT blocking Phase 0.75)
- migrations 008-021, 018 (`shadow_signals`), 020 (`signal truth`): fresh-install
  replay risk — recommend a one-shot `scripts/verify_schema_parity.py` that runs
  the full migration chain against a scratch DB and diffs
  information_schema.columns against production. LOW priority (documented here).
- Any future extension of predictions/prediction_outcomes MUST use
  `ALTER TABLE ... ADD COLUMN IF NOT EXISTS` (enforced by review checklist).
