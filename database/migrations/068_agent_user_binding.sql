-- 068_agent_user_binding.sql
-- Links agent WebSocket IDs (as used in trading.trade_results.account_id = 'agent:<id>')
-- to the owning user via the license key presented at MASTER_INIT.
-- Written by the realtime engine on every successful license validation.
-- Enables per-user trading reports (CSV/XLSX/PDF) from the control plane.

BEGIN;

CREATE TABLE IF NOT EXISTS trading.agent_user_bindings (
  agent_id     varchar(100) PRIMARY KEY,
  license_key  varchar(255) NOT NULL,
  user_id      uuid NOT NULL REFERENCES iam.users(id),
  bound_at     timestamptz NOT NULL DEFAULT now(),
  last_seen_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_agent_user_bindings_user ON trading.agent_user_bindings(user_id);

CREATE INDEX IF NOT EXISTS idx_trade_results_account ON trading.trade_results(account_id);

COMMIT;
