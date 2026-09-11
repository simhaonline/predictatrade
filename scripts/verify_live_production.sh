#!/usr/bin/env bash
set -euo pipefail
PROJECT_ROOT="/srv/predictatrade/xauusd"
cd "${PROJECT_ROOT}"
FAILED=0

echo "========================================"
echo "PREDICT-A-TRADE PRODUCTION VERIFICATION"
echo "========================================"

# 1. Latency Check
LATENCY=$(curl -s -o /dev/null -w "%{time_total}" "http://127.0.0.1:13081/api/v1/market/snapshot" 2>/dev/null)
LATENCY_MS=$(echo "$LATENCY * 1000" | bc -l 2>/dev/null || echo "999")
if [ $(echo "${LATENCY_MS} < 50" | bc -l) -eq 1 ]; then
  echo "✅ Latency: ${LATENCY_MS}ms"
else
  echo "⚠️ Latency: ${LATENCY_MS}ms (warning, non-blocking)"
fi

# 2. Stale Data Check (accepts ISO-8601, the actual snapshot format)
SNAP_TS=$(curl -s "http://127.0.0.1:13081/api/v1/market/snapshot" 2>/dev/null | python3 -c "
import sys,json
from datetime import datetime, timezone
try:
    d=json.load(sys.stdin)
    ts=d.get('timestamp','') or d.get('server_time','')
    if ts:
        # ISO-8601 (e.g. 2026-09-11T09:25:09Z) — strip trailing Z and parse
        t=datetime.fromisoformat(ts.replace('Z','+00:00'))
        age=(datetime.now(timezone.utc)-t).total_seconds()
        print(int(age))
    else:
        print(999)
except Exception:
    print(999)
" 2>/dev/null || echo "999")

if [ "${SNAP_TS}" -lt 60 ] 2>/dev/null; then
  echo "✅ Fresh data: ${SNAP_TS}s ago"
else
  echo "❌ Stale data: ${SNAP_TS}s ago"
  FAILED=1
fi

# 3. Math Parity (skip gracefully if numpy unavailable in this env)
if python3 -c "import numpy" 2>/dev/null; then
  if python3 scripts/verify_math_parity.py --samples 100 --threshold 0.0001 2>/dev/null; then
    echo "✅ Math parity: PASS"
  else
    echo "❌ Math parity: FAIL"
    FAILED=1
  fi
else
  echo "⚠️ Math parity: SKIPPED (numpy not installed in this env — non-blocking)"
fi

# 4. Geometry Validation (check last 10 signals)
GEOM_OK=$(curl -s "http://127.0.0.1:13081/api/v1/signals" 2>/dev/null | python3 -c "
import sys,json
d=json.load(sys.stdin)
signals=[s for s in d['signals'] if s.get('Direction')!='NO-TRADE'][:10]
if not signals:
    print('SKIP')
    sys.exit(0)
ok=True
for s in signals:
    entry=float(s.get('EntryPrice','0') or 0)
    sl=float(s.get('StopLoss','0') or 0)
    tp1=float(s.get('TP1','0') or 0)
    tp2=float(s.get('TP2','0') or 0)
    tp3=float(s.get('TP3','0') or 0)
    direction=s.get('Direction','')
    if entry==0 or sl==0:
        continue
    if 'BUY' in direction:
        if sl>=entry: ok=False; break
        if tp1>0 and tp1<=entry: ok=False; break
        if tp2>0 and tp2<=tp1: ok=False; break
        if tp3>0 and tp3<=tp2: ok=False; break
    elif 'SELL' in direction:
        if sl<=entry: ok=False; break
        if tp1>0 and tp1>=entry: ok=False; break
        if tp2>0 and tp2>=tp1: ok=False; break
        if tp3>0 and tp3>=tp2: ok=False; break
print('PASS' if ok else 'FAIL')
" 2>/dev/null || echo "FAIL")

if [ "${GEOM_OK}" = "PASS" ] || [ "${GEOM_OK}" = "SKIP" ]; then
  echo "✅ Geometry validation: ${GEOM_OK}"
else
  echo "❌ Geometry validation: FAIL"
  FAILED=1
fi

# 5. Signal Flow Rate
SIGNAL_COUNT=$(curl -s "http://127.0.0.1:13081/api/v1/signals" 2>/dev/null | python3 -c "
import sys,json
d=json.load(sys.stdin)
directional=[s for s in d['signals'] if s.get('Direction')!='NO-TRADE']
print(len(directional))
" 2>/dev/null || echo "0")

if [ "${SIGNAL_COUNT}" -gt 0 ]; then
  echo "✅ Signal flow: ${SIGNAL_COUNT} directional signals"
else
  echo "❌ Blockage detected: 0 directional signals"
  FAILED=1
fi

# 6. Goroutine Leak Check
GOROUTINES=$(curl -s "http://127.0.0.1:13081/metrics" 2>/dev/null | grep "^go_goroutines" | awk '{print $2}' | head -1)
if [ "${GOROUTINES:-0}" -gt 600 ]; then
  echo "⚠️ Warning: ${GOROUTINES} goroutines (monitor for leaks)"
else
  echo "✅ Goroutines: ${GOROUTINES:-0}"
fi

# 7. ML Pipeline
ML_ENABLED=$(grep -c '^ML_ENABLED=true' infra/env/realtime.env 2>/dev/null || echo 0)
if [ "${ML_ENABLED}" -eq 1 ]; then
  echo "✅ ML_ENABLED: true"
else
  echo "⚠️ ML_ENABLED: not set"
fi

# 8. Ollama
OLLAMA=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:11434/api/tags" 2>/dev/null || echo "000")
if [ "${OLLAMA}" = "200" ]; then
  echo "✅ Ollama: connected"
else
  echo "⚠️ Ollama: ${OLLAMA} (non-blocking)"
fi

# 9. COT + DXY (docker-compose: read from the realtime container logs; fall back to systemd journal)
COT_STATUS=0; DXY_STATUS=0
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^pat-realtime$'; then
  COT_STATUS=$(docker logs --tail 30 pat-realtime 2>/dev/null | grep -c "COT.*AVAILABLE" || true)
  DXY_STATUS=$(docker logs --tail 30 pat-realtime 2>/dev/null | grep -c "DXY.*AVAILABLE" || true)
elif command -v journalctl >/dev/null 2>&1; then
  COT_STATUS=$(journalctl -u predictatrade-realtime -n 30 --no-pager -o cat 2>/dev/null | grep -c "COT.*AVAILABLE" || true)
  DXY_STATUS=$(journalctl -u predictatrade-realtime -n 30 --no-pager -o cat 2>/dev/null | grep -c "DXY.*AVAILABLE" || true)
fi
COT_STATUS=${COT_STATUS:-0}; DXY_STATUS=${DXY_STATUS:-0}
if [ "${COT_STATUS}" -gt 0 ]; then
  echo "✅ COT: available"
else
  echo "⚠️ COT: not found in recent logs"
fi
if [ "${DXY_STATUS}" -gt 0 ]; then
  echo "✅ DXY: available"
else
  echo "⚠️ DXY: not found in recent logs"
fi

# 10. Engine Status (docker-compose first; fall back to systemd)
ENGINE="unknown"
if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^pat-realtime$'; then
  # docker-compose deployment (canonical): check the container is running + healthy
  ENGINE_STATE=$(docker inspect -f '{{.State.Status}}' pat-realtime 2>/dev/null || echo "missing")
  ENGINE_HEALTH=$(docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}na{{end}}' pat-realtime 2>/dev/null || echo "na")
  if [ "${ENGINE_STATE}" = "running" ]; then
    ENGINE="active"
    echo "✅ Engine: active (docker container 'pat-realtime', state=${ENGINE_STATE}, health=${ENGINE_HEALTH})"
  else
    ENGINE="${ENGINE_STATE}"
    echo "❌ Engine: docker container 'pat-realtime' state=${ENGINE_STATE}"
    FAILED=1
  fi
elif command -v systemctl >/dev/null 2>&1; then
  ENGINE=$(systemctl is-active predictatrade-realtime 2>/dev/null || echo "unknown")
  if [ "${ENGINE}" = "active" ]; then
    echo "✅ Engine: active (systemd)"
  else
    echo "❌ Engine: ${ENGINE}"
    FAILED=1
  fi
else
  echo "⚠️ Engine: cannot determine status (no docker/systemctl) — non-blocking"
fi

# 11. Cloudflare edge posture (non-blocking report)
# Runs the dedicated Cloudflare verifier if present. Failures here are reported
# but do NOT flip the overall status to BLOCKED — the origin checks above are the
# hard gate; the edge check is advisory (proxy DNS may still be propagating).
if [ -x "${PROJECT_ROOT}/scripts/verify-cloudflare.sh" ]; then
  echo "── Cloudflare edge check ─────────────────────"
  if "${PROJECT_ROOT}/scripts/verify-cloudflare.sh" --quiet 2>&1 | sed 's/^/   /'; then
    echo "✅ Cloudflare edge: verification passed"
  else
    echo "⚠️ Cloudflare edge: see details above (non-blocking)"
  fi
else
  echo "⚠️ Cloudflare edge: verify-cloudflare.sh not found (non-blocking)"
fi

# FINAL DECISION
if [ ${FAILED} -eq 0 ]; then
  echo "========================================"
  echo "🚀 STATUS: PRODUCTION-LIVE - ALL GREEN"
  echo "========================================"
  exit 0
else
  echo "========================================"
  echo "🔴 STATUS: BLOCKED - FIX FAILED CHECKS"
  echo "========================================"
  exit 1
fi
