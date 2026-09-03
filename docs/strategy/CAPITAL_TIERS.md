# Capital-Tiered Signal Engine

## v1.23 — 3 September 2026

Every customer capital category gets tradeable, suitably-sized signals from the
same engine. Commits `3b56b96` (engine), `f86505d` (docs), plus the
v1.23.1 follow-up (unconditional tier evaluation + admin Signal Engine page).

## Admin visibility

**`/admin/signal-engine` — dedicated Admin Dashboard page** (nav: Trading
Operations → Signal Engine). Backed by the Go engine's
`/api/v1/admin/signal-engine` (admin-JWT gated, proxied via nginx
`location = /api/v1/admin/signal-engine` → realtime:13081). Shows: 24h
pipeline stats (enqueued/acked/expired/pending + tier-restricted count),
devices by capital tier with summed equity, delivery outcomes per tier, and
the last 12h of executable signals with per-signal `EligibleTiers`,
suggested lot and ack counts. 15s auto-refresh, PAT token design system.

## v1.23.1 fix — unconditional tier evaluation

The v1.23 initial wiring computed `EligibleTiers` inside the account-known
sizing guard. A stale/missing account snapshot (>60s since last ACCOUNT_INFO)
therefore silently degraded eligibility to empty → the enqueue default
(all-tiers) delivered wide-swing signals to tiny accounts. v1.23.1 computes
tier eligibility **unconditionally** from the seeded strategy geometry
(entry/SL after `seedMarketLevels` + `sanitizeStratResult`) before the gates,
and attaches it in both the sizing-run and sizing-skip branches. Tier
classification now depends only on signal geometry and round-trip cost —
never on account freshness. (Diagnosed live: signal `71217aff`, 13.9-pt
STANDARD_SWING stop, enqueued all-tiers despite exceeding the MICRO/STANDARD
caps because the account snapshot had gone stale at evaluation time.)

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

## Signal freshness + entry-drift gates (MT5 v1.24 / MT4 v1.25 — 3 Sep 2026)

Defense-in-depth hardening after a full signal-timing audit (master-node
broker time → UTC conversion verified accurate to 0.5s live; server-side
expiry already enforced at poll time in edge-poll before claim):

1. **`PAT_SignalFresh()` was dead code** — fully implemented in both EAs
   (server `ExpiresAt` first, `MaxSignalAgeSeconds` fallback) but never
   invoked. Now wired into the exec path before any order: expired signals
   are counted as filtered and refused, protecting against any future
   server-side sweep regression.
2. **Per-strategy entry-drift gate** — the EA executes at CURRENT market
   price with the signal's SL. Between engine decision and EA execution
   (2–15s normal, longer on spikes) price can run, silently distorting the
   R:R the engine sized for. Beyond the budget the signal is refused
   (fail-closed, logged `SIGNAL REJECTED (entry drift)`):

   | Strategy | Drift budget (points) | Rationale |
   |---|---|---|
   | ULTRA_SCALPING | 15 | 3m TTL, thin edge — tightest |
   | STANDARD_SCALPING | 25 | 10m TTL |
   | STANDARD_SWING | 60 | 60m TTL zone strategy |
   | TREND_SWING | 80 | 60m TTL, wider targets |
   | MARNIE_FIB / ATEN | 80 | zone/swing class |
   | unknown strategy | 60 | conservative default |

   Budgets sit above per-strategy slippage (5/10/20/30) so normal
   delivery latency never trips them; only genuine price runs do.

Both gates are EA-side only — no server change, no schema change. Sources
synced to `frontend/public/downloads/`. Recompile required on the Windows
MetaEditor side to activate. Admin per-tier counters remain a follow-up
(pipeline-monitor).