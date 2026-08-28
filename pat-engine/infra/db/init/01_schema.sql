-- pat-engine schema (TimescaleDB, database: pat_engine)
-- Idempotent: safe to re-run.

CREATE EXTENSION IF NOT EXISTS timescaledb;

-- Raw market bars (time-series). One row per ingested bar/tick.
CREATE TABLE IF NOT EXISTS bars (
    ts          timestamptz NOT NULL,
    symbol      text         NOT NULL,
    open        double precision,
    high        double precision,
    low         double precision,
    close       double precision,
    spread      double precision
);
SELECT create_hypertable('bars', 'ts', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_bars_symbol_ts ON bars (symbol, ts DESC);

-- Every signal decision (executable OR blocked) for full auditability.
CREATE TABLE IF NOT EXISTS signals (
    id           text         PRIMARY KEY,
    ts           timestamptz NOT NULL,
    symbol       text,
    strategy_id  text,
    direction    text,
    entry        double precision,
    sl           double precision,
    tp1          double precision,
    tp2          double precision,
    tp3          double precision,
    raw_score    double precision,
    grade        text,
    signal_class text,
    status       text,        -- EXECUTABLE | BLOCKED
    reasons      jsonb,       -- array of reason codes
    created_at   timestamptz DEFAULT now()
);
SELECT create_hypertable('signals', 'ts', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_signals_strategy_ts ON signals (strategy_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_signals_status ON signals (status);
