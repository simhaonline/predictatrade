# Predict-A-Trade XAUUSD

Predict-A-Trade is a multi-plane XAUUSD trading and subscription platform. The repository contains the Go real-time trading plane, NestJS control plane, Next.js presentation plane, Python research plane, Windows/MetaTrader edge components, PostgreSQL/TimescaleDB persistence, Valkey cache, and Docker deployment configuration.

This README reflects the repository audit performed on **22 August 2026**. It records what is present and wired in this checkout; it does not claim production readiness where provider, broker, security, or acceptance evidence is missing.

## Current status

Overall status: **PARTIAL / CONDITIONAL — not a full `prompt.md` acceptance**.

Verified in the current workspace:

- Go realtime tests pass with `go test ./...`.
- Frontend tests pass: 16 suites, 84 tests.
- Frontend TypeScript check passes.
- Frontend production build passes and generates 48 routes.
- Frontend ESLint has 0 errors and 14 warnings.
- Docker `pat-postgres`, `pat-valkey`, `pat-realtime`, `pat-control`, and `pat-frontend` are healthy at audit time.
- Subscription migrations 024 and 025 are present and additive/effective-dated.
- No production payment, subscription, commission, payout, or live-trading mutation was performed during the audit.

Full dashboard-v3 work remains blocked by authenticated user-scoped WebSocket authorization, complete API entitlement filtering, payment-provider activation, signal distribution/quota consumption, admin entitlement controls, and required persona/security acceptance tests. See [pending-work.md](pending-work.md).

## Architecture and runtime wiring

```text
MT4/MT5 terminal
        │
        ▼
Windows Agent / Master Node ── WebSocket ──► Go Real-Time Engine :13081
                                                   │
                         ┌─────────────────────────┼──────────────────────┐
                         ▼                         ▼                      ▼
                 Market ingestion            Features/PTB             Strategies
                 candles/ticks               indicators/structure      four engines
                         │                         │                      │
                         └─────────────────────────┴──────────┬───────────┘
                                                               ▼
                                                   deterministic signal engine
                                                   + hard risk gates
                                                               │
                                              TimescaleDB + Valkey + WebSocket
                                                               │
                 ┌─────────────────────────────────────────────┴──────────┐
                 ▼                                                        ▼
       Next.js presentation :13082                              Windows/MT delivery
       user/admin dashboards

       NestJS control plane :13080 ── IAM, billing, plans, entitlements,
                                     licensing, devices, referrals,
                                     commissions, payouts, audit, backtests

       Python research plane ── datasets, backtests, calibration, ML/RL research
```

### Plane boundaries

| Plane | Location | Responsibility | Must not become |
|---|---|---|---|
| Real-time trading | `realtime/` | Market data, features, strategies, signals, hard gates, delivery and reconciliation | A synchronous billing/referral dependency |
| SaaS/control | `control/` | IAM, MFA, RBAC, subscriptions, billing, entitlements, licensing, devices, referrals, commissions, payouts, admin | The tick-to-signal hot path |
| Presentation | `frontend/` | Public site, user dashboard, admin console, charts and commercial UI | The authority for risk, entitlement, finance or probability |
| Research | `research/` and `scripts/` | Historical data, backtesting, validation, calibration, ML/RL research | A mandatory dependency for every live tick |
| Windows/MT edge | `windows-agent/`, `mql/` | Broker/terminal adapter, heartbeat, signed signal handling and execution guards | Primary intelligence or private server credentials |

## Docker services

All application services are defined in [docker-compose.yml](docker-compose.yml). The repository’s intended runtime is Docker Compose; systemd files under `infra/systemd/` are compatibility/deployment artifacts and are not the active compose runtime.

| Service | Container | Port | Role |
|---|---|---:|---|
| `postgres` | `pat-postgres` | 5432 | PostgreSQL 17 / TimescaleDB |
| `valkey` | `pat-valkey` | 6379 | Cache, counters and hot state |
| `realtime` | `pat-realtime` | 13081 | Go HTTP/WebSocket realtime engine |
| `control` | `pat-control` | 13080 | NestJS API/control plane |
| `frontend` | `pat-frontend` | 13082 | Next.js presentation plane |
| `status` | `pat-status` | 13083 | Status page service |
| `nginx` | `pat-nginx` | 80/443 | Reverse proxy, TLS and WebSocket routing |
| `prometheus` | `pat-prometheus` | 9090 internal | Metrics collection |
| `grafana` | `pat-grafana` | 3001→3000 | Dashboards |
| `ntfy` | `pat-ntfy` | 8091→80 | Optional self-hosted notifications |

Useful commands:

```bash
docker compose up -d
docker compose up -d --build
docker compose ps
docker compose logs -f realtime
docker compose restart frontend
docker compose build control && docker compose up -d control
```

The compose healthchecks currently use PostgreSQL readiness, Valkey ping, realtime `/health`, control `/api/v1/health`, and frontend `/` checks.

## Go realtime plane

The realtime binary is built from `realtime/cmd/realtime-engine`. Important packages include:

- `internal/marketdata`: agent provider, tick/candle aggregation, historical bootstrap, persistence, COT and DXY providers.
- `internal/features`: indicators, rolling values, VWAP, Fibonacci, FVG, liquidity, structure, regime, session and multi-timeframe state.
- `internal/strategy`: four independent products, scoring, geometry, confluence, thresholds, capability checks and no-trade behavior.
- `internal/gates`: data quality, session/news, spread/slippage/cost, exposure, margin, R:R, entitlement, license, execution permission and capital protection.
- `internal/signal`: signal lifecycle, cooldown, duplicate prevention and delivery helpers.
- `internal/gateway`: HTTP, dashboard WebSocket and Windows Agent WebSocket handlers.
- `internal/cache`: Valkey candle and hot-state access.
- `internal/ptb`: Professional Trader Brain shared analysis; PTB modules are not allowed to bypass hard gates.
- `internal/reconciliation`, `internal/recovery`, `internal/oco`, `internal/hedging`: operational and trade-management support.
- `pkg/health`, `pkg/news`, `pkg/macro`, `pkg/mlengine`, `pkg/ollama`, and `pkg/notifications`: health, provider and optional intelligence integrations.

The four strategy identifiers are:

```text
STANDARD_SCALPING
ULTRA_SCALPING
STANDARD_SWING
TREND_SWING
```

`NO-TRADE`, `BLOCKED`, `WAIT`, and degraded/unknown states are valid outcomes. A score, UI request, or frequency target must not force a trade.

### Data truth

- Production signal generation is intended to require a live Master Node/MT5 Agent source.
- Broker tick volume is a proxy, not centralized XAUUSD exchange volume.
- DOM, CVD, aggressor-side, footprint, iceberg and global resting-liquidity claims require an explicitly available provider capability.
- Missing required data must degrade quality or produce `NO-TRADE`; synthetic/replay data must remain visibly non-production.
- ML/RL/LLM components are optional/research or asynchronous presentation capabilities and cannot override deterministic gates.

### Realtime endpoints and events

The Go gateway exposes HTTP health/snapshot routes and WebSocket paths including `/ws`, `/ws/v1`, `/ws/v1/agent`, and `/ws/agent`. Event envelopes contain event ID, stream ID, sequence, schema version, timestamp, type, priority, payload, and optional correlation ID.

The current repository has strategy filtering code in the WebSocket broadcaster, but authenticated browser identity binding and complete user entitlement hydration/refresh remain pending. Do not treat the current WebSocket as complete user-level authorization.

## NestJS control plane

The NestJS application is under `control/src/` and is assembled by `control/src/app.module.ts`. Current modules include `auth`, `users`, `admin`, `health`, `plans`, `subscriptions`, `billing`, `licensing`, `device-auth`, `referrals`, `commissions`, `payouts`, `audit`, `operations`, `backtest`, and `guest-preview`.

Representative API groups are:

```text
/api/v1/auth/*
/api/v1/users/*
/api/v1/plans
/api/v1/subscriptions
/api/v1/subscriptions/entitlements
/api/v1/billing/invoices
/api/v1/billing/webhook
/api/v1/referrals/*
/api/v1/commissions/*
/api/v1/payouts/*
/api/v1/licensing/*
/api/v1/devices/*
/api/v1/admin/*
/api/v1/backtest/*
/api/v1/health
```

JWT authentication and admin guards exist. The subscription policy validates known strategy IDs, plan strategy limits, Free restrictions, and Standard restrictions. The entitlement endpoint currently returns the effective subscription row and plan entitlement map; a complete dashboard capability manifest and per-resource authorization layer are still pending.

The billing webhook service currently acknowledges/logs received events but does not constitute a verified provider adapter. Paid activation, signature verification, refunds, chargebacks, and lifecycle propagation therefore remain unverified.

## Next.js presentation plane

The Next.js app is under `frontend/src/app`.

User routes include:

```text
/dashboard/live
/dashboard/signals
/dashboard/strategies
/dashboard/backtest
/dashboard/trading-reports
/dashboard/billing
/dashboard/referrals
/dashboard/mt4-mt5-client
/dashboard/settings
```

Admin routes include dashboard, users, subscriptions, billing, referrals, commissions, payouts, licenses, devices, activations, operations, signals, indicator monitoring, strategies, backtesting, reports, health, logs, and settings pages.

Shared layout and design tokens are in `frontend/src/components/layout` and `frontend/src/styles/globals.css`. The current dashboard palette follows `predictatrade-live-patched.html`; API and backend code are not responsible for visual token changes.

The frontend is a presentation layer. It must render server-authoritative plan, entitlement, signal, risk, execution, commission and payout state. Full capability-driven navigation, Free-only signal allocation UI, response-field redaction, and all direct-URL/API security tests remain pending; see [pending-work.md](pending-work.md).

## Subscription, referral and financial model

The active commercial packages are:

| Code | Monthly | Annual | New sales |
|---|---:|---:|---|
| FREE | $0 | not configured | Enabled |
| STANDARD | $99 | $990 | Enabled |
| PRO | $299 | $2,990 | Enabled |
| ELITE | $699 | $6,990 | Enabled |
| BASIC | historical | historical | Hidden/legacy |

The current v3 referral configuration is effective-dated:

- Standard: 10% / 3% / 1% at levels 1 / 2 / 3.
- Pro: 15% / 4% / 2% at levels 1 / 2 / 3.
- Elite: 18% / 5% / 2% at levels 1 / 2 / 3.
- First purchase: 100% multiplier through level 3.
- Second purchase: 75% multiplier at level 1.
- Recurring purchase: 50% multiplier through level 3.

Money is represented with PostgreSQL `DECIMAL(18,8)` and commission records are ledger-oriented. Existing historical rows are preserved. Calculation policy exists and is unit-tested, but provider-backed event activation and complete production reconciliation remain pending.

## Database structure

Database files are in `database/migrations/`. There are 25 migration files in this checkout, with historical duplicate numeric prefixes retained for compatibility. `scripts/migrate.sh` is the canonical forward runner and records status in `audit.migration_history`; rollback is explicitly not implemented by the runner and requires PITR or a forward correction.

### Schemas

| Schema | Main responsibility |
|---|---|
| `iam` | Organizations, users, roles, memberships, sessions and authentication state |
| `control` | Plans, plan entitlements, strategy configuration, feature/config state |
| `billing` | Subscriptions, invoices, invoice items, payments, payment events, refunds and credits |
| `licensing` | Licenses, devices, MT accounts and activation state |
| `referral` | Affiliate profiles, codes, attribution, five-level relationships, rules, immutable commission ledger, wallets, payout methods and payouts |
| `trading` | Signals, candidates, rejections, delivery, market state, positions, risk, PTB, strategy and audit history |
| `market` | Market data, candles, provider/provenance and capability state |
| `research` | Research/backtest and model-related durable data |
| `audit` | Audit events, migration history and operational traceability |
| `support` | Support/complaint-related records where enabled by migrations |

### Important migrations

| Migration | Purpose |
|---|---|
| 001–008 | Schemas/roles, IAM, plans/billing/licensing, referral/commission/payout, trading tables, session/token, device activation |
| 009–011 | Signal delivery/replay, completion audit, COT/capability WAL |
| 012–014 | PTB intelligence, synthesis/performance, advanced risk/adaptation/hedging/sentiment |
| 015–023 | Backtesting, stale-operation fixes, reconciliation, regime/slippage/SLTP/trade-management, news/OCO/notifications, guest preview |
| 024 | v3 plan metadata, effective entitlement versions, strategy preferences, commercial events, signal-delivery ledger, commission snapshots, v3 rules and feature flags |
| 025 | Subscription billing interval persistence and index |

Migration 024 creates the durable `trading.signal_delivery_ledger`, but a complete production distribution writer and concurrent Free-quota consumption path are not yet proven. Valkey is cache/optimization state, not the durable financial or quota authority.

## Research plane

The Python package under `research/src/patresearch` provides datasets, reference math, vectorized indicators, calibration, backtesting, ML training and RL training. Tests are under `research/tests`. Python research artifacts must not silently become mandatory for the live Go decision path.

Typical research commands:

```bash
cd research
python -m pytest tests/ -v --tb=short
python -m patresearch.backtesting.cli run --strategy STANDARD_SCALPING --seed 42
```

## Windows Agent and MetaTrader edge

The Go Windows Agent is under `windows-agent/`; MQL adapters are under `mql/mt4` and `mql/mt5`. The edge responsibilities include installation/update support, device identity, heartbeats, IPC/pipe support, broker/terminal state, signed signal handling, and execution guards. Server-side prediction, private signing keys, and primary entitlement authority do not belong in EAs.

Build/validation entry points:

```bash
./scripts/build-windows-agent.sh --bump
cd windows-agent && go test -race -count=1 ./...
```

Live broker/terminal qualification still requires controlled Windows/MT4/MT5 runtime evidence and must not be inferred from compilation alone.

## Observability and operations

Prometheus and Grafana configuration is under `infra/prometheus` and `infra/grafana`. Go metrics include realtime health, WebSocket connections/messages, PTB analysis, gate state, signal flow and provider state. Structured logging is used by the Go engine; NestJS exposes health and metrics support through its modules.

Operational and validation documentation is under:

- `docs/reports/` — audit, production, GO/NO-GO and traceability reports.
- `docs/database/` — schema, migration, traceability and performance documentation.
- `docs/operations/` — operational procedures.
- `docs/guides/` — user/admin/MT setup guides.
- `docs/strategy/` — strategy and capability documentation.
- `docs/SUBSCRIPTION_*` — current subscription/referral design and evidence.
- [pending-work.md](pending-work.md) — open work required for complete `prompt.md` acceptance.

## Canonical development commands

```bash
# Infrastructure
make infra-up
make infra-down
docker compose ps

# Database
make db-migrate
make db-seed
make db-test

# Go
make go-build
make go-test
make go-lint

# Control plane
make control-build
make control-test
make control-lint

# Frontend
make frontend-build
make frontend-test
make frontend-lint

# Research and edge
make research-test
make agent-build
make agent-test
```

Use `npm run lint` from `frontend/` for the frontend lint check because the production frontend image intentionally contains the compiled runtime rather than the source/configuration tree.

## Security and production boundary

Do not enable live automated trading, mutate real subscriptions/commissions/wallets/payouts, run destructive production migrations, export secrets, rotate signing keys, or publish unsupported performance claims without explicit operator authorization and the applicable release evidence.

Local compose defaults include development credentials and must not be used as production secrets. Production requires injected secrets, TLS, provider credentials, broker/agent qualification, backup/restore evidence, financial reconciliation, entitlement/security tests, and rollback readiness.

## Primary project controls

- [AGENTS.md](AGENTS.md) — repository operating contract.
- [prompt.md](prompt.md) — user-dashboard v3 entitlement/subscription/access-control requirements.
- [docs/Predict-A-Trade_FINAL_SCOPE_OF_WORK.md](docs/Predict-A-Trade_FINAL_SCOPE_OF_WORK.md) — canonical SOW.
- [pending-work.md](pending-work.md) — current verified gaps and acceptance work.
- [plans-summary.md](plans-summary.md) — current plan and referral summary.
- [MANIFEST.md](MANIFEST.md) — repository manifest.
