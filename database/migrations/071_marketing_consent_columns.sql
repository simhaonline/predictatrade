-- LEGACY DUPLICATE PREFIX 071: shares its numeric prefix with another migration file. Tolerated legacy collision (see database/migrations/MIGRATION_ORDER.md); DO NOT rename (rename risks re-applying applied schema). The CI guard scripts/check_migrations.sh blocks NEW duplicate prefixes only.
-- 071_marketing_consent_columns.sql
-- Add marketing opt-in columns to iam.users for consent tracking
-- Forward-only migration — adds nullable columns with safe defaults

ALTER TABLE iam.users
  ADD COLUMN IF NOT EXISTS marketing_email_optin BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS marketing_sms_optin BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS marketing_phone_optin BOOLEAN NOT NULL DEFAULT FALSE;

-- Add consent version column for audit trail
ALTER TABLE iam.users
  ADD COLUMN IF NOT EXISTS consent_version TEXT,
  ADD COLUMN IF NOT EXISTS consent_timestamp TIMESTAMPTZ;

COMMENT ON COLUMN iam.users.marketing_email_optin IS 'User opt-in for email marketing communications (GDPR/PDPL consent)';
COMMENT ON COLUMN iam.users.marketing_sms_optin IS 'User opt-in for SMS marketing communications (GDPR/PDPL consent)';
COMMENT ON COLUMN iam.users.marketing_phone_optin IS 'User opt-in for phone call marketing communications (GDPR/PDPL consent)';
COMMENT ON COLUMN iam.users.consent_version IS 'Version of the consent text shown during registration';
COMMENT ON COLUMN iam.users.consent_timestamp IS 'Timestamp when consent was recorded during registration';
