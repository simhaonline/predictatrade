-- 122: iam.trusted_devices — "remember this device" MFA bypass (30d)
--
-- When a user completes MFA with "remember this device" ticked, the
-- backend issues a random opaque token in an HttpOnly cookie
-- (pat_trusted_device) and stores only its sha256 hash here. Future
-- /auth/login calls presenting a valid (unrevoked, unexpired) cookie
-- hash skip the TOTP challenge for that browser — the password remains
-- mandatory and the token rotates on every use (presented row is
-- revoked, a replacement is issued).
--
-- Security notes:
--   * cookie is HttpOnly + Secure (production) + SameSite=Lax,
--     path-scoped to /api/v1/auth
--   * reuse detection: a consumed hash can never validate again
--   * rows are per-user; logging out elsewhere does not clear them
--     (device-level trust, not session-level)
CREATE TABLE IF NOT EXISTS iam.trusted_devices (
    id uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id uuid NOT NULL REFERENCES iam.users(id),
    token_hash text NOT NULL UNIQUE,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL,
    revoked_at timestamptz
);

CREATE INDEX IF NOT EXISTS idx_trusted_devices_user
    ON iam.trusted_devices(user_id);

-- Housekeeping: purge dead rows older than 90 days (revoked or expired).
-- Run ad hoc or via the maintenance job; cheap because the table is tiny.