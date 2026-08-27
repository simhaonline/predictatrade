-- Rollback for reconcile_migrations.sql
-- Reverses the reconciliation: restores deleted orphans, removes back-filled
-- rows, drops the backup table and the added bookkeeping columns.
BEGIN;
INSERT INTO audit.migration_history (filename, status, started_at, completed_at)
SELECT filename, status, started_at, completed_at FROM audit.migration_history_reconcile_backup
ON CONFLICT (filename) DO NOTHING;
DELETE FROM audit.migration_history WHERE filename IN ('074_backtest_artifact_payload.sql','075_finance_ledger_entries.sql','076_market_data_metadata.sql','077_backtest_runs_owner.sql','078_subscription_paused_state.sql','079_payouts_idempotency.sql','080_devil_liquidity.sql','080_signal_quality_diagnostics.sql','081_market_candles_retention.sql','082_elite_marnie_fib_strategy.sql','083_backfill_trade_metrics.sql','084_add_trade_timeframe.sql','085_set_plan_risk_caps.sql','086_agent_device_id_binding.sql','087_signal_verification_columns.sql');
DROP TABLE IF EXISTS audit.migration_history_reconcile_backup;
ALTER TABLE audit.migration_history DROP COLUMN IF EXISTS checksum;
ALTER TABLE audit.migration_history DROP COLUMN IF EXISTS reconciled_note;
COMMIT;
