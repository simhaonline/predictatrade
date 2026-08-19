-- Predict-A-Trade v1.0.0 — Migration 016
-- Fix Stale Instantaneous Operations & Add completed_at Column
--
-- BUG: RESUME_TRADING, RESUME_SIGNALS, and ENABLE_STRATEGY operations were
-- created with status 'ACTIVE' and never transitioned to a terminal state.
-- These are instantaneous actions (not persistent states like HALT_TRADING),
-- so they accumulated as duplicate "Active Operations" in the admin UI.
--
-- This migration:
-- 1. Adds a `completed_at` column for tracking when instantaneous ops complete.
-- 2. Marks all stale ACTIVE RESUME_*/ENABLE_* operations as COMPLETED.
--
-- NON-DESTRUCTIVE: Only adds a nullable column and updates status values.
-- Forward-only; no rollback needed (status COMPLETED is terminal).

-- ============================================================
-- Section 1: Add completed_at column
-- ============================================================
ALTER TABLE control.platform_operations
    ADD COLUMN IF NOT EXISTS completed_at TIMESTAMPTZ;

COMMENT ON COLUMN control.platform_operations.completed_at IS
    'Timestamp when an instantaneous operation (RESUME_*, ENABLE_*) completed';

-- ============================================================
-- Section 2: Clean up stale instantaneous operations
-- ============================================================
-- Mark all RESUME_TRADING, RESUME_SIGNALS, ENABLE_STRATEGY records that are
-- still ACTIVE as COMPLETED. These should never persist as ACTIVE.
UPDATE control.platform_operations
SET status = 'COMPLETED', completed_at = now()
WHERE operation_type IN ('RESUME_TRADING', 'RESUME_SIGNALS', 'ENABLE_STRATEGY')
  AND status = 'ACTIVE';

-- Update the status comment to include COMPLETED
COMMENT ON COLUMN control.platform_operations.status IS
    'ACTIVE, REVERTED, EXPIRED, COMPLETED — instantaneous ops use COMPLETED';
