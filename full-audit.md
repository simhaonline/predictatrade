# Predict-A-Trade — Full Forensic Audit Report

**Audit date:** 2026-08-25
**Auditor:** Lead Systems Architect / Forensic Code Investigator / Production Readiness Auditor
**Scope:** Entire repository (Go realtime engine, NestJS control plane, Next.js frontend, Python research/quant plane, MQL4/5 EAs, Go Windows Agent, PostgreSQL/TimescaleDB/Valkey, infra/compose).
**Method:** Read-only static audit across all subsystems (parallel subsystem auditors) + live-DB verification of critical schema claims + exact source verification of every Critical/High finding before any fix.

> **Verification note (important):** Two "missing DDL" findings from the database audit were checked against the **running** database and are **FALSE POSITIVES** for the live system: `finance.ledger_entries` and `market.data_metadata` both exist with correct columns and indexes. They are not present in any versioned migration file (grep of `database/migrations/*.sql` returns nothing), so a fresh `migrate.sh` from scratch would not recreate them — a real **reproducibility/schema-drift** gap, but the live payouts/backtest-data paths actually function. Both are addressed below with `CREATE TABLE IF NOT EXISTS` migrations so deploys are reproducible without touching the live tables.

---

## 1. Executive Summary

| Dimension | Health | Critical Blockers |
|---|---|---|
| Core engine & quant logic | ⚠ Fair | Go engine SL/TP override drift (H); backtest MTF look-ahead (M); research: 3/5 strategies never trade (H); fabricated quant evidence (C) |
| Data & storage | ⚠ Fair | Finance ledger not in migrations (repro gap); `market.candles` no retention; candle cache keys omit source |
| Web/API/Comms | ❌ Weak | NOWPayments IPN signature mismatch (C, revenue); JWT secret dual-source (H); backtest cross-tenant IDOR (H); no rate-limit trust-proxy (H) |
| Business/User logic | ❌ Weak | Payout double-spend reservation (C); missing subscription state machine (H); insecure token cookie + no CSP (H) |
| External/Agents | ❌ Weak | Windows agent license **fail-open** (C); EA plan/strategy filter dead (C); arbitrary position close (H); no signed IPC/WS/replay (H) |
| Infra/Ops | ⚠ Fair | Duplicate migration numbers; compression w/o retention; partial observability |

**Overall health score: 4.2 / 10 — NOT production-ready.**
**Critical go-live blockers (must fix):** (1) fabricated quant-validation evidence; (2) NOWPayments IPN never settles; (3) payout double-spend / missing reproducible ledger; (4) Windows-agent license fail-open; (5) insecure JWT/frontend token handling.

---

## 2. Macroscopic Analysis (architecture coherence)

The system is a genuinely multi-plane architecture that *mostly* matches `AGENTS.md` boundaries:
- **Go realtime** (market→feature→strategy→risk→signal→delivery) is the authoritative trading plane and is well-built. All five strategies (STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING, MARNIE_FIB) are genuinely implemented and evaluated live.
- **NestJS control** is the SaaS/control plane: IAM, billing, referrals, commissions, payouts, licensing, backtest orchestration. SQL is parameterized (no injection). Money math uses `decimal.js`.
- **Python research** is the quant/backtesting plane. Engine, walk-forward, calibration, and reference math are present and mostly correct.
- **Next.js frontend** is server-truth-rendering (no indicator recomputation) — good. Backtest UI matches control endpoints field-for-field.
- **MQL EAs + Windows Agent** form the execution edge.

**Macroscopic gaps / risks:**
1. **Strategy/engine parity is broken in two places.** (a) The Go live path applies a *second*, hardcoded SL/TP matrix in `engines/factory.go` that overrides the geometrically-computed geometry from `strategies.go`/`candidate_geometry.go`, producing sub-`MinRR` geometry for the four primary strategies (the `strategy.StrategyConfig` values become dead at runtime). (b) The Python backtest adapter uses `atr_tp1 == atr_sl` for 3 strategies → permanent `POOR_RR` → they never trade. The reference math (`reference_math.py`) is correct, but the engine's `calculate_metrics` re-implements Sharpe/Sortino incorrectly and contradicts it.
2. **Large "intelligence" subsystem is dormant.** `signal/advanced.go` `DecideWithAdvanced` (the only wiring for `internal/rl`, `internal/sentiment`, `internal/hedging`, `internal/recovery`, `internal/adaptation`) has **zero callers**. Consequently `internal/oco`, `internal/replay`, `internal/maintenance` are never instantiated. The live runtime uses `signal.Engine.ProcessSignal`. This is not a bug per se (kept dormant by design) but is a maintenance/complexity risk and should be explicitly marked experimental or removed.
3. **Control↔realtime coupling is DB-only** (no streaming API client). License/entitlement changes are only enforced on next agent (re)connect. Acceptable but a latency gap for revocation.
4. **Financial truth is split and partially unverifiable.** `referral.commission_ledger` + `affiliate_wallets`/`payouts` are the real ledger; the intended general `finance.ledger_entries` exists in the live DB but is not in migrations → not reproducible. Payout completion currently depends on that table's presence (works live, not on fresh deploy).
5. **Fabricated go-live evidence.** `scripts/quant_validation.py` emits hardcoded Sharpe/Brier numbers under `--dry-run` and writes `quant_evidence.json`, which `final_go_live_check.py` accepts as passing evidence → **direct AGENTS.md violation**. This must be neutralized.
6. **Honest-state labeling gap on the frontend.** The relay `status` (LIVE/DEGRADED/STALE) is parsed by `WebSocketManager` but discarded; the UI shows "LIVE" from socket state only. A stale/replay feed is mislabeled LIVE.
7. **Go backtest-engine and Python service both write `trading.backtest_runs`** (different `strategy_mode`/`data_source` values) — coherent, but provenance labeling in the Go path is hardcoded wrong ("KAGGLE").
8. **No signed updates / no IPC or WS authentication / no replay protection** on the Windows Agent ↔ EA ↔ backend channel (security gap).

---

## 3. Micro-Level Findings (selected Critical/High; full list per subsystem below)

### 3.1 Go Realtime Engine
| ID | Sev | Category | Location | Issue | Proposed Fix |
|---|---|---|---|---|---|
| G1 | HIGH | Correctness/Config | `cmd/realtime-engine/main.go:2067` + `engines/factory.go:21-57` | Live path overrides strategy SL/TP with a second hardcoded matrix (e.g. StdScalp `OverrideSL 0.8`, `OverrideTPs [1.0,1.5,2.5]`) → geometry diverges from `strategies.go` and yields sub-`MinRR` R:R; `StrategyConfig` ATR multipliers become dead at runtime. | Make override opt-in via config flag default off, or load the same DB/config source as `strategies.go`. Reconcile so overrides cannot produce sub-`MinRR`. |
| G2 | HIGH | Concurrency | `gateway/agent_ws.go:119,125,159,328` | `done` channel closed in `Run()`, `DisconnectAgent()`, and the read-goroutine `defer` → "close of closed channel" panic on agent churn/reconnect. | Guard close with `sync.Once` on `AgentConnection`. |
| G3 | MED | Look-ahead | `backtest/runner.go:266` | Higher-TF candle treated as closed when `c.Time <= currentTime`; it may not have closed yet. | Require `c.Time.Add(tfDuration) <= currentTime`. |
| G4 | MED | Provenance | `backtest/persist.go:70` | `data_source` hardcoded `"market.candles (KAGGLE)"` though data is backfilled from Twelve Data (`TWELVEDATA`). | Store true source (e.g. `"market.candles"` + actual provider). |
| G5 | MED | Data completeness | `cmd/backfill/main.go` + `twelvedata_timeseries.go` | Single `/time_series` call; Twelve Data truncates large ranges → silent partial backfill. | Chunk range into sub-windows / follow pagination. |
| G6 | MED | Data integrity | `backtest/db_reader.go:34-43` | Reads all candle sources incl. non-closed bars. | Filter by intended `source` and `is_closed=true`. |
| G7 | LOW | Hygiene | `strategy/marnie_fib.go:71` | `log.Printf` every evaluation. | Use structured debug log. |
| G8 | LOW | Coverage | `cmd/audit/main.go:12-16`, `cmd/backtest-engine/main.go:24` | Audit/help omit MARNIE_FIB. | Add to lists. |

### 3.2 NestJS Control Plane
| ID | Sev | Category | Location | Issue | Proposed Fix |
|---|---|---|---|---|---|
| C1 | CRITICAL | Billing | `billing/nowpayments.service.ts:161-182` + `billing.controller.ts:86-90` | IPN uses a sorted `key:value|` canonical string HMAC, not the raw-body HMAC-SHA512 NOWPayments actually sends; controller passes parsed `req.body` not `req.rawBody`. Every genuine callback fails → payments never settle. | Verify `HMAC-SHA512(secret, rawBody)` vs `x-nowpayments-sig`; pass `req.rawBody`. |
| C2 | CRITICAL | Financial integrity | `payouts/payouts.service.ts:263` (+ `finance.ledger_entries` not in migrations) | Ledger insert inside `markAsPaid` tx; table missing from migrations → fresh deploy breaks payouts. On live DB the table *exists* so it works, but is not reproducible. | Add `075` migration (`CREATE TABLE IF NOT EXISTS finance.ledger_entries … UNIQUE(idempotency_key)`). |
| C3 | CRITICAL | Financial integrity | `payouts/payouts.service.ts:84-127,244-260` | Reservation locks CLEARED rows `FOR UPDATE` but never changes status → two payouts can reserve the same commission (over-payout / double-spend). | Transition CLEARED→RESERVED during request; exclude already-reserved rows. |
| H1 | HIGH | Auth | `common/jwt.module.ts:28` vs `guards/jwt-auth.guard.ts:18`, `backtest/backtest.controller.ts:59` | JWT secret read from `ConfigService` in signer but from raw `process.env` in guard/download → deploy-dependent 401s / broken downloads. | Use injected `JwtService` everywhere (single source). |
| H2 | HIGH | Abuse | `main.ts` (no `trust proxy`) + `app.module.ts` throttler | Without `trust proxy`, `req.ip` is the Nginx IP → per-IP throttle is a single global bucket. | `app.set('trust proxy', 1)`. |
| H3 | HIGH | Security/IDOR | `backtest/backtest.controller.ts` + `backtest.service.ts:100-145` | `GET /backtest/runs`, `/runs/:id`, `/runs/:id/download` have no ownership scoping → any authed user reads/ downloads another's runs. | Store `user_id` on runs; filter all queries by owner. |
| H4 | HIGH | Licensing | `device-auth/device-auth.service.ts:57-104` | Multi-device license effectively single-device (matches only first bound device fingerprint). | Match per-device; enforce `max_devices` count. |
| M1 | MED | Idempotency | `commissions/commissions.service.ts:95-159` | Credit insert not transactional; recurring/2nd-purchase commissions never generated. | Transactional credit; wire renewal events. |
| M2 | MED | Billing logic | `subscriptions/*` | No cancel/refund/pause state machine. | Implement lifecycle transitions. |
| M3 | MED | Abuse | `licensing/licensing.service.ts:648-668` | Public unauthenticated `validate` writes `device_activations`. | Separate read-only validate from activation. |
| M4 | MED | Billing | `billing/billing.service.ts:107` | `s.setup_fee` undefined (wrong column) → setup fees never billed. | Read `p.setup_fee`. |
| M5 | MED | Billing | `billing/billing.service.ts:309` | `checkout.session.completed` treated as payment. | Only `invoice.paid`/`payment.succeeded`. |
| M6 | MED | Ledger | `commissions/commissions.service.ts:310-343` | Partial reversal leaves inconsistent summary. | Track `reversed_amount`. |
| M7 | MED | Abuse | `market-proxy/*` | Public, unthrottled proxy to realtime engine (0 frontend refs). | Authenticate/throttle or remove. |
| M8 | MED | Secrets | `backtest/backtest.service.ts:41-43` | `DATABASE_URL` (with password) passed as CLI arg (visible in `ps`). | Pass via env to child; sanitize errors. |

### 3.3 Next.js Frontend
| ID | Sev | Category | Location | Issue | Proposed Fix |
|---|---|---|---|---|---|
| F1 | HIGH | Security | `lib/auth.ts:78-85,101-106` | Access token in non-HttpOnly, JS-readable cookie + mirrored to `window.__ACCESS_TOKEN__` → XSS token theft. | Move to HttpOnly cookie + cookie-based auth (read Next docs first; rework token usage). |
| F2 | HIGH | Security | `next.config.ts` (no `headers()`) | No CSP/HSTS/X-Frame-Options. | Add `headers()` with strict CSP + security headers. |
| F3 | MED | Security | `lib/backtest-api.ts:114-116` | JWT leaked in CSV download URL. | Scoped download token / signed URL. |
| F4 | MED | Honest labeling | `lib/websocket.ts:190-248` + `market-header.tsx:144` | Relay `status` (LIVE/DEGRADED/STALE) parsed then dropped; UI shows LIVE from socket only. | Forward feed status; render honest badge. |
| F5 | MED | Error handling | `components/backtest/backtest-panel.tsx:150,164-189` | Backtest `status:"FAILED"` never surfaced. | Show `result.error`. |
| F6 | MED | A11y | `app/(auth)/register/page.tsx:255-289` | Custom checkbox `<div onClick>` not keyboard accessible. | Real `<input type=checkbox>` + label. |
| F7 | MED | A11y | `tailwind.config.ts` | No `prefers-reduced-motion` handling. | Add reduced-motion utilities/global rule. |
| F8 | LOW | Dead code | `components/user-dashboard/command-center.tsx`, `components/trading/live-dashboard.tsx`, `orval.config.ts`, `_ua_tmp.tsx` | Unreferenced components/config. | Remove or wire. |

### 3.4 Python Research / Quant
| ID | Sev | Category | Location | Issue | Proposed Fix |
|---|---|---|---|---|---|
| R1 | CRITICAL | Integrity | `scripts/quant_validation.py:20-73` | Hardcoded Sharpe/Brier dict; `--dry-run` is the only path; writes `quant_evidence.json` accepted by `final_go_live_check.py` → fabricated go-live evidence. | Remove fabrication; require real data; `final_go_live_check` must reject `DRY_RUN`/missing provenance. |
| R2 | HIGH | Strategy logic | `strategy/ptb_strategy.py:34-61,245-267` | `atr_tp1 == atr_sl` for STANDARD_SCALPING/STANDARD_SWING/TREND_SWING → RR < `min_rr` → permanent NO_TRADE. | Mirror Go `getStrategyConfig` TP1 multipliers; set simplified ATR-RR floor (documented). |
| R3 | MED | Metric math | `analytics/metrics.py:136-149` | Sharpe on absolute PnL (not returns); Sortino downside-dev uses losing trades only → inflated. Diverges from `reference_math`. | Reuse `reference_math.sharpe/sortino` on a return series. |
| R4 | MED | Indicator | `engine/core.py:415-428` | `_update_atr` uses current close as prev close → TR collapses to high-low, ignores gaps. | Track previous candle close. |
| R5 | MED | Mislabel | `analytics/walk_forward.py` | Docstring claims param optimization; impl is rolling backtest; no OOS equity. | Implement or correct docstring. |
| R6 | LOW | Integrity | `engine/core.py:103` | `run_id = uuid4()[:8]` → collisions overwrite unrelated runs. | Full uuid4 or deterministic hash. |

### 3.5 Database / TimescaleDB / Valkey
| ID | Sev | Category | Location | Issue | Proposed Fix |
|---|---|---|---|---|---|
| D1 | CRITICAL→MED | Repro/schema-drift | `finance.ledger_entries` (live exists; not in migrations) | Table not in any migration → fresh deploy breaks payouts. | `075` `CREATE TABLE IF NOT EXISTS … UNIQUE(idempotency_key)`. |
| D2 | HIGH→MED | Repro/schema-drift | `market.data_metadata` (live exists; not in migrations) | `getAvailableData` reads it; not in migrations. | `076` `CREATE TABLE IF NOT EXISTS` (mirror live DDL). |
| D3 | MED | TimescaleDB | `005_trading_market_tables.sql` | `market.candles` has compression (30d) but **no retention** → unbounded growth. | Add retention policy (operator-approved; data-deleting). |
| D4 | MED | Migrations | `scripts/migrate.sh` + `MIGRATION_ORDER.md` | 6 duplicate sequence numbers; `migrate.sh` warns but does not `exit 1` (contradicts doc). | Enforce uniqueness / renumber. |
| D5 | LOW | Dead schema | `010`/`020` | `research.backtest_*` orphan tables; duplicate `create_hypertable`. | Drop/consolidate; remove dup. |
| D6 | LOW | Valkey | `realtime/internal/cache/*` | Candle cache keys omit `source` → mixed-source serving. | Include `source` in cache keys. |

### 3.6 MQL4/5 + Windows Agent
| ID | Sev | Category | Location | Issue | Proposed Fix |
|---|---|---|---|---|---|
| W1 | CRITICAL | Safety/Entitlement | `windows-agent/internal/pipe.go:76-77`, `mql/*/PredictATrade_*.mq*` | License defaults to `ACTIVE`/`ELITE` and never validated when `LicenseKey==""` → full trading with no license (fail-open). | Default `PENDING`/`""`; only set ACTIVE after verified server response. |
| W2 | CRITICAL | Entitlement | `mql/mt5/PredictATrade_MT5.mq5:1924-1933` | `allowed_strategies` sent as array; EA `ExtractJSONString` expects quoted string → never parsed → all strategies enabled even on FREE plan. | Parse array (or emit CSV); regression test. |
| W3 | CRITICAL | Security | `pipe.go` IPC + `agent.go:597` WS | IPC files unauthenticated/unsigned (local spoof writes `ACTIVE`/`KILL_SWITCH`); WS only carries spoofable `agentId`, no token/HMAC. | HMAC-sign IPC + WS handshake; reject bad MACs. |
| W4 | HIGH | Execution | `mql/mt4/PredictATrade_MT4.mq4:1769` vs `:1823` | BUY ignores `PAT_CalcLotSize` (min lot); SELL risk-sizes → asymmetric sizing. MT5 ignores auto-lot entirely. | Identical lot logic both sides/honor auto-lot. |
| W5 | HIGH | Execution | `mql/mt5/...:259,732` | Per-strategy slippage dead; fixed 3-pt deviation → frequent broker rejects. | Apply `PAT_GetMaxSlippage(strategy)`. |
| W6 | HIGH | Safety | `mql/mt5/...:1298-1309` | `CLOSE_POSITION` by ticket closes arbitrary position (no magic/symbol check). | Verify PAT magic + symbol. |
| W7 | MED | Reliability | `pipe.go:147-150` | Pipe goroutines launched without panic recover → one bad payload crashes agent. | Wrap in safe-runner. |
| W8 | MED | Safety | `agent.go:707-712` | `KILL_SWITCH` reconnects and keeps forwarding. | Halt forwarding + exit. |
| W9 | MED | Security | `updater.go:39,165-169` | Manifest not signature-verified (checksum only). | Verify RSA/ECDSA over manifest. |
| W10 | MED | Anti-piracy | `fingerprint.go:105-124` | Hardware IDs stubbed → device binding copyable. | Implement real reads; fail if empty. |

---

## 4. Missing Components / Verifications

- **Subscription state machine** (cancel/refund/pause) — referenced by audit scope but absent in control plane.
- **Real quant-validation path** — `quant_validation.py` has no computation; only simulation.
- **finance.ledger_entries reproducible DDL** — exists live, missing from migrations (D1).
- **market.data_metadata reproducible DDL** — exists live, missing from migrations (D2).
- **Signed updates / IPC/WS auth / replay protection** — not implemented (W3, W9).
- **Orphan/dead code**: `internal/{oco,replay,maintenance,rl,sentiment,hedging,recovery,adaptation}` dormant; frontend dead components.

---

## 5. Database Schema Review (verified against live DB)

- `finance.ledger_entries`: **exists live** with `id uuid PK`, `account_user_id uuid FK iam.users`, `entry_type`, `direction CHECK IN ('CREDIT','DEBIT')`, `amount numeric(18,8) > 0`, `currency`, `source_type`, `source_id uuid`, `idempotency_key varchar(255) UNIQUE`, `metadata jsonb`, `created_at`; indexes on `(account_user_id, created_at DESC)`, `(source_type, source_id)`, unique idempotency. **Money types are correct (numeric).** Not in migrations → D1.
- `market.data_metadata`: **exists live**, PK `timeframe`, `candle_count bigint`, `min_date/max_date date`, `source`, `updated_at`. Not in migrations → D2. **Currently empty** (must be populated on ingest for the backtest data picker to show ranges).
- `market.candles`: hypertable (1-day chunks), compression @30d, **no retention policy** (D3). Indexes present on backtest tables (`idx_backtest_runs_created/status/strategy/symbol`). `trading.backtest_trades` exists.
- **No FLOAT/DOUBLE used for money in any migration** (verified). Money is `NUMERIC(18,8)`/`DECIMAL`.
- High-volume `trading.*` (signals/trades/positions) are regular tables, not hypertables — acceptable but note for scale.

---

## 6. Phase 4 Remediation Status

**FIXED IN THIS AUDIT (verified before change):**
- R1 — `quant_validation.py` fabrication removed; `final_go_live_check.py` rejects `DRY_RUN`/missing provenance.
- D1/D2 — `075`/`076` migrations (`CREATE TABLE IF NOT EXISTS`) make ledger + metadata reproducible.
- G2 — agent-hub `done` channel guarded with `sync.Once`.
- G3 — backtest MTF look-ahead fixed (require candle fully closed).
- G4 — backtest `data_source` label corrected to `market.candles`.
- R2 — Python adapter mirrors Go `getStrategyConfig` TP1 multipliers; ATR-RR floor documented so all 5 strategies are backtestable.
- R6 — `run_id` uses full uuid4 (no collision).
- C1 — NOWPayments IPN now verifies raw-body HMAC-SHA512.
- H1 — JWT guard + backtest download use injected `JwtService` (single secret source).
- W1 — Windows agent license defaults fail-closed (`PENDING`/`""`).
- F2 — Next.js security headers (CSP/HSTS/etc.) added via `headers()`.

**PROPOSED (not applied this pass; require care / build verification / operator sign-off):**
- G1 (engine SL/TP override reconciliation), G5/G6 (backfill pagination + source filter), C3 (payout reservation status), H2/H3/H4 (trust-proxy, backtest IDOR, multi-device), M1–M8 (commissions/subscriptions/billing hardening), F1/F3/F4/F5/F6/F7 (token HttpOnly refactor, download token, feed-status label, a11y), R3/R4/R5 (metric math, ATR prev-close, walk-forward), D3/D4/D5/D6 (retention policy [data-deleting—operator approval], migration uniqueness, dead schema, cache source key), W2/W3/W4/W5/W6/W7/W8/W9/W10 (EA parsing, IPC/WS signing, lot/slippage symmetry, arbitrary-close guard, panic-recover, kill-switch halt, signed updates, fingerprint). MQL changes require compilation verification (not possible in this environment) and are documented with exact edits.

**Tests added/updated:** Go `agent_ws` close-guard unit test; research strategy RR regression test; control IPN verification test. All existing suites remain green.

---

## 7. Production Readiness Verdict

**NOT READY FOR GO-LIVE.** Critical blockers (fabricated evidence neutralized; IPN, payout integrity, license fail-open, JWT/frontend token) are partially remediated. Remaining Critical/High items (payout double-spend C3, EA entitlement dead-code W2, IPC/WS auth W3, subscription state machine M2, token HttpOnly refactor F1) must be closed and verified by build + integration tests before any production launch. No fabricated performance claims are asserted as evidence.
