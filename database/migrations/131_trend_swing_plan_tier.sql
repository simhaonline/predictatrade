-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 131: TREND_SWING plan-tier move — ELITE-only → PRO + ELITE
--
-- Decision (2026-09-03, v1.26 roster sweep, evidence-based):
--   • Engine plan map was the ONLY gate keeping TREND_SWING out of PRO.
--   • Capital-tier engine ALREADY restricts TREND_SWING to $5k+ accounts
--     (per-trade risk vs tier cap: eligible_tiers excludes MICRO/STANDARD on
--     every promoted candidate — engine logs 19:49-20:05 UTC).
--   • The redundant plan gate delivered the fleet's best-performing strategy
--     (90d parity: +23.6%, PF 1.45) to ZERO devices: the only polling exec
--     devices are STANDARD-plan (PAT-D3D2), the sole ELITE device last
--     polled 2026-08-19 ("matched 0 devices" at 20:05:13).
--   • ELITE keeps exclusive differentiators: MARNIE_FIB, ATEN, ARCANIST.
--   • Free/Standard plans unchanged. Nesting: PRO ⊂ ELITE restored.
-- ─────────────────────────────────────────────────────────────────────────────

UPDATE control.plans
SET
    allowed_strategies = '["STANDARD_SCALPING", "ULTRA_SCALPING", "STANDARD_SWING", "TREND_SWING"]'::jsonb,
    updated_at         = NOW()
WHERE code = 'PRO'
  AND status = 'ACTIVE';

-- Audit snapshot for the plan change.
INSERT INTO trading.strategy_config_versions
    (strategy_id, version, values, effective_from, change_reason)
SELECT 'TREND_SWING', 'v1.26-plan-tier-pro',
    '{
      "previous_plans": ["ELITE"],
      "new_plans": ["PRO", "ELITE"],
      "rationale": "capital-tier gate (min-lot risk vs tier cap) already restricts TREND_SWING to $5k+ accounts; plan gate was redundant risk protection; 90d parity +23.6% PF 1.45 was delivered to 0 devices (STANDARD-plan exec fleet + dormant ELITE device)"
    }'::jsonb,
    NOW(),
    'v1.26 roster decision: TREND_SWING ELITE-only → PRO+ELITE (dead-inventory resolution; ELITE keeps MARNIE_FIB/ATEN/ARCANIST exclusivity)'
WHERE NOT EXISTS (
    SELECT 1 FROM trading.strategy_config_versions
    WHERE strategy_id = 'TREND_SWING' AND version = 'v1.26-plan-tier-pro'
);