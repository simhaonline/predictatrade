# Phase 0.95 Audit — Spot Audit + Drift Check (Task E) + Power Analysis (Task F)
Date: 2026-09-17 · Verdict: NO SILENT DRIFT; one EXPLAINED drift (migration 151 dedupe)

| Item | Before (Phase 0.9) | After (this audit) | Verdict |
|---|---|---|---|
| DR-1 valkey restart → cache rebuild | cache ok in 4s | cache ok in 4s (4096-snapshot DB healthy) | PASS, unchanged |
| DB fail-closed code path | 503 + ADVISORY degrade | unchanged (verified design; prod DB stop needs approval) | PASS |
| Backup/restore round-trip | 384/172/3702/110 | re-run: same tables restore; migration_history now 111 (151 added, recorded) | PASS |
| RawValue coverage (measured reads) | 100% (3,377 snaps/24h) | 100% (4,096 snaps/24h; RSI/ADX/ATR/CCI/EMA9/EMA21/PSAR) | PASS, no regression |
| Backfill re-audit | 227/145/82 | **222/140/82** — the −5 = migration-151 dedupe of 5 confirmed duplicate TRADE_RESULT pairs (Aug 25/26 legacy, same signal_id+ticket, same-day) | EXPLAINED DRIFT, expected |
| Duplicate (signal_id, ticket) pairs | 5 | **0** (natural key now enforced) | FIXED |
| Migration history | 110 recorded | 111 recorded; 151 recorded; no partial states | PASS |
| Flags | — | ASTRO_SIZING unset (default OFF ✓), REFERENCE_TIER_LADDER unset (default ✓), ENGINE_OVERRIDE_SLTP default false ✓, ML_ENABLED/OLLAMA false ✓; operator-set OVERRIDE_LIVE_EDGE_NEGATIVE=true + TIER_RISK_CAP_MULT=3.0 unchanged | PASS — all in expected default state |
| Cross-check N/gate/coverage (independent queries) | — | metrics §2/§4 vs sample_sufficiency_gate.sql vs phase1_status.sql all agree (47/25/11/4/2/1; gate BLOCKED) | PASS |
| Test suite | 40/40 | 40/40 (incl. new boundary tests), vet clean, deadlock regression green | PASS |

## Task F — Candidate C power analysis re-run (design NOT amended)

Two-sample, one-sided α=0.05, power 80%: effect 0.10R needs n≈1,237/group
(total ≈2,475); 0.20R → ≈310/group; 0.30R → ≈138/group; 0.50R → ≈50/group.
Current EXEC N max 47 (STANDARD_SCALPING). **Candidate C remains blocked by
~6-40×** depending on strategy; per-regime cells (≥100 OOS each) are further
out. The pre-registered design in PHASE2_REGIME_GATE_STUDY.md is directionally
correct — no amendment filed (the only refinement: the 0.10R-effect gate at
n=300/strategy is under-powered on its own; the permutation test + BH-FDR in
the harness handles this honestly by refusing underpowered cells via the
effect-size + significance gates). Study design NOT executed.

## Operator actions (unchanged, sole Phase-1 blockers)

| Action | Owner | Status | Impact if unresolved |
|---|---|---|---|
| EA patch 0001 + recompile + deploy | Operator | PENDING | MAE/MFE stay 0/0; rehearsal (Task A) impossible |
| Client EA attach (exec role) | Operator | Since Sep 12 | N stays 0/day; Phase 1 blocked |
| Demo MT5 account for Task A rehearsal | Operator | Requested | MT5↔server boundary stays unproven on real hardware |