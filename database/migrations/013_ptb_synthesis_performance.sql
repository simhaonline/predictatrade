-- Predict-A-Trade v1.0.0 — Migration 013
-- Stage 4 PTB: Synthesis history + signal performance feedback
-- All tables ADDITIVE — no existing audit history modified.

-- ============================================================
-- PTB Analysis History (Section 27)
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.ptb_analysis_history (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    analysis_id             VARCHAR(200) NOT NULL,
    timestamp               TIMESTAMPTZ NOT NULL DEFAULT now(),
    symbol                  VARCHAR(50) NOT NULL,

    regime                  VARCHAR(50),
    gold_role               VARCHAR(50),
    volatility_state        VARCHAR(50),
    manipulation_index      DECIMAL(5,2),

    bias                    VARCHAR(30),
    bias_strength           DECIMAL(5,4),
    confidence              DECIMAL(5,2),

    confluence_score        DECIMAL(5,2),
    setup_quality           VARCHAR(10),
    action                  VARCHAR(20),

    narrative               TEXT,
    key_drivers             JSONB,
    risk_factors            JSONB,

    recommended_setup       VARCHAR(200),
    position_size_multiplier DECIMAL(4,3),
    stop_distance_multiplier DECIMAL(4,3),

    component_scores        JSONB,
    reason_codes            JSONB,
    data_quality            JSONB,

    model_version           VARCHAR(20),
    config_version          VARCHAR(20),
    shadow_mode             BOOLEAN NOT NULL DEFAULT true,

    -- Link to signal if one was generated
    signal_id               UUID REFERENCES trading.signals(id) ON DELETE SET NULL,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_ptb_analysis_symbol_ts ON trading.ptb_analysis_history(symbol, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_ptb_analysis_quality ON trading.ptb_analysis_history(setup_quality);
CREATE INDEX IF NOT EXISTS idx_ptb_analysis_action ON trading.ptb_analysis_history(action);
CREATE INDEX IF NOT EXISTS idx_ptb_analysis_signal ON trading.ptb_analysis_history(signal_id);

-- Convert to hypertable if TimescaleDB extension is available
-- (idempotent — fails silently if extension not present)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('trading.ptb_analysis_history', 'timestamp', if_not_exists => TRUE);
    END IF;
EXCEPTION WHEN OTHERS THEN
    -- TimescaleDB not available — proceed as regular table
END $$;

-- ============================================================
-- Signal Performance / Feedback (Section 28)
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.signal_performance (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id               UUID NOT NULL REFERENCES trading.signals(id) ON DELETE CASCADE,
    analysis_id             VARCHAR(200),

    entry_price             DECIMAL(18,8),
    exit_price              DECIMAL(18,8),
    pnl                     DECIMAL(18,8),
    pnl_points              DECIMAL(18,8),

    mae                     DECIMAL(18,8),
    mfe                     DECIMAL(18,8),
    time_in_trade_seconds   INTEGER,
    slippage                DECIMAL(18,8),
    execution_quality       VARCHAR(30),

    strategy                VARCHAR(50),
    setup_quality           VARCHAR(10),
    regime_at_entry         VARCHAR(50),
    bias_at_entry           VARCHAR(30),
    gold_role_at_entry      VARCHAR(50),
    manipulation_at_entry   DECIMAL(5,2),
    volatility_at_entry     VARCHAR(50),

    tp1_hit                 BOOLEAN DEFAULT false,
    tp2_hit                 BOOLEAN DEFAULT false,
    tp3_hit                 BOOLEAN DEFAULT false,
    sl_hit                  BOOLEAN DEFAULT false,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at               TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_signal_perf_signal ON trading.signal_performance(signal_id);
CREATE INDEX IF NOT EXISTS idx_signal_perf_strategy ON trading.signal_performance(strategy);
CREATE INDEX IF NOT EXISTS idx_signal_perf_quality ON trading.signal_performance(setup_quality);
CREATE INDEX IF NOT EXISTS idx_signal_perf_regime ON trading.signal_performance(regime_at_entry);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('trading.signal_performance', 'created_at', if_not_exists => TRUE);
    END IF;
EXCEPTION WHEN OTHERS THEN
    NULL;
END $$;
