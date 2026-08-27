# Macroscopic Audit — Predict-A-Trade

**Audit type:** System-level (codebase + database + pipeline + compliance), read-only
**Audit date:** 28 August 2026
**Auditor:** IT & Compliance Auditor (skills: `repo-audit`, `compliance-legal`, `api-security-audit`, `security-supply-chain`, `financial-ledger-audit`, `nestjs-service-audit`, `database-migrations`, `trading-risk-safety`, `broker-execution-qualification`, `mt4-mt5-windows`, `signal-reconciliation`)
**Scope:** `/srv/predictatrade/xauusd` — Go realtime engine, NestJS control plane, Next.js frontend, Python research plane, Windows/MQL edge, PostgreSQL 17 + TimescaleDB, Valkey, and the full market→signal→delivery→broker and auth→subscription→billing→payout pipelines.
**Supersedes:** `docs/reports/MACROSCOPIC_AUDIT_REPORT.md` (27 Aug 2026) — prior criticals re-verified and **still open / worsened**.

---

## 0. GO / NO-GO Verdict

**NO-GO for production launch.**

The platform's trading-core discipline is strong (server-side SL/TP authority, fail-closed hard-risk gates, exact-decimal money in the DB, honest classed NO-TRADE reasons, real OHLC forwarding, live calibration loaded). **But it cannot be released** because of launch-blocking compliance and data-integrity defects that were identified in the prior audit and **remain unremediated — and the database migration drift has worsened**:

- **Tracked live secrets** (JWT signing key + DB superuser password) are still committed in `docker-compose.yml` in git.
- **Migration state is corrupt and actively diverging** (64 files on disk vs 62 in `audit.migration_history`; 15 unrecorded; 13 history rows point to files that no longer exist).
- **The engine still emits a fabricated/PROVISIONAL probability** to subscribers (violates AGENTS.md: "Never fabricate confidence/probability").
- **A critical `tar` path-traversal CVE** sits in the control dependency tree behind a non-blocking `npm audit || true` gate.

These are independently sufficient to block go-live.

**Overall risk rating: HIGH.** Per-area: Backend HIGH (PARTIAL), Control HIGH, Frontend LOW, Database **CRITICAL**, CI/CD MEDIUM (secret-scan gap HIGH), Security/Compliance **HIGH**.

---

## 1. System Map (verified)

| Plane | Dir | Lang | Key deps |
|-------|-----|------|----------|
| Realtime engine | `realtime/` | Go 1.25 | pgx/v5, gorilla/websocket, shopspring/decimal, redis/v9, zerolog, onnxruntime |
| Control plane | `control/` | TS / Node 22 / NestJS 10 | @nestjs/jwt, passport-jwt, bcrypt, decimal.js, otplib, helmet, throttler |
| Frontend | `frontend/` | TS / Next.js 16.3 / React 19 | TanStack, recharts, lightweight-charts |
| Research | `research/` | Python ≥3.11 | numpy, pandas, scipy, scikit-learn |
| Windows/MQL edge | `windows-agent/`, `mql/` | Go 1.23 + MQL4/5 | x/sys/windows/svc |

Single PostgreSQL 17 + TimescaleDB (`pat-postgres`) + Valkey (`pat-valkey`). 11 docker-compose services; nginx edge. Go engine authoritative for trading; NestJS for SaaS/finance; Next.js renders server-authoritative truth only.

---

## 2. Findings — Severity Index

| ID | Area | Finding | Severity | Evidence |
|----|------|---------|----------|----------|
| SEC-1 | Sec | Live `JWT_SECRET` + `POSTGRES_PASSWORD` (superuser) committed in tracked `docker-compose.yml` | **CRITICAL** | `docker-compose.yml:24,69,71,92,95,149,290` (secret values redacted in this report) |
| DB-1 | DB | Migration drift **worsened**: 64 files vs 62 history; 15 unrecorded; 13 history rows reference deleted files | **CRITICAL** | `ls database/migrations/*.sql`=64; `SELECT count(*) FROM audit.migration_history`=62 |
| DB-2 | DB | Duplicate migration prefixes: 018,019,020,028,062,071,080 | **CRITICAL** | `ls database/migrations/*.sql` |
| BE-5 | Comp | Engine emits **fabricated/PROVISIONAL probability** to subscribers | **CRITICAL** | `realtime/internal/.../consumer.go:67-71,87-112`; `frontend/.../signal-evidence.tsx:126,145` |
| SUP-1 | Supply | 29 npm vulns in `control` incl. critical `tar` path traversal | **CRITICAL** | `control/package.json` tree |
| SEC-3 | Sec | Seeded test accounts w/ known password `Demo@1234` via migration 029 | HIGH | `database/migrations/029_*.sql` |
| CI-1 / SUP-3 | CI | Secret scan misses `docker-compose.yml` (`JWT_SECRET=`/colon `POSTGRES_PASSWORD:`) | HIGH | `.github/workflows/ci.yml:80-93` |
| SUP-2 | CI | `npm audit --audit-level=high || true` never fails build | HIGH | `ci.yml:96-97` |
| PII-1 | Comp | No GDPR consent/erasure/retention workflow evident | HIGH | `compliance.service.ts` (no matches); `audit.client_events` INET w/o retention |
| BE-4 | Risk | **News hard-gate fails OPEN** on feed outage (removes mandated protection) | HIGH | `realtime/internal/gates/implementations.go:90-100` |
| BE-1 | Maint | `realtime-engine/main.go` = **4,756 lines / 43+ fns** holds live-execution safety logic | HIGH | `realtime/cmd/realtime-engine/main.go` |
| CP-2 | Fin | Float `Number()` money math in `payouts.service.ts` & `commissions.service.ts` | HIGH | `control/src/modules/payouts/payouts.service.ts:35,79,96,228`; `commissions.service.ts:380-396` |
| CP-3 | Auth | Coarse binary RBAC (admin-only guard); no tenant/resource roles | MEDIUM | `control/src/common/guards/admin.guard.ts:10` |
| AUTH-1 | Auth | MFA opt-in; not enforced for ADMIN/operators | MEDIUM | `control/src/.../auth.service.ts:213-225` |
| SEC-2 | Sec | Live GitHub PAT plaintext in `mcp.env` (gitignored but broad scope) | MEDIUM | `mcp.env:2` |
| FE-3 | FE | Default API base hits `:13080` directly (bypasses nginx TLS/auth); WS port mismatch | MEDIUM | `frontend/src/lib/axios-instance.ts:9`; `.env.example:1-2` |
| CP-4 | DB | Duplicate live `billing.subscription_events` vs unused `_event_v3` | MEDIUM | `subscriptions.service.ts:178` |
| BE-2 | Risk | `SL_MISMATCH` records but does NOT close position | MEDIUM | `main.go:1702-1711` |
| BE-6 | Pipe | No full signal↔broker-fill reconciliation / ACK timeout | MEDIUM | `reconciliation/reconciler.go`; `main.go:1644-1723` |
| CI-3 | CI | Frontend `jest`/Playwright e2e not run in CI | MEDIUM | `ci.yml:43-49` |
| DB-5 | DB | Dual migration mechanism (`initdb.d` auto-run vs `migrate.sh`) | HIGH | `docker-compose.yml:32` |
| DB-6 | CI | No CI migration-reconciliation check | MEDIUM | `ci.yml` |
| PII-2 | Comp | PII encryption-at-rest unverified | MEDIUM | migrations |
| AUTH-2 | Auth | No global deny-by-default `APP_GUARD` | LOW | `app.module.ts` |
| BE-3 | Risk | SL-match tolerance hardcoded `0.5` (not digits-aware) | LOW | `main.go:1700` |
| BE-9 | Comp | Calibration promotion by sample count only (not AUC/Brier) | LOW | `loader.go:113` |
| SEC-4 | Sec | Secret duplication across `infra/env/*.env` + compose | LOW | `infra/env/realtime.env:132` |
| AUD-1 | Comp | Audit coverage depends on decorator placement | LOW | `compliance.interceptor.ts` |
| FIN-1 | Fin | JS `number` at Stripe boundary (acceptable but unvalidated) | LOW | `stripe.service.ts` |

---

## 3. Launch-Blocking Issues (must fix before GO)

1. **SEC-1 — Tracked live secrets.** Rotate `JWT_SECRET` and `POSTGRES_PASSWORD` immediately (current values are redacted in this report but present in git history / `docker-compose.yml`); remove from `docker-compose.yml`; inject via `infra/env/*.env` (already gitignored) or Docker secrets; add `docker-compose.yml` + `.env*` to CI secret scan. *Unremediated since prior audit.*
2. **DB-1/DB-2/DB-5 — Corrupt, diverging migration state.** Rebuild `audit.migration_history` from a renamed, unique, monotonic migration set; abolish the `initdb.d` auto-run; make `scripts/migrate.sh` the single source of truth; add a CI guard that fails when history ≠ disk or prefixes collide. *Worsened vs prior audit (drift grew ~7 migrations in 24h).*
3. **BE-5 — Fabricated probability.** Gate the displayed `CalibratedProbability` on a `VALIDATED` model/`ProbabilityCalibrated` flag; remove the synthetic `score/200+0.25` fallback in `Calibrate()`; never surface PROVISIONAL models to subscribers. *Direct AGENTS.md violation.*
4. **SUP-1 + SUP-2 — Critical CVE behind a non-blocking gate.** Run `npm audit` as a hard gate; patch/override the `tar` path-traversal CVE; add `govulncheck`/SAST (Trivy/Gitleaks) to CI.
5. **BE-4 — News gate fails open.** Treat news-data-unavailable as fail-closed (or require a recent successful sync within TTL) so a feed outage cannot silently remove news protection.

---

## 4. Status of Prior Audit's Critical Findings (27 Aug)

| Prior | Status 28 Aug | Note |
|-------|--------------|------|
| C-1 hardcoded secrets | **OPEN / UNREMEDIATED** | still in `docker-compose.yml` |
| C-2 migration drift (64 vs 62) | **OPEN / WORSENED** | now 15 unrecorded + 13 orphan history rows |
| C-3 duplicate prefixes | **OPEN / UNREMEDIATED** | 018,019,020,028,062,071,080 |

---

## 5. What Is Working (PASS — do not regress)

- **Build/runtime:** `go build ./...` clean; `pat-realtime` healthy; v1.0.0 calibration loaded; `LiveCalibrator` monotonic + sample-count precedence prevents clobber.
- **Signal pipeline integrity:** every gate veto is classed + persisted (`RejectionGate`, `ReasonCodes`, audit `LogSignal` for ALL); proven-negative candidates downgraded to explicit `NO_TRADE`; `Executable` flag set; **no silent drops**.
- **Server-side SL/TP authority:** `EXECUTION_ACK` verifies `SL>0` + SL-match; `checkPositionSLs`→`CLOSE_POSITION`; `EMERGENCY_STOP`/`KILL_SWITCH` present; 3 SL violations → agent suspend. Recent fixes (`VolatilityScale`, `MinSLSpreadMult`, `enforceSLDirection`) correctly widen stops to dominate spread and correct wrong-side SL.
- **Financial data types:** `DECIMAL(18,8)`/`DECIMAL(10,4)` everywhere in DB; no FLOAT/DOUBLE money columns (grep false positives only).
- **Frontend discipline:** renders server truth only; derives R:R from server prices; no client-side indicator/risk recomputation; no hardcoded secrets in `src`.
- **Audit logging:** `ComplianceInterceptor` auto-writes `audit.client_events` (TimescaleDB hypertable) for `@ComplianceLog` endpoints; financial endpoints require `AdminGuard`; Stripe webhook verified by signature.
- **Reconciliation (partial):** signal delivery/ack tracked; engine↔EA SL verification present.

---

## 6. Remediation Roadmap (suggested order)

1. **Immediate (hours):** rotate + un-track secrets (SEC-1); CI secret-scan + hard `npm audit` (CI-1/SUP-2/SUP-3); remove synthetic probability path + surface `ProbabilityCalibrated` (BE-5).
2. **Short (1–3 days):** freeze migrations, rename to unique monotonic prefixes, rebuild `audit.migration_history`, kill `initdb.d` (DB-1/2/5/6); convert `payouts`/`commissions` to `decimal.js` (CP-2); make news gate fail-closed (BE-4).
3. **Near (1–2 weeks):** fine-grained RBAC + tenant isolation (CP-3); MFA enforcement for privileged roles (AUTH-1); GDPR erasure/retention (PII-1); full signal↔fill reconciliation + ACK timeout (BE-6); extract execution safety from god file (BE-1); seed-user gating (SEC-3); SAST/SCA in CI.

---

## 7. SOW Traceability (condensed)

| SOW requirement | Implementation | Tests/Checks | Status | Evidence |
|-----------------|----------------|--------------|--------|----------|
| Hard-risk fail-closed gates | `realtime/internal/gates` | gate unit tests | PARTIAL | BE-4 news fails open |
| Server-side SL/TP enforcement | `strategies.go computeEntrySLTP` | ack/sl-scan | PASS | BE-1 god-file risk |
| No fabricated probability | `calibration/consumer.go` | — | **FAIL** | BE-5 |
| Financial exact-decimal | DB `DECIMAL(18,8)`; `decimal.js` (partial) | — | PARTIAL | CP-2 float in payouts |
| Audit logging | `compliance.interceptor.ts` | — | PARTIAL | AUD-1 coverage |
| Migration integrity | `scripts/migrate.sh` + `audit.migration_history` | — | **FAIL** | DB-1/2/5 |
| Secret management | `docker-compose.yml` | CI scan | **FAIL** | SEC-1/CI-1 |
| Tenant isolation / RBAC | `admin.guard.ts` | — | PARTIAL | CP-3 |
| Reconciliation | `reconciler.go` | — | PARTIAL | BE-6 |

**Status legend:** PASS = verified; PARTIAL = working with known gaps; FAIL = launch-blocking defect.

---

*Prepared read-only; no source files modified during audit. The engine signal/SL/TP hardening delivered earlier in this session (commits `4344ca0`, `2e12ea1`) is reflected in §5 as PASS and is not part of the launch-blocking set.*
