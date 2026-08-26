-- Predict-A-Trade Bug 7 + ADDENDUM 5: canonical 4-tier plan model with
-- strategy access, signal caps, feature-access levels and per-plan risk caps.
-- Additive and idempotent. Never rewrites applied migration history.

ALTER TABLE control.plans
  ADD COLUMN IF NOT EXISTS max_signals_per_day INTEGER,
  ADD COLUMN IF NOT EXISTS feature_access_level VARCHAR(20) NOT NULL DEFAULT 'core',
  ADD COLUMN IF NOT EXISTS monthly_profit_target_pct DECIMAL(5,2),
  ADD COLUMN IF NOT EXISTS monthly_loss_cap_pct DECIMAL(5,2) NOT NULL DEFAULT -5.00,
  ADD COLUMN IF NOT EXISTS daily_loss_cap_pct DECIMAL(5,2) NOT NULL DEFAULT -2.00,
  ADD COLUMN IF NOT EXISTS weekly_loss_cap_pct DECIMAL(5,2) NOT NULL DEFAULT -4.00,
  ADD COLUMN IF NOT EXISTS per_trade_risk_pct DECIMAL(4,2);

INSERT INTO control.plans
  (code, name, description, setup_fee, monthly_price, annual_price, currency,
   billing_interval, max_active_strategy_slots, allowed_strategies, status,
   visible, legacy, billing_enabled, sort_order,
   max_signals_per_day, feature_access_level, per_trade_risk_pct)
VALUES
  ('FREE', 'Free', 'Explore Predict-A-Trade: Standard Swing only, 3 signals/day', 0, 0, NULL,
   'USD', 'MONTHLY', 1, '["STANDARD_SWING"]'::jsonb, 'ACTIVE',
   TRUE, FALSE, FALSE, 0, 3, 'core', 1.0),
  ('STANDARD', 'Standard', 'Any 1 of the 4 strategies, unlimited signals', 0, 99, 990,
   'USD', 'MONTHLY', 1, '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING"]'::jsonb, 'ACTIVE',
   TRUE, FALSE, TRUE, 10, NULL, 'advanced', 1.0),
  ('PRO', 'Pro', 'Any 2 of the 4 strategies, structure/SMC + cross-market view', 0, 299, 2990,
   'USD', 'MONTHLY', 2, '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING"]'::jsonb, 'ACTIVE',
   TRUE, FALSE, TRUE, 20, NULL, 'smc', 1.5),
  ('ELITE', 'Elite', 'All 5 strategies + full confluence + MarnieFib evidence audit', 0, 699, 6990,
   'USD', 'MONTHLY', 5, '["STANDARD_SCALPING","ULTRA_SCALPING","STANDARD_SWING","TREND_SWING","MARNIE_FIB"]'::jsonb, 'ACTIVE',
   TRUE, FALSE, TRUE, 30, NULL, 'full', 1.5)
ON CONFLICT (code) DO UPDATE SET
  name = EXCLUDED.name,
  description = EXCLUDED.description,
  monthly_price = EXCLUDED.monthly_price,
  annual_price = EXCLUDED.annual_price,
  setup_fee = EXCLUDED.setup_fee,
  billing_interval = EXCLUDED.billing_interval,
  max_active_strategy_slots = EXCLUDED.max_active_strategy_slots,
  allowed_strategies = EXCLUDED.allowed_strategies,
  status = EXCLUDED.status,
  visible = EXCLUDED.visible,
  legacy = FALSE,
  billing_enabled = EXCLUDED.billing_enabled,
  sort_order = EXCLUDED.sort_order,
  max_signals_per_day = EXCLUDED.max_signals_per_day,
  feature_access_level = EXCLUDED.feature_access_level,
  per_trade_risk_pct = EXCLUDED.per_trade_risk_pct,
  updated_at = now();

-- Mirror the new plan-level limits into the existing key/value entitlement rows
-- so entitlement consumers that read control.plan_entitlements see them too.
INSERT INTO control.plan_entitlements (plan_id, entitlement_key, entitlement_value)
SELECT p.id, k.key,
       CASE
         WHEN k.key IN ('max_signals_per_day', 'max_active_strategies') THEN to_jsonb(
           CASE WHEN k.key = 'max_signals_per_day' THEN p.max_signals_per_day ELSE p.max_active_strategy_slots END)
         ELSE to_jsonb(p.feature_access_level)
       END
FROM control.plans p
CROSS JOIN (VALUES
  ('max_signals_per_day'),
  ('max_active_strategies'),
  ('feature_access_level')
) AS k(key)
WHERE p.code IN ('FREE','STANDARD','PRO','ELITE')
  AND (k.key <> 'max_signals_per_day' OR p.max_signals_per_day IS NOT NULL)
ON CONFLICT (plan_id, entitlement_key) DO UPDATE SET
  entitlement_value = EXCLUDED.entitlement_value;

INSERT INTO control.plan_entitlements (plan_id, entitlement_key, entitlement_value)
SELECT p.id, k.key, to_jsonb(k.val)
FROM control.plans p
CROSS JOIN LATERAL (VALUES
  ('daily_loss_cap_pct', p.daily_loss_cap_pct),
  ('weekly_loss_cap_pct', p.weekly_loss_cap_pct),
  ('monthly_loss_cap_pct', p.monthly_loss_cap_pct),
  ('per_trade_risk_pct', p.per_trade_risk_pct)
) AS k(key, val)
WHERE p.code IN ('FREE','STANDARD','PRO','ELITE')
ON CONFLICT (plan_id, entitlement_key) DO UPDATE SET
  entitlement_value = EXCLUDED.entitlement_value;
