-- Migration 129: STANDARD_SCALPING win-rate-first rebuild (v1.26)
--
-- The 90-day production-parity backtest (Q4-2025 Kaggle, gates ON) measured
-- STANDARD_SCALPING at −45.97% (PF 0.81, wr 41.5%, 337 trades). Root causes:
--   1. TP1 at 2.5×ATR vs SL 1.5×ATR requires wr ≳ 47% just to cover round-trip
--      cost (~0.4pt/oz spread+commission+slippage) — realized wr 41.5% bleeds.
--   2. Weak-momentum entries passed the old EMA+VWAP-only gate.
--
-- Rebuild (cost-aware win-rate-first): tight stop + close target —
--   exit_profiles (PERCENTAGE, ~$4000 gold): tp1 0.15% → 0.09% (~1.2×ATR),
--   tp2 0.30% → 0.15% (~2.0×ATR), tp3 0.45% → 0.26% (~3.5×ATR);
--   stop 0.15% → 0.08% (~0.8×ATR). SL 0.8×ATR + TP1 1.2×ATR puts the
--   cost-aware breakeven win rate at ≈44% — achievable by the entry read,
--   vs 58.5% demanded by the intermediate SL 1.5 geometry (0 trades: EV gate
--   correctly rejected every read) and ~47%+cost demanded by the original
--   SL 1.5/TP1 2.5 (realized 41.5%).
-- Code-side counterparts (same commit): StrategyExitSpec + StrategyConfig
-- SL/TP1/TP2/TP3 ATR multipliers, MinRR 2.0 → 1.0, momentum+ADX entry gate.
-- Sessions remain open (LONDON/NY-only variant vetoed 5,688 of ~8k reads —
-- the bleed is EV/geometry-driven, not session-driven).
--
-- NOTE: micro-TP cost coverage (MicroTPProfitable) is unaffected — micro level
-- moves CLOSER (0.5×ATR) but the gate checks microDist > cost, which holds for
-- ATR ≥ ~0.85pt at the broker's real ~0.35pt round trip.

UPDATE trading.exit_profiles
SET stop_pct     = 0.0008,
    tp1_pct     = 0.0009,
    tp2_pct     = 0.0015,
    tp3_pct     = 0.0026,
    min_tp1_atr_mult = 0.30,
    change_reason = 'v1.26 STANDARD_SCALPING cost-aware win-rate rebuild: SL 0.8xATR (0.08%), TP1 1.2xATR (0.09%), TP2 2.0xATR (0.15%), TP3 3.5xATR (0.26%); forensics: -45.97% PF 0.81 wr 41.5% under old geometry',
    code_commit   = 'ss-rebuild'
WHERE strategy_id = 'STANDARD_SCALPING';

-- Version the config snapshot for audit (values mirror the code defaults).
INSERT INTO trading.strategy_config_versions
    (strategy_id, version, values, effective_from, change_reason)
SELECT 'STANDARD_SCALPING', 'v1.26-scalp-rebuild',
    '{
      "min_confluence": 65,
      "min_mtf_alignment": 40,
      "min_adx": 20,
      "min_rr": 1.0,
      "sl_pct": 0.0008,
      "tp1_pct": 0.0009,
      "tp2_pct": 0.0015,
      "tp3_pct": 0.0026,
      "min_stop_atr": 0.5,
      "max_stop_atr": 2.0,
      "calculation_mode": "PERCENTAGE",
      "accepted_regimes": ["TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT", "MEAN_REVERSION", "RANGE", "HIGH_VOLATILITY"],
      "accepted_sessions": ["LONDON", "NEW_YORK", "OVERLAP", "TOKYO", "SYDNEY"]
    }'::jsonb,
    NOW(),
    'v1.26 cost-aware win-rate rebuild: SL 0.8xATR + TP1 1.2xATR (breakeven wr ~44%), momentum>=1 + ADX>=20 entry gate, sessions open'
WHERE NOT EXISTS (
    SELECT 1 FROM trading.strategy_config_versions
    WHERE strategy_id = 'STANDARD_SCALPING' AND version = 'v1.26-scalp-rebuild'
);