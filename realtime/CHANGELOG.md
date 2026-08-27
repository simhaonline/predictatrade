# Predict-A-Trade Changelog

## v1.17.2 (27 August 2026) — CRITICAL: Master Node snapshot delivery (IPC truncation race)
### Problem
- The Go engine received `MASTER_TICK` from the Master Node EA but **never** a
  `MARKET_SNAPSHOT` (or `MASTER_INIT`). Verified live: `snapshot_count:0`,
  `last_snapshot_at:0`, market bars served from stale DB (~6h old) while ticks
  flowed. Signals were therefore evaluated on stale/absent market state and the
  live data feed was effectively dead despite agents being "connected".
- Root cause: `MasterAppend()` in the Master Node EA opened the IPC file with
  `FILE_WRITE`, which **truncates**. Because `MASTER_TICK` is written on every
  tick (far more often than the throttled `MARKET_SNAPSHOT`), each snapshot was
  clobbered by the next tick write before the Windows Agent's 5ms read loop could
  drain it. The one-time `MASTER_INIT` was also clobbered by the first tick.
- Confirmed the engine side is correct: a simulated Master Node delivering
  `MARKET_SNAPSHOT` over WS is accepted instantly (`snapshot_count` climbs,
  `data_health` → HEALTHY, `last_snapshot_at` updates, zero unmarshal errors).

### Fix
- **EA (MT4 + MT5 Master Node):** `MasterAppend()` now opens the IPC file
  `FILE_READ|FILE_WRITE` and `FileSeek(..., SEEK_END)` so it truly appends. Ticks
  and snapshots now coexist in `PAT_master_data.txt` until the Windows Agent
  reads and clears them — snapshots are no longer lost to the truncation race.
- **Engine observability:** added a production-safe `log` on `MARKET_SNAPSHOT`
  unmarshal failure (the exact silent-failure class that hid this bug).

### Verification
- Engine accepts snapshots end-to-end (simulated) → `data_health:HEALTHY`.
- Requires recompile + re-attach of the MT4/MT5 Master Node EAs for live flow.

## v1.17.1 (27 August 2026) — Silent Data-Feed Detection + Auto-Recovery
### Problem
- After a Windows Agent / EA restart, the live market-data (MARKET_SNAPSHOT) feed
  could go silent with NO error and NO alert: `healthManager.Update()` only ran
  inside `processCandle`, so when snapshots stopped, no candle formed, the health
  check never re-ran, and the engine went silently blind (observed ~1h of
  NO-TRADE with no operator signal). Ticks kept arriving, which masked the dead
  snapshot feed in naive liveness views.

### Fixes
- **Data-independent health monitor** (`cmd/realtime-engine/main.go` `startHealthMonitor`):
  a 10s ticker (not driven by market data) that runs the health assessment, emits
  `DATA_FEED_STALE` via ntfy on `STALE_DATA_CRITICAL` (60s warmup + agents
  connected), nudges agents with `REQUEST_SNAPSHOT` every 30s, and emits
  `DATA_FEED_RESTORED` + clears the alert on recovery.
- **Snapshot-receipt tracking** (`marketdata.AgentProvider.LastSnapshotAt`):
  market-state health is now based on MARKET_SNAPSHOT arrival, not bare ticks, so
  a lone tick can no longer hide a dead snapshot feed. `/api/v1/agents/status`
  now exposes `last_snapshot_at`, `last_market_data_at`, `data_stale_secs`,
  `data_health` (NO_DATA/HEALTHY/STALE/CRITICAL).
- **Agent (Go) recovery forwarding** (`windows-agent/internal/master.go` +
  `pipe.go`): `REQUEST_SNAPSHOT` now writes `PAT_resync.txt` into every
  MetaQuotes Common\Files folder the EA polls.
- **MQL resilience** (`mql/mt5/PredictATrade_MasterNode_MT5.mq5` +
  `mql/mt4/PredictATrade_MasterNode_MT4.mq4`): added `OnTimer()` (1s) that emits
  `MARKET_SNAPSHOT` independent of `OnTick` (ticks can stall if the broker quote
  stream hiccups), plus `PAT_resync.txt` polling that forces an immediate
  snapshot on a `REQUEST_SNAPSHOT` nudge. `EventSetTimer`/`EventKillTimer`
  lifecycle added to `OnInit`/`OnDeinit`.

### Safety
- All recovery paths are fail-open and observation-only: they never change
  strategy, risk gates, or trade eligibility. Alerts are deterministic and
  version/freshness stamped.

## v1.17.0 (27 August 2026) — Per-Client Risk Isolation + Ingest/Signal Decoupling
### New Features
- **Per-client risk isolation at signal delivery** (`AgentHub.SetRiskCheck` +
  `AgentProvider.AgentAccountOK`): executable signals are delivered only to
  clients whose OWN broker account has buying power. A blown/over-exposed client
  is isolated and can never block or contaminate another client's signals. The
  check is **fail-open** (unknown/stale account → allowed), so it cannot cause a
  global blackout.
- **Per-agent account registry** in `AgentProvider`: each client's broker account
  is tracked individually (no shared global account-driven gate). `agentID` is now
  threaded through `BrokerAccountHydrateFn`.
- **Ingest/signal decoupling seam** (`pkg/bus`): inbound Windows-Agent messages
  route through an abstract `IngestBus`. Default is `DirectBus` (in-process,
  identical to the prior direct call). `NatsBus` activates when `NATS_URL` is set,
  enqueuing messages on NATS which a subscriber dispatches to the engine — fully
  decoupling data-collection from the signal engine and enabling a separate ingest
  service. On NATS connect failure it falls back to in-process (safe no-op).
- **`pat-nats` service** added to `docker-compose.yml` (`nats:2.10-alpine`,
  JetStream enabled). Off by default; enable by setting
  `NATS_URL=nats://nats:4222` on the `realtime` service.

### Tests
- Per-client isolation unit tests (incl. stale fail-open) in
  `marketdata/agent_provider_account_test.go`.
- `pkg/bus` dispatch + NATS round-trip tests (round-trip skips when no NATS).
- All touched packages pass `go test -race`.

### Known follow-up (next sub-phase)
- Global `broker` equity still drives **lot sizing** for all clients. Per-client
  SIZING at delivery (built on the new per-agent registry) remains the next step.
- NATS path is verified by tests + fallback; a staging soak is recommended
  before enabling `NATS_URL` in production.

## v1.16.0 (26 August 2026) — P2 Activation + Production Readiness Audit
### New Features
- P2-001: Session ORB features ACTIVE — Asian/London/NY opening range computation, breakout detection, SESSION_ORB evidence pillar
- P2-002: Pin Bar geometry ACTIVE — body/wick ratios, rejection direction, quality score in CANDLE pillar
- P2-003: Pullback detection ACTIVE — depth %, ATR-normalized retracement, continuation confirmation in STRUCTURE pillar
- P2-004: Trade Group ID ACTIVE — auto-populated on multi-TP signals
- P2-005: SLO targets documented

### Production Readiness Audit
- 17-report production readiness audit completed (CONDITIONAL GO, 70/100)
- All 5 critical production blockers CLOSED
- 28/28 Go test packages pass
- All 16 risk gates verified

### Documentation Consolidation
- Consolidated all documentation into 6 primary files
- Removed 50+ stale/obsolete documents
- Root-level SCOPE_OF_WORK.md, DEPLOYMENT_GUIDE.md, API_REFERENCE.md, CHANGELOG.md

## v1.15.0 (25 August 2026) — Server-Side SL Enforcement
- 8 safety gaps closed: EXECUTION_ACK handler, position SL monitoring, CLOSE_POSITION, EMERGENCY_STOP, KILL_SWITCH
- Agent suspension for SL violations (3-strike)
- MQL EA v1.09, Windows Agent v1.2.18
- Legal compliance: Terms, Privacy, DPA published

## v1.14.0 (25 August 2026) — DXY Macro Health Fix
- DXY→macroHealth wiring fix (ML + Sentiment re-enabled)
- Calibration DB tables (migration 072)
- Signal engine audit (5 engines verified)

## v1.13.0 (25 August 2026) — CI/CD Stabilization
- All 6 CI jobs passing
- Go test race fix, React 19 peer-dep fix
- Security scan precision improvements

## v1.12.0 (25 August 2026) — Legal Compliance
- Market-standard login/signup with consent checkboxes
- 3 legal documents (Terms, Privacy, DPA)
- Backend consent tracking, migration 071

## v1.11.0 (24 August 2026) — Live Dashboard
- Neural shell indicator flow
- Bloomberg-style terminal polish
- Service worker cache bumps

## v1.10.0 (23 August 2026) — Cross-Check Remediation
- News risk wiring to strategy evaluation
- Migration 022 applied
- Guest preview registration gate

## v1.9.0 (22 August 2026) — Subscription Referral V3
- Subscription state machine
- Referral tracking with commission rules
- Multi-plan support with billing intervals

## v1.8.0 - v1.0.0
- Initial platform build: 5 strategy engines, 42 indicators, 12 risk gates
- Docker Compose architecture
- MT4/MT5 bridge with Windows agent
- PTB intelligence layer
- Backtesting framework