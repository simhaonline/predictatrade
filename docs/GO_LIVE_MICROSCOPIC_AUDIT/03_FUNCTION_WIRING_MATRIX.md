# 03 — Function Wiring Matrix (production-critical functions)

Legend: WIRED = reachable from entrypoint in prod config; DEAD = zero production callers; PARTIAL = wired with material gaps.

## Go realtime engine

| Function | Defined | Called from | Status | Evidence / consequence |
|---|---|---|---|---|
| `AgentProvider.HandleAgentMessage` | agent_provider.go | WS agent hub | **WIRED, UNAUTH** | any client can inject ticks/account_info |
| `FeatureEngine.Evaluate` | internal/features | main loop per closed candle | WIRED | 13 engines registered |
| `RegimeEngine.Evaluate` v2.0.0 | features/regime.go | main loop | WIRED | BREAKOUT state unreachable |
| Strategy `Evaluate` ×4 | pkg/strategy | engines addon | WIRED | BUY/SELL/NO-TRADE all reachable |
| `Scorer.Score` | pkg/strategy/scorer.go | pipeline | WIRED | deterministic evidence-sum |
| `CalibrationService.Apply` | calibration/consumer.go | pipeline | WIRED but UNTRAINED | fabricated VALIDATED metadata |
| `StopHuntFilterGate.Evaluate` | gates/stop_hunt_filter.go | — | **DEAD** | missing from order slice |
| `MinAbsoluteATRGate.Evaluate` | gates | — | **DEAD** | missing from order slice |
| `SlippageGate.Evaluate` | gates/implementations.go:154 | EvaluateAll | WIRED but vacuous | state never hydrated → always PASS |
| `EntitlementGate/LicenseGate` hydrate | main.go:1792-1866 | 10s ticker | WIRED but GLOBAL | any ACTIVE row flips for everyone; not per-user |
| `DeliveryManager.MarkExpired` | gateway/delivery.go:189 | — | **DEAD** | no expiry sweeper; expired signals linger |
| `DeliveryManager.AcknowledgeSignal` | delivery.go | — | **DEAD** | EA CLOSE_ACK logged & dropped (`agent_provider.go:519-533`) |
| `GetPendingOutboxEvents/Mark*` | persistence.go:559-606 | — | **DEAD** | 2697 outbox rows PENDING forever (DB-verified) |
| `SaveSignalSafe` | persistence.go:644 | — | **DEAD** | all call sites use raw SaveSignal |
| `KillSwitch` / `ExecEmergencyStop` | types.go:567,519 | — | **DEAD** | no emergency stop exists in live plane |
| `Reconciler.RecordDelivery/Ack/UpdateStatus` | reconciler.go | — | **DEAD** | reconciliation memory-only stub |
| `mlengine.LoadAndRun` | pkg/mlengine | main per candle | WIRED-BUT-INERT | models absent at runtime → fail-open; results discarded anyway |
| Ollama client `Analyze` | pkg/ollama/client.go | discarded ML block | WIRED-BUT-INERT | host `localhost:11434` unreachable from container → always neutral fallback |
| PTB engine Evaluate | internal/ptb | main.go:1076 | WIRED (SHADOW) | ScoreContrib forced 0 — honest |
| SimulatedProvider | marketdata | guarded | DISABLED in prod | NODE_ENV=production guard |

## NestJS control

| Function | Status | Note |
|---|---|---|
| Auth register/login/MFA/refresh/reset | WIRED | rotation+advisory locks OK; audit writes best-effort |
| `BillingService.handleWebhook` | **STUB** | logs + ack; unauthenticated route; signature_verified=false row exists in DB |
| Subscription create | WIRED-PARTIAL | INCOMPLETE→ACTIVE transition has **no writer**; the one ACTIVE sub was created by fake webhook path/manual insert |
| `CommissionEngine.CalculateCommissions` | **DEAD** | imported only by spec; commission_ledger=0 rows |
| `PayoutsService.requestPayout/approvePayout` | WIRED-BROKEN | INSERT references nonexistent column `amount` → 42703 runtime failure; approval flips status without ledger debit |
| `verifyRequestSignature` (device protocol) | **DEAD** | nonce replay protection unused |
| Entitlement policy validator | WIRED | enforced only at subscription creation; not at delivery |
| Signal quota ledger writer | **DOES NOT EXIST** | `trading.signal_delivery_ledger` empty; monthly limits unenforced |

## Frontend

WS entitlement filter `isEntitled()` exists server-side in Go but `Client.entitlements` is never populated → dashboard WS signal push unreachable (UI falls back to REST polling). Frontend recomputes RR/trend/MTF-consensus client-side (violates presentation-plane authority), fabricates performance metrics.

**Conclusion:** implemented-but-never-called logic is pervasive on safety-critical paths (emergency stop, expiry, ack/reconciliation, outbox drain, commissions). Per §5 these are BROKEN wiring even though code exists.
