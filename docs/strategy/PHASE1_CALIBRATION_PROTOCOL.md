# Phase 1 Calibration Protocol — PRE-REGISTERED (frozen)

Committed: 2026-09-17 (Phase 0.75, prompt.md Task D).
**This protocol is immutable for the duration of Phase 1.** Any deviation must
be documented as a pre-registered amendment (section 10) with justification
BEFORE the corresponding experiment runs. Retro-fitting is prohibited.

## 1. Hypotheses (tuning candidates, and why)

Baseline evidence: PHASE0_BASELINE_REPORT.md (145 trades, 0-29% win rates,
PF 0.00-1.16, 71% of trades in RANGE regime with −$97 net, RawValue 98.6%
zeros pre-Phase-0.75).

| # | Candidate | Current | Hypothesis |
|---|---|---|---|
| H1 | Family caps (TREND 0.35, others) | strategies.go familyMax | Caps over-weight correlated TREND evidence in RANGE regimes; recalibrate to regime-conditioned caps |
| H2 | Candidate/trade bars (per strategy+regime) | getRegimeThresholds | Thresholds were set pre-calibration data; evidence-based bars may differ |
| H3 | Quality grade ladder (70/55/15 + expectancy R) | quality_grade.go | Tier boundaries uncalibrated; Brier score will say whether probabilities discriminate |
| H4 | Gate thresholds (MinATR, MaxSpread*, RegimeDirection) | gates/*.go | Fail-closed gates may be over/under-restrictive vs realized outcomes |
| H5 | Astro sizing multiplier | ASTRO_SIZING flag | Astro sizing multiplier vs realized R (only if ASTRO_SIZING was live during sample) |
| H6 | Crossmarket scaling (DriverMomentum overlays) | crossmarket engine | Overlay weight vs realized outcome lift, execution channel only |

Non-candidates (frozen): strategy identity logic, evidence weights inside
family caps beyond H1 scope, gate ORDER, fail-closed semantics, the deadlock
fix, delivery pipeline.

## 2. Metrics

Primary (decision metrics):
- Expectancy per R (mean r_multiple, execution channel)
- Profit factor (gross win R / gross loss R)
- Max drawdown (equity curve from r_multiples, fixed 1R stake)
- Sharpe / Sortino (per-trade returns, annualized by trades/day observed)

Secondary (guardrail metrics):
- Win rate, Brier score, calibration curve (predicted calibrated_probability
  vs realized win), precision/recall per tier, NO-TRADE share, veto mix.

Channel labels (mandatory): every metric is reported as
`[EXEC]` execution-channel or `[SHADOW]` shadow-channel. Mixed numbers are
prohibited anywhere (reports, dashboards, commits).

## 3. Method

- Walk-forward with strict splits: train → validation → OOS. 70/15/15 within
  each walk window; rolling windows; OOS never touches parameter selection.
- Minimum N per strategy for ANY tuning experiment: **300 linked execution
  outcomes** (the sufficiency gate; shadow N reported separately and never
  counted toward the gate).
- Minimum effect size: ≥ 0.10R mean-expectancy improvement on OOS AND
  ≥ 5% PF improvement on OOS. Smaller effects are noise by definition here.
- Significance: paired bootstrap (10k resamples) on per-trade r_multiple
  delta; p < 0.05 required.
- Multiple-comparison correction: Benjamini-Hochberg FDR q=0.10 across all
  experiments executed in Phase 1.

## 4. Decision rules

- IMPROVE: OOS primary metrics pass section-3 thresholds AND secondary
  guardrails do not degrade (win rate within −2pp, max DD not worse).
- NOISE: significance not met → keep baseline; the tuned parameters are
  discarded (not "kept for later").
- REGRESS: OOS worse → revert immediately, log as failed experiment.
- ROLLBACK: any live canary breaching kill-switch (section 6) → automatic
  reversion to pre-tune parameters, incident-documented.

## 5. Flag-gated rollout plan

1. SHADOW: tuned parameters run in shadow evaluation only (labeled [SHADOW]);
   duration ≥ 2 weeks or ≥ 300 shadow outcomes, whichever is LATER.
2. CANARY: flag-gated (default OFF), single strategy, capped size; kill-switch
   criteria active; duration ≥ 1 week.
3. PROMOTE: flag flipped per strategy after canary review; staged 1 strategy
   at a time; each promotion documented.

Kill-switch criteria (any one triggers automatic reversion):
- Rolling 20-trade expectancy < −0.30R
- Drawdown exceeds 1.5× baseline max DD on the canary window
- Outcome writer UNLINKED ratio > 20% during canary (linkage broken)
- Any P0 safety/financial-integrity violation

## 5a. Shadow→execution transfer assumption (explicit)

Tuned parameters validated on the shadow channel transfer to the execution
channel ONLY as a hypothesis; canary must confirm on execution data before
promotion. Shadow and execution numbers stay separated in every artifact.

## 6. Anti-overfitting rules

- Parameter budget cap: ≤ 12 tunable parameters total across H1-H6 per
  Phase 1 walk; each parameter change must be justified by a hypothesis above.
- No per-strategy bespoke tuning without cross-strategy validation: a
  parameter change specific to one strategy must be tested against the other
  strategies' OOS to prove it is not sample-specific.
- No tuning on the shadow sample as if it were execution: shadow data informs
  hypotheses only, never gate/parameter acceptance.
- No re-running the same experiment with tweaked thresholds until it passes
  (p-hacking). Each experiment = one pre-registered entry in the experiment
  log; failures are logged as failures.

## 7. Sufficiency gate (blocks Phase 1 start)

Query: scripts/sample_sufficiency_gate.sql (Task C). Phase 1 tuning may begin
ONLY when, on the EXECUTION channel with link_status='LINKED':
- STANDARD_SCALPING ≥ 300, and each other strategy intended for tuning ≥ 300.
Current status is reported in the Phase 0.75 report (fail-closed: if below,
Phase 1 stays blocked — collection continues, no invention).

## 8. Exit criteria for Phase 1

All of:
1. Every executed experiment has a pre-registered entry, OOS result, and
   accept/reject decision recorded.
2. Accepted changes: flag-gated through shadow → canary → promote.
3. Post-promotion 30-day review: expectancy improvement holds OOS-style
   (fresh data after promotion), guardrails intact.
4. Calibration curve slope within [0.8, 1.2] of ideal on OOS (probabilities
   are meaningful).
5. Documentation updated: PHASE1_EXPERIMENT_LOG.md (append-only).

## 9. Immutability & amendment procedure

- This file is committed and frozen before any Phase 1 tuning begins.
- Amendments: append a dated entry to section 10 BEFORE running the affected
  experiment, with: what changes, why, what risk it introduces, and who
  approved it (operator approval required — prompt.md process).
- No silent edits: git history is the audit trail.

## 10. Amendment log

(empty — no amendments as of freeze)