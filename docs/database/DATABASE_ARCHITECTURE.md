# Database Architecture
## v1.27.0 — 04 September 2026

### Stack
- PostgreSQL 17 + TimescaleDB (hypertables) — `timescale/timescaledb-ha:pg17`
- pgvector (AI embeddings) · pgcrypto (PII encryption at rest)
- Exact-decimal money: `NUMERIC(18,8)` / `NUMERIC(10,4)` everywhere; `decimal.js` in the control plane
- Valkey 8 = hot/cache only — **never** sole durable financial or trading truth

### Schemas (16 application schemas, 210+ tables — verified live)

| Schema | Purpose | Key tables |
|--------|---------|------------|
| `iam` | Users, roles, memberships, sessions, API creds | users, roles, memberships, sessions, login_events, api_credentials, consent_records |
| `licensing` | Licenses, devices, MT accounts | licenses, devices, license_devices, device_activations, device_credentials, mt_accounts, mt_connections, entitlement_leases, license_events, client_releases |
| `billing` | Subscriptions, invoices, payments | subscriptions, invoices, payments, coupons, subscription_events |
| `finance` | Commissions, payouts, ledger | commission_ledger, commission_rules, commission_caps, payouts, ledger_entries, affiliate_wallets |
| `control` | Plans + commercial flags + operations | plans, commercial_feature_flags, platform_operations |
| `referral` | Affiliate tree + risk | affiliate_profiles, affiliate_risk_flags, referral_* |
| `trading` | Signals, execution, results, backtests | signals, trade_results, execution_commands, positions, exit_profiles, blocked_signals, broker_execution_profiles, cross_market_* (12), backtest_* (7), bar_processing_log |
| `market` | Time-series + macro | candles (hypertable), candles_m5/h1_agg, ticks, cot_* (8), economic_events, data_capabilities, data_provenance_log |
| `calibration` | Model registry + predictions | calibration_profiles, calibration_reports, model_versions |
| `ptb` / `ai` | PTB intelligence + feature flags + ML | ptb_feature_flags, ai.* model artifacts |
| `compliance` | GDPR + client telemetry | gdpr_operations (088), consent-driven client_event_log |
| `audit` | Audit events + migration history | audit_events, client_events, migration_history |
| `research` | Feature parity / quant studies | feature_parity_runs, parity artifacts |
| `live_preview` | Anonymous 5-minute preview funnel | anonymous_trials, funnel_stats |
| `support` | Tickets | support.* |
| `system` | Framework tables | roles/permissions seeds |

Entity relationships for the core domains: **[DB_ERD.md](DB_ERD.md)** (mermaid `erDiagram`).

### Hypertables & retention

| Hypertable | Chunk interval | Retention |
|---|---|---|
| `market.candles` | 1 day | 081_market_candles_retention (per-timeframe policies) |
| `audit.client_event_log` | 1 day | 064_audit_retention_and_logging |
| `market.cot_raw_reports` | 7 days | COT ingestion lifecycle (011) |

### Migration discipline (65 files, unique prefixes)

- Forward-only SQL under `database/migrations/001…095`; `audit.migration_history` is reconciled
  to disk by `scripts/check_migrations.sh` (also enforced in CI).
- The 7 duplicate-prefix pairs were renumbered to `089–095` (v1.17.2) — never rewrite applied
  history; `MIGRATION_ORDER.md` is the canonical sequence.
- Dual mechanism removed: `initdb.d` no longer auto-runs migrations (DB-5); `scripts/migrate.sh`
  is the only runner.

**v1.17.4 — payments honesty (USDT-only era):**
- `billing.payments.status` values now include `UNDERPAID` (IPN amount
  verification failed — subscription NOT activated; audit row in
  `audit.audit_events`).
- `billing.payment_events` = IPN dedupe/replay ledger keyed
  `(provider, provider_event_id)` with `signature_verified`.
- Access tokens embed `permissions[]` (role_permissions ⋈ memberships ⋈
  permissions) — @RequirePermissions guards rely on this.

### Recent schema changes

**v1.27.0 — account-type detection (migs 133/134, applied 2026-09-04):**
- `licensing.account_types` — per-login detected classification
  (`account_login BIGINT`, `detected_type`, `detection_timestamp`,
  `confirmation_count`, `is_verified`, `strategy_override`); 12 baseline
  `licensing.strategy_parameters` rows (cent lot ÷100, ECN commission ×2
  round-trip, STP +2pt slippage, Islamic swap-zero, demo tag).
- `licensing.edge_device_state.account_type` + `.account_type_verified`,
  `licensing.devices.account_type` — heartbeat-persisted EA detection
  (edge-poll v1.27 fail-open update).
- Detection truth chain: EA account-type detector (INLINED in every
  .mq5/.mq4 — no external files; lazy per-login cache,
  Islamic 3× rollover confirm, fail-safe Standard) → INIT/ACCOUNT_INFO/
  EXECUTION_ACK/heartbeat payloads → engine `SnapshotAccount.AccountType` →
  `edge_device_state` → dashboards/fan-out. Spec corrections documented in
  the .mqh header (no SYMBOL_COMMISSION_TICK/_LOT in MQL5 — deal-history
  scan; ACCOUNT_CURRENCY_DIGITS not standard — min-lot + balance heuristic).

**v1.17.3 — runtime-probe fixes:**
- `billing.subscriptions` INSERT explicit `$5::text` cast — PG17 strict parameter-type inference
  rejected the mixed use of `billing_interval` as column value and `CASE` operand (500 on
  create); subscription creation verified end-to-end.

**v1.17.2 — carryovers:**
- Renumbered migrations 089–095 + `migration_history` reconciliation.
- `088_gdpr_erasure_retention` → `compliance.gdpr_operations`; GDPR erase/anonymize in control.
- `trading.trade_results` populated exclusively by real broker fills (EA `TRADE_RESULT`
  enrichment, never synthetic); Trading Reports read only this table.
- Agent connection state bridged into `compliance.agent_status` by the Go engine
  (id, version, connected, last_seen, license_status).
- `072_backtest_artifact_payload` — equity/metrics JSONB artifacts per run.
- `082_elite_marnie_fib_strategy` — MARNIE_FIB strategy registration (ELITE).
- `085_set_plan_risk_caps` — `per_trade_risk_pct` / loss caps on plans (engine `LicenseRiskCaps`).

### Data model invariants

1. **Money:** `NUMERIC` only; ledger mutations are append-only; corrections are compensating
   rows (`finance.ledger_entries.revokes`).
2. **Time:** `TIMESTAMPTZ` everywhere; UTC internal truth; broker time = display conversion.
3. **Referential:** FKs enforced at DB level; soft-delete via `deleted_at`; GDPR hard erasure
   replaces rows with `deleted_<uuid>@anonymized.local` tombstones (verified in `iam.users`).
4. **Indices:** btree on all FKs; GIN on jsonb query paths (`signals.reason_codes`,
   `data_capabilities`); hypertable time+space partition keys on `candles(time, symbol, timeframe)`.
5. **Backups:** `pg_dump --format=custom` nightly + off-host copy; `scripts/backup/restore_test.sh`
   validates row counts and latest timestamps (`backup_metadata` audit trail).