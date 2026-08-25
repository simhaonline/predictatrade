-- Backtest run ownership (H3 remediation).
-- trading.backtest_runs previously had NO owner column, so any authenticated
-- user could list / read / download ANY user's backtest runs (IDOR). Add an
-- owner reference so reads can be scoped per-user (admins see all).
-- Created idempotently so it is a no-op on the running system.
ALTER TABLE trading.backtest_runs
  ADD COLUMN IF NOT EXISTS user_id uuid REFERENCES iam.users(id) ON DELETE SET NULL;

CREATE INDEX IF NOT EXISTS idx_backtest_runs_user
  ON trading.backtest_runs (user_id);
