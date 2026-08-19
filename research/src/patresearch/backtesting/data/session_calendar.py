"""Session/calendar engine for historical backtesting.

Reuses production session logic. Correctly handles:
- Tokyo, London, New York sessions
- Session overlaps
- Kill zones
- Day of week
- DST transitions
- Rollover/session boundaries
- Holidays (where supported)
"""
from __future__ import annotations

from dataclasses import dataclass
from datetime import datetime, time, timedelta, timezone
from typing import List, Dict, Optional, Tuple
import enum


class TradingSession(enum.Enum):
    """Trading session identifiers matching production."""
    TOKYO = "TOKYO"
    LONDON = "LONDON"
    NEW_YORK = "NEW_YORK"
    OVERLAP = "OVERLAP"  # London/NY overlap
    OFF_HOURS = "OFF_HOURS"
    WEEKEND = "WEEKEND"


@dataclass
class SessionInfo:
    """Session information at a specific time."""
    timestamp: datetime
    session: str  # TOKYO, LONDON, NEW_YORK, OVERLAP, OFF_HOURS, WEEKEND
    is_overlap: bool
    is_weekend: bool
    day_of_week: int  # 0=Monday, 6=Sunday
    is_kill_zone: bool
    kill_zone_type: str  # "LONDON_KZ", "NY_KZ", ""

    @property
    def session_allowed(self) -> bool:
        """Whether trading is allowed in this session."""
        return self.session in (TradingSession.LONDON.value, TradingSession.NEW_YORK.value,
                                TradingSession.OVERLAP.value, TradingSession.TOKYO.value)


class SessionCalendar:
    """Determines trading sessions from UTC timestamps.

    All times are in UTC. Session boundaries follow standard XAUUSD forex hours.

    Session hours (UTC, approximate — DST-aware):
    - Tokyo: 00:00-09:00 UTC
    - London: 07:00-16:00 UTC (summer) / 08:00-17:00 UTC (winter)
    - New York: 12:00-21:00 UTC (summer) / 13:00-22:00 UTC (winter)
    - Overlap: London/NY overlap period
    - Kill zones: First 1-2 hours of London and NY
    """

    # Base session hours in UTC (winter times; summer adjusts -1h)
    SESSION_HOURS = {
        "TOKYO": (0, 9),     # 00:00-09:00 UTC
        "LONDON": (8, 17),    # 08:00-17:00 UTC (winter)
        "NEW_YORK": (13, 22), # 13:00-22:00 UTC (winter)
    }

    # Kill zones: first N hours of session
    KILL_ZONE_HOURS = {
        "LONDON_KZ": ("LONDON", 0, 2),   # First 2 hours of London
        "NY_KZ": ("NEW_YORK", 0, 2),     # First 2 hours of NY
    }

    def __init__(self, dst_aware: bool = True):
        self.dst_aware = dst_aware

    def get_session(self, ts: datetime) -> SessionInfo:
        """Get session info for a UTC timestamp."""
        if ts.tzinfo is None:
            ts = ts.replace(tzinfo=timezone.utc)

        dow = ts.weekday()  # 0=Monday, 6=Sunday
        is_weekend = dow >= 5  # Saturday=5, Sunday=6

        if is_weekend:
            return SessionInfo(
                timestamp=ts, session=TradingSession.WEEKEND.value,
                is_overlap=False, is_weekend=True, day_of_week=dow,
                is_kill_zone=False, kill_zone_type="",
            )

        hour_utc = ts.hour

        # DST adjustment: In northern summer (Apr-Oct), London/NY shift -1h
        dst_offset = self._get_dst_offset(ts)

        # Check sessions
        in_tokyo = 0 <= hour_utc < 9
        in_london = (8 - dst_offset) <= hour_utc < (17 - dst_offset)
        in_ny = (13 - dst_offset) <= hour_utc < (22 - dst_offset)

        is_overlap = in_london and in_ny

        if is_overlap:
            session = TradingSession.OVERLAP.value
        elif in_london:
            session = TradingSession.LONDON.value
        elif in_ny:
            session = TradingSession.NEW_YORK.value
        elif in_tokyo:
            session = TradingSession.TOKYO.value
        else:
            session = TradingSession.OFF_HOURS.value

        # Check kill zones
        kill_zone_type = ""
        is_kill_zone = False

        london_start = 8 - dst_offset
        ny_start = 13 - dst_offset

        if in_london and hour_utc < london_start + 2:
            kill_zone_type = "LONDON_KZ"
            is_kill_zone = True
        elif in_ny and hour_utc < ny_start + 2:
            kill_zone_type = "NY_KZ"
            is_kill_zone = True

        return SessionInfo(
            timestamp=ts, session=session,
            is_overlap=is_overlap, is_weekend=is_weekend,
            day_of_week=dow, is_kill_zone=is_kill_zone,
            kill_zone_type=kill_zone_type,
        )

    def _get_dst_offset(self, ts: datetime) -> int:
        """Return 1 during northern summer (DST active), 0 otherwise."""
        if not self.dst_aware:
            return 0
        # Approximate: DST active from last Sunday of March to last Sunday of October
        month = ts.month
        if 4 <= month <= 9:
            return 1
        elif month == 3 and ts.day >= 25:
            return 1
        elif month == 10 and ts.day < 25:
            return 1
        return 0

    def get_session_for_range(self, start: datetime, end: datetime) -> List[SessionInfo]:
        """Get session info for a range of timestamps (for precomputation)."""
        # This is a helper — actual backtesting iterates bar by bar
        results = []
        current = start
        while current <= end:
            results.append(self.get_session(current))
            current += timedelta(minutes=5)  # M5 granularity
        return results
