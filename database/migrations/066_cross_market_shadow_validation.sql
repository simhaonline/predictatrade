-- 066: Cross-Market shadow validation infrastructure tables

-- 1. Shadow signal snapshots — one per signal evaluation for later validation
CREATE TABLE IF NOT EXISTS trading.cross_market_shadow_snapshots (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timestamp               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    signal_id               TEXT,
    strategy                TEXT,
    direction               TEXT,
    technical_score         DOUBLE PRECISION,
    cross_market_score      DOUBLE PRECISION,
    cross_market_confidence DOUBLE PRECISION,
    cross_market_regime     TEXT,
    dxy_contribution        DOUBLE PRECISION DEFAULT 0,
    eurusd_contribution     DOUBLE PRECISION DEFAULT 0,
    cot_contribution        DOUBLE PRECISION DEFAULT 0,
    real_yield_contribution DOUBLE PRECISION DEFAULT 0,
    vix_contribution        DOUBLE PRECISION DEFAULT 0,
    btc_contribution        DOUBLE PRECISION DEFAULT 0,
    oil_contribution        DOUBLE PRECISION DEFAULT 0,
    driver_health           TEXT,
    driver_quality          TEXT,
    candidate_decision      TEXT,
    signal_decision         TEXT,
    entry                   DOUBLE PRECISION,
    stop_loss               DOUBLE PRECISION,
    tp1                     DOUBLE PRECISION,
    tp2                     DOUBLE PRECISION,
    tp3                     DOUBLE PRECISION,
    expiry                  TIMESTAMPTZ,
    outcome                 TEXT DEFAULT 'UNRESOLVED',
    mfe                     DOUBLE PRECISION,
    mae                     DOUBLE PRECISION,
    r_multiple              DOUBLE PRECISION,
    time_to_tp              INTEGER,
    time_to_sl              INTEGER,
    resolved_at             TIMESTAMPTZ
);

SELECT create_hypertable('trading.cross_market_shadow_snapshots', 'timestamp', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');

CREATE INDEX IF NOT EXISTS idx_shadow_strategy ON trading.cross_market_shadow_snapshots (strategy, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_shadow_outcome ON trading.cross_market_shadow_snapshots (outcome) WHERE outcome = 'UNRESOLVED';
CREATE INDEX IF NOT EXISTS idx_shadow_signal_id ON trading.cross_market_shadow_snapshots (signal_id);

ALTER TABLE trading.cross_market_shadow_snapshots SET (timescaledb.compress, timescaledb.compress_segmentby = 'strategy');
SELECT add_compression_policy('trading.cross_market_shadow_snapshots', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('trading.cross_market_shadow_snapshots', INTERVAL '365 days', if_not_exists => TRUE);

-- 2. Validation status record — single row tracking activation eligibility
CREATE TABLE IF NOT EXISTS trading.cross_market_validation_status (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    mode                    TEXT NOT NULL DEFAULT 'shadow',
    shadow_start            TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    usable_days             INTEGER DEFAULT 0,
    resolved_samples        INTEGER DEFAULT 0,
    baseline_test_completed BOOLEAN DEFAULT FALSE,
    ablation_completed      BOOLEAN DEFAULT FALSE,
    walk_forward_completed  BOOLEAN DEFAULT FALSE,
    regression_completed    BOOLEAN DEFAULT FALSE,
    latency_test_completed  BOOLEAN DEFAULT FALSE,
    validation_status       TEXT DEFAULT 'PENDING',
    activation_eligible     BOOLEAN DEFAULT FALSE,
    validated_at            TIMESTAMPTZ,
    validated_by            TEXT,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

INSERT INTO trading.cross_market_validation_status (mode, shadow_start)
VALUES ('shadow', NOW())
ON CONFLICT DO NOTHING;

-- 3. Ablation results — stores results of each ablation run
CREATE TABLE IF NOT EXISTS trading.cross_market_ablation_results (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    config_name     TEXT NOT NULL,
    strategy        TEXT NOT NULL,
    signal_count    INTEGER,
    win_rate        DOUBLE PRECISION,
    profit_factor   DOUBLE PRECISION,
    expectancy      DOUBLE PRECISION,
    avg_r           DOUBLE PRECISION,
    median_r        DOUBLE PRECISION,
    max_drawdown    DOUBLE PRECISION,
    tp1_hit_rate    DOUBLE PRECISION,
    tp2_hit_rate    DOUBLE PRECISION,
    tp3_hit_rate    DOUBLE PRECISION,
    sl_rate         DOUBLE PRECISION,
    sharpe          DOUBLE PRECISION,
    sortino         DOUBLE PRECISION
);

CREATE INDEX IF NOT EXISTS idx_ablation_config ON trading.cross_market_ablation_results (config_name, strategy);

GRANT INSERT, SELECT, UPDATE ON ALL TABLES IN SCHEMA trading TO pat_admin;
