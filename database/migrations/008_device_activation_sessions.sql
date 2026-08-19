-- Predict-A-Trade v1.0.0 — Migration 008
-- Device Activation, Session Leasing, Credential Rotation
-- (SOW Sections 40-42, 62 — MT4/MT5 Licensing & Device Binding)
--
-- Implements: one license = one device, one active session, hardware fingerprint,
-- device credentials, refresh token rotation, session lease with TTL.

-- ============================================================
-- Device Activations
-- ============================================================
CREATE TABLE licensing.device_activations (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id          UUID NOT NULL REFERENCES licensing.licenses(id),
    device_id           UUID NOT NULL REFERENCES licensing.devices(id),
    client_type         VARCHAR(10) NOT NULL, -- MT4, MT5
    terminal_build      VARCHAR(50),
    ea_version          VARCHAR(50),
    broker_name         VARCHAR(255),
    broker_server       VARCHAR(255),
    mt_account_login    VARCHAR(50),
    installation_id     VARCHAR(100),
    fingerprint_version VARCHAR(20) NOT NULL DEFAULT 'hwfp-v1',
    fingerprint_hash    VARCHAR(64),
    fingerprint_components JSONB NOT NULL DEFAULT '{}',
    activation_ip       INET,
    activated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_device_activations_license ON licensing.device_activations(license_id);
CREATE INDEX idx_device_activations_device ON licensing.device_activations(device_id);

-- ============================================================
-- Device Credentials (long-lived but rotatable)
-- ============================================================
CREATE TABLE licensing.device_credentials (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id           UUID NOT NULL REFERENCES licensing.devices(id) ON DELETE CASCADE,
    credential_hash     VARCHAR(255) NOT NULL, -- SHA-256 of device secret
    token_family        UUID NOT NULL, -- for rotation tracking
    issued_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ,
    revoked_at          TIMESTAMPTZ,
    revocation_reason   TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_device_credentials_hash UNIQUE (credential_hash)
);

CREATE INDEX idx_device_credentials_device ON licensing.device_credentials(device_id);
CREATE INDEX idx_device_credentials_family ON licensing.device_credentials(token_family);

-- ============================================================
-- Refresh Tokens (rotating, per device)
-- ============================================================
CREATE TABLE licensing.refresh_tokens (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id           UUID NOT NULL REFERENCES licensing.devices(id) ON DELETE CASCADE,
    token_hash          VARCHAR(255) NOT NULL, -- SHA-256 of refresh token
    token_family        UUID NOT NULL,
    issued_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ NOT NULL,
    used_at             TIMESTAMPTZ, -- set when rotated
    revoked_at          TIMESTAMPTZ,
    revoked_reason      TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_refresh_tokens_hash UNIQUE (token_hash)
);

CREATE INDEX idx_refresh_tokens_device ON licensing.refresh_tokens(device_id);
CREATE INDEX idx_refresh_tokens_family ON licensing.refresh_tokens(token_family);

-- ============================================================
-- Session Leases (TTL-based, one active per license)
-- ============================================================
CREATE TABLE licensing.session_leases (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    license_id          UUID NOT NULL REFERENCES licensing.licenses(id),
    device_id           UUID NOT NULL REFERENCES licensing.devices(id),
    session_id          UUID NOT NULL DEFAULT gen_random_uuid(),
    status              VARCHAR(20) NOT NULL DEFAULT 'ACTIVE', -- ACTIVE, DEGRADED, EXPIRED, REVOKED
    last_heartbeat_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    lease_expires_at    TIMESTAMPTZ NOT NULL,
    source_ip           INET,
    metadata            JSONB NOT NULL DEFAULT '{}',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at          TIMESTAMPTZ,
    revocation_reason   TEXT
);

CREATE INDEX idx_session_leases_license ON licensing.session_leases(license_id);
CREATE INDEX idx_session_leases_status ON licensing.session_leases(status) WHERE status = 'ACTIVE';
CREATE INDEX idx_session_leases_device ON licensing.session_leases(device_id);

-- ============================================================
-- Request Nonces (replay protection, bounded TTL)
-- ============================================================
CREATE TABLE licensing.request_nonces (
    nonce               VARCHAR(100) PRIMARY KEY,
    device_id           UUID REFERENCES licensing.devices(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_request_nonces_expires ON licensing.request_nonces(expires_at);

-- ============================================================
-- Add columns to existing devices table
-- ============================================================
ALTER TABLE licensing.devices
    ADD COLUMN IF NOT EXISTS bound_license_id UUID REFERENCES licensing.licenses(id),
    ADD COLUMN IF NOT EXISTS installation_id VARCHAR(100),
    ADD COLUMN IF NOT EXISTS fingerprint_version VARCHAR(20) DEFAULT 'hwfp-v1',
    ADD COLUMN IF NOT EXISTS fingerprint_hash VARCHAR(64),
    ADD COLUMN IF NOT EXISTS fingerprint_components JSONB DEFAULT '{}',
    ADD COLUMN IF NOT EXISTS device_credential_hash VARCHAR(255),
    ADD COLUMN IF NOT EXISTS last_activation_at TIMESTAMPTZ;

-- ============================================================
-- Add license_key column to licenses for activation lookup
-- ============================================================
ALTER TABLE licensing.licenses
    ADD COLUMN IF NOT EXISTS license_key VARCHAR(100) UNIQUE;

-- ============================================================
-- Configuration defaults (stored in a config table)
-- ============================================================
CREATE TABLE IF NOT EXISTS licensing.policy_config (
    key         VARCHAR(100) PRIMARY KEY,
    value       TEXT NOT NULL,
    description TEXT,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO licensing.policy_config (key, value, description) VALUES
    ('access_token_ttl_seconds', '600', 'Access token TTL (10 min)'),
    ('refresh_token_ttl_seconds', '2592000', 'Refresh token TTL (30 days)'),
    ('session_lease_seconds', '45', 'Session lease TTL (45 sec)'),
    ('heartbeat_seconds', '10', 'Expected heartbeat interval'),
    ('clock_skew_seconds', '30', 'Max acceptable clock skew'),
    ('nonce_ttl_seconds', '120', 'Nonce storage TTL'),
    ('max_devices_default', '1', 'Default max devices per license'),
    ('max_sessions_default', '1', 'Default max active sessions'),
    ('device_match_threshold', '75', 'Min weighted match score for device rebind'),
    ('rebind_cooldown_days', '7', 'Cooldown before rebind allowed')
ON CONFLICT (key) DO NOTHING;

COMMENT ON TABLE licensing.device_activations IS 'Records each device activation with hardware fingerprint proof';
COMMENT ON TABLE licensing.device_credentials IS 'Long-lived but rotatable device credentials (HMAC secrets)';
COMMENT ON TABLE licensing.refresh_tokens IS 'Rotating refresh tokens with family-based reuse detection';
COMMENT ON TABLE licensing.session_leases IS 'TTL-based session leases — one active per license';
COMMENT ON TABLE licensing.request_nonces IS 'Replay protection — bounded TTL nonce storage';
