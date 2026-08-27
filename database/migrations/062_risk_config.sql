-- LEGACY DUPLICATE PREFIX 062: shares its numeric prefix with another migration file. Tolerated legacy collision (see database/migrations/MIGRATION_ORDER.md); DO NOT rename (rename risks re-applying applied schema). The CI guard scripts/check_migrations.sh blocks NEW duplicate prefixes only.
-- 062: Persistent global risk configuration for the admin Risk Center.
-- Stores kill switches, numeric risk limits and session/news blackout flags
-- so that "Save Risk Config" is durable instead of local-only/preview.

CREATE SCHEMA IF NOT EXISTS control;

CREATE TABLE IF NOT EXISTS control.risk_config (
  id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
  config_key       VARCHAR(50) UNIQUE NOT NULL,
  kill_switches    JSONB NOT NULL DEFAULT '{}'::jsonb,
  limits           JSONB NOT NULL DEFAULT '{}'::jsonb,
  session_blackout BOOLEAN DEFAULT FALSE,
  news_blackout    BOOLEAN DEFAULT FALSE,
  blackout_reason  TEXT,
  updated_by       UUID,
  updated_at       TIMESTAMPTZ DEFAULT now()
);

-- One canonical GLOBAL row. Limits defaults align with the admin UI numeric fields.
INSERT INTO control.risk_config (config_key, kill_switches, limits, session_blackout, news_blackout)
VALUES (
  'GLOBAL',
  '{"strategy": false, "account": false, "broker": false, "symbol": false}'::jsonb,
  '{"max_exposure": 100000, "max_spread": 5.0, "max_slippage": 2.0, "max_drawdown": 15, "max_daily_loss": 5}'::jsonb,
  FALSE,
  FALSE
)
ON CONFLICT (config_key) DO NOTHING;
