-- 099: Institutional Gold Signal (IGS) + AI research reports
--
-- IGS is the deterministic composite of institutional gold intelligence inputs
-- described in check.md: ETF flows, COT positioning, DXY/real-yield regime,
-- central-bank demand signals, and LLM-generated institutional research bias.
--
-- Design rules:
--   * IGS is a CONFIRMATION layer, never a trade trigger (same as crossmarket).
--   * Shadow mode by default: IGS snapshots are persisted for research but the
--     score never reaches production signals until promotion (shadow → active).
--   * Missing feeds degrade quality/UNAVAILABLE — data is NEVER fabricated.
--
-- Part 1: IGS component snapshots — one row per driver per refresh cycle.
CREATE TABLE IF NOT EXISTS trading.igs_component_snapshots (
    event_time        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    id                UUID NOT NULL DEFAULT gen_random_uuid(),
    component         TEXT NOT NULL,              -- etf_flows, cot_positioning, dxy_regime, real_yield_regime, central_bank, ai_research
    raw_value         DOUBLE PRECISION,
    normalized_value  DOUBLE PRECISION,           -- -100 to +100
    direction         TEXT,                       -- BULLISH, BEARISH, NEUTRAL
    confidence        DOUBLE PRECISION DEFAULT 0, -- 0 to 1
    quality           TEXT DEFAULT 'UNAVAILABLE', -- CONNECTED, DEGRADED, STALE, UNAVAILABLE
    source            TEXT,                       -- twelvedata, fmp, fred, tradingagents, ...
    reason            TEXT,
    metadata          JSONB,
    PRIMARY KEY (event_time, id)
);
SELECT create_hypertable('trading.igs_component_snapshots', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');
CREATE INDEX IF NOT EXISTS idx_igs_component ON trading.igs_component_snapshots (component, event_time DESC);
CREATE INDEX IF NOT EXISTS idx_igs_component_quality ON trading.igs_component_snapshots (quality) WHERE quality != 'CONNECTED';

-- Part 2: IGS composite results — one row per evaluation cycle.
CREATE TABLE IF NOT EXISTS trading.igs_results (
    event_time          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    id                  UUID NOT NULL DEFAULT gen_random_uuid(),
    signal_id           UUID,
    symbol              TEXT NOT NULL DEFAULT 'XAUUSD',
    score               DOUBLE PRECISION NOT NULL DEFAULT 0,  -- -100 to +100 (IGS classification scale)
    classification      TEXT NOT NULL DEFAULT 'NEUTRAL_CONFLICT', -- EXTREME_INSTITUTIONAL_BULLISH .. EXTREME_INSTITUTIONAL_BEARISH
    direction           TEXT NOT NULL DEFAULT 'NEUTRAL',
    confidence          DOUBLE PRECISION DEFAULT 0,
    agreement           DOUBLE PRECISION DEFAULT 0,           -- 0 to 1
    conflict            DOUBLE PRECISION DEFAULT 0,           -- 0 to 1
    data_quality        TEXT DEFAULT 'UNAVAILABLE',
    components_available INT NOT NULL DEFAULT 0,
    components_total     INT NOT NULL DEFAULT 0,
    missing_components  JSONB,
    warnings            JSONB,
    score_adjustment    DOUBLE PRECISION DEFAULT 0,  -- bounded, 0 in shadow mode
    mode                TEXT NOT NULL DEFAULT 'shadow',
    model_version       TEXT NOT NULL DEFAULT '1.0.0',
    weights_version     TEXT NOT NULL DEFAULT '1.0.0',
    component_snapshot  JSONB,   -- full component breakdown for audit
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (event_time, id)
);
SELECT create_hypertable('trading.igs_results', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');
CREATE INDEX IF NOT EXISTS idx_igs_results_signal ON trading.igs_results (signal_id);
CREATE INDEX IF NOT EXISTS idx_igs_results_classification ON trading.igs_results (classification, event_time DESC);

-- Part 3: AI research reports (TradingAgents adapter output, LLM-generated).
-- Research-plane artifact ONLY: rendered for humans / admin intelligence,
-- NEVER an automated execution authority. Bias is consumed through the IGS
-- ai_research component with its own bounded weight.
CREATE TABLE IF NOT EXISTS trading.ai_research_reports (
    id                UUID NOT NULL DEFAULT gen_random_uuid(),
    created_at        TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    run_date          DATE NOT NULL,
    symbol            TEXT NOT NULL DEFAULT 'XAUUSD',
    framework         TEXT NOT NULL DEFAULT 'tradingagents', -- framework identifier
    framework_version TEXT,
    model             TEXT,                                  -- LLM backbone used
    bias              TEXT,                                  -- BULLISH, BEARISH, NEUTRAL
    confidence        DOUBLE PRECISION,                      -- 0 to 1 (self-reported)
    summary           TEXT,
    bull_thesis       TEXT,
    bear_thesis       TEXT,
    risks             TEXT,
    key_drivers       JSONB,
    full_report       JSONB,
    provenance        JSONB,                                 -- inputs, model, timestamps, hashes
    quality           TEXT NOT NULL DEFAULT 'GENERATED',     -- GENERATED, REVIEWED, REJECTED
    UNIQUE (run_date, symbol, framework)
);
CREATE INDEX IF NOT EXISTS idx_ai_research_reports_date ON trading.ai_research_reports (run_date DESC);

-- Part 4: IGS weight versioning (auditable, like strategy_config_versions).
CREATE TABLE IF NOT EXISTS trading.igs_weight_versions (
    id              UUID NOT NULL DEFAULT gen_random_uuid(),
    version         TEXT NOT NULL,
    weights         JSONB NOT NULL,           -- {component_key: weight}
    freshness_ttl   JSONB,                    -- {component_key: seconds}
    mode            TEXT NOT NULL DEFAULT 'shadow',
    effective_from  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    effective_to    TIMESTAMPTZ,
    change_reason   TEXT,
    created_by      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (version, effective_from)
);

-- Part 5: Compression + retention (065 convention; keep tick-frequency growth bounded)
ALTER TABLE trading.igs_component_snapshots SET (timescaledb.compress, timescaledb.compress_segmentby = 'component');
SELECT add_compression_policy('trading.igs_component_snapshots', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('trading.igs_component_snapshots', INTERVAL '90 days', if_not_exists => TRUE);

ALTER TABLE trading.igs_results SET (timescaledb.compress, timescaledb.compress_segmentby = 'signal_id');
SELECT add_compression_policy('trading.igs_results', INTERVAL '7 days', if_not_exists => TRUE);
SELECT add_retention_policy('trading.igs_results', INTERVAL '365 days', if_not_exists => TRUE);

-- Seed the default weight version row (engine ships WeightsVersion 1.0.0).
INSERT INTO trading.igs_weight_versions (version, weights, freshness_ttl, mode, change_reason, created_by)
VALUES (
    '1.0.0',
    '{"usd_regime":20,"real_yield_regime":20,"central_bank_flow":15,"etf_flows":15,"cot_positioning":12,"options_gamma":8,"institutional_research":6,"physical_demand":4}'::JSONB,
    '{"usd_regime":600,"real_yield_regime":86400,"central_bank_flow":2592000,"cot_positioning":604800,"etf_flows":86400,"options_gamma":86400,"institutional_research":259200,"physical_demand":2592000}'::JSONB,
    'shadow',
    'initial check.md-tier hierarchy seed',
    'migration'
) ON CONFLICT (version, effective_from) DO NOTHING;

GRANT INSERT, SELECT ON trading.igs_results, trading.igs_component_snapshots, trading.ai_research_reports, trading.igs_weight_versions TO pat_admin;