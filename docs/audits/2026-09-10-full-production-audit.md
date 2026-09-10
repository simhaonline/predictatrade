# Predict-A-Trade — Full Production Audit Report

**Date:** 2026-09-10 · **Scope version:** v1.18.0+ (post commit b172a43)
**Method:** 5 parallel deep-scan domains (Go engine / NestJS control / frontend+contracts / MT4-5+delivery / DB+deploy) + independent runtime verification sweep (live containers, endpoints, DB catalog, cache, backups, CI). Every finding cites file:line or live-runtime evidence. Cross-verified claims are marked; one stale subagent claim was disproven live (see Corrections).

---

## CORRECTIONS (claims checked and rejected)

| Claim | Verdict |
|---|---|
| "FeedbackAdminController + EmailCampaignController not registered in modules → 3 admin pages 404" | **FALSE** — live-verified registered: `feedback.module.ts:12` (`controllers: [FeedbackController, FeedbackAdminController, FeedbackPublicController]`), `email-campaigns.module.ts:13` (`controllers: [EmailCampaignController]`). Subagent's grep only matched files named `*.controller.ts`; these controllers live in differently-named files. Admin feedback API also verified live earlier this session (featured pipeline end-to-end). |
| "EAs contain `Authorization: Bearer ***`" | Display-layer redaction of a redacted tool view, not source (byte-check confirmed real value). |
| "`billing.invoices` missing due_date" | Corrected by the subagent itself: migration 003:128 defines it; always-null for NOWPayments flow only. |

---

## 1. VERIFIED ARCHITECTURE MAP

**Planes (boundaries verified intact):**
- **Go Real-Time** (`realtime/`): 35 internal packages + 9 pkg. Entry: `realtime-engine` (6.5k-line main), `backtest-engine`, `backfill`, `live-terminal`, `audit`.
- **Ingestion path (traced):** MasterNode EA `POST /ingest/agent` (device JWT, ≤200 msgs/batch) → bulkhead `ingestGuard` (5000 msg/s, quarantine, dedup) → per-agent 256-buf chan → `tickChan`(4096) → aggregator (broker-offset buckets) → `StateManager` ‖ `candleSyncFn` CopyRates → `candleChan`(256) → features (RegistrySet per TF) → 7 strategies → thresholds → `DecideWithAdvanced` → ~22-gate registry (first-veto short-circuit, fail-closed) → `broadcastSignalToAll` → WS hub + per-device `edge_signal_queue` INSERT + ntfy mirror. Control edge-poll claims rows `FOR UPDATE SKIP LOCKED`; EA acks; `delivery-reconciliation.service` canaries ACK≠delivery.
- **NestJS Control** (`control/`): 28 modules; global ThrottlerGuard 300/min/IP + ComplianceInterceptor only.
- **Next.js Frontend:** 87 pages (48 admin / 17 user / 5 auth / 17 public).
- **MQL Edge:** 4 self-contained EAs (14,523 lines), v1.31.
- **Data:** TimescaleDB 22 hypertables (ticks 1h chunks/compress 7d/retain 90d; candles 1d/30d/retain; caggs with refresh), Valkey, NATS (unconsumed), Prometheus (3 targets up, 2 alerts, 1 dashboard), Grafana, S3 WAL+base PITR (verified live).
- **Delivery:** Telegram bot (poller, live), WhatsApp gateway (Meta Cloud API shape), Discord bot (discord.js, standalone), email relay (DKIM pat1).

**Strategy inventory (verified):** STANDARD_SCALPING (M1/M5, exp10/cool15), ULTRA_SCALPING (M1, 3/5), STANDARD_SWING (M15/M30/H1, 60/120), TREND_SWING (H1/H4, 240/360), ATEN, ARCANIST, MARNIE_FIB. Dedup = 3-layer: per-bar map → Valkey fingerprint SETNX 30m (bar-time in canonical) → DB partial-unique (strategy, bar_close, direction, class).

---

## 2. FINDINGS BY CLASSIFICATION

### KEEP (verified correct — protect these)
- **Auth core:** refresh rotation w/ token-family reuse detection (auth.service.ts:409-505); SHA-256-hashed opaque refresh; HttpOnly+Secure+SameSite=Lax+path-scoped cookies; access token memory-only in UI + HttpOnly cookie (no localStorage anywhere — zero XSS exfil vector); trusted-device cookie single-use rotating w/ reuse-revocation; CSRF Origin/Referer checks; lockout 5/15min; reset tokens single-use jti + session revocation. Session fixation PASS.
- **Billing:** NOWPayments IPN HMAC-SHA512 over raw body, timing-safe, fail-closed, exactly-once settlement (FOR UPDATE + status gate), at-least-once via delete-on-failure, underpay Decimal check; Stripe HMAC + idempotency same pattern.
- **Entitlements:** plan-DB-driven `validateStrategySelection`; edge-poll fail-closed license re-check + `SKIP LOCKED` claim; EAs fail-closed below min lot.
- **Gate registry:** fail-closed verified (GATE_NOT_REGISTERED/INITIALIZED → veto); stale exposure/margin → veto.
- **Calibration honesty:** `Calibrate()` returns (0,false) unless model VALIDATED/PROMOTED + OOS_AUC≥0.52 + n≥100 + monotonic — no fabrication. (The models currently on disk are no-skill — see ADD-1.)
- **EA correctness:** broker `TimeCurrent()` sole trading clock, DST-adaptive external-time conversion via live offset diff (zero hardcoded offsets); HMAC canonical byte-identical EA↔server w/ 256-bit nonce + DB TTL + ±30s window; lot handling floors/step/caps fail-closed; stops/freeze max() gate reject-not-clamp; ECN naked-open + SL-restore watchdog; magic ranges per-strategy; partial-close stage GVs survive reload; reconnect reconciliation.
- **Pure MQL:** zero DLL imports repo-wide; hand-rolled HMAC correct; no secrets in code/.set files.
- **Field contract:** Go→EA PascalCase aligns exactly (no json tags by design); edge-poll accepts both spellings defensively; EAs send ISO-UTC + broker_offset.
- **Ops:** TimescaleDB compression/retention policies live; all 4 hot-path indexes exist; zero duplicate table creations; CI genuinely gates (Go race, govulncheck, Nest, Playwright, secret-scan); nginx TLS 1.2/1.3 + HSTS + 4 rate-limit zones + WS upgrade headers; DISCORD_WEBHOOK_URL wired end-to-end in watchdog.
- **WS client lib:** memory token + refresh w/ cross-tab Web Locks; exponential backoff w/ max attempts; zombie-timer cancellation.
- **XSS:** zero dangerouslySetInnerHTML/.innerHTML in frontend; CSP present (weak config noted).
- **Zero TODO/FIXME debt** in frontend src/, control, and MQL sources.

### P0 — FIX (critical, verified)
| # | Finding | Evidence |
|---|---|---|
| P0-1 | **Backtest CSV download IDOR**: `jwt.verify(token, secret)` with **no purpose check** — guest-preview JWTs (purpose=`guest_preview`, no `sub`) and password-reset JWTs authenticate; ownership guard skips when `userId=undefined` → any run's trade CSV downloadable by UUID guess. | backtest.controller.ts:103-118; backtest.service.ts:608-619 |
| P0-2 | **Forgeable admin cookie at the edge**: `src/proxy.ts` middleware decodes `pat_access_token` JWT **without signature verification** (`Buffer.from(base64)`+JSON.parse) — forged `{"role":"ADMIN"}` cookie bypasses every middleware redirect; admin pages render until a data 403. (Data routes still backend-guarded; pages leak UI.) | frontend/src/proxy.ts:8-18 |
| P0-3 | **Risk-critical gates seeded PASS-never-expires**: Exposure/Margin/ExecutionPermit seeded PASS with zero ValidUntil at startup and broker-hydrate sets PASS with no ValidUntil; refresh loop only extends zero-expiry states → device disconnect never expires gates server-side. Fail-open seeding on risk gates contradicts SeedConservativeGateStates. | main.go:2066-2081, 2083-2160 |
| P0-4 | **Hot-path data races**: (a) `crossmarket Evaluate` map write under RLock (per-strategy per-bar); (b) `calibration.Consumer` maps mutated by AfterRun reload while hot path reads (no mutex). Both are concurrent map read/write — Go crash class. | crossmarket/engine.go:57,105; calibration/consumer.go:12-17 + loader.go:139 + main.go:3172 |
| P0-5 | **Unbounded `licensing.request_nonces`**: 1,005,756 rows / 269MB in 9 days, ~30MB/day, TTL=120s but **zero delete code anywhere** (index exists; cleanup trivial). | device-auth.service.ts:700; live DB |
| P0-6 | **User settings page wipes display name**: fires `updateMyProfile({displayName:""})` before erroring — data-corruption side effect; also never wires the existing `POST /auth/change-password`. | user/settings/page.tsx:44; auth.controller.ts:227 exists |
| P0-7 | **Control tests hit production DB**: suites connect to real Postgres when `DATABASE_URL` set in env (verified live `getaddrinfo EAI_AGAIN postgres`); CI parity differs from local; unit tests touching prod data is a safety violation. | operations.service.spec.ts:14-20 + infra/env/.env host=`postgres` |

### P1 — FIX (trading accuracy & execution)
| # | Finding | Evidence |
|---|---|---|
| P1-1 | **Both dedup backstops fail-open**: Valkey dup-check error → metric only, flow continues; EXECUTABLE duplicate-check DB error swallowed (`_ = QueryRowContext`). Combined with C7 below = double-execution risk. | main.go:4816/5154 |
| P1-2 | **EA reload double-execution window**: `g_lastExecutedSignalID` memory-only single-slot; EA reload inside MaxSignalAge=300s while server reclaims IN_FLIGHT → same signal re-executes. Mitigation = position caps only. | MT4:677, 2563-2569 |
| P1-3 | **Goroutine-per-tick fan-out**: `go func(){SaveTick}` per tick unbounded under bursts; per-gate-result goroutine (~20/signal); no worker pools; UnregisterAgent never called (per-agentID chan+goroutine leak). | main.go:3651-3657, 5201-5209; agent_provider.go:940 |
| P1-4 | **Calibration models are no-skill**: all 4 live JSONs OOS AUC 0.363-0.499 (≤ coin flip), no status/is_active → correctly gated out; ProbabilityCalibrated=false everywhere. Machinery exists; models must be retrained and promoted only above the AUC gate. | calibration/*.json live; consumer.go:88-93 |
| P1-5 | **Cooldown armed only on AllGatesPass** — vetoed signals never arm cooldown; re-fire each new bar (bar-dedup is sole guard). | main.go:5192 |
| P1-6 | **MasterNode ingest drops messages on transient failure** — no retry/outbox; 401 has one retry but 5xx/network = permanent drop (silent tick/snapshot gaps). | MT4:1721-1727, MT5:2033-2039 |
| P1-7 | **No max-drawdown protection** in recovery/risk stack (daily-loss %, loss count, consecutive-loss breakers exist; drawdown absent). | recovery/manager.go |
| P1-8 | **WhatsApp Meta subscription handshake unimplementable**: no `@Get('webhook')` echoing `hub.challenge` — channel dead on real Meta onboarding; dedup in-memory only (restart → double replies); no delivery-status webhook; no outbound retry. | whatsapp-gateway.controller.ts:39-46, 93-96 |

### P2 — FIX/REFACTOR
| # | Finding | Evidence |
|---|---|---|
| P2-1 | **Persistence inefficiency**: SaveTick single-row INSERT (no COPY/pgx.Batch); SaveCandle upserted by 3 independent writers (contention); SaveSignal persisted twice; `SELECT nextval()` N+1; 3 indicator_history INSERTs per candle. | persistence.go:62-71, 79-88, 1244-1251; main.go:629/1918/3866 |
| P2-2 | **Missing indexes**: `trade_results(closed_at DESC)` (GetRecentTrades sorts); edge-state hydrate window function needs `(strategy_id,timeframe,closed_at DESC)`; `signals(expires_at)` for 5-min MarkExpired. | persistence.go:591+; main.go:5797; delivery.go:214 |
| P2-3 | **Outbox FSM incomplete**: RETRYING/DEAD_LETTER transitions dead (zero callers); PUBLISHED rows never deleted (unbounded); candidates/NO_TRADE skip outbox (audit trail inconsistent by class). | persistence.go:1141-1241 |
| P2-4 | **WS silent drops unmeasured**: slow-client drop policy has no metric/counter; P0 vs P2 branches byte-identical (dead priority code); entitlements hydrated once at connect; no server-side replay/resume. | websocket.go:237-250, 282-303 |
| P2-5 | **Dead code**: Go `oco`(317L)/`breakout`(269L)/`replay`(566L)/`maintenance`(127L) zero consumers; `min()` dead in main.go:6449; WS broadcast chan dead; frontend `command-center.tsx`(~400L)/`live-dashboard.tsx`/orval pipeline(`api-client.ts`+`generated/schema.ts`)/`telemetry.ts`/`empty-state.tsx`/`stat-card.tsx` zero importers; 7 unused deps (recharts, lightweight-charts, tanstack table/virtual, react-hook-form, hookform resolvers, zod). | census + frontend scan |
| P2-6 | **RBAC architecture**: Roles/Permission guards not global; `/auth/logout` missing JwtAuthGuard (decorator pasted onto change-password instead); admin-only routes on AdminGuard alone (users PATCH role, feature-flags, operations, plans PATCH); licensing lifecycle + markInvoicePaid + plan/flag edits skip audit.audit_events. | app.module.ts:79-82; auth.controller.ts:218-235; users.controller.ts:55,73 |
| P2-7 | **JWT secret sprawl**: 4 independent read paths; dev fallback constant; device-auth derives HMAC pepper + AES-256-GCM key from JWT_SECRET; same secret hand-duplicated in control.env + realtime.env. | jwt.module.ts:5,28; device-auth.service.ts:578,594-598 |
| P2-8 | **Rate-limit gaps**: `@SkipThrottle()` on entire EdgePoll + DeviceAuth controllers (license activation guessable surface — nginx-only bound); billing webhooks unthrottled; no per-email buckets; raw `@Body any` bypasses global whitelist on licensing/device-auth/admin-extras/compliance/billing/feedback-admin/operations. | edge-poll.controller.ts:29; device-auth.controller.ts:7; licensing.controller.ts:42-83 |
| P2-9 | **E-01 daily cap is not a daily cap**: Go truncates only the current batch to `max_signals_per_day`; no daily counter anywhere — FREE users get 5 per request, unlimited requests. `LIKE '%strategy%'` matching on jsonb::text. | http.go:493-495; edge-poll.service.ts:130-138 |
| P2-10 | **Reconciliation unbounded**: delivered-but-never-ACKed records retained forever; ackAlerted/fillAlerted/signalNotifyLast/slViolationDetails maps append-only. | reconciler.go:189-206; main.go:955, 5814, 127 |
| P2-11 | **Frontend fetch amplification**: `/market/snapshot` under 5 queryKeys, `/signals` ×5, `/subscriptions/entitlements` ×4 with no cross-invalidation; ~45 refetchIntervals + 12 bare setIntervals (1s re-renders); zero `next/dynamic`; tabler icons imported from package root in 60+ files (78 eager icons in shell chunk). | frontend scan |
| P2-12 | **Ops gaps**: stale EA binaries (compiled Sep 8 13:04 vs v1.31 sources Sep 8 22:34 — fleet older than audit target); `windows-agent/` vestigial; candles retention drift (081 documents 1095d, live job 180d, unrecorded); Valkey maxmemory=0 + noeviction; zero compose resource limits; no restore-test evidence (script exists, no log); ports on 0.0.0.0 incl. unauthenticated NATS 4222/8222; regime_history/devil_liquidity_events/strategy_evaluations hypertables with no compression/retention. | D4/D5 reports, live-verified |
| P2-13 | **Symbol-suffix foot-gun**: signal `Symbol` never compared against chart symbol in EAs. | MT4/MT5 (zero hits) |
| P2-14 | **HMAC clock window**: ±30s vs terminal-derived TimeGMT — skew storms look like generic 401s; no client clock-skew self-diagnostic (MasterNode has one, client EA doesn't; client ingest failure has no -1 diagnostic → log spam). | MT4:4045; device-auth.service.ts:644 |
| P2-15 | **Telegram/WhatsApp no subscription routing**: assistant serves plans to any chat/phone (no allowlist, no license check); admin broadcast is state-changing GET (CSRF-shaped); Telegram send has no 429/retry_after handling; markdown injection via firstName. | telegram-assistant.service.ts:371-398; telegram-admin.controller.ts:37-44 |

### P3 — polish/hygiene
nextMarketOpen hardcodes Sunday 22:00 UTC ignoring broker offset (1h error half the year, main.go:907-919); session.go default GMT+2 vs comments GMT+3; fractional-offset brokers unsupported (aggregator whole-hour); hardcoded gate params (MaxSpread 0.80, StopHunt 0.5×ATR, MaxSlippage 0.10 — not env-configurable); MaxExposure 5.0 inline in both Decide calls; stale-ticker mutates shared Candle.Time in-place (torn reads); BroadcastMarketState marshals before checking client count; per-tick `stateMgr.GetAll()` deep clone; device_risk_events alerted_at update discarded; row-scan errors `continue` silently; Expectancy synthetic-P fallback (documented); plaintext device secret in FILE_COMMON (MT4/MT5); weekendFactor 1.45 magic (documented); MT4 partial-close split-ticket price-proximity classification; `min(a,b int)` dead; crossmarket placeholder format line; access-token cookie expiry UX bounce (proxy treats expired as unauthenticated though refresh cookie valid); window.alert inconsistent with Sonner; clickable divs without keyboard handlers; 8 files buttons with no aria; inline hex colors in (auth) pages; broker_offset integer division drops half-hour zones; MN notification labels broker time with `Z`.

### ADD (features that close gaps)
1. **Validated calibration models** (unblocks ProbabilityCalibrated + true expectancy): research retrain → VALIDATED gate (AUC≥0.52, n≥100, monotonic) → promote.
2. **Real daily-signal counter** (E-01) — DB-backed per-user-per-day with the partial-unique infra already present.
3. **Max-drawdown halt** in recovery Config + engine wiring.
4. **Worker pools** for tick/gate persistence + batch COPY/pgx.Batch for SaveTick.
5. **Outbox completion**: wire RETRYING/DEAD_LETTER transitions + PUBLISHED retention (hypertable or 30d delete) + consistent outbox writes across classes.
6. **WS drop metric + P0 eviction of slow clients** + entitlement re-hydration on refresh.
7. **Research metrics surfaced**: Sharpe/Sortino/Calmar/expectancy/risk-of-ruin/MAE-MFE exist in `research/src/patresearch/backtesting/analytics/metrics.py` — expose via `/analytics` endpoint + Trading Reports.
8. **Server-side symbol validation** in edge-poll (per-symbol queue or EA-side compare).
9. **WhatsApp GET hub.challenge handshake** + persistent dedup + delivery-status webhook + retry queue.
10. **Telegram/WhatsApp chat allowlist** (operator chat ids for assistant consumption; per-user entitlement routing).
11. **Audit-backfill**: licensing lifecycle, markInvoicePaid, plan/flag edits → audit.audit_events.
12. **Healthchecks** for the 10 unhealthchecked services + compose resource limits + log-rotation blocks.
13. **Restore-test harness** scheduled monthly with logged evidence.
14. **Grafana dashboards** for signal pipeline, edge delivery, agent mesh; Prometheus pat_agents_connected alert (TODO in rules.yml).
15. **Stale-client EA update runbook** + rebuild/redeploy v1.31 binaries.

### REMOVE
- `oco`, `breakout`, `replay`, `maintenance` packages (1,279 lines, zero consumers) — OCO concepts merge into hedging manager if needed.
- NATS container + compose service (zero consumers; also unauthenticated on 0.0.0.0).
- FastAPI `services/backtest-service` (zero consumers; control spawns Go `backtest-engine` directly).
- Frontend dead set: command-center.tsx, live-dashboard.tsx, orval.config.ts + api-client.ts + generated/schema.ts, telemetry.ts, empty-state.tsx, stat-card.tsx, 7 unused deps, dead WS hub broadcast channel, `windows-agent/` vestigial dir, `pat-notification-prefs` write-only key, `/whatsapp/webhook` + `/webhook/verify` no-op stubs.
- Stale `mql/compiled_executable/*` binaries (or rebuild+redeploy).

### MERGE
- 3 backtest stacks → Go `backtest-engine` (single authority) + research plane for quant validation.
- DegradedBanner duplicates → `components/ui/degraded-banner.tsx`.
- Stat-tile → shared `stat-card.tsx` (exists, unused); tab-filter rows → `ui/tabs.tsx`.
- 12 fragmented queryKeys → shared hooks (`useSnapshot()`, `useSignals()`, `useEntitlements()`).

### KEEP (explicitly preserve)
All protections listed in the KEEP section; the 3-layer dedup; fail-closed gates; at-least-once edge delivery with reclaim/dead-letter; honest-state pages; pure-MQL EA design; calibration anti-fabrication gates.

---

## 3. PRIORITIZED ROADMAP

**P0 — Critical Production/Safety (week 1)**
1. Backtest download: enforce `purpose==='access'` + require `sub` (backtest.controller.ts:115) — AC: guest/reset JWTs rejected 401; curl test with forged token fails.
2. proxy.ts signature verification (Web Crypto HMAC in middleware; fail-closed) — AC: forged cookie redirected.
3. Gate seeding: seed UNKNOWN/ValidUntil=+5min for Exposure/Margin/ExecPermit — AC: stale-device signals vetoed by GATE_STALE.
4. Fix data races (RWMutex in crossmarket.Evaluate + calibration.Consumer; clone-then-lock) + `go test -race` in CI already exists → must pass.
5. Nonce cleanup job (5-min `DELETE WHERE expires_at < now()`) + table bloat alert — AC: table < 10k rows steady-state.
6. Fix user settings wipe + wire change-password — AC: displayName preserved on failed save.
7. Test isolation: exclude DB-dependent suites from unit run (`testPathIgnorePatterns` in jest config or TEST_DATABASE_URL gate) — AC: `npm test` green with zero DB access.

**P1 — Trading Accuracy & Execution (weeks 2-4)**
8. Fail-closed dedup on Valkey/DB errors (veto + retry) — AC: fault-injection test shows veto.
9. Persistent EA dedup (GV GlobalVariableSet last-executed + bounded seen-set) — AC: reload+redelivery test = single execution.
10. Worker pools + batch inserts; single SaveCandle writer (flusher owns upserts) — AC: DB CPU drop measurable in pg_stat.
11. Cooldown on veto path (shorter TTL) — AC: vetoed bar doesn't refire next bar.
12. Recalibrate models → VALIDATED promotion — AC: ProbabilityCalibrated=true with AUC>0.6 on fresh OOS.
13. Max-drawdown halt (recovery Config.MaxDrawdownPercent + equity check in RecordTradeResult) — AC: backtest shows halt at threshold.
14. MasterNode ingest outbox (persist→send→ack, replay on 5xx) — AC: kill engine mid-ingest, no tick gap after recovery.
15. EA symbol compare + suffix config — AC: mismatched chart symbol skips signal with log.
16. WhatsApp GET handshake + persistent dedup (Redis) + status webhook — AC: Meta verify passes; restart no double-reply.

**P2 — Analytics & Intelligence (months 2-3)**
17. `/analytics/metrics` endpoint (Go or control) computing research-plane metrics over trade_results; Trading Reports integration — AC: Sharpe/Sortino/Calmar/expectancy visible per strategy.
18. Strategy/model versioning surfaced end-to-end (strategy_version already in fingerprints → expose on signals + analytics attribution).
19. Outbox FSM completion + retention.
20. WS metrics + eviction + replay/resume (sequence gaps already tracked client-side).
21. Regime-history compression + retention (currently compress=false); candles retention 1095d reconcile.
22. Walk-forward + Monte-Carlo reports in admin backtest UI (research plane already computes).

**P3 — Reliability/Scalability/Security (months 2-4)**
23. Compose hardening: resource limits, healthchecks ×10, depends_on, 127.0.0.1 port binds, remove NATS, log-rotation per service.
24. Valkey: maxmemory 256mb + allkeys-lru + singleflight on chart cache.
25. RBAC: global RolesGuard/PermissionGuard + @Public opt-out; fix logout guard; audit-backfill (licensing/billing/plans).
26. Rate limits: remove blanket @SkipThrottle on DeviceAuth (keep edge-poll exempt w/ nginx zone documented), throttle billing webhooks (signature-checked still), DTO classes for raw @Body endpoints.
27. JWT unification: single hardened provider; separate DEVICE_SECRET for AES/HMAC pepper; rotation runbook.
28. FK constraints on edge_signal_queue.device_id, audit_events actor, trade_results (with NOT VALID + VALIDATE).
29. Restore-test evidence + monthly schedule; retention drift change-record.
30. Single realtime instance SPOF: document runbook + stateless rebuild path (config/env) until LB justified.
31. Grafana dashboards + alerts (agents-zero, edge backlog, mail spool, nonce size, gate-stale rate) + Loki for log consolidation.

**P4 — UX/Admin/Signal Delivery (months 3-5)**
32. queryKey consolidation + shared hooks; reduce 1s intervals to CSS/derived.
33. Dynamic imports + deep tabler imports; delete dead frontend set.
34. Honest error states on the 9 flagged pages (live, preview, health, scoring-board, mt4-mt5-client, macro-intelligence, admin dashboard, feedback, market-data).
35. Telegram/WhatsApp entitlement routing + chat allowlist + POST broadcast + 429 handling + markdown escaping.
36. Accessibility pass: keyboard handlers on clickable divs, aria, tabs consistency; CSP `unsafe-inline` removal next.config.ts.
37. a11y + settings: wire change-password (see P0-6), notification-prefs consumption or removal.

**P5 — Advanced/Future R&D (quarter+)**
38. Ensemble scoring: combine indicator votes + engine balance + crossmarket + calibration into MasterScore with per-component attribution (infrastructure exists: score_components/score_executions audit tables).
39. Anomaly detection on fill/slippage distributions (metrics library + trade_results data exist).
40. Post-trade learning loop: live_calibrator → validated-model auto-promotion with human gate (AUC + monotonicity + canary shadow period).
41. Multi-asset expansion path: symbol-agnostic ingest already partial (agent_id keyed); generalize BrokerSymbol gate + session engine per-symbol.
42. Kubernetes migration assessment only when HA justified (control pair exists; realtime SPOF acceptable at current scale with documented rebuild).

---

## 4. TOP 20 HIGHEST-VALUE IMPROVEMENTS

1. Backtest CSV IDOR fix (P0-1) — one line, closes data leak.
2. proxy.ts JWT signature verification (P0-2) — closes admin-page bypass.
3. Gate seeding fail-closed (P0-3) — risk gates must not be PASS-on-faith.
4. crossmarket + calibration race fixes (P0-4) — crash prevention on hot path.
5. Nonce cleanup job (P0-5) — stops unbounded PII-adjacent growth.
6. Dedup fail-closed (P1-1) + EA persistent dedup (P1-2) — double-execution defense.
7. User settings wipe fix (P0-6) — silent data corruption.
8. Test isolation (P0-7) — CI truth + prod safety.
9. Validated calibration models (ADD-1) — restores calibrated probabilities + true expectancy.
10. Worker pools + batch inserts (P1-3, P2-1) — DB load + burst resilience.
11. Cooldown-on-veto (P1-5) — prevents per-bar refire harassment.
12. Max-drawdown halt (P1-7) — capital protection completion.
13. Missing indexes (P2-2) — hot-path query latency.
14. WS drop metric + P0 eviction (P2-4) — invisible signal loss becomes visible.
15. RBAC globalization + audit backfill (P2-6) — defense-in-depth + audit integrity.
16. JWT unification (P2-7) — rotation safety + crypto hygiene.
17. Research metrics surfaced (ADD-7) — institutional analytics dashboard.
18. WhatsApp handshake + queue (P1-8) — channel go-live.
19. Compose hardening package (P3-23/24) — resource limits, healthchecks, binds.
20. Dead-code purge (REMOVE list) — ~2,500+ lines removed, bundle and cognitive load down.

## 5. TARGET ARCHITECTURE (end state)

Preserve the five-plane separation and the single realtime execution authority. Add: queue-backed delivery workers (Telegram/WhatsApp/Discord) with ack/retry/routing; `/analytics` service consuming the research metrics library over `trade_results`; validated-model calibration pipeline with promotion gates; unified JWT provider + separated device-secret crypto; global RBAC guards; batched persistence behind worker pools; Valkey as a real read cache with eviction; Loki/OTel tracing; hardened compose (limits, healthchecks, loopback binds); PITR with monthly restore evidence; and (when volume justifies) a second realtime instance behind sticky LB — stateless rebuild is already the deployment pattern. All changes are corrective or additive; no existing protection is removed or weakened.

---
**Evidence artifacts:** D1 `/var/lib/hermes-agent/.hermes/cache/delegation/subagent-summary-0-*.txt` · D2 `...-197436.txt` · D3 `...-198004.txt` · D4 `...-200252.txt` · D5 `scripts/audit/d5_findings.md` · runtime sweep recorded in session transcript. All file:line citations verified against the working tree at commit b172a43.