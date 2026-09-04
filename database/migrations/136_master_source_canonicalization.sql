-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 136: Master Node MT4/MT5 canonical data-source keys
--
-- Problem: Master Node MT4 and MT5 market data share market.ticks /
-- market.candles (single hypertable each), separated by the `source` column —
-- the correct design for parity joins, retention and backfills. But the
-- source strings had drifted into aliases (the real accuracy risk):
--     MT5_MASTER_NODE  1,799 tick rows  ← MT5 Master bar_event path (EA line 910)
--     M_MASTER             2 tick rows  ← MT5 Master duplicates (exact
--                                        MT5_MASTER twins within 5 s)
-- Same feed under three names silently fragments parity queries.
--
-- Decision (user directive, 2026-09-04): keep ONE shared table per kind, but
-- make per-terminal querying first-class:
--   1) backfill: alias sources → canonical MT4_MASTER / MT5_MASTER,
--      delete exact duplicate M_MASTER rows first,
--   2) triggers: normalize legacy alias source strings on write (backstop so
--      an un-recompiled EA cannot re-fragment the feed),
--   3) CHECK constraints: any '*MASTER*' source must be exactly
--      MT4_MASTER or MT5_MASTER (non-Master feeds unaffected),
--   4) views: market.v_ticks_master_mt4 / v_ticks_master_mt5 and
--      market.v_candles_master_mt4 / v_candles_master_mt5 — per-terminal
--      "separate table" ergonomics without split storage,
--   5) partial indexes for the hot per-terminal lookups,
--   6) history reconcile: 135 was applied out-of-band (idempotent) — record it
--      so `make db-migrate` history matches disk.
-- No columns are added, removed, or retyped. Read-safe for the engine: the
-- INSERT paths (realtime/internal/marketdata/persistence.go) already send
-- canonical strings; triggers only rewrite the legacy aliases.
-- ─────────────────────────────────────────────────────────────────────────────

-- ── 0. Decompression budget: the backfill UPDATE touches a compressed
--      hypertable (millions of tuples). Lift the per-DML decompression cap so
--      the UPDATE cannot abort mid-backfill on fresh deploys. Session-scoped.
SET timescaledb.max_tuples_decompressed_per_dml_transaction = 0;

-- ── 1a. Remove duplicate M_MASTER rows (exact MT5_MASTER twins ≤ 5 s apart) ──
DELETE FROM market.ticks t
WHERE t.source = 'M_MASTER'
  AND EXISTS (
        SELECT 1
        FROM market.ticks c
        WHERE c.source = 'MT5_MASTER'
          AND c.symbol = t.symbol
          AND c.bid = t.bid
          AND c.ask = t.ask
          AND abs(extract(epoch FROM (c.time - t.time))) < 5
      );

-- ── 1b. Guard against primary-key collisions before the backfill UPDATE ─────
DELETE FROM market.ticks t
USING market.ticks c
WHERE t.source = 'MT5_MASTER_NODE'
  AND c.source = 'MT5_MASTER'
  AND t.time = c.time
  AND t.symbol = c.symbol;

-- ── 1c. Backfill alias sources to canonical values ───────────────────────────
UPDATE market.ticks SET source = 'MT5_MASTER' WHERE source = 'MT5_MASTER_NODE';
UPDATE market.ticks SET source = 'MT5_MASTER' WHERE source = 'M_MASTER';

-- ── 2. Alias-normalizing triggers (BEFORE write, idempotent) ─────────────────
CREATE OR REPLACE FUNCTION market.normalize_master_source()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.source = 'MT5_MASTER_NODE' THEN
        NEW.source := 'MT5_MASTER';
    ELSIF NEW.source = 'M_MASTER' THEN
        NEW.source := 'MT5_MASTER';
    ELSIF NEW.source = 'MT4_MASTER_NODE' THEN
        NEW.source := 'MT4_MASTER';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_ticks_master_source ON market.ticks;
CREATE TRIGGER trg_ticks_master_source
    BEFORE INSERT OR UPDATE OF source ON market.ticks
    FOR EACH ROW EXECUTE FUNCTION market.normalize_master_source();

DROP TRIGGER IF EXISTS trg_candles_master_source ON market.candles;
CREATE TRIGGER trg_candles_master_source
    BEFORE INSERT OR UPDATE OF source ON market.candles
    FOR EACH ROW EXECUTE FUNCTION market.normalize_master_source();

DROP TRIGGER IF EXISTS trg_bar_log_master_source ON trading.bar_processing_log;
CREATE TRIGGER trg_bar_log_master_source
    BEFORE INSERT OR UPDATE OF source ON trading.bar_processing_log
    FOR EACH ROW EXECUTE FUNCTION market.normalize_master_source();

-- ── 3. CHECK constraints: Master feeds must use canonical keys only ──────────
-- Non-Master feeds (MT5, MT4, CSV, AGGREGATOR, research datasets) are
-- untouched; only '*MASTER*' strings are policed.
ALTER TABLE market.ticks
    DROP CONSTRAINT IF EXISTS chk_ticks_master_source;
ALTER TABLE market.ticks
    ADD CONSTRAINT chk_ticks_master_source
    CHECK (source NOT LIKE '%MASTER%' OR source IN ('MT4_MASTER', 'MT5_MASTER'));

ALTER TABLE market.candles
    DROP CONSTRAINT IF EXISTS chk_candles_master_source;
ALTER TABLE market.candles
    ADD CONSTRAINT chk_candles_master_source
    CHECK (source NOT LIKE '%MASTER%' OR source IN ('MT4_MASTER', 'MT5_MASTER'));

-- bar_processing_log (currently empty): canonical default + same guard.
ALTER TABLE trading.bar_processing_log
    ALTER COLUMN source SET DEFAULT 'MT5_MASTER';
ALTER TABLE trading.bar_processing_log
    DROP CONSTRAINT IF EXISTS chk_bar_log_master_source;
ALTER TABLE trading.bar_processing_log
    ADD CONSTRAINT chk_bar_log_master_source
    CHECK (source NOT LIKE '%MASTER%' OR source IN ('MT4_MASTER', 'MT5_MASTER'));

-- ── 4. Per-terminal views ("separate table" ergonomics, shared storage) ──────
CREATE OR REPLACE VIEW market.v_ticks_master_mt4 AS
SELECT time, symbol, bid, ask, mid, spread, tick_volume, source,
       source_timestamp, gateway_receipt_time, quality
FROM market.ticks
WHERE source = 'MT4_MASTER';

CREATE OR REPLACE VIEW market.v_ticks_master_mt5 AS
SELECT time, symbol, bid, ask, mid, spread, tick_volume, source,
       source_timestamp, gateway_receipt_time, quality
FROM market.ticks
WHERE source = 'MT5_MASTER';

CREATE OR REPLACE VIEW market.v_candles_master_mt4 AS
SELECT time, symbol, timeframe, open, high, low, close, volume, source,
       quality, alignment_profile, is_closed
FROM market.candles
WHERE source = 'MT4_MASTER';

CREATE OR REPLACE VIEW market.v_candles_master_mt5 AS
SELECT time, symbol, timeframe, open, high, low, close, volume, source,
       quality, alignment_profile, is_closed
FROM market.candles
WHERE source = 'MT5_MASTER';

-- ── 5. Partial indexes for the hot per-terminal lookups ──────────────────────
CREATE INDEX IF NOT EXISTS idx_ticks_mt5_master_symbol_time
    ON market.ticks (symbol, time DESC) WHERE source = 'MT5_MASTER';
CREATE INDEX IF NOT EXISTS idx_ticks_mt4_master_symbol_time
    ON market.ticks (symbol, time DESC) WHERE source = 'MT4_MASTER';
CREATE INDEX IF NOT EXISTS idx_candles_mt5_master_lookup
    ON market.candles (symbol, timeframe, time DESC) WHERE source = 'MT5_MASTER';
CREATE INDEX IF NOT EXISTS idx_candles_mt4_master_lookup
    ON market.candles (symbol, timeframe, time DESC) WHERE source = 'MT4_MASTER';

-- ── 6. History reconcile: 135 was applied out-of-band (idempotent re-run of
--      its INSERT ... ON CONFLICT DO NOTHING verified no-op) ──────────────────
INSERT INTO audit.migration_history (filename, status, started_at, completed_at, reconciled_note)
VALUES ('135_account_type_spec_conformance.sql', 'COMPLETED', now(), now(),
        'Reconciled 2026-09-04 by mig 136: applied out-of-band on 2026-09-04; idempotent re-run verified harmless.')
ON CONFLICT (filename) DO NOTHING;