---
name: release-gates
description: Apply final promotion/release gates requiring test, security, quant, parity, execution, finance, Windows/MT, compliance and rollback evidence.
---

# release-gates

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Promotion
research → backtest → OOS → replay → paper → shadow → signals → limited execution. No self-promotion.

## Workflow
1. Collect build/lint/unit/integration/E2E, migrations/backup/restore, security/SBOM/signing evidence.
2. Require deterministic/golden/parity, exit-aware cost backtest, OOS/calibration/sample sufficiency.
3. Require gate latency/fail-closed, delivery reliability, broker qualification, ledger reconciliation, Windows/MT and observability evidence.
4. Verify compliance activation and performance-claim evidence.
5. Mark traceability PASS/FAIL/BLOCKED and issue GO only when all applicable P0 gates pass.

## Validate
Feature parity, exit-aware backtest, gate health, security, finance reconciliation and rollback readiness.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.

## AUTO-PUSH RULE (CRITICAL — ALWAYS FOLLOW)

**After every code change, ALWAYS:**
1. `git add -A`
2. `git commit -m "<descriptive message>"`
3. `git push origin main`

**Never leave uncommitted or unpushed changes.**
**Never ask the user "should I push?" — just push.**
**The remote is: https://github.com/simhaonline/predictatrade.git (main branch)**

## DOCKER-FIRST ARCHITECTURE (CRITICAL — ALWAYS FOLLOW)

**ALL services run in Docker containers via `docker compose`.**
**NEVER use systemd services. ALL systemd services are DISABLED.**

Services (all in docker-compose.yml):
- pat-postgres (TimescaleDB)
- pat-valkey (cache)
- pat-realtime (Go engine)
- pat-control (NestJS)
- pat-frontend (Next.js)
- pat-status (status page)
- pat-nginx (reverse proxy + SSL)
- pat-prometheus (metrics)
- pat-grafana (dashboards)
- pat-ntfy (notifications)

**To restart a service:** `docker compose restart <service>`
**To rebuild a service:** `docker compose build <service> && docker compose up -d <service>`
**To view logs:** `docker compose logs -f <service>`
**To check status:** `docker compose ps`

## BUILD COMMANDS

- **Realtime engine:** `docker compose build realtime && docker compose up -d realtime`
- **Frontend:** `docker compose build frontend && docker compose up -d frontend`
- **Control:** `docker compose build control && docker compose up -d control`
- **Windows Agent:** `./scripts/build-windows-agent.sh --bump` (builds + updates deploy folder)
- **All services:** `docker compose up -d --build`

## TIME ZONE (CRITICAL — ALWAYS FOLLOW)

- Internal time truth is **UTC** — always use `time.Now().UTC()` in Go
- MT5 EA sends `TimeGMT()` in ISO8601 format (`2026-08-21T16:25:11Z`)
- Broker time (UTC+3) is kept as `broker_timestamp` for reference only
- All API responses include `server_time` in RFC3339 UTC
- Frontend displays UTC with clock drift compensation
