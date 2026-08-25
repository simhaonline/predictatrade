-- Migration: Add idempotency_key to referral.payouts
-- Fixes audit 42703: payouts.service.ts references idempotency_key on
-- referral.payouts for deduplication, but no migration ever added the column.
-- The column was added to commission_ledger (024) and finance.ledger_entries (075)
-- but was missed for payouts.
ALTER TABLE referral.payouts ADD COLUMN IF NOT EXISTS idempotency_key VARCHAR(255);
CREATE UNIQUE INDEX IF NOT EXISTS idx_payouts_idempotency
  ON referral.payouts(user_id, idempotency_key)
  WHERE idempotency_key IS NOT NULL;
