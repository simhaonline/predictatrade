# Incident Response Plan

## Predict-A-Trade v1.16.0 — 26 August 2026

### 1. Purpose and Scope

This document defines the incident response procedures for the Predict-A-Trade
platform. It covers detection, classification, response, and post-mortem for
security incidents, service outages, and data integrity events across all four
planes: Go real-time engine, NestJS control plane, Next.js frontend, and
Python research plane.

### 2. Incident Classification

| Severity | Definition | Response Time | Example |
|----------|-----------|:---:|---------|
| **P0 — Critical** | Live trading at risk, data loss, breach | 15 min | Unauthorized order execution, database corruption |
| **P1 — High** | Service degradation, partial outage | 1 hour | Single service down, stale signals > 5 min |
| **P2 — Medium** | Non-critical feature broken | 4 hours | Reports unavailable, UI glitch |
| **P3 — Low** | Cosmetic, no user impact | 24 hours | Log warning, minor UI issue |

### 3. Incident Response Team

| Role | Responsibility |
|------|---------------|
| **Incident Commander (IC)** | Owns the incident, coordinates response |
| **Operations Lead** | Service health, restarts, rollbacks |
| **Security Lead** | Breach containment, forensic collection |
| **Communications Lead** | Status page updates, user notifications |

### 4. Detection Sources

| Source | What It Detects | Alert Channel |
|--------|----------------|---------------|
| Prometheus + Alertmanager | Service health, latency, error rates | ntfy push + Grafana |
| pat-realtime health endpoint | Engine liveness | Prometheus blackbox |
| pat-control health endpoint | API availability | Prometheus blackbox |
| Go engine internal metrics | Signal gaps, gate rejections, stale data | Prometheus metrics |
| Docker health checks | Container crashes | System logs |
| PostgreSQL logs | Slow queries, connection exhaustion | pgBadger reports |
| nginx access/error logs | 5xx spikes, DDoS patterns | Log aggregation |

### 5. Response Procedure

#### 5.1 Declare and Triage (0-15 min)

1. **Acknowledge alert** — IC acknowledges the alert in the incident channel.
2. **Classify severity** — Determine P0/P1/P2/P3 using the classification table.
3. **Assemble team** — Notify relevant leads. For P0, all hands on deck.
4. **Open incident** — Create an incident entry with timestamp, severity, and
   initial observations.

#### 5.2 Contain and Mitigate (15 min — 2 hours)

| Incident Type | Containment Action |
|---------------|-------------------|
| Unauthorized trading | Immediately stop the realtime engine: `docker compose stop realtime` |
| Data corruption | Isolate affected tables, halt writes, restore from backup |
| Credential leak | Rotate all affected keys/secrets immediately |
| DDoS / traffic spike | Enable nginx rate limiting, scale up if possible |
| Service crash | Restart service via `docker compose restart <service>` |
| Disk full | Delete old logs, prune Docker images: `docker system prune -a` |
| Database exhaustion | Kill long-running queries, increase connection pool timeout |

#### 5.3 Investigate and Root Cause

1. **Collect evidence**: logs, metrics, database snapshots, core dumps.
2. **Preserve forensic data**: copy logs to secure storage before cleanup.
3. **Determine root cause**: trace the incident to its origin.
4. **Document timeline**: what happened, when, and what was affected.

#### 5.4 Recover and Verify

1. **Restore service**: bring affected components back online.
2. **Verify integrity**: run health checks, validate data consistency.
3. **Monitor**: watch for 30 minutes post-recovery for recurrence.
4. **Communicate**: update status page, notify affected users.

#### 5.5 Post-Mortem (within 48 hours)

1. **Write incident report**: timeline, root cause, impact, resolution.
2. **Identify preventive measures**: what changes prevent recurrence.
3. **Assign action items**: with owners and deadlines.
4. **Update runbooks and this document**: reflect lessons learned.

### 6. Service-Specific Recovery

#### 6.1 Go Real-Time Engine (pat-realtime)

```bash
# Stop
docker compose stop realtime

# Check logs
docker compose logs --tail=200 realtime

# Restart
docker compose up -d realtime

# Verify
curl -s http://localhost:13081/health
```

Rollback procedure:
```bash
docker compose stop realtime
git checkout <last-known-good-commit> -- realtime/
docker compose build realtime
docker compose up -d realtime
```

#### 6.2 NestJS Control Plane (pat-control)

```bash
docker compose stop control
docker compose logs --tail=200 control
docker compose up -d control
curl -s http://localhost:13080/api/v1/health
```

#### 6.3 PostgreSQL Database

```bash
# Check health
docker exec pat-postgres pg_isready -U pat_admin

# Kill blocking queries
docker exec pat-postgres psql -U pat_admin -d predictatrade \
  -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE state = 'active' AND query_start < now() - interval '5 minutes';"

# Restore from backup (see docs/operations/BACKUP_RESTORE.md)
```

#### 6.4 Full Platform Recovery

```bash
docker compose down
docker compose up -d --build
docker compose ps  # verify all healthy
```

### 7. Communication Templates

#### Status Page Update (P0/P1)

```
Investigating: We are investigating reports of [ISSUE] affecting [SERVICE].
Start: [TIMESTAMP UTC]
Impact: [DESCRIPTION]
Next update: [TIME + 30 min]
```

#### Post-Incident Notification

```
Resolved: [ISSUE] has been resolved.
Duration: [START] — [END] UTC
Root Cause: [BRIEF DESCRIPTION]
Action: [WHAT WAS DONE]
Prevention: [WHAT CHANGES PREVENT RECURRENCE]
```

### 8. Key Contacts and Escalation

| Role | Contact |
|------|---------|
| Primary Operator | ops@predictatrade.com |
| Emergency | Via ntfy push notification channel |
| Hosting Provider | Hetzner Cloud Console / API |

### 9. Compliance Alignment

This plan aligns with:
- **ISO 27001** A.16 — Information security incident management
- **NIST CSF** — Respond (RS) and Recover (RC) functions
- **SOC 2** — CC7.3-CC7.5 (incident detection, response, recovery)

### 10. Plan Maintenance

- **Review cycle**: Quarterly (or after every P0/P1 incident)
- **Test cycle**: Annual tabletop exercise
- **Owner**: Operations Lead
- **Version**: 1.0.0 (26 August 2026)

### Appendix A: Quick Reference Card

```
P0 CRITICAL — 15 min response
  → Stop trading engine FIRST
  → Assemble all leads
  → Contain, then investigate

P1 HIGH — 1 hour response
  → Restart affected service
  → Check logs and metrics
  → Escalate if > 1 hour

P2 MEDIUM — 4 hour response
  → Diagnose during business hours
  → Fix or workaround
  → No user communication needed

P3 LOW — 24 hour response
  → Log ticket
  → Fix in next maintenance window
```
