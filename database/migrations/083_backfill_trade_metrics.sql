-- Backfill derived trade metrics so dashboards show real direction, R:R points
-- and hold-time instead of empty/zero placeholders.
-- Idempotent: only touches rows that are still missing the derived values.

UPDATE trading.trade_results tr
SET
  direction = COALESCE(
    NULLIF(tr.direction, ''),
    (SELECT CASE
       WHEN s.direction LIKE '%BUY%' THEN 'BUY'
       WHEN s.direction LIKE '%SELL%' THEN 'SELL'
       ELSE s.direction
     END FROM trading.signals s WHERE s.id = tr.signal_id),
    'UNKNOWN'
  ),
  opened_at = COALESCE(
    tr.opened_at,
    (SELECT s.created_at FROM trading.signals s WHERE s.id = tr.signal_id)
  ),
  pnl_points = CASE
    WHEN tr.pnl_points <> 0 THEN tr.pnl_points
    WHEN tr.lot_size > 0 THEN tr.pnl / tr.lot_size   -- XAUUSD: 1 lot=100oz, 1pt=0.01
    ELSE tr.pnl_points
  END,
  time_in_trade_seconds = CASE
    WHEN tr.time_in_trade_seconds <> 0 THEN tr.time_in_trade_seconds
    WHEN tr.opened_at IS NOT NULL
      THEN EXTRACT(EPOCH FROM (tr.closed_at - tr.opened_at))::bigint
    ELSE 0
  END
WHERE tr.direction = ''
   OR tr.opened_at IS NULL
   OR tr.pnl_points = 0
   OR tr.time_in_trade_seconds = 0;
