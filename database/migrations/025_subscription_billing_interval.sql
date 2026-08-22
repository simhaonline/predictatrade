-- Persist the billing interval selected by the customer.
-- Additive and idempotent; historical subscription rows are retained.
ALTER TABLE billing.subscriptions
  ADD COLUMN IF NOT EXISTS billing_interval VARCHAR(20) NOT NULL DEFAULT 'MONTHLY';

UPDATE billing.subscriptions s
SET billing_interval = COALESCE(p.billing_interval, 'MONTHLY')
FROM control.plans p
WHERE p.id = s.plan_id
  AND (s.billing_interval IS NULL OR s.billing_interval = 'MONTHLY');

CREATE INDEX IF NOT EXISTS idx_subscriptions_billing_interval
  ON billing.subscriptions(billing_interval, status);
