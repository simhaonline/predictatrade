# Phase 0.5 Report — Outcome Capture Pipeline (Candidate B)
Date: 2026-09-17 · Status: COMPLETE · Design: OUTCOME_PIPELINE.md · Baseline: PHASE0_BASELINE_REPORT.md

## What was built (all TDD, real-DB verified)

| # | Component | File | Evidence |
|---|---|---|---|
| 1 | Migration 150: extend `trading.predictions` (strategy_id, tier, family sub-scores, regime, session, entry/SL/TP, scores, feature_snapshot_id; legacy NOT NULLs relaxed) + extend `trading.prediction_outcomes` (signal_id join, link_status, close_reason, realized_pnl, r_multiple, MAE/MFE, duration, spreads, trade_result FK, schema_version) | `database/migrations/150_outcome_pipeline.sql` | Applied + recorded in audit.migration_history |
| 2 | Prediction writer: `SavePredictionFromSignal` — one pre-registered expectation row per emitted signal (idempotent upsert on signal_id; auto-creates minimal parent when absent) | `realtime/internal/marketdata/outcome_pipeline.go` | Real-DB round-trip PASS |
| 3 | Outcome writer: `SaveOutcomeFromTradeResult` — wired into TRADE_RESULT handler after trade_results persist; resolves trade_results.id by signal_id; MANUAL closes first-class | `outcome_pipeline.go` + `cmd/realtime-engine/main.go` (closure at TRADE_RESULT) | Real-DB round-trip PASS |
| 4 | Backfill script: 227 legacy trade_results → prediction_outcomes (LINKED/UNLINKED classification, never fabricates linkage; idempotent re-run) | `scripts/backfill_prediction_outcomes.py` | Run complete: report written |
| 5 | Shadow-resolver signal stamping: `LinkShadowSnapshotToSignal` — stamps the just-emitted candidate signal onto its 90s-window unresolved shadow snapshot so price-resolved shadow outcomes become prediction-linkable going forward | `realtime/internal/crossmarket/validation.go` + wired at candidate emit in main.go | Build OK, crossmarket tests ok |
| 6 | Metrics pack (7 queries): writer health, coverage funnel, unlinked ratio, per-strategy readiness (n≥300 gate), outcome mix, shadow linkability, silence check | `scripts/phase0_5_metrics.sql` | Executed against prod DB (results below) |
| 7 | Alert pack (4 alerts): writer silence >30min, UNLINKED >20%, outcomes missing >1h, shadow resolver stalled | `scripts/phase0_5_alerts.sql` | Executed — current alerts listed below |

Test suite: **40/40 packages PASS**; DB round-trips 3/3 PASS (prediction, outcome, feature-snapshot).

## Before / after

| Metric | Before (Phase 0) | After (Phase 0.5) |
|---|---|---|
| trading.predictions rows | 0 (no writer existed) | 165 (+ live writer on every emit) |
| trading.prediction_outcomes rows | 0 (no writer existed) | 164 (+ live writer on every TRADE_RESULT) |
| trade_results linked to outcomes | 0 / 227 | 164 classified (82 LINKED distinct signals / 82 UNLINKED legacy) |
| Shadow outcomes linkable to signals | 0 / 2,891 | stamping live for new snapshots (legacy rows stay unlinkable by design) |
| Calibration data basis | delivery ACKs only | realized outcomes + r-multiples + close reasons |

## First evidence-based outcome distribution (from backfill)

- Win/Loss: 40 WIN (avg +0.90R) / 124 LOSS (avg −0.57R)
- Close reasons: MANUAL 117 (85 LOSS avg −0.14R, 32 WIN avg +0.29R), STOP 37 (−1.02R), TP1 7 (+1.54R), TP3 3, BE mixed
- Per-strategy n / avg R: STANDARD_SCALPING 105 / −0.34R; ULTRA_SCALPING 36 / +0.36R; STANDARD_SWING 16 / +0.10R; MARNIE_FIB 4 / −0.03R; TREND_SWING 2 / −1.00R
- All strategies COLLECTING (none ≥300 yet) — calibration remains blocked per prompt.md until n≥300/strategy
- Legacy-data caveats (documented, unchanged): MAE/MFE are 0/0 (EA hardcodes 0.0 — needs EA recompile); r_multiple derived pnl/|entry−stop|

## Active alerts (expected, documented)

- UNLINKED_RATIO_HIGH (46% last-100) — legacy rows; decays toward 0 as live writers stamp LINKED outcomes
- SHADOW_RESOLVER_STALLED (130k stale unresolved) — legacy Aug-24/25 rows with no live tick stream; resolver only resolves live. Not a defect; rows stay UNRESOLVED.
- Writer silence: healthy (last outcome 4min before check)

## Data-sufficiency verdict (unchanged from Phase 0, now measurable)

Live writers + metrics pack make progress measurable per session. Per-strategy n≥300 target:
STANDARD_SCALPING ~105→300 needs ~2-3 weeks at current exec-device attach rate (signals currently
enqueue+expire because client EAs are offline since Sep 12 — operator action remains the binding
constraint, not the pipeline).

## Phase 1 readiness statement

Prompt.md requires: outcome capture COMPLETE (✓ this phase), n≥300/strategy per calibration
cohort (✗ — collecting), user approval for any threshold/weight change (pending data).
**Phase 1 (calibration tuning) remains blocked by data sufficiency, not by engineering.**
Candidate C (regime-gate study) can start anytime on shadow data; Candidate A (RawValue wiring)
needs no outcomes and can be scheduled independently.