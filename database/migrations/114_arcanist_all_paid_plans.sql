-- Migration 114: Expose ARCANIST (IMLR) on ALL paid plans.
-- Per operator directive: the Institutional MSNR Liquidity Reversal Model (Arcanist)
-- is now a feature of every paid subscription (BASIC, STANDARD, PRO, ELITE),
-- not just ELITE. Idempotent: only adds ARCANIST where missing.
UPDATE control.plans
SET allowed_strategies = allowed_strategies || '["ARCANIST"]'::jsonb
WHERE code IN ('BASIC', 'STANDARD', 'PRO', 'ELITE')
  AND NOT (allowed_strategies ? 'ARCANIST');

COMMENT ON COLUMN control.plans.allowed_strategies IS
  'Strategies permitted by the plan. All paid tiers (BASIC/STANDARD/PRO/ELITE) now include ARCANIST (IMLR); FREE remains STANDARD_SCALPING only.';
