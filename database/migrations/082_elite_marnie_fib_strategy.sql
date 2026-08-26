-- Migration 082: Add MARNIE_FIB to ELITE plan so it is available to Elite subscribers.
-- The ELITE plan previously excluded MARNIE_FIB from allowed_strategies even though
-- the strategy engine was running it live in SHADOW mode. This migration makes
-- MARNIE_FIB selectable by Elite users via subscription and licensing.

-- 1. Update the plans table
UPDATE control.plans
SET allowed_strategies = '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING","MARNIE_FIB"]'::jsonb,
    max_active_strategy_slots = 5,
    description = 'All 5 strategies + full confluence + MarnieFib evidence audit',
    updated_at = now()
WHERE code = 'ELITE'
RETURNING id, code, allowed_strategies, max_active_strategy_slots;

-- 2. Mirror into plan_entitlements
INSERT INTO control.plan_entitlements (plan_id, entitlement_key, entitlement_value)
SELECT p.id, 'max_active_strategies', to_jsonb(5)
FROM control.plans p
WHERE p.code = 'ELITE'
ON CONFLICT (plan_id, entitlement_key) DO UPDATE SET
  entitlement_value = EXCLUDED.entitlement_value;

-- 3. Update ELITE test user license (migration 029 uses hardcoded allowed_strategies)
UPDATE licensing.licenses
SET allowed_strategies = '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING","MARNIE_FIB"]'::jsonb,
    updated_at = now()
WHERE license_key = 'PAT-A1B2C3D4-0004-4000-8000-000000000004'
  AND user_id = 'a1b2c3d4-0004-4000-8000-000000000004';
