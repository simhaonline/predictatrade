-- Predict-A-Trade v1.0.0 — Migration 015
-- Backtesting Framework Persistence
-- All tables ADDITIVE — no existing tables modified.

-- ============================================================
-- Backtest Runs — Top-level run tracking
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.backtest_runs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id                  VARCHAR(20) NOT NULL UNIQUE,
    symbol                  VARCHAR(50) NOT NULL DEFAULT 'XAUUSD',
    strategy_id             VARCHAR(50) NOT NULL,
    strategy_mode           VARCHAR(30) NOT NULL DEFAULT 'ptb',
    primary_timeframe       VARCHAR(10) NOT NULL DEFAULT 'M5',
    start_timestamp         TIMESTAMPTZ,
    end_timestamp           TIMESTAMPTZ,
    initial_balance         DECIMAL(18,8) NOT NULL DEFAULT 10000,
    random_seed             INTEGER NOT NULL DEFAULT 42,

    status                  VARCHAR(20) NOT NULL DEFAULT 'COMPLETED',
    bars_processed          INTEGER NOT NULL DEFAULT 0,
    trades_count            INTEGER NOT NULL DEFAULT 0,
    no_trade_count          INTEGER NOT NULL DEFAULT 0,
    blocked_count           INTEGER NOT NULL DEFAULT 0,

    -- Metrics
    final_balance           DECIMAL(18,8),
    total_return_pct        DECIMAL(10,4),
    sharpe_ratio             DECIMAL(10,4),
    sortino_ratio            DECIMAL(10,4),
    max_drawdown_pct         DECIMAL(10,4),
    win_rate_pct             DECIMAL(5,2),
    profit_factor            DECIMAL(10,4),
    expectancy               DECIMAL(18,8),

    -- Configuration
    configuration            JSONB NOT NULL DEFAULT '{}',
    execution_assumptions   JSONB NOT NULL DEFAULT '{}',
    risk_config             JSONB NOT NULL DEFAULT '{}',

    -- Provenance
    data_source              VARCHAR(100),
    data_hash                VARCHAR(64),
    feature_version          VARCHAR(20) DEFAULT '1.0',
    model_version             VARCHAR(20),
    git_commit_sha            VARCHAR(40),
    application_version       VARCHAR(20),
    artifact_locations        JSONB NOT NULL DEFAULT '{}',

    started_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at              TIMESTAMPTZ,
    duration_seconds         DECIMAL(10,4),

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backtest_runs_strategy ON trading.backtest_runs(strategy_id);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_status ON trading.backtest_runs(status);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_created ON trading.backtest_runs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_backtest_runs_symbol ON trading.backtest_runs(symbol);

-- ============================================================
-- Backtest Trades — Individual trade records
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.backtest_trades (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id                  VARCHAR(20) NOT NULL REFERENCES trading.backtest_runs(run_id) ON DELETE CASCADE,
    trade_id                VARCHAR(100) NOT NULL,
    strategy_id             VARCHAR(50) NOT NULL,
    direction               VARCHAR(10) NOT NULL,
    entry_time              TIMESTAMPTZ NOT NULL,
    entry_price             DECIMAL(18,8) NOT NULL,
    exit_time               TIMESTAMPTZ,
    exit_price              DECIMAL(18,8),
    exit_reason             VARCHAR(50),
    size                    DECIMAL(10,4),
    pnl                     DECIMAL(18,8) NOT NULL DEFAULT 0,
    pnl_r                   DECIMAL(10,4) NOT NULL DEFAULT 0,
    commission               DECIMAL(18,8) DEFAULT 0,
    slippage_cost            DECIMAL(18,8) DEFAULT 0,
    spread_cost              DECIMAL(18,8) DEFAULT 0,
    mae                      DECIMAL(18,8),
    mfe                      DECIMAL(18,8),
    duration_bars            INTEGER,
    duration_seconds         DECIMAL(10,2),
    regime                   VARCHAR(50),
    session                  VARCHAR(50),
    confluence               DECIMAL(5,2),
    confidence               DECIMAL(5,2),
    setup_grade              VARCHAR(10),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backtest_trades_run ON trading.backtest_trades(run_id);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_strategy ON trading.backtest_trades(strategy_id);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_direction ON trading.backtest_trades(direction);
CREATE INDEX IF NOT EXISTS idx_backtest_trades_exit_reason ON trading.backtest_trades(exit_reason);

-- ============================================================
-- Backtest Fold Results — Walk-forward fold outcomes
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.backtest_fold_results (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id                  VARCHAR(20) REFERENCES trading.backtest_runs(run_id) ON DELETE CASCADE,
    fold_id                 INTEGER NOT NULL,
    train_start             INTEGER NOT NULL,
    train_end               INTEGER NOT NULL,
    test_start              INTEGER NOT NULL,
    test_end                INTEGER NOT NULL,
    in_sample_metrics       JSONB NOT NULL DEFAULT '{}',
    out_of_sample_metrics   JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backtest_fold_results_run ON trading.backtest_fold_results(run_id);

-- ============================================================
-- Backtest Artifacts — File locations for reports
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.backtest_artifacts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id                  VARCHAR(20) NOT NULL REFERENCES trading.backtest_runs(run_id) ON DELETE CASCADE,
    artifact_type           VARCHAR(50) NOT NULL, -- summary, trades, equity, metrics, config, data_quality, manifest
    file_path               TEXT NOT NULL,
    file_size_bytes         BIGINT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backtest_artifacts_run ON trading.backtest_artifacts(run_id);
CREATE INDEX IF NOT EXISTS idx_backtest_artifacts_type ON trading.backtest_artifacts(artifact_type);

-- ============================================================
-- Backtest Parameter Sets — Parameter search grid
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.backtest_parameter_sets (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id                  VARCHAR(20) REFERENCES trading.backtest_runs(run_id) ON DELETE CASCADE,
    parameter_name          VARCHAR(100) NOT NULL,
    parameter_value         JSONB NOT NULL,
    is_base                 BOOLEAN NOT NULL DEFAULT false,
    metrics                 JSONB NOT NULL DEFAULT '{}',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backtest_params_run ON trading.backtest_parameter_sets(run_id);

COMMENT ON TABLE trading.backtest_runs IS 'Top-level backtest run tracking with full reproducibility manifest';
COMMENT ON TABLE trading.backtest_trades IS 'Individual trade records from backtest runs';
COMMENT ON TABLE trading.backtest_fold_results IS 'Walk-forward fold outcomes with in-sample and OOS metrics';
COMMENT ON TABLE trading.backtest_artifacts IS 'File locations for generated reports';
COMMENT ON TABLE trading.backtest_parameter_sets IS 'Parameter search grid for sensitivity analysis';
