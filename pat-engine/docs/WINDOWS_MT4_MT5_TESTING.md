# Windows MT4 / MT5 Client Testing Runbook (v1.0.0)

Scope: validate the PAT Windows Agent + MQL4/MQL5 EA against the Go engine **in
paper / demo only**. No live automated trading, no live broker orders, no real
money. This is the client-side (edge) counterpart to the Go real-time trading
plane. The EA is a lightweight execution adapter/guard — it never computes
signals, probability, risk eligibility or strategy. It only receives server-
authoritative signals and enforces server-sent SL/TP and emergency commands.

---

## 0. Safety boundary (read first)

- `AutoExecute = false` and `ExecuteCandidates = false` are the **default** in
  `mql/PredictATrade_MT4.mq4` / `PredictATrade_MT5.mq5` (SIGNAL-ONLY mode).
  You must explicitly opt in, and ONLY on a **demo** account.
- Never enable auto-execute on a live account without explicit operator
  authorization outside this test work.
- The server (Go engine) is the **enforcement authority** for SL/TP and the
  `EMERGENCY_STOP` / `KILL_SWITCH` commands (SOW v1.15.0). The EA obeys them.

---

## 1. Build the Windows binaries

From the `pat-engine` repo root (Linux/macOS host is fine — cross-compile):

```bash
./scripts/build-windows-agent.sh          # -> dist/pat-windows-agent.exe, dist/pat-gateway.exe
# or with a patch-version bump:
./scripts/build-windows-agent.sh --bump   # (edit internal/version/version.go first)
```

Copy `dist/pat-windows-agent.exe` and `dist/pat-gateway.exe` to the Windows
test machine (or just run the gateway on the Linux engine host and the EA on
Windows — they talk via the signal file / HTTP).

---

## 2. Start the backend signal pipeline (Linux engine host)

The gateway ingests bars, runs the strategy pipeline, and writes the signal
file the EA consumes. Use **real or replay data** — never synthetic when you
want a genuine test.

```bash
# Replay a real XAUUSD CSV through the reference agent -> gateway (SIGNAL_FILE
# points at the MT4/MT5 COMMON files dir the EA reads):
SIGNAL_FILE="C:/Users/<you>/AppData/Roaming/MetaQuotes/Terminal/<id>/MQL4/Files/PAT_signals.txt" \
PAT_LICENSE="<dev-or-test-license>" \
BARS_CSV=data/xauusd_2024q4.csv \
go run ./cmd/agent &         # feeds bars to the gateway (or your real feed)
go run ./cmd/gateway &       # writes PAT_signals.txt
```

Note: the EA reads `PAT_signals.txt` via `FILE_COMMON`, i.e. the terminal's
`MQL4/Files` (MT4) or `MQL5/Files` (MT5) common folder. Point `SIGNAL_FILE`
there, or symlink the gateway's `signals/` dir to that folder.

---

## 3. Install the EA in MT4 / MT5

1. Copy `mql/PredictATrade_MT4.mq4` -> `MQL4/Experts/` (MT4) or
   `mql/PredictATrade_MT5.mq5` -> `MQL5/Experts/` (MT5).
2. In the Navigator, attach **PredictATrade** to an **XAUUSD** chart
   (M1 for scalping strategies; M15/H1 for swing).
3. Set inputs:
   - `LicenseKey` = your test license.
   - `BrokerSymbol` = `XAUUSD` (or leave blank to use the chart symbol).
   - `ChartTimeframe` = `M1`.
   - `AutoExecute` = **false** (signal-only — default).
   - `ExecuteCandidates` = **false** (default).

---

## 4. Paper / shadow validation (no trades)

Confirm the wiring before any execution:

- The EA `Print`s `SIGNAL RECEIVED` lines as the gateway writes signals.
- With `AutoExecute=false`, the EA must **display** signals but open **zero**
  positions. This is the shadow / paper check.
- Reconciliation: every server signal (BUY/SELL/CLOSE) should map 1:1 to an EA
  log line. Mismatches = wiring bug, fix before proceeding.

---

## 5. Enable DEMO execution (only after step 4 passes)

On a **demo** account only:

- Set `AutoExecute = true` and `ExecuteCandidates = true`.
- Verify the EA places a position with the **server-sent SL** on each signal.
- **EXECUTION_ACK check**: the server requires `SL > 0` and `|client SL -
  server SL| <= 0.5 points`. If the EA cannot set the SL, the trade is
  rejected server-side (no more than 3 violations before agent suspension).

---

## 6. Server-side SL enforcement test

- Open a PAT position, then **manually delete its SL in the terminal**.
- The server scans the broker snapshot for PAT positions with missing SL and
  sends `CLOSE_POSITION`. Confirm the EA closes that ticket.
- Repeat 3 times on the same agent with bad/missing SL -> the agent is
  **disconnected** and receives no further signals (other agents unaffected).

---

## 7. Emergency stop & kill switch tests

- `EMERGENCY_STOP` (server -> agent -> EA): all PAT positions close AND trading
  halts. Verify zero new entries afterward.
- `KILL_SWITCH`: all PAT positions close, `ExpertRemove()` runs, agent
  disconnects. Verify the EA unloads.

---

## 8. Acceptance / GO-NO-GO

- PASS only if: shadow mode shows 0 unintended trades; demo execution matches
  server signals 1:1; SL enforcement + CLOSE_POSITION works; EMERGENCY_STOP and
  KILL_SWITCH work; 3 SL violations suspend the agent.
- Do **not** promote to live automated trading without explicit operator
  sign-off. Demo test data must remain clearly labeled as non-live.

---

## 9. Manual one-liner smoke test (no Windows needed)

The Go reference agent + gateway can be exercised headless to prove the signal
path end-to-end before involving MT4/MT5:

```bash
BARS_CSV=data/xauusd_2024q4.csv go run ./cmd/agent   # posts bars to /candles
# (gateway prints signal file; inspect signals/PAT_signals.txt)
```
