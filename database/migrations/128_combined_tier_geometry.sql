-- Combined tier-geometry model (v1.25, user-approved a+b+c, 2026-09-03).
--
-- Problem (verified 2026-09-03): swing TP1 == SL distance (1:1 R:R) makes
-- swing EV-negative at wr <= 0.5 (the profitability gate vetoed most strong
-- swing reads as loss candidates), and the 0.25% SL made every wide-ATR
-- swing PRO-only (min-lot risk > $10 STANDARD cap) — the fleet got zero
-- swing signals after 08:04 UTC.
--
-- Fix = tighten stops AND widen TP1 (better R:R AND smaller min-lot risk
-- AND more signals pass the profitability gate), plus higher tier caps so
-- the tightened geometry reaches MICRO/STANDARD.
--
-- STANDARD_SWING: 0.25/0.25/0.50/1.00 → 0.18/0.40/0.70/1.20
--   netRR (TP1 vs SL, cost 0.50): 0.91 → 2.03  (MinRR 2.0 gate satisfied)
--   EV at modelled wr 0.63/RANGE: +2.41 → +7.84 per 1R
--   min-lot risk at $4,485: 11.2pts → 8.1pts ($8.07 ≤ $25 STANDARD cap)
-- TREND_SWING: 0.40/0.40/1.20/1.60 → 0.30/0.65/1.10/1.50
--   netRR: 0.95 → 2.05
--   min-lot risk: 17.9pts → 13.5pts (≤ $25 STANDARD cap)
--
-- Tier caps changed in Go (capitaltier): MICRO 2%→4%, STANDARD 2%→5%,
-- PRO 2% unchanged. Effective per-trade cap stays min(plan cap, tier cap).

UPDATE trading.exit_profiles
SET stop_pct = 0.0018,
    tp1_pct = 0.0040,
    tp2_pct = 0.0070,
    tp3_pct = 0.0120,
    change_reason = 'v1.25 combined tier model: tighten SL 0.25->0.18, widen TP1 0.25->0.40 (netRR 0.91->2.03); unblocks STANDARD tier + profitability gate'
WHERE strategy_id = 'STANDARD_SWING';

UPDATE trading.exit_profiles
SET stop_pct = 0.0030,
    tp1_pct = 0.0065,
    tp2_pct = 0.0110,
    tp3_pct = 0.0150,
    change_reason = 'v1.25 combined tier model: tighten SL 0.40->0.30, widen TP1 0.40->0.65 (netRR 0.95->2.05)'
WHERE strategy_id = 'TREND_SWING';

SELECT strategy_id, version, status, calculation_mode, stop_pct, tp1_pct, tp2_pct, tp3_pct
FROM trading.exit_profiles
WHERE strategy_id IN ('STANDARD_SWING', 'TREND_SWING');