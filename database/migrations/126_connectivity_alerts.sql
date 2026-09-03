-- Connectivity watchdog (2026-09-04).
-- Server-side guarantee: if MT clients stop polling (or the realtime engine
-- goes down), the user AND the admin are notified — clients must never
-- silently lose trade signals.
--
-- system.connectivity_alerts: one row per (alert_key) with OPEN/CLOSED
-- lifecycle and dedup. The watchdog UPSERTs with last_seen_at tracking so a
-- persisting condition does not spam; re-notification happens at most once
-- per NOTIFY_COOLDOWN (enforced in code).

CREATE SCHEMA IF NOT EXISTS system;

CREATE TABLE IF NOT EXISTS system.connectivity_alerts (
    id             uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    alert_key      varchar(120) NOT NULL UNIQUE,
    severity       varchar(20)  NOT NULL DEFAULT 'WARNING', -- INFO|WARNING|CRITICAL
    scope          varchar(30)  NOT NULL,                   -- DEVICE|ENGINE|API
    device_id      uuid NULL REFERENCES licensing.devices(id),
    message        text         NOT NULL,
    status         varchar(20)  NOT NULL DEFAULT 'OPEN',    -- OPEN|RESOLVED
    occurrences    integer      NOT NULL DEFAULT 1,
    first_seen_at  timestamptz  NOT NULL DEFAULT now(),
    last_seen_at   timestamptz  NOT NULL DEFAULT now(),
    resolved_at    timestamptz  NULL,
    notified_at    timestamptz  NULL
);

CREATE INDEX IF NOT EXISTS idx_conn_alerts_open
    ON system.connectivity_alerts (status, last_seen_at DESC);
CREATE INDEX IF NOT EXISTS idx_conn_alerts_device
    ON system.connectivity_alerts (device_id) WHERE device_id IS NOT NULL;