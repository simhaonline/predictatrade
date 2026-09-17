# MT5-Boundary Failure Modes — Phase 0.95 Task B
Date: 2026-09-17 · Status: TESTED (software-in-the-loop, real DB) · Complement to DRESS_REHEARSAL.md

Boundary ownership: the EA→server boundary is real MT5 on operator hardware;
engineering owns the server half of every failure mode. Mode matrix below.
Each mode: expected (design) vs actual (verified by test or code-path audit).

| # | Mode | Expected (fail-closed design) | Actual (verified) | Test/proof |
|---|---|---|---|---|
| F1 | Repeated TRADE_RESULT (same position re-delivered: reconnect/EA restart) | Upsert on (signal_id, broker_ticket); exactly one row; no double-count | **DRILL-PROVEN**: plain INSERT double-counted (2 rows; also 5 legacy dupes found) → migration 151 (dedupe + unique index `uq_trade_results_signal_ticket`) + handler upsert → 1 row, latest wins | `TestBoundaryDuplicateTradeResult` PASS (real DB) |
| F2 | Out-of-order / delayed TRADE_RESULT | Same upsert path; newest payload wins | PASS — exit 2404 (second write) over 2405 (first) in one row | same test |
| F3 | Malformed TRADE_RESULT JSON | Handler parse-fail → warn log → return, no DB write | Verified by handler code path (`json.Unmarshal` error → return at main.go:2478/2486) — no partial rows possible | code-path audit |
| F4 | Unknown close_reason | Normalized to MANUAL (recorded, never dropped) | **Already proven Phase 0.75** (`normalizeCloseReason` fallback) | `TestOutcomeReasonMappingLowercase` PASS |
| F5 | Missing mae_r/mfe_r (old EA build) | Missing JSON fields decode to 0 — backward compatible, row accepted | Go json.Unmarshal leaves absent fields zero (handler struct + outcome writer default) — accepted by design | handler code path + Phase 0.5 round-trip |
| F6 | Empty signal_id (old EA) | Fallback UUID generated + logged; outcome goes UNLINKED path (fail-closed parent, never fabricated linkage) | Handler line 2500-2504; outcome writer resolves/preserves provenance via link_status | Phase 0.5 tests + handler audit |
| F7 | Zero-size fill | Row still records; pnl_points derivation guards lot>0 | PASS — zero-lot row persists, no div-by-zero | `TestBoundaryZeroLot` PASS (real DB) |
| F8 | Duplicate ACK (same delivery) | Queue `acked_at` update is idempotent per row (device_id, signal_id key) | Existing queue semantics; ACKs never feed calibration (delivery ≠ outcome) | Phase 0.5 design |
| F9 | Partial fills | Each partial close = own broker_ticket → distinct rows under (signal_id, ticket); outcome upsert keeps ONE outcome per signal with cumulative R | Consistent with natural key; EA emits per-deal | design + F1 test |
| F10 | Requote/failed order | No fill → no TRADE_RESULT; reconciliation gap-report flags the missing fill (existing reconciler) | Existing behavior | BE-6 reconciler |
| F11 | Connection drop mid-trade | Trade closes later; delayed result hits F1/F2 upsert path; no corruption | F1/F2 | same tests |
| F12 | EA restart with open positions | Stage persistence via GlobalVariables (PAT_SaveStage/PAT_LoadStage) + registry rebuild; close still reported via HistoryPoll | Existing (v1.27 stage persistence) | EA source audit |
| F13 | Terminal clock skew | openedAt parsed as RFC3339; server records its own created_at as truth for latency math; time_in_trade falls back to server-side derivation when EA clock is off | Handler lines 2538-2569 (server-authoritative fallback) | handler audit |
| F14 | Unknown extra JSON fields | Ignored by Go json.Unmarshal (forward compatible — mae_r/mfe_r additive) | Go stdlib behavior | Phase 0.5 ingest audit |

## Double-count audit (pre-existing data)
5 same-day duplicate pairs (Aug 25/26 legacy) found by drill; migration 151
deduplicated them (keep earliest) and made the natural key enforced going
forward. prediction_outcomes was never affected (upsert on signal_id since
Phase 0.5).

## Fail-closed confirmation
Every mode either (a) writes a clearly-labeled correct row, (b) upserts
latest-wins, or (c) refuses to write and logs. No mode corrupts predictions or
outcomes; none double-counts after migration 151.