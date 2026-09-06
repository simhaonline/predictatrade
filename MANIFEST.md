# Predict-A-Trade Project Manifest

## Version: v1.29.0 — Docs & Repo Hygiene (2026-09-06)

> Version lineage: v1.18.0 macro-audit remediation (2026-08-30) → v1.19.0 Option B
> EA-direct transport (2026-09-01) → v1.24/1.25 tier-geometry + observability →
> v1.26 STANDARD_SCALPING rebuild → v1.27 account-type detection → v1.28 EA
> capital guards → v1.29.0 docs sweep (2026-09-05) → v1.29.1 EQFE/IMLR display
> completion + repo cleanup (2026-09-06). Live structured-log version pin:
> realtime v1.24.2. Changelog: `realtime/CHANGELOG.md`.

## Repository Structure

```
/srv/predictatrade/xauusd/
├── AGENTS.md                  # Agent operational instructions (canonical)
├── MANIFEST.md                # This file
├── README.md                  # System overview
├── docs.md                    # Full pipeline + maths + all 7 strategies in plain words (from live engine source)
├── Makefile                   # Canonical build/lint/test commands
├── docker-compose.yml         # All services (16 containers)
├── .gitleaks.toml             # Secret-scanning config
├── .gitignore                 # Comprehensive exclusions
│
├── realtime/                  # Go — Real-Time Trading Plane (port 13081)
│   ├── cmd/realtime-engine/   # Main entrypoint
│   ├── cmd/backtest-engine/   # Backtest engine entrypoint
│   ├── cmd/live-terminal/     # Public live-terminal service entrypoint
│   ├── cmd/audit/             # Audit utility
│   ├── cmd/backfill/          # Historical candle backfill
│   ├── internal/              # Internal packages (34 modules)
│   │   ├── adaptation/        # Loss recovery & adaptation
│   │   ├── agent/             # Edge-transport agent session state
│   │   ├── astro/             # ATEN astro-confluence engine
│   │   ├── backtest/          # Backtesting engine (production-parity)
│   │   ├── breakout/          # Session ORB breakout
│   │   ├── cache/             # Valkey cache + candle cache
│   │   ├── calibration/       # Probability calibration (VALIDATED-gated)
│   │   ├── capitaltier/       # Capital-tiered signal engine (v1.23)
│   │   ├── config/            # Configuration loading
│   │   ├── crossmarket/       # Cross-market driver synthesis
│   │   ├── devilliquidity/    # Liquidity/stop-hunt analysis
│   │   ├── engstatus/         # Engine status surface
│   │   ├── features/          # 42-feature indicator engine
│   │   ├── gates/             # 16 hard risk gates
│   │   ├── gateway/           # HTTP + WebSocket server (pprof enabled)
│   │   ├── hedging/           # Hedging engine
│   │   ├── igs/               # Institutional Gold Intelligence layer
│   │   ├── livepreview/       # Live preview funnel
│   │   ├── maintenance/       # Daily maintenance scheduler
│   │   ├── marketdata/        # Market data providers + persistence
│   │   ├── ml/                # ML inference (ONNX)
│   │   ├── observability/     # OpenTelemetry + Prometheus
│   │   ├── oco/               # OCO order management
│   │   ├── ptb/               # PTB synthesis engine
│   │   ├── reconciliation/    # EXECUTION_ACK → fill reconciliation
│   │   ├── recovery/          # State recovery
│   │   ├── replay/            # Signal replay & idempotency
│   │   ├── risk/              # Risk event persistence
│   │   ├── rl/                # Reinforcement learning
│   │   ├── sentiment/         # Sentiment analysis (Ollama)
│   │   ├── signal/            # Signal generation
│   │   ├── strategy/          # 7 strategies + geometry
│   │   └── types/             # Shared types
│   ├── pkg/                   # Public packages
│   │   ├── health/            # Health manager
│   │   ├── macro/             # COT + DXY providers
│   │   ├── math/              # Math parity (Wilder smoothing)
│   │   ├── mlengine/          # ML engine + models/ watcher
│   │   ├── mt5/               # MT5 protocol
│   │   ├── news/              # Economic calendar provider + risk engine
│   │   ├── notifications/     # External notification adapters
│   │   ├── ollama/            # Ollama client
│   │   └── strategy/          # Strategy definitions
│   ├── configs/               # Strategy & gate configs
│   ├── calibration/           # Live calibrator outputs (JSON, mounted into containers)
│   ├── migrations/            # Go-level migrations (DB schema lives in database/migrations)
│   └── bin/                   # Compiled binaries (gitignored; backtest-engine mounted into compose)
│
├── control/                   # NestJS — SaaS/Control Plane (port 13080)
│   └── src/modules/           # 16 modules: auth, users, billing, licensing,
│                              # device-auth, backtest, commissions, payouts,
│                              # referrals, operations, admin, audit, …
│
├── frontend/                  # Next.js — Presentation Plane (port 13082)
│   ├── src/app/(admin)/       # Admin pages (25 routes)
│   ├── src/app/(user)/        # User dashboard pages (19 routes)
│   ├── src/components/        # React components
│   ├── src/lib/               # API hooks & utilities (incl. strategy-labels.ts)
│   └── public/downloads/      # Public EA sources + binaries (mirror of mql/)
│
├── research/                  # Python — Intelligence/Research Plane
│   ├── src/patresearch/       # Library (indicators, backtesting, ML, ai_research)
│   ├── tests/                 # 154 tests (152 pass, 2 skip) — `uv run pytest`
│   └── scripts/               # Research scripts
│
├── mql/                       # MQL4/MQL5 Expert Advisors (Option B — EA-direct cloud transport)
│   ├── mt4/                   # PredictATrade_MT4.mq4 + MasterNode
│   ├── mt5/                   # PredictATrade_MT5.mq5 + MasterNode
│   └── compiled_executable/   # Compiled .ex4/.ex5 mirrors
│
├── windows-agent/             # Windows Agent (legacy installers, v1.2.x — REMOVED from runtime in v1.19.0 Option B)
│
├── database/                  # SQL Migrations
│   └── migrations/            # 99 files (unique prefixes, numbered to 138) + MIGRATION_ORDER.md
│
├── infra/                     # Infrastructure
│   ├── env/                   # Environment files (gitignored; secrets only here)
│   ├── nginx/                 # Nginx configs
│   ├── systemd/               # Systemd service files — DISABLED (docker-first; do not use)
│   ├── ntfy/                  # ntfy config
│   ├── prometheus/            # Prometheus config
│   └── grafana/               # Grafana dashboards
│
├── scripts/                   # Operations scripts (migrate.sh, full_audit.sh, security-scan.sh, …)
├── services/backtest-service/ # Python backtest API (container pat-backtest, 127.0.0.1:8088)
├── status/                    # Status page (Node.js, port 13083)
├── live-dashboard/            # Live dashboard PWA assets (served via nginx /var/www/pat-live/)
├── live-terminal/             # Live terminal service Dockerfile (pat-live-terminal, 13090)
├── mail-relay/                # Send-only SMTP relay (Go — pat-mail-relay)
├── marketing/                 # Marketing site + content (static, nginx-served)
├── models/                    # ML models (ONNX + metadata; watcher.go live-reload)
├── legal/                     # Terms, privacy, cookie policy, trust center
├── nginx/                     # Nginx site configs (platform., downloads., docs. vhosts)
├── artifacts/                 # Evidence artifacts (go_live_evidence/)
│
└── docs/                      # Documentation (see docs/README.md + docs/INDEX.md)
    ├── architecture/          # ARCHITECTURE, FLOW_DIAGRAMS, IGS design
    ├── strategy/              # Playbooks, indicators, risk gates, capital tiers
    ├── api/                   # API_REFERENCE + openapi.json
    ├── database/              # DATABASE_ARCHITECTURE + DB_ERD
    ├── operations/            # Deployment, backup/restore, incident, DR
    ├── runbooks/              # mt-connectivity-502, edge-poll-429
    ├── guides/                # ADMIN_GUIDE, USER_GUIDE, EA_CLIENT_GUIDE
    ├── reports/               # Whitepaper, thesis, audits, remediation
    ├── nginx/                 # docs.predictatrade.com serving config
    └── MT4_MT5_CLIENT_TESTING.md # EA client test procedure
```

## Key Numbers

| Metric | Value |
|--------|-------|
| Go Test Packages | 39 pass, 0 fail (container `golang:1.25`, host has no Go toolchain) |
| Python Tests | 154 (152 pass, 2 skip) — `cd research && uv run pytest` |
| Frontend Tests | 84 pass + 18 e2e; `tsc --noEmit` clean |
| Control (NestJS) Tests | 14 suites / 174 pass; `tsc --noEmit` clean |
| DB Migrations | 99 files (unique prefixes, numbered to 138), all applied to live DB |
| ML Features | 42 (35 live, 7 warming) |
| ML Models | 5 (bootstrap placeholders — honest `bootstrap-v1.0.0`; NOT production-trained) |
| Strategies | 7 (Standard Scalping, Ultra Scalping, Standard Swing, Trend Swing, EQFE, ATEN); IMLR = 7th, ADVISORY-only |
| Strategy display naming | Internal IDs `MARNIE_FIB`/`ARCANIST` never user-facing — displayed as **EQFE**/**IMLR** everywhere |
| Risk Gates | 16 (per-strategy/timeframe isolated, fail-closed) |
| Indicators Live | 35/42 |
| API Latency | < 3ms |

## Service Inventory (Docker-First — no systemd)

All services run as Docker containers via `docker compose --env-file infra/env/.env`.
Systemd units (in `infra/systemd/`) are DISABLED and must not be used.

| Service | Container | Port | Status |
|---------|-----------|------|--------|
| Real-Time Engine | pat-realtime | 13081 | ✅ Active (paper/sandbox/advisory) |
| Control Plane (HA pair) | pat-control / pat-control-b | 13080 | ✅ Active (dual-control + nginx failover) |
| Frontend | pat-frontend | 13082 | ✅ Active |
| Status Page | pat-status | 13083 | ✅ Active |
| Backtest Service | pat-backtest | 8088 (127.0.0.1 only) | ✅ Active |
| Live Terminal | pat-live-terminal | 13090 | ✅ Active |
| Mail Relay | pat-mail-relay | 25/587 | ✅ Active (send-only, spool+retry) |
| PostgreSQL 17 + TimescaleDB | pat-postgres | 5432 | ✅ Active |
| Valkey | pat-valkey | 6379 | ✅ Active |
| Nginx | pat-nginx | 80/443 | ✅ Active |
| Prometheus | pat-prometheus | 9090 | ✅ Active |
| Grafana | pat-grafana | 3001 | ✅ Active |
| ntfy | pat-ntfy | 8091 | ✅ Active |
| NATS (optional bus) | pat-nats | 4222 | Optional (ingest decoupling seam) |
| Backup Sync | pat-backup-sync | — | Hetzner S3 WAL + pg_dump off-host sync |
| Ollama | host/container | 11434 | ✅ Active (sentiment; NOT_AI_VERIFIED provenance) |

## Live Data Status

| Data | Source | Status |
|------|--------|--------|
| XAUUSD Price | Master Node EA via Option B HTTPS ingest (`PROVIDER_MODE=agent`, authoritative); Twelve Data macro cross-check only | ✅ Live when a Master Node streams `MARKET_SNAPSHOT`; honest `NO_DATA` otherwise (never faked) |
| COT Data | FMP API | ✅ Available |
| DXY Data | Twelve Data API | ✅ Available |
| Candle Cache | Valkey + PostgreSQL (Timescale hypertables) | ✅ Active |
| ML Inference | ONNX Runtime + `models/` watcher | ✅ Active (bootstrap placeholders — NOT production-trained) |
| Sentiment | Ollama (local LLM) | ✅ Connected (provenance: NOT_AI_VERIFIED) |

## Canonical Commands (Docker-First)

```bash
# Build (host has no Go toolchain — use the container)
cd realtime && go build -o bin/realtime-engine ./cmd/realtime-engine/   # requires golang:1.25 container
cd frontend && npx next build
cd control && npm run build

# Test
docker run --rm -v $PWD/realtime:/app -w /app golang:1.25 go test ./...
cd research && uv run pytest
cd frontend && npx jest --passWithNoTests && npx tsc -p tsconfig.json --noEmit
cd control && npm test   # NODE_OPTIONS=--experimental-vm-modules (in npm script)

# Lint
docker run --rm -v $PWD/realtime:/app -w /app golang:1.25 go vet ./...
cd frontend && npx tsc --noEmit

# Audit
bash scripts/full_audit.sh

# Deploy / operate (ALL commands MUST use --env-file infra/env/.env)
docker compose --env-file infra/env/.env build <service>
docker compose --env-file infra/env/.env up -d <service>
docker compose --env-file infra/env/.env logs -f <service>
docker compose --env-file infra/env/.env ps

# Migrate (NEVER auto-applied; never rewrite applied history)
./scripts/migrate.sh up
```

## Repo Hygiene (2026-09-06)

Removed from the repo root as superseded scratch: `check.md` (EA task-input
scratch, all tasks completed), `summary.md` (stale 2026-09-01 snapshot —
superseded by this manifest + docs/), `AGENT.md` (dead Codex-compat pointer),
`SKILLS.md` (stale v1.0.0 skill index), `TESTING_LICENSE.md` (dev test license
already seeded), `error.log` (stale MetaEditor output — the `PAT_ATD_histComm`
errors were fixed in commit `545dbb8`). Root credential files
`database_url.txt` / `jwt_secret.txt` remain (gitignored, consumed by scripts).
Agent task-instructions workflow: drop files as `read.md`/`check.md` in the
root, they are consumed and deleted in the same session.

## Production Status (honest)

- **GO** for paper / sandbox / advisory signal operation.
- **LIVE TRADING ARMING AUTHORIZED BY OPERATOR (2026-08-30)**: `LIVE_TRADING_AUTHORIZED=true`
  in `infra/env/realtime.env` (gitignored deploy config). Capital-protection gates remain
  **fail-closed**: signals stay ADVISORY unless a verified broker equity/order feed exists.
- No profitability, accuracy, hit-rate, or live-trading-capability claims are made without
  evidence. Demo/replay/sandbox data is labeled and cannot mutate live trading or real finance.