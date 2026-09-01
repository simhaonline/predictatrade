# Predict-A-Trade — IT Disaster Recovery Plan
## v1.18.0 — 01 September 2026

---

## 1. Document Control

| Field | Value |
|-------|-------|
| Document ID | PAT-DR-001 |
| Version | 1.0.0 |
| Classification | CONFIDENTIAL — Operations |
| Owner | Infrastructure Team |
| Last Review | 26 August 2026 |
| Next Review | 26 November 2026 |
| Distribution | Admin, DevOps, CTO |

---

## 2. Scope

This Disaster Recovery Plan covers all Predict-A-Trade production services deployed at the primary hosting location. It defines recovery objectives, asset inventory, risk assessment, backup strategies, testing procedures, and escalation protocols. The scope includes:

- PostgreSQL/TimescaleDB (primary data store)
- Valkey cache (hot state)
- Go Realtime Engine (signal generation)
- NestJS Control Plane (IAM, billing, licensing)
- Next.js Frontend (user/admin dashboards)
- Nginx reverse proxy (TLS termination, routing)
- Prometheus + Grafana (monitoring)
- ntfy (notifications)
- Docker host infrastructure

**Out of scope:** Windows/MT4/MT5 agent machines (client-side, per-user), third-party API providers (TwelveData, FMP, Stripe, NOWPayments).

---

## 3. Recovery Objectives

### 3.1 RTO — Recovery Time Objective

| Service | Tier | RTO | Strategy |
|---------|:----:|:---:|----------|
| PostgreSQL | P0 | 1 hour | Point-in-time recovery from WAL + base backup |
| Realtime Engine | P0 | 30 minutes | Rebuild from Docker image + env, reconnect to DB |
| Control Plane | P0 | 30 minutes | Rebuild from Docker image + env, reconnect to DB |
| Nginx | P0 | 15 minutes | Rebuild from config + certs |
| Frontend | P1 | 1 hour | Rebuild from Docker image |
| Valkey | P1 | 15 minutes | Warm cache from DB on restart (stateless) |
| Prometheus | P2 | 2 hours | Rebuild from config, metrics loss acceptable |
| Grafana | P2 | 2 hours | Rebuild from provisioned dashboards |
| ntfy | P2 | 1 hour | Rebuild from Docker image |
| Status Page | P2 | 1 hour | Rebuild from Docker image |
| Live Terminal | P2 | 1 hour | Rebuild from Docker image |

### 3.2 RPO — Recovery Point Objective

| Data Store | RPO | Backup Frequency | Method |
|-----------|:---:|:----------------:|--------|
| PostgreSQL (market.candles) | 1 hour | Hourly | WAL archiving + pg_dump |
| PostgreSQL (trading.signals) | 1 hour | Hourly | WAL archiving |
| PostgreSQL (iam, billing, finance) | 1 hour | Hourly | WAL archiving |
| PostgreSQL (compliance.audit_events) | 1 hour | Hourly | WAL archiving |
| Nginx config + certs | 24 hours | Daily | File backup |
| Docker images | Point-in-time | On build | Git + Docker registry |
| Valkey | 0 (cache) | None | Stateless — rebuilt from DB |

### 3.3 RTO/RPO Summary

| Metric | Target |
|--------|:------:|
| Maximum acceptable downtime | 1 hour |
| Maximum acceptable data loss | 1 hour |
| Recovery priority order | PostgreSQL → Realtime Engine → Control → Nginx → Frontend → Rest |

---

## 4. Asset Inventory

### 4.1 Hardware / Host Infrastructure

| Asset | Specification | Purpose | Location |
|-------|:------------:|---------|----------|
| Primary VPS | 64 GB RAM, 8 vCPU, 500 GB SSD | Docker host — all services | Primary DC |
| External Volume | 200 GB | PostgreSQL data volume (pat-pgdata) | Primary DC |
| External Volume | 50 GB | Backup storage | Primary DC |
| Reserved IP | 1x IPv4, 1x IPv6 | Public endpoints | Primary DC |

### 4.2 Software Services

| Service | Container | Image | Source |
|---------|-----------|-------|--------|
| pat-postgres | PostgreSQL 17 + TimescaleDB | timescale/timescaledb-ha:pg17 | Docker Hub |
| pat-valkey | Valkey 8.0 | valkey/valkey:8.0 | Docker Hub |
| pat-realtime | Go 1.26 engine | Built | realtime/Dockerfile |
| pat-control | NestJS | Built | control/Dockerfile |
| pat-frontend | Next.js 16 | Built | frontend/Dockerfile |
| pat-live-terminal | Go terminal | Built | live-terminal/Dockerfile |
| pat-status | Status service | Built | status/Dockerfile |
| pat-nginx | Nginx Alpine | Built | Custom + nginx:alpine |
| pat-prometheus | Prometheus | prom/prometheus:latest | Docker Hub |
| pat-grafana | Grafana | grafana/grafana:latest | Docker Hub |
| pat-ntfy | ntfy | binwiederhier/ntfy:latest | Docker Hub |

### 4.3 Data Assets

| Asset | Location | Size (est.) | Criticality |
|-------|----------|:-----------:|:-----------:|
| market.candles (hypertable) | pat-pgdata | ~50 GB growing | HIGH |
| trading.signals | pat-pgdata | ~500 MB | CRITICAL |
| iam.users | pat-pgdata | ~10 MB | CRITICAL |
| billing.subscriptions | pat-pgdata | ~5 MB | CRITICAL |
| finance.ledger_entries | pat-pgdata | ~50 MB | CRITICAL |
| compliance.* | pat-pgdata | ~100 MB | HIGH |
| SSL certificates | /etc/letsencrypt | ~1 MB | HIGH |
| Nginx config | infra/nginx/ | ~50 KB | MEDIUM |
| Environment files | infra/env/ | ~10 KB | CRITICAL |

### 4.4 External Dependencies

| Dependency | Provider | Failure Impact | Mitigation |
|-----------|----------|---------------|------------|
| TwelveData API | twelvedata.com | No XAUUSD candles | FMP fallback, local OHLC replay |
| FMP API | financialmodelingprep.com | No COT/DXY data | Stale data, zero contribution |
| Stripe API | stripe.com | No card payments | NOWPayments fallback |
| NOWPayments | nowpayments.io | No crypto payments | Stripe fallback |
| Ollama (local) | self-hosted | No sentiment | Zero sentiment contribution |
| Docker Hub | hub.docker.com | No image pulls | Local image cache |
| GitHub | github.com | No code deploys | Local git clone |

---

## 5. Risk Assessment

### 5.1 Threat Matrix

| Threat | Likelihood | Impact | Risk Score | Mitigation |
|--------|:----------:|:------:|:----------:|------------|
| Disk failure | Medium (3) | Critical (5) | 15 | RAID/volume redundancy + offsite backup |
| Host hardware failure | Low (2) | Critical (5) | 10 | Secondary host standby + backup restore |
| PostgreSQL corruption | Low (2) | Critical (5) | 10 | WAL archiving + PITR + pg_dump |
| Accidental data deletion | Medium (3) | High (4) | 12 | Point-in-time recovery from WAL |
| Ransomware | Low (2) | Critical (5) | 10 | Offsite immutable backups + air-gapped copy |
| DDoS attack | Medium (3) | Medium (3) | 9 | Cloudflare/CDN + rate limiting |
| API key revocation | Low (2) | High (4) | 8 | Key rotation procedure, backup keys |
| Docker daemon crash | Low (2) | Medium (3) | 6 | systemd auto-restart, compose up |
| Network partition | Low (2) | High (4) | 8 | Multi-homed network, fallback routes |
| Human error (misconfig) | Medium (3) | Medium (3) | 9 | Git version control, rollback procedure |

### 5.2 Risk Scoring Legend

| Score | Response |
|:-----:|----------|
| 20-25 | CRITICAL — immediate mitigation required |
| 12-19 | HIGH — mitigation within 1 week |
| 6-11 | MEDIUM — mitigation within 1 month |
| 1-5 | LOW — monitor and address in maintenance cycle |

### 5.3 Single Points of Failure

| Component | SPOF? | Mitigation |
|-----------|:-----:|------------|
| Primary VPS | YES | Standby VPS in different availability zone |
| PostgreSQL instance | YES | Streaming replication to standby (planned) |
| Nginx instance | YES | Quick rebuild from config |
| DNS provider | YES | Secondary DNS provider |

---

## 6. Backup Strategy

### 6.1 Local Backup (Primary DC)

#### PostgreSQL — Hourly WAL + Daily Full

```bash
# WAL archiving (continuous)
# Enabled in postgresql.conf:
#   wal_level = replica
#   archive_mode = on
#   archive_command = 'cp %p /backups/wal/%f'

# Daily full backup (cron: 0 2 * * *)
#!/bin/bash
BACKUP_DIR="/backups/postgres"
RETENTION_DAYS=30

pg_dump -U pat_admin -d predictatrade \
    --format=custom \
    --compress=9 \
    --file="${BACKUP_DIR}/predictatrade_$(date +%Y%m%d).dump"

# Rotate old backups
find "${BACKUP_DIR}" -name "*.dump" -mtime +${RETENTION_DAYS} -delete
```

#### Configuration Files — Daily

```bash
# cron: 0 1 * * *
#!/bin/bash
BACKUP_DIR="/backups/config"
tar -czf "${BACKUP_DIR}/config_$(date +%Y%m%d).tar.gz" \
    /srv/predictatrade/xauusd/docker-compose.yml \
    /srv/predictatrade/xauusd/infra/env/ \
    /srv/predictatrade/xauusd/infra/nginx/ \
    /srv/predictatrade/xauusd/.env 2>/dev/null
```

#### SSL Certificates — Weekly

```bash
# cron: 0 3 * * 0
#!/bin/bash
tar -czf "/backups/ssl/certs_$(date +%Y%m%d).tar.gz" \
    /etc/letsencrypt/
```

### 6.2 Cloud Backup (Offsite — Hetzner S3, ACTIVE)

> **v1.18.0 status:** live and verified. The `pat-backup-sync` sidecar
> (docker-compose.yml) ships WAL segments and 6-hourly logical dumps to
> `s3://pat-backup` at `hel1.your-objectstorage.com` every 60 seconds —
> no cron needed. Credentials/config: `infra/env/.env` (`BACKUP_S3_*`).
> Operational detail, verify/restore commands: see
> [BACKUP_RESTORE.md §8](BACKUP_RESTORE.md#8-off-host-backup-hetzner-s3--active).

```bash
# The sidecar covers this continuously; the manual equivalent is:
S3_BUCKET="s3://pat-backup/predictatrade"
ENDPOINT="https://hel1.your-objectstorage.com"   # from BACKUP_S3_ENDPOINT

# Sync PostgreSQL dumps (sidecar: /pgbackups -> predictatrade/db)
aws s3 sync /var/backups/predictatrade/ "${S3_BUCKET}/db/" \
    --exclude '*' --include 'backup_2*.dump' --include 'backup_2*.sha256' \
    --endpoint-url "${ENDPOINT}"

# Sync WAL archives (sidecar: /pgdata/wal_archive -> predictatrade/wal)
aws s3 sync /var/lib/docker/volumes/xauusd_pat-pgdata/_data/wal_archive/ \
    "${S3_BUCKET}/wal/" --endpoint-url "${ENDPOINT}"
```

### 6.3 Offsite Backup (Alternative Provider — Google Cloud Storage)

```bash
# Weekly offsite copy to GCS (cron: 0 5 * * 0)
#!/bin/bash
GCS_BUCKET="gs://predictatrade-dr-backups"

# Latest daily dump to GCS
LATEST=$(ls -t /backups/postgres/predictatrade_*.dump | head -1)
gsutil -o "GSUtil:parallel_composite_upload_threshold=150M" \
    cp "${LATEST}" "${GCS_BUCKET}/weekly/"
```

### 6.4 Immutable Cold Storage (Monthly)

```bash
# Monthly archive to S3 Glacier Deep Archive (cron: 0 6 1 * *)
#!/bin/bash
S3_GLACIER="s3://predictatrade-archive/monthly"

# Create monthly archive
MONTHLY_FILE="/backups/postgres/predictatrade_monthly_$(date +%Y%m).dump"
pg_dump -U pat_admin -d predictatrade \
    --format=custom --compress=9 \
    --file="${MONTHLY_FILE}"

# Upload to Glacier
aws s3 cp "${MONTHLY_FILE}" "${S3_GLACIER}/" \
    --storage-class GLACIER \
    --sse AES256

# Set legal hold (immutability)
aws s3api put-object-legal-hold \
    --bucket predictatrade-archive \
    --key "monthly/$(basename ${MONTHLY_FILE})" \
    --legal-hold Status=ON
```

### 6.5 Backup Schedule Summary

| Backup | Frequency | Retention | Storage | Encryption |
|--------|:---------:|:---------:|---------|:----------:|
| WAL archiving | Continuous | Local archive | Local + Hetzner S3 (`predictatrade/wal`) | TLS in transit |
| Full DB dump | Every 6 hours | 30 days local | Local + Hetzner S3 (`predictatrade/db`) | TLS in transit |
| Offsite (GCS) | Weekly (planned) | 52 weeks | GCS | GCS-managed |
| Immutable archive | Monthly (planned) | 7 years | S3 Glacier | AES256 + legal hold |

Note: the GCS and Glacier rows are future hardening options — Hetzner S3 is
the only off-host destination currently configured.

---

## 7. Recovery Procedures

### 7.1 Full System Recovery (from scratch)

**Prerequisites:**
- New VPS provisioned with Ubuntu 22.04+
- Docker + Docker Compose installed
- Git access to repository
- S3/GCS credentials configured
- API keys available (from secure vault)

**Step-by-step recovery (~2 hours):**

```bash
# 1. Install dependencies
sudo apt update && sudo apt install -y docker.io docker-compose-v2 git awscli

# 2. Clone repository
git clone https://github.com/simhaonline/predictatrade.git /srv/predictatrade/xauusd
cd /srv/predictatrade/xauusd

# 3. Download latest backup from S3
aws s3 cp s3://predictatrade-backups/production/postgres/ \
    /backups/postgres/ --recursive
LATEST_DUMP=$(ls -t /backups/postgres/predictatrade_*.dump | head -1)

# 4. Restore SSL certificates
aws s3 cp s3://predictatrade-backups/production/ssl/ \
    /etc/letsencrypt/ --recursive

# 5. Start database only
docker compose up -d postgres
sleep 10

# 6. Restore database
docker exec -i pat-postgres pg_restore \
    -U pat_admin -d predictatrade \
    --clean --if-exists --no-owner \
    < "${LATEST_DUMP}"

# 7. Apply any migrations since backup
for f in database/migrations/*.sql; do
    docker exec -i pat-postgres psql \
        -U pat_admin -d predictatrade < "$f"
done

# 8. Restore environment files
aws s3 cp s3://predictatrade-backups/production/config/ \
    /tmp/config-restore/ --recursive
tar -xzf /tmp/config-restore/config_*.tar.gz -C /

# 9. Start all services
docker compose up -d

# 10. Verify
curl http://localhost:13081/health
curl http://localhost:13080/api/v1/health
docker compose ps
```

### 7.2 PostgreSQL Point-in-Time Recovery

```bash
# 1. Restore base backup
pg_restore -U pat_admin -d predictatrade base_backup.dump

# 2. Create recovery.conf
cat > /var/lib/postgresql/data/recovery.conf <<EOF
restore_command = 'cp /backups/wal/%f %p'
recovery_target_time = '2026-08-26 14:30:00 UTC'
recovery_target_action = 'promote'
EOF

# 3. Restart PostgreSQL
docker compose restart postgres
```

### 7.3 Single Service Recovery

```bash
# Rebuild and restart one service
docker compose up -d --build realtime
docker compose restart frontend

# Check health
curl http://localhost:13081/health
```

### 7.4 Emergency Rollback

```bash
# Rollback to last known good commit
cd /srv/predictatrade/xauusd
git log --oneline -5              # Find the good commit
git reset --hard <good_commit_sha>
docker compose up -d --build
```

---

## 8. Testing Schedule

### 8.1 Recovery Test Types

| Test | Frequency | Scope | Duration | Owner |
|------|:---------:|-------|:--------:|-------|
| Backup verification | Daily | Check backup files exist, correct size | 5 min | Automated |
| Restore to test instance | Monthly | Full DB restore to isolated test | 1 hour | DevOps |
| Single service recovery | Quarterly | Rebuild + restart one service | 15 min | DevOps |
| Full DR simulation | Bi-annual | Full system recovery on clean host | 4 hours | DevOps + CTO |
| Offsite restore test | Annual | Restore from S3 Glacier to new region | 8 hours | DevOps |

### 8.2 Test Procedure — Monthly Restore

```bash
#!/bin/bash
# Monthly restore test script
set -e

TEST_DIR="/tmp/dr-test-$(date +%Y%m%d)"
mkdir -p "${TEST_DIR}"

echo "=== DR TEST: Monthly DB Restore ==="
echo "Date: $(date)"
echo "Test ID: DR-MONTHLY-$(date +%Y%m)"

# 1. Download latest backup
LATEST=$(aws s3 ls s3://predictatrade-backups/production/postgres/ | sort | tail -1 | awk '{print $4}')
aws s3 cp "s3://predictatrade-backups/production/postgres/${LATEST}" "${TEST_DIR}/"

# 2. Start test PostgreSQL
docker run -d --name pat-postgres-test \
    -e POSTGRES_USER=pat_admin \
    -e POSTGRES_PASSWORD=test \
    -e POSTGRES_DB=predictatrade \
    -p 15432:5432 \
    timescale/timescaledb-ha:pg17

sleep 15

# 3. Restore
docker exec -i pat-postgres-test pg_restore \
    -U pat_admin -d predictatrade \
    < "${TEST_DIR}/${LATEST}"

# 4. Verify
TABLES=$(docker exec pat-postgres-test psql -U pat_admin -d predictatrade \
    -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema NOT IN ('pg_catalog','information_schema');")

echo "Tables restored: ${TABLES}"

# 5. Cleanup
docker rm -f pat-postgres-test
rm -rf "${TEST_DIR}"

echo "=== DR TEST: PASS ==="
```

### 8.3 Test Log Template

```
DR TEST LOG
============
Test Date: ___________
Test Type: [ ] Monthly Restore  [ ] Quarterly Service  [ ] Bi-annual Full DR  [ ] Annual Offsite
Test ID: DR-___________
Performed By: ___________

Pre-Test Checks:
[ ] Backup files accessible
[ ] Restore procedure reviewed
[ ] Test environment isolated
[ ] Rollback plan ready

Test Results:
[ ] PASS — All checks passed
[ ] PASS WITH ISSUES — Restored but ___________
[ ] FAIL — ___________

Issues Found:
1. ___________
2. ___________

Corrective Actions:
1. ___________
2. ___________

Sign-off: ___________  Date: ___________
```

---

## 9. Monitoring & Alerting

### 9.1 Backup Monitoring

| Check | Method | Alert Threshold | Channel |
|-------|--------|:---------------:|---------|
| Last backup age | File timestamp | > 25 hours | Email + Telegram |
| Backup file size | File stat | < 100 MB or > 50% change | Email |
| WAL archive lag | pg_stat_archiver | > 5 minutes behind | Email + Telegram |
| S3 sync success | aws s3 ls exit code | Exit != 0 | Email |
| Disk space | df -h | < 20% free | Email + Telegram |

### 9.2 Service Health Monitoring

| Service | Check | Interval | Alert |
|---------|-------|:--------:|-------|
| PostgreSQL | pg_isready | 30s | Service down > 2m |
| Realtime Engine | HTTP /health | 30s | Non-200 > 2m |
| Control Plane | HTTP /api/v1/health | 30s | Non-200 > 2m |
| Frontend | HTTP / | 30s | Non-200/307 > 2m |
| Nginx | systemctl is-active | 60s | Inactive |
| Docker daemon | docker info | 60s | Not running |

### 9.3 Prometheus Alert Rules

```yaml
# infra/prometheus/rules.yml
groups:
  - name: disaster_recovery
    rules:
      - alert: BackupAgeTooHigh
        expr: time() - node_file_mtime_seconds{file=~".*backups.*"} > 90000
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Last backup is more than 25 hours old"

      - alert: DatabaseDown
        expr: pg_up == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "PostgreSQL is not responding"

      - alert: DiskSpaceLow
        expr: (node_filesystem_avail_bytes / node_filesystem_size_bytes) < 0.2
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Disk space below 20%"
```

---

## 10. Escalation Procedures

### 10.1 Incident Severity Levels

| Level | Definition | Response Time | Escalation |
|:-----:|------------|:------------:|------------|
| L1 | Service degradation, no data loss | 4 hours | DevOps → Team Lead |
| L2 | Single service down, partial impact | 1 hour | DevOps → CTO |
| L3 | Multiple services down, business impact | 30 minutes | DevOps → CTO → CEO |
| L4 | Data loss or security breach | 15 minutes | DevOps → CTO → CEO → Legal |

### 10.2 Escalation Contact Matrix

| Role | L1 | L2 | L3 | L4 |
|------|:--:|:--:|:--:|:--:|
| DevOps Engineer | ✓ | ✓ | ✓ | ✓ |
| Infrastructure Lead | — | ✓ | ✓ | ✓ |
| CTO | — | ✓ | ✓ | ✓ |
| CEO | — | — | ✓ | ✓ |
| Legal Counsel | — | — | — | ✓ |
| DPO (Data Protection) | — | — | — | ✓ |

### 10.3 Communication Plan

| Audience | Channel | Frequency | Content |
|----------|---------|:---------:|---------|
| Internal team | Slack/Teams | Real-time | Technical updates |
| Users | Status page | Hourly | Service status |
| Premium clients | Email | Per SLA | Personalized update |
| Public | Twitter/X | As needed | Brief status |

---

## 11. Disaster Declaration Criteria

The DR plan is activated when **any one** of the following occurs:

1. PostgreSQL database is unrecoverable from local backups
2. Primary host is unresponsive for > 30 minutes
3. Data corruption detected in billing or financial ledgers
4. Security breach with confirmed data exfiltration
5. Physical disaster at primary data center
6. Ransomware detected on any production system

### Declaration Authority

| Condition | Authority |
|-----------|-----------|
| Technical failure (1-2) | CTO or Infrastructure Lead |
| Data integrity (3) | CTO + CEO |
| Security incident (4) | CTO + CEO + Legal |
| Physical disaster (5) | CEO |
| Ransomware (6) | CTO + CEO + Legal |

---

## 12. Post-Recovery Procedures

### 12.1 Recovery Verification Checklist

After any recovery operation, verify:

- [ ] All 11 Docker services running (`docker compose ps`)
- [ ] PostgreSQL accepting connections (`pg_isready`)
- [ ] All health endpoints returning 200
- [ ] Signal generation active (check for new signals in last 5 min)
- [ ] WebSocket connections functioning
- [ ] Admin dashboard accessible
- [ ] User dashboard accessible
- [ ] SSL certificates valid
- [ ] API endpoints responding
- [ ] Payment webhooks receiving (Stripe + NOWPayments)

### 12.2 Root Cause Analysis

Within 48 hours of recovery, document:
1. Timeline of events
2. Root cause
3. What failed and why
4. What worked as expected
5. What could have prevented the incident
6. Action items with owners and deadlines

### 12.3 Plan Update

After every DR test or real incident:
1. Update this document with lessons learned
2. Revise RTO/RPO if not achievable
3. Update contact matrix
4. Bump document version
5. Distribute to all stakeholders

---

## 13. Appendices

### Appendix A: Required Tools & Credentials

| Tool | Purpose | Access Method |
|------|---------|---------------|
| AWS CLI | S3 backup access | IAM access key + secret |
| gsutil | GCS backup access | Service account JSON |
| Docker | Container management | Host root/sudo |
| Git | Source code | SSH key or token |
| psql | Database operations | pat_admin credentials |
| valkey-cli | Cache operations | Local socket |
| certbot | SSL management | Root access |

### Appendix B: Quick Reference Commands

```bash
# Check all backups exist
ls -lh /backups/postgres/
ls -lh /backups/wal/
aws s3 ls s3://predictatrade-backups/production/

# Database health
docker exec pat-postgres pg_isready -U pat_admin -d predictatrade
docker exec pat-postgres psql -U pat_admin -d predictatrade -c "SELECT now() - last_archived_time FROM pg_stat_archiver;"

# Service health
curl -s http://localhost:13081/health | jq .
curl -s http://localhost:13080/api/v1/health | jq .
docker compose ps --format "table {{.Name}}\t{{.Status}}\t{{.Health}}"

# Disk space
df -h / /backups /var/lib/docker

# Manual backup trigger
pg_dump -U pat_admin -d predictatrade --format=custom --compress=9 \
    --file="/backups/postgres/predictatrade_manual_$(date +%Y%m%d_%H%M%S).dump"
```

### Appendix C: DR Runbook Location

- **This document:** `/srv/predictatrade/xauusd/docs/operations/DR_PLAN.md`
- **Backup scripts:** `/srv/predictatrade/xauusd/scripts/backup/`
- **Recovery scripts:** `/srv/predictatrade/xauusd/scripts/recovery/`
- **Monitoring rules:** `/srv/predictatrade/xauusd/infra/prometheus/rules.yml`
- **Contact list:** Secure vault (not in repository)

---

**Document Version:** 1.0.0
**Last Updated:** 26 August 2026
**Next Review:** 26 November 2026
