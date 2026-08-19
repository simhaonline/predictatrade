-- Migration 019: Signal bug closure — add timestamp, exit lifecycle, and candidate classification fields
-- SOW Phase 2 Sections 26-35: Detailed timestamp model, exit lifecycle, candidate thresholds

-- Add timestamp model fields
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS market_time timestamptz;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS detected_at timestamptz;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS candidate_detected_at timestamptz;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS qualified_at timestamptz;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS published_at timestamptz;

-- Add candidate classification fields
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS signal_class varchar(20) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS candidate_threshold numeric(5,2) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS trade_threshold numeric(5,2) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS entry_type varchar(10) DEFAULT '';

-- Add exit lifecycle fields (NULL/zero until trade actually closes)
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS exit_price numeric(18,8) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS exit_reason varchar(30) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS closed_at timestamptz;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS realized_pnl numeric(18,8) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS realized_r numeric(10,4) DEFAULT 0;

-- Add versioning and linkage fields
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS geometry_version varchar(20) DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS conflict_penalty numeric(10,4) DEFAULT 0;
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS parent_candidate_id varchar(100) DEFAULT '';

-- Fix strategy_evaluations column widths for candidate direction values
ALTER TABLE trading.strategy_evaluations ALTER COLUMN direction TYPE varchar(20);
ALTER TABLE trading.strategy_evaluations ALTER COLUMN reason TYPE varchar(2000);

-- Add index for exit lifecycle queries
CREATE INDEX IF NOT EXISTS idx_signals_exit_unclosed ON trading.signals (strategy_id, created_at DESC) 
  WHERE exit_reason = '' OR exit_reason IS NULL;

COMMENT ON TABLE trading.signals IS 'Trading signals with full lifecycle: candidate detection, qualification, execution, and exit tracking';
