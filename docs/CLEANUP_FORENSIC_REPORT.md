# Cleanup Forensic Report — 2026-08-23

## Executive Summary

| Metric | Value |
|--------|-------|
| Total tracked files | 871 |
| Total untracked files | 0 |
| Cleanup candidate files | 12 |
| Cleanup candidate directories | 4 |
| Estimated cleanup size | ~3.0 MB |
| Protected items | .git, .github, .codex, asset_kit, models, all source dirs |
| Generated/cache items | .ruff_cache, test-results, logs |
| Obsolete documentation | predictatrade-live-patched.html, .codex/config.toml.backup |
| Backup copies | .codex/config.toml.backup.20260817-100313 |

## Classification Table

| Path | Type | Size | Git State | Classification | Evidence | Action |
|------|------|------|-----------|---------------|----------|--------|
| .git/ | dir | — | tracked | KEEP — CRITICAL | Git repository metadata | NONE |
| .github/ | dir | — | tracked | KEEP — CI/CD | Contains ci.yml workflow | NONE |
| .codex/ | dir | — | tracked | KEEP — DEV | AGENTS.md references .codex/agents/ and config.toml | NONE |
| .codex/config.toml.backup.20260817-100313 | file | 1.5KB | tracked | MOVE — BACKUP COPY | Obsolete backup of config.toml, diff shows minor differences | MOVE |
| .ruff_cache/ | dir | 20KB | ignored | MOVE — GENERATED CACHE | Standard Ruff cache, already in .gitignore | MOVE |
| asset_kit/ | dir | — | tracked | KEEP — PRODUCTION ASSET | Referenced in docs/frontend/theme-system.md, MANIFEST.md | NONE |
| test-results/ | dir | — | ignored | MOVE — OLD TEST RESULT | Generated E2e test output, already in .gitignore | MOVE |
| audit/ | dir | — | tracked (AUDIT_REPORT.md) | KEEP — CURRENT DOCS | Contains audit report still referenced | NONE |
| logs/ | dir | 2.9MB | ignored | MOVE — OLD LOGS | 590 log files, already in .gitignore | MOVE |
| data/ | dir | — | untracked | KEEP — DATA | Historical data for research | NONE |
| models/ | dir | — | tracked | KEEP — PRODUCTION | ML model artifacts used by Go engine | NONE |
| predictatrade-live-patched.html | file | 90KB | untracked | MOVE — OBSOLETE | Not referenced anywhere, old patched HTML | MOVE |
| error.txt | file | 116B | untracked | KEEP — ignored | Already in .gitignore, runtime file | NONE |
| project-details.md | file | 9KB | tracked | KEEP — CURRENT DOCS | Project details document | NONE |
| project-summary.md | file | 7KB | tracked | KEEP — CURRENT DOCS | Project summary document | NONE |
| artifacts/ | dir | — | tracked | KEEP — EVIDENCE | Go-live evidence artifacts | NONE |

## Special Directory Decision Table

| Directory | Decision | Explanation |
|-----------|----------|-------------|
| .git/ | KEEP | Git repository metadata — protected, never touch |
| .github/ | KEEP | Contains CI/CD workflow (ci.yml) — production infrastructure |
| .codex/ | KEEP | AGENTS.md references .codex/agents/ and .codex/config.toml — dev workflow |
| .ruff_cache/ | MOVE | Standard Ruff cache, already in .gitignore, 20KB, rebuildable |
| asset_kit/ | KEEP | Brand assets referenced in docs and frontend, production-critical |
| test-results/ | MOVE | Generated E2E test output, already in .gitignore, rebuildable |
| logs/ | MOVE | 590 log files (2.9MB), already in .gitignore, runtime generated |
| audit/ | KEEP | Contains audit report, tracked in git, current documentation |
| data/ | KEEP | Historical data for research, untracked but useful |
| models/ | KEEP | ML model artifacts (ONNX), tracked, used by Go engine |
| artifacts/ | KEEP | Go-live evidence, tracked in git |

## Cleanup Candidates (Exact Paths)

1. `.ruff_cache/` — generated cache, 20KB, gitignored
2. `test-results/` — generated test output, gitignored
3. `logs/` — 590 runtime log files, 2.9MB, gitignored
4. `.codex/config.toml.backup.20260817-100313` — obsolete backup copy, tracked in git
5. `predictatrade-live-patched.html` — old patched HTML, not referenced, untracked

