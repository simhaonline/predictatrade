-- 070: Remove the legacy BASIC $99 plan everywhere it can surface.
--
-- BASIC was already hidden from new sales by 024 (visible=FALSE, legacy=TRUE,
-- billing_enabled=FALSE). This migration retires it fully so it cannot appear
-- in admin lists, plan lookups, checkout or entitlement decisions:
--   * status -> 'RETIRED' (plans.service listActive filters status='ACTIVE')
--   * renamed to an honest discontinued label (historical invoices/payments
--     keep their own amounts; nothing financial is rewritten)
--   * entitlements and referral rate rows removed so no policy path can
--     resolve BASIC anymore.
--
-- The plan row itself is retained solely for FK integrity of historical
-- subscriptions (never rewrite applied history / never orphan financial rows).

BEGIN;

UPDATE control.plans
SET status = 'RETIRED',
    name = 'Basic (Discontinued)',
    visible = FALSE,
    legacy = TRUE,
    billing_enabled = FALSE,
    updated_at = now()
WHERE code = 'BASIC';

DELETE FROM control.plan_entitlements
WHERE plan_id IN (SELECT id FROM control.plans WHERE code = 'BASIC');

DELETE FROM referral.commission_rules
WHERE plan_id IN (SELECT id FROM control.plans WHERE code = 'BASIC');

DELETE FROM control.plan_entitlement_versions
WHERE plan_id IN (SELECT id FROM control.plans WHERE code = 'BASIC');

COMMIT;
