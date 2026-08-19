"""Dataset import and provenance — SOW Section 137
"""
import json
from dataclasses import dataclass, field
from datetime import datetime
from typing import List, Optional
import numpy as np

@dataclass
class DataProvenance:
    source: str
    symbol: str
    timeframe: str
    start_date: str
    end_date: str
    record_count: int
    import_timestamp: str = field(default_factory=lambda: datetime.utcnow().isoformat())
    checksum: str = ""
    quality_notes: str = ""

@dataclass
class TickData:
    timestamp: float
    bid: float
    ask: float
    volume: int = 0

def import_csv_candles(filepath: str, symbol: str = "XAUUSD", timeframe: str = "M5") -> tuple:
    """Import candles from CSV file with provenance tracking."""
    import csv
    candles = []
    with open(filepath, 'r') as f:
        reader = csv.DictReader(f)
        for row in reader:
            candles.append({
                'time': float(row.get('time', row.get('timestamp', 0))),
                'open': float(row['open']),
                'high': float(row['high']),
                'low': float(row['low']),
                'close': float(row['close']),
                'volume': int(row.get('volume', 0)),
            })
    provenance = DataProvenance(
        source="CSV_IMPORT",
        symbol=symbol,
        timeframe=timeframe,
        start_date=str(candles[0]['time']) if candles else "",
        end_date=str(candles[-1]['time']) if candles else "",
        record_count=len(candles),
    )
    return candles, provenance

def generate_synthetic_candles(n: int = 1000, base_price: float = 2430.0, volatility: float = 2.0):
    """Generate synthetic XAUUSD candles for testing (clearly labeled as ESTIMATED)."""
    rng = np.random.RandomState(42)
    candles = []
    price = base_price
    for i in range(n):
        change = rng.randn() * volatility
        open_price = price
        close_price = price + change
        high = max(open_price, close_price) + abs(rng.randn()) * 0.5
        low = min(open_price, close_price) - abs(rng.randn()) * 0.5
        candles.append({
            'time': float(i * 300),  # 5-minute intervals
            'open': open_price,
            'high': high,
            'low': low,
            'close': close_price,
            'volume': int(rng.randint(10, 1000)),
        })
        price = close_price
    return candles
