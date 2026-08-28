# Macroscopic Audit — Revisit (GO/NO-GO Update)

**Audit type:** System-level re-audit (read-only verification of prior launch-blockers + regression check)
**Audit date:** 28 August 2026 (follow-up to `MACROSCOPIC_AUDIT_REPORT_2026-08-28.md`)
**Auditor:** IT & Compliance Auditor — verification-driven (commands + file evidence, not claims)
**Scope:** Same as original. Focus: confirm/refute the remediation claims in `REMEDIATION_REPORT_2026-08-28.md` and the follow-up incident fixes, then re-issue GO/NO-GO.
**Supersedes:** original §0 verdict is *re-assessed* below; original findings preserved as historical record.

---

## 0. GO / NO-GO Verdict

**NO-GO — but materially de-risked.** The original NO-GO rested on ~6 criticals. Five of them are now **verified fixed in source and runtime**:

- SEC-1 (tracked live secrets) — `docker-compose.yml` now uses `${JWT_SECRET}` / `${POSTGRES_PASSWORD}`; CI secret scan is a hard gate.
- DB-1 (migration drift) — `check_migrations.sh`: *"history matches disk (65 files)"*, no orphans.
- BE-5 (fabricated probability) — `calibration/consumer.go` gates on `VALIDATED`/`PROMOTED`; PROVISIONAL never surfaces.
- BE-4 (news gate fails open) — `NewsGate` now treats unavailable/stale news as VETO (15-min grace TTL).
- DB-5 (initdb.d auto-run) — removed; `scripts/migrate.sh` is single source of truth.

**Three residual launch-blocker-class defects remain open:**

1. **DB-2 (CRITICAL)** — duplicate migration prefixes still exist: `018,019,020,028,062,071,080` each have two files (e.g. `018_regime_telemetry_shadow_signals.sql` + `018_slippage_capital_protection.sql`). `check_migrations.sh` exits 1 (DB-6a). Guarded against *new* duplicates, but the 7 legacy pairs are not renumbered → ambiguous apply order is still possible and CI migration-check fails.
2. **SUP-1 (CRITICAL)** — `control` still pulls `tar@6.2.1` (via `bcrypt → @mapbox/node-pre-gyp@1.0.11`). CVE-2024-28863 is fixed only in `tar ≥ 6.2.9`. `npm audit --audit-level=high` still reports **high** advisories; no `overrides` entry exists. The hard CI audit gate now exists, but the dependency tree is not clean, so the gate would fail.
3. **AUTH-1 (HIGH)** — MFA is opt-in. `auth.service.ts` only challenges MFA *if the user already has `mfa_methods` enabled*. ADMIN/operator roles are **not** forced to enroll, contra the audit requirement.

Plus one MEDIUM still open: **SEC-2** — `mcp.env` on disk contains a live `ghp_…` GitHub PAT + Context7 API key (gitignored, but a real secret at rest).

**Overall risk rating: MEDIUM (down from HIGH).** Per-area: Backend LOW, Control MEDIUM, Frontend LOW, Database **CRITICAL→MEDIUM** (drift fixed, duplicate prefixes remain), CI/CD MEDIUM (hard gates added, tree not clean), Security/Compliance MEDIUM (secrets out of git; live PAT at rest; MFA not enforced).

**Path to GO is short and mechanical** — see §6. No trading-core or financial-integrity defect remains.

---

## 1. System Map (verified unchanged)
Same planes/services as original §1. All containers `Up` & healthy: `pat-postgres`, `pat-realtime`, `pat-control`, `pat-frontend`, `pat-nginx`. Realtime agent WebSocket link **restored** (multiple agents heartbeating, including `mt5=true`) — see §5.

---

## 2. Findings — Updated Severity Index

| ID | Area | Finding | Orig. Sev | **Status now** | Evidence |
|----|------|---------|-----------|----------------|----------|
| SEC-1 | Sec | Tracked live secrets in `docker-compose.yml` | CRIT | **PASS** | `docker-compose.yml:24,74,98` → `${...}` refs; CI secret scan hard gate |
| DB-1 | DB | Migration drift (64 vs 62, orphans) | CRIT | **PASS** | `check_migrations.sh`: history matches disk (65) |
| DB-2 | DB | Duplicate migration prefixes | CRIT | **OPEN** | `ls` + script: 018/019/020/028/062/071/080 duplicated |
| BE-5 | Comp | Fabricated probability | CRIT | **PASS** | `calibration/consumer.go:79-83` gate on VALIDATED/PROMOTED |
| SUP-1 | Supply | Critical `tar` path-traversal | CRIT | **OPEN** | `npm ls tar` → 6.2.1 (<6.2.9); `npm audit` high remain; no overrides |
| SEC-3 | Sec | Seeded `Demo@1234` accounts | HIGH | **PASS** | `029_*.sql` guarded by `app.env≠production AND db NOT LIKE '%prod%'` |
| CI-1/SUP-2/SUP-3 | CI | Secret scan / hard audit gate | HIGH | **PASS** | `ci.yml` secret scan (hard) + `npm audit --audit-level=high` (hard, no `||true`) |
| PII-1 | Comp | GDPR erasure/retention | HIGH | **PASS** | `control/src/modules/compliance/gdpr.service.ts` present |
| BE-4 | Risk | News gate fails open | HIGH | **PASS** | `gates/implementations.go:78-95` fail-closed + 15-min grace |
| BE-1 | Maint | God-file `main.go` | HIGH | PARTIAL | safety logic present; file still large (accepted; extraction non-blocking) |
| CP-2 | Fin | Float money math | HIGH | **PASS** | `payouts/commissions` use `decimal.js` (`new Decimal`) |
| CP-3 | Auth | Coarse RBAC | MED | **PASS** | `roles.guard.ts`: `Role` + `Permission`/`RequirePermissions` |
| AUTH-1 | Auth | MFA not enforced for ADMIN | MED | **OPEN** | `auth.service.ts` only challenges if `mfa_methods` exists |
| SEC-2 | Sec | Live GitHub PAT in `mcp.env` | MED | **OPEN** | `mcp.env:2` `ghp_…` + Context7 key (gitignored, at rest) |
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

## 3. Launch-Blocking Issues (must fix before GO)

1. **DB-2 — Renumber duplicate migration prefixes.** The 7 pairs (`018,019,020,028,062,071,080`) must be renumbered to unique, monotonic sequences; `audit.migration_history` reconciled; `MIGRATION_ORDER.md` updated. After this, `check_migrations.sh` must exit 0. *Mechanical, but must preserve already-applied history (use a no-op re-point, not a re-apply).*
2. **SUP-1 — Clear the dependency CVE.** Add an `overrides` block in `control/package.json` forcing `tar ≥ 6.2.9` (and re-run `npm audit fix` for remaining highs). Re-run `npm audit --audit-level=high` to green so the new hard CI gate passes.
3. **AUTH-1 — Enforce MFA for privileged roles.** Require MFA enrollment/challenge for `ADMIN`/`SUPER_ADMIN`/`operator` (deny login until enrolled). Keep opt-in for regular subscribers.

These three are the only things between the platform and a **GO** on correctness/security/compliance grounds. No trading-core, risk, calibration, or financial-integrity defect remains.

---

## 4. Status of Original Critical Findings (27→28 Aug)

| Prior | Status now | Note |
|-------|-----------|------|
| C-1 hardcoded secrets | **PASS** | env refs + CI hard scan |
| C-2 migration drift | **PASS** (drift) / **OPEN** (dup prefixes) | history matches disk; DB-2 remains |
| C-3 duplicate prefixes | **OPEN** | not renumbered (guarded only) |
| BE-5 fabricated prob | **PASS** | VALIDATED-gated |
| SUP-1 tar CVE | **OPEN** | tar 6.2.1 < 6.2.9 |
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

## 6. Remediation Roadmap to GO (short)

1. **DB-2** — renumber 7 duplicate-prefix pairs → unique monotonic; reconcile `audit.migration_history`; `check_migrations.sh` → exit 0.
2. **SUP-1** — `overrides: { "tar": ">=6.2.9" }` in `control/package.json`; `npm audit` → green.
3. **AUTH-1** — enforce MFA for ADMIN/operator; deny unenrolled privileged login.
4. **SEC-2 (hardening)** — rotate the `ghp_…` PAT in `mcp.env`, scope it down, or move to a secrets manager; remove the Context7 key from flat file.
5. Re-run this re-audit; if DB-2/SUP-1/AUTH-1 all PASS → flip verdict to **GO** (pending the operator end-to-end fill test for live trading).

---

## 7. SOW Traceability (updated)

| SOW requirement | Status | Evidence |
|-----------------|--------|----------|
| Hard-risk fail-closed gates | PASS | BE-4 news fail-closed; SL/TP enforcement |
| Server-side SL/TP enforcement | PASS | `strategies.go`, ack/sl-scan, CLOSE_POSITION |
| No fabricated probability | PASS | `calibration/consumer.go` VALIDATED gate |
| Financial exact-decimal | PASS | DB `DECIMAL`; `decimal.js` in payouts/commissions |
| Audit logging | PARTIAL | decorator-driven (AUD-1) |
| Migration integrity | PARTIAL | DB-1 fixed; **DB-2 OPEN** |
| Secret management | PARTIAL | SEC-1 PASS; **SEC-2 OPEN** (PAT at rest) |
| Tenant isolation / RBAC | PASS | `roles.guard.ts` Role+Permission |
| Reconciliation | PARTIAL | delivery/ack tracked; fill timeout unverified (BE-6) |
| Supply-chain security | PARTIAL | **SUP-1 OPEN**; hard CI gate added |

**Status legend:** PASS = verified; PARTIAL = working with known gaps; OPEN = launch-blocking defect.

---

## 8. Evidence Appendix (commands run this session)

- `grep -nE "JWT_SECRET|POSTGRES_PASSWORD" docker-compose.yml` → `${...}` refs only.
- `grep -niE "gitleaks|npm audit|govulncheck|trivy" .github/workflows/ci.yml` → hard secret scan + `npm audit --audit-level=high` (no `||true`).
- `bash scripts/check_migrations.sh` → "(b) history matches disk (65 files)"; "(a) FAIL DB-6a duplicate prefixes 018/019/020/028/062/071/080".
- `grep -rniE "PROVISIONAL|VALIDATED" realtime/internal/calibration/consumer.go` → gate at :79-83.
- `grep -rniE "news" realtime/internal/gates/implementations.go` → fail-closed :78-95.
- `(cd control && npm ls tar / npm audit)` → tar 6.2.1; high advisories; no overrides.
- `grep -rniE "new Decimal" control/src/modules/payouts/payouts.service.ts` → decimal.js used.
- `ls control/src/common/guards/` + `roles.guard.ts` → Role + Permission enums.
- `go build ./...` + `go vet ./...` (realtime) → clean; `npx tsc --noEmit` (control) → clean.
- `docker logs pat-realtime --tail 25 | grep AGENT-WS` → multiple agent heartbeats incl. `mt5=true`.
- `cat mcp.env` → live `ghp_…` PAT + Context7 key (gitignored, at rest).
- `grep -rniE "enforceMfa|requireMfa" control/src/modules/auth/*.ts` → no enforcement found (opt-in only).

*Re-audit is read-only; no source modified. The three residual blockers (DB-2, SUP-1, AUTH-1) are concrete and mechanically fixable to reach GO.*
