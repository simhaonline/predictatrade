#!/usr/bin/env bash
# Predict-A-Trade PHYSICAL base backup (pg_basebackup, tar + gz + WAL).
#
# WHY: the 2026-09-02 restore drill proved pg_dump -Fc logical dumps cannot
# restore TimescaleDB COMPRESSED chunk data (5,874/5,897 market.candles
# chunks compressed → "chunk not found" on restore). Only a physical base
# backup + WAL archive gives full market-data recovery (PITR).
#
# Install as root cron (host):
#   0 */6 * * * /srv/predictatrade/xauusd/scripts/backup/physical_backup.sh >> /var/backups/predictatrade/basebackup.log 2>&1
#
# WAL archiving is already on (archive_mode=on, wal_level=replica) and
# /var/lib/docker/volumes/xauusd_pat-pgdata/_data/wal_archive is synced to
# Hetzner S3 continuously by pat-backup-sync. Base backup tarballs land in
# the same S3 bucket via the /pgbackups mount (below).
set -euo pipefail

CONTAINER_NAME="${CONTAINER_NAME:-pat-postgres}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/predictatrade/base}"
RETENTION_BASE=7      # days of physical base backups to keep locally
TIMESTAMP=$(date -u +%Y%m%d_%H%M%S)
TARGET="${BACKUP_DIR}/base_${TIMESTAMP}"

log() { echo "[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] $*"; }

mkdir -p "$BACKUP_DIR"

log "=== Physical base backup started: base_${TIMESTAMP} ==="

if docker exec "$CONTAINER_NAME" pg_basebackup \
    -U pat_admin \
    -D "/var/lib/postgresql/backups/base_${TIMESTAMP}" \
    -Ft -Xs -z -P; then

    # Move the tarball set into the S3-synced /pgbackups mount
    docker exec "$CONTAINER_NAME" sh -c \
        "mv /var/lib/postgresql/backups/base_${TIMESTAMP} /pgbackups/base_${TIMESTAMP} 2>/dev/null" \
        || mv "${BACKUP_DIR}/base_${TIMESTAMP}" "/var/backups/predictatrade/base_${TIMESTAMP}" 2>/dev/null || true

    log "Base backup completed: base_${TIMESTAMP}"
    docker run --rm -v /var/backups/predictatrade:/v alpine sh -c \
        "cd /v && sha256sum base_${TIMESTAMP}/*.tar.gz > base_${TIMESTAMP}.sha256" 2>/dev/null || true
else
    log "ERROR: pg_basebackup failed"
    curl -s -H "Title: PAT Physical Backup Failed" \
         -d "pg_basebackup failed at $(date -u)" \
         http://127.0.0.1:8091/pat-alerts >/dev/null 2>&1 || true
    exit 1
fi

# Retention: keep last N base backups
ls -1dt "${BACKUP_DIR}"/base_* 2>/dev/null | tail -n +$((RETENTION_BASE + 1)) | xargs -r rm -rf
log "Retention: kept last ${RETENTION_BASE} base backups"
log "=== Physical base backup done: base_${TIMESTAMP} ==="