# Predict-A-Trade Architecture
## v1.16.0 — 26 August 2026

### Plane Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                    EXTERNAL DATA SOURCES                     │
│  MT5 Broker | TwelveData | FMP (COT) | Ollama (AI) | News  │
└────────────────────────┬────────────────────────────────────┘
                         ▼
┌─────────────────────────────────────────────────────────────┐
│              Go REAL-TIME ENGINE (:13081)                    │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌───────────────┐  │
│  │ Market   │ │ Features │ │ Strategy │ │ Signal Engine │  │
│  │ Ingest   │→│ Registry │→│ Engines  │→│ + 16 Gates    │  │
│  │ 42 ind.  │ │ 35 live  │ │ 5 engines│ │ deterministic │  │
│  └──────────┘ └──────────┘ └──────────┘ └───────┬───────┘  │
│                                                  │          │
│  ┌───────────────────────────────────────────────┘          │
│  │  TimescaleDB + Valkey + WebSocket                        │
└──┼──────────────────────────────────────────────────────────┘
   │
   ├──────────────┬──────────────────┐
   ▼              ▼                  ▼
┌──────────┐ ┌──────────┐  ┌──────────────┐
│ Next.js  │ │ NestJS   │  │ Windows/MQL  │
│ Frontend │ │ Control  │  │ Edge         │
│ :13082   │ │ :13080   │  │              │
│          │ │ IAM,     │  │ MT4/MT5 EAs  │
│          │ │ Billing, │  │ Trade Exec   │
│          │ │ Licensing│  │              │
└──────────┘ └──────────┘  └──────────────┘
```

### Plane Boundaries (mandatory enforcement)

| Plane | Dir | Authority | Must NOT |
|-------|-----|-----------|----------|
| Go Realtime | realtime/ | Market data, features, signals, gates | Synchronous billing |
| NestJS Control | control/ | IAM, subscriptions, billing, licensing | Tick-to-signal path |
| Next.js Frontend | frontend/ | UI rendering | Risk/entitlement authority |
| Python Research | research/ | Backtesting, calibration, ML | Live tick dependency |
| MQL Edge | mql/, windows-agent/ | Order execution | Primary intelligence |

### Data Flow (per tick → signal)

1. Market data arrives (MT5/API) → candle cache (Valkey)
2. Feature Registry computes 42 indicators on bar-close
3. Each strategy evaluates MarketState against its criteria
4. Evidence scored across 13 pillars (family-capped)
5. 16 hard gates evaluate in order (fail-closed)
6. Signal Engine assigns direction, SL, TP
7. Signal persisted to TimescaleDB + broadcast via WebSocket
8. Outcome Resolver tracks TP/SL hits for performance

### Service Inventory

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