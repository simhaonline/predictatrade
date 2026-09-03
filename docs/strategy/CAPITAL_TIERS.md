# Capital-Tiered Signal Engine

## v1.23 — 3 September 2026

Every customer capital category gets tradeable, suitably-sized signals from the
same engine. Commit `3b56b96`.

## Requirement (user directive)

Customers must be served by capital band:

| Tier | Equity band |
|---|---|
| `MICRO` | < $500 (floor $100) |
| `STANDARD` | $500 – $4,999.99 |
| `PRO` | ≥ $5,000 |

Subscription-plan entitlements (control.plans / plan_entitlements) compose with
capital tiers — they are separate axes and neither weakens the other.

## Design decision: ONE engine, tier-aware (Option A)

Not three engine processes. The engine evaluates all strategies once; each
executable signal computes **per-tier viability** and carries it; the delivery
fan-out filters per device. Same coverage, one gate state, one ops surface.

## Mechanics

### 1. Tier classification (per device)

Each EA heartbeat (and ACCOUNT_INFO ingest) streams account **equity**. The
engine/control plane upserts:

- `licensing.edge_device_state.last_equity` (numeric)
- `licensing.edge_device_state.capital_tier` — `MICRO | STANDARD | PRO | ''`
- `licensing.edge_device_state.tier_changed_at` — audit trail of tier moves

`''` (unknown) = device never streamed equity. Unknown devices are **never
tier-excluded** (fail-open), so a telemetry gap cannot silently starve signals.

### 2. Per-tier signal viability (`realtime/internal/capitaltier`)

At signal finalization (sizing annotation block, main.go) the engine computes
`EligibleTiers` from real geometry:

```
minLotRisk$ = SL_distance_points × $1   (XAUUSD: 0.01 lot ≈ $1 per 1.00 point)
tierCap$    = reference_equity × tier_cap%   (reference: $100 / $500 / $5,000)
eligible(t) ⇔ minLotRisk$ ≤ cap(t)
```

- Tight scalp (SL ≤ 2pt) → eligible for **all three tiers**.
- 22-point swing stop → **PRO only** (would risk $22 ≥ min-lot on a $100–500
  account — exactly the "Lot: —" failure mode, now structurally prevented).

Restrictions are logged: `[CAPITAL-TIER] signal restricted to capital
categories by min-lot risk` with per-tier exclusion reasons.

### 3. Tiered delivery (enqueue fan-out, main.go `enqueueSignalForDevices`)

The enqueue SQL LEFT JOINs `edge_device_state` and delivers only when the
device's `capital_tier` is in the payload's `EligibleTiers`. Fail-open rules:

- payload without a JSON array `EligibleTiers` (legacy) → deliver
- device tier unknown/`''` → deliver
- nil slice default: enqueue defaults to all three tiers before marshal (JSON
  `null` would otherwise exclude every known-tier device)

### 4. Per-tier sizing cap

At the sizing call site: effective per-trade cap = **min(plan
`per_trade_risk_pct`, tier cap)**. Tier cap = 2% for all concrete tiers. The
tier cap can only tighten. Unknown tier → plan cap governs.

### 5. EA-side lot backstop (MT5 v1.23)

The payload carries ONE `SuggestedLot` (sized from the funded account
snapshot). MT5 `ExecuteBuy`/`ExecuteSell` now cap it:

```
vol = min(server_lot, PAT_CalcLotSize(own_equity, |entry−SL|))
```

so a lot sized for a large account can never execute on a small terminal
(fail-closed `REJECTED lot_below_min` if the capped lot is under broker
minimum). MT4 v1.24 already sizes from its own equity and additionally streams
equity in its heartbeat.

## Data flow

```
EA heartbeat {terminal, equity}                    (per device, every ~60s)
  → control edge-poll /edge-heartbeat
  → edge_device_state.last_equity + capital_tier
Engine sizes signal for funded account, computes EligibleTiers (SL distance)
  → enqueueSignalForDevices SQL: device.capital_tier ∈ payload.EligibleTiers
  → edge_signal_queue → EA poll → execute
```

## Safety invariants kept

- No cap weakened: tier composes as min(plan, tier); unknown tier fail-open.
- Fail-closed sizing unchanged; NO-TRADE remains valid.
- `trading.agent_user_bindings` untouched.
- EA executes what the server sends; the min() backstop only prevents a
  larger-account lot on a smaller terminal.

## Verification performed (live, 3 Sep 2026)

- `go test ./internal/capitaltier/` + full `go test ./...` pass.
- Real signal 07:31 UTC (STANDARD_SCALPING, 22.14pt SL) logged
  `eligible_tiers: [PRO]` with MICRO/STANDARD exclusion reasons.
- Delivery SQL dry-run on live devices: PRO-only payload → MICRO + STANDARD
  devices excluded, unknown-tier device delivered (fail-open); tight payloads
  → all delivered.
- Migration 120 applied; 118 unblocked (guarded COMMENT on dropped table).
- Containers rebuilt + healthy (control 07:29, realtime 07:29).

## Rollout notes

- Deployed EAs (≤ v1.22) don't stream heartbeat equity yet → tiers stay
  unknown → **fail-open delivery (current behavior) until recompile**.
  After MT5 v1.23 / MT4 v1.24 compile, classification starts automatically.
- Admin dashboards: per-tier counters are a follow-up (pipeline-monitor).