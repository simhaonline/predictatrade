-- 098_broker_account_types.sql — check.md 2026-08-30: playbook §8 broker account structure layer
CREATE TABLE IF NOT EXISTS licensing.broker_account_types (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code                  VARCHAR(30)  NOT NULL UNIQUE,
    label                 VARCHAR(80)  NOT NULL,
    execution_model       VARCHAR(30)  NOT NULL,
    typical_spread_pips   NUMERIC(6,2) NOT NULL DEFAULT 1.40,
    commission_per_side   NUMERIC(6,2) NOT NULL DEFAULT 0,
    commission_per_lot_r  NUMERIC(6,2) NOT NULL DEFAULT 0,
    min_deposit           NUMERIC(10,2) NOT NULL DEFAULT 0,
    mt4_supported         BOOLEAN NOT NULL DEFAULT true,
    mt5_supported         BOOLEAN NOT NULL DEFAULT true,
    webtrader             BOOLEAN NOT NULL DEFAULT false,
    best_for              VARCHAR(50),
    strategy_suitability  JSONB NOT NULL DEFAULT '{}',
    is_active             BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO licensing.broker_account_types (code, label, execution_model, typical_spread_pips, commission_per_side, commission_per_lot_r, min_deposit, mt4_supported, mt5_supported, webtrader, best_for, strategy_suitability) VALUES
  ('ecn_raw', 'Premier / Raw / ECN', 'ecn', 0.0, 3.5, 20, 3000, true, true, false, 'scalping', '{"ULTRA_SCALPING":"best","STANDARD_SCALPING":"best","STANDARD_SWING":"excellent","TREND_SWING":"excellent"}'),
  ('stp_standard', 'Standard / STP', 'stp', 1.4, 0, 15, 0, true, true, true, 'beginner', '{"ULTRA_SCALPING":"borderline","STANDARD_SCALPING":"acceptable","STANDARD_SWING":"good","TREND_SWING":"good"}'),
  ('dealing_desk', 'Classic / Dealing Desk', 'dealing_desk', 2.0, 0, 25, 0, false, true, false, 'beginner', '{"ULTRA_SCALPING":"avoid","STANDARD_SCALPING":"borderline","STANDARD_SWING":"good","TREND_SWING":"good"}'),
  ('micro_cent', 'Micro / Cent', 'micro', 1.6, 0, 20, 0, true, true, false, 'practice', '{"ULTRA_SCALPING":"practice_only","STANDARD_SCALPING":"practice_only","STANDARD_SWING":"good_for_testing","TREND_SWING":"good_for_testing"}'),
  ('islamic_swapfree', 'Islamic / Swap-Free', 'swapfree', 1.4, 0, 15, 0, true, true, false, 'overnight', '{"ULTRA_SCALPING":"irrelevant","STANDARD_SCALPING":"irrelevant","STANDARD_SWING":"very_useful","TREND_SWING":"strong_fit"}'),
  ('demo', 'Demo', 'demo', 1.4, 0, 0, 0, true, true, true, 'practice', '{"ULTRA_SCALPING":"practice","STANDARD_SCALPING":"practice","STANDARD_SWING":"practice","TREND_SWING":"practice"}'),
  ('institutional', 'Institutional / Prime', 'institutional', 0.0, 0, 0, 0, true, true, true, 'all', '{"ULTRA_SCALPING":"best","STANDARD_SCALPING":"best","STANDARD_SWING":"excellent","TREND_SWING":"excellent"}')
ON CONFLICT (code) DO NOTHING;

CREATE TABLE IF NOT EXISTS licensing.strategy_cost_gates (
    id                    UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    strategy_id           VARCHAR(50) NOT NULL,
    broker_account_code   VARCHAR(30) NOT NULL REFERENCES licensing.broker_account_types(code),
    cost_as_pct_of_1r     NUMERIC(5,2) NOT NULL,
    suitability           VARCHAR(20)  NOT NULL,
    allowed               BOOLEAN NOT NULL DEFAULT true,
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (strategy_id, broker_account_code)
);

INSERT INTO licensing.strategy_cost_gates (strategy_id, broker_account_code, cost_as_pct_of_1r, suitability, allowed) VALUES
  ('ULTRA_SCALPING',    'ecn_raw',           20, 'best',        true),
  ('ULTRA_SCALPING',    'stp_standard',      35, 'borderline',  false),
  ('ULTRA_SCALPING',    'dealing_desk',      50, 'avoid',       false),
  ('ULTRA_SCALPING',    'micro_cent',        40, 'practice_only', false),
  ('ULTRA_SCALPING',    'islamic_swapfree',  20, 'irrelevant',  true),
  ('ULTRA_SCALPING',    'demo',              20, 'practice',    true),
  ('ULTRA_SCALPING',    'institutional',     10, 'best',        true),
  ('STANDARD_SCALPING', 'ecn_raw',           15, 'best',        true),
  ('STANDARD_SCALPING', 'stp_standard',      30, 'borderline',  false),
  ('STANDARD_SCALPING', 'dealing_desk',      40, 'avoid',       false),
  ('STANDARD_SCALPING', 'micro_cent',        40, 'practice_only', false),
  ('STANDARD_SCALPING', 'islamic_swapfree',  15, 'irrelevant',  true),
  ('STANDARD_SCALPING', 'demo',              15, 'practice',    true),
  ('STANDARD_SCALPING', 'institutional',     10, 'best',        true),
  ('STANDARD_SWING',    'ecn_raw',            8, 'excellent',   true),
  ('STANDARD_SWING',    'stp_standard',      15, 'good',        true),
  ('STANDARD_SWING',    'dealing_desk',      20, 'good',        true),
  ('STANDARD_SWING',    'micro_cent',        20, 'good_for_testing', true),
  ('STANDARD_SWING',    'islamic_swapfree',  10, 'very_useful', true),
  ('STANDARD_SWING',    'demo',              15, 'practice',    true),
  ('STANDARD_SWING',    'institutional',      8, 'excellent',   true),
  ('TREND_SWING',       'ecn_raw',            5, 'strong',      true),
  ('TREND_SWING',       'stp_standard',      10, 'good_default', true),
  ('TREND_SWING',       'dealing_desk',      15, 'acceptable',  true),
  ('TREND_SWING',       'micro_cent',        15, 'practice_only', true),
  ('TREND_SWING',       'islamic_swapfree',  10, 'strong_fit',  true),
  ('TREND_SWING',       'demo',              10, 'practice',    true),
  ('TREND_SWING',       'institutional',      5, 'strong',      true)
ON CONFLICT (strategy_id, broker_account_code) DO NOTHING;

ALTER TABLE licensing.mt_accounts ADD COLUMN IF NOT EXISTS account_type_code VARCHAR(30);
ALTER TABLE licensing.mt_accounts ADD COLUMN IF NOT EXISTS account_leverage INT;
ALTER TABLE licensing.mt_accounts ADD COLUMN IF NOT EXISTS is_swap_free BOOLEAN DEFAULT false;
ALTER TABLE licensing.device_activations ADD COLUMN IF NOT EXISTS account_type_code VARCHAR(30);
