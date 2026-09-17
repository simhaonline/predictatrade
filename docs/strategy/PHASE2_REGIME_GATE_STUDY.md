# Phase 2 Study Design — Candidate C: Regime-Gate Effectiveness
Pre-registered: 2026-09-17 (Phase 0.9 Task E) · Status: DESIGN ONLY — NOT EXECUTED
Frozen per the same amendment procedure as PHASE1_CALIBRATION_PROTOCOL.md §9.

## 1. What still blocks Candidate C (confirmed in writing)
1. **Sample size**: gate is ≥300 linked EXECUTION outcomes per strategy; current max
   47 (STANDARD_SCALPING). The regime-conditioned analysis needs ≥300 per
   (strategy × regime) cell — worst case ~2-3× the Phase-1 gate. BLOCKED.
2. **Walk-forward feasibility**: per-regime cells make each walk window smaller;
   feasibility requires ≥150 per regime cell for train+val and ≥100 OOS — not
   achievable until the Phase-1 collection has run for several weeks. BLOCKED.
3. **p < 0.05 achievability**: with realistic per-regime effect sizes (~0.1-0.3R),
   power analysis at n=300/cell detects 0.3R at ~55% power; 0.1R needs n≈900.
   Underpowered until EAs run for weeks. BLOCKED.
4. **Shadow-channel independence**: shadow data may inform the STUDY DESIGN
   (e.g., which regimes even occur) but never counts as execution evidence —
   the regime-gate counterfactual replay must run on execution-channel data.
   Design constraint accepted; no contamination path exists in the design below.

## 2. Hypothesis (pre-registered)
H-C1: The current binary regime gate (RegimeDirectionGate) rejects trades that
would be profitable in RANGE regimes (71% of baseline trades were RANGE with
−$97 net — the gate may be mis-conditional, not wrong).
H-C2: A regime-conditioned candidate/trade bar (per-strategy × regime thresholds)
dominates the binary gate on OOS expectancy without degrading max DD.

## 3. Counterfactual replay method
- Replay historical signals (execution channel, linked outcomes only) through a
  SIMULATED gate variant — never through live signal generation. Each replayed
  trade maps 1:1 to a REAL stored outcome (r_multiple); no trade is invented.
- The replay must preserve: entry/SL/TP geometry as stored, close_reason, and
  the original outcome. Only the gate DECISION is counterfactual.
- Baseline = actual outcomes (what happened). Candidate = actual outcomes for
  signals the candidate gate would ALSO have passed. Signals the candidate gate
  would veto contribute 0R (no trade) — never a synthetic outcome.

## 4. Cost model
- Spread/commission cost per replayed entry: use realized outcome r_multiples
  (they already embed real fills) — no modeled cost adjustment needed.
- Additional candidate-gate cost: rejected-but-would-have-been-profitable signals
  are opportunity cost, reported separately (NOT added to expectancy).

## 5. Decision rules (same machinery as Phase 1)
- Walk-forward 70/15/15 per strategy×regime cell; BH-FDR q=0.10 across cells;
  effect ≥0.10R mean-expectancy OOS; p<0.05 paired bootstrap.
- PROMOTE_RECOMMENDATION requires BOTH: OOS expectancy improvement AND max DD
  not worse than baseline in the same window.
- KILL-SWITCHES: (a) any cell N<100 → cell excluded, decision on remaining cells
  only; (b) candidate gate increases NO-TRADE share >15pp without expectancy
  gain → auto-HOLD; (c) UNLINKED ratio >20% in the replay window → abort (linkage
  unreliable).

## 6. Pre-execution checklist (hard requirements)
- [ ] Sufficiency gate PASS (≥300 linked execution outcomes/strategy)
- [ ] Phase 1 completed or explicitly skipped with operator approval
- [ ] Regime cells enumerated with counts (from signal_feature_snapshots regime field)
- [ ] Replay harness review (quant_validator role, read-only mode) before any run

## 7. Amendment log
(empty — frozen as of 2026-09-17)