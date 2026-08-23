-- Migration 027: audit.client_events — TimescaleDB hypertable for compliance audit logging
-- Follows prompt.md specification for table name in audit schema

-- Create table in audit schema (existing schema, no new schema needed)
CREATE TABLE IF NOT EXISTS audit.client_events (
    event_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_id UUID NOT NULL DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,

    request_id UUID,
    correlation_id UUID,

    user_id UUID,
    account_id UUID,
    session_id UUID,
    prediction_id UUID,

    http_method TEXT,
    endpoint TEXT,
    http_status SMALLINT,
    latency_ms INTEGER,

    client_ip INET,

    country_code CHAR(2),
    country_name TEXT,
    region TEXT,
    city TEXT,
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

    application_version TEXT,

    metadata JSONB,

    PRIMARY KEY (event_time, event_id)
);

-- Convert to TimescaleDB hypertable
SELECT create_hypertable(
    'audit.client_events',
    'event_time',
    if_not_exists => TRUE,
    chunk_time_interval => INTERVAL '7 days'
);

-- Indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_client_events_user_time
ON audit.client_events (user_id, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_client_events_request
ON audit.client_events (request_id);

CREATE INDEX IF NOT EXISTS idx_client_events_ip_time
ON audit.client_events (client_ip, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_client_events_type_time
ON audit.client_events (event_type, event_time DESC);

CREATE INDEX IF NOT EXISTS idx_client_events_prediction
ON audit.client_events (prediction_id, event_time DESC);

-- Grant permissions
GRANT INSERT, SELECT ON audit.client_events TO pat_admin;
