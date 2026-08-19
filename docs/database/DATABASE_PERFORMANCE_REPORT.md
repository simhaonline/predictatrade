# Predict-A-Trade — Database Performance Report

**Generated:** 2026-08-18  
**All measurements from live database.**

## Environment

| Component | Version |
|-----------|---------|
| PostgreSQL | 17.11 (Debian) |
| pgvector | 0.8.6 |
| TimescaleDB | NOT INSTALLED |
| PgBouncer | Configured (transaction pool, max 200 clients, 20 pool) |
| Connection Pool (app) | pg Pool: max=20, idle=30s, timeout=5s |

## Database Size

| Metric | Value |
|--------|-------|
| Total DB size | ~50 MB |
| Largest table | market.ticks (143K rows) |
| Second largest | trading.signals (592 rows) |
| Third largest | market.candles (205 rows) |

## Largest Tables (by row count)

| Table | Rows |
|-------|------|
| market.ticks | 143,728 |
| trading.signals | 592 |
| market.candles | 205 |
| iam.role_permissions | 70 |
| control.plan_entitlements | 55 |

## Index Analysis

### New indexes added (Migration 010)
- `idx_signals_symbol_strategy_created` — signal history queries
- `idx_risk_decisions_gate_time` — risk decision audit
- `idx_audit_events_actor_time` — admin audit search
- `idx_audit_events_resource_time` — resource-based audit lookup
- `idx_login_events_user_time` — login history
- `idx_ticks_symbol_time` — latest tick lookup for dashboard
- `idx_licenses_user_status` — license validation
- `idx_subscriptions_user_status` — subscription status check
- `idx_commission_ledger_affiliate` — commission history
- `idx_candidates_*` (4 indexes) — candidate history queries
- `idx_rejections_*` (3 indexes) — rejection history
- `idx_security_events_*` (4 indexes) — security event investigation
- `idx_vector_embeddings_cosine` — HNSW vector similarity search

### Check Constraints Added
- Candle OHLC validation (5 constraints)
- Tick bid/ask validation (3 constraints)
- Signal entry price, confidence range, expiry validation (3 constraints)
- Commission non-negative (1 constraint)
- Position lot size positive (1 constraint)

### Immutability Triggers
- `audit.audit_events`: BEFORE UPDATE/DELETE → reject
- `referral.commission_ledger`: BEFORE UPDATE/DELETE → reject

## Query Performance

### Critical query paths verified:
- Latest candle by symbol/timeframe: Uses composite PK `(time, symbol, timeframe, source)` — index scan
- Latest tick by symbol: New `idx_ticks_symbol_time` — index scan
- Signal history by strategy: New `idx_signals_symbol_strategy_created` — index scan
- License validation: New `idx_licenses_user_status` with partial index — index scan
- Commission ledger: New `idx_commission_ledger_affiliate` — index scan

## Data Integrity Check Results

| Check | Status | Count |
|-------|--------|-------|
| Duplicate candles | PASS | 0 |
| Impossible OHLC | PASS | 0 |
| Negative spread | PASS | 0 |
| Negative commission | PASS | 0 |

## Autovacuum

Default PostgreSQL 17 autovacuum settings are active. With current data volume (~144K rows), no per-table autovacuum tuning is needed. Monitor when tick table exceeds 10M rows.

## Disk Growth Forecast

Based on current ingestion rate (~144K ticks, ~205 candles, ~592 signals):

| Period | Estimated Ticks | Estimated Size |
|--------|----------------|----------------|
| 30 days | ~4.3M | ~1.5 GB |
| 90 days | ~13M | ~4.5 GB |
| 1 year | ~53M | ~18 GB |
| 3 years | ~159M | ~54 GB |

With TimescaleDB compression (typically 8-20x for time-series), compressed storage would be ~2-7 GB for 3 years of ticks.

## Remaining Recommendations

1. **Install TimescaleDB** in production Docker image for hypertable partitioning, compression, and continuous aggregates
2. **Configure WAL archiving** for PITR in production
3. **Off-host backup storage** — current backups are on the same server
4. **pg_stat_statements** — enable for slow query identification
5. **Per-table autovacuum** tuning when tick table exceeds 10M rows
