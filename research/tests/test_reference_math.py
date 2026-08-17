"""Tests for the reference math library — parity with Go implementations (SOW Section 137)."""
import pytest
import math
from patresearch.reference_math import (
    gross_rr, net_rr, expectancy, profit_factor, cost_to_target,
    wilson_interval, brier_score, ece, mtf_alignment_score,
    atr, true_range, ema, rsi, monte_carlo_drawdown,
    sharpe_ratio, sortino_ratio,
)


class TestGrossRR:
    def test_basic(self):
        assert gross_rr(2430, 2426, 2435) == pytest.approx(1.25)

    def test_zero_stop(self):
        assert gross_rr(2430, 2430, 2435) == 0.0


class TestNetRR:
    def test_basic(self):
        # (5 - 0.5) / (4 + 0.5) = 4.5/4.5 = 1.0
        assert net_rr(2430, 2426, 2435, 0.5) == pytest.approx(1.0)


class TestExpectancy:
    def test_basic(self):
        # 0.55*1.5 - 0.45*1.0 = 0.375
        assert expectancy(0.55, 1.5, 1.0) == pytest.approx(0.375)


class TestWilsonInterval:
    def test_98_of_100(self):
        lower, upper = wilson_interval(98, 100, 1.96)
        assert lower < 0.97  # Wider than naive ±1%
        assert upper > 0.99

    def test_zero_success(self):
        lower, _ = wilson_interval(0, 100, 1.96)
        assert lower == 0.0

    def test_all_success(self):
        _, upper = wilson_interval(100, 100, 1.96)
        assert upper == pytest.approx(1.0, abs=0.001)

    def test_empty(self):
        lower, upper = wilson_interval(0, 0, 1.96)
        assert lower == 0.0
        assert upper == 1.0


class TestBrierScore:
    def test_perfect(self):
        assert brier_score([1.0, 0.0], [True, False]) == pytest.approx(0.0)

    def test_basic(self):
        # (0.1^2 + 0.1^2 + 0.2^2 + 0.3^2) / 4 = 0.0375
        score = brier_score([0.9, 0.1, 0.8, 0.3], [True, False, True, False])
        assert 0.03 < score < 0.04


class TestECE:
    def test_perfect_calibration(self):
        ece_val = ece([10, 10, 10, 10], [0.1, 0.3, 0.5, 0.7], [0.1, 0.3, 0.5, 0.7])
        assert ece_val == pytest.approx(0.0, abs=0.001)


class TestMTFAlignment:
    def test_basic(self):
        # (0.3*1 + 0.3*1 + 0.2*1 + 0.2*(-1)) / 1.0 * 100 = 60
        score = mtf_alignment_score([0.3, 0.3, 0.2, 0.2], [1, 1, 1, -1])
        assert 59 < score < 61


class TestATR:
    def test_nonzero(self):
        highs = [2430, 2432, 2435, 2431, 2433]
        lows = [2428, 2429, 2430, 2427, 2430]
        closes = [2429, 2431, 2432, 2428, 2431]
        assert atr(highs, lows, closes, 4) > 0


class TestRSI:
    def test_rising(self):
        closes = [2430 + i * 0.5 for i in range(20)]
        assert rsi(closes, 14) > 90  # All rising → RSI near 100


class TestMonteCarlo:
    def test_reproducible(self):
        returns = [0.01, -0.02, 0.03, -0.01, 0.02, -0.03, 0.01, 0.02]
        r1 = monte_carlo_drawdown(returns, n_paths=1000, seed=42)
        r2 = monte_carlo_drawdown(returns, n_paths=1000, seed=42)
        assert r1 == r2  # Same seed → same result

    def test_returns_expected_keys(self):
        r = monte_carlo_drawdown([0.01, -0.01], n_paths=100)
        assert 'median' in r
        assert 'p95' in r
        assert 'p99' in r
        assert 'worst' in r
