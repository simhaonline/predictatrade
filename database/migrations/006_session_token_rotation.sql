-- Predict-A-Trade v1.0.0 — Migration 006
-- Session token family for refresh-token rotation + reuse detection
-- (SOW Section 34, 72 — Session security hardening)
--
-- Adds token_family column to iam.sessions for:
--   - Grouping rotated refresh tokens into a family
--   - Detecting reuse of a previously-rotated token
--   - Revoking the entire family when reuse is detected
-- Also adds an index on refresh_token_hash for fast lookups.

ALTER TABLE iam.sessions
    ADD COLUMN IF NOT EXISTS token_family UUID;

-- Index for fast refresh-token-hash lookups during /auth/refresh
CREATE INDEX IF NOT EXISTS idx_sessions_refresh_hash
    ON iam.sessions(refresh_token_hash)
    WHERE revoked_at IS NULL;

-- Index for fast family lookups during reuse detection
CREATE INDEX IF NOT EXISTS idx_sessions_family
    ON iam.sessions(token_family)
    WHERE token_family IS NOT NULL;

COMMENT ON COLUMN iam.sessions.token_family IS
    'Groups a chain of rotated refresh tokens. If a previously-rotated token is reused, the entire family is revoked.';
