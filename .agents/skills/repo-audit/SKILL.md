---
name: repo-audit
description: Audit repository before major changes: map services/dependencies/data flows/tests/migrations/legacy code and classify REUSE/EXTEND/ADAPT/REPLACE/NEW/DEPRECATE.
---

# repo-audit

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and repository `AGENTS.md`.

## Workflow
1. Read AGENTS.md and the canonical SOW.
2. Map repo layout, services, packages, workers, binaries, frontend apps and deployment configs.
3. Trace market→feature→strategy→risk→signal→delivery→Windows/MT→broker and auth→subscription→referral→commission→payout flows.
4. Inventory DB/migrations, APIs, events, caches, observability and exact build/test commands.
5. Identify active vs duplicate/dead/legacy code.
6. Classify relevant components and create/update change-impact + SOW traceability maps.

## Validate
- No greenfield rewrite without evidence.
- Unknowns are recorded, not guessed.
- Active runtime and tests/migrations are identified.

## Output Contract
Return SOW sections addressed, files examined/changed, tests/checks + exact results, unresolved risks/blockers, and rollback/next action where applicable.
