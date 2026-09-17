# OPERATOR HANDOFF — Your 3 Actions
Date: 2026-09-17 · Engineering is finished and waiting. Everything below is on your side.

## Action 1 — Patch + recompile the EA (fixes MAE/MFE tracking)

- **File**: `tools/ea/0001-fix-mae-mfe-tracking.patch`
- **Guide**: `docs/guides/EA_RECOMPILE_RUNBOOK.md` (~20 min per terminal)
- **Do**:
  1. Back up your current `PredictATrade_MT5.mq5` (rename `.bak`).
  2. In the repo folder: `patch -p1 -i tools/ea/0001-fix-mae-mfe-tracking.patch`
  3. Open the file in MetaEditor, press F7 to recompile. It must show 0 errors.
  4. Re-attach the EA to your demo XAUUSD chart and let one demo trade close.
- **Confirm success** (run this on the server):
  ```
  docker exec pat-postgres psql -U pat_admin -d predictatrade -c \
    "SELECT close_reason, mae, mfe, created_at FROM trading.trade_results \
     ORDER BY created_at DESC LIMIT 5;"
  ```
  **Expected**: mae/mfe columns are NON-ZERO on new rows. (0.0 everywhere = patch not applied.)
- **If something breaks**: restore the `.bak` file, recompile, re-attach. Nothing else changes.
- **You are done when…** new trade rows show real (non-zero) MAE/MFE values.
- **Skip impact**: excursion data stays fake (all zeros) forever — future calibration research can't use it.

## Action 2 — Attach your client EAs (starts the data flowing)

- **Guide**: `docs/guides/EA_ATTACH_CHECKLIST.md` (~10 min)
- **Do**: install/attach the Client EA with the license key + server URL on the
  account(s) the license is bound to; AutoTrading ON.
- **Confirm success**:
  ```
  bash scripts/operator_status.sh
  ```
  **Expected, in order**: your device flips to ONLINE (exec) in the EDGE DEVICES
  section → `acked` starts growing in the SIGNAL ENQUEUE section → TRADE_RESULT
  RATE > 0 after your first trade closes → OUTCOMES WRITTEN grows with LINKED rows.
- **If something breaks**: the checklist has a failure table (wrong license, wrong role, etc.).
- **You are done when…** `operator_status.sh` shows your device ONLINE and a
  non-zero TRADE_RESULT rate within the first trading hour.
- **Skip impact**: signals keep enqueuing and expiring; Phase 1 can never start (this is THE blocker).

## Action 3 — (Recommended) 20-signal dress rehearsal on DEMO

- **Guide**: `docs/strategy/DRESS_REHEARSAL.md` (~1-2 h on a demo terminal)
- **Do**: with the patched EA attached to a demo account, trade 20 signals end-to-end
  (include 2-3 manual closes). Verify after each close with:
  ```
  docker exec pat-postgres psql -U pat_admin -d predictatrade -c \
    "SELECT signal_id, close_reason, mae, mfe, r_multiple, link_status \
     FROM trading.prediction_outcomes ORDER BY created_at DESC LIMIT 20;"
  ```
  **Expected**: every row LINKED, non-zero MAE/MFE, sensible close reasons — 20/20 = PASS.
- **You are done when…** 20 signals verified PASS and nothing needed fixing.
- **Skip impact**: the first real-money attach becomes your first test; the demo run is where surprises are cheap.

---

**Summary of what happens after you finish**: data collects automatically; a daily
one-line heartbeat is all you'll hear from engineering until Phase 1 can start.
No further action is needed from you beyond the three actions above.

| Action | Owner | Est. time | If skipped |
|---|---|---|---|
| 1. Patch + recompile | You | 20 min | MAE/MFE unusable for calibration |
| 2. Attach client EAs | You | 10 min | Phase 1 never starts (signals expire) |
| 3. Demo rehearsal | You | 1-2 h | First attach happens on live (surprises cost money) |