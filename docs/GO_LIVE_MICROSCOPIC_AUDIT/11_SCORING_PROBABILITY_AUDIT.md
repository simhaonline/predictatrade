# 11 — Scoring / Probability / Confidence Forensic Audit

## Scoring equation (reconstructed from `pkg/strategy/scorer.go` + evidence.go)

- Per-strategy feature contributions (evidence pillars) summed with configured weights; dominance rules; conflict penalty (+10 while RANGE regime); per-regime threshold tables select min-score for CONFIRMED vs RESEARCH (`regime_thresholds.go:73-88`); caps bound RawScore to [0,100].
- Long/Short scores computed separately; directional winner drives Direction. NO-TRADE rows legitimately carry RawScore=0.
- Dead remnants `phi/computeFinalScore` (`strategies.go:158-170,283-291`) — NOT in live path.

DB reconciliation sample (SELL_CANDIDATE ULTRA_SCALPING): RawScore 39.67 < CONFIRMED threshold ⇒ RESEARCH grade — internally consistent. NO-TRADE rows score 0 with reason codes. **No fabricated scores found server-side** (frontend fabrication tracked in 24).

## Probability — what it actually is

`CalibratedProbability = sigmoid(a·z + b)` where z = normalized score distance, via `internal/calibration/consumer.go`. **The only models ever loaded are hardcoded seed sigmoids** stamped:

```
Status="VALIDATED" SampleSize=100 WilsonLower=0.45 Brier=0.21   ← FABRICATED metadata
```

No isotonic regression exists anywhere; no DB/loader path replaces seeds (`SetModel` callers: seeder + tests). Therefore:
- PROB is a **fixed monotone transform of the score**, not an empirical calibrated probability.
- Every NO-TRADE row persists prob≈0.450166 regardless of market state (DB-verified across rows) — proving non-informativeness of the transform outside scored candidates.
- SOW violation: raw score masquerades as validated calibration. Marketing/UI must label as UNCALIBRATED heuristic until real calibration on realized outcomes exists (no prediction_outcomes writer wired → calibration loop closed nowhere).

## Confidence

Signal Grade (RESEARCH/CONFIRMED/etc.) derives from score vs thresholds + gate outcomes — deterministic label, not a probability; frontend renders "Pending" until calibrated which is honest-ish but still shows the fake-calibrated number after.

## Verdicts

| Item | Status |
|---|---|
| Score reproducible/deterministic | VERIFIED (code+DB consistency) |
| Score independently recomputed end-to-end | UNVERIFIED (needs golden fixture §98) |
| Probability = calibrated | **FALSE — fabricated VALIDATED metadata (P0 F-006)** |
| Confidence meaningful | PARTIAL (deterministic grade) |
