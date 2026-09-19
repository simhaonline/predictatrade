-- Migration 152: client telemetry + EA versioning (2026-09-18)
-- Operator directive: (1) unified EA versioning across the fleet, (2) live
-- client-connectivity telemetry with per-device issue visibility.
--
-- Adds to licensing.edge_device_state:
--   ea_version        — the EA build reported in telemetry (INIT/heartbeat),
--                       surfaced in the admin MT-clients page so stale
--                       binaries are visible at a glance.
--   last_error        — the most recent client-side error the EA reported
--                        (transport, auth, sizing, watchdog) — the admin
--                        portal shows "what is happening" per client.
--   last_error_at     — when that error was reported.
--   last_error_code   — machine-readable class (HTTP/auth/slippage/watchdog).
-- Adds a heartbeat error surface to licensing.edge_signal_queue-free zone:
--   system.client_errors — append-only error log from EA telemetry.

ALTER TABLE licensing.edge_device_state
  ADD COLUMN IF NOT EXISTS ea_version varchar(20) NOT NULL DEFAULT '',
  ADD COLUMN last_error varchar(500),
  ADD COLUMN last_error_at timestamptz,
  ADD COLUMN last_error_code varchar(40);

CREATE INDEX IF NOT EXISTS idx_edge_device_state_last_error_at
  ON licensing.edge_device_state (last_error_at DESC NULLS LAST);

-- Client error event log (append-only — history for the admin portal).
CREATE TABLE IF NOT EXISTS licensing.client_error_events (
  id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
  device_id    uuid NOT NULL,
  error_code   varchar(40)  NOT NULL,   -- HTTP_429 | AUTH | SIZING | WATCHDOG | TRANSPORT | ...
  message      varchar(500) NOT NULL,
  detail       jsonb,
  reported_at  timestamptz  NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_client_error_events_device
  ON licensing.client_error_events (device_id, reported_at DESC);
CREATE INDEX IF NOT EXISTS idx_client_error_events_recent
  ON licensing.client_error_events (reported_at DESC);

INSERT INTO audit.migration_history (filename, executed_by, notes)
VALUES ('152_client_telemetry.sql', 'pat_migration', 'client telemetry + EA version columns')
ON CONFLICT (filename) DO NOTHING;