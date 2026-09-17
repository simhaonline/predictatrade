-- Phase 0.5 metrics pack (Candidate B, prompt.md)
-- Purpose: single source of truth queries for the outcome-capture pipeline health,
-- coverage, and per-strategy calibration readiness. Run with:
--   docker exec -i pat-postgres psql -U pat_admin -d predictatrade < scripts/phase0_5_metrics.sql
-- (Queries only — no DDL, no writes. Safe to run against production.)

\echo '=== 1. PIPELINE HEALTH: writers alive ==='
-- Last prediction written (emit-side writer)
SELECT 'last_prediction' AS metric,
       to_char(max(created_at), 'YYYY-MM-DD HH24:MI') AS value
FROM trading.predictions
UNION ALL
-- Last outcome written (TRADE_RESULT-side writer)
SELECT 'last_outcome',
       to_char(max(created_at), 'YYYY-MM-DD HH24:MI')
FROM trading.prediction_outcomes
UNION ALL
-- Last shadow outcome resolved (independent shadow channel)
SELECT 'last_shadow_resolved',
       to_char(max(resolved_at), 'YYYY-MM-DD HH24:MI')
FROM trading.cross_market_shadow_snapshots
WHERE resolved_at IS NOT NULL;

\echo '=== 2. COVERAGE: signals -> predictions -> outcomes ==='
SELECT
  (SELECT count(*) FROM trading.signals)                                        AS signals_total,
  (SELECT count(*) FROM trading.predictions)                                    AS predictions_total,
  (SELECT count(*) FROM trading.prediction_outcomes)                            AS outcomes_total,
  (SELECT count(*) FROM trading.prediction_outcomes WHERE link_status='LINKED')   AS outcomes_linked,
  (SELECT count(*) FROM trading.prediction_outcomes WHERE link_status='UNLINKED') AS outcomes_unlinked;

\echo '=== 3. UNLINKED RATIO (alert threshold: >20%) ==='
SELECT round(100.0 * count(*) FILTER (WHERE link_status='UNLINKED') / NULLIF(count(*),0), 1) AS unlinked_pct
FROM trading.prediction_outcomes;

\echo '=== 4. PER-STRATEGY CALIBRATION READINESS (n>=300 target) ==='
SELECT strategy_id,
       count(*)                                          AS n,
       round(avg(r_multiple)::numeric, 3)                AS avg_r,
       round(100.0*count(*) FILTER (WHERE outcome_type='WIN') / NULLIF(count(*),0), 1) AS win_pct,
       max(created_at)                                   AS last_outcome_at,
       CASE WHEN count(*) >= 300 THEN 'READY' ELSE 'COLLECTING' END AS calibration_status
FROM trading.prediction_outcomes
WHERE strategy_id IS NOT NULL AND strategy_id <> ''
GROUP BY strategy_id
ORDER BY n DESC;

\echo '=== 5. OUTCOME MIX (close reasons — manual-close awareness) ==='
SELECT close_reason, outcome_type, count(*), round(avg(r_multiple)::numeric,3) AS avg_r
FROM trading.prediction_outcomes
GROUP BY close_reason, outcome_type
ORDER BY count(*) DESC;

\echo '=== 6. SHADOW CHANNEL: resolver coverage + linkability (post-stamping) ==='
SELECT outcome,
       count(*)                                                            AS n,
       round(avg(r_multiple)::numeric, 3)                                  AS avg_r,
       round(100.0*count(*) FILTER (WHERE COALESCE(signal_id,'')<>'') / NULLIF(count(*),0), 1) AS signal_linked_pct
FROM trading.cross_market_shadow_snapshots
GROUP BY outcome
ORDER BY n DESC;

\echo '=== 7. WRITER SILENCE CHECK (alert if >30min while signals emit) ==='
SELECT
  EXTRACT(EPOCH FROM (now() - max(created_at)))/60 AS minutes_since_last_outcome
FROM trading.prediction_outcomes;