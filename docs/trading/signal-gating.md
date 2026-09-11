# Signal Gating & Delivery — Rebuilt Model (v1.19.x Signal-Funnel Fix)

This document explains how a candidate signal becomes `EXECUTABLE` (delivered to
MT4/MT5 EAs), the defect that previously starved ~99.5% of candidates, and the
rebuild that fixes it **without relaxing any HARD safety gate**.

## 1. The two surfaces (why the dashboard ≠ EA)

- `live.predictatrade.com` → `GET /api/v1/signals` → signals for your plan's
  allowed strategies, **including non-executable candidates** (no `executable`
  filter). This is by design: the dashboard is a research/transparency view.
- Your MT4/MT5 EA → `POST /api/v1/devices/edge-poll` → **only `Executable=true`**
  rows that pass plan/strategy entitlement **and** the gate stack.

So "dashboard shows signals but EA receives none" is expected whenever the
executable pool is empty for your plan — see §4 for the per-cause breakdown.

## 2. Gate taxonomy (prompt.md Sections 1-6)

Every gate is classified for reporting (carried on `GateEvaluation.Classification`):

| Class | Gates | Behavior |
|---|---|---|
| `HARD_SAFETY` | DataQuality, Session, News, Spread, Slippage, TotalCost, Margin, Exposure, RRNetExpectancy, License, ExecutionPermission | **Veto = never deliver.** Invariant across all profiles. |
| `EXECUTION_QUALITY` | OutcomeQuality, Drawdown, VolatilityRegime, Liquidity | Downgrade size/tier; never silently starves the strategy. |
| `SOFT_ALPHA` | **Profitability** | Decides `QualityTier` + `ShadowExecutable`. No longer a blanket veto. |
| `STRATEGY_SPECIFIC` | UniqueEntryGate (per strategy) | Reject inefficient entries; geometry distinct per strategy. |
| `ENTITLEMENT` / `DELIVERY` | Entitlement, License, edge-poll gating | Entitlement + license gate. |

Acceptance criterion #1 (HARD-safety regression) is enforced by
`gates/safety_regression_test.go`: any change that relaxes a HARD gate fails CI.

## 3. The defect that was suppressing ~99.5% of signals

`EvaluateProfitability` (strategy/refinement.go) previously fed `IsLossCandidate`
to the delivery `ProfitabilityGate`, which hard-vetoed on it. Two faults made
that flag fire on almost every candidate:

1. **Synthetic win-rate model used as a veto.** `estimateWinRate(score)` centered
   at score 55 → win rate 0.50. For any score ≤55 (most candidates) and a
   cost-skewed ~1:1 R:R, `EV = wr·netWin − (1−wr)·netLoss ≤ 0` → marked loss
   candidate. An **uncalibrated probability** was being used as a hard gate
   (forbidden by prompt.md §63).
2. **Broken micro-TP coverage test.** `MicroTPProfitable = microDist > cost`
   where `microDist = 0.5·ATR`. Whenever round-trip cost ≥ 0.5·ATR (normal for
   XAUUSD ECN), this was FALSE → `LossCandidate = true` **regardless of actual
   EV**. A structural scalping killer (scalping had 0 executable signals).

Net effect: a SOFT/alpha gate behaved as a 100% hard veto. That is the root cause
of "dashboard shows / EA gets none" at scale.

## 4. The rebuild (what changed)

### 4.1 Strategy-aware, sample-confident expectancy
`EvaluateProfitability` now uses a **versioned `GateProfile`** with a
**per-strategy assumed win rate** (a validated prior, not an uncalibrated
probability-as-veto) and a **confidence band**: `wr = assumed ± 0.12·(score-55)/45`.
EV uses the best TP, net of a single round-trip cost. The result is classified:

- `A_PLUS` / `A` / `B` → delivery-grade (`Executable = true`).
- `C` → delivered but lower grade.
- `WATCH` → **ShadowExecutable = true** (tracked, not force-executed).
- `REJECT` → only when EV is materially negative **AND** evidence is SUFFICIENT
  (≥ `MinEvidenceTrades` live outcomes). Never on UNKNOWN evidence.

### 4.2 Micro-TP fixed
`MicroTPProfitable = microDist > halfCost` (the partial close already recovered
part of the cost). Normal-cost scalping now passes.

### 4.3 `IsLossCandidate` redefined
Still exists for backward compatibility, but is now `true` **only** for
genuinely negative-EV-with-sufficient-evidence setups. It no longer nukes valid
candidates.

### 4.4 Delivery gate no longer blanket-vetoes
`ProfitabilityGate.Evaluate` maps the tier to `GatePass` + `ShadowExecutable`
(and `GateVeto` only for a real `REJECT`). It stays fail-closed when geometry or
score is genuinely absent.

### 4.5 Engine wiring
`signal/engine.go` derives `QualityTier` from the profitability gate eval and sets:
`Executable` (A+/A/B, not shadow), `ShadowOnly`, `ShadowExecutable`,
`QualityGrade`, and `GatePolicyVersion` on the emitted signal.

## 5. Shadow signals (prompt.md Section 79)

Soft-rejected candidates are emitted as `ShadowExecutable=true` / `ShadowOnly=true`
signals. They are **never auto-traded** but are recorded so we can measure how
often the soft gate would have caught a winner (calibration feedback loop). No
historical backfill is fabricated — `EvidenceQuality` is `SUFFICIENT` / `LIMITED`
/ `UNKNOWN` and only SUFFICIENT feeds the win-rate update.

## 6. Rollback (prompt.md Section 226)

The profile is selected at startup by the `GATE_PROFILE_VERSION` environment
variable on the realtime service (read by `gates/profile.go`). Values:

- `CONSERVATIVE` — most restrictive soft-alpha (pre-fix-adjacent behavior, still
  without the synthetic-veto bug).
- `BALANCED` — **default.** Restored executable flow with safety intact.
- `OPPORTUNITY` — most permissive soft-alpha; for measuring headroom only.

All three share the **identical HARD thresholds** (only the SOFT-ALPHA band
differs), so rollback is a one-line env change + restart — no code change, no
migration. Verified by `TestGateProfileRollback`.

## 7. How to verify after deploy

```bash
# 1) Re-run the dashboard-vs-EA diagnostic (see docs/runbooks/signal-delivery-verification.md)
python3 scripts/verify_signal_delivery.py

# 2) Confirm the realtime gate model changed version
docker compose logs realtime 2>/dev/null | grep -i gate_profile_version || true

# 3) Per-device deep dive on a silent EA
python3 scripts/verify_signal_delivery.py --device <device-uuid>
```

## 8. What this does NOT change

- No HARD gate is relaxed (proven by `gates/safety_regression_test.go`).
- No auto order-placement is enabled (Devil Liquidity stays signal-only; the
  Master-Node EA → AgentProvider feed and edge-poll EA-direct path are intact).
- Trailing-performance calibration is recorded but only SUFFICIENT-evidence
  samples influence the win-rate prior; no fabricated history.
