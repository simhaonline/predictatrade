-- 120: Capital-tier awareness for the tiered signal engine (Option A: one
-- engine, tier-aware delivery).
--
-- Customers are served by capital category:
--   MICRO   — equity < $500
--   STANDARD— equity $500..$4,999.99
--   PRO     — equity >= $5,000
--
-- The Go engine classifies each exec device from the equity streamed in its
-- ACCOUNT_INFO heartbeats, stores it here, and only enqueues signals whose
-- per-tier viability covers the device's tier. Subscription-plan risk caps
-- (control.plan_entitlements) stay layered on top: effective cap =
-- min(plan per_trade_risk_pct, tier cap) — suitability composes with plan,
-- never weakens a cap.

-- Equity (account currency USD) as last reported by the device's EA.
ALTER TABLE licensing.edge_device_state
    ADD COLUMN IF NOT EXISTS last_equity numeric NOT NULL DEFAULT 0;

-- Classified capital tier for the device. '' = unknown (no ACCOUNT_INFO yet);
-- unknown devices are NEVER tier-excluded (fail-open to current behavior) so
-- a telemetry gap cannot silently starve an existing customer's signals.
ALTER TABLE licensing.edge_device_state
    ADD COLUMN IF NOT EXISTS capital_tier text NOT NULL DEFAULT ''
    CHECK (capital_tier IN ('', 'MICRO', 'STANDARD', 'PRO'));

-- Human/audit context: when the tier last changed and from/to values, so
-- support can explain "why did my signal flow change" after a deposit.
ALTER TABLE licensing.edge_device_state
    ADD COLUMN IF NOT EXISTS tier_changed_at timestamptz;

CREATE INDEX IF NOT EXISTS idx_edge_device_state_capital_tier
    ON licensing.edge_device_state (capital_tier);

COMMENT ON COLUMN licensing.edge_device_state.last_equity IS
    'Last equity streamed via ACCOUNT_INFO (USD); source of truth for capital-tier classification';
COMMENT ON COLUMN licensing.edge_device_state.capital_tier IS
    'MICRO <500 | STANDARD 500-4999.99 | PRO >=5000 | '''' = unknown (fail-open delivery)';