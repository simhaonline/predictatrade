#!/usr/bin/env bash
# Predict-A-Trade Off-Host Backup Script
# SOW Section 18: Configurable off-host backup support
# Supports S3-compatible (Hetzner, AWS, MinIO...) and NFS destinations.
#
# NOTE: the pat-backup-sync sidecar (docker-compose.yml) already mirrors every
# dump from /var/backups/predictatrade to S3 within ~60s of completion. This
# script is the standalone/on-demand fallback for operators (e.g. hosts without
# the compose stack, or an explicit manual push) and writes the same target
# key layout, so either path produces an identical bucket layout.
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/predictatrade}"
# Canonical config lives in infra/env/.env as BACKUP_S3_* (consumed by the
# backup-sync sidecar). Legacy BACKUP_* names still win if explicitly set.
OFFHOST_PROVIDER="${BACKUP_STORAGE_PROVIDER:-s3}"
OFFHOST_BUCKET="${BACKUP_BUCKET:-${BACKUP_S3_BUCKET:-}}"
OFFHOST_REGION="${BACKUP_REGION:-${BACKUP_S3_REGION:-}}"
OFFHOST_ENDPOINT="${BACKUP_ENDPOINT:-${BACKUP_S3_ENDPOINT:-}}"
OFFHOST_PREFIX="${BACKUP_PREFIX:-${BACKUP_S3_DB_PREFIX:-predictatrade/db}}"
OFFHOST_ENCRYPTION="${BACKUP_ENCRYPTION:-none}"
OFFHOST_ACCESS_KEY="${BACKUP_ACCESS_KEY:-${BACKUP_S3_ACCESS_KEY:-}}"
OFFHOST_SECRET_KEY="${BACKUP_SECRET_KEY:-${BACKUP_S3_SECRET_KEY:-}}"
LOG_FILE="${BACKUP_DIR}/offhost_backup.log"

mkdir -p "${BACKUP_DIR}" 2>/dev/null || true
# If the default log location isn't writable (non-root operator), fall back to
# /tmp — logging must never block the off-host upload.
if ! touch "${LOG_FILE}" 2>/dev/null; then
    LOG_FILE="$(mktemp "${TMPDIR:-/tmp}/offhost_backup.XXXXXX.log")"
fi
# Never let logging kill the upload: if the log file isn't writable (e.g.
# non-root operator), still print the message and continue.
log() {
    local msg
    msg="[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] $*"
    echo "${msg}"
    echo "${msg}" >> "${LOG_FILE}" 2>/dev/null || true
}

if [ -z "${OFFHOST_BUCKET}" ]; then
    log "OFF_HOST_BACKUP_CONFIGURATION_REQUIRED"
    log "Set BACKUP_S3_* in infra/env/.env (canonical) or legacy BACKUP_* vars"
    log "Required: BACKUP_S3_BUCKET (or BACKUP_BUCKET)"
    log "Optional: BACKUP_S3_ENDPOINT, BACKUP_S3_DB_PREFIX, BACKUP_ENCRYPTION"
    exit 2
fi

export AWS_ACCESS_KEY_ID="${OFFHOST_ACCESS_KEY}"
export AWS_SECRET_ACCESS_KEY="${OFFHOST_SECRET_KEY}"
export AWS_DEFAULT_REGION="${OFFHOST_REGION:-us-east-1}"

# Host may not have the aws CLI installed — fall back to the same amazon/aws-cli
# image the backup-sync sidecar uses. BACKUP_DIR is mounted read-only, so the
# container-visible source path is /backups/<name>, not the host path.
if command -v aws >/dev/null 2>&1; then
    AWS_SRC_PREFIX=""
else
    AWS_CFG_DIR="$(mktemp -d)"
    # Same Hetzner-safe tuning as the backup-sync sidecar: serialized uploads,
    # adaptive retry (bursty multipart triggers botocore NoneType errors).
    printf '[default]\ns3 =\n  max_concurrent_requests = 1\n  max_queue_size = 1\n  multipart_threshold = 64MB\nmax_attempts = 10\nretry_mode = adaptive\n' > "${AWS_CFG_DIR}/config"
    AWS_SRC_PREFIX="/backups"
fi

run_aws() {
    if [ -z "${AWS_SRC_PREFIX}" ]; then
        aws "$@"
    else
        docker run --rm \
            -e AWS_ACCESS_KEY_ID -e AWS_SECRET_ACCESS_KEY -e AWS_DEFAULT_REGION \
            -e AWS_CONFIG_FILE=/aws-config/config \
            -v "${AWS_CFG_DIR}:/aws-config:ro" \
            -v "${BACKUP_DIR}:/backups:ro" \
            amazon/aws-cli:latest "$@"
    fi
}

log "Off-host backup starting — provider: ${OFFHOST_PROVIDER}"

# Find latest backup
LATEST_BACKUP=$(ls -t "${BACKUP_DIR}"/backup_*.dump 2>/dev/null | head -1)
if [ -z "${LATEST_BACKUP}" ]; then
    log "ERROR: No local backup found to sync"
    exit 1
fi

BACKUP_NAME=$(basename "${LATEST_BACKUP}")
CHECKSUM=$(sha256sum "${LATEST_BACKUP}" | awk '{print $1}')
SIZE=$(stat -c%s "${LATEST_BACKUP}")

# Host-CLI mode reads the dump at its absolute path; docker fallback reads the
# same file through the read-only /backups mount.
if [ -n "${AWS_SRC_PREFIX}" ]; then
    SRC_PATH="${AWS_SRC_PREFIX}/${BACKUP_NAME}"
else
    SRC_PATH="${LATEST_BACKUP}"
fi

case "${OFFHOST_PROVIDER}" in
    s3|aws-s3)
        S3_OPTS=""
        if [ -n "${OFFHOST_ENDPOINT}" ]; then
            S3_OPTS="--endpoint-url ${OFFHOST_ENDPOINT}"
        fi
        S3_TARGET="s3://${OFFHOST_BUCKET}/${OFFHOST_PREFIX}/${BACKUP_NAME}"
        log "Uploading to S3: ${S3_TARGET}"
        # NOTE: --sse omitted when OFFHOST_ENCRYPTION=none — sending `--sse none`
        # is rejected by Hetzner S3. Only pass the flag for a real SSE algorithm.
        SSE_ARGS=""
        if [ -n "${OFFHOST_ENCRYPTION}" ] && [ "${OFFHOST_ENCRYPTION}" != "none" ]; then
            SSE_ARGS="--sse ${OFFHOST_ENCRYPTION}"
        fi
        # shellcheck disable=SC2086
        if run_aws s3 cp ${S3_OPTS} ${SSE_ARGS} "${SRC_PATH}" "${S3_TARGET}" \
            2>>"${LOG_FILE}"; then
            log "S3 upload complete — ${SIZE} bytes, checksum: ${CHECKSUM}"
        else
            log "ERROR: S3 upload failed"
            exit 1
        fi
        ;;
    nfs)
        NFS_TARGET="${OFFHOST_BUCKET:-/mnt/backups}"
        log "Copying to NFS: ${NFS_TARGET}/${BACKUP_NAME}"
        if cp "${LATEST_BACKUP}" "${NFS_TARGET}/${BACKUP_NAME}" 2>>"${LOG_FILE}"; then
            log "NFS copy complete — ${SIZE} bytes"
        else
            log "ERROR: NFS copy failed — ensure mount point ${NFS_TARGET} is mounted"
            exit 1
        fi
        ;;
    *)
        log "ERROR: Unknown provider '${OFFHOST_PROVIDER}'. Supported: s3, nfs"
        exit 1
        ;;
esac

# Record in database (best-effort bookkeeping — never fail the upload over it)
if ! PGPASSWORD="${PGPASSWORD:-pat_local_dev_only}" psql -h 127.0.0.1 -U pat_admin -d predictatrade <<SQL
UPDATE system.backup_configuration
SET config_value = 'configured', is_configured = true, updated_at = now()
WHERE config_key = 'off_host_backup_provider';
SQL
then
    log "WARN: could not update system.backup_configuration (non-fatal)"
fi

log "Off-host backup complete"