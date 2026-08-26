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
