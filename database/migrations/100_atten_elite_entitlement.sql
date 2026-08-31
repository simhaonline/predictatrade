-- Migration 100: Expose ATEN (6th strategy) to ELITE plan entitlements.
-- The MANIFEST lists 6 strategies; ELITE is the top tier and must allow ATEN.
-- Idempotent: only adds ATEN if missing from allowed_strategies.
UPDATE control.plans
SET allowed_strategies = allowed_strategies || '["ATEN"]'::jsonb
WHERE code = 'ELITE'
  AND NOT (allowed_strategies ? 'ATEN');

-- Keep the backend canonical STRATEGIES list in sync (documentation only;
-- the Go/NestJS constants are updated in code). ELITE now permits all 6.
COMMENT ON COLUMN control.plans.allowed_strategies IS
  'Strategies permitted by the plan. ELITE permits STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING, MARNIE_FIB(EQFE), ATEN.';
