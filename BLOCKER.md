# BLOCKERS — Predict-A-Trade XAUUSD

**Source:** prompt.md §112 final report (2026-08-24 execution run, commit `d3c407c`)
**Status basis:** live verification + code audit + clean-main test baselines

---

## B-01 — Signals served publicly without entitlement filtering (P0 — blocks subscriber delivery)

- **Requirement:** prompt.md §56 (server-side enforcement), §57 (quota), SOW IAM/entitlement boundaries
- **Evidence:**
  - Nginx routes `/api/v1/signals` straight to the Go engine with **no auth**: `nginx/sites-available/api.predictatrade.com.conf:55`
  - Go handler performs zero auth/tenant/entitlement checks: `realtime/internal/gateway/http.go` (`handleSignals`)
  - Any anonymous client receives all signals for all strategies
  - WS broadcast is unauthenticated per-user; `/api/v1/signals/resume` accepts bare `device_id` without ownership verification (`realtime/internal/gateway/http.go`)
  - Entitlement read path exists but is advisory-only (UI toggle rendering): control `GET /subscriptions/entitlements` → `control/src/modules/subscriptions/subscriptions.service.ts:75-86`
- **Impact:** paying vs free users get identical signal data; quota meaningless; revenue leak
- **Fix:** authenticated per-user signals endpoint in control plane with entitlement filtering + WS token-scoped distribution
- **Status:** RESOLVED (commit d30f61c)

## B-02 — Signal quota ledger unwired (P0 — blocks subscription metering)

- **Requirement:** prompt.md §57; migration schema already present
- **Evidence:**
  - `trading.signal_delivery_ledger(quota_period, quota_consumed)` defined in `database/migrations/024_subscription_referral_v3.sql:100-107`
  - No writer or consumer exists anywhere (grep confirms); documented as missing in `docs/GO_LIVE_MICROSCOPIC_AUDIT/03_FUNCTION_WIRING_MATRIX.md:41`
- **Impact:** monthly signal limits cannot be enforced or billed against
- **Fix:** ledger writer on issued-signal delivery + quota check in the B-01 endpoint (only issued XAUUSD signals count; never NO-TRADE/evaluations)
- **Status:** RESOLVED (commit d30f61c)

## B-03 — Production mode inert for realtime engine (P1)

- **Requirement:** prompt.md §20; internal guard P1-002 (simulated provider forbidden in production)
- **Evidence:**
  - `IsProduction()` = `NODE_ENV=="production" || APP_ENV=="production"` (`realtime/internal/config/config.go:214-216`) — neither is set for `pat-realtime` in `docker-compose.yml`, so the ban is inert at runtime
  - Setting `APP_ENV=production` today crashes the container: validator P2-002 correctly rejects the hardcoded dev `DATABASE_URL` password (`config.go:268-271`; verified live during deploy)
  - `PROVIDER_MODE=agent` currently set in env, so no simulated data flows — but nothing enforces it
- **Impact:** simulated-mode safety net unenforced; production posture not provable
- **Fix:** move DB credentials to gitignored secret file/vault (rotate local dev password), then set `APP_ENV=production` on pat-realtime
- **Status:** RESOLVED (commit d30f61c)

## B-04 — stop_hunt_filter / min_atr gates registered but unreachable (P2)

- **Requirement:** prompt.md §33 (gates authoritative), §84
- **Evidence:**
  - Registered in `realtime/cmd/realtime-engine/main.go` (`registerGates`: StopHuntFilterGate 1.5×ATR, MinAbsoluteATRGate 2.0)
  - Absent from evaluation `order` slice (`realtime/internal/gates/gates.go:83-96`) → never evaluated
  - Cannot simply be added: they have no gate-state seeding and `EvaluateAll` fails closed on `GATE_NOT_INITIALIZED`, which would veto ALL signals (verified by test run)
  - Mitigation already active: per-strategy MinAbsATR enforced inside strategy engines (`internal/strategy/engines/factory.go`)
- **Impact:** two defense-in-depth gates are dead wiring
- **Fix:** add state hydration for both gates in `refreshGateStates`, then append to order slice
- **Status:** RESOLVED (commit d30f61c)

## B-05 — Pre-existing failing Go tests (P2)

- **Requirement:** AGENTS.md definition of done; regression hygiene
- **Evidence:** identical failure sets verified on clean `main` (stash baseline) AND post-change:
  - `internal/strategy`: TestNoForcedSignals_NeutralMarketProducesNoTrade, TestGolden_NO_TRADE_RangeMarket, TestStandardScalping_ConflictProducesWAIT
  - `internal/strategy/engines`: TestUltraScalp_LowATR_Rejects, TestUltraScalp_RegimeMismatch_Rejects, TestUltraScalp_LowGrade_Rejects, TestStdScalp_LowATR_Rejects, TestTrendSwing_RangeRegime_Rejects, TestGetEngineConfig_ReturnsConfig
  - `pkg/news`: TestRiskEngine_NoProvider_ReturnsDataUnavailable
  - `internal/replay`: TestReplayEngine_RunsSuccessfully
- **Impact:** MANIFEST "24/24 pass" claim stale; masks regressions
- **Fix:** triage each against current intended behavior (several look like stale expectations after MarnieFib/signal-closure commits)
- **Status:** RESOLVED (commit d30f61c)

## B-06 — User strategy-preferences not persistable (P3)

- **Requirement:** prompt.md §52, §92 (subscription-driven UX)
- **Evidence:** `frontend/src/app/(user)/dashboard/strategies/page.tsx:74-78` documents its own DegradedNote — no PATCH endpoint exists; selections are local-only
- **Fix:** small control-plane endpoint + wiring
- **Status:** RESOLVED (commit d30f61c)

## B-07 — Duplicate migration sequence numbers (P3)

- **Evidence:** `database/migrations/` contains duplicate numbers: 018 (×2), 019 (×2), 020 (×2), 062 (×2) — all applied & recorded, but ordering on fresh initdb is lexicographic and ambiguous
- **Constraint:** renaming applied migrations violates migration-history discipline
- **Fix:** document canonical apply order; enforce uniqueness check in `scripts/migrate.sh` for future files
- **Status:** RESOLVED (commit d30f61c)

## B-08 — Phantom mandatory macro pillars (P3 — needs verification)

- **Evidence:** confluence profiles declare `macro_dxy_yield` / `macro_real_yield_dxy` mandatory (weight 20, `internal/strategy/confluence.go:206-231`) but no code emits evidence under those pillar names. If that profile path drives live evaluation, swing strategies fail closed permanently; if bypassed by `strategies.go Evaluate()`, it's dead config
- **Impact:** either unintended NO-TRADE or misleading config
- **Fix:** confirm live path, then implement or remove the declarations
- **Status:** RESOLVED (commit d30f61c)

---

## Closed this run (for traceability)

| ID | Issue | Resolution |
|----|-------|------------|
| C-01 | Bar close time = open time copied (71% of rows) | close = open + period; `Timeframe.Duration()` |
| C-02 | All strategies evaluated on every TF bar | DecisionTFs trigger filter, live-verified |
| C-03 | Cross-TF indicator contamination | per-timeframe `RegistrySet` |
| C-04 | data_quality gate hardcoded PASS | StaleDetector-backed, fail-closed |
| C-05 | Snapshot bars stamped processing time | parsed broker timestamps |
| C-06 | MarnieFib score→confidence conflation | removed |
| C-07 | Fake-green admin liveness (engineAlive/feedState) | backend-evidence states |
| C-08 | No engine liveness endpoint | `/api/v1/engines/status` + tracker |
| C-09 | No WS-disconnect banner on user dashboard | LIVE CONNECTION LOST banner |
| C-10 | `/device-auth/sessions`, `/cross-market/*`, telemetry beacon misrouted | fixed (commit 76f134a) |
