"""
Predict-A-Trade Reference Math Library (SOW Section 134)
Canonical quantitative math implementations for the Python research plane.
These must match the Go production implementations (SOW Section 137 — parity).
"""
import math
from collections.abc import Sequence

import numpy as np


def gross_rr(entry: float, stop_loss: float, take_profit: float) -> float:
    """Gross R:R — SOW Section 134.1."""
    target_dist = abs(take_profit - entry)
    stop_dist = abs(entry - stop_loss)
    if stop_dist == 0:
        return 0.0
    return target_dist / stop_dist


def net_rr(entry: float, stop_loss: float, take_profit: float, round_trip_cost: float) -> float:
    """Net (cost-adjusted) R:R — SOW Section 134.1."""
    target_dist = abs(take_profit - entry)
    stop_dist = abs(entry - stop_loss)
    numerator = target_dist - round_trip_cost
    denominator = stop_dist + round_trip_cost
    if denominator == 0:
        return 0.0
    return numerator / denominator


def expectancy(p_win: float, avg_win_r: float, avg_loss_r: float) -> float:
    """Expected R value — SOW Section 134.2."""
    p_loss = 1 - p_win
    return p_win * avg_win_r - p_loss * avg_loss_r


def profit_factor(gross_profit: float, gross_loss: float) -> float:
    """Profit factor — SOW Section 134.2."""
    if gross_loss == 0:
        return float('inf')
    return gross_profit / abs(gross_loss)


def cost_to_target(entry: float, tp1: float, round_trip_cost: float) -> float:
    """Cost-to-target ratio — SOW Section 134.11."""
    target_dist = abs(tp1 - entry)
    if target_dist == 0:
        return 0.0
    return round_trip_cost / target_dist


def wilson_interval(successes: int, total: int, z: float = 1.96) -> tuple[float, float]:
    """Wilson score confidence interval — SOW Section 134.7."""
    if total == 0:
        return 0.0, 1.0
    n = float(total)
    p_hat = successes / n
    z2 = z * z
    denominator = 1 + z2 / n
    center = (p_hat + z2 / (2 * n)) / denominator
    spread = z * math.sqrt(p_hat * (1 - p_hat) / n + z2 / (4 * n * n)) / denominator
    return center - spread, center + spread


def brier_score(probabilities: Sequence[float], outcomes: Sequence[bool]) -> float:
    """Brier score — SOW Section 134.6."""
    if len(probabilities) != len(outcomes) or len(probabilities) == 0:
        return 0.0
    total = 0.0
    for p, outcome in zip(probabilities, outcomes):
        o = 1.0 if outcome else 0.0
        total += (p - o) ** 2
    return total / len(probabilities)


def ece(bin_counts: Sequence[int], bin_mean_forecasts: Sequence[float],
        bin_observed_freqs: Sequence[float]) -> float:
    """Expected Calibration Error — SOW Section 134.6."""
    total = sum(bin_counts)
    if total == 0:
        return 0.0
    result = 0.0
    for count, mean_forecast, observed_freq in zip(bin_counts, bin_mean_forecasts, bin_observed_freqs):
        weight = count / total
        result += weight * abs(observed_freq - mean_forecast)
    return result


def mtf_alignment_score(weights: Sequence[float], states: Sequence[int]) -> float:
    """Multi-timeframe alignment score — SOW Section 134.10."""
    if len(weights) != len(states) or len(weights) == 0:
        return 0.0
    weight_sum = sum(weights)
    if weight_sum == 0:
        return 0.0
    weighted_state = sum(w * s for w, s in zip(weights, states))
    return 100.0 * weighted_state / weight_sum


def atr(highs: Sequence[float], lows: Sequence[float], closes: Sequence[float], period: int) -> float:
    """Average True Range using Wilder smoothing - SOW Section 132, prompt.md Section 1.4.

    First ATR = mean(TR, period)
    Subsequent: ATR_t = (ATR_{t-1} * (period - 1) + TR_t) / period
    """
    if len(highs) <= period or period <= 0:
        return 0.0
    # Compute TR series
    trs = [highs[0] - lows[0]]  # First TR = H-L (no prev close)
    for i in range(1, len(highs)):
        trs.append(true_range(highs[i], lows[i], closes[i - 1]))
    # Wilder smoothing: seed with simple average, then recursive
    atr_val = sum(trs[:period]) / period
    for i in range(period, len(trs)):
        atr_val = (atr_val * (period - 1) + trs[i]) / period
    return atr_val

def true_range(high: float, low: float, prev_close: float) -> float:
    """True Range for a single bar."""
    hl = abs(high - low)
    hc = abs(high - prev_close)
    lc = abs(low - prev_close)
    return max(hl, hc, lc)


def ema(values: Sequence[float], period: int) -> float:
    """Exponential Moving Average."""
    if len(values) == 0 or period <= 0:
        return 0.0
    multiplier = 2 / (period + 1)
    result = values[0]
    for i in range(1, len(values)):
        result = values[i] * multiplier + result * (1 - multiplier)
    return result


def rsi(closes: Sequence[float], period: int) -> float:
    """Relative Strength Index using Wilder smoothing - prompt.md Section 1.3.

    First avg_gain/avg_loss = simple mean of first `period` values.
    Subsequent: avg_gain_t = (avg_gain_{t-1} * (period-1) + gain_t) / period
    If avg_loss is zero -> RSI = 100. If both zero -> RSI = 50 (undefined).
    """
    if len(closes) <= period or period <= 0:
        return 50.0
    # Compute gains and losses for all bars
    gains = []
    losses = []
    for i in range(1, len(closes)):
        change = closes[i] - closes[i - 1]
        gains.append(max(change, 0))
        losses.append(max(-change, 0))
    # Seed: simple average of first `period` gains/losses
    avg_gain = sum(gains[:period]) / period
    avg_loss = sum(losses[:period]) / period
    # Wilder recursive smoothing for remaining values
    for i in range(period, len(gains)):
        avg_gain = (avg_gain * (period - 1) + gains[i]) / period
        avg_loss = (avg_loss * (period - 1) + losses[i]) / period
    if avg_loss == 0:
        if avg_gain == 0:
            return 50.0  # Flat price - undefined
        return 100.0
    rs = avg_gain / avg_loss
    return 100 - (100 / (1 + rs))

def monte_carlo_drawdown(returns: Sequence[float], n_paths: int = 10000, seed: int = 42) -> dict:
    """Monte Carlo drawdown simulation — SOW Section 134.9."""
    rng = np.random.RandomState(seed)
    returns = np.array(returns)
    n = len(returns)

    max_dds = []
    for _ in range(n_paths):
        sampled = rng.choice(returns, size=n, replace=True)
        cumulative = np.cumsum(sampled)
        running_max = np.maximum.accumulate(cumulative)
        drawdowns = running_max - cumulative
        max_dds.append(np.max(drawdowns) if len(drawdowns) > 0 else 0)

    max_dds = np.array(max_dds)
    return {
        'median': float(np.median(max_dds)),
        'p90': float(np.percentile(max_dds, 90)),
        'p95': float(np.percentile(max_dds, 95)),
        'p99': float(np.percentile(max_dds, 99)),
        'worst': float(np.max(max_dds)),
        'n_paths': n_paths,
        'seed': seed,
    }


def sharpe_ratio(returns: Sequence[float], risk_free: float = 0.0) -> float:
    """Sharpe ratio — SOW Section 134.3."""
    returns = np.array(returns)
    if len(returns) < 2:
        return 0.0
    excess = returns - risk_free
    std = np.std(excess, ddof=1)
    if std == 0:
        return 0.0
    return float(np.mean(excess) / std)


def sortino_ratio(returns: Sequence[float], target: float = 0.0) -> float:
    """Sortino ratio — SOW Section 134.3."""
    returns = np.array(returns)
    if len(returns) < 2:
        return 0.0
    excess = returns - target
    downside = excess[excess < 0]
    if len(downside) == 0:
        return float('inf')
    downside_std = np.std(downside, ddof=1)
    if downside_std == 0:
        return 0.0
    return float(np.mean(excess) / downside_std)
