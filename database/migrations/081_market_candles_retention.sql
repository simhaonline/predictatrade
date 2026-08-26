-- ============================================================================
-- 081_market_candles_retention.sql
-- Predict-A-Trade v1.16.0 — Candle Retention Policy
--
-- Adds automated retention to the market.candles hypertable to prevent
-- unbounded storage growth. All other hypertables already have retention.
--
-- Policy: 3 years (1095 days) — balances backtest history needs against
-- storage costs. XAUUSD M1 data at ~40KB/row ≈ ~21M rows/year ≈ 840MB/year.
-- 3-year window caps storage at ~2.5GB for candles.
-- ============================================================================

SELECT add_retention_policy('market.candles', INTERVAL '1095 days', if_not_exists => TRUE);

-- ============================================================================
-- Verify: all market/audit/trading hypertables now have retention policies
-- ============================================================================
COMMENT ON POLICY retention_policy ON market.candles IS
    'Automated 3-year candle retention (v1.16.0 P2 closure: SOW data-integrity gap D3)';
