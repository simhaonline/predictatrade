-- 061_device_activation_account_columns.sql
-- Completes the trading-account + agent operational columns on
-- licensing.device_activations so the Windows Agent heartbeat can persist genuine
-- account state (go-prompt.md §6) and GET /licensing/devices can return it.
-- Also adds agent_started_at to licensing.devices (§3).
-- Idempotent (IF NOT EXISTS); safe to re-run. Matches columns already referenced
-- by control/src/modules/device-auth/device-auth.service.ts and licensing.service.ts.

ALTER TABLE licensing.device_activations
  ADD COLUMN IF NOT EXISTS account_balance       NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS account_equity        NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS account_profit        NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS account_currency      VARCHAR(3),
  ADD COLUMN IF NOT EXISTS open_positions        INTEGER,
  ADD COLUMN IF NOT EXISTS buy_positions         INTEGER,
  ADD COLUMN IF NOT EXISTS sell_positions        INTEGER,
  ADD COLUMN IF NOT EXISTS total_lots            NUMERIC(18,4),
  ADD COLUMN IF NOT EXISTS floating_pnl          NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS last_account_update   TIMESTAMPTZ,
  ADD COLUMN IF NOT EXISTS leverage              NUMERIC(10,2),
  ADD COLUMN IF NOT EXISTS margin                NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS free_margin           NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS margin_level          NUMERIC(10,2),
  ADD COLUMN IF NOT EXISTS account_type          VARCHAR(20),
  ADD COLUMN IF NOT EXISTS pending_orders_count  INTEGER;

ALTER TABLE licensing.devices
  ADD COLUMN IF NOT EXISTS agent_started_at TIMESTAMPTZ;

CREATE INDEX IF NOT EXISTS idx_device_activations_account
  ON licensing.device_activations(device_id, mt_account_login);
