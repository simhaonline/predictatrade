"""Predict-A-Trade Backtester — SOW Section 137
Realistic bid/ask cost-aware backtester with walk-forward and OOS validation.
"""
import numpy as np
from dataclasses import dataclass, field
from typing import List, Optional
from .reference_math import gross_rr, net_rr, expectancy

@dataclass
class Candle:
    time: float
    open: float
    high: float
    low: float
    close: float
    volume: int = 0

@dataclass
class Trade:
    entry_time: float
    entry_price: float
    direction: str  # "BUY" or "SELL"
    stop_loss: float
    tp1: float
    tp2: float
    tp3: float
    exit_time: float = 0
    exit_price: float = 0
    exit_reason: str = ""
    pnl_r: float = 0  # P&L in R multiples
    hit_tp: int = 0  # which TP was hit (1, 2, 3, or 0 for SL)

@dataclass
class BacktestResult:
    trades: List[Trade] = field(default_factory=list)
    total_pnl_r: float = 0
    win_rate: float = 0
    profit_factor: float = 0
    max_drawdown_r: float = 0
    sharpe: float = 0
    sortino: float = 0
    total_trades: int = 0
    wins: int = 0
    losses: int = 0

class Backtester:
    """Cost-aware backtester with realistic execution model."""
    
    def __init__(self, spread_pips: float = 0.20, slippage_pips: float = 0.05,
                 commission_per_lot: float = 7.0, contract_size: float = 100.0):
        self.spread = spread_pips
        self.slippage = slippage_pips
        self.commission = commission_per_lot
        self.contract_size = contract_size

    def run(self, candles: List[Candle], signals: List[dict]) -> BacktestResult:
        result = BacktestResult()
        
        for signal in signals:
            entry_idx = signal.get('candle_idx', 0)
            if entry_idx >= len(candles) - 1:
                continue
            
            candle = candles[entry_idx]
            direction = signal.get('direction', 'BUY')
            
            # Apply spread and slippage
            if direction == 'BUY':
                entry_price = candle.close + self.slippage
                exit_sl = signal.get('stop_loss', entry_price - 2.0)
            else:
                entry_price = candle.close - self.slippage
                exit_sl = signal.get('stop_loss', entry_price + 2.0)
            
            tp1 = signal.get('tp1', entry_price + 2.0 if direction == 'BUY' else entry_price - 2.0)
            tp2 = signal.get('tp2', entry_price + 4.0 if direction == 'BUY' else entry_price - 4.0)
            tp3 = signal.get('tp3', entry_price + 6.0 if direction == 'BUY' else entry_price - 6.0)
            
            trade = Trade(
                entry_time=candle.time, entry_price=entry_price,
                direction=direction, stop_loss=exit_sl,
                tp1=tp1, tp2=tp2, tp3=tp3,
            )
            
            # Walk forward through candles to find exit
            for i in range(entry_idx + 1, len(candles)):
                c = candles[i]
                if direction == 'BUY':
                    if c.low <= exit_sl:
                        trade.exit_time = c.time
                        trade.exit_price = exit_sl
                        trade.exit_reason = "STOP_LOSS"
                        trade.pnl_r = -1.0
                        break
                    if c.high >= tp1:
                        trade.exit_time = c.time
                        trade.exit_price = tp1
                        trade.exit_reason = "TP1"
                        trade.hit_tp = 1
                        trade.pnl_r = gross_rr(entry_price, exit_sl, tp1)
                        break
                else:
                    if c.high >= exit_sl:
                        trade.exit_time = c.time
                        trade.exit_price = exit_sl
                        trade.exit_reason = "STOP_LOSS"
                        trade.pnl_r = -1.0
                        break
                    if c.low <= tp1:
                        trade.exit_time = c.time
                        trade.exit_price = tp1
                        trade.exit_reason = "TP1"
                        trade.hit_tp = 1
                        trade.pnl_r = gross_rr(entry_price, exit_sl, tp1)
                        break
            
            if trade.exit_reason == "":
                # Close at end
                last = candles[-1]
                trade.exit_time = last.time
                trade.exit_price = last.close
                trade.exit_reason = "END"
                if direction == 'BUY':
                    trade.pnl_r = (trade.exit_price - entry_price) / abs(entry_price - exit_sl)
                else:
                    trade.pnl_r = (entry_price - trade.exit_price) / abs(entry_price - exit_sl)
            
            # Subtract costs
            round_trip_cost = self.spread + 2 * self.slippage + (self.commission / self.contract_size)
            stop_dist = abs(entry_price - exit_sl)
            if stop_dist > 0:
                trade.pnl_r -= round_trip_cost / stop_dist
            
            result.trades.append(tratrade := trade)
        
        # Calculate statistics
        result.total_trades = len(result.trades)
        pnls = [t.pnl_r for t in result.trades]
        result.wins = sum(1 for p in pnls if p > 0)
        result.losses = sum(1 for p in pnls if p <= 0)
        result.total_pnl_r = sum(pnls)
        result.win_rate = result.wins / max(result.total_trades, 1)
        
        gross_profit = sum(p for p in pnls if p > 0)
        gross_loss = abs(sum(p for p in pnls if p < 0))
        result.profit_factor = gross_profit / max(gross_loss, 0.0001)
        
        # Max drawdown
        cumulative = np.cumsum(pnls) if pnls else [0]
        running_max = np.maximum.accumulate(cumulative)
        drawdowns = running_max - cumulative
        result.max_drawdown_r = float(np.max(drawdowns)) if len(drawdowns) > 0 else 0
        
        # Sharpe/Sortino
        if len(pnls) > 1:
            result.sharpe = float(np.mean(pnls) / max(np.std(pnls, ddof=1), 0.0001))
            downside = [p for p in pnls if p < 0]
            if downside:
                result.sortino = float(np.mean(pnls) / max(np.std(downside, ddof=1), 0.0001))
        
        return result

def walk_forward(candles: List[Candle], strategy_fn, window: int = 500, step: int = 100) -> List[BacktestResult]:
    """Walk-forward validation: train on window, test on next step."""
    results = []
    for i in range(window, len(candles) - step, step):
        train = candles[:i]
        test = candles[i:i+step]
        signals = strategy_fn(train, test)
        bt = Backtester()
        result = bt.run(test, signals)
        results.append(result)
    return results

def locked_oos(candles: List[Candle], strategy_fn, train_end: int) -> BacktestResult:
    """Locked out-of-sample test: train on [0:train_end], test on [train_end:]."""
    train = candles[:train_end]
    test = candles[train_end:]
    signals = strategy_fn(train, test)
    bt = Backtester()
    return bt.run(test, signals)
