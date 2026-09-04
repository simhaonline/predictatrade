-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 137: candle timeframe alias MN1 → MN
--
-- The Master EAs emit the MQL5 standard name "MN1" (ENUM PERIOD_MN1,
-- g_tfNames[] in both PredictATrade_MasterNode_MT5.mq5:578 and
-- PredictATrade_MasterNode_MT4.mq4:577), but market.candles'
-- candles_timeframe_whitelist CHECK canonicalizes monthly bars as "MN"
-- (558 historical rows). Every MN1 bar_event from the freshly recompiled
-- Master nodes was therefore rejected on insert
-- (SaveCandle error: candles_timeframe_whitelist, SQLSTATE 23514 — seen
-- every ~10 s in pat-realtime logs from 2026-09-04 ~05:06 UTC).
--
-- Fix: normalize the alias on write (BEFORE-INSERT row trigger), aligning the
-- EA wire format with the DB canonical value. No engine rebuild, no EA
-- recompile, historical MN rows stay valid, and the ON CONFLICT upsert key
-- (time, symbol, timeframe, source) now matches existing MN rows so monthly
-- bars upsert instead of erroring.
-- ─────────────────────────────────────────────────────────────────────────────

CREATE OR REPLACE FUNCTION market.normalize_candle_timeframe()
RETURNS trigger
LANGUAGE plpgsql
AS $$
BEGIN
    IF NEW.timeframe = 'MN1' THEN
        NEW.timeframe := 'MN';
    END IF;
    RETURN NEW;
END;
$$;

DROP TRIGGER IF EXISTS trg_candles_tf_mn_alias ON market.candles;
CREATE TRIGGER trg_candles_tf_mn_alias
    BEFORE INSERT OR UPDATE OF timeframe ON market.candles
    FOR EACH ROW EXECUTE FUNCTION market.normalize_candle_timeframe();