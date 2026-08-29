# Predict-A-Trade Architecture
## v1.17.4 — 30 August 2026

> Visual flows: tick→signal sequence, execution reconciliation, auth, licensing, payments,
> update/rollback, backup/DR are in **[FLOW_DIAGRAMS.md](FLOW_DIAGRAMS.md)** (Mermaid).

### Signal Flow

```
EXTERNAL SOURCES (MT5, TwelveData, FMP, Ollama, News)
                    |
                    v
Windows Agents (WS :13081 / data :13091)
                    |
                    v  (IngestBus: DirectBus in-process, or NatsBus when NATS_URL set)
Go REALTIME ENGINE (:13081)
  Market Ingest -> Feature Registry (42 ind) -> Strategy Engines (5)
                                                    |
                                                    v
                                          Signal Engine + 16 Gates
                                          (deterministic, fail-closed)
                                                    |
                               TimescaleDB + Valkey + WebSocket (per-client risk filter)
                                    |              |            |
                                    v              v            v
                            Next.js :13082  NestJS :13080  Windows/MQL
                            (Frontend)     (IAM/Billing)  (Execution)
```

> **Ingest/signal decoupling (v1.17.0):** inbound agent messages travel through
> an `IngestBus` abstraction (`realtime/pkg/bus`). The default `DirectBus` calls
> the engine handler in-process (identical to the pre-NATS path). When
> `NATS_URL` is set on the `realtime` service, a `NatsBus` enqueues messages on
> the `pat-nats` service and a subscriber dispatches them to the same engine
> handler — isolating data-collection throughput from signal processing and
> allowing a dedicated ingest service later. Connection failure falls back to
> in-process automatically.

### Timezone Model
The broker server runs in GMT+3 (standard XAUUSD FX broker time, no DST). All session classification, ORB ranges, and hour-of-day logic use broker-local time via `BrokerLocation()`, configurable through `BROKER_TIMEZONE` environment variable. Absolute instants are stored as TIMESTAMPTZ (UTC) in Postgres; only hour-of-day logic converts to broker time.

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
| Realtime data-WS | 13091 | Go (Master Node ingest) | LIVE |
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

### Recent Architectural Changes (v1.17.x)

**Per-Client Risk Isolation at Delivery:** Signal delivery now applies a
per-receiving-client risk check (`AgentHub.SetRiskCheck` →
`AgentProvider.AgentAccountOK`). Executable signals are sent only to clients
whose OWN broker account reports free margin > 0. A client with a blown account
is isolated — it can never block or contaminate another client's signals. The
check is fail-open (unknown or >60s-stale account state → allowed), and each
client's account is tracked individually in a per-agent registry, so there is no
shared global account-driven gate.

**Ingest/Signal Decoupling Seam:** The Windows-Agent inbound path uses the
`pkg/bus` abstraction. `DirectBus` (default) preserves the original in-process
behavior; `NatsBus` (when `NATS_URL` is set) routes inbound messages through the
`pat-nats` service, decoupling the data-collection plane from the signal engine.
A separate ingest service can later subscribe to / publish on the same subject
without changing the engine.

### Recent Architectural Changes (v1.16.x)

**Broker Timezone (GMT+3):** Session engine, ORB ranges, and all hour-of-day classification now use broker-local time instead of UTC. This fixes a 3-hour offset in session boundaries that affected signal timing display. Controlled by `BROKER_TIMEZONE` env var (defaults to `GMT+3`).

**Capital-Loss Protection (5%):** New `SeedCapitalProtection` gate enforces a fail-closed daily loss cap of 5% of account equity. Engine computes account-size-aware position sizing (`SuggestedLot`, `RiskDollars`, `RiskPctOfEquity`, `SLDistancePoints`) and annotates every signal with these metrics.

**Client EA daily-loss guard (separate from server gates):** The Execution EA enforces its own client-side guard independent of the server risk gates above — a **soft** limit (`WarningLossPct`) blocks new entries only and recovers intraday (day boundary = broker/server day; day-open balance derived from realized P&L so re-attaching mid-day does not overstate the loss), and a **hard** limit (`MaxDailyLossPct`) closes all positions as a non-bypassable emergency backstop. The soft limit can be bypassed by the operator via the `BypassDailyLossBlock` EA input. `AutoExecute` now defaults to **false** (signal-only). See the [Windows Agent Guide](../guides/WINDOWS_AGENT.md).

**Gate State Isolation:** All 16 gates now maintain per-(strategy, timeframe) state to prevent cross-strategy contamination. Each strategy+timeframe pair has independent gate tracking.

**Operator Edge-Arming:** Gates can be armed by operator for specific strategies, enabling broker-position authorization for live EXECUTABLE delivery. `ExecutionPermission` gate now supports per-strategy operator override.

**Live Agent Bridge:** The Go engine now bridges live agent connection state (WebSocket heartbeat status, agent version, license status) into the control-plane database for unified agent monitoring from the admin dashboard.

**Frontend Signal Panel Pagination:** Both admin (20/page) and user (15/page) signal tables use client-side pagination to prevent browser lockup with large signal volumes. Full TP1/TP2/TP3 columns with per-level R:R ratios.

**Windows Agent Role Split (v1.2.40):** The Windows Agent ships as two **separate binaries** — **Client Agent** (execution, `pat-agent.exe`, built from `cmd/client`, service `pat-agent-client`, engine exec port 13081) and **Master Node** (data-only, `pat-master.exe`, built from `cmd/master`, service `pat-agent-master`, engine data port 13091). The role is fixed by the binary (no runtime `--mode` flag). The download server serves role-specific subfolders (`…/windows-agent/client/` and `…/master/`), each with per-arch (`386`/`amd64`/`arm64`) binaries + `update-manifest.json`, while shared assets (NSSM, settings, scripts, version) come from the root. See the [Windows Agent Guide](../guides/WINDOWS_AGENT.md).
