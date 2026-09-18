# OPERATOR WINDOWS RUNBOOK — from "EA offline" to "pipeline flowing"
One page, follow top to bottom. Do all steps on the **Windows machine with MT5**.
SQL steps run on the origin server (or any machine that can reach it — see §5 note).
Time total: ~30 min (+ market-wait time for trades to close).

## 1. Prerequisites

- [ ] MT5 terminal installed and logged in (demo account first).
- [ ] MetaEditor: MT5's built-in editor — open it with **F4** from the terminal (or
      `C:\Program Files\MetaTrader 5\metaeditor64.exe`).
- [ ] Get the repo files onto the Windows box (any ONE):
  - `git clone https://github.com/simhaonline/predictatrade.git` (easiest), or
  - download the ZIP from the repo web page and extract; you need exactly two
    things from it: `mql/mt5/PredictATrade_MT5.mq5` and
    `tools/ea/0001-fix-mae-mfe-tracking.patch`.
- [ ] Put a copy of the terminal's `PredictATrade_MT5.mq5` (the one currently in
      `MQL5\Experts\`) next to the repo copy — you will patch YOUR copy. Keep a
      `.bak` of the unpatched file first.

## 2. Apply the patch

From the folder containing both the patch file and your `.mq5` (Command Prompt):

```
cd C:\path\to\repo\mql\mt5
copy PredictATrade_MT5.mq5 PredictATrade_MT5.mq5.bak
git apply --check ..\..\tools\ea\0001-fix-mae-mfe-tracking.patch   (dry run)
git apply ..\..\tools\ea\0001-fix-mae-mfe-tracking.patch
```

**Confirm the patch applied** — open the patched file in Notepad, go to the
`PAT_ReportResult` JSON block (search for `time_in_trade_seconds`, ~line 1336):

BEFORE (unpatched, lines ~1337-1338):
```
    msg += ",\"mae\":0.0";
    msg += ",\"mfe\":0.0";
```
AFTER (patched):
```
    msg += ",\"mae\":" + DoubleToString(p_mae, _Digits);
    msg += ",\"mfe\":" + DoubleToString(p_mfe, _Digits);
    msg += ",\"mae_r\":" + DoubleToString(p_maeR, 3);
    msg += ",\"mfe_r\":" + DoubleToString(p_mfeR, 3);
```

(If `git` is not installed: open the patch file in Notepad and make the same
three edits manually — every changed block is marked with a `v1.30` comment.)

## 3. Recompile

1. Open MetaEditor (F4 in the terminal), open `MQL5\Experts\PredictATrade_MT5.mq5`
   — paste/overwrite with the patched content if you edited remotely.
2. Press **F7** (Compile).
3. **Success looks like**: `0 errors` in the ToolBox output pane (warnings that
   existed before are fine), and a fresh `PredictATrade_MT5.ex5` timestamp in
   `MQL5\Experts\`.
4. If you get errors → stop, §7 row 2.

## 4. Attach to a demo chart

1. Drag `PredictATrade_MT5` onto an **XAUUSD demo** chart (M1 or M5).
2. In the Inputs dialog set:
   | Input | Value | Why |
   |---|---|---|
   | `LicenseKey` | your license key | device activation |
   | `PATCloudURL` | `https://api.predictatrade.com` | server endpoint |
   | `ChartTimeframe` | `M1` (Ultra) or `M5`/`H1` (other strategies) | match your terminal's role |
   | `AutoExecute` | `true` (default since v1.33) | trade signals automatically |
   | `PATPollMs` | 3000 | default |
   Leave everything else default.
3. **AutoTrading button ON**: MT5 toolbar 'Algo Trading' / MT4 'AutoTrading' —
   the EA prints `*** AUTO-TRADING IS OFF ***` at attach if it's disabled.
4. **Allow WebRequest**: Tools → Options → Expert Advisors → check "Allow
   WebRequest…" and add `https://api.predictatrade.com` to the list (the EA
   prints this reminder itself if missed).
5. **Running looks like**: EA name + chart shows a normal (no-alert) state;
   Experts tab prints:
   ```
   Predict-A-Trade MT5 EA v1.28 initializing...
   [Predict-A-Trade] account_type=Demo login=... confirmed
   ```
   Any line mentioning the WebRequest allowlist → do step 5 of this section again.

## 5. First-signal verification (SQL, in order — run each on the origin server)

DB access: `docker exec -i pat-postgres psql -U pat_admin -d predictatrade` on the
origin (2.29.23.42), or expose port 5432 (already bound to 127.0.0.1 on the
origin; use the server itself via SSH).

| # | What | Query | Expected |
|---|---|---|---|
| 1 | Device ONLINE | `SELECT connection_status, role, last_seen_at FROM licensing.devices WHERE role='exec' ORDER BY last_seen_at DESC LIMIT 3;` | your device `ONLINE` + fresh timestamp |
| 2 | Signal enqueued | `SELECT status, count(*) FROM licensing.edge_signal_queue WHERE created_at > now() - interval '1 hour' GROUP BY 1;` | `PENDING`/`IN_FLIGHT` counts growing |
| 3 | ACK received | same query | `ACK`-family counts appear; device leaves the stale list in `operator_status.sh` |
| 4 | Prediction row written | `SELECT signal_id, strategy_id, direction, created_at FROM trading.predictions ORDER BY created_at DESC LIMIT 3;` | rows timestamped at signal-emit time |
| 5 | Order opened | terminal: Trade tab shows the position; Experts log: `TRADE_RESULT reported:` only after close | position visible |
| 6 | First TRADE_RESULT | `SELECT signal_id, close_reason, mae, mfe, created_at FROM trading.trade_results ORDER BY created_at DESC LIMIT 3;` | newest row = your demo trade |
| 7 | Outcome row with non-zero MAE/MFE | `SELECT signal_id, close_reason, mae_points, mfe_points, r_multiple, link_status FROM trading.prediction_outcomes ORDER BY created_at DESC LIMIT 3;` | `mae_points`/`mfe_points` NON-ZERO, `link_status=LINKED` |

One command prints the whole live picture at once: `bash scripts/operator_status.sh` (on the origin).

## 6. 24h watch

| When | Check | Good | Investigate |
|---|---|---|---|
| T+1h | `bash scripts/operator_status.sh` | device ONLINE; acked growing; first TRADE_RESULT | device OFFLINE >10 min after attach (§7 row 3); acked=0 while enqueued grows (row 5) |
| T+6h | outcome query above | every closed trade has an outcome row, MAE/MFE non-zero ≥80% | MAE/MFE all zero → patch not applied on THIS terminal (redo §2-3); outcomes missing → row 8 |
| T+24h | gate query below | N climbing, UNLINKED ≈ 0 on new rows | UNLINKED >20% on NEW rows (row 8) |

Gate progress (also the §8 target):
```
docker exec -i pat-postgres psql -U pat_admin -d predictatrade < scripts/sample_sufficiency_gate.sql
```

## 7. Troubleshooting (the 8 most likely)

| # | Symptom | Diagnostic | Fix |
|---|---|---|---|
| 1 | `git apply` rejects | patch file path/line context | use manual edit: all changes marked `v1.30` in README `tools/ea/README.md` |
| 2 | Compile errors after patch | ToolBox output names line | you patched a drifted file — start from a fresh repo copy of the .mq5 |
| 3 | EA attaches but device stays OFFLINE | Experts log for license/activation errors | check LicenseKey; confirm device not REVOKED: `SELECT connection_status, revocation_reason FROM licensing.devices ORDER BY last_seen_at DESC;` |
| 4 | No signals arriving | `SELECT count(*) FROM licensing.edge_signal_queue WHERE created_at > now() - interval '1 hour';` | market closed or all-veto window (triple-swap day, session filter) — normal outside London/NY |
| 5 | Signals enqueued, no ACK | EA Experts log for poll errors; device ONLINE? | WebRequest allowlist missing (§4 step 5); `PATCloudURL` typo |
| 6 | ACK but no order | AutoTrading OFF (terminal toolbar)? Daily-loss block? Algo button OFF → Experts log prints `AUTO-TRADING BLOCKED` once/min since v1.33 | click 'Algo Trading' ON; `AutoExecute` now defaults true (v1.33); check Experts log `CAPITAL PROTECTION` lines |
| 7 | Order but no TRADE_RESULT | position still open (wait) or HistoryPoll failing | EA polls history every ~30s; if closed position but no `TRADE_RESULT reported:` line in 5 min → Experts log excerpt to engineering |
| 8 | TRADE_RESULT but no outcome row | `SELECT count(*) FROM trading.prediction_outcomes WHERE created_at > now() - interval '1 hour';` | outcome writer silent → `curl http://localhost:13081/health` on origin; check `outcome_pipeline` block; if schema_guard≠verified, an unapplied migration — do not self-fix, call engineering |
| 9 | EA never asks for LicenseKey / "No device credentials and no LicenseKey" never appears | EA loaded a stale compiled `.ex5` (pre-license build) or a saved `.set` profile overrides inputs | delete the old `.ex5`, recompile the patched `.mq5` (F7), re-attach with a FRESH inputs dialog — do not load an old `.set` |
| 10 | Device limit exceeded (`DEVICE_LIMIT_EXCEEDED`, HTTP 409) | old/test devices still occupy the license slots | Admin → Licenses → revoke stale devices (or raise max_devices); activation retries on the next EA cycle automatically |
| 11 | Every ingest rejected (`[INGEST-AUTH] signature mismatch` storm in engine log) | engine JWT_SECRET ≠ control-plane JWT_SECRET (device tokens signed by control) | both must come from the same value (root `.env` → `${JWT_SECRET}`); after fixing, recreate `pat-realtime` AND `pat-live-terminal`; EA recovers by re-activating automatically |

## 8. Done-when (resume trigger — all four)

| # | Condition | Verify |
|---|---|---|
| a | Patch applied: non-zero MAE/MFE in trade_results | `SELECT count(*) FROM trading.trade_results WHERE COALESCE(mae,0)<>0 AND COALESCE(mfe,0)<>0 AND created_at > now() - interval '48 hours';` → count > 0 |
| b | EAs attached, TRADE_RESULT flow last 24h | `SELECT count(*) FROM trading.trade_results WHERE created_at > now() - interval '24 hours';` → count > 0 (and device ONLINE in §5-1) |
| c | Gate PASS for a strategy | `docker exec -i pat-postgres psql -U pat_admin -d predictatrade < scripts/sample_sufficiency_gate.sql` → at least one `PASS` row (n ≥ 300) — est. ~2 weeks of flowing data for STANDARD_SCALPING |
| d | Your written approval for Phase 1 start | your message to engineering saying "Phase 1 approved" |

All four true → message engineering: **"Phase 1 approved"** (plus any gate output).