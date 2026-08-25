-- 073_device_activation_unique.sql
-- Add unique constraint on device_activations (license_id, mt_account_login)
-- to prevent duplicate activations and enforce one-license-per-MT-account binding.
-- Also add agent connection logging table.

-- Unique constraint: one MT account per license (prevents duplicate activations)
CREATE UNIQUE INDEX IF NOT EXISTS idx_device_act_license_mt_unique
ON licensing.device_activations (license_id, mt_account_login)
WHERE mt_account_login IS NOT NULL AND mt_account_login != '';

-- Add updated_at for tracking last activation update
ALTER TABLE licensing.device_activations
  ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ DEFAULT now();

-- Agent connection log table — records agent connect/disconnect events for uptime tracking
CREATE TABLE IF NOT EXISTS audit.agent_connections (
    id           UUID NOT NULL DEFAULT gen_random_uuid(),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    agent_id     TEXT NOT NULL,
    event_type   TEXT NOT NULL,
    license_key  TEXT,
    mt_account   TEXT,
    broker_name  TEXT,
    ip_address   INET,
    user_agent   TEXT,
    metadata     JSONB DEFAULT '{}'::jsonb,
    PRIMARY KEY (id, created_at)
);

SELECT create_hypertable('audit.agent_connections', 'created_at',
    if_not_exists => TRUE, chunk_time_interval => INTERVAL '7 days');

CREATE INDEX IF NOT EXISTS idx_agent_conn_agent ON audit.agent_connections(agent_id);
CREATE INDEX IF NOT EXISTS idx_agent_conn_event ON audit.agent_connections(event_type);
CREATE INDEX IF NOT EXISTS idx_agent_conn_time ON audit.agent_connections(created_at DESC);

COMMENT ON TABLE audit.agent_connections IS 'Agent connection lifecycle events for uptime tracking and audit';
