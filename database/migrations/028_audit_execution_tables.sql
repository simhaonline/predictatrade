-- Migration 028: Audit execution tables for pipeline, score, and signal traceability
-- Extends existing audit schema with execution logging (TimescaleDB hypertables)

-- 1. Pipeline executions
CREATE TABLE IF NOT EXISTS audit.pipeline_executions (
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    pipeline_execution_id UUID NOT NULL DEFAULT gen_random_uuid(),
    
    asset TEXT,
    timeframe TEXT,
    market_data_timestamp TIMESTAMPTZ,
    
    pipeline_version TEXT,
    strategy_version TEXT,
    configuration_version TEXT,
    
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    latency_ms INTEGER,
    status TEXT NOT NULL DEFAULT 'RUNNING',
    
    signal_id UUID,
    prediction_id UUID,
    request_id UUID,
    correlation_id UUID,
    
    metadata JSONB,
    
    PRIMARY KEY (event_time, pipeline_execution_id)
);

SELECT create_hypertable('audit.pipeline_executions', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');

CREATE INDEX IF NOT EXISTS idx_pipeline_signal ON audit.pipeline_executions (signal_id);
CREATE INDEX IF NOT EXISTS idx_pipeline_prediction ON audit.pipeline_executions (prediction_id);

-- 2. Pipeline steps (engine/indicator execution)
CREATE TABLE IF NOT EXISTS audit.pipeline_steps (
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    step_id UUID NOT NULL DEFAULT gen_random_uuid(),
    pipeline_execution_id UUID NOT NULL,
    
    engine_name TEXT NOT NULL,
    engine_version TEXT,
    timeframe TEXT,
    
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    latency_ms INTEGER,
    status TEXT NOT NULL DEFAULT 'RUNNING',
    
    raw_value NUMERIC,
    normalized_value NUMERIC,
    direction TEXT,
    confidence NUMERIC,
    weight NUMERIC,
    
    metadata JSONB,
    
    PRIMARY KEY (event_time, step_id)
);

SELECT create_hypertable('audit.pipeline_steps', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');

CREATE INDEX IF NOT EXISTS idx_pipeline_steps_exec ON audit.pipeline_steps (pipeline_execution_id);

-- 3. Score executions
CREATE TABLE IF NOT EXISTS audit.score_executions (
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    score_execution_id UUID NOT NULL DEFAULT gen_random_uuid(),
    pipeline_execution_id UUID NOT NULL,
    
    score_version TEXT,
    raw_score NUMERIC,
    normalized_score NUMERIC,
    bullish_score NUMERIC,
    bearish_score NUMERIC,
    confidence NUMERIC,
    signal TEXT,
    signal_grade TEXT,
    
    strategy_id TEXT,
    asset TEXT,
    timeframe TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (event_time, score_execution_id)
);

SELECT create_hypertable('audit.score_executions', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');

CREATE INDEX IF NOT EXISTS idx_score_pipeline ON audit.score_executions (pipeline_execution_id);

-- 4. Score components (pillar contributions)
CREATE TABLE IF NOT EXISTS audit.score_components (
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    component_id UUID NOT NULL DEFAULT gen_random_uuid(),
    score_execution_id UUID NOT NULL,
    
    pillar_name TEXT NOT NULL,
    raw_score NUMERIC,
    weight NUMERIC,
    weighted_contribution NUMERIC,
    normalized_score NUMERIC,
    confidence NUMERIC,
    direction TEXT,
    status TEXT,
    feature_name TEXT,
    
    PRIMARY KEY (event_time, component_id)
);

SELECT create_hypertable('audit.score_components', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');

CREATE INDEX IF NOT EXISTS idx_score_components_exec ON audit.score_components (score_execution_id);

-- 5. Signal executions (final decision)
CREATE TABLE IF NOT EXISTS audit.signal_executions (
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    signal_id UUID NOT NULL DEFAULT gen_random_uuid(),
    pipeline_execution_id UUID NOT NULL,
    score_execution_id UUID NOT NULL,
    
    asset TEXT,
    timeframe TEXT,
    signal TEXT NOT NULL,
    decision TEXT,
    score NUMERIC,
    confidence NUMERIC,
    signal_grade TEXT,
    
    entry NUMERIC,
    stop_loss NUMERIC,
    take_profit NUMERIC,
    risk_reward NUMERIC,
    
    decision_reason TEXT,
    strategy_id TEXT,
    
    market_data_timestamp TIMESTAMPTZ,
    data_source TEXT,
    
    application_version TEXT,
    
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (event_time, signal_id)
);

SELECT create_hypertable('audit.signal_executions', 'event_time', if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');

CREATE INDEX IF NOT EXISTS idx_signal_exec_pipeline ON audit.signal_executions (pipeline_execution_id);
CREATE INDEX IF NOT EXISTS idx_signal_exec_strategy ON audit.signal_executions (strategy_id, event_time DESC);

-- Grant permissions
GRANT INSERT, SELECT ON ALL TABLES IN SCHEMA audit TO pat_admin;
