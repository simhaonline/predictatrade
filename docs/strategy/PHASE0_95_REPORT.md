# Phase 0.95 Report — Production Dress Rehearsal + Promotion-Path Proof + First-24h Readiness
Date: 2026-09-17 · prompt.md Phase 0.95 · Status: COMPLETE (A flagged, B/C/D/E/F complete)
Commits: this phase (151 migration + upsert + boundary tests + synthetic branches + playbook + audit)

## Task A — Real MT5 dress rehearsal: BLOCKED (stated plainly, not simulated)

**Demo MT5 is NOT available on the origin host** — Linux server, no wine/MT5
runtime, and installing MT5 server-side contradicts the operator's architecture
(terminals are managed manually on his Windows machines). Per prompt.md the
boundary is NOT simulated. What engineering delivered instead:
- `docs/strategy/DRESS_REHEARSAL.md`: availability verdict + the complete
  operator-executed rehearsal kit (20-signal procedure, per-signal server
  verification query, latency-measurement queries, failure-log template,
  PASS/FAIL criteria: 20/20 with zero UNLINKED and non-zero MAE/MFE).
- Everything server-side is pre-verified: schema guard, writers, alerts, drills
  (Phase 0.9), plus Task B's boundary coverage below.
- **Verdict: FAIL-BY-BLOCKER — the MT5↔server boundary remains unproven on real
  hardware until the operator runs the rehearsal.** This is the outstanding
  operator-side item; owner: operator.

## Task B — MT5-boundary failure-mode coverage (software-in-the-loop, real DB)

Matrix (mode × tested × result) in `docs/strategy/MT5_BOUNDARY_FAILURE_MODES.md`.
Headline discovery: **F1 — repeated TRADE_RESULT double-counted** in
trading.trade_results (drill-proven: plain INSERT produced 2 rows; 5 legacy
dupes confirmed in data). Fixed fail-closed:
- Migration 151: dedupe (5 rows removed, earliest kept) + unique index
  `uq_trade_results_signal_ticket` + recorded in audit.migration_history.
- Handler INSERT → upsert (latest-wins) on the natural key.
- Tests: `TestBoundaryDuplicateTradeResult`, `TestBoundaryZeroLot` (real DB) PASS.
- F2-F14 verified by test or code-path audit (malformed→refuse, unknown
  reason→MANUAL, missing mae_r/mfe_r→accepted, zero-lot→records, clock-skew→
  server-authoritative fallback). No mode corrupts or double-counts.

## Task C — PROMOTE-path proof (synthetic, quarantined)

All four branches proven (matrix in `docs/strategy/PHASE1_HARNESS_BRANCHES.md`):

| Branch | Result |
|---|---|
| BLOCKED→REFUSED | live EXEC run: exit 2 ✓ |
| HOLD | effect +0.0116R, p 0.374 → HOLD ✓ |
| PROMOTE | effect +1.1856R, p 0.0001, BH-FDR survives → **PROMOTE_RECOMMENDATION** (never auto-promote) ✓ |
| ROLLBACK | effect −1.1927R, p 0.0001 → ROLLBACK_RECOMMENDATION ✓ |

Synthetic datasets in `docs/synthetic/` carry the enforced `SYNTHETIC_DRYRUN`
label (loader refuses unlabeled files); no real tables touched. Statistical
fix found during proof: the naive two-sample bootstrap has zero power for mean
differences — replaced with a one-sided **permutation test** (pooled label
shuffle, 10k×, seeded). Golden tests green on all branches.

## Task D — First-24h readiness

- `docs/strategy/FIRST_24H_AFTER_ATTACH.md` committed: T-0 checklist, T+0→1h
  milestone watch (enqueue→ACK→prediction→TRADE_RESULT→outcome, expected values
  and "wrong" signatures), T+1h→24h watch, kill-switch procedures with exact
  commands + rollback.
- `operator_status.sh` extended with FIRST-24H READINESS section: MAE/MFE
  non-zero % (1h), LINKED % (1h), per-strategy N (1h), time-to-300 per strategy,
  explicit "NO EAs ATTACHED — no false alarms expected" gate, writer/schema state.
- Verified live on the EA-off system: prints NO EAs ATTACHED, no false alarms.

## Task E — Spot audit (before/after in PHASE0_95_AUDIT.md)

No silent drift. One EXPLAINED drift: backfill report 227→222 total exactly
because migration 151 removed 5 duplicate rows (the F1 fix). Coverage 100%
(4,096 snapshots/24h), migrations 111 recorded with 151 present, all flags in
expected default state, independent cross-checks agree.

## Task F — Candidate C re-confirmed BLOCKED

Power analysis (α=0.05 one-sided, 80% power): 0.10R → n≈1,237/group;
0.30R → n≈138/group; current max N=47 → blocked by ~6-40×. Per-regime cells
further out. No amendment needed (design directionally correct); NOT executed.

## Current N per strategy per channel + time-to-300 (live)

| Strategy | EXEC linked | gate | est. weeks | SHADOW (separate) |
|---|---|---|---|---|
| STANDARD_SCALPING | 47 | BLOCKED | 1.7 | 1,336 |
| ULTRA_SCALPING | 25 | BLOCKED | 7.3 | 652 |
| STANDARD_SWING | 11 | BLOCKED | 17.8 | 469 |
| MARNIE_FIB | 4 | BLOCKED | 74.0 | 3 |
| TREND_SWING | 2 | BLOCKED | 149.0 | 309 |

## Phase 1 readiness statement

**Engineering is Phase-1-ready the moment EAs attach; three operator actions
remain the entire gap:** (1) EA patch 0001 recompile + deploy, (2) client EA
attach, (3) demo MT5 rehearsal per DRESS_REHEARSAL.md (proves the MT5 boundary
on real hardware). Everything downstream of attach — first-24h watch, gate
progression, harness execution, promotion recommendation — is prepared, tested,
and mechanical. No tuning starts until operator approval + gate PASS.

## Risks & rollback

- Risk: rehearsal may surface EA-side issues (expected on first hardware run) —
  the runbook's failure table captures and resolves each; the rehearsal is the
  last unproven hop.
- Rollback: engine rebuild from any prior commit; schema guard fails loudly on
  drift; migration 151 is additive-safe (unique index + upsert; no data loss).
- 40/40 packages pass; vet clean; deadlock regression green; no tuning; no live
  behavior change without approval.