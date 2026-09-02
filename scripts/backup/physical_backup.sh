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
# Hetzner S3 continuously by pat-backup-sync. Base backups land in
# /var/backups/predictatrade/base/ which pat-backup-sync ships to S3.
set -euo pipefail

CONTAINER_NAME="${CONTAINER_NAME:-pat-postgres}"
BACKUP_DIR="${BACKUP_DIR:-/var/backups/predictatrade/base}"
RETENTION_BASE=7      # days of physical base backups to keep locally
TIMESTAMP=$(date -u +%Y%m%d_%H%M%S)
TARGET="${BACKUP_DIR}/base_${TIMESTAMP}"

log() { echo "[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] $*"; }

mkdir -p "$BACKUP_DIR"

log "=== Physical base backup started: base_${TIMESTAMP} ==="

# 1) Run pg_basebackup INSIDE the postgres container (tar format, gzipped,
#    WAL streamed). Output dir is internal to the container.
if docker exec "$CONTAINER_NAME" pg_basebackup \
    -U pat_admin \
    -D "/var/lib/postgresql/backups/base_${TIMESTAMP}" \
    -Ft -Xs -z; then

    log "pg_basebackup complete; streaming to host"

    # 2) Stream the tarball set to the host (root cron can write BACKUP_DIR;
    #    avoids relying on any rw container mount).
    mkdir -p "$TARGET"
    docker exec "$CONTAINER_NAME" tar \
        -C "/var/lib/postgresql/backups/base_${TIMESTAMP}" -cf - . \
        | tar -C "$TARGET" -xf -

    # 3) Cleanup inside the container
    docker exec "$CONTAINER_NAME" rm -rf "/var/lib/postgresql/backups/base_${TIMESTAMP}"

    # 4) Integrity manifest
    ( cd "$TARGET" && sha256sum *.tar.gz > "${TARGET}.sha256" )
    log "Base backup completed: ${TARGET} ($(du -sh "$TARGET" | cut -f1))"
else
    log "ERROR: pg_basebackup failed"
    curl -s -H "Title: PAT Physical Backup Failed" \
         -d "pg_basebackup failed at $(date -u)" \
         http://127.0.0.1:8091/pat-alerts >/dev/null 2>&1 || true
    exit 1
fi

# 5) Retention: keep the last N base backups
ls -1dt "${BACKUP_DIR}"/base_* 2>/dev/null | tail -n +$((RETENTION_BASE + 1)) | xargs -r rm -rf
log "Retention: kept last ${RETENTION_BASE} base backups"
log "=== Physical base backup done: base_${TIMESTAMP} ==="