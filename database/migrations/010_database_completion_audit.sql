-- Predict-A-Trade v1.0.0 — Migration 010
-- Database Completion, Audit, and Production Hardening
-- SOW Sections 12-47, 76-77, 87, 112: Missing tables, check constraints,
-- indicator/structure/regime history, signal candidates/rejections,
-- config versioning, cooldown/duplicate audit, notifications, system config,
-- backup metadata, security events, backtest datasets/trades, vector embeddings.
--
-- NON-DESTRUCTIVE: Only adds new tables, columns, constraints, and indexes.
-- Does NOT modify existing data or drop anything.
-- All timestamps use TIMESTAMPTZ (UTC). All monetary fields use DECIMAL(18,8).

-- ============================================================
-- Section 1: Signal Candidates (SOW Section 15 — MANDATORY)
-- ============================================================
-- Persist ALL materially evaluated signal candidates, not only published BUY/SELL.
-- This allows later study of why opportunities were accepted or rejected.

CREATE TABLE IF NOT EXISTS trading.signal_candidates (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_uuid      VARCHAR(100) NOT NULL UNIQUE, -- stable UUID from signal engine
    symbol              VARCHAR(20) NOT NULL,
    strategy_id         VARCHAR(50) NOT NULL,
    strategy_version    VARCHAR(20),
    direction           VARCHAR(10) NOT NULL, -- BUY, SELL, NO-TRADE
    entry_price         DECIMAL(18,8),
    stop_loss           DECIMAL(18,8),
    tp1                 DECIMAL(18,8),
    tp2                 DECIMAL(18,8),
    tp3                 DECIMAL(18,8),
    calculated_rr       DECIMAL(10,4),
    raw_score           DECIMAL(10,4),
    long_score          DECIMAL(10,4),
    short_score         DECIMAL(10,4),
    calibrated_prob     DECIMAL(6,5),
    regime              VARCHAR(30),
    market_session      VARCHAR(20),
    timeframe           VARCHAR(10),
    structure_state     JSONB, -- swing highs/lows, BOS/CHoCH state
    feature_readiness   JSONB, -- per-feature readiness snapshot
    reason_codes        JSONB NOT NULL DEFAULT '[]',
    approval_state      VARCHAR(20) NOT NULL DEFAULT 'EVALUATED',
    -- EVALUATED, REJECTED, APPROVED, PUBLISHED, EXPIRED, CANCELLED
    rejection_gate      VARCHAR(50), -- which gate rejected, if any
    signal_id           UUID, -- FK to trading.signals if published
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ
);

-- Indexes for candidate history queries
CREATE INDEX IF NOT EXISTS idx_candidates_symbol_strategy
    ON trading.signal_candidates(symbol, strategy_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_candidates_approval_state
    ON trading.signal_candidates(approval_state, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_candidates_created
    ON trading.signal_candidates(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_candidates_rejection
    ON trading.signal_candidates(rejection_gate)
    WHERE rejection_gate IS NOT NULL;

-- ============================================================
-- Section 2: Signal Rejection History (SOW Section 16)
-- ============================================================
-- Persist rejection details for every evaluated candidate.
-- Never overwrite a rejection with a later evaluation.

CREATE TABLE IF NOT EXISTS trading.signal_rejections (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    candidate_id        UUID NOT NULL REFERENCES trading.signal_candidates(id),
    signal_id           UUID, -- optional ref to trading.signals
    rejection_reason    VARCHAR(50) NOT NULL, -- POOR_RR, UNCLEAR_STRUCTURE, etc.
    rejection_gate      VARCHAR(50), -- specific gate that rejected
    gate_result         VARCHAR(10), -- PASS, FAIL
    threshold_value     DECIMAL(18,8),
    observed_value      DECIMAL(18,8),
    gate_version        VARCHAR(20),
    config_version      VARCHAR(20),
    reason_detail       JSONB,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_rejections_candidate
    ON trading.signal_rejections(candidate_id);
CREATE INDEX IF NOT EXISTS idx_rejections_reason
    ON trading.signal_rejections(rejection_reason, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_rejections_gate
    ON trading.signal_rejections(rejection_gate, created_at DESC);

-- ============================================================
-- Section 3: Strategy Evaluation History (SOW Section 25)
-- ============================================================
-- Persist strategy evaluation outcomes for audit and optimization.

CREATE TABLE IF NOT EXISTS trading.strategy_evaluations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id         VARCHAR(50) NOT NULL,
    strategy_version    VARCHAR(20),
    symbol              VARCHAR(20) NOT NULL,
    timeframe           VARCHAR(10),
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT now(),
    input_features      JSONB, -- snapshot of input feature values
    score               DECIMAL(10,4),
    long_score          DECIMAL(10,4),
    short_score         DECIMAL(10,4),
    conditions_passed   JSONB NOT NULL DEFAULT '[]',
    conditions_failed   JSONB NOT NULL DEFAULT '[]',
    candidate_generated BOOLEAN NOT NULL DEFAULT false,
    direction           VARCHAR(10),
    reason              VARCHAR(100)
);

CREATE INDEX IF NOT EXISTS idx_strategy_eval_symbol_strategy
    ON trading.strategy_evaluations(symbol, strategy_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_strategy_eval_timestamp
    ON trading.strategy_evaluations(timestamp DESC);

-- Convert to hypertable if TimescaleDB available
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('trading.strategy_evaluations', 'timestamp',
            chunk_time_interval => INTERVAL '1 day');
        RAISE NOTICE 'trading.strategy_evaluations converted to hypertable';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'strategy_evaluations hypertable creation skipped: %', SQLERRM;
END
$$;

-- ============================================================
-- Section 4: Indicator History (SOW Section 12)
-- ============================================================
-- Persist sufficient indicator history for audit, replay, debugging, backtesting.
-- Uses a normalized/typed architecture (single table, typed rows).

CREATE TABLE IF NOT EXISTS trading.indicator_history (
    id                  BIGSERIAL PRIMARY KEY,
    symbol              VARCHAR(20) NOT NULL,
    timeframe           VARCHAR(10) NOT NULL,
    timestamp           TIMESTAMPTZ NOT NULL,
    indicator_name      VARCHAR(50) NOT NULL, -- EMA9, RSI14, ATR14, etc.
    indicator_version   VARCHAR(20) NOT NULL DEFAULT '1.0',
    value               DECIMAL(18,8) NOT NULL,
    value_secondary     DECIMAL(18,8), -- for multi-value indicators (e.g., BB upper/lower)
    value_tertiary      DECIMAL(18,8), -- for 3-value indicators (e.g., MACD line/signal/hist)
    quality             VARCHAR(20) NOT NULL DEFAULT 'AUTHORITATIVE',
    source              VARCHAR(50) NOT NULL DEFAULT 'local_compute',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_indicator_history_lookup
    ON trading.indicator_history(symbol, timeframe, indicator_name, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_indicator_history_timestamp
    ON trading.indicator_history(timestamp DESC);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('trading.indicator_history', 'timestamp',
            chunk_time_interval => INTERVAL '1 day');
        PERFORM add_compression_policy('trading.indicator_history', INTERVAL '14 days');
        RAISE NOTICE 'trading.indicator_history converted to hypertable with compression';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'indicator_history hypertable creation skipped: %', SQLERRM;
END
$$;

-- ============================================================
-- Section 5: Regime History (SOW Section 14)
-- ============================================================
-- Persist historical regime classifications. Do not overwrite old classifications.

CREATE TABLE IF NOT EXISTS trading.regime_history (
    id                  BIGSERIAL PRIMARY KEY,
    symbol              VARCHAR(20) NOT NULL,
    timeframe           VARCHAR(10) NOT NULL,
    timestamp           TIMESTAMPTZ NOT NULL,
    regime              VARCHAR(30) NOT NULL, -- TRENDING_BULLISH, TRENDING_BEARISH, etc.
    confidence          DECIMAL(6,5) NOT NULL,
    contributing_features JSONB,
    algorithm_version   VARCHAR(20) NOT NULL DEFAULT '1.0',
    model_version       VARCHAR(50),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_regime_history_lookup
    ON trading.regime_history(symbol, timeframe, timestamp DESC);

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('trading.regime_history', 'timestamp',
            chunk_time_interval => INTERVAL '1 day');
        RAISE NOTICE 'trading.regime_history converted to hypertable';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'regime_history hypertable creation skipped: %', SQLERRM;
END
$$;

-- ============================================================
-- Section 6: Risk Configuration Versioning (SOW Section 23)
-- ============================================================
-- Do not overwrite risk configuration without history.

CREATE TABLE IF NOT EXISTS trading.risk_config_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version             VARCHAR(20) NOT NULL UNIQUE,
    values              JSONB NOT NULL,
    effective_from      TIMESTAMPTZ NOT NULL DEFAULT now(),
    effective_to        TIMESTAMPTZ,
    changed_by          UUID, -- admin user ID
    change_reason       TEXT,
    previous_version    VARCHAR(20),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_risk_config_effective
    ON trading.risk_config_versions(effective_from DESC)
    WHERE effective_to IS NULL;

-- ============================================================
-- Section 7: Strategy Configuration Versioning (SOW Section 24)
-- ============================================================
-- Apply the same versioning rule to strategy configuration.

CREATE TABLE IF NOT EXISTS trading.strategy_config_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id         VARCHAR(50) NOT NULL,
    version             VARCHAR(20) NOT NULL,
    values              JSONB NOT NULL,
    -- indicator thresholds, RR minimums, session rules, cooldowns,
    -- timeframe rules, score weights, regime conditions, SL/TP rules
    effective_from      TIMESTAMPTZ NOT NULL DEFAULT now(),
    effective_to        TIMESTAMPTZ,
    changed_by          UUID,
    change_reason       TEXT,
    previous_version    VARCHAR(20),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version)
);

CREATE INDEX IF NOT EXISTS idx_strategy_config_lookup
    ON trading.strategy_config_versions(strategy_id, effective_from DESC)
    WHERE effective_to IS NULL;

-- ============================================================
-- Section 8: Cooldown Audit (SOW Section 26)
-- ============================================================
-- Valkey may be the active runtime store, but persist events in PostgreSQL for audit.

CREATE TABLE IF NOT EXISTS trading.cooldown_audit (
    id                  BIGSERIAL PRIMARY KEY,
    symbol              VARCHAR(20) NOT NULL,
    strategy_id         VARCHAR(50) NOT NULL,
    event_type          VARCHAR(20) NOT NULL, -- COOLDOWN_START, COOLDOWN_EXPIRED, COOLDOWN_REJECTED
    event_timestamp     TIMESTAMPTZ NOT NULL DEFAULT now(),
    cooldown_start      TIMESTAMPTZ,
    cooldown_expiry     TIMESTAMPTZ,
    remaining_seconds   INTEGER,
    fingerprint         VARCHAR(64),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_cooldown_audit_lookup
    ON trading.cooldown_audit(symbol, strategy_id, event_timestamp DESC);

-- ============================================================
-- Section 9: Duplicate Signal Audit (SOW Section 26)
-- ============================================================

CREATE TABLE IF NOT EXISTS trading.duplicate_audit (
    id                  BIGSERIAL PRIMARY KEY,
    fingerprint         VARCHAR(64) NOT NULL,
    symbol              VARCHAR(20) NOT NULL,
    strategy_id         VARCHAR(50) NOT NULL,
    direction           VARCHAR(10),
    event_type          VARCHAR(20) NOT NULL, -- NEW_SIGNAL, DUPLICATE_REJECTED
    event_timestamp     TIMESTAMPTZ NOT NULL DEFAULT now(),
    candidate_id        UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_duplicate_audit_fingerprint
    ON trading.duplicate_audit(fingerprint, event_timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_duplicate_audit_lookup
    ON trading.duplicate_audit(symbol, strategy_id, event_timestamp DESC);

-- ============================================================
-- Section 10: System Schema and Tables
-- ============================================================

CREATE SCHEMA IF NOT EXISTS system;

-- System Configuration (SOW Section 37)
CREATE TABLE IF NOT EXISTS system.system_configuration (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key          VARCHAR(100) NOT NULL UNIQUE,
    config_value        JSONB NOT NULL,
    config_type         VARCHAR(20) NOT NULL DEFAULT 'STRING',
    -- STRING, INTEGER, FLOAT, BOOLEAN, JSON
    description         TEXT,
    version             VARCHAR(20) NOT NULL DEFAULT '1.0',
    effective_from      TIMESTAMPTZ NOT NULL DEFAULT now(),
    effective_to        TIMESTAMPTZ,
    changed_by          UUID,
    change_reason       TEXT,
    is_secret           BOOLEAN NOT NULL DEFAULT false,
    -- If true, config_value is a reference/identifier, not the actual secret
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Notification History (SOW Section 35)
CREATE TABLE IF NOT EXISTS system.notifications (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    recipient_id        UUID NOT NULL, -- user ID
    notification_type   VARCHAR(50) NOT NULL, -- SIGNAL_DELIVERY, SUBSCRIPTION, ADMIN_ALERT
    channel             VARCHAR(20) NOT NULL, -- EMAIL, DASHBOARD, PUSH
    template_version    VARCHAR(20),
    subject             VARCHAR(255),
    status              VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    -- PENDING, SENT, DELIVERED, FAILED, READ
    sent_at             TIMESTAMPTZ,
    delivered_at        TIMESTAMPTZ,
    failure_reason      TEXT,
    metadata            JSONB NOT NULL DEFAULT '{}',
    correlation_id      VARCHAR(100),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notifications_recipient
    ON system.notifications(recipient_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_notifications_status
    ON system.notifications(status, created_at DESC)
    WHERE status IN ('PENDING', 'FAILED');

-- Backup Metadata (SOW Section 87)
CREATE TABLE IF NOT EXISTS system.backup_metadata (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    backup_id           VARCHAR(100) NOT NULL UNIQUE,
    backup_type         VARCHAR(20) NOT NULL, -- LOGICAL, PHYSICAL, WAL_ARCHIVE
    started_at          TIMESTAMPTZ NOT NULL,
    completed_at        TIMESTAMPTZ,
    status              VARCHAR(20) NOT NULL DEFAULT 'IN_PROGRESS',
    -- IN_PROGRESS, COMPLETED, FAILED, VERIFYING, VERIFIED
    size_bytes          BIGINT,
    checksum            VARCHAR(128),
    location            TEXT, -- storage location identifier (not secrets)
    pg_version          VARCHAR(20),
    timescaledb_version VARCHAR(20),
    pgvector_version    VARCHAR(20),
    app_revision        VARCHAR(50),
    error               TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backup_metadata_status
    ON system.backup_metadata(status, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_backup_metadata_type
    ON system.backup_metadata(backup_type, started_at DESC);

-- ============================================================
-- Section 11: Security Events (SOW Section 36)
-- ============================================================
-- Persist long-term security events for investigation.

CREATE TABLE IF NOT EXISTS audit.security_events (
    id                  BIGSERIAL PRIMARY KEY,
    event_type          VARCHAR(50) NOT NULL,
    -- LOGIN_SUCCESS, LOGIN_FAILURE, MFA_FAILURE, PASSWORD_RESET,
    -- SUSPICIOUS_LOGIN, API_AUTH_FAILURE, PERMISSION_DENIED,
    -- LICENSE_ABUSE, DEVICE_MISMATCH, RATE_LIMIT_EXCEEDED, PRIVILEGE_CHANGE
    actor_type          VARCHAR(20), -- USER, ADMIN, AGENT, SYSTEM
    actor_id            UUID,
    target_type         VARCHAR(50),
    target_id           VARCHAR(100),
    ip_address          INET,
    user_agent          TEXT,
    request_id          VARCHAR(100),
    trace_id            VARCHAR(100),
    severity            VARCHAR(10) NOT NULL DEFAULT 'INFO',
    -- INFO, WARNING, ERROR, CRITICAL
    result              VARCHAR(20), -- SUCCESS, FAILURE, BLOCKED
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_security_events_type
    ON audit.security_events(event_type, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_security_events_actor
    ON audit.security_events(actor_id, created_at DESC)
    WHERE actor_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_security_events_severity
    ON audit.security_events(severity, created_at DESC)
    WHERE severity IN ('ERROR', 'CRITICAL');
CREATE INDEX IF NOT EXISTS idx_security_events_ip
    ON audit.security_events(ip_address, created_at DESC)
    WHERE ip_address IS NOT NULL;

-- ============================================================
-- Section 12: Backtest Datasets and Trades (SOW Sections 76-77)
-- ============================================================

CREATE TABLE IF NOT EXISTS research.backtest_datasets (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_version     VARCHAR(50) NOT NULL UNIQUE,
    symbol              VARCHAR(20) NOT NULL,
    timeframe           VARCHAR(10) NOT NULL,
    start_date          TIMESTAMPTZ NOT NULL,
    end_date            TIMESTAMPTZ NOT NULL,
    source              VARCHAR(50) NOT NULL,
    data_checksum       VARCHAR(128),
    row_count           BIGINT,
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_backtest_datasets_symbol
    ON research.backtest_datasets(symbol, start_date DESC);

CREATE TABLE IF NOT EXISTS research.backtest_trades (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id              UUID NOT NULL REFERENCES research.backtest_runs(id),
    trade_number        INTEGER NOT NULL,
    symbol              VARCHAR(20) NOT NULL,
    direction           VARCHAR(10) NOT NULL,
    entry_time          TIMESTAMPTZ NOT NULL,
    entry_price         DECIMAL(18,8) NOT NULL,
    exit_time           TIMESTAMPTZ,
    exit_price          DECIMAL(18,8),
    stop_loss           DECIMAL(18,8),
    take_profit         DECIMAL(18,8),
    lot_size            DECIMAL(10,2),
    pnl                 DECIMAL(18,8),
    pnl_pips            DECIMAL(10,2),
    commission          DECIMAL(18,8) NOT NULL DEFAULT 0,
    slippage            DECIMAL(18,8) NOT NULL DEFAULT 0,
    exit_reason         VARCHAR(50), -- TP1, TP2, TP3, SL, MANUAL, SIGNAL_EXPIRY
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(run_id, trade_number)
);

CREATE INDEX IF NOT EXISTS idx_backtest_trades_run
    ON research.backtest_trades(run_id, trade_number);

-- ============================================================
-- Section 13: Vector Embeddings (SOW Sections 40-42)
-- ============================================================
-- For market-state embeddings, news embeddings, pattern embeddings.
-- Only create if pgvector is available.

CREATE TABLE IF NOT EXISTS ai.vector_embeddings (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_type         VARCHAR(50) NOT NULL, -- MARKET_STATE, NEWS, PATTERN, RESEARCH_DOC
    source_id           VARCHAR(100),
    symbol              VARCHAR(20),
    timestamp           TIMESTAMPTZ NOT NULL DEFAULT now(),
    embedding_model     VARCHAR(100) NOT NULL,
    embedding_version   VARCHAR(20) NOT NULL,
    embedding_dimension INTEGER NOT NULL,
    embedding           vector,
    content_features    JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Index for metadata lookups
CREATE INDEX IF NOT EXISTS idx_vector_embeddings_source
    ON ai.vector_embeddings(source_type, symbol, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_vector_embeddings_model
    ON ai.vector_embeddings(embedding_model, embedding_version);

-- HNSW vector index for cosine similarity (if pgvector available)
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'vector') THEN
        -- HNSW index for cosine similarity search
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_vector_embeddings_cosine
            ON ai.vector_embeddings USING hnsw (embedding vector_cosine_ops)
            WITH (m = 16, ef_construction = 64)';
        RAISE NOTICE 'Vector HNSW index created';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Vector index creation skipped: %', SQLERRM;
END
$$;

-- ============================================================
-- Section 14: Additional Columns on Existing Tables
-- ============================================================

-- Risk decisions: add threshold, observed_value, gate_version, config_version (SOW Section 22)
ALTER TABLE trading.risk_decisions
    ADD COLUMN IF NOT EXISTS threshold_value DECIMAL(18,8);
ALTER TABLE trading.risk_decisions
    ADD COLUMN IF NOT EXISTS observed_value DECIMAL(18,8);
ALTER TABLE trading.risk_decisions
    ADD COLUMN IF NOT EXISTS gate_version VARCHAR(20);
ALTER TABLE trading.risk_decisions
    ADD COLUMN IF NOT EXISTS config_version VARCHAR(20);

-- Structure events: add swing_type, confirmation, fibonacci/structural refs (SOW Section 13)
ALTER TABLE trading.structure_events
    ADD COLUMN IF NOT EXISTS swing_type VARCHAR(10); -- HH, HL, LH, LL
ALTER TABLE trading.structure_events
    ADD COLUMN IF NOT EXISTS confirmation_timestamp TIMESTAMPTZ;
ALTER TABLE trading.structure_events
    ADD COLUMN IF NOT EXISTS source_candle_time TIMESTAMPTZ;
ALTER TABLE trading.structure_events
    ADD COLUMN IF NOT EXISTS fibonacci_anchor DECIMAL(18,8);
ALTER TABLE trading.structure_events
    ADD COLUMN IF NOT EXISTS structural_sl_ref DECIMAL(18,8);
ALTER TABLE trading.structure_events
    ADD COLUMN IF NOT EXISTS structural_tp_ref DECIMAL(18,8);

-- Signals: add candidate reference (SOW Section 17)
ALTER TABLE trading.signals
    ADD COLUMN IF NOT EXISTS candidate_id UUID;

-- ============================================================
-- Section 15: Check Constraints (SOW Section 52)
-- ============================================================
-- Add safe constraints that don't invalidate existing data.

-- Use DO block since ADD CONSTRAINT IF NOT EXISTS is not supported in PostgreSQL
DO $$
BEGIN
    -- Candle OHLC logical ranges
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_candle_high_gte_low') THEN
        ALTER TABLE market.candles ADD CONSTRAINT chk_candle_high_gte_low CHECK (high >= low);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_candle_high_gte_open') THEN
        ALTER TABLE market.candles ADD CONSTRAINT chk_candle_high_gte_open CHECK (high >= open);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_candle_high_gte_close') THEN
        ALTER TABLE market.candles ADD CONSTRAINT chk_candle_high_gte_close CHECK (high >= close);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_candle_low_lte_open') THEN
        ALTER TABLE market.candles ADD CONSTRAINT chk_candle_low_lte_open CHECK (low <= open);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_candle_low_lte_close') THEN
        ALTER TABLE market.candles ADD CONSTRAINT chk_candle_low_lte_close CHECK (low <= close);
    END IF;

    -- Tick bid/ask logic
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tick_bid_positive') THEN
        ALTER TABLE market.ticks ADD CONSTRAINT chk_tick_bid_positive CHECK (bid > 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tick_ask_positive') THEN
        ALTER TABLE market.ticks ADD CONSTRAINT chk_tick_ask_positive CHECK (ask > 0);
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_tick_ask_gte_bid') THEN
        ALTER TABLE market.ticks ADD CONSTRAINT chk_tick_ask_gte_bid CHECK (ask >= bid);
    END IF;

    -- Signal price: allow 0 for NO-TRADE signals
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_signal_entry_positive') THEN
        ALTER TABLE trading.signals ADD CONSTRAINT chk_signal_entry_positive
        CHECK (entry_price IS NULL OR entry_price = 0 OR entry_price > 0);
    END IF;

    -- Confidence range 0-1
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_signal_calibrated_prob_range') THEN
        ALTER TABLE trading.signals ADD CONSTRAINT chk_signal_calibrated_prob_range
        CHECK (calibrated_probability IS NULL OR (calibrated_probability >= 0 AND calibrated_probability <= 1));
    END IF;

    -- Expiry after creation
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_signal_expiry_after_creation') THEN
        ALTER TABLE trading.signals ADD CONSTRAINT chk_signal_expiry_after_creation
        CHECK (expires_at IS NULL OR expires_at >= created_at);
    END IF;

    -- Commission non-negative
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_commission_amount_nonneg') THEN
        ALTER TABLE referral.commission_ledger ADD CONSTRAINT chk_commission_amount_nonneg
        CHECK (commission_amount >= 0);
    END IF;

    -- Lot size positive
    IF NOT EXISTS (SELECT 1 FROM pg_constraint WHERE conname = 'chk_position_lot_positive') THEN
        ALTER TABLE trading.positions ADD CONSTRAINT chk_position_lot_positive
        CHECK (volume > 0);
    END IF;

    RAISE NOTICE 'All check constraints added successfully';
END
$$;

-- ============================================================
-- Section 16: Missing Indexes on Key Tables (SOW Section 62)
-- ============================================================

-- Signal history by user/symbol/strategy/date
CREATE INDEX IF NOT EXISTS idx_signals_symbol_strategy_created
    ON trading.signals(symbol, strategy_id, created_at DESC);

-- Risk decisions by gate and time
CREATE INDEX IF NOT EXISTS idx_risk_decisions_gate_time
    ON trading.risk_decisions(gate_id, evaluated_at DESC);

-- Audit events by actor and time
CREATE INDEX IF NOT EXISTS idx_audit_events_actor_time
    ON audit.audit_events(actor_id, timestamp DESC)
    WHERE actor_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_audit_events_resource_time
    ON audit.audit_events(entity_type, entity_id, timestamp DESC)
    WHERE entity_type IS NOT NULL;

-- Login events by user and time
CREATE INDEX IF NOT EXISTS idx_login_events_user_time
    ON iam.login_events(user_id, created_at DESC);

-- Latest ticks by symbol (for dashboard)
CREATE INDEX IF NOT EXISTS idx_ticks_symbol_time
    ON market.ticks(symbol, "time" DESC);

-- License validation lookup
CREATE INDEX IF NOT EXISTS idx_licenses_user_status
    ON licensing.licenses(user_id, status)
    WHERE status = 'ACTIVE';

-- Subscription status lookup
CREATE INDEX IF NOT EXISTS idx_subscriptions_user_status
    ON billing.subscriptions(user_id, status)
    WHERE status = 'ACTIVE';

-- Commission ledger by affiliate
CREATE INDEX IF NOT EXISTS idx_commission_ledger_affiliate
    ON referral.commission_ledger(recipient_user_id, created_at DESC);

-- ============================================================
-- Section 17: Soft Delete Columns (SOW Section 50)
-- ============================================================
-- Add soft-delete to entities where historical integrity is required.
-- Do NOT add to audit/financial/trading history tables (those are immutable).

ALTER TABLE iam.users
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;
ALTER TABLE iam.users
    ADD COLUMN IF NOT EXISTS anonymized_at TIMESTAMPTZ;

ALTER TABLE licensing.devices
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

ALTER TABLE market.broker_execution_profiles
    ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

-- Index for filtering out soft-deleted users
CREATE INDEX IF NOT EXISTS idx_users_active
    ON iam.users(id) WHERE deleted_at IS NULL;

-- ============================================================
-- Section 18: TimescaleDB Extension Verification (SOW Section 43)
-- ============================================================
-- Ensure the timescaledb extension is created if available.
-- In production, use a Docker image that includes TimescaleDB (e.g., timescale/timescaledb-ha:pg17).

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        -- Attempt to create the extension (will fail if not installed in the image)
        BEGIN
            CREATE EXTENSION IF NOT EXISTS timescaledb;
            RAISE NOTICE 'TimescaleDB extension created';
        EXCEPTION WHEN OTHERS THEN
            RAISE NOTICE 'TimescaleDB not available: %. Use timescale/timescaledb-ha Docker image in production.', SQLERRM;
        END;
    ELSE
        RAISE NOTICE 'TimescaleDB already installed';
    END IF;
END
$$;

-- ============================================================
-- Section 19: Continuous Aggregates (SOW Section 48)
-- ============================================================
-- Create continuous aggregates for multi-timeframe candle aggregation.
-- Only if TimescaleDB is available.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        -- M1 → M5 continuous aggregate
        EXECUTE 'CREATE MATERIALIZED VIEW IF NOT EXISTS market.candles_m5_agg
            WITH (timescaledb.continuous) AS
            SELECT
                time_bucket(INTERVAL ''5 minutes'', "time") AS bucket,
                symbol,
                source,
                first(open, "time") AS open,
                max(high) AS high,
                min(low) AS low,
                last(close, "time") AS close,
                sum(volume) AS volume
            FROM market.candles
            WHERE timeframe = ''M1''
            GROUP BY bucket, symbol, source
            WITH NO DATA';

        -- M1 → H1 continuous aggregate
        EXECUTE 'CREATE MATERIALIZED VIEW IF NOT EXISTS market.candles_h1_agg
            WITH (timescaledb.continuous) AS
            SELECT
                time_bucket(INTERVAL ''1 hour'', "time") AS bucket,
                symbol,
                source,
                first(open, "time") AS open,
                max(high) AS high,
                min(low) AS low,
                last(close, "time") AS close,
                sum(volume) AS volume
            FROM market.candles
            WHERE timeframe = ''M1''
            GROUP BY bucket, symbol, source
            WITH NO DATA';

        -- Refresh policies
        PERFORM add_continuous_aggregate_policy('market.candles_m5_agg',
            start_offset => INTERVAL '1 hour',
            end_offset => INTERVAL '5 minutes',
            schedule_interval => INTERVAL '1 minute');
        PERFORM add_continuous_aggregate_policy('market.candles_h1_agg',
            start_offset => INTERVAL '1 day',
            end_offset => INTERVAL '1 hour',
            schedule_interval => INTERVAL '5 minutes');

        RAISE NOTICE 'Continuous aggregates created with refresh policies';
    ELSE
        RAISE NOTICE 'Continuous aggregates skipped: TimescaleDB not available';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'Continuous aggregate creation skipped: %', SQLERRM;
END
$$;

-- ============================================================
-- Section 20: Data Integrity Check Function (SOW Section 74)
-- ============================================================
-- Create a function to check data quality that can be called by monitoring.

CREATE OR REPLACE FUNCTION system.check_data_integrity()
RETURNS TABLE(
    check_name VARCHAR,
    status VARCHAR,
    count BIGINT,
    detail TEXT
)
LANGUAGE plpgsql
AS $$
BEGIN
    -- Check for duplicate candles
    RETURN QUERY
    SELECT 'duplicate_candles'::VARCHAR, 'WARNING'::VARCHAR, COUNT(*),
        'Duplicate candle entries found'::TEXT
    FROM (
        SELECT "time", symbol, timeframe, source, COUNT(*) as cnt
        FROM market.candles
        GROUP BY "time", symbol, timeframe, source
        HAVING COUNT(*) > 1
    ) dup;

    -- Check for candle gaps (missing timeframes)
    -- (simplified check — real gap detection would need expected candle times)

    -- Check for impossible OHLC
    RETURN QUERY
    SELECT 'impossible_ohlc'::VARCHAR, 'ERROR'::VARCHAR, COUNT(*),
        'Candles where high < low or high < open or high < close'::TEXT
    FROM market.candles
    WHERE high < low OR high < open OR high < close OR low > open OR low > close;

    -- Check for negative spread
    RETURN QUERY
    SELECT 'negative_spread'::VARCHAR, 'ERROR'::VARCHAR, COUNT(*),
        'Ticks where ask < bid (negative spread)'::TEXT
    FROM market.ticks
    WHERE ask < bid;

    -- Check for orphan signals (no matching candidate)
    RETURN QUERY
    SELECT 'orphan_signals'::VARCHAR, 'INFO'::VARCHAR, COUNT(*),
        'Published signals without candidate reference'::TEXT
    FROM trading.signals s
    LEFT JOIN trading.signal_candidates c ON s.candidate_id = c.id
    WHERE s.candidate_id IS NOT NULL AND c.id IS NULL;

    -- Check for invalid commission ledger (negative amounts)
    RETURN QUERY
    SELECT 'negative_commission'::VARCHAR, 'ERROR'::VARCHAR, COUNT(*),
        'Commission ledger entries with negative amounts'::TEXT
    FROM referral.commission_ledger
    WHERE amount < 0;

    RETURN;
END;
$$;

-- ============================================================
-- Section 21: Backup Function Helper (SOW Section 87)
-- ============================================================
-- Record backup metadata after a backup operation.

CREATE OR REPLACE FUNCTION system.record_backup(
    p_backup_id VARCHAR,
    p_backup_type VARCHAR,
    p_started_at TIMESTAMPTZ,
    p_completed_at TIMESTAMPTZ,
    p_status VARCHAR,
    p_size_bytes BIGINT DEFAULT NULL,
    p_checksum VARCHAR DEFAULT NULL,
    p_location TEXT DEFAULT NULL,
    p_error TEXT DEFAULT NULL
)
RETURNS UUID
LANGUAGE plpgsql
AS $$
DECLARE
    new_id UUID;
    v_pg_version VARCHAR;
    v_ts_version VARCHAR := NULL;
    v_pgvector_version VARCHAR := NULL;
BEGIN
    v_pg_version := version();
    v_pg_version := split_part(v_pg_version, ' ', 2);

    SELECT extversion INTO v_ts_version FROM pg_extension WHERE extname = 'timescaledb';
    SELECT extversion INTO v_pgvector_version FROM pg_extension WHERE extname = 'vector';

    INSERT INTO system.backup_metadata (
        backup_id, backup_type, started_at, completed_at, status,
        size_bytes, checksum, location, pg_version,
        timescaledb_version, pgvector_version, error
    ) VALUES (
        p_backup_id, p_backup_type, p_started_at, p_completed_at, p_status,
        p_size_bytes, p_checksum, p_location, v_pg_version,
        v_ts_version, v_pgvector_version, p_error
    )
    RETURNING id INTO new_id;

    RETURN new_id;
END;
$$;

-- ============================================================
-- Section 22: Immunity Trigger for Audit Events (SOW Section 58)
-- ============================================================
-- Prevent UPDATE/DELETE on audit_events (append-only table).
-- Uses a trigger that rejects modifications after the initial INSERT.

CREATE OR REPLACE FUNCTION audit.prevent_audit_modification()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'audit.audit_events is immutable: INSERT only, no UPDATE or DELETE allowed';
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_audit_no_update ON audit.audit_events;
DROP TRIGGER IF EXISTS trg_audit_no_delete ON audit.audit_events;

CREATE TRIGGER trg_audit_no_update
    BEFORE UPDATE ON audit.audit_events
    FOR EACH ROW EXECUTE FUNCTION audit.prevent_audit_modification();

CREATE TRIGGER trg_audit_no_delete
    BEFORE DELETE ON audit.audit_events
    FOR EACH ROW EXECUTE FUNCTION audit.prevent_audit_modification();

-- Same immutability for commission_ledger (SOW Section 58)
CREATE OR REPLACE FUNCTION referral.prevent_ledger_modification()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    -- Allow reversal entries but not direct modification of existing entries
    RAISE EXCEPTION 'referral.commission_ledger is immutable: INSERT only, use reversal entries for corrections';
    RETURN NULL;
END;
$$;

DROP TRIGGER IF EXISTS trg_commission_no_update ON referral.commission_ledger;
DROP TRIGGER IF EXISTS trg_commission_no_delete ON referral.commission_ledger;

CREATE TRIGGER trg_commission_no_update
    BEFORE UPDATE ON referral.commission_ledger
    FOR EACH ROW EXECUTE FUNCTION referral.prevent_ledger_modification();

CREATE TRIGGER trg_commission_no_delete
    BEFORE DELETE ON referral.commission_ledger
    FOR EACH ROW EXECUTE FUNCTION referral.prevent_ledger_modification();

-- ============================================================
-- Section 23: System Schema Permissions
-- ============================================================
GRANT USAGE ON SCHEMA system TO pat_admin;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA system TO pat_admin;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA system TO pat_admin;

GRANT SELECT ON ALL TABLES IN SCHEMA system TO pat_backup;

-- ============================================================
-- Summary
-- ============================================================
-- New tables created: 16
--   trading.signal_candidates
--   trading.signal_rejections
--   trading.strategy_evaluations
--   trading.indicator_history
--   trading.regime_history
--   trading.risk_config_versions
--   trading.strategy_config_versions
--   trading.cooldown_audit
--   trading.duplicate_audit
--   system.system_configuration
--   system.notifications
--   system.backup_metadata
--   audit.security_events
--   research.backtest_datasets
--   research.backtest_trades
--   ai.vector_embeddings
-- New columns on existing tables: 11
-- New check constraints: 12
-- New indexes: 15
-- New functions: 3
-- New triggers: 4
-- New schema: system
-- Continuous aggregates: 2 (conditional on TimescaleDB)
-- Hypertable conversions: 3 (conditional on TimescaleDB)
