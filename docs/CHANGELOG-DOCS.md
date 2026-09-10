# Documentation Changelog

Maintenance record for documentation restructuring. Historical records are retained deliberately; no sensitive values are reproduced here.

## 2026-09-10 — Full documentation audit + cleanup (second-pass baseline)

**Classification:** KEEP / UPDATE / MERGE / ARCHIVE / DELETE applied across ~60 active documents.

### Redacted (sensitive)
- `docs/reports/MACROSCOPIC_AUDIT_REPORT.md` — the live JWT_SECRET value was printed verbatim in the historical C-1 finding (re-leaking it after the compose fix). Values redacted; secret treated as COMPROMISED; `docs/operations/SECRET_ROTATION.md` created with the rotation procedure. **Rotation itself is an operator action in the next maintenance window.**

### ARCHIVED (moved to `docs/archive/2026-08-reports/`, inbound links updated atomically)
- MACROSCOPIC_AUDIT_REPORT_2026-08-28.md · MACROSCOPIC_AUDIT_REVISIT_2026-08-28.md · REMEDIATION_REPORT_2026-08-28.md · MACRO_AUDIT_2026-08-30.md · PROJECT_RESET_PLAN_2026-08-28_ARCHIVED.md
- Inbound references updated: `docs/INDEX.md`, `docs/_sidebar.md`, `docs/README.md`, `docs/operations/DEPLOYMENT_GUIDE.md`, `docs/operations/DOCKER_DEPLOYMENT.md`, root `README.md`.

### CORRECTED (stale references in active docs)
- **NATS / pkg/bus seam** — README, ARCHITECTURE, MANIFEST claimed an ingest bus ("NatsBus when NATS_URL set") that does not exist in the current tree (zero Go references; `realtime/pkg/bus` absent). Marked HISTORICAL/PLANNED-REMOVAL; `pat-nats` container flagged as removal candidate.
- **16-gate pipeline** — actual registry is 24 gate IDs (`types.go:242-268`). Updated in README, `docs/INDEX.md`, `docs/strategy/RISK_GATES.md` (with current order + nested capital caps + fail-closed PnL notes).
- **Strategy table** — README's Min-Score column (65/60/68/70) predated the v1.1 threshold revision; replaced with source-true trade thresholds (25/candidate 10), correct Decision TFs and expiries from `strategies.go` + `candidate_threshold.go`. IMLR row now carries its real ID (ARCANIST) and TFs (M5/M15).
- **"42 indicators"** — replaced with verified count: 76 feature fields across the registry (16 in the core indicator engine + structure/FVG/liquidity/session/ORB/VWAP families).
- **Version drift** — README said v1.19.0, docs said v1.29.0; engine is 1.24.2, EA v1.31. Header + docs index normalized (engine version is the canonical marker).
- **Services table** — added pat-control-b (HA ×2), mail-relay, watchdog, backup-sync, discord-bot; corrected `pat-nginx` → `xauusd-nginx`; marked pat-backtest (FastAPI) and pat-nats as removal candidates with rationale.
- **frontend/README.md** — stale ports (3000/8080) corrected to 13080/13081/13082 reality.
- **Broken links** — `docs/INDEX.md` `../../` links (pointed outside the repo) fixed to `../`.

### Governance (new)
- Canonical sources: README (entry point) → `docs/INDEX.md` (navigation) → topic docs. One topic, one document; duplicates must link, not fork.
- Never document a feature ahead of its code (the NATS/pkg/bus lesson).
- Migration count/version claims must cite the source (e.g. `ls database/migrations | wc -l`, `realtime/internal/version`).
- Audit reports are append-only historical records: archive by date folder, never delete; redact secrets but preserve findings.

## 2026-09-10 (earlier same day)
- `docs/audits/2026-09-10-full-production-audit.md` — 5-domain verified production audit.
- `docs/audits/2026-09-10-second-pass-validation-blueprint.md` — forensic re-validation + remediation blueprint.
- `docs/runbooks/telegram-gold-desk.md`, `docs/runbooks/whatsapp-market-desk.md`, `docs/runbooks/discord-bot.md`, `docs/runbooks/email-deliverability-gmail.md` — delivery + deliverability runbooks.
- `database/migrations/MIGRATION_ORDER.md` — 139/141/142 registered (inventory 102, numbered 001→142).