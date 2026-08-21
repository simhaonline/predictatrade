-- Migration 020: Composite indexes for fast candle queries + Valkey cache support
--
-- Problem: Bootstrap and chart queries on market.candles had ~2 second planning
-- time because TimescaleDB couldn't do efficient chunk exclusion without a
-- composite (symbol, timeframe, time) index.
--
-- Fix: Add composite indexes that allow TimescaleDB to:
-- 1. Quickly locate the right chunks (time-range exclusion)
-- 2. Filter by symbol + timeframe within those chunks
-- 3. Return results in DESC order without sorting
--
-- Architecture: PostgreSQL/TimescaleDB (durable) + Valkey (hot cache)
--   - Writes: Ticks/candles → PostgreSQL/TimescaleDB (durable)
--   - Hot reads: Snapshots/state/signals → Valkey (sub-ms, 5-10s TTL)
--   - Candle reads: Valkey cache first (60s TTL) → PostgreSQL fallback
--   - Bootstrap: Valkey cache first (5min TTL) → PostgreSQL fallback

-- Composite index for bootstrap queries (symbol + timeframe + time DESC)
CREATE INDEX IF NOT EXISTS candles_symbol_tf_time_idx
  ON market.candles (symbol, timeframe, time DESC);

-- Timeframe + time index for chart data queries
CREATE INDEX IF NOT EXISTS candles_tf_time_idx
  ON market.candles (timeframe, time DESC);

-- Analyze table for updated query planner statistics
ANALYZE market.candles;

-- Verify indexes
SELECT indexname, indexdef
FROM pg_indexes
WHERE tablename = 'candles'
ORDER BY indexname;
