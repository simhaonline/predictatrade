#!/usr/bin/env bash
set -euo pipefail

PROJECT_ROOT="/srv/predictatrade/xauusd"
ENV_FILE="${PROJECT_ROOT}/infra/env/realtime.env"
ONNX_LIB="/usr/local/lib/libonnxruntime.so"
ONNX_LIB_V1="/usr/local/lib/libonnxruntime.so.1"
ERROR_LOG="${PROJECT_ROOT}/logs/ml_fix_error.log"

mkdir -p "${PROJECT_ROOT}/logs"
fail() { echo "[$(date)] ERROR: $1" | tee -a "${ERROR_LOG}"; exit 1; }

echo "[$(date)] Step 1: Fix ONNX Runtime version..."
# Pin Python onnxruntime to 1.20.0 to match C library
pip3 install --break-system-packages 'onnxruntime==1.20.0' 2>>"${ERROR_LOG}" || \
  echo "  Warning: onnxruntime==1.20.0 install failed, keeping current version"

echo "[$(date)] Step 2: Create symlink..."
if [ -f "${ONNX_LIB_V1}" ] && [ ! -f "${ONNX_LIB}" ]; then
    ln -sf "${ONNX_LIB_V1}" "${ONNX_LIB}"
    echo "  Symlink created: ${ONNX_LIB} -> ${ONNX_LIB_V1}"
elif [ -f "${ONNX_LIB}" ]; then
    echo "  ${ONNX_LIB} already exists"
else
    echo "  WARNING: neither ${ONNX_LIB} nor ${ONNX_LIB_V1} found"
fi

ldconfig 2>>"${ERROR_LOG}" || true

echo "[$(date)] Step 3: Install Python deps..."
pip3 install --break-system-packages xgboost 2>>"${ERROR_LOG}" || echo "  xgboost already installed"
pip3 install --break-system-packages torch --index-url https://download.pytorch.org/whl/cpu 2>>"${ERROR_LOG}" || echo "  torch already installed"
pip3 install --break-system-packages skl2onnx onnx 2>>"${ERROR_LOG}" || echo "  skl2onnx/onnx already installed"
pip3 install --break-system-packages pandas-ta 2>>"${ERROR_LOG}" || echo "  pandas-ta install skipped (non-critical)"

echo "[$(date)] Step 4: Update .env..."
# Update env file with correct ONNXRUNTIME_LIB
if grep -q "ONNXRUNTIME_LIB" "${ENV_FILE}"; then
    sed -i "s|^ONNXRUNTIME_LIB=.*|ONNXRUNTIME_LIB=${ONNX_LIB}|" "${ENV_FILE}"
else
    echo "ONNXRUNTIME_LIB=${ONNX_LIB}" >> "${ENV_FILE}"
fi
echo "  .env updated: ONNXRUNTIME_LIB=${ONNX_LIB}"

echo "[$(date)] Step 5: Verify..."
ldconfig -p | grep -q "libonnxruntime" && echo "  ldconfig: OK" || fail "ldconfig check failed"
python3 -c "import onnxruntime; print(f'  onnxruntime: {onnxruntime.__version__}')" 2>>"${ERROR_LOG}" || fail "onnxruntime import failed"
python3 -c "import xgboost; print(f'  xgboost: {xgboost.__version__}')" 2>>"${ERROR_LOG}" || fail "xgboost import failed"
python3 -c "import torch; print(f'  torch: {torch.__version__}')" 2>>"${ERROR_LOG}" || fail "torch import failed"

echo ""
echo "ML Environment Fix Complete."
echo "  ONNXRUNTIME_LIB=${ONNX_LIB}"
echo "  onnxruntime (Python)=$(python3 -c 'import onnxruntime; print(onnxruntime.__version__)')"
exit 0
