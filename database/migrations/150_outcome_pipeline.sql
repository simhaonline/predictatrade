-- 150_outcome_pipeline.sql
-- Phase 0.5 (prompt.md, Candidate B): outcome capture pipeline.
-- Schema APPROVED in docs/strategy/OUTCOME_PIPELINE.md (commit 966a4c5).
--
-- Part 1: trading.predictions — one row per EXECUTABLE signal at emit time
--         (pre-registered expectation). Idempotent on signal_id.
-- Part 2: trading.prediction_outcomes — additive columns on the EXISTING
--         table (backward compatible; existing columns/FK preserved).
-- Part 3: indexes + link_status/close_reason CHECK constraints.
-- Part 4: retention registration follows the mig-147 pruning framework
--         (NOT registered here — outcomes are calibration source data,
--         retention is an operator decision; do not prune silently).

-- ── Part 1: predictions (pre-registered expectations) ───────────────────
-- NOTE: trading.predictions ALREADY EXISTS with a legacy shape (0 rows).
-- CREATE TABLE IF NOT EXISTS silently no-op'd on first apply — discovered by
-- the real-DB round-trip test (column "strategy_id" does not exist). This
-- migration therefore ALTERS the existing table additively to the approved
-- schema instead of creating it.

ALTER TABLE trading.predictions
  ADD COLUMN IF NOT EXISTS strategy_id             TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS strategy_version        TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS entry                   NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS stop_loss               NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS tp1                     NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS tp2                     NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS tp3                     NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS grade                   TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS signal_class            TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS tier                    TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS tier_reason             TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS composite_score         NUMERIC(10,4),
  ADD COLUMN IF NOT EXISTS family_sub_scores       JSONB,
  ADD COLUMN IF NOT EXISTS regime                  TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS session                 TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS gate_decisions          JSONB,
  ADD COLUMN IF NOT EXISTS astro_sizing_multiplier NUMERIC(10,4),
  ADD COLUMN IF NOT EXISTS yoga_bias               NUMERIC(10,4),
  ADD COLUMN IF NOT EXISTS macro_bias_x6           NUMERIC(10,4),
  ADD COLUMN IF NOT EXISTS session_at_emit         TEXT NOT NULL DEFAULT '';

-- Legacy NOT NULL columns lack defaults; relax them (additive, backward
-- compatible — legacy writers still supply values).
ALTER TABLE trading.predictions ALTER COLUMN prediction_horizon DROP NOT NULL;
ALTER TABLE trading.predictions ALTER COLUMN expires_at DROP NOT NULL;
ALTER TABLE trading.predictions ALTER COLUMN confidence_band DROP NOT NULL;
ALTER TABLE trading.predictions ALTER COLUMN model_version DROP NOT NULL;
ALTER TABLE trading.predictions ALTER COLUMN strategy_version DROP NOT NULL;
ALTER TABLE trading.predictions ALTER COLUMN target_definition DROP NOT NULL;
ALTER TABLE trading.predictions ALTER COLUMN invalidation_definition DROP NOT NULL;

-- idempotency: one prediction per signal, ever. signal_id already NOT NULL
-- UNIQUE in the legacy shape? Verify defensively:
DO $$
BEGIN
  IF NOT EXISTS (
    SELECT 1 FROM pg_constraint
    WHERE conrelid = 'trading.predictions'::regclass AND conname = 'predictions_signal_id_unique'
  ) THEN
    ALTER TABLE trading.predictions
      ADD CONSTRAINT predictions_signal_id_unique UNIQUE (signal_id);
  END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_predictions_strategy_created
  ON trading.predictions (strategy_id, created_at);
CREATE INDEX IF NOT EXISTS idx_predictions_signal_id
  ON trading.predictions (signal_id);

-- ── Part 2: prediction_outcomes additive columns ────────────────────────
-- Existing columns preserved: id, prediction_id (FK), outcome_type,
-- outcome_value, realized_rr, mfe, mae, time_to_outcome, created_at.
ALTER TABLE trading.prediction_outcomes
  ADD COLUMN IF NOT EXISTS signal_id        UUID,
  ADD COLUMN IF NOT EXISTS link_status      TEXT NOT NULL DEFAULT 'UNLINKED',
  ADD COLUMN IF NOT EXISTS strategy_id      TEXT NOT NULL DEFAULT '',
  ADD COLUMN IF NOT EXISTS close_reason     TEXT,
  ADD COLUMN IF NOT EXISTS realized_pnl     NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS r_multiple       NUMERIC(10,4),
  ADD COLUMN IF NOT EXISTS mae_points       NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS mfe_points       NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS duration_seconds BIGINT,
  ADD COLUMN IF NOT EXISTS slippage_est     NUMERIC(10,4),
  ADD COLUMN IF NOT EXISTS spread_at_entry  NUMERIC(10,4),
  ADD COLUMN IF NOT EXISTS spread_at_exit   NUMERIC(10,4),
  ADD COLUMN IF NOT EXISTS trade_result_id  UUID,
  ADD COLUMN IF NOT EXISTS schema_version   TEXT NOT NULL DEFAULT '1.0';

-- link_status / close_reason as CHECK constraints (no enum types in use
-- elsewhere in trading schema — CHECK keeps migrations simple and additive).
ALTER TABLE trading.prediction_outcomes DROP CONSTRAINT IF EXISTS po_link_status_chk;
ALTER TABLE trading.prediction_outcomes ADD CONSTRAINT po_link_status_chk
  CHECK (link_status IN ('LINKED', 'UNLINKED'));

ALTER TABLE trading.prediction_outcomes DROP CONSTRAINT IF EXISTS po_close_reason_chk;
ALTER TABLE trading.prediction_outcomes ADD CONSTRAINT po_close_reason_chk
  CHECK (close_reason IS NULL OR close_reason IN (
    'AUTO','MANUAL','BE','TRAIL','TP1','TP2','TP3',
    'FRIDAY_FLATTEN','STOP','HARD_STOP','TIMEOUT'));

-- signal_id uniqueness (idempotency key): unique where not null
CREATE UNIQUE INDEX IF NOT EXISTS idx_po_signal_id
  ON trading.prediction_outcomes (signal_id) WHERE signal_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_po_strategy_created
  ON trading.prediction_outcomes (strategy_id, created_at);

ALTER TABLE trading.prediction_outcomes DROP CONSTRAINT IF EXISTS po_trade_result_fk;
ALTER TABLE trading.prediction_outcomes ADD CONSTRAINT po_trade_result_fk
  FOREIGN KEY (trade_result_id) REFERENCES trading.trade_results (id) ON DELETE SET NULL;