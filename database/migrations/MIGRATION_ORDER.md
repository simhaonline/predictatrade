# Database Migration Order

## Canonical Apply Order

Migrations are applied lexicographically by filename. As of the 2026-08-28
renumber, all previously-duplicate sequence numbers have been resolved: the second
file of each former duplicate pair was renumbered to a unique sequence (089–095)
and the corresponding `audit.migration_history` rows were updated, so each migration
applies exactly once.

### Resolved duplicates (renumbered — no longer collide):
- 018 `018_regime_telemetry_shadow_signals.sql` + 089 `089_slippage_capital_protection.sql`
- 019 `019_percentage_sltp_config.sql` + 090 `090_signal_bug_closure_fields.sql`
- 020 `020_signal_truth_durability.sql` + 091 `091_valkey_candle_cache_indexes.sql`
- 028 `028_audit_execution_tables.sql` + 092 `092_bar_processing_metadata.sql`
- 062 `062_licensing_lifecycle.sql` + 093 `093_risk_config.sql`
- 071 `071_live_preview_anonymous_trials.sql` + 094 `094_marketing_consent_columns.sql`
- 080 `080_devil_liquidity.sql` + 095 `095_signal_quality_diagnostics.sql`

`scripts/check_migrations.sh` now passes (no duplicate prefixes, history matches disk).

### Future migrations:
- Use 3-digit zero-padded sequence numbers (028, 029, 030, ...)
- Before creating a new migration, run: `ls database/migrations/ | sort | tail -5`
- Never reuse an existing sequence number
- The `scripts/migrate.sh` script enforces uniqueness for new files
- `scripts/check_migrations.sh` is the CI guard (DB-6): it FAILS on any duplicate
  prefix (legacy AND new) and on any history-vs-disk filename drift.

## Reconciliation

`audit.migration_history` must equal the on-disk `database/migrations/*.sql` set.
Run `scripts/reconcile_migrations.sh --apply` to back-fill missing rows and
delete orphan rows. A rollback SQL is generated alongside it.

## Current inventory (as of v1.17.4)

69 unique migrations on disk, `001` → `099`. Recent additions:

- `096_ai_providers.sql` — AI provider registry (ollama|openai|custom)
- `097_playbook_exit_profiles.sql` — playbook exit profiles
- `098_broker_account_types.sql` — broker account types
- `099_igs_institutional_gold_signal.sql` — IGS composites + AI research reports
  + versioned weights (seed row `1.0.0/shadow`) + compression/retention policies

Rollback: no `down` migrations exist by design; point-in-time recovery (PITR)
from the 6-hourly backup is the documented rollback path (see docs/operations).
