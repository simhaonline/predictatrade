-- Per-plan capital-protection caps (Bug 7 + recommendation: tier-based loss caps).
-- The realtime engine reads these columns during license validation and applies
-- them to the live DailyLoss/ProfitTarget/RiskOversize gates. Loss caps are stored
-- as NEGATIVE percentages (e.g. -5.00 = max 5% loss). Idempotent UPDATE — never
-- rewrites applied migration history.
UPDATE control.plans SET
  daily_loss_cap_pct   = -2.00,
  weekly_loss_cap_pct  = -3.00,
  monthly_loss_cap_pct = -5.00,
  per_trade_risk_pct   = 1.0
WHERE code = 'FREE';

UPDATE control.plans SET
  daily_loss_cap_pct   = -2.00,
  weekly_loss_cap_pct  = -4.00,
  monthly_loss_cap_pct = -6.00,
  per_trade_risk_pct   = 1.0
WHERE code = 'BASIC';

UPDATE control.plans SET
  daily_loss_cap_pct   = -3.00,
  weekly_loss_cap_pct  = -5.00,
  monthly_loss_cap_pct = -8.00,
  per_trade_risk_pct   = 1.5
WHERE code = 'STANDARD';

UPDATE control.plans SET
  daily_loss_cap_pct   = -5.00,
  weekly_loss_cap_pct  = -8.00,
  monthly_loss_cap_pct = -12.00,
  per_trade_risk_pct   = 1.5
WHERE code = 'PRO';

UPDATE control.plans SET
  daily_loss_cap_pct   = -5.00,
  weekly_loss_cap_pct  = -10.00,
  monthly_loss_cap_pct = -15.00,
  per_trade_risk_pct   = 2.0
WHERE code = 'ELITE';

-- Keep the mirrored key/value entitlement rows in sync so any consumer reading
-- control.plan_entitlements sees the same caps.
INSERT INTO control.plan_entitlements (plan_id, entitlement_key, entitlement_value)
SELECT p.id, k.key, to_jsonb(k.val)
FROM control.plans p
CROSS JOIN LATERAL (VALUES
  ('daily_loss_cap_pct', p.daily_loss_cap_pct),
  ('weekly_loss_cap_pct', p.weekly_loss_cap_pct),
  ('monthly_loss_cap_pct', p.monthly_loss_cap_pct),
  ('per_trade_risk_pct', p.per_trade_risk_pct)
) AS k(key, val)
ON CONFLICT (plan_id, entitlement_key) DO UPDATE SET entitlement_value = EXCLUDED.entitlement_value;
