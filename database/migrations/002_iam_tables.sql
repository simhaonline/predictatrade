-- Predict-A-Trade v1.0.0 — Migration 002
-- Core IAM Tables (SOW Sections 34, 35, 36, 61)
-- All monetary fields use DECIMAL. All timestamps use TIMESTAMPTZ (UTC).

-- ============================================================
-- Organizations / Tenants (SOW Section 36)
-- ============================================================
CREATE TABLE iam.organizations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    slug            VARCHAR(100) NOT NULL UNIQUE,
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    settings        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Users (SOW Sections 34, 61)
-- ============================================================
CREATE TABLE iam.users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id       UUID REFERENCES iam.organizations(id),
    email           VARCHAR(320) NOT NULL,
    email_verified  BOOLEAN NOT NULL DEFAULT FALSE,
    username        VARCHAR(100),
    password_hash   VARCHAR(255),
    full_name       VARCHAR(255),
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    -- ACTIVE, SUSPENDED, LOCKED, DELETED
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    locked_until    TIMESTAMPTZ,
    last_login_at   TIMESTAMPTZ,
    last_login_ip   INET,
    preferences     JSONB NOT NULL DEFAULT '{}',
    legal_acceptances JSONB NOT NULL DEFAULT '[]',
    -- Array of {document_type, version, accepted_at, source_ip}
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(email)
);

CREATE INDEX idx_users_tenant ON iam.users(tenant_id);
CREATE INDEX idx_users_status ON iam.users(status);
CREATE INDEX idx_users_email ON iam.users(email);

-- ============================================================
-- Memberships (user <-> organization)
-- ============================================================
CREATE TABLE iam.memberships (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES iam.organizations(id) ON DELETE CASCADE,
    role_id         UUID,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(user_id, organization_id)
);

-- ============================================================
-- Roles & Permissions (SOW Section 35)
-- ============================================================
CREATE TABLE iam.roles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(50) NOT NULL UNIQUE,
    -- SUPER_ADMIN, ADMIN, RISK_MANAGER, TRADING_OPERATOR, SUPPORT, ANALYST, AUDITOR, USER
    description     TEXT,
    is_system       BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE iam.permissions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100) NOT NULL UNIQUE,
    -- e.g. license:create, risk:modify, signal:read, device:revoke
    description     TEXT,
    resource        VARCHAR(50) NOT NULL,
    action          VARCHAR(50) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE iam.role_permissions (
    role_id         UUID NOT NULL REFERENCES iam.roles(id) ON DELETE CASCADE,
    permission_id   UUID NOT NULL REFERENCES iam.permissions(id) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_id)
);

-- ============================================================
-- Sessions (SOW Section 34, 72)
-- ============================================================
CREATE TABLE iam.sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    token_hash      VARCHAR(255) NOT NULL,
    refresh_token_hash VARCHAR(255),
    ip_address      INET,
    user_agent      TEXT,
    device_id       UUID,
    expires_at      TIMESTAMPTZ NOT NULL,
    refresh_expires_at TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_user ON iam.sessions(user_id);
CREATE INDEX idx_sessions_token ON iam.sessions(token_hash);
CREATE INDEX idx_sessions_expires ON iam.sessions(expires_at) WHERE revoked_at IS NULL;

-- ============================================================
-- MFA Methods (SOW Section 34)
-- ============================================================
CREATE TABLE iam.mfa_methods (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    type            VARCHAR(20) NOT NULL,
    -- TOTP, WEBAUTHN, PASSKEY
    secret_hash     VARCHAR(255),
    -- For TOTP: encrypted secret. For WebAuthn: credential_id + public_key
    credential_data JSONB,
    label           VARCHAR(100),
    is_primary      BOOLEAN NOT NULL DEFAULT FALSE,
    enabled         BOOLEAN NOT NULL DEFAULT TRUE,
    last_used_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_mfa_user ON iam.mfa_methods(user_id);

-- ============================================================
-- Recovery Codes (SOW Section 34)
-- ============================================================
CREATE TABLE iam.recovery_codes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    code_hash       VARCHAR(255) NOT NULL,
    used_at         TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_recovery_user ON iam.recovery_codes(user_id);

-- ============================================================
-- Login Events (SOW Section 34)
-- ============================================================
CREATE TABLE iam.login_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES iam.users(id),
    event_type      VARCHAR(50) NOT NULL,
    -- LOGIN_SUCCESS, LOGIN_FAILED, MFA_CHALLENGE, MFA_SUCCESS, MFA_FAILED, LOCKOUT, RECOVERY_USED
    ip_address      INET,
    user_agent      TEXT,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_login_events_user ON iam.login_events(user_id);
CREATE INDEX idx_login_events_type ON iam.login_events(event_type);
CREATE INDEX idx_login_events_created ON iam.login_events(created_at);

-- ============================================================
-- API Credentials (SOW Section 61)
-- ============================================================
CREATE TABLE iam.api_credentials (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id) ON DELETE CASCADE,
    name            VARCHAR(100) NOT NULL,
    key_hash        VARCHAR(255) NOT NULL,
    key_prefix      VARCHAR(20) NOT NULL,
    scopes          JSONB NOT NULL DEFAULT '[]',
    last_used_at    TIMESTAMPTZ,
    expires_at      TIMESTAMPTZ,
    revoked_at      TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_creds_user ON iam.api_credentials(user_id);

-- ============================================================
-- Seed default roles (SOW Section 35)
-- ============================================================
INSERT INTO iam.roles (name, description, is_system) VALUES
    ('SUPER_ADMIN', 'Full system access', TRUE),
    ('ADMIN', 'Administrative access', TRUE),
    ('RISK_MANAGER', 'Risk control management', TRUE),
    ('TRADING_OPERATOR', 'Trading operations', TRUE),
    ('SUPPORT', 'Customer support', TRUE),
    ('ANALYST', 'Read-only analytics', TRUE),
    ('AUDITOR', 'Audit log access', TRUE),
    ('USER', 'Standard user/subscriber', TRUE)
ON CONFLICT (name) DO NOTHING;

-- Seed default permissions
INSERT INTO iam.permissions (name, description, resource, action) VALUES
    ('license:create', 'Create licenses', 'license', 'create'),
    ('license:revoke', 'Revoke licenses', 'license', 'revoke'),
    ('license:read', 'Read licenses', 'license', 'read'),
    ('risk:modify', 'Modify risk settings', 'risk', 'modify'),
    ('risk:read', 'Read risk settings', 'risk', 'read'),
    ('execution:stop', 'Stop execution', 'execution', 'stop'),
    ('strategy:activate', 'Activate strategies', 'strategy', 'activate'),
    ('strategy:read', 'Read strategy config', 'strategy', 'read'),
    ('user:suspend', 'Suspend users', 'user', 'suspend'),
    ('user:read', 'Read user profiles', 'user', 'read'),
    ('audit:read', 'Read audit logs', 'audit', 'read'),
    ('signal:read', 'Read signals', 'signal', 'read'),
    ('device:revoke', 'Revoke devices', 'device', 'revoke'),
    ('device:read', 'Read device info', 'device', 'read'),
    ('commission:read', 'Read commission ledger', 'commission', 'read'),
    ('commission:hold', 'Hold commission', 'commission', 'hold'),
    ('commission:reverse', 'Reverse commission', 'commission', 'reverse'),
    ('payout:approve', 'Approve payouts', 'payout', 'approve'),
    ('payout:reject', 'Reject payouts', 'payout', 'reject'),
    ('plan:manage', 'Manage plans and pricing', 'plan', 'manage'),
    ('referral:read', 'Read referral network', 'referral', 'read'),
    ('referral:change_sponsor', 'Change sponsor', 'referral', 'change_sponsor'),
    ('admin:access', 'Access admin portal', 'admin', 'access')
ON CONFLICT (name) DO NOTHING;

-- Assign permissions to roles
-- SUPER_ADMIN gets all
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM iam.roles r, iam.permissions p WHERE r.name = 'SUPER_ADMIN'
ON CONFLICT DO NOTHING;

-- ADMIN gets most except system-level
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM iam.roles r, iam.permissions p 
WHERE r.name = 'ADMIN' AND p.name NOT LIKE 'admin:%'
ON CONFLICT DO NOTHING;

-- RISK_MANAGER
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM iam.roles r, iam.permissions p 
WHERE r.name = 'RISK_MANAGER' AND p.name IN ('risk:modify', 'risk:read', 'execution:stop', 'strategy:read', 'audit:read', 'signal:read')
ON CONFLICT DO NOTHING;

-- TRADING_OPERATOR
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM iam.roles r, iam.permissions p 
WHERE r.name = 'TRADING_OPERATOR' AND p.name IN ('execution:stop', 'signal:read', 'strategy:read', 'risk:read')
ON CONFLICT DO NOTHING;

-- SUPPORT
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM iam.roles r, iam.permissions p 
WHERE r.name = 'SUPPORT' AND p.name IN ('user:read', 'license:read', 'device:read', 'signal:read')
ON CONFLICT DO NOTHING;

-- ANALYST
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM iam.roles r, iam.permissions p 
WHERE r.name = 'ANALYST' AND p.name IN ('signal:read', 'strategy:read', 'risk:read', 'audit:read')
ON CONFLICT DO NOTHING;

-- AUDITOR
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM iam.roles r, iam.permissions p 
WHERE r.name = 'AUDITOR' AND p.name IN ('audit:read', 'user:read', 'commission:read', 'referral:read')
ON CONFLICT DO NOTHING;

-- USER
INSERT INTO iam.role_permissions (role_id, permission_id)
SELECT r.id, p.id FROM iam.roles r, iam.permissions p 
WHERE r.name = 'USER' AND p.name IN ('signal:read', 'license:read', 'device:read')
ON CONFLICT DO NOTHING;
