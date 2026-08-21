#!/usr/bin/env bash
# ─────────────────────────────────────────────────────────────────────────────
# scripts/setup_ml_env.sh — Prepare Ubuntu VPS for Predict-A-Trade ML pipeline
#
# Idempotent: safe to run multiple times. Logs errors to logs/setup_error.log.
# Exit code 1 on any failure.
# ─────────────────────────────────────────────────────────────────────────────
set -euo pipefail

PROJECT_ROOT="/srv/predictatrade/xauusd"
ENV_FILE="${PROJECT_ROOT}/infra/env/realtime.env"
LOG_DIR="${PROJECT_ROOT}/logs"
ERROR_LOG="${LOG_DIR}/setup_error.log"
MODELS_DIR="${PROJECT_ROOT}/models"
ONNX_VERSION="1.20.0"
ONNX_URL="https://github.com/microsoft/onnxruntime/releases/download/v${ONNX_VERSION}/onnxruntime-linux-x64-${ONNX_VERSION}.tgz"
ONNX_LIB="/usr/local/lib/libonnxruntime.so"
TMP_DIR="/tmp/onnxruntime_setup"

mkdir -p "${LOG_DIR}" "${MODELS_DIR}"

log()  { echo "[$(date '+%Y-%m-%d %H:%M:%S')] $1"; }
fail() { echo "[$(date '+%Y-%m-%d %H:%M:%S')] ERROR: $1" | tee -a "${ERROR_LOG}"; exit 1; }

# ─── Step 1: Install system packages ─────────────────────────────────────────
log "Step 1: Installing system packages..."
export DEBIAN_FRONTEND=noninteractive
if ! apt-get update -qq 2>>"${ERROR_LOG}"; then
    fail "apt-get update failed"
fi

PACKAGES="build-essential cmake libssl-dev wget python3-pip"
if ! apt-get install -y -qq ${PACKAGES} 2>>"${ERROR_LOG}"; then
    fail "apt-get install failed for: ${PACKAGES}"
fi
log "  System packages installed."

# ─── Step 2: Download and install libonnxruntime.so ──────────────────────────
log "Step 2: Installing ONNX Runtime v${ONNX_VERSION}..."

if [ -f "${ONNX_LIB}" ]; then
    log "  libonnxruntime.so already exists at ${ONNX_LIB} — skipping download."
else
    mkdir -p "${TMP_DIR}"
    cd "${TMP_DIR}"

    TGZ_FILE="onnxruntime-linux-x64-${ONNX_VERSION}.tgz"
    log "  Downloading from ${ONNX_URL}..."
    if ! wget -q "${ONNX_URL}" -O "${TGZ_FILE}" 2>>"${ERROR_LOG}"; then
        fail "wget failed for ONNX Runtime"
    fi

    log "  Extracting..."
    if ! tar xzf "${TGZ_FILE}" 2>>"${ERROR_LOG}"; then
        fail "tar extraction failed"
    fi

    EXTRACTED_DIR="onnxruntime-linux-x64-${ONNX_VERSION}"
    if [ ! -f "${EXTRACTED_DIR}/lib/libonnxruntime.so" ]; then
        fail "libonnxruntime.so not found in extracted archive"
    fi

    log "  Copying to /usr/local/lib/..."
    cp "${EXTRACTED_DIR}/lib/libonnxruntime.so" "${ONNX_LIB}"
    cd "${PROJECT_ROOT}"
    rm -rf "${TMP_DIR}"
fi

log "  Running ldconfig..."
if ! ldconfig 2>>"${ERROR_LOG}"; then
    fail "ldconfig failed"
fi
# Ensure symlink for unversioned library name
if [ ! -f "${ONNX_LIB}" ] && [ -f "${ONNX_LIB}.1" ]; then
    ln -sf "${ONNX_LIB}.1" "${ONNX_LIB}" 2>/dev/null || true
fi
ldconfig 2>>"${ERROR_LOG}" || true

# ─── Step 3: Install Python dependencies (CPU-only) ─────────────────────────
log "Step 3: Installing Python dependencies (CPU-only)..."

log "  Installing xgboost..."
if ! pip3 install --quiet --break-system-packages xgboost 2>>"${ERROR_LOG}"; then
    fail "pip install xgboost failed"
fi

log "  Installing torch (CPU-only)..."
if ! pip3 install --quiet --break-system-packages torch --index-url https://download.pytorch.org/whl/cpu 2>>"${ERROR_LOG}"; then
    fail "pip install torch failed"
fi

log "  Installing remaining ML dependencies..."
if ! pip3 install --quiet --break-system-packages onnx onnxruntime skl2onnx pandas-ta psycopg2-binary sqlalchemy python-dotenv scikit-learn 2>>"${ERROR_LOG}"; then
    fail "pip install ML dependencies failed"
fi
log "  Python dependencies installed."

# ─── Step 4: Create models/ directory ────────────────────────────────────────
log "Step 4: Ensuring models/ directory exists..."
mkdir -p "${MODELS_DIR}"
log "  models/ directory ready."

# ─── Step 5: Update .env file ────────────────────────────────────────────────
log "Step 5: Updating .env file..."

update_env_key() {
    local key="$1"
    local value="$2"
    local file="${ENV_FILE}"

    if grep -q "^${key}=" "${file}" 2>/dev/null; then
        # Key exists — update it
        sed -i "s|^${key}=.*|${key}=${value}|" "${file}"
    else
        # Key doesn't exist — append it
        echo "${key}=${value}" >> "${file}"
    fi
}

update_env_key "ONNXRUNTIME_LIB" "/usr/local/lib/libonnxruntime.so"
update_env_key "ML_ENABLED" "true"

log "  .env updated: ONNXRUNTIME_LIB and ML_ENABLED set."

# ─── Step 6: Verify installation ─────────────────────────────────────────────
log "Step 6: Verifying installation..."

log "  Checking libonnxruntime.so..."
if ! ldconfig -p | grep -q "onnx"; then
    fail "ldconfig does not show onnxruntime library"
fi
log "  libonnxruntime.so verified in ldconfig."

log "  Checking Python onnxruntime import..."
if ! python3 -c "import onnxruntime; print('ONNX OK')" 2>>"${ERROR_LOG}"; then
    fail "python3 'import onnxruntime' failed"
fi
log "  Python onnxruntime import verified."

log "  Checking xgboost..."
if ! python3 -c "import xgboost; print('XGB OK')" 2>>"${ERROR_LOG}"; then
    fail "python3 'import xgboost' failed"
fi
log "  XGBoost verified."

log "  Checking torch (CPU)..."
if ! python3 -c "import torch; print(f'Torch OK: {torch.__version__}, CUDA={torch.cuda.is_available()}')" 2>>"${ERROR_LOG}"; then
    fail "python3 'import torch' failed"
fi
log "  PyTorch verified."

# ─── Done ────────────────────────────────────────────────────────────────────
log ""
log "════════════════════════════════════════════════════════════"
log " ML Environment Setup Complete"
log "════════════════════════════════════════════════════════════"
log "  ONNX Runtime: ${ONNX_LIB}"
log "  Models dir:   ${MODELS_DIR}"
log "  Env file:     ${ENV_FILE}"
log "  ML_ENABLED:   true"
log "  ONNXRUNTIME_LIB=/usr/local/lib/libonnxruntime.so"
log "════════════════════════════════════════════════════════════"
log ""
echo "Setup completed successfully."
exit 0
