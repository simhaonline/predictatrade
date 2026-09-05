# Risk Gates
## v1.29.0 — 05 September 2026

> **v1.23 capital tiers:** signal delivery and sizing are now capital-tier
> aware (MICRO < $500 / STANDARD $500–5k / PRO ≥ $5k). Effective per-trade
> cap = min(plan cap, tier cap 2%). See [CAPITAL_TIERS.md](CAPITAL_TIERS.md).

### Gate Pipeline (16 gates, ordered execution)

| # | Gate | Type | Behaviour |
|---|------|------|-----------|
| 1 | ExecutionPermission | Entitlement | Fail-closed. Supports operator edge-arming per strategy for broker-position authorization. |
| 2 | BrokerSymbolValidation | P0 safety | Degrade on missing meta. Validates SL/TP/lot against broker symbol metadata. |
| 3 | SeedCapitalProtection | Capital | Fail-closed. 5% daily loss cap enforced per account equity snapshot. |
| 4 | DailyLossLimit | Capital | Fail-closed. Per-(strategy, timeframe) loss tracking. |
| 5 | MaxSpread | Market | Fail-closed. Relaxed thresholds for wider market conditions. |
| 6 | NewsRisk | Event | Fail-closed. |
| 7 | Slippage | Execution | Fail-closed. |
| 8 | MaxPositions | Exposure | Fail-closed. |
| 9 | MaxExposure | Exposure | Fail-closed. |
|10 | Cooldown | Timing | Fail-closed. |
|11 | StopHuntFilter | Structural | Advisory. |
|12 | MarginCheck | Broker | Fail-closed. |
|13 | OvertradeProtection | Frequency | Fail-closed. |
|14 | MaxDailyTrades | Frequency | Fail-closed. |
|15 | RegimeFilter | Market | Advisory. |
|16 | ProfitTarget | Capital | Fail-closed. |

### Key Changes (v1.17.x → v1.23 delivery model)

**Per-Device Entitlement Delivery (EA-direct era, v1.19+):**
Beyond the server-side 16-gate pipeline (which evaluates signal-worthiness at
generation time), signal **delivery** enforces per-receiving-device entitlement
in SQL at enqueue time (`enqueueSignalForDevices` → `licensing.edge_signal_queue`)
and re-checks it at poll time (control-plane `edge-poll` handler). An executable
signal is queued only for devices whose license is ACTIVE/PENDING, whose license
+ plan whitelist includes the signal's strategy, whose device role is `exec`,
and whose resolved capital tier matches the signal's `EligibleTiers` (unknown
tier or missing tier list → fail-open per the v1.23 delivery rules; unresolvable
plan → device skipped, fail-closed). One ineligible device can never suppress or
contaminate another device's signals.

**Account-state primitive:** `AgentProvider.AgentAccountOK` remains the
per-device broker-account guard (free margin > 0, snapshot < 60s old). As of the
5244776 remediation it is **fail-closed**: unknown, stale, incomplete, or
no-buying-power account state rejects the receiving device.

### Key Changes (v1.16.x)

**Per-(Strategy, Timeframe) Gate Isolation:**
Gate state is now scoped to `(strategy_id, timeframe)` pairs. Each strategy+timeframe combination maintains independent gate tracking (cooldown timers, loss counters, trade counts), preventing cross-strategy contamination.

**Seed Capital Protection (5% Daily Loss Cap):**
New P0 fail-closed gate that enforces a 5% daily capital loss limit. Engine computes account-size-aware position sizing from broker account snapshots and annotates every signal with `SuggestedLot`, `RiskDollars`, `RiskPctOfEquity`, and `SLDistancePoints`.

**Operator Edge-Arming & Broker-Position Authorization:**
`ExecutionPermission` gate now supports per-strategy operator arming. When armed for a strategy, the gate requires broker-position authorization before delivering EXECUTABLE-class signals. This enables controlled live trading with an explicit operator approval step.

**ProfitabilityGate — Entry Veto Removed:**
Hard veto removed from ProfitabilityGate for entry decisions. The profitability filter now degrades to advisory rather than blocking entry, while still enforcing fail-closed on exit decisions.

**Fail-Closed Capital Veto:**
Strategies with proven-negative live-edge performance receive a fail-closed veto (`cc8353a`). This prevents strategies whose live performance is demonstrably negative from continuing to generate EXECUTABLE signals.

### P0-001: BrokerSymbolValidationGate
Validates SL/TP/lot against broker symbol metadata (min stop, min freeze, max spread). Degrades (doesn't veto) when broker metadata unavailable. Price rounding to broker digits applied in signal engine (P1-001).

### Safety Principles
- All gates registered via `RegisterOrdered()` — order is enforced
- NO-TRADE is a valid first-class result
- Gate failures produce distinct status (never masked as NO-TRADE)
- Engine liveness tracking distinguishes DEGRADED from NO-TRADE
- Gate state is isolated per (strategy, timeframe) — no cross-contamination

> **Distinction — server gates vs. client EA guard:** The gates above are enforced by the **Go real-time engine** (server-side). The MetaTrader **Client EA** additionally enforces its own client-side daily-loss guard, which is independent of these gates: a **soft** limit (`WarningLossPct`) blocks new entries only and recovers intraday (bypassable via the `BypassDailyLossBlock` EA input), while a **hard** limit (`MaxDailyLossPct`) closes all positions and is never bypassable. `AutoExecute` defaults to **false** (signal-only). See the [EA Client Guide](../guides/EA_CLIENT_GUIDE.md).
