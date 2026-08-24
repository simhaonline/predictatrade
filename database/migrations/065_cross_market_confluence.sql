-- 065: Cross-Market Macro & Intermarket Confluence Engine tables
-- TimescaleDB hypertables for driver snapshots, confluence results, and signal linkage.

-- 1. Macro driver snapshots — one row per driver per refresh cycle
CREATE TABLE IF NOT EXISTS trading.cross_market_driver_snapshots (
    event_time    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    driver        TEXT NOT NULL,              -- dxy, eurusd, real_yields, vix, cot, btc, oil, usdjpy, usdchf
    raw_value     DOUBLE PRECISION,
    normalized_value DOUBLE PRECISION,        -- -100 to +100
    impact_score  DOUBLE PRECISION,           -- -100 to +100
    direction     TEXT,                       -- BULLISH, BEARISH, NEUTRAL
    confidence    DOUBLE PRECISION DEFAULT 0, -- 0 to 1
    freshness     DOUBLE PRECISION DEFAULT 0, -- 0 to 1
    quality       TEXT DEFAULT 'UNKNOWN',     -- CONNECTED, DEGRADED, STALE, MISSING, ERROR
    source        TEXT,
    timeframe     TEXT,
    reason        TEXT,
    metadata      JSONB,
    PRIMARY KEY (event_time, id)
);
SELECT create_hypertable('trading.cross_market_driver_snapshots', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');
CREATE INDEX IF NOT EXISTS idx_xm_driver_driver ON trading.cross_market_driver_snapshots (driver, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_xm_driver_quality ON trading.cross_market_driver_snapshots (quality) WHERE quality != 'CONNECTED';

-- 2. Confluence results — one row per signal evaluation cycle
CREATE TABLE IF NOT EXISTS trading.cross_market_confluence_results (
    event_time        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    id                UUID NOT NULL DEFAULT gen_random_uuid(),
    signal_id         UUID,
    symbol            TEXT NOT NULL DEFAULT 'XAUUSD',
    score             DOUBLE PRECISION NOT NULL DEFAULT 0,  -- -100 to +100
    direction         TEXT NOT NULL DEFAULT 'NEUTRAL',       -- BULLISH, BEARISH, NEUTRAL
    confidence        DOUBLE PRECISION DEFAULT 0,            -- 0 to 1
    agreement         DOUBLE PRECISION DEFAULT 0,            -- 0 to 1
    conflict          DOUBLE PRECISION DEFAULT 0,            -- 0 to 1
    data_quality      TEXT DEFAULT 'UNKNOWN',
    regime            TEXT,
    event_risk        TEXT DEFAULT 'NORMAL',
    correlation_regime TEXT,
    primary_drivers   JSONB,   -- list of driver names that agreed
    opposing_drivers  JSONB,   -- list of driver names that opposed
    missing_drivers   JSONB,   -- list of unavailable drivers
    warnings          JSONB,
    divergence_severity TEXT DEFAULT 'NONE',
    score_adjustment  DOUBLE PRECISION DEFAULT 0,  -- bounded adjustment applied to signal
    mode              TEXT DEFAULT 'shadow',        -- shadow, active, disabled
    model_version     TEXT DEFAULT '1.0.0',
    weights_version   TEXT DEFAULT '1.0.0',
    driver_snapshot   JSONB,   -- full driver breakdown for audit
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_time, id)
);
SELECT create_hypertable('trading.cross_market_confluence_results', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');
CREATE INDEX IF NOT EXISTS idx_xm_conf_signal ON trading.cross_market_confluence_results (signal_id);
CREATE INDEX IF NOT EXISTS idx_xm_conf_direction ON trading.cross_market_confluence_results (direction, event_time DESC);

-- 3. Correlation regime snapshots
CREATE TABLE IF NOT EXISTS trading.cross_market_correlation_regimes (
    event_time    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    pair          TEXT NOT NULL,               -- XAUUSD_DXY, XAUUSD_EURUSD, etc.
    correlation   DOUBLE PRECISION,
    window        INTEGER,
    regime        TEXT,                         -- NORMAL, WEAK, INVERSE, BREAKDOWN, SHIFT, INSUFFICIENT
    stability     DOUBLE PRECISION,
    direction_persistence DOUBLE PRECISION,
    PRIMARY KEY (event_time, id)
);
SELECT create_hypertable('trading.cross_market_correlation_regimes', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');
CREATE INDEX IF NOT EXISTS idx_xm_corr_pair ON trading.cross_market_correlation_regimes (pair, event_time DESC);

-- 4. Provider health
CREATE TABLE IF NOT EXISTS trading.cross_market_provider_health (
    event_time    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    id            UUID NOT NULL DEFAULT gen_random_uuid(),
    provider      TEXT NOT NULL,
    status        TEXT NOT NULL,               -- CONNECTED, DEGRADED, STALE, MISSING, ERROR, NOT_CONFIGURED
    last_success  TIMESTAMPTZ,
    last_error    TEXT,
    error_count   INTEGER DEFAULT 0,
    latency_ms    INTEGER,
    PRIMARY KEY (event_time, id)
);
SELECT create_hypertable('trading.cross_market_provider_health', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');

-- Compression + retention
ALTER TABLE trading.cross_market_driver_snapshots SET (timescaledb.compress, timescaledb.compress_segmentby = 'driver');
SELECT add_compression_policy('trading.cross_market_driver_snapshots', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('trading.cross_market_driver_snapshots', INTERVAL '90 days', if_not_exists => TRUE);

ALTER TABLE trading.cross_market_confluence_results SET (timescaledb.compress, timescaledb.compress_segmentby = 'signal_id');
SELECT add_compression_policy('trading.cross_market_confluence_results', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('trading.cross_market_confluence_results', INTERVAL '365 days', if_not_exists => TRUE);

ALTER TABLE trading.cross_market_correlation_regimes SET (timescaledb.compress, timescaledb.compress_segmentby = 'pair');
SELECT add_compression_policy('trading.cross_market_correlation_regimes', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('trading.cross_market_correlation_regimes', INTERVAL '90 days', if_not_exists => TRUE);

ALTER TABLE trading.cross_market_provider_health SET (timescaledb.compress);
SELECT add_compression_policy('trading.cross_market_provider_health', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('trading.cross_market_provider_health', INTERVAL '30 days', if_not_exists => TRUE);

GRANT INSERT, SELECT ON ALL TABLES IN SCHEMA trading TO pat_admin;
