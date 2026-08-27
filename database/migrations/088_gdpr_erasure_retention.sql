-- 078: GDPR erasure / anonymization / retention support
--
-- Implements PII-1 (no GDPR erasure/retention workflow previously existed).
-- The GdprService (control/src/modules/compliance/gdpr.service.ts) executes the
-- SQL below at runtime; this migration only creates the auditable log table and
-- documents the canonical anonymization/retention statements.
--
-- === Canonical user PII anonymization (iam.users) ===
--   UPDATE iam.users
--     SET email        = 'gdpr-erased-' || substr(md5(id::text), 1, 16) || '@anonymized.local',
--         email_verified = false,
--         username     = NULL,
--         full_name    = NULL,
--         last_login_ip = NULL,
--         password_hash = 'ERASED',          -- eraseUser only
--         status       = 'DELETED',         -- eraseUser only
--         updated_at   = now()
--   WHERE id = :userId;
--
-- === Canonical client-event PII anonymization (audit.client_events) ===
--   UPDATE audit.client_events
--     SET client_ip = NULL, proxy_chain = NULL,
--         geo_country_code = NULL, geo_region = NULL, geo_city = NULL,
--         isp = NULL, asn = NULL, as_org = NULL,
--         user_agent = NULL, browser_name = NULL, browser_version = NULL,
--         os_name = NULL, os_version = NULL, device_type = NULL,
--         languages = NULL, client_hints = NULL
--   WHERE user_id = :userId;
--
-- === Canonical retention (anonymize PII older than N days) ===
--   UPDATE audit.client_events
--     SET client_ip = NULL, proxy_chain = NULL,
--         geo_country_code = NULL, geo_region = NULL, geo_city = NULL,
--         isp = NULL, asn = NULL, as_org = NULL,
--         user_agent = NULL, browser_name = NULL, browser_version = NULL,
--         os_name = NULL, os_version = NULL, device_type = NULL,
--         languages = NULL, client_hints = NULL
--   WHERE event_time < now() - (:days || ' days')::interval;
--
-- NOTE: TimescaleDB chunk retention (migration 064) may later DROP old chunks
-- entirely; retention here scrubs PII first as defense-in-depth.

CREATE SCHEMA IF NOT EXISTS compliance;

CREATE TABLE IF NOT EXISTS compliance.gdpr_operations (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    operation       VARCHAR(50) NOT NULL,   -- GDPR_USER_ERASURE | GDPR_USER_ANONYMIZED | GDPR_RETENTION_RUN
    target_user_id  UUID,
    actor_id        UUID,
    details         JSONB,
    rows_affected   INTEGER NOT NULL DEFAULT 0,
    executed_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_gdpr_ops_user ON compliance.gdpr_operations (target_user_id);
CREATE INDEX IF NOT EXISTS idx_gdpr_ops_time ON compliance.gdpr_operations (executed_at DESC);

GRANT INSERT, SELECT ON compliance.gdpr_operations TO pat_admin;
