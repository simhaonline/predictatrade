-- ─────────────────────────────────────────────────────────────────────────────
-- Migration 130: ATEN undorm — exit_profile + config version snapshot
--
-- Context (2026-09-03, v1.26 roster sweep):
--   ATEN (astro engine) was structurally dormant:
--   1. UniqueEntryGate had no ATEN handling → EntryGatePassed zero-value false
--      on every directional read → parity gate blocked 4,909/4,909 backtest bars
--      and live reads were delivered advisory-only.
--   2. Default ExitSpec MaxSpreadPips 2.5 sat below both the synthetic spread
--      ($0.30 = 3.0 pips) and the real broker spread (~33-35 pts = 3.3-3.5
--      pips) → every read additionally vetoed by the spread gate.
--   3. astro.Compute() hora index panicked on 00:00-05:59 UTC timestamps
--      (negative modulo) — latent live crash, fixed at root in astro/state.go.
--   4. ATEN hardcoded SL bid−10 / TP bid+20 — BUY-only geometry (wrong-side
--      stops if SELL fired) and bypassed the DB exit-profile path every other
--      strategy honors.
--
-- This migration inserts the ATEN exit profile (PERCENTAGE mode) mirroring the
-- designed astro geometry ($10 SL / $20 TP1 = 1:2 gross R:R) as percentages of
-- entry (reference price ~$4,000): SL 0.25% ≙ $10, TP1 0.50% ≙ $20, TP2 0.50%,
-- TP3 0.75% ≙ $30. ATR guardrails: M15 ATR ~15-25 → SL 1.5xATR-band [0.6x, 3.0x],
-- TP1 3.0xATR-band [1.5x, 6.0x].
--
-- HONEST-EDGE NOTE (not masked by this migration): the 90d Q4-2025 parity
-- backtest of per-bar astro direction measured wr 32.9% / PF 0.96 (−24%)
-- against a 1:2 geometry that breaks even near ~36-38% — the astro composite
-- carried NO positive directional edge on this window. Structural plumbing
-- fixed here; edge assessment stands as measured. ATEN remains UNARMED
-- (advisory-only) until a forward window shows positive expectancy.
-- ─────────────────────────────────────────────────────────────────────────────

UPDATE trading.exit_profiles
SET
    calculation_mode      = 'PERCENTAGE',
    stop_pct              = 0.0025,
    tp1_pct               = 0.0050,
    tp2_pct               = 0.0050,
    tp3_pct               = 0.0075,
    min_stop_atr_mult     = 0.6,
    max_stop_atr_mult     = 3.0,
    min_tp1_atr_mult      = 1.5,
    max_tp1_atr_mult      = 6.0,
    change_reason         = 'v1.26 ATEN undorm: DB-governed astro geometry (SL 0.25% / TP1 0.50% ≙ designed $10/$20 1:2 at ~$4000 ref), direction-correct for SELL, replaces BUY-only hardcoded SL/TP',
    code_commit           = 'aten-undorm'
WHERE strategy_id = 'ATEN';

INSERT INTO trading.exit_profiles
    (strategy_id, version, calculation_mode, stop_pct, tp1_pct, tp2_pct, tp3_pct,
     min_stop_atr_mult, max_stop_atr_mult, min_tp1_atr_mult, max_tp1_atr_mult,
     effective_from, change_reason, code_commit)
SELECT
    'ATEN', 'v1.26-aten-undorm', 'PERCENTAGE', 0.0025, 0.0050, 0.0050, 0.0075,
    0.6, 3.0, 1.5, 6.0,
    NOW(),
    'v1.26 ATEN undorm: DB-governed astro geometry (SL 0.25% / TP1 0.50% ≙ designed $10/$20 1:2), direction-correct for SELL',
    'aten-undorm'
WHERE NOT EXISTS (
    SELECT 1 FROM trading.exit_profiles WHERE strategy_id = 'ATEN'
);

-- Config snapshot for audit (mirrors code: refinement.go ATEN ExitSpec case +
-- aten_strategy.go shared-geometry wiring).
INSERT INTO trading.strategy_config_versions
    (strategy_id, version, values, effective_from, change_reason)
SELECT 'ATEN', 'v1.26-aten-undorm',
    '{
      "calculation_mode": "PERCENTAGE",
      "stop_pct": 0.0025,
      "tp1_pct": 0.0050,
      "tp2_pct": 0.0050,
      "tp3_pct": 0.0075,
      "min_stop_atr": 0.6,
      "max_stop_atr": 3.0,
      "min_tp1_atr": 1.5,
      "max_tp1_atr": 6.0,
      "min_rr": 1.5,
      "max_spread_pips": 4.0,
      "bias_threshold": 25,
      "expiry_minutes": 120,
      "armed": false,
      "armed_note": "plumbing fixed 2026-09-03; astro edge Q4-2025 measured negative (wr 32.9% PF 0.96) — remains advisory-only pending forward evidence"
    }'::jsonb,
    NOW(),
    'v1.26 ATEN undorm: refinement wiring + ATEN ExitSpec + per-bar astro timestamp + hora negative-modulo panic fix; UNARMED pending positive forward edge'
WHERE NOT EXISTS (
    SELECT 1 FROM trading.strategy_config_versions
    WHERE strategy_id = 'ATEN' AND version = 'v1.26-aten-undorm'
);