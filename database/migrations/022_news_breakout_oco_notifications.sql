-- Migration 022: News Breakout, OCO Groups, and Notification Delivery
-- Operator-authorized additive migration (v1.9.0)
-- All new tables are additive — no existing tables are modified.

-- Economic calendar events (durable normalized event history)
CREATE TABLE IF NOT EXISTS trading.economic_events (
    id              SERIAL PRIMARY KEY,
    event_id        TEXT NOT NULL UNIQUE,
    provider        TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    event_name      TEXT NOT NULL,
    country         TEXT NOT NULL DEFAULT '',
    currency        TEXT NOT NULL DEFAULT '',
    impact          TEXT NOT NULL DEFAULT 'NONE',
    scheduled_at_utc TIMESTAMPTZ NOT NULL,
    actual          TEXT DEFAULT '',
    forecast        TEXT DEFAULT '',
    previous        TEXT DEFAULT '',
    event_category  TEXT DEFAULT '',
    source_timestamp TIMESTAMPTZ,
    received_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    revision        INTEGER NOT NULL DEFAULT 0,
    metadata        JSONB DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_economic_events_scheduled_at ON trading.economic_events (scheduled_at_utc);
CREATE INDEX IF NOT EXISTS idx_economic_events_provider_event_id ON trading.economic_events (provider, provider_event_id);
CREATE INDEX IF NOT EXISTS idx_economic_events_currency_impact ON trading.economic_events (currency, impact);

-- News provider sync status (health tracking)
CREATE TABLE IF NOT EXISTS trading.news_provider_health (
    id              SERIAL PRIMARY KEY,
    provider_name   TEXT NOT NULL UNIQUE,
    healthy         BOOLEAN NOT NULL DEFAULT FALSE,
    last_successful_sync TIMESTAMPTZ,
    last_error      TEXT DEFAULT '',
    event_count     INTEGER NOT NULL DEFAULT 0,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- News risk decisions (audit trail)
CREATE TABLE IF NOT EXISTS trading.news_risk_decisions (
    id              SERIAL PRIMARY KEY,
    risk_level      TEXT NOT NULL,
    reason_code     TEXT NOT NULL,
    evidence        TEXT DEFAULT '',
    next_event_id   TEXT DEFAULT '',
    computed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_news_risk_decisions_computed_at ON trading.news_risk_decisions (computed_at);

-- News breakout plans (pending order plans)
CREATE TABLE IF NOT EXISTS trading.breakout_plans (
    plan_id         TEXT PRIMARY KEY,
    event_id        TEXT NOT NULL,
    symbol          TEXT NOT NULL DEFAULT 'XAUUSD',
    strategy        TEXT NOT NULL DEFAULT 'NEWS_BREAKOUT',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    event_time      TIMESTAMPTZ NOT NULL,
    buy_stop_entry  NUMERIC(18,5) NOT NULL,
    sell_stop_entry NUMERIC(18,5) NOT NULL,
    buy_stop_sl     NUMERIC(18,5) NOT NULL,
    sell_stop_sl    NUMERIC(18,5) NOT NULL,
    buy_stop_tp     NUMERIC(18,5) NOT NULL,
    sell_stop_tp    NUMERIC(18,5) NOT NULL,
    volume          NUMERIC(18,2) NOT NULL,
    risk_pct        NUMERIC(5,2) NOT NULL,
    expiry          TIMESTAMPTZ NOT NULL,
    oco_group_id    TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'CREATED',
    rejection_reason TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_breakout_plans_event_id ON trading.breakout_plans (event_id);
CREATE INDEX IF NOT EXISTS idx_breakout_plans_oco_group_id ON trading.breakout_plans (oco_group_id);
CREATE INDEX IF NOT EXISTS idx_breakout_plans_status ON trading.breakout_plans (status);
CREATE INDEX IF NOT EXISTS idx_breakout_plans_event_time ON trading.breakout_plans (event_time);

-- OCO groups (durable one-cancels-other correlation)
CREATE TABLE IF NOT EXISTS trading.oco_groups (
    group_id            TEXT PRIMARY KEY,
    breakout_plan_id    TEXT NOT NULL,
    buy_order_id        TEXT NOT NULL,
    sell_order_id       TEXT NOT NULL,
    broker_buy_order_id TEXT DEFAULT '',
    broker_sell_order_id TEXT DEFAULT '',
    state               TEXT NOT NULL DEFAULT 'CREATED',
    winner              TEXT DEFAULT '',
    cancelled_side      TEXT DEFAULT '',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    triggered_at        TIMESTAMPTZ,
    cancel_requested_at TIMESTAMPTZ,
    cancel_confirmed_at TIMESTAMPTZ,
    reconciliation_state TEXT DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_oco_groups_breakout_plan_id ON trading.oco_groups (breakout_plan_id);
CREATE INDEX IF NOT EXISTS idx_oco_groups_state ON trading.oco_groups (state);
CREATE INDEX IF NOT EXISTS idx_oco_groups_broker_buy_order_id ON trading.oco_groups (broker_buy_order_id);
CREATE INDEX IF NOT EXISTS idx_oco_groups_broker_sell_order_id ON trading.oco_groups (broker_sell_order_id);

-- Notification deliveries (audit trail for all channels)
CREATE TABLE IF NOT EXISTS trading.notification_deliveries (
    id              SERIAL PRIMARY KEY,
    notification_id TEXT NOT NULL UNIQUE,
    event_type      TEXT NOT NULL,
    severity        TEXT NOT NULL DEFAULT 'INFO',
    user_account    TEXT DEFAULT '',
    trade_id        TEXT DEFAULT '',
    signal_id       TEXT DEFAULT '',
    order_id        TEXT DEFAULT '',
    title           TEXT NOT NULL,
    message         TEXT NOT NULL,
    structured_payload JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    delivery_channel TEXT NOT NULL,
    delivery_status  TEXT NOT NULL DEFAULT 'PENDING',
    attempt_count   INTEGER NOT NULL DEFAULT 0,
    last_error      TEXT DEFAULT '',
    delivered_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_notification_deliveries_notification_id ON trading.notification_deliveries (notification_id);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_event_type ON trading.notification_deliveries (event_type);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_status ON trading.notification_deliveries (delivery_status);
CREATE INDEX IF NOT EXISTS idx_notification_deliveries_created_at ON trading.notification_deliveries (created_at);

-- Add OCO group reference to existing signals table (additive)
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS oco_group_id TEXT DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS breakout_plan_id TEXT DEFAULT '';
