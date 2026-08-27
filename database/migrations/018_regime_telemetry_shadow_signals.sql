-- LEGACY DUPLICATE PREFIX 018: shares its numeric prefix with another migration file. Tolerated legacy collision (see database/migrations/MIGRATION_ORDER.md); DO NOT rename (rename risks re-applying applied schema). The CI guard scripts/check_migrations.sh blocks NEW duplicate prefixes only.
-- Predict-A-Trade v1.0.0 — Migration 018
-- Phase 2: Regime Transition Telemetry & Shadow Signal Tables
-- SOW Phase 2 Sections 6, 9, 33
--
-- This migration adds:
-- 1. trading.regime_transitions — Records all regime state changes with market snapshots
-- 2. trading.shadow_signals — Stores shadow (non-executable) signal evaluations
-- 3. Version columns on trading.signals — regime_engine_version, strategy_version, etc.
--
-- NON-DESTRUCTIVE: Only adds new tables/columns. Uses IF NOT EXISTS.

-- ============================================================
-- Regime Transitions (SOW Phase 2 Section 6)
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.regime_transitions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    symbol          VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    from_regime     VARCHAR(30) NOT NULL,
    to_regime       VARCHAR(30) NOT NULL,
    transition_time TIMESTAMPTZ NOT NULL,
    reason          TEXT NOT NULL,
    -- Market snapshot at transition
    rsi             NUMERIC(10,4),
    adx             NUMERIC(10,4),
    atr             NUMERIC(10,4),
    ema_alignment   VARCHAR(20),
    -- Engine metadata
    engine_version  VARCHAR(20) NOT NULL DEFAULT '2.0.0',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_regime_transitions_time ON trading.regime_transitions(transition_time);
CREATE INDEX IF NOT EXISTS idx_regime_transitions_from ON trading.regime_transitions(from_regime);
CREATE INDEX IF NOT EXISTS idx_regime_transitions_to ON trading.regime_transitions(to_regime);

-- ============================================================
-- Shadow Signals (SOW Phase 2 Section 9)
-- Stores hypothetical signal evaluations for regime-mismatched strategies.
-- These are NEVER delivered to clients and NEVER executed.
-- ============================================================
CREATE TABLE IF NOT EXISTS trading.shadow_signals (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signal_id              UUID, -- References trading.signals(id) if linked
    strategy_id            VARCHAR(50) NOT NULL,
    symbol                 VARCHAR(20) NOT NULL DEFAULT 'XAUUSD',
    timestamp              TIMESTAMPTZ NOT NULL,
    regime                 VARCHAR(30) NOT NULL,
    hypothetical_direction VARCHAR(20) NOT NULL,
    hypothetical_score     NUMERIC(10,4),
    hypothetical_long      NUMERIC(10,4),
    hypothetical_short     NUMERIC(10,4),
    hypothetical_entry     NUMERIC(18,8),
    hypothetical_sl        NUMERIC(18,8),
    hypothetical_tp1       NUMERIC(18,8),
    hypothetical_tp2       NUMERIC(18,8),
    hypothetical_tp3       NUMERIC(18,8),
    hypothetical_rr        NUMERIC(10,4),
    evidence               JSONB,
    failed_production_reason TEXT NOT NULL,
    -- CRITICAL: These markers ensure shadow signals are never confused with real signals
    shadow_only            BOOLEAN NOT NULL DEFAULT TRUE,
    executable             BOOLEAN NOT NULL DEFAULT FALSE,
    -- Versioning
    regime_engine_version  VARCHAR(20) NOT NULL DEFAULT '2.0.0',
    strategy_version       VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    created_at             TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_shadow_signals_strategy ON trading.shadow_signals(strategy_id);
CREATE INDEX IF NOT EXISTS idx_shadow_signals_timestamp ON trading.shadow_signals(timestamp);
CREATE INDEX IF NOT EXISTS idx_shadow_signals_regime ON trading.shadow_signals(regime);
-- Critical constraint: shadow_only must always be TRUE and executable must always be FALSE
ALTER TABLE trading.shadow_signals ADD CONSTRAINT chk_shadow_not_executable
    CHECK (shadow_only = TRUE AND executable = FALSE);

-- ============================================================
-- Version columns on signals (SOW Phase 2 Section 33)
-- ============================================================
DO $$
BEGIN
    -- Add version columns if they don't exist
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'trading' AND table_name = 'signals'
                   AND column_name = 'regime_engine_version') THEN
        ALTER TABLE trading.signals ADD COLUMN regime_engine_version VARCHAR(20) DEFAULT '2.0.0';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'trading' AND table_name = 'signals'
                   AND column_name = 'strategy_version') THEN
        ALTER TABLE trading.signals ADD COLUMN strategy_version VARCHAR(20) DEFAULT '1.0.0';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'trading' AND table_name = 'signals'
                   AND column_name = 'scoring_version') THEN
        ALTER TABLE trading.signals ADD COLUMN scoring_version VARCHAR(20) DEFAULT '1.0.0';
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'trading' AND table_name = 'signals'
                   AND column_name = 'gate_config_version') THEN
        ALTER TABLE trading.signals ADD COLUMN gate_config_version VARCHAR(20) DEFAULT '1.0.0';
    END IF;

    -- Shadow markers on signals
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'trading' AND table_name = 'signals'
                   AND column_name = 'shadow_only') THEN
        ALTER TABLE trading.signals ADD COLUMN shadow_only BOOLEAN NOT NULL DEFAULT FALSE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'trading' AND table_name = 'signals'
                   AND column_name = 'executable') THEN
        ALTER TABLE trading.signals ADD COLUMN executable BOOLEAN NOT NULL DEFAULT TRUE;
    END IF;

    IF NOT EXISTS (SELECT 1 FROM information_schema.columns
                   WHERE table_schema = 'trading' AND table_name = 'signals'
                   AND column_name = 'failed_production_reason') THEN
        ALTER TABLE trading.signals ADD COLUMN failed_production_reason TEXT;
    END IF;
END $$;

-- ============================================================
-- Audit log entry
-- ============================================================
INSERT INTO control.audit_log (id, user_id, action, entity_type, entity_id, old_values, new_values, metadata, created_at)
SELECT gen_random_uuid(),
       NULL,
       'MIGRATION_APPLIED',
       'MIGRATION',
       '018',
       NULL,
       NULL,
       '{"description": "Phase 2: Regime transition telemetry, shadow signals, version columns", "migration": "018"}'::jsonb,
       now()
WHERE EXISTS (SELECT 1 FROM information_schema.tables WHERE table_schema = 'control' AND table_name = 'audit_log');
