# Database Architecture
## v1.16.0 — 26 August 2026

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
| trading | Signals, orders, positions | signals, orders, positions |
| market | Candles (hypertable), COT, ticks | candles, cot_data, ticks |
| calibration | Model versions, predictions | model_versions, predictions |
| ptb | PTB intelligence | synthesis, performance |
| compliance | Audit events, client telemetry | client_event_log, audit_events |
| backtest | Backtesting results | backtest_runs, backtest_results |

### Migrations
- 30 migrations applied (001-030)
- Located in `database/migrations/`
- Run via `./scripts/migrate.sh up`

### Hypertables
- market.candles: 1-hour chunks, TimescaleDB compression
- Retention policy: 3 years on market.candles (v1.16.0) ✅

### Money Types
- All financial columns use `NUMERIC(18,8)` — no float/double anywhere
- Ledger entries: double-entry with RESERVED → SETTLED state machine
