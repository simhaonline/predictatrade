-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 135: account-type spec conformance (additive)
--
-- The two spec tables (licensing.account_types, licensing.strategy_parameters)
-- exist from mig 133. This migration:
--   1) seeds licensing.strategy_parameters with the 7 default parameter sets
--      (the type-specific math the EA already implements — mirrored server-side
--      for traceability/dashboards),
--   2) adds a per-login "latest detection" upsert path via a partial unique
--      index + is_latest flag so account_types doubles as a live-state table
--      (history rows retained: is_latest flips on supersede),
--   3) adds is_latest BOOLEAN + detection_source for that purpose.
-- No existing columns or rows are modified or removed.
-- ─────────────────────────────────────────────────────────────────────────────

ALTER TABLE licensing.account_types
    ADD COLUMN IF NOT EXISTS is_latest BOOLEAN NOT NULL DEFAULT TRUE,
    ADD COLUMN IF NOT EXISTS detection_source VARCHAR(20) NOT NULL DEFAULT 'EA';

-- Per-login "one live row" enforcement: only one is_latest row per login.
CREATE UNIQUE INDEX IF NOT EXISTS uq_account_types_login_live
    ON licensing.account_types (account_login)
    WHERE is_latest;

-- Seed the 7 parameter sets (idempotent; EA-side detector remains source of truth).
INSERT INTO licensing.strategy_parameters (account_type, parameter_name, parameter_value, priority)
SELECT * FROM (VALUES
    ('Standard',  'lot_scale',              '1.0',   100),
    ('Standard',  'slippage_buffer_points', '0',     100),
    ('Standard',  'swap_adjustment',        'raw',   100),
    ('Standard',  'commission_model',       'none',  100),
    ('Standard',  'demo_tag',               'false', 100),
    ('Demo',      'lot_scale',              '1.0',   100),
    ('Demo',      'slippage_buffer_points', '0',     100),
    ('Demo',      'swap_adjustment',        'raw',   100),
    ('Demo',      'commission_model',       'none',  100),
    ('Demo',      'demo_tag',               'true',  100),
    ('Contest',   'lot_scale',              '1.0',   100),
    ('Contest',   'slippage_buffer_points', '0',     100),
    ('Contest',   'swap_adjustment',        'raw',   100),
    ('Contest',   'commission_model',       'none',  100),
    ('Contest',   'demo_tag',               'false', 100),
    ('Islamic',   'lot_scale',              '1.0',   100),
    ('Islamic',   'slippage_buffer_points', '0',     100),
    ('Islamic',   'swap_adjustment',        'zero',  100),
    ('Islamic',   'commission_model',       'none',  100),
    ('Islamic',   'demo_tag',               'false', 100),
    ('MicroCent', 'lot_scale',              '0.01',  100),
    ('MicroCent', 'slippage_buffer_points', '1',     100),
    ('MicroCent', 'swap_adjustment',        'raw',   100),
    ('MicroCent', 'commission_model',       'round_trip_cent', 100),
    ('MicroCent', 'demo_tag',               'false', 100),
    ('ECN',       'lot_scale',              '1.0',   100),
    ('ECN',       'slippage_buffer_points', '0',     100),
    ('ECN',       'swap_adjustment',        'raw',   100),
    ('ECN',       'commission_model',       'round_trip', 100),
    ('ECN',       'rr_erosion_widen',       'true',  100),
    ('ECN',       'demo_tag',               'false', 100),
    ('STP',       'lot_scale',              '1.0',   100),
    ('STP',       'slippage_buffer_points', '2',     100),
    ('STP',       'swap_adjustment',        'raw',   100),
    ('STP',       'commission_model',       'none',  100),
    ('STP',       'demo_tag',               'false', 100)
) AS seed(account_type, parameter_name, parameter_value, priority)
ON CONFLICT DO NOTHING;