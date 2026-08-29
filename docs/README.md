# Predict-A-Trade XAUUSD — Documentation

> Multi-plane XAUUSD algorithmic trading platform · v1.17.4 · GO — weekend-hardened: market_closed liveness, USDT-only payments, mail relay, agents v1.2.44; CI 6/6 green, dashboards runtime-audited, Windows Agent verified

Welcome to the official documentation for Predict-A-Trade — a production-grade XAUUSD trading signal generation platform. Use the sidebar to navigate, or start with the sections below.

## Quick Navigation

| Section | Description |
|---------|-------------|
| [Architecture](architecture/ARCHITECTURE.md) | Five-plane model, 11 services, broker timezone, data flow |
| [Strategy Playbooks](strategy/STRATEGY_PLAYBOOKS.md) | 5 trading engines with per-strategy exit specs, micro TP, quality grades |
| [Indicators & Features](strategy/INDICATORS_AND_FEATURES.md) | 42 indicators, 13 evidence pillars, P2 features (all ACTIVE) |
| [Risk Gates](strategy/RISK_GATES.md) | 16-gate pipeline with per-(strategy, timeframe) isolation, 5% capital protection |
| [API Reference](api/API_REFERENCE.md) | REST + WebSocket endpoints, signal object schema, license validation |
| [Database](database/DATABASE_ARCHITECTURE.md) | Schemas, 30+ migrations, trade_results, agent bridging |
| [Docker Deployment](operations/DOCKER_DEPLOYMENT.md) | Step-by-step Docker Compose guide |
| [Host Deployment](operations/HOST_DEPLOYMENT.md) | Step-by-step bare-metal/VPS guide |
| [Backup & Restore](operations/BACKUP_RESTORE.md) | Automated backup scripts, restore procedures |
| [Incident Response](operations/INCIDENT_RESPONSE_PLAN.md) | Classification, response, communication |
| [Disaster Recovery](operations/DR_PLAN.md) | RTO/RPO, backups, testing |
| [Mail Relay Runbook](https://github.com/simhaonline/predictatrade/tree/main/mail-relay) | Go SMTP submission relay, DNS records |
| [Admin Guide](guides/ADMIN_GUIDE.md) | System administration, agent monitoring, signals |
| [User Guide](guides/USER_GUIDE.md) | Dashboard, MT4/MT5 setup, signal interpretation |
| [Windows Agent Guide](guides/WINDOWS_AGENT.md) | Client Agent + Master Node roles, install/update/uninstall, health endpoints |
| [Whitepaper](reports/WHITEPAPER.md) | 12-section technical whitepaper |
| [PhD Thesis](reports/PHD_THESIS.md) | 9-chapter academic thesis |
| [UI/UX Audit Report](reports/UI_UX_AUDIT_REPORT.md) | Dashboard accessibility, UX, visual, performance audit |
| [Macroscopic Audit Report](reports/MACROSCOPIC_AUDIT_REPORT.md) | System-wide codebase + database audit |
| [Macroscopic Audit (28 Aug)](reports/MACROSCOPIC_AUDIT_REPORT_2026-08-28.md) | IT & Compliance re-audit — NO-GO, launch-blockers |
| [Macroscopic Audit Revisit (28 Aug)](reports/MACROSCOPIC_AUDIT_REVISIT_2026-08-28.md) | GO/NO-GO update — CONDITIONAL GO |
| [Remediation Report (28 Aug)](reports/REMEDIATION_REPORT_2026-08-28.md) | Launch-blocker fixes + incident post-mortem |

## Key Metrics

| Metric | Value |
|--------|-------|
| Strategy Engines | 5 (4 active, 1 shadow) |
| Technical Indicators | 42 (35 live, 7 warming) |
| Evidence Pillars | 13 |
| Risk Gates | 16 (per-strategy/timeframe isolated) |
| Services (Docker) | 13 healthy |
| Tests | Go 28 pkgs · control 167 · frontend 84 + e2e 18 · Python 139 — all PASS |
| CI jobs | 6/6 green |
| API surface | 64 documented paths (OpenAPI 3.0 in [`api/openapi.json`](api/openapi.json)) |
| Dashboard pages | 38 runtime-audited (25 admin + 19 user routes) |
| Windows Agent | v1.2.44, self-healing installer + honest telemetry |
| Payments | USDT-only (NOWPayments verified settlement; Stripe off) |
| Outbound Mail | Go SMTP relay (pat.predictatrade.com), spool+retry |

## Repository

[github.com/simhaonline/predictatrade](https://github.com/simhaonline/predictatrade)
