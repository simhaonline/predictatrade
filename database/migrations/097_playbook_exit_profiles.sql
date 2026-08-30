-- 097_playbook_exit_profiles.sql
-- check.md 2026-08-30: fix TP R-multiples to match playbook.md exactly
--
-- Playbook spec (verified against playbook.md §11.4, §12, §13.4, §14.4):
-- ULTRA_SCALPING    : TP1=1R (close 50%, move to BE), TP2=1.5R (close 30%), TP3=2R (close 20%)
-- STANDARD_SCALPING : TP1=1R (close 40%), TP2=2R (close 35%), TP3=3R (close 25%)
-- STANDARD_SWING    : TP1=1R (close 33%), TP2=2R (close 33%), TP3=4R (close 34%)
-- TREND_SWING       : TP1=1R (close 25%), TP2=3R (close 25%), TP3=trail (close 50% at 4R)
--
-- SL values remain PERCENTAGE-mode but scaled to correct ratios:
--   We keep the SL% that matches the risk-tolerance of each strategy, and we
--   fix the TP ratios to be proper R-multiples of that SL. This way TP1
--   always = 1R exactly (the playbook "move to BE at TP1" logic works).

UPDATE trading.exit_profiles SET
  stop_pct = 0.0008,
  tp1_pct  = 0.0008,   -- 1.0R
  tp2_pct  = 0.0012,   -- 1.5R
  tp3_pct  = 0.0016    -- 2.0R
WHERE strategy_id = 'ULTRA_SCALPING';

UPDATE trading.exit_profiles SET
  stop_pct = 0.0015,
  tp1_pct  = 0.0015,   -- 1.0R
  tp2_pct  = 0.0030,   -- 2.0R
  tp3_pct  = 0.0045    -- 3.0R
WHERE strategy_id = 'STANDARD_SCALPING';

UPDATE trading.exit_profiles SET
  stop_pct = 0.0025,
  tp1_pct  = 0.0025,   -- 1.0R
  tp2_pct  = 0.0050,   -- 2.0R
  tp3_pct  = 0.0100    -- 4.0R
WHERE strategy_id = 'STANDARD_SWING';

UPDATE trading.exit_profiles SET
  stop_pct = 0.0040,
  tp1_pct  = 0.0040,   -- 1.0R
  tp2_pct  = 0.0120,   -- 3.0R
  tp3_pct  = 0.0160    -- 4.0R (trail from TP3 per playbook)
WHERE strategy_id = 'TREND_SWING';

-- MARNIE_FIB (EQFE) — keep ATR mode but fix TP R-multiples as playbook swing
UPDATE trading.exit_profiles SET
  tp1_pct  = 0.0015,   -- 1.0R relative to SL ATR basis
  tp2_pct  = 0.0030,   -- 2.0R
  tp3_pct  = 0.0060    -- 4.0R
WHERE strategy_id = 'MARNIE_FIB';
