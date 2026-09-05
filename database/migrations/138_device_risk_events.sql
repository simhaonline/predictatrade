-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 138: device risk events (EA capital-guard telemetry)
--
-- v1.28 Client EAs emit CAPITAL_PROTECTION|{"event_type":"FLOATING_DD_BREAKER",
-- ...} through POST /ingest/agent (role=exec) when the terminal-local
-- floating-drawdown breaker fires. The engine logged + de-duplicated these
-- messages in memory (agent_provider.go:1478) but never persisted them, so
-- breakers were invisible to admins after the log rolled.
--
-- This table is the durable audit trail: one row per EA-emitted risk event
-- (FLOATING_DD_BREAKER, SOFT_HALT, RECOVER, ...). The realtime engine writes
-- it on ingest; the control plane reads it for the admin device view and ntfy
-- alerting. Write path is engine-side (same DB pool as tick persistence) so
-- the telemetry survives even if the control plane is down.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS licensing.device_risk_events (
    id              bigserial PRIMARY KEY,
    device_id       uuid,                       -- licensing.devices.id (null-safe: engine inserts even if the device row vanished)
    event_type      character varying(50) NOT NULL,
                    -- FLOATING_DD_BREAKER | SOFT_HALT | RECOVER | HARD_HALT | ...
    details         jsonb NOT NULL DEFAULT '{}'::jsonb,
                    -- event payload as sent by the EA (floating_loss, floating_loss_pct, action, ...)
    ingested_at     timestamptz NOT NULL DEFAULT now(),
    alerted_at      timestamptz                 -- set by the alert path (de-dupe: alert once per event row)
);

CREATE INDEX IF NOT EXISTS idx_device_risk_events_device_time
    ON licensing.device_risk_events (device_id, ingested_at DESC);

CREATE INDEX IF NOT EXISTS idx_device_risk_events_type_recent
    ON licensing.device_risk_events (event_type, ingested_at DESC)
    WHERE event_type IN ('FLOATING_DD_BREAKER', 'HARD_HALT');

COMMENT ON TABLE licensing.device_risk_events IS
    'EA capital-guard telemetry (v1.28): one row per CAPITAL_PROTECTION risk event reported by a Client EA via /ingest/agent. Durable audit trail for the floating-DD breaker, soft/halt halts and recoveries; surfaced in the admin device view and ntfy ops alerts.';