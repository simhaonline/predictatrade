# Client-EA Attach Checklist (operator)

Goal: exec-role client EAs come ONLINE so signals dequeue and TRADE_RESULTs
flow. This is the binding constraint on Phase 1 (N≥300/strategy). Owner:
operator. Engineering contact: repo issues.

## 1. Pre-checks (server side, one command)

```
bash scripts/operator_status.sh
```
Confirm:
- ENGINE HEALTH: status ok, db ok, cache ok
- EDGE DEVICES: your client devices appear; note their status (OFFLINE expected now)
- SIGNAL ENQUEUE: enqueued>0 (signals are being produced — they expire while EAs are offline)

## 2. On the client terminal (Windows, MT5)

Follow `docs/guides/EA_CLIENT_GUIDE.md` for the standard install. In brief:
1. Install/verify the Client EA build (latest .ex5 from nginx/downloads or recompiled with patch 0001 — see `tools/ea/README.md`).
2. Enter license key + server URL (api.predictatrade.com) in EA inputs.
3. AutoTrading ON. Exec role requires the account the license is bound to.

## 3. Verify attach (server side, one command)

```
bash scripts/operator_status.sh
```
Expected after attach:
- EDGE DEVICES BY ROLE: your device flips to ONLINE (exec)
- STALE EXEC DEVICES: your device leaves that list (last_seen fresh)
- SIGNAL ENQUEUE: `acked` grows (deliveries now consumed)
- TRADE_RESULT RATE: >0 after the first trade closes
- OUTCOMES WRITTEN LAST 24H: grows with LINKED rows

## 4. Verify outcome linkage (EA trading on a real account)

```
docker exec pat-postgres psql -U pat_admin -d predictatrade -c \
  "SELECT link_status, count(*) FROM trading.prediction_outcomes \
   WHERE created_at > now() - interval '2 hours' GROUP BY 1;"
```
Expected: LINKED count grows; UNLINKED stays ~0 (only legacy rows).

## 5. Rollback / detach

Remove the EA from the chart; device watchdog flips OFFLINE within
`DEVICE_STALE_MS`; enqueued signals resume expiring (TTL). No server changes.

## 6. Common failures

| Symptom | Cause | Fix |
|---|---|---|
| Device stays OFFLINE | wrong license key / revoked | check REVOKED list; re-activate license |
| ONLINE but no ACKs | device role not exec | confirm role in licensing.devices (this script lists it) |
| TRADE_RESULTs but UNLINKED high | old fallback-UUID signal ids | expected only on legacy rows; alert clears as live rows dominate |