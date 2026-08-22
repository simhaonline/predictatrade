"""
Marnie Fib Indicator — Fibonacci Retracement/Extension Engine
Computes Fibonacci levels from confirmed swing highs/lows using
Marnie's custom rules: 0.236, 0.382, 0.5, 0.618, 0.786, 1.0, 1.272, 1.618.
Returns both retracement and extension levels with confluence scoring.
"""
from dataclasses import dataclass
from typing import List, Optional, Tuple
import math

# Standard Fibonacci ratios
FIB_RATIOS = [0.236, 0.382, 0.5, 0.618, 0.786, 1.0]
EXT_RATIOS = [1.272, 1.618, 2.618]

@dataclass
class FibLevel:
    ratio: float
    price: float
    label: str
    is_retracement: bool

@dataclass
class MarnieFibResult:
    swing_high: float
    swing_low: float
    direction: str  # "bull" or "bear"
    retracement_levels: List[FibLevel]
    extension_levels: List[FibLevel]
    golden_zone_low: float  # 0.618 level
    golden_zone_high: float  # 0.786 level
    confluence_score: float  # 0-100

def compute_marnie_fib(
    swing_high: float,
    swing_low: float,
    current_price: float,
    direction: str = "bull",
    previous_fib_levels: Optional[List[FibLevel]] = None,
) -> MarnieFibResult:
    """
    Compute Marnie Fib retracement and extension levels.
    direction: "bull" = retracement from low to high (buy dips)
               "bear" = retracement from high to low (sell rallies)
    """
    if swing_high <= swing_low:
        swing_high, swing_low = swing_low, swing_high
        direction = "bear" if direction == "bull" else "bull"

    range_val = swing_high - swing_low

    retracement_levels = []
    for r in FIB_RATIOS:
        if direction == "bull":
            price = swing_high - (range_val * r)
            label = f"{r:.1%} Retracement"
        else:
            price = swing_low + (range_val * r)
            label = f"{r:.1%} Retracement"
        retracement_levels.append(FibLevel(ratio=r, price=price, label=label, is_retracement=True))

    extension_levels = []
    for r in EXT_RATIOS:
        if direction == "bull":
            price = swing_high + (range_val * (r - 1.0))
            label = f"{r:.3f} Extension"
        else:
            price = swing_low - (range_val * (r - 1.0))
            label = f"{r:.3f} Extension"
        extension_levels.append(FibLevel(ratio=r, price=price, label=label, is_retracement=False))

    golden_zone_low = swing_high - (range_val * 0.618)
    golden_zone_high = swing_high - (range_val * 0.786)

    # Confluence scoring: price near golden zone = high score
    if golden_zone_low <= current_price <= golden_zone_high:
        confluence_score = 100.0
    else:
        dist_from_golden = min(
            abs(current_price - golden_zone_low),
            abs(current_price - golden_zone_high),
        )
        confluence_score = max(0.0, 100.0 - (dist_from_golden / range_val * 200.0))

    # Add confluence from previous Fib levels
    if previous_fib_levels:
        for prev_level in previous_fib_levels:
            for curr_level in retracement_levels + extension_levels:
                if abs(curr_level.price - prev_level.price) / range_val < 0.02:
                    confluence_score = min(100.0, confluence_score + 10.0)

    return MarnieFibResult(
        swing_high=swing_high,
        swing_low=swing_low,
        direction=direction,
        retracement_levels=retracement_levels,
        extension_levels=extension_levels,
        golden_zone_low=golden_zone_low,
        golden_zone_high=golden_zone_high,
        confluence_score=confluence_score,
    )

def detect_swing_points(highs: List[float], lows: List[float], lookback: int = 5) -> Tuple[float, float]:
    """Detect most recent confirmed swing high and low."""
    if len(highs) < lookback or len(lows) < lookback:
        return max(highs) if highs else 0.0, min(lows) if lows else 0.0
    swing_high = max(highs[-lookback:])
    swing_low = min(lows[-lookback:])
    return swing_high, swing_low
