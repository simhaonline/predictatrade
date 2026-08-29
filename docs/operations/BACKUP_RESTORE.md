# Backup and Restore Procedure

## Predict-A-Trade v1.17.3 — 29 August 2026

### 1. Overview

This document defines the backup strategy and step-by-step restore procedure
for the Predict-A-Trade platform. All production data lives in PostgreSQL;
Docker volumes, configuration files, and nginx configs are version-controlled
in the repository.

### 2. Backup Strategy

| Component | Method | Frequency | Retention |
|-----------|--------|:---------:|----------:|
| PostgreSQL | pg_dump (custom format) | Daily | 30 days |
| PostgreSQL | pg_dump (custom format) | Weekly | 12 weeks |
| PostgreSQL | pg_dump (custom format) | Monthly | 12 months |
| Docker volumes | Not backed up (stateless/rebuilt) | N/A | N/A |
| Config files | Git repository | On push | Indefinite |
| SSL certificates | certbot auto-renew | 90 days | Certbot managed |
| Logs | Rotated (nginx), timescaledb retention | Automated | Per retention policy |

### 3. Automated Backup (Daily)

Create this cron job on the production host:

```bash
#!/bin/bash
# /etc/cron.daily/pat-backup
# Predict-A-Trade daily PostgreSQL backup

BACKUP_DIR="/var/backups/predictatrade"
RETENTION_DAYS=30
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_FILE="${BACKUP_DIR}/pat_backup_${TIMESTAMP}.dump"

mkdir -p "$BACKUP_DIR"

# Full database dump in custom format (compressible, parallel restore)
docker exec pat-postgres pg_dump \
  -U pat_admin \
  -d predictatrade \
  -Fc \
  -v \
  --no-owner \
  --no-acl \
  > "$BACKUP_FILE"

# Verify the backup
if [ $? -eq 0 ] && [ -s "$BACKUP_FILE" ]; then
    echo "Backup OK: $BACKUP_FILE ($(du -h "$BACKUP_FILE" | cut -f1))"
else
    echo "BACKUP FAILED at $(date)" >&2
    # Send alert via ntfy
    curl -H "Title: PAT Backup Failed" \
         -d "Backup failed at $(date). Check logs." \
         https://ntfy.predictatrade.com/pat-alerts
    exit 1
fi

# Remove backups older than retention
find "$BACKUP_DIR" -name "pat_backup_*.dump" -mtime +$RETENTION_DAYS -delete

echo "Cleanup: removed backups older than ${RETENTION_DAYS} days"
```

### 4. Weekly and Monthly Retention

```bash
#!/bin/bash
# /etc/cron.weekly/pat-backup-weekly
BACKUP_DIR="/var/backups/predictatrade/weekly"
TIMESTAMP=$(date +%Y%m%d)
mkdir -p "$BACKUP_DIR"
docker exec pat-postgres pg_dump -U pat_admin -d predictatrade -Fc \
  > "${BACKUP_DIR}/pat_backup_weekly_${TIMESTAMP}.dump"
find "$BACKUP_DIR" -mtime +84 -delete  # 12 weeks
```

```bash
#!/bin/bash
# /etc/cron.monthly/pat-backup-monthly
BACKUP_DIR="/var/backups/predictatrade/monthly"
TIMESTAMP=$(date +%Y%m)
mkdir -p "$BACKUP_DIR"
docker exec pat-postgres pg_dump -U pat_admin -d predictatrade -Fc \
  > "${BACKUP_DIR}/pat_backup_monthly_${TIMESTAMP}.dump"
find "$BACKUP_DIR" -mtime +365 -delete  # 12 months
```

### 5. Restore Procedure

#### 5.1 Full Database Restore

```bash
# 1. Stop all application services (keep postgres running)
docker compose stop realtime control frontend live-terminal backtest

# 2. Drop and recreate the database
docker exec pat-postgres psql -U pat_admin -c "DROP DATABASE IF EXISTS predictatrade;"
docker exec pat-postgres psql -U pat_admin -c "CREATE DATABASE predictatrade OWNER pat_admin;"

# 3. Restore from backup
docker exec -i pat-postgres pg_restore \
  -U pat_admin \
  -d predictatrade \
  -v \
  --no-owner \
  --no-acl \
  --clean \
  --if-exists \
  < /var/backups/predictatrade/pat_backup_YYYYMMDD_HHMMSS.dump

# 4. Run any pending migrations (if backup is older than current schema)
# The migrations are idempotent (IF NOT EXISTS guards)

# 5. Start application services
docker compose up -d realtime control frontend live-terminal backtest

# 6. Verify
docker compose ps
curl -s http://localhost:13081/health
curl -s http://localhost:13080/api/v1/health
```

#### 5.2 Point-in-Time Recovery

```bash
# For PITR, you need WAL archives. Enable WAL archiving first:
# docker exec pat-postgres psql -U pat_admin -c "ALTER SYSTEM SET archive_mode = on;"
# docker exec pat-postgres psql -U pat_admin -c "ALTER SYSTEM SET archive_command = 'cp %p /var/backups/predictatrade/wal/%f';"

# Then restore to a specific point:
# pg_restore with --target-time or use pg_basebackup + WAL replay
```

#### 5.3 Single Table Restore

```bash
# Restore a single table from backup
docker exec -i pat-postgres pg_restore \
  -U pat_admin \
  -d predictatrade \
  --data-only \
  --table=market.candles \
  --verbose \
  < /var/backups/predictatrade/pat_backup_YYYYMMDD.dump
```

### 6. Restore Validation Checklist

After any restore, complete ALL checks:

- [ ] All Docker containers are running: `docker compose ps` (all "healthy" or "running")
- [ ] PostgreSQL is accessible: `docker exec pat-postgres pg_isready -U pat_admin`
- [ ] Real-time engine health: `curl http://localhost:13081/health` returns 200
- [ ] Control plane health: `curl http://localhost:13080/api/v1/health` returns 200
- [ ] Frontend is serving: `curl http://localhost:13082/` returns HTML
- [ ] Row counts sanity check:
  ```bash
  docker exec pat-postgres psql -U pat_admin -d predictatrade -c "
    SELECT 'users' AS tbl, count(*) FROM iam.users
    UNION ALL SELECT 'strategies', count(*) FROM trading.strategies
    UNION ALL SELECT 'signals', count(*) FROM trading.signals
    UNION ALL SELECT 'candles', count(*) FROM market.candles;
  "
  ```
- [ ] Latest candle timestamp is within 5 minutes of current time:
  ```bash
  docker exec pat-postgres psql -U pat_admin -d predictatrade -t -c \
    "SELECT max(timestamp) FROM market.candles;"
  ```
- [ ] Signal generation is active: check metrics or watch logs for "signal generated"
- [ ] SSL certificates are valid:
  ```bash
  certbot certificates | grep -A2 "docs.predictatrade.com"
  ```

### 7. Scheduled Restore Test

Run a full restore test **monthly** on a staging/isolated environment:

```bash
#!/bin/bash
# restore-test.sh — runs a full restore to a temp database and validates

TEMP_DB="predictatrade_restore_test"

# Get latest backup
LATEST=$(ls -t /var/backups/predictatrade/pat_backup_*.dump | head -1)

echo "Testing restore of: $LATEST"

# Create temp database
docker exec pat-postgres psql -U pat_admin -c "DROP DATABASE IF EXISTS $TEMP_DB;"
docker exec pat-postgres psql -U pat_admin -c "CREATE DATABASE $TEMP_DB OWNER pat_admin;"

# Restore
docker exec -i pat-postgres pg_restore -U pat_admin -d "$TEMP_DB" --no-owner --no-acl < "$LATEST"

# Validate
docker exec pat-postgres psql -U pat_admin -d "$TEMP_DB" -c "
  SELECT 'users' AS tbl, count(*) FROM iam.users
  UNION ALL SELECT 'signals', count(*) FROM trading.signals
  UNION ALL SELECT 'candles', count(*) FROM market.candles;
"

# Cleanup
docker exec pat-postgres psql -U pat_admin -c "DROP DATABASE IF EXISTS $TEMP_DB;"

echo "Restore test PASSED: $LATEST"
```

### 8. Offsite Backup

For disaster recovery (covered in DR_PLAN.md), maintain at least one offsite
copy:

```bash
# Sync to remote storage (example with rsync)
rsync -avz --delete \
  /var/backups/predictatrade/ \
  backups@offsite-storage:/backups/predictatrade/
```

### 9. Recovery Time Objectives (RTO) and Recovery Point Objectives (RPO)

| Scenario | RTO | RPO | Method |
|----------|:---:|:---:|--------|
| Database corruption | 30 min | 24 hours | Full restore from daily backup |
| Accidental deletion | 15 min | 24 hours | Single-table restore |
| Full infrastructure loss | 2 hours | 24 hours | Rebuild from git + restore DB |
| Ransomware | 4 hours | 24 hours | Offsite restore + rebuild |

### 10. Related Documents

- [Disaster Recovery Plan](DR_PLAN.md)
- [Incident Response Plan](INCIDENT_RESPONSE_PLAN.md)
- [Database Architecture](../database/DATABASE_ARCHITECTURE.md)
