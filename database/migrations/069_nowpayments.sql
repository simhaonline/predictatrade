-- 069: NOWPayments (USDT) crypto collection support.
--
-- REUSE-first: the live schema already carries the columns this integration
-- needs, so this migration is intentionally minimal and idempotent:
--   * billing.invoices.provider_invoice_id (varchar 255)  -> NOWPayments invoice id
--   * billing.invoices.provider_hosted_url (text)         -> hosted payment URL
--   * billing.payments.provider / provider_payment_id / provider_event_id
--
-- Nothing here rewrites applied history or mutates financial rows.

BEGIN;

-- Idempotency anchor for gateway callbacks (replays must be no-ops).
-- Matches the pre-existing partial unique index if already applied.
CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_provider_event
  ON billing.payments (provider, provider_event_id)
  WHERE provider_event_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_payments_provider_payment
  ON billing.payments (provider, provider_payment_id)
  WHERE provider_payment_id IS NOT NULL;

COMMIT;
