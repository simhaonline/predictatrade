# Predict-A-Trade Project Manifest

## Version: v1.10.1 — Cross-Check Remediation + News Risk Wiring + Migration 022 Applied

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
├── .gitleaks.toml             # Secret-scanning config with dev-test allowlists
├── .mcp.json                  # MCP portability representation
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
│   ├── migrations/            # Go-level migrations
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
│   ├── tests/                 # 127 Python tests
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
│   ├── migrations/            # 21+ migration files
│   ├── roles/                 # DB role definitions
│   └── seeds/                 # Seed data
│
├── infra/                     # Infrastructure
│   ├── env/                   # Environment files (6 services)
│   ├── nginx/                 # Nginx configs
│   ├── systemd/               # Systemd service files
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
├── .agents/skills/            # Codex skills (SKILL.md files)
├── .codex/                    # Codex config & agents
└── .github/workflows/         # CI/CD workflows
```

## Key Numbers

| Metric | Value |
|--------|-------|
| Go Test Packages | 24/24 pass |
| Python Tests | 127 pass |
| Frontend Tests | 70 pass |
| TypeScript Errors | 0 |
| DB Migrations | 21+ |
| DB Tables (user schemas) | 156 |
| ML Features | 42 |
| ML Models | 5 (XGBoost, LSTM, scaler, columns, version) |
| Strategies | 4 (STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING) |
| Risk Gates | 12+ |
| Audit Checks | 51 (all PASS) |
| Directional Signals | 50 |
| Indicators Live | 35/42 |
| API Latency | < 3ms |

## Service Inventory

| Service | Binary/Process | Port | Systemd Unit | Status |
|---------|---------------|------|--------------|--------|
| Real-Time Engine | `realtime/bin/realtime-engine` | 13081 | predictatrade-realtime | ✅ Active |
| Frontend | `next-server` | 13082 | predictatrade-frontend | ✅ Active |
| Control Plane | `node` (NestJS) | 13080 | predictatrade-control | ✅ Active |
| Status Page | `node` (status/server.js) | 13083 | predictatrade-status | ⚠️ Inactive |
| PostgreSQL | Docker | 5432 | — | ✅ Active |
| Valkey | Docker | 6379 | — | ✅ Active |
| Ollama | `ollama` | 11434 | — | ✅ Active |
| Nginx | `nginx` | 80/443 | — | ✅ Active |

## Live Data Status

| Data | Source | Status |
|------|--------|--------|
| XAUUSD Price | Twelve Data API | ✅ Live |
| COT Data | FMP API | ✅ Available (net_position=141636, percentile=0.22) |
| DXY Data | Twelve Data API | ✅ Available (value=98.7451) |
| Candle Cache | Valkey + PostgreSQL | ✅ Active |
| ML Inference | ONNX Runtime | ✅ Active (non-constant output) |
| Sentiment | Ollama (local LLM) | ✅ Connected |

## Canonical Commands

```bash
# Build
cd realtime && go build -o bin/realtime-engine ./cmd/realtime-engine/
cd realtime && go build -o bin/backtest-engine ./cmd/backtest-engine/
cd frontend && npx next build
cd control && npm run build

# Test
cd realtime && go test ./internal/... ./pkg/... -count=1
cd research && python3 -m pytest -q
cd frontend && npx jest --passWithNoTests
cd control && npx jest --passWithNoTests

# Lint
cd realtime && go vet ./...
cd frontend && npx tsc --noEmit

# Audit
bash scripts/full_audit.sh

# Deploy
systemctl restart predictatrade-realtime
systemctl restart predictatrade-frontend
systemctl restart predictatrade-control

# Migrate
./scripts/migrate.sh up
```
