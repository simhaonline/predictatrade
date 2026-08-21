#!/usr/bin/env bash
set -euo pipefail

# ─── Predict-A-Trade ML Training Cron Script ─────────────────────────────────
# Sources environment, runs the ML training pipeline, and triggers model reload.
# Designed for cron: e.g. 0 2 * * 0 /srv/predictatrade/xauusd/scripts/run_training.sh
# ──────────────────────────────────────────────────────────────────────────────

PROJECT_ROOT="/srv/predictatrade/xauusd"
ENV_FILE="${PROJECT_ROOT}/infra/env/realtime.env"
LOG_DIR="${PROJECT_ROOT}/logs"
MODELS_DIR="${PROJECT_ROOT}/models"
LOG_FILE="${LOG_DIR}/training_$(date +%Y%m%d).log"

# Source environment
if [ -f "${ENV_FILE}" ]; then
    set -a
    source "${ENV_FILE}"
    set +a
fi

# Ensure directories exist
mkdir -p "${LOG_DIR}" "${MODELS_DIR}"

echo "[$(date)] Starting ML training pipeline..." | tee -a "${LOG_FILE}"

# Calculate date range (2 years ago to today)
START_DATE=$(date --date="2 years ago" +%Y-%m-%d 2>/dev/null || date -v-2y +%Y-%m-%d 2>/dev/null || echo "2024-01-01")
END_DATE=$(date +%Y-%m-%d)

echo "[$(date)] Date range: ${START_DATE} to ${END_DATE}" | tee -a "${LOG_FILE}"
echo "[$(date)] Models dir: ${MODELS_DIR}" | tee -a "${LOG_FILE}"

# Run the training script
cd "${PROJECT_ROOT}"
python3 scripts/train_ml_model.py \
    --start_date "${START_DATE}" \
    --end_date "${END_DATE}" \
    --models_dir "${MODELS_DIR}" \
    2>&1 | tee -a "${LOG_FILE}"

TRAINING_EXIT_CODE=${PIPESTATUS[0]}

if [ ${TRAINING_EXIT_CODE} -eq 0 ]; then
    echo "[$(date)] Training completed successfully." | tee -a "${LOG_FILE}"
    # Touch the flag file that triggers the Go fsnotify watcher to reload models
    touch "${MODELS_DIR}/updated.flag"
    # Also touch the .onnx files to trigger the watcher
    if [ -f "${MODELS_DIR}/xgb_model.onnx" ]; then
        touch "${MODELS_DIR}/xgb_model.onnx"
    fi
    if [ -f "${MODELS_DIR}/lstm_model.onnx" ]; then
        touch "${MODELS_DIR}/lstm_model.onnx"
    fi
    echo "[$(date)] Model update flag set. Go watcher will reload." | tee -a "${LOG_FILE}"
else
    echo "[$(date)] Training FAILED with exit code ${TRAINING_EXIT_CODE}." | tee -a "${LOG_FILE}"
fi

echo "[$(date)] Done. Log: ${LOG_FILE}" | tee -a "${LOG_FILE}"
exit ${TRAINING_EXIT_CODE}
