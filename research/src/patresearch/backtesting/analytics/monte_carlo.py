"""Monte Carlo robustness analysis.

Uses trade returns to estimate distributions of:
- Final equity
- Total return
- Maximum drawdown
- Longest losing streak
- Probability of loss
- Probability of exceeding defined drawdown

Does not report a single curve — reports percentile distributions.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from typing import List, Optional, Dict
import numpy as np

from ..engine.portfolio import TradeRecord


@dataclass
class MonteCarloConfig:
    """Monte Carlo analysis configuration."""
    n_simulations: int = 1000
    random_seed: int = 42
    confidence_levels: List[float] = field(default_factory=lambda: [0.05, 0.25, 0.50, 0.75, 0.95])


@dataclass
class MonteCarloResult:
    """Monte Carlo analysis result."""
    n_simulations: int = 0
    final_equity_percentiles: Dict = field(default_factory=dict)
    total_return_percentiles: Dict = field(default_factory=dict)
    max_drawdown_percentiles: Dict = field(default_factory=dict)
    longest_losing_streak_percentiles: Dict = field(default_factory=dict)
    prob_of_loss: float = 0.0
    prob_of_drawdown_exceeding: Dict = field(default_factory=dict)
    random_seed: int = 42


class MonteCarloAnalyzer:
    """Monte Carlo analysis using trade resampling."""

    def __init__(self, config: MonteCarloConfig):
        self.config = config

    def run(self, trades: List[TradeRecord], initial_balance: float = 10000.0,
            drawdown_thresholds: List[float] = None) -> MonteCarloResult:
        """Run Monte Carlo simulation.

        Resamples trade returns with replacement to generate
        distribution of possible outcomes.
        """
        if not trades:
            return MonteCarloResult()

        result = MonteCarloResult(
            n_simulations=self.config.n_simulations,
            random_seed=self.config.random_seed,
        )

        rng = np.random.RandomState(self.config.random_seed)
        trade_pnls = np.array([t.pnl for t in trades])
        n_trades = len(trade_pnls)

        # Run simulations
        final_equities = []
        total_returns = []
        max_drawdowns = []
        losing_streaks = []

        for _ in range(self.config.n_simulations):
            # Resample with replacement
            sampled = rng.choice(trade_pnls, size=n_trades, replace=True)

            # Compute equity curve
            equity = initial_balance + np.cumsum(sampled)
            final_equities.append(equity[-1])
            total_returns.append((equity[-1] - initial_balance) / initial_balance * 100)

            # Max drawdown
            running_max = np.maximum.accumulate(equity)
            drawdowns = (running_max - equity) / running_max * 100
            max_dd = np.max(drawdowns) if len(drawdowns) > 0 else 0
            max_drawdowns.append(max_dd)

            # Longest losing streak
            streak = 0
            max_streak = 0
            for p in sampled:
                if p < 0:
                    streak += 1
                    max_streak = max(max_streak, streak)
                else:
                    streak = 0
            losing_streaks.append(max_streak)

        # Compute percentiles
        for cl in self.config.confidence_levels:
            pct = int(cl * 100)
            result.final_equity_percentiles[f"p{pct}"] = float(np.percentile(final_equities, pct))
            result.total_return_percentiles[f"p{pct}"] = float(np.percentile(total_returns, pct))
            result.max_drawdown_percentiles[f"p{pct}"] = float(np.percentile(max_drawdowns, pct))
            result.longest_losing_streak_percentiles[f"p{pct}"] = int(np.percentile(losing_streaks, pct))

        # Probability of loss
        result.prob_of_loss = float(np.mean(np.array(total_returns) < 0))

        # Probability of exceeding drawdown thresholds
        if drawdown_thresholds is None:
            drawdown_thresholds = [5.0, 10.0, 15.0, 20.0]

        for threshold in drawdown_thresholds:
            result.prob_of_drawdown_exceeding[f"{threshold}%"] = float(
                np.mean(np.array(max_drawdowns) > threshold)
            )

        return result
