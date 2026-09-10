-- Devil Liquidity: promote default operational mode to PRODUCTION LIVE and add
-- bounded contribution controls.
--
-- Migration 080 seeded devil_liquidity_config with mode='shadow' (observation
-- only). The engine detected and persisted Devil's Marks but was never wired into
-- the live signal pipeline, so it ran effectively as a shadow observer. We now
-- make the engine a bounded, fail-closed contributing intelligence (mode =
-- 'confluence') that nudges the existing signal score within [max_penalty,
-- max_bonus] and never triggers or blocks a trade on its own (risk gates remain
-- authoritative). Operators can still downgrade to 'shadow' or 'disabled' at
-- runtime via the /admin API / devil_liquidity_config row.
--
-- Conservative day-one bounds: max_bonus = +5.0, max_penalty = -5.0 (half the
-- cross-market magnitude) so Devil Liquidity can only ever fine-tune, never
-- dominate, the live signal while we watch its behavior in production.
--
-- This migration is idempotent: it only flips shadow/empty rows to confluence,
-- adds the bound columns if missing, and leaves 'disabled' rows alone.

-- 1) Add bounded-contribution columns (idempotent).
ALTER TABLE devil_liquidity_config
    ADD COLUMN IF NOT EXISTS max_bonus   DOUBLE PRECISION DEFAULT 5.0,
    ADD COLUMN IF NOT EXISTS max_penalty DOUBLE PRECISION DEFAULT -5.0;

-- 2) Promote shadow/empty rows to production-live with conservative bounds.
UPDATE devil_liquidity_config
SET
    mode = 'confluence',
    max_bonus = COALESCE(max_bonus, 5.0),
    max_penalty = COALESCE(max_penalty, -5.0),
    min_signal_score = COALESCE(min_signal_score, 60.0),
    updated_by = 'migration_143',
    updated_at = NOW(),
    version = COALESCE(version, '1.0.0')
WHERE mode IS NULL OR mode = '' OR mode = 'shadow';

-- 3) If the 080 seed never ran (e.g. fresh install), create the production-live
--    default now (confluence + conservative bounds).
INSERT INTO devil_liquidity_config (
    enabled, mode, max_bonus, max_penalty,
    flat_wick_ratio, flat_wick_atr_tol, minimum_tick_tol,
    minimum_body_ratio, minimum_range_atr, minimum_body_exp, close_extreme_ratio,
    approach_distance_atr, minimum_sweep_depth_atr, maximum_sweep_depth_atr,
    reclaim_max_bars, reversal_body_ratio, mark_expiry_bars, min_mark_quality,
    min_signal_score, news_handling, updated_by, version
) SELECT
    TRUE, 'confluence', 5.0, -5.0, 0.03, 0.15, 2,
    0.70, 1.20, 1.50, 0.15,
    3.0, 0.20, 2.5,
    10, 0.50, 240, 40.0,
    60.0, 'analyze_separately', 'migration_143', '1.0.0'
WHERE NOT EXISTS (SELECT 1 FROM devil_liquidity_config);
