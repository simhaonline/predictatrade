-- Predict-A-Trade v1.0.0 — Migration 014
-- Advanced Risk, Adaptation, Hedging, ML/RL, Sentiment persistence
-- All tables ADDITIVE — no existing tables modified.

-- ============================================================
-- Recovery States — Loss Recovery / Capital Protection state machine
-- Per account+strategy isolation (SOW: state isolation)
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.recovery_states (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id              VARCHAR(100) NOT NULL,
    strategy_id             VARCHAR(50) NOT NULL,
    symbol                  VARCHAR(50) NOT NULL DEFAULT 'XAUUSD',

    state                   VARCHAR(20) NOT NULL DEFAULT 'NORMAL',
    -- NORMAL, RECOVERY, HALTED, DAILY_LIMIT

    consecutive_losses      INTEGER NOT NULL DEFAULT 0,
    daily_loss_count        INTEGER NOT NULL DEFAULT 0,
    daily_loss_percent      DECIMAL(8,4) NOT NULL DEFAULT 0,
    daily_pnl               DECIMAL(18,8) NOT NULL DEFAULT 0,
    starting_equity         DECIMAL(18,8),

    recovery_trades_taken   INTEGER NOT NULL DEFAULT 0,
    recovery_wins            INTEGER NOT NULL DEFAULT 0,

    cooldown_until          TIMESTAMPTZ,
    halt_until               TIMESTAMPTZ,
    halt_reason             TEXT,

    last_trade_at           TIMESTAMPTZ,
    last_loss_at            TIMESTAMPTZ,
    last_close_event_id     VARCHAR(200), -- dedup key for broker close events

    trading_day             DATE NOT NULL,

    config_snapshot         JSONB NOT NULL DEFAULT '{}',

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(account_id, strategy_id, symbol, trading_day)
);

CREATE INDEX IF NOT EXISTS idx_recovery_states_account_strategy ON trading.recovery_states(account_id, strategy_id);
CREATE INDEX IF NOT EXISTS idx_recovery_states_state ON trading.recovery_states(state);
CREATE INDEX IF NOT EXISTS idx_recovery_states_trading_day ON trading.recovery_states(trading_day);

-- ============================================================
-- Trade Results — Closed trade outcome persistence
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.trade_results (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id               UUID REFERENCES trading.signals(id) ON DELETE SET NULL,
    account_id              VARCHAR(100) NOT NULL,
    strategy_id             VARCHAR(50) NOT NULL,
    symbol                  VARCHAR(50) NOT NULL DEFAULT 'XAUUSD',
    direction               VARCHAR(10) NOT NULL,
    broker_ticket           VARCHAR(100),

    entry_price             DECIMAL(18,8),
    exit_price               DECIMAL(18,8),
    stop_loss               DECIMAL(18,8),
    take_profit              DECIMAL(18,8),

    pnl                     DECIMAL(18,8) NOT NULL DEFAULT 0,
    pnl_points               DECIMAL(18,8) NOT NULL DEFAULT 0,
    pnl_percent             DECIMAL(8,4) NOT NULL DEFAULT 0,

    lot_size                DECIMAL(10,2),
    is_win                  BOOLEAN NOT NULL DEFAULT false,
    is_loss                 BOOLEAN NOT NULL DEFAULT false,
    is_breakeven            BOOLEAN NOT NULL DEFAULT false,

    mae                     DECIMAL(18,8), -- maximum adverse excursion
    mfe                     DECIMAL(18,8), -- maximum favorable excursion
    time_in_trade_seconds   INTEGER,

    close_reason             VARCHAR(50), -- TP, SL, MANUAL, EA_CLOSE, TIMEOUT, PARTIAL_TP
    close_event_id           VARCHAR(200) UNIQUE, -- dedup key for broker close events

    recovery_mode           BOOLEAN NOT NULL DEFAULT false,
    adaptation_phase         VARCHAR(50),
    sentiment_score          DECIMAL(5,2),

    opened_at                TIMESTAMPTZ,
    closed_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    trading_day              DATE NOT NULL,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_trade_results_account ON trading.trade_results(account_id);
CREATE INDEX IF NOT EXISTS idx_trade_results_strategy ON trading.trade_results(strategy_id);
CREATE INDEX IF NOT EXISTS idx_trade_results_signal ON trading.trade_results(signal_id);
CREATE INDEX IF NOT EXISTS idx_trade_results_trading_day ON trading.trade_results(trading_day);
CREATE INDEX IF NOT EXISTS idx_trade_results_win ON trading.trade_results(is_win);
CREATE INDEX IF NOT EXISTS idx_trade_results_close_event ON trading.trade_results(close_event_id);

-- ============================================================
-- Blocked Signals — Audit of signals blocked by recovery/risk
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.blocked_signals (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id               UUID REFERENCES trading.signals(id) ON DELETE CASCADE,
    account_id              VARCHAR(100),
    strategy_id             VARCHAR(50) NOT NULL,
    symbol                  VARCHAR(50) NOT NULL DEFAULT 'XAUUSD',

    block_reason             VARCHAR(100) NOT NULL,
    -- RECOVERY_MODE, DAILY_LIMIT, HALT, COOLDOWN, LOW_CONFLUENCE_RECOVERY,
    -- LOW_QUALITY_RECOVERY, LOW_CONFIDENCE_RECOVERY, MAX_RECOVERY_TRADES

    recovery_state           VARCHAR(20),
    adaptation_phase         VARCHAR(50),
    blocked_at               TIMESTAMPTZ NOT NULL DEFAULT now(),

    context                  JSONB NOT NULL DEFAULT '{}',

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_blocked_signals_strategy ON trading.blocked_signals(strategy_id);
CREATE INDEX IF NOT EXISTS idx_blocked_signals_reason ON trading.blocked_signals(block_reason);
CREATE INDEX IF NOT EXISTS idx_blocked_signals_account ON trading.blocked_signals(account_id);

-- ============================================================
-- Adaptation History — Rule-based adaptation decisions
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.adaptation_history (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id              VARCHAR(100),
    strategy_id             VARCHAR(50),
    symbol                  VARCHAR(50) NOT NULL DEFAULT 'XAUUSD',

    market_phase             VARCHAR(50) NOT NULL,
    -- TRENDING, RANGING, HIGH_VOLATILITY, LOW_VOLATILITY, MANIPULATIVE, UNCERTAIN

    regime                   VARCHAR(50),
    volatility_state         VARCHAR(50),
    manipulation_index       DECIMAL(5,2),

    adjustments              JSONB NOT NULL DEFAULT '{}',
    -- stop_distance_multiplier, risk_multiplier, min_confluence, min_confidence,
    -- preferred_strategies, weight_adjustments

    reason                   TEXT,
    source                   VARCHAR(20) NOT NULL DEFAULT 'RULE_BASED',
    -- RULE_BASED, ML_BASED, FALLBACK

    timestamp                TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_adaptation_history_phase ON trading.adaptation_history(market_phase);
CREATE INDEX IF NOT EXISTS idx_adaptation_history_strategy ON trading.adaptation_history(strategy_id);
CREATE INDEX IF NOT EXISTS idx_adaptation_history_timestamp ON trading.adaptation_history(timestamp DESC);

-- ============================================================
-- Hedge Positions — Active hedge lifecycle tracking
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.hedge_positions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_trade_id        VARCHAR(200) NOT NULL,
    hedge_trade_id           VARCHAR(200) NOT NULL UNIQUE,
    account_id              VARCHAR(100) NOT NULL,
    strategy_id             VARCHAR(50) NOT NULL,
    symbol                  VARCHAR(50) NOT NULL DEFAULT 'XAUUSD',

    original_direction       VARCHAR(10) NOT NULL,
    hedge_direction          VARCHAR(10) NOT NULL,
    original_size            DECIMAL(10,2) NOT NULL,
    hedge_size               DECIMAL(10,2) NOT NULL,
    original_entry           DECIMAL(18,8),
    hedge_entry              DECIMAL(18,8),
    hedge_sl                 DECIMAL(18,8),
    hedge_tp                 DECIMAL(18,8),

    reason_opened            TEXT NOT NULL,
    reason_closed            TEXT,
    status                   VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    -- OPEN, CLOSED_TP, CLOSED_SL, CLOSED_MANUAL, CLOSED_AUTO, EXPIRED

    pnl                      DECIMAL(18,8),
    aggregate_exposure       DECIMAL(18,8),

    opened_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at                TIMESTAMPTZ,
    expires_at               TIMESTAMPTZ,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_hedge_positions_account ON trading.hedge_positions(account_id);
CREATE INDEX IF NOT EXISTS idx_hedge_positions_original ON trading.hedge_positions(original_trade_id);
CREATE INDEX IF NOT EXISTS idx_hedge_positions_status ON trading.hedge_positions(status);

-- ============================================================
-- Hedge History — Closed hedge audit trail
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.hedge_history (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    original_trade_id        VARCHAR(200) NOT NULL,
    hedge_trade_id           VARCHAR(200) NOT NULL,
    account_id              VARCHAR(100) NOT NULL,
    strategy_id             VARCHAR(50) NOT NULL,
    symbol                  VARCHAR(50) NOT NULL DEFAULT 'XAUUSD',

    original_direction       VARCHAR(10) NOT NULL,
    hedge_direction          VARCHAR(10) NOT NULL,
    original_size            DECIMAL(10,2),
    hedge_size               DECIMAL(10,2),
    original_entry           DECIMAL(18,8),
    hedge_entry              DECIMAL(18,8),
    hedge_sl                 DECIMAL(18,8),
    hedge_tp                 DECIMAL(18,8),

    reason_opened            TEXT,
    reason_closed            TEXT,
    status                   VARCHAR(20) NOT NULL,

    pnl                      DECIMAL(18,8),
    pnl_original             DECIMAL(18,8),
    net_pnl                  DECIMAL(18,8),

    opened_at                TIMESTAMPTZ,
    closed_at                TIMESTAMPTZ NOT NULL DEFAULT now(),
    duration_seconds         INTEGER,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_hedge_history_account ON trading.hedge_history(account_id);
CREATE INDEX IF NOT EXISTS idx_hedge_history_strategy ON trading.hedge_history(strategy_id);
CREATE INDEX IF NOT EXISTS idx_hedge_history_closed ON trading.hedge_history(closed_at DESC);

-- ============================================================
-- RL Training History — RL model training run records
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.rl_training_history (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_name              VARCHAR(100) NOT NULL,
    model_version           VARCHAR(50) NOT NULL,

    mode                    VARCHAR(20) NOT NULL DEFAULT 'disabled',
    -- disabled, shadow, filter_only, live_approved

    training_start           TIMESTAMPTZ,
    training_end             TIMESTAMPTZ,
    episodes                INTEGER NOT NULL DEFAULT 0,

    -- Validation metrics (out-of-sample)
    total_reward             DECIMAL(18,8),
    avg_reward               DECIMAL(18,8),
    max_drawdown             DECIMAL(8,4),
    sharpe_ratio             DECIMAL(8,4),
    sortino_ratio            DECIMAL(8,4),
    profit_factor            DECIMAL(8,4),
    win_rate                 DECIMAL(5,4),
    expectancy               DECIMAL(18,8),
    trade_count              INTEGER,
    avg_trade_duration       INTEGER,

    oos_start                TIMESTAMPTZ,
    oos_end                  TIMESTAMPTZ,
    walk_forward_folds       INTEGER,

    reward_config            JSONB NOT NULL DEFAULT '{}',
    feature_config            JSONB NOT NULL DEFAULT '{}',
    hyperparameters          JSONB NOT NULL DEFAULT '{}',

    status                   VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    -- PENDING, RUNNING, COMPLETED, FAILED, CANCELLED

    artifact_path            TEXT,
    checksum                  VARCHAR(64),

    error_message            TEXT,
    created_by                UUID,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at             TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_rl_training_model ON trading.rl_training_history(model_name, model_version);
CREATE INDEX IF NOT EXISTS idx_rl_training_status ON trading.rl_training_history(status);
CREATE INDEX IF NOT EXISTS idx_rl_training_mode ON trading.rl_training_history(mode);

-- ============================================================
-- Sentiment Snapshots — Cached real-time sentiment state
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.sentiment_snapshots (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol                  VARCHAR(50) NOT NULL DEFAULT 'XAUUSD',

    overall_score            DECIMAL(5,2) NOT NULL DEFAULT 0,  -- -100 to +100
    overall_confidence       DECIMAL(5,4) NOT NULL DEFAULT 0, -- 0 to 1
    category                 VARCHAR(50), -- BULLISH, BEARISH, NEUTRAL, MIXED

    item_count               INTEGER NOT NULL DEFAULT 0,
    source_count             INTEGER NOT NULL DEFAULT 0,

    provider_health          JSONB NOT NULL DEFAULT '{}',
    -- provider_name -> {status: OK|DEGRADED|ERROR, last_success: ts, error_count: n}

    last_successful_update   TIMESTAMPTZ,
    data_age_seconds         INTEGER,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE(symbol)
);

-- ============================================================
-- Sentiment Items — Individual sentiment data points with provenance
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.sentiment_items (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id              UUID REFERENCES trading.sentiment_snapshots(id) ON DELETE CASCADE,
    symbol                  VARCHAR(50) NOT NULL DEFAULT 'XAUUSD',

    source                   VARCHAR(100) NOT NULL, -- gdelt, reuters, fed, reddit, twitter_x, internal
    provider                 VARCHAR(100) NOT NULL,
    headline_id               VARCHAR(500), -- URL or unique identifier

    score                    DECIMAL(5,2) NOT NULL DEFAULT 0,  -- -100 to +100
    confidence               DECIMAL(5,4) NOT NULL DEFAULT 0, -- 0 to 1
    category                 VARCHAR(50), -- BULLISH, BEARISH, NEUTRAL, MIXED

    text_preview             TEXT,
    language                 VARCHAR(10) DEFAULT 'en',

    source_timestamp         TIMESTAMPTZ,
    fetched_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    age_seconds              INTEGER,

    created_at              TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sentiment_items_snapshot ON trading.sentiment_items(snapshot_id);
CREATE INDEX IF NOT EXISTS idx_sentiment_items_source ON trading.sentiment_items(source);
CREATE INDEX IF NOT EXISTS idx_sentiment_items_fetched ON trading.sentiment_items(fetched_at DESC);
CREATE INDEX IF NOT EXISTS idx_sentiment_items_category ON trading.sentiment_items(category);

-- Convert time-series tables to hypertables if TimescaleDB available
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        PERFORM create_hypertable('trading.trade_results', 'closed_at', if_not_exists => TRUE);
        PERFORM create_hypertable('trading.adaptation_history', 'timestamp', if_not_exists => TRUE);
        PERFORM create_hypertable('trading.sentiment_items', 'fetched_at', if_not_exists => TRUE);
    END IF;
EXCEPTION WHEN OTHERS THEN
    -- TimescaleDB not available — proceed as regular tables
END $$;

COMMENT ON TABLE trading.recovery_states IS 'Loss recovery state machine per account+strategy — anti-martingale, anti-revenge-trading';
COMMENT ON TABLE trading.trade_results IS 'Closed trade outcome persistence for recovery tracking and analytics';
COMMENT ON TABLE trading.blocked_signals IS 'Audit trail of signals blocked by recovery/risk gates';
COMMENT ON TABLE trading.adaptation_history IS 'Rule-based and ML-based adaptation decisions with full context';
COMMENT ON TABLE trading.hedge_positions IS 'Active hedge lifecycle tracking — original/hedge correlation audit';
COMMENT ON TABLE trading.hedge_history IS 'Closed hedge audit trail with net PnL';
COMMENT ON TABLE.rl_training_history IS 'RL model training runs with OOS validation metrics';
COMMENT ON TABLE trading.sentiment_snapshots IS 'Cached real-time sentiment state with provider health';
COMMENT ON TABLE trading.sentiment_items IS 'Individual sentiment data points with full provenance';
