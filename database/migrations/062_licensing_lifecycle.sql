-- LEGACY DUPLICATE PREFIX 062: shares its numeric prefix with another migration file. Tolerated legacy collision (see database/migrations/MIGRATION_ORDER.md); DO NOT rename (rename risks re-applying applied schema). The CI guard scripts/check_migrations.sh blocks NEW duplicate prefixes only.
-- 062_licensing_lifecycle.sql
-- License lifecycle + device security-action support (vertical slice).
-- Adds columns required for:
--   * license create/suspend/revoke/renew/reset/force-logout
--   * device reset / force-upgrade / disable-signal
-- Idempotent (IF NOT EXISTS); safe to re-run.

-- === licensing.licenses ===
ALTER TABLE licensing.licenses
  ADD COLUMN IF NOT EXISTS suspended_at TIMESTAMPTZ;

COMMENT ON COLUMN licensing.licenses.suspended_at IS 'Timestamp of last admin-initiated suspension';

-- === licensing.devices ===
ALTER TABLE licensing.devices
  ADD COLUMN IF NOT EXISTS force_upgrade_pending BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS signal_enabled       BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS last_reset_at        TIMESTAMPTZ;

COMMENT ON COLUMN licensing.devices.force_upgrade_pending IS 'When TRUE the Windows Agent must upgrade before next entitlement lease';
COMMENT ON COLUMN licensing.devices.signal_enabled IS 'When FALSE the agent must stop delivering signals for this device';
COMMENT ON COLUMN licensing.devices.last_reset_at IS 'Timestamp of last admin-initiated device reset';

-- === licensing.device_activations (soft deactivate support) ===
ALTER TABLE licensing.device_activations
  ADD COLUMN IF NOT EXISTS deactivated_at TIMESTAMPTZ;

COMMENT ON COLUMN licensing.device_activations.deactivated_at IS 'Soft-deactivate timestamp; NULL = active';

CREATE INDEX IF NOT EXISTS idx_device_activations_license_active
  ON licensing.device_activations(license_id) WHERE deactivated_at IS NULL;
