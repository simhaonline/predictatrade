-- Predict-A-Trade v1.0.0 — Migration 072
-- Online backtesting artifact payload.
-- ADDITIVE: adds a JSONB payload column to trading.backtest_artifacts so
-- equity curves / metrics can be stored and retrieved online (no file I/O).
-- SOW Section 144 (online backtesting + reporting).

ALTER TABLE trading.backtest_artifacts
    ADD COLUMN IF NOT EXISTS artifact_payload JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_backtest_artifacts_payload_type
    ON trading.backtest_artifacts(run_id, artifact_type);
