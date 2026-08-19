"""Tests for portfolio engine: PnL, equity, SL/TP, trailing, break-even, time exit."""
import pytest
import numpy as np
from datetime import datetime, timezone, timedelta

from patresearch.backtesting.engine.portfolio import Portfolio, Position, TradeRecord
from patresearch.backtesting.engine.events import FillEvent
from patresearch.backtesting.data.loader import HistoricalCandle


def make_candle(ts, o, h, l, c):
    return HistoricalCandle(
        timestamp=ts, open=o, high=h, low=l, close=c,
        volume=100, timeframe="M5", source="SYNTHETIC",
    )


class TestPortfolio:
    def test_long_pnl(self):
        """Long position PnL should be correct."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100)
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="BUY", fill_price=2400, requested_price=2400,
            size=1.0, commission=7.0,
        )
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2390, 2410)

        # Price goes up to 2410 → TP hit
        candle = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2405, 2415, 2400, 2410)
        exits = portfolio.update_positions(candle)

        assert len(exits) == 1
        assert exits[0].exit_reason == "TAKE_PROFIT"
        # PnL = (2410 - 2400) * 1 * 100 - 7 = 993
        assert exits[0].realized_pnl == pytest.approx(993, abs=1)

    def test_short_pnl(self):
        """Short position PnL should be correct."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100)
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="SELL", fill_price=2400, requested_price=2400,
            size=1.0, commission=7.0,
        )
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2410, 2390)

        # Price goes down to 2390 → TP hit
        candle = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2395, 2400, 2385, 2390)
        exits = portfolio.update_positions(candle)

        assert len(exits) == 1
        assert exits[0].exit_reason == "TAKE_PROFIT"
        # PnL = (2400 - 2390) * 1 * 100 - 7 = 993
        assert exits[0].realized_pnl == pytest.approx(993, abs=1)

    def test_stop_loss_long(self):
        """Stop loss should trigger for long position."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100)
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="BUY", fill_price=2400, requested_price=2400,
            size=1.0, commission=7.0,
        )
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2390, 2410)

        # Price drops to 2385 → SL hit
        candle = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2395, 2396, 2385, 2388)
        exits = portfolio.update_positions(candle)

        assert len(exits) == 1
        assert exits[0].exit_reason == "STOP_LOSS"

    def test_conservative_same_bar_sl_tp(self):
        """When both SL and TP hit same bar, conservative policy assumes SL first."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100,
                               conservative_sl_tp=True)
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="BUY", fill_price=2400, requested_price=2400,
            size=1.0, commission=7.0,
        )
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2390, 2410)

        # Both SL (2390) and TP (2410) hit in same candle
        candle = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2400, 2415, 2385, 2400)
        exits = portfolio.update_positions(candle)

        assert len(exits) == 1
        assert exits[0].exit_reason == "STOP_LOSS"  # conservative = SL first

    def test_end_of_test_close(self):
        """Open positions should be closed at end of backtest."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100)
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="BUY", fill_price=2400, requested_price=2400,
            size=1.0, commission=7.0,
        )
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2390, 2410)

        candle = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2400, 2405, 2395, 2402)
        exits = portfolio.close_all_positions(candle)

        assert len(exits) == 1
        assert exits[0].exit_reason == "END_OF_BACKTEST"

    def test_trailing_stop(self):
        """Trailing stop should move stop loss in favorable direction."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100,
                               trailing_stop_enabled=True, trailing_atr_multiplier=1.0)
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="BUY", fill_price=2400, requested_price=2400,
            size=1.0, commission=0.0,
        )
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2390, 2450)

        # Price moves up
        candle1 = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2405, 2410, 2400, 2408)
        portfolio.update_positions(candle1, atr=5.0)

        # Trailing stop should have moved up
        pos = list(portfolio.positions.values())[0]
        assert pos.trailing_active
        assert pos.stop_loss > 2390  # moved up from initial

    def test_break_even(self):
        """Break-even should move stop to entry when price moves 1R in favor."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100,
                               break_even_enabled=True, break_even_trigger_r=1.0)
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="BUY", fill_price=2400, requested_price=2400,
            size=1.0, commission=0.0,
        )
        # SL = 2390, so risk = 10, 1R = 2410
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2390, 2420)

        # Price reaches 2410 (1R)
        candle = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2405, 2415, 2400, 2412)
        portfolio.update_positions(candle)

        pos = list(portfolio.positions.values())[0]
        assert pos.break_even_active
        assert pos.stop_loss >= 2400  # moved to entry (break-even)

    def test_time_exit(self):
        """Time exit should close position after max holding bars."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100,
                               max_holding_bars=2)
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="BUY", fill_price=2400, requested_price=2400,
            size=1.0, commission=0.0,
        )
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2390, 2450)

        # Bar 1 — no exit
        candle1 = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2400, 2405, 2398, 2402)
        exits1 = portfolio.update_positions(candle1)
        assert len(exits1) == 0

        # Bar 2 — time exit
        candle2 = make_candle(datetime(2025, 1, 1, 0, 10, tzinfo=timezone.utc), 2402, 2408, 2400, 2405)
        exits2 = portfolio.update_positions(candle2)
        assert len(exits2) == 1
        assert exits2[0].exit_reason == "TIME_EXIT"

    def test_equity_tracking(self):
        """Equity should track balance + unrealized PnL."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100)
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="BUY", fill_price=2400, requested_price=2400,
            size=1.0, commission=0.0,
        )
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2390, 2410)

        # Price goes up (but NOT hitting TP=2410)
        candle = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2405, 2409, 2400, 2408)
        portfolio.update_positions(candle)

        # Position still open, equity = balance + unrealized = 10000 + (2408-2400)*100 = 10800
        assert portfolio.open_position_count == 1
        assert portfolio.equity == pytest.approx(10800, abs=1)

    def test_realized_vs_unrealized(self):
        """Realized PnL and unrealized PnL should be separate."""
        portfolio = Portfolio(initial_balance=10000, contract_size=100)

        # Open position
        fill = FillEvent(
            timestamp=datetime(2025, 1, 1, 0, 0, tzinfo=timezone.utc),
            direction="BUY", fill_price=2400, requested_price=2400,
            size=1.0, commission=0.0,
        )
        portfolio.open_position(fill, "sig1", "STANDARD_SCALPING", 2390, 2410)

        # Price goes up but NOT hitting TP=2410
        candle = make_candle(datetime(2025, 1, 1, 0, 5, tzinfo=timezone.utc), 2405, 2409, 2400, 2408)
        portfolio.update_positions(candle)

        # Position still open — unrealized should be positive, realized should be 0
        assert portfolio.open_position_count == 1
        assert portfolio.unrealized_pnl > 0
        assert portfolio.realized_pnl == 0
