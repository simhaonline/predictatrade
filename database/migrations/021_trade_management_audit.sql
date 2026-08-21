-- Migration 021: Trade management audit — add SL modification tracking
-- ADDITIVE ONLY — no existing columns or data are removed or altered.
-- All new columns are nullable or have safe defaults.

-- ─── Add trade management fields to existing positions table ───
ALTER TABLE trading.positions
  ADD COLUMN IF NOT EXISTS initial_entry_price  numeric(18,8),
  ADD COLUMN IF NOT EXISTS initial_stop_loss   numeric(18,8),
  ADD COLUMN IF NOT EXISTS initial_risk_distance numeric(18,8),
  ADD COLUMN IF NOT EXISTS confirmed_sl        numeric(18,8),
  ADD COLUMN IF NOT EXISTS requested_sl        numeric(18,8),
  ADD COLUMN IF NOT EXISTS previous_confirmed_sl numeric(18,8),
  ADD COLUMN IF NOT EXISTS sl_version           integer NOT NULL DEFAULT 0,
  ADD COLUMN IF NOT EXISTS management_stage    varchar(30) NOT NULL DEFAULT 'OPEN_INITIAL_RISK',
  ADD COLUMN IF NOT EXISTS broker_ack_status    varchar(20) NOT NULL DEFAULT 'NONE',
  ADD COLUMN IF NOT EXISTS broker_ack_retcode  integer,
  ADD COLUMN IF NOT EXISTS last_sl_update       timestamp with time zone,
  ADD COLUMN IF NOT EXISTS initial_r            numeric(10,4);

-- ─── SL modification history table (audit trail) ───
CREATE TABLE IF NOT EXISTS trading.sl_modification_history (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    position_id         uuid NOT NULL,
    signal_id           uuid,
    strategy_id         varchar(50),
    direction           varchar(10) NOT NULL,
    symbol              varchar(20) NOT NULL DEFAULT 'XAUUSD',
    old_confirmed_sl    numeric(18,8),
    proposed_sl         numeric(18,8) NOT NULL,
    new_confirmed_sl    numeric(18,8),
    current_r           numeric(10,4),
    management_stage    varchar(30) NOT NULL,
    trigger_reason      varchar(100),
    broker_ack_status   varchar(20) NOT NULL DEFAULT 'PENDING',
    broker_ack_retcode  integer,
    management_version  integer NOT NULL DEFAULT 0,
    is_monotonic        boolean NOT NULL DEFAULT true,
    created_at          timestamp with time zone NOT NULL DEFAULT now(),
    FOREIGN KEY (position_id) REFERENCES trading.positions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_sl_mod_position ON trading.sl_modification_history (position_id);
CREATE INDEX IF NOT EXISTS idx_sl_mod_version ON trading.sl_modification_history (position_id, management_version);
CREATE UNIQUE INDEX IF NOT EXISTS idx_sl_mod_idempotent ON trading.sl_modification_history (position_id, management_version)
    WHERE management_version > 0;

-- ─── Verify ───
SELECT 'positions columns added' as status;
SELECT 'sl_modification_history created' as status;
