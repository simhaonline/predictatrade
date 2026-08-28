-- LEGACY DUPLICATE PREFIX 018: shares its numeric prefix with another migration file. Tolerated legacy collision (see database/migrations/MIGRATION_ORDER.md); DO NOT rename (rename risks re-applying applied schema). The CI guard scripts/check_migrations.sh blocks NEW duplicate prefixes only.
-- Migration 018: Slippage tracking and capital protection events
-- Stores execution slippage, swap costs, and daily capital protection decisions

-- ─── Slippage events table ───
CREATE TABLE IF NOT EXISTS trading.slippage_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    broker_ticket   VARCHAR(255),
    symbol          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    direction       VARCHAR(10) NOT NULL,
    requested_price NUMERIC(18,8) NOT NULL,
    filled_price    NUMERIC(18,8) NOT NULL,
    slippage_points NUMERIC(10,4) NOT NULL,    -- slippage in points
    slippage_cost   NUMERIC(18,8) NOT NULL DEFAULT 0, -- slippage cost in account currency
    volume          NUMERIC(18,4) NOT NULL,
    spread_at_fill  NUMERIC(10,4),            -- spread at time of fill
    is_rollover     BOOLEAN NOT NULL DEFAULT false, -- was this during rollover/swap time?
    swap_charged    NUMERIC(18,8) DEFAULT 0,   -- swap charge for this trade
    strategy        VARCHAR(50),
    signal_id       VARCHAR(100),
    account_id      VARCHAR(100),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_slippage_events_time ON trading.slippage_events(created_at DESC);
CREATE INDEX idx_slippage_events_ticket ON trading.slippage_events(broker_ticket);
CREATE INDEX idx_slippage_events_rollover ON trading.slippage_events(is_rollover) WHERE is_rollover = true;

-- ─── Capital protection events table ───
CREATE TABLE IF NOT EXISTS trading.capital_protection_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type      VARCHAR(50) NOT NULL,     -- DAILY_LOSS_WARNING, DAILY_LOSS_LIMIT_HIT, EMERGENCY_CLOSE, TRADING_BLOCKED
    account_id      VARCHAR(100),
    symbol          VARCHAR(20) DEFAULT 'XAUUSD',
    account_balance NUMERIC(18,8),            -- current account balance
    daily_pnl       NUMERIC(18,8),             -- daily P&L
    daily_pnl_pct   NUMERIC(10,4),             -- daily P&L as percentage of balance
    max_loss_pct    NUMERIC(5,2) NOT NULL DEFAULT 5.0,  -- configured max loss percentage
    equity          NUMERIC(18,8),             -- current equity
    open_positions  INT,                       -- number of open positions
    action_taken    VARCHAR(50),              -- BLOCKED_NEW_TRADES, CLOSED_ALL_POSITIONS, WARNED
    metadata        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_capital_prot_time ON trading.capital_protection_events(created_at DESC);
CREATE INDEX idx_capital_prot_type ON trading.capital_protection_events(event_type);
CREATE INDEX idx_capital_prot_account ON trading.capital_protection_events(account_id);

-- ─── Record migration ───
INSERT INTO schema_migrations (version, description, applied_at)
VALUES ('018', 'Slippage tracking and capital protection events', now())
ON CONFLICT DO NOTHING;
