# Predict-A-Trade XAUUSD — Documentation

> Multi-plane XAUUSD algorithmic trading platform · engine v1.24.2 / EA v1.31 (10 Sep 2026) · GO for paper/sandbox/advisory; live-trading arming is operator-authorized with fail-closed capital-protection gates. Docker-first deployment via `docker compose --env-file infra/env/.env`. EA-direct cloud transport verified.

Welcome to the official documentation for Predict-A-Trade — a production-grade XAUUSD trading signal generation platform. Use the sidebar to navigate, or start with the sections below.

## Documentation governance

- **Canonical sources:** root `README.md` (entry point) → this index (navigation) → one topic document per subject. Duplicates must link to the canonical doc, never fork content.
- **Update-with-code:** any PR changing ports, env vars, endpoints, gates, strategies, migrations, or service topology must update the affected doc in the same change (see `docs/CHANGELOG-DOCS.md`).
- **Never document ahead of code.** Features marked PLANNED/EXPERIMENTAL must say so; audit reports are append-only history (archive by date folder, redact secrets, never delete).
- **Version/migration claims must cite source** (`realtime/internal/version`, `ls database/migrations | wc -l`).
- **Banners:** active docs carry a date+engine-version header; archived material lives under `docs/archive/` with an ARCHIVED marker.

## Quick Navigation

| Section | Description |
|---------|-------------|
| [Architecture](architecture/ARCHITECTURE.md) | Five-plane model, 11 services, broker timezone, data flow |
| [Strategy Playbooks](strategy/STRATEGY_PLAYBOOKS.md) | 5 trading engines with per-strategy exit specs, micro TP, quality grades |
| [Indicators & Features](strategy/INDICATORS_AND_FEATURES.md) | 42 indicators, 13 evidence pillars, P2 features (all ACTIVE) |
| [Risk Gates](strategy/RISK_GATES.md) | 24-gate pipeline with per-(strategy, timeframe) isolation, nested daily/weekly/monthly capital protection |
| [API Reference](api/API_REFERENCE.md) | REST + WebSocket endpoints, signal object schema, license validation |
| [Database](database/DATABASE_ARCHITECTURE.md) | Schemas, 102 migrations (numbered to 142), trade_results, agent bridging |
| [Docker Deployment](operations/DOCKER_DEPLOYMENT.md) | Step-by-step Docker Compose guide |
| [Host Deployment](operations/HOST_DEPLOYMENT.md) | Step-by-step bare-metal/VPS guide |
| [Backup & Restore](operations/BACKUP_RESTORE.md) | Automated backup scripts, restore procedures |
| [Incident Response](operations/INCIDENT_RESPONSE_PLAN.md) | Classification, response, communication |
| [Disaster Recovery](operations/DR_PLAN.md) | RTO/RPO, backups, testing |
| [Mail Relay Runbook](https://github.com/simhaonline/predictatrade/tree/main/mail-relay) | Go SMTP submission relay, DNS records |
| [Admin Guide](guides/ADMIN_GUIDE.md) | System administration, agent monitoring, signals |
| [User Guide](guides/USER_GUIDE.md) | Dashboard, MT4/MT5 EA setup, signal interpretation |
| [EA Client Guide](guides/EA_CLIENT_GUIDE.md) | EA-direct cloud transport: Client + Master Node roles, WebRequest allowlist, signal-delivery guarantees |
| [Whitepaper](reports/WHITEPAPER.md) | 12-section technical whitepaper |
| [PhD Thesis](reports/PHD_THESIS.md) | 9-chapter academic thesis |
| [UI/UX Audit Report](reports/UI_UX_AUDIT_REPORT.md) | Dashboard accessibility, UX, visual, performance audit |
| [Macroscopic Audit Report](reports/MACROSCOPIC_AUDIT_REPORT.md) | System-wide codebase + database audit |
| [Macroscopic Audit (28 Aug)](archive/2026-08-reports/MACROSCOPIC_AUDIT_REPORT_2026-08-28.md) | ARCHIVED: IT & Compliance re-audit — NO-GO, launch-blockers |
| [Macroscopic Audit Revisit (28 Aug)](archive/2026-08-reports/MACROSCOPIC_AUDIT_REVISIT_2026-08-28.md) | ARCHIVED: GO/NO-GO update — CONDITIONAL GO |
| [Remediation Report (28 Aug)](archive/2026-08-reports/REMEDIATION_REPORT_2026-08-28.md) | ARCHIVED: launch-blocker fixes + incident post-mortem |

## Key Metrics

| Metric | Value |
|--------|-------|
| Strategy Engines | 7 (Standard Scalping, Ultra Scalping, Standard Swing, Trend Swing, EQFE, ATEN, IMLR) |
| Technical Indicators | 42 (35 live, 7 warming) |
| Evidence Pillars | 13 |
| Risk Gates | 16 (per-strategy/timeframe isolated) |
| Services (Docker) | 16 (docker compose, all healthy) |
| Tests | Go 39 pkgs · control 14 suites/174 · frontend 84 + e2e 18 · Python 154 (152 pass, 2 skip) — all PASS |
| CI jobs | 6/6 green |
| API surface | 64 documented paths (OpenAPI 3.0 in [`api/openapi.json`](api/openapi.json)) |
| Dashboard pages | 38 runtime-audited (25 admin + 19 user routes) |
| Windows Agent | REMOVED (v1.19.0 Option B) — EAs talk to the cloud directly; see [EA Client Guide](guides/EA_CLIENT_GUIDE.md) |
| Payments | USDT-only (NOWPayments verified settlement; Stripe off) |
| Outbound Mail | Go SMTP relay (pat.predictatrade.com), spool+retry |

## Repository

[github.com/simhaonline/predictatrade](https://github.com/simhaonline/predictatrade)
