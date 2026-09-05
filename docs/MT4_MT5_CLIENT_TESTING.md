# MT4 / MT5 Client Testing — Procedure (EA-direct era)

How to validate the PAT realtime engine end-to-end against a real Windows MT4/MT5
terminal. Goal: confirm signal delivery, server-side SL/TP enforcement, execution
authorization and reconciliation **on a DEMO/SANDBOX account only** — never live
money during this phase. Since v1.19.0 (Option B) there is **no Windows Agent**:
the EAs talk to the cloud directly over HTTPS.

## 1. Provision the EA sources

From a Linux shell (no Windows needed):

```bash
# The fleet is 4 single-file EAs (Client + Master Node × MT4 + MT5), pure MQL5
ls mql/mt4/*.mq4 mql/mt5/*.mq5
# They are served to terminals by nginx from the repo's mql/ mount:
docker compose --env-file infra/env/.env up -d nginx
# https://downloads.predictatrade.com/mql/PredictATrade_MT5.mq5 (+ .mq4, MasterNode variants)
```

## 2. Install the EA on the terminal

1. Download the matching `.mq5`/`.mq4` from `https://downloads.predictatrade.com/mql/`
   and copy it into the terminal's `MQL5/Experts` or `MQL4/Experts` folder.
2. Compile in MetaEditor (F7) — must show **0 errors, 0 undeclared identifiers**.
3. Allowlist the cloud (one time): Tools → Options → Expert Advisors →
   *"Allow WebRequest for listed URL"* → add `https://api.predictatrade.com`.
4. Attach the **Client EA** (`PredictATrade_MT5.mq5` / `.mq4`) to an **XAUUSD**
   chart and set:
   - `LicenseKey` → your license key (Dashboard → MetaTrader Client page)
   - `AutoExecute` → `false` for the first pass (signal-only), then `true` for the
     execution test
   - `Mode` → `SANDBOX` / `PAPER` (hard requirement for this phase)
   - Verify the broker's exact instrument name (e.g. `XAUUSD.e`) matches the
     symbol mapping
5. On the data terminal, attach the **Master Node EA**
   (`PredictATrade_MasterNode_MT5.mq5` / `.mq4`) with `MasterLicenseKey` set.
6. Enable "Allow Automated Trading".

## 3. Connect & register

1. The EA activates its cloud device automatically on start (state file per
   platform in `Common\Files`: `PAT_device.txt` / `PAT_device_mt4.txt` and the
   Master variants). Never share state files between terminals.
2. Confirm Admin → Devices shows the device **ONLINE** with a fresh edge-poll
   heartbeat and (after first ACCOUNT_INFO) the broker account type.
3. Verify real XAUUSD ticks flow: `GET /api/v1/market/snapshot` must show the
   broker's bid/ask (compare to the terminal's Market Watch — within the broker's
   spread, not a synthetic/futures print). `data_health` in
   `GET /api/v1/agents/status` should be `HEALTHY`.

## 4. Signal → execution test (server is the authority)

With the engine running the strategies you just backtested:

- A queued `SIGNAL` arrives on the next edge-poll (≤ `PATPollMs`, default 3s) →
  the EA opens the position **only if** `AutoExecute=true` and the signal is
  executable; the EA then ACKs via `POST /ingest/agent`.
- Server-side SL check: open a position, then manually move its SL in the
  terminal → engine must queue `CLOSE_POSITION` (SL missing/violated, ±2 tick
  tolerance).
- **EMERGENCY_STOP**: trigger from the control plane → signal generation and
  delivery halt.
- **KILL_SWITCH**: trigger → positions close + `ExpertRemove()` on the EA.
- **Device suspension**: force 3 SL violations → device disconnected, other
  devices unaffected.

## 5. Reconciliation & data-integrity checks

- Compare the EA's open positions/fills against the engine's position ledger and
  the delivery ledger (`signal_deliveries` + `edge_signal_queue` ACKs).
- Confirm spread/slippage charged matches the **broker** tick (not a flat assumption).
- Watch the realtime latency metric (signal→fill). Gold's $3 spikes mean >1–2s
  latency will degrade the already-modest ~37% win rate further.
- Confirm `NO-TRADE` is emitted during widened spread (spread > 2× session median)
  and during news-block windows — these are the cost-control gates.
- Verify poll-time entitlement re-check: revoke the license mid-session → the
  next poll expires queued signals, none delivered.

## 6. What "genuine" means here

- Use a **real broker DEMO** feed, not the synthetic generator (`generate_synthetic`
  in the research loader is clearly flagged and the backtest CLI now *refuses* to
  emit a report on it unless `--allow-synthetic` is passed).
- Every backtest you compare against must show `data_integrity.is_synthetic == false`
  and a real `data_start`/`data_end` in `summary.json` (added in this change).
- Demo results are still **labeled DEMO** in the UI; they must never mutate
  live trading or real finance.

## 7. Exit criteria for promotion to paper/live

- EXECUTION_ACK, EMERGENCY_STOP, KILL_SWITCH, device-suspension all verified.
- Position reconciliation mismatch == 0 over a full session.
- Signal→fill latency p95 < 2s on the target broker.
- At least one full session of SANDBOX trading with no stale/fake ticks.