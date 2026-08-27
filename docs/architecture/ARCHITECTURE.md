# Predict-A-Trade Architecture
## v1.16.0 — 26 August 2026

### Signal Flow

```
EXTERNAL SOURCES (MT5, TwelveData, FMP, Ollama, News)
                    |
                    v
Go REALTIME ENGINE (:13081)
  Market Ingest -> Feature Registry (42 ind) -> Strategy Engines (5)
                                                    |
                                                    v
                                          Signal Engine + 16 Gates
                                          (deterministic, fail-closed)
                                                    |
                               TimescaleDB + Valkey + WebSocket
                                    |              |            |
                                    v              v            v
                            Next.js :13082  NestJS :13080  Windows/MQL
                            (Frontend)     (IAM/Billing)  (Execution)
```

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

### Services (11 total)

| Service | Port | Tech | Status |
|---------|:----:|------|:------:|
| Realtime Engine | 13081 | Go | LIVE |
| Control Plane | 13080 | NestJS | LIVE |
| Frontend | 13082 | Next.js | LIVE |
| Status Page | 13083 | Go | LIVE |
| Live Terminal | 13090 | Go | LIVE |
| PostgreSQL | 5432 | TimescaleDB | LIVE |
| Valkey | 6379 | Cache | LIVE |
| Prometheus | 9090 | Metrics | LIVE |
| Grafana | 3001 | Dashboards | LIVE |
| ntfy | 8091 | Alerts | LIVE |
| Nginx | 80/443 | Reverse proxy | LIVE |

### Recent Architectural Changes (v1.16.x)

**Broker Timezone (GMT+3):** Session engine, ORB ranges, and all hour-of-day classification now use broker-local time instead of UTC. This fixes a 3-hour offset in session boundaries that affected signal timing display. Controlled by `BROKER_TIMEZONE` env var (defaults to `GMT+3`).

**Capital-Loss Protection (5%):** New `SeedCapitalProtection` gate enforces a fail-closed daily loss cap of 5% of account equity. Engine computes account-size-aware position sizing (`SuggestedLot`, `RiskDollars`, `RiskPctOfEquity`, `SLDistancePoints`) and annotates every signal with these metrics.

**Client EA daily-loss guard (separate from server gates):** The Execution EA enforces its own client-side guard independent of the server risk gates above — a **soft** limit (`WarningLossPct`) blocks new entries only and recovers intraday (day boundary = broker/server day; day-open balance derived from realized P&L so re-attaching mid-day does not overstate the loss), and a **hard** limit (`MaxDailyLossPct`) closes all positions as a non-bypassable emergency backstop. The soft limit can be bypassed by the operator via the `BypassDailyLossBlock` EA input. `AutoExecute` now defaults to **false** (signal-only). See the [Windows Agent Guide](../guides/WINDOWS_AGENT.md).

**Gate State Isolation:** All 16 gates now maintain per-(strategy, timeframe) state to prevent cross-strategy contamination. Each strategy+timeframe pair has independent gate tracking.

**Operator Edge-Arming:** Gates can be armed by operator for specific strategies, enabling broker-position authorization for live EXECUTABLE delivery. `ExecutionPermission` gate now supports per-strategy operator override.

**Live Agent Bridge:** The Go engine now bridges live agent connection state (WebSocket heartbeat status, agent version, license status) into the control-plane database for unified agent monitoring from the admin dashboard.

**Frontend Signal Panel Pagination:** Both admin (20/page) and user (15/page) signal tables use client-side pagination to prevent browser lockup with large signal volumes. Full TP1/TP2/TP3 columns with per-level R:R ratios.

**Windows Agent Role Split (v1.2.32):** The Windows Agent ships as two **separate binaries** — **Client Agent** (execution, `pat-agent.exe`, built from `cmd/client`, service `pat-agent-client`, engine exec port 13081) and **Master Node** (data-only, `pat-master.exe`, built from `cmd/master`, service `pat-agent-master`, engine data port 13091). The role is fixed by the binary (no runtime `--mode` flag). The download server serves role-specific subfolders (`…/windows-agent/client/` and `…/master/`) so each role's binary is fetched from an unambiguous URL, while shared assets (NSSM, settings, scripts, version) come from the root. See the [Windows Agent Guide](../guides/WINDOWS_AGENT.md).
