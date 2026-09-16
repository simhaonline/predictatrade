# Predict-A-Trade XAUUSD

Multi-plane XAUUSD trading signal generation and analytics platform.

**Engine v1.24.2 · EA v1.31 | Date:** 16 September 2026 | **Status:** GO — paper/sandbox/advisory signal operation. **LIVE TRADING ARMING AUTHORIZED BY OPERATOR (2026-08-30):** `LIVE_TRADING_AUTHORIZED=true` in `infra/env/realtime.env` (fail-closed capital-protection gates still require a verified broker equity/order feed; no self-promotion to live execution without it). **EA-direct cloud transport:** MetaTrader 4/5 EAs talk to the cloud directly over HTTPS (device activation → HMAC-signed edge-poll for signals/commands, Bearer ingest for market data). No local binaries, no services, no open ports on the trader's machine.

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

> **IMPORTANT:** All `docker compose` commands MUST include `--env-file infra/env/.env` (the compose file no longer contains secret values — see `docs/archive/2026-08-reports/REMEDIATION_REPORT_2026-08-28.md`, SEC-1). Running `docker compose up -d` without it starts containers with blank secrets.

## Architecture

```
MT4/MT5 (Master Node — data)  ──POST /ingest/agent──▶  Go Realtime Engine :13081
MT4/MT5 (Client Node — exec)  ◀──edge-poll signals/commands──┤
                                                      │
                    ┌──────────────────────────────────┼───────────────────────┐
                    ▼                                  ▼                       ▼
            Market Ingestion                   Feature Registry         Strategy Engines
            (candles/ticks)                   (76 feature fields)       (7 engines)
                    │                                  │                       │
                    └──────────────────────────────────┴─────────┬─────────────┘
                                                                 ▼
                                                   Signal Engine + 23 Risk Gates
                                                   (deterministic, fail-closed)
                                                                 │
                                         TimescaleDB + Valkey + WebSocket
                                                                 │
                           ┌───────────────────────────────────────┴───────────────┐
                           ▼                                                       ▼
                   Next.js Frontend :13082                               NestJS Control :13080 ×2 (HA)
                   (Live Command Center)                                (IAM/billing/licensing)
```

> **Option B (v1.19.0):** MetaTrader EAs connect DIRECTLY to the cloud — no Windows Agent,
> no WebSocket transport. The **Master EA** (data node, any MT4/MT5 terminal) ingests XAUUSD
> snapshots to the engine via `POST /ingest/agent` (Bearer device JWT, `PROVIDER_MODE=agent`);
> the **Client EA** activates its device with the license key and polls the control plane
> (`POST /api/v1/devices/edge-poll`, HMAC-signed, 3s cadence) for executable signals and
> server commands, ACKing each. Delivery is fail-closed and plan-filtered: only
> `Executable == true` signals enqueue, and only for devices whose license + plan whitelist
> the signal's strategy. The engine never fabricates ticks — when the Master EA stops
> streaming, the feed reports `NO_DATA`, not a fake "live".

## Services

| Service | Container | Port | Role |
|---------|-----------|:----:|------|
| Realtime Engine | pat-realtime | 13081 | Go HTTP/WebSocket signal engine |
| Control Plane (HA ×2) | pat-control, pat-control-b | 13080 | NestJS IAM, billing, licensing |
| Frontend | pat-frontend | 13082 | Next.js user/admin dashboards |
| Status Page | pat-status | 13083 | System health status |
| Live Terminal | pat-live-terminal | 13090 | Public preview terminal + 5-min trials |
| PostgreSQL | pat-postgres | 5432 | TimescaleDB hypertables |
| Valkey | pat-valkey | 6379 | Cache and hot state |
| Nginx | pat-nginx | 80/443 | Reverse proxy, TLS |
| Prometheus | pat-prometheus | 9090 | Metrics collection |
| Grafana | pat-grafana | 3001 | Dashboards |
| ntfy | pat-ntfy | 8091 | Notifications |
| Mail Relay | pat-mail-relay | 25/587 | DKIM-signed send-only SMTP relay |
| Backtest (FastAPI) | pat-backtest | 8088 (127.0.0.1) | ⚠️ removal candidate — control spawns the Go backtest-engine binary directly (bind-mounted `realtime/bin/backtest-engine`); no code consumer of :8088 was found |
| Watchdog | pat-watchdog | — | Health reconcile + alert mirrors (ntfy/Telegram/Discord). TICKS_STALE is alert-only — the master-tick feed comes from the external MT5 agent, so an engine restart cannot heal staleness (and each boot resets the in-process TwelveData daily credit budget). |
| Backup Sync | pat-backup-sync | — | Hetzner S3 (R2) WAL + pg_dump off-host sync |
| Discord Bot | pat-discord-bot | — | Alerts/portfolio/prediction slash commands (operator opt-in; exits 0 when `DISCORD_BOT_TOKEN` unset) |

Removed: `pat-nats` (never wired, zero Go consumers — audit 0ab2502; no longer in compose), `pat-control` legacy Windows Agent surface (v1.19.0 Option B).

## Strategy Engines

Source of truth: `realtime/internal/strategy/strategies.go` + `candidate_threshold.go` (thresholds current as of v1.24.2 — the earlier "Min Score 65/60/68/70" values predate the v1.1 threshold revision and were wrong).

| Engine | ID | Decision TFs | Trade threshold | Expiry | Status |
|--------|----|-----|:---------:|:------:|:------:|
| Standard Scalping | STANDARD_SCALPING | M1/M5 | 25 (candidate 10) | 10m | LIVE |
| Ultra Scalping | ULTRA_SCALPING | M1 | 25 (candidate 10) | 3m | LIVE |
| Standard Swing | STANDARD_SWING | M15/M30/H1 | 25 (candidate 10) | 60m | LIVE |
| Trend Swing | TREND_SWING | H1/H4 | 25 (candidate 10) | 240m | LIVE |
| EQFE | MARNIE_FIB | M15/H1 | 25 (candidate 10) | 120m | LIVE |
| ATEN | ATEN | H1 (extends to 120m) | 25 (candidate 10) | 60–120m | LIVE |
| IMLR | ARCANIST | M5/M15 | 25 (candidate 10) | 180m | ADVISORY |

> `MARNIE_FIB` is the internal strategy ID and is displayed to users as **EQFE**. The seven strategies are gated by plan entitlement: FREE → STANDARD_SCALPING only (max 5 signals/day); STANDARD → STANDARD_SCALPING + STANDARD_SWING; PRO → all 4 core; ELITE → all 7 (incl. EQFE, ATEN, IMLR). IMLR is delivered ADVISORY-only (not operator-armed for execution) until it completes validation/backtesting. Signal visibility is server-enforced.

## Evidence Scoring

13 pillars with family caps (source: `realtime/internal/strategy/strategies.go` `applyFamilyCaps`): TREND(0.35), MOMENTUM(0.30), STRUCTURE(0.25), LIQUIDITY(0.20), SMC(0.20), MTF(0.20), CANDLE(0.20), REGIME(0.15), VWAP(0.15), VOLATILITY(0.15), ML(0.25), SENTIMENT(0.25), SESSION_ORB(0.15)

Feature registry: ~90 distinct feature fields populated across the engine modules (50 in the core `IndicatorFeatures` struct + structure/FVG/liquidity/session-ORB/regime/VWAP/fibonacci/ichimoku families). The readiness map (`features/registry.go`) tracks 19 named capabilities: 10 READY, 7 WARMING_UP (rolling stats / pivot windows), 2 with data-source constraints (`Fibonacci`/`Structure` need confirmed swings), plus `VolumeProfile`/`CumulativeDelta` = UNSUPPORTED_BY_DATA_SOURCE (broker tick volume only) and `COT` = external-dependency status.

## Risk Gates (23 gates, ordered, per-(strategy, timeframe) isolated)

Source of truth: `realtime/internal/types/types.go` (GateID consts) + `realtime/internal/gates/gates.go` (`NewRegistry` base order) + `realtime/cmd/realtime-engine/main.go` (`RegisterOrdered` insertions). First veto short-circuits; unregistered/uninitialized gates fail closed.

DataQuality → WrongSideSL → Session → News → Spread → Slippage → TotalCost → MinATR → StopHuntFilter → Exposure → Margin → RiskOversize → PositionCaps → DailyLoss → ProfitTarget → MartingaleBan → RRNetExpectancy → Profitability → Entitlement → License → ExecutionPermit → EdgeValidation → BrokerSymbolValidation

(The earlier "24 gates" claim over-counted by one: 23 distinct GateIDs exist and all 23 are registered. `GatePolicyVersion`/`GateConfigVersion` are metadata fields on the signal object, not gates.)

Gate state is isolated per (strategy, timeframe) to prevent cross-strategy contamination. Operator edge-arming (`LIVE_TRADING_AUTHORIZED=true` + `EDGE_ARMED_STRATEGIES` list) enables per-strategy broker-position authorization for EXECUTABLE delivery.

## Market-Data Providers

| Feed | Provider | Notes |
|------|----------|-------|
| XAUUSD ticks/candles | MT5/MT4 Master Node EA → `POST /ingest/agent` | Authoritative; honest `NO_DATA` when the EA is offline |
| DXY (mandatory) | TwelveData | `DXY_ENABLED=true`; a hard 429 ⇒ strategy **NO-TRADE** (fail-closed) |
| Macro assets (VIX/BTC/WTI/EURUSD/USDCHF) | TwelveData batched `/quote` | 1 credit per batch call |
| COT | FMP API, CFTC Socrata fallback | FMP 402/403 = permanent → CFTC public report (verified live) |
| FRED real yield | FRED (DFII10) | No API key needed |
| Sentiment | Ollama (local) | Advisory only, `NOT_AI_VERIFIED` provenance |

**TwelveData free-tier budget:** 800 credits/day hard cap. The engine self-limits to 640/day (80% safety margin, `realtime/internal/marketdata/twelve_data_ratelimit.go`) — shared UTC-day budget across DXY + macro providers, resets at UTC midnight. ⚠️ The budget is **in-process**: every `pat-realtime` restart resets it, so watchdog restart-churn can re-burn credits (this is why TICKS_STALE remediation no longer restarts the engine).

## Current Development (16 September 2026)

- **Watchdog reconcile hardening (2026-09-16):** nginx container-name fixed (`pat-nginx` everywhere — the stale `xauusd-nginx-1` caused a false CONTAINER_DOWN alarm every 30s + broken disk probes); stack-reconcile mounts the whole `infra/env` dir (env_file resolution previously always FAILED); reconcile is now `up -d --no-recreate` (the in-container compose's config-hash differs from the host's — a plain `up -d` "recreated" healthy containers as drift, killing the watchdog itself mid-command); TICKS_STALE is alert-only.
- **TwelveData UTC-day credit budget guard (2026-09-13):** 800/day free tier guarded at 640; DXY/macro fail to NO-TRADE instead of feeding a 429 retry storm.
- **COT CFTC fallback (2026-09-13):** FMP 402/403 permanent failures now fall back to the CFTC public report — COT stays AVAILABLE.
- **EXECUTABLE delivery enabled (operator-authorized):** entitlement + gate cascade loosened per operator authorization; `LIVE_TRADING_AUTHORIZED=true` with armed strategies in `EDGE_ARMED_STRATEGIES`.
- **Server Migration runbook (2026-09-12):** full VPS→VPS procedure with R2 snapshot, secret inventory, restore, DNS cutover (`docs/operations/SERVER_MIGRATION.md`).
- **Capacity plan + tool (2026-09-11):** measured VPS vs dedicated sizing at 10k subscribers (`docs/operations/CAPACITY_PLAN.md`).
- **Cloudflare proxy hardening (2026-09-10):** all nginx upstreams moved to resolver-backed variabled `proxy_pass` — forced engine recreates no longer 502 the public edge.
- **v1.19.0 — Option B: EA-direct cloud transport (Windows Agent REMOVED).** See `docs/guides/EA_CLIENT_GUIDE.md`.
- **Engine "Market Feed Stale" fixes (honest status, no fake data):** `/api/v1/feeds` thresholds realigned (90s degraded / 180s stale), candle-quality monitor on real data-arrival windows, data-role re-established on first MARKET_SNAPSHOT after reconnect.

## Plane Boundaries (mandatory)

| Plane | Location | Authority | Must NOT become |
|-------|----------|-----------|-----------------|
| Go Realtime | realtime/ | Market data, features, signals, gates | Synchronous billing |
| NestJS Control | control/ | IAM, subscriptions, billing, licensing | Tick-to-signal hot path |
| Next.js Frontend | frontend/ | UI rendering | Risk/entitlement authority |
| Python Research | research/ | Backtesting, calibration, ML | Live tick dependency |
| Windows/MQL Edge | mql/ | Order execution | Primary intelligence |

## Current Status (16 September 2026)

| Check | Status |
|-------|:------:|
| Go tests (39/39 packages, cgo-enabled run) | PASS |
| Control tests (NestJS 12, Jest 30, NODE_OPTIONS=--experimental-vm-modules) | PASS |
| Frontend tests + e2e | PASS |
| Python tests (uv-managed) | PASS |
| CI — 6/6 jobs green (+ docs-quality advisory job) | PASS |
| All 17 compose services healthy | PASS |
| 23 risk gates active (ordered, fail-closed) | PASS |
| SL enforcement server-side | ACTIVE |
| Broker symbol validation (P0-001) | ACTIVE |
| Price precision rounding (P1-001) | ACTIVE |
| Math parity (MAPE < 0.0001) | PASS |
| Migrations (108 files, numbered 001–148, unique prefixes) | PASS |
| Secrets out of git (env-file injection) | PASS |
| MT5 clients connected (EA attach + license) | Operator action |
| Demo fill test (one signal round-trip) | Operator action |
| Backup/restore drill | Operator action (R2 snapshot verified 2026-09-12) |
| Live automated trading arming | Authorized by operator (`LIVE_TRADING_AUTHORIZED=true`); fail-closed on verified broker equity feed |

> **Deployment is Docker-First.** All services run via `docker compose --env-file infra/env/.env`.
> Systemd units in `infra/systemd/` are DISABLED. Live automated trading is fail-closed: signals
> run in paper/sandbox/advisory mode only until an operator authorizes arming AND a verified broker
> equity/order feed exists. No profitability, accuracy, or hit-rate claims are made without evidence.

## Go Realtime Plane

Located in `realtime/`. Key packages:

- `internal/marketdata` — agent provider (Option B ingest), tick/candle aggregation, COT (FMP + CFTC fallback), DXY, TwelveData multi-asset (batched + daily credit budget), FRED real yield
- `internal/features` — feature registry (~90 feature fields: indicators, structure, FVG/liquidity, regime, VWAP, Fibonacci, session/ORB, pivots, readiness map)
- `internal/strategy` — 7 strategy engines, evidence scoring, confluence, geometry
- `internal/gates` — 23 hard risk gates (ordered, fail-closed)
- `internal/signal` — master decision engine, cooldown, duplicate prevention
- `internal/gateway` — HTTP + dashboard WS handlers (browser relay; EA traffic is HTTPS ingest + edge-poll)
- `internal/crossmarket` — DXY, BTC, Oil macro module
- `internal/ml` — ONNX model inference (advisory)
- `internal/sentiment` — Ollama sentiment analysis (advisory)
- `internal/ptb` — Professional Trader Brain intelligence layer
- `pkg/health`, `pkg/news`, `pkg/macro`, `pkg/mt5` — public utilities

## Documentation

- [SCOPE_OF_WORK.md](realtime/SCOPE_OF_WORK.md) — Full project scope and specifications
- [CHANGELOG.md](realtime/CHANGELOG.md) — Version history v1.0–v1.29.x
- [docs.md](docs.md) — Full pipeline, indicator maths, all 7 strategies in plain words (from live engine source)
- [indicator.md](indicator.md) — Indicator & signal-generation reference (candle reading, formulas, evidence scoring, gates)
- [project-brief.md](project-brief.md) — Full-project reference: planes, services, lifecycle, delivery internals, maths layer, DB schema + ERD
- [docs/](docs/) — Architecture, strategy playbooks, indicators, gates, API, database
- [Docker Deployment Guide](docs/operations/DOCKER_DEPLOYMENT.md) — Step-by-step Docker Compose
- [Hetzner Deployment Runbook](docs/operations/HETZNER_DEPLOYMENT.md) — VPS provisioning, R2-backed
- [Server Migration Runbook](docs/operations/SERVER_MIGRATION.md) — VPS→VPS move (R2 snapshot, DNS cutover)
- [Admin Guide](docs/guides/ADMIN_GUIDE.md) — System administration
- [User Guide](docs/guides/USER_GUIDE.md) — Dashboard, strategies, MT4/MT5 setup
- [EA Client Guide](docs/guides/EA_CLIENT_GUIDE.md) — Option B EA-direct transport: install, licensing, troubleshooting
- [Runbooks](docs/runbooks/) — MT connectivity, edge-poll 429, signal-delivery verification, Discord/Telegram/WhatsApp desks, mail deliverability

## Canonical Project Files

- [AGENTS.md](AGENTS.md) — Authoritative agent operational instructions (read first).
- [MANIFEST.md](MANIFEST.md) — Project scope, structure, service inventory.
- [realtime/SCOPE_OF_WORK.md](realtime/SCOPE_OF_WORK.md) — Full statement of work.

## Build & Test

```bash
# All services
make build && make test && make lint

# Individual planes
make go-build          # Go realtime engine
make go-test           # Go tests (39 packages — host has no Go; container golang:1.25 + gcc for cgo/mlengine)
make control-build     # NestJS control plane
make frontend-build    # Next.js frontend
make research-test     # Python tests (uv-managed)

# Docker (ALL commands MUST use --env-file infra/env/.env)
docker compose --env-file infra/env/.env up -d --build
docker compose --env-file infra/env/.env ps

# Deploy gotcha: pat-control and pat-control-b build to SEPARATE images.
# After any control change: docker compose build control control-b frontend && docker compose up -d control control-b frontend
```

## Production Safety

Without explicit operator authorization, do NOT:
- Enable live automated trading
- Place or close live broker orders/positions
- Mutate real subscriptions, commissions, wallets or payouts
- Run destructive production migrations
- Export secrets or rotate signing keys

NO-TRADE is a valid first-class result. ML/AI components are advisory only and cannot override deterministic gates.

## Documentation & Security

- **Documentation index:** [`docs/INDEX.md`](docs/INDEX.md) — canonical navigation; `docs/README.md` holds the full directory map. Architecture: `docs/architecture/ARCHITECTURE.md`. Runbooks live in `docs/runbooks/`; audits in `docs/audits/`.
- **Secrets:** all credentials live in `infra/env/*.env` (gitignored). **Never** commit secrets to compose, docs, or code — for rotation procedures see `docs/operations/SECRET_ROTATION.md` (the live JWT_SECRET was historically leaked in tracked files and requires rotation — treated as compromised).
- **Changelog:** `realtime/CHANGELOG.md`.

## License

MIT License — see [LICENSE](LICENSE)