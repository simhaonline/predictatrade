# Documentation Index

## Root Control Files
- `README.md` — System overview, architecture, strategies, PTB, advanced features, backtesting, testing
- `AGENTS.md` — Codex repository instructions (canonical behavioral contract)
- `AGENT.md` — Compatibility pointer to AGENTS.md
- `MANIFEST.md` — Project manifest (key numbers, repository structure, service inventory)
- `SKILLS.md` — Skills index (human-readable)
- `Makefile` — Canonical build/lint/test commands for all four planes
- `run.sh` — Startup orchestration script
- `docker-compose.yml` — Local Docker infrastructure (Postgres, Valkey, Prometheus, Grafana)
- `.gitleaks.toml` — Gitleaks secret-scanning configuration with dev-test allowlists
- `.mcp.json` — MCP portability representation (Codex-native truth is `.codex/config.toml`)

## Forensic Audit
- `docs/FEATURE_CAPABILITY_FORENSIC_AUDIT.md` — Full forensic capability audit of 26 feature groups with runtime call-path verification

## Architecture & Boundaries
- `docs/Predict-A-Trade_FINAL_SCOPE_OF_WORK.md` — Canonical SOW v1.0.0 (implementation contract)
- `docs/ADVANCED_RISK_ADAPTATION_INTELLIGENCE.md` — Loss recovery, adaptation, hedging, ML/RL, sentiment, daily maintenance, metrics, pipeline wiring
- `docs/ADVANCED_XAUUSD_INTELLIGENCE.md` — PTB module status table and synthesis output
- `docs/AI_AGENT_VS_DETERMINISTIC_MATRIX.md` — Component classification (all deterministic, no AI)
- `docs/LIVE_MARKET_DATA_PROVENANCE.md` — Data flow, authenticity guard, no-fake-data policy
- `docs/FINAL_TRACEABILITY_MATRIX.md` — SOW requirement to implementation traceability
- `docs/IMPLEMENTATION_STATUS.md` — Current implementation status per plane

## Strategy & Mathematics
- `docs/strategy/STRATEGY_PLAYBOOKS.md` — Four strategy configurations, candidate thresholds, TP/SL ATR multipliers, and playbooks
- `docs/strategy/INDICATORS_AND_FEATURES.md` — Indicator inventory and feature engines (42 features)
- `docs/SIGNAL_TYPES_AND_PROBABILITY.md` — Signal direction types, candidate thresholds, calibrated probability, TP/SL geometry, MQL strategy selection, and signal delivery chain
- `docs/TRADING_MATHEMATICS_SPECIFICATION.md` — All formulas: indicators, scoring, calibration, PTB synthesis, correlation
- `docs/BACKTESTING.md` — Backtesting framework guide: architecture, data, execution, PTB parity, walk-forward, Monte Carlo, CLI, reports
- `docs/XAUUSD_ACCURACY_ENHANCEMENT_REPORT.md` — Accuracy enhancement methodology and results

## MT4/MT5 & Windows Agent
- `docs/MT4_MT5_TRADE_MANAGEMENT_PARITY.md` — MT4/MT5 trade management parity matrix
- `docs/TRADE_MANAGEMENT_FORENSIC_REPORT.md` — Trade management forensic audit report
- `docs/LIVE_MT4_MT5_RUNTIME_VALIDATION.md` — Operator runbook for live terminal validation (13 read-only steps)

## News, Breakout, OCO & Notifications (v1.10.0)
- `docs/guides/SELF_HOSTED_NOTIFICATIONS_SETUP.md` — Self-hosted notification adapter setup guide (email, Telegram, WhatsApp, push)
- `realtime/pkg/news/` — Economic calendar provider interface, FMP adapter, risk engine (fail-safe)
- `realtime/internal/breakout/` — News breakout pending-order planning engine (disabled by default)
- `realtime/internal/oco/` — OCO durable state machine with race handling and restart reconciliation
- `realtime/pkg/notifications/` — External notification adapters (email, Telegram, WhatsApp, push)

## API Reference
- `docs/api/API_REFERENCE.md` — REST API and WebSocket event schema reference

## Database
- `docs/database/DATABASE_ARCHITECTURE.md` — Schema design, hypertables, indexes, constraints
- `docs/database/DATABASE_MIGRATION_REPORT.md` — Migration history and status
- `docs/database/DATABASE_BACKUP_AND_RECOVERY_REPORT.md` — Backup and recovery procedures
- `docs/database/DATABASE_DISASTER_RECOVERY.md` — DR plan and RTO/RPO targets
- `docs/database/DATABASE_PERFORMANCE_REPORT.md` — Performance benchmarks and optimization
- `docs/database/DATABASE_TRACEABILITY_MATRIX.md` — Database requirement traceability
- `docs/database/FINAL_DATABASE_STATUS.md` — Final database status summary

## Frontend
- `docs/frontend/README.md` — Frontend architecture overview
- `docs/frontend/admin-api-route-matrix.md` — Admin API route coverage matrix
- `docs/frontend/admin-dashboard.md` — Admin dashboard features and pages
- `docs/frontend/admin-requirements-traceability.md` — Admin requirements traceability
- `docs/frontend/backend-api-inventory.md` — Backend API inventory for frontend consumption
- `docs/frontend/cookie-consent.md` — Cookie consent implementation
- `docs/frontend/frontend-route-api-matrix.md` — Frontend route to API mapping
- `docs/frontend/legal-pages.md` — Legal pages (privacy policy, terms, etc.)
- `docs/frontend/theme-system.md` — Theme system (light/dark tokens, CSS variables, Tailwind config)

## Operations
- `docs/operations/DEPLOYMENT_GUIDE.md` — Deployment guide for all services
- `docs/operations/DOMAIN_ROUTING_MATRIX.md` — Domain routing and Nginx configuration
- `docs/operations/TROUBLESHOOTING.md` — Common issues and solutions

## Guides
- `docs/guides/ADMIN_GUIDE.md` — Admin operations guide
- `docs/guides/INSTALL.md` — Installation and setup guide
- `docs/guides/USER_GUIDE.md` — End-user guide

## Reports
- `docs/reports/PRODUCTION_STATUS_REPORT.md` — Current production status (live, consolidated)
- `docs/reports/GO_NOGO_REPORT.md` — Go/No-Go decision report
- `docs/reports/FINAL_PRODUCTION_READINESS_REPORT.md` — Final production readiness assessment
- `docs/reports/COMPREHENSIVE_PROJECT_REPORT.md` — Comprehensive project report
- `docs/reports/PRODUCTION_FULL_AUDIT_REPORT.md` — Full production audit results
- `docs/reports/REMAINING_EXTERNAL_DEPENDENCIES.md` — Remaining external dependencies

## Changelog
- `docs/CHANGELOG.md` — Version history and changes

## Color & Brand
- `asset_kit/` — Brand assets (logos, icons, previews, app icons)
- `frontend/src/styles/globals.css` — CSS variables implementing the approved palette
- `frontend/tailwind.config.ts` — Tailwind semantic color tokens (pat-success, pat-danger, etc.)
