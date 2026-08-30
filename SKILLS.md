# SKILLS.md — Predict-A-Trade v1.0.0 Skill Index

> Human-readable index. Skills live at `.hermes/skills/<name>/SKILL.md`.

| Skill | Purpose |
|---|---|
| `repo-audit` | Audit repository before major changes: map services/dependencies/data flows/tests/migrations/legacy code and classify REUSE/EXTEND/ADAPT/REPLACE/NEW/DEPRECATE. |
| `architecture-guardrails` | Enforce Go realtime, Python research, NestJS control, Next.js presentation and Windows/MQL edge boundaries without hot-path commercial coupling. |
| `xauusd-market-data` | Implement/review XAUUSD/GC data capabilities, provenance, quality, history, futures roll/basis, session/calendar and macro feeds without fabricated flow. |
| `xauusd-strategy-spec` | Specify four distinct versioned XAUUSD strategies with timeframes/features/confluence/risk/execution/prediction/exit profiles and net-expectancy logic. |
| `xauusd-quant-validation` | Validate strategies/models with leakage-safe data, realistic costs, walk-forward/OOS, calibration, confidence intervals, Monte Carlo, parity, drift and claim evidence. |
| `trading-risk-safety` | Enforce hard-veto data/news/session/spread/slippage/cost/margin/exposure/broker/license/entitlement/TTL/idempotency/emergency safety. |
| `mt4-mt5-windows` | Implement/review Go Windows Agent and lightweight MQL4/MQL5 licensing, signed signal stream, IPC, broker mapping, execution guards and signed updates. |
| `control-plane-saas` | Implement/review NestJS IAM/MFA/RBAC, subscriptions/billing, entitlements/licensing, referrals/commissions/payouts, audit and admin with exact financial integrity. |
| `frontend-trading-ui` | Build/review Next.js user/admin/public UI and Live XAUUSD Command Center using real server-authoritative realtime and commercial state. |
| `database-migrations` | Design/review PostgreSQL 17, TimescaleDB, pgvector, PgBouncer and Valkey contracts with canonical migrations, exact decimals and backup/rollback safety. |
| `api-contracts` | Design/review versioned OpenAPI/WebSocket/event contracts with auth, entitlement, idempotency, errors, backward compatibility and resumable delivery. |
| `security-supply-chain` | Perform threat modeling, IAM/session/tenant, secrets, SAST/dependency/SBOM, signing/updater, protocol, financial-abuse and release security. |
| `observability-sre` | Implement/review OpenTelemetry, Prometheus, Grafana, structured logs, SLOs, alerts, backup/restore, DR and failure coverage across trading and finance. |
| `release-gates` | Apply final promotion/release gates requiring test, security, quant, parity, execution, finance, Windows/MT, compliance and rollback evidence. |
| `docs-runbooks` | Maintain architecture/ADR/API/event/strategy/admin/user/MT/incident/backup/DR/release/finance-reconciliation/validation documentation. |
| `broker-execution-qualification` | Qualify each broker/strategy execution class using measured XAUUSD symbol economics, sizing, margin, order behavior, latency, spread/slippage/rejects and locality. |

## Discovery

The agent runtime scans `.hermes/skills`. Invoke explicitly with `$<skill-name>` or allow description-based selection.

Skills operationalize the SOW; they never override it or bypass live-trading, financial, security, signing-key or compliance controls.
