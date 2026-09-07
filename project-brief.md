# Predict-A-Trade — Project Brief (project-brief.md)

The complete project reference: what this platform is, every plane and
service, the full tick→signal→delivery lifecycle, the SaaS/commercial layer,
data model, operations, and safety rules — followed by a plain-English Part II
that explains the same system for anyone new.

Companion file: **indicator.md** (indicator maths, evidence scoring, gates —
the "brain" in depth). This file covers the whole organism around it.

Sources: `README.md`, `MANIFEST.md`, `AGENTS.md`, `docs/` (ARCHITECTURE,
DATABASE_ARCHITECTURE, API_REFERENCE, RISK_GATES, CAPITAL_TIERS,
STRATEGY_PLAYBOOKS), `docker-compose.yml`, `docs.md`. Version: **v1.29.x**
(live structured-log pin: realtime v1.24.2). Repo: `/srv/predictatrade/xauusd`
→ `github.com/simhaonline/predictatrade` (main).

---

# Part I — Precise Reference

## 1. What Predict-A-Trade is

A multi-plane XAUUSD (gold) trading **signal platform**:

- **One brain** — a Go realtime engine that ingests broker ticks, builds
  candles, computes 42 indicators, runs 7 strategy engines, scores evidence,
  applies hard risk gates, and emits signals.
- **Thin edge** — MetaTrader 4/5 Expert Advisors on traders' Windows machines.
  Master EAs stream market data up; Client EAs poll for executable signals
  and place orders **on the trader's own broker account**. Money never
  transits the platform.
- **SaaS wrapper** — NestJS control plane for IAM, subscriptions (USDT-only
  via NOWPayments), licensing/devices, referrals/commissions/payouts.
- **Presentation** — Next.js dashboards (user portal + admin console) that
  render server truth; they never recompute risk or entitlement.

**Status (honest):** GO for paper/sandbox/advisory signal operation. Live
trading arming was authorized by the operator (2026-08-30,
`LIVE_TRADING_AUTHORIZED=true`), but capital-protection gates remain
fail-closed: signals stay ADVISORY unless a verified broker equity/order feed
exists. No profitability/accuracy claims are published without evidence.

## 2. The five planes (hard boundaries)

| Plane | Tech | Authority | Never does |
|---|---|---|---|
| Real-Time Trading | Go 1.25 (`realtime/`) | market-data, features, strategies, signals, risk gates, execution authorization, delivery, reconciliation | no sync billing/referral/commission/payout in the tick path |
| Research/Intelligence | Python (`research/`) | datasets, backtesting, walk-forward, calibration, ML training | never a mandatory per-tick dependency |
| SaaS/Control | NestJS 12 (`control/`) | IAM/MFA/RBAC, subscriptions, billing, licensing, devices, referrals, commissions, payouts, audit | never computes trading signals |
| Presentation | Next.js 16 + React 19 (`frontend/`) | renders server-authoritative truth | never re-indicators / re-risk / re-entitlement / re-finance |
| MQL Edge | MQL4/MQL5 (`mql/`) | lightweight execution adapters | no primary intelligence, no server credentials in EAs |

## 3. Service inventory (Docker-first — systemd is disabled)

All via `docker compose --env-file infra/env/.env` (the env-file is MANDATORY
— plain `docker compose up` starts with blank secrets).

| Service | Container | Port | Role |
|---|---|---|---|
| Realtime engine | pat-realtime | 13081 | Go HTTP/WS signal engine |
| Control plane (HA pair) | pat-control / pat-control-b | 13080 | NestJS IAM/billing/licensing; nginx failover |
| Frontend | pat-frontend | 13082 | Next.js user + admin dashboards |
| Status page | pat-status | 13083 | Public status & compliance page |
| Backtest service | pat-backtest | 8088 (loopback) | Python walk-forward/OOS API |
| Live terminal | pat-live-terminal | 13090 | Bloomberg-style terminal |
| Mail relay | pat-mail-relay | 465/587 | Send-only SMTP (spool + retry) |
| PostgreSQL 17 + TimescaleDB | pat-postgres | 5432 | 16 schemas, 210+ tables, hypertables |
| Valkey 8 | pat-valkey | 6379 | Hot/cache state (never sole financial truth) |
| Nginx | pat-nginx (xauusd-nginx-1) | 80/443 | TLS, reverse proxy, WS routing |
| Prometheus | pat-prometheus | 9090 | Metrics |
| Grafana | pat-grafana | 3001 | Dashboards (loopback) |
| ntfy | pat-ntfy | 8091 | Notifications |
| NATS | pat-nats | 4222 | Optional ingest-decoupling bus |
| Backup sync | pat-backup-sync | — | Hetzner S3 WAL + pg_dump off-host |
| Ollama | host | 11434 | LLM market-context sentiment (NOT_AI_VERIFIED) |

Public domains: `platform.predictatrade.com` (app), `api.predictatrade.com`
(control), `live.predictatrade.com` (live command center),
`downloads.predictatrade.com` (EA binaries), `docs.predictatrade.com`.

## 4. End-to-end data & signal lifecycle

```
MT4/MT5 MASTER EA (any terminal, data role)
  │  POST /ingest/agent — Bearer device JWT, TYPE|{json} lines
  │  MARKET_SNAPSHOT (OHLCV bars per TF) + MASTER_TICK
  ▼
Nginx :443 → Go REALTIME ENGINE :13081
  │  server receive time = truth (PC clocks drift)
  ├─ market.ticks / market.candles (Timescale hypertables)
  ├─ per-timeframe Feature Registry (42 indicators)
  ├─ 7 Strategy engines (evidence → score → grade → geometry)
  ├─ Signal engine + hard gates (deterministic, fail-closed)
  ├─ capital-tier viability (MICRO < $500 / STANDARD $500–5k / PRO ≥ $5k)
  ▼
TimescaleDB + Valkey + WebSocket (/ws/v1) ──► Next.js dashboards
  │
  └─ enqueueSignalForDevices → licensing.edge_signal_queue
       (SQL filter: license ACTIVE + plan whitelists strategy +
        device role=exec + capital tier ∈ EligibleTiers)
       │
MT4/MT5 CLIENT EA ──HMAC edge-poll every ~3s (always-ACK)──► control plane
  │  receives executable signals + server commands, ACKs each
  ▼
Places order on the trader's OWN broker account (EA-side safety gates too)
  │  EXECUTION_ACK back → reconciliation (fill verification, SL check ±0.5pt)
```

Transport era: **Option B (v1.19.0)** — EAs talk directly to the cloud over
HTTPS. The old Windows Agent (installers, port 13091, `/ws/v1/agent`) is
removed. One ingest port, one poll API, nothing open inbound on the
trader's machine.

Honest-feed rule: when the Master EA stops streaming, the feed reports
`NO_DATA` — never a fake "live".

## 5. The Go realtime engine (internal map)

`realtime/cmd/realtime-engine/main.go` (6.4k lines) orchestrates;
`internal/` holds 34 modules. Key ones:

| Module | Role |
|---|---|
| `features/` | 42-feature indicator engine (per-TF registries), structure, liquidity, FVG, VWAP, SAR, Ichimoku, StochRSI, Fib, pivots, ORB, pullback, regime, MTF, session |
| `strategy/` | 7 strategies + geometry + regime thresholds + transition/range evidence |
| `signal/` | Decision engine, cooldowns, duplicate-bar idempotency |
| `gates/` | Hard gates (registry, short-circuit, fail-closed, per-(strategy,TF) state) |
| `marketdata/` | AgentProvider ingest, persister (candles/regime/indicators → DB) |
| `gateway/` | HTTP + WebSocket (pprof-enabled) |
| `calibration/` | Score → probability (VALIDATED-gated; raw score is never published as probability) |
| `capitaltier/` | MICRO/STANDARD/PRO banding + per-tier signal viability |
| `ml/` + `pkg/mlengine/` | ONNX inference, `models/` watcher hot-reload (5 bootstrap models — honestly labelled `bootstrap-v1.0.0`, NOT production-trained) |
| `rl/`, `sentiment/`, `adaptation/`, `recovery/`, `hedging/` | Advanced layers — additional fail-closed filters, never gate-weakening |
| `reconciliation/` | EXECUTION_ACK → fill verification |
| `ptb/`, `igs/`, `astro/`, `crossmarket/`, `devilliquidity/` | Shadow/synthesis intelligence layers (calculate + persist; contribute zero to live scores unless activated) |
| `pkg/bus/` | Ingest decoupling seam (DirectBus in-process; NatsBus when NATS_URL set) |
| `pkg/math/` | Canonical Wilder maths (parity with Python `reference_math.py`) |

Performance note: indicators compute in float64 with parallel window mirrors
(~0.3 ms/bar whole set) after a pprof-backed optimisation; decimal is used
where money math happens.

## 6. Strategies (7)

| Engine | Internal ID | Decision TFs | Min score | Expiry | Mode |
|---|---|---|:--:|:--:|:--:|
| Standard Scalping | STANDARD_SCALPING | M1/M5 | 65 | 10m | LIVE |
| Ultra Scalping | ULTRA_SCALPING | M1 | 60 | 5m | LIVE |
| Standard Swing | STANDARD_SWING | M15/H1 | 68 | 30m | LIVE |
| Trend Swing | TREND_SWING | H1/H4 | 70 | 60m | LIVE |
| EQFE | MARNIE_FIB | H1 | 70 | 60m | SHADOW→LIVE |
| ATEN | ATEN (astro-confluence) | H1/H4 | 70 | 60m | LIVE |
| IMLR | (internal) | M5/M15 | 70 | 180m | ADVISORY-only |

- Display names: internal IDs `MARNIE_FIB`/`ARCANIST` never user-facing —
  rendered as **EQFE**/**IMLR** everywhere (DB/signals keep internal IDs).
- Plan entitlement (server-enforced): FREE → STANDARD_SCALPING only (max
  5 signals/day); STANDARD → + STANDARD_SWING; PRO → all 4 core; ELITE → all 7.
- IMLR delivers ADVISORY-only until validation/backtesting completes.

## 7. Risk gates & capital tiers

16 ordered gates (per-(strategy, timeframe) isolated state):
ExecutionPermission → BrokerSymbolValidation (P0) → SeedCapitalProtection
(5% daily cap) → DailyLossLimit → MaxSpread → NewsRisk → Slippage →
MaxPositions → MaxExposure → Cooldown → StopHuntFilter → MarginCheck →
OvertradeProtection → MaxDailyTrades → RegimeFilter → ProfitTarget.

Delivery is a second wall (EA-direct era): SQL entitlement filter at enqueue
+ re-check at edge-poll. One ineligible device can never suppress another's
delivery. Per-client risk isolation: executable signals forward only to
devices whose own account has free margin (fail-open on stale state; the
v1.23.1 fix made tier eligibility unconditional so stale account snapshots
cannot widen delivery).

Capital tiers: MICRO <$500 (floor $100), STANDARD $500–4,999.99, PRO ≥$5,000.
Effective per-trade risk cap = min(plan cap, tier cap 2%). Admin visibility:
`/admin/signal-engine` page (24h pipeline stats, devices by tier, per-signal
EligibleTiers).

Server-side SL enforcement: EXECUTION_ACK verifies SL matches server-sent
value (±0.5 pt); positions with missing SL get CLOSE_POSITION;
EMERGENCY_STOP closes all + halts; KILL_SWITCH closes all + ExpertRemove();
3 SL violations → device suspension via disconnection.

## 8. Control plane (NestJS) — 25 modules

`auth` (JWT + TOTP MFA, trusted-device cookie, refresh rotation), `users`,
`plans`/`subscriptions`/`billing` (USDT-only via NOWPayments; HMAC IPN +
amount verification + UNDERPAID handling; Stripe disabled), `licensing`
(licenses, devices, MT accounts, entitlement leases), `device-auth`
(activation + HMAC `edge-poll`/`edge-ack`/`edge-heartbeat` signal delivery),
`backtest`, `commissions`, `payouts`, `referrals` (exact-decimal ledger,
compensating records — never rewritten history), `operations` (halt-trading,
pause-signals, per-strategy kill switch, model activation), `admin`,
`admin-extras`, `audit`, `health`, `market-proxy`, `monitoring`,
`feature-flags`, `guest-preview` (anonymous 5-min preview funnel),
`compliance` (GDPR erasure), `brokers`, `plans`, `reports`.

Money math: NUMERIC(18,8)/NUMERIC(10,4) in Postgres, decimal.js in the
control plane — never floats.

## 9. Database

PostgreSQL 17 + TimescaleDB HA + pgvector. 100 migration files
(unique prefixes, numbered to 138, all applied; forward-only, never rewrite
history; `scripts/check_migrations.sh` reconciles `audit.migration_history`).
16 application schemas, 210+ tables, 232 CREATE TABLE statements.

Key schemas: `iam` (users/roles/sessions), `licensing` (licenses/devices/
mt_accounts), `billing`, `finance` (commission_ledger/payouts/ledger),
`trading` (signals, trade_results, execution_commands, exit_profiles,
blocked_signals, backtest_*), `market` (candles hypertable, ticks, cot_*,
economic_events, data_provenance_log), `calibration`, `audit`, `compliance`,
`live_preview`, `support`. Hypertables: `market.candles` (1-day chunks,
per-TF retention), `audit.client_event_log`, `market.cot_raw_reports`.

Master MT4/MT5 share ONE `market.ticks`/`market.candles` with a `source`
column (MT4_MASTER / MT5_MASTER); aggregator writes AGGREGATOR rows per TF.

## 10. Frontend (Next.js 16 + React 19)

User dashboard (19 routes): signals (paginated, TP1/TP2/TP3 columns, quality
grades A+/A/B/REJECTED, EV_R metrics, capital-protection sizing in expandable
rows), live chart/command center, strategies, backtest, billing, payouts,
referrals, mt4-mt5-client (license + downloads), security, settings,
signal-accuracy, trading-reports, activity-log, notifications, support.

Admin console (25+ routes): signal engine, indicators/indicator-monitor,
market-data, macro-news/intelligence, astro/aten, backtesting, devil-liquidity,
agent-mesh, device-auth, licenses, activations, mt-accounts, brokers,
broker-qualification, billing/payments/payments, commission-control-center/
operations, payout-operations, finance-referral-reports, users/logs/health,
operations, ai-providers, feature-flags, backup-dr, market-data.

Rule: the UI renders server truth; it never recomputes indicators, risk,
entitlement or money. Timestamps render on the broker clock
(`formatBrokerTimestamp`), not the browser's timezone.

## 11. MQL edge (pure MQL, self-contained)

4 EAs, ~11.8k lines: `PredictATrade_MT4.mq4` (3.9k) / `PredictATrade_MT5.mq5`
(4.0k) — Client EAs; `PredictATrade_MasterNode_MT4/5` (1.8k/2.1k) — Master
data EAs. Requirements (user mandate): PURE MQL — no external scripts, no
.mqh includes, no WebRequest beyond the cloud transport, no file reads; all
trading logic on broker time `TimeCurrent()`; DST-adaptive
`PAT_UTCToBrokerWall` / `PAT_LocalToBroker` bridges (zero hardcoded offsets);
token self-heal on refresh-401; EA-side capital guards (floating-DD breaker /
soft halt / recover → `licensing.device_risk_events` via migration 138).
Compiled in MetaEditor on Windows; error.log loop for fixes.

## 12. Research plane (Python)

`research/src/patresearch/`: indicators, backtester + backtesting, dataset,
calibration, ml_training, rl_training, quantitative_strategy_engine,
liquidity, ai_research, reference_math (canonical parity source for Go).
154 tests (152 pass, 2 skip) via `uv run pytest`. Not on the live tick path.

## 13. Operations

- Canonical commands: `make test / lint / build / format`; per-plane builds
  in MANIFEST. Go builds/tests run in `golang:1.25` container (host has no Go).
- Deploy: `docker compose --env-file infra/env/.env build <svc> && up -d <svc>`.
- Migrations: `./scripts/migrate.sh up` (never auto-applied).
- Observability: OpenTelemetry + Prometheus + Grafana + structured JSON logs;
  gate p50/p95/p99, WS reconnect stats, ingest/signal latency, backup checks.
- Ops scripts: `full_audit.sh`, `security-scan.sh` (gitleaks), `go_live.sh`,
  `final_go_live_check.py`, `quant_validation.py`,
  `oos_walkforward_calibrate.py`, `strategy_change_gate.py`,
  `backup_restore_validate.py`, `reconcile_migrations.sh`, `benchmark_latency.sh`.
- Backup/DR: pat-backup-sync → Hetzner S3 (WAL streaming + pg_dump),
  `scripts/backup/`, restore validation script.
- Time model: UTC is internal truth (TIMESTAMPTZ everywhere); broker time is
  GMT+2 winter / GMT+3 summer (Equiti, DST-following) observed live from the
  Master Node tick offset — authoritative for hour-of-day logic (sessions,
  ORB, swap windows). `/health` reports `time_mode: BROKER_ALIGNED` +
  `broker_offset`.

## 14. Non-negotiable rules (from AGENTS.md)

1. Safety precedence always outranks convenience: data quality → hard risk
   vetoes → news/session → cost limits → margin/exposure → broker constraints
   → license/entitlement → TTL/idempotency → emergency stop → ledger
   correctness → security.
2. **NO-TRADE is a first-class result.** Never force trades for frequency.
3. **Never fabricate**: ticks, fills, P&L, order flow, CVD/DOM, confidence,
   performance claims, AI activity. Demo/replay must be unmistakably labeled.
4. Production mutation boundary: no live orders, subscription/payout
   mutations, destructive migrations, key rotation, secret exports, DB
   superuser grants, DNS changes without explicit operator authorization.
5. Auto-push after every change: `git add -A && git commit && git push origin main`.
6. Docker-first; systemd disabled; every compose command uses
   `--env-file infra/env/.env`.
7. Raw score is not probability; subscriber-facing probability must be
   calibrated (VALIDATED-gated).
8. Financial truth is exact-decimal, transactional, idempotent, ledger-backed,
   isolated from the trading hot path.

## 15. Key numbers (v1.29)

| Metric | Value |
|---|---|
| Go tests | 39 packages pass (container toolchain) |
| Python tests | 154 (152 pass, 2 skip) |
| Frontend tests | 84 + 18 e2e; tsc clean |
| Control tests | 14 suites / 174 pass; tsc clean |
| DB migrations | 99→138 (unique prefixes), all applied |
| Indicators | 42 (35 live, 7 warming) |
| ML models | 5 (bootstrap placeholders, honest label) |
| Strategies | 7 |
| Gates | 16 server-side + capital gates + EA-side guards |
| API latency | < 3 ms |
| Edge EAs | 4 files, ~11.8k lines MQL |

---

# Part II — Plain-English Guide (the whole platform in plain words)

## 16. The one-paragraph version

Predict-A-Trade watches gold prices from real broker terminals, thinks about
them with a big box of indicators and strategies, and — only when enough
evidence stacks up AND every safety check passes — sends a small instruction
card ("buy here, stop there, target there") to the trader's MetaTrader
terminal, which places the trade on the trader's own account. The platform
sells access by subscription, tracks licences and devices, and shows
everything on web dashboards. It refuses to trade far more often than it
trades, by design.

## 17. The cast of characters

| Piece | Plain role |
|---|---|
| Master EA | A microphone in a real MT4/MT5 terminal — streams every price tick up to the cloud |
| Go engine | The brain — turns ticks into candles, candles into indicator readings, readings into decisions |
| Strategy engines | Seven specialists, each with its own personality (quick scalper, patient swing trader…) |
| Risk gates | The bouncer — 16+ checks every signal must survive |
| Client EA | The hands — receives approved instructions and clicks the buttons on the trader's own broker account |
| Control plane | The back office — accounts, subscriptions, licences, devices, partner payouts |
| Web dashboards | The windows — show what's happening, decide nothing |
| Postgres/Timescale | The memory — every tick, candle, signal, payment, audit event |
| Valkey | The sticky notes — fast hot-state, never the source of truth |
| Prometheus/Grafana | The gauges — system health and latency |

## 18. A signal's life, told simply

1. **A tick arrives.** A Master EA posts the latest gold price (and finished
   bars) to the engine over HTTPS. The engine stamps it with *its own clock*,
   because the trader's PC clock might be wrong.
2. **Candles form.** Ticks are grouped into M1/M5/M15/… bars. A bar is only
   used once it's fully closed — never a half-formed candle.
3. **Indicators compute.** 42 measurements (trend, momentum, volatility,
   structure, liquidity…) update for that timeframe, each in its own little
   room so M1 math never contaminates H4 math.
4. **Strategies vote.** Each strategy that "owns" this timeframe collects the
   indicator opinions into buy-evidence vs sell-evidence, and produces a
   direction + score. Most of the time the honest answer is NO-TRADE.
5. **Gates filter.** Spread too wide? Margin thin? News about to drop?
   Daily loss cap hit? Proven-losing strategy? Any single no → the signal is
   blocked (but still shown on the dashboard with the reason).
6. **Sizing & tiers.** For surviving signals, the engine computes entry,
   stop, three targets, a risk-appropriate lot, and which account sizes
   (MICRO/STANDARD/PRO) this trade geometry even fits.
7. **Delivery.** The signal is queued per-device in SQL: only devices with an
   active licence whose plan includes this strategy, in the right capital
   tier, get it. The Client EA polls every ~3 seconds, receives it, and
   trades it on its own account — then ACKs back.
8. **Reconciliation.** The server verifies the ACK: did the stop loss actually
   get set, matching the server's value? A position with no stop gets closed
   by command. Three violations → the device is disconnected. Safety is
   enforced server-side, not trusted to the edge.

## 19. Why this architecture (the reasoning behind the rules)

- **Money never transits us.** The platform sends numbers, not orders to a
  pooled account. The trader's broker, the trader's account, the trader's
  risk. This keeps the platform out of the funds-custody business entirely.
- **One brain, thin edge.** All intelligence lives server-side, so it can be
  fixed, versioned, audited and rolled back centrally. The EA is deliberately
  dumb (plus its own safety guards) — no secrets, no strategy logic to steal.
- **Fail closed everywhere.** If data is stale, a gate errors, or a licence
  can't be resolved, the answer is "don't trade". Uncertainty can only
  suppress action, never enable it.
- **Honesty as a feature.** No fake ticks, no fake "live" badge when the feed
  is down, no invented win-rates. ML models are labelled bootstrap until
  actually trained; sentiment is labelled NOT_AI_VERIFIED. A trading product
  lives or dies on trust in the numbers.
- **Planes that can't corrupt each other.** A billing bug can never pause
  your signals; a dashboard bug can never re-risk a trade; a Python experiment
  can never sit in the live tick path.
- **Everything is replayable.** Every signal keeps its full evidence list,
  gate results and versions, so any historical decision can be reconstructed
  and audited.

## 20. Who sees what (commercial model)

- **Plans:** FREE (one strategy, capped daily signals), STANDARD (+ swing),
  PRO (core four), ELITE (all seven). Visibility is enforced server-side —
  the dashboard shows what your plan allows; there's nothing to bypass.
- **Payments:** USDT crypto via NOWPayments with HMAC-verified callbacks and
  underpayment handling. Card payments are deliberately disabled.
- **Licensing:** a licence key activates devices (MT4/MT5, Master/Client
  roles); entitlement leases keep the edge honest between polls.
- **Partners:** referrals, commissions and payouts run on an exact-decimal
  ledger with rules, caps, holds and reversals — append-only, never edited.
- **Admin console:** operations (halt trading, pause signals, kill a
  strategy), signal engine telemetry, device risk events, indicator monitor,
  market data, finance — an operations console, not a second trading terminal.

## 21. A day in the life of the platform

- **Market opens (broker Sunday night):** Master EAs reconnect and resume
  streaming; the engine's health monitor spots the fresh feed; indicators
  warm up on the first bars; the first honest signals flow to dashboards.
- **London/NY overlap:** all strategies active, most signals generated in
  these liquid hours; ORB tracks session opening ranges; spread/cost gates
  are most protective here.
- **A news window (e.g. NFP):** the calendar provider raises news risk →
  strategies pre-emptively stand down (NTHighNewsRisk); nothing trades
  through the spike.
- **A quiet Asian session:** scores sit below candidate bars → a stream of
  NO-TRADEs and occasional BUY_CANDIDATE advisories. This is correct.
- **A device loses its connection:** heartbeats stop; the queue holds its
  signals until TTL expiry; the connectivity watchdog alerts; nothing is
  executed on a stale link.
- **A losing streak on an account:** daily-loss and seed-capital gates trip,
  the account stops receiving executable signals until it cools down —
  protection, not punishment.
- **Nightly:** backups stream to Hetzner S3, retention jobs trim old chunks,
  migrations stay reconciled, Grafana keeps the gauges.

## 22. Glossary (project-level)

| Term | Meaning |
|---|---|
| Plane | An isolated technology domain with a hard boundary (trading / research / control / presentation / edge) |
| Option B | The v1.19.0 transport: EAs talk directly to the cloud; no Windows Agent |
| Master / Client EA | Data-streaming EA vs signal-executing EA |
| edge-poll | The ~3s HMAC-signed HTTPS poll by which Client EAs receive signals/commands |
| EXECUTION_ACK | The client's confirmation of execution; server verifies SL/TP |
| Capital tier | Account-size band (MICRO/STANDARD/PRO) that filters which signals are viable |
| EligibleTiers | Per-signal list of account-size bands the trade geometry fits |
| Entitlement | What the subscriber's plan allows (strategies, frequency) |
| Quality grade | A+/A/B/REJECTED badge on signals |
| EV_R | Expected value per unit risk |
| Walk-forward / OOS | Out-of-sample backtesting discipline in the research plane |
| Calibration | Score → honest probability, gated on validation |
| Hypertable | TimescaleDB time-partitioned table (candles, client events) |
| Retention policy | How long old market/audit rows are kept before trimming |
| Fail closed | Doubt blocks action; never the reverse |
| Emergency stop / kill switch | Server commands that close positions and halt/disconnect |
| SOW | The Scope of Work — the canonical implementation contract |

## 23. FAQ

**Q: Is this live trading?**
Signals are live; execution happens on each client's own account via their
EA. Server-side arming exists but capital-protection gates keep everything
ADVISORY until a verified broker equity/order feed is present. No claims of
profitability are made without evidence.

**Q: Where does the price data come from?**
The Master EA in a real MT4/MT5 terminal (Equiti, XAUUSD) — authoritative.
Twelve Data/FMP are macro cross-checks only. No fabricated ticks, ever.

**Q: What happens if the Master terminal goes offline?**
The feed honestly reports NO_DATA; dashboards show stale/no-data states;
the silent-feed watchdog alerts via ntfy and nudges with REQUEST_SNAPSHOT.
No signals are generated on stale prices.

**Q: Can two clients interfere with each other?**
No. Delivery is per-device with per-client margin checks — one blown account
can never block another's signals.

**Q: What if the control plane dies?**
It runs as an HA pair (pat-control + pat-control-b) with nginx failover;
there's a runbook for the edge-poll 502 scenario.

**Q: How do I deploy a change?**
Build + up the affected service with the env-file, per AGENTS.md; tests and
lint first (make test / lint); migrations forward-only via migrate.sh; and
push every change immediately (auto-push rule).

**Q: Where's the indicator maths?**
`indicator.md` (same repo root) — the deep companion to this file.

**Q: Where are the canonical docs?**
`docs/INDEX.md` is the index; `docs.md` is the plain-words pipeline; this
file is the project brief; `MANIFEST.md` is the structure/inventory contract.

---

*Maintainer note: keep this file in sync with MANIFEST.md, AGENTS.md and
docs/. When services, strategies, gates, tiers, or deployment topology change,
update this brief in the same commit.*