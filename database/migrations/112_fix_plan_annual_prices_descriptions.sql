-- Migration 112: Correct annual prices + plan descriptions to match the
-- MASTER PROMPT tiers defined in migration 110.
-- Annual = 10x monthly (2 months free) so annual_savings_percent is positive.
-- Descriptions reflect the real strategy matrix and Free daily cap.

UPDATE control.plans SET
  annual_price = NULL,
  description = 'STANDARD_SCALPING only — up to 5 signals/day. No card required.',
  updated_at = now()
WHERE code = 'FREE';

UPDATE control.plans SET
  annual_price = 490,
  description = 'STANDARD_SCALPING + STANDARD_SWING. Unlimited signals.',
  updated_at = now()
WHERE code = 'STANDARD';

UPDATE control.plans SET
  annual_price = 1990,
  description = 'All 4 core strategies — scalping & swing, ultra & trend. Unlimited signals.',
  updated_at = now()
WHERE code = 'PRO';

UPDATE control.plans SET
  annual_price = 4990,
  description = 'All 6 strategies including EQFE (MARNIE_FIB) and ATEN. Unlimited signals.',
  updated_at = now()
WHERE code = 'ELITE';
