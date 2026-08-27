# Macroscopic Audit — Predict-A-Trade

**Audit type:** System-level (codebase + database), not line-by-line linting
**Audit date:** 27 August 2026
**Auditor:** Mehulkumar Bhatt
**Scope:** `/srv/predictatrade/xauusd` — Go realtime engine, NestJS control plane, Next.js frontend, Python research plane, Windows/MQL edge, and the live PostgreSQL/TimescaleDB database.

---

## 1. Executive Summary

Predict-A-Trade is a **modular monolith** split across five planes (Go realtime, NestJS control, Next.js frontend, Python research, Windows/MQL edge) with a single PostgreSQL 17 + TimescaleDB datastore and Valkey cache. The architecture is genuinely well-separated at the plane level, financial math uses `DECIMAL(18,8)` throughout (no floating-point money), and the time-series layer is properly hypertabled with compression + retention policies. CI is comprehensive (vet, race detector, lint, build, pytest, cross-compile, secret scan, npm audit).

The system is **not yet production-safe** for three reasons, in priority order:

1. **Hardcoded production secrets in a tracked file.** `docker-compose.yml` (committed to git) contains a live `JWT_SECRET` and the Postgres `POSTGRES_PASSWORD`. Anyone with repo read access can forge auth tokens and reach the database.
2. **Migration drift between code and database.** 64 migration files exist on disk, but `audit.migration_history` records only 62, and the newest applied is `073`. Migrations `074`–`080` (finance ledger, backtest artifacts, devil-liquidity, payouts idempotency) are on disk and their tables exist in the DB, but are **not** recorded as applied — evidence of two competing migration mechanisms (Docker `initdb.d` auto-run vs. `scripts/migrate.sh`).
3. **God files and duplicated financial logic.** `realtime/cmd/realtime-engine/main.go` is 4,600 lines / 43 functions; `control/src/modules/admin/admin.service.ts` is 995 lines. Duplicate migration numbers (`018`, `019`, `020`, `028`, `062`, `071`, `080`) and duplicate tables (`subscription_events` vs `subscription_event_v3`) indicate ad-hoc evolution without cleanup.

**Overall risk rating: HIGH** — the data-integrity and money-type discipline is strong, but the secret exposure and migration drift are launch-blocking.

---

## 2. System Map

### Languages & runtimes
| Plane | Dir | Language | Runtime | Key deps |
|-------|-----|----------|---------|----------|
| Realtime engine | `realtime/` | Go | 1.25 (go.mod) | pgx/v5, gorilla/websocket, shopspring/decimal, onnxruntime_go, go-redis/v9, zerolog |
| Control plane | `control/` | TypeScript | Node 22 (NestJS 10) | @nestjs/jwt, passport-jwt, bcrypt, class-validator, decimal.js, otplib, helmet, @nestjs/throttler |
| Frontend | `frontend/` | TypeScript | Next.js 16.3 / React 19.2 | TanStack Query/Table/Virtual, recharts, lightweight-charts, sonner, @tabler/icons |
| Research | `research/` | Python | ≥3.11 | numpy, pandas, scipy, scikit-learn |
| Windows/MQL edge | `windows-agent/`, `mql/` | Go + MQL4/5 | Go 1.23 | golang.org/x/sys/windows/svc |

### Architecture pattern
Modular monolith with **five enforced plane boundaries** (documented in `AGENTS.md` / `.hermes.md`):
- **Go Realtime Plane** (`realtime/`, 230 `.go` files): market data, features, strategies, signals, gates, risk, execution — authoritative for trading.
- **NestJS Control Plane** (`control/`): IAM/MFA/RBAC, subscriptions, billing, licensing, referrals, commissions, payouts, audit.
- **Next.js Presentation Plane** (`frontend/`): renders server-authoritative truth only.
- **Python Research Plane** (`research/`): backtesting, calibration, ML — not on the live tick path.
- **Windows/MQL Edge** (`windows-agent/`, `mql/`): execution adapters, no primary intelligence.

### Data stores
- **PostgreSQL 17 + TimescaleDB HA** (`pat-postgres`, port 5432) — single relational + time-series store. 20 schemas, ~188 tables (99 in `trading` alone).
- **Valkey 8.0** (`pat-valkey`, port 6379) — cache (candle cache, signal outbox).

### Communication
- REST (Go `:13081`, NestJS `:13080`), WebSocket (Go `:13081` exec + `:13091` data), direct DB access from both Go and NestJS (shared `DATABASE_URL`), nginx reverse proxy (80/443).

### Third-party integrations
Market data (TwelveData, FMP, FRED, CoT), Ollama (sentiment), ONNX runtime (ML inference), Stripe + NOWPayments (billing), nodemailer (email), ntfy (notifications), MT4/MT5 (broker execution).

### Deployment
Docker Compose (11 services), GitHub Actions CI (`.github/workflows/ci.yml`), nginx reverse proxy.

---

## 3. Findings by Severity

### CRITICAL

**C-1 — Hardcoded production secrets in a tracked file**
- **What:** `docker-compose.yml` (tracked in git) hardcodes `JWT_SECRET=lyFoqwbIPuflF/6PjNtrZXCM1wXGVfYJhj7ZxDAWKYA=` (lines 71, 95) and `POSTGRES_PASSWORD: pat_local_dev_only` (line 24), plus `DATABASE_URL` with the DB password inline.
- **Why it matters:** The JWT secret is the signing key for all auth tokens (shared between Go and NestJS — see `realtime/internal/gateway/websocket.go:96` and `realtime/cmd/live-terminal/main.go:376`). Anyone with repo access can mint valid admin tokens. The DB password is the superuser credential.
- **Remediation:** Move all secrets to `infra/env/*.env` (already gitignored) or Docker secrets; rotate the JWT secret and DB password immediately; add a pre-commit secret scan that also flags `docker-compose.yml`.
- **Note:** `.gitignore` correctly excludes `jwt_secret.txt`, `database_url.txt`, `mcp.env`, and `infra/env/*.env` — but the compose file bypasses that protection.

**C-2 — Migration drift: schema ahead of migration history**
- **What:** `database/migrations/` has 64 `.sql` files. `audit.migration_history` records 62 distinct files, newest = `073_device_activation_unique.sql`. Migrations `074`–`080` (e.g. `075_finance_ledger_entries.sql`, `080_devil_liquidity.sql`, `079_payouts_idempotency.sql`) are **not** in `migration_history`, yet their tables exist (`finance.ledger_entries`, `public.devil_liquidity_*`, `trading.backtest_artifacts`).
- **Why it matters:** Two competing mechanisms — `docker-compose.yml` mounts `./database/migrations` to `/docker-entrypoint-initdb.d:ro` (runs every `.sql` alphabetically on first init), while `scripts/migrate.sh` tracks state in `audit.migration_history`. A fresh environment and a migrated environment will diverge; rollback and auditability are broken.
- **Remediation:** Pick one mechanism. Either make `migrate.sh` the single source of truth (and stop auto-running migrations via `initdb.d`), or reconcile `migration_history` to match the on-disk set. Add a CI check that fails when `migration_history` ≠ on-disk files.

**C-3 — Duplicate migration sequence numbers**
- **What:** Seven numeric prefixes are used twice: `018`, `019`, `020`, `028`, `062`, `071`, `080` (e.g. `018_regime_telemetry_shadow_signals.sql` and `018_slippage_capital_protection.sql`; `080_devil_liquidity.sql` and `080_signal_quality_diagnostics.sql`).
- **Why it matters:** Ordering is non-deterministic (alphabetical tie-break), and the migration tracker keys on filename, so two files can silently shadow each other or apply in an unintended order.
- **Remediation:** Renumber to strictly monotonic unique prefixes; enforce uniqueness in CI.

### HIGH

**H-1 — God file: `realtime/cmd/realtime-engine/main.go` (4,600 lines, 43 functions)**
- **What:** The engine entrypoint contains 43 top-level functions spanning wiring, loops, and business logic (e.g. `runPnLAnchorLoop` at line 3899).
- **Why it matters:** Highest-risk file in the system (live trading path) is also the least maintainable; changes here are hard to review and test in isolation.
- **Remediation:** Extract into `internal/` packages (wiring, loops, lifecycle); keep `main.go` to composition only.

**H-2 — God service: `control/src/modules/admin/admin.service.ts` (995 lines)**
- **What:** Single NestJS service doing disproportionate work.
- **Why it matters:** Admin surface (users, billing, payouts, operations) is a security-sensitive boundary; a 995-line service is hard to audit for authorization gaps.
- **Remediation:** Split into per-domain services with explicit guards.

**H-3 — Duplicate/legacy tables not cleaned up**
- **What:** `billing.subscription_events` and `billing.subscription_event_v3` coexist (the `_v3` suffix implies the old one was superseded but not dropped). Similar risk in `trading` (99 tables).
- **Why it matters:** Two tables for one concept invites writes to the wrong one and inconsistent reads.
- **Remediation:** Confirm `subscription_events` is dead, then drop it (or add a deprecation migration).

**H-4 — ML models are placeholder-sized**
- **What:** `models/lstm_model.onnx` (711 bytes) and `models/xgb_model.onnx` (710 bytes) are committed. A real LSTM/XGBoost model is hundreds of KB to MB.
- **Why it matters:** If the engine loads these as the "calibrated" model, predictions are effectively untrained. The frontend already labels probability as "Pending until calibration model is validated" (`signals/page.tsx:231`), which is honest — but the committed artifacts could be mistaken for real models.
- **Remediation:** Either train and commit real models, or remove the placeholder artifacts and gate the ML path behind an explicit "model not trained" flag.

**H-5 — Research plane is thin relative to claims**
- **What:** `research/` contains 3 test files and 2 scripts (`import_xauusd_historical.py`, `strategy_edge_calibration.py`). The docs describe backtesting, walk-forward, and ML calibration as first-class capabilities.
- **Why it matters:** The "quant validation" story is largely in `scripts/` (e.g. `quant_validation.py`, `walk_forward_baseline.py`, `oos_walkforward_calibrate.py`) rather than the research package, and there's no committed evidence of a reproducible model-training pipeline.
- **Remediation:** Consolidate the scripts into the research package with a documented, reproducible entrypoint.

### MEDIUM

**M-1 — Two migration mechanisms (see C-2) also means no rollback story.** `migrate.sh` has no down/rollback path; migrations are forward-only.

**M-2 — `control` vs `billing` schema split for plans/subscriptions.** `control.plans` / `control.plan_entitlements` hold plan definitions while `billing.subscriptions` holds subscriptions. This is defensible (catalog vs. ledger) but the boundary is implicit — worth documenting to prevent future drift.

**M-3 — High sequential-scan counts on small tables.** `billing.subscriptions` (54,045 seq scans vs 13 idx scans, 5 rows) and `licensing.licenses` (52,561 vs 12, 15 rows). Not a perf problem at current scale (tables are tiny), but the ratio suggests queries aren't using the indexes that exist (`idx_subscriptions_user`, `idx_subscriptions_status`, etc.). Worth a query-plan review before scale-up.

**M-4 — `trading` schema has 99 tables.** Breadth is a maintainability and cognitive-load risk; several appear to be per-feature audit/telemetry tables. Consider consolidating or documenting the table taxonomy.

**M-5 — `main.go` uses `queueMicrotask`/`rafBatch` patterns in the frontend but the Go engine has no equivalent backpressure story visible in the entrypoint** — inferred, not verified; needs confirmation from the team on how the engine handles slow consumers.

### LOW

**L-1 — `error.log` (11.9 KB) and `ui_audit.md` (72 KB) sit at repo root.** `error.log` is gitignored (`*.log`) but `ui_audit.md` is a stray artifact that duplicates the audit report now in `docs/reports/`.

**L-2 — `database_url.txt` and `jwt_secret.txt` exist on disk (root-owned, gitignored).** Correctly excluded, but their presence alongside a tracked `docker-compose.yml` that also contains the secret is redundant and confusing.

**L-3 — `models/` and `data/` are root-owned** while the rest of the repo is `hermes`-owned — minor permission inconsistency that can trip up tooling.

---

## 4. Database-Specific Findings

### Schema design — GOOD
- **Money types are correct.** All monetary columns use `DECIMAL(18,8)` (prices, amounts, balances, commissions, P&L) or `DECIMAL(8,4)` (rates/multipliers). **No `FLOAT`/`DOUBLE PRECISION`/`REAL` in any trading table** (verified across `005_trading_market_tables.sql`). This is the single most important correctness decision for a trading system and it is done right.
- **Timestamps** use `TIMESTAMPTZ` (UTC) per the migration header comment in `004_referral_commission_payout.sql`.
- **Referential integrity is enforced at the DB level:** 143 foreign-key constraints across the domain schemas, plus UNIQUE and CHECK constraints (e.g. `trading.signals` has 3 CHECK + 3 FK; `trading.positions` has 1 CHECK + 3 FK).

### Time-series — GOOD
- **21 hypertables** (TimescaleDB), including `candles` (1,865 chunks), `ticks` (165 chunks), `signals`, `indicator_history`, `strategy_evaluations`.
- **Compression + retention policies are configured** (e.g. `ticks`: compress after 7 days, drop after 90 days; `candles`: compress after 30 days; `signals`: compress 30d / drop 180d). This avoids unbounded row growth.

### Integrity gaps
- **Migration drift (C-2/C-3)** is the primary integrity risk — the schema is ahead of the recorded migration state.
- **No evidence of orphan/duplicate data** at current scale (largest tables: `risk_decisions` 61k rows, `signal_candidates` 32k rows — all small). A deeper orphan-scan (FK-violation check) is a recommended follow-up but not run here.

### Performance
- Indexes exist on hot paths (`idx_subscriptions_user`, `idx_subscriptions_status`, etc.), but seq-scan ratios (M-3) suggest under-utilization. At 5–15 rows per table this is immaterial today; it becomes material at production scale.

### Security & compliance
- **Encryption at rest / in transit:** not verifiable from static inspection — needs confirmation (TLS termination at nginx is present; DB encryption at rest is an open question).
- **Least-privilege DB roles:** `database/roles/` exists and is mounted into the container, suggesting role separation — but the app connects as `pat_admin` (superuser-level) via `DATABASE_URL`. **This is a least-privilege gap:** the application should use a non-superuser role.
- **Backup:** `scripts/backup/` and `scripts/backup_restore_validate.py` exist; whether restore has been tested in production is an open question.

---

## 5. Technical Debt Inventory (prioritized by risk-reduction-per-effort)

1. **Rotate secrets + move out of `docker-compose.yml`** (C-1) — highest risk, low effort.
2. **Reconcile migration mechanism** (C-2) — pick `migrate.sh` as source of truth, stop `initdb.d` auto-run.
3. **Renumber duplicate migrations** (C-3) — mechanical, low effort.
4. **Split `main.go` and `admin.service.ts`** (H-1, H-2) — high effort, high long-term value.
5. **Drop/consolidate legacy tables** (`subscription_events` vs `_v3`) (H-3).
6. **Resolve ML model placeholders** (H-4) — either train real models or gate the path.
7. **Consolidate research scripts into the package** (H-5).
8. **Switch app DB role to least-privilege** (DB security).

---

## 6. Quick Wins

1. Add `docker-compose.yml` to the CI secret scan (currently only scans `.ts/.go/.py/.tsx/.yml/.json` for specific patterns — extend to flag `JWT_SECRET=` and `POSTGRES_PASSWORD=` in compose).
2. Add a CI check: `migration_history` filenames must equal `database/migrations/*.sql` filenames.
3. Add a CI check: migration numeric prefixes must be unique.
4. Delete the stray `ui_audit.md` at repo root (superseded by `docs/reports/UI_UX_AUDIT_REPORT.md`).
5. Add a `down`/rollback note to `migrate.sh` (or document that migrations are forward-only).
6. Document the `control` (catalog) vs `billing` (ledger) schema boundary in `docs/database/DATABASE_ARCHITECTURE.md`.

---

## 7. Open Questions

1. **Is DB encryption at rest enabled?** (not verifiable from static inspection).
2. **Has backup/restore ever been exercised in production?** (`backup_restore_validate.py` exists; execution history unknown).
3. **Are the ONNX models in `models/` actually loaded by the engine, or are they dead artifacts?** (needs confirmation from `realtime/pkg/mlengine`).
4. **Does the Go engine have backpressure for slow WebSocket consumers?** (inferred gap, not verified).
5. **Which migration mechanism is authoritative in production** — `initdb.d` auto-run or `migrate.sh`? (determines the C-2 fix).
6. **Is `billing.subscription_events` still written anywhere, or fully superseded by `subscription_event_v3`?** (determines whether it's safe to drop).
7. **Load testing** — the seq-scan ratios and connection-pool sizing need a load test before production scale; none was observed in the repo.

---

*This audit is based on static inspection of the repository and read-only queries against the live database. Findings marked "inferred" or listed under Open Questions were not fully verifiable from static analysis alone.*
