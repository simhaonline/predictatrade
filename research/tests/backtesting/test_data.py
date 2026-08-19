"""Tests for data loading, quality validation, multi-timeframe alignment, no-lookahead."""
import pytest
import numpy as np
from datetime import datetime, timezone, timedelta

from patresearch.backtesting.data.loader import DataLoader, HistoricalCandle
from patresearch.backtesting.data.quality import DataQualityValidator
from patresearch.backtesting.data.alignment import MultiTimeframeAligner
from patresearch.backtesting.data.session_calendar import SessionCalendar


class TestDataLoading:
    def test_generate_synthetic(self):
        """Synthetic data should be generated correctly."""
        candles, meta = DataLoader.generate_synthetic("XAUUSD", "M5", 100, seed=42)
        assert len(candles) == 100
        assert meta.symbol == "XAUUSD"
        assert meta.source == "SYNTHETIC"
        assert meta.record_count == 100

    def test_all_timestamps_utc(self):
        """All timestamps must be UTC."""
        candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 10, seed=42)
        for c in candles:
            assert c.timestamp.tzinfo is not None
            assert c.timestamp.tzinfo == timezone.utc

    def test_timestamps_monotonic(self):
        """Timestamps should be monotonically increasing."""
        candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 50, seed=42)
        for i in range(1, len(candles)):
            assert candles[i].timestamp > candles[i-1].timestamp

    def test_deterministic_seed(self):
        """Same seed should produce same data."""
        c1, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 50, seed=42)
        c2, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 50, seed=42)
        for a, b in zip(c1, c2):
            assert a.close == b.close

    def test_from_candle_list(self):
        """Loading from candle list should work."""
        raw = [
            {"time": 1640995200, "open": 2400, "high": 2405, "low": 2398, "close": 2402, "volume": 100},
            {"time": 1640995500, "open": 2402, "high": 2408, "low": 2400, "close": 2406, "volume": 150},
        ]
        candles, meta = DataLoader.from_candle_list(raw, "XAUUSD", "M5")
        assert len(candles) == 2
        assert candles[0].open == 2400
        assert meta.symbol == "XAUUSD"


class TestDataQuality:
    def test_valid_data_passes(self):
        """Valid synthetic data should pass quality check."""
        candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 100, seed=42)
        validator = DataQualityValidator()
        report = validator.validate(candles, "XAUUSD", "M5")
        assert report.passed
        assert report.quality_score >= 0.8

    def test_empty_data_fails(self):
        """Empty dataset should fail."""
        validator = DataQualityValidator()
        report = validator.validate([], "XAUUSD", "M5")
        assert not report.passed

    def test_negative_prices_detected(self):
        """Negative prices should be detected."""
        raw = [
            {"time": f"2025-01-01T00:{i:02d}:00+00:00", "open": -1, "high": 1, "low": -2, "close": 0, "volume": 10}
            for i in range(5)
        ]
        candles, _ = DataLoader.from_candle_list(raw, "XAUUSD", "M5")
        validator = DataQualityValidator()
        report = validator.validate(candles, "XAUUSD", "M5")
        assert not report.passed
        assert report.errors > 0

    def test_ohlc_inconsistency_detected(self):
        """OHLC inconsistency (high < low) should be detected."""
        raw = [
            {"time": f"2025-01-01T00:{i:02d}:00+00:00", "open": 2400, "high": 2390, "low": 2410, "close": 2405, "volume": 10}
            for i in range(5)
        ]
        candles, _ = DataLoader.from_candle_list(raw, "XAUUSD", "M5")
        validator = DataQualityValidator()
        report = validator.validate(candles, "XAUUSD", "M5")
        assert not report.passed

    def test_duplicate_timestamps_detected(self):
        """Duplicate timestamps should be detected."""
        raw = [
            {"time": "2025-01-01T00:00:00+00:00", "open": 2400, "high": 2405, "low": 2398, "close": 2402, "volume": 100},
            {"time": "2025-01-01T00:00:00+00:00", "open": 2402, "high": 2408, "low": 2400, "close": 2406, "volume": 150},
        ]
        candles, _ = DataLoader.from_candle_list(raw, "XAUUSD", "M5")
        validator = DataQualityValidator()
        report = validator.validate(candles, "XAUUSD", "M5")
        assert report.warnings > 0


class TestMultiTimeframeAlignment:
    def test_no_lookahead_in_alignment(self):
        """Higher TF candles must not look ahead past primary timestamp."""
        # Create M5 and H1 data
        m5_candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 200, seed=42,
                                                        start_time=datetime(2025, 1, 1, tzinfo=timezone.utc))
        h1_candles, _ = DataLoader.generate_synthetic("XAUUSD", "H1", 20, seed=42,
                                                       start_time=datetime(2025, 1, 1, tzinfo=timezone.utc))

        aligner = MultiTimeframeAligner("M5", ["H1"])
        alignments = aligner.align(m5_candles, {"H1": h1_candles})

        # Verify no look-ahead
        assert aligner.verify_no_lookahead(alignments)

    def test_alignment_returns_closed_candles(self):
        """Alignment should return closed higher-TF candles only."""
        m5_candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 50, seed=42,
                                                        start_time=datetime(2025, 1, 1, tzinfo=timezone.utc))
        h1_candles, _ = DataLoader.generate_synthetic("XAUUSD", "H1", 10, seed=42,
                                                       start_time=datetime(2025, 1, 1, tzinfo=timezone.utc))

        aligner = MultiTimeframeAligner("M5", ["H1"])
        alignments = aligner.align(m5_candles, {"H1": h1_candles})

        # First alignments should not have H1 data (H1 hasn't closed yet)
        for align in alignments[:11]:  # first 11 M5 bars = less than 1 H1
            h1 = align.get_closed_candle("H1")
            if h1 is not None:
                # If H1 is present, it must have closed before the M5 timestamp
                h1_close = h1.timestamp + timedelta(hours=1)
                assert h1_close <= align.timestamp, f"Look-ahead: H1 close {h1_close} > M5 {align.timestamp}"

    def test_decision_unchanged_with_future_data(self):
        """Decision at bar N must be unchanged when bars N+1...N+k are modified."""
        m5_candles, _ = DataLoader.generate_synthetic("XAUUSD", "M5", 100, seed=42)
        h1_candles, _ = DataLoader.generate_synthetic("XAUUSD", "H1", 20, seed=42)

        aligner = MultiTimeframeAligner("M5", ["H1"])
        alignments1 = aligner.align(m5_candles[:50], {"H1": h1_candles})

        # Modify future H1 data
        h1_modified = list(h1_candles)
        for i in range(10, len(h1_modified)):
            h1_modified[i] = HistoricalCandle(
                timestamp=h1_modified[i].timestamp,
                open=h1_modified[i].open * 2,
                high=h1_modified[i].high * 2,
                low=h1_modified[i].low * 2,
                close=h1_modified[i].close * 2,
                volume=h1_modified[i].volume,
                timeframe="H1",
                source=h1_modified[i].source,
            )

        alignments2 = aligner.align(m5_candles[:50], {"H1": h1_modified})

        # First 50 M5 bars should have same alignments (future H1 changes don't affect past)
        for i in range(min(50, len(alignments1), len(alignments2))):
            a1 = alignments1[i]
            a2 = alignments2[i]
            h1_1 = a1.get_closed_candle("H1")
            h1_2 = a2.get_closed_candle("H1")
            if h1_1 and h1_2:
                assert h1_1.close == h1_2.close, f"Bar {i}: H1 changed when future modified"


class TestSessionCalendar:
    def test_weekend_detected(self):
        """Weekend should be detected."""
        cal = SessionCalendar()
        saturday = datetime(2025, 1, 4, 12, 0, tzinfo=timezone.utc)  # Saturday
        info = cal.get_session(saturday)
        assert info.is_weekend
        assert info.session == "WEEKEND"

    def test_weekday_not_weekend(self):
        """Weekday should not be weekend."""
        cal = SessionCalendar()
        wednesday = datetime(2025, 1, 1, 12, 0, tzinfo=timezone.utc)  # Wednesday
        info = cal.get_session(wednesday)
        assert not info.is_weekend

    def test_london_session(self):
        """London session should be detected."""
        cal = SessionCalendar()
        # Winter: London = 08:00-17:00 UTC
        london_time = datetime(2025, 1, 6, 10, 0, tzinfo=timezone.utc)  # Monday 10:00 UTC winter
        info = cal.get_session(london_time)
        assert info.session in ("LONDON", "OVERLAP")

    def test_kill_zone(self):
        """Kill zone should be detected at session start."""
        cal = SessionCalendar()
        # London winter kill zone: 08:00-10:00 UTC
        london_kz = datetime(2025, 1, 6, 8, 30, tzinfo=timezone.utc)
        info = cal.get_session(london_kz)
        assert info.is_kill_zone
        assert info.kill_zone_type == "LONDON_KZ"
