# Risk Gates
## v1.16.0 — 26 August 2026

### Gate Pipeline (16 gates, ordered execution)

| # | Gate | Type | Behaviour |
|---|------|------|-----------|
| 1 | ExecutionPermission | Entitlement | Fail-closed |
| 2 | BrokerSymbolValidation | P0 safety | Degrade on missing meta |
| 3 | SeedCapitalProtection | Capital | Fail-closed |
| 4 | DailyLossLimit | Capital | Fail-closed |
| 5 | MaxSpread | Market | Fail-closed |
| 6 | NewsRisk | Event | Fail-closed |
| 7 | Slippage | Execution | Fail-closed |
| 8 | MaxPositions | Exposure | Fail-closed |
| 9 | MaxExposure | Exposure | Fail-closed |
| 10 | Cooldown | Timing | Fail-closed |
| 11 | StopHuntFilter | Structural | Advisory |
| 12 | MarginCheck | Broker | Fail-closed |
| 13 | OvertradeProtection | Frequency | Fail-closed |
| 14 | MaxDailyTrades | Frequency | Fail-closed |
| 15 | RegimeFilter | Market | Advisory |
| 16 | ProfitTarget | Capital | Fail-closed |

### P0-001: BrokerSymbolValidationGate (v1.16.0)
Validates SL/TP/lot against broker symbol metadata (min stop, min freeze, max spread). Degrades (doesn't veto) when broker metadata unavailable. Price rounding to broker digits applied in signal engine (P1-001).

### Safety Principles
- All gates registered via `RegisterOrdered()` — order is enforced
- NO-TRADE is a valid first-class result
- Gate failures produce distinct status (never masked as NO-TRADE)
- Engine liveness tracking distinguishes DEGRADED from NO-TRADE
