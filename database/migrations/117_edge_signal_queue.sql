-- ═══════════════════════════════════════════════════════════════════════════
-- 117: EA-direct edge polling (Option B — no Windows agent required)
--
-- Customer MT4/MT5 EAs poll POST /api/v1/devices/edge-poll (HMAC-signed with
-- their device secret) instead of relying on the Windows agent + local pipes.
-- The realtime engine enqueues EXECUTABLE signals here for devices that are
-- NOT connected via the agent WebSocket hub; the EA fetches, executes, and ACKs.
-- ═══════════════════════════════════════════════════════════════════════════

-- Durable per-device delivery queue.
CREATE TABLE IF NOT EXISTS licensing.edge_signal_queue (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       uuid NOT NULL,              -- licensing.devices.id
    signal_id       character varying(100) NOT NULL,
    payload         jsonb NOT NULL,             -- full signal envelope for the EA
    status          character varying(20) NOT NULL DEFAULT 'PENDING',
                    -- PENDING | IN_FLIGHT | ACKED | EXPIRED | FAILED
    attempts        integer NOT NULL DEFAULT 0,
    last_error      text,
    created_at      timestamptz NOT NULL DEFAULT now(),
    in_flight_at    timestamptz,
    acked_at        timestamptz,
    ack_result      jsonb
);

CREATE INDEX IF NOT EXISTS idx_edge_signal_queue_device_pending
    ON licensing.edge_signal_queue (device_id, created_at)
    WHERE status IN ('PENDING', 'IN_FLIGHT');

CREATE INDEX IF NOT EXISTS idx_edge_signal_queue_device_signal
    ON licensing.edge_signal_queue (device_id, signal_id);

-- Per-device delivery cursor / liveness for the EA-direct transport.
CREATE TABLE IF NOT EXISTS licensing.edge_device_state (
    device_id           uuid PRIMARY KEY,
    transport           character varying(20) NOT NULL DEFAULT 'EA_DIRECT',
                    -- EA_DIRECT | AGENT_WS (whichever last delivered)
    last_poll_at        timestamptz,
    last_ack_at         timestamptz,
    last_heartbeat_at   timestamptz,
    polls_total         bigint NOT NULL DEFAULT 0,
    signals_delivered   bigint NOT NULL DEFAULT 0,
    signals_acked       bigint NOT NULL DEFAULT 0,
    updated_at          timestamptz NOT NULL DEFAULT now()
);

COMMENT ON TABLE licensing.edge_signal_queue IS
    'EA-direct signal delivery queue: EXECUTABLE signals waiting for a customer EA poll (no Windows agent).';
COMMENT ON TABLE licensing.edge_device_state IS
    'Per-device liveness/delivery counters for EA-direct (edge-poll) transport.';