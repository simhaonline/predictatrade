-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 133: account type detection & strategy adaptation tables
--
-- CAccountTypeDetector (MQL edge) reports the detected account type per
-- device/account; the platform stores it here for traceability and for
-- type-specific strategy parameter lookup. The EA-side detector is the
-- source of truth (it sees ACCOUNT_TRADE_MODE, swaps, min-lot, execution
-- mode and commission that server-side data cannot observe).
--
-- Schema matches the platform spec:
--   account_types(id, account_login, detected_type, detection_timestamp,
--                 confirmation_count, is_verified, strategy_override)
--   strategy_parameters(id, account_type, parameter_name, parameter_value,
--                       effective_from, priority)
-- Postgres port: SERIAL ids, BIGINT logins, timestamptz datetimes.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE TABLE IF NOT EXISTS licensing.account_types (
    id                 SERIAL PRIMARY KEY,
    account_login      BIGINT        NOT NULL,
    detected_type      VARCHAR(50)   NOT NULL,
    detection_timestamp TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    confirmation_count INTEGER       NOT NULL DEFAULT 0,
    is_verified        BOOLEAN       NOT NULL DEFAULT FALSE,
    strategy_override  VARCHAR(50),
    -- Platform integration columns (additive; EAs never send these)
    device_id          UUID,
    detection_reason   TEXT,
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_account_types_login
    ON licensing.account_types (account_login);
CREATE UNIQUE INDEX IF NOT EXISTS uq_account_types_login_latest
    ON licensing.account_types (account_login, detection_timestamp DESC);

CREATE TABLE IF NOT EXISTS licensing.strategy_parameters (
    id             SERIAL PRIMARY KEY,
    account_type   VARCHAR(50)   NOT NULL,
    parameter_name VARCHAR(100)  NOT NULL,
    parameter_value TEXT         NOT NULL,
    effective_from TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    priority       INTEGER       NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_strategy_parameters_type_prio
    ON licensing.strategy_parameters (account_type, priority DESC);

-- Baseline strategy parameters per account type (spec §math).
-- Inserted once; future changes version through new rows with higher priority.
INSERT INTO licensing.strategy_parameters (account_type, parameter_name, parameter_value, priority)
SELECT v.atype, v.pname, v.pval, v.prio
FROM (VALUES
    ('MicroCent', 'lot_scale_divisor',          '100',   100),
    ('MicroCent', 'pip_value_adjustment',       '0.01',  100),
    ('MicroCent', 'slippage_buffer_points',     '1',     100),
    ('ECN',       'commission_round_trip_mult', '2',     100),
    ('ECN',       'rr_commission_erosion',      'true',  100),
    ('ECN',       'order_mode',                 'open_then_modify_sltp', 100),
    ('STP',       'slippage_buffer_pips',       '2',     100),
    ('Islamic',   'swap_in_pnl',                '0',     100),
    ('Islamic',   'swap_projection_disabled',   'true',  100),
    ('Demo',      'simulation_mode',            'true',  100),
    ('Demo',      'signal_flag',                'demo',  100),
    ('Standard',  'model',                      'baseline', 100)
) AS v(atype, pname, pval, prio)
WHERE NOT EXISTS (
    SELECT 1 FROM licensing.strategy_parameters sp
    WHERE sp.account_type = v.atype AND sp.parameter_name = v.pname
);

-- Detection history audit table comment
COMMENT ON TABLE licensing.account_types IS
    'CAccountTypeDetector results per account login (EA-reported; priority: Demo>Contest>Islamic>MicroCent>ECN>STP>Standard)';
COMMENT ON TABLE licensing.strategy_parameters IS
    'Per-account-type strategy math parameters; highest priority row wins for (account_type, parameter_name)';