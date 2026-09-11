-- 145_hypertable_retention_policies.sql
-- Production hardening (audit P2-12): the following hypertables were created
-- WITHOUT compression/retention policies, so they grow unbounded forever:
--   * trading.regime_history
--   * public.devil_liquidity_events
--   * trading.strategy_evaluations
--   * market.candles  (compression existed, but NO retention policy → audit
--     flagged candles retention drift; we standardize on 1095 days / 3y)
-- TimescaleDB 2.29 requires compression to be ENABLED on the hypertable
-- (ALTER TABLE ... SET (timescaledb.compress)) before add_compression_policy.
-- Idempotent: enabled/added only when currently absent. Each table handled in
-- its own block so a single missing table can't abort the rest.

DO $$
BEGIN
    -- ─── trading.regime_history ────────────────────────────────────────────
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM timescaledb_information.compression_settings
            WHERE hypertable_name = 'regime_history'
        ) THEN
            EXECUTE 'ALTER TABLE trading.regime_history SET (timescaledb.compress)';
        END IF;
        PERFORM add_compression_policy('trading.regime_history', INTERVAL '7 days', if_not_exists => TRUE);
        PERFORM add_retention_policy('trading.regime_history', INTERVAL '1095 days', if_not_exists => TRUE);
    EXCEPTION WHEN OTHERS THEN
        RAISE NOTICE 'regime_history policy skipped: %', SQLERRM;
    END;

    -- ─── public.devil_liquidity_events ────────────────────────────────────
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM timescaledb_information.compression_settings
            WHERE hypertable_name = 'devil_liquidity_events'
        ) THEN
            EXECUTE 'ALTER TABLE public.devil_liquidity_events SET (timescaledb.compress)';
        END IF;
        PERFORM add_compression_policy('public.devil_liquidity_events', INTERVAL '7 days', if_not_exists => TRUE);
        PERFORM add_retention_policy('public.devil_liquidity_events', INTERVAL '1095 days', if_not_exists => TRUE);
    EXCEPTION WHEN OTHERS THEN
        RAISE NOTICE 'devil_liquidity_events policy skipped: %', SQLERRM;
    END;

    -- ─── trading.strategy_evaluations ──────────────────────────────────────
    BEGIN
        IF NOT EXISTS (
            SELECT 1 FROM timescaledb_information.compression_settings
            WHERE hypertable_name = 'strategy_evaluations'
        ) THEN
            EXECUTE 'ALTER TABLE trading.strategy_evaluations SET (timescaledb.compress)';
        END IF;
        PERFORM add_compression_policy('trading.strategy_evaluations', INTERVAL '7 days', if_not_exists => TRUE);
        PERFORM add_retention_policy('trading.strategy_evaluations', INTERVAL '1095 days', if_not_exists => TRUE);
    EXCEPTION WHEN OTHERS THEN
        RAISE NOTICE 'strategy_evaluations policy skipped: %', SQLERRM;
    END;

    -- ─── market.candles ────────────────────────────────────────────────────
    -- Compression already configured (migration 005). Ensure retention exists.
    BEGIN
        PERFORM add_retention_policy('market.candles', INTERVAL '1095 days', if_not_exists => TRUE);
    EXCEPTION WHEN OTHERS THEN
        RAISE NOTICE 'candles retention skipped: %', SQLERRM;
    END;

    RAISE NOTICE 'hypertable retention/compression policies applied (145)';
END
$$;
