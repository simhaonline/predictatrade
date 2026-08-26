-- 080: Signal quality diagnostics + expectancy + config versioning
-- prompt.md Sections 12-14, 17-18, 30-32

BEGIN;

-- Add quality grade, expectancy, rejection diagnostics to trading.signals
ALTER TABLE trading.signals
  ADD COLUMN IF NOT EXISTS quality_grade VARCHAR(10),
  ADD COLUMN IF NOT EXISTS expectancy_r NUMERIC(18,8),
  ADD COLUMN IF NOT EXISTS expectancy_score DOUBLE PRECISION,
  ADD COLUMN IF NOT EXISTS primary_rejection_reason VARCHAR(50),
  ADD COLUMN IF NOT EXISTS rejection_reasons TEXT[],
  ADD COLUMN IF NOT EXISTS strategy_config_version VARCHAR(20);

-- Backfill existing rows with sane defaults
UPDATE trading.signals
SET quality_grade = CASE
    WHEN direction IN ('BUY', 'SELL') AND grade IN ('A+', 'A') THEN grade::VARCHAR
    WHEN direction IN ('BUY', 'SELL') THEN 'B'
    WHEN direction = 'NO-TRADE' THEN 'NO-TRADE'
    ELSE 'UNRATED'
END
WHERE quality_grade IS NULL;

-- Index for admin dashboard queries
CREATE INDEX IF NOT EXISTS idx_signals_quality_grade
  ON trading.signals (strategy_id, quality_grade, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_signals_expectancy
  ON trading.signals (strategy_id, expectancy_score DESC NULLS LAST)
  WHERE expectancy_score IS NOT NULL;

-- Rejection diagnostics table for strategy starvation monitoring (prompt.md Section 17)
CREATE TABLE IF NOT EXISTS trading.signal_rejection_diagnostics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id VARCHAR(30) NOT NULL,
    window_start TIMESTAMPTZ NOT NULL,
    window_end TIMESTAMPTZ NOT NULL,
    total_candidates INTEGER NOT NULL DEFAULT 0,
    total_rejected INTEGER NOT NULL DEFAULT 0,
    total_qualified INTEGER NOT NULL DEFAULT 0,
    rejection_counts JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (strategy_id, window_start)
);

CREATE INDEX IF NOT EXISTS idx_rejection_diag_strategy
  ON trading.signal_rejection_diagnostics (strategy_id, window_start DESC);

-- Add config version tracking to control.strategy_config
ALTER TABLE control.strategy_config
  ADD COLUMN IF NOT EXISTS config_version VARCHAR(20) DEFAULT '1.0.0';

COMMIT;
