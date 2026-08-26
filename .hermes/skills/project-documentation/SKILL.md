---
name: project-documentation
description: "Maintain README, ADRs, changelog, docs site."
---

# project-documentation

Use when updating PAT docs: README, ADRs, CHANGELOG.

## Docs
docs/INDEX.md, docs/CHANGELOG.md
docs/api/, docs/database/, docs/frontend/, docs/strategy/
docs/operations/, docs/guides/

## README Standards
Architecture diagram, current status, quick start, build/test commands per plane, known limitations.

## ADRs
Format: docs/adr/NNNN-title.md (Status, Context, Decision, Consequences)
Never skip ADRs for boundary changes.

## CHANGELOG
keepachangelog.com format, Unreleased updated per PR, breaking changes explicit.
