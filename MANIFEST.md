# Predict-A-Trade Project Manifest

## Version: v1.3.0 — Production Remediation + External Feeds

## Repository Structure

```
predictatrade/xauusd/
├── realtime/                    # Go real-time trading engine
│   ├── cmd/realtime-engine/     # Main entrypoint — full signal pipeline
│   ├── internal/
│   │   ├── adaptation/          # Rule-based market phase adaptation
│   │   ├── cache/               # Valkey hot cache
│   │   ├── calibration/         # Sigmoid probability calibration
│   │   ├── config/              # Engine configuration + advanced config
│   │   ├── features/            # 19+ feature engines (indicators, structure, etc.)
│   │   ├── gates/               # 12 hard risk gates (short-circuit)
│   │   ├── gateway/             # HTTP + WebSocket + Agent WS gateway
│   │   ├── hedging/             # Controlled hedge manager (disabled by default)
│   │   ├── maintenance/         # Daily maintenance scheduler (UTC)
│   │   ├── marketdata/          # Market data provider + persistence + agent
│   │   ├── ml/                  # ML adaptation manager + model registry
│   │   ├── observability/       # Structured logging + 50+ Prometheus metrics
│   │   ├── ptb/                 # Professional Trader Brain (9 files, 20+ modules)
│   │   ├── reconciliation/      # Signal reconciliation
│   │   ├── recovery/            # Loss recovery / capital protection manager
│   │   ├── rl/                  # RL strategy optimizer (disabled→shadow→filter→live)
│   │   ├── sentiment/           # Real-time sentiment engine (async, cached)
│   │   ├── signal/              # Signal engine + advanced integration + cooldown + delivery
│   │   ├── strategy/            # 4 strategy engines + confluence + golden tests
│   │   └── types/               # Canonical types
│   └── pkg/math/                # Quantitative math functions
├── control/                     # NestJS SaaS control plane
│   └── src/modules/             # Auth, MFA, billing, licensing, referrals, commissions, payouts
├── frontend/                    # Next.js frontend (admin + user dashboards, 40+ pages)
├── database/migrations/         # 15 SQL migrations (165 tables)
├── research/                    # Python research plane
│   └── src/patresearch/
│       ├── backtesting/         # Event-driven backtesting framework (27 modules)
│       │   ├── data/            # Loader, quality validator, MTF alignment, session calendar
│       │   ├── engine/          # Event-driven core, execution simulator, portfolio
│       │   ├── strategy/        # PTB adapter, precomputed PTB, RL standalone, RL filter
│       │   ├── analytics/       # Metrics, walk-forward, Monte Carlo, sensitivity
│       │   ├── reporting/       # Report generator, run manifest
│       │   ├── features/        # PTB feature precomputation
│       │   └── config/          # Environment configuration
│       ├── backtester.py        # Legacy backtester (retained)
│       ├── calibration.py       # Brier, ECE, Wilson, sigmoid calibration
│       ├── dataset.py           # Data import + provenance
│       ├── ml_training.py       # ML training pipeline (chronological split, walk-forward)
│       ├── reference_math.py    # Canonical quant math (parity with Go)
│       └── rl_training.py       # RL training environment (reward function)
├── windows-agent/               # Go Windows Agent for MT4/MT5
├── mql/                         # MT4 + MT5 Expert Advisors (4 EAs)
├── infra/
│   ├── env/                     # 6 environment files (canonical, control, frontend, realtime, status, windows-agent)
│   ├── nginx/                   # Nginx config + 5 site configs + security headers
│   ├── prometheus/              # Prometheus scrape config
│   ├── systemd/                 # 4 systemd service units (security hardened)
│   └── grafana/                 # Grafana dashboards + provisioning
├── docs/                        # 30+ documentation files
├── scripts/
│   ├── backup/                  # Database backup + restore scripts
│   ├── migrate.sh               # Migration runner
│   └── windows/                 # 7 PowerShell validation scripts
├── README.md                    # System overview
├── AGENTS.md                    # Codex instructions
├── MANIFEST.md                  # This file
├── SKILLS.md                    # Skills index
├── PRODUCTION_FULL_AUDIT_REPORT.md  # Full forensic audit (671 lines)
└── Makefile                     # Build/test/lint targets for all planes
```

## Key Numbers

| Metric | Value |
|--------|-------|
| Go source files | ~70 |
| Python source files | ~40 |
| Go internal packages | 22 |
| PTB module files | 9 (20+ modules, all SHADOW) |
| Database migrations | 15 |
| Database tables | 165 |
| Database schemas | 12 (iam, control, billing, referral, licensing, trading, market, audit, ai, research, system, support) |
| Strategy engines | 4 (STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING) |
| PTB modules | 20 + synthesis + correlation + gold role |
| Hard risk gates | 12 (short-circuit) |
| Signal states | 6 (BUY/SELL/WAIT/NO-TRADE/BLOCKED/ERROR) |
| Advanced modules | 7 (recovery, adaptation, hedging, ML, RL, sentiment, maintenance) |
| Backtesting modules | 27 (data, engine, strategy, analytics, reporting, features, config) |
| Prometheus metrics | 50+ |
| Total tests | 448 (243 Go + 98 Python + 68 NestJS + 39 Frontend) |
| Frontend pages | 40+ (admin 20+, user 20+, auth 4) |
| MQL EAs | 4 (MT4 × 2, MT5 × 2) |
| Nginx site configs | 5 |
| Systemd units | 4 |
| Environment files | 6 |
| Documentation files | 30+ |
