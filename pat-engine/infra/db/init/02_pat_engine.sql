-- pat-engine schema part 2: reference + user/license/device/risk data.
-- Static reference data vs dynamic time-series/telemetry is explicitly separated so
-- the frontend never mixes config with high-volume streams.

-- STATIC: per-symbol broker execution economics (reference, rarely changes).
CREATE TABLE IF NOT EXISTS broker_execution_profiles (
    symbol           text PRIMARY KEY,
    digits           int,
    tick_size        double precision,
    tick_value       double precision,
    contract_size    double precision,
    min_lot          double precision,
    max_lot          double precision,
    lot_step         double precision,
    stops_level      int,
    freeze_level     int,
    swap_long        double precision,
    swap_short       double precision,
    commission_per_lot double precision,
    typical_spread   double precision,
    leverage         double precision,
    timezone_offset  int,
    rollover_hour    int,
    sessions         jsonb
);

-- STATIC-ish: plan-level risk mandates (operator/control-plane owned).
CREATE TABLE IF NOT EXISTS risk_config (
    plan                text PRIMARY KEY,
    max_daily_loss_pct  double precision,
    max_positions       int,
    risk_per_trade_pct  double precision,
    max_leverage        double precision,
    min_rr              double precision,
    allowed_strategies  jsonb
);

-- STATIC: end users (control-plane owned; mirrored here for local queries).
CREATE TABLE IF NOT EXISTS users (
    id         text PRIMARY KEY,
    email      text,
    full_name  text,
    status     text,
    created_at timestamptz DEFAULT now()
);

-- STATIC-ish: license entitlements (control-plane owned; mirrored for gating).
CREATE TABLE IF NOT EXISTS licenses (
    license_key     text PRIMARY KEY,
    user_id         text,
    plan            text,
    allowed_strategies jsonb,
    max_devices     int,
    status          text,
    expires_at      timestamptz,
    device_binding  boolean DEFAULT true
);

-- DYNAMIC-ish: registered devices (hardware fingerprint binding for license misuse).
CREATE TABLE IF NOT EXISTS devices (
    id          text PRIMARY KEY,
    license_id  text,
    fingerprint_hash text,
    fingerprint_components jsonb,
    installation_id text,
    hostname    text,
    os          text,
    last_seen   timestamptz DEFAULT now()
);

-- DYNAMIC, high-volume: per-heartbeat agent telemetry (CPU/RAM/connection/latency).
CREATE TABLE IF NOT EXISTS device_telemetry (
    device_id   text NOT NULL,
    ts          timestamptz NOT NULL,
    latency_ms  double precision,
    mt4_conn    boolean,
    mt5_conn    boolean,
    broker      text,
    account_masked text,
    equity      double precision,
    balance     double precision,
    open_positions int,
    floating_pnl double precision,
    cpu_pct     double precision,
    ram_pct     double precision,
    version     text,
    status      text
);
SELECT create_hypertable('device_telemetry', 'ts', if_not_exists => TRUE);

-- DYNAMIC: executed/closed trades ledger (P&L, swap, commission, reason).
CREATE TABLE IF NOT EXISTS positions (
    id           text PRIMARY KEY,
    user_id      text,
    device_id    text,
    strategy_id  text,
    symbol       text,
    side         text,
    open_ts      timestamptz,
    open_price   double precision,
    sl           double precision,
    tp           double precision,
    lot          double precision,
    close_ts     timestamptz,
    close_price  double precision,
    pnl          double precision,
    swap         double precision,
    commission   double precision,
    reason       text
);

-- DYNAMIC: equity snapshots for drawdown tracking.
CREATE TABLE IF NOT EXISTS equity_snapshots (
    user_id   text NOT NULL,
    ts        timestamptz NOT NULL,
    equity    double precision,
    balance   double precision,
    drawdown_pct double precision
);
SELECT create_hypertable('equity_snapshots', 'ts', if_not_exists => TRUE);

CREATE INDEX IF NOT EXISTS idx_device_telemetry_device_ts ON device_telemetry (device_id, ts DESC);
CREATE INDEX IF NOT EXISTS idx_positions_user_ts ON positions (user_id, open_ts DESC);

-- FUNCTIONS: common frontend/API aggregations (avoid N+1 and app-side math).
CREATE OR REPLACE FUNCTION daily_pnl(p_user text, p_day date)
RETURNS double precision AS $$
    SELECT COALESCE(SUM(pnl), 0) FROM positions
    WHERE user_id = p_user AND close_ts::date = p_day AND close_ts IS NOT NULL;
$$ LANGUAGE sql STABLE;

CREATE OR REPLACE FUNCTION account_daily_loss_pct(p_user text, p_day date)
RETURNS double precision AS $$
DECLARE
    start_eq double precision;
    end_eq   double precision;
BEGIN
    SELECT equity INTO start_eq FROM equity_snapshots
        WHERE user_id = p_user AND ts::date = p_day ORDER BY ts ASC LIMIT 1;
    SELECT equity INTO end_eq FROM equity_snapshots
        WHERE user_id = p_user AND ts::date = p_day ORDER BY ts DESC LIMIT 1;
    IF start_eq IS NULL OR start_eq = 0 THEN RETURN 0; END IF;
    RETURN (end_eq - start_eq) / start_eq * 100;
END;
$$ LANGUAGE plpgsql STABLE;

CREATE OR REPLACE FUNCTION signal_counts(p_strategy text, p_day date)
RETURNS TABLE(status text, cnt bigint) AS $$
    SELECT s.status, count(*) FROM signals s
    WHERE s.strategy_id = p_strategy AND s.ts::date = p_day
    GROUP BY s.status;
$$ LANGUAGE sql STABLE;
