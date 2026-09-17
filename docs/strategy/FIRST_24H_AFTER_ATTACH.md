# First-24h Playbook — After the Operator Attaches EAs
Phase 0.95 Task D · Owner: SRE/operator · Status: playbook committed; T-0 executable now

## T-0 checklist (run immediately before/after attach)

```
bash scripts/operator_status.sh
```
| Check | Expected | "Wrong" looks like |
|---|---|---|
| Engine health | status ok, db ok, cache ok | db down → runbook §8 |
| Schema guard | outcome_pipeline.schema_guard=verified | drift → runbook §6 (apply migration) |
| EA patch version | recompiled .ex5 newer than patch date; demo smoke-test shows non-zero MAE/MFE | all mae/mfe zero → EA not patched |
| EA role/magic mapping | device ONLINE with role=exec | device ONLINE but role≠exec → license/role mismatch |
| Entitlements | device's licenses ACTIVE | PENDING/REVOKED → control-plane fix |
| Caps | risk caps in engine env (MAX_SL_USD etc.) unchanged | unexpected env drift → investigate before trading |

## T+0 → T+1h (the first-mechanics watch)

Re-run `operator_status.sh` every ~10 min. Milestones in order:

1. **First signal enqueue** (min 0-15): SIGNAL ENQUEUE 'enqueued' grows; EA ACKs
   appear (`acked` grows). Wrong: enqueued grows but acked stays 0 → delivery
   problem (device role, license, or EA poll).
2. **First predictions row** (min ~1): predictions_total grows at emit time —
   if outcomes grow but predictions don't, the emit-side writer is broken
   (alert 3 fires: OUTCOMES_MISSING).
3. **First TRADE_RESULT** (first close): TRADE_RESULT RATE > 0. Wrong: >30 min
   of enqueued+acked signals with no TRADE_RESULT after any close → check EA
   Experts log + ingest.
4. **First outcome row**: OUTCOMES WRITTEN LAST 24H grows with LINKED=100% of
   new rows. Wrong: UNLINKED on NEW rows → signal_id mismatch (stop, root-cause
   before continuing).

## T+1h → T+24h watch

- **N growth**: per-strategy N in the FIRST-24H section (est. rates: STANDARD_SCALPING ~47/day on a trading day).
- **UNLINKED ratio (last 1h)**: must stay ~0 for new rows; legacy share dilutes daily.
- **MAE/MFE non-zero ratio (last 1h)**: >80% after EA patch (0% → patch not applied/recompiled).
- **Writer silence**: /health outcome_pipeline.outcome_writer must return ok during market hours; SILENT while signals emit → Alert 1.
- **Gate-veto reasons**: metrics pack §5 close-mix; sudden veto-pattern shifts → regime/market-data drift.
- **Deadlock detector**: gate evals per minute steady (Phase 0 metrics); flat 10+ min → runbook §3.
- **DB pool / disk**: health db ok; disk <80%.
- **Alert fatigue**: all 4 alerts should progressively self-clear (UNLINKED falls, writer resumes). Any alert that stays firing 24h with healthy inputs → investigate, don't silence.

## Kill-switch procedures (exact commands)

**Pause ALL signals (server-authoritative halt):**
```
curl -X POST http://localhost:13081/api/v1/admin/emergency-halt -H 'Content-Type: application/json' -d '{"level":"FULL","reason":"first24h-kill","active":true}'
# rollback: same with active:false
```
**Pause outcomes only (capture without blocking signal truth):** stop the
outcome goroutine is not flag-gated — instead stop the EA (AutoTrading off);
outcomes stop flowing with trades. DB-side: outcome writes are fail-open and
never block signal truth.

**Pause ONE strategy:** engine restart with the strategy disabled via its env
flag (STRATEGY_<NAME>_ENABLED=false in infra/env/.env + recreate) — rollback is
re-enabling the flag. (No signal-behavior change otherwise.)

**Full engine rollback:** `git checkout <prev> && docker compose build realtime && docker compose --env-file infra/env/.env up -d --force-recreate realtime`.

## False-alarm guarantee (verified live)

On today's EA-off system the playbook's FIRST-24H section reports "no EAs
attached" (STALE EXEC DEVICES lists all, TRADE_RESULT rate 0) and the expected
alerts are the known legacy ones (UNLINKED legacy ratio, shadow legacy stall) —
documented, self-clearing, NOT false alarms (each has a real, explained cause).
Verification evidence: operator_status.sh output in PHASE0_95_AUDIT.md §D.