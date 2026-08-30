# Predict-A-Trade XAUUSD

Multi-plane XAUUSD trading signal generation and analytics platform.

**Version:** v1.18.0 | **Date:** 30 August 2026 | **Status:** GO — paper/sandbox/advisory signal operation. **NO-GO for live automated trading arming** (`LIVE_TRADING_AUTHORIZED=false`, fail-closed) pending explicit operator authorization + verified broker equity/order feed. Weekend-hardened: MT4/MT5 online through closed market, market_closed-aware signal gate, USDT-only payments with anti-scam settlement verification, mail relay live; v1.2.44 agents.

## Quick Start

```bash
git clone https://github.com/simhaonline/predictatrade.git
cd predictatrade/xauusd
# Secrets live in infra/env/.env (gitignored). Copy the template and fill values:
cp infra/env/.env.example infra/env/.env   # if template exists; otherwise use the provided infra/env/.env
# Edit infra/env/.env: JWT_SECRET, POSTGRES_PASSWORD, DATABASE_URL, BACKTEST_DB_URL, GF_SECURITY_ADMIN_PASSWORD
# Per-service config (API keys etc.) still comes from realtime/.env / control/.env / frontend/.env
docker compose --env-file infra/env/.env up -d
curl http://localhost:13081/health
```

> **IMPORTANT:** All `docker compose` commands MUST include `--env-file infra/env/.env` (the compose file no longer contains secret values — see `docs/reports/REMEDIATION_REPORT_2026-08-28.md`, SEC-1). Running `docker compose up -d` without it starts containers with blank secrets.

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

## Risk Gates (16 gates, ordered, per-(strategy, timeframe) isolated)

ExecutionPermission → BrokerSymbolValidation (P0) → SeedCapitalProtection (5% daily cap) → DailyLossLimit → MaxSpread → NewsRisk → Slippage → MaxPositions → MaxExposure → Cooldown → StopHuntFilter → MarginCheck → OvertradeProtection → MaxDailyTrades → RegimeFilter → ProfitTarget

Gate state is isolated per (strategy, timeframe) to prevent cross-strategy contamination. Operator edge-arming enables per-strategy broker-position authorization for EXECUTABLE delivery.

## v1.17.x Features

- **Per-client risk isolation at delivery** — executable signals are forwarded only to clients whose own broker account has free margin; a blown client can never block others (fail-open on stale/unknown state).
- **Ingest/signal decoupling seam** — inbound agent messages route through a `pkg/bus` abstraction (`DirectBus` in-process by default; `NatsBus` when `NATS_URL` is set).
- **Silent data-feed detection + auto-recovery** — a data-independent 10s health monitor detects a dead `MARKET_SNAPSHOT` feed (not masked by ticks), alerts via ntfy, and nudges agents with `REQUEST_SNAPSHOT`.
- **Master Node snapshot delivery fix** — `MasterAppend()` now truly appends (was truncating), so snapshots are no longer clobbered by tick writes.
- **Launch-blocker remediation** — secrets moved out of `docker-compose.yml` (env-file injection), migrations renumbered to unique prefixes + reconciled, fabricated probability eliminated (VALIDATED-gated calibration), news gate fails closed, GDPR erasure service, RBAC roles+permissions guard, decimal.js money math.

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

## v1.17.3 Features (29 August 2026)

- **NestJS 10→12 + TypeScript 6 + Jest 30** — closes the last supply-chain residual (js-yaml prototype-pollution HIGH); production dependency tree now 0 high/critical (lodash eliminated, multer 2.2.0, express 5 via platform-express@12). Jest 30 required because Nest 12 is ESM-only (`--experimental-vm-modules`).
- **BE-6 fill-level reconciliation (closed)** — TRADE_RESULT now closes the ACK→fill leg keyed by broker ticket; 30s reconciliation monitor with ACK TTL (2m) and fill TTL (10m), per-signal deduped ntfy alerts, Prometheus gauges `pat_reconciliation_{acks_timeout,fills_timeout,tracked_signals}`, retention pruning. Fail-observing only — never blocks trading.
- **Devil Liquidity duplicate-mark fix** — the reversal candle could itself re-qualify as a NEW displacement mark (median body shifts after the first mark), double-charging the same level. Guard: level match normalized by the mark's DETECTION ATR + recency window (`RECLAIM_MAX_BARS` x timeframe duration).
- **CI rebuilt** — YAML indentation bug (30 consecutive failed runs, 0 jobs) fixed; secret-scan self-exclusion glob fixed; control `.npmrc` legacy-peer-deps documented; psycopg2 importorskip; 6/6 jobs green.
- **Dashboards runtime-audited (38/38 pages)** — every ADMIN and USER page probed against the live edge with USER + ADMIN tokens; fixed: `POST /subscriptions` 500 (PG17 `$5` type inference), stale e2e specs (cookie seeding, nav counts), 13 pre-existing lint errors.
- **Windows Agent v1.2.40 verified live** — binaries + manifests served byte-exact; installer env contract (7 vars) fully wired; EA sources (v1.17.2 MasterAppend fix) now downloadable from downloads.predictatrade.com.

## Plane Boundaries (mandatory)

| Plane | Location | Authority | Must NOT become |
|-------|----------|-----------|-----------------|
| Go Realtime | realtime/ | Market data, features, signals, gates | Synchronous billing |
| NestJS Control | control/ | IAM, subscriptions, billing, licensing | Tick-to-signal hot path |
| Next.js Frontend | frontend/ | UI rendering | Risk/entitlement authority |
| Python Research | research/ | Backtesting, calibration, ML | Live tick dependency |
| Windows/MQL Edge | windows-agent/, mql/ | Order execution | Primary intelligence |

## Current Status (29 August 2026)

| Check | Status |
|-------|:------:|
| Go tests (40/40 packages) | PASS |
| Control tests (NestJS 12, Jest 30) 167/167 | PASS |
| Frontend tests 84/84 + e2e 18/18 | PASS |
| Python tests 154 (153 pass, 1 skip) | PASS |
| CI — 6/6 jobs green | PASS |
| All 13 containers healthy | PASS |
| 16 risk gates active | PASS |
| SL enforcement server-side | ACTIVE |
| Broker symbol validation (P0-001) | ACTIVE |
| Price precision rounding (P1-001) | ACTIVE |
| Math parity (MAPE < 0.0001) | PASS |
| 49/49 geometry validations | PASS |
| Launch blockers (SEC-1/DB-1/DB-2/DB-5/BE-5/BE-4) | ALL CLOSED |
| BE-6 reconciliation monitor (ACK + fill legs) | LIVE |
| Supply chain (NestJS 12, 0 high/critical prod) | CLOSED |
| Dashboards wiring (38/38 pages runtime-probed) | PASS |
| Windows Agent v1.2.40 + installers + EA downloads | VERIFIED |
| Migration integrity (69 files, numbered to 099, unique prefixes) | PASS |
| Secrets out of git (env-file injection) | PASS |
| MT5 clients connected (EA attach + license) | Operator action |
| Demo fill test (one signal round-trip) | Operator action |
| Backup/restore drill | Operator action |
| Live automated trading arming | NO-GO (`LIVE_TRADING_AUTHORIZED=false`) |

> **Deployment is Docker-First.** All services run via `docker compose --env-file infra/env/.env`.
> Systemd units in `infra/systemd/` are DISABLED. Live automated trading is fail-closed: signals
> run in paper/sandbox/advisory mode only until an operator authorizes arming AND a verified broker
> equity/order feed exists. No profitability, accuracy, or hit-rate claims are made without evidence.

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
- [CHANGELOG.md](realtime/CHANGELOG.md) — Version history v1.0-v1.17.2
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
make go-test           # Go tests (40 packages)
make control-build     # NestJS control plane
make frontend-build    # Next.js frontend
make research-test     # Python tests (154: 153 pass, 1 skip)

# Docker (ALL commands MUST use --env-file infra/env/.env)
docker compose --env-file infra/env/.env up -d --build
docker compose --env-file infra/env/.env ps
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
