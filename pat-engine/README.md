# pat-engine

A clean-room, standalone **signal engine** for Predict-A-Trade XAUUSD strategies.
Built in Go (zero app-level dependencies beyond `pgx` + `go-redis`), it owns the
full signal brain: strategy evaluation, broker-policy gating, license/entitlement
gating, hard risk gates, a research-grade backtest that runs the **exact same**
strategy code as live, and persistence of every bar + signal to **TimescaleDB** with
live distribution through **Valkey**.

It is reference-only on the existing project and reuses **no** existing service or
database — its own datastore is the `pat_engine` database inside a dedicated
TimescaleDB service. The current production project is left untouched.

---

## 1. Architecture

```
                         ┌──────────────────────────────────────────┐
   market data bars      │                pat-engine                 │
   (Windows Agent)       │                                            │
        │  POST /bar     │  cmd/gateway                               │
        └───────────────▶│   1. provider.Gateway.IngestBar            │
                         │   2. backtest.StateFromBars (indicators)   │
                         │   3. signal.Decide:                        │
                         │        strategy  ─┐                        │
                         │        broker policy (scalping) ─┤ gates  │
                         │        license (allowed_strategies) ─┘     │
                         │   4. emit SIGNAL|<json> ─▶ PAT_signals.txt│
                         │   5. store: TimescaleDB (bars, signals)    │
                         │            Valkey   (cache + pub/sub)      │
                         └──────────────────────────────────────────┘
                                    │ reads file
                                    ▼
                              MQL EA (MT4/MT5)  ──▶ trade
```

All decisions are **server-authoritative**: the EA never computes strategy, risk,
entitlement, or probability — it only consumes the signal file.

---

## 2. Components

| Path | Role |
|---|---|
| `internal/types` | Shared contracts (`MarketState`, `Signal`, `Indicators`, `StrategyID`) |
| `internal/config` | **Single source of truth** for SL/TP/RR/score thresholds per strategy |
| `internal/broker` | `BrokerPolicy`: scalping toggle, stop/freeze vetoes |
| `internal/license` | Signed license token (entitlements, expiry, device) |
| `internal/strategy` | 4 products + shared confluence scoring & trade geometry |
| `internal/gates` | Hard risk gates (R:R floor, broker stop/freeze) |
| `internal/signal` | `Decide()` — orchestrates strategy + broker + license + gates |
| `internal/indicators` | EMA/SMA/ATR/RSI/MACD/ADX/Stoch/Boll/VWAP (pure math) |
| `internal/backtest` | Bars→snapshots (same math) + exit simulator + real CSV loader |
| `internal/provider` | Live HTTP gateway → `PAT_signals.txt` |
| `internal/store` | TimescaleDB + Valkey persistence (degradable) |
| `cmd/{engine,backtest,gateway,agent,sim}` | Binaries |
| `mql/` | Reference EAs that already parse `PAT_signals.txt` |
| `infra/db/init/01_schema.sql` | TimescaleDB schema (database `pat_engine`) |
| `docker-compose.yml` | TimescaleDB + Valkey + engine services |

---

## 3. Strategies (config-backed, versioned)

| ID | Class | SL×ATR | TP1×ATR | Min R:R | Min Conf | Min ADX |
|---|---|---|---|---|---|---|
| `ULTRA_SCALPING` | SCALP | 1.0 | 2.0 | 2.0 | 65 | 25 |
| `STANDARD_SCALPING` | SCALP | 1.0 | 1.5 | 1.5 | 65 | 20 |
| `STANDARD_SWING` | SWING | 1.5 | 2.5 | 1.5 | 55 | 20 |
| `TREND_SWING` | TREND | 2.0 | 3.0 | 1.5 | 50 | 20 |

All thresholds live in `internal/config` — one place, historically reproducible.

---

## 4. Broker scalping policy (first-class)

Many brokers forbid scalping. `BrokerPolicy.AllowsScalping=false` **excludes both
scalping strategies** (`BROKER_SCALPING_NOT_ALLOWED`); only `STANDARD_SWING` and
`TREND_SWING` remain eligible. Verified by the backtest harness and decision tests.

> Strategy enable/disable is a **server/control-plane (license)** concern
> (`allowed_strategies`), never an EA input.

---

## 5. License management

Enforced **server-side in the engine** (never an EA input):
- `internal/license`: HMAC-signed token (`base64(json).hex(hmac)`) with `key`, `plan`,
  `allowed_strategies`, `device_id`, `expires_at`, optional `broker_scalping`.
- Gateway + backtest select **only** license-entitled strategies; non-entitled ones are
  suppressed even when the broker would allow them.
- Env: `PAT_LICENSE` (signed token) + `PAT_LICENSE_SECRET`. No license ⇒ DEV license
  (all strategies). Full detail in `SCOPE_OF_WORK.md` §4.

---

## 6. Data & persistence

**Database: `pat_engine` (TimescaleDB).** Two hypertables:
- `bars(ts, symbol, o/h/l/c, spread)` — every ingested bar.
- `signals(id PK, ts, symbol, strategy_id, direction, entry, sl, tp1..tp3, raw_score,
  grade, signal_class, status, reasons jsonb)` — **every** decision (executable or
  blocked) for full auditability.

**Valkey** holds:
- `pat:signal:latest:<strategy>` — cached last executable signal (TTL 24h).
- `pat:signals` pub/sub channel — live push to connected agents (EA file is the
  primary delivery; pub/sub is the low-latency alternative).

The store is **degradable**: if Postgres/Valkey are unreachable the engine keeps running
on an in-memory ring buffer and logs the degradation — a missing datastore never blocks
signal generation.

---

## 7. Run with Docker (recommended)

```bash
cd pat-engine
cp infra/env/ENV_SAMPLE infra/env/.env      # set PAT_DB_PASSWORD etc.
docker compose --env-file infra/env/.env up -d
#   pat-engine-db      :5433  (TimescaleDB, db=pat_engine)
#   pat-engine-cache   :6380  (Valkey)
#   pat-engine         :8080  (gateway)
```

To start only the datastore for local dev:
```bash
docker compose up -d pat-engine-db pat-engine-cache
```

---

## 8. Local (without Docker) — binaries

```bash
cd pat-engine
go build ./...

# Demo: evaluate all 4 strategies over the sample snapshot feed
go run ./cmd/engine

# Research: PF / win-rate per strategy (synthetic; swap real bars via CSV)
go run ./cmd/backtest
#   real data:  BARS_CSV=real_2025.csv go run ./cmd/backtest

# LIVE loop (terminal 1 gateway, 2 EA simulator, 3 agent)
export PAT_DB_DSN="postgres://pat:pat_engine_dev@localhost:5433/pat_engine"
export PAT_REDIS_URL="redis://localhost:6380"
go run ./cmd/gateway &          # writes signals/PAT_signals.txt
go run ./cmd/sim &              # consumes it like the EA would
go run ./cmd/agent              # streams 3000 synthetic bars -> gateway
# real bars instead:  BARS_CSV=my_xauusd.csv go run ./cmd/agent
```

### Gateway environment

| Var | Default | Meaning |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `SIGNAL_FILE` | `signals/PAT_signals.txt` | EA signal file path |
| `PAT_DB_DSN` | _(empty)_ | Postgres DSN; empty ⇒ persistence off (in-memory) |
| `PAT_REDIS_URL` | _(empty)_ | Valkey URL; empty ⇒ no cache/pubsub |
| `PAT_LICENSE` / `PAT_LICENSE_SECRET` | _(empty)_ | entitlement token |

### Gateway endpoints
- `POST /bar` — JSON `{time,open,high,low,close,spread}`; runs the pipeline.
- `GET /signal` — last emitted signal line.
- `GET /health` — liveness.

---

## 9. Signal file format (consumed by the EA)

`SIGNAL|<json>` where json carries exactly the keys the MQL `ExtractJSONString` parser
expects: `ID, Direction, Grade, StrategyID, SignalClass, EntryPrice, StopLoss, TP1,
TP2, TP3, SuggestedLot, RawScore, CalibratedProbability`.

---

## 10. Testing

```bash
go test ./...
```
Covers: strategy executability + R:R floor, broker scalping exclusion, high-spread
block, license sign/parse/tamper/expiry/device + license suppression of non-entitled
strategies, and store degraded-mode (no datastore ⇒ no panic, in-memory retained).

**Live persistence verified:** streaming bars persisted 6000 `bars` + 967 `signals`
rows to TimescaleDB; Valkey cached the latest signal and published on `pat:signals`.

---

## 11. Operations

- **Backup:** `pg_dump pat_engine > pat_engine.sql` (TimescaleDB-aware with
  `--format=custom` for hypertables). Valkey is a cache; not a durability requirement.
- **Schema migrations:** idempotent `infra/db/init/01_schema.sql`, applied on first
  container start; new columns/migrations are additive and re-runnable.
- **No systemd:** all services are containers (`docker compose`); never systemd.

---

## 12. Status & deferred

**Done:** 4 strategies, broker scalping policy, license management, hard risk gates,
backtest (real-data loader), live gateway, TimescaleDB + Valkey persistence, E2E proof.

**Deferred (per plan: backend + MQL first):** frontend/Command Center, control-plane
license **issuance** service (engine already verifies), rebuilt Windows Agent binary
(reuse `windows-agent`, point at `POST /bar`), calibrated probability model, and
walk-forward/OOS validation on real 2025 data to lock honest v1 stats.

See `SCOPE_OF_WORK.md` for traceability and `docs/PROJECT_RESET_PLAN.md` for the
reset rationale.
