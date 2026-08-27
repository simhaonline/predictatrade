# Database Migration Order

## Canonical Apply Order

Migrations are applied lexicographically by filename. Duplicate sequence numbers
exist in the deployed database and cannot be renamed (migration history discipline).

### Duplicates (already applied, do NOT rename — renumber only in a future maintenance window):
- 018: `018_regime_telemetry_shadow_signals.sql` + `018_slippage_capital_protection.sql`
- 019: `019_percentage_sltp_config.sql` + `019_signal_bug_closure_fields.sql`
- 020: `020_signal_truth_durability.sql` + `020_valkey_candle_cache_indexes.sql`
- 028: `028_audit_execution_tables.sql` + `028_bar_processing_metadata.sql`
- 062: `062_licensing_lifecycle.sql` + `062_risk_config.sql`
- 071: `071_live_preview_anonymous_trials.sql` + `071_marketing_consent_columns.sql`
- 080: `080_devil_liquidity.sql` + `080_signal_quality_diagnostics.sql`

These files carry a `-- LEGACY DUPLICATE PREFIX NN` header comment noting the
collision. They must NOT be renamed: renaming would risk re-applying already
applied schema. The collision is tolerated until a deliberate renumber window.

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
