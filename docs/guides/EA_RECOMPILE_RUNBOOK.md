# EA Recompile + Rollout Runbook (MT5 client/master EAs)

Scope: apply patch 0001 (MAE/MFE), recompile, smoke-test on demo, confirm
server receives non-zero excursions, roll out to remaining terminals, roll back
if needed. Owner: operator. Estimated time: 20 min/terminal.

## Pre-flight (one time)

| # | Step | Verify |
|---|---|---|
| P1 | Pull latest repo on the terminal machine (or copy `tools/ea/0001-fix-mae-mfe-tracking.patch` over) | `ls tools/ea/0001-fix-mae-mfe-tracking.patch` |
| P2 | Confirm engine is healthy BEFORE rollout (so failures are attributable) | `bash scripts/operator_status.sh` → engine health ok |

## Per-terminal procedure

1. **Close chart instances of the EA** (drag EA off chart or AutoTrading off).
2. **Backup**: `cp PredictATrade_MT5.mq5 PredictATrade_MT5.mq5.bak`
3. **Apply patch**: `cd mql/mt5 && patch -p1 --dry-run -i ../../tools/ea/0001-fix-mae-mfe-tracking.patch && patch -p1 -i ../../tools/ea/0001-fix-mae-mfe-tracking.patch`
   - Expected: `patching file mql/mt5/PredictATrade_MT5.mq5` (no rejects).
   - If rejects: STOP, restore .bak, escalate (file changed since patch).
4. **Recompile** MetaEditor → open `PredictATrade_MT5.mq5` → F7.
   - Expected: `0 errors` (this file), .ex5 timestamp updated.
5. **Re-attach** EA to the XAUUSD chart (same inputs as before).
6. **Smoke-test on demo** (demo account only):
   - Wait for/trigger one demo trade; let it close.
   - MT5 Experts log: one `TRADE_RESULT reported: ...` line per close.
7. **Server verification** (origin):
   ```
   docker exec pat-postgres psql -U pat_admin -d predictatrade -c \
     "SELECT close_reason, mae_points, mfe_points, created_at \
      FROM trading.prediction_outcomes WHERE created_at > now() - interval '1 hour' \
      ORDER BY created_at DESC LIMIT 5;"
   ```
   Expected: mae/mfe non-zero (or 0 only if price never moved off entry).

## Rollout order

1. One demo terminal (this procedure) → verify steps 6-7.
2. Master-node terminals (data role) — no trade behavior, but recompile for
   consistency (patch is telemetry-only; skipping masters is acceptable).
3. Client terminals (exec role) — the critical rollout for outcome data.

## Rollback

- Per terminal: `cp PredictATrade_MT5.mq5.bak PredictATrade_MT5.mq5`, recompile (F7), re-attach.
- No server changes to revert. Old binary sending mae:0.0 remains accepted.

## Common failures

| Symptom | Cause | Fix |
|---|---|---|
| patch rejects | source drifted since patch | re-diff; escalate to engineering |
| 0 values persist post-patch | position not PAT-magic or UpdateExcursions not reached | Experts log excerpt → engineering |
| TRADE_RESULT not arriving | EA ingest path down | `operator_status.sh` → EA offline section of runbook |