# Database Migration Order

## Canonical Apply Order

Migrations are applied lexicographically by filename. Duplicate sequence numbers
exist in the deployed database and cannot be renamed (migration history discipline).

### Duplicates (already applied, do NOT rename):
- 018: `018_regime_telemetry_shadow_signals.sql` + `018_slippage_capital_protection.sql`
- 019: `019_percentage_sltp_config.sql` + `019_signal_bug_closure_fields.sql`
- 020: `020_signal_truth_durability.sql` + `020_valkey_candle_cache_indexes.sql`
- 062: (if exists, two files with same prefix)

### Future migrations:
- Use 3-digit zero-padded sequence numbers (028, 029, 030, ...)
- Before creating a new migration, run: `ls database/migrations/ | sort | tail -5`
- Never reuse an existing sequence number
- The `scripts/migrate.sh` script enforces uniqueness for new files

## Enforcement

`scripts/migrate.sh` checks for duplicate sequence numbers before applying
and will reject new files with duplicate numbers.
