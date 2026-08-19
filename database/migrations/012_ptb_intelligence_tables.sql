-- Predict-A-Trade v1.0.0 — Migration 012
-- Stage 4: Professional Trader Brain / Advanced Market Intelligence
-- PTB evidence snapshots, feature flags, module provenance
-- All new tables are ADDITIVE — no existing audit history modified.

-- ============================================================
-- PTB Feature Flags (Stage 4 Section 30)
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.ptb_feature_flags (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    module_name     VARCHAR(100) NOT NULL UNIQUE,
    mode            VARCHAR(20) NOT NULL DEFAULT 'SHADOW',
    -- OFF, SHADOW, ACTIVE, DISABLED, UNSUPPORTED, RESEARCH
    set_by          VARCHAR(100),
    set_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    reason          TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Seed all modules as SHADOW (Stage 4 Section 29)
INSERT INTO trading.ptb_feature_flags (module_name, mode, reason)
SELECT m.module_name, 'SHADOW', 'Initial seed — all modules start in shadow mode'
FROM (VALUES
    ('liquidity_void'), ('wick_fill'), ('session_imbalance'),
    ('candle_range_projector'), ('time_at_mode'), ('engineered_liquidity_proxy'),
    ('market_phase'), ('relative_tick_volume_flow'), ('price_delivery'),
    ('stop_hunt_proxy'), ('time_cycle_analytics'), ('algo_activity_proxy'),
    ('complete_liquidity_map'), ('manipulation_proxy'),
    ('mtf_bias_engine'), ('volatility_regime_engine'),
    ('sr_quality_engine'), ('microstructure_engine'),
    ('statistical_performance_engine'), ('data_quality_engine')
) AS m(module_name)
ON CONFLICT (module_name) DO NOTHING;

-- Institutional Footprint is UNSUPPORTED by data source
INSERT INTO trading.ptb_feature_flags (module_name, mode, reason)
VALUES ('institutional_footprint', 'UNSUPPORTED',
        'broker tick data cannot provide DOM/Level2/Time&Sales/aggressor-side')
ON CONFLICT (module_name) DO NOTHING;

-- ============================================================
-- PTB Evidence Snapshots (Stage 4 Sections 45, 44)
-- Records per-evaluation advanced module results for audit trail
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.ptb_evidence_snapshots (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id           UUID REFERENCES trading.signals(id) ON DELETE CASCADE,
    source_snapshot_id  VARCHAR(200),
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT now(),
    data_source         VARCHAR(50) NOT NULL,
    is_live             BOOLEAN NOT NULL DEFAULT false,
    data_age_ms         BIGINT,

    data_quality_state  VARCHAR(50),
    data_quality_score  DECIMAL(5,4),

    -- JSON columns for each module result
    liquidity_void          JSONB,
    wick_fill               JSONB,
    session_imbalance       JSONB,
    candle_range_projector  JSONB,
    time_at_mode            JSONB,
    engineered_liquidity    JSONB,
    market_phase            JSONB,
    relative_volume_flow    JSONB,
    price_delivery          JSONB,
    stop_hunt_proxy          JSONB,
    institutional_footprint JSONB,
    time_cycle              JSONB,
    algo_activity           JSONB,
    complete_liquidity_map  JSONB,
    manipulation_proxy      JSONB,
    mtf_bias                JSONB,
    volatility_regime       JSONB,
    sr_quality              JSONB,
    microstructure          JSONB,

    feature_availability    JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ptb_evidence_signal ON trading.ptb_evidence_snapshots(signal_id);
CREATE INDEX IF NOT EXISTS idx_ptb_evidence_ts ON trading.ptb_evidence_snapshots(timestamp);

-- ============================================================
-- Data Provenance Log (Stage 4 Section 1)
-- Records data source authenticity for each evaluation cycle
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.data_provenance_log (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_snapshot_id  VARCHAR(200),
    source_type         VARCHAR(50) NOT NULL,
    master_node_id      VARCHAR(200),
    terminal_id         VARCHAR(200),
    broker              VARCHAR(200),
    symbol              VARCHAR(50) NOT NULL,
    market_timestamp    TIMESTAMPTZ,
    received_timestamp  TIMESTAMPTZ NOT NULL DEFAULT now(),
    evaluation_timestamp TIMESTAMPTZ NOT NULL DEFAULT now(),
    data_age_ms         BIGINT,
    is_live             BOOLEAN NOT NULL DEFAULT false,
    is_stale            BOOLEAN NOT NULL DEFAULT false,
    is_complete         BOOLEAN NOT NULL DEFAULT false,
    sequence            BIGINT,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_provenance_ts ON trading.data_provenance_log(market_timestamp);
CREATE INDEX IF NOT EXISTS idx_provenance_source ON trading.data_provenance_log(source_type);

-- ============================================================
-- Add columns to trading.signals for new signal states (Stage 4 Section 42)
-- BUY/SELL/WAIT/NO_TRADE/BLOCKED/ERROR
-- No migration needed — direction column is VARCHAR and already supports new values.
-- This index helps filter by the new state values.
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_signals_direction_v2 ON trading.signals(direction, created_at DESC);
