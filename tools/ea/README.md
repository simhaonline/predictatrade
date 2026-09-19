# EA Patch 0001 — MAE/MFE excursion tracking (fixes hardcoded 0.0)

> **Removed 2026-09-19:** the old `PredictATrade_MT5.mq5.patched` reference copy
> (v1.30 era) was deleted — it predated v1.32–v1.35 and mislead compiles.
> The live source of truth is `mql/mt5/PredictATrade_MT5.mq5` (v1.36) and
> `mql/mt5/PredictATrade_MasterNode_MT5.mq5` (v1.35.1) — recompile those.


> **EA v1.34 note (2026-09-18):** the patch hunks still apply cleanly, but the
> repo source now also carries v1.32 (HTTP transport hardening + retries),
> v1.33 (`AutoExecute=true` default, terminal AutoTrading self-diagnostics),
> and v1.34 (XAUUSD slippage thresholds 3→60 pts, per-strategy 60/60/80/100 —
> the old 3-pt guard instantly closed every gold trade at the spread).
> If your terminal binary is older than v1.34, a fresh recompile of the repo
> source is preferable to patching an old copy — the MAE/MFE fix and the
> transport/slippage fixes then arrive in one compile.
> See `README_v1_32_TRANSPORT_HARDENING.md` for the failure-mode text mapping.

Fixes `PredictATrade_MT5.mq5` TRADE_RESULT reporting: `mae`/`mfe` were hardcoded
`0.0` (server-side confirmed — every outcome row had mae/mfe NULL/0), making
excursion-based calibration impossible. This patch makes the EA track and emit
real per-trade Maximum Adverse / Favorable Excursion, in price AND in R.

Patch file: `tools/ea/0001-fix-mae-mfe-tracking.patch` (verified to apply with
`patch -p1 --dry-run` against the committed `mql/mt5/PredictATrade_MT5.mq5`).

## What the patch does (telemetry only — ZERO trading-behavior change)

1. Adds per-magic registry columns `g_regMAE/g_regMFE` (price distance) and
   `g_regMAE_R/g_regMFE_R` (normalized by initial risk |entry − sl0|), reset on
   `PAT_RegPut` (new trade) and after the final close emit.
2. `PAT_UpdateExcursions()` — new pure-telemetry function, called every tick per
   open PAT position inside `PAT_ManagePositions()`:
   - BUY: favorable = bid − entry, adverse = entry − bid
   - SELL: favorable = entry − ask, adverse = ask − entry
   - Running max preserved; R guarded against zero-risk (stays 0 — never fabricates).
3. `PAT_ReportResult()` gains 4 defaulted parameters (`p_mae, p_mfe, p_maeR,
   p_mfeR` = 0.0) — backward compatible with every existing call site — and
   emits `mae`, `mfe`, plus additive `mae_r`/`mfe_r` JSON fields. Older server
   versions ignore unknown JSON keys (verified: engine JSON parser reads the
   fields it knows).
4. `PAT_HistoryPoll()` final-close path passes the lifetime excursion snapshot
   then resets the trackers (trade lifecycle complete).

Server-side compatibility: `mae_r`/`mfe_r` are new keys; the Go ingest binds
`mae`/`mfe` today. `mae_r`/`mfe_r` will flow into `prediction_outcomes` when the
writer is extended (separate, flag-gated change — NOT in this phase).

## Apply (operator, MT5 terminal machine)

```
cd <repo>/mql/mt5
patch -p1 -i ../../tools/ea/0001-fix-mae-mfe-tracking.patch   # verify with --dry-run first
# or apply manually: the patch hunks are small and fully commented (v1.30 markers)
```

Then recompile in MetaEditor (F7) — must show `0 errors, 0 warnings` in the
toolchain output for this file (warnings elsewhere unchanged).

## Demo verification checklist (run ALL before production attach)

1. Attach patched EA to a demo XAUUSD chart; confirm OnInit prints
   `Predict-A-Trade MT5 EA v1.30` (bump the version string if you also apply the
   version marker — optional).
2. Open ONE demo trade from a signal (or manually with PAT magic) and let it hit
   TP or SL.
3. In MT5 Experts log confirm: `TRADE_RESULT reported: ...` appears once.
4. Server-side verification (origin, one command):

   docker exec pat-postgres psql -U pat_admin -d predictatrade -c \
   "SELECT signal_id, close_reason, mae_points, mfe_points, created_at \
    FROM trading.prediction_outcomes ORDER BY created_at DESC LIMIT 5;"

   EXPECTED: `mae_points`/`mfe_points` NON-ZERO on rows produced after the patch
   (values > 0 on a real excursion; 0 only if price never moved off entry).

5. If values remain 0.0 across several trades → check EA Experts log for
   `PAT_UpdateExcursions` coverage (the position must be PAT-magic) → escalate
   to engineering with the log excerpt.

## Rollback

- MT5: recompile the previous .mq5 (pre-patch copy is any release before this
  patch; keep `PredictATrade_MT5.mq5.bak` next to the patched file before applying).
- No server-side rollback needed: the engine accepts both 0.0 and real values.

## Owner & status

- Patch: engineering (committed) — Status: DELIVERED, awaiting operator recompile
- Demo verification: operator — Status: PENDING (see runbook)
- Impact of NOT applying: MAE/MFE stay 0/0 → excursion-based calibration
  features (Phase 1 secondary metrics, future stop/target research) unusable.