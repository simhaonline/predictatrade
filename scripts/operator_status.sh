#!/usr/bin/env bash
# operator_status.sh — one-command operator status for Predict-A-Trade.
# Phase 0.9 Task A.3. Prints: engine health, EA presence by client, signal
# enqueue/expiry rates, TRADE_RESULT rate, outcomes written last 24h,
# per-strategy N (sufficiency-gate context). Read-only.
set -u
PSQL="docker exec pat-postgres psql -U pat_admin -d predictatrade -tAc"
API="http://localhost:13081"

hr() { printf '\n========== %s ==========\n' "$1"; }

hr "ENGINE HEALTH"
curl -s --max-time 5 "$API/health" | head -c 400; echo

hr "REALTIME AGENT STATUS (EA presence)"
curl -s --max-time 5 "$API/api/v1/agents/status" | python3 -c '
import json,sys
try:
    d = json.load(sys.stdin)
    for k in ("agents_connected","agents_online","mt4_connected","mt5_connected",
              "edge_devices_online","data_health","last_snapshot_at"):
        print(f"  {k}: {d.get(k)}")
except Exception as e:
    print("parse error:", e)'

hr "EDGE DEVICES BY ROLE (licensing.devices; exec devices must be ONLINE)"
$PSQL "SELECT connection_status, role, count(*) FROM licensing.devices WHERE role IN ('exec','master','data') OR role IS NULL GROUP BY 1,2 ORDER BY 3 DESC;"

hr "STALE EXEC DEVICES (no ingest liveness >15min — the EAs that must attach)"
$PSQL "SELECT id, device_name, connection_status, to_char(last_seen_at,'MM-DD HH24:MI') FROM licensing.devices WHERE role='exec' AND (last_seen_at IS NULL OR last_seen_at < now() - interval '15 minutes') ORDER BY last_seen_at DESC NULLS FIRST LIMIT 10;" || echo "  (licensing.devices query unavailable)"

hr "SIGNAL ENQUEUE / EXPIRY / ACK (last 24h)"
$PSQL "SELECT 'enqueued', count(*) FROM licensing.edge_signal_queue WHERE created_at > now() - interval '24 hours'
UNION ALL SELECT 'acked', count(*) FROM licensing.edge_signal_queue WHERE acked_at > now() - interval '24 hours'
UNION ALL SELECT 'pending', count(*) FROM licensing.edge_signal_queue WHERE status='PENDING'
UNION ALL SELECT 'in_flight', count(*) FROM licensing.edge_signal_queue WHERE status='IN_FLIGHT';"

hr "TRADE_RESULT RATE (last 24h)"
$PSQL "SELECT count(*) FROM trading.trade_results WHERE created_at > now() - interval '24 hours';"

hr "OUTCOMES WRITTEN LAST 24H (execution channel)"
$PSQL "SELECT count(*), count(*) FILTER (WHERE link_status='LINKED') FROM trading.prediction_outcomes WHERE created_at > now() - interval '24 hours';"

hr "PER-STRATEGY N + GATE (execution channel, LINKED; target 300)"
$PSQL "SELECT strategy_id, count(*) FILTER (WHERE link_status='LINKED') AS n,
       CASE WHEN count(*) FILTER (WHERE link_status='LINKED') >= 300 THEN 'PASS' ELSE 'BLOCKED' END
FROM trading.prediction_outcomes WHERE strategy_id <> '' GROUP BY 1 ORDER BY 2 DESC;"

hr "UNLINKED RATIO (last 100 outcomes; alert >20%)"
$PSQL "SELECT round(100.0*count(*) FILTER (WHERE link_status='UNLINKED')/NULLIF(count(*),0),1) FROM (SELECT link_status FROM trading.prediction_outcomes ORDER BY created_at DESC LIMIT 100) t;"

hr "WRITER SILENCE (minutes since last outcome; alert >30 while signals emit)"
$PSQL "SELECT round(EXTRACT(EPOCH FROM (now()-max(created_at)))/60,1) FROM trading.prediction_outcomes;"

echo
echo "Legend: exec-device rows must be ONLINE for TRADE_RESULTs to flow."
echo "Gate: Phase 1 tuning needs >=300 LINKED execution outcomes per strategy."