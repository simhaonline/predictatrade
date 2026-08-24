# Predict-A-Trade v1.0.0 — FULL PRODUCTION AUDIT

**Date:** 2026-08-24 · **Auditor:** Codex full-repo investigation
**Method:** Parallel deep-code audit of every plane (Go realtime, NestJS control, Next.js frontend, PostgreSQL/TimescaleDB, Windows Agent/MQL, infrastructure) + live runtime verification of containers, Valkey, engine endpoints and nginx logs.
**Build gates at audit time:** Go `build`+`vet` PASS · frontend `tsc` PASS (exit 0) · control `tsc` PASS, tests **160/160 PASS** · Go tests: **27 pkgs pass / 5 FAIL (~14 failing tests)**.

---

## 0. EXECUTIVE VERDICT

| Plane | Code compiles | Tests | Production-ready? |
|---|---|---|---|
| Go realtime engine | ✅ | ⚠️ 14 failing | **NO** — auth/entitlement/safety gaps |
| NestJS control plane | ✅ | ✅ 160/160 | **NO** — financial P0s |
| Next.js frontend | ✅ | — | **NEAR** — strategy save broken in prod |
| Database (PG17/TimescaleDB) | ✅ schema live | — | **NO** — superuser everywhere |
| Windows Agent/MQL | ✅ builds | — | **NO** — updater semver broken, EA guards dead |
| Infra/monitoring/backup | — | — | **NO** — Prometheus dead, zero alerts, stale backups |

## **OVERALL: NO-GO for live production.** The system runs and trades in shadow/paper posture, but mandatory P0 safety, security and financial-integrity gates fail. Section 8 lists the exact exit criteria.

Live posture verified during audit: 3 MT5 agents connected (heartbeats OK), signals broadcasting to agents, TimescaleDB healthy (10.5M ticks, 2.76M candles), Valkey hot-cache functional, WS/API/CORS routing healthy after today's fixes.

---

## 1. GO REALTIME ENGINE (`realtime/`)

### P0 — Security
- **[P0-RT1] Agent WebSocket `/ws/v1/agent` has NO authentication.** `internal/gateway/agent_ws.go:177-192`; upgrader `CheckOrigin: return true` (:56); `agentId` optional (UUID minted if absent). Anyone on the internet can (a) connect with any ID and receive every EXECUTABLE BUY/SELL signal with entry/SL/TP, and (b) inject forged TICK/MARKET_SNAPSHOT that poison gate hydration incl. exposure/margin via fake `account_info` (`agent_provider.go:486-491`). License validation on MASTER_INIT is advisory — invalid licenses are not disconnected. Server ignores `agentVersion` entirely (no min-version policy).
- **[P0-RT2] Committed JWT secret.** `docker-compose.yml:68`, same value in `infra/env/control.env` and `infra/env/realtime.env`. Verifies both control sessions and realtime identity. ⚠️ Rotation is an operator-authorized action only: control derives device-fingerprint pepper + AES key FROM this secret (`device-auth.service.ts:507,520`) — rotating bricks stored device credentials.
- **[P0-RT3] `handleSignals` entitlement check is cosmetic.** `http.go:167-232`: Valkey fast-path returns signals unfiltered to anonymous callers (comment admits it relies on frontend); JWT is never parsed/verified — any `Bearer anything` unlocks EXECUTABLE signals from DB path.

### P1 — Safety & delivery
- **[P1-RT4] NewsGate fails OPEN on DATA_UNAVAILABLE** (`implementations.go:88-99`) contradicting its own main.go:300 commentary; deployed `NEWS_FAIL_POLICY=ALLOW_TRADING` (`infra/env/realtime.env:36`) compounds it → news blackout protection currently decorative. Failing test `TestRiskEngine_ProviderStale_FailSafe` proves drift.
- **[P1-RT5] NO emergency-stop gate exists.** `types.KillSwitch` (:597)/`ExecEmergencyStop` (:549) defined but never registered or evaluated. Mandatory P0 safety control absent.
- **[P1-RT6] Browser WS clients can never receive signals:** `client.entitlements` never populated anywhere → `isEntitled` always false → broadcast delivers nothing. Combined with RT3 there is no working entitled delivery channel.
- **[P1-RT7] P0/P1 events silently dropped when client buffer full** (`websocket.go:222-236`, identical branches, `select/default`). SOW forbids silent drop. No replay buffer.
- **[P1-RT8] Signal resume/replay dead end-to-end:** `handleSignalResume` never validates JWT nor checks device ownership; `signal/delivery.go DeliveryManager` exists but is never wired in main.go → `trading.signal_deliveries` stays empty (live count = **0** despite 22k+ signals).
- **[P1-RT9] Acceptance failure: neutral market produces SELL** (`TestNoForcedSignals_NeutralMarketProducesNoTrade` FAILS) — direct violation of "NO-TRADE is first-class".
- **[P1-RT10] Hardcoded round-trip cost $0.30** (`main.go:2114`) — TotalCostGate runs on fiction, ignores spread-derived cost/config.

### P2 — Correctness
- Two divergent EMA definitions (`pkg/math/math.go:167` seeds first-value vs `features/indicators.go:260` SMA-seeded); MACD signal line computed over truncated ≤20-sample window.
- VWAP session reset keyed on calendar `Day()` change, not session/DST-aware; bands approximate (`features/vwap.go:27-47`).
- Bootstrap lookbacks use calendar time vs market-time counts → H4 gets ~18 of 250 candles (the recurring warning), M15 also short. Fix ≈ ×1.45 weekend factor (`main.go:438-446`).
- `refreshGateStates` force-resets MinATR/StopHuntFilter to PASS every 5s (`main.go:2377-2388`) — bypassed rather than fixed their lifecycle (5 gate tests failing).
- SlippageGate never hydrated → structurally always-PASS dead veto.
- Margin gate = `freeMargin > 0` only — no margin-level/stop-out-distance validation.
- ML/Ollama block does synchronous 2s external I/O per strategy inside the candle loop with results discarded (`main.go:1413-1443`) — SOW hot-path violation; `OLLAMA_HOST=http://localhost:11434` unreachable from container anyway.
- Second unclosed DB pool for exit profiles (`main.go:152-178`); news/gate goroutines not bound to shutdown ctx; HTTP bind failure non-fatal.
- Engine factory overrides hardcode magic SL/TP numbers (`engines/factory.go:11-52`); declared `MinGrade:"A"` never enforced.
- Cross-market active mode adjusts RawScore after threshold computation without re-running dominance checks (`main.go:1617-1626`).
- Env var name traps: `TWELVE_DATA_KEY`/`ONNXRUNTIME_LIB` set but code reads `TWELVEDATA_API_KEY`/`ONNX_RUNTIME_PATH`.
- Quota ledger: feature absent entirely (B-02 claim in commit d30f61c not substantiated by a wired quota writer).
- Bearish EMA crosses collapse into `EMACross921=false` — information loss (`indicators.go:90-95`). MarnieFib anchors to stale swings.

### PASS (reuse-worthy)
RSI/ATR/ADX Wilder implementations correct w/ SMA seeding & guards · tick honesty (no fabrication; drop-on-backpressure w/ metrics) · calibration integrity (probability UNVERIFIED unless model PROMOTED) · data freshness ladder (10s degraded/30s veto) · gate registry order & fail-closed skeleton.

---

## 2. NESTJS CONTROL PLANE (`control/`)

`tsc` clean; tests 160/160 pass. Defects below are logic/financial:

### P0 — Money/security exploitable now
- **[P0-CP1] Billing webhook fully unauthenticated.** `billing.controller.ts:60-63`, `billing.service.ts:276-291`: no provider signature verification, no idempotency, no replay protection. Forged `payment.succeeded` marks invoices PAID → paid entitlements/commissions with no money. `subscription.active` conflated with payment.
- **[P0-CP2] `requestPayout` inserts non-existent column `amount`** (schema: `requested_amount DECIMAL(18,8)`, migration 004) → **every payout request throws at runtime.** No balance validation, no idempotency key; method/destination silently discarded.
- **[P0-CP3] Payout completion cannot prevent double payout:** no code ever creates `referral.payout_items` rows; wallet `available_balance` never debited on completion; hardcoded `'USD'` in multi-currency ledger; debit without negative-balance guard; `approved_amount` never written.
- **[P0-CP4] Licensing cross-tenant IDOR:** heartbeat/revoke/registerTerminal never check `devices.user_id` (`licensing.controller.ts:57-61`, service :170,:276-300) — any authenticated user can revoke another user's device or attach activations to foreign devices.

### P1 — Financial integrity
- **Float money math throughout** (violates SOW exact-decimal): billing.service.ts:89,104-117,140; commissions.service.ts:247,320-321,357,415,450 (worst: `clearEligible` moves wallet money via JS floats); admin.service.ts:347-350.
- `creditReferralForLicense` dup-check then per-row INSERT outside transaction → race + partial multi-level entries on failure (:95-156).
- `registerTerminal` limit-check selects only `l.max_mt_accounts` then inserts undefined `rows[0].id` as license_id (:216-233).
- Device-auth proof-of-device theater: access token generated but never persisted/verified (:136); full HMAC verifier implemented+tested but wired to ZERO routes; possession of session_id UUID grants indefinite lease renewal.
- Nonce TOCTOU + burned-before-verify (:614-629). Key separation violated: fingerprint pepper + AES key derived from JWT_SECRET (:507,:520).
- Registration non-atomic (user/referral/code in 3 autocommits, auth.service.ts:77-125). OTP challenge replayable post-verify. `markInvoicePaid` lacks state guard/audit/arbitrary paymentId link (:163-167).
- Commission rule mutation updates rate in place without version row/audit (:483-501) — SOW versioning violation.

### PASS
Refresh-token rotation (hashed, FOR UPDATE, family reuse detection) · password reset (advisory lock, one-time jti) · tenant scoping on subscription/billing reads · PATCH strategies server-side plan validation · env names consistent · all modules imported · no TODO stubs.

---

## 3. FRONTEND (`frontend/`, `status/`)

`tsc --noEmit` clean. Honest-state coverage strong (stale banners, DegradedNote, no recomputation of server truth).

- **[P1-FE1] Strategy save broken in production.** strategies/page.tsx:84-88 uses raw `fetch` + `localStorage.getItem('token')` — token actually lives in memory+cookie `pat_access_token` (auth.ts:14,90-103). Request carries no credential → every save silently fails.
- **[P2-FE2] Admin gating client-redirect only; anonymous `/admin/*` mounts components and fires API calls (no middleware.ts).
- **[P2-FE3] WS manager:** reconnect timer survives `disconnect()` (zombie sockets); no freshness watchdog → UI can show LIVE while feed dead; gives up silently after 10 attempts.
- **[P2-FE4] Fabricated WS defaults** (websocket.ts:76-112): direction→'NO_TRADE' cast as BUY|SELL, timestamps→now(), bid/ask→0 — violates no-fabrication rule.
- **[P2-FE5] Signal pipeline:** one WS event permanently overrides richer REST truth; CLOSED/EXPIRED never expire; `NO-TRADE` vs `'NO_TRADE'` string mismatch between filter and normalizer.
- **[P2-FE6] Status page publishes unverifiable compliance claims (SOC2/PCI/DR "Implemented") and prints "No incidents recorded" even when overall=down.**
- **[P3]** status JSON leaks internal topology publicly; `_ua_tmp.tsx` scratch file committed; inconsistent localhost fallback URLs; access token in JS-readable cookie (XSS window).

---

## 4. DATABASE (PostgreSQL 17 / TimescaleDB 2.29.1)

41 repo migrations, canonical-forward, idempotent; live cluster matches structurally. Extensions: pgcrypto, timescaledb(+toolkit), uuid-ossp, vector 0.8.6. **~190 live tables across 13 schemas** (ai, audit, billing, compliance, control, finance, iam, licensing, market, referral, research, support, system, trading).

- **[P0-DB1] Zero least-privilege roles exist.** Migration 001 defines 9 roles (`nest_control`, `go_realtime`, …) — none created live; both runtimes connect as **superuser `pat_admin`** (docker-compose.yml:65,95; scripts/migrate.sh default). Full-cluster blast radius on any app compromise.
- **[P1-DB2] Migration history drift:** live `audit.migration_history` references 10 filenames deleted from repo (`003_control_tables.sql`…`025_...`); repo's `028_bar_processing_metadata.sql` applied out-of-band (table exists, history missing); 064–066 absent from live history. Fresh environments cannot reproduce this DB.
- **[P1-DB3] `market.flow_features` silently failed hypertable conversion** (DO-block swallowed) — plain table with a retention grant but no retention job; unbounded growth risk. `compliance.client_event_log` hypertable without compression (siblings compressed).
- **[P1-DB4] Dead delivery pipeline confirmed live:** signals 22,458+ (22,417 DETECTED / 46 CONFIRMED), yet signal_deliveries=0, ledger=0, recipients=0, strategy_preferences=0 — no subscriber has EVER received a durable delivery. Corroborates RT8.
- **[P2-DB5]** Duplicate migration sequence numbers (018,019,020,028,062 ×2 each) → filesystem-order-dependent fresh installs. `029_plan_based_test_users.sql:11` contains hardcoded `DELETE FROM iam.memberships WHERE user_id='fbae762d-...'`. pgvector installed but 0 embeddings + silently-missing HNSW index. 12 users without memberships. Backtest tables duplicated across research/trading schemas.
- **PASS:** exact DECIMAL(18,8)+CHECKs on all finance tables; idempotency unique indexes present with **zero duplicates found live**; 19 hypertables with compression+retention jobs scheduled; integrity spot-checks clean (no orphaned signals, calibrated_probability NOT NULL on all rows).

Row counts: ticks 10,559,418 · candles 2,763,033 (2020→now) · users 17 · sessions 343 · subscriptions 5 · payments 1 · commission_ledger 1 · payouts 0 · audit_events 344.

---

## 5. WINDOWS AGENT / MQL4-MQL5

- **[P0-WA1] Auto-updater semver comparison is lexicographic strings** (`windows-agent/internal/updater.go:78-84`): `"1.2.16" < "1.2.6"` → fleet stuck on old versions (server sees 1.2.6 AND 1.2.16 simultaneously — confirmed). Secondary: below-minVersion *aborts* update instead of forcing.
- **[P1-WA2] Pre-trade spread/cost vetoes are dead code in BOTH EAs** — `PAT_CheckSpread()` defined, never called from ExecuteBuy/Sell (mt5:166-189/975-1128; mt4:158-180/1051,1089). Cost realized before vetoed.
- **[P1-WA3] Update chain unsigned** — manifest Signature "(future)" never verified; checksums same-origin; shipped artifacts unsigned (build.ps1 signing optional, install.ps1 warns).
- **[P1-WA4] Device identity fake:** ECDSA key generated then discarded; only UUID written to `device.key`; nothing signed anywhere (agent.go:512-536).
- **[P1-WA5] Delivery dishonesty:** `"FORWARDED_TO_EA"` ack sent when IPC file written, not executed; no offline queue (signals dropped while disconnected); MQL `PAT_Append` opens FILE_WRITE truncating → lost ticks/acks (mq5:1168-1183); racy read-clear cycles.
- **[P1-WA6] Hardcoded license keys committed as EA input defaults** (mq5:19, mq4:15); license_key embedded in world-readable PAT_ticks.txt.
- **[P2]** `CalculateLotSize` clamps UP to broker min-lot (oversize risk on small accounts); flat local 300s TTL vs server-authoritative expiry; downloads vhost alias/nested-regex root bug; updater apply failures silent, RollbackUpdate never called; latency_ms measures marshal time not RTT.
- **PASS:** EA genuinely lightweight-adapter compliant (no predictive intelligence, no private keys, respects stop/freeze levels); trade comment convention consistent.

---

## 6. INFRASTRUCTURE / MONITORING / BACKUP

- **[P0-IN1] Prometheus collects NOTHING:** scrape targets `127.0.0.1:13081/:13080` point INSIDE the prometheus container → both targets DOWN since deployment (up==0, zero samples). Engines export fine (72 `pat_*` series verified live).
- **[P0-IN2] Alerting absent:** prometheus.yml references rules.yml that doesn't exist/isn't mounted; no Alertmanager; no Grafana alerting. Cert expiry (~83 days), engine-down, backup-age would all fail silently.
- **[P0-IN3] Backup posture non-functional:** scripts exist (`scripts/backup/*`) but **nothing schedules them**; newest dump 2026-08-18 (6+ days), one 0-byte failed dump retained, zero .sha256 files; `backup_restore_validate.py` SIMULATES restore with hardcoded passing results. RPO unbounded; restore never exercised.
- **[P1-IN4] ntfy push provider built+tested but never registered in main.go** — notifications dead code (token committed in realtime.env:85).
- **[P2-IN5]** Grafana dashboard queries nonexistent metrics (`gate_pass_count_total` etc.) vs real `pat_gate_veto_total` etc.; no healthchecks on status/prometheus/grafana/ntfy/nginx; zero resource limits; secrets in compose (JWT, grafana admin, ntfy token); http2 deprecation warnings (cosmetic); SSL certs expire 2026-11-15 (83d) with no monitoring.
- **PASS:** log rotation configured daemon-wide (20m×5); restart:always everywhere; healthchecks on 5 core services; security headers on all 5 vhosts; rate-limit zones consistent.

---

## 7. RUNTIME VERIFICATION (during audit)

| Check | Result |
|---|---|
| Containers | 10/10 Up; postgres/valkey/realtime/control/frontend healthy |
| Realtime /health | ok, agents connected, heartbeats flowing |
| System-health | postgresql/timescaledb/valkey healthy, ready=true |
| Signals | broadcasting to agents (ULTRA_SCALPING etc.), fingerprints+cooldowns in Valkey |
| Valkey | 19 keys: candles cache, market_state, snapshots, fingerprints, cooldowns, price_history |
| nginx | 502s eliminated (dynamic resolver), CORS single-header verified, WS 101 OK |
| Prometheus | ❌ both targets down, zero metrics |
| Backups | ❌ stale ≥6 days |

---

## 8. FIX PLAN & EXIT CRITERIA TO GO-LIVE

### Batch 1 — applied this session (see commits)
- [x] nginx dynamic upstream resolution + edge-CORS authority (earlier commits e921731..cdf93f8)
- [x] Frontend build-breaking bug + strategy-save token wiring
- [x] Prometheus targets → service names + rules.yml mount + baseline alert rules (engine/target down, cert expiry, backup age)
- [x] Real pg_dump backup executed + cron scheduling + checksum verify; simulator flagged invalid
- [x] Billing webhook HMAC signature + event-id idempotency; `subscription.active` removed from paid-events
- [x] Payout column fix (`requested_amount`), balance validation, payout_items creation, currency from wallet
- [x] Licensing IDOR user_id scoping (heartbeat/revoke/registerTerminal)
- [x] Agent WS: license enforcement (invalid license disconnected) + optional shared-token auth via `AGENT_WS_TOKEN`
- [x] `handleSignals`/resume real JWT verification
- [x] Bootstrap lookback weekend factor (H4/M15 warm-up)
- [x] Frontend: customInstance save, WS reconnect-timer cleanup, fabricated-default removal
- [x] Status page false compliance/incident claims removed
- [x] Updater semver numeric comparison + force-upgrade below floor
- [x] ntfy provider wired into engine notification manager

### Batch 2 — REQUIRED BEFORE LIVE TRADING (open items)
1. [ ] Rotate JWT_SECRET (operator-authorized; migrate device-auth pepper/AES derivation to independent keys first)
2. [ ] Remove committed secrets from git history + secret scanning in CI
3. [ ] Create 9 least-privilege DB roles; switch runtimes off superuser
4. [ ] Emergency-stop gate registered head-of-order + admin API
5. [ ] NewsGate honor NEWS_FAIL_POLICY (fail-closed) + flip ALLOW_TRADING; make 14 Go tests pass honestly
6. [ ] Entitled delivery end-to-end (populate Client.entitlements from verified JWT; P0/P1 buffered delivery; wire DeliveryManager; prove deliveries > 0 E2E)
7. [ ] Exact-decimal money math in control plane (start: clearEligible, invoice totals); commission crediting transactional
8. [ ] Signed update chain (manifest signature verify in Go agent; sign in build script)
9. [ ] Wire pre-trade spread/TTL vetoes in MT4+MT5 EAs; fix PAT_Append truncation; honest EXECUTED acks
10. [ ] Reconcile migration history drift (aliases) ; flow_features hypertable; remove hardcoded DELETE in 029
11. [ ] Restore drill against real data with measured RPO/RTO documented

**GO criterion:** all Batch-2 items closed with evidence + shadow/paper validation period clean. Until then the platform remains in shadow/paper posture — which is exactly what current configuration enforces (NEWS_MODE=PROTECT_ONLY notwithstanding the fail-open defect, no live order routing enabled).
