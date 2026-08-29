# Implementation Status — Living Ledger
## v1.17.3 — 29 August 2026

Per-requirement status as of the 29 August session. Historical audits
(28 Aug `NO-GO`, revisit `CONDITIONAL GO`) are preserved under
[`reports/`](.) — every launch-blocker they identified is closed below with
fresh evidence.

## Launch-blockers (from 28 Aug audits) — all CLOSED

| ID | Finding | Closure evidence |
|----|---------|------------------|
| SEC-1 | Secrets in `docker-compose.yml` | `${...}` env-file injection; CI secret-scan hard gate; runner-verified scan finds no values in tracked files |
| DB-1 | Migration drift (disk vs history) | `check_migrations.sh` PASS — 65 files == history; enforced in CI |
| DB-2 | Duplicate migration prefixes | Renumbered 089–095; history reconciled |
| DB-5 | Dual migration mechanism | `initdb.d` auto-run removed; `migrate.sh` only |
| BE-5 | Fabricated/PROVISIONAL probability | `calibration/consumer.go` gates on VALIDATED/PROMOTED only |
| BE-4 | News gate fail-open | Fail-closed + 15-min grace (`gates/implementations.go`) |
| SEC-3 | Seeded `Demo@1234` prod risk | Migration 029 guarded `app.env ≠ production AND db NOT LIKE '%prod%'` |
| SUP-1 | Critical `tar` CVE + js-yaml HIGH | `tar@7.5.22`; **NestJS 12 upgrade done 29 Aug** — js-yaml 5.4.1, lodash eliminated, multer 2.2.0, express 5 — `npm audit --omit=dev` = **0 high/critical** |
| SEC-2 | Live PAT in `mcp.env` | Redacted to placeholders |
| AUTH-1 | MFA not enforced for privileged roles | `mfaEnrollmentRequired` gate in login for ADMIN/SUPER_ADMIN/OPERATOR |
| CI-1/SUP-2/SUP-3 | Secret-scan/audit-gate gaps | Hard gates, prod-scoped audit, `gitleaks` config; **CI 6/6 green** |
| PII-1 | GDPR erasure absent | `compliance/gdpr.service.ts` + migration 088 (`compliance.gdpr_operations`) |
| BE-2 | SL_MISMATCH not closed | CLOSE_POSITION sent on NO_SL + SL_MISMATCH; 3-strike agent suspension |
| CP-2 | Float money math | `decimal.js` in payouts/commissions; `NUMERIC` columns |
| CP-3 | Coarse RBAC | `roles.guard.ts` Role + Permission/RequirePermissions |
| BE-3 | Hardcoded SL tolerance | Digits-aware (tick_size×2, min 0.5) |
| BE-9 | Calibration promotion by count | Monotonicity + sample-sufficiency precedence |
| FE-3 | API base hits `:13080` directly | `/api/v1` via nginx TLS edge (`api.predictatrade.com`) |

## Newly verified / fixed this session (29 Aug)

| Area | Item | Evidence |
|------|------|----------|
| CI | Workflow YAML parse error killed 31 runs | `security` job step indentation fixed; run 33250798333 executed, follow-ups green |
| CI | Secret-scan self-exclusion | `--exclude` glob fix; verified scan exits clean on tracked-files-only tree |
| Control | NestJS 10→12 + TS 6 + Jest 30 | 167/167 tests; ESM runtime flag wired into scripts + CI |
| Control | `POST /subscriptions` 500 (PG17 `$5` inference) | `$5::text` casts; FREE-plan subscription created live |
| Realtime | BE-6 fill reconciliation | `RecordFill` broker-ticket-keyed; 30s monitor + gauges + ntfy; `-race` suite clean |
| Realtime | Devil Liquidity duplicate-mark bug | Detection-ATR-normalized level guard + recency window; 2 regression tests; production double-charge eliminated |
| Python | `np.float64` DB serialization; CAGR OverflowError; importorskip | 139/139 tests incl. live-TimescaleDB roundtrip |
| Frontend | All 38 pages runtime-probed (USER+ADMIN) | All data endpoints 200; `POST /subscriptions` fix; e2e 18/18 ×3; 13 lint errors fixed |
| Windows Agent | v1.2.40 artifacts, installers, env contract | Live downloads hashed byte-exact; 7 env vars wired; EA sources now published on downloads host |
| Docs | README/docs/INDEX v1.17.3; FLOW_DIAGRAMS (9 mermaid); DB_ERD; full API_REFERENCE + OpenAPI mirror | This docs update |

## Remaining operator actions (not code)

| # | Action | Blocks |
|---|--------|--------|
| 1 | Attach recompiled Master Node EA (v1.17.2 `MasterAppend` fix) to XAUUSD chart on MT5 box | market data feed (currently NO_DATA) |
| 2 | Enter ACTIVE license key in Client EA (old activations bind to REVOKED `PAT-EE710BF6…`) | executable signal delivery |
| 3 | One demo fill round-trip: SIGNAL → EXECUTION_ACK → TRADE_RESULT (reconciliation gauges stay 0) | go-live attestation |
| 4 | Rotate `JWT_SECRET` in a coordinated maintenance window (agents re-issue tokens) | security hygiene |
| 5 | Enroll MFA for all ADMIN/SUPER_ADMIN/OPERATOR accounts | AUTH-1 completion |
| 6 | Run `scripts/backup/restore_test.sh` on production and record result | DR attestation |

## Traceability matrix (condensed)

| SOW requirement | Implementation | Tests | Observability | Status |
|---|---|---|---|---|
| Hard-risk fail-closed gates | `realtime/internal/gates/*` (16) | gates_* tests; geometry 49/49 | `pat_gate_*` metrics | PASS |
| Server-side SL/TP authority | EXECUTION_ACK verify + CLOSE_POSITION (digits-aware) | edge tests | logs + suspension | PASS |
| No fabricated probability | `calibration/consumer.go` VALIDATED gate | parity tests | `pat_calibrated_probability` | PASS |
| Signal↔fill reconciliation | `internal/reconciliation` + monitor | reconciler + shouldAlert tests | `pat_reconciliation_*` | PASS |
| Financial exact-decimal | DECIMAL columns; decimal.js | commission/payout specs | reconciliation pages | PASS |
| Supply chain | NestJS 12, tar override, secret-scan CI, gitleaks | CI 6/6 | audit trail | PASS |
| Tenant isolation/RBAC | roles+permissions guards | admin.guard.spec | audit events | PASS |
| Auth/MFA | mandatory enrollment for privileged | auth.service.spec | login events | PASS |
| GDPR | gdpr.service + migration 088 | anon. tombstones verified | gdpr_operations | PASS |
| Backups | pg_dump custom + off-host + restore drill script | `restore_test.sh` | `backup_metadata` | PASS (script verified; prod drill pending) |
| Observability | OTel/Prometheus/Grafana/JSON logs | — | `pat_*` families + dashboards | PASS |
| Windows/MT edge | agent v1.2.40, installers, EA publication | build + hash verification | agent telemetry + heartbeats | PASS (EA attach pending) |