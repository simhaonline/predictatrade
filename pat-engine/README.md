# pat-engine

A clean, standalone signal engine for Predict-A-Trade XAUUSD strategies. Built from
scratch (Go, **zero external dependencies**) as the "core reset" of the signal brain:
single-source strategy config, broker-aware gating, a research-grade backtest that runs
the **exact same** strategy code as live, and a live gateway that hands signals to the
MQL EA via `PAT_signals.txt`.

It reuses **no** existing service or database — it is reference-only on the old repo.
The current production project is left untouched.

## Layout

```
pat-engine/
  cmd/
    engine/    demo: runs all 4 strategies over a CSV snapshot, prints decisions
    backtest/  research: PF / win-rate per strategy on synthetic OR real bars
    gateway/   LIVE: HTTP ingest of bars -> strategy pipeline -> PAT_signals.txt
    agent/     reference data feeder (streams bars to the gateway)
    sim/       EA-file simulator (proves the file handoff end-to-end)
  internal/
    types/       shared contracts (MarketState, Signal, Indicators, StrategyID)
    config/      single-source SL/TP/RR per strategy + AllDefaults()
    broker/      BrokerPolicy: scalping toggle, min-hold, stop/freeze vetoes
    strategy/    the 4 products: ULTRA_SCALPING, STANDARD_SCALPING,
                 STANDARD_SWING, TREND_SWING (+ shared scoring/geometry)
    gates/       hard risk gates (R:R floor, broker stop/freeze)
    signal/      Decide(): strategy + broker policy + gates -> Decision
    indicators/  pure TA math (EMA, SMA, ATR, RSI, MACD, ADX, Stoch, Boll, VWAP)
    marketdata/  CSV snapshot replay provider (reference)
    backtest/    builds snapshots from bars + exit simulator + real-data CSV loader
    provider/    live gateway that writes the EA signal file
  data/sample.csv   demo snapshot feed
  mql/              reference MQL4/MQL5 EAs (read PAT_signals.txt)
```

## Strategies (distinct, versioned, config-backed)

| ID                | Class  | SL×ATR | TP1×ATR | Min R:R | Min Confluence | Min ADX |
|-------------------|--------|--------|---------|---------|----------------|---------|
| ULTRA_SCALPING    | SCALP  | 1.0    | 2.0     | 2.0     | 65             | 25      |
| STANDARD_SCALPING | SCALP  | 1.0    | 1.5     | 1.5     | 65             | 20      |
| STANDARD_SWING    | SWING  | 1.5    | 2.5     | 1.5     | 55             | 20      |
| TREND_SWING       | TREND  | 2.0    | 3.0     | 1.5     | 50             | 20      |

All thresholds live in `internal/config` — one place, historically reproducible.

## Broker scalping policy (first-class)

Many brokers forbid scalping. `broker.BrokerPolicy.AllowsScalping=false` **excludes**
both scalping strategies entirely (`BROKER_SCALPING_NOT_ALLOWED`). On a no-scalping
broker only `STANDARD_SWING` and `TREND_SWING` remain eligible — verified by the
backtest harness and the live decision tests.

> Strategy enable/disable is a **server/control-plane (license)** concern
> (`allowed_strategies`), never an EA input. Do not add strategy toggles to MQL.

## Build & run

```bash
cd pat-engine
go build ./...

# Demo: evaluate all 4 strategies over the sample snapshot feed
go run ./cmd/engine

# Research: PF / win-rate per strategy (synthetic data; swap in real bars via CSV)
go run ./cmd/backtest

# LIVE loop (terminal 1: gateway, terminal 2: EA simulator, terminal 3: agent)
go run ./cmd/gateway &          # writes signals/PAT_signals.txt
go run ./cmd/sim &              # consumes it like the EA would
go run ./cmd/agent              # streams 3000 synthetic bars -> gateway
# real data instead:  BARS_CSV=my_xauusd.csv go run ./cmd/agent
```

### Real backtest data

```bash
# bars CSV: header time,open,high,low,close,spread (spread optional)
BARS_CSV=real_2025.csv go run ./cmd/backtest
```

### Real bars into the live gateway

```bash
# the Windows agent (rebuilt to stream ticks) POSTs to /bar; the gateway
# maintains the rolling window, computes indicators, and writes the signal file
# the MQL EA already reads from FILE_COMMON.
```

## Testing

```bash
go test ./...
```

Covers: strategy executability + R:R floor, broker scalping exclusion, high-spread
block, and the broker-policy allow/deny matrix.

## Signal file format (consumed by the EA)

`SIGNAL|<json>` where json carries exactly the keys the MQL `ExtractJSONString`
parser expects: `ID, Direction, Grade, StrategyID, SignalClass, EntryPrice,
StopLoss, TP1, TP2, TP3, SuggestedLot, RawScore, CalibratedProbability`.

## What is intentionally NOT here yet

- Frontend / Command Center (deferred per plan; backend + MQL first).
- Control-plane licensing WS (the EA's license check stays on the NestJS backend).
- A rebuilt Windows Agent binary (reuse `windows-agent` and point it at `/bar`).
