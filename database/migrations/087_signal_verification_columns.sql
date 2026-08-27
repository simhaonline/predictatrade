-- SOW: signal verification / risk-decision transparency.
-- Adds AI Verification and Risk Decision columns so dashboards can render
-- them for historical signals (live WebSocket signals already carry them).
-- Forward-only, safe additive change.

ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS ai_verification TEXT NOT NULL DEFAULT '';
ALTER TABLE trading.signals ADD COLUMN IF NOT EXISTS risk_decision TEXT NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_signals_ai_verification ON trading.signals(ai_verification);
CREATE INDEX IF NOT EXISTS idx_signals_risk_decision ON trading.signals(risk_decision);
