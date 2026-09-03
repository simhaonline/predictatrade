-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 132: fleet entitlement fix — operator license STANDARD → PRO
--
-- Context (2026-09-03, v1.26 "fix remaining gap"):
--   mig 131 moved TREND_SWING to PRO+ELITE in engine plan map + control.plans,
--   but the ONLY polling exec devices (Xelans, ADS, Equiti masters) are bound
--   to operator license a3b61096 (PAT-D3D2, plan=STANDARD). The edge-poll
--   enqueue SQL (realtime main.go ~703) filters on BOTH license.allowed_
--   strategies AND plan.allowed_strategies — TREND_SWING was in neither, so
--   every EXECUTABLE TREND_SWING matched 0 devices ("[DELIVERY] executable
--   signal matched 0 devices" 20:05/20:31 UTC).
--
-- Decision:
--   • Operator license moves to PRO plan (billing integrity: owned by
--     admin@predictatrade.com, pre-billing era, NO billing subscription
--     attached — no subscription mutation; plan cap change: daily loss
--     −3%→−5%, per-trade 1.5% unchanged, weekly −5%→−8%).
--   • license.allowed_strategies += TREND_SWING (mirror PRO canonical set).
--   • Capital-tier filter REMAINS the per-signal safety gate (by design):
--     devices whose equity cannot support a signal's min-lot risk still
--     won't receive it. MICRO-tier $8.96 devices need top-up (operator
--     action, separately tracked).
--   • Dry-run verified in rolled-back txn: all 3 exec devices match.
-- ─────────────────────────────────────────────────────────────────────────────

UPDATE licensing.licenses
SET
    plan_id             = (SELECT id FROM control.plans WHERE code = 'PRO' AND status = 'ACTIVE' LIMIT 1),
    allowed_strategies  = '["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"]'::jsonb,
    updated_at          = NOW()
WHERE id = 'a3b61096-90d9-471a-9d84-cdd7d59a947e'
  AND status = 'ACTIVE'
  AND user_id = (SELECT id FROM iam.users WHERE email = 'admin@predictatrade.com');

-- Audit snapshot for the fleet entitlement change.
INSERT INTO trading.strategy_config_versions
    (strategy_id, version, values, effective_from, change_reason)
SELECT 'TREND_SWING', 'v1.26-fleet-entit',
    '{
      "license": "a3b61096 (PAT-D3D2, admin@predictatrade.com operator fleet)",
      "previous_plan": "STANDARD",
      "new_plan": "PRO",
      "license_allowed_strategies": ["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"],
      "affected_devices": ["Xelans Markets Limited", "ADS Securities - LLC - S.P.C", "Equiti Brokerage (Seychelles) Limited"],
      "note": "capital-tier filter remains per-signal safety gate; MICRO-tier devices require equity top-up to receive TREND_SWING"
    }'::jsonb,
    NOW(),
    'v1.26 fleet entitlement fix: operator license STANDARD→PRO (mig 131 made TREND_SWING PRO-eligible; license+plan filter pair still excluded all polling exec devices)'
WHERE NOT EXISTS (
    SELECT 1 FROM trading.strategy_config_versions
    WHERE strategy_id = 'TREND_SWING' AND version = 'v1.26-fleet-entit'
);