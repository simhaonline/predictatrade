---
name: repo-audit
description: "Audit repo maps before major changes."
---

# repo-audit

Authority: `Predict-A-Trade_FINAL_SCOPE_OF_WORK_v1.0.0.md` and `AGENTS.md`.

## Workflow
1. Read AGENTS.md and the canonical SOW.
2. Map repo layout, services, packages, workers, binaries, frontend apps, deployment configs.
3. Trace market→feature→strategy→risk→signal→delivery→Windows/MT→broker and auth→subscription→referral→commission→payout flows.
4. Inventory DB/migrations (50+), APIs, events, caches, observability, exact build/test commands.
5. Identify active vs duplicate/dead/legacy code.
6. Classify REUSE/EXTEND/ADAPT/REPLACE/NEW/DEPRECATE. Produce change-impact + SOW traceability maps.

## Validate
No greenfield rewrite without evidence. Unknowns recorded, not guessed.

## Output Contract
Return SOW sections, files examined/changed, tests/checks + results, unresolved risks/blockers, next action.
