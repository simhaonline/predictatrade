#!/usr/bin/env bash
# Predict-A-Trade Off-Host Backup Script
# SOW Section 18: Configurable off-host backup support
# Supports S3-compatible and NFS destinations
set -euo pipefail

BACKUP_DIR="${BACKUP_DIR:-/var/backups/predictatrade}"
OFFHOST_PROVIDER="${BACKUP_STORAGE_PROVIDER:-}"
OFFHOST_BUCKET="${BACKUP_BUCKET:-}"
OFFHOST_REGION="${BACKUP_REGION:-}"
OFFHOST_ENDPOINT="${BACKUP_ENDPOINT:-}"
OFFHOST_PREFIX="${BACKUP_PREFIX:-predictatrade}"
OFFHOST_ENCRYPTION="${BACKUP_ENCRYPTION:-none}"
LOG_FILE="${BACKUP_DIR}/offhost_backup.log"

mkdir -p "${BACKUP_DIR}"
log() { echo "[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] $*" | tee -a "${LOG_FILE}"; }

if [ -z "${OFFHOST_PROVIDER}" ]; then
    log "OFF_HOST_BACKUP_CONFIGURATION_REQUIRED"
    log "Set BACKUP_STORAGE_PROVIDER (s3, nfs, or compatible)"
    log "Required env vars: BACKUP_STORAGE_PROVIDER, BACKUP_BUCKET, BACKUP_REGION"
    log "Optional: BACKUP_ENDPOINT, BACKUP_PREFIX, BACKUP_ENCRYPTION"
    log "Example: BACKUP_STORAGE_PROVIDER=s3 BACKUP_BUCKET=my-bucket BACKUP_REGION=us-east-1"
    exit 2
fi

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

case "${OFFHOST_PROVIDER}" in
    s3|aws-s3)
        if [ -z "${OFFHOST_BUCKET}" ]; then
            log "ERROR: BACKUP_BUCKET not set for S3 provider"
            exit 1
        fi
        S3_OPTS=""
        if [ -n "${OFFHOST_ENDPOINT}" ]; then
            S3_OPTS="--endpoint-url ${OFFHOST_ENDPOINT}"
        fi
        S3_TARGET="s3://${OFFHOST_BUCKET}/${OFFHOST_PREFIX}/${BACKUP_NAME}"
        log "Uploading to S3: ${S3_TARGET}"
        if aws s3 cp "${S3_OPTS}" "${LATEST_BACKUP}" "${S3_TARGET}" \
            --sse "${OFFHOST_ENCRYPTION}" 2>>"${LOG_FILE}"; then
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

# Record in database
PGPASSWORD="${PGPASSWORD:-pat_local_dev_only}" psql -h 127.0.0.1 -U pat_admin -d predictatrade <<SQL
UPDATE system.backup_configuration 
SET config_value = 'configured', is_configured = true, updated_at = now()
WHERE config_key = 'off_host_backup_provider';
SQL

log "Off-host backup complete"
