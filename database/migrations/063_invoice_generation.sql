-- 063: Invoice generation support
-- The billing.invoices / invoice_items / payments tables already exist (migration 003).
-- This migration adds a monotonic sequence used to build human-readable invoice numbers
-- and a small configuration row for default tax handling (kept at 0 until a jurisdiction
-- policy is configured — no fabricated tax is ever applied).

CREATE SEQUENCE IF NOT EXISTS billing.invoice_seq START WITH 1000;

CREATE TABLE IF NOT EXISTS billing.invoice_settings (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    default_tax_rate DECIMAL(6,4) NOT NULL DEFAULT 0,
    tax_label       VARCHAR(40) NOT NULL DEFAULT 'Tax',
    brand_name      VARCHAR(100) NOT NULL DEFAULT 'Predict-A-Trade',
    currency        VARCHAR(3) NOT NULL DEFAULT 'USD',
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO billing.invoice_settings (brand_name, currency)
VALUES ('Predict-A-Trade', 'USD')
ON CONFLICT DO NOTHING;
