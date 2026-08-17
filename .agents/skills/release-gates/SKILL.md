---
name: release-gates
description: Apply final promotion/release gates requiring test, security, quant, parity, execution, finance, Windows/MT, compliance and rollback evidence.
---

# release-gates

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Promotion
research → backtest → OOS → replay → paper → shadow → signals → limited execution. No self-promotion.

## Workflow
1. Collect build/lint/unit/integration/E2E, migrations/backup/restore, security/SBOM/signing evidence.
2. Require deterministic/golden/parity, exit-aware cost backtest, OOS/calibration/sample sufficiency.
3. Require gate latency/fail-closed, delivery reliability, broker qualification, ledger reconciliation, Windows/MT and observability evidence.
4. Verify compliance activation and performance-claim evidence.
5. Mark traceability PASS/FAIL/BLOCKED and issue GO only when all applicable P0 gates pass.

## Validate
Feature parity, exit-aware backtest, gate health, security, finance reconciliation and rollback readiness.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
