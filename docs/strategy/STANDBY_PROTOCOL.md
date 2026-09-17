# STANDBY PROTOCOL — Engineering behavior while waiting for the operator
Frozen 2026-09-17 · Cannot be renegotiated without operator approval.

## Resume trigger (ALL FOUR must be true; any one missing = stay in standby)

- (a) EA patch applied and recompiled — verified by NON-ZERO MAE/MFE appearing in
  `trading.trade_results` (query in OPERATOR_HANDOFF.md Action 1).
- (b) Client EAs attached — verified by enqueue + ACK + TRADE_RESULT flow within
  the last 24h (query: `scripts/operator_status.sh`).
- (c) Execution-channel N ≥ 300 for at least one strategy — confirmed by
  `scripts/sample_sufficiency_gate.sql` showing gate=PASS for that strategy.
- (d) Operator has explicitly approved Phase 1 start in conversation.

## Until then, per prompt.md:

- Do NOT propose Phase 1. Do NOT propose Candidate C. Do NOT propose new engineering.
- No tuning. No new phases. No hardening passes. No refactors. No "while we're here".
- No live behavior change without explicit operator approval.
- Execution and shadow channels stay strictly separated in every number.

## Daily check (once per day, not more)

Run, and evaluate only:
```
bash scripts/operator_status.sh
curl -s http://localhost:13081/health | python3 -m json.tool | head -20
docker exec -i pat-postgres psql -U pat_admin -d predictatrade < scripts/sample_sufficiency_gate.sql
```
Watch: engine health, writer liveness, alert state, gate status, N per strategy,
RawValue coverage, deadlock detector (gate evals), migration state, disk headroom.

## Reporting rules

- Nothing changed → single-line heartbeat, nothing else.
- Something changed or an alert fired → report exactly that, with evidence, one paragraph max.
- Something degraded or broke → alert + fail-closed + minimal fix (test, deploy,
  report). Do not expand scope.
- Operator asks for something outside standby scope → state what it would change
  and wait for approval. Do not silently expand.

## Commitment

Preserve P0–P3, Phases 0 / 0.5 / 0.75 / 0.9 / 0.95. Deadlock fix (44de001) and its
regression test must keep passing. Fail closed, never fabricate. The next move
belongs to the operator.