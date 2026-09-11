-- Migration 148: add missing volume_weight column to devil_liquidity_config.
--
-- Root cause (2026-09-11): realtime/internal/devilliquidity/store.go LoadConfig()
-- selects volume_weight (Go default 10.0 — signal-score weighting of the volume
-- pillar), but migration 080 created devil_liquidity_config WITHOUT that column
-- and 143 only added max_bonus/max_penalty. Every engine start logged
--   devilliquidity load config: ERROR: column "volume_weight" does not exist
--   (SQLSTATE 42703)
-- and silently fell back to in-code defaults, so any DB-side config tuning
-- (mode=confluence, enabled=true) never reached the live engine.
--
-- Additive + idempotent; default matches DefaultConfig().VolumeWeight.

ALTER TABLE devil_liquidity_config
    ADD COLUMN IF NOT EXISTS volume_weight DOUBLE PRECISION DEFAULT 10.0;

-- Re-align the stored config row version so the load path is traceable.
UPDATE devil_liquidity_config
SET version = '1.1.0',
    updated_by = 'migration_148',
    updated_at = NOW()
WHERE version = '1.0.0';