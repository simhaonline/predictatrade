-- Migration 144: seed an example coupon so the admin Coupons tab is populated
-- with a real, usable row (not a fabricated placeholder). Idempotent: only
-- inserts if the table is empty. Adjust or remove the sample as needed.
INSERT INTO billing.coupons (code, description, discount_type, discount_value, currency, max_redemptions, redemption_count, active, valid_from, valid_until, created_at)
SELECT 'WELCOME10', 'Welcome discount — 10% off the first billing cycle', 'PERCENTAGE', 10.0, 'USD', 500, 0, TRUE, now(), now() + interval '1 year', now()
WHERE NOT EXISTS (SELECT 1 FROM billing.coupons);
