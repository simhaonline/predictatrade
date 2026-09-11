-- Rollback for reconcile_migrations.sql
-- Reverses the reconciliation: restores deleted orphans, removes back-filled
-- rows, drops the backup table and the added bookkeeping columns.
BEGIN;
DELETE FROM audit.migration_history WHERE filename IN ('145_hypertable_retention_policies.sql','148_devil_liquidity_volume_weight.sql');
DROP TABLE IF EXISTS audit.migration_history_reconcile_backup;
ALTER TABLE audit.migration_history DROP COLUMN IF EXISTS checksum;
ALTER TABLE audit.migration_history DROP COLUMN IF EXISTS reconciled_note;
COMMIT;
