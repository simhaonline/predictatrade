-- Predict-A-Trade v1.0.0 — Migration 115
-- Fix signal delivery persistence: agent device IDs carry a role suffix
-- (e.g. "<uuid>-exec" / "<uuid>-data") and are not valid UUIDs, which made
-- the UUID-typed device_id columns reject every RecordDelivery insert.
-- Switch signal_deliveries.device_id and signal_sequences.device_id to TEXT
-- so the runtime agent identifier (a string) is stored verbatim.
-- (Forward migration only — applied history is never rewritten.)

ALTER TABLE trading.signal_deliveries
    ALTER COLUMN device_id TYPE TEXT USING device_id::text;

ALTER TABLE trading.signal_sequences
    ALTER COLUMN device_id TYPE TEXT USING device_id::text;

-- The unique index on (signal_id, device_id) is preserved automatically when
-- the column type changes. Re-assert it explicitly for clarity/durability.
DROP INDEX IF EXISTS trading.uq_signal_deliveries_signal_device;
CREATE UNIQUE INDEX uq_signal_deliveries_signal_device
    ON trading.signal_deliveries(signal_id, device_id) WHERE device_id IS NOT NULL;

COMMENT ON COLUMN trading.signal_deliveries.device_id IS 'Runtime agent identifier (string, may carry -exec/-data suffix), not a UUID';
COMMENT ON COLUMN trading.signal_sequences.device_id IS 'Runtime agent identifier (string, may carry -exec/-data suffix), not a UUID';
