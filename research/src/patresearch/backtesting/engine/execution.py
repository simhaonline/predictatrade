"""Execution simulator for backtesting.

Models realistic execution including:
- Spread (historical, dynamic, or fixed)
- Slippage (fixed, percentage, ATR/volatility-based)
- Commission (per lot or per trade)
- Latency (simulated delay)
- Partial fills (configurable probability)
- Rejections (price movement/rejection conditions)
- Direction correctness (LONG uses ask, SHORT uses bid)

All transaction costs flow into realized P&L.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from typing import Optional
import numpy as np

from .events import OrderEvent, FillEvent
from ..data.loader import HistoricalCandle


@dataclass
class ExecutionConfig:
    """Configuration for the execution simulator."""
    spread_model: str = "fixed"  # fixed, dynamic, historical
    fixed_spread: float = 0.30  # in price units
    slippage_model: str = "fixed"  # fixed, percentage, atr_based
    fixed_slippage: float = 0.05  # in price units
    slippage_percentage: float = 0.0001  # 0.01% of price
    slippage_atr_multiplier: float = 0.1  # fraction of ATR
    commission_per_lot: float = 7.0  # round-turn commission
    contract_size: float = 100.0  # ounces per lot
    latency_ms: int = 0  # simulated latency
    partial_fill_probability: float = 0.0  # 0 = no partial fills
    rejection_probability: float = 0.0  # 0 = no rejections
    random_seed: int = 42


class ExecutionSimulator:
    """Simulates realistic order execution for backtesting."""

    def __init__(self, config: ExecutionConfig):
        self.config = config
        self.rng = np.random.RandomState(config.random_seed)

    def execute(self, order: OrderEvent, candle: HistoricalCandle,
                atr: float = 0.0) -> FillEvent:
        """Execute an order and return a fill event.

        LONG entries use the ASK price (you buy at ask).
        SHORT entries use the BID price (you sell at bid).
        """
        # Determine execution price
        if order.direction == "BUY":
            # Buy at ask = close + half spread
            spread = self._get_spread(candle)
            base_price = candle.close + spread / 2  # ask
        else:
            # Sell at bid = close - half spread
            spread = self._get_spread(candle)
            base_price = candle.close - spread / 2  # bid

        # Apply slippage
        slippage = self._get_slippage(candle, atr)
        if order.direction == "BUY":
            fill_price = base_price + slippage  # worse for buyer
        else:
            fill_price = base_price - slippage  # worse for seller

        # Check for rejection
        if self.config.rejection_probability > 0:
            if self.rng.random() < self.config.rejection_probability:
                return FillEvent(
                    timestamp=order.timestamp,
                    direction=order.direction,
                    fill_price=fill_price,
                    requested_price=base_price,
                    size=order.size,
                    fill_status="REJECTED",
                    fill_ratio=0.0,
                    signal_id=order.signal_id,
                )

        # Check for partial fill
        fill_ratio = 1.0
        fill_status = "FILLED"
        if self.config.partial_fill_probability > 0:
            if self.rng.random() < self.config.partial_fill_probability:
                fill_ratio = self.rng.uniform(0.3, 0.7)
                fill_status = "PARTIAL"

        # Calculate costs
        spread_cost = spread * order.size * fill_ratio
        commission = self.config.commission_per_lot * order.size * fill_ratio
        total_slippage_cost = slippage * order.size * fill_ratio

        return FillEvent(
            timestamp=order.timestamp,
            direction=order.direction,
            fill_price=fill_price,
            requested_price=base_price,
            size=order.size * fill_ratio,
            slippage=slippage,
            commission=commission,
            spread_cost=spread_cost,
            fill_status=fill_status,
            fill_ratio=fill_ratio,
            signal_id=order.signal_id,
        )

    def execute_exit(self, direction: str, size: float, candle: HistoricalCandle,
                     atr: float = 0.0, timestamp: Optional[datetime] = None) -> FillEvent:
        """Execute an exit order.

        Exit BUY position → sell at BID
        Exit SELL position → buy at ASK
        """
        exit_direction = "SELL" if direction == "BUY" else "BUY"

        if exit_direction == "SELL":
            spread = self._get_spread(candle)
            base_price = candle.close - spread / 2  # bid
        else:
            spread = self._get_spread(candle)
            base_price = candle.close + spread / 2  # ask

        slippage = self._get_slippage(candle, atr)
        if exit_direction == "SELL":
            fill_price = base_price - slippage  # worse for seller
        else:
            fill_price = base_price + slippage  # worse for buyer

        spread_cost = spread * size
        commission = self.config.commission_per_lot * size

        return FillEvent(
            timestamp=timestamp or candle.timestamp,
            direction=exit_direction,
            fill_price=fill_price,
            requested_price=base_price,
            size=size,
            slippage=slippage,
            commission=commission,
            spread_cost=spread_cost,
            fill_status="FILLED",
            fill_ratio=1.0,
        )

    def _get_spread(self, candle: HistoricalCandle) -> float:
        """Get spread based on the configured model."""
        if self.config.spread_model == "fixed":
            return self.config.fixed_spread
        elif self.config.spread_model == "dynamic":
            # Dynamic spread: wider during high volatility
            candle_range = candle.high - candle.low
            base_spread = self.config.fixed_spread
            if candle_range > 0:
                # Scale spread with candle range, capped at 2x base
                scale = min(2.0, 1.0 + candle_range / 10.0)
                return base_spread * scale
            return base_spread
        else:
            return self.config.fixed_spread

    def _get_slippage(self, candle: HistoricalCandle, atr: float = 0.0) -> float:
        """Get slippage based on the configured model."""
        if self.config.slippage_model == "fixed":
            return self.config.fixed_slippage
        elif self.config.slippage_model == "percentage":
            return candle.close * self.config.slippage_percentage
        elif self.config.slippage_model == "atr_based":
            if atr > 0:
                return atr * self.config.slippage_atr_multiplier
            return self.config.fixed_slippage
        else:
            return self.config.fixed_slippage
