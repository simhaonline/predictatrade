from __future__ import annotations

from patresearch.backtesting.strategy.ptb_strategy import STRATEGY_CONFIGS


def test_all_strategies_clear_rr_gate():
    """Regression: every strategy's ATR-based TP1 must exceed SL by at least
    min_rr, otherwise evaluate() returns a permanent NO_TRADE (POOR_RR).

    Historically STANDARD_SCALPING / STANDARD_SWING / TREND_SWING had
    atr_tp1 == atr_sl (RR==1.0) which is below their min_rr, so they never
    traded. This guard prevents that regression.
    """
    assert STRATEGY_CONFIGS, "strategy configs missing"
    for name, cfg in STRATEGY_CONFIGS.items():
        sl = cfg["atr_sl"]
        tp1 = cfg["atr_tp1"]
        min_rr = cfg["min_rr"]
        assert sl > 0 and tp1 > 0, f"{name}: invalid ATR multipliers"
        assert tp1 > sl, f"{name}: TP1 must exceed SL (RR>1)"
        rr = tp1 / sl
        assert rr >= min_rr, (
            f"{name}: R:R {rr:.3f} < min_rr {min_rr} -> permanent NO_TRADE"
        )
