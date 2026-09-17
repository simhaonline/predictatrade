# LIVE_OPERATIONS_RUNBOOK.md — Predict-A-Trade live operations
Phase 0.9 Task B.8 · Owner: SRE/operator · Updated: 2026-09-17

## 1. Deploy (engine)
```
cd /srv/predictatrade/xauusd
docker compose build realtime && docker compose --env-file infra/env/.env up -d --no-deps --force-recreate realtime
```
Verify (expected output):
```
curl -s http://localhost:13081/health | python3 -m json.tool
→ status ok, db ok, cache ok, outcome_pipeline.schema_guard verified
```
Rollback: `git checkout <previous-commit> && docker compose build realtime && up -d --force-recreate realtime`.

## 2. Rollback
Same as deploy with the previous git commit. DB migrations are additive; the
schema guard fails closed only when writer columns are missing (then apply the
pending migration instead of downgrading).

## 3. Deadlock detection
Symptom: gate evaluation counter flat for minutes (metrics pack §1/§7 + logs).
Check: `docker logs pat-realtime | grep -iE 'gate evals|deadlock'`;
deep: `curl -s localhost:13081/debug/pprof/goroutine?debug=1 | head -100`.
Fix: fixed at 44de001 (snapshot-then-evaluate). Regression test in
internal/gates must stay green. Recovery: engine restart.

## 4. EA offline (the known state since Sep 12)
```
bash scripts/operator_status.sh   → STALE EXEC DEVICES + enqueue/expiry rates
```
Signals enqueue and TTL-expire until a client EA attaches. Fix: operator attaches
client EAs (docs/guides/EA_ATTACH_CHECKLIST.md). Recovery automatic once device
ingest resumes (liveness write-through flips ONLINE).

## 5. Writer silence
/health `outcome_pipeline.outcome_writer: SILENT` (>30 min while signals emit)
+ ALERT 1 in scripts/phase0_5_alerts.sql. Diagnosis: EAs offline (most likely),
DB down, or TRADE_RESULT parse failures (check ingest logs for warn
"prediction_outcomes write failed"). Rollback: n/a (fail-open telemetry; signal truth unaffected).

## 6. Schema drift
Engine refuses to start with `SCHEMA DRIFT: trading.<table>.<column> missing`.
Fix: apply the pending migration (150+), restart. Never bypass the guard.

## 7. Migration failure
Migrations are recorded in audit.migration_history (110 rows; round-trip verified).
A failed write stops the migration runner with the SQL error. Recovery: fix SQL,
re-run; never hand-edit production tables.

## 8. DB down
/ready → 503 `database_unavailable`; engine degrades (ADVISORY-only) after the
90s connect budget (fail-closed). Recovery: postgres restart; the engine's connect
loop retries (30×3s) — `docker compose restart realtime` if it degraded.

## 9. Shadow stalled
ALERT 4 (stale unresolved > 50 beyond 24h). Expected noise: 130k legacy Aug-24/25
rows never resolve (no live stream then) — keep the alert but scope to NEW rows
only (`timestamp > '2026-09-16'` filter is the production version; legacy rows are
frozen). Recovery: engine restart re-arms the resolver.

## 10. Sufficiency gate status
```
docker exec -i pat-postgres psql -U pat_admin -d predictatrade < scripts/phase1_status.sql
docker exec -i pat-postgres psql -U pat_admin -d predictatrade < scripts/sample_sufficiency_gate.sql
```
Phase 1 tuning may begin ONLY when every target strategy shows gate=PASS
(≥300 linked EXECUTION outcomes). Currently BLOCKED (see PHASE0_75_REPORT.md).

## 11. Capacity & retention (proposals — NO deletions without approval)
- Disk: 66G/301G used (23%), 223G headroom.
- Growth: signal_feature_snapshots ≈ 3.7k/day (each ~2-4KB JSON) ≈ 15MB/day;
  prediction_outcomes ≈ hundreds/day max; shadow 138.8k rows total (frozen legacy).
- Proposals (require approval): (a) snapshot retention 180d then archive to R2
  (Timescale retention policy on the hypertable); (b) legacy shadow rows older
  than 2026-08-26 → move to an archive schema; (c) edge_signal_queue TTL sweep
  already prunes (16.8k acked/expired rows — candidate for 30d purge).
- Index bloat: monitor pg_stat_user_indexes; reindex candidate if
  idx_scan/idx_size ratio degrades (no action needed at current sizes).

## 12. One-command operator status
`bash scripts/operator_status.sh` (engine health, EA presence, enqueue/ack,
TRADE_RESULT rate, outcomes 24h, per-strategy gate N, UNLINKED ratio, writer silence).
