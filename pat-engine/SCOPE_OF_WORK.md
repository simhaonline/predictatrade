# SCOPE_OF_WORK — pat-engine (clean-room signal engine)

> Tracking document for the standalone signal engine built under the Core Reset plan.
> Current project (Go realtime / Python / NestJS / Next.js / MQL agents) is **untouched**;
> this engine is reference-only and reuses no existing service or database.

## 1. Objective & approach

Rebuild the signal brain clean-room (Go, **zero external dependencies**) so it can be:
proven on real data, risk-gated, and honestly packaged per broker/license entitlement.
Start with ULTRA_SCALPING-grade fidelity, then add the other products behind the same
single-source config. Backend engine + MQL first; frontend deferred.

## 2. Architecture (within pat-engine)

```
data/ticks ─▶ cmd/agent ─POST /candles▶ cmd/gateway ─▶ strategy pipeline
                                                    │  (broker policy + license + gates)
                                                    ▼
                                            PAT_signals.txt ─▶ MQL EA (FILE_COMMON)
```

- **Single source of truth:** `internal/config` defines every SL/TP/RR/score threshold.
- **Decision pipeline:** `signal.Decide(state, strategy, cfg, policy)` →
  strategy evaluates → broker policy gate → hard risk gates → `Signal{Executable}`.
- **Entitlement gate (`LICENSE`):** applied at strategy *selection* (gateway + backtest),
  never inside the EA. A license may only *narrow* what the broker policy already allows.

## 3. Implemented modules

| Module | Path | Purpose |
|---|---|---|
| Types / contracts | `internal/types` | `MarketState`, `Signal`, `Indicators`, `StrategyID`, `Direction` |
| Config | `internal/config` | Single-source SL/TP/RR + `AllDefaults()` for 4 products |
| Broker policy | `internal/broker` | `BrokerPolicy`, scalping toggle, stop/freeze vetoes |
| License | `internal/license` | Signed token, expiry, device binding, `allowed_strategies` |
| Strategies | `internal/strategy` | `ULTRA_SCALPING`, `STANDARD_SCALPING`, `STANDARD_SWING`, `TREND_SWING` + shared scoring/geometry |
| Gates | `internal/gates` | R:R floor, broker stop/freeze |
| Signal pipeline | `internal/signal` | `Decide()` orchestration |
| Indicators | `internal/indicators` | EMA/SMA/ATR/RSI/MACD/ADX/Stoch/Boll/VWAP |
| Backtest | `internal/backtest` | Bars→snapshots (same math) + exit simulator + real CSV loader |
| Live gateway | `internal/provider` | HTTP ingest → pipeline → `PAT_signals.txt` |
| Binaries | `cmd/{engine,backtest,gateway,agent,sim}` | demo / research / live / feeders |
| MQL | `mql/` | Reference EAs that already parse `PAT_signals.txt` |

## 4. License Management (first-class)

This is the entitlement backbone. It is enforced **server-side in the engine**, matching
the architecture rule that strategy enable/disable is a control-plane (license)
concern and **never an EA input**.

### 4.1 Token format
`base64(json) + "." + hex(HMAC-SHA256(json, secret))`
- Signed by the **control plane** (NestJS) using `PAT_LICENSE_SECRET`.
- Verified by the engine with the same secret. Tamper → `signature mismatch`.
- Default dev secret lets the engine run standalone; production **must** inject the real
  secret (never commit it).

### 4.2 Fields (`internal/license.License`)
- `key` — license identifier.
- `plan` — plan tier (DEV / STANDARD / PRO / ELITE…), used for packaging/analytics.
- `allowed_strategies` — explicit entitlement list. `"*"` = all. **This is the gate that
  decides which strategies a client may receive.**
- `device_id` — optional device binding; mismatch → rejected.
- `expires_at` — unix seconds; `0` = never. Expired → invalid.
- `broker_scalping` — optional override of scalping permission (defers to broker policy
  when nil).

### 4.3 Enforcement points
- **Gateway** (`provider.Gateway`): `LoadLicense(token, secret)` installs the license;
  `bestExecutable` skips any strategy not in `allowed_strategies`.
- **Backtest** (`backtest.RunAll`): same filter, marked `EXCLUDED (LICENSE_STRATEGY_NOT_ALLOWED)`.
- **EA**: receives only already-licensed signals; it never sees or decides entitlement.

### 4.4 Acceptance / tests
- `internal/license/license_test.go`: sign/parse round-trip, tamper rejection, expiry,
  device binding, wildcard.
- `internal/provider/gateway_test.go`: a TREND_SWING-only license suppresses scalpers
  even when the broker would allow them.

### 4.5 Future (control-plane integration)
- NestJS issues signed licenses bound to user/device/plan; engine verifies only.
- Per-signal **calibrated probability** (never raw score) attached by the engine from the
  named prediction target + active exit profile.

## 5. Broker scalping policy

`BrokerPolicy.AllowsScalping=false` excludes both scalping strategies
(`BROKER_SCALPING_NOT_ALLOWED`). Verified by backtest harness and decision tests. This is
how we safely offer a **no-scalping-broker-eligible package** (swing/trend only).

## 5b. Data & persistence (PostgreSQL + TimescaleDB + Valkey)

Required for auditability: **every bar and every signal decision is tracked.**

- **TimescaleDB**, database `pat_engine` (dedicated service; does not touch the main
  project's `pat-postgres`). Schema `infra/db/init/01_schema.sql`:
  - `bars(ts, symbol, o/h/l/c, spread)` — hypertable, every ingested bar.
  - `signals(id PK, ts, symbol, strategy_id, direction, entry, sl, tp1..tp3, raw_score,
    grade, signal_class, status, reasons jsonb)` — hypertable, **every** decision
    (executable or blocked).
- **Valkey**: `pat:signal:latest:<strategy>` cache (TTL 24h) + `pat:signals` pub/sub
  channel for low-latency live push (the EA file remains the primary delivery).
- **`internal/store`** wraps `pgxpool` + `go-redis` and is **degradable**: if either
  backend is unreachable the engine keeps running on an in-memory ring buffer (logs the
  degradation). A missing datastore never blocks signal generation.
- **Docker**: `docker-compose.yml` runs `pat-engine-db` (TimescaleDB, port 5433),
  `pat-engine-cache` (Valkey, port 6380), and `pat-engine` (gateway). Env sample in
  `infra/env/ENV_SAMPLE` (real `.env` gitignored).
- **Verified live:** streaming bars persisted 6000 `bars` + 967 `signals` rows; Valkey
  cached the latest signal and published on `pat:signals`.

> Note: earlier "zero external dependency" guidance is relaxed for the datastore only;
> `pgx` and `go-redis` are the sole added modules, both standard, well-maintained, and
> required for the persistence/audit requirement.

## 6. Strategies (config-backed, versioned)

| ID | Class | SL×ATR | TP1×ATR | Min R:R | Min Conf | Min ADX |
|---|---|---|---|---|---|---|
| ULTRA_SCALPING | SCALP | 1.0 | 2.0 | 2.0 | 65 | 25 |
| STANDARD_SCALPING | SCALP | 1.0 | 1.5 | 1.5 | 65 | 20 |
| STANDARD_SWING | SWING | 1.5 | 2.5 | 1.5 | 55 | 20 |
| TREND_SWING | TREND | 2.0 | 3.0 | 1.5 | 50 | 20 |

## 7. Backtest harness

Runs the **exact live strategy code** on (`internal/backtest`):
- Synthetic generator (demo) **and** `FromCSV` real-bar loader (`time,open,high,low,close,spread`).
- Exit simulator walks forward bars, returns on first SL/TP1 hit; reports trades, win%,
  PF, gross win/loss per strategy, honoring broker policy + license.
- Synthetic demo PF (trend-friendly data): ULTRA ~3.1, SWING ~7.2, TREND ~7.7.
  **These are synthetic; real KAGGLE/MT5 bars must be plugged in to lock v1 stats.**

## 8. Live path (proven end-to-end)

`cmd/agent` streams bars → `POST /candles` → `cmd/gateway` → `signal.Decide` →
`SIGNAL|<json>` written to `PAT_signals.txt` → `cmd/sim` consumes it like the MQL EA.
Verified manually; the existing `mql/` EAs parse the identical format.

## 9. Testing status

- `go test ./...` — **green**: strategy executability + R:R floor, broker scalping
  exclusion, high-spread block, license sign/parse/tamper/expiry/device, and license
  suppression of non-entitled strategies.
- End-to-end gateway→file→EA-sim verified.

## 10. Deferred (per plan: backend + MQL first)

- Frontend / Command Center.
- Control-plane licensing **issuance** service (NestJS) — engine already verifies.
- Rebuilt Windows Agent binary (reuse `windows-agent`, point at `POST /candles`).
- Calibrated probability model + walk-forward/OOS on real 2025 data.

## 11. Traceability

| Requirement | Implementation | Tests | Status |
|---|---|---|---|
| 4 distinct versioned strategies | `internal/strategy/*` | — | DONE |
| Single-source SL/TP/RR config | `internal/config` | — | DONE |
| Broker scalping restriction | `internal/broker` + pipeline | broker_test, backtest | DONE |
| License / entitlement gating | `internal/license` + gateway/backtest | license_test, gateway_test | DONE |
| Hard risk gates | `internal/gates` | signal_test | DONE |
| Cost-aware backtest on live code | `internal/backtest` | — | DONE (real-data loader wired) |
| Live signal→EA handoff | `internal/provider` + `mql/` | e2e | DONE |
| Persistence: TimescaleDB + Valkey | `internal/store` + `infra/db/init` + compose | store degraded test + live smoke | DONE |
| Broker execution correctness (digits, contract, commission, swap, spread, leverage) | `internal/broker/profile.go` + `LoadExecutionFromEnv` | profile_test | DONE |
| Broker-server-time sessions (timezone, not local/UTC) | `broker.BrokerPolicy.Session` + `TimezoneOffset` | profile_test (UTC 09:30→OVERLAP) | DONE |
| Cost-aware NET R:R gate (spread+commission+swap, price units) | `internal/gates` | gates_test (negative-after-cost) | DONE |
| Capital-loss control (position size, daily loss, max positions/leverage) | `internal/risk` | risk_test | DONE |
| Full backend REST API (`/api/v1`: strategies, broker, risk, session, signals, bars, devices, licensing) | `cmd/gateway/main.go` | live smoke (curl) | DONE |
| Agent hardware fingerprint + telemetry device binding | `internal/agentlib` + `cmd/agent` + `/devices/*` | — | DONE (collector + endpoints) |
| Web/domain routing (nginx) | `nginx/pat-engine.conf` + compose `pat-engine-nginx` | — | DONE |
| Real-data backtest replay (XAUUSD 15m, net-R:R gate live) | `internal/backtest` + `cmd/backtest` + MetaTrader loader | gateway/backtest run | DONE (research snapshot; walk-forward/OOS deferred) |
| Control-plane license issuer + device-bound validation | `cmd/license-issuer` + `/licensing/validate` + `store.GetDevice` | E2E curl (invalid→active) | DONE |
| Frontend (Next.js) consuming `/api/v1` | `pat-engine/frontend` + nginx `/` route + compose `pat-engine-frontend` | `next build` + render smoke | DONE |
| Honest packaging (no false claims) | broker/license exclusion + real-data backtest §12 | backtest run on real 2024Q4 + 2025 m1 | PARTIAL — real-data baseline locked: only ULTRA_SCALPING OOS PF>1 (~1.08); others net-losing. No claims published; calibration (Next action #2) required before any performance claim. |
| Calibrated probability (named target) | `internal/calibrate` + `Signal.CalibratedProbability` + gateway attach | unit tests pass; gateway emits; backtest `CALIBRATE=1` fits+validates | DONE — probability is empirical, regime×score-decile, Laplace-smoothed, validated OOS reliability; never the raw score. Uncalibrated when no model loaded. |
| Calibration research harness (OOS) | `internal/backtest.EvalStrategy` + `cmd/calibrate` | ran on real 2025 m1 (ULTRA_SCALPING) | DONE (tooling) — harness is strict train/test OOS. Result: NO candidate showed genuine OOS edge (test PF 0.28–0.63). Nothing published; honest negative finding. |

## 12. Real-data backtest — honest v1 stats (LOCKED)

Ran `cmd/backtest` on the two real XAUUSD 1h datasets in `data/` (NOT synthetic):
- `xauusd_2024q4.csv` — 86,400 bars, 2024-09-30 → 2025-01-28 (UTC)
- `xauusd_m1.csv` — 84,960 bars, 2025-02-28 → 2025-06-28 (UTC)

Both served through the **same** `BuildSnapshots` → `signal.Decide` → `Simulate`
pipeline used live. Staleness flag is expected (data predates the run); it is
shown so the result is never mistaken for fresh. Walk-forward = contiguous,
non-overlapping OOS folds with **no parameter re-fitting** (params are playbook-
derived, not data-mined).

### 12.1 Full-period results (real data)

| Strategy | 2024Q4 PF | 2024Q4 win% | 2025 m1 PF | 2025 m1 win% |
|---|---|---|---|---|
| ULTRA_SCALPING | 1.07 (26) | 53.8% | 1.27 (34) | 64.7% |
| STANDARD_SCALPING | 0.82 (26) | 57.7% | 0.62 (88) | 43.2% |
| STANDARD_SWING | 0.34 (161) | 36.6% | 0.55 (323) | 43.7% |
| TREND_SWING | 0.25 (23) | 17.4% | 1.53 (127) | 68.5% |

### 12.2 Walk-forward OOS (aggregated across folds)

| Strategy | 2024Q4 OOS PF | 2025 m1 OOS PF |
|---|---|---|
| ULTRA_SCALPING | 1.08 | 1.08 |
| STANDARD_SCALPING | 0.76 | 0.76 |
| STANDARD_SWING | 0.37 | 0.55 |
| TREND_SWING | 0.25 | 1.33 |

### 12.3 Honest interpretation

- **No strategy is reliably profitable across both regimes.** Only `ULTRA_SCALPING`
  cleared PF>1 in both periods (marginal ~1.08). `TREND_SWING` shows PF 1.33 in
  2025 but 0.25 in 2024Q4 — i.e. period-dependent, not robust.
- `STANDARD_SCALPING` and `STANDARD_SWING` are **net-losing** on real data as
  configured. `NO-TRADE` / modest-edge is a valid first-class result; we do NOT
  force profitability or fabricate win rates.
- This is exactly why the SOW defers any publishable performance claim until
  walk-forward/OOS calibration on real data. The numbers above ARE the locked
  baseline — they are honest and must not be smoothed.
- Sample size per fold is small (tens of trades), so even PF>1 readings carry
  wide confidence intervals. Larger/cleaner 2025 history is required before any
  external claim.

## 13. Next actions

1. **(DONE — baseline locked)** Real-data backtest: honest v1 stats recorded above.
   No claims are made; the engine currently shows weak/negative edge on XAUUSD 1h.
2. **Calibration research (required before any claim):** run strict walk-forward
   with parameter search on a *training* window and validate PF on a held-out
   *test* window (never the same bars). Re-evaluate only after OOS PF>1 with
   adequate trade count and reasonable drawdown. Models/optimizers must not
   self-promote (SOW quant-integrity rule).
3. Issue signed licenses from control plane; bind device/plan/strategies.
4. Wire live Windows Agent → `POST /candles`; deploy EA reading `PAT_signals.txt`;
   verify end-to-end with the bridged `PAT_signals.txt` handoff on a remote terminal.
5. Add calibrated probability (named prediction target + active exit profile);
   only then package & publish performance — and only with the OOS-evidenced edge.

## 14. Calibrated probability (NEW — SOW §4.5)

`internal/calibrate` attaches a **calibrated probability** to every executable signal. It
is explicitly NOT the raw strategy score (raw score is not probability).

- **Named target:** `TP1_BEFORE_SL` — P(price reaches the 1R partial target before the
  SL), under the same exit profile the backtest `Simulate` uses. Calibration is thus
  consistent between research and live.
- **Model:** `EmpiricalModel` buckets realized outcomes by `(strategy, regime) × score-decile`
  and reports a Laplace-smoothed win fraction. Simple, monotonic, no market-prediction
  claim, no over-fitting surface.
- **Fitting:** `backtest.FitCalibration(train)` over a TRAIN window; `ValidateCalibration`
  measures reliability on a held-out TEST window (predicted vs realized per bucket).
- **Live wiring:** the gateway loads a fitted model from `CALIBRATION_MODEL_PATH`;
  `calibrate.Attach` sets `Signal.CalibratedProbability / ProbabilityTarget /
  ProbabilityModel`. With no model loaded, signals are emitted `UNCALIBRATED` (prob 0) —
  never a guessed number.
- **CLI:** `cmd/backtest` with `CALIBRATE=1` fits on TRAIN (default 60%), validates on
  TEST, and writes the model JSON (`CALIBRATION_OUT`) for the gateway.

## 15. Calibration research harness (NEW — SOW §12.2)

`cmd/calibrate` searches a parameter grid for ONE strategy on a TRAIN window and reports
the out-of-sample (TEST) profit factor for every candidate, strictly separated.

- Reuses `backtest.EvalStrategy` (the exact live pipeline) so no research drift.
- Honesty guards: a config is only written out as "having edge" when **TEST PF > 1** AND
  **TEST PF ≥ 0.8 × TRAIN PF** (no silent over-fit) AND adequate sample. Otherwise it
  publishes nothing and says so.
- **Honest result on real 2025 m1 (ULTRA_SCALPING, 72-grid):** every candidate's TEST PF
  was 0.28–0.63 (all < 1) while some TRAIN PF reached ~1.8 — i.e. the apparent full-sample
  edge does not survive a strict holdout. This is exactly why OOS validation precedes any
  performance claim. The default v1 configs therefore remain UNPUBLISHED pending a
  genuinely OOS-validated parameter set (broader grid, more history, or a different edge).
