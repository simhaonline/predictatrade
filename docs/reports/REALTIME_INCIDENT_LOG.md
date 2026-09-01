# Realtime Engine — Incident & Fix Log (Master Feed / Signals / NO_TRADE)

> Purpose: durable record of root causes and fixes so logic is never lost across
> sessions. Append new entries at the top. Keep it honest: no fabricated fixes.

## 2026-09-01 — "All signals NO_TRADE" after price-symbol fix

**Symptom:** After commit `0aac63c` (normalize master snapshot symbol to `XAUUSD`),
user reported all signals are `NO-TRADE`. User suspected over-gating.

**Diagnosis (ground truth from running engine):**
- Master (`-data`) and client (`-exec`) agents connected & heartbeating. Relay WS
  `/ws/v1/relay` reports `status:"LIVE"`.
- `GET /api/v1/market/state` → symbol `XAUUSD`, `LastTick.Source = MT5_MASTER`,
  `Quality = AUTHORITATIVE`, Bid/Ask fresh. **9 timeframes of candles present.**
- Indicators fully populated (RSI 31.08, ADX 36.95, MACD, ATR 43.1),
  Regime `TRENDING_BULLISH` (conf 0.8), Session `TOKYO`, NewsRisk `NONE`.
- A recent signal: `RawScore=8.71`, `LongScore=11.71`, `Direction=NO-TRADE`,
  `RiskDecision="NO-TRADE — gates not evaluated"`, nested per-strategy directions
  show mixed `BUY/SELL/WAIT` → **multi-strategy consensus disagreement → NO_TRADE**.
- `CandidateThreshold`/`TradeThreshold` are 0 ONLY because `createNoTradeSignal`
  (main.go:5548) is the generic NO-TRADE builder and never populates them.
  Gates are NOT failing closed; the strategy/consensus layer returned NO_TRADE.

**Root cause (regression vector, NOT a gate I added):**
- Before `0aac63c`, master sent symbol `XAUUSD.sd`. It was stored under that raw
  symbol, so the dashboard showed the wrong price AND the **strategies fell back to
  AGGREGATOR candles** (keyed `XAUUSD`) → they agreed enough to emit trades (but on
  aggregator data = wrong price).
- After `0aac63c`, master data is now correctly keyed `XAUUSD` and the strategies
  consume the **authoritative MT5 master candles** instead of aggregator. The shift
  in inputs changed strategy scores so the 4 strategies now disagree → consensus
  legitimately outputs NO_TRADE.
- This is **correct, fail-closed behavior given corrected data**, not over-gating.
  The candidate exists (RawScore 8.71); the aggregate refuses to trade on conflict.

**Conclusion:** NO_TRADE is NOT caused by a gate I introduced (I only changed the
symbol at intake). It is the consensus layer reacting to the now-correct master data.
The durable fix is strategy/calibration validation **on master data** (research side),
not removing safety gates.

## 2026-09-01 — Relay feed "STALE" during container restart

**Symptom:** On Admin Dashboard the market feed showed `STALE` after a realtime
container restart.

**Diagnosis:** Restarting the container drops the Master WS connection. While the
master is down the engine has no live price, falls back to stale aggregator, and the
relay marks `status:STALE`. On a host where the master auto-reconnected, feed returns
to `LIVE` (verified) with `XAUUSD` authoritative price. On a host where the master
agent did not reconnect, the feed stays STALE — same root cause as the original
"master node agent stopped" report.

**Root cause:** Master (Windows) agent is not resilient to server-side restarts;
it must be manually restarted (NSSM service). No server-side lever to restart agents.

## 2026-08 (prior) — Calibration / level fixes (baseline f73c750)
- Repaired signal level corruption, SELL TP ordering, calibration query, delivery ledger.
- These are the working baseline the engine was restored to after over-engineering
  was reverted (`ffd3904`).

## Recurring root cause (the thing to fix PERMANENTLY)
The Windows **Master agent disconnecting** (server restart, network blip, or agent
crash) cascades into: stale/aggregator price → wrong price → NO_TRADE → stale feed.
Single durable fix = **make the Master/Client agent auto-reconnect robustly** so a
server restart never strands it. That one change stops the entire cascade.

## Hard constraints (do not violate)
- Do NOT over-engineer (no watchdog/monitor container, no role-forcing, no /health
  bloat, no delivery/ACK changes) — reverted at `ffd3904` and must stay reverted.
- Do NOT break the Windows agent installer. Past break: `init()` `rand.Seed` +
  `os.Exit` in `halt()` caused "windows depender" install failure. Keep a clean
  unsigned `CGO_ENABLED=0` build; CI does NOT bump agent version.
- NO-TRADE is a first-class valid result; never force trades to hit a frequency.
- Never fabricate ticks/fills/PnL/confidence.

## Open decisions (ask operator before acting)
1. **Master agent resilience** — implement minimal reconnect-with-backoff in
   `windows-agent/internal/agent.go` WITHOUT the `init()`/`os.Exit` changes that
   broke install. High value, medium risk (must verify installer still works).
2. **Strategy/calibration on master data** — research-side: re-validate the 4
   strategies' consensus thresholds against the now-correct MT5 master feed so valid
   setups produce trades. Not a gate removal; a calibration exercise.
