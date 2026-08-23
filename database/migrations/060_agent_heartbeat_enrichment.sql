-- 060_agent_heartbeat_enrichment.sql
-- Extends device heartbeat reporting with genuine agent/OS/terminal/XAUUSD
-- operational metadata required by the client dashboard (go-prompt.md §3-8, §11).
-- Additive only; safe to apply on existing databases.

ALTER TABLE licensing.devices
  ADD COLUMN IF NOT EXISTS os_name            VARCHAR(32),
  ADD COLUMN IF NOT EXISTS architecture       VARCHAR(32),
  ADD COLUMN IF NOT EXISTS agent_uptime_seconds BIGINT,
  ADD COLUMN IF NOT EXISTS service_status     VARCHAR(20) NOT NULL DEFAULT 'unknown',
  ADD COLUMN IF NOT EXISTS health_status      VARCHAR(20) NOT NULL DEFAULT 'unknown';

ALTER TABLE licensing.device_activations
  ADD COLUMN IF NOT EXISTS terminal_connected BOOLEAN NOT NULL DEFAULT true,
  ADD COLUMN IF NOT EXISTS terminal_version   VARCHAR(50),
  ADD COLUMN IF NOT EXISTS xauusd_available   BOOLEAN NOT NULL DEFAULT false,
  ADD COLUMN IF NOT EXISTS xauusd_bid         NUMERIC(20,5),
  ADD COLUMN IF NOT EXISTS xauusd_ask         NUMERIC(20,5),
  ADD COLUMN IF NOT EXISTS xauusd_spread      NUMERIC(20,5),
  ADD COLUMN IF NOT EXISTS xauusd_digits      INTEGER,
  ADD COLUMN IF NOT EXISTS xauusd_last_tick_time TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_devices_health ON licensing.devices(service_status, health_status);
