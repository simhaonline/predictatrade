# Predict-A-Trade Documentation Index
## v1.29.0 — 05 September 2026

### Architecture
- [Architecture & Boundaries](architecture/ARCHITECTURE.md) — 13+ services, NestJS 12/ESM notes, plane boundaries, BE-6 reconciliation, recent changes
- [Flow Diagrams](architecture/FLOW_DIAGRAMS.md) — Mermaid: system planes, tick→signal lifecycle, EXECUTION_ACK → fill reconciliation, auth/session, licensing, payments, update/rollback, backup/DR

### Strategy & Trading
- [Strategy Playbooks](strategy/STRATEGY_PLAYBOOKS.md) — 5 engine configurations, per-strategy exit specs, micro profit-taking, quality grades, expectancy metrics
- [Indicators & Features](strategy/INDICATORS_AND_FEATURES.md) — 42 indicators, 13 evidence pillars, P2 features (all ACTIVE), broker time ORB
- [Risk Gates](strategy/RISK_GATES.md) — 16 gates with per-(strategy, timeframe) isolation, 5% capital protection, operator edge-arming
- [Capital-Tiered Signal Engine](strategy/CAPITAL_TIERS.md) — MICRO/STANDARD/PRO capital bands, per-tier signal viability + delivery, tier-aware sizing (v1.23)

### API
- [REST & WebSocket Reference](api/API_REFERENCE.md) — Full surface: both backends, all 16 control modules + realtime, signal schema, events, rate limits (64-path OpenAPI mirror)
- [OpenAPI 3.0 Spec](api/openapi.json) — machine-readable contract mirrored from `control/openapi.json`

### Operations
- [MT Client Connectivity & 502 Prevention](runbooks/mt-connectivity-502.md) — edge-poll 502 root cause, dual-control HA + nginx failover, connectivity watchdog, triage flow (2026-09-03)

### Database
- [Database Architecture](database/DATABASE_ARCHITECTURE.md) — 16 schemas, 210+ tables live, 65 migrations (unique prefixes), hypertables + retention, money/time invariants
- [ERD (mermaid)](database/DB_ERD.md) — entity relationships for IAM, licensing, commercial, finance, trading, market, audit + invariants

### Mail & Payments
- [Mail Relay Runbook](../../mail-relay/README.md) — pat.predictatrade.com send-only SMTP relay: deployment, DNS (MX/SPF/DKIM/DMARC), env reference
- Payments policy: **USDT-only** (NOWPayments; Stripe disabled at controller). Anti-scam: HMAC IPN + amount verification + UNDERPAID handling — see [API Reference](api/API_REFERENCE.md) §5 and the user billing banner.

### Deployment & Operations
- [Docker Deployment](operations/DOCKER_DEPLOYMENT.md) — Step-by-step Docker Compose guide (includes live terminal, broker timezone)
- [Host Deployment](operations/HOST_DEPLOYMENT.md) — Step-by-step bare-metal/VPS guide (14 steps)- [Backup & Restore](operations/BACKUP_RESTORE.md) — 6-hourly pg_dump + continuous WAL archiving, Hetzner S3 off-host sync (pat-backup-sync), restore & PITR procedures, validation checklist
- [Incident Response Plan](operations/INCIDENT_RESPONSE_PLAN.md) — Classification, response procedures, communication templates
- [Disaster Recovery Plan](operations/DR_PLAN.md) — RTO/RPO, asset inventory, risk assessment, backup strategy, recovery procedures, testing schedule

### Guides
- [Admin Guide](guides/ADMIN_GUIDE.md) — System administration: users, billing, signals, device monitoring, security
- [User Guide](guides/USER_GUIDE.md) — Dashboard, strategies, signal interpretation, MT4/MT5 setup, troubleshooting
- [EA Client Guide (MT4/MT5)](guides/EA_CLIENT_GUIDE.md) — Option B EA-direct cloud transport: install, WebRequest allowlist, signal-delivery guarantees, troubleshooting

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
- [Changelog](../../realtime/CHANGELOG.md) — version history (v1.29.0 current)
