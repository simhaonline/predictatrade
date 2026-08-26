# Database Architecture
## v1.16.0 — 26 August 2026

### Stack
- PostgreSQL 17 + TimescaleDB (hypertables)
- pgvector (AI embeddings)
- pgcrypto (encryption)

### Schemas
| Schema | Purpose |
|--------|---------|
| iam | Users, roles, sessions, devices |
| billing | Subscriptions, plans, licenses |
| finance | Commissions, payouts, ledger |
| trading | Signals, orders, positions |
| market | Candles (hypertable), COT, ticks |
| calibration | Model versions, predictions, outcomes |
| ptb | PTB intelligence tables |
| compliance | Audit events, client events |
| backtest | Backtesting results |

### Migrations: 30 applied (001-030)
### Hypertables: market.candles (1h chunks)
### Money Types: NUMERIC(18,8) — no float/double
