# AGENTS.md — Predict-A-Trade v1.0.0

## Authority

Canonical implementation contract: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md`.

This file operationalizes Codex behavior; it never replaces or weakens the SOW. If anything conflicts, the SOW wins.

## Mission

Audit the existing repository first, preserve working behavior and data, reuse/extend before replacing, implement the complete v1.0.0 backend/frontend/realtime/research/data/Windows/MT4/MT5/licensing/subscription/referral/commission/payout/dashboard/security/observability/release scope, and finish with evidence-backed traceability plus a production GO/NO-GO report.

Never call the project complete merely because code compiles, UI renders, or a partial test set passes.

## Architecture Boundaries

1. **Go — Real-Time Trading Plane**: authoritative market-data, feature, strategy, signal, hard-risk, execution-authorization, realtime delivery and reconciliation path. No synchronous billing/referral/commission/payout dependency in tick-to-signal processing.
2. **Python — Intelligence/Research Plane**: datasets, research, backtesting, walk-forward/OOS, calibration, ML/NLP/vision research, feature studies, drift and validation. Python must not become a mandatory dependency for every live tick decision.
3. **NestJS — SaaS/Control Plane**: IAM/MFA/RBAC, tenants, users, subscriptions, billing/webhooks, entitlements, licensing, devices, MT accounts, referrals, commissions, payouts, audit, config and admin operations.
4. **Next.js — Presentation Plane**: public site, user portal, admin operations console, XAUUSD Live Command Center, charting, licensing/downloads and growth/financial UI. It renders server-authoritative truth and never becomes the authority for risk, strategy, entitlement or finance.
5. **Windows/MQL Edge**: Go Windows Agent + MQL4/MQL5 remain lightweight execution adapters/guards. No primary predictive intelligence or server/private signing credentials in EAs.

## Non-Regression and Safety Precedence

The following always outrank convenience, AI output, score, entitlement, trade frequency and UI requests:

- data quality/freshness;
- hard risk vetoes;
- news/session restrictions;
- spread/slippage/total-cost limits;
- margin/exposure/account restrictions;
- broker specification/execution constraints;
- license/entitlement/device/account permissions;
- signal TTL/replay/idempotency;
- emergency stop;
- financial-ledger correctness;
- security/privacy/compliance.

`NO-TRADE` is a first-class valid result. Never force a trade to meet a frequency target.

Do not fabricate live ticks, fills, orders, P&L, liquidity/order-flow capability, CVD/DOM/aggressor-side/depth data, referral registrations, commissions, payouts, confidence/probability, time-to-target, performance claims or AI activity. Demo/replay/sandbox data must be unmistakably labeled and unable to mutate live trading or real finance.

## Production Mutation Boundary

Without explicit operator authorization outside normal coding work, do not:

- enable live automated trading;
- place or close live broker orders/positions;
- mutate real subscriptions, commissions, wallets or payouts;
- run destructive production migrations;
- rotate production signing keys;
- export production secrets;
- grant tools/agents database superuser;
- change production DNS;
- publish unsupported performance/profitability/accuracy claims.

Use least-privilege local/test/staging/paper/shadow fixtures and credentials.

## Required Work Loop

For every implementation slice:

`READ → MAP REQUIREMENT → DESIGN MINIMUM COMPATIBLE CHANGE → WRITE/UPDATE TEST → IMPLEMENT → LINT/FORMAT → UNIT TEST → INTEGRATION TEST → SECURITY/BOUNDARY CHECK → DOCUMENT → REPORT EVIDENCE`

Before editing:
1. identify applicable SOW sections;
2. locate the existing implementation, tests, migrations and runtime wiring;
3. classify relevant components as `REUSE`, `EXTEND`, `ADAPT`, `REPLACE_WITH_JUSTIFICATION`, `NEW` or `DEPRECATE`;
4. identify risk/security/data/financial boundaries;
5. choose the smallest compatible change.

After editing, run narrow tests first, then broader repository-standard checks, update contracts/docs/observability/migrations where applicable, and record rollback plus unresolved evidence.

## Repository Audit First

At the start of major work, invoke the `repo-audit` skill and use `repo_explorer` for read-heavy mapping. Produce/update:

- repository/service/dependency map;
- runtime/data-flow map;
- exact build/lint/test/run/migration commands;
- DB/migration inventory;
- API/WebSocket/event schema inventory;
- deployment/service inventory;
- duplicate/legacy/dead-code findings;
- SOW traceability matrix;
- change-impact map.

Do not start a greenfield rewrite when a compatible subsystem exists.

## Required Skills

Use repository skills under `.agents/skills/`:

`repo-audit`, `architecture-guardrails`, `xauusd-market-data`, `xauusd-strategy-spec`, `xauusd-quant-validation`, `trading-risk-safety`, `mt4-mt5-windows`, `control-plane-saas`, `frontend-trading-ui`, `database-migrations`, `api-contracts`, `security-supply-chain`, `observability-sre`, `release-gates`, `docs-runbooks`, `broker-execution-qualification`.

Skills are procedures, not permission to bypass the SOW.

## Required Subagents

Use project agents under `.codex/agents/` for bounded work:

`repo_explorer`, `platform_architect`, `go_realtime_engineer`, `python_quant_researcher`, `nestjs_control_engineer`, `nextjs_frontend_engineer`, `database_engineer`, `windows_mql_engineer`, `quant_validator`, `security_reviewer`, `qa_reliability_engineer`, `release_manager`, `broker_execution_validator`.

Coordination:
- parent Codex session owns the integrated plan and resolves conflicts;
- parallelize read-heavy exploration/review; avoid overlapping parallel writes;
- independent reviewers do not silently modify what they review;
- every subagent returns findings, files examined, files changed, tests run, unresolved risks and recommended next action.

## Strategy and Quantitative Integrity

The four strategy products must remain distinct versioned behavior:

- `STANDARD_SCALPING`
- `ULTRA_SCALPING`
- `STANDARD_SWING`
- `TREND_SWING`

All strategy-critical thresholds and profiles are configuration-backed, versioned, auditable and historically reproducible. Raw score is not probability. Subscriber-facing probability must be calibrated to a named prediction target and active exit profile.

Trading changes require applicable deterministic/golden fixtures, realistic bid/ask + cost-aware backtests, walk-forward/OOS, calibration/sample sufficiency, replay, paper/shadow and broker/execution qualification. Models/optimizers cannot self-promote.

## Market-Data Truth

Preserve provenance and capability semantics. Never equate broker tick volume with centralized XAUUSD volume, GC flow with spot truth, or estimated/derived flow with authoritative exchange flow. Required capability absence must degrade quality or cause `NO-TRADE`. Futures roll/basis and session/fix/holiday logic must be explicit and timezone/DST-aware. Internal time truth is UTC.

## Risk and Execution

Hard gates are deterministic, freshness/version stamped and fail closed.

### Server-Side SL Enforcement (v1.15.0)

The backend is the ENFORCEMENT authority for S/L and TP, not just the calculation authority:

- **EXECUTION_ACK verification**: Server verifies SL > 0 and SL matches server-sent value (±0.5 points).
- **Position SL monitoring**: Server scans broker snapshot for PAT positions with missing SL → sends CLOSE_POSITION.
- **CLOSE_POSITION command**: Server → Windows Agent → EA — closes individual position by ticket/magic.
- **EMERGENCY_STOP command**: Server → Windows Agent → EA — closes ALL PAT positions + halts trading.
- **KILL_SWITCH command**: Server → Windows Agent → EA — closes all + ExpertRemove() + agent disconnect.
- **Agent suspension**: 3 SL violations → agent disconnected, no future signals. Other agents unaffected.
- **Signal delivery NOT blocked**: Suspension works through disconnection, not broadcast filtering. Mandatory per-candidate gates must not perform synchronous external I/O. Validate aggregate XAUUSD exposure, net R:R, total cost-to-target, exit geometry, margin/headroom, broker stop-out, news/session/rollover restrictions, TTL/replay/idempotency and emergency stops.

Execution math uses validated broker profile data: symbol mapping, digits, tick size/value, contract economics, volume min/max/step, stop/freeze levels, fill/order modes, margin/stop-out, sessions/rollover and swaps/carry. Model missed/partial fills, rejects, latency, jitter, spread and slippage. Do not assume a universal XAUUSD pip definition or 100-ounce contract.

## Financial Integrity

Subscription/referral/commission/payout truth belongs in the control/financial domain and must remain isolated from the Go trading hot path. Financial operations are exact-decimal, transactional, idempotent, auditable and ledger-backed. Use compensating/reversal records rather than rewriting financial history. Commission derives only from canonical eligible validated revenue policy. Frontend renders backend ledger/policy outcomes; it does not recode them.

## API / Event / Realtime Contracts

Use explicit versions, authentication/authorization, tenant ownership, plan/strategy entitlement, idempotency for mutation/execution, stable machine-readable errors, correlation IDs and compatibility/deprecation policy.

Realtime event envelopes preserve event/stream IDs, sequence, schema version, timestamps, provenance and quality. P0/P1 events may not be silently dropped. UI may format and visualize, but must not recompute production indicators, structure, probability, risk, execution eligibility, commission or payout truth.

## Database and Migration Discipline

Inspect current schema/migration history first. Use canonical forward migrations, compatibility windows, migration tests, constraints/indexes, exact-decimal financial types, least-privilege roles and backup-aware rollback/forward-fix plans. Never rewrite applied migration history. Valkey is hot/cache state, not sole durable financial/trading truth.

## Security and Supply Chain

Apply threat modeling, IAM/MFA/RBAC/tenant isolation, secret scanning, SAST/dependency checks, SBOM, signed releases/client artifacts, key separation/rotation, secure updater manifests, replay/idempotency protections, input validation, rate/abuse controls and audit logging. MCP/plugins must not receive unrestricted production broker/payment/payout/signing-key/DB-superuser/secret-vault access.

## UI / Command Center

Implement server-authoritative `MARKET`, `TRADING`, `GROWTH` and `COMMAND_CENTER` modes with honest loading/empty/stale/degraded/error/demo/replay states. Preserve live XAUUSD chart/overlays, indicator/structure/liquidity/evidence, signal/trade-readiness, execution/position, referral network, commission/payout/plan analytics, light/dark tokens, responsive/accessibility/reduced-motion and 4K/full-screen requirements. Admin remains an operations/business console, not a duplicate trader terminal.

## Observability and Reliability

Canonical stack: OpenTelemetry + Prometheus + Grafana + structured JSON logs. Instrument market/data quality, processing/signal/risk/delivery latency, gate p50/p95/p99, WS reconnect/resume/resync, broker execution quality, calibration/drift/feature parity, financial reconciliation, Windows/MT heartbeats, API/DB/PgBouncer/Valkey health, backup/restore and DR.

## Documentation and Primary Sources

For version-sensitive APIs use authoritative primary documentation for Next.js, NestJS, Go, Python, PostgreSQL, TimescaleDB, pgvector, Valkey, MetaTrader/MQL, market-data providers and payment providers. Match the repository's installed versions. Maintain architecture/ADRs, APIs/events, strategy playbooks, admin/user manuals, MT setup, incident/emergency, backup/DR, release/rollback, finance reconciliation and validation reports.

## Definition of Done

Applicable completion evidence includes implementation, migration, backward compatibility, unit, integration, E2E, deterministic/golden/parity, replay, load/performance, chaos/failure, security, observability, documentation, backup/restore, rollback and acceptance criteria. Resolve implementable failures; do not re-label them as limitations merely to finish.

## Reporting

After each phase report: phase, SOW requirements, files changed, migrations, exact test results, security checks, performance checks, quant validation, compatibility impact, blockers, rollback and next phase.

Final traceability columns:

`SOW requirement | implementation files | tests | migrations | API/UI | observability | status | evidence`

Final status is `PASS`, `PARTIAL` or `BLOCKED`. `PASS` is forbidden while any applicable mandatory P0 safety, security, financial-integrity, execution, data-quality or acceptance gate is failed or unverified.

## Project Control Files

- `AGENTS.md` — Codex-native repository instructions.
- `AGENT.md` — compatibility pointer only.
- `SKILLS.md` — human-readable index.
- `.agents/skills/*/SKILL.md` — actual Codex skills.
- `.codex/config.toml` — Codex-native project/subagent/MCP config.
- `.codex/agents/*.toml` — actual project subagents.
- `.mcp.json` — portability/compatibility representation; Codex-native MCP truth is `.codex/config.toml`.

## Auto-Push Rule (CRITICAL — ALWAYS FOLLOW)

**After every code change, ALWAYS:**
1. `git add -A`
2. `git commit -m "<descriptive message>"`
3. `git push origin main`

**Never leave uncommitted or unpushed changes.**
**Never ask the user "should I push?" — just push.**
**Remote: https://github.com/simhaonline/predictatrade.git (main branch)**

## Docker-First Architecture (CRITICAL — ALWAYS FOLLOW)

**ALL services run in Docker containers via `docker compose`.**
**NEVER use systemd services. ALL systemd services are DISABLED.**

Services (all in docker-compose.yml):
- pat-postgres (TimescaleDB) — pat-valkey (cache) — pat-realtime (Go engine)
- pat-control (NestJS) — pat-frontend (Next.js) — pat-status (status page)
- pat-nginx (reverse proxy + SSL) — pat-prometheus (metrics)
- pat-grafana (dashboards) — pat-ntfy (notifications)

**Restart:** `docker compose restart <service>`
**Rebuild:** `docker compose build <service> && docker compose up -d <service>`
**Logs:** `docker compose logs -f <service>`
**Status:** `docker compose ps`

## Build Commands

- **Realtime:** `docker compose build realtime && docker compose up -d realtime`
- **Frontend:** `docker compose build frontend && docker compose up -d frontend`
- **Control:** `docker compose build control && docker compose up -d control`
- **Windows Agent:** `./scripts/build-windows-agent.sh --bump`
- **All:** `docker compose up -d --build`
