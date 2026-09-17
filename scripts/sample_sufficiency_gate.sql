-- Phase 0.75 Task C: sample-sufficiency gate (blocks Phase 1 tuning).
-- Pre-registered in docs/strategy/PHASE1_CALIBRATION_PROTOCOL.md §7.
-- EXECUTION channel = trading.prediction_outcomes (real EA TRADE_RESULTs).
-- SHADOW channel = trading.cross_market_shadow_snapshots (price-resolved
-- candidates; NEVER counted toward the execution gate; labeled SHADOW always).
-- Run: docker exec -i pat-postgres psql -U pat_admin -d predictatrade < scripts/sample_sufficiency_gate.sql

\echo '=== SUFFICIENCY GATE (execution channel, LINKED only; gate: n >= 300) ==='
SELECT strategy_id,
       count(*) FILTER (WHERE link_status='LINKED')                          AS exec_n_linked,
       count(*) FILTER (WHERE link_status='LINKED' AND r_multiple IS NOT NULL) AS exec_n_with_r,
       round(avg(r_multiple) FILTER (WHERE link_status='LINKED')::numeric, 3)  AS exec_avg_r,
       CASE WHEN count(*) FILTER (WHERE link_status='LINKED') >= 300
            THEN 'PASS' ELSE 'BLOCKED' END                                    AS gate_status,
       round((300 - count(*) FILTER (WHERE link_status='LINKED'))
             / GREATEST(count(*) FILTER (WHERE created_at > now() - interval '7 days'), 1) / 7.0, 1)
                                                                             AS est_weeks_to_300
FROM trading.prediction_outcomes
WHERE strategy_id IS NOT NULL AND strategy_id <> ''
GROUP BY strategy_id
ORDER BY exec_n_linked DESC;

\echo '=== SHADOW channel (reported separately; NOT counted toward the gate) ==='
SELECT strategy,
       count(*) FILTER (WHERE outcome IN ('TP1_HIT','TP2_HIT','TP3_HIT','SL_HIT')) AS shadow_resolved_n,
       round(avg(r_multiple) FILTER (WHERE outcome IN ('TP1_HIT','TP2_HIT','TP3_HIT','SL_HIT'))::numeric, 3) AS shadow_avg_r,
       round(100.0*count(*) FILTER (WHERE COALESCE(signal_id,'') <> '') / NULLIF(count(*),0), 1) AS signal_linked_pct
FROM trading.cross_market_shadow_snapshots
GROUP BY strategy
ORDER BY shadow_resolved_n DESC;

\echo '=== EMISSION RATE (last 7d, per strategy; execution channel context) ==='
SELECT strategy_id,
       count(*) FILTER (WHERE created_at > now() - interval '24 hours') AS outcomes_24h,
       count(*) FILTER (WHERE created_at > now() - interval '7 days')  AS outcomes_7d
FROM trading.prediction_outcomes
WHERE strategy_id IS NOT NULL AND strategy_id <> ''
GROUP BY strategy_id
ORDER BY outcomes_7d DESC;

\echo '=== OVERALL VERDICT (single row; Phase 1 tuning allowed only when PASS) ==='
SELECT CASE WHEN count(*) FILTER (WHERE gate_pass) >= 1 AND count(*) FILTER (WHERE NOT gate_pass) = 0
            THEN 'PASS — Phase 1 may begin on gated strategies'
            ELSE 'BLOCKED — collection continues (fail-closed: no tuning)'
       END AS phase1_gate
FROM (
  SELECT count(*) FILTER (WHERE link_status='LINKED') >= 300 AS gate_pass
  FROM trading.prediction_outcomes
  WHERE strategy_id IS NOT NULL AND strategy_id <> ''
  GROUP BY strategy_id
) g;