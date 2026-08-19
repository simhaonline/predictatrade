-- Migration 020: Signal truth, traceability, probability, and data durability closure
-- Prompt.md Sections 5-9, 12-20, 27-34, 57-63

-- ============================================================
-- 1. SEQUENCES for monotonic evaluation and signal numbering
-- ============================================================
CREATE SEQUENCE IF NOT EXISTS trading.evaluation_seq START 1;
CREATE SEQUENCE IF NOT EXISTS trading.signal_seq START 1;

-- ============================================================
-- 2. SIGNAL TABLE: Add traceability, provenance, score status fields
-- ============================================================
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS evaluation_sequence BIGINT DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS signal_sequence BIGINT DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS signal_reference VARCHAR(40) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS source_mode VARCHAR(30) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS source_sequence BIGINT DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS source_timestamp TIMESTAMPTZ;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS ingest_timestamp TIMESTAMPTZ;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS market_bar_open_time TIMESTAMPTZ;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS market_bar_close_time TIMESTAMPTZ;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS bid_price NUMERIC(18,8) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS ask_price NUMERIC(18,8) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS bar_closed VARCHAR(25) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS provenance_state VARCHAR(20) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS calibration_status VARCHAR(20) DEFAULT 'UNVERIFIED';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS calibration_model_id VARCHAR(50) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS calibration_model_version VARCHAR(20) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS calibration_target VARCHAR(50) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS calibration_sample_count INTEGER DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS calibration_artifact_hash VARCHAR(64) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS score_status VARCHAR(25) DEFAULT 'COMPUTED';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS dominance NUMERIC(10,4) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS transition_long_score NUMERIC(10,4) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS transition_short_score NUMERIC(10,4) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS transition_conflict NUMERIC(10,4) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS transition_final_score NUMERIC(10,4) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS transition_candidate_threshold NUMERIC(5,2) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS is_transition_candidate BOOLEAN DEFAULT FALSE;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS primary_blocker VARCHAR(50) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS secondary_blockers JSONB DEFAULT '[]';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS input_hash VARCHAR(64) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS decision_hash VARCHAR(64) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS replay_verification VARCHAR(10) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS agent_id VARCHAR(100) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS strategy_version VARCHAR(20) DEFAULT '1.0';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS feature_version VARCHAR(20) DEFAULT '1.0';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS risk_profile_version VARCHAR(20) DEFAULT '1.0';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS regime_version VARCHAR(20) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS outbox_state VARCHAR(20) DEFAULT '';

-- Make calibrated_probability nullable (prompt.md Section 18)
ALTER TABLE trading.signals ALTER COLUMN calibrated_probability DROP NOT NULL;

-- Index for signal reference search
CREATE INDEX IF NOT EXISTS idx_signals_signal_reference ON trading.signals (signal_reference) WHERE signal_reference != '';
CREATE INDEX IF NOT EXISTS idx_signals_evaluation_seq ON trading.signals (evaluation_sequence) WHERE evaluation_sequence > 0;
CREATE INDEX IF NOT EXISTS idx_signals_signal_seq ON trading.signals (signal_sequence) WHERE signal_sequence > 0;

-- ============================================================
-- 3. STRATEGY_EVALUATIONS: Add evaluation_sequence, score_status
-- ============================================================
ALTER TABLE trading.strategy_evaluations ADD COLUMN IF NOT EXISTS evaluation_sequence BIGINT DEFAULT 0;
ALTER TABLE trading.strategy_evaluations ADD COLUMN IF NOT EXISTS score_status VARCHAR(25) DEFAULT 'COMPUTED';
ALTER TABLE trading.strategy_evaluations ADD COLUMN IF NOT EXISTS source_mode VARCHAR(30) DEFAULT '';
ALTER TABLE trading.strategy_evaluations ADD COLUMN IF NOT EXISTS conflict_penalty NUMERIC(10,4) DEFAULT 0;
ALTER TABLE trading.strategy_evaluations ADD COLUMN IF NOT EXISTS dominance NUMERIC(10,4) DEFAULT 0;
ALTER TABLE trading.strategy_evaluations ALTER COLUMN direction TYPE varchar(30);
ALTER TABLE trading.strategy_evaluations ALTER COLUMN reason TYPE text;

-- ============================================================
-- 4. SIGNAL_CANDIDATES: Fix varchar widths, add reference
-- ============================================================
ALTER TABLE trading.signal_candidates ALTER COLUMN direction TYPE varchar(30);
ALTER TABLE trading.signal_candidates ALTER COLUMN approval_state TYPE varchar(30);
ALTER TABLE trading.signal_candidates ALTER COLUMN rejection_gate TYPE varchar(60);
ALTER TABLE trading.signal_candidates ADD COLUMN IF NOT EXISTS signal_reference VARCHAR(40) DEFAULT '';
ALTER TABLE trading.signal_candidates ADD COLUMN IF NOT EXISTS evaluation_sequence BIGINT DEFAULT 0;
ALTER TABLE trading.signal_candidates ADD COLUMN IF NOT EXISTS score_status VARCHAR(25) DEFAULT 'COMPUTED';
ALTER TABLE trading.signal_candidates ADD COLUMN IF NOT EXISTS source_mode VARCHAR(30) DEFAULT '';

-- ============================================================
-- 5. SIGNAL OUTBOX table (prompt.md Sections 32-34)
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.signal_outbox (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id UUID NOT NULL,
    signal_reference VARCHAR(40) DEFAULT '',
    event_type VARCHAR(30) NOT NULL DEFAULT 'SIGNAL_CREATED',
    payload JSONB NOT NULL DEFAULT '{}',
    state VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    attempt_count INTEGER NOT NULL DEFAULT 0,
    max_attempts INTEGER NOT NULL DEFAULT 10,
    last_attempt_at TIMESTAMPTZ,
    next_retry_at TIMESTAMPTZ DEFAULT NOW(),
    last_error TEXT DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_outbox_pending ON trading.signal_outbox (next_retry_at) WHERE state = 'PENDING';
CREATE INDEX IF NOT EXISTS idx_outbox_retrying ON trading.signal_outbox (next_retry_at) WHERE state = 'RETRYING';
CREATE INDEX IF NOT EXISTS idx_outbox_signal_id ON trading.signal_outbox (signal_id);

-- ============================================================
-- 6. HYPERTABLE: make strategy_evaluations a hypertable if not already
-- ============================================================
SELECT create_hypertable('trading.strategy_evaluations', 'timestamp', if_not_exists => TRUE);

-- ============================================================
-- 7. UNIQUE CONSTRAINT for canonical idempotency (prompt.md Section 13)
-- ============================================================
-- Prevent duplicate canonical signals for same strategy+bar+direction+class
CREATE UNIQUE INDEX IF NOT EXISTS idx_signals_canonical_idempotency 
ON trading.signals (strategy_id, market_bar_close_time, direction, signal_class)
WHERE signal_class IN ('EXECUTABLE', 'ADVISORY') 
  AND market_bar_close_time IS NOT NULL
  AND direction IN ('BUY', 'SELL', 'BUY_CANDIDATE', 'SELL_CANDIDATE');

COMMENT ON TABLE trading.signals IS 'Canonical trading signals with full traceability: signal_reference, evaluation_sequence, source provenance, calibration status, outbox state';
COMMENT ON TABLE trading.signal_outbox IS 'Transactional outbox for durable signal publication (prompt.md Sections 32-34)';
-- Fix session column widths for overlap session names
ALTER TABLE trading.signal_candidates ALTER COLUMN market_session TYPE varchar(50);
ALTER TABLE trading.signals ALTER COLUMN session TYPE varchar(50);
