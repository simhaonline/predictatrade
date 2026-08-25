"""Historical data quality validation.

Performs deterministic validation checks before a backtest begins.
A backtest must fail clearly when data quality is below configured minimum.
"""
from __future__ import annotations

from dataclasses import dataclass, field
from datetime import datetime, timedelta, timezone
from typing import List, Optional, Dict
import numpy as np

from .loader import HistoricalCandle


@dataclass
class QualityIssue:
    """A single data quality issue."""
    severity: str  # ERROR, WARNING, INFO
    category: str  # DUPLICATE, GAP, OHLC_INCONSISTENCY, etc.
    timestamp: Optional[datetime]
    message: str


@dataclass
class DataQualityReport:
    """Complete data quality assessment."""
    symbol: str
    timeframe: str
    total_candles: int
    issues: List[QualityIssue] = field(default_factory=list)
    errors: int = 0
    warnings: int = 0
    warning_counts: Dict[str, int] = field(default_factory=dict)
    passed: bool = True
    min_quality_score: float = 0.0
    quality_score: float = 1.0

    def add_issue(self, severity: str, category: str, message: str,
                  timestamp: Optional[datetime] = None):
        issue = QualityIssue(severity=severity, category=category,
                            timestamp=timestamp, message=message)
        self.issues.append(issue)
        if severity == "ERROR":
            self.errors += 1
            self.passed = False
        elif severity == "WARNING":
            self.warnings += 1
            self.warning_counts[category] = self.warning_counts.get(category, 0) + 1

    def compute_score(self, warn_tolerances: Optional[Dict[str, float]] = None):
        """Compute quality score: 1.0 = perfect, 0.0 = unusable.

        Genuine corruption (ERROR issues) always fails. Warning categories are
        judged by RATE against a per-category tolerance, so normal market
        behavior (weekend gaps, gold's fat-tailed volatility outliers) on
        production-scale datasets does not sink the score. A warning category
        only fails the dataset when its rate materially exceeds tolerance.
        """
        if self.total_candles == 0:
            self.quality_score = 0.0
            return
        tol = warn_tolerances or {}
        penalty = self.errors * 0.1
        for cat, cnt in self.warning_counts.items():
            rate = cnt / self.total_candles
            max_rate = tol.get(cat, 0.01)  # default 1% tolerance
            if rate > max_rate:
                excess = rate - max_rate
                penalty += excess * 5.0
                self.passed = False
        self.quality_score = max(0.0, 1.0 - penalty)
        if self.quality_score < self.min_quality_score:
            self.passed = False

    def summary(self) -> Dict:
        return {
            "symbol": self.symbol,
            "timeframe": self.timeframe,
            "total_candles": self.total_candles,
            "errors": self.errors,
            "warnings": self.warnings,
            "quality_score": round(self.quality_score, 4),
            "passed": self.passed,
            "issue_categories": list(set(i.category for i in self.issues)),
        }


class DataQualityValidator:
    """Validates historical data before backtesting."""

    def __init__(self, max_gap_ratio: float = 2.0, min_quality_score: float = 0.8,
                 max_outlier_std: float = 10.0, allow_weekend_gaps: bool = True,
                 warn_tolerances: Optional[Dict[str, float]] = None):
        self.max_gap_ratio = max_gap_ratio
        self.min_quality_score = min_quality_score
        self.max_outlier_std = max_outlier_std
        self.allow_weekend_gaps = allow_weekend_gaps
        # Per-category WARNING tolerances as a fraction of total bars.
        # Weekend gaps are normal for XAUUSD; allow up to 5%. Outliers /
        # abnormal ranges are expected on a fat-tailed instrument; 1% each.
        self.warn_tolerances = warn_tolerances or {
            "GAP": 0.05,
            "OUTLIER": 0.01,
            "ABNORMAL_RANGE": 0.01,
            "DUPLICATE": 0.001,
        }

    def validate(self, candles: List[HistoricalCandle], symbol: str = "XAUUSD",
                timeframe: str = "M5") -> DataQualityReport:
        """Run all quality checks and return a report."""
        report = DataQualityReport(
            symbol=symbol, timeframe=timeframe,
            total_candles=len(candles),
            min_quality_score=self.min_quality_score,
        )

        if len(candles) == 0:
            report.add_issue("ERROR", "EMPTY", "No candles in dataset")
            report.compute_score()
            return report

        self._check_timestamps(candles, report)
        self._check_utc(candles, report)
        self._check_ordering(candles, report)
        self._check_duplicates(candles, report)
        self._check_ohlc_consistency(candles, report)
        self._check_negative_prices(candles, report)
        self._check_outliers(candles, report)
        self._check_gaps(candles, report, timeframe)
        self._check_spread(candles, report)

        report.compute_score(self.warn_tolerances)
        return report

    def _check_timestamps(self, candles: List[HistoricalCandle], report: DataQualityReport):
        for c in candles:
            if c.timestamp is None:
                report.add_issue("ERROR", "NULL_TIMESTAMP", "Null timestamp found")

    def _check_utc(self, candles: List[HistoricalCandle], report: DataQualityReport):
        for c in candles:
            if c.timestamp.tzinfo is None:
                report.add_issue("ERROR", "NAIVE_TIMESTAMP",
                                f"Non-UTC timestamp at {c.timestamp}", c.timestamp)

    def _check_ordering(self, candles: List[HistoricalCandle], report: DataQualityReport):
        for i in range(1, len(candles)):
            if candles[i].timestamp <= candles[i-1].timestamp:
                report.add_issue("ERROR", "ORDERING",
                                f"Timestamps not monotonically increasing at index {i}",
                                candles[i].timestamp)

    def _check_duplicates(self, candles: List[HistoricalCandle], report: DataQualityReport):
        seen = set()
        for c in candles:
            ts_key = c.timestamp.replace(tzinfo=None)  # compare naive for dedup
            if ts_key in seen:
                report.add_issue("WARNING", "DUPLICATE",
                                f"Duplicate timestamp: {c.timestamp}", c.timestamp)
            seen.add(ts_key)

    def _check_ohlc_consistency(self, candles: List[HistoricalCandle], report: DataQualityReport):
        for i, c in enumerate(candles):
            if c.high < c.low:
                report.add_issue("ERROR", "OHLC_INCONSISTENCY",
                                f"High < Low at {c.timestamp}: H={c.high} L={c.low}",
                                c.timestamp)
            if c.high < max(c.open, c.close):
                report.add_issue("ERROR", "OHLC_INCONSISTENCY",
                                f"High < max(open,close) at {c.timestamp}",
                                c.timestamp)
            if c.low > min(c.open, c.close):
                report.add_issue("ERROR", "OHLC_INCONSISTENCY",
                                f"Low > min(open,close) at {c.timestamp}",
                                c.timestamp)

    def _check_negative_prices(self, candles: List[HistoricalCandle], report: DataQualityReport):
        for c in candles:
            if c.open <= 0 or c.high <= 0 or c.low <= 0 or c.close <= 0:
                report.add_issue("ERROR", "NEGATIVE_PRICE",
                                f"Non-positive price at {c.timestamp}", c.timestamp)

    def _check_outliers(self, candles: List[HistoricalCandle], report: DataQualityReport):
        closes = np.array([c.close for c in candles])
        mean = closes.mean()
        std = closes.std()
        if std > 0:
            for i, c in enumerate(candles):
                z = abs(c.close - mean) / std
                if z > self.max_outlier_std:
                    report.add_issue("WARNING", "OUTLIER",
                                    f"Outlier at {c.timestamp}: close={c.close}, z={z:.1f}",
                                    c.timestamp)

    def _check_gaps(self, candles: List[HistoricalCandle], report: DataQualityReport, timeframe: str):
        tf_minutes = {"M1": 1, "M5": 5, "M15": 15, "M30": 30, "H1": 60, "H4": 240, "D1": 1440}
        expected_interval = tf_minutes.get(timeframe, 5) * 60  # seconds

        for i in range(1, len(candles)):
            gap = (candles[i].timestamp - candles[i-1].timestamp).total_seconds()
            if gap <= 0:
                continue  # handled by ordering check

            ratio = gap / expected_interval
            if ratio > self.max_gap_ratio:
                # Skip the normal XAUUSD weekend closure (Friday close ->
                # Sunday open), which can span up to ~3 days. Skip when the
                # previous bar is Fri/Sat and the next is Sat/Sun/Mon.
                if self.allow_weekend_gaps:
                    prev_day = candles[i-1].timestamp.weekday()
                    curr_day = candles[i].timestamp.weekday()
                    max_weekend_ratio = (3 * 24 * 3600) / expected_interval
                    if prev_day in (4, 5) and curr_day in (5, 6, 0) and ratio <= max_weekend_ratio:
                        continue

                report.add_issue("WARNING", "GAP",
                                f"Large gap at {candles[i].timestamp}: {gap:.0f}s (ratio={ratio:.1f})",
                                candles[i].timestamp)

    def _check_spread(self, candles: List[HistoricalCandle], report: DataQualityReport):
        """Check for abnormal intrabar ranges that suggest bad data."""
        ranges = [c.high - c.low for c in candles]
        if not ranges:
            return
        median_range = np.median(ranges)
        if median_range > 0:
            for c in candles:
                bar_range = c.high - c.low
                if bar_range > median_range * 20:
                    report.add_issue("WARNING", "ABNORMAL_RANGE",
                                    f"Abnormal range at {c.timestamp}: {bar_range:.2f} vs median {median_range:.2f}",
                                    c.timestamp)
