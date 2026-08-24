-- ─── Bar Processing Metadata (prompt.md Section 30) ───
-- Tracks which bars have been received and processed for signal generation.
-- Used for idempotency: a duplicate bar_closed event from the MT5 Master Node
-- or Windows Agent retransmission must NOT trigger a second strategy evaluation.

CREATE TABLE IF NOT EXISTS trading.bar_processing_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol VARCHAR(20) NOT NULL,
    timeframe VARCHAR(10) NOT NULL,
    bar_open_time TIMESTAMPTZ NOT NULL,
    bar_close_time TIMESTAMPTZ NOT NULL,
    source VARCHAR(50) NOT NULL DEFAULT 'MT5_MASTER_NODE',
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processing_status VARCHAR(20) NOT NULL DEFAULT 'RECEIVED',
    bar_event_id VARCHAR(200),
    sequence BIGINT,
    -- Idempotency: one row per unique bar
    UNIQUE(symbol, timeframe, bar_open_time)
);

CREATE INDEX IF NOT EXISTS idx_bar_processing_log_time
    ON trading.bar_processing_log(symbol, timeframe, bar_open_time DESC);

CREATE INDEX IF NOT EXISTS idx_bar_processing_log_status
    ON trading.bar_processing_log(processing_status)
    WHERE processing_status != 'PROCESSED';
