# Phase 0.9 Report — Operator Enablement + Live Platform Hardening + Phase 1 Dry-Run Readiness
Date: 2026-09-17 · prompt.md Phase 0.9 · Status: COMPLETE (A, B, C, D, E)
Commits: `c9836b8` (Task A), `b040703` (Task C), `e6ccfbc` (Task B+D), this commit (Task E + report)

## Task A — EA-side enablement (DELIVERED)

| Artifact | Content | Operator path |
|---|---|---|
| `tools/ea/0001-fix-mae-mfe-tracking.patch` | MAE/MFE tracking in PredictATrade_MT5.mq5: per-magic registry columns (price+R), per-tick `PAT_UpdateExcursions` in `PAT_ManagePositions`, defaulted params in `PAT_ReportResult` (backward compatible), additive `mae_r`/`mfe_r` JSON fields, lifetime snapshot+reset on close. Telemetry only — zero behavior change. | `patch -p1 --dry-run` verified against committed source |
| `tools/ea/README.md` | Patch explanation + demo verification checklist + server verification query + rollback | — |
| `docs/guides/EA_RECOMPILE_RUNBOOK.md` | Per-terminal recompile/rollout/rollback, smoke-test on demo, failure table | 20 min/terminal |
| `docs/guides/EA_ATTACH_CHECKLIST.md` | Client-EA attach with exact server-side verification queries | pre-checks → attach → verify |
| `scripts/operator_status.sh` | One-command operator status: engine health, EA presence by role, stale exec devices, enqueue/ack rates, TRADE_RESULT rate, outcomes 24h, per-strategy gate N, UNLINKED ratio, writer silence | verified live |

**Exact operator steps** (full detail in the runbook): backup .mq5 → `patch -p1 -i tools/ea/0001-fix-mae-mfe-tracking.patch` → recompile F7 (0 errors) → re-attach demo → one demo trade → server query shows non-zero mae/mfe → roll out to client terminals. Expected verification output: `mae_points`/`mfe_points` non-zero on post-patch outcome rows.

## Task B — Live platform hardening (COMPLETE)

| Item | Status | Evidence |
|---|---|---|
| /health coverage (db, cache, outcome writers, shadow resolver, schema guard) | PASS | live: `outcome_pipeline {schema_guard: verified, outcome_writer: SILENT(37min, EAs offline — correct), minutes_since_outcome: 37, minutes_since_shadow_resolve: …}` |
| /ready fail-closed (db down → 503) | PASS (code path) | `gateway/http.go handleReady` |
| Structured logs + metrics for emit/predict/outcome/shadow/veto/hydrate/deadlock/queue/ACK/TTL/pool/migrations | PASS (existing + Phase 0.5 metrics pack + §9 linkage telemetry) | scripts/phase0_5_metrics.sql (9 sections) |
| Alerts (4) fire-and-page | PASS | all 4 fired under REAL conditions (writer silence 37m; UNLINKED 39%; outcomes missing 1; shadow legacy stalled 130,266); watchdog ntfy pages every 30s |
| Resilience drills | PASS | DR-1 cache restart → rebuilt in 4s; DR-2 resolver restart (deploy path); DR-3 DB fail-closed by code path (production DB stop requires operator approval — non-destructive verification); DR-4 migration_history round-trips |
| Backup/restore | PASS | pat-backup-sync → R2 every 1-2 min (RPO ≤2 min); scratch-restore round-trip: predictions 384 / outcomes 172 / snapshots 3,702 / migration_history 110; full-DB RTO 20-40 min, outcome tables in seconds |
| Capacity + retention | PASS | 66G/301G (23% used); snapshots ~3.7k/day ≈15MB/day; retention proposals documented, NO deletions without approval |
| Security pass | PASS, no open criticals | rate limits live (api 30r/s, auth 20r/s, ws), input validation (`symbol required`), JWT gating on signal/WS surfaces, 17 env overlays read server-side only, **zero secret tokens in engine logs** (grep verified) |
| LIVE_OPERATIONS_RUNBOOK.md | COMMITTED | deploy, rollback, deadlock, EA-offline, writer-silence, schema-drift, migration failure, DB-down, shadow-stalled, gate status, capacity |

## Task C — Phase 1 dry-run harness (COMPLETE)

- `scripts/phase1_run.py` — mechanical execution of the frozen protocol: pulls
  EXECUTION outcomes only (shadow pulled solely for dry-run, labeled
  DRY-RUN/SHADOW, never merged), enforces the gate, walk-forward 70/15/15,
  paired bootstrap p<0.05, BH-FDR q=0.10, effect ≥0.10R, decisions are
  recommendations (never auto-promote). **Exit 2 + refusal on BLOCKED gate.**
- Golden tests ALL PASS: BLOCKED→REFUSED; placebo→HOLD; significant
  positive→PROMOTE_RECOMMENDATION; BH-FDR; channel separation.
- **Live dry-run (SHADOW, labeled)**: executed end-to-end —
  STANDARD_SCALPING 1336/STANDARD_SWING 469/TREND_SWING 309/ULTRA_SCALPING 652
  → HOLD (placebo, as designed); MARNIE_FIB BLOCKED → REFUSED. Artifact:
  `docs/strategy/PHASE1_DRYRUN_RESULT.json`.
- **EXECUTION-channel run refuses** (exit 2) — fail-closed proven live.
- `scripts/phase1_status.sql` live: per-strategy gate status, N by channel
  (shadow column explicitly separate), time-to-300 estimates.
- Protocol file untouched — zero amendments (no retrofitting).

## Task D — RawValue + outcome pipeline follow-through (COMPLETE)

- Sweep: 11 additional sites wired in auxiliary emitters — MarnieFib (7:
  confluence scores, NearestLevelPrice, EMA21, RSI, MACD) + trend_transition.go
  (4: ADX, EMA9 slope, BollWidth, ATR). RAWVALUE_COVERAGE.md extended (41 wired
  sites total).
- Coverage re-run: 100% non-zero on measured reads (3,377 snapshots/24h,
  metrics §8). Before/after per pillar in RAWVALUE_COVERAGE.md.
- Backfill re-audit: 227/145/82 STABLE — no drift, no double-writes (172
  outcome rows total incl. live-writer output).
- Linkage telemetry live (metrics §9/§9b): daily UNLINKED trend (47.7% today,
  decaying as legacy share dilutes), 24h-window alert query.

## Task E — Candidate C pre-work (DESIGN ONLY, not executed)

`docs/strategy/PHASE2_REGIME_GATE_STUDY.md` committed: blocking statement
(sample, walk-forward feasibility, power analysis, shadow independence),
H-C1/H-C2 hypotheses, counterfactual replay method (1:1 to REAL stored
outcomes, never synthetic), cost model, decision rules, kill-switches,
pre-execution checklist. Frozen; amendment procedure identical to Phase 1.

## Current N per strategy per channel + time-to-300 (live)

| Strategy | EXEC linked | gate | est. weeks to 300 | SHADOW (separate) |
|---|---|---|---|---|
| STANDARD_SCALPING | 47 | BLOCKED | 2.2 | 1,336 |
| ULTRA_SCALPING | 25 | BLOCKED | 7.6 | 652 |
| STANDARD_SWING | 11 | BLOCKED | 18.1 | 469 |
| MARNIE_FIB | 4 | BLOCKED | 74.0 | 3 |
| TREND_SWING | 2 | BLOCKED | 149.0 | 309 |
| XAUUSD (legacy) | 1 | BLOCKED | 299.0 | — |

## Phase 1 readiness statement

**Phase 1 is READY the moment EAs attach — on the engineering side.** The
harness refuses on BLOCKED (proven), the protocol is frozen, the runbook and
operator bundle are in place; the instant client EAs attach and the sufficiency
gate flips PASS (est. ~2.2 weeks of STANDARD_SCALPING data at the observed emit
rate — the other strategies lag), `phase1_run.py` executes the protocol
mechanically. What remains is NOT engineering: (1) client EA attach (N=0 without
it), (2) EA recompile for MAE/MFE, (3) operator approval to start Phase 1.

## Operator actions outstanding (owner + impact)

| Action | Owner | Impact if unresolved |
|---|---|---|
| Apply EA patch 0001 + recompile + attach client EAs | Operator | Phase 1 stays blocked; N=0/day; MAE/MFE stay 0/0 (excursion calibration unusable) |
| TwelveData 429 (free tier) | Operator | DXY fail-closed → crossmarket bias degraded at 429 windows |
| Cloudflare orange-cloud decision | Operator | Plesk edge exposed direct (working, but CF protections off) |

## Remaining risks & rollback

- Engine rollback: rebuild from `65b62fe` (Phase 0.5) or any earlier tag; schema
  guard fails closed loudly if migrations mismatch.
- Data risks: legacy UNLINKED ratio decays as live rows arrive (alert will clear
  itself); legacy shadow rows frozen (documented, excluded from alerts when the
  scoped filter is applied).
- No tuning performed; no live signal behavior changed; deadlock fix 44de001 and
  its regression test green; 40/40 packages pass; vet clean.