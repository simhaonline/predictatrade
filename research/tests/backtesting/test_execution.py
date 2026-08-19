"""Tests for execution simulator: spread, slippage, commission, direction correctness."""
import pytest
import numpy as np
from datetime import datetime, timezone

from patresearch.backtesting.engine.execution import ExecutionSimulator, ExecutionConfig
from patresearch.backtesting.engine.events import OrderEvent
from patresearch.backtesting.data.loader import HistoricalCandle


def make_candle(price=2400.0, high=None, low=None):
    return HistoricalCandle(
        timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
        open=price, high=high or price + 1, low=low or price - 1, close=price,
        volume=100, timeframe="M5", source="SYNTHETIC",
    )


class TestExecutionSimulator:
    def test_long_uses_ask(self):
        """LONG entries should use ASK (close + half spread)."""
        sim = ExecutionSimulator(ExecutionConfig(fixed_spread=0.30, fixed_slippage=0.0))
        order = OrderEvent(
            timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
            direction="BUY", size=1.0, stop_loss=2390, take_profit=2410,
        )
        candle = make_candle(2400.0)
        fill = sim.execute(order, candle)
        # Ask = 2400 + 0.15 = 2400.15
        assert fill.fill_price == pytest.approx(2400.15, abs=0.01)

    def test_short_uses_bid(self):
        """SHORT entries should use BID (close - half spread)."""
        sim = ExecutionSimulator(ExecutionConfig(fixed_spread=0.30, fixed_slippage=0.0))
        order = OrderEvent(
            timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
            direction="SELL", size=1.0, stop_loss=2410, take_profit=2390,
        )
        candle = make_candle(2400.0)
        fill = sim.execute(order, candle)
        # Bid = 2400 - 0.15 = 2399.85
        assert fill.fill_price == pytest.approx(2399.85, abs=0.01)

    def test_slippage_applied(self):
        """Slippage should be applied in the correct direction."""
        sim = ExecutionSimulator(ExecutionConfig(fixed_spread=0.0, fixed_slippage=0.10))
        order = OrderEvent(
            timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
            direction="BUY", size=1.0, stop_loss=2390, take_profit=2410,
        )
        candle = make_candle(2400.0)
        fill = sim.execute(order, candle)
        # Ask = 2400, slippage = +0.10 → fill at 2400.10
        assert fill.fill_price == pytest.approx(2400.10, abs=0.01)
        assert fill.slippage == 0.10

    def test_commission_calculated(self):
        """Commission should be calculated correctly."""
        sim = ExecutionSimulator(ExecutionConfig(fixed_spread=0.0, fixed_slippage=0.0,
                                                  commission_per_lot=7.0))
        order = OrderEvent(
            timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
            direction="BUY", size=2.0, stop_loss=2390, take_profit=2410,
        )
        fill = sim.execute(order, make_candle(2400.0))
        assert fill.commission == pytest.approx(14.0, abs=0.01)  # 7.0 * 2.0

    def test_percentage_slippage(self):
        """Percentage slippage model should work."""
        sim = ExecutionSimulator(ExecutionConfig(
            spread_model="fixed", fixed_spread=0.0,
            slippage_model="percentage", slippage_percentage=0.001,
        ))
        order = OrderEvent(
            timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
            direction="BUY", size=1.0, stop_loss=2390, take_profit=2410,
        )
        fill = sim.execute(order, make_candle(2400.0))
        # Slippage = 2400 * 0.001 = 2.4
        assert fill.slippage == pytest.approx(2.4, abs=0.01)

    def test_atr_based_slippage(self):
        """ATR-based slippage should use ATR value."""
        sim = ExecutionSimulator(ExecutionConfig(
            spread_model="fixed", fixed_spread=0.0,
            slippage_model="atr_based", slippage_atr_multiplier=0.1,
        ))
        order = OrderEvent(
            timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
            direction="BUY", size=1.0, stop_loss=2390, take_profit=2410,
        )
        fill = sim.execute(order, make_candle(2400.0), atr=5.0)
        # Slippage = 5.0 * 0.1 = 0.5
        assert fill.slippage == pytest.approx(0.5, abs=0.01)

    def test_rejection_simulation(self):
        """Rejection should produce REJECTED fill."""
        sim = ExecutionSimulator(ExecutionConfig(
            fixed_spread=0.0, fixed_slippage=0.0,
            rejection_probability=1.0, random_seed=42,
        ))
        order = OrderEvent(
            timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
            direction="BUY", size=1.0, stop_loss=2390, take_profit=2410,
        )
        fill = sim.execute(order, make_candle(2400.0))
        assert fill.fill_status == "REJECTED"

    def test_partial_fill_simulation(self):
        """Partial fill should produce PARTIAL status."""
        sim = ExecutionSimulator(ExecutionConfig(
            fixed_spread=0.0, fixed_slippage=0.0,
            partial_fill_probability=1.0, random_seed=42,
        ))
        order = OrderEvent(
            timestamp=datetime(2025, 1, 1, tzinfo=timezone.utc),
            direction="BUY", size=1.0, stop_loss=2390, take_profit=2410,
        )
        fill = sim.execute(order, make_candle(2400.0))
        assert fill.fill_status == "PARTIAL"
        assert fill.fill_ratio < 1.0

    def test_exit_long_uses_bid(self):
        """Exit of a BUY position should sell at BID."""
        sim = ExecutionSimulator(ExecutionConfig(fixed_spread=0.30, fixed_slippage=0.0))
        fill = sim.execute_exit("BUY", 1.0, make_candle(2400.0))
        # Sell at bid = 2400 - 0.15 = 2399.85
        assert fill.fill_price == pytest.approx(2399.85, abs=0.01)

    def test_exit_short_uses_ask(self):
        """Exit of a SELL position should buy at ASK."""
        sim = ExecutionSimulator(ExecutionConfig(fixed_spread=0.30, fixed_slippage=0.0))
        fill = sim.execute_exit("SELL", 1.0, make_candle(2400.0))
        # Buy at ask = 2400 + 0.15 = 2400.15
        assert fill.fill_price == pytest.approx(2400.15, abs=0.01)
