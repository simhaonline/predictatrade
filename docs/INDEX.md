# Predict-A-Trade Documentation Index
## v1.17.3 — 29 August 2026

### Architecture
- [Architecture & Boundaries](architecture/ARCHITECTURE.md) — 13+ services, NestJS 12/ESM notes, plane boundaries, BE-6 reconciliation, recent changes
- [Flow Diagrams](architecture/FLOW_DIAGRAMS.md) — Mermaid: system planes, tick→signal lifecycle, EXECUTION_ACK → fill reconciliation, auth/session, licensing, payments, update/rollback, backup/DR

### Strategy & Trading
- [Strategy Playbooks](strategy/STRATEGY_PLAYBOOKS.md) — 5 engine configurations, per-strategy exit specs, micro profit-taking, quality grades, expectancy metrics
- [Indicators & Features](strategy/INDICATORS_AND_FEATURES.md) — 42 indicators, 13 evidence pillars, P2 features (all ACTIVE), broker time ORB
- [Risk Gates](strategy/RISK_GATES.md) — 16 gates with per-(strategy, timeframe) isolation, 5% capital protection, operator edge-arming

### API
- [REST & WebSocket Reference](api/API_REFERENCE.md) — Full surface: both backends, all 16 control modules + realtime, signal schema, events, rate limits (64-path OpenAPI mirror)
- [OpenAPI 3.0 Spec](api/openapi.json) — machine-readable contract mirrored from `control/openapi.json`

### Database
- [Database Architecture](database/DATABASE_ARCHITECTURE.md) — 16 schemas, 210+ tables live, 65 migrations (unique prefixes), hypertables + retention, money/time invariants
- [ERD (mermaid)](database/DB_ERD.md) — entity relationships for IAM, licensing, commercial, finance, trading, market, audit + invariants

### Deployment & Operations
- [Docker Deployment](operations/DOCKER_DEPLOYMENT.md) — Step-by-step Docker Compose guide (includes live terminal, broker timezone)
- [Host Deployment](operations/HOST_DEPLOYMENT.md) — Step-by-step bare-metal/VPS guide (14 steps)- [Backup & Restore](operations/BACKUP_RESTORE.md) — Automated backup scripts, restore procedures, validation checklist
- [Incident Response Plan](operations/INCIDENT_RESPONSE_PLAN.md) — Classification, response procedures, communication templates
- [Disaster Recovery Plan](operations/DR_PLAN.md) — RTO/RPO, asset inventory, risk assessment, backup strategy, recovery procedures, testing schedule

### Guides
- [Admin Guide](guides/ADMIN_GUIDE.md) — System administration: users, billing, signals, agent monitoring, security
- [User Guide](guides/USER_GUIDE.md) — Dashboard, strategies, signal interpretation, MT4/MT5 setup, troubleshooting
- [Windows Agent Guide](guides/WINDOWS_AGENT.md) — Client Agent + Master Node roles, install/update/uninstall, health endpoints, EA downloads, deploy files

### Reports (historical record — pre-29-Aug state)
- [Whitepaper](reports/WHITEPAPER.md) — Technical whitepaper: architecture, risk, AI governance, commercial model
- [PhD Thesis](reports/PHD_THESIS.md) — Academic thesis: 9 chapters, literature review, formal contributions
- [UI/UX Audit Report](reports/UI_UX_AUDIT_REPORT.md) — Dashboard accessibility, UX, visual-consistency, and performance audit (41 findings)
- [Macroscopic Audit Report](reports/MACROSCOPIC_AUDIT_REPORT.md) — System-wide codebase + database audit (initial)
- [Macroscopic Audit (28 Aug)](reports/MACROSCOPIC_AUDIT_REPORT_2026-08-28.md) — IT & Compliance re-audit: NO-GO verdict, 29 findings, launch-blockers
- [Macroscopic Audit Revisit (28 Aug)](reports/MACROSCOPIC_AUDIT_REVISIT_2026-08-28.md) — GO/NO-GO update: CONDITIONAL GO, blockers resolved
- [Remediation Report (28 Aug)](reports/REMEDIATION_REPORT_2026-08-28.md) — Launch-blocker fixes + post-remediation incident

### Current status (living documents)
- [Implementation Status](reports/IMPLEMENTATION_STATUS.md) — per-requirement implementation ledger (29 Aug)
- [Changelog](../../realtime/CHANGELOG.md) — version history (v1.17.3 current)
