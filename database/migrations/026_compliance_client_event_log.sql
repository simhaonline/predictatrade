-- Migration 026: Compliance Client Event Log
-- Production-grade audit telemetry table with TimescaleDB hypertable
-- Extends existing audit.audit_events with full client/network/telemetry context

-- 1. Add columns to existing audit.audit_events table (additive, no destructive changes)
ALTER TABLE audit.audit_events
    ADD COLUMN IF NOT EXISTS event_type_detailed TEXT,
    ADD COLUMN IF NOT EXISTS event_version INTEGER DEFAULT 1,
    ADD COLUMN IF NOT EXISTS http_method TEXT,
    ADD COLUMN IF NOT EXISTS endpoint TEXT,
    ADD COLUMN IF NOT EXISTS http_status SMALLINT,
    ADD COLUMN IF NOT EXISTS latency_ms INTEGER,
    ADD COLUMN IF NOT EXISTS client_ip INET,
    ADD COLUMN IF NOT EXISTS proxy_chain JSONB,
    ADD COLUMN IF NOT EXISTS geo_country_code CHAR(2),
    ADD COLUMN IF NOT EXISTS geo_region TEXT,
    ADD COLUMN IF NOT EXISTS geo_city TEXT,
    ADD COLUMN IF NOT EXISTS isp TEXT,
    ADD COLUMN IF NOT EXISTS asn BIGINT,
    ADD COLUMN IF NOT EXISTS as_org TEXT,
    ADD COLUMN IF NOT EXISTS browser_name TEXT,
    ADD COLUMN IF NOT EXISTS browser_version TEXT,
    ADD COLUMN IF NOT EXISTS os_name TEXT,
    ADD COLUMN IF NOT EXISTS os_version TEXT,
    ADD COLUMN IF NOT EXISTS device_type TEXT,
    ADD COLUMN IF NOT EXISTS language TEXT,
    ADD COLUMN IF NOT EXISTS languages JSONB,
    ADD COLUMN IF NOT EXISTS timezone TEXT,
    ADD COLUMN IF NOT EXISTS timezone_offset_minutes SMALLINT,
    ADD COLUMN IF NOT EXISTS screen_width INTEGER,
    ADD COLUMN IF NOT EXISTS screen_height INTEGER,
    ADD COLUMN IF NOT EXISTS screen_available_width INTEGER,
    ADD COLUMN IF NOT EXISTS screen_available_height INTEGER,
    ADD COLUMN IF NOT EXISTS viewport_width INTEGER,
    ADD COLUMN IF NOT EXISTS viewport_height INTEGER,
    ADD COLUMN IF NOT EXISTS device_pixel_ratio NUMERIC(6,3),
    ADD COLUMN IF NOT EXISTS color_depth SMALLINT,
    ADD COLUMN IF NOT EXISTS touch_points SMALLINT,
    ADD COLUMN IF NOT EXISTS client_hints JSONB,
    ADD COLUMN IF NOT EXISTS prediction_id UUID,
    ADD COLUMN IF NOT EXISTS application_version TEXT,
    ADD COLUMN IF NOT EXISTS api_version TEXT,
    ADD COLUMN IF NOT EXISTS risk_flags JSONB,
    ADD COLUMN IF NOT EXISTS metadata_jsonb JSONB,
    ADD COLUMN IF NOT EXISTS telemetry_schema_version INTEGER DEFAULT 1;

-- 2. Create compliance schema for extended event log
CREATE SCHEMA IF NOT EXISTS compliance;

-- 3. Create client_event_log table for high-volume telemetry events
CREATE TABLE IF NOT EXISTS compliance.client_event_log (
    event_id UUID NOT NULL DEFAULT gen_random_uuid(),
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    event_type TEXT NOT NULL,
    event_version INTEGER NOT NULL DEFAULT 1,
    telemetry_schema_version INTEGER NOT NULL DEFAULT 1,

    request_id UUID,
    correlation_id UUID,

    user_id UUID,
    account_id UUID,
    session_id UUID,

    source TEXT NOT NULL,

    http_method TEXT,
    endpoint TEXT,
    http_status SMALLINT,
    latency_ms INTEGER,

    client_ip INET,
    proxy_chain JSONB,

    geo_country_code CHAR(2),
    geo_region TEXT,
    geo_city TEXT,

    isp TEXT,
    asn BIGINT,
    as_org TEXT,

    user_agent TEXT,
    browser_name TEXT,
    browser_version TEXT,
    os_name TEXT,
    os_version TEXT,
    device_type TEXT,

    language TEXT,
    languages JSONB,

    timezone TEXT,
    timezone_offset_minutes SMALLINT,

    screen_width INTEGER,
    screen_height INTEGER,
    screen_available_width INTEGER,
    screen_available_height INTEGER,

    viewport_width INTEGER,
    viewport_height INTEGER,

    device_pixel_ratio NUMERIC(6,3),
    color_depth SMALLINT,
    touch_points SMALLINT,

    client_hints JSONB,

    prediction_id UUID,

    application_version TEXT,
    api_version TEXT,

    risk_flags JSONB,
    metadata JSONB,

    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (event_time, event_id)
);

-- 4. Convert to TimescaleDB hypertable
SELECT create_hypertable(
    'compliance.client_event_log',
    by_column => 'event_time',
    if_not_exists => TRUE,
    chunk_time_interval => INTERVAL '7 days'
);

-- 5. Create indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_client_event_user_time
ON compliance.client_event_log (user_id, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_client_event_request
ON compliance.client_event_log (request_id);

CREATE INDEX IF NOT EXISTS idx_client_event_ip_time
ON compliance.client_event_log (client_ip, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_client_event_type_time
ON compliance.client_event_log (event_type, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_client_event_prediction
ON compliance.client_event_log (prediction_id, event_time DESC);

-- 6. Add indexes to existing audit_events for new columns
CREATE INDEX IF NOT EXISTS idx_audit_events_client_ip
ON audit.audit_events (source_ip);

CREATE INDEX IF NOT EXISTS idx_audit_events_event_type_detailed
ON audit.audit_events (event_type_detailed);

-- 7. Grant permissions (audit writer can INSERT but not UPDATE/DELETE)
GRANT INSERT ON compliance.client_event_log TO pat_admin;
GRANT SELECT ON compliance.client_event_log TO pat_admin;

-- 8. Optional retention policy (disabled by default, enable via env config)
-- To enable: SELECT add_retention_policy('compliance.client_event_log', INTERVAL '365 days');

-- Record migration
INSERT INTO audit.migration_history (migration_id, applied_at, description)
VALUES ('026', NOW(), 'Compliance client event log with TimescaleDB hypertable')
ON CONFLICT DO NOTHING;
