-- 149_signal_feature_snapshots.sql
-- P0-2 (prompt.md): per-signal feature snapshots — the verifiable indicator
-- reading flow. Every signal stores the EXACT raw indicator values used by
-- its decision, so any signal can be replayed/audited: signal → snapshot →
-- endpoint returns the same values.
--
-- Design:
--   trading.signal_feature_snapshots — one row per signal (1:1, uuid PK =
--   the signal's id). Payload is JSONB (schema-versioned) with the full
--   MarketState indicator read set + feature/code/indicator-set versions.
--   trading.signals.feature_snapshot_id (existing uuid column, currently
--   always NULL) is set to the snapshot id on insert.
--
-- Retention: 90 days (diagnostic data, not audit/financial) — registered with
-- the existing system.data_retention_policies pruning framework from mig 147.

CREATE TABLE IF NOT EXISTS trading.signal_feature_snapshots (
  id              UUID PRIMARY KEY,                -- = trading.signals.id (1:1)
  signal_id       UUID        NOT NULL,
  symbol          TEXT        NOT NULL,
  timeframe       TEXT        NOT NULL,
  strategy_id     TEXT        NOT NULL,
  strategy_version TEXT       NOT NULL DEFAULT '',
  created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

  -- schema/version metadata
  schema_version  TEXT        NOT NULL DEFAULT '1.0',
  feature_version TEXT        NOT NULL DEFAULT '1.0',
  indicator_set_version TEXT NOT NULL DEFAULT '1.0',

  -- the raw indicator read set (all values used by the decision)
  indicators      JSONB       NOT NULL,

  -- provenance
  input_hash      TEXT        NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_sfs_created_at ON trading.signal_feature_snapshots (created_at);
CREATE INDEX IF NOT EXISTS idx_sfs_signal_id  ON trading.signal_feature_snapshots (signal_id);

COMMENT ON TABLE trading.signal_feature_snapshots IS
  'P0-2 per-signal raw indicator snapshots (1:1 with trading.signals); payload JSONB, schema-versioned';