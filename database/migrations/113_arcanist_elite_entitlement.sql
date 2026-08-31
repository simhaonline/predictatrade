-- Migration 113: Expose ARCANIST (7th strategy) to ELITE plan entitlements.
-- ARCANIST is the institutional-liquidity (ICT) engine, gated to ELITE along
-- with the other premium engines (MARNIE_FIB/EQFE, ATEN). Idempotent: only
-- adds ARCANIST if missing from allowed_strategies.
UPDATE control.plans
SET allowed_strategies = allowed_strategies || '["ARCANIST"]'::jsonb
WHERE code = 'ELITE'
  AND NOT (allowed_strategies ? 'ARCANIST');

-- Keep the documentation comment in sync: ELITE now permits all 7 engines.
COMMENT ON COLUMN control.plans.allowed_strategies IS
  'Strategies permitted by the plan. ELITE permits STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING, MARNIE_FIB(EQFE), ATEN, ARCANIST.';
