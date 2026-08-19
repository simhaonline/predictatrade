"""Predict-A-Trade Research Plane — backtesting, calibration, ML, feature parity."""
from .reference_math import (
    gross_rr, net_rr, expectancy, profit_factor, cost_to_target,
    wilson_interval, brier_score, ece, mtf_alignment_score,
    atr, true_range, ema, rsi, monte_carlo_drawdown, sharpe_ratio, sortino_ratio,
)
from .backtester import Backtester, BacktestResult, walk_forward, locked_oos, Candle, Trade
from .calibration import (
    brier_score as calib_brier, ece as calib_ece,
    wilson_interval as calib_wilson, sample_sufficiency,
    fit_sigmoid_calibration, apply_calibration,
)
from .dataset import DataProvenance, import_csv_candles, generate_synthetic_candles
