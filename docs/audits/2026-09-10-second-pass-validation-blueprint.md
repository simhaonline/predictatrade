# Predict-A-Trade — Second-Pass Forensic Validation & Remediation Blueprint

**Date:** 2026-09-10 · **Baseline:** first-pass audit (5cae774) · **Repo state:** commit b172a43+
**Method:** every prior finding re-checked against current source and live runtime; four iteration-capped workstreams completed (feature-consumption sweep, calibration forensics, capital-protection depth, delivery/entitlement routing). No files modified in this pass except this blueprint.

## 1. REMEDIATION MATRIX (prior findings #1–#23 + Top-20)

Status legend: **VERIFIED** (still true, evidence current) · **REJECTED** (was wrong) · **PARTIALLY-VERIFIED** · **ALREADY-FIXED** · **NEEDS-DESIGN**

| # | Prior finding | Status | Current evidence (file:path:line) |
|---|---|---|---|
| 1 | Backtest CSV IDOR (guest/reset JWT accepted, no purpose check) | **VERIFIED** | control/src/modules/backtest/backtest.controller.ts:103-131 — `jwt.verify(token, JWT_SECRET)`; no `purpose` check; `userId=undefined` path skips ownership |
| 2 | proxy.ts forges admin cookie (unsigned decode) | **VERIFIED** | frontend/src/proxy.ts:8-19 — split+base64+JSON.parse, exp check only |
| 3 | Risk gates seeded PASS-never-expires (Exposure/Margin/ExecPermit + broker hydrate no ValidUntil) | **VERIFIED** | main.go:2037-2077 (seeding PASS w/o ValidUntil), 2085-2160 hydrate: "No ValidUntil — gate state never expires" |
| 4 | Data races: crossmarket Evaluate map-write under RLock; calibration Consumer unsynchronized | **VERIFIED** | crossmarket/engine.go:56 (RLock) + 105 (`e.drivers[name] = snap` inside loop); calibration consumer/loader maps + main.go:3172 reload goroutine |
| 5 | request_nonces unbounded (1M rows/269MB, TTL stored, no delete) | **VERIFIED** | device-auth.service.ts:700; repo-wide grep zero `DELETE FROM licensing.request_nonces` |
| 6 | User settings wipes displayName | **VERIFIED** | frontend/src/app/(user)/dashboard/settings/page.tsx:44 — `updateMyProfile({ displayName: "" })` |
| 7 | Control tests reach prod DB | **VERIFIED** | operations.service.spec.ts:7-15 (DB_URL from env); infra/env/.env host=`postgres` |
| 8 | Dedup fail-open (Valkey dup-check error → metric only; EXECUTABLE dedup DB error swallowed) | **VERIFIED** | main.go:4816-4820 (`if dupErr != nil { observability...Inc() }` then continues), main.go:5154 `_ = QueryRowContext` |
| 9 | EA dedup memory-only single-slot | **VERIFIED** | MT5: g_lastExecutedSignalID only; zero GlobalVariable persistence for signal IDs |
| 10 | Goroutine-per-tick/per-gate fan-out; UnregisterAgent leak | **VERIFIED** | main.go:3651-3657, 5201-5209; agent_provider.go:940 (zero callers of UnregisterAgent) |
| 11 | Calibration models no-skill (AUC 0.36-0.50) | **UPGRADED+ROOT-CAUSED** | live JSONs (shadow ANY-TP labels) AUC 0.36-0.50; **walk-forward report (real signal labels, chronological folds) shows SS 0.579/US 0.546/SW 0.558** — label source is the defect, not the score. Label mismatch verified: JSON target=`TP1_HIT` but code trains ANY-TP (live_calibrator.go:212-214); TREND_SWING 15% base rate + n=339 (imbalance); single-pool in-sample fit; shadow fills are synthetic slippage-free |
| 12 | Cooldown only on AllGatesPass | **VERIFIED** | main.go:5192 (cooldown set post-gate-pass only) |
| 13 | MasterNode ingest no-retry drops on 5xx | **VERIFIED** | MT4:1721-1727 (401 retry only), non-200 → `g_ingestErrCount++` drop |
| 14 | No max-drawdown halt | **VERIFIED** (equity floor also absent) | recovery/manager.go + capital_protection.go (daily/weekly/monthly staged present; drawdown absent) |
| 15 | WS silent drops + dead priority branches + entitlements once-at-connect + no replay | **VERIFIED** | websocket.go:237-250 (identical branches), 282-303 |
| 16 | Dead packages oco/breakout/replay/maintenance | **VERIFIED with MERGE amendment** | 0 consumers at cmd/strategy/signal; replay implements SOW §12-19 historical regime replay (useful for replay validation) → MERGE replay into backtest-engine cmd; oco→hedging concepts; breakout/maintenance REMOVE |
| 17 | NATS unused | **VERIFIED** — REMOVE (NATS_URL commented docker-compose.yml:111-112; zero Go/TS imports; also unauthenticated 0.0.0.0:4222) |
| 18 | RBAC guards not global; logout unguarded; audit gaps (licensing lifecycle, markInvoicePaid, plans/flags) | **VERIFIED** | app.module.ts:79-82; auth.controller.ts:218-235; users.controller.ts:55,73 |
| 19 | JWT secret sprawl (4 read paths, dev fallback, device-auth derives AES key from JWT_SECRET, hand-duplicated env) | **VERIFIED** | jwt.module.ts:5,28; device-auth.service.ts:578,594-598 |
| 20 | Rate-limit gaps (@SkipThrottle on DeviceAuth+EdgePoll; billing webhooks unthrottled; raw @Body any) | **VERIFIED** | device-auth.controller.ts:7; edge-poll.controller.ts:29; licensing.controller.ts:42-83 |
| 21 | E-01 daily cap = batch truncation only; LIKE jsonb matching | **VERIFIED** | http.go:492-497; edge-poll.service.ts:130-138 |
| 22 | Telegram/WhatsApp no consumption allowlist; GET broadcast; no 429 handling | **VERIFIED** | telegram-bot.service.ts:74 (incoming bypass adminChatIds; used only at :117 broadcast); whatsapp-gateway phone-keyed |
| 23 | OTel absent; restore-test evidence absent; compose limits absent; Valkey maxmemory=0/noeviction | **VERIFIED** | no otel in go.mod/package.json; scripts/backup_restore_validate.py exists, no logged run; live CONFIG GET |

**NEW P0 DISCOVERED IN SECOND PASS:**
| # | Finding | Evidence |
|---|---|---|
| P0-NEW | **Control-plane Halt Trading does not stop the engine.** Admin Risk Center → `/operations/halt-trading` writes `control.platform_operations` ONLY. The engine hot path + delivery gate exclusively on its own `globalEmergencyHalt` (engine `/api/v1/admin/emergency-stop`). Zero Go references to `platform_operations` (repo-wide grep empty); edge-poll checks only license state. An operator pressing "Halt Trading" in the UI halts NOTHING in the signal→EA path. Emergency-stop/resume (http.go:189-191) is a separate endpoint the UI never calls. | main.go:134-137, 615-621, 3667; operations.service.ts:20; risk-center/page.tsx:126-132; admin-api.ts:52 |

**UPGRADES to KEEP (deeper than first audit credited):**
- Capital protection is nested + staged: DailyLossGate (capital_gates.go:296-360) enforces daily 5% / weekly 8% / monthly caps with recovery-band vs hard-halt ×2 multiplier, **fail-closed on unknown PnL** (`pnl_state_unknown` veto); DynamicRiskTapering below $200 equity; staged Soft(4%)/Hard(6%) halt; plan caps applied to live gates (main.go:5389-5391). Equity-floor + max-drawdown remain the only missing pieces.
- Walk-forward infrastructure exists and is BETTER than the live calibrator: scripts/oos_walkforward_calibrate.py (rolling train/test folds, real-signal labels `TP1_HIT_BEFORE_SL`, n=1.6k-4.7k, research Brier/ECE/Wilson in patresearch/calibration.py:7-30).
- IGS = "Institutional Gold Signal" composite (igs/engine.go:15-21) — the MasterScoring-equivalent, shadow-mode default, wired.

**NEEDS-DESIGN (not defects, gaps):** reconciliation chain depth (ack≠fill today — see §5), entitlement-routed delivery workers, OTel tracing, analytics surfacing, promotion pipeline formalization.

## 2. P0/P1 capital-loss sweep (new checks this pass)

- **Lot/risk math:** verified fail-closed (floors to lot step, refuses below min lot, caps at max) — EA (MT4:799-807, MT5:838-846) + server sizing (risk/sizing.go). No defect found.
- **Margin-level:** ingested (`margin_level` stored via device-auth.service.ts:401) and MarginGate is state-driven bool (implementations.go:316-350) — **but freshness handling is exactly the P0-3 seeding defect**: gate PASS-never-expires means a dead broker leaves margin protection armed on stale faith. Fix inside P0-3.
- **Equity floor / margin-level %:** absent — ADD (P1).
- **Spread/commission/swap/slippage:** TotalCost + Slippage + Profitability gates verified in registry order (gates.go:107-125); EA commission capture verified (MT5:508).
- **Partial fills / retcodes:** EA side verified (MT4 split-ticket classification; MT5 retcode paths) — OK; server-side partial-fill reconciliation absent — NEEDS-DESIGN (P1).
- **OCO:** package dead; hedging manager covers the hedge case; OCO merge decision: fold `oco/group.go` semantics into hedging manager (TP3-stop pairing) in P2, or remove.
- **Breakeven/trailing:** stage machine verified (stage GVs survive reload, ATR-trail remainder, breakeven at TP1) — KEEP.
- **Market gaps / news spikes:** WrongSideSL + StopHunt + News BE-4 fail-closed verified; gap protection beyond spread gate — NEEDS-DESIGN (P1, gap-vs-stop slippage ledger).
- **Emergency halt/flatten:** engine-side halt verified REAL (main.go:615-621) but UI-reachable halt is the P0-NEW gap. FLATTEN (close-all) exists only as an EA concept — server-initiated flatten NEEDS-DESIGN (P1, dangerous; require signed operator command + EA-side two-man rule).
- **Simultaneous strategy exposure:** Exposure gate + PositionCaps verified; correlated-exposure beyond XAUUSD (same symbol across strategies) — partially covered by total-exposure; per-strategy cap present. OK.
- **Broker/terminal disconnect recovery:** EA GVs + history poll + reclaim/dead-letter verified; server-side gate staleness = P0-3 fix.

## 3. Production-readiness scores

**Overall: 71/100** (production-viable with P0 batch; the platform's honesty/fail-closed architecture is its core strength)
- Trading Accuracy & Capital Safety: **68** (calibration label defect; halt gap; drawdown absent)
- Security: **62** (IDOR, forged-cookie edge, RBAC non-global, JWT sprawl)
- Reliability/Ops: **74** (backups live, CI real; healthcheck/limit gaps)
- Data Integrity: **80** (compression/retention/dedup verified)
- Delivery: **70** (channels work; queue/ack/routing immature)
- Observability: **62** (metrics up, 2 alerts, 1 dashboard, no OTel/Loki)
- Code Quality: **78** (zero TODO debt, tests 39 Go suites + 198 Nest green, tsc clean both planes)
- Documentation: **82** (runbooks current, audits committed)

## 4. SAFE PHASE-1 CHANGESET (atomic, high-confidence, rollback-safe)

**Implement immediately (all verified current, low blast radius, test-covered):**
1. request_nonces cleanup: engine-side or control-side 5-min loop `DELETE FROM licensing.request_nonces WHERE expires_at < now()` (batched 5k, idx_request_nonces_expires exists) + retention metric + alert at >50k rows. Rollback: disable via env flag NONCE_CLEANUP_ENABLED.
2. Test isolation: jest `testPathIgnorePatterns` for the 3 DB suites + rename their connection gate to REQUIRE explicit `TEST_DATABASE_URL` (never DATABASE_URL). Rollback: config revert.
3. JWT unification: jwt.module becomes the only read path; device-auth stops deriving AES/HMAC from JWT_SECRET (new DEVICE_SECRET env, migration re-encrypt in maintenance window). Regression tests: auth spec green; rollback: env re-point (secrets unchanged in value, only source unified).
4. Healthchecks ×10 + depends_on (compose-only change; rollback = compose revert).
5. Backtest download purpose check (`payload.purpose === 'access'` + require sub; password-reset/guest JWTs → 401) + regression test with forged purpose tokens.
6. proxy.ts HMAC verification via Web Crypto (fail-closed) — middleware-level; regression: login flow E2E.
7. User settings fix: remove the `displayName:""` call; wire POST /auth/change-password.
8. crossmarket RLock→clone-under-lock (or upgrade to full Lock for the mutate branch); calibration Consumer RWMutex. `go test -race` must stay green.

**Requires shadow/benchmark validation before enabling (do NOT ship blind):**
9. Cooldown-on-veto (changes signal frequency — shadow for 2 weeks).
10. Calibration rewiring to real-signal labels (TP1_HIT_BEFORE_SL) + walk-forward promotion (use existing script as the model; Brier/ECE gates) — ship as shadow models first.
11. Max-drawdown + equity-floor + margin-level thresholds (capital behavior changes — walk-forward + demo).
12. EA persistent dedup GV (behavior change in field fleet — staged rollout, one device canary).
13. Halt-bridge (control halt → engine) — design decision needed: engine polls platform_operations every 5s (simple, adds control-plane dependency to hot path) vs control calls engine endpoint (adds engine dependency to control). Recommended: engine polls platform_operations every 5s via its existing DB pool read-only — no new network dependency (same Postgres), atomic, fail-closed on DB error. **This is the Phase-1 flagship patch.**

## 5. PHASE-1 SEQUENCE (exact)

1. P0-NEW halt-bridge: engine poller `opsHaltPoller` (every 10s, `SELECT EXISTS(SELECT 1 FROM control.platform_operations WHERE operation_type='HALT_TRADING' AND status='ACTIVE')` → globalEmergencyHalt.Activate("ops_halt")/Resume) + tests + deploy both HA peers + UI "Halt Trading" now actually stops delivery (verify via /engines/diagnostics).
2. request_nonces cleanup loop + monitoring metric.
3. Test isolation (jest ignore + explicit TEST_DATABASE_URL gate) → full `npm test` green with no DB.
4. JWT unification (single provider; DEVICE_SECRET split) + re-encrypt device secrets.
5. Backtest download purpose check + tests.
6. proxy.ts HMAC verification + login-redirect regression test.
7. User settings change-password + no-wipe fix.
8. Gate seeding ValidUntil (+5min, refresh loop extends) for Exposure/Margin/ExecPermit.
9. Race fixes (crossmarket clone-under-lock; calibration RWMutex) + `go test -race` green.
10. Healthchecks + depends_on (compose) for the 10 services.
Each step: commit → gate (tsc/lint/tests/race) → deploy control×2/realtime → verify live. Rollback = git revert per step (all independent).

## 6. 30/60/90-day roadmap
- **30d:** Phase-1 above + weekly/monthly cap verification in staging + calibration shadow-models v1 (real-signal labels) + analytics endpoint skeleton.
- **60d:** calibration promotion (Brier/ECE gates) → ProbabilityCalibrated live; WS metrics+eviction; RBAC globalization + audit backfill; EA canary persistent-dedup rollout; worker pools + COPY batching; analytics dashboards v1.
- **90d:** drawdown/equity/margin-level protections shadow→live; WhatsApp Cloud handshake + queue-backed delivery with lifecycle updates; reconciliation chain in admin; PITR restore rehearsal documented; promotion pipeline formalized (DEV→BACKTEST→WALK-FORWARD→MONTE-CARLO→SHADOW→LIMITED LIVE→FULL LIVE).

## 7. Final target architecture
Five planes preserved; single realtime authority; queue-backed channel delivery (Telegram/WhatsApp Cloud/Discord) with per-recipient status + entitlement routing + lifecycle-aware templates; platform_operations as the ONE authoritative halt (bridged into engine, fail-closed); calibration promoted only via walk-forward chronology + Brier/ECE/PR-AUC gates; analytics drill-down portfolio→strategy→trade→signal→gate evidence (audit.score_components already stores gate-by-gate rows — surface them); SPOFs documented with rebuild runbooks, HA added only at measured need; NATS removed; no Kubernetes until load data demands it.