"""Historical data loader for XAUUSD backtesting.

Supports loading from:
- CSV files (for external data like DXY, yields)
- In-memory candle lists (for synthetic/fixture data)
- Future: TimescaleDB/PostgreSQL (when database is available)

All timestamps are normalized to UTC.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timezone
from typing import List, Optional, Dict, Iterator
import csv
import hashlib
import numpy as np


@dataclass
class HistoricalCandle:
    """A single OHLCV candle with UTC timestamp."""
    timestamp: datetime  # UTC
    open: float
    high: float
    low: float
    close: float
    volume: int = 0
    timeframe: str = "M5"
    source: str = "UNKNOWN"

    def __post_init__(self):
        if self.timestamp.tzinfo is None:
            raise ValueError(f"Candle timestamp must be timezone-aware (UTC), got naive: {self.timestamp}")

    @property
    def time_epoch(self) -> float:
        return self.timestamp.timestamp()

    @property
    def time_iso(self) -> str:
        return self.timestamp.isoformat()


@dataclass
class DatasetMetadata:
    """Provenance metadata for a loaded dataset."""
    symbol: str
    timeframe: str
    source: str
    start_time: datetime
    end_time: datetime
    record_count: int
    data_hash: str = ""
    loaded_at: str = field(default_factory=lambda: datetime.now(timezone.utc).isoformat())

    def compute_hash(self, candles: List[HistoricalCandle]) -> str:
        h = hashlib.sha256()
        for c in candles:
            h.update(f"{c.timestamp.isoformat()}{c.open}{c.high}{c.low}{c.close}{c.volume}".encode())
        self.data_hash = h.hexdigest()[:16]
        return self.data_hash


class DataLoader:
    """Loads historical XAUUSD data from various sources."""

    @staticmethod
    def from_csv(filepath: str, symbol: str = "XAUUSD", timeframe: str = "M5",
                 time_col: str = "timestamp", tz: str = "UTC") -> tuple[List[HistoricalCandle], DatasetMetadata]:
        """Load candles from CSV file. All timestamps normalized to UTC."""
        candles = []
        with open(filepath, "r") as f:
            reader = csv.DictReader(f)
            for row in reader:
                ts_str = row.get(time_col, row.get("time", row.get("datetime", "")))
                # Parse timestamp — handle both epoch and ISO
                try:
                    ts = datetime.fromisoformat(ts_str)
                except (ValueError, TypeError):
                    try:
                        ts = datetime.fromtimestamp(float(ts_str), tz=timezone.utc)
                    except (ValueError, TypeError):
                        continue

                if ts.tzinfo is None:
                    ts = ts.replace(tzinfo=timezone.utc)

                candles.append(HistoricalCandle(
                    timestamp=ts,
                    open=float(row["open"]),
                    high=float(row["high"]),
                    low=float(row["low"]),
                    close=float(row["close"]),
                    volume=int(row.get("volume", 0)),
                    timeframe=timeframe,
                    source="CSV",
                ))

        candles.sort(key=lambda c: c.timestamp)
        meta = DatasetMetadata(
            symbol=symbol, timeframe=timeframe, source="CSV",
            start_time=candles[0].timestamp if candles else datetime.min.replace(tzinfo=timezone.utc),
            end_time=candles[-1].timestamp if candles else datetime.min.replace(tzinfo=timezone.utc),
            record_count=len(candles),
        )
        meta.compute_hash(candles)
        return candles, meta

    @staticmethod
    def from_candle_list(candles: list, symbol: str = "XAUUSD", timeframe: str = "M5",
                         source: str = "SYNTHETIC") -> tuple[List[HistoricalCandle], DatasetMetadata]:
        """Load from in-memory candle dicts (for fixtures/synthetic data)."""
        result = []
        for c in candles:
            ts = c.get("timestamp") or c.get("time")
            if isinstance(ts, (int, float)):
                dt = datetime.fromtimestamp(float(ts), tz=timezone.utc)
            elif isinstance(ts, datetime):
                dt = ts if ts.tzinfo else ts.replace(tzinfo=timezone.utc)
            elif isinstance(ts, str):
                dt = datetime.fromisoformat(ts)
                if dt.tzinfo is None:
                    dt = dt.replace(tzinfo=timezone.utc)
            else:
                raise ValueError(f"Cannot parse timestamp: {ts}")

            result.append(HistoricalCandle(
                timestamp=dt,
                open=float(c["open"]),
                high=float(c["high"]),
                low=float(c["low"]),
                close=float(c["close"]),
                volume=int(c.get("volume", 0)),
                timeframe=timeframe,
                source=source,
            ))

        result.sort(key=lambda c: c.timestamp)
        meta = DatasetMetadata(
            symbol=symbol, timeframe=timeframe, source=source,
            start_time=result[0].timestamp if result else datetime.min.replace(tzinfo=timezone.utc),
            end_time=result[-1].timestamp if result else datetime.min.replace(tzinfo=timezone.utc),
            record_count=len(result),
        )
        meta.compute_hash(result)
        return result, meta

    @staticmethod
    def generate_synthetic(symbol: str = "XAUUSD", timeframe: str = "M5",
                           n_candles: int = 1000, base_price: float = 2400.0,
                           volatility: float = 2.0, seed: int = 42,
                           start_time: Optional[datetime] = None) -> tuple[List[HistoricalCandle], DatasetMetadata]:
        """Generate deterministic synthetic XAUUSD candles for testing.

        Clearly labeled as SYNTHETIC — never used as production data.
        """
        if start_time is None:
            start_time = datetime(2025, 1, 1, tzinfo=timezone.utc)

        tf_minutes = {"M1": 1, "M5": 5, "M15": 15, "M30": 30, "H1": 60, "H4": 240, "D1": 1440}
        interval_sec = tf_minutes.get(timeframe, 5) * 60

        rng = np.random.RandomState(seed)
        candles = []
        price = base_price

        for i in range(n_candles):
            ts = datetime.fromtimestamp(start_time.timestamp() + i * interval_sec, tz=timezone.utc)
            change = rng.randn() * volatility
            o = price
            c = price + change
            h = max(o, c) + abs(rng.randn()) * 0.5
            l = min(o, c) - abs(rng.randn()) * 0.5
            candles.append(HistoricalCandle(
                timestamp=ts, open=o, high=h, low=l, close=c,
                volume=int(rng.randint(10, 1000)),
                timeframe=timeframe, source="SYNTHETIC",
            ))
            price = c

        meta = DatasetMetadata(
            symbol=symbol, timeframe=timeframe, source="SYNTHETIC",
            start_time=candles[0].timestamp, end_time=candles[-1].timestamp,
            record_count=len(candles),
        )
        meta.compute_hash(candles)
        return candles, meta

    @staticmethod

    def from_database(db_url: str, symbol: str = "XAUUSD", timeframe: str = "M5",
                      start_time: Optional[datetime] = None,
                      end_time: Optional[datetime] = None,
                      source: Optional[str] = None) -> tuple[list[HistoricalCandle], DatasetMetadata]:
        """Load candles from PostgreSQL market.candles table.

        Args:
            db_url: PostgreSQL connection string
            symbol: Trading symbol (e.g., XAUUSD)
            timeframe: Candle timeframe (M5, M15, H1, H4, D1)
            start_time: Optional start datetime (UTC)
            end_time: Optional end datetime (UTC)
            source: Optional data source filter (e.g., KAGGLE_NOVANDRAANUGRAH_2004_2026)

        Returns:
            Tuple of (candles, metadata) — same format as from_csv.
        """
        import psycopg2

        conn = psycopg2.connect(db_url)

        query = """
            SELECT time, open, high, low, close, volume, source
            FROM market.candles
            WHERE symbol = %s AND timeframe = %s
        """
        params: list = [symbol, timeframe]

        if start_time:
            query += " AND time >= %s"
            params.append(start_time)
        if end_time:
            query += " AND time <= %s"
            params.append(end_time)
        if source:
            query += " AND source = %s"
            params.append(source)

        query += " ORDER BY time ASC"

        with conn.cursor() as cur:
            cur.execute(query, params)
            rows = cur.fetchall()

        conn.close()

        candles = []
        for row in rows:
            ts = row[0]
            if ts.tzinfo is None:
                ts = ts.replace(tzinfo=timezone.utc)

            candles.append(HistoricalCandle(
                timestamp=ts,
                open=float(row[1]),
                high=float(row[2]),
                low=float(row[3]),
                close=float(row[4]),
                volume=int(row[5]) if row[5] else 0,
                timeframe=timeframe,
                source=row[6] if len(row) > 6 else "DATABASE",
            ))

        if not candles:
            meta = DatasetMetadata(
                symbol=symbol, timeframe=timeframe, source="DATABASE",
                start_time=datetime.min.replace(tzinfo=timezone.utc),
                end_time=datetime.min.replace(tzinfo=timezone.utc),
                record_count=0,
            )
            return candles, meta

        meta = DatasetMetadata(
            symbol=symbol, timeframe=timeframe, source=candles[0].source,
            start_time=candles[0].timestamp,
            end_time=candles[-1].timestamp,
            record_count=len(candles),
        )
        meta.compute_hash(candles)
        return candles, meta
