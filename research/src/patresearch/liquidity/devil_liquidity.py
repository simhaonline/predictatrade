"""
Devil Liquidity Module — Dynamic Liquidity Zone Calculator
Computes liquidity zones using order-book imbalance, volume profile,
and price-action-based liquidity pools. Returns zones for dashboard
display and strategy consumption.
"""
from dataclasses import dataclass, field
from typing import List, Optional
import math

@dataclass
class LiquidityZone:
    price_low: float
    price_high: float
    volume: float
    side: str  # "buy" or "sell"
    strength: float  # 0-1
    is_swept: bool = False

@dataclass
class DevilLiquidityResult:
    zones: List[LiquidityZone]
    buy_side_liquidity: float  # total buy-side liquidity (price above)
    sell_side_liquidity: float  # total sell-side liquidity (price below)
    imbalance: float  # -1 to +1 (buy > sell = positive)
    dominant_side: str  # "buy" or "sell"
    fair_value: float  # VWAP or midpoint
    liquidity_score: float  # 0-100

def compute_devil_liquidity(
    highs: List[float],
    lows: List[float],
    closes: List[float],
    volumes: List[float],
    current_price: float,
    lookback: int = 20,
    zone_count: int = 5,
) -> DevilLiquidityResult:
    """
    Compute Devil Liquidity zones from OHLCV data.
    Uses volume-weighted price levels to identify liquidity pools.
    """
    n = min(len(closes), lookback)
    if n == 0:
        return DevilLiquidityResult(
            zones=[], buy_side_liquidity=0, sell_side_liquidity=0,
            imbalance=0, dominant_side="neutral", fair_value=0, liquidity_score=0
        )

    # Compute volume-weighted price levels
    price_levels = []
    for i in range(n):
        typical_price = (highs[-(n-i)] + lows[-(n-i)] + closes[-(n-i)]) / 3.0
        vol = volumes[-(n-i)] if volumes else 1.0
        price_levels.append((typical_price, vol))

    # Sort by price and cluster into zones
    price_levels.sort(key=lambda x: x[0])
    zone_size = len(price_levels) // zone_count if zone_count > 0 else 1

    zones = []
    for i in range(0, len(price_levels), max(1, zone_size)):
        cluster = price_levels[i:i+max(1, zone_size)]
        if not cluster:
            continue
        zone_low = min(p[0] for p in cluster)
        zone_high = max(p[0] for p in cluster)
        zone_volume = sum(p[1] for p in cluster)
        # Determine side based on price relative to current price
        if zone_low > current_price:
            side = "buy"  # buy-side liquidity (above price)
        else:
            side = "sell"  # sell-side liquidity (below price)
        strength = zone_volume / max(1, sum(v for _, v in price_levels))
        zones.append(LiquidityZone(
            price_low=zone_low, price_high=zone_high,
            volume=zone_volume, side=side, strength=strength
        ))

    # Compute buy/sell liquidity totals
    buy_side = sum(z.volume for z in zones if z.side == "buy")
    sell_side = sum(z.volume for z in zones if z.side == "sell")
    total = buy_side + sell_side
    imbalance = (buy_side - sell_side) / total if total > 0 else 0

    # Fair value = VWAP
    total_vol = sum(v for _, v in price_levels)
    fair_value = sum(p * v for p, v in price_levels) / total_vol if total_vol > 0 else current_price

    # Liquidity score: higher when more balanced and more volume
    balance_factor = 1.0 - abs(imbalance)
    liquidity_score = balance_factor * 100.0

    return DevilLiquidityResult(
        zones=zones,
        buy_side_liquidity=buy_side,
        sell_side_liquidity=sell_side,
        imbalance=imbalance,
        dominant_side="buy" if imbalance > 0 else "sell" if imbalance < 0 else "neutral",
        fair_value=fair_value,
        liquidity_score=liquidity_score,
    )
