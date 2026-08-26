# Predict-A-Trade XAUUSD

Multi-plane XAUUSD trading signal generation and analytics platform.

**Version:** v1.16.0 | **Date:** 26 August 2026 | **Verdict:** GO (100/100)

## Quick Start

```bash
git clone https://github.com/simhaonline/predictatrade.git
cd predictatrade/xauusd
cp realtime/.env.example realtime/.env
# Edit realtime/.env: set TWELVEDATA_API_KEY, FMP_API_KEY
docker compose up -d
curl http://localhost:13081/health
```

## Architecture

```
MT4/MT5 → Windows Agent → WebSocket → Go Realtime Engine :13081
                                          │
                    ┌─────────────────────┼──────────────────┐
                    ▼                     ▼                  ▼
            Market Ingestion        Feature Registry    Strategy Engines
            (candles/ticks)        (42 indicators)     (5 engines)
                    │                     │                  │
                    └─────────────────────┴────────┬─────────┘
                                                    ▼
                                          Signal Engine + 16 Risk Gates
                                          (deterministic, fail-closed)
                                                    │
                                    TimescaleDB + Valkey + WebSocket
                                                    │
                          ┌─────────────────────────┴──────────┐
                          ▼                                    ▼
                  Next.js Frontend :13082              Windows/MT Delivery
                  NestJS Control :13080
```

## Services

| Service | Container | Port | Role |
|---------|-----------|:----:|------|
| Realtime Engine | pat-realtime | 13081 | Go HTTP/WebSocket signal engine |
| Control Plane | pat-control | 13080 | NestJS IAM, billing, licensing |
| Frontend | pat-frontend | 13082 | Next.js user/admin dashboards |
| Status Page | pat-status | 13083 | System health status |
| Live Terminal | pat-live-terminal | 13090 | Bloomberg-style terminal |
| PostgreSQL | pat-postgres | 5432 | TimescaleDB hypertables |
| Valkey | pat-valkey | 6379 | Cache and hot state |
| Nginx | pat-nginx | 80/443 | Reverse proxy, TLS, WS routing |
| Prometheus | pat-prometheus | 9090 | Metrics collection |
| Grafana | pat-grafana | 3001 | Dashboards |
| ntfy | pat-ntfy | 8091 | Notifications |

## Strategy Engines

| Engine | ID | TFs | Min Score | Expiry | Status |
|--------|----|-----|:---------:|:------:|:------:|
| Standard Scalping | STANDARD_SCALPING | M1/M5 | 65 | 10m | LIVE |
| Ultra Scalping | ULTRA_SCALPING | M1 | 60 | 5m | LIVE |
| Standard Swing | STANDARD_SWING | M15/H1 | 68 | 30m | LIVE |
| Trend Swing | TREND_SWING | H1/H4 | 70 | 60m | LIVE |
| MARNIE_FIB | MARNIE_FIB | H1 | 70 | 60m | SHADOW |

## Evidence Scoring

13 pillars with family caps: TREND(0.35), MOMENTUM(0.30), STRUCTURE(0.25), LIQUIDITY(0.20), SMC(0.20), MTF(0.20), CANDLE(0.20), REGIME(0.15), VWAP(0.15), VOLATILITY(0.15), ML(0.25), SENTIMENT(0.25), SESSION_ORB(0.15)

42 indicators, 35 live, 7 warming up.

## Risk Gates (16 gates, ordered)

ExecutionPermission → BrokerSymbolValidation (P0) → SeedCapitalProtection → DailyLossLimit → MaxSpread → NewsRisk → Slippage → MaxPositions → MaxExposure → Cooldown → StopHuntFilter → MarginCheck → OvertradeProtection → MaxDailyTrades → RegimeFilter → ProfitTarget

## v1.16.0 Features (P2 — ACTIVE)

- P2-001: Session ORB — Asian/London/NY opening ranges, breakout detection
- P2-002: Pin Bar geometry — body/wick ratios, rejection scoring
- P2-003: Pullback detection — depth %, ATR retracement, continuation
- P2-004: Trade Group ID — multi-position signal tracking
- P2-005: SLO targets — availability, latency, error budgets

### v1.16.0 Frontend Features
- **Signal Panel pagination:** 20/page admin, 15/page user — prevents browser lockup with large signal volumes
- **TP2/TP3 columns:** Displayed alongside TP1 in both admin and user signal tables with per-level R:R ratios
- **Quality Grade:** A+, A, B, REJECTED badges on signal rows
- **Expectancy metrics:** EV_R (expected value per unit risk) and ExpectancyScore (0-100)
- **Capital-protection sizing:** SuggestedLot, RiskDollars, RiskPctOfEquity, SLDistancePoints displayed in expandable rows
- **Calibrated probability:** Shows "Pending" until calibration model is validated (§16, §36)
- **Signal Class:** ADVISORY vs EXECUTABLE classification with color coding
- **Multi-tab strategy filtering:** All 5 strategy engines (including MARNIE_FIB) with directional sub-filters

## Plane Boundaries (mandatory)

| Plane | Location | Authority | Must NOT become |
|-------|----------|-----------|-----------------|
| Go Realtime | realtime/ | Market data, features, signals, gates | Synchronous billing |
| NestJS Control | control/ | IAM, subscriptions, billing, licensing | Tick-to-signal hot path |
| Next.js Frontend | frontend/ | UI rendering | Risk/entitlement authority |
| Python Research | research/ | Backtesting, calibration, ML | Live tick dependency |
| Windows/MQL Edge | windows-agent/, mql/ | Order execution | Primary intelligence |

## Current Status (26 August 2026)

| Check | Status |
|-------|:------:|
| Go tests (28/28 packages) | PASS |
| Frontend tests (70) | PASS |
| Python tests (127) | PASS |
| TypeScript check | PASS |
| All services running | PASS |
| 16 risk gates active | PASS |
| SL enforcement server-side | ACTIVE |
| Broker symbol validation (P0-001) | ACTIVE |
| Price precision rounding (P1-001) | ACTIVE |
| Math parity (MAPE < 0.0001) | PASS |
| 49/49 geometry validations | PASS |
| 5 production blockers | ALL CLOSED |
| MQL EAs compiled | Operator action |
| Production API keys | Operator action |
| Backup/restore tested | Operator action |

## Go Realtime Plane

Located in `realtime/`. Key packages:

- `internal/marketdata` — agent provider, tick/candle aggregation, COT, DXY providers
- `internal/features` — 42 indicators, structure, regime, VWAP, Fibonacci, FVG, pivot points
- `internal/strategy` — 5 strategy engines, evidence scoring, confluence, geometry
- `internal/gates` — 16 hard risk gates (ordered, fail-closed)
- `internal/signal` — master decision engine, cooldown, duplicate prevention
- `internal/gateway` — HTTP, dashboard WS, Windows Agent WS handlers
- `internal/crossmarket` — DXY, BTC, Oil macro module
- `internal/ml` — ONNX model inference (advisory)
- `internal/sentiment` — Ollama sentiment analysis (advisory)
- `internal/ptb` — Professional Trader Brain intelligence layer
- `pkg/health`, `pkg/news`, `pkg/macro`, `pkg/mt5` — public utilities

## Documentation

- [SCOPE_OF_WORK.md](realtime/SCOPE_OF_WORK.md) — Full project scope and specifications
- [CHANGELOG.md](realtime/CHANGELOG.md) — Version history v1.0-v1.16.0
- [DOCKER_COMPOSE_REFERENCE.md](realtime/DOCKER_COMPOSE_REFERENCE.md) — Docker architecture
- [PRODUCTION_READINESS_AUDIT.md](realtime/PRODUCTION_READINESS_AUDIT.md) — Audit: 100/100
- [docs/](docs/) — Architecture, strategy playbooks, indicators, gates, API, database
- [Docker Deployment Guide](docs/operations/DOCKER_DEPLOYMENT.md) — Step-by-step Docker Compose (14 steps)
- [Host Deployment Guide](docs/operations/HOST_DEPLOYMENT.md) — Step-by-step bare-metal/VPS (14 steps)
- [Admin Guide](docs/guides/ADMIN_GUIDE.md) — System administration
- [User Guide](docs/guides/USER_GUIDE.md) — Dashboard, strategies, MT4/MT5 setup

## Build & Test

```bash
# All services
make build && make test && make lint

# Individual planes
make go-build          # Go realtime engine
make go-test           # Go tests (28 packages)
make control-build     # NestJS control plane
make frontend-build    # Next.js frontend
make research-test     # Python tests (127 tests)

# Docker
docker compose up -d --build
docker compose ps
```

## Production Safety

Without explicit operator authorization, do NOT:
- Enable live automated trading
- Place or close live broker orders/positions
- Mutate real subscriptions, commissions, wallets or payouts
- Run destructive production migrations
- Export secrets or rotate signing keys

NO-TRADE is a valid first-class result. ML/AI components are advisory only and cannot override deterministic gates.

## License

MIT License — see [LICENSE](LICENSE)
