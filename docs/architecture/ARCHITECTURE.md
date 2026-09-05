# Predict-A-Trade Architecture
## v1.29.0 — 04 September 2026

> Visual flows: tick→signal sequence, execution reconciliation, auth, licensing, payments,
> update/rollback, backup/DR are in **[FLOW_DIAGRAMS.md](FLOW_DIAGRAMS.md)** (Mermaid).

### Signal Flow

```
EXTERNAL SOURCES (MT5, TwelveData, FMP, Ollama, News)
                    |
                    v   Option B (v1.19+): EA-direct HTTPS transport — no Windows Agent
MT4/MT5 EAs (Master Nodes stream data; Client EAs poll signals)
   |  POST /ingest/agent  (Bearer device JWT, TYPE|{json} lines)
   |  HMAC edge-poll every ~3s (always-ACK) for signals + server commands
                    |
                    v  (IngestBus: DirectBus in-process, or NatsBus when NATS_URL set)
Go REALTIME ENGINE (:13081, single port — WS /ws/v1 for browsers, REST /api/v1/*)
  Ingest -> marketdata AgentProvider -> [tick goroutine | candle goroutine]  (split loops, v1.28.3)
       |                                     |
  Feature Registry (42 ind)          Strategy Engines (7)
                    |                         |
                    v                         v
             Signal Engine + 16 Gates  (deterministic, fail-closed)
                    |
   TimescaleDB + Valkey + WebSocket (per-client risk filter)
        |              |            |
        v              v            v
  Next.js :13082   NestJS :13080   MQL Client EAs (execution via edge-poll + ACK)
  (Frontend)       (IAM/Billing)
```

> **Transport (v1.19.0 Option B, current):** MetaTrader EAs connect DIRECTLY to the
> cloud — the Windows Agent tree, installers, dedicated data port 13091, and
> `audit.agent_connections` were all removed. The **Master EA** (any MT4/MT5
> terminal with the data role) ingests `MARKET_SNAPSHOT`/`MASTER_TICK` bars and
> ticks via `POST /ingest/agent` (Bearer device JWT, `PROVIDER_MODE=agent`); the
> **Client EA** activates its device with the license key and polls the control
> plane (`edge-poll`, HMAC-signed) for executable signals and server commands,
> ACKing each. Delivery is fail-closed and plan-filtered: only `Executable == true`
> signals enqueue, and only for devices whose license + plan whitelist the signal's
> strategy — with a second entitlement re-check at poll time. The engine never
> fabricates ticks — when the Master EA stops streaming, the feed reports
> `NO_DATA`, not a fake "live".

> **Ingest/signal decoupling (v1.17.0):** inbound EA messages travel through
> a `pkg/bus` abstraction (`realtime/pkg/bus`). The default `DirectBus` calls
> the engine handler in-process (identical to the pre-NATS path). When
> `NATS_URL` is set on the `realtime` service, a `NatsBus` enqueues messages on
> the `pat-nats` service and a subscriber dispatches them to the same engine
> handler — isolating data-collection throughput from signal processing and
> allowing a dedicated ingest service later. Connection failure falls back to
> in-process automatically.

### Timezone Model
The broker server runs on **GMT+2 winter / GMT+3 summer** (Equiti Master Node; DST-following). All session classification, ORB ranges, and hour-of-day logic use broker-local time via `BrokerLocation()`. Precedence (v1.28): (1) the offset observed **live from the Master Node** (`TimeGMTOffset()` reported on every tick — authoritative, matches the exact clock the EAs' `TimeCurrent()` runs on, and rolls automatically at the DST change), (2) `BROKER_TIMEZONE` env (IANA name or fixed `+2`/`-5` offset), (3) fixed GMT+2 default (winter value). Absolute instants are stored as TIMESTAMPTZ (UTC) in Postgres and every API timestamp is a UTC RFC3339 instant; only hour-of-day logic converts to broker time. `/health` reports the resolved mode as `time_mode: "BROKER_ALIGNED"` plus the live `broker_offset` hours.

**EA clock authority (v1.28):** ALL EA trading logic — `iTime()`/`CopyRates` bar math, session/swap-hour windows (`IsNearSwapTime`, `IsTripleSwapDay`), signal TTL (`ExpiresAt`, `MaxSignalAgeSeconds`), and order expiry — runs on the broker clock `TimeCurrent()` returns, never on Windows-local time. Client EAs anchor the TTL age gate on the payload's server issue timestamp (`CreatedAt`/`IssuedAt`, UTC ISO) when present, so edge-queue replays cannot reset a signal's TTL. External timestamps from other zones convert via `PAT_LocalToBroker(iso, srcOffsetMinutes)` (in every EA file), which maps a source wall-clock instant (e.g. Dubai GMT+4 → `240`, UTC → `0`) onto the broker timeline — **DST-adaptive with no hardcoded offsets** (v1.28.1): the conversion subtracts the live `TimeLocal() − TimeGMT()` diff and adds the live `TimeLocal() − TimeCurrent()` diff (`PAT_UTCToBrokerWall` bridge), so the Equiti GMT+2 winter / GMT+3 summer change and the Windows PC's own DST roll are both picked up automatically on the next tick. The frontend renders signal timestamps on the same broker clock via `formatBrokerTimestamp()` (`frontend/src/lib/use-server-time.ts`), not the browser's timezone.

### Master Node market-data storage (v1.28.2)
Master MT4 and MT5 data share the single `market.ticks` / `market.candles` hypertables, separated by the `source` column (`MT4_MASTER` / `MT5_MASTER`) — one table keeps parity joins, retention and backfills cheap while the ON CONFLICT key `(time, symbol, source)` prevents cross-terminal overwrites. Per-terminal "separate table" ergonomics are provided by views `market.v_ticks_master_mt4`, `market.v_ticks_master_mt5`, `market.v_candles_master_mt4`, `market.v_candles_master_mt5` (mig 136). Legacy alias source strings (`MT5_MASTER_NODE`, `M_MASTER`, `MT4_MASTER_NODE`) are auto-normalized to canonical keys by `market.normalize_master_source()` triggers on write, and CHECK constraints (`chk_*_master_source`) reject any other `*MASTER*` value — the alias drift that fragmented parity queries is structurally closed. The MT5 Master EA bar-event path emits the canonical `MT5_MASTER` string directly (v1.28.2). Monthly candles: EAs emit the MQL5-standard `MN1` timeframe name; a write trigger (mig 137) normalizes it to the canonical `MN` used by `market.candles` — monthly bars persist cleanly and the `candles_timeframe_whitelist` rejection path is closed.

### Realtime drain loops (v1.28.3)
The engine runs **separate goroutines** for tick and candle drainage (`realtime/cmd/realtime-engine/main.go`). The earlier single `select` loop starved ticks whenever candle-driven strategy evaluation saturated the goroutine (observed ~5 candles/s across timeframes → a 20-minute tick backlog behind fresh candles). Now ticks (`tickChan`, cap 4096) and candles (`candleChan`) drain independently; candle bursts can no longer delay tick persistence, signal freshness, or TTL math. Deployed alongside mig137 on 2026-09-04 — tick lag verified 0.0 min on both Master feeds after the split.

### Plane Boundaries (mandatory)

| Plane | Dir | Authority | Must NOT |
|-------|-----|-----------|----------|
| Go Realtime | realtime/ | Market data, features, signals, gates | Synchronous billing |
| NestJS Control | control/ | IAM, subscriptions, billing | Tick-to-signal path |
| Next.js Frontend | frontend/ | UI rendering | Risk/entitlement authority |
| Python Research | research/ | Backtesting, calibration | Live tick dependency |
| MQL Edge | mql/ | Order execution | Primary intelligence |

### Services (13 LIVE + 1 OPT-IN)

| Service | Port | Tech | Status |
|---------|:----:|------|:------:|
| Realtime Engine | 13081 | Go 1.25 | LIVE |
| Control Plane | 13080 | NestJS 12 | LIVE |
| Frontend | 13082 | Next.js 16.3 | LIVE |
| Status Page | 13083 | Node | LIVE |
| Live Terminal | 13090 | Go | LIVE |
| PostgreSQL | 5432 | TimescaleDB | LIVE |
| Valkey | 6379 | Cache | LIVE |
| Prometheus | 9090 | Metrics | LIVE |
| Grafana | 3001 | Dashboards | LIVE |
| ntfy | 8091 | Alerts | LIVE |
| Nginx | 80/443 | Reverse proxy/TLS | LIVE |
| Backtest service | 8088 | Python (FastAPI) | LIVE |
| NATS | 4222/8222 | Ingest bus (optional) | OPT-IN (`NATS_URL`) |

> Port 13091 (dedicated agent data-WS) was removed with the Windows Agent in v1.19.0 — all agent traffic now shares 13081 via `POST /ingest/agent`.

### NestJS 12 / ESM notes (v1.17.3)

- Control plane upgraded NestJS 10→12 with TypeScript 6 + Jest 30; Nest 12 packages are
  ESM-only, so the Jest runtime requires `NODE_OPTIONS=--experimental-vm-modules`
  (Node 24.19) — baked into `npm test` and CI.
- `@nestjs/throttler@6.5.0` peer-declares ≤11 but is runtime-compatible with 12
  (API surface verified, 167/167 tests); `control/.npmrc` pins `legacy-peer-deps=true`.
- PostgreSQL 17 strict parameter-type inference surfaced a real bug in
  `POST /subscriptions` (untyped `$5` used as column + CASE operand) — fixed with
  `$5::text` casts; subscription creation verified end-to-end.

### BE-6 Reconciliation (v1.17.3)

Signal↔execution lifecycle is now fully observed end-to-end:
`RecordDelivery` (broadcast) → `RecordAcknowledgement` (verified EXECUTION_ACK) →
`RecordFill` (TRADE_RESULT, keyed by broker ticket). A 30s monitor emits
`pat_reconciliation_{acks_timeout,fills_timeout,tracked_signals}` gauges + deduped ntfy
alerts (ACK TTL 2m, fill TTL 10m) and prunes reconciled records. Fail-observing only.

### Account-Type Detection & Adaptation (v1.27, mig 133/134/135)

Every EA classifies its own account — `Demo > Contest > Islamic > MicroCent > ECN >
STP > Standard` priority — via the account-type detector, INLINED into each
`.mq5`/`.mq4` (operator mandate: no external MQL files — every EA compiles
standalone; the shared-source `.mqh` was deleted after inlining; facade functions
`PAT_ATD_*` delegate to the inlined core). Detection is lazy-cached per login,
fail-safe to `Standard`, and Islamic detection is confirmed by 3 hourly rollover
checks (auto-reclassify if swap is ever observed). Execution adapts per type: cent
accounts size lots ÷100; ECN accounts open naked and attach SL/TP post-fill
(3 attempts, watchdog fail-closed backstop); STP adds a +2pt slippage buffer
to the cost yardstick; Islamic zeroes swap in P&L math. The classification
travels on every INIT/ACCOUNT_INFO/LICENSE_CHECK/EXECUTION_ACK/heartbeat and
persists through `SnapshotAccount.AccountType` → `edge_device_state.account_type`
(see DATABASE_ARCHITECTURE + API_REFERENCE). mig 135 seeds the
`licensing.strategy_parameters` reference (47 rows) and `account_types` is_latest
upserts; the engine heartbeat persists `licensing.account_types` per login.

### Per-Device Entitlement Delivery

Signal delivery applies a per-receiving-device entitlement check at enqueue time
(`enqueueSignalForDevices` → SQL filters on `licensing.edge_signal_queue`) and
re-checks entitlement at poll time (control-plane `edge-poll` handler).
Executable signals are queued only for `exec` devices whose license is
ACTIVE/PENDING and whose license + plan whitelist includes the signal's strategy,
with v1.23 capital-tier `EligibleTiers` matching (unknown tier → fail-open;
unresolvable plan → device skipped). A device with a blown account is isolated
via the fail-closed `AgentProvider.AgentAccountOK` account guard (free margin
> 0, snapshot < 60s old) — it can never block or contaminate another device's
signals.

### Ingest/Signal Decoupling Seam

The EA-direct inbound path uses the `pkg/bus` abstraction. `DirectBus` (default)
preserves the original in-process behavior; `NatsBus` (when `NATS_URL` is set)
routes inbound messages through the `pat-nats` service, decoupling the
data-collection plane from the signal engine. A separate ingest service can
later subscribe to / publish on the same subject without changing the engine.

### Broker Timezone (live-detected, v1.28)

Session engine, ORB ranges, and all hour-of-day classification use broker-local time instead of UTC. `BrokerLocation()` resolves the offset from the Master Node's live report first (authoritative — matches the EAs' `TimeCurrent()` clock), then `BROKER_TIMEZONE` env (IANA name or fixed `+HH` offset), then the GMT+2 default. Wired from the engine's `AgentProvider.BrokerOffsetHours()` every 30s via `features.SetLiveBrokerOffset`.

### Capital-Loss Protection (5%)

`SeedCapitalProtection` gate enforces a fail-closed daily loss cap of 5% of account equity. Engine computes account-size-aware position sizing (`SuggestedLot`, `RiskDollars`, `RiskPctOfEquity`, `SLDistancePoints`) and annotates every signal with these metrics.

### Client EA daily-loss guard (separate from server gates)

The Execution EA enforces its own client-side guard independent of the server risk gates — a **soft** limit (`WarningLossPct`) blocks new entries only and recovers intraday (day boundary = broker/server day; day-open balance derived from realized P&L so re-attaching mid-day does not overstate the loss), and a **hard** limit (`MaxDailyLossPct`) closes all positions as a non-bypassable emergency backstop. The soft limit can be bypassed by the operator via the `BypassDailyLossBlock` EA input. `AutoExecute` defaults to **false** (signal-only). See the [EA Client Guide](../guides/EA_CLIENT_GUIDE.md).

### Gate State Isolation

All 16 gates maintain per-(strategy, timeframe) state to prevent cross-strategy contamination. Each strategy+timeframe pair has independent gate tracking.

### Operator Edge-Arming

Gates can be armed by operator for specific strategies, enabling broker-position authorization for live EXECUTABLE delivery. `ExecutionPermission` gate supports per-strategy operator override.

### Live Agent Bridge

The Go engine bridges live EA connection state (edge-poll heartbeat status, EA version, license status, account type) into the control-plane database for unified device monitoring from the admin dashboard.