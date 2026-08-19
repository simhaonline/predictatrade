# Predict-A-Trade — Database Disaster Recovery Runbook

## 1. Failure Identification

- Check `system.backup_metadata` for latest backup status
- Check PostgreSQL logs: `docker logs pat-postgres`
- Check application health endpoint
- Identify failure type: corruption, accidental deletion, hardware failure

## 2. Stop Writes

```bash
# Stop the realtime engine
pkill -f realtime-engine

# Stop the control plane
systemctl stop pat-control

# Stop the frontend (optional)
systemctl stop pat-frontend
```

## 3. Choose Recovery Point

- Identify the target recovery time from `system.backup_metadata`
- Note the backup ID and file location

## 4. Provision PostgreSQL (if needed)

```bash
# If the PostgreSQL container is corrupted:
docker stop pat-postgres
docker rm pat-postgres
# Recreate from docker-compose with fresh volume
```

## 5. Restore Base Backup

```bash
# Copy backup to container
docker cp /var/backups/predictatrade/<backup_id>.dump pat-postgres:/tmp/restore.dump

# Restore
docker exec pat-postgres pg_restore -U pat_admin -d predictatrade \
    --no-owner --no-privileges --clean --if-exists /tmp/restore.dump
```

## 6. Validate TimescaleDB

```sql
SELECT extname, extversion FROM pg_extension WHERE extname = 'timescaledb';
-- If missing: CREATE EXTENSION timescaledb;
```

## 7. Validate pgvector

```sql
SELECT extname, extversion FROM pg_extension WHERE extname = 'vector';
-- If missing: CREATE EXTENSION vector;
```

## 8. Post-Recovery Validation

```sql
-- Check schemas
SELECT count(*) FROM information_schema.schemata 
WHERE schema_name IN ('iam','control','trading','market','audit','system','ai');

-- Check critical table counts
SELECT 'signals' as t, count(*) FROM trading.signals
UNION ALL SELECT 'ticks', count(*) FROM market.ticks
UNION ALL SELECT 'audit_events', count(*) FROM audit.audit_events
UNION ALL SELECT 'users', count(*) FROM iam.users;

-- Run integrity checks
SELECT * FROM system.check_data_integrity();

-- Check backup metadata
SELECT backup_id, status, completed_at FROM system.backup_metadata 
ORDER BY completed_at DESC LIMIT 5;
```

## 9. Reconnect Application

```bash
# Start realtime engine
cd /srv/predictatrade/xauusd/realtime
nohup ./bin/realtime-engine > /tmp/rt-engine.log 2>&1 &

# Start control plane
systemctl start pat-control

# Start frontend
systemctl start pat-frontend

# Verify health
curl http://127.0.0.1:13080/api/v1/health
curl http://127.0.0.1:13081/health
```

## 10. Post-Recovery Consistency Checks

- Verify MT5 agent reconnects
- Verify signal pipeline generates signals
- Verify dashboard loads
- Verify WebSocket connections work
- Verify admin operations work
- Verify no duplicate signals after restart (cooldown/fingerprint in Valkey)
