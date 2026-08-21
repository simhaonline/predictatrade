-- Predict-A-Trade v1.0.0 — Migration 023
-- Guest Preview → Registration Gate funnel (UAE PDPL compliant)
--
-- Adds dedicated audit-grade tables for:
--   1. Consent records — immutable per-signup consent log (timestamp, version,
--      exact consent text, checkbox states, IP, user-agent) for PDPL compliance.
--   2. Registration challenges — pending passwordless email-OTP registrations
--      (code stored HASHED, 10-min expiry, max-attempts, cooldown, consent snapshot).
--   3. Marketing unsubscribes — persisted opt-outs honored immediately across all
--      marketing/email communications.
--
-- All timestamps TIMESTAMPTZ (UTC). All monetary fields would be DECIMAL (none here).
-- Forward-only, additive migration. No existing table is rewritten.

-- ============================================================
-- 1. Consent records (PDPL audit log)
-- ============================================================
CREATE TABLE IF NOT EXISTS iam.consent_records (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    -- The user the consent belongs to (nullable until OTP verification creates the account;
    -- pre-verification consents are logged against the email so the audit trail is complete).
    user_id         UUID REFERENCES iam.users(id) ON DELETE CASCADE,
    email           VARCHAR(320) NOT NULL,
    -- Which stage of the funnel this consent was captured at.
    stage           VARCHAR(50) NOT NULL DEFAULT 'REGISTRATION',
    -- The three distinct consent flags (never combined, never pre-ticked).
    terms_accepted          BOOLEAN NOT NULL,
    risk_acknowledged       BOOLEAN NOT NULL,
    marketing_opt_in        BOOLEAN NOT NULL,
    -- Exact consent text + version shown to the user (immutable audit copy).
    terms_text              TEXT NOT NULL,
    risk_text               TEXT NOT NULL,
    marketing_text          TEXT NOT NULL,
    consent_version         VARCHAR(20) NOT NULL,
    ip_address             INET,
    user_agent              TEXT,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_consent_records_user ON iam.consent_records(user_id);
CREATE INDEX IF NOT EXISTS idx_consent_records_email ON iam.consent_records(email);
CREATE INDEX IF NOT EXISTS idx_consent_records_created ON iam.consent_records(created_at);

-- ============================================================
-- 2. Registration challenges (passwordless email-OTP)
-- ============================================================
CREATE TABLE IF NOT EXISTS iam.registration_challenges (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(320) NOT NULL,
    -- SHA-256 hash of the 6-digit OTP. NEVER store plaintext.
    code_hash       VARCHAR(255) NOT NULL,
    -- Pending registration payload captured at submit, applied on successful verify.
    full_name       VARCHAR(255) NOT NULL,
    phone           VARCHAR(50),
    broker          VARCHAR(100),
    -- Frozen consent snapshot captured at submit (re-logged to consent_records on verify).
    consent_snapshot JSONB NOT NULL,
    attempts        INTEGER NOT NULL DEFAULT 0,
    max_attempts    INTEGER NOT NULL DEFAULT 5,
    expires_at      TIMESTAMPTZ NOT NULL,
    -- Cooldown for resend: a new challenge row supersedes older ones for the same email.
    consumed_at     TIMESTAMPTZ,
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_reg_challenges_email ON iam.registration_challenges(email);
CREATE INDEX IF NOT EXISTS idx_reg_challenges_email_active
    ON iam.registration_challenges(email, created_at DESC)
    WHERE consumed_at IS NULL;

-- ============================================================
-- 3. Marketing unsubscribes (persisted, honored immediately)
-- ============================================================
CREATE TABLE IF NOT EXISTS iam.marketing_unsubscribes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(320) NOT NULL UNIQUE,
    -- Scope of the opt-out. 'marketing' = promotional only; 'all' = all non-essential email.
    scope           VARCHAR(20) NOT NULL DEFAULT 'marketing',
    ip_address      INET,
    user_agent      TEXT,
    unsubscribed_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_marketing_unsub_email ON iam.marketing_unsubscribes(email);

COMMENT ON TABLE iam.consent_records IS
    'Immutable PDPL consent audit log. One row per signup with exact consent text, version, checkbox states, IP and user-agent.';
COMMENT ON TABLE iam.registration_challenges IS
    'Pending passwordless email-OTP registrations. OTP stored hashed with 10-min expiry and max-attempts limit.';
COMMENT ON TABLE iam.marketing_unsubscribes IS
    'Persisted marketing/email opt-outs. Honored immediately across all communications.';
