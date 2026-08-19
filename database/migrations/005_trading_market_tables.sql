-- Predict-A-Trade v1.0.0 — Migration 005
-- Trading, Market Data, and Strategy Configuration Tables
-- SOW Sections 6, 6A, 8, 8A, 10, 11, 12, 12A-12F, 63, 63A, 140.1
-- Uses TimescaleDB hypertables for high-volume time-series where available.

-- ============================================================
-- Strategy Definitions (SOW Section 12A)
-- ============================================================
CREATE TABLE trading.strategy_definitions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id             VARCHAR(50) NOT NULL,
    -- STANDARD_SCALPING, ULTRA_SCALPING, STANDARD_SWING, TREND_SWING
    version                 VARCHAR(20) NOT NULL,
    status                  VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    -- DRAFT, RESEARCH, BACKTESTED, OOS_VALIDATED, PAPER, SHADOW, APPROVED, ACTIVE, SUSPENDED, DEPRECATED, ROLLED_BACK
    prediction_target_id    UUID,
    timeframe_profile_id    UUID,
    session_profile_id      UUID,
    feature_profile_id      UUID,
    confluence_profile_id   UUID,
    risk_profile_id         UUID,
    execution_profile_id    UUID,
    calibration_profile_id  UUID,
    exit_profile_id         UUID,
    gate_policy_version     VARCHAR(20),
    code_commit             VARCHAR(40),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at             TIMESTAMPTZ,
    approved_by             UUID REFERENCES iam.users(id),
    UNIQUE(strategy_id, version)
);

CREATE INDEX idx_strategy_def_id ON trading.strategy_definitions(strategy_id);
CREATE INDEX idx_strategy_def_status ON trading.strategy_definitions(status);

-- ============================================================
-- Strategy Timeframe Profiles (SOW Section 10A)
-- ============================================================
CREATE TABLE trading.strategy_timeframe_profiles (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id                 VARCHAR(50) NOT NULL,
    version                     VARCHAR(20) NOT NULL,
    context_timeframes          JSONB NOT NULL DEFAULT '[]',
    setup_timeframes            JSONB NOT NULL DEFAULT '[]',
    confirmation_timeframes     JSONB NOT NULL DEFAULT '[]',
    entry_timeframes            JSONB NOT NULL DEFAULT '[]',
    execution_resolution        VARCHAR(20) NOT NULL DEFAULT 'M1',
    required_freshness_ms_by_tf JSONB NOT NULL DEFAULT '{}',
    alignment_weights           JSONB NOT NULL DEFAULT '{}',
    conflict_policy             VARCHAR(40) NOT NULL DEFAULT 'HARD_REJECT',
    -- HARD_REJECT, REQUIRE_REVERSAL_CONFIRMATION, DOWNGRADE_GRADE, IGNORE_NON_REQUIRED_TF
    invalidation_timeframe      VARCHAR(10),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version)
);

-- ============================================================
-- Strategy Session Profiles (SOW Section 14A)
-- ============================================================
CREATE TABLE trading.strategy_session_profiles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id         VARCHAR(50) NOT NULL,
    version             VARCHAR(20) NOT NULL,
    allowed_sessions    JSONB NOT NULL DEFAULT '[]',
    blocked_sessions    JSONB NOT NULL DEFAULT '[]',
    news_policy         JSONB NOT NULL DEFAULT '{}',
    weekend_policy      VARCHAR(30) NOT NULL DEFAULT 'NO_NEW_ENTRY',
    rollover_policy     VARCHAR(30) NOT NULL DEFAULT 'NO_NEW_ENTRY',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version)
);

-- ============================================================
-- Confluence Profiles (SOW Section 12C)
-- ============================================================
CREATE TABLE trading.confluence_profiles (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id                 VARCHAR(50) NOT NULL,
    version                     VARCHAR(20) NOT NULL,
    mandatory_pillars           JSONB NOT NULL DEFAULT '[]',
    optional_pillars            JSONB NOT NULL DEFAULT '[]',
    weights                     JSONB NOT NULL DEFAULT '{}',
    minimum_score               DECIMAL(5,2) NOT NULL DEFAULT 75.0,
    minimum_long_short_separation DECIMAL(5,2) NOT NULL DEFAULT 20.0,
    minimum_confluence_count    INTEGER NOT NULL DEFAULT 3,
    maximum_missing_optional_weight DECIMAL(5,2) NOT NULL DEFAULT 15.0,
    grade_ceiling_by_data_capability JSONB NOT NULL DEFAULT '{}',
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version)
);

-- ============================================================
-- Confluence Weights (SOW Section 12C.1 — seed weight matrices)
-- ============================================================
CREATE TABLE trading.confluence_weights (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    confluence_profile_id UUID NOT NULL REFERENCES trading.confluence_profiles(id) ON DELETE CASCADE,
    evidence_group  VARCHAR(100) NOT NULL,
    weight          DECIMAL(5,2) NOT NULL,
    is_mandatory    BOOLEAN NOT NULL DEFAULT FALSE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(confluence_profile_id, evidence_group)
);

-- ============================================================
-- Risk Profiles (SOW Section 25A)
-- ============================================================
CREATE TABLE trading.risk_profiles (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id                 VARCHAR(50) NOT NULL,
    version                     VARCHAR(20) NOT NULL,
    risk_per_trade_policy       JSONB NOT NULL DEFAULT '{}',
    maximum_daily_loss          DECIMAL(18,8) NOT NULL DEFAULT 0,
    maximum_drawdown            DECIMAL(8,4) NOT NULL DEFAULT 0,
    maximum_concurrent_xau_exposure DECIMAL(18,8) NOT NULL DEFAULT 0,
    maximum_positions           INTEGER NOT NULL DEFAULT 1,
    maximum_new_trades_per_day  INTEGER NOT NULL DEFAULT 1,
    loss_cooldown_minutes       INTEGER NOT NULL DEFAULT 30,
    minimum_gross_rr            DECIMAL(8,4) NOT NULL DEFAULT 1.0,
    minimum_net_expectancy      DECIMAL(8,4) NOT NULL DEFAULT 0,
    maximum_spread_absolute     DECIMAL(18,8) NOT NULL DEFAULT 0.35,
    maximum_spread_to_atr       DECIMAL(8,4) NOT NULL DEFAULT 0.5,
    maximum_expected_slippage   DECIMAL(18,8) NOT NULL DEFAULT 0.10,
    maximum_total_cost_to_target DECIMAL(8,4) NOT NULL DEFAULT 0.25,
    minimum_margin_headroom     DECIMAL(8,4) NOT NULL DEFAULT 0.20,
    allowed_sessions            JSONB NOT NULL DEFAULT '[]',
    allowed_regimes             JSONB NOT NULL DEFAULT '[]',
    news_policy                 JSONB NOT NULL DEFAULT '{}',
    weekend_policy              VARCHAR(30) NOT NULL DEFAULT 'NO_NEW_ENTRY',
    carry_policy                VARCHAR(30) NOT NULL DEFAULT 'INCLUDE',
    correlation_policy          JSONB NOT NULL DEFAULT '{}',
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version)
);

-- ============================================================
-- Execution Profiles (SOW Section 26A)
-- ============================================================
CREATE TABLE trading.execution_profiles (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id             VARCHAR(50) NOT NULL,
    version                 VARCHAR(20) NOT NULL,
    allowed_order_types     JSONB NOT NULL DEFAULT '["MARKET"]',
    preferred_order_type    VARCHAR(20) NOT NULL DEFAULT 'MARKET',
    max_order_age_seconds   INTEGER NOT NULL DEFAULT 30,
    max_entry_deviation     DECIMAL(18,8) NOT NULL DEFAULT 0.10,
    max_slippage            DECIMAL(18,8) NOT NULL DEFAULT 0.10,
    max_queue_wait_seconds  INTEGER NOT NULL DEFAULT 5,
    partial_fill_policy     VARCHAR(30) NOT NULL DEFAULT 'ACCEPT_PARTIAL',
    cancel_replace_policy   VARCHAR(30) NOT NULL DEFAULT 'ALLOW',
    fallback_policy         VARCHAR(30) NOT NULL DEFAULT 'REJECT',
    max_latency_ms          INTEGER NOT NULL DEFAULT 100,
    max_jitter_ms           INTEGER NOT NULL DEFAULT 50,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version)
);

-- ============================================================
-- Exit Profiles (SOW Section 135)
-- ============================================================
CREATE TABLE trading.exit_profiles (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id             VARCHAR(50) NOT NULL,
    version                 VARCHAR(20) NOT NULL,
    entry_reference_policy  VARCHAR(30) NOT NULL DEFAULT 'MARKET',
    stop_model              VARCHAR(30) NOT NULL DEFAULT 'ATR',
    stop_atr_multiplier     DECIMAL(5,2) NOT NULL DEFAULT 1.0,
    structure_stop_policy   VARCHAR(30) NOT NULL DEFAULT 'PREFER_STRUCTURE',
    stop_buffer_policy      JSONB NOT NULL DEFAULT '{}',
    minimum_stop_distance_policy JSONB NOT NULL DEFAULT '{}',
    maximum_stop_distance_policy JSONB NOT NULL DEFAULT '{}',
    tp1_selection_policy    VARCHAR(50) NOT NULL DEFAULT 'NEAREST_LIQUIDITY',
    tp2_selection_policy    VARCHAR(50) NOT NULL DEFAULT 'NEXT_LIQUIDITY',
    tp3_selection_policy    VARCHAR(50) NOT NULL DEFAULT 'PROFILE_OBJECTIVE',
    tp1_fraction            DECIMAL(5,2) NOT NULL DEFAULT 0.50,
    tp2_fraction            DECIMAL(5,2) NOT NULL DEFAULT 0.50,
    tp3_fraction            DECIMAL(5,2) NOT NULL DEFAULT 0.00,
    breakeven_trigger       VARCHAR(30) NOT NULL DEFAULT 'AFTER_TP1_FILL',
    breakeven_buffer_policy JSONB NOT NULL DEFAULT '{}',
    trailing_trigger        VARCHAR(30) NOT NULL DEFAULT 'AFTER_TP2',
    trailing_model          VARCHAR(30) NOT NULL DEFAULT 'ATR_TRAIL',
    time_stop_policy        JSONB NOT NULL DEFAULT '{}',
    news_open_position_policy VARCHAR(30) NOT NULL DEFAULT 'HOLD',
    partial_fill_policy     VARCHAR(30) NOT NULL DEFAULT 'ACCEPT_PARTIAL',
    rounding_policy         JSONB NOT NULL DEFAULT '{}',
    broker_constraint_policy JSONB NOT NULL DEFAULT '{}',
    status                  VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    approved_by             UUID REFERENCES iam.users(id),
    approved_at             TIMESTAMPTZ,
    code_commit             VARCHAR(40),
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version)
);

-- ============================================================
-- Prediction Targets (SOW Section 15A)
-- ============================================================
CREATE TABLE trading.prediction_targets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     VARCHAR(50) NOT NULL,
    version         VARCHAR(20) NOT NULL,
    target_type     VARCHAR(100) NOT NULL,
    -- P(TP1 before SL within horizon), P(TP2 before SL), etc.
    target_definition JSONB NOT NULL,
    horizon         VARCHAR(50) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version, target_type)
);

-- ============================================================
-- Calibration Profiles (SOW Section 107, 130)
-- ============================================================
CREATE TABLE trading.calibration_profiles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id         VARCHAR(50) NOT NULL,
    version             VARCHAR(20) NOT NULL,
    method              VARCHAR(30) NOT NULL DEFAULT 'PLATT',
    -- PLATT, ISOTONIC
    brier_score         DECIMAL(10,6),
    ece                 DECIMAL(10,6),
    bin_count           INTEGER NOT NULL DEFAULT 10,
    sample_count        INTEGER NOT NULL DEFAULT 0,
    minimum_sample_size INTEGER NOT NULL DEFAULT 100,
    report_id           UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version)
);

-- ============================================================
-- Grade Policies (SOW Section 17A)
-- ============================================================
CREATE TABLE trading.grade_policies (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id                 VARCHAR(50) NOT NULL,
    prediction_target_id        UUID REFERENCES trading.prediction_targets(id),
    calibration_version         VARCHAR(20) NOT NULL,
    minimum_sample_size         INTEGER NOT NULL DEFAULT 100,
    probability_bin             JSONB NOT NULL DEFAULT '{}',
    confidence_interval_policy  JSONB NOT NULL DEFAULT '{}',
    minimum_expectancy          DECIMAL(8,4) NOT NULL DEFAULT 0,
    maximum_drawdown_condition  DECIMAL(8,4) NOT NULL DEFAULT 0.25,
    grade                       VARCHAR(5) NOT NULL,
    -- A+, A, B, C
    effective_from              TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Validation Policies (SOW Section 107A)
-- ============================================================
CREATE TABLE trading.validation_policies (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id                 VARCHAR(50) NOT NULL,
    version                     VARCHAR(20) NOT NULL,
    minimum_completed_labels    INTEGER NOT NULL DEFAULT 100,
    minimum_regime_coverage     JSONB NOT NULL DEFAULT '{}',
    minimum_session_coverage    JSONB NOT NULL DEFAULT '{}',
    minimum_calibration_quality JSONB NOT NULL DEFAULT '{}',
    minimum_cost_adjusted_expectancy DECIMAL(8,4) NOT NULL DEFAULT 0,
    maximum_drawdown            DECIMAL(8,4) NOT NULL DEFAULT 0.20,
    minimum_execution_quality   JSONB NOT NULL DEFAULT '{}',
    maximum_error_rate          DECIMAL(5,2) NOT NULL DEFAULT 5.0,
    required_shadow_duration_hours INTEGER NOT NULL DEFAULT 168,
    required_reviewer_roles     JSONB NOT NULL DEFAULT '["SUPER_ADMIN","RISK_MANAGER"]',
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(strategy_id, version)
);

-- ============================================================
-- Strategy Variant Definitions (SOW Section 12F)
-- ============================================================
CREATE TABLE trading.strategy_variant_definitions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    variant_id          VARCHAR(100) NOT NULL UNIQUE,
    strategy_id         VARCHAR(50) NOT NULL,
    version             VARCHAR(20) NOT NULL,
    name                VARCHAR(255) NOT NULL,
    status              VARCHAR(30) NOT NULL DEFAULT 'DRAFT',
    required_capabilities JSONB NOT NULL DEFAULT '[]',
    required_features   JSONB NOT NULL DEFAULT '[]',
    entry_rule_expression TEXT,
    invalidation_rule_expression TEXT,
    target_rule_expression TEXT,
    session_profile_id  UUID,
    risk_profile_id     UUID,
    execution_profile_id UUID,
    prediction_target_id UUID,
    minimum_data_quality VARCHAR(20) NOT NULL DEFAULT 'COMPLETE',
    grade_ceiling       VARCHAR(5) NOT NULL DEFAULT 'B',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    approved_at         TIMESTAMPTZ,
    approved_by         UUID REFERENCES iam.users(id),
    code_commit         VARCHAR(40)
);

-- ============================================================
-- Indicator Parameter Profiles (SOW Section 12B, 132)
-- ============================================================
CREATE TABLE trading.indicator_parameter_profiles (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    profile_key         VARCHAR(200) NOT NULL,
    -- symbol + strategy + timeframe + regime + broker_class + feature_version
    parameters          JSONB NOT NULL DEFAULT '{}',
    version             VARCHAR(20) NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(profile_key, version)
);

-- ============================================================
-- Feature Definitions (SOW Section 63A, 132)
-- ============================================================
CREATE TABLE trading.feature_definitions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    indicator_id        VARCHAR(100) NOT NULL UNIQUE,
    implementation_version VARCHAR(20) NOT NULL,
    formula_version     VARCHAR(20) NOT NULL,
    parameter_schema    JSONB NOT NULL DEFAULT '{}',
    default_seed_parameters JSONB NOT NULL DEFAULT '{}',
    required_capabilities JSONB NOT NULL DEFAULT '[]',
    timeframe_applicability JSONB NOT NULL DEFAULT '[]',
    strategy_applicability JSONB NOT NULL DEFAULT '[]',
    warmup_requirement  INTEGER NOT NULL DEFAULT 0,
    missing_data_policy VARCHAR(30) NOT NULL DEFAULT 'REJECT',
    quality_state_rules JSONB NOT NULL DEFAULT '{}',
    normalization_policy JSONB NOT NULL DEFAULT '{}',
    approved_status     VARCHAR(20) NOT NULL DEFAULT 'DRAFT',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Candle Pattern Definitions (SOW Section 133)
-- ============================================================
CREATE TABLE trading.candle_pattern_definitions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_id          VARCHAR(100) NOT NULL UNIQUE,
    pattern_version     VARCHAR(20) NOT NULL,
    numeric_definition  JSONB NOT NULL,
    context_requirements JSONB NOT NULL DEFAULT '[]',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Market Calendars & Session Definitions (SOW Section 14A)
-- ============================================================
CREATE TABLE market.session_definitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(50) NOT NULL,
    timezone        VARCHAR(50) NOT NULL,
    -- IANA timezone, e.g. Europe/London, America/New_York
    local_start     TIME NOT NULL,
    local_end       TIME NOT NULL,
    calendar_id     VARCHAR(50) NOT NULL DEFAULT 'default',
    dst_policy      VARCHAR(20) NOT NULL DEFAULT 'IANA',
    holiday_policy  VARCHAR(20) NOT NULL DEFAULT 'CLOSED',
    allowed_strategies JSONB NOT NULL DEFAULT '[]',
    volatility_profile VARCHAR(30) NOT NULL DEFAULT 'MEDIUM',
    liquidity_profile VARCHAR(30) NOT NULL DEFAULT 'MEDIUM',
    spread_multiplier DECIMAL(5,2) NOT NULL DEFAULT 1.0,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(name, calendar_id)
);

CREATE TABLE market.holiday_calendars (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id     VARCHAR(50) NOT NULL,
    date            DATE NOT NULL,
    name            VARCHAR(100) NOT NULL,
    is_closed       BOOLEAN NOT NULL DEFAULT TRUE,
    early_close_time TIME,
    late_open_time  TIME,
    timezone        VARCHAR(50) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(calendar_id, date)
);

CREATE TABLE market.gold_fix_windows (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(50) NOT NULL,
    -- LBMA_AM, LBMA_PM
    timezone        VARCHAR(50) NOT NULL DEFAULT 'Europe/London',
    local_time      TIME NOT NULL,
    duration_minutes INTEGER NOT NULL DEFAULT 15,
    dst_policy      VARCHAR(20) NOT NULL DEFAULT 'IANA',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(name)
);

-- ============================================================
-- Data Provider Capabilities (SOW Section 6A.1)
-- ============================================================
CREATE TABLE market.data_provider_capabilities (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id         VARCHAR(100) NOT NULL,
    dataset_id          VARCHAR(100) NOT NULL,
    instrument          VARCHAR(50) NOT NULL,
    capabilities        JSONB NOT NULL DEFAULT '[]',
    source_timezone     VARCHAR(50) NOT NULL DEFAULT 'UTC',
    timestamp_precision VARCHAR(20) NOT NULL DEFAULT 'MILLI',
    sequence_support    BOOLEAN NOT NULL DEFAULT FALSE,
    historical_depth    VARCHAR(50) NOT NULL DEFAULT '5Y',
    redistribution_rights BOOLEAN NOT NULL DEFAULT FALSE,
    retention_rights    VARCHAR(50) NOT NULL DEFAULT '5Y',
    quality_sla         VARCHAR(100),
    enabled             BOOLEAN NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(provider_id, dataset_id, instrument)
);

-- ============================================================
-- Futures Contracts & Roll Calendar (SOW Section 6A.4)
-- ============================================================
CREATE TABLE market.futures_contracts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol          VARCHAR(50) NOT NULL,
    -- GC
    contract_month  VARCHAR(10) NOT NULL,
    -- F,G,H,J,K,M,N,Q,U,V,X,Z
    contract_year   INTEGER NOT NULL,
    expiry_date     DATE NOT NULL,
    first_notice_date DATE,
    active          BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(symbol, contract_month, contract_year)
);

CREATE TABLE market.futures_roll_calendar (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol          VARCHAR(50) NOT NULL,
    from_contract_id UUID NOT NULL REFERENCES market.futures_contracts(id),
    to_contract_id   UUID NOT NULL REFERENCES market.futures_contracts(id),
    roll_date       DATE NOT NULL,
    roll_method     VARCHAR(30) NOT NULL DEFAULT 'VOLUME_OI',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Broker Execution Profiles (SOW Section 103A)
-- ============================================================
CREATE TABLE market.broker_execution_profiles (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    broker                      VARCHAR(100) NOT NULL,
    server                      VARCHAR(255) NOT NULL,
    platform                    VARCHAR(10) NOT NULL,
    -- MT4, MT5
    canonical_symbol            VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    broker_symbol               VARCHAR(50) NOT NULL,
    aliases                     JSONB NOT NULL DEFAULT '[]',
    base_currency               VARCHAR(3) NOT NULL DEFAULT 'USD',
    quote_currency              VARCHAR(3) NOT NULL DEFAULT 'USD',
    account_currency            VARCHAR(3) NOT NULL DEFAULT 'USD',
    digits                      INTEGER NOT NULL DEFAULT 2,
    point                       DECIMAL(18,10) NOT NULL DEFAULT 0.01,
    tick_size                   DECIMAL(18,10) NOT NULL DEFAULT 0.01,
    tick_value                  DECIMAL(18,8) NOT NULL DEFAULT 1.0,
    tick_value_currency         VARCHAR(3) NOT NULL DEFAULT 'USD',
    contract_size               DECIMAL(18,4) NOT NULL DEFAULT 100,
    minimum_lot                 DECIMAL(18,4) NOT NULL DEFAULT 0.01,
    maximum_lot                 DECIMAL(18,4) NOT NULL DEFAULT 100,
    lot_step                    DECIMAL(18,4) NOT NULL DEFAULT 0.01,
    stops_level                 INTEGER NOT NULL DEFAULT 0,
    freeze_level                INTEGER NOT NULL DEFAULT 0,
    fill_modes                  JSONB NOT NULL DEFAULT '["IOC","FOK"]',
    market_sessions             JSONB NOT NULL DEFAULT '[]',
    maintenance_breaks          JSONB NOT NULL DEFAULT '[]',
    swap_long                   DECIMAL(18,8) NOT NULL DEFAULT 0,
    swap_short                  DECIMAL(18,8) NOT NULL DEFAULT 0,
    swap_calculation_method     VARCHAR(30) NOT NULL DEFAULT 'POINTS',
    triple_swap_day             VARCHAR(20) NOT NULL DEFAULT 'WEDNESDAY',
    margin_calculation          VARCHAR(30) NOT NULL DEFAULT 'FOREX',
    leverage_tiers              JSONB NOT NULL DEFAULT '[]',
    commission_model            JSONB NOT NULL DEFAULT '{}',
    typical_spread              DECIMAL(18,8) NOT NULL DEFAULT 0.30,
    spread_p95                  DECIMAL(18,8) NOT NULL DEFAULT 0.50,
    slippage_distribution       JSONB NOT NULL DEFAULT '{}',
    last_observed_at            TIMESTAMPTZ,
    last_validated_at           TIMESTAMPTZ,
    qualification_result        VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    -- APPROVED_SIGNAL_ONLY, APPROVED_MANUAL, APPROVED_ASSISTED, APPROVED_AUTO_STANDARD, APPROVED_AUTO_ULTRA, REJECTED
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE(broker, server, platform, broker_symbol)
);

-- ============================================================
-- Signals (SOW Section 63)
-- ============================================================
CREATE TABLE trading.signals (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id           VARCHAR(100) NOT NULL UNIQUE,
    symbol              VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    strategy_id         VARCHAR(50) NOT NULL,
    strategy_definition_id UUID REFERENCES trading.strategy_definitions(id),
    direction           VARCHAR(10) NOT NULL,
    -- BUY, SELL, NO-TRADE
    grade               VARCHAR(5),
    -- A+, A, B, C, RESEARCH, UNRATED, SHADOW
    raw_score           DECIMAL(5,2),
    long_score          DECIMAL(5,2),
    short_score         DECIMAL(5,2),
    calibrated_probability DECIMAL(8,6),
    prediction_target_id UUID REFERENCES trading.prediction_targets(id),
    entry_price         DECIMAL(18,8),
    entry_zone_low      DECIMAL(18,8),
    entry_zone_high     DECIMAL(18,8),
    stop_loss           DECIMAL(18,8),
    tp1                 DECIMAL(18,8),
    tp2                 DECIMAL(18,8),
    tp3                 DECIMAL(18,8),
    gross_rr_tp1        DECIMAL(10,4),
    gross_rr_tp2        DECIMAL(10,4),
    gross_rr_tp3        DECIMAL(10,4),
    net_rr_tp1          DECIMAL(10,4),
    net_rr_tp2          DECIMAL(10,4),
    net_rr_tp3          DECIMAL(10,4),
    expected_cost       DECIMAL(18,8),
    regime              VARCHAR(50),
    session             VARCHAR(50),
    news_risk           VARCHAR(20),
    timeframe           VARCHAR(10),
    ttl_seconds         INTEGER NOT NULL DEFAULT 900,
    status              VARCHAR(30) NOT NULL DEFAULT 'DETECTED',
    -- Full lifecycle from SOW Section 19
    reason_codes        JSONB NOT NULL DEFAULT '[]',
    feature_snapshot_id UUID,
    exit_profile_id     UUID REFERENCES trading.exit_profiles(id),
    gate_policy_version VARCHAR(20),
    gate_results        JSONB NOT NULL DEFAULT '[]',
    evidence_summary    JSONB NOT NULL DEFAULT '{}',
    performance_claim_id UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ NOT NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_signals_strategy ON trading.signals(strategy_id);
CREATE INDEX idx_signals_status ON trading.signals(status);
CREATE INDEX idx_signals_created ON trading.signals(created_at DESC);
CREATE INDEX idx_signals_direction ON trading.signals(direction);

-- ============================================================
-- Signal Events (SOW Section 19, 63)
-- ============================================================
CREATE TABLE trading.signal_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id       UUID NOT NULL REFERENCES trading.signals(id) ON DELETE CASCADE,
    event_type      VARCHAR(50) NOT NULL,
    -- All lifecycle states from SOW Section 19
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_signal_events_signal ON trading.signal_events(signal_id);
CREATE INDEX idx_signal_events_type ON trading.signal_events(event_type);

-- ============================================================
-- Signal Snapshots (SOW Section 64 — full reproducibility)
-- ============================================================
CREATE TABLE trading.signal_snapshots (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id       UUID NOT NULL REFERENCES trading.signals(id) ON DELETE CASCADE,
    snapshot_type   VARCHAR(30) NOT NULL,
    -- FULL_DECISION, FEATURE_STATE, RISK_STATE, EXECUTION_STATE
    market_data     JSONB NOT NULL DEFAULT '{}',
    features        JSONB NOT NULL DEFAULT '{}',
    indicators      JSONB NOT NULL DEFAULT '{}',
    regime          JSONB NOT NULL DEFAULT '{}',
    structure       JSONB NOT NULL DEFAULT '{}',
    liquidity       JSONB NOT NULL DEFAULT '{}',
    volume          JSONB NOT NULL DEFAULT '{}',
    vwap            JSONB NOT NULL DEFAULT '{}',
    volatility      JSONB NOT NULL DEFAULT '{}',
    dxy             JSONB NOT NULL DEFAULT '{}',
    yields          JSONB NOT NULL DEFAULT '{}',
    news            JSONB NOT NULL DEFAULT '{}',
    session         JSONB NOT NULL DEFAULT '{}',
    pillar_scores   JSONB NOT NULL DEFAULT '{}',
    model_versions  JSONB NOT NULL DEFAULT '{}',
    ai_outputs      JSONB NOT NULL DEFAULT '{}',
    risk_rules      JSONB NOT NULL DEFAULT '{}',
    config_version  VARCHAR(50) NOT NULL,
    strategy_version VARCHAR(50) NOT NULL,
    exit_profile_id UUID,
    gate_policy_version VARCHAR(20),
    gate_results    JSONB NOT NULL DEFAULT '[]',
    indicator_versions JSONB NOT NULL DEFAULT '{}',
    candle_pattern_flags JSONB NOT NULL DEFAULT '[]',
    final_decision  JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_signal_snapshots_signal ON trading.signal_snapshots(signal_id);

-- ============================================================
-- Signal Recipients (SOW Section 63)
-- ============================================================
CREATE TABLE trading.signal_recipients (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id       UUID NOT NULL REFERENCES trading.signals(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    delivery_channel VARCHAR(30) NOT NULL,
    -- WS, REST, WINDOWS_AGENT, MT4, MT5
    delivered_at    TIMESTAMPTZ,
    acknowledged_at TIMESTAMPTZ,
    delivery_status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
    -- PENDING, DELIVERED, ACKNOWLEDGED, FAILED, EXPIRED
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_signal_recipients_signal ON trading.signal_recipients(signal_id);
CREATE INDEX idx_signal_recipients_user ON trading.signal_recipients(user_id);

-- ============================================================
-- Predictions (SOW Section 15)
-- ============================================================
CREATE TABLE trading.predictions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id           UUID NOT NULL REFERENCES trading.signals(id) ON DELETE CASCADE,
    symbol              VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    direction           VARCHAR(10) NOT NULL,
    prediction_horizon  VARCHAR(50) NOT NULL,
    timeframe           VARCHAR(10) NOT NULL,
    entry_model         VARCHAR(100),
    target_definition   JSONB NOT NULL DEFAULT '{}',
    invalidation_definition JSONB NOT NULL DEFAULT '{}',
    raw_score           DECIMAL(5,2),
    calibrated_probability DECIMAL(8,6),
    expected_move       DECIMAL(18,8),
    expected_rr         DECIMAL(10,4),
    confidence_band     JSONB NOT NULL DEFAULT '{}',
    model_version       VARCHAR(50) NOT NULL,
    strategy_version    VARCHAR(50) NOT NULL,
    feature_snapshot_id UUID,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ NOT NULL
);

-- ============================================================
-- Prediction Outcomes (SOW Section 63)
-- ============================================================
CREATE TABLE trading.prediction_outcomes (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prediction_id   UUID NOT NULL REFERENCES trading.predictions(id) ON DELETE CASCADE,
    outcome_type    VARCHAR(50) NOT NULL,
    -- TP1_HIT, TP2_HIT, TP3_HIT, SL_HIT, EXPIRED, CLOSED
    outcome_value   BOOLEAN,
    realized_rr     DECIMAL(10,4),
    mfe             DECIMAL(18,8),
    mae             DECIMAL(18,8),
    time_to_outcome INTERVAL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Risk Decisions (SOW Section 63)
-- ============================================================
CREATE TABLE trading.risk_decisions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id       UUID REFERENCES trading.signals(id) ON DELETE CASCADE,
    decision        VARCHAR(20) NOT NULL,
    -- PASS, DENIED, NO-TRADE
    gate_id         VARCHAR(50) NOT NULL,
    gate_result     VARCHAR(20) NOT NULL,
    -- PASS, VETO, DEGRADED, UNKNOWN, STALE
    reason_codes    JSONB NOT NULL DEFAULT '[]',
    metadata        JSONB NOT NULL DEFAULT '{}',
    evaluated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_risk_decisions_signal ON trading.risk_decisions(signal_id);

-- ============================================================
-- Execution Commands (SOW Section 49, 50, 63)
-- ============================================================
CREATE TABLE trading.execution_commands (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id      VARCHAR(100) NOT NULL UNIQUE,
    signal_id       UUID NOT NULL REFERENCES trading.signals(id),
    license_id      UUID REFERENCES licensing.licenses(id),
    device_id       UUID REFERENCES licensing.devices(id),
    account_id      UUID REFERENCES licensing.mt_accounts(id),
    symbol          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    direction       VARCHAR(10) NOT NULL,
    order_type      VARCHAR(20) NOT NULL,
    volume          DECIMAL(18,4) NOT NULL,
    entry           DECIMAL(18,8),
    stop_loss       DECIMAL(18,8),
    take_profit     DECIMAL(18,8),
    strategy        VARCHAR(50) NOT NULL,
    issued_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at      TIMESTAMPTZ NOT NULL,
    nonce           VARCHAR(100) NOT NULL,
    signature       TEXT NOT NULL,
    execution_status VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    -- PENDING, SENT, ACKNOWLEDGED, FILLED, PARTIALLY_FILLED, REJECTED, CANCELLED, TIMEOUT
    broker_ticket   VARCHAR(255),
    received_at     TIMESTAMPTZ,
    executed_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_exec_commands_signal ON trading.execution_commands(signal_id);
CREATE UNIQUE INDEX idx_exec_commands_command ON trading.execution_commands(command_id);

-- ============================================================
-- Execution Events (SOW Section 63)
-- ============================================================
CREATE TABLE trading.execution_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id      UUID NOT NULL REFERENCES trading.execution_commands(id) ON DELETE CASCADE,
    event_type      VARCHAR(50) NOT NULL,
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_exec_events_command ON trading.execution_events(command_id);

-- ============================================================
-- Positions (SOW Section 63)
-- ============================================================
CREATE TABLE trading.positions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    mt_account_id   UUID REFERENCES licensing.mt_accounts(id),
    signal_id       UUID REFERENCES trading.signals(id),
    command_id      UUID REFERENCES trading.execution_commands(id),
    symbol          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    strategy_id     VARCHAR(50) NOT NULL,
    direction       VARCHAR(10) NOT NULL,
    volume          DECIMAL(18,4) NOT NULL,
    entry_price     DECIMAL(18,8) NOT NULL,
    current_price   DECIMAL(18,8),
    stop_loss       DECIMAL(18,8),
    take_profit1    DECIMAL(18,8),
    take_profit2    DECIMAL(18,8),
    take_profit3    DECIMAL(18,8),
    status          VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    -- OPEN, PARTIALLY_CLOSED, CLOSED
    realized_pnl    DECIMAL(18,8) NOT NULL DEFAULT 0,
    unrealized_pnl  DECIMAL(18,8) NOT NULL DEFAULT 0,
    mfe             DECIMAL(18,8) NOT NULL DEFAULT 0,
    mae             DECIMAL(18,8) NOT NULL DEFAULT 0,
    breakeven_moved BOOLEAN NOT NULL DEFAULT FALSE,
    trailing_active BOOLEAN NOT NULL DEFAULT FALSE,
    opened_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at       TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_positions_user ON trading.positions(user_id);
CREATE INDEX idx_positions_status ON trading.positions(status);

-- ============================================================
-- Trades (SOW Section 63)
-- ============================================================
CREATE TABLE trading.trades (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    position_id     UUID NOT NULL REFERENCES trading.positions(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    symbol          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    strategy_id     VARCHAR(50) NOT NULL,
    direction       VARCHAR(10) NOT NULL,
    volume          DECIMAL(18,4) NOT NULL,
    entry_price     DECIMAL(18,8) NOT NULL,
    exit_price      DECIMAL(18,8),
    realized_pnl    DECIMAL(18,8) NOT NULL DEFAULT 0,
    realized_rr     DECIMAL(10,4),
    commission      DECIMAL(18,8) NOT NULL DEFAULT 0,
    swap            DECIMAL(18,8) NOT NULL DEFAULT 0,
    slippage        DECIMAL(18,8) NOT NULL DEFAULT 0,
    exit_reason     VARCHAR(50),
    -- TP1, TP2, TP3, SL, MANUAL, TIME_STOP, SIGNAL_EXPIRED
    opened_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at       TIMESTAMPTZ,
    duration        INTERVAL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_trades_user ON trading.trades(user_id);
CREATE INDEX idx_trades_position ON trading.trades(position_id);

-- ============================================================
-- Broker Events (SOW Section 63)
-- ============================================================
CREATE TABLE trading.broker_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    command_id      UUID REFERENCES trading.execution_commands(id),
    position_id     UUID REFERENCES trading.positions(id),
    event_type      VARCHAR(50) NOT NULL,
    broker_ticket   VARCHAR(255),
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Manual Interventions (SOW Section 84)
-- ============================================================
CREATE TABLE trading.manual_interventions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    position_id     UUID REFERENCES trading.positions(id),
    actor_id        UUID REFERENCES iam.users(id),
    action          VARCHAR(50) NOT NULL,
    -- CLOSE, MODIFY_SL, MODIFY_TP, PARTIAL_CLOSE, MANUAL_OVERRIDE
    old_value       JSONB,
    new_value       JSONB,
    reason          TEXT,
    policy          VARCHAR(30) NOT NULL DEFAULT 'ALLOW',
    -- ALLOW, WARN, RE-ALIGN, BLOCK_NEW_AUTOMATION
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Liquidity Pools (SOW Section 12D)
-- ============================================================
CREATE TABLE trading.liquidity_pools (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pool_id         VARCHAR(100) NOT NULL UNIQUE,
    symbol          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    type            VARCHAR(50) NOT NULL,
    -- BSL, SSL, EQUAL_HIGHS, EQUAL_LOWS, PREVIOUS_DAY_HIGH, etc.
    side            VARCHAR(10) NOT NULL,
    -- BUY, SELL
    price           DECIMAL(18,8) NOT NULL,
    price_tolerance DECIMAL(18,8) NOT NULL DEFAULT 0.10,
    strength        DECIMAL(5,2) NOT NULL DEFAULT 50.0,
    timeframe       VARCHAR(10) NOT NULL,
    session         VARCHAR(50),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_touched_at TIMESTAMPTZ,
    swept_at        TIMESTAMPTZ,
    mitigated_at    TIMESTAMPTZ,
    invalidated_at  TIMESTAMPTZ,
    status          VARCHAR(20) NOT NULL DEFAULT 'ACTIVE',
    -- ACTIVE, TOUCHED, SWEPT, MITIGATED, INVALIDATED, EXPIRED
    source_snapshot_id UUID,
    feature_version VARCHAR(20) NOT NULL DEFAULT '1.0'
);

CREATE INDEX idx_liquidity_pools_symbol ON trading.liquidity_pools(symbol);
CREATE INDEX idx_liquidity_pools_status ON trading.liquidity_pools(status);
CREATE INDEX idx_liquidity_pools_price ON trading.liquidity_pools(price);

-- ============================================================
-- Sweep Events (SOW Section 12D)
-- ============================================================
CREATE TABLE trading.sweep_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sweep_id        VARCHAR(100) NOT NULL UNIQUE,
    pool_id         UUID NOT NULL REFERENCES trading.liquidity_pools(id),
    event_time      TIMESTAMPTZ NOT NULL,
    extreme_price   DECIMAL(18,8) NOT NULL,
    penetration_distance DECIMAL(18,8) NOT NULL,
    close_back_distance DECIMAL(18,8) NOT NULL,
    rejection_wick_ratio DECIMAL(8,4),
    displacement_after DECIMAL(18,8),
    volume_or_flow_confirmation BOOLEAN NOT NULL DEFAULT FALSE,
    bos_after       BOOLEAN NOT NULL DEFAULT FALSE,
    quality         VARCHAR(20) NOT NULL DEFAULT 'DERIVED'
);

-- ============================================================
-- Structure Events (SOW Section 12D.2)
-- ============================================================
CREATE TABLE trading.structure_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id        VARCHAR(100) NOT NULL UNIQUE,
    symbol          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    event_type      VARCHAR(30) NOT NULL,
    -- BOS, CHoCH, MSS, HH, HL, LH, LL
    timeframe       VARCHAR(10) NOT NULL,
    direction       VARCHAR(10),
    -- BULLISH, BEARISH
    price_level     DECIMAL(18,8),
    break_method    VARCHAR(20),
    -- WICK_BREAK, CLOSE_BREAK, BODY_CONFIRMATION
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_structure_events_symbol ON trading.structure_events(symbol);
CREATE INDEX idx_structure_events_type ON trading.structure_events(event_type);

-- ============================================================
-- FVG Zones (SOW Section 12D.3)
-- ============================================================
CREATE TABLE trading.fvg_zones (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    type            VARCHAR(20) NOT NULL,
    -- BULLISH, BEARISH
    timeframe       VARCHAR(10) NOT NULL,
    high_price      DECIMAL(18,8) NOT NULL,
    low_price       DECIMAL(18,8) NOT NULL,
    fill_ratio      DECIMAL(5,2) NOT NULL DEFAULT 0.0,
    status          VARCHAR(20) NOT NULL DEFAULT 'FRESH',
    -- FRESH, PARTIAL, MITIGATED, INVERTED, INVALIDATED
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Order Blocks (SOW Section 12D.4)
-- ============================================================
CREATE TABLE trading.order_blocks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    type            VARCHAR(20) NOT NULL,
    -- BULLISH, BEARISH, BREAKER
    timeframe       VARCHAR(10) NOT NULL,
    high_price      DECIMAL(18,8) NOT NULL,
    low_price       DECIMAL(18,8) NOT NULL,
    status          VARCHAR(20) NOT NULL DEFAULT 'FRESH',
    -- FRESH, MITIGATED, INVALIDATED
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- TimescaleDB Hypertables (SOW Section 57)
-- ============================================================
-- These are created as hypertables if TimescaleDB is available

-- Ticks table
CREATE TABLE market.ticks (
    time            TIMESTAMPTZ NOT NULL,
    symbol          VARCHAR(20) NOT NULL,
    bid             DECIMAL(18,8) NOT NULL,
    ask             DECIMAL(18,8) NOT NULL,
    mid             DECIMAL(18,8) NOT NULL,
    spread          DECIMAL(18,8) NOT NULL,
    tick_volume     BIGINT NOT NULL DEFAULT 0,
    source          VARCHAR(50) NOT NULL,
    source_timestamp TIMESTAMPTZ,
    gateway_receipt_time TIMESTAMPTZ NOT NULL DEFAULT now(),
    quality         VARCHAR(20) NOT NULL DEFAULT 'AUTHORITATIVE',
    PRIMARY KEY (time, symbol, source)
);

-- Candles table
CREATE TABLE market.candles (
    time            TIMESTAMPTZ NOT NULL,
    symbol          VARCHAR(20) NOT NULL,
    timeframe       VARCHAR(10) NOT NULL,
    open            DECIMAL(18,8) NOT NULL,
    high            DECIMAL(18,8) NOT NULL,
    low             DECIMAL(18,8) NOT NULL,
    close           DECIMAL(18,8) NOT NULL,
    volume          BIGINT NOT NULL DEFAULT 0,
    source          VARCHAR(50) NOT NULL,
    quality         VARCHAR(20) NOT NULL DEFAULT 'COMPLETE',
    -- COMPLETE, PARTIAL, ESTIMATED, STALE, INVALID
    alignment_profile VARCHAR(30) NOT NULL DEFAULT 'BROKER_ALIGNED_UTC_PLUS_3',
    is_closed       BOOLEAN NOT NULL DEFAULT FALSE,
    PRIMARY KEY (time, symbol, timeframe, source)
);

-- Market states
CREATE TABLE market.market_states (
    time            TIMESTAMPTZ NOT NULL,
    symbol          VARCHAR(20) NOT NULL,
    regime          VARCHAR(50) NOT NULL,
    session         VARCHAR(50) NOT NULL,
    volatility      DECIMAL(18,8),
    spread          DECIMAL(18,8),
    data_quality    VARCHAR(20) NOT NULL DEFAULT 'AUTHORITATIVE',
    metadata        JSONB NOT NULL DEFAULT '{}',
    PRIMARY KEY (time, symbol)
);

-- Flow features (SOW Section 57A)
CREATE TABLE market.flow_features (
    time            TIMESTAMPTZ NOT NULL,
    symbol          VARCHAR(20) NOT NULL,
    source_instrument VARCHAR(50) NOT NULL,
    futures_contract VARCHAR(50),
    feature_name    VARCHAR(100) NOT NULL,
    value           DECIMAL(18,8) NOT NULL,
    quality         VARCHAR(20) NOT NULL DEFAULT 'AUTHORITATIVE',
    strategy_relevance VARCHAR(50),
    source_sequence BIGINT,
    latency_ms      INTEGER,
    derivation_version VARCHAR(20),
    PRIMARY KEY (time, symbol, source_instrument, feature_name)
);

-- Signal delivery receipts (SOW Section 140.1)
CREATE TABLE trading.signal_delivery_receipts (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id       UUID NOT NULL REFERENCES trading.signals(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    stream_id       VARCHAR(100),
    sequence        BIGINT NOT NULL,
    delivered_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    acknowledged_at TIMESTAMPTZ
    -- PK is id (defined above)
);

-- Convert high-volume tables to TimescaleDB hypertables if available
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'timescaledb') THEN
        -- Ticks: 1-hour chunks
        PERFORM create_hypertable('market.ticks', 'time', chunk_time_interval => INTERVAL '1 hour');
        -- Candles: 1-day chunks
        PERFORM create_hypertable('market.candles', 'time', chunk_time_interval => INTERVAL '1 day');
        -- Market states: 1-hour chunks
        PERFORM create_hypertable('market.market_states', 'time', chunk_time_interval => INTERVAL '1 hour');
        -- Flow features: 1-hour chunks
        PERFORM create_hypertable('market.flow_features', 'time', chunk_time_interval => INTERVAL '1 hour');

        -- Compression policies (SOW Section 57)
        PERFORM add_compression_policy('market.ticks', INTERVAL '7 days');
        PERFORM add_compression_policy('market.candles', INTERVAL '30 days');
        PERFORM add_compression_policy('market.market_states', INTERVAL '7 days');
        PERFORM add_compression_policy('market.flow_features', INTERVAL '7 days');

        -- Retention policies
        PERFORM add_retention_policy('market.ticks', INTERVAL '90 days');
        PERFORM add_retention_policy('market.market_states', INTERVAL '365 days');
        PERFORM add_retention_policy('market.flow_features', INTERVAL '365 days');

        RAISE NOTICE 'TimescaleDB hypertables created with compression and retention policies';
    ELSE
        RAISE NOTICE 'TimescaleDB not available — tables created as regular tables';
    END IF;
EXCEPTION WHEN OTHERS THEN
    RAISE NOTICE 'TimescaleDB hypertable creation skipped: %', SQLERRM;
END
$$;

-- ============================================================
-- Gate Definitions (SOW Section 131, 140.1)
-- ============================================================
CREATE TABLE trading.gate_definitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    gate_id         VARCHAR(50) NOT NULL UNIQUE,
    -- data_quality, session, news, spread, slippage, total_cost, exposure,
    -- margin, rr_net_expectancy, entitlement, license, execution_permission
    class           VARCHAR(20) NOT NULL,
    -- fast (<1ms), mid (<5ms), background
    description     TEXT,
    fail_closed     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Gate Policy Versions (SOW Section 131.7)
-- ============================================================
CREATE TABLE trading.gate_policy_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    version         VARCHAR(20) NOT NULL UNIQUE,
    gate_ids        JSONB NOT NULL DEFAULT '[]',
    ordering        JSONB NOT NULL DEFAULT '[]',
    freshness_requirements JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Performance Claims (SOW Section 130.6)
-- ============================================================
CREATE TABLE trading.performance_claims (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name         VARCHAR(100) NOT NULL,
    metric_definition   JSONB NOT NULL,
    strategy_id         VARCHAR(50) NOT NULL,
    strategy_definition_id UUID,
    prediction_target_id UUID,
    exit_profile_id     UUID,
    dataset_id          VARCHAR(100),
    period_start        DATE NOT NULL,
    period_end          DATE NOT NULL,
    sample_size         INTEGER NOT NULL,
    cost_model_version  VARCHAR(50),
    slippage_assumption VARCHAR(100),
    broker_profile_scope VARCHAR(100),
    oos_status          VARCHAR(30) NOT NULL,
    confidence_interval_method VARCHAR(50),
    point_estimate      DECIMAL(10,6),
    lower_bound         DECIMAL(10,6),
    upper_bound         DECIMAL(10,6),
    report_id           UUID,
    report_version      VARCHAR(20),
    approved_by         UUID REFERENCES iam.users(id),
    approved_at         TIMESTAMPTZ,
    expires_at          TIMESTAMPTZ,
    public_visibility   BOOLEAN NOT NULL DEFAULT FALSE,
    verification_state  VARCHAR(20) NOT NULL DEFAULT 'UNVERIFIED',
    -- VERIFIED, UNVERIFIED, EXPIRED, REVOKED
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Calibration Reports (SOW Section 140.1)
-- ============================================================
CREATE TABLE trading.calibration_reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     VARCHAR(50) NOT NULL,
    strategy_definition_id UUID,
    prediction_target_id UUID,
    calibration_version VARCHAR(20),
    brier_score     DECIMAL(10,6),
    ece             DECIMAL(10,6),
    bin_count       INTEGER NOT NULL,
    sample_count    INTEGER NOT NULL,
    bins            JSONB NOT NULL DEFAULT '[]',
    -- [{bin_low, bin_high, mean_forecast, observed_frequency, count}]
    report_version  VARCHAR(20) NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Feature Parity Runs (SOW Section 137.5)
-- ============================================================
CREATE TABLE trading.feature_parity_runs (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    fixture_set_version VARCHAR(50) NOT NULL,
    go_commit           VARCHAR(40) NOT NULL,
    python_commit       VARCHAR(40) NOT NULL,
    config_version      VARCHAR(50) NOT NULL,
    mismatch_count      INTEGER NOT NULL DEFAULT 0,
    max_numeric_error   DECIMAL(18,12) NOT NULL DEFAULT 0,
    status              VARCHAR(20) NOT NULL DEFAULT 'PASS',
    -- PASS, FAIL
    artifact_path       TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Compliance Approvals (SOW Section 138.5)
-- ============================================================
CREATE TABLE control.compliance_approvals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    legal_review_id VARCHAR(100) NOT NULL,
    jurisdiction    VARCHAR(100) NOT NULL,
    scope           VARCHAR(200) NOT NULL,
    counsel_provider VARCHAR(255),
    issued_at       TIMESTAMPTZ NOT NULL,
    review_at       TIMESTAMPTZ,
    conditions      JSONB NOT NULL DEFAULT '[]',
    prohibited_features JSONB NOT NULL DEFAULT '[]',
    approved_features JSONB NOT NULL DEFAULT '[]',
    document_checksum VARCHAR(64),
    approval_status  VARCHAR(30) NOT NULL DEFAULT 'PENDING',
    -- PENDING, APPROVED, REJECTED, EXPIRED
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Backtest/Walk-Forward/Validation Tables (SOW Section 63A)
-- ============================================================
CREATE TABLE research.backtest_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     VARCHAR(50) NOT NULL,
    strategy_definition_id UUID,
    exit_profile_id UUID,
    broker_profile_id UUID,
    dataset_id      VARCHAR(100) NOT NULL,
    start_date      DATE NOT NULL,
    end_date        DATE NOT NULL,
    total_trades    INTEGER NOT NULL DEFAULT 0,
    win_rate        DECIMAL(8,4),
    profit_factor   DECIMAL(10,4),
    expectancy      DECIMAL(10,4),
    avg_rr          DECIMAL(10,4),
    max_drawdown    DECIMAL(8,4),
    mfe             DECIMAL(18,8),
    mae             DECIMAL(18,8),
    cost_model_version VARCHAR(50),
    slippage_version VARCHAR(50),
    code_commit     VARCHAR(40),
    config_version  VARCHAR(50),
    is_oos          BOOLEAN NOT NULL DEFAULT FALSE,
    report_path     TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE research.walk_forward_runs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     VARCHAR(50) NOT NULL,
    strategy_definition_id UUID,
    train_start     DATE NOT NULL,
    train_end       DATE NOT NULL,
    test_start      DATE NOT NULL,
    test_end        DATE NOT NULL,
    fold_number     INTEGER NOT NULL,
    in_sample_metrics JSONB NOT NULL DEFAULT '{}',
    out_of_sample_metrics JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE research.validation_reports (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id     VARCHAR(50) NOT NULL,
    strategy_definition_id UUID,
    report_type     VARCHAR(50) NOT NULL,
    -- BACKTEST, WALK_FORWARD, OOS, PAPER, SHADOW, CALIBRATION, DRIFT
    metrics         JSONB NOT NULL DEFAULT '{}',
    approved        BOOLEAN NOT NULL DEFAULT FALSE,
    approved_by     UUID REFERENCES iam.users(id),
    approved_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE research.promotion_approvals (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_definition_id UUID NOT NULL REFERENCES trading.strategy_definitions(id),
    from_status     VARCHAR(30) NOT NULL,
    to_status       VARCHAR(30) NOT NULL,
    approved_by     UUID NOT NULL REFERENCES iam.users(id),
    reason          TEXT NOT NULL,
    evidence_report_ids JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- ============================================================
-- Audit Table (SOW Section 65 — append-only)
-- ============================================================
CREATE TABLE audit.audit_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id        UUID NOT NULL DEFAULT gen_random_uuid(),
    actor_type      VARCHAR(30) NOT NULL,
    -- USER, ADMIN, SYSTEM, API
    actor_id        UUID,
    tenant_id       UUID,
    action          VARCHAR(100) NOT NULL,
    entity_type     VARCHAR(50) NOT NULL,
    entity_id       UUID,
    request_id      VARCHAR(100),
    timestamp       TIMESTAMPTZ NOT NULL DEFAULT now(),
    source_ip       INET,
    user_agent      TEXT,
    old_value       JSONB,
    new_value       JSONB,
    reason          TEXT,
    correlation_id  UUID
);

CREATE INDEX idx_audit_actor ON audit.audit_events(actor_id);
CREATE INDEX idx_audit_action ON audit.audit_events(action);
CREATE INDEX idx_audit_entity ON audit.audit_events(entity_type, entity_id);
CREATE INDEX idx_audit_timestamp ON audit.audit_events(timestamp DESC);
CREATE INDEX idx_audit_correlation ON audit.audit_events(correlation_id);

-- ============================================================
-- Support System (SOW Section 70)
-- ============================================================
CREATE TABLE support.support_tickets (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    subject         VARCHAR(255) NOT NULL,
    category        VARCHAR(50) NOT NULL,
    -- BILLING, TECHNICAL, REFERRAL, COMMISSION, PAYOUT, MT4, MT5, GENERAL
    priority        VARCHAR(20) NOT NULL DEFAULT 'MEDIUM',
    -- LOW, MEDIUM, HIGH, URGENT
    status          VARCHAR(20) NOT NULL DEFAULT 'OPEN',
    -- OPEN, IN_PROGRESS, WAITING_CUSTOMER, RESOLVED, CLOSED
    assigned_to     UUID REFERENCES iam.users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    closed_at       TIMESTAMPTZ
);

CREATE TABLE support.support_messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    ticket_id       UUID NOT NULL REFERENCES support.support_tickets(id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES iam.users(id),
    message         TEXT NOT NULL,
    is_internal     BOOLEAN NOT NULL DEFAULT FALSE,
    attachments     JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_support_messages_ticket ON support.support_messages(ticket_id);
