# Predict-A-Trade XAUUSD — Documentation

> Multi-plane XAUUSD algorithmic trading platform · v1.16.0 · GO (100/100)

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
| [Admin Guide](guides/ADMIN_GUIDE.md) | System administration, agent monitoring, signals |
| [User Guide](guides/USER_GUIDE.md) | Dashboard, MT4/MT5 setup, signal interpretation |
| [Whitepaper](reports/WHITEPAPER.md) | 12-section technical whitepaper |
| [PhD Thesis](reports/PHD_THESIS.md) | 9-chapter academic thesis |

## Key Metrics

| Metric | Value |
|--------|-------|
| Strategy Engines | 5 (4 active, 1 shadow) |
| Technical Indicators | 42 (35 live, 7 warming) |
| Evidence Pillars | 13 |
| Risk Gates | 16 (per-strategy/timeframe isolated) |
| Services (Docker) | 11 |
| Test Packages | 28/28 PASS |
| Production Readiness | 100/100 |

## Repository

[github.com/simhaonline/predictatrade](https://github.com/simhaonline/predictatrade)
