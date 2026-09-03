-- Async backtest job queue (2026-09-03).
-- Long-range backtests (full-history M1 ≈ 10.5 min even after the 11.5x
-- float64 optimization) cannot run inside an HTTP request. POST /backtest/run
-- with an over-budget range enqueues a job and returns 202; an in-process
-- worker in pat-control polls QUEUED rows, runs the engine detached from the
-- request, and backfills run_id from the engine's --store persistence.
CREATE TABLE IF NOT EXISTS trading.backtest_jobs (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id uuid NOT NULL,
  strategy_id varchar(50) NOT NULL,
  timeframe varchar(10) NOT NULL,
  start_date date NOT NULL,
  end_date date NOT NULL,
  initial_balance numeric(18,2) NOT NULL DEFAULT 10000,
  status varchar(20) NOT NULL DEFAULT 'QUEUED'
    CHECK (status IN ('QUEUED', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED')),
  run_id varchar(40),
  error text,
  created_at timestamptz NOT NULL DEFAULT now(),
  started_at timestamptz,
  finished_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_backtest_jobs_status
  ON trading.backtest_jobs (status, created_at);
CREATE INDEX IF NOT EXISTS idx_backtest_jobs_user
  ON trading.backtest_jobs (user_id, created_at DESC);