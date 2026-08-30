# Macroscopic Production-Readiness Audit — 2026-08-30

Scope: entire Predict-A-Trade stack (Go realtime, NestJS control, Next.js frontend, Python research, DB, Windows/MQL edge, docs, wiring).
Method: three parallel read-only audits (realtime signal path; control plane + DB + security; stale/fake/orphaned files) followed by immediate remediation of P0 findings.

**Verdict at audit time: NO-GO (5 P0 classes).**
**Post-remediation status: P0 code fixes applied and verified by build+vet+full go test (40 pkgs pass). Remaining items are listed with owners below."

---

## 1. P0 findings — status

| # | Finding (audit evidence) | Status |
|---|---|---|
| P0-1 | **EMERGENCY_STOP / KILL_SWITCH unauthenticated** (`gateway/http.go:112-113,1218,1243`) and broadcast-only — "halts trading" was not true server-side | **FIXED**: `requireAdminAction` wrapper (control-plane HS256 JWT, role ADMIN/SUPER_ADMIN, constant-time compare, fail-closed when JWT_SECRET unset); new `EmergencyHalt` engine state — processCandle returns immediately, `broadcastSignalToAll` suppresses everything, `/health` reports `HALTED` + `emergency_halt` detail; `/api/v1/admin/emergency-resume` (also admin-only) is the only clear path |
| P0-2 | **Main loop one-panic death + nil-persister panic** (`main.go:2454` recover on goroutine; `persister.GetDB()` unguarded inside `AllGatesPass`) | **FIXED**: per-message recover wrappers (`handleTick`/`handleCandle`) keep the pipeline alive; nil-guard downgrades EXECUTABLE→ADVISORY with `DEDUP_UNAVAILABLE_NO_PERSISTENCE` (fail-closed, never emits without durable dedup) |
| P0-3 | **v1.15.0 SL enforcement dead**: `checkPositionSLs` never called; `isAgentSuspended` never consulted; suspension bypassable by reconnect | **FIXED**: `AgentProvider.SetPositionSLCheckFn` hook fires on every broker snapshot → `checkPositionSLs` (CLOSE_POSITION for PAT magic range); suspension now enforced in `SetRiskCheck` delivery filter — suspended agents receive nothing for the process lifetime |
| P0-4 | **Fabricated "AI Verified" label** — every signal labeled AI Verified merely because `OLLAMA_ENABLED=true`; NO-TRADE signals claimed "DISABLED — ollama off" | **FIXED**: honest `NOT_AI_VERIFIED — no per-signal LLM verification performed` everywhere (`aiVerificationStatus`, `createNoTradeSignal`, `signal/engine.go`); related: stale ML/sentiment globals now reset per bar (a prior candle's ±2.25-score leak eliminated), LLM price-prompt no longer labeled news sentiment and runs once/bar not per-strategy |
| P0-5 | **ENABLE_SHORTS silently unenforced without ML** | **FIXED**: `shortsActive` enforced unconditionally — SELL candidates + SELL_CANDIDATE suppressed at generation with `shorts_disabled` reason regardless of ML state |

Fixed in this pass; evidence: `go build ./... && go vet ./...` clean, `go test ./...` 40/40 packages pass.

## 2. High-severity items NOT YET fixed (next work queue, in order)

1. **Agent/data WS endpoints unauthenticated in practice** — `AGENT_WS_TOKEN` present in neither `infra/env/realtime.env` nor `windows-agent.env`; docker publishes 0.0.0.0:13081/13091. Anyone reachable can inject ticks/snapshots that become signals. Required: set `AGENT_WS_TOKEN` in both env files + restrict publish IPs/firewall + roadmap to per-device crypto. (OWNER: ops, now; code support already exists — `agent_ws.go:355`.)
2. **Control-plane Jest red on Node 24** (10/13 suites fail with `require(esm)`) → CI gate is failing/ignored. Fix jest config/transform or pin Node 20 until green; then re-enable CI as blocking.
3. **`services/backtest-service` unauthenticated + publicly proxied** via `qr-temp.conf` (8088). Required: remove the public proxy, add token auth, enforce `user_id` scoping.
4. **Commission credited on license assignment, not validated revenue** (`admin.service.ts:370`); NOWPayments settlement never credits. Also float money in billing/nowpayments (`Number()` math). Required: move crediting to the settlement webhook path, switch to decimal.js consistently.
5. **`PAT_PAPER_EQUITY=10000` + raised position caps in production compose** with `LIVE_TRADING_AUTHORIZED=true` in env — paper equity seeds capital-protection anchors. Remove from prod or gate arming on verified broker equity.
6. **Calibration floor too low** — `oos_auc >= 0.5` admits coin-flip models; a 37-sample live model currently exists. Raise floor (e.g. >0.52 and n≥100 for live promotion).
7. **`GO_ENGINE_AGENTS_URL` unset** → control `/agents/status` hardcodes `127.0.0.1:13081` (fails in-container). Add to control.env: `GO_ENGINE_AGENTS_URL=http://realtime:13081`.
9. **Dead metric graveyard** (~25 registered-but-never-instrumented Prometheus metrics → misleading dashboards) and `/health` always "ok" (now also reports halt state; still not DB-aware — compose should use `/ready`).
10. **Realtime "AI sentiment" & ATEN provenance** — LLM context now honestly labeled, but sentiment engine remains nil-provider dead weight and ATEN/astro attaches AUTHORITATIVE to non-market data; flag or quarantine.

## 3. Database

- All 99 migrations now **applied to the live DB** (096–099 were pending; `migrate.sh` run completed clean).
- 099 extended: compression+retention (065 convention), and the `1.0.0/shadow` weight-version seed row so weights are auditable from day one (`trading.igs_weight_versions`).
- `MIGRATION_ORDER.md` tail de-corrupted; inventory section added (documents the 030–059 gap and PITR-only rollback honestly).
- Open: `migrate.sh down` unimplemented (PITR is the rollback path); several migrations mutate live financial/config rows (085/097) without versioned audit trail — future config changes should go through the versions tables.

## 4. Stale/fake/orphaned files (removal & fix queue)

Safe removals (next commit batch): `realtime/live-terminal` (16MB committed binary), `realtime/web/live.html`, `nginx/Dockerfile` (never built), `nginx/sites-enabled/` (never included), `live-dashboard/*.bak` + add `.dockerignore`, `docs/Dockerfile`, `frontend/_ua_tmp.tsx`, on-disk crumbs (`error.log`, `prompt.md`, `.cleanup_backup/`, `screenshot/`, stray Windows `device.key` file).

Risky / coordinate first: three divergent copies of compiled EAs (`.ex4/.ex5` — must rebuild from current sources and serve ONE canonical copy); `models/*.onnx` are 710-byte placeholders labeled `real-v1.0.0` (replace with honest `bootstrap-v1.0.0` or genuinely trained models — mislabeled provenance); `artifacts/go_live_evidence/*` (all DRY_RUN-labeled — keep, but the directory name overpromises); `check.md` (astro SOW, conflicts with actual scope — relocate under docs/ with honest name; NOTE: `realtime/internal/igs/*` comments cite check.md line numbers that don't match the current file — the IGS tier hierarchy is fully specified in `docs/architecture/IGS_tradingagents_design.md`, which is authoritative).

Docs fixed now: `AGENTS.md`/`AGENT.md`/`SKILLS.md` pointed at nonexistent `.agents/`, `.codex/`, `.mcp.json`, and a deleted SOW file — every agent session was following dead instructions. All now point at `.hermes/` reality.

Stale docs still open: `MANIFEST.md` (frozen at v1.10.1, wrong migration/test counts, systemd-era deploy instructions), `docs/README.md` service/migration counts, `docs/operations/DOCKER_DEPLOYMENT.md` "migrations auto-apply on first start" contradiction, `WINDOWS_AGENT.md` version header, `Makefile:164` → missing `scripts/security-scan.sh`, committed SMTP default in `docker-compose.yml:289` (rotate + move to env).

## 5. Honest strengths observed (preserve)

- Hard-gate ordering with fail-closed semantics, Executable-gated delivery, market-closed short-circuit, negative-edge veto, cooldown/dedup stack.
- NestJS: 22 modules fully registered; admin routes layered JWT+Roles+Permission guards; HttpOnly+CSRF+MFA-mandatory auth; webhook HMAC + idempotent payment events; payout reservation double-spend guards.
- Frontend: no fake data found in 78 pages; honest empty/error states.
- Command contract (EXECUTION_ACK/CLOSE_POSITION/EMERGENCY_STOP/KILL_SWITCH) present and byte-identical across all four planes.

## 6. Bottom line

The engine's safety skeleton was significantly weaker than documented (kill endpoints open, SL monitor unwired, fabricated AI label) — this pass closed those at the code level with regression tests passing. **Production GO remains NO-GO until items 2.1–2.5 (WS auth env + firewall, CI green, backtest service lock-down, commission trigger point, paper-equity removal) are evidenced.** Signal generation in sandbox/paper mode matches the documented behavior and is production-shaped; live trading arming must stay off until the finance-commission trigger and the paper-equity config contradiction are resolved.