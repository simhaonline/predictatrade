# Predict-A-Trade — Database Backup and Recovery Report

**Generated:** 2026-08-18

## Current Backup Mechanism

- **Script:** `scripts/backup/backup.sh`
- **Method:** `pg_dump --format=custom` via Docker exec (matches server version)
- **Location:** `/var/backups/predictatrade/`
- **Format:** PostgreSQL custom format (compressed)
- **Includes:** All schemas, tables, data, extensions, functions, triggers
- **Locking:** Prevents overlapping backup runs
- **Logging:** All operations logged to `backup.log`

## Changes Implemented

1. **Backup script** (`scripts/backup/backup.sh`):
   - Uses Docker exec for version-matched pg_dump
   - Calculates SHA-256 checksum
   - Records metadata in `system.backup_metadata`
   - Verifies backup via `pg_restore --list`
   - Cleans up old backups (30-day retention)
   - Reports size, duration, checksum

2. **Restore test script** (`scripts/backup/restore_test.sh`):
   - Restores to disposable test database (`pat_restore_test`)
   - Validates schemas, tables, extensions, data counts
   - Never touches production database
   - Cleans up test database after validation

3. **Backup metadata table** (`system.backup_metadata`):
   - Records backup ID, type, timestamps, status, size, checksum
   - Tracks PostgreSQL/TimescaleDB/pgvector versions
   - Records application revision

## Backup Frequency

| Type | Frequency | RPO | Status |
|------|-----------|-----|--------|
| Logical backup | On-demand / cron | < 24h | Implemented |
| Physical backup | Not configured | < 1h | RECOMMENDED |
| WAL archiving | Not configured | < 15 min | RECOMMENDED for production |

## RPO/RTO

| Metric | Target | Current |
|--------|--------|---------|
| RPO (Recovery Point Objective) | < 1 hour | < 24 hours (logical only) |
| RTO (Recovery Time Objective) | < 2 hours | < 30 minutes (verified) |

## Retention

| Period | Count | Storage |
|--------|-------|---------|
| Daily | 30 days | On-host (RECOMMEND off-host) |
| Weekly | Not configured | — |
| Monthly | Not configured | — |

## Encryption

- **At rest:** NOT YET CONFIGURED (recommend LUKS or cloud KMS)
- **In transit:** TCP to localhost (production: TLS to remote storage)
- **Checksums:** SHA-256 for every backup

## Off-host Status

**NOT YET CONFIGURED.** Current backups are on the same server. Production should:
1. Sync backups to off-host storage (S3, NFS, etc.)
2. Use immutable/object-lock storage where available
3. Never store backup credentials in the repository

## WAL Archiving / PITR

**NOT YET CONFIGURED.** Production should:
1. Set `archive_mode = on`
2. Set `archive_command` to copy WAL files to off-host storage
3. Configure `recovery_target_time` for PITR
4. Test PITR in isolated environment

## Last Backup Test

| Date | Backup ID | Status | Size | Checksum |
|------|-----------|--------|------|----------|
| 2026-08-18 | backup_20260818_094632_UTC | VERIFIED | 3,404,526 bytes | 4f292f9e... |

## Last Restore Test

| Date | Status | Tables | Signals | Ticks | pgvector |
|------|--------|--------|---------|-------|----------|
| 2026-08-18 | PASS | 137 | 592 | 144,335 | Present |

**Restore test was performed in a disposable test database (`pat_restore_test`) and cleaned up after validation.**

## Disaster Recovery Procedure

See `DATABASE_DISASTER_RECOVERY.md` for the full runbook.
