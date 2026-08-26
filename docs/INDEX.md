# Predict-A-Trade XAUUSD — Documentation Index
## v1.16.0 — 26 August 2026

### Root-Level Documents (authoritative single source of truth)

| File | Purpose |
|------|---------|
| `/SCOPE_OF_WORK.md` | Full project scope, strategy engines, risk gates, P2 features |
| `/API_REFERENCE.md` | REST + WebSocket API reference |
| `/DEPLOYMENT_GUIDE.md` | Docker Compose setup, env vars, build commands |
| `/DOCKER_COMPOSE_REFERENCE.md` | 11 services, networks, volumes, health checks |
| `/CHANGELOG.md` | Condensed version history v1.0-v1.16.0 |
| `/PRODUCTION_READINESS_AUDIT.md` | Consolidated audit: CONDITIONAL GO, 70/100 |
| `/README.md` | System overview + quick start |

### Architecture
- [Architecture & Boundaries](architecture/ARCHITECTURE.md) — Plane boundaries, service diagrams, data flow

### Strategy & Trading
- [Strategy Playbooks](strategy/STRATEGY_PLAYBOOKS.md) — 5 engine configurations, thresholds, SL/TP
- [Indicators & Features](strategy/INDICATORS_AND_FEATURES.md) — 42 indicators, 13 evidence pillars

### API Reference
- [API Reference](api/API_REFERENCE.md) — REST endpoints + WebSocket events

### Database
- [Database Architecture](database/DATABASE_ARCHITECTURE.md) — Schema design, migrations, hypertables

### Frontend
- [Frontend Overview](frontend/FRONTEND_OVERVIEW.md) — Pages, components, theme system

### Operations
- [Deployment Guide](operations/DEPLOYMENT_GUIDE.md) — Full deployment instructions
- [Troubleshooting](operations/TROUBLESHOOTING.md) — Common issues and solutions

### Compliance
- [Data Retention](compliance/DATA_RETENTION.md) — Data retention policies
- [Audit Logging](compliance/AUDIT_LOGGING.md) — Audit trail implementation

### Reports
- [Production Readiness Audit](reports/PRODUCTION_READINESS_AUDIT.md) — Consolidated audit v1.16.0