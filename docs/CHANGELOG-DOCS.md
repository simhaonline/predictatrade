# Documentation Changelog

Maintenance record for documentation restructuring. Historical records are retained deliberately; no sensitive values are reproduced here.

## 2026-09-18 — Day-2 session: loss-streak forensics, EA v1.32–v1.34, backup-pipeline resurrection

- `docs/operations/RESUME_TRIGGER_SESSION_REPORT_2026-09-18.md` — new: full
  day-2 forensics (slippage-guard root cause with hold-time evidence, ATEN
  armed-list bypass, backup-pipeline cascade, daily-loss halt timeline) and
  5 binding operational takeaways.
- **EA v1.34** (`mql/mt4|mt5/PredictATrade_MT{4,5}.mq5`) — slippage guard
  3→60 pts, per-strategy 5/10/20/30→60/60/80/100 (XAUUSD spread is 20–40 pts;
  old FX-scale values guaranteed instant spread-loss closes). Plus v1.32
  transport hardening and v1.33 AutoExecute=true + terminal self-diagnostics.
- **EdgeValidationGate** — un-armed + un-proven strategies now hard-veto
  (was soft-degraded → ATEN promoted past the armed list).
- **Backup pipeline resurrected** — archive dir permissions fixed, 238GB
  pre-checkpoint WAL reclaimed, fresh physical base backup, 6-hourly cron.
- **nginx per-IP rate limiting removed** (operator decision) — per-IP zones
  throttle the single Plesk edge IP, not clients; app-layer auth is the
  abuse boundary.
- `docs/OPERATOR_WINDOWS_RUNBOOK.md` — failure modes 9–11 added (stale
  binaries, DEVICE_LIMIT_EXCEEDED, JWT mismatch storm), AutoTrading step,
  v1.33/v1.34 behavior notes.
- `docs/strategy/PHASE1_CALIBRATION_PROTOCOL.md` — §7a subscription/
  entitlement invariants added.

## 2026-09-17 — Resume-trigger session: docs + dashboards updated

- `docs/operations/RESUME_TRIGGER_SESSION_REPORT_2026-09-17.md` — new: full
  bug-fix + production-readiness sweep record (7 fixes with verification
  evidence, plus the no-action-needed list and remaining go-live gates).
- `docs/OPERATOR_HANDOFF.md` — header updated to point at the session report;
  the 3 operator actions unchanged (still the blockers for Phase 1).
- `docs/strategy/STANDBY_PROTOCOL.md` — closed by operator resume trigger
  (historical record retained; the Phase-1 ≥300 gate stays binding).
- Admin **health page** (`frontend/src/app/(admin)/admin/health`) — new
  "Outcome Pipeline (writer + shadow resolver)" service row sourced from the
  Go engine's `/api/v1/system-health` (endpoint now includes the
  `outcome_pipeline` block: writer state, minutes since last outcome, minutes
  since last shadow resolve, schema guard). SILENT writer shown as DEGRADED
  with the note that it is expected while no client EAs are attached.
- `realtime/internal/gateway/http.go` — `/api/v1/system-health` now includes
  the same `outcome_pipeline` liveness block the public `/health` reports.

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