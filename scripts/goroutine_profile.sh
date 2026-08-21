#!/usr/bin/env bash
set -euo pipefail
PROJECT_ROOT="/srv/predictatrade/xauusd"
LOG_DIR="${PROJECT_ROOT}/logs"
mkdir -p "${LOG_DIR}"

GOROUTINES=$(curl -s "http://127.0.0.1:13081/metrics" 2>/dev/null | grep "^go_goroutines" | awk '{print $2}' | head -1)
TIMESTAMP=$(date +%s)
echo "{\"timestamp\":\"$(date -Iseconds)\",\"goroutines\":${GOROUTINES:-0}}" >> "${LOG_DIR}/goroutine_history.jsonl"

if [ "${GOROUTINES:-0}" -gt 500 ]; then
  echo "[$(date)] WARNING: ${GOROUTINES} goroutines (threshold: 500)" >> "${LOG_DIR}/goroutine_alerts.log"
fi
echo "Goroutines: ${GOROUTINES:-0}"
