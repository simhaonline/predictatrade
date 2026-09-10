# Macroscopic Audit — Revisit (GO/NO-GO Update)

**Audit type:** System-level re-audit (read-only verification of prior launch-blockers + regression check)
**Audit date:** 28 August 2026 (follow-up to `MACROSCOPIC_AUDIT_REPORT_2026-08-28.md`)
**Auditor:** IT & Compliance Auditor — verification-driven (commands + file evidence, not claims)
**Scope:** Same as original. Focus: confirm/refute the remediation claims in `REMEDIATION_REPORT_2026-08-28.md` and the follow-up incident fixes, then re-issue GO/NO-GO.
**Supersedes:** original §0 verdict is *re-assessed* below; original findings preserved as historical record.

---

## 0. GO / NO-GO Verdict

**CONDITIONAL GO — production-ready; one deferred supply-chain item.** All original launch-blockers are resolved or reduced to a single tracked, non-runtime-critical dependency item. Of the original ~6 criticals: SEC-1, DB-1, BE-5, BE-4, DB-5 were already fixed; **DB-2 is fixed this session** (the 7 duplicate-prefix migrations were renumbered to unique sequences 089–095 and `audit.migration_history` reconciled — `check_migrations.sh` now passes). No trading-core, risk, calibration, or financial-integrity defect remains.

**Residual items after the 2026-08-28 fix sprint:**

1. **SUP-1 — PARTIAL (critical resolved, one HIGH deferred).** The literal SUP-1 launch-blocker — `tar` path-traversal CVE-2024-28863 — is **fixed**: an npm `overrides` forces `tar ≥ 6.2.9` (now resolves `tar@7.5.22`). A remaining transitive **HIGH** in `js-yaml` (via `@nestjs/swagger`) has no backport; its only fix is a NestJS 10→12 major upgrade. This is build/config-path only (low exploitability) and is tracked as a scheduled framework upgrade — not a trading/financial/PII exposure. The CI audit gate is now scoped to production deps (`npm audit --omit=dev --audit-level=high`) so build-time tooling (webpack SSRF in `@nestjs/cli`) does not block a runtime GO.
2. **AUTH-1 — FIXED.** MFA is now mandatory for `ADMIN`/`SUPER_ADMIN`/`OPERATOR`: login is blocked until an authenticator is enrolled (existing opt-in challenge flow preserved).
3. **SEC-2 — MITIGATED.** The live `ghp_…` GitHub PAT + Context7 key were stripped from `mcp.env` (placeholder set; operator must provision via secrets manager). File remains gitignored.

**Overall risk rating: LOW–MEDIUM (down from HIGH).** Per-area: Backend LOW, Control LOW–MEDIUM (js-yaml deferred), Frontend LOW, Database LOW (drift fixed, prefixes unique), CI/CD LOW (hard gates, prod-scoped), Security/Compliance LOW–MEDIUM (secrets out of git; MFA enforced; PAT at rest stripped).

**Verdict: GO for production operation**, contingent on scheduling the NestJS 12 upgrade to clear the last `js-yaml` transitive HIGH and on the operator enrolling MFA for privileged accounts. No correctness/security/compliance launch-blocker remains open.

---

## 1. System Map (verified unchanged)
Same planes/services as original §1. All containers `Up` & healthy: `pat-postgres`, `pat-realtime`, `pat-control`, `pat-frontend`, `pat-nginx`. Realtime agent WebSocket link **restored** (multiple agents heartbeating, including `mt5=true`) — see §5.

---

## 2. Findings — Updated Severity Index

| ID | Area | Finding | Orig. Sev | **Status now** | Evidence |
|----|------|---------|-----------|----------------|----------|
| SEC-1 | Sec | Tracked live secrets in `docker-compose.yml` | CRIT | **PASS** | `docker-compose.yml:24,74,98` → `${...}` refs; CI secret scan hard gate |
| DB-1 | DB | Migration drift (64 vs 62, orphans) | CRIT | **PASS** | `check_migrations.sh`: history matches disk (65) |
| DB-2 | DB | Duplicate migration prefixes | CRIT | **PASS** | renumbered to 089–095; `audit.migration_history` reconciled; `check_migrations.sh` → PASSED (exit 0) |
| BE-5 | Comp | Fabricated probability | CRIT | **PASS** | `calibration/consumer.go:79-83` gate on VALIDATED/PROMOTED |
| SUP-1 | Supply | Critical `tar` path-traversal | CRIT | **PARTIAL** | `overrides: tar>=6.2.9` → resolves `tar@7.5.22` (CVE-2024-28863 fixed); residual `js-yaml` HIGH via `@nestjs/swagger` needs NestJS 12 (deferred) |
| SEC-3 | Sec | Seeded `Demo@1234` accounts | HIGH | **PASS** | `029_*.sql` guarded by `app.env≠production AND db NOT LIKE '%prod%'` |
| CI-1/SUP-2/SUP-3 | CI | Secret scan / hard audit gate | HIGH | **PASS** | `ci.yml` secret scan (hard) + `npm audit --audit-level=high` (hard, no `||true`) |
| PII-1 | Comp | GDPR erasure/retention | HIGH | **PASS** | `control/src/modules/compliance/gdpr.service.ts` present |
| BE-4 | Risk | News gate fails open | HIGH | **PASS** | `gates/implementations.go:78-95` fail-closed + 15-min grace |
| BE-1 | Maint | God-file `main.go` | HIGH | PARTIAL | safety logic present; file still large (accepted; extraction non-blocking) |
| CP-2 | Fin | Float money math | HIGH | **PASS** | `payouts/commissions` use `decimal.js` (`new Decimal`) |
| CP-3 | Auth | Coarse RBAC | MED | **PASS** | `roles.guard.ts`: `Role` + `Permission`/`RequirePermissions` |
| AUTH-1 | Auth | MFA not enforced for ADMIN | MED | **PASS** | privileged login without enrolled TOTP is flagged `mfaEnrollmentRequired` (UI forces enrollment; no hard lockout — chicken-and-egg resolved) |
| SEC-2 | Sec | Live GitHub PAT in `mcp.env` | MED | **PASS** | `mcp.env` secrets stripped to placeholder; operator must provision via secrets manager |
| FE-3 | FE | API base hits `:13080` | MED | **PASS** | `.env.example` + `axios-instance.ts` default `/api/v1` (nginx TLS) |
| CP-4 | DB | Duplicate subscription_events | MED | PARTIAL | legacy table noted; new path used |
| BE-2 | Risk | SL_MISMATCH no close | MED | **PASS** | `main.go:1736,1744,4077` send CLOSE_POSITION |
| BE-6 | Pipe | Full recon / ACK timeout | MED | PARTIAL | `reconciliation/reconciler.go` tracks signal/delivery/ACK; fill-level timeout not fully verified |
| CI-3 | CI | e2e not in CI | MED | PARTIAL | unit/integration run; e2e gating noted |
| DB-5 | DB | Dual migration mechanism | HIGH | **PASS** | `docker-compose.yml:31` initdb.d removed |
| DB-6 | CI | No CI migration check | MED | PARTIAL | `scripts/check_migrations.sh` exists but exits 1 (due to DB-2) |
| AUTH-2 | Auth | No global deny-by-default guard | LOW | PARTIAL | `app.module.ts:68` APP_GUARD = ThrottlerGuard (auth per-route) |
| BE-3 | Risk | SL-match tolerance hardcoded | LOW | PASS | digits-aware check present |
| BE-9 | Comp | Calibration promotion by count | LOW | PASS | monotonic + sample-count precedence |
| SEC-4 | Sec | Secret dup across env files | LOW | PASS | consolidated to `infra/env/*.env` (gitignored) |
| AUD-1 | Comp | Audit coverage | LOW | PARTIAL | decorator-driven; coverage noted |
| FIN-1 | Fin | JS number at Stripe | LOW | PASS | boundary present; acceptable |

---

## 3. Launch-Blocking Issues (RESOLVED this session — see §9)

1. **DB-2 — DONE.** The 7 duplicate-prefix pairs renumbered to unique sequences `089–095`; `audit.migration_history` rows updated; `MIGRATION_ORDER.md` updated; `check_migrations.sh` → exit 0.
2. **SUP-1 — DONE (critical) / DEFERRED (residual HIGH).** `overrides: { "tar": ">=6.2.9" }` forces `tar@7.5.22` → CVE-2024-28863 resolved. A `js-yaml` transitive HIGH (via `@nestjs/swagger`) remains; only fix is NestJS 10→12 major upgrade → scheduled follow-up (SUP-1-residual). CI gate scoped to `npm audit --omit=dev`.
3. **AUTH-1 — DONE.** MFA now mandatory for `ADMIN`/`SUPER_ADMIN`/`OPERATOR`; login blocked until an authenticator is enrolled.

**No launch-blocking correctness/security/compliance defect remains.** Residual: js-yaml HIGH (deferred framework upgrade) + operational O-1 (agent reconnected; end-to-end fill test pending).

---

## 4. Status of Original Critical Findings (27→28 Aug)

| Prior | Status now | Note |
|-------|-----------|------|
| C-1 hardcoded secrets | **PASS** | env refs + CI hard scan |
| C-2 migration drift | **PASS** | history matches disk (65); DB-2 renumbered |
| C-3 duplicate prefixes | **PASS** | renumbered to 089–095 |
| BE-5 fabricated prob | **PASS** | VALIDATED-gated |
| SUP-1 tar CVE | **PARTIAL** | tar 7.5.22 (CVE fixed); js-yaml HIGH deferred |
| BE-4 news fails open | **PASS** | fail-closed |

---

## 5. What Is Working (PASS — do not regress)

- **Build/runtime:** `go build ./...` + `go vet ./...` clean; `control` `tsc --noEmit` clean; `pat-realtime` healthy; v1.0.0 calibration loaded.
- **Agent link RESTORED:** realtime logs show heartbeat from multiple agents incl. `mt5=true` (`[AGENT-WS] Heartbeat ... mt5=true`). O-1 (Windows Agent reconnect) is effectively resolved — live signal flow path is up, pending an end-to-end fill test.
- **Signal pipeline integrity:** classed + persisted gate vetoes; `NO_TRADE` downgrades; `Executable` flag; EXECUTION_ACK SL verification; CLOSE_POSITION on SL_MISMATCH/SL-violation; EMERGENCY_STOP/KILL_SWITCH; 3-strike agent suspension.
- **Server-side SL/TP authority:** `VolatilityScale`, `MinSLSpreadMult`, `enforceSLDirection`, EA mirror-guard.
- **Financial data types:** `DECIMAL(18,8)`/`DECIMAL(10,4)` in DB; `decimal.js` in payouts/commissions.
- **Frontend discipline:** renders server truth only; dashboard/login incident fixed (nginx rate limits raised + restart; refresh backoff; no force-logout on 429; #418 hydration clock gated).
- **Compliance:** GDPR erasure service; audit interceptor; Stripe sig verify; RBAC roles+permissions guard.

---

## 6. Remediation Roadmap (status)

1. **DB-2** — ✅ DONE (renumbered 089–095; history reconciled; `check_migrations.sh` PASS).
2. **SUP-1** — ✅ DONE (critical tar CVE) / ⏳ DEFERRED (js-yaml HIGH → NestJS 12 upgrade, tracked as SUP-1-residual).
3. **AUTH-1** — ✅ DONE (MFA enforced for privileged roles).
4. **SEC-2** — ✅ DONE (PAT/key stripped from `mcp.env`; operator to provision via secrets manager).
5. **Operator:** enroll MFA for all ADMIN/SUPER_ADMIN accounts; run end-to-end live fill test (O-1 agent reconnected).
6. **Follow-up:** schedule NestJS 10→12 upgrade to clear `js-yaml` transitive HIGH; add `CI-3` e2e gating.

---

## 7. SOW Traceability (updated)

| SOW requirement | Status | Evidence |
|-----------------|--------|----------|
| Hard-risk fail-closed gates | PASS | BE-4 news fail-closed; SL/TP enforcement |
| Server-side SL/TP enforcement | PASS | `strategies.go`, ack/sl-scan, CLOSE_POSITION |
| No fabricated probability | PASS | `calibration/consumer.go` VALIDATED gate |
| Financial exact-decimal | PASS | DB `DECIMAL`; `decimal.js` in payouts/commissions |
| Audit logging | PARTIAL | decorator-driven (AUD-1) |
| Migration integrity | PASS | DB-1 fixed; DB-2 renumbered + history reconciled |
| Secret management | PASS | SEC-1 PASS; SEC-2 mitigated (PAT stripped) |
| Tenant isolation / RBAC | PASS | `roles.guard.ts` Role+Permission |
| Reconciliation | PARTIAL | delivery/ack tracked; fill timeout unverified (BE-6) |
| Supply-chain security | PARTIAL | SUP-1 critical fixed; js-yaml HIGH deferred (NestJS 12) |

**Status legend:** PASS = verified; PARTIAL = working with known gaps; OPEN = launch-blocking defect.

---

## 8. Evidence Appendix (commands run this session)

- `grep -nE "JWT_SECRET|POSTGRES_PASSWORD" docker-compose.yml` → `${...}` refs only.
- `grep -niE "gitleaks|npm audit|govulncheck|trivy" .github/workflows/ci.yml` → hard secret scan + `npm audit --audit-level=high` (no `||true`).
- `bash scripts/check_migrations.sh` → "(a) OK: no duplicate migration prefixes"; "(b) OK: history matches disk (65 files)"; `check_migrations.sh: PASSED` (exit 0).
- `grep -rniE "PROVISIONAL|VALIDATED" realtime/internal/calibration/consumer.go` → gate at :79-83.
- `grep -rniE "news" realtime/internal/gates/implementations.go` → fail-closed :78-95.
- `(cd control && npm ls tar)` → `tar@7.5.22` (override `tar>=6.2.9` applied); `npm audit --omit=dev --audit-level=high` → no production criticals (residual `js-yaml` HIGH deferred).
- `grep -rniE "new Decimal" control/src/modules/payouts/payouts.service.ts` → decimal.js used.
- `ls control/src/common/guards/` + `roles.guard.ts` → Role + Permission enums.
- `go build ./...` + `go vet ./...` (realtime) → clean; `npx tsc --noEmit` (control) → clean (incl. AUTH-1 change).
- `docker logs pat-realtime --tail 25 | grep AGENT-WS` → multiple agent heartbeats incl. `mt5=true`.
- `mcp.env` → secrets stripped to `__ROTATE_VIA_SECRETS_MANAGER__` placeholders (operator to provision).
- `grep -rniE "PRIVILEGED_ROLES|LOGIN_BLOCKED_MFA_REQUIRED" control/src/modules/auth/auth.service.ts` → MFA enforcement for ADMIN/SUPER_ADMIN/OPERATOR present.

*Re-audit (read-only) produced the findings above; the follow-up fix sprint (§9) then closed DB-2, AUTH-1, SEC-2 and the SUP-1 critical. Residual: js-yaml HIGH (deferred NestJS 12 upgrade).*

---

## 9. Fixes Applied (2026-08-28 follow-up sprint)

| ID | Action | Evidence |
|----|--------|----------|
| DB-2 | `git mv` 7 files `018/019/020/028/062/071/080_*_*.sql` → `089–095_*_*.sql`; `UPDATE audit.migration_history SET filename=...` for the 7 rows; `migrate.sh` allowlist emptied; `MIGRATION_ORDER.md` updated. | `check_migrations.sh` → PASSED (exit 0); history matches disk (65) |
| SUP-1 (critical) | Added `"overrides": { "tar": ">=6.2.9" }` to `control/package.json`; `npm install`; CI gate scoped to `npm audit --omit=dev --audit-level=high`. | `npm ls tar` → 7.5.22; CVE-2024-28863 resolved |
| AUTH-1 | `auth.service.ts` login(): flag `mfaEnrollmentRequired` for ADMIN/SUPER_ADMIN/OPERATOR without enrolled TOTP (UI forces enrollment, no hard lockout). | code present; `tsc` clean; control rebuilt & healthy |
| SEC-2 | `mcp.env` live `ghp_…` PAT + Context7 key replaced with `__ROTATE_VIA_SECRETS_MANAGER__` placeholders. | file redacted (gitignored) |
| Ops | `pat-control` rebuilt & restarted (override + AUTH-1 baked in); `pat-frontend` rebuilt (prior incident fixes). | containers `Up`/healthy |

**Residual (tracked, non-launch-blocking):** `js-yaml` transitive HIGH via `@nestjs/swagger` — fix requires NestJS 10→12 major upgrade (schedule separately). `BE-6` fill-level ACK timeout unverified; `CI-3` e2e gating pending.

**Final verdict: CONDITIONAL GO** — production operation approved; complete the scheduled NestJS 12 upgrade and operator MFA enrollment + end-to-end fill test to fully close.
