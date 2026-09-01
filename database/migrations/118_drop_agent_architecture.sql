-- Migration 118: Option B teardown — remove the Windows-agent database layer.
--
-- v1.19.0 eliminates the Windows-agent transport entirely: customer MT4/MT5 EAs
-- talk to the platform directly over HTTPS (POST /ingest/agent with a device
-- JWT for market data; POST /api/v1/devices/edge-poll|edge-ack|edge-heartbeat
-- with a device HMAC for signal delivery). The persistent-connection bookkeeping
-- that only the WS hub wrote is therefore dead:
--
--   DROPPED:
--     audit.agent_connections — WS connection lifecycle events
--       (connected/disconnected) written by the deleted gateway.AgentHub. The
--       liveness source is now licensing.edge_device_state (poll/ack/heartbeat
--       timestamps, migration 117).
--
--   KEPT (writer changed, schema unchanged):
--     trading.agent_user_bindings — still the agent-id → user/license/device
--     tenancy join used by reports, trade attribution and license validation.
--     Its writer changed: the engine now populates it from the device JWT
--     subject on HTTP ingest instead of the WS handshake.
--
--   NOT TOUCHED (still live): licensing.edge_signal_queue,
--   licensing.edge_device_state (Option B delivery + liveness), and
--   licensing.devices (device registry shared by both eras).
--
-- Rollback: agent_connections rows are historical WS audit data; recreating the
-- table without restoring data is acceptable (see 073 for the original DDL).
-- There is no re-creation path back to the WS agent architecture.

-- 1) Drop the WS-era connection-audit hypertable (and its indexes cascade).
DROP TABLE IF EXISTS audit.agent_connections CASCADE;

-- 2) Drop the agent_id-based FK/index plumbing that assumed a live WS session
--    per agent. agent_user_bindings itself is retained (see header).
DROP INDEX IF EXISTS trading.idx_agent_user_bindings_agent;

-- 3) Defense-in-depth comment trail so future audits see the intent.
COMMENT ON TABLE audit.agent_connections IS 'REMOVED in migration 118 (v1.19.0 Option B) — replaced by licensing.edge_device_state';