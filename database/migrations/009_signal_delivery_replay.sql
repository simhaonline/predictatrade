-- Predict-A-Trade v1.0.0 — Migration 009
-- Signal Delivery, Sequence Replay, Execution Tracking, AI/ML Pipeline
-- (SOW Sections 19, 29, 44, 47 — Signal Sequence Resume + Idempotent Execution)

-- ============================================================
-- Create ai schema if not exists
-- ============================================================
CREATE SCHEMA IF NOT EXISTS ai;



-- ============================================================
-- Signal Deliveries — tracks per-device signal delivery state
-- ============================================================
CREATE TABLE trading.signal_deliveries (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id           UUID NOT NULL REFERENCES trading.signals(id),
    device_id           UUID,
    license_id          UUID,
    account_id          VARCHAR(50),
    terminal_id         VARCHAR(100),
    sequence_number     BIGINT NOT NULL,
    delivery_state      VARCHAR(20) NOT NULL DEFAULT 'GENERATED',
    -- GENERATED, GATED, QUEUED, SENT, DELIVERED, ACKNOWLEDGED, EXECUTING, EXECUTED, REJECTED, FAILED, EXPIRED, CANCELLED
    sent_at             TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    acknowledged_at     TIMESTAMPTZ,
    executed_at         TIMESTAMPTZ,
    broker_ticket       VARCHAR(100),
    execution_result    JSONB,
    slippage            NUMERIC(10,5),
    total_latency_ms    INTEGER,
    send_attempts       INTEGER NOT NULL DEFAULT 0,
    replay_count        INTEGER NOT NULL DEFAULT 0,
    failure_reason      TEXT,
    last_error          TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_signal_deliveries_signal ON trading.signal_deliveries(signal_id);
CREATE INDEX idx_signal_deliveries_device ON trading.signal_deliveries(device_id);
CREATE INDEX idx_signal_deliveries_license ON trading.signal_deliveries(license_id);
CREATE INDEX idx_signal_deliveries_state ON trading.signal_deliveries(delivery_state);
CREATE UNIQUE INDEX uq_signal_deliveries_signal_device ON trading.signal_deliveries(signal_id, device_id) WHERE device_id IS NOT NULL;

-- ============================================================
-- Signal Sequence Tracking — per-device sequence counter
-- ============================================================
CREATE TABLE trading.signal_sequences (
    device_id           UUID PRIMARY KEY,
    last_sent_sequence  BIGINT NOT NULL DEFAULT 0,
    last_acked_sequence BIGINT NOT NULL DEFAULT 0,
    last_ack_at         TIMESTAMPTZ,
    last_heartbeat_at   TIMESTAMPTZ,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- AI/ML Model Registry
-- ============================================================
CREATE TABLE ai.models (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                VARCHAR(100) NOT NULL,
    version             VARCHAR(50) NOT NULL,
    model_type          VARCHAR(50) NOT NULL, -- classification, regression, ensemble
    description         TEXT,
    status              VARCHAR(20) NOT NULL DEFAULT 'TRAINING',
    -- TRAINING, VALIDATING, READY, ACTIVE, INACTIVE, FAILED, ARCHIVED
    features            JSONB NOT NULL DEFAULT '[]',
    hyperparameters     JSONB NOT NULL DEFAULT '{}',
    metrics             JSONB NOT NULL DEFAULT '{}',
    artifact_path       TEXT,
    activated_at        TIMESTAMPTZ,
    deactivated_at       TIMESTAMPTZ,
    created_by          UUID REFERENCES iam.users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(name, version)
);

CREATE INDEX idx_ai_models_status ON ai.models(status);

-- ============================================================
-- AI/ML Training Jobs
-- ============================================================
CREATE TABLE ai.training_jobs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id             UUID REFERENCES ai.models(id),
    job_name            VARCHAR(100) NOT NULL,
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    -- PENDING, RUNNING, COMPLETED, FAILED, CANCELLED
    dataset_config      JSONB NOT NULL DEFAULT '{}',
    feature_config      JSONB NOT NULL DEFAULT '[]',
    target_config       JSONB NOT NULL DEFAULT '{}',
    train_start         TIMESTAMPTZ,
    train_end           TIMESTAMPTZ,
    validation_start    TIMESTAMPTZ,
    validation_end      TIMESTAMPTZ,
    test_start          TIMESTAMPTZ,
    test_end            TIMESTAMPTZ,
    metrics             JSONB NOT NULL DEFAULT '{}',
    error_message       TEXT,
    started_at          TIMESTAMPTZ,
    completed_at        TIMESTAMPTZ,
    created_by          UUID REFERENCES iam.users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ai_training_jobs_status ON ai.training_jobs(status);
CREATE INDEX idx_ai_training_jobs_model ON ai.training_jobs(model_id);

-- ============================================================
-- AI/ML Inference History
-- ============================================================
CREATE TABLE ai.inference_history (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_id            UUID NOT NULL REFERENCES ai.models(id),
    model_version       VARCHAR(50) NOT NULL,
    feature_timestamp   TIMESTAMPTZ NOT NULL,
    prediction_timestamp TIMESTAMPTZ NOT NULL,
    inference_latency_ms INTEGER,
    prediction          JSONB NOT NULL DEFAULT '{}',
    confidence          NUMERIC(5,4),
    model_health        VARCHAR(20) DEFAULT 'OK',
    stale_feature       BOOLEAN DEFAULT FALSE,
    fallback_used       BOOLEAN DEFAULT FALSE,
    signal_id           UUID REFERENCES trading.signals(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_ai_inference_model ON ai.inference_history(model_id);
CREATE INDEX idx_ai_inference_created ON ai.inference_history(created_at);

-- ============================================================
-- Platform Operations — Admin controls
-- ============================================================
CREATE TABLE control.platform_operations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation_type      VARCHAR(50) NOT NULL,
    -- HALT_TRADING, RESUME_TRADING, ENABLE_STRATEGY, DISABLE_STRATEGY,
    -- PAUSE_SIGNALS, RESUME_SIGNALS, ENABLE_EXECUTION, DISABLE_EXECUTION
    target_type         VARCHAR(50), -- strategy, signal_engine, execution, system
    target_id           VARCHAR(100),
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, REVERTED, EXPIRED
    reason              TEXT,
    actor_id            UUID REFERENCES iam.users(id),
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    reverted_at         TIMESTAMPTZ
);

CREATE INDEX idx_platform_ops_status ON control.platform_operations(status);
CREATE INDEX idx_platform_ops_type ON control.platform_operations(operation_type);



COMMENT ON TABLE trading.signal_deliveries IS 'Per-device signal delivery state machine with sequence tracking';
COMMENT ON TABLE trading.signal_sequences IS 'Per-device sequence counter for resume protocol';
COMMENT ON TABLE ai.models IS 'AI/ML model registry with lifecycle states';
COMMENT ON TABLE ai.training_jobs IS 'Training job tracking with time-series-safe splits';
COMMENT ON TABLE ai.inference_history IS 'Inference provenance with model version, confidence, latency';
COMMENT ON TABLE control.platform_operations IS 'Admin operational controls with audit trail';
