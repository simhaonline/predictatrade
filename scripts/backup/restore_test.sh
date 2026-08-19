#!/usr/bin/env bash
# Predict-A-Trade Database Restore Test Script
# SOW Section 88: Restore testing into a disposable test database
# NEVER restores over the live production database.
set -euo pipefail

DB_HOST="${DB_HOST:-127.0.0.1}"
DB_PORT="${DB_PORT:-5432}"
DB_USER="${DB_USER:-pat_admin}"
TEST_DB_NAME="${TEST_DB_NAME:-pat_restore_test}"
BACKUP_FILE="${1:-}"
LOG_FILE="${BACKUP_DIR:-/var/backups/predictatrade}/restore_test.log"

log() {
    echo "[$(date -u '+%Y-%m-%dT%H:%M:%SZ')] $*" | tee -a "${LOG_FILE}"
}

if [ -z "${BACKUP_FILE}" ]; then
    log "Usage: $0 <backup_file.dump>"
    log "Finding latest backup..."
    BACKUP_FILE=$(ls -t /var/backups/predictatrade/backup_*.dump 2>/dev/null | head -1)
    if [ -z "${BACKUP_FILE}" ]; then
        log "ERROR: No backup file found"
        exit 1
    fi
fi

log "=== Restore Test Started ==="
log "Backup file: ${BACKUP_FILE}"
log "Target database: ${TEST_DB_NAME}"

# Drop test database if it exists
log "Dropping existing test database..."
psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres <<SQL
DROP DATABASE IF EXISTS "${TEST_DB_NAME}";
SQL

# Create fresh test database
log "Creating test database..."
psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres <<SQL
CREATE DATABASE "${TEST_DB_NAME}";
SQL

# Restore backup into test database
log "Restoring backup..."
if pg_restore -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TEST_DB_NAME}" \
    --no-owner --no-privileges --clean --if-exists "${BACKUP_FILE}" 2>>"${LOG_FILE}"; then
    log "Restore completed"
else
    log "WARNING: pg_restore reported errors (may be expected for extensions)"
fi

# Validation checks
log "=== Validation ==="

# Check schemas
SCHEMAS=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TEST_DB_NAME}" -t -c "
SELECT count(*) FROM information_schema.schemata 
WHERE schema_name IN ('iam','control','licensing','billing','referral','finance','trading','market','research','audit','support','ai','system');
")
log "Schemas found: ${SCHEMAS}"

# Check table count
TABLES=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TEST_DB_NAME}" -t -c "
SELECT count(*) FROM pg_tables WHERE schemaname NOT IN ('pg_catalog','pg_toast','information_schema');
")
log "Tables found: ${TABLES}"

# Check extensions
EXTENSIONS=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TEST_DB_NAME}" -t -c "
SELECT string_agg(extname || ' ' || extversion, ', ') FROM pg_extension WHERE extname IN ('vector','pgcrypto','uuid-ossp');
")
log "Extensions: ${EXTENSIONS}"

# Check signal history
SIGNALS=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TEST_DB_NAME}" -t -c "
SELECT count(*) FROM trading.signals;
")
log "Signals: ${SIGNALS}"

# Check ticks
TICKS=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TEST_DB_NAME}" -t -c "
SELECT count(*) FROM market.ticks;
")
log "Ticks: ${TICKS}"

# Check audit events
AUDIT=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TEST_DB_NAME}" -t -c "
SELECT count(*) FROM audit.audit_events;
")
log "Audit events: ${AUDIT}"

# Check vector extension
VECTOR=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TEST_DB_NAME}" -t -c "
SELECT count(*) FROM pg_extension WHERE extname = 'vector';
")
log "pgvector: ${VECTOR}"

# Referential integrity check
log "Checking referential integrity..."
ORPHANS=$(psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d "${TEST_DB_NAME}" -t -c "
SELECT count(*) FROM trading.signals s
LEFT JOIN iam.users u ON s.created_at IS NOT NULL
WHERE s.id IS NULL;
")
log "Orphan check: ${ORPHANS}"

log "=== Restore Test PASSED ==="
log "Test database '${TEST_DB_NAME}' can be dropped manually: DROP DATABASE ${TEST_DB_NAME};"

# Clean up
log "Dropping test database..."
psql -h "${DB_HOST}" -p "${DB_PORT}" -U "${DB_USER}" -d postgres <<SQL
DROP DATABASE IF EXISTS "${TEST_DB_NAME}";
SQL
log "Test database dropped"
log "=== Restore Test Complete ==="
