"""Performance metrics calculation.

Correctly calculates all required metrics with safe handling of edge cases:
- Zero trades, zero losses, zero variance, one-day datasets
- Never returns misleading infinity/NaN without explicit policy
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timedelta
from typing import List, Optional, Dict
import numpy as np
import math

from ..engine.portfolio import TradeRecord, EquityPoint
from ...reference_math import sharpe_ratio as ref_sharpe_ratio, sortino_ratio as ref_sortino_ratio


@dataclass
class PerformanceMetrics:
    """Complete performance metrics for a backtest."""
    # Basic
    initial_balance: float = 0.0
    final_balance: float = 0.0
    net_profit: float = 0.0
    total_return_pct: float = 0.0
    cagr: float = 0.0

    # Risk-adjusted
    sharpe_ratio: float = 0.0
    sortino_ratio: float = 0.0
    max_drawdown_pct: float = 0.0
    calmar_ratio: float = 0.0

    # Trade statistics
    total_trades: int = 0
    wins: int = 0
    losses: int = 0
    win_rate_pct: float = 0.0
    gross_profit: float = 0.0
    gross_loss: float = 0.0
    profit_factor: float = 0.0
    expectancy: float = 0.0
    avg_win: float = 0.0
    avg_loss: float = 0.0
    avg_rr: float = 0.0
    best_trade: float = 0.0
    worst_trade: float = 0.0
    max_consecutive_wins: int = 0
    max_consecutive_losses: int = 0

    # Duration
    avg_holding_time_seconds: float = 0.0
    avg_holding_bars: float = 0.0

    # Risk
    var_95: float = 0.0  # Value at Risk 95%
    longest_losing_streak: int = 0

    # Segmentation (optional)
    by_session: Dict = field(default_factory=dict)
    by_regime: Dict = field(default_factory=dict)
    by_direction: Dict = field(default_factory=dict)
    by_strategy: Dict = field(default_factory=dict)


def calculate_metrics(trades: List[TradeRecord],
                      equity_curve: List,
                      initial_balance: float) -> PerformanceMetrics:
    """Calculate all performance metrics from trade records and equity curve."""
    metrics = PerformanceMetrics(initial_balance=initial_balance)

    if not trades:
        metrics.final_balance = initial_balance
        return metrics

    # Basic metrics
    pnls = [t.pnl for t in trades]
    pnl_rs = [t.pnl_r for t in trades]

    metrics.total_trades = len(trades)
    metrics.net_profit = sum(pnls)
    metrics.final_balance = initial_balance + metrics.net_profit

    if initial_balance > 0:
        metrics.total_return_pct = (metrics.net_profit / initial_balance) * 100

    # CAGR (simplified — based on duration)
    if trades:
        duration = (trades[-1].exit_time - trades[0].entry_time).total_seconds()
        if duration > 0:
            years = duration / (365.25 * 24 * 3600)
            if years > 0 and metrics.total_return_pct != 0:
                ratio = metrics.final_balance / initial_balance
                if ratio > 0:
                    try:
                        # CAGR needs a positive ratio and a sane exponent; a
                        # blown account (ratio<=0) or extreme compounding
                        # (tiny `years`) overflows float math — fall back to
                        # the simple annualized return.
                        metrics.cagr = ((ratio) ** (1 / years) - 1) * 100
                    except OverflowError:
                        metrics.cagr = metrics.total_return_pct
                else:
                    metrics.cagr = metrics.total_return_pct

    # Win/loss
    wins = [p for p in pnls if p > 0]
    losses = [p for p in pnls if p < 0]
    breakevens = [p for p in pnls if p == 0]

    metrics.wins = len(wins)
    metrics.losses = len(losses)
    metrics.win_rate_pct = (metrics.wins / metrics.total_trades * 100) if metrics.total_trades > 0 else 0

    metrics.gross_profit = sum(wins)
    metrics.gross_loss = abs(sum(losses))

    # Profit factor
    if metrics.gross_loss > 0:
        metrics.profit_factor = metrics.gross_profit / metrics.gross_loss
    elif metrics.gross_profit > 0:
        metrics.profit_factor = float('inf')  # No losses — explicitly mark
    else:
        metrics.profit_factor = 0.0

    # Expectancy
    if metrics.total_trades > 0:
        metrics.expectancy = metrics.net_profit / metrics.total_trades

    # Average win/loss
    metrics.avg_win = metrics.gross_profit / metrics.wins if metrics.wins > 0 else 0.0
    metrics.avg_loss = metrics.gross_loss / metrics.losses if metrics.losses > 0 else 0.0

    # Average R:R
    if pnl_rs:
        metrics.avg_rr = np.mean(pnl_rs)

    # Best/worst trade
    metrics.best_trade = max(pnls) if pnls else 0.0
    metrics.worst_trade = min(pnls) if pnls else 0.0

    # Max consecutive wins/losses
    metrics.max_consecutive_wins = _max_consecutive(pnls, positive=True)
    metrics.max_consecutive_losses = _max_consecutive(pnls, positive=False)
    metrics.longest_losing_streak = metrics.max_consecutive_losses

    # Sharpe / Sortino — computed from the equity-curve RETURN SERIES, not from
    # absolute PnL. Absolute PnL inflates both ratios and contradicts the
    # canonical reference_math contracts (SOW 134.3). Must use a returns series
    # (pct change of the equity curve); raw pnls are not returns.
    equity_returns = None
    if equity_curve:
        equities = [ep.equity if hasattr(ep, 'equity') else ep.get('equity', initial_balance) for ep in equity_curve]
        equities = np.asarray(equities, dtype=float)
        if len(equities) >= 2 and np.all(equities[:-1] != 0):
            rets = np.diff(equities) / equities[:-1]
            rets = rets[np.isfinite(rets)]
            if len(rets) >= 2:
                equity_returns = rets

    if equity_returns is not None:
        metrics.sharpe_ratio = ref_sharpe_ratio(equity_returns)
        metrics.sortino_ratio = ref_sortino_ratio(equity_returns)

    # Max drawdown
    if equity_curve:
        equities = [ep.equity if hasattr(ep, 'equity') else ep.get('equity', initial_balance) for ep in equity_curve]
        running_max = np.maximum.accumulate(equities)
        drawdowns = [(1 - e / m) * 100 for e, m in zip(equities, running_max) if m > 0]
        # float() — np.maximum.accumulate yields np.float64 which psycopg2
        # cannot adapt (it renders as np.float64(...) into SQL).
        metrics.max_drawdown_pct = float(max(drawdowns)) if drawdowns else 0.0

    # Calmar ratio
    if metrics.max_drawdown_pct > 0:
        metrics.calmar_ratio = metrics.total_return_pct / metrics.max_drawdown_pct

    # VaR 95%
    if len(pnls) > 10:
        metrics.var_95 = float(np.percentile(pnls, 5))

    # Average holding time
    durations = [t.duration_seconds for t in trades if t.duration_seconds > 0]
    if durations:
        metrics.avg_holding_time_seconds = np.mean(durations)
    bars = [t.duration_bars for t in trades if t.duration_bars > 0]
    if bars:
        metrics.avg_holding_bars = np.mean(bars)

    # Segmentation
    metrics.by_direction = _segment_by(trades, lambda t: t.direction)
    metrics.by_session = _segment_by(trades, lambda t: t.session or "UNKNOWN")
    metrics.by_regime = _segment_by(trades, lambda t: t.regime or "UNKNOWN")
    metrics.by_strategy = _segment_by(trades, lambda t: t.strategy_id)

    return metrics


def _max_consecutive(pnls: List[float], positive: bool = True) -> int:
    """Count max consecutive wins or losses."""
    max_count = 0
    current = 0
    for p in pnls:
        if positive and p > 0:
            current += 1
            max_count = max(max_count, current)
        elif not positive and p < 0:
            current += 1
            max_count = max(max_count, current)
        else:
            current = 0
    return max_count


def _segment_by(trades: List[TradeRecord], key_fn) -> Dict:
    """Segment trades by a key function."""
    groups = {}
    for t in trades:
        key = key_fn(t)
        if key not in groups:
            groups[key] = {"count": 0, "pnl": 0.0, "wins": 0}
        groups[key]["count"] += 1
        groups[key]["pnl"] += t.pnl
        if t.pnl > 0:
            groups[key]["wins"] += 1
    return groups
