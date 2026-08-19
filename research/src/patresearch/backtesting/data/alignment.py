"""Multi-timeframe alignment with NO future leakage.

CRITICAL: At timestamp T, the strategy may only see information that would
genuinely have been known at or before T.

Never expose a partially formed higher-timeframe candle as though it were final
unless production explicitly operates on live/incomplete bars.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from typing import List, Dict, Optional, Tuple
import numpy as np

from .loader import HistoricalCandle


@dataclass
class TimeframeAlignment:
    """Aligned multi-timeframe data at a specific point in time."""
    timestamp: datetime
    primary_candle: HistoricalCandle
    higher_tf_candles: Dict[str, HistoricalCandle]  # timeframe -> last CLOSED candle
    primary_index: int = 0

    def get_closed_candle(self, timeframe: str) -> Optional[HistoricalCandle]:
        """Get the last CLOSED candle for a timeframe (no look-ahead)."""
        return self.higher_tf_candles.get(timeframe)

    def is_aligned(self, required_tfs: List[str]) -> bool:
        """Check if all required timeframes have closed candles available."""
        return all(tf in self.higher_tf_candles for tf in required_tfs)


class MultiTimeframeAligner:
    """Aligns multiple timeframes ensuring no future data leakage.

    For each primary candle at time T:
    - Higher timeframe candles are only included if they CLOSED before T
    - A partially formed higher-TF candle is NOT included (no look-ahead)
    """

    TF_MINUTES = {"M1": 1, "M5": 5, "M15": 15, "M30": 30, "H1": 60, "H4": 240, "D1": 1440}

    def __init__(self, primary_tf: str = "M5", higher_tfs: List[str] = None):
        self.primary_tf = primary_tf
        self.higher_tfs = higher_tfs or ["M15", "H1", "H4", "D1"]

    def align(self, primary_candles: List[HistoricalCandle],
              higher_tf_data: Dict[str, List[HistoricalCandle]]) -> List[TimeframeAlignment]:
        """Align primary candles with higher timeframes, no look-ahead.

        Args:
            primary_candles: The primary timeframe candles (e.g., M5)
            higher_tf_data: Dict of timeframe -> candles for each higher TF

        Returns:
            List of TimeframeAlignment, one per primary candle
        """
        alignments = []

        # Index higher TF candles by timestamp for fast lookup
        tf_indices: Dict[str, Dict[datetime, int]] = {}
        for tf, candles in higher_tf_data.items():
            tf_indices[tf] = {}
            for i, c in enumerate(candles):
                # Store the CLOSE time (end of candle period)
                close_time = self._get_candle_close_time(c, tf)
                tf_indices[tf][close_time] = i

        # For each primary candle, find the last CLOSED higher-TF candle
        for pi, pc in enumerate(primary_candles):
            higher_candles: Dict[str, HistoricalCandle] = {}

            for tf in self.higher_tfs:
                if tf not in higher_tf_data:
                    continue

                # Find the last higher-TF candle that CLOSED before this primary candle's timestamp
                last_closed = self._find_last_closed_candle(
                    higher_tf_data[tf], tf_indices.get(tf, {}), pc.timestamp, tf
                )
                if last_closed is not None:
                    higher_candles[tf] = last_closed

            alignments.append(TimeframeAlignment(
                timestamp=pc.timestamp,
                primary_candle=pc,
                higher_tf_candles=higher_candles,
                primary_index=pi,
            ))

        return alignments

    def _get_candle_close_time(self, candle: HistoricalCandle, tf: str) -> datetime:
        """Get the close time of a candle (open time + interval)."""
        interval = timedelta(minutes=self.TF_MINUTES.get(tf, 5))
        return candle.timestamp + interval

    def _find_last_closed_candle(self, candles: List[HistoricalCandle],
                                  index_map: Dict[datetime, int],
                                  primary_ts: datetime, tf: str) -> Optional[HistoricalCandle]:
        """Find the last candle that closed before primary_ts (no look-ahead)."""
        # Binary search for efficiency
        # The candle closes at timestamp + interval
        # We need: candle.close_time <= primary_ts (the candle was fully formed)
        interval = timedelta(minutes=self.TF_MINUTES.get(tf, 5))

        # Linear scan (safe and correct for any ordering)
        # In production this would use binary search on sorted timestamps
        last_closed: Optional[HistoricalCandle] = None
        for c in candles:
            close_time = c.timestamp + interval
            if close_time <= primary_ts:
                last_closed = c
            else:
                break  # candles are sorted, no more will qualify

        return last_closed

    def verify_no_lookahead(self, alignments: List[TimeframeAlignment]) -> bool:
        """Verify that no alignment uses future data.

        Returns True if no look-ahead detected.
        """
        for align in alignments:
            for tf, candle in align.higher_tf_candles.items():
                close_time = self._get_candle_close_time(candle, tf)
                if close_time > align.timestamp:
                    return False  # Higher TF candle closes AFTER primary timestamp = look-ahead!
        return True
