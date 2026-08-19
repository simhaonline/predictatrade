"""Base strategy interface for backtesting.

Strategies receive aligned historical data and produce signals.
The base interface ensures all strategies can be swapped in the engine.
"""
from __future__ import annotations

from abc import ABC, abstractmethod
from typing import List, Optional, Dict
from ..data.loader import HistoricalCandle
from ..data.alignment import TimeframeAlignment
from ..engine.events import SignalEvent
from ..engine.portfolio import Portfolio


class BaseStrategy(ABC):
    """Abstract base for backtesting strategies."""

    @property
    @abstractmethod
    def strategy_id(self) -> str:
        """Return the strategy identifier."""
        pass

    @abstractmethod
    def initialize(self, candles: List[HistoricalCandle],
                   higher_tf_data: Optional[Dict[str, List[HistoricalCandle]]] = None):
        """Initialize the strategy with historical data."""
        pass

    @abstractmethod
    def evaluate(self, alignment: TimeframeAlignment, portfolio: Portfolio,
                 engine=None) -> SignalEvent:
        """Evaluate the strategy at the current bar.

        Must produce a SignalEvent. Direction can be BUY, SELL, NO_TRADE, or BLOCKED.
        """
        pass

    def reset(self):
        """Reset strategy state (for walk-forward fold isolation)."""
        pass
