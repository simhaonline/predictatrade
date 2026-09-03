-- 124: backtest_runs.raw_output — verbatim engine stdout per run
--
-- The control plane parses a handful of metrics out of backtest-engine's
-- stdout (parseMetric) and the dashboard shows those numbers. To let the
-- operator VERIFY dashboard numbers against the engine's own output, the
-- last ~10KB of engine stdout is persisted per run. Exposed to admins via
-- GET /backtest/runs/:id (raw_output field) and rendered in the admin UI
-- "Verify engine output" viewer.
--
-- Historical runs (before this column) have NULL raw_output — the viewer
-- shows a clear note instead of pretending data exists.
ALTER TABLE trading.backtest_runs
    ADD COLUMN IF NOT EXISTS raw_output text;
