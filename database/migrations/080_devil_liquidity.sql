-- Devil Liquidity / Devil's Mark engine (prompt.md Sections 50-55)
-- Canonical PostgreSQL state + TimescaleDB append-only event hypertable.

CREATE TABLE IF NOT EXISTS devil_liquidity_marks (
    id                      UUID PRIMARY KEY,
    symbol                  TEXT NOT NULL,
    timeframe               TEXT NOT NULL,
    direction               TEXT NOT NULL,            -- BULLISH | BEARISH
    mark_price              DOUBLE PRECISION NOT NULL,

    open                    DOUBLE PRECISION,
    high                    DOUBLE PRECISION,
    low                     DOUBLE PRECISION,
    close                   DOUBLE PRECISION,
    range                   DOUBLE PRECISION,
    body                    DOUBLE PRECISION,
    body_ratio              DOUBLE PRECISION,

    upper_wick              DOUBLE PRECISION,
    lower_wick              DOUBLE PRECISION,
    upper_wick_ratio        DOUBLE PRECISION,
    lower_wick_ratio        DOUBLE PRECISION,

    atr                     DOUBLE PRECISION,
    range_atr_ratio         DOUBLE PRECISION,
    body_expansion_ratio    DOUBLE PRECISION,

    volume                  BIGINT,
    volume_ratio            DOUBLE PRECISION,
    volume_zscore           DOUBLE PRECISION,

    spread                  DOUBLE PRECISION,
    digits                  INTEGER,
    tick_size               DOUBLE PRECISION,

    fvg_present             BOOLEAN DEFAULT FALSE,
    fvg_id                  TEXT,
    bos_present             BOOLEAN DEFAULT FALSE,
    mss_present             BOOLEAN DEFAULT FALSE,
    choch_present           BOOLEAN DEFAULT FALSE,

    formation_session       TEXT,
    formation_regime        TEXT,

    mark_quality_score      DOUBLE PRECISION,
    priority_score          DOUBLE PRECISION,

    status                  TEXT NOT NULL,

    first_approach_at       TIMESTAMPTZ,
    first_touch_at          TIMESTAMPTZ,
    first_sweep_at          TIMESTAMPTZ,
    sweep_low               DOUBLE PRECISION,
    sweep_high              DOUBLE PRECISION,
    reclaim_at              TIMESTAMPTZ,
    reversal_confirmed_at   TIMESTAMPTZ,

    sweep_depth_atr         DOUBLE PRECISION,
    reclaim_strength        DOUBLE PRECISION,

    reversal_score          DOUBLE PRECISION,
    combined_score          DOUBLE PRECISION,
    distance_atr            DOUBLE PRECISION,

    expired_at              TIMESTAMPTZ,
    invalidated_at          TIMESTAMPTZ,
    resolved_at             TIMESTAMPTZ,

    feed_source             TEXT,
    broker                  TEXT,
    server_identifier       TEXT,
    config_version          TEXT,

    detected_at             TIMESTAMPTZ NOT NULL,
    updated_at              TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_devil_marks_symbol_tf ON devil_liquidity_marks (symbol, timeframe);
CREATE INDEX IF NOT EXISTS idx_devil_marks_status ON devil_liquidity_marks (status);
CREATE INDEX IF NOT EXISTS idx_devil_marks_detected ON devil_liquidity_marks (detected_at DESC);

-- Append-only event stream (hypertable on event_time).
CREATE TABLE IF NOT EXISTS devil_liquidity_events (
    event_time      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    mark_id         UUID NOT NULL,
    symbol          TEXT NOT NULL,
    timeframe       TEXT NOT NULL,
    event_type      TEXT NOT NULL,
    state_from      TEXT,
    state_to        TEXT,
    price           DOUBLE PRECISION,
    mark_price      DOUBLE PRECISION,
    distance_atr    DOUBLE PRECISION,
    atr             DOUBLE PRECISION,
    spread          DOUBLE PRECISION,
    regime          TEXT,
    session         TEXT,
    quality_score   DOUBLE PRECISION,
    reversal_score  DOUBLE PRECISION,
    metadata        JSONB
);

SELECT create_hypertable('devil_liquidity_events', 'event_time', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_devil_events_mark ON devil_liquidity_events (mark_id, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_devil_events_type ON devil_liquidity_events (event_type, event_time DESC);

-- Configuration table (prompt.md Section 52).
CREATE TABLE IF NOT EXISTS devil_liquidity_config (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    mode                TEXT NOT NULL DEFAULT 'shadow',
    strategy            TEXT,
    timeframe           TEXT,
    flat_wick_ratio     DOUBLE PRECISION DEFAULT 0.03,
    flat_wick_atr_tol   DOUBLE PRECISION DEFAULT 0.15,
    minimum_tick_tol    INTEGER DEFAULT 2,
    minimum_body_ratio  DOUBLE PRECISION DEFAULT 0.70,
    minimum_range_atr   DOUBLE PRECISION DEFAULT 1.20,
    minimum_body_exp    DOUBLE PRECISION DEFAULT 1.50,
    close_extreme_ratio DOUBLE PRECISION DEFAULT 0.15,
    approach_distance_atr DOUBLE PRECISION DEFAULT 3.0,
    minimum_sweep_depth_atr DOUBLE PRECISION DEFAULT 0.20,
    maximum_sweep_depth_atr DOUBLE PRECISION DEFAULT 2.5,
    reclaim_max_bars    INTEGER DEFAULT 10,
    reversal_body_ratio DOUBLE PRECISION DEFAULT 0.50,
    mark_expiry_bars    INTEGER DEFAULT 240,
    min_mark_quality    DOUBLE PRECISION DEFAULT 40.0,
    min_signal_score    DOUBLE PRECISION DEFAULT 60.0,
    news_handling       TEXT DEFAULT 'analyze_separately',
    updated_by          TEXT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    version             TEXT NOT NULL DEFAULT '1.0.0'
);

-- Seed default configuration row.
INSERT INTO devil_liquidity_config (
    enabled, mode, flat_wick_ratio, flat_wick_atr_tol, minimum_tick_tol,
    minimum_body_ratio, minimum_range_atr, minimum_body_exp, close_extreme_ratio,
    approach_distance_atr, minimum_sweep_depth_atr, maximum_sweep_depth_atr,
    reclaim_max_bars, reversal_body_ratio, mark_expiry_bars, min_mark_quality,
    min_signal_score, news_handling, updated_by, version
) VALUES (
    TRUE, 'shadow', 0.03, 0.15, 2,
    0.70, 1.20, 1.50, 0.15,
    3.0, 0.20, 2.5,
    10, 0.50, 240, 40.0,
    60.0, 'analyze_separately', 'migration_080', '1.0.0'
) ON CONFLICT DO NOTHING;
