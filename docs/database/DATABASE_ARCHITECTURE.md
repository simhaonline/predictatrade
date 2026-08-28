# Database Architecture
## v1.17.2 — 28 August 2026

### Stack
- PostgreSQL 17 + TimescaleDB (hypertables)
- pgvector (AI embeddings)
- pgcrypto (encryption)

### Schemas
| Schema | Purpose | Key Tables |
|--------|---------|------------|
| iam | Users, roles, sessions, devices | users, roles, sessions, devices |
| billing | Subscriptions, plans, licenses | subscriptions, plans, licenses |
| finance | Commissions, payouts, ledger | ledger_entries, payouts |
| trading | Signals, orders, trade results | signals, orders, positions, trade_results, strategies |
| market | Candles (hypertable), COT, ticks | candles, cot_data, ticks |
| calibration | Model versions, predictions | model_versions, predictions |
| ptb | PTB intelligence | synthesis, performance |
| compliance | Audit events, client telemetry | client_event_log, audit_events |
| backtest | Backtesting results | backtest_runs, backtest_results |

### Recent Schema Changes (v1.17.x)

**Migration renumbering + reconciliation (v1.17.2):** The 7 duplicate-prefix migration pairs were renumbered to unique sequences 089–095 and `audit.migration_history` reconciled to match disk (65 files). `scripts/check_migrations.sh` now passes (no duplicate prefixes, history == disk). See `database/migrations/MIGRATION_ORDER.md`.

**GDPR erasure/retention (v1.17.2):** Migration `088_gdpr_erasure_retention.sql` adds `compliance.gdpr_operations`; the control plane exposes admin-only erase/anonymize/retention endpoints via `compliance/gdpr.service.ts`.

**trade_results table:** New table for real executed-trade metrics (P&L, R:R realized, entry/exit prices, broker ticket IDs). Populated by backfill from broker telemetry and live execution. Trading Reports dashboard queries this table exclusively — no estimated/derived values are substituted for real fills.

**Agent connection state bridging:** The Go engine bridges live agent WebSocket heartbeat status into `compliance.agent_status` for unified agent monitoring from the admin dashboard. Fields: agent_id, version, connected, last_seen, license_status.

**Migration 085:** `085_set_plan_risk_caps.sql` — sets per-plan risk caps for seed capital protection (5% daily loss limit enforcement per plan tier).

### Migrations
- 65 migrations applied (001-095), all with unique sequence prefixes
- Located in `database/migrations/` (canonical order in `MIGRATION_ORDER.md`)
- Run via `./scripts/migrate.sh up` (single source of truth — `initdb.d` auto-run removed)
- All migrations are idempotent (IF NOT EXISTS guards)
- `scripts/check_migrations.sh` is the CI guard: fails on duplicate prefixes or history-vs-disk drift
- `audit.migration_history` reconciled to match disk (65 = 65)

### Hypertables
- market.candles: 1-hour chunks, TimescaleDB compression
- Retention policy: 3 years on market.candles (v1.16.0) ✅

### Money Types
- All financial columns use `NUMERIC(18,8)` — no float/double anywhere
- Ledger entries: double-entry with RESERVED → SETTLED state machine
- Trade P&L stored as `NUMERIC(18,8)` in trading.trade_results
