-- 072_calibration_tables.sql
-- Create calibration tables for storing trained calibration models and predictions
-- SOW Section 16: Calibrated probability rather than raw confidence scores

CREATE SCHEMA IF NOT EXISTS calibration;

-- Calibration models (trained sigmoid parameters per strategy)
CREATE TABLE IF NOT EXISTS calibration.model_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     TEXT NOT NULL,
    prediction_target TEXT NOT NULL DEFAULT 'TP1_HIT',
    sigmoid_a       DOUBLE PRECISION NOT NULL,
    sigmoid_b       DOUBLE PRECISION NOT NULL,
    brier_score     DOUBLE PRECISION DEFAULT 0,
    ece             DOUBLE PRECISION DEFAULT 0,
    sample_size     INTEGER NOT NULL DEFAULT 0,
    wilson_lower    DOUBLE PRECISION DEFAULT 0,
    is_active       BOOLEAN NOT NULL DEFAULT FALSE,
    status          TEXT NOT NULL DEFAULT 'UNVERIFIED', -- UNVERIFIED, SHADOW, VALIDATED, PROMOTED
    oos_brier       DOUBLE PRECISION,
    oos_ece         DOUBLE PRECISION,
    oos_log_loss    DOUBLE PRECISION,
    train_start     TIMESTAMPTZ,
    train_end       TIMESTAMPTZ,
    oos_start       TIMESTAMPTZ,
    oos_end         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT chk_status CHECK (status IN ('UNVERIFIED', 'SHADOW', 'VALIDATED', 'PROMOTED', 'REJECTED'))
);

CREATE INDEX IF NOT EXISTS idx_calib_model_strategy ON calibration.model_versions(strategy_id);
CREATE INDEX IF NOT EXISTS idx_calib_model_active ON calibration.model_versions(is_active) WHERE is_active = true;

-- Calibration predictions (raw score → calibrated probability log)
CREATE TABLE IF NOT EXISTS calibration.predictions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id         TEXT NOT NULL,
    signal_id           TEXT,
    raw_score           DOUBLE PRECISION NOT NULL,
    calibrated_probability DOUBLE PRECISION NOT NULL,
    model_version_id    UUID REFERENCES calibration.model_versions(id),
    model_status        TEXT NOT NULL DEFAULT 'PROVISIONAL',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_calib_pred_strategy ON calibration.predictions(strategy_id);
CREATE INDEX IF NOT EXISTS idx_calib_pred_created ON calibration.predictions(created_at DESC);

-- Calibration outcomes (actual TP1 hit/miss for backtesting calibration)
CREATE TABLE IF NOT EXISTS calibration.outcomes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     TEXT NOT NULL,
    signal_id       TEXT NOT NULL,
    raw_score       DOUBLE PRECISION NOT NULL,
    calibrated_probability DOUBLE PRECISION NOT NULL,
    target_hit      BOOLEAN NOT NULL,
    target_name     TEXT NOT NULL DEFAULT 'TP1_HIT',
    entry_price     DOUBLE PRECISION,
    tp_price        DOUBLE PRECISION,
    sl_price        DOUBLE PRECISION,
    close_price     DOUBLE PRECISION,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at     TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_calib_outcome_strategy ON calibration.outcomes(strategy_id);
CREATE INDEX IF NOT EXISTS idx_calib_outcome_resolved ON calibration.outcomes(resolved_at) WHERE resolved_at IS NULL;

COMMENT ON TABLE calibration.model_versions IS 'Calibration model versions — sigmoid parameters per strategy (SOW Section 16)';
COMMENT ON TABLE calibration.predictions IS 'Calibration prediction log — raw score to calibrated probability mapping';
COMMENT ON TABLE calibration.outcomes IS 'Calibration outcome tracking — actual target hit/miss for model validation';
