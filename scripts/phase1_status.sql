-- phase1_status.sql — one query set returning: gate status per strategy,
-- N by channel, time-to-300 estimate, harness last dry-run result reference.
-- Companion to scripts/phase1_run.py (prompt.md Phase 0.9 Task C.4).

\echo '=== GATE + N BY CHANNEL + TIME-TO-300 (per strategy) ==='
WITH exec_n AS (
  SELECT strategy_id, count(*) FILTER (WHERE link_status='LINKED') AS n
  FROM trading.prediction_outcomes
  WHERE strategy_id IS NOT NULL AND strategy_id <> ''
  GROUP BY 1
),
exec_rate AS (
  SELECT strategy_id, count(*) AS outcomes_7d
  FROM trading.prediction_outcomes
  WHERE created_at > now() - interval '7 days' AND strategy_id IS NOT NULL AND strategy_id <> ''
  GROUP BY 1
),
shadow_n AS (
  SELECT strategy, count(*) FILTER (WHERE outcome IN ('TP1_HIT','TP2_HIT','TP3_HIT','SL_HIT')) AS n
  FROM trading.cross_market_shadow_snapshots
  GROUP BY 1
)
SELECT
  e.strategy_id,
  e.n                                        AS exec_n,
  CASE WHEN e.n >= 300 THEN 'PASS' ELSE 'BLOCKED' END AS gate,
  s.n                                        AS shadow_n_separate,
  GREATEST(round((300 - e.n)::numeric / GREATEST(r.outcomes_7d,1) * 7 / 7, 1), 0) AS est_weeks_to_300
FROM exec_n e
LEFT JOIN shadow_n s ON s.strategy = e.strategy_id
LEFT JOIN exec_rate r ON r.strategy_id = e.strategy_id
ORDER BY e.n DESC;

\echo '=== HARNESS LAST DRY-RUN (reads the output artifact written by phase1_run.py) ==='
-- phase1_run.py writes JSON to docs/strategy/PHASE1_DRYRUN_RESULT.json when --output given.
-- This query surfaces the file state from the DB side is not possible; the runbook
-- instructs operators to check the JSON file directly:
--   cat docs/strategy/PHASE1_DRYRUN_RESULT.json | python3 -m json.tool | head -30
SELECT 'see docs/strategy/PHASE1_DRYRUN_RESULT.json (phase1_run.py --output)' AS harness_hint;