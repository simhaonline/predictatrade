# Database Migration Order

## Canonical Apply Order

Migrations are applied lexicographically by filename. As of the 2026-08-28
renumber, all previously-duplicate sequence numbers have been resolved: the second
file of each former duplicate pair was renumbered to a unique sequence (089–095)
and the corresponding `audit.migration_history` rows were updated, so each migration
applies exactly once.

### Resolved duplicates (renumbered — no longer collide):
- 018 `018_regime_telemetry_shadow_signals.sql` + 089 `089_slippage_capital_protection.sql`
- 019 `019_percentage_sltp_config.sql` + 090 `090_signal_bug_closure_fields.sql`
- 020 `020_signal_truth_durability.sql` + 091 `091_valkey_candle_cache_indexes.sql`
- 028 `028_audit_execution_tables.sql` + 092 `092_bar_processing_metadata.sql`
- 062 `062_licensing_lifecycle.sql` + 093 `093_risk_config.sql`
- 071 `071_live_preview_anonymous_trials.sql` + 094 `094_marketing_consent_columns.sql`
- 080 `080_devil_liquidity.sql` + 095 `095_signal_quality_diagnostics.sql`

`scripts/check_migrations.sh` now passes (no duplicate prefixes, history matches disk).

### Future migrations:
- Use 3-digit zero-padded sequence numbers (028, 029, 030, ...)
- Before creating a new migration, run: `ls database/migrations/ | sort | tail -5`
- Never reuse an existing sequence number
- The `scripts/migrate.sh` script enforces uniqueness for new files
- `scripts/check_migrations.sh` is the CI guard (DB-6): it FAILS on any duplicate
  prefix (legacy AND new) and on any history-vs-disk filename drift.

## Reconciliation

`audit.migration_history` must equal the on-disk `database/migrations/*.sql` set.
Run `scripts/reconcile_migrations.sh --apply` to back-fill missing rows and
delete orphan rows. A rollback SQL is generated alongside it.

## Current inventory (as of v1.18.0)

100 unique migrations on disk, `001` → `139` (0xx–1xx sequences + this index
file). The early section below documents migrations 096–099; the recent
additions list (121+) covers the latest schema changes:

### Sequence gaps (intentional)

Prefixes **030–059** and **101–109** are absent on disk AND absent from
`audit.migration_history` — verified 2026-09-10. They were never used (the
001–029 run predates the SOW re-plan; 101–109 were skipped when numbering
jumped from 100 straight to 110). Do NOT reuse these numbers casually: run
`ls database/migrations/ | sort | tail -5` and take the next free number
(140+) for new migrations, per the rules below.

- `096_ai_providers.sql` — AI provider registry (ollama|openai|custom)
- `097_playbook_exit_profiles.sql` — playbook exit profiles
- `098_broker_account_types.sql` — broker account types
- `099_igs_institutional_gold_signal.sql` — IGS composites + AI research reports
  + versioned weights (seed row `1.0.0/shadow`) + compression/retention policies
- `121_backtest_runs_user_id.sql` — trading.backtest_runs.user_id + index (R1 per-user
  attribution; fixes HTTP 500 on /backtest/runs when the column was missing)
- `122_trusted_devices.sql` — iam.trusted_devices: hashed trusted-device tokens for the
  "remember this device" MFA bypass (30d, single-use rotation)
- `123_backtest_runs_subscription.sql` — trading.backtest_runs.subscription_id +
  plan_code/plan_name snapshot + index (R9 per-plan backtest/revenue attribution)
- `100_atten_elite_entitlement.sql` — ATEN strategy entitlement on ELITE
- `110_align_plans_to_master_spec.sql` — plan tiers to MASTER PROMPT (FREE 1/5, STD 2, PRO 4, ELITE 6)
- `111_user_onboarding_fields.sql` — signup onboarding columns
- `112_fix_plan_annual_prices_descriptions.sql` — annual pricing/description fixes
- `113_arcanist_elite_entitlement.sql` — ARCANIST on ELITE
- `114_arcanist_all_paid_plans.sql` — ARCANIST on all paid plans
- `115_device_id_text.sql` — devices.id text compatibility
- `116_add_mfa_recovery_codes.sql` — MFA recovery codes
- `117_edge_signal_queue.sql` — licensing.edge_signal_queue (EA-direct delivery)
- `118_drop_agent_architecture.sql` — Windows-agent WS architecture removal
- `119_device_role_column.sql` — devices.role (data|exec)
- `120_capital_tiers.sql` — capital-tier classification columns
- `125_backtest_jobs.sql` — backtest job queue
- `126_connectivity_alerts.sql` — system.connectivity_alerts
- `127_delivery_reconciliation.sql` — delivery reconciliation ledger
- `128_combined_tier_geometry.sql` — combined tier geometry (v1.25)
- `129_standard_scalping_rebuild.sql` — STANDARD_SCALPING rebuild
- `130_aten_undorm.sql` — ATEN un-dorm
- `131_trend_swing_plan_tier.sql` — TREND_SWING plan tier
- `132_fleet_entitlement_pro.sql` — fleet entitlement for PRO
- `133_account_type_tables.sql` — broker account type tables
- `134_account_type_columns.sql` — account type columns
- `135_account_type_spec_conformance.sql` — account type spec conformance
- `136_master_source_canonicalization.sql` — master data source canonicalization
- `137_candle_tf_mn_alias.sql` — candle TF MN alias
- `138_device_risk_events.sql` — device risk events
- `139_plan_entitlements_realignment.sql` — re-align control.plan_entitlements
  display rows to migration 110 (MASTER PROMPT spec: FREE 1 slot / 5 signals/day,
  STANDARD 2, PRO 4, ELITE 6) + restore ELITE api.access=true (regressed by 024).
  Fixes admin "Plans & Entitlements" showing stale slot caps contradicting the
  enforced plans-table columns.
- `124_backtest_runs_raw_output.sql` — trading.backtest_runs.raw_output: verbatim
  backtest-engine stdout per run (R9-verify audit trail; admin-only exposure)

Rollback: no `down` migrations exist by design; point-in-time recovery (PITR)
from the 6-hourly backup is the documented rollback path (see docs/operations).
