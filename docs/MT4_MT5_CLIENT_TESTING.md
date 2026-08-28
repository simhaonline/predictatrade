# Windows MT4 / MT5 Client Testing — Procedure

How to validate the PAT realtime engine end-to-end against a real Windows MT4/MT5
terminal. Goal: confirm signal delivery, server-side SL/TP enforcement, execution
authorization and reconciliation **on a DEMO/SANDBOX account only** — never live
money during this phase.

## 1. Build the Windows Agent (Go → Windows binary)

From a Linux shell (cross-compile, no Windows needed):

```bash
./scripts/build-windows-agent.sh --bump
# produces windows-agent/bin/pat-agent-signed.exe (Client / execution adapter)
# and windows-agent/bin/pat-master-amd64.exe      (Master Node / data)
```

The binaries are served to Windows by nginx at
`https://downloads.predictatrade.com/windows-agent/` (see `windows-agent/deploy/`).
Checksums are written to the update manifest — verify them on the client before
installing.

## 2. Install the EA on the terminal

Copy the matching EA into the terminal's `MQL4/Experts` or `MQL5/Experts` folder:

- MT4 → `mql/mt4/PredictATrade_MT4.mq4`  (compiled `mql/compiled_executable/PredictATrade.ex4`)
- MT5 → `mql/mt5/PredictATrade_MT5.mq5`  (compiled `mql/compiled_executable/PredictATrade.ex5`)

Restart the terminal, attach `Predict-A-Trade` to an **XAUUSD** chart (M5), and
set inputs:
- `GatewayURL` → `wss://<your-domain>/realtime` (the Go `pat-realtime` service)
- `DeviceToken` → the token issued by control plane for this device
- `Mode` → `SANDBOX` / `PAPER` (hard requirement for this phase)
- `SymbolMap` → `XAUUSD` (verify the broker's exact instrument name, e.g. `XAUUSD.sml`)

The terminal must allow **DLL / WebSocket / file** access for the agent bridge
(Options → Expert Advisors → "Allow automated trading" + "Allow DLL imports").

## 3. Connect & register

1. Start `pat-agent-signed.exe` (or let the EA spawn the bridge).
2. Confirm the agent shows `CONNECTED` and the gateway lists the device as
   `AUTHORITATIVE` with a heartbeat in `windows-agent/AppData/Roaming/MetaQuotes/Terminal/Common/Files/PAT_heartbeat.txt`.
3. Verify the gateway streams real XAUUSD bids/asks to the agent (compare the
   agent's printed price to the terminal's Market Watch — they must match within
   the broker's spread, not a synthetic/futures-print).

## 4. Signal → execution test (server is the authority)

With the engine running the strategies you just backtested:

- A `BUY/SELL` signal arrives → EA opens the position **only if** the engine's
  `EXECUTION_ACK` matches (SL > 0 and SL == server value ±0.5 points).
- Kill-list checks to perform manually:
  - **EXECUTION_ACK**: open a position, then manually move its SL in the terminal
    → engine must send `CLOSE_POSITION` (SL missing/violated).
  - **EMERGENCY_STOP**: trigger from control plane → all PAT positions close + trading halts.
  - **KILL_SWITCH**: trigger → positions close + `ExpertRemove()` + agent disconnect.
  - **Agent suspension**: force 3 SL violations → agent disconnected, other agents unaffected.

## 5. Reconciliation & data-integrity checks

- Compare the EA's open positions/fills against the engine's position ledger.
- Confirm spread/slippage charged matches the **broker** tick (not a flat assumption).
- Watch the realtime latency metric (signal→fill). Gold's $3 spikes mean >1–2s
  latency will degrade the already-modest ~37% win rate further.
- Confirm `NO-TRADE` is emitted during widened spread (spread > 2× session median)
  and during news-block windows — these are the cost-control gates.

## 6. What "genuine" means here

- Use a **real broker DEMO** feed, not the synthetic generator (`generate_synthetic`
  in the research loader is clearly flagged and the backtest CLI now *refuses* to
  emit a report on it unless `--allow-synthetic` is passed).
- Every backtest you compare against must show `data_integrity.is_synthetic == false`
  and a real `data_start`/`data_end` in `summary.json` (added in this change).
- Demo results are still **labeled DEMO** in the UI/agent; they must never mutate
  live trading or real finance.

## 7. Exit criteria for promotion to paper/live

- EXECUTION_ACK, EMERGENCY_STOP, KILL_SWITCH, agent-suspension all verified.
- Position reconciliation mismatch == 0 over a full session.
- Signal→fill latency p95 < 2s on the target broker.
- At least one full session of SANDBOX trading with no stale/fake ticks.
