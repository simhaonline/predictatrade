# Predict-A-Trade Documentation Index
## v1.16.0 — 26 August 2026

### Architecture
- [Architecture & Boundaries](architecture/ARCHITECTURE.md) — 11 services, broker timezone model, plane boundaries, recent architectural changes

### Strategy & Trading
- [Strategy Playbooks](strategy/STRATEGY_PLAYBOOKS.md) — 5 engine configurations, per-strategy exit specs, micro profit-taking, quality grades, expectancy metrics
- [Indicators & Features](strategy/INDICATORS_AND_FEATURES.md) — 42 indicators, 13 evidence pillars, P2 features (all ACTIVE), broker time ORB
- [Risk Gates](strategy/RISK_GATES.md) — 16 gates with per-(strategy, timeframe) isolation, 5% capital protection, operator edge-arming

### API
- [REST & WebSocket Reference](api/API_REFERENCE.md) — All endpoints for both backends, signal object schema, license validation, agent status

### Database
- [Database Architecture](database/DATABASE_ARCHITECTURE.md) — Schemas, 30+ migrations, trade_results table, agent connection bridging, hypertables, money types

### Deployment & Operations
- [Docker Deployment](operations/DOCKER_DEPLOYMENT.md) — Step-by-step Docker Compose guide (includes live terminal, broker timezone)
- [Host Deployment](operations/HOST_DEPLOYMENT.md) — Step-by-step bare-metal/VPS guide (14 steps)
- [Backup & Restore](operations/BACKUP_RESTORE.md) — Automated backup scripts, restore procedures, validation checklist
- [Incident Response Plan](operations/INCIDENT_RESPONSE_PLAN.md) — Classification, response procedures, communication templates
- [Disaster Recovery Plan](operations/DR_PLAN.md) — RTO/RPO, asset inventory, risk assessment, backup strategy, recovery procedures, testing schedule

### Guides
- [Admin Guide](guides/ADMIN_GUIDE.md) — System administration: users, billing, signals, agent monitoring, security
- [User Guide](guides/USER_GUIDE.md) — Dashboard, strategies, signal interpretation, MT4/MT5 setup, troubleshooting
- [Windows Agent Guide](guides/WINDOWS_AGENT.md) — Client Agent + Master Node roles, install/update/uninstall, health endpoints, deploy files

### Reports
- [Whitepaper](reports/WHITEPAPER.md) — Technical whitepaper: architecture, risk, AI governance, commercial model
- [PhD Thesis](reports/PHD_THESIS.md) — Academic thesis: 9 chapters, literature review, formal contributions
- [UI/UX Audit Report](reports/UI_UX_AUDIT_REPORT.md) — Dashboard accessibility, UX, visual-consistency, and performance audit (41 findings)
