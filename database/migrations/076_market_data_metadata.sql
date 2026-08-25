-- Reproducible DDL for market.data_metadata.
-- VERIFIED: this table already exists in the live database, but was not present
-- in any prior migration. Without it, a fresh `migrate.sh` would not recreate the
-- table and the backtest data-range picker (getAvailableData) would fail on new
-- deployments. Created idempotently so it is a no-op on the running system.
CREATE TABLE IF NOT EXISTS market.data_metadata (
    timeframe    varchar(16) PRIMARY KEY,
    candle_count bigint      NOT NULL DEFAULT 0,
    min_date     date,
    max_date     date,
    source       varchar(64),
    updated_at   timestamptz NOT NULL DEFAULT now()
);
