# Predict-A-Trade Changelog

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