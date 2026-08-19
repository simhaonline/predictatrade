-- Predict-A-Trade v1.0.0 — Migration 011
-- COT Provider, Data Capability Registry, WAL Configuration
-- SOW Sections 8, 10, 11, 12, 16, 17 (prompt.md Blockers B, C, D, E)

-- ============================================================
-- Section 1: COT Data Tables (SOW Section 8 — Blocker B)
-- ============================================================
-- COT (Commitment of Traders) is macro/positioning context, NOT execution-critical.
-- Provider adapter architecture for ingesting official COT data.
-- Non-blocking: COT weight = 0 for all strategies by default.

CREATE TABLE IF NOT EXISTS trading.cot_raw_reports (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_date         DATE NOT NULL,
    publication_date    DATE,
    source              VARCHAR(100) NOT NULL, -- CFTC, provider name
    source_url          TEXT,
    raw_payload         JSONB NOT NULL,
    raw_payload_hash    VARCHAR(128) NOT NULL, -- SHA-256 of raw payload
    ingestion_run_id    UUID,
    ingested_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(report_date, source)
);

CREATE TABLE IF NOT EXISTS trading.cot_reports (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    raw_report_id       UUID NOT NULL REFERENCES trading.cot_raw_reports(id),
    report_date         DATE NOT NULL,
    publication_date    DATE,
    market              VARCHAR(50) NOT NULL, -- GOLD, SILVER, etc.
    contract_code       VARCHAR(20) NOT NULL, -- GC, SI, etc.
    source              VARCHAR(100) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(report_date, contract_code, source)
);

CREATE TABLE IF NOT EXISTS trading.cot_positions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id           UUID NOT NULL REFERENCES trading.cot_reports(id),
    report_date         DATE NOT NULL,
    contract_code       VARCHAR(20) NOT NULL,
    commercial_long     BIGINT,
    commercial_short    BIGINT,
    non_commercial_long BIGINT,
    non_commercial_short BIGINT,
    non_reportable_long BIGINT,
    non_reportable_short BIGINT,
    open_interest       BIGINT,
    net_positioning     BIGINT, -- non-commercial long - short
    weekly_delta        BIGINT, -- change from previous week
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS trading.cot_features (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id           UUID NOT NULL REFERENCES trading.cot_reports(id),
    report_date         DATE NOT NULL,
    contract_code       VARCHAR(20) NOT NULL,
    net_percentile      DECIMAL(6,5), -- 0-1 percentile of net positioning
    net_zscore          DECIMAL(10,6), -- z-score of net positioning
    index_26week        DECIMAL(6,5), -- 26-week index
    index_52week        DECIMAL(6,5), -- 52-week index
    calculated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS trading.cot_ingestion_runs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider            VARCHAR(100) NOT NULL,
    started_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at        TIMESTAMPTZ,
    status              VARCHAR(20) NOT NULL DEFAULT 'RUNNING',
    -- RUNNING, COMPLETED, FAILED, PARTIAL
    reports_fetched     INTEGER NOT NULL DEFAULT 0,
    error_message       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS trading.cot_provider_health (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider            VARCHAR(100) NOT NULL UNIQUE,
    status              VARCHAR(20) NOT NULL DEFAULT 'UNCONFIGURED',
    -- AVAILABLE, STALE, UNAVAILABLE, DISABLED, UNCONFIGURED
    last_success_at     TIMESTAMPTZ,
    last_attempt_at     TIMESTAMPTZ,
    last_report_date     DATE,
    stale_after_hours   INTEGER NOT NULL DEFAULT 168, -- 1 week
    error_count         INTEGER NOT NULL DEFAULT 0,
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Insert default provider health (unconfigured)
INSERT INTO trading.cot_provider_health (provider, status)
SELECT 'CFTC', 'UNCONFIGURED'
ON CONFLICT (provider) DO NOTHING;

-- COT indexes
CREATE INDEX IF NOT EXISTS idx_cot_raw_date ON trading.cot_raw_reports(report_date DESC);
CREATE INDEX IF NOT EXISTS idx_cot_reports_date ON trading.cot_reports(report_date DESC);
CREATE INDEX IF NOT EXISTS idx_cot_positions_date ON trading.cot_positions(report_date DESC, contract_code);
CREATE INDEX IF NOT EXISTS idx_cot_features_date ON trading.cot_features(report_date DESC, contract_code);

-- ============================================================
-- Section 2: Data Capability Registry (SOW Section 12)
-- ============================================================
-- Central mechanism so signal engine consults capability state
-- rather than scattered if/else checks throughout the codebase.

CREATE TABLE IF NOT EXISTS trading.data_capabilities (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    capability          VARCHAR(50) NOT NULL,
    -- PRICE, BID_ASK, BROKER_TICK_ACTIVITY, REAL_VOLUME,
    -- VOLUME_PROFILE, CUMULATIVE_DELTA, COT, MARKET_STRUCTURE,
    -- BOS_CHOCH, LIQUIDITY_SWEEP, ATR, SPREAD, MARKET_SESSION, NEWS
    instrument          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    source              VARCHAR(100), -- broker name, EXCHANGE, DERIVED, UNAVAILABLE
    provenance          VARCHAR(50) NOT NULL DEFAULT 'UNAVAILABLE',
    -- BROKER, EXCHANGE, DERIVED, PROXY, UNAVAILABLE
    status              VARCHAR(50) NOT NULL DEFAULT 'UNAVAILABLE',
    -- AVAILABLE, AVAILABLE_PROXY, DEGRADED, STALE, UNAVAILABLE, DISABLED
    quality_score       DECIMAL(6,5) NOT NULL DEFAULT 0, -- 0-1
    last_event_at       TIMESTAMPTZ,
    max_staleness_ms    BIGINT, -- max acceptable staleness
    strategy_eligible   BOOLEAN NOT NULL DEFAULT false,
    fallback            VARCHAR(50), -- fallback capability name
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(capability, instrument)
);

-- Seed known capabilities
INSERT INTO trading.data_capabilities (capability, instrument, source, provenance, status, strategy_eligible)
VALUES
    ('PRICE', 'XAUUSD', 'BROKER', 'BROKER', 'AVAILABLE', true),
    ('BID_ASK', 'XAUUSD', 'BROKER', 'BROKER', 'AVAILABLE', true),
    ('BROKER_TICK_ACTIVITY', 'XAUUSD', 'BROKER', 'BROKER', 'AVAILABLE', true),
    ('REAL_VOLUME', 'XAUUSD', NULL, 'UNAVAILABLE', 'UNAVAILABLE', false),
    ('VOLUME_PROFILE', 'XAUUSD', NULL, 'UNAVAILABLE', 'UNAVAILABLE', false),
    ('CUMULATIVE_DELTA', 'XAUUSD', NULL, 'UNAVAILABLE', 'UNAVAILABLE', false),
    ('COT', 'XAUUSD', NULL, 'UNAVAILABLE', 'UNAVAILABLE', false),
    ('MARKET_STRUCTURE', 'XAUUSD', 'DERIVED', 'DERIVED', 'AVAILABLE', true),
    ('BOS_CHOCH', 'XAUUSD', 'DERIVED', 'DERIVED', 'AVAILABLE', true),
    ('LIQUIDITY_SWEEP', 'XAUUSD', 'DERIVED', 'DERIVED', 'AVAILABLE', true),
    ('ATR', 'XAUUSD', 'DERIVED', 'DERIVED', 'AVAILABLE', true),
    ('SPREAD', 'XAUUSD', 'BROKER', 'BROKER', 'AVAILABLE', true),
    ('MARKET_SESSION', 'XAUUSD', 'DERIVED', 'DERIVED', 'AVAILABLE', true),
    ('NEWS', 'XAUUSD', NULL, 'UNAVAILABLE', 'UNAVAILABLE', false)
ON CONFLICT (capability, instrument) DO NOTHING;

CREATE INDEX IF NOT EXISTS idx_data_capabilities_lookup
    ON trading.data_capabilities(capability, instrument);

-- Function to check if a capability is available for signal generation
CREATE OR REPLACE FUNCTION trading.is_capability_available(
    p_capability VARCHAR,
    p_instrument VARCHAR DEFAULT 'XAUUSD'
) RETURNS BOOLEAN AS $$
DECLARE
    v_status VARCHAR;
    v_eligible BOOLEAN;
BEGIN
    SELECT status, strategy_eligible INTO v_status, v_eligible
    FROM trading.data_capabilities
    WHERE capability = p_capability AND instrument = p_instrument;

    IF NOT FOUND THEN
        RETURN false;
    END IF;

    RETURN v_eligible AND v_status IN ('AVAILABLE', 'AVAILABLE_PROXY');
END;
$$ LANGUAGE plpgsql;

-- ============================================================
-- Section 3: WAL Archiving Configuration (SOW Section 17 — Blocker E)
-- ============================================================
-- This migration documents WAL archiving requirements.
-- Actual WAL configuration must be set in postgresql.conf or via ALTER SYSTEM.
-- We create a metadata table to track WAL archive status.

CREATE TABLE IF NOT EXISTS system.wal_archive_status (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wal_file            VARCHAR(100) NOT NULL UNIQUE,
    archived_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    archive_location    TEXT NOT NULL,
    size_bytes          BIGINT,
    checksum            VARCHAR(128),
    status              VARCHAR(20) NOT NULL DEFAULT 'ARCHIVED',
    -- ARCHIVED, FAILED, VERIFIED
    error               TEXT
);

CREATE INDEX IF NOT EXISTS idx_wal_status_time
    ON system.wal_archive_status(archived_at DESC);

-- Record WAL archiving configuration status
CREATE TABLE IF NOT EXISTS system.backup_configuration (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_key          VARCHAR(100) NOT NULL UNIQUE,
    config_value        TEXT,
    is_configured        BOOLEAN NOT NULL DEFAULT false,
    required_for_prod   BOOLEAN NOT NULL DEFAULT false,
    description         TEXT,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO system.backup_configuration (config_key, config_value, is_configured, required_for_prod, description)
VALUES
    ('wal_level', 'replica', false, true, 'WAL level for archiving (replica or logical)'),
    ('archive_mode', 'off', false, true, 'WAL archiving on/off'),
    ('archive_command', '', false, true, 'Command to copy WAL files to archive'),
    ('max_wal_senders', '0', false, true, 'Max WAL sender connections'),
    ('wal_keep_size', '0', false, false, 'WAL size to retain for replication'),
    ('off_host_backup_provider', '', false, true, 'Off-host backup storage provider (S3, NFS, etc.)'),
    ('off_host_backup_bucket', '', false, true, 'Backup bucket/container name'),
    ('off_host_backup_region', '', false, false, 'Backup storage region'),
    ('off_host_backup_endpoint', '', false, false, 'Custom endpoint (non-AWS)'),
    ('off_host_backup_prefix', 'predictatrade', false, false, 'Backup path prefix'),
    ('off_host_backup_encryption', 'none', false, true, 'Backup encryption (none, sse, cse)')
ON CONFLICT (config_key) DO NOTHING;
