-- Phase 0.5 alert queries (Candidate B) — drop into watchdog/monitoring.
-- Each query returns alert rows ONLY when the condition is breached; empty result = healthy.

\echo '=== ALERT 1: outcome writer silent >30min while signals emit ==='
SELECT 'OUTCOME_WRITER_SILENT' AS alert,
       EXTRACT(EPOCH FROM (now() - (SELECT max(created_at) FROM trading.prediction_outcomes)))/60 AS minutes_silent,
       (SELECT count(*) FROM trading.signals WHERE created_at > now() - interval '30 minutes') AS signals_last_30m
WHERE (SELECT count(*) FROM trading.signals WHERE created_at > now() - interval '30 minutes') > 0
  AND (SELECT coalesce(max(created_at), '1970-01-01'::timestamptz) FROM trading.prediction_outcomes)
      < now() - interval '30 minutes';

\echo '=== ALERT 2: UNLINKED ratio > 20% (last 100 outcomes) ==='
SELECT 'UNLINKED_RATIO_HIGH' AS alert,
       round(100.0 * count(*) FILTER (WHERE link_status='UNLINKED') / NULLIF(count(*),0), 1) AS unlinked_pct
FROM (
  SELECT link_status FROM trading.prediction_outcomes
  ORDER BY created_at DESC LIMIT 100
) recent
HAVING round(100.0 * count(*) FILTER (WHERE link_status='UNLINKED') / NULLIF(count(*),0), 1) > 20;

\echo '=== ALERT 3: predictions exist but outcomes missing for >1h old predictions ==='
SELECT 'OUTCOMES_MISSING' AS alert,
       count(*) AS predictions_without_outcome
FROM trading.predictions p
WHERE p.created_at < now() - interval '1 hour'
  AND NOT EXISTS (SELECT 1 FROM trading.prediction_outcomes o WHERE o.signal_id = p.signal_id)
  AND p.created_at > now() - interval '24 hours';

\echo '=== ALERT 4: shadow resolver stalled (no resolutions in 24h while market open) ==='
SELECT 'SHADOW_RESOLVER_STALLED' AS alert,
       count(*) FILTER (WHERE outcome='UNRESOLVED' AND timestamp < now() - interval '24 hours') AS stale_unresolved
FROM trading.cross_market_shadow_snapshots
HAVING count(*) FILTER (WHERE outcome='UNRESOLVED' AND timestamp < now() - interval '24 hours') > 50;
