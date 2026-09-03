-- 121: backtest_runs.user_id — per-user run ownership (R1 design)
--
-- control/src/modules/backtest/backtest.service.ts references
-- trading.backtest_runs.user_id in three places:
--   1. runBacktest(): UPDATE ... SET user_id = $2 (claim run post-completion)
--   2. getRunDetails(): SELECT user_id FROM trading.backtest_runs (ownership check)
--   3. listRuns(): WHERE user_id = $2 (non-admin scoping)
--
-- The column never existed, so any authenticated call whose JWT role was
-- not ADMIN/SUPER_ADMIN produced:
--   error: column "user_id" does not exist  →  HTTP 500
-- on GET /api/v1/backtest/runs ("Failed to fetch runs" on the backtest page).
-- Admin-shaped calls skipped the WHERE clause and worked, which masked the
-- bug until a token with a non-admin role claim hit the endpoint
-- (2026-09-03 11:47 UTC incident).
--
-- Additive, backfill-safe: existing rows stay user_id NULL (platform-era
-- runs predate per-user ownership). New runs are claimed by runBacktest().
ALTER TABLE trading.backtest_runs
    ADD COLUMN IF NOT EXISTS user_id uuid REFERENCES iam.users(id);

CREATE INDEX IF NOT EXISTS idx_backtest_runs_user
    ON trading.backtest_runs(user_id);