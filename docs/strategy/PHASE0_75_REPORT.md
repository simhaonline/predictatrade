# Phase 0.75 Report — Observability Completion + Calibration Readiness
Date: 2026-09-17 · prompt.md Phase 0.75 · Status: COMPLETE (A, B, C, D) · Commits: `7d65bba`, `21cb924`, `bb1ea7a` (+ this report)

## Task A — RawValue wiring (COMPLETE, deployed)

- **27 call sites wired** across StandardScalping, UltraScalping, StandardSwing, TrendSwing —
  every indicator-bearing evidence row now carries the REAL read (EMA prices, ADX, RSI,
  OsMA, StochMain, CCI, Bollinger band touched, Ichimoku Tenkan/Kijun, Fib 0.618, SAR,
  VWAP, MTF score, candle range, pivot level). Boolean/structural facts (BOS, CHoCH,
  sweeps, OB/FVG, regime, streaks) deliberately keep zero RawValue — documented in
  `docs/strategy/RAWVALUE_COVERAGE.md`.
- TDD: RED (test detected OSMA zero-RawValue before wiring) → GREEN. Per-strategy
  equality asserts RawValue == live state field; warmup-omission test (RSI=0 → NO row —
  absence ≠ fabricated 0).
- **Live coverage (metrics pack §8, last 24h, 3,377 snapshots): RSI/ADX/ATR/CCI/EMA9/
  EMA21/PSAR = 100% non-zero each.** Target was >90% — exceeded.
- Before/after per pillar: before Phase 0.75 only TREND/VWAP/MACD/RSI/ADX carried reads
  (98.6% of all RawValues were zero because most pillars were boolean); after this phase
  the snapshot-level indicator coverage is 100% non-zero post-warmup on all measured reads.
- Endpoint↔snapshot parity: covered by the migration-149 round-trip tests (unchanged, green).

## Task B — Writer hardening (COMPLETE, deployed)

- **reasonMap + normalizeCloseReason** ported into the Go writer: lowercase legacy EA
  reasons map to the canonical enum (`sl→STOP`, `tp1→TP1`, `be→BE`, `friday→
  FRIDAY_FLATTEN`, …); unknown → MANUAL (recorded, never dropped). Real-DB test: 10
  mappings PASS.
- **VerifyOutcomeSchema** startup guard: checks every writer-expected column on
  `trading.predictions` + `trading.prediction_outcomes`, fails closed with the missing
  column named; wired into the engine connect path (drift ⇒ clear panic, no silent
  degradation). Live evidence: engine log `outcome-pipeline schema verified` at 14:24.
- Drift-detection test made NON-destructive after it wiped r_multiple on the shared DB
  (caught by the sufficiency-gate query; backfill re-run restored R data; lesson committed
  in `bb1ea7a`).
- Migration-150 lesson audited: `docs/strategy/MIGRATION_ANTIPATTERN_AUDIT.md` — 46
  historical `CREATE TABLE IF NOT EXISTS` sites documented, no history rewrite, rule for
  future migrations (ALTER-extend, never assume-create) + follow-up list.

## Task C — Data acceleration + non-fabricated sample strategy (COMPLETE)

- **Sufficiency gate live**: `scripts/sample_sufficiency_gate.sql` — per-strategy
  execution-channel N (LINKED only), time-to-300 estimates, overall BLOCKED/PASS verdict,
  shadow channel reported SEPARATELY and explicitly not counted toward the gate.
- **Progress dashboard query** extended in `scripts/phase0_5_metrics.sql` (§8 RawValue
  coverage; §1-7 writer health/coverage/readiness from Phase 0.5).
- No synthesis: nothing replays signals as live, shadow rows keep `SHADOW` label, no
  merging of channels in any query.

## Task D — Phase 1 protocol PRE-REGISTERED and frozen (COMPLETE)

- `docs/strategy/PHASE1_CALIBRATION_PROTOCOL.md` committed BEFORE any tuning: hypotheses
  H1-H6, primary/secondary metrics, walk-forward 70/15/15, min effect ≥0.10R + p<0.05
  (BH-FDR q=0.10), decision rules, shadow→canary→promote flag-gated rollout with
  kill-switches, anti-overfitting rules (≤12 parameters, cross-strategy validation,
  no shadow-as-execution), exit criteria, immutable amendment procedure.

## Current N per strategy per channel (live query result)

| Strategy | EXEC linked | gate | est. weeks to 300 | SHADOW resolved (separate) |
|---|---|---|---|---|
| STANDARD_SCALPING | 47 | BLOCKED | 0.3 | 1,336 (avg −0.13R) |
| ULTRA_SCALPING | 25 | BLOCKED | 1.0 | 652 (+0.39R) |
| STANDARD_SWING | 11 | BLOCKED | 2.6 | 469 (−0.25R) |
| MARNIE_FIB | 4 | BLOCKED | 10.6 | 3 |
| TREND_SWING | 2 | BLOCKED | 21.3 | 309 (−0.67R) |

Execution-channel outcomes flow only when client EAs trade. Shadow outcomes flow
independently (price-resolved) but stay labeled SHADOW. **Overall verdict: BLOCKED —
collection continues (fail-closed).**

## Phase 1 readiness statement

**Phase 1 is NOT ready to begin.** Gating N: ≥300 linked EXECUTION outcomes per strategy.
Current max: STANDARD_SCALPING 47 (est. ~2 days once TRADE_RESULTs flow at the observed
signal-emit rate — but emission requires attached client EAs). What is missing:
1. Client EAs attached (the single binding constraint — signals enqueue+expire today).
2. EA recompile for real MAE/MFE (currently hardcoded 0.0).
Until then the engine collects every real outcome that arrives; no tuning starts.

## Operator actions outstanding

| Action | Impact if unresolved | Owner | Status |
|---|---|---|---|
| Attach client EAs (exec devices) | No TRADE_RESULTs → N stays 0/day → Phase 1 blocked indefinitely | Operator | Since Sep 12 |
| EA recompile (MAE/MFE + PATCloudURLFallback removal) | MAE/MFE stay 0/0 → excursions not calibratable; risk-sizing features degrade | Operator | Pending |
| TwelveData 429 (free tier) | DXY fail-closed → crossmarket bias degraded at 429 windows | Operator (upgrade or UTC reset) | Reset UTC midnight |

## Candidate C schedulability

**Not yet.** Per prompt.md: needs (a) sufficient B sample (blocked, see gate) and (b)
A coverage live (✓ now). Re-evaluate when the sufficiency gate passes.

## Remaining risks & rollback

- Engine rollback: `docker compose build realtime` from commit `65b62fe` (Phase 0.5) if the
  hardening misbehaves; schema guard is fail-closed so a schema mismatch stops startup
  loudly rather than writing degraded rows.
- Legacy data limitations remain (documented): MAE/MFE 0/0 until EA recompile; UNLINKED
  legacy ratio decays as live writers stamp LINKED outcomes.
- Deadlock fix (44de001) regression test green; 40/40 packages pass; vet clean.
- No tuning performed; no live behavior changed (writers are additive, gates untouched).