# 01 — System Architecture Map (as-built, from source + runtime)

Verified against running stack (`docker compose ps`, all 10 services up) and code invocation chains from entrypoints. Documentation claims in `MANIFEST.md` were cross-checked and corrected below.

## Planes

| Plane | Tech | Entrypoint | Port (published) | Runtime status |
|---|---|---|---|---|
| Real-Time Trading | Go 1.x `realtime/cmd/realtime-engine` | `main.go` | 13081 (0.0.0.0 — **must be private**) | healthy; 1 agent connected |
| Intelligence/Research | Python `research/src/patresearch` | pytest/scripts | none | offline tooling only |
| SaaS Control | NestJS `control/src/main.ts` | port 13080 | 13080 public via docker publish + nginx `/api` | healthy |
| Presentation | Next.js 16 `frontend` | port 13082 | 13082 public + nginx | healthy |
| Status page | Node `status/server.js` | 13083 | published | up |
| Edge | nginx (repo-root `nginx/`, NOT `infra/nginx`) | 80/443 | public | TLS OK for *.predictatrade.com |
| Data | TimescaleDB pg17 (timescaledb 2.29.1, toolkit, vector, pgcrypto, uuid-ossp) | 5432 | **public — P0** | healthy |
| Cache | Valkey 8.0 | 6379 | **public — P0** | healthy |
| Monitoring | Prometheus (9090 internal), Grafana (3001 public), ntfy (8091 public) | | | up |

## Critical data flow (verified wiring)

```
MT4/MT5 EA ⇄ windows-agent (Go) ⇄ WS /ws/v1/agent (NO AUTH — F-003)
   → AgentProvider.HandleAgentMessage → validation → processTick
   → Aggregator (M1..D1 OHLC) → Valkey candle cache + market.candles/ticks (Timescale hypertables)
   → FeatureEngine (13 engines, closed-bar) → RegimeEngine v2.0.0
   → 4 strategies (pkg/strategy configs) → Scorer (evidence-sum) → Calibration consumer (UNTRAINED sigmoid)
   → GateRegistry.EvaluateAll (12 of 14 registered gates evaluated; fixed order slice)
   → signal engine (fingerprint SETNX dedup, cooldown, sequence refs)
   → Postgres: trading.signals / signal_candidates / risk_decisions / strategy_evaluations / signal_outbox(stuck)
   → BroadcastSignalToAll → Frontend hub (entitlement filter DEAD → no WS signals to UI)
                            → AgentHub (unfiltered broadcast to agents)
   → REST /api/v1/signals (NO AUTH) ← Next.js dashboards poll
Control plane: IAM/JWT/MFA/RBAC → licensing/devices → subscriptions (stub billing) → referrals/commissions(dead)/payouts(broken)
```

## Subsystem ownership map (condensed)

| Subsystem | Owner files | Inputs | Outputs | DB | Failure behavior |
|---|---|---|---|---|---|
| Market ingestion | `realtime/internal/marketdata/*`, `main.go` processTick | agent ticks/snapshots | candles, ticks, market_states | market.* | drop-on-full backpressure; weekend gap honest |
| Indicators | `internal/features/indicators.go` + registry | closed candles | 42-feature vector | trading.indicator_history | NaN guards; capability-honest |
| Regime | `internal/features/regime.go` | EMA9/21/50, ADX, RSI, ATR | 5 reachable states of 11 defined | trading.regime_history | fallback RANGE; hysteresis 3-confirm/5-min dwell |
| Strategies | `pkg/strategy/*.go`, `configs/*.yaml` | features+regime | BUY/SELL/NO-TRADE/WAIT | trading.strategy_evaluations | per-strategy cooldown |
| Scoring | `pkg/strategy/scorer.go` | evidence contributions | RawScore 0–100 | trading.signals.raw_score | deterministic |
| Calibration | `internal/calibration/consumer.go` | score | CalibratedProbability | signals.calibrated_probability | untrained default sigmoid — fabricated VALIDATED metadata |
| Gates | `internal/gates/*` registered `main.go:1712-1714` | gate states+inputs | VETO/PASS/DEGRADED | trading.risk_decisions | fail-closed core; order-slice omits 2 gates; slippage stub-pass |
| Signals | `internal/signal/*`, `main.go` pipeline | scored candidates | signals + refs PAT-XAU-YYYYMMDD-NNNNNN | trading.signals(+outbox stuck) | dedup fail-open if Valkey down |
| Delivery | `gateway/websocket.go`, `agent_ws.go`, `http.go resume` | signals | WS events w/ envelope | signal_deliveries=0 rows | ack path unwired |
| IAM | `control/src/modules/auth` | credentials | JWT access/refresh, MFA TOTP | iam.*, audit.audit_events | rotation+reuse detection OK |
| Licensing/devices | `control/src/modules/licensing`, `device-auth` | activation requests | device credentials, leases | licensing.* | IDORs on legacy routes (F-P1) |
| Billing | `control/src/modules/billing` | webhook POSTs | **stub ack only** | billing.payments (1 fake row) | signature never verified |
| Referrals/commissions/payouts | `control/src/modules/referrals|commissions|payouts` | REST reads; payout POST | ledger reads; broken writes | referral.* (ledger empty) | payouts INSERT fails 42703 |
| Frontend | `frontend/src/**` | REST polling + WS | UI | none direct | fabricates some metrics (F-P1) |

## Dead / duplicate inventory

Dead in live plane: `internal/rl`, `internal/adaptation`, `internal/hedging`, `internal/ml/adaptation.go`, `internal/sentiment/engine.go`, breakout/OCO engines, replay provider, outbox dispatcher methods, `DeliveryManager.AcknowledgeSignal`, `SaveSignalSafe`, entitlement setter for WS clients.
Duplicates: admin vs domain controllers for payouts/commissions/users/licenses; two device-revoke semantics; `/auth/me` ≈ `/users/me`; stale `control/openapi.json`; frontend dead `api-client.ts` + `marketDataWorker.ts`.
