-- Predict-A-Trade v1.0.0 — Migration 007
-- Auth hardening: align mfa_methods columns with code, add unique constraints,
-- add unconditional index on refresh_token_hash, add unique constraint.
-- (SOW Section 34, 72 — Authentication/Session security hardening)
--
-- This migration renames columns in iam.mfa_methods to match the application code
-- and adds a unique constraint needed for ON CONFLICT upsert.
-- It also fixes the refresh_token_hash index to be unconditional (reuse detection
-- requires looking up revoked sessions too) and adds a unique constraint.

-- ============================================================
-- MFA Methods: rename columns to match application code
-- ============================================================

-- Rename type → method_type
ALTER TABLE iam.mfa_methods
    RENAME COLUMN type TO method_type;

-- Rename secret_hash → secret (TOTP secret stored for verification; not a password hash)
ALTER TABLE iam.mfa_methods
    RENAME COLUMN secret_hash TO secret;

-- Rename enabled → is_enabled
ALTER TABLE iam.mfa_methods
    RENAME COLUMN enabled TO is_enabled;

-- Add unique constraint for ON CONFLICT (user_id, method_type) upsert
ALTER TABLE iam.mfa_methods
    ADD CONSTRAINT uq_mfa_methods_user_method UNIQUE (user_id, method_type);

-- ============================================================
-- Sessions: fix refresh_token_hash index and add unique constraint
-- ============================================================

-- Drop the partial index that only covers revoked_at IS NULL.
-- Reuse detection requires looking up revoked sessions too, so the index
-- must be unconditional.
DROP INDEX IF EXISTS iam.idx_sessions_refresh_hash;

-- Create an unconditional index for fast refresh-token-hash lookups.
-- Used by /auth/refresh to find a session by its token hash (both active
-- and revoked sessions need to be found for reuse detection).
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_hash
    ON iam.sessions(refresh_token_hash);

-- Add unique constraint: a refresh token hash should belong to at most one session.
-- With 384-bit random tokens, collisions are practically impossible, but the
-- constraint prevents accidental duplicates and allows SELECT ... FOR UPDATE
-- to lock exactly one row.
ALTER TABLE iam.sessions
    ADD CONSTRAINT uq_sessions_refresh_token_hash UNIQUE (refresh_token_hash);

-- ============================================================
-- Composite index for family revocation during reuse detection
-- (already exists from migration 006 but ensure it's present)
-- ============================================================
CREATE INDEX IF NOT EXISTS idx_sessions_family
    ON iam.sessions(token_family)
    WHERE token_family IS NOT NULL;

COMMENT ON COLUMN iam.mfa_methods.method_type IS 'MFA method type: TOTP, WEBAUTHN, PASSKEY';
COMMENT ON COLUMN iam.mfa_methods.secret IS 'TOTP secret (raw) or WebAuthn credential data reference';
COMMENT ON COLUMN iam.mfa_methods.is_enabled IS 'Whether this MFA method is active for login';
