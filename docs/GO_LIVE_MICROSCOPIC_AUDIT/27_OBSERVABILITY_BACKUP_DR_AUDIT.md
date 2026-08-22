# 27 — Observability / Backup / DR Audit

## Observability

- Prometheus scraping engine `/metrics` (25 `pat_*` metrics observed: regime age/confidence, cooldown/duplicate errors, entitlement denials, daily-limit hits, WS connections, ML fallback...).
- Grafana up (provisioned dashboards in `infra/grafana`) — dashboard content coverage spot-checked only; admin password committed (26-2).
- Structured JSON logs from realtime engine (verified fields level/service/version/time); NestJS logger structured-ish.
- **Gaps:** no alerting rules verified (no Alertmanager service); OTel collector configured (`infra/otel`) but no trace evidence sampled; frontend WS client drops envelope sequence (breaks client-side gap alarms); health semantics partially honest — Go `/health` = process+agents count, `/ready` = DB ping ✅, but control-plane self-health is hardcoded HEALTHY (02-6), and "Market Feed ONLINE" is not explicitly labeled during weekend close (06).

## Backup

```
/srv/backups/: last dumps Aug 18 (3.4MB each) — 4+ days stale
backup_20260818_094550_UTC.dump = 0 bytes (failed run kept)
offhost_backup.log exists (off-host copy attempted once)
crontab: NO scheduled pg_dump — only verify_live_production.sh (5min) + weekly training
```

**Verdict: backups exist as artifacts, NOT as a working process** (§67 explicitly rejects this). Timescale hypertables + 7.1M ticks not covered by any current schedule. Restore procedure documented? `docs/operations` has runbooks — restore drill NOT performed in audit ⇒ UNVERIFIED.

## DR

RPO: undefined (no schedule). RTO: undefined. VPS-loss recovery would require: fresh host, compose up, restore latest dump (Aug 18 → **5 days of financial/trading data loss** at time of audit), reissue certs, rotate secrets. No tested runbook evidence.

## Failure-mode behavior (chaos-lite observations)

- Weekend close: engine stable, news sync continued, no false staleness alarms — PASS.
- Valkey-down dedup fail-open (05-1) and control-plane DB-failure gates→UNKNOWN=fail-closed (Go) — mixed posture.
