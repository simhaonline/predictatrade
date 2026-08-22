# 13 — Risk Management Audit (hard gates)

Framework: `internal/gates` — EvaluateAll iterates a **fixed order slice** (DataQuality→Session→News→Spread→Slippage→TotalCost→Exposure→Margin→RRNetExpectancy→Entitlement→License→ExecutionPermit), first VETO short-circuits; unregistered/uninitialized ⇒ UNKNOWN veto (fail-closed core).

## Gate-by-gate verdict

| Gate | Fail mode | Verdict |
|---|---|---|
| DataQuality | fail-closed on input; cached state auto-PASS | SOFT — freshness theater (`main.go:1736-1743`) |
| Session | input-driven, weekend/OFF blocked | PASS |
| News | vetoes HIGH/EXTREME/DATA_UNAVAILABLE/BLOCKED; NONE when provider disabled by operator env | PASS w/ operator caveat |
| Spread | >0.50 abs or >30% ATR veto | PASS (limits hardcoded main.go:1710) |
| **Slippage** | state seeded 0, never hydrated | **ALWAYS-PASS STUB** |
| TotalCost | RoundTrip/target >0.15 veto | PASS |
| Exposure | counts positions not notional | COARSE (5 trades ≠ exposure) |
| Margin | freeMargin>0 from agent telemetry | forgeable via unauth agent WS |
| RRNetExpectancy | grossRR≥2.0 enforced; net-expectancy half unimplemented | PARTIAL |
| Entitlement/License | GLOBAL flip: any ACTIVE row unlocks platform-wide | NOT per-subscriber |
| ExecutionPermit | any agent heartbeat within 60s | forgeable (F-003) |
| **StopHuntFilter / MinAbsoluteATR** | registered but ABSENT from order slice | **NEVER EVALUATED** |

## Missing safety controls (P0 cluster)

1. **Emergency stop:** `KillSwitch`/`ExecEmergencyStop` have zero usages; no endpoint/goroutine consults it. EA CAPITAL_PROTECTION/CAPITAL_WARNING messages are logged and dropped.
2. **Daily-loss protection:** `gates/capital_protection.go` + `trading.capital_protection_events` table exist; referenced only by `cmd/audit`. The 5% boundary math (§22) is UNVERIFIED because it is not in the live plane at all. `pat_daily_limit_hits_total=0` forever consistent with dead code.
3. Aggregate XAUUSD notional exposure limit absent (position-count only).

## Persistence & observability

Every evaluated gate decision persists to `trading.risk_decisions`; metrics `pat_entitlement_denial_total`, `pat_daily_limit_hits_total` exported. Gate policy/config version columns exist on signals (GatePolicyVersion) but InputHash/DecisionHash unpopulated.

**Verdict:** RISK MANAGEMENT = NOT VERIFIED for production. Deterministic skeleton is good; the three safety-critical controls (emergency stop, daily loss, true slippage) are absent/vacuous, two gates unreachable, and gate inputs are spoofable through F-003.
