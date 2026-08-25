"""Walk-forward analysis (rolling in-sample / out-of-sample backtest).

Methodology:
1. Reserve a final untouched holdout dataset (never used for any tuning).
2. Slide a train/test window across the remaining data.
3. Run the FIXED strategy configuration on the in-sample (train) window.
4. Run the SAME fixed configuration on the out-of-sample (test) window,
   with a freshly-reset strategy instance to guarantee no state leakage.
5. Roll the window forward and repeat.
6. Aggregate only the out-of-sample trades/metrics (plus a final holdout run).

NOTE: This implementation evaluates a FIXED parameter set across folds. It does
NOT perform parameter optimization on the training window (no grid/random search,
no locked-parameter selection). It is a walk-forward STABILITY / OOS evaluation,
not an optimizer. Never tune on the final untouched holdout dataset.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import List, Optional, Dict, Callable, Tuple
import numpy as np

from ..data.loader import HistoricalCandle
from ..engine.core import BacktestEngine, BacktestConfig, BacktestRunResult
from ..strategy.base import BaseStrategy


@dataclass
class WalkForwardConfig:
    """Walk-forward analysis configuration."""
    train_size: int = 500  # bars in training window
    test_size: int = 100  # bars in test window
    step_size: int = 100  # step between windows
    min_trades: int = 10  # minimum trades per fold
    objective: str = "sharpe"  # sharpe, sortino, profit_factor, expectancy


@dataclass
class WalkForwardFold:
    """A single walk-forward fold."""
    fold_id: int
    train_start: int
    train_end: int
    test_start: int
    test_end: int
    in_sample_result: Optional[BacktestRunResult] = None
    out_of_sample_result: Optional[BacktestRunResult] = None


@dataclass
class WalkForwardResult:
    """Complete walk-forward analysis result."""
    folds: List[WalkForwardFold] = field(default_factory=list)
    aggregate_oos_metrics: Optional[Dict] = None
    final_holdout_result: Optional[BacktestRunResult] = None


class WalkForwardAnalyzer:
    """Performs walk-forward analysis with strict train/test isolation."""

    def __init__(self, config: WalkForwardConfig):
        self.config = config

    def run(self, candles: List[HistoricalCandle],
            strategy_factory: Callable[[], BaseStrategy],
            bt_config: BacktestConfig,
            higher_tf_data: Optional[Dict[str, List[HistoricalCandle]]] = None,
            final_holdout_pct: float = 0.2) -> WalkForwardResult:
        """Run walk-forward analysis.

        Args:
            candles: Full historical dataset
            strategy_factory: Function that creates a fresh strategy instance
            bt_config: Backtest configuration
            higher_tf_data: Higher timeframe data
            final_holdout_pct: Percentage of data reserved as final untouched holdout

        Returns:
            WalkForwardResult with all folds and aggregate OOS metrics
        """
        result = WalkForwardResult()

        # Reserve final holdout
        holdout_start = int(len(candles) * (1 - final_holdout_pct))
        wf_candles = candles[:holdout_start]
        holdout_candles = candles[holdout_start:]

        # Generate fold boundaries
        fold_id = 0
        all_oos_trades = []

        for train_end in range(self.config.train_size, len(wf_candles) - self.config.test_size,
                                self.config.step_size):
            test_start = train_end
            test_end = min(train_end + self.config.test_size, len(wf_candles))

            if test_end - test_start < self.config.min_trades * 2:
                break  # Not enough test data

            train_candles = wf_candles[:train_end]
            test_candles = wf_candles[test_start:test_end]

            fold = WalkForwardFold(
                fold_id=fold_id,
                train_start=0, train_end=train_end,
                test_start=test_start, test_end=test_end,
            )

            # Run in-sample
            in_sample_engine = BacktestEngine(bt_config)
            in_sample_engine.set_strategy(strategy_factory())
            fold.in_sample_result = in_sample_engine.run(train_candles, higher_tf_data)

            # Run out-of-sample (strategy must be reset for isolation)
            oos_engine = BacktestEngine(bt_config)
            oos_strategy = strategy_factory()
            oos_strategy.reset()  # Ensure no state leakage
            oos_engine.set_strategy(oos_strategy)
            fold.out_of_sample_result = oos_engine.run(test_candles, higher_tf_data)

            # Collect OOS trades
            if fold.out_of_sample_result and fold.out_of_sample_result.trades:
                all_oos_trades.extend(fold.out_of_sample_result.trades)

            result.folds.append(fold)
            fold_id += 1

        # Aggregate OOS metrics
        if all_oas_trades := all_oos_trades:
            from ..analytics.metrics import calculate_metrics
            agg_metrics = calculate_metrics(all_oas_trades, [], bt_config.initial_balance)
            result.aggregate_oos_metrics = {
                "total_oos_trades": len(all_oas_trades),
                "total_pnl": sum(t.pnl for t in all_oas_trades),
                "win_rate": sum(1 for t in all_oas_trades if t.pnl > 0) / len(all_oas_trades) * 100,
                "avg_pnl": sum(t.pnl for t in all_oas_trades) / len(all_oas_trades),
                "profit_factor": agg_metrics.profit_factor,
                "sharpe": agg_metrics.sharpe_ratio,
                "max_drawdown_pct": agg_metrics.max_drawdown_pct,
            }

        # Run final holdout (never used for optimization)
        if holdout_candles and len(holdout_candles) > 50:
            holdout_engine = BacktestEngine(bt_config)
            holdout_strategy = strategy_factory()
            holdout_strategy.reset()
            holdout_engine.set_strategy(holdout_strategy)
            result.final_holdout_result = holdout_engine.run(holdout_candles, higher_tf_data)

        return result
