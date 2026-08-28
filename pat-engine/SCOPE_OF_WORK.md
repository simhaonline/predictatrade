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
data/ticks ─▶ cmd/agent ─POST /bar▶ cmd/gateway ─▶ strategy pipeline
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

`cmd/agent` streams bars → `POST /bar` → `cmd/gateway` → `signal.Decide` →
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
- Rebuilt Windows Agent binary (reuse `windows-agent`, point at `POST /bar`).
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
| Honest packaging (no false claims) | broker/license exclusion + real-data TODO | — | PARTIAL (needs real data) |

## 12. Next actions

1. Plug real 2025 KAGGLE/MT5 bars into `cmd/backtest` → lock honest v1 stats.
2. Issue signed licenses from control plane; bind device/plan/strategies.
3. Wire live Windows Agent → `POST /bar`; deploy EA reading `PAT_signals.txt`.
4. Add calibrated probability; only then package & publish performance.
