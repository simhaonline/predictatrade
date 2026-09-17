-- 151_trade_results_idempotency.sql
-- Phase 0.95 Task B (prompt.md): MT5-boundary failure-mode hardening.
-- Failure mode F7 proven by drill: a repeated TRADE_RESULT (same position
-- re-delivered after reconnect / EA restart) double-inserted into
-- trading.trade_results (5 pre-existing same-day dupes confirmed on Aug 25/26
-- legacy rows). prediction_outcomes is already idempotent (upsert on signal_id);
-- trade_results was not. Fix: dedupe + natural-key unique index + handler upsert.

-- 1) Dedupe existing duplicates, keep the EARLIEST delivery of each pair.
DELETE FROM trading.trade_results a
USING trading.trade_results b
WHERE a.signal_id = b.signal_id
  AND a.broker_ticket = b.broker_ticket
  AND a.ctid > b.ctid;

-- 2) Natural key: one trade_results row per (signal_id, broker_ticket).
CREATE UNIQUE INDEX IF NOT EXISTS uq_trade_results_signal_ticket
  ON trading.trade_results (signal_id, broker_ticket);

COMMENT ON INDEX uq_trade_results_signal_ticket IS
  'Phase 0.95: repeated TRADE_RESULT for the same position must upsert, not double-count';
