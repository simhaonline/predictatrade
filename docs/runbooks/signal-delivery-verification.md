# Signal Delivery Verification — dashboard vs MT4/MT5 EA

**Symptom:** "I see signals on https://live.predictatrade.com but my MT4/MT5 EA
receives nothing."

**This is usually EXPECTED, not a broken pipeline.** The two surfaces read
different data:

| Surface | Endpoint | Data shown |
|---|---|---|
| Dashboard (`live.predictatrade.com`) | `GET /api/v1/signals` | Recent signals for your plan's **allowed_strategies** — **including non-executable candidates** (the endpoint does NOT filter `Executable`). |
| MT4/MT5 EA | `POST /api/v1/devices/edge-poll` (HMAC) | **Only `Executable==true`** signals that pass plan/strategy entitlement. |

So the dashboard can show dozens of signals while the EA legitimately receives
zero — because the shown signals are `Executable=false` candidates, or belong to
strategies your plan doesn't allow.

## Verify with the tool

Read-only, no data mutated, no secrets printed:

```bash
python3 scripts/verify_signal_delivery.py                 # full fleet report
python3 scripts/verify_signal_delivery.py --device <uuid> # single device deep-dive
python3 scripts/verify_signal_delivery.py --days 30
```

The report shows, per device: license status, plan, allowed_strategies, last_seen,
and PENDING/ACKED/EXPIRED counts from `licensing.edge_signal_queue`, plus the
exact reason a device receives nothing.

## Most common root causes (from the report)

1. **Plan does not allow the currently-executable strategies.** E.g. Free plan
   allows only `STANDARD_SCALPING`, but in the measured window the engine produced
   **0 executable STANDARD_SCALPING** signals (all 205 executable were
   TREND_SWING / STANDARD_SWING / MARNIE_FIB). Fix: upgrade plan, or accept that
   no executable signal exists for your allowed strategy right now.
2. **License not ACTIVE** (REVOKED / expired). Fix: renew/activate license.
3. **EA not polling** — device `last_seen` stale or empty. Fix on the terminal:
   - Tools → Options → Expert Advisors → allow WebRequest for `https://api.predictatrade.com`
   - AlgoTrading enabled on the chart
   - Device activated (LicenseKey) — terminal log should show
     `Cloud device ready — edge-poll mode`
   - Terminal actually running (not closed)
4. **Profitability soft-gate was over-vetoing (FIXED 2026-09-11).** Historically
   the system marked ~99.5% of candidates as loss-candidates via an uncalibrated
   win-rate model + a broken micro-TP coverage test, and the delivery gate
   hard-vetoed on that flag — so even entitled, polling EAs got almost nothing.
   The gate model was rebuilt (`docs/trading/signal-gating.md`): HARD gates are
   unchanged, but the profitability SOFT-ALPHA gate now uses a versioned,
   strategy-specific, sample-confident expectancy and emits `ShadowExecutable`
   signals instead of starving the strategy. Rollback via `GATE_PROFILE_VERSION`.
   After this change, re-run the tool: a previously-silent entitled device should
   now show `ACKED` > 0 (or `ShadowExecutable` candidates being tracked).
5. **Profitability veto still appropriate but now calibrated** — system-wide the
   executable pool should be materially larger. If your allowed strategy still
   produces 0 executable signals, that's a genuine strategy-performance gap (see
   the gating doc §4), not a delivery bug. Adjust refinement/executability only
   with explicit operator sign-off.

## Is the pipeline itself broken?

Check the GLOBAL section of the report: if `edge_signal_queue` shows thousands of
`ACKED` rows, the engine→EA path works and the issue is entitlement/polling
(above). If `ACKED` is ~0 fleet-wide, that's a real engine-enqueue incident —
escalate to the realtime engine team.
