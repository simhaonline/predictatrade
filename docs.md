# Predict-A-Trade — Full Pipeline in Plain Words

> docs.md — written 2026-09-03. This explains, in easy language, everything
> that happens from a price tick arriving to a trade executing on a client's
> MT4/MT5 terminal — the pipeline, the maths, and the logic. Every claim
> below is backed by code in `realtime/` (the Go engine), `control/`
> (NestJS API), and `mql/` (the Windows EAs).

---

## The Big Picture (one paragraph)

Your broker terminal (the "Master EA") watches XAUUSD and sends every price
tick to our server. The server's Go engine turns ticks into candles, candles
into features (indicators), features into strategy decisions, decisions into
gated/scored signals, and delivers executable signals to client EAs, which
place the actual trades on each client's broker account. Money never moves
through our servers — clients' own terminals execute; we only decide WHAT and
deliver it safely.

```
Broker MT5 terminal (Master EA)
        |  POST /ingest/agent  (every tick, HTTPS)
        v
[Go Real-Time Engine]  ← THE CORE: everything below happens here
  1. Ingest + validate        (freshness, spread sanity, dedupe)
  2. Candle builder           (ticks → M1/M5/... candles)
  3. Feature engine           (EMA/RSI/ATR/FVG/Fib/ichimoku... ~0.3ms/bar)
  4. 7 strategies vote        (ULTRA/STANDARD scalping, swings, ATEN...)
  5. Calibration              (raw score → true probability)
  6. 22 hard gates            (risk veto checks — see list below)
  7. Capital tiers            (which account sizes may take this trade)
  8. Persist signal           (PostgreSQL, store-then-deliver)
  9. Enqueue per device       (edge_signal_queue, entitlement-filtered)
        |                          |
        v                          v
  Dashboard (WebSocket)      Client EAs poll every ~2s
                                   |  (own safety gates: drift, TTL, spread)
                                   v
                            EA places trade on client's broker account
                                   |
                                   v
                            ACK back → server marks delivered
```

---

## Stage 1 — Getting prices in (Ingest)

- The Master EA on your MT5 terminal POSTs every tick to
  `https://api.predictatrade.com/ingest/agent` with a device JWT.
- The engine validates: is the tick fresh (not from a stale clock), is the
  bid/ask sane, is it a duplicate? Bad ticks are dropped, never processed.
- Every tick is stamped `gateway_receipt_time` = **server clock (UTC)**.
  This is the platform's time truth. (The EA's own clock can drift — that
  only affects diagnostics, never decisions. All engine decisions use the
  server clock.)
- If ticks stop arriving for 3 minutes, the connectivity watchdog pages
  user + admin (ntfy topic `pat-connectivity`).

**Why**: one authoritative, clean stream of prices — everything downstream
reads from this, so nothing can "disagree".

## Stage 2 — Ticks become candles

- Ticks are grouped into candles per timeframe (M1, M5, M15, H1, D1...).
  A candle = open / high / low / close over its period, built from ticks.
- Candles are stored in PostgreSQL (TimescaleDB) — millions of bars,
  used live AND by the backtest engine (same maths, same code path).

**Easy words**: a candle is just "where price went during one minute" —
the alphabet everything else reads.

## Stage 3 — Features (the maths toolbox)

For every closed candle the feature engine computes indicators:

- **EMAs / SMAs** — smoothed averages showing trend direction
- **RSI** — how overbought/oversold we are (0–100)
- **ATR** — "average true range" = how much price typically moves; sets
  sensible stop-loss distances
- **FVG (fair value gaps)** — price holes where price often returns
- **Swing highs/lows, Fibonacci levels, Ichimoku** — structure and targets
- **Session tags** (Asia/London/NY), regime (trending vs ranging)

Performance: the whole feature engine was rewritten (2026-09-03) to plain
float64 maths — **11.5× faster (~0.3ms per bar)**; a full 2020–2026 M1
backtest (2.1M bars) runs in ~5.5 minutes.

**Easy words**: features are the "weather instruments" — they describe the
market's mood in numbers.

## Stage 4 — Seven strategies vote

Each strategy looks at the features and says BUY / SELL / WAIT / NO-TRADE,
with a raw score 0–100 and its own entry/SL/TP geometry:

| Strategy | Style |
|---|---|
| ULTRA_SCALPING | very fast M1 scalps, tight stops, micro profit-taking |
| STANDARD_SCALPING | M1/M5 scalps |
| STANDARD_SWING | slower swings, wider stops |
| TREND_SWING | rides established trends |
| MARNIE_FIB | Fibonacci-based retracement entries |
| ATEN | session/liquidity model |
| ARCANIST | Asia-range sweep → break-of-structure model |

Each strategy has its own veto reasons (e.g. ARCANIST needs Asia range,
a sweep, and a BOS before it will ever fire) — **NO-TRADE is a valid,
normal outcome.** The engine never forces trades to hit frequency targets.

**Easy words**: seven analysts each give their opinion; most of the time
most of them say "nothing here".

## Stage 5 — Calibration (score → honest probability)

A raw strategy score (0–100) is just enthusiasm. Calibration converts it to
a **calibrated probability** the trade actually wins, using a sigmoid fit:

```
calibrated_probability = sigmoid(a × (raw_score / 100) + b)
```

`a` and `b` are learned from live trade history (isotonic/sigmoid
calibration in `internal/calibration`). This is the number compared against
the trade threshold — so "70" doesn't mean 70% unless history says so.

**Easy words**: calibration is the reality-check that stops a confident
strategy from lying about its odds.

## Stage 6 — The 23 hard gates (risk veto wall)

Before ANY signal is executable it must pass every gate. One VETO = no
trade, no exceptions, fail-closed (if a value is unknown, block):

- **Data quality** — fresh ticks, sane prices
- **Session / news** — right session, no red-folder news
- **Spread / slippage / total cost** — broker costs can't eat the trade
- **Min ATR** — market moving enough to be worth trading
- **Stop-hunt filter** — don't enter into obvious stop-hunt wicks
- **Wrong-side SL / R:R net expectancy** — geometry must still profit
  AFTER broker costs (micro-TP must cover round-trip cost)
- **Margin / exposure / position caps** — never over-leveraged
- **Risk-oversize** — lot size fits the risk budget
- **Daily loss / profit target** — the capital circuit-breakers
  (halt at −MAX_DAILY_LOSS_PCT, lock at +MAX_DAILY_PROFIT_PCT; weekly
  and monthly variants too). These run on equity anchors in Valkey and
  fail CLOSED if P&L state is unknown.
- **Martingale ban** — no lot-increasing after losses
- **Edge validation** — if a strategy's last 50 live trades don't prove
  edge, its signals downgrade to advisory-only
- **Broker symbol validation** — symbol spec sanity
- **Entitlement / license / execution permission** — plan allows it,
  license active, automation permitted

**Easy words**: 23 bouncers at the door. One "no" = no trade. This is why
you see DETECTED signals on the dashboard that never execute — they were
vetoed on purpose (each carries a reason code like
`profit_target_hit:daily` or `INSUFFICIENT_SCORE`).

## Stage 7 — Capital tiers (who may receive what)

Small accounts cannot survive wide stops, so every signal is classified
per tier BEFORE delivery:

```
min-lot risk $ = SL distance in price points × $1/point (0.01 lot XAUUSD)
tier cap $   = tier reference equity × 2%
   MICRO    ref $100  → cap $2
   STANDARD ref $500  → cap $10
   PRO      ref $5000 → cap $100
```

A signal is delivered only to devices whose tier can afford its min-lot
risk. Example from 2026-09-03: a STANDARD_SWING with a 22.5-point SL
needs $22.50 min-lot risk → PRO only; MICRO/STANDARD devices correctly
get nothing. Tight-stop scalps (SL ≈ 4–8 pts → $4–8) fit STANDARD and
sometimes MICRO.

Tier caps and plan caps combine as `min(plan cap, tier cap)` — neither
can be weakened.

**Easy words**: a $150 account is never asked to risk $22 on one trade.
That's protection, not a bug.

## Stage 8 — Position sizing maths

```
risk $       = equity × MAX_RISK_PER_TRADE_PCT (default 1.5%)
risk per lot = SL distance / tick size × tick value  (XAUUSD ≈ $100/point/1.0 lot)
lot          = floor-to-step(risk $ / risk-per-lot)   [min 0.01]
margin check = required margin ≤ free margin × 30%
```

If the computed lot < broker minimum, the trade is vetoed (a $100 account
with a 30pt SL simply cannot trade it safely).

## Stage 8b — Micro profit-taking (the scalping edge)

Every signal carries a `MicroTP` — the first, smallest take-profit used
for a partial close. The ProfitabilityGate requires that this micro
target covers round-trip broker cost (spread + commission + slippage)
with margin to spare. If the broker fee eats the scalp, the signal is
vetoed (`MICRO_TP_UNPROFITABLE`).

**Easy words**: for ultra scalping, trade 1 must at minimum pay the
broker's bill; otherwise we don't take it.

## Stage 9 — Persist + deliver (store-then-deliver)

1. Signal is saved to PostgreSQL FIRST (`trading.signals`) — even if
   delivery fails, the record exists for audit/replay.
2. Only `Executable == true` signals are enqueued (fail-closed: a vetoed
   signal can never be traded, dashboard-only).
3. Enqueue writes one row per entitled device into
   `licensing.edge_signal_queue`, filtering by license plan + strategy
   whitelist + capital tier, in SQL, server-side.
4. A delivery-ledger record is written for reconciliation.

## Stage 10 — Client EAs pick up and execute (Option B)

- Each client's MT4/MT5 EA polls `POST /api/v1/devices/edge-poll` every
  ~2s (HMAC-signed device token). It gets queued signals and server
  commands (e.g. LICENSE_STATUS).
- The EA then applies ITS OWN final safety gates before touching money:
  - **entry drift budget** — if price moved too far from the signal's
    entry (ULTRA 15pts, SCALP 25, SWING 60, TREND/MARNIE/ATEN 80), the
    R:R geometry is stale → NO-TRADE
  - **signal TTL** — expired signals are ignored
  - **spread cap, risk $ cap, margin check** — re-checked locally
- If all pass → order placed on the client's own broker account with SL,
  TP1/TP2/TP3, and micro-TP partial close.
- The EA ACKs the signal (`edge-ack`); the server marks the queue row
  ACKED. Un-ACKed signals are flagged for reconciliation.

**Easy words**: the server decides, the client executes on its own
account, with its own final safety check. Client never loses control of
its money; server never touches it.

## Stage 11 — Review loop (learning)

Closed trades feed `trading.trade_results` → rolling forward-test stats
(profit factor, expectancy R per strategy) → the EdgeValidation gate
downgrades strategies that stop proving edge, and the calibrator
recalibrates. The pipeline measures itself.

---

## What runs where (planes)

| Plane | Tech | Does |
|---|---|---|
| Real-time (Go, :13081) | one binary, one clock | everything above: ticks → signals → queue |
| Control (NestJS, :13080, ×2 replicas) | NestJS | logins, licenses, devices, edge-poll API, billing, monitoring/watchdog |
| Presentation (Next.js, :13082) | dashboard | shows server truth; decides NOTHING |
| Edge (MQL EA on Windows) | MT4/MT5 | ingest (master) + execute (clients) with local safety gates |
| Research (Python) | offline | datasets, backtests, calibration research — never in the live tick path |

**Rule**: no plane borrows another's authority. Billing never touches
trading; the dashboard never decides; the EA never invents intelligence.

## Safety principles (the "why it's safe" checklist)

1. **Fail-closed everywhere** — unknown value ⇒ veto, not gamble.
2. **NO-TRADE is first-class** — most evaluations end in NO-TRADE by design.
3. **Store-then-deliver** — every signal is persisted before delivery;
   nothing is lost on a disconnect.
4. **Server filters entitlement, EA re-checks locally** — two independent
   walls against unauthorized/stale trades.
5. **Money never transits our servers** — clients' terminals execute on
   their own accounts with their own brokers.
6. **Everything observable** — every veto carries a reason code; the
   watchdog pages humans when the pipeline breaks; the admin dashboard
   shows live connectivity.

## Known behaviour you will see (so it doesn't surprise you)

- **DETECTED but not executed** = vetoed by a gate or capital tier — check
  `reason_codes` on the signal row / pipeline page.
- **A 502 once in a while on ingest** = realtime deploy window; the EA
  retries automatically. Realtime is a singleton by design (splitting it
  would corrupt state).
- **Signal timestamps ~15 min "late"** = the Master terminal's Windows
  clock is behind; server timestamps are correct. Sync that PC.
- **Fewer signals for MICRO/STANDARD** = capital-tier protection on
  wide-stop signals. Tight scalps are the volume driver for small
  accounts — exactly the ultra-scalping focus.

---

*Sources: `realtime/cmd/realtime-engine/main.go`, `realtime/internal/{signal,gates,risk,capitaltier,calibration,features,strategy}/`, `control/src/modules/edge-poll/`, `mql/mt5/PredictATrade_MT5.mq5`, `nginx/sites-available/api.predictatrade.com.conf`.*