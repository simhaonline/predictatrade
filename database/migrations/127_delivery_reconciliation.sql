-- Delivery-reconciliation watchdog (2026-09-04, permanent delivery guardrail).
--
-- After the 2026-09-03 incident (old MT5 builds ACKed every signal
-- PROCESSED with type:"" but never executed — a silent total delivery drop
-- that client ACKs reported as SUCCESS), ACKs alone are NOT proof of
-- delivery. The watchdog now tracks dispatch + end-to-end outcome per
-- device so any future silent drop pages within minutes.
--
-- system.delivery_reconciliation: one row per (device_id) tracking the
-- device's dispatch contract (payload->>'type' handling) and end-to-end
-- delivery health:
--   last_signal_ack_type   — 'SIGNAL' means the device dispatched correctly
--   empty_ack_count_24h    — PROCESSED ACKs with no type = likely silent drops
--   pending_over_5m        — enqueue backlog aging (engine or poll backlog)

CREATE SCHEMA IF NOT EXISTS system;

CREATE TABLE IF NOT EXISTS system.delivery_reconciliation (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id            uuid NOT NULL REFERENCES licensing.devices(id),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    last_signal_ack_at   timestamptz NULL,
    last_signal_ack_type varchar(40) NULL,
    empty_ack_count_24h  integer NOT NULL DEFAULT 0,
    pending_count        integer NOT NULL DEFAULT 0,
    pending_oldest_secs  integer NOT NULL DEFAULT 0,
    dropped_count_24h    integer NOT NULL DEFAULT 0,
    delivered_count_24h  integer NOT NULL DEFAULT 0,
    UNIQUE (device_id)
);

CREATE INDEX IF NOT EXISTS idx_deliv_recon_device
    ON system.delivery_reconciliation (device_id);