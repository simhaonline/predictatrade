-- Add per-timeframe tagging to executed trades so edge statistics can be computed
-- and gated per (strategy, timeframe), not conflated across timeframes.
-- Existing rows default to '' (catch-all scope) until the EA starts reporting it.
ALTER TABLE trading.trade_results
    ADD COLUMN IF NOT EXISTS timeframe VARCHAR(10) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_trade_results_strategy_tf
    ON trading.trade_results (strategy_id, timeframe);
