"""Tests for performance metrics: return, Sharpe, Sortino, drawdown, edge cases."""
import pytest
import numpy as np
from datetime import datetime, timezone, timedelta

from patresearch.backtesting.engine.portfolio import TradeRecord, EquityPoint
from patresearch.backtesting.analytics.metrics import calculate_metrics, PerformanceMetrics


def make_trade(pnl, pnl_r=0.0, direction="BUY", days_ago=0):
    entry = datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc) + timedelta(days=days_ago)
    return TradeRecord(
        trade_id="t1", signal_id="s1", strategy_id="STANDARD_SCALPING",
        direction=direction, entry_time=entry, entry_price=2400,
        exit_time=entry + timedelta(hours=1), exit_price=2400 + pnl / 100,
        exit_reason="TP", size=1.0, pnl=pnl, pnl_r=pnl_r,
    )


class TestMetrics:
    def test_zero_trades(self):
        """Zero trades should produce safe metrics."""
        metrics = calculate_metrics([], [], 10000)
        assert metrics.total_trades == 0
        assert metrics.final_balance == 10000
        assert metrics.net_profit == 0
        assert not np.isnan(metrics.sharpe_ratio)

    def test_all_wins(self):
        """All wins should handle zero losses gracefully."""
        trades = [make_trade(100, 1.0), make_trade(200, 2.0), make_trade(150, 1.5)]
        metrics = calculate_metrics(trades, [], 10000)
        assert metrics.wins == 3
        assert metrics.losses == 0
        # Profit factor with zero losses should be inf, not NaN
        assert metrics.profit_factor == float('inf')

    def test_all_losses(self):
        """All losses should handle zero wins gracefully."""
        trades = [make_trade(-100, -1.0), make_trade(-200, -2.0)]
        metrics = calculate_metrics(trades, [], 10000)
        assert metrics.wins == 0
        assert metrics.losses == 2
        assert metrics.win_rate_pct == 0

    def test_win_rate(self):
        """Win rate should be calculated correctly."""
        trades = [make_trade(100), make_trade(-50), make_trade(200), make_trade(-30)]
        metrics = calculate_metrics(trades, [], 10000)
        assert metrics.win_rate_pct == 50.0

    def test_profit_factor(self):
        """Profit factor should be gross profit / gross loss."""
        trades = [make_trade(300), make_trade(-100), make_trade(200), make_trade(-50)]
        metrics = calculate_metrics(trades, [], 10000)
        assert metrics.profit_factor == pytest.approx(500 / 150, abs=0.01)

    def test_expectancy(self):
        """Expectancy should be average PnL per trade."""
        trades = [make_trade(100), make_trade(-50), make_trade(200), make_trade(-100)]
        metrics = calculate_metrics(trades, [], 10000)
        assert metrics.expectancy == pytest.approx(150 / 4, abs=0.01)

    def test_max_consecutive_wins(self):
        """Max consecutive wins should be tracked."""
        trades = [make_trade(10), make_trade(20), make_trade(-5), make_trade(30), make_trade(40), make_trade(-10)]
        metrics = calculate_metrics(trades, [], 10000)
        assert metrics.max_consecutive_wins == 2  # first two + later two

    def test_max_consecutive_losses(self):
        """Max consecutive losses should be tracked."""
        trades = [make_trade(10), make_trade(-5), make_trade(-10), make_trade(-15), make_trade(20), make_trade(-30)]
        metrics = calculate_metrics(trades, [], 10000)
        assert metrics.max_consecutive_losses == 3

    def test_sharpe_not_nan(self):
        """Sharpe ratio should never be NaN with sufficient trades."""
        trades = [make_trade(np.random.randn() * 50) for _ in range(20)]
        metrics = calculate_metrics(trades, [], 10000)
        assert not np.isnan(metrics.sharpe_ratio)

    def test_sortino_not_nan(self):
        """Sortino ratio should never be NaN with sufficient trades."""
        trades = [make_trade(np.random.randn() * 50) for _ in range(20)]
        metrics = calculate_metrics(trades, [], 10000)
        assert not np.isnan(metrics.sortino_ratio)

    def test_drawdown_from_equity_curve(self):
        """Max drawdown should be computed from equity curve."""
        trades = [make_trade(100), make_trade(-200)]
        equity_curve = [
            {"timestamp": "2025-01-01T00:00:00+00:00", "equity": 10100, "balance": 10100, "drawdown": 0},
            {"timestamp": "2025-01-01T00:05:00+00:00", "equity": 9900, "balance": 9900, "drawdown": 200},
        ]
        metrics = calculate_metrics(trades, equity_curve, 10000)
        assert metrics.max_drawdown_pct > 0

    def test_best_worst_trade(self):
        """Best and worst trade should be tracked."""
        trades = [make_trade(100), make_trade(-50), make_trade(300), make_trade(-200)]
        metrics = calculate_metrics(trades, [], 10000)
        assert metrics.best_trade == 300
        assert metrics.worst_trade == -200

    def test_var_95(self):
        """VaR 95% should be computed with enough trades."""
        trades = [make_trade(np.random.randn() * 50) for _ in range(50)]
        metrics = calculate_metrics(trades, [], 10000)
        assert metrics.var_95 != 0  # should have a value

    def test_segmentation_by_direction(self):
        """Trades should be segmented by direction."""
        trades = [make_trade(100, direction="BUY"), make_trade(-50, direction="SELL"),
                  make_trade(200, direction="BUY")]
        metrics = calculate_metrics(trades, [], 10000)
        assert "BUY" in metrics.by_direction
        assert "SELL" in metrics.by_direction
