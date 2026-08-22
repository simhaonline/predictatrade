# 02 — Codebase Forensic Audit

Method: full-tree keyword sweep (`TODO|FIXME|HACK|mock|fake|dummy|stub|simulate|bypass|hardcoded|except Exception|pass`) with contextual evaluation, plus invocation-chain tracing. Zero TODO/FIXME/HACK comments survive in production code (scrubbed) — findings below are **behavioral**, found by tracing.

## Fabrication / simulation in production paths

| ID | Sev | Location | Evidence |
|---|---|---|---|
| 02-1 | P0 | `realtime/internal/calibration/consumer.go:83-107` | `SeedDefaultModels` stamps untrained sigmoid models `Status:"VALIDATED", SampleSize:100, WilsonLower:0.45, Brier:0.21`; no DB/loader path ever replaces them (`SetModel` called only by seeder+tests) |
| 02-2 | P1 | `realtime/cmd/realtime-engine/main.go:1143-1145` | ML + sentiment results computed then discarded: `_ = stratResult; _ = mlDir; _ = sentimentScore`. Injector `ApplyMLAndSentiment` referenced only by `scorer_test.go` → AI participation is theater while dashboards/config advertise it |
| 02-3 | P1 | `frontend/src/lib/use-signal-performance.ts:172,194-225` | Hit-Rate/Accuracy/Avg-R "estimated" client-side from directional conviction when no closed trades exist; missing evidence counted as match; renders as authoritative performance matrix |
| 02-4 | P1 | `frontend/src/components/trading/live-dashboard.tsx:44,51,58` | Hardcoded `2500/.00/.50` rendered as Bid/Ask/Spread before first tick — indistinguishable from live price to a user |
| 02-5 | P2 | `frontend/src/app/(admin)/admin/signals/page.tsx:65,69` | WS rows get `RawScore = probability*100` and `crypto.randomUUID()` IDs — invented engine fields |
| 02-6 | P2 | `control/src/modules/admin/admin.service.ts:430-437` | Control-plane health self-reports `HEALTHY, latency_ms:0` unconditionally |

## Stubs / bypasses / dead switches

- `control/billing.service.ts:19-22` webhook stub (unauthenticated, no verification) — highest-risk stub.
- Slippage gate state seeded `0`, never hydrated → always PASS (`gates/implementations.go:154-159`, `main.go:1714`).
- Registered-but-unreachable gates: StopHuntFilterGate, MinAbsoluteATRGate absent from fixed order slice (`gates/gates.go:83-96`, registered `main.go:1712-1713`).
- Device access tokens minted but never persisted/verifiable (`device-auth.service.ts:136-137,265`).
- `verifyRequestSignature` (nonce replay table) fully implemented, invoked nowhere.
- Dead modules with zero import chains from any `cmd/`: rl/, adaptation/, hedging/, ml/adaptation, sentiment/engine, breakout, oco, maintenance, recovery, replay provider. `NEWS_BREAKOUT_ENABLED=true` env parsed but nothing instantiated.
- Outbox dispatcher methods defined (`persistence.go:559-606`) — zero callers; 2697 rows prove it.
- Frontend dead code: `lib/api-client.ts` (0 callers), `workers/marketDataWorker.ts`.

## Error handling

>20 swallowed-error sites in control plane (audit-log inserts, rollback-swallow `catch{}` patterns at `auth.service.ts:362,537`, `device-auth.service.ts:186,281,362`). Go: sequence fetches ignore errors (`main.go:1175,1335,1379,1601` → ref degradation on DB blip); bootstrap scan skips silent. Dedup/cooldown fail-open when Valkey down (`cooldown.go:108-112`) — contradicts fail-closed precedence for duplicate protection.

## Concurrency / resource

- Subscription creation has no uniqueness on one-active-subscription-per-user → duplicate race.
- Licensing max-device/max-MT-account checks are read-check-insert without locks (TOCTOU).
- Reconciliation map grows unbounded, memory-only.
- No statement timeouts on the shared control pg Pool(max 20).

## Data-quality defects

- Candle aggregator persists bars with `open=0, low=0` labeled `quality=COMPLETE` (DB proof: 553 rows Aug 18–21, incl. final Friday M5) → structure/pattern/FVG features ingest corrupted bars.
- MT5 snapshot bars merged with `Time: now()` instead of bar time (`main.go:332,864`) → age math on context candles distorted.

## Tautological tests (coverage theater)

- `security-validation.spec.ts:21-26,35-38` asserts literals contain themselves; does not exercise real blocklist.
- Frontend `engineAlive = !!(length ?? 0) >= 0` always-true (`admin/dashboard/page.tsx:108`).

## Secrets hygiene (details in 30)

Plaintext provider keys/SMTP/Telegram in `infra/env/realtime.env`; committed compose creds; JWT-secret-derived AES key for device credentials with dev fallback `'pat_local_dev_secret_change_in_production'` reachable in device-auth paths not covered by startup guard.
