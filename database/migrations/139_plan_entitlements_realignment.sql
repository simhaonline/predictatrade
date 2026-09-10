-- Migration 139: Re-align plan_entitlements to MASTER PROMPT spec (migration 110).
--
-- Migration 024 seeded the control.plan_entitlements display rows BEFORE
-- migration 110 realigned control.plans to the MASTER PROMPT tier spec. The
-- entitlement rows were never re-aligned, so the admin "Plans & Entitlements"
-- page (and any consumer of these keys) shows values that contradict the
-- authoritative plans-table columns the engine actually enforces:
--
--   plan      plans-table slots (110)   entitlement rows said
--   FREE      1 slot, 5 signals/day     1 slot, 3/day
--   STANDARD  2 slots                   1 slot
--   PRO       4 slots                   2 slots
--   ELITE     6 slots                   4-5 slots
--
-- Also restores ELITE api.access=true (seeded true in 003; 024 flipped it to
-- false — a downgrade below PRO, breaking ELITE ⊇ PRO monotonicity).
--
-- Idempotent: keyed on plan code + entitlement key. Only rows that disagree
-- with migration 110 are touched.

-- ── Slot caps: mirror control.plans.max_active_strategy_slots ──
-- FREE
UPDATE control.plan_entitlements pe
SET entitlement_value = '1'::jsonb
FROM control.plans p
WHERE pe.plan_id = p.id
  AND p.code = 'FREE'
  AND pe.entitlement_key IN ('strategy.max_active_slots', 'max_active_strategies')
  AND pe.entitlement_value::text <> '1';

-- STANDARD
UPDATE control.plan_entitlements pe
SET entitlement_value = '2'::jsonb
FROM control.plans p
WHERE pe.plan_id = p.id
  AND p.code = 'STANDARD'
  AND pe.entitlement_key IN ('strategy.max_active_slots', 'max_active_strategies')
  AND pe.entitlement_value::text <> '2';

-- PRO
UPDATE control.plan_entitlements pe
SET entitlement_value = '4'::jsonb
FROM control.plans p
WHERE pe.plan_id = p.id
  AND p.code = 'PRO'
  AND pe.entitlement_key IN ('strategy.max_active_slots', 'max_active_strategies')
  AND pe.entitlement_value::text <> '4';

-- ELITE
UPDATE control.plan_entitlements pe
SET entitlement_value = '6'::jsonb
FROM control.plans p
WHERE pe.plan_id = p.id
  AND p.code = 'ELITE'
  AND pe.entitlement_key IN ('strategy.max_active_slots', 'max_active_strategies')
  AND pe.entitlement_value::text <> '6';

-- ── FREE daily cap: 5/day per MASTER PROMPT spec (110) ──
UPDATE control.plan_entitlements pe
SET entitlement_value = '5'::jsonb
FROM control.plans p
WHERE pe.plan_id = p.id
  AND p.code = 'FREE'
  AND pe.entitlement_key = 'max_signals_per_day'
  AND pe.entitlement_value::text <> '5';

-- ── ELITE api.access: restore true (003 seeded true; 024 regressed it) ──
UPDATE control.plan_entitlements pe
SET entitlement_value = 'true'::jsonb
FROM control.plans p
WHERE pe.plan_id = p.id
  AND p.code = 'ELITE'
  AND pe.entitlement_key = 'api.access'
  AND pe.entitlement_value::text <> 'true';

-- ── Sanity: entitlement slot values must equal the plans-table columns ──
DO $$
DECLARE
  bad int;
BEGIN
  SELECT count(*) INTO bad
  FROM control.plan_entitlements pe
  JOIN control.plans p ON p.id = pe.plan_id
  WHERE pe.entitlement_key IN ('strategy.max_active_slots', 'max_active_strategies')
    AND (
      (p.code = 'FREE'    AND pe.entitlement_value::text <> '1')
   OR (p.code = 'STANDARD' AND pe.entitlement_value::text <> '2')
   OR (p.code = 'PRO'      AND pe.entitlement_value::text <> '4')
   OR (p.code = 'ELITE'    AND pe.entitlement_value::text <> '6')
    );
  IF bad > 0 THEN
    RAISE EXCEPTION 'migration 139: % plan_entitlements rows still disagree with plans-table slots', bad;
  END IF;
END $$;