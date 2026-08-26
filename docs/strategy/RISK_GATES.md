# Risk Gates
## v1.16.0 — 26 August 2026

### Gate Pipeline (16 gates, ordered)

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
Validates SL/TP/lot against broker symbol metadata. Degrades (doesn't veto) when metadata unavailable. Rounding applied in signal engine.

### Safety Principles
All gates registered via RegisterOrdered(). NO-TRADE is valid first-class result. Gate failures produce distinct status from NO-TRADE.
