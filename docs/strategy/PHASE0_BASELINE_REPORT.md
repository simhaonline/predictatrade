# Phase 0 Baseline Report & Fine-Tuning Proposal
Date: 2026-09-17 · Engine v1.24.2 · prompt.md Phase-0 deliverable (no live behavior changes)

## 1. Data inventory (what evidence exists)

| Source | Window | Volume | Usable for |
|--------|--------|--------|-----------|
| trading.signals | Aug 18 – Sep 17 | 156,820 | volumes, veto/tier/grade distribution |
| trading.signals with trade_results join | Aug 25 – Sep 11 | 145 trades | realized PnL, win rate, expectancy |
| trading.trade_results | Aug 25 – Sep 11 | 227 | only realized-outcome sample |
| trading.prediction_outcomes | — | **0** | (empty — nothing ever written) |
| licensing.edge_signal_queue ACKED | — | 43,981 | delivery, NOT outcomes |
| feature snapshots | since P0 deploy | ~2,400 | indicator reads at signal time |

**Data sufficiency verdict:** insufficient for calibration. 145 realized trades
across 7 strategies (per-strategy n: 139/58/19/6/5) is far below any
statistically meaningful N for tier-threshold or weight tuning (need ≥ ~300
outcomes per strategy for a 95% CI on win rate). **No calibration or threshold
change is justifiable on current data.**

## 2. Baseline metrics (realized sample, 145 trades)

| Strategy | Trades | Win rate | Avg PnL | Profit factor | Total PnL |
|----------|-------:|---------:|--------:|--------------:|----------:|
| STANDARD_SCALPING | 139 | 27.3% | −1.22 | 0.40 | −169.12 |
| ULTRA_SCALPING | 58 | 29.3% | +0.16 | 1.16 | +9.09 |
| STANDARD_SWING | 19 | 21.1% | −1.09 | 0.55 | −20.72 |
| MARNIE_FIB | 6 | 0% | −0.93 | 0.00 | −5.58 |
| TREND_SWING | 5 | 0% | −5.47 | 0.00 | −27.33 |

Portfolio: max DD **$229.46** on a peak-to-trough range of −$6.57…−$236.03;
daily ann. Sharpe **−9.64**, Sortino **−6.86** (negative — the realized sample
loses). Close reasons: manual 150 (66% — discretionary intervention dominates
the sample), SL 58 (**0 wins on SL-hit**), TP1 9 (+4.97 avg), TP3 10.
MAE/MFE columns are sign-inverted/unreliable (win MAE 843 vs loss MAE 1076 —
do not trust; EA-side recording bug, see findings).

Gate veto counts (7d, gate_results): risk_oversize 6,585 · martingale_ban
2,229 · entitlement VETO+DEGRADED 3,996 · profitability 1,960 ·
regime_direction 1,292 VETO + 1,440 DEGRADED (deployed today only) ·
data_quality 1,081 · margin 770 · daily_loss 657. Reason codes on signals:
INSUFFICIENT_SCORE 25,184 · NT_CONFLICTING_DIRECTION 1,526 ·
SESSION_UNSUITABLE 660 · UNCLEAR_STRUCTURE 505 · REGIME_MISMATCH 263.

## 3. P0–P3 live verification (claims vs reality)

| Claim | Verified | Notes |
|-------|----------|-------|
| RawValue real reads | **PARTIAL** | 1.4% non-zero over 7d (only TREND/VWAP/DEVIL_LIQUIDITY wired); 16 pillars still ship 0 |
| Snapshot ↔ endpoint parity | **YES** (same builder) | endpoint = live state; snapshot = at-signal state (expected time difference) |
| Snapshots persisted | **YES** | ~2,400 rows; BUT stalled during the deadlock (see §4 #1) |
| 40/40 packages, vet clean | **YES** | re-run today |
| Flag states | REFERENCE_TIER_LADDER **OFF**, ASTRO_SIZING **OFF** (not in env) | defaults held |
| MAX_SL_USD=8 live | **YES** | post-deploy signals carry SL dist ≤ 8.00 exactly |

## 4. Top 5 accuracy problems (quantified)

**#1 — Engine-wide gate deadlock (CRITICAL, fixed this phase).**
EvaluateAll held the registry RLock across the gate loop; gates call
GetState (recursive RLock); hydrate loops queue writers → self-deadlock.
Live: strategy evaluation frozen 35+ min (12:27–13:02 UTC), 0 signals, 0
snapshots. Fixed via snapshot-then-evaluate + regression test
(`TestEvaluateAllNoDeadlockWithConcurrentWriters`). Confidence: high
(pprof 4-goroutine cycle captured).

**#2 — Outcome tracking is absent (data problem, blocks all calibration).**
43,981 ACKED deliveries → 227 trade_results → 0 prediction_outcomes; signals
carry no exit price. Cannot compute post-delivery win rate, expectancy, or
tier precision/recall. PnL impact: unmeasurable today. Fix = data collection:
wire EA TRADE_RESULT acks → trade_results → prediction_outcomes (signal_id
join exists). No calibration until N ≥ 300/strategy.

**#3 — RANGE-regime trades dominated the losing sample.** 103/145 (71%) of
executed trades were in RANGE regime: −$97 net (STANDARD_SCALPING −74).
The P1 regime_direction squeeze veto (RANGE ⇒ no trades) deployed today
would have blocked that entire cohort; historically that veto has zero
outcome evidence — the "avoided losers vs missed winners" question cannot be
answered without #2. Opportunity cost: unmeasurable; confidence: medium.

**#4 — RawValue exposure incomplete (98.6% zeros).** addEvidenceRaw was wired
only into TREND/VWAP/DEVIL_LIQUIDITY; MOMENTUM (RSI/MACD), SMC, CANDLE, MTF,
VOLATILITY, REGIME and astro/crossmarket pillars still ship 0. Accuracy cost:
evidence consumers (EA HUD, dashboards, snapshot audits) cannot verify
indicator reads. Fix: wire the same reads into the remaining call sites.

**#5 — SL/geometry vs outcome mismatch.** SL-hit closes: 0 wins / 58 losses,
avg −3.03; but 150/227 (66%) of trades closed MANUAL, meaning outcomes do
not reflect the strategy contract (TP1 +4.97 avg when it runs). Also MAE/MFE
sign convention inverted in trade_results (win MAE 843 > loss MAE 1076 —
recording bug). Risk-adjusted measurement impossible until EA records
honest exits + MAE/MFE orientation.

## 5. Top 3 fine-tuning candidates (proposals only — NOT implemented)

**Candidate A — Complete RawValue wiring (accuracy, zero risk).**
Hypothesis: exposing real reads (RSI/MACD/ADX/CLV/chop/squeeze/linR2…) in
MOMENTUM/SMC/CANDLE/MTF/VOLATILITY pillars makes evidence auditable and
enables downstream calibration once outcomes exist. Expected impact: 1.4% →
>90% non-zero RawValue; no behavior change (read-only fields). Test plan:
unit test per pillar (read ≠ 0 given non-zero source), snapshot parity test,
SQL assertion. Flag: none needed (pure observability).

**Candidate B — Outcome capture pipeline (data, prerequisite for all tuning).**
Hypothesis: EA TRADE_RESULT messages already flow (Option B ingest) — wiring
them into trade_results + prediction_outcomes gives the 300+/strategy sample
needed for ANY threshold/weight tuning. Expected impact: unlocks precision/
recall per tier, calibration curves, Brier scores, and regime-gate
effectiveness measurement (avoided losers vs missed winners). Test plan:
ingest handler → trade_results upsert (idempotent by broker ticket), outcome
backfill job, join coverage report. Flag: none (data plumbing).

**Candidate C — Regime-gate effectiveness study (after B).**
Hypothesis: the RANGE squeeze veto improves risk-adjusted return. Test plan:
shadow-compare — replay 90d signals with/without the RANGE veto (geometry
already snapshotted); measure avoided losers (RANGE cohort lost −$97 on 103
trades) vs missed winners (TP1-hit RANGE trades). Requires candidate B's
outcomes + walk-forward split. Flag: `REGIME_SQUEEZE_VETO` (default current
behavior) — promote only if OOS-positive with N ≥ 300 and significance
p < 0.05.

## 6. Guardrail compliance

- Phase 0 changed ONE behavior: the deadlock fix (44de001) — that is a
  correctness repair restoring intended behavior, not a tuning change, and
  was required to make the engine measurable at all. Regression-tested.
- REFERENCE_TIER_LADDER / ASTRO_SIZING remain OFF.
- No calibration/threshold change made (insufficient N — see §1).
- Rollback: single-commit revert per change; no data migrations beyond 149.