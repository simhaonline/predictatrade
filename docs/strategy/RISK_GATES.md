# Risk Gates
## Documentation — 10 September 2026 (engine v1.24.2)

> **v1.23 capital tiers:** signal delivery and sizing are now capital-tier
> aware (MICRO < $500 / STANDARD $500–5k / PRO ≥ $5k). Effective per-trade
> cap = min(plan cap, tier cap 2%). See [CAPITAL_TIERS.md](CAPITAL_TIERS.md).

### Gate Pipeline (24 registered gate IDs, ordered execution — source: types.go:242-268, gates.go)

The historical "16 gates" table below described v1.17-era names. The current
registry evaluates (first veto short-circuits; unregistered/uninitialized →
fail-closed veto):

DataQuality → WrongSideSL → Session → News → Spread → Slippage → TotalCost →
MinATR → StopHuntFilter → Exposure → Margin → RiskOversize → PositionCaps →
DailyLoss → ProfitTarget → MartingaleBan → RRNetExpectancy → Profitability →
Entitlement → License → ExecutionPermit → BrokerSymbolValidation →
EdgeValidation.

Highlights (source: `realtime/internal/gates/`):
- **DailyLossGate** (`capital_gates.go:296`): nested daily 5% / weekly 8% /
  monthly caps; soft recovery-band vs hard-halt at 2× cap; **fail-closed on
  unknown PnL** (`pnl_state_unknown` veto until a broker snapshot hydrates).
- **MarginGate** (`implementations.go:316`): state-driven headroom veto.
- **News BE-4**: news-data-unavailable/stale ⇒ VETO, never silently pass.
- **Staged capital halt** (`capital_protection.go`): soft 4% (block new
  entries) / hard 6% (halt) + dynamic risk tapering below $200 equity.
- Per-(strategy, timeframe) state isolation prevents cross-strategy
  contamination; operator edge-arming enables per-strategy broker-position
  authorization for EXECUTABLE delivery.

### Key Changes (v1.17.x → v1.23 delivery model)

**Per-Device Entitlement Delivery (EA-direct era, v1.19+):**
Beyond the server-side gate pipeline (which evaluates signal-worthiness at
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
