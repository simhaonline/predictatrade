"""Portfolio/position engine for backtesting.

Tracks:
- Starting balance, cash, balance, equity
- Realized P&L, unrealized P&L
- Current positions (entry, volume, direction, SL, TP)
- Trailing stop, break-even state
- Trade duration, commissions, slippage
- Equity curve, drawdown

Conservative same-bar SL/TP policy:
When both SL and TP would be hit in the same candle, assume SL is hit first
(conservative assumption) unless finer tick data resolves the order.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import List, Optional, Dict
import uuid

from .events import FillEvent, ExitEvent
from ..data.loader import HistoricalCandle


@dataclass
class TradeRecord:
    """A complete trade record for analytics."""
    trade_id: str
    signal_id: str
    strategy_id: str
    direction: str  # BUY, SELL
    entry_time: datetime
    entry_price: float
    exit_time: Optional[datetime] = None
    exit_price: float = 0.0
    exit_reason: str = ""
    size: float = 0.0
    stop_loss: float = 0.0
    take_profit: float = 0.0
    pnl: float = 0.0
    pnl_r: float = 0.0
    commission: float = 0.0
    slippage_cost: float = 0.0
    spread_cost: float = 0.0
    mae: float = 0.0  # max adverse excursion
    mfe: float = 0.0  # max favorable excursion
    duration_bars: int = 0
    duration_seconds: float = 0.0
    regime: str = ""
    session: str = ""
    confluence: float = 0.0
    confidence: float = 0.0
    setup_grade: str = ""


@dataclass
class Position:
    """An open position."""
    position_id: str
    signal_id: str
    strategy_id: str
    direction: str  # BUY, SELL
    size: float
    entry_price: float
    entry_time: datetime
    stop_loss: float
    take_profit: float
    tp1: float = 0.0
    tp2: float = 0.0
    tp3: float = 0.0
    trailing_stop: float = 0.0
    trailing_active: bool = False
    break_even_active: bool = False
    break_even_price: float = 0.0
    commission: float = 0.0
    slippage_cost: float = 0.0
    spread_cost: float = 0.0
    mae: float = 0.0
    mfe: float = 0.0
    bars_held: int = 0
    regime: str = ""
    session: str = ""
    confluence: float = 0.0
    confidence: float = 0.0
    setup_grade: str = ""

    @property
    def is_long(self) -> bool:
        return self.direction == "BUY"

    def unrealized_pnl(self, current_price: float, contract_size: float = 100.0) -> float:
        """Calculate unrealized PnL."""
        if self.is_long:
            return (current_price - self.entry_price) * self.size * contract_size
        else:
            return (self.entry_price - current_price) * self.size * contract_size

    def check_sl_tp(self, candle: HistoricalCandle, conservative: bool = True) -> Optional[tuple]:
        """Check if SL or TP is hit by this candle.

        Returns (exit_reason, exit_price) or None.

        Conservative same-bar policy: if both SL and TP would be hit,
        assume SL is hit first (worst case).
        """
        sl_hit = False
        tp_hit = False
        exit_price = 0.0

        if self.is_long:
            # SL: price goes below stop_loss
            if candle.low <= self.stop_loss:
                sl_hit = True
            # TP: price goes above take_profit
            if candle.high >= self.take_profit:
                tp_hit = True

            if sl_hit and tp_hit:
                if conservative:
                    # Assume SL hit first (conservative)
                    return ("STOP_LOSS", self.stop_loss)
                else:
                    # Non-conservative: assume TP hit first
                    return ("TAKE_PROFIT", self.take_profit)
            elif sl_hit:
                return ("STOP_LOSS", self.stop_loss)
            elif tp_hit:
                return ("TAKE_PROFIT", self.take_profit)
        else:
            # Short position
            # SL: price goes above stop_loss
            if candle.high >= self.stop_loss:
                sl_hit = True
            # TP: price goes below take_profit
            if candle.low <= self.take_profit:
                tp_hit = True

            if sl_hit and tp_hit:
                if conservative:
                    return ("STOP_LOSS", self.stop_loss)
                else:
                    return ("TAKE_PROFIT", self.take_profit)
            elif sl_hit:
                return ("STOP_LOSS", self.stop_loss)
            elif tp_hit:
                return ("TAKE_PROFIT", self.take_profit)

        return None

    def update_mae_mfe(self, candle: HistoricalCandle, contract_size: float = 100.0):
        """Update maximum adverse/favorable excursion."""
        if self.is_long:
            adverse = self.entry_price - candle.low  # how far below entry
            favorable = candle.high - self.entry_price  # how far above entry
        else:
            adverse = candle.high - self.entry_price
            favorable = self.entry_price - candle.low

        if adverse > self.mae:
            self.mae = adverse
        if favorable > self.mfe:
            self.mfe = favorable

    def update_trailing_stop(self, candle: HistoricalCandle, atr: float = 0.0,
                               atr_multiplier: float = 2.0):
        """Update trailing stop based on ATR."""
        if atr <= 0:
            return

        trail_distance = atr * atr_multiplier

        if self.is_long:
            new_stop = candle.close - trail_distance
            if new_stop > self.trailing_stop:
                self.trailing_stop = new_stop
                self.trailing_active = True
                # Also update stop_loss if trailing is tighter
                if self.trailing_stop > self.stop_loss:
                    self.stop_loss = self.trailing_stop
        else:
            new_stop = candle.close + trail_distance
            if self.trailing_stop == 0 or new_stop < self.trailing_stop:
                self.trailing_stop = new_stop
                self.trailing_active = True
                if self.trailing_stop < self.stop_loss:
                    self.stop_loss = self.trailing_stop

    def check_break_even(self, candle: HistoricalCandle, trigger_r: float = 1.0):
        """Move stop to break-even when price moves trigger_r * R in favor."""
        risk = abs(self.entry_price - self.stop_loss)
        if risk <= 0:
            return

        if self.is_long:
            if candle.high >= self.entry_price + trigger_r * risk:
                if not self.break_even_active:
                    self.break_even_active = True
                    self.break_even_price = self.entry_price
                    self.stop_loss = max(self.stop_loss, self.entry_price)
        else:
            if candle.low <= self.entry_price - trigger_r * risk:
                if not self.break_even_active:
                    self.break_even_active = True
                    self.break_even_price = self.entry_price
                    self.stop_loss = min(self.stop_loss, self.entry_price)


@dataclass
class EquityPoint:
    """A single point on the equity curve."""
    timestamp: datetime
    equity: float
    balance: float
    unrealized_pnl: float
    drawdown: float


class Portfolio:
    """Deterministic portfolio engine tracking all positions and equity."""

    def __init__(self, initial_balance: float = 10000.0, contract_size: float = 100.0,
                 max_holding_bars: int = 0, trailing_stop_enabled: bool = True,
                 trailing_atr_multiplier: float = 2.0, break_even_enabled: bool = True,
                 break_even_trigger_r: float = 1.0, conservative_sl_tp: bool = True):
        self.initial_balance = initial_balance
        self.balance = initial_balance
        self.equity = initial_balance
        self.contract_size = contract_size
        self.cash = initial_balance
        self.realized_pnl = 0.0
        self.unrealized_pnl = 0.0

        self.positions: Dict[str, Position] = {}  # position_id -> Position
        self.closed_trades: List[TradeRecord] = []
        self.equity_curve: List[EquityPoint] = []

        self.max_holding_bars = max_holding_bars
        self.trailing_stop_enabled = trailing_stop_enabled
        self.trailing_atr_multiplier = trailing_atr_multiplier
        self.break_even_enabled = break_even_enabled
        self.break_even_trigger_r = break_even_trigger_r
        self.conservative_sl_tp = conservative_sl_tp

        self.max_equity = initial_balance
        self.max_drawdown = 0.0

    def open_position(self, fill: FillEvent, signal_id: str, strategy_id: str,
                      stop_loss: float, take_profit: float,
                      tp1: float = 0.0, tp2: float = 0.0, tp3: float = 0.0,
                      regime: str = "", session: str = "",
                      confluence: float = 0.0, confidence: float = 0.0,
                      setup_grade: str = "") -> str:
        """Open a new position from a fill event."""
        pos_id = str(uuid.uuid4())
        pos = Position(
            position_id=pos_id, signal_id=signal_id, strategy_id=strategy_id,
            direction=fill.direction, size=fill.size, entry_price=fill.fill_price,
            entry_time=fill.timestamp, stop_loss=stop_loss, take_profit=take_profit,
            tp1=tp1, tp2=tp2, tp3=tp3,
            commission=fill.commission, slippage_cost=fill.slippage * fill.size,
            spread_cost=fill.spread_cost,
            regime=regime, session=session,
            confluence=confluence, confidence=confidence, setup_grade=setup_grade,
        )
        self.positions[pos_id] = pos
        return pos_id

    def update_positions(self, candle: HistoricalCandle, atr: float = 0.0) -> List[ExitEvent]:
        """Update all open positions with the latest candle.

        Order of operations (CRITICAL for correctness):
        1. Check SL/TP with the ORIGINAL stop_loss (before trailing/break-even modify it)
        2. If position survives, update trailing stop and break-even for NEXT bar
        3. Check time exit
        Returns list of exit events for closed positions.
        """
        exits = []
        to_close = []

        for pos_id, pos in self.positions.items():
            pos.bars_held += 1
            pos.update_mae_mfe(candle, self.contract_size)

            # 1. Check SL/TP FIRST (with original stop_loss, before trailing/break-even)
            exit_result = pos.check_sl_tp(candle, self.conservative_sl_tp)
            if exit_result:
                exit_reason, exit_price = exit_result
                to_close.append((pos_id, exit_reason, exit_price))
                continue

            # 2. Update trailing stop (for surviving positions — applies next bar)
            if self.trailing_stop_enabled and atr > 0:
                pos.update_trailing_stop(candle, atr, self.trailing_atr_multiplier)

            # 3. Check break-even (for surviving positions — applies next bar)
            if self.break_even_enabled:
                pos.check_break_even(candle, self.break_even_trigger_r)

            # 4. Check time exit
            if self.max_holding_bars > 0 and pos.bars_held >= self.max_holding_bars:
                to_close.append((pos_id, "TIME_EXIT", candle.close))
                continue

        # Close positions
        for pos_id, exit_reason, exit_price in to_close:
            pos = self.positions[pos_id]
            exit_event = self._close_position(pos, exit_reason, exit_price, candle)
            exits.append(exit_event)

        # Update equity
        self._update_equity(candle)
        return exits

    def _close_position(self, pos: Position, exit_reason: str, exit_price: float,
                         candle: HistoricalCandle) -> ExitEvent:
        """Close a position and record the trade."""
        # Calculate PnL
        if pos.is_long:
            gross_pnl = (exit_price - pos.entry_price) * pos.size * self.contract_size
        else:
            gross_pnl = (pos.entry_price - exit_price) * pos.size * self.contract_size

        # Subtract costs
        total_commission = pos.commission  # entry commission
        total_slippage = pos.slippage_cost
        total_spread = pos.spread_cost
        total_costs = total_commission + total_slippage + total_spread

        net_pnl = gross_pnl - total_costs

        # Calculate R multiple
        risk = abs(pos.entry_price - pos.stop_loss)
        pnl_r = net_pnl / (risk * pos.size * self.contract_size) if risk > 0 else 0.0

        # Create trade record
        trade = TradeRecord(
            trade_id=pos.position_id, signal_id=pos.signal_id,
            strategy_id=pos.strategy_id, direction=pos.direction,
            entry_time=pos.entry_time, entry_price=pos.entry_price,
            exit_time=candle.timestamp, exit_price=exit_price,
            exit_reason=exit_reason, size=pos.size,
            stop_loss=pos.stop_loss, take_profit=pos.take_profit,
            pnl=net_pnl, pnl_r=pnl_r,
            commission=total_commission, slippage_cost=total_slippage,
            spread_cost=total_spread, mae=pos.mae, mfe=pos.mfe,
            duration_bars=pos.bars_held,
            duration_seconds=(candle.timestamp - pos.entry_time).total_seconds(),
            regime=pos.regime, session=pos.session,
            confluence=pos.confluence, confidence=pos.confidence,
            setup_grade=pos.setup_grade,
        )
        self.closed_trades.append(trade)
        self.realized_pnl += net_pnl
        self.balance += net_pnl

        # Remove position
        del self.positions[pos.position_id]

        return ExitEvent(
            timestamp=candle.timestamp, position_id=pos.position_id,
            exit_price=exit_price, exit_reason=exit_reason,
            realized_pnl=net_pnl, commission=total_commission,
            slippage=total_slippage,
        )

    def close_all_positions(self, candle: HistoricalCandle) -> List[ExitEvent]:
        """Close all open positions at market (end of backtest)."""
        exits = []
        for pos_id in list(self.positions.keys()):
            pos = self.positions[pos_id]
            exit_event = self._close_position(pos, "END_OF_BACKTEST", candle.close, candle)
            exits.append(exit_event)
        self._update_equity(candle)
        return exits

    def _update_equity(self, candle: HistoricalCandle):
        """Update equity, unrealized PnL, and drawdown."""
        unrealized = 0.0
        for pos in self.positions.values():
            unrealized += pos.unrealized_pnl(candle.close, self.contract_size)

        self.unrealized_pnl = unrealized
        self.equity = self.balance + unrealized

        if self.equity > self.max_equity:
            self.max_equity = self.equity

        drawdown = self.max_equity - self.equity
        if drawdown > self.max_drawdown:
            self.max_drawdown = drawdown

        self.equity_curve.append(EquityPoint(
            timestamp=candle.timestamp, equity=self.equity,
            balance=self.balance, unrealized_pnl=unrealized,
            drawdown=drawdown,
        ))

    @property
    def open_position_count(self) -> int:
        return len(self.positions)

    @property
    def total_exposure(self) -> float:
        """Total position exposure in lots."""
        return sum(p.size for p in self.positions.values())
