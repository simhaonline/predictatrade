# Predict-A-Trade Project Manifest

## Version: v1.18.0 — Macro-Audit Remediation (2026-08-30)

## Repository Structure

```
/srv/predictatrade/xauusd/
├── AGENTS.md                  # Codex repository instructions (canonical)
├── AGENT.md                   # Compatibility pointer
├── SKILLS.md                  # Skills index
├── MANIFEST.md                # This file
├── README.md                  # System overview
├── Makefile                   # Canonical build/lint/test commands
├── run.sh                     # Startup orchestration
├── docker-compose.yml         # Local Docker infra (Postgres, Valkey, Prometheus, Grafana)
├── .gitleaks.toml              # Secret-scanning config with dev-test allowlists
├── .gitignore                 # Comprehensive exclusions
│
├── realtime/                  # Go — Real-Time Trading Plane (port 13081)
│   ├── cmd/realtime-engine/   # Main entrypoint
│   ├── cmd/backtest-engine/   # Backtesting engine entrypoint
│   ├── cmd/audit/             # Audit utility
│   ├── internal/              # Internal packages (20+ modules)
│   │   ├── adaptation/        # Loss recovery & adaptation
│   │   ├── backtest/          # Backtesting engine
│   │   ├── cache/             # Valkey cache + candle cache
│   │   ├── calibration/       # Probability calibration
│   │   ├── config/            # Configuration loading
│   │   ├── features/          # 42-feature indicator engine
│   │   ├── gates/             # Hard risk gates (12+ gates)
│   │   ├── gateway/           # HTTP + WebSocket server (pprof enabled)
│   │   ├── hedging/           # Hedging engine
│   │   ├── maintenance/       # Daily maintenance scheduler
│   │   ├── marketdata/        # Market data providers + persistence
│   │   ├── ml/                # ML inference (ONNX)
│   │   ├── observability/     # OpenTelemetry + Prometheus
│   │   ├── ptb/               # PTB synthesis engine
│   │   ├── reconciliation/    # Trade reconciliation
│   │   ├── recovery/          # State recovery
│   │   ├── replay/            # Signal replay & idempotency
│   │   ├── rl/                # Reinforcement learning
│   │   ├── sentiment/         # Sentiment analysis (Ollama)
│   │   ├── signal/            # Signal generation
│   │   ├── strategy/          # 4 strategies + geometry
│   │   └── types/             # Shared types
│   ├── pkg/                   # Public packages
│   │   ├── health/            # Health manager
│   │   ├── news/              # Economic calendar provider + risk engine
│   │   ├── notifications/     # External notification adapters
│   │   ├── macro/             # COT + DXY providers
│   │   ├── math/              # Math parity (Wilder smoothing)
│   │   ├── mlengine/          # ML engine interface
│   │   ├── mt5/               # MT5 protocol
│   │   ├── ollama/            # Ollama client
│   │   └── strategy/          # Strategy definitions
│   ├── configs/               # Strategy & gate configs
│   ├── migrations/            # Go-level migrations (DB schema lives in database/migrations)
│   ├── testdata/              # Test fixtures
│   ├── web/                   # Static assets
│   └── bin/                   # Compiled binaries (gitignored)
│
├── control/                   # NestJS — SaaS/Control Plane (port 13080)
│   ├── src/
│   │   ├── modules/
│   │   │   ├── admin/         # Admin operations
│   │   │   ├── audit/         # Audit log
│   │   │   ├── auth/          # IAM, MFA, RBAC
│   │   │   ├── backtest/      # Backtest API
│   │   │   ├── billing/       # Subscriptions & billing
│   │   │   ├── commissions/   # Commission engine
│   │   │   ├── device-auth/   # Device activation
│   │   │   ├── licensing/     # License management
│   │   │   ├── payouts/       # Payout engine
│   │   │   ├── referrals/     # Referral network
│   │   │   └── users/         # User management
│   │   └── common/            # Shared utilities
│   └── test/                  # E2E tests
│
├── frontend/                  # Next.js — Presentation Plane (port 13082)
│   ├── src/
│   │   ├── app/
│   │   │   ├── (admin)/       # Admin pages
│   │   │   └── (user)/        # User dashboard pages
│   │   ├── components/        # React components
│   │   ├── lib/               # API hooks & utilities
│   │   ├── config/            # Navigation config
│   │   └── styles/            # Global CSS (theme tokens)
│   └── public/                # Static assets + downloads
│
├── research/                  # Python — Intelligence/Research Plane
│   ├── src/patresearch/
│   │   ├── reference_math.py  # Reference math (parity baseline)
│   │   ├── ml_training.py     # ML training pipeline
│   │   └── quantitative_strategy_engine.py
│   ├── tests/                 # 154 Python tests (153 pass, 1 skip)
│   ├── scripts/               # Research scripts
│   └── fixtures/              # Test fixtures
│
├── windows-agent/             # Go — Windows Agent (MT4/MT5 bridge)
│   ├── cmd/                   # Entrypoint
│   ├── internal/              # Agent logic, IPC, fingerprint
│   └── validation/            # Validation tests
│
├── mql/                       # MQL4/MQL5 Expert Advisors
│   ├── mt4/PredictATrade_MT4.mq4
│   └── mt5/PredictATrade_MT5.mq5
│
├── database/                  # SQL Migrations
│   ├── migrations/            # 69 migration files (numbered to 099)
│   ├── roles/                 # DB role definitions
│   └── seeds/                 # Seed data
│
├── infra/                     # Infrastructure
│   ├── env/                   # Environment files (gitignored; secrets only here)
│   ├── nginx/                 # Nginx configs
│   ├── systemd/               # Systemd service files — DISABLED (docker-first; do not use)
│   ├── docker/                # Docker configs
│   ├── otel/                  # OpenTelemetry config
│   ├── prometheus/            # Prometheus config
│   └── grafana/               # Grafana dashboards
│
├── scripts/                   # Operations scripts
│   ├── full_audit.sh          # Full production audit (51 checks)
│   ├── go_live.sh             # Go-live checklist
│   ├── verify_math_parity.py  # Math parity verification
│   ├── train_ml_model.py      # ML model training
│   ├── migrate.sh             # Database migration runner
│   ├── setup_ml_env.sh        # ML environment setup
│   ├── run_training.sh        # Training pipeline runner
│   ├── verify_live_production.sh
│   ├── benchmark_latency.sh
│   ├── goroutine_profile.sh
│   ├── setup_crons.sh
│   └── bootstrap_artifacts.py
│
├── status/                    # Status page (Node.js, port 13083)
│   └── server.js
│
├── models/                    # ML models (ONNX + metadata)
│   ├── xgb_model.onnx
│   ├── lstm_model.onnx
│   ├── scaler.json
│   ├── feature_columns.json
│   └── model_version.txt
│
├── data/                      # Historical data (gitignored)
│   └── xauusd_historical/     # CSV files (9 timeframes)
│
├── asset_kit/                 # Brand assets
│   ├── app-icons/             # App icons (SVG + PNG)
│   ├── png/                   # Logos (PNG)
│   ├── svg/                   # Logos (SVG)
│   └── previews/              # Preview images
│
├── audit/                     # Audit reports (generated)
│   ├── AUDIT_REPORT.md
│   └── report_YYYYMMDD.json
│
├── logs/                      # Runtime logs (gitignored)
│
├── docs/                      # Documentation
│   ├── INDEX.md               # Documentation index
│   ├── CHANGELOG.md           # Version history
│   ├── Predict-A-Trade_FINAL_SCOPE_OF_WORK.md
│   ├── IMPLEMENTATION_STATUS.md
│   ├── FINAL_TRACEABILITY_MATRIX.md
│   ├── strategy/              # Strategy docs
│   ├── api/                   # API reference
│   ├── database/              # Database docs
│   ├── frontend/              # Frontend docs
│   ├── operations/            # Ops docs
│   ├── guides/                # User/admin guides
│   └── reports/               # Status & audit reports
│
├── .hermes/skills/           # Actual skill library (SKILL.md files)
└── .github/workflows/         # CI/CD workflows
```

## Key Numbers

| Metric | Value |
|--------|-------|
| Go Test Packages | 40/40 pass (`cd realtime && go test ./...`) |
| Python Tests | 154 (153 pass, 1 skip) (`cd research && python3 -m pytest -q`) |
| Frontend Tests | 84 pass (+18 e2e) (`npx jest --passWithNoTests`) |
| Control (NestJS) Tests | 167 pass / 13 suites (`cd control && npm test`) |
| TypeScript Errors | 0 (`tsc --noEmit`) |
| DB Migrations | 69 (database/migrations, numbered to 099) + 0 (realtime/migrations); all applied to live DB |
| DB Tables (user schemas) | 156 (per prior audit) |
| ML Features | 42 |
| ML Models | 5 (bootstrap placeholders — relabeled; NOT production-trained) |
| Strategies | 6 (STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING, MARNIE_FIB, ATEN) |
| Risk Gates | 16 (per-strategy/timeframe isolated, fail-closed) |
| Audit Checks | 51 (all PASS) |
| Directional Signals | 50 |
| Indicators Live | 35/42 |
| API Latency | < 3ms |

## Service Inventory (Docker-First — no systemd)

All services run as Docker containers via `docker compose` with `--env-file infra/env/.env`.
Systemd units (in `infra/systemd/`) are DISABLED and must not be used.

| Service | Container | Port | Status |
|---------|-----------|------|--------|
| Real-Time Engine | pat-realtime | 13081 | ✅ Active (paper/sandbox/advisory) |
| Frontend | pat-frontend | 13082 | ✅ Active |
| Control Plane | pat-control | 13080 | ✅ Active |
| Status Page | pat-status | 13083 | ✅ Active |
| Backtest Service | pat-backtest | 8088 (127.0.0.1 only) | ✅ Active |
| PostgreSQL | pat-postgres | 5432 | ✅ Active |
| Valkey | pat-valkey | 6379 | ✅ Active |
| Ollama | host/container | 11434 | ✅ Active |
| Nginx | pat-nginx | 80/443 | ✅ Active |
| Prometheus | pat-prometheus | 9090 | ✅ Active |
| Grafana | pat-grafana | 3001 | ✅ Active |
| ntfy | pat-ntfy | 8091 | ✅ Active |

## Live Data Status

| Data | Source | Status |
|------|--------|--------|
| XAUUSD Price | Twelve Data API | ✅ Live |
| COT Data | FMP API | ✅ Available (net_position=141636, percentile=0.22) |
| DXY Data | Twelve Data API | ✅ Available (value=98.7451) |
| Candle Cache | Valkey + PostgreSQL | ✅ Active |
| ML Inference | ONNX Runtime | ✅ Active (bootstrap placeholders — NOT production-trained) |
| Sentiment | Ollama (local LLM) | ✅ Connected (provenance: NOT_AI_VERIFIED) |

## Canonical Commands (Docker-First)

```bash
# Build
cd realtime && go build -o bin/realtime-engine ./cmd/realtime-engine/
cd realtime && go build -o bin/backtest-engine ./cmd/backtest-engine/
cd frontend && npx next build
cd control && npm run build

# Test
cd realtime && go test ./...
cd research && python3 -m pytest -q
cd frontend && npx jest --passWithNoTests
cd control && npm test   # NODE_OPTIONS=--experimental-vm-modules (in npm script)

# Lint
cd realtime && go vet ./...
cd frontend && npx tsc --noEmit

# Audit
bash scripts/full_audit.sh

# Deploy / operate (ALL commands MUST use --env-file infra/env/.env)
docker compose --env-file infra/env/.env build realtime
docker compose --env-file infra/env/.env up -d
docker compose --env-file infra/env/.env restart <service>
docker compose --env-file infra/env/.env logs -f <service>
docker compose --env-file infra/env/.env ps

# Migrate (NEVER auto-applied; never rewrite applied history)
./scripts/migrate.sh up
```

## Recent Macro Audit (2026-08-30)

A macroscopic audit (`docs/reports/MACRO_AUDIT_2026-08-30.md`) was performed; its P0 code
fixes were applied earlier and the remaining high-severity items were resolved in this pass:

- **2.1** Agent/data WebSocket now requires `AGENT_WS_TOKEN` (set in `infra/env/realtime.env`
  + `windows-agent.env`; documented in `infra/env/realtime.env.example`).
- **2.2** NestJS Jest green: 13 suites / 167 tests pass via `npm test`
  (`NODE_OPTIONS=--experimental-vm-modules`, already in the script); `tsc --noEmit` clean.
- **2.3** Backtest service public nginx proxy removed; Docker port bound to `127.0.0.1:8088`
  only; control-plane backtest API enforces JWT + `user_id` scoping.
- **2.4** Commission credited only from validated NOWPayments settlement (not license
  assignment); money math moved to `decimal.js` in `billing/nowpayments`.
- **2.5** `PAT_PAPER_EQUITY` removed from `docker-compose.yml`; demo position caps reverted to
  safe production defaults (fail-closed until a verified broker equity feed exists).
- **2.6** Calibration floor raised (`minCalibratedOOSAUC` 0.5 → 0.52, min sample `n>=100`).
- **2.7** `GO_ENGINE_AGENTS_URL` added to `infra/env/control.env` (`http://realtime:13081`).
- **2.9** Dead Prometheus metric graveyard (~57 dead metrics) removed; `/health` and `/ready`
  now DB-aware.
- **2.10** ATEN/astro provenance honestly labeled `QualityDerived` (not Authoritative); LLM
  sentiment remains `NOT_AI_VERIFIED`.

### Production Status (honest)

- **GO** for paper / sandbox / advisory signal operation.
- **NO-GO for live automated trading arming**: `LIVE_TRADING_AUTHORIZED=false` in
  `infra/env/realtime.env` (fail-closed). Live automated trading requires explicit operator
  authorization plus a verified broker equity/order feed before arming.
- No profitability, accuracy, hit-rate, or live-trading-capability claims are made without
  evidence. Demo/replay/sandbox data is labeled and cannot mutate live trading or real finance.
