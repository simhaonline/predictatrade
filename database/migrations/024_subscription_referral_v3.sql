-- Predict-A-Trade v3 commercial model.
-- Additive and idempotent: legacy plans, subscriptions, payments and ledger rows
-- are retained. v3 rules are effective-dated and never rewrite historical rows.

ALTER TABLE control.plans
  ADD COLUMN IF NOT EXISTS annual_price DECIMAL(18,8),
  ADD COLUMN IF NOT EXISTS visible BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS legacy BOOLEAN NOT NULL DEFAULT FALSE,
  ADD COLUMN IF NOT EXISTS billing_enabled BOOLEAN NOT NULL DEFAULT TRUE,
  ADD COLUMN IF NOT EXISTS sort_order INTEGER NOT NULL DEFAULT 0;

UPDATE control.plans
SET visible = FALSE, legacy = TRUE, billing_enabled = FALSE, sort_order = 99
WHERE code = 'BASIC';

INSERT INTO control.plans
  (code, name, description, setup_fee, monthly_price, annual_price, currency,
   billing_interval, max_active_strategy_slots, allowed_strategies, status,
   visible, legacy, billing_enabled, sort_order)
VALUES
  ('FREE', 'Free', 'Explore Predict-A-Trade', 0, 0, NULL, 'USD', 'MONTHLY', 1,
   '["STANDARD_SCALPING"]'::jsonb, 'ACTIVE', TRUE, FALSE, FALSE, 0)
ON CONFLICT (code) DO UPDATE SET
  monthly_price = EXCLUDED.monthly_price,
  annual_price = EXCLUDED.annual_price,
  visible = EXCLUDED.visible,
  legacy = EXCLUDED.legacy,
  billing_enabled = EXCLUDED.billing_enabled,
  sort_order = EXCLUDED.sort_order,
  updated_at = now();

UPDATE control.plans SET annual_price = 990, sort_order = 10
WHERE code = 'STANDARD' AND legacy = FALSE;
UPDATE control.plans SET monthly_price = 299, annual_price = 2990, sort_order = 20
WHERE code = 'PRO' AND legacy = FALSE;
UPDATE control.plans SET monthly_price = 699, annual_price = 6990, sort_order = 30
WHERE code = 'ELITE' AND legacy = FALSE;
UPDATE control.plans SET monthly_price = 99, annual_price = 990, setup_fee = 0
WHERE code = 'STANDARD' AND legacy = FALSE;

CREATE TABLE IF NOT EXISTS control.plan_entitlement_versions (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  plan_id UUID NOT NULL REFERENCES control.plans(id),
  feature_key VARCHAR(100) NOT NULL,
  enabled BOOLEAN NOT NULL DEFAULT TRUE,
  limit_value DECIMAL(18,8),
  limit_period VARCHAR(30),
  metadata JSONB NOT NULL DEFAULT '{}',
  effective_from TIMESTAMPTZ NOT NULL,
  effective_until TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(plan_id, feature_key, effective_from)
);

CREATE TABLE IF NOT EXISTS control.strategy_preferences (
  user_id UUID PRIMARY KEY REFERENCES iam.users(id) ON DELETE CASCADE,
  selected_strategies JSONB NOT NULL DEFAULT '[]',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS billing.subscription_event_v3 (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES iam.users(id),
  subscription_id UUID REFERENCES billing.subscriptions(id),
  plan_id UUID REFERENCES control.plans(id),
  previous_plan_id UUID REFERENCES control.plans(id),
  event_type VARCHAR(40) NOT NULL,
  billing_interval VARCHAR(20),
  gross_amount DECIMAL(18,8) NOT NULL DEFAULT 0,
  discount_amount DECIMAL(18,8) NOT NULL DEFAULT 0,
  tax_amount DECIMAL(18,8) NOT NULL DEFAULT 0,
  eligible_amount DECIMAL(18,8) NOT NULL DEFAULT 0,
  currency VARCHAR(3) NOT NULL DEFAULT 'USD',
  payment_id UUID REFERENCES billing.payments(id),
  external_reference VARCHAR(255),
  metadata JSONB NOT NULL DEFAULT '{}',
  effective_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_subscription_event_v3_external_reference
  ON billing.subscription_event_v3(external_reference)
  WHERE external_reference IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_subscription_event_v3_user_time
  ON billing.subscription_event_v3(user_id, effective_at DESC);
CREATE INDEX IF NOT EXISTS idx_subscription_event_v3_type
  ON billing.subscription_event_v3(event_type, effective_at DESC);

CREATE TABLE IF NOT EXISTS trading.signal_delivery_ledger (
  id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  user_id UUID NOT NULL REFERENCES iam.users(id),
  -- `trading.signals` is a Timescale hypertable with a composite
  -- (id, created_at) primary key in the deployed schema, so a scalar FK
  -- cannot be declared here. Referential integrity is checked by the
  -- distribution writer and this indexed identifier preserves compatibility.
  signal_id UUID NOT NULL,
  strategy VARCHAR(50) NOT NULL,
  plan_id UUID NOT NULL REFERENCES control.plans(id),
  entitlement_snapshot JSONB NOT NULL DEFAULT '{}',
  delivery_channel VARCHAR(30) NOT NULL,
  quota_period DATE,
  quota_consumed BOOLEAN NOT NULL DEFAULT FALSE,
  delivered_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(user_id, signal_id, delivery_channel)
);
CREATE INDEX IF NOT EXISTS idx_signal_delivery_ledger_quota
  ON trading.signal_delivery_ledger(user_id, quota_period, quota_consumed);
CREATE INDEX IF NOT EXISTS idx_signal_delivery_ledger_signal
  ON trading.signal_delivery_ledger(signal_id);

ALTER TABLE referral.commission_ledger
  ADD COLUMN IF NOT EXISTS subscription_event_id UUID REFERENCES billing.subscription_event_v3(id),
  ADD COLUMN IF NOT EXISTS eligible_amount DECIMAL(18,8),
  ADD COLUMN IF NOT EXISTS subscription_event_type VARCHAR(40),
  ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_commission_v3_idempotency
  ON referral.commission_ledger(idempotency_key)
  WHERE idempotency_key IS NOT NULL;

-- v3 configuration is separate from existing effective-dated legacy rules.
INSERT INTO control.plan_entitlements (plan_id, entitlement_key, entitlement_value)
SELECT p.id, v.key, v.value
FROM control.plans p
JOIN LATERAL jsonb_each(
  CASE p.code
    WHEN 'FREE' THEN '{"strategy.standard_scalping":true,"strategy.max_active_slots":1,"signal.monthly_delivery_limit":5,"signal.tp1":true,"signal.confidence":true,"signal.score":true,"history.days":7,"notification.in_app":true}'::jsonb
    WHEN 'STANDARD' THEN '{"strategy.standard_scalping":true,"strategy.standard_swing":true,"strategy.max_active_slots":1,"signal.monthly_delivery_limit":null,"signal.tp1":true,"signal.tp2":true,"signal.tp3":true,"signal.probability":true,"signal.score":true,"history.days":90,"notification.standard":true}'::jsonb
    WHEN 'PRO' THEN '{"strategy.max_active_slots":2,"signal.monthly_delivery_limit":null,"signal.advanced_evidence":true,"trade.trailing_updates":true,"history.days":365,"analytics.advanced":true,"analytics.strategy_comparison":true,"notification.push":true}'::jsonb
    WHEN 'ELITE' THEN '{"strategy.max_active_slots":4,"history.unlimited":true,"analytics.regime":true,"export.csv":true,"api.access":false,"support.elite":true}'::jsonb
  END
) v(key, value) ON TRUE
WHERE p.code IN ('FREE','STANDARD','PRO','ELITE')
ON CONFLICT (plan_id, entitlement_key) DO UPDATE SET entitlement_value = EXCLUDED.entitlement_value;

-- Only new v3 events use these rules. Existing v1/v2 rows remain untouched.
INSERT INTO referral.commission_rules (plan_id, level, base_rate, effective_from, rule_version, active)
SELECT p.id, r.level, r.rate, '2026-08-22 00:00:00+00', 300, TRUE
FROM control.plans p
JOIN (VALUES
  ('STANDARD', 1, 0.10::DECIMAL), ('STANDARD', 2, 0.03), ('STANDARD', 3, 0.01),
  ('STANDARD', 4, 0.00), ('STANDARD', 5, 0.00),
  ('PRO', 1, 0.15), ('PRO', 2, 0.04), ('PRO', 3, 0.02),
  ('PRO', 4, 0.00), ('PRO', 5, 0.00),
  ('ELITE', 1, 0.18), ('ELITE', 2, 0.05), ('ELITE', 3, 0.02),
  ('ELITE', 4, 0.00), ('ELITE', 5, 0.00)
) r(code, level, rate) ON r.code = p.code
ON CONFLICT (plan_id, level, effective_from) DO NOTHING;

INSERT INTO referral.purchase_commission_rules
  (purchase_type, multiplier, max_referral_level, effective_from, rule_version, active)
VALUES
  ('FIRST_PURCHASE', 1.00, 3, '2026-08-22 00:00:00+00', 300, TRUE),
  ('SECOND_PURCHASE', 0.75, 1, '2026-08-22 00:00:00+00', 300, TRUE),
  ('RECURRING_PURCHASE', 0.50, 3, '2026-08-22 00:00:00+00', 300, TRUE)
ON CONFLICT (purchase_type, effective_from) DO NOTHING;

CREATE TABLE IF NOT EXISTS control.commercial_feature_flags (
  flag_key VARCHAR(80) PRIMARY KEY,
  enabled BOOLEAN NOT NULL DEFAULT FALSE,
  metadata JSONB NOT NULL DEFAULT '{}',
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
INSERT INTO control.commercial_feature_flags(flag_key, enabled) VALUES
  ('SUBSCRIPTION_V3_ENABLED', FALSE), ('FREE_PLAN_ENABLED', FALSE),
  ('ANNUAL_BILLING_ENABLED', FALSE), ('REFERRAL_V3_ENABLED', FALSE)
ON CONFLICT (flag_key) DO NOTHING;
