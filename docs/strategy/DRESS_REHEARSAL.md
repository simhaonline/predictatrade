# Dress Rehearsal — Real MT5 end-to-end (Phase 0.95 Task A)
Date: 2026-09-17 · Status: **BLOCKED — operator hardware required** (per prompt.md: stated explicitly, not simulated)

## Availability verdict (the honest answer first)

**Demo MT5 is NOT available on the origin host.** This server is a Linux box
(Hetzner origin, 2.29.23.42) with no wine/MT5 runtime, no MT5 containers, and
installing MT5 server-side is explicitly against the operator's architecture
(MT4/MT5 terminals are managed manually on his Windows machines — project
constraint, not a limitation I can lift). All 16 exec-role devices in
`licensing.devices` are OFFLINE; they live on operator Windows terminals.

Per prompt.md §Task A.6: the MT5 boundary is NOT simulated and NOT called a
rehearsal. Task A therefore is an OPERATOR-EXECUTED rehearsal with an
engineering-prepared kit; everything that can be prepared server-side has been.

## What engineering prepared (complete, waiting on operator)

| Artifact | Purpose |
|---|---|
| `tools/ea/0001-fix-mae-mfe-tracking.patch` + `tools/ea/README.md` | patched EA build + demo verification checklist |
| `docs/guides/EA_RECOMPILE_RUNBOOK.md` | per-terminal recompile + smoke-test + server verification + rollback |
| `docs/guides/EA_ATTACH_CHECKLIST.md` | attach procedure + server-side verification queries |
| `scripts/operator_status.sh` | one-command status the operator runs between rehearsal steps |
| `docs/strategy/REHEARSAL_OPERATOR_SCRIPT.md` (this phase) | the 20-signal rehearsal script with per-signal server verification (below) |

## Rehearsal procedure (operator, ~1-2h on a demo terminal)

0. **Pre-flight**: `bash scripts/operator_status.sh` → engine healthy, schema
   guard verified, device OFFLINE. Apply patch 0001, recompile (0 errors), attach
   to a DEMO XAUUSD chart → device flips ONLINE.
1. **Collect 20+ signals**: run through a trading session (or until 20 closes).
   Signals enqueue → EA ACKs → EA trades them per its normal logic (AUTO closes
   dominate; 2-3 MANUAL closes should be included deliberately — the runbook
   explains how: manual close of a live position).
2. **Per-signal verification** (server-side, single command):
   ```sql
   SELECT signal_id, close_reason, mae_points, mfe_points, r_multiple, link_status
   FROM trading.prediction_outcomes WHERE created_at > now() - interval '4 hours'
   ORDER BY created_at DESC;
   ```
   Expected per row: predictions row exists (signal_id joins), feature_snapshot
   links, MAE/MFE NON-ZERO (patch works), close_reason mapped (STOP/TP1/…),
   r_multiple computed, link_status=LINKED, UNLINKED = 0.
3. **Latency per hop** (measure with timestamps):
   - signal emit → enqueue: `licensing.edge_signal_queue.created_at - trading.signals.created_at`
   - enqueue → ACK: `acked_at - created_at`
   - ACK → fill: not recorded server-side (EA-side; note from terminal logs)
   - close → TRADE_RESULT: `trading.trade_results.created_at - (ack of entry)` (approx; documented limitation)
4. **Failure modes encountered**: log each in this file's §Failures table (T+0
   issues are expected on first attach — that is what the rehearsal is for).
5. **Verdict**: PASS requires 20/20 signals with zero UNLINKED rows and non-zero
   MAE/MFE on closed trades; any FAIL blocks Phase 1 start until resolved.

## Alternative if the operator cannot run it this week

Software-in-the-loop (Task B in this repo) covers the server-side half of every
failure mode and is COMPLETE — the residual unproven surface is strictly
"real MT5 terminal over a real network", which only the operator's hardware can
prove. The rehearsal remains the single outstanding pre-Phase-1 blocker on the
operator's side.

## Verification (server-side, after the operator runs it)

```bash
bash scripts/operator_status.sh   # FIRST-24H section + per-strategy N
```
`docs/strategy/DRESS_REHEARSAL.md` gets the operator's log excerpts/latency table
appended (owner: operator, reviewer: engineering).