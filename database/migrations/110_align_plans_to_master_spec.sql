-- Migration 110: Align plan tiers to MASTER PROMPT spec (prompt.md)
--   FREE     $0   : STANDARD_SCALPING only, max 5 signals/day
--   STANDARD $49  : STANDARD_SCALPING + STANDARD_SWING
--   PRO      $199 : 4 core strategies
--   ELITE    $499 : all 6 (incl MARNIE_FIB/EQFE + ATEN)
-- Idempotent UPDATEs keyed on plan code. Plan names are intentionally left
-- unchanged to avoid breaking existing UI/test assertions.

UPDATE control.plans
SET monthly_price = 0,
    max_active_strategy_slots = 1,
    max_signals_per_day = 5,
    allowed_strategies = '["STANDARD_SCALPING"]'::jsonb,
    updated_at = now()
WHERE code = 'FREE';

UPDATE control.plans
SET monthly_price = 49,
    max_active_strategy_slots = 2,
    max_signals_per_day = NULL,
    allowed_strategies = '["STANDARD_SCALPING","STANDARD_SWING"]'::jsonb,
    updated_at = now()
WHERE code = 'STANDARD';

UPDATE control.plans
SET monthly_price = 199,
    max_active_strategy_slots = 4,
    max_signals_per_day = NULL,
    allowed_strategies = '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING"]'::jsonb,
    updated_at = now()
WHERE code = 'PRO';

UPDATE control.plans
SET monthly_price = 499,
    max_active_strategy_slots = 6,
    max_signals_per_day = NULL,
    allowed_strategies = '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING","MARNIE_FIB","ATEN"]'::jsonb,
    updated_at = now()
WHERE code = 'ELITE';
