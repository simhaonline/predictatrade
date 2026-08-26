-- 086_agent_device_id_binding.sql
-- Correlates the realtime engine's in-memory agent connection (agent_id) with the
-- control-plane device (licensing.devices.id). The Windows Agent now forwards its
-- control-plane device_id to the engine during MASTER_INIT / heartbeat so the engine
-- can publish authoritative live connection state (online/offline + MT4/MT5 link)
-- into licensing.devices / device_activations, which the Admin + User dashboards read.
-- Without this column, the engine could not map its live agents to a dashboard-visible
-- device row, so client dashboards never showed a live connection despite agents being
-- connected to the engine.

BEGIN;

ALTER TABLE trading.agent_user_bindings
  ADD COLUMN IF NOT EXISTS device_id varchar(100);

CREATE INDEX IF NOT EXISTS idx_agent_user_bindings_device
  ON trading.agent_user_bindings (device_id);

COMMIT;
