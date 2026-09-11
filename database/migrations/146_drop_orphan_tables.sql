-- 146_drop_orphan_tables.sql
-- Production DB hygiene: drop tables that are orphaned (no application code
-- reads or writes them; only their original CREATE migration references them)
-- and that hold zero rows. Verified by grep across realtime/control/frontend and
-- by row-count + FK inspection on the live database.
--
--   * licensing.client_releases        — Windows Agent binary registry; the
--                                         /admin/releases feature was removed.
--   * licensing.download_events        — FK child of client_releases (release_id);
--                                         orphaned, zero rows.
--   * trading.backtest_fold_results    — walk-forward fold outcomes; referenced only
--                                         by migration 015, never written at runtime.
--   * calibration.model_versions       — model registry rows; not used by the live
--                                         Go calibrator (it uses calibration_profiles/
--                                         calibration_reports). Only offline research scripts.
--   * calibration.predictions          — FK child of model_versions; orphaned.
--   * calibration.outcomes            — orphaned; zero rows.
--
-- Drop order respects FK dependencies (children first). All guarded with
-- IF EXISTS so the migration is safe to re-run.

DROP TABLE IF EXISTS licensing.download_events;
DROP TABLE IF EXISTS licensing.client_releases;
DROP TABLE IF EXISTS calibration.predictions;
DROP TABLE IF EXISTS calibration.model_versions;
DROP TABLE IF EXISTS calibration.outcomes;
DROP TABLE IF EXISTS trading.backtest_fold_results;

-- Ephemeral funnel data: anonymous preview trials are intentionally short-lived.
-- Remove only rows that are already expired (status='expired' or past
-- trial_expires_at). Idempotent — a no-op on a fresh DB.
DELETE FROM live_preview.anonymous_trials
WHERE status = 'expired' OR trial_expires_at < now();
