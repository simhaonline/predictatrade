-- M2 (MED): subscription lifecycle state machine supports PAUSED.
-- Adds a paused_at timestamp and permits 'PAUSED' as a subscriptions.status
-- value. NOT VALID keeps existing rows untouched while enforcing the new
-- allowed set for future writes.

ALTER TABLE billing.subscriptions ADD COLUMN IF NOT EXISTS paused_at TIMESTAMPTZ;

ALTER TABLE billing.subscriptions
  DROP CONSTRAINT IF EXISTS chk_subscriptions_status;

ALTER TABLE billing.subscriptions
  ADD CONSTRAINT chk_subscriptions_status CHECK (
    status IN (
      'INCOMPLETE', 'TRIAL', 'ACTIVE', 'PAST_DUE', 'GRACE', 'SUSPENDED',
      'CANCEL_AT_PERIOD_END', 'CANCELLED', 'EXPIRED', 'PAUSED'
    )
  ) NOT VALID;
