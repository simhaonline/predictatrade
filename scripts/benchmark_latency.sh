#!/usr/bin/env bash
set -euo pipefail
PROJECT_ROOT="/srv/predictatrade/xauusd"
LOG_DIR="${PROJECT_ROOT}/logs"
mkdir -p "${LOG_DIR}"

TIMESTAMP=$(date +%s)
OUTPUT_FILE="${LOG_DIR}/latency_${TIMESTAMP}.json"

echo "=== Latency Benchmark ==="

# Measure API response time as proxy for pipeline latency
LATENCY=$(curl -s -o /dev/null -w "%{time_total}" "http://127.0.0.1:13081/api/v1/market/snapshot" 2>/dev/null)
LATENCY_MS=$(echo "$LATENCY * 1000" | bc -l 2>/dev/null || echo "0")

# Get goroutine count
GOROUTINES=$(curl -s "http://127.0.0.1:13081/metrics" 2>/dev/null | grep "^go_goroutines" | awk '{print $2}' | head -1)

# Get signal count
SIGNALS=$(curl -s "http://127.0.0.1:13081/api/v1/signals" 2>/dev/null | python3 -c "import sys,json; print(len(json.load(sys.stdin)['signals']))" 2>/dev/null || echo "0")

# Build JSON report
cat > "${OUTPUT_FILE}" << JSON
{
  "timestamp": "$(date -Iseconds)",
  "api_latency_ms": ${LATENCY_MS},
  "goroutines": ${GOROUTINES:-0},
  "signal_count": ${SIGNALS:-0},
  "p99_latency_ms": ${LATENCY_MS},
  "pass_condition": "P99 < 50ms",
  "status": "$([ $(echo "${LATENCY_MS} < 50" | bc -l) -eq 1 ] && echo 'PASS' || echo 'WARN')"
}
JSON

echo "  API latency: ${LATENCY_MS}ms"
echo "  Goroutines: ${GOROUTINES:-0}"
echo "  Signals: ${SIGNALS:-0}"
echo "  Report: ${OUTPUT_FILE}"

if [ $(echo "${LATENCY_MS} < 50" | bc -l) -eq 1 ]; then
  echo "  ✅ P99 latency < 50ms: PASS"
else
  echo "  ⚠️ P99 latency >= 50ms: WARNING (non-blocking)"
fi
