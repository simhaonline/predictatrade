-- Migration 019: Percentage-based SL/TP configuration (database-driven, no recompile needed)
-- Replaces hardcoded ATR multipliers with configurable percentages.
-- SL/TP = Entry × (1 ± percentage), with ATR as guardrails.

-- ─── Add percentage columns to exit_profiles ───
ALTER TABLE trading.exit_profiles 
    ADD COLUMN IF NOT EXISTS stop_pct        NUMERIC(6,4) NOT NULL DEFAULT 0.0015,  -- 0.15% default SL
    ADD COLUMN IF NOT EXISTS tp1_pct         NUMERIC(6,4) NOT NULL DEFAULT 0.0015,  -- 0.15% default TP1
    ADD COLUMN IF NOT EXISTS tp2_pct         NUMERIC(6,4) NOT NULL DEFAULT 0.0025,  -- 0.25% default TP2
    ADD COLUMN IF NOT EXISTS tp3_pct         NUMERIC(6,4) NOT NULL DEFAULT 0.0035,  -- 0.35% default TP3
    ADD COLUMN IF NOT EXISTS min_stop_atr_mult  NUMERIC(4,2) NOT NULL DEFAULT 0.50,  -- SL must be >= 0.5×ATR
    ADD COLUMN IF NOT EXISTS max_stop_atr_mult  NUMERIC(4,2) NOT NULL DEFAULT 3.00,  -- SL must be <= 3.0×ATR
    ADD COLUMN IF NOT EXISTS min_tp1_atr_mult   NUMERIC(4,2) NOT NULL DEFAULT 0.50,  -- TP1 must be >= 0.5×ATR
    ADD COLUMN IF NOT EXISTS max_tp1_atr_mult   NUMERIC(4,2) NOT NULL DEFAULT 3.00,  -- TP1 must be <= 3.0×ATR
    ADD COLUMN IF NOT EXISTS calculation_mode   VARCHAR(10) NOT NULL DEFAULT 'PERCENTAGE'; -- PERCENTAGE or ATR

-- ─── Insert default exit profiles for all 4 strategies ───
-- STANDARD_SCALPING: tight scalping, small percentages
INSERT INTO trading.exit_profiles (strategy_id, version, stop_model, calculation_mode,
    stop_pct, tp1_pct, tp2_pct, tp3_pct,
    min_stop_atr_mult, max_stop_atr_mult, min_tp1_atr_mult, max_tp1_atr_mult,
    tp1_selection_policy, tp2_selection_policy, tp3_selection_policy,
    tp1_fraction, tp2_fraction, tp3_fraction,
    breakeven_trigger, trailing_trigger, trailing_model,
    effective_from, change_reason)
VALUES ('STANDARD_SCALPING', '1.0.0', 'PERCENTAGE', 'PERCENTAGE',
    0.0015, 0.0015, 0.0025, 0.0035,   -- SL=0.15%, TP1=0.15%, TP2=0.25%, TP3=0.35%
    0.50, 2.00, 0.50, 2.00,            -- ATR guardrails: min 0.5x, max 2x
    'NEAREST_LIQUIDITY', 'NEXT_LIQUIDITY', 'PROFILE_OBJECTIVE',
    0.50, 0.30, 0.20,
    'AFTER_TP1_FILL', 'AFTER_TP2', 'ATR_TRAIL',
    now(), 'Initial percentage-based config')
ON CONFLICT DO NOTHING;

-- ULTRA_SCALPING: very tight ultra-fast scalping
INSERT INTO trading.exit_profiles (strategy_id, version, stop_model, calculation_mode,
    stop_pct, tp1_pct, tp2_pct, tp3_pct,
    min_stop_atr_mult, max_stop_atr_mult, min_tp1_atr_mult, max_tp1_atr_mult,
    tp1_selection_policy, tp2_selection_policy, tp3_selection_policy,
    tp1_fraction, tp2_fraction, tp3_fraction,
    breakeven_trigger, trailing_trigger, trailing_model,
    effective_from, change_reason)
VALUES ('ULTRA_SCALPING', '1.0.0', 'PERCENTAGE', 'PERCENTAGE',
    0.0008, 0.0008, 0.0012, 0.0016,   -- SL=0.08%, TP1=0.08%, TP2=0.12%, TP3=0.16%
    0.30, 1.50, 0.30, 1.50,            -- ATR guardrails: tighter for ultra
    'NEAREST_LIQUIDITY', 'NEXT_LIQUIDITY', 'PROFILE_OBJECTIVE',
    0.50, 0.30, 0.20,
    'AFTER_TP1_FILL', 'AFTER_TP1', 'ATR_TRAIL',
    now(), 'Initial percentage-based config')
ON CONFLICT DO NOTHING;

-- STANDARD_SWING: medium-term swing trading
INSERT INTO trading.exit_profiles (strategy_id, version, stop_model, calculation_mode,
    stop_pct, tp1_pct, tp2_pct, tp3_pct,
    min_stop_atr_mult, max_stop_atr_mult, min_tp1_atr_mult, max_tp1_atr_mult,
    tp1_selection_policy, tp2_selection_policy, tp3_selection_policy,
    tp1_fraction, tp2_fraction, tp3_fraction,
    breakeven_trigger, trailing_trigger, trailing_model,
    effective_from, change_reason)
VALUES ('STANDARD_SWING', '1.0.0', 'PERCENTAGE', 'PERCENTAGE',
    0.0030, 0.0030, 0.0055, 0.0080,   -- SL=0.30%, TP1=0.30%, TP2=0.55%, TP3=0.80%
    1.00, 3.00, 1.00, 3.00,            -- ATR guardrails: wider for swing
    'NEAREST_LIQUIDITY', 'NEXT_LIQUIDITY', 'PROFILE_OBJECTIVE',
    0.40, 0.35, 0.25,
    'AFTER_TP1_FILL', 'AFTER_TP2', 'ATR_TRAIL',
    now(), 'Initial percentage-based config')
ON CONFLICT DO NOTHING;

-- TREND_SWING: long-term trend following
INSERT INTO trading.exit_profiles (strategy_id, version, stop_model, calculation_mode,
    stop_pct, tp1_pct, tp2_pct, tp3_pct,
    min_stop_atr_mult, max_stop_atr_mult, min_tp1_atr_mult, max_tp1_atr_mult,
    tp1_selection_policy, tp2_selection_policy, tp3_selection_policy,
    tp1_fraction, tp2_fraction, tp3_fraction,
    breakeven_trigger, trailing_trigger, trailing_model,
    effective_from, change_reason)
VALUES ('TREND_SWING', '1.0.0', 'PERCENTAGE', 'PERCENTAGE',
    0.0050, 0.0050, 0.0100, 0.0150,   -- SL=0.50%, TP1=0.50%, TP2=1.00%, TP3=1.50%
    1.50, 4.00, 1.50, 4.00,            -- ATR guardrails: widest for trend
    'NEAREST_LIQUIDITY', 'NEXT_LIQUIDITY', 'PROFILE_OBJECTIVE',
    0.30, 0.35, 0.35,
    'AFTER_TP1_FILL', 'AFTER_TP2', 'ATR_TRAIL',
    now(), 'Initial percentage-based config')
ON CONFLICT DO NOTHING;

-- ─── Add strategy config versions with full parameter set ───
INSERT INTO trading.strategy_config_versions (strategy_id, version, values, effective_from, change_reason)
VALUES 
('STANDARD_SCALPING', '1.0.0', 
 '{"min_confluence":65,"min_mtf_alignment":40,"min_adx":20,"min_rr":1.2,
  "sl_pct":0.0015,"tp1_pct":0.0015,"tp2_pct":0.0025,"tp3_pct":0.0035,
  "calculation_mode":"PERCENTAGE","min_stop_atr":0.5,"max_stop_atr":2.0,
  "accepted_regimes":["TRENDING_BULLISH","TRENDING_BEARISH","BREAKOUT","MEAN_REVERSION"],
  "accepted_sessions":["LONDON","NEW_YORK","OVERLAP","TOKYO"]}'::jsonb,
 now(), 'Initial percentage-based config'),
('ULTRA_SCALPING', '1.0.0',
 '{"min_confluence":65,"min_mtf_alignment":50,"min_adx":25,"min_rr":1.0,
  "sl_pct":0.0008,"tp1_pct":0.0008,"tp2_pct":0.0012,"tp3_pct":0.0016,
  "calculation_mode":"PERCENTAGE","min_stop_atr":0.3,"max_stop_atr":1.5,
  "accepted_regimes":["TRENDING_BULLISH","TRENDING_BEARISH","BREAKOUT"],
  "accepted_sessions":["LONDON","NEW_YORK","OVERLAP","TOKYO"]}'::jsonb,
 now(), 'Initial percentage-based config'),
('STANDARD_SWING', '1.0.0',
 '{"min_confluence":55,"min_mtf_alignment":30,"min_adx":18,"min_rr":1.8,
  "sl_pct":0.0030,"tp1_pct":0.0030,"tp2_pct":0.0055,"tp3_pct":0.0080,
  "calculation_mode":"PERCENTAGE","min_stop_atr":1.0,"max_stop_atr":3.0,
  "accepted_regimes":["TRENDING_BULLISH","TRENDING_BEARISH","BREAKOUT","RANGE"],
  "accepted_sessions":["LONDON","NEW_YORK","OVERLAP"]}'::jsonb,
 now(), 'Initial percentage-based config'),
('TREND_SWING', '1.0.0',
 '{"min_confluence":50,"min_mtf_alignment":25,"min_adx":15,"min_rr":2.5,
  "sl_pct":0.0050,"tp1_pct":0.0050,"tp2_pct":0.0100,"tp3_pct":0.0150,
  "calculation_mode":"PERCENTAGE","min_stop_atr":1.5,"max_stop_atr":4.0,
  "accepted_regimes":["TRENDING_BULLISH","TRENDING_BEARISH","BREAKOUT"],
  "accepted_sessions":["LONDON","NEW_YORK","OVERLAP"]}'::jsonb,
 now(), 'Initial percentage-based config')
ON CONFLICT DO NOTHING;
