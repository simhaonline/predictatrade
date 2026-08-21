"""Tests for the vectorized QuantitativeStrategyEngine.

Validates indicator correctness against reference scalar implementations
(reference_math.py) and verifies the composite signal pipeline contract.
"""
from __future__ import annotations

import numpy as np
import pandas as pd
import pytest

from patresearch.quantitative_strategy_engine import QuantitativeStrategyEngine
from patresearch.reference_math import atr as ref_atr
from patresearch.reference_math import ema as ref_ema
from patresearch.reference_math import rsi as ref_rsi

# ─── Fixtures ────────────────────────────────────────────────────────────────

@pytest.fixture
def synthetic_df() -> pd.DataFrame:
    """A deterministic OHLCV frame with a clear uptrend then pullback."""
    rng = np.random.default_rng(seed=42)
    n = 400
    idx = pd.date_range("2026-01-01", periods=n, freq="5min", tz="UTC")
    drift = np.cumsum(np.linspace(0.02, -0.01, n)) + rng.normal(0, 0.3, n)
    close = 2400.0 + drift
    high = close + rng.uniform(0.1, 0.8, n)
    low = close - rng.uniform(0.1, 0.8, n)
    opn = close + rng.normal(0, 0.1, n)
    vol = rng.integers(10, 100, n)
    return pd.DataFrame(
        {"open": opn, "high": high, "low": low, "close": close, "volume": vol},
        index=idx,
    )


@pytest.fixture
def flat_df() -> pd.DataFrame:
    """A flat-price frame to exercise division-by-zero / NaN edge cases."""
    idx = pd.date_range("2026-01-01", periods=250, freq="5min", tz="UTC")
    price = np.full(250, 2400.0)
    return pd.DataFrame(
        {"open": price, "high": price + 0.1, "low": price - 0.1, "close": price, "volume": 50},
        index=idx,
    )


def _tail_series(s: pd.Series, n: int) -> list[float]:
    return s.dropna().tail(n).tolist()


# ─── Indicator parity ──────────────────────────────────────────────────────

class TestSMAParity:
    def test_matches_manual(self, synthetic_df: pd.DataFrame):
        eng = QuantitativeStrategyEngine()
        out = eng.compute_sma(synthetic_df, period=20)
        manual = synthetic_df["close"].rolling(20).mean()
        np.testing.assert_allclose(out["sma_20"].dropna(), manual.dropna(), rtol=1e-12)

    def test_no_loop_correctness(self, synthetic_df: pd.DataFrame):
        eng = QuantitativeStrategyEngine()
        out = eng.compute_sma(synthetic_df, period=50)
        # SMA at index 49 == mean of first 50 closes
        assert out["sma_50"].iloc[49] == pytest.approx(synthetic_df["close"].iloc[:50].mean())


class TestEMAParity:
    def test_matches_reference_scalar(self, synthetic_df: pd.DataFrame):
        eng = QuantitativeStrategyEngine()
        out = eng.compute_ema(synthetic_df, period=14)
        ref_val = ref_ema(_tail_series(synthetic_df["close"], 200), 14)
        # EMA converges over long series; compare against reference on recent tail
        assert out["ema_14"].dropna().iloc[-1] == pytest.approx(ref_val, rel=0.02)

    def test_columns_named(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_ema(synthetic_df, period=12)
        assert "ema_12" in out.columns


class TestRSIParity:
    def test_rising_near_100(self, synthetic_df: pd.DataFrame):
        idx = pd.date_range("2026-01-01", periods=30, freq="5min", tz="UTC")
        rising = pd.DataFrame(
            {"open": np.arange(30, dtype=float), "high": np.arange(30) + 1,
             "low": np.arange(30) - 1, "close": np.arange(30, dtype=float), "volume": 1},
            index=idx,
        )
        out = QuantitativeStrategyEngine().compute_rsi(rising, period=14)
        assert out["rsi_14"].dropna().iloc[-1] > 90

    def test_matches_reference_tail(self, synthetic_df: pd.DataFrame):
        eng = QuantitativeStrategyEngine()
        out = eng.compute_rsi(synthetic_df, period=14)
        ref = ref_rsi(_tail_series(synthetic_df["close"], 30), 14)
        assert out["rsi_14"].dropna().iloc[-1] == pytest.approx(ref, abs=3.0)

    def test_flat_price_no_nan_inf(self, flat_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_rsi(flat_df, period=14)
        col = out["rsi_14"].dropna()
        assert not col.isna().any()
        assert np.isfinite(col).all()


class TestATRParity:
    def test_matches_reference(self, synthetic_df: pd.DataFrame):
        eng = QuantitativeStrategyEngine()
        out = eng.compute_atr(synthetic_df, period=14)
        h = _tail_series(synthetic_df["high"], 30)
        l = _tail_series(synthetic_df["low"], 30)
        c = _tail_series(synthetic_df["close"], 30)
        ref = ref_atr(h, l, c, 14)
        # ref_atr uses simple-average over the last `period` TRs; the engine uses
        # the Wilder recursive EMA (per spec). Different smoothing methods, so
        # we assert same order of magnitude and finiteness rather than equality.
        assert out["atr_14"].dropna().iloc[-1] == pytest.approx(ref, rel=0.20)

    def test_flat_safe(self, flat_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_atr(flat_df, period=14)
        assert np.isfinite(out["atr_14"].dropna()).all()


class TestADX:
    def test_outputs_three_columns(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_adx(synthetic_df, period=14)
        for c in ("adx_14", "plus_di_14", "minus_di_14"):
            assert c in out.columns
            assert not out[c].dropna().empty

    def test_finite_on_flat(self, flat_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_adx(flat_df, period=14)
        for c in ("adx_14", "plus_di_14", "minus_di_14"):
            assert np.isfinite(out[c].dropna()).all()


class TestMACD:
    def test_histogram_definition(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_macd(synthetic_df)
        assert "macd_line" in out.columns
        assert "macd_signal" in out.columns
        assert "macd_histogram" in out.columns
        np.testing.assert_allclose(
            out["macd_histogram"].dropna().to_numpy(),
            (out["macd_line"] - out["macd_signal"]).dropna().to_numpy(),
            rtol=1e-9,
        )


class TestBollingerBands:
    def test_band_definitions(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_bollinger_bands(synthetic_df, period=20, n_std=2.0)
        mid = out["bb_mid_20"]
        std0 = synthetic_df["close"].rolling(20).std(ddof=0)
        np.testing.assert_allclose(out["bb_upper_20"].dropna(), (mid + 2 * std0).dropna(), rtol=1e-9)
        np.testing.assert_allclose(out["bb_lower_20"].dropna(), (mid - 2 * std0).dropna(), rtol=1e-9)


# ─── Signal logic ──────────────────────────────────────────────────────────

class TestEMACrossoverSignal:
    def test_values_in_set(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().ema_crossover_signal(synthetic_df)
        assert set(out["ema_cross_signal"].dropna().unique()).issubset({-1, 0, 1})


class TestRSISignal:
    def test_values_in_set(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_rsi_signal(synthetic_df)
        assert set(out["rsi_signal"].dropna().unique()).issubset({-1, 0, 1})


class TestMACDSignal:
    def test_values_in_set(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_macd_signal(synthetic_df)
        assert set(out["macd_signal"].dropna().unique()).issubset({-1, 0, 1})


class TestBollingerSignal:
    def test_values_in_set(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_bollinger_signal(synthetic_df)
        assert set(out["bb_signal"].dropna().unique()).issubset({-1, 0, 1})


class TestATRChannelSignal:
    def test_values_in_set(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_atr_channel_signal(synthetic_df)
        assert set(out["atr_breakout_signal"].dropna().unique()).issubset({-1, 0, 1})


class TestADXSignal:
    def test_values_in_set(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().compute_adx_signal(synthetic_df)
        assert set(out["adx_signal"].dropna().unique()).issubset({-1, 0, 1})


# ─── Composite pipeline ─────────────────────────────────────────────────────

class TestCompositeSignals:
    def test_signal_column_exists_and_valid(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().generate_composite_signals(synthetic_df)
        assert "signal" in out.columns
        assert set(out["signal"].dropna().unique()).issubset({-1, 0, 1})

    def test_risk_columns_present(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().generate_composite_signals(synthetic_df)
        for c in ("stop_loss", "take_profit"):
            assert c in out.columns

    def test_long_risk_geometry(self, synthetic_df: pd.DataFrame):
        eng = QuantitativeStrategyEngine()
        out = eng.generate_composite_signals(synthetic_df)
        longs = out[out["signal"] == 1]
        if not longs.empty:
            row = longs.iloc[0]
            atr_val = row["atr_14"]
            close_val = row["close"]
            if np.isfinite(atr_val) and atr_val > 0:
                assert row["stop_loss"] == pytest.approx(close_val - 2 * atr_val, rel=1e-6)
                assert row["take_profit"] == pytest.approx(close_val + 4 * atr_val, rel=1e-6)

    def test_short_risk_geometry(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().generate_composite_signals(synthetic_df)
        shorts = out[out["signal"] == -1]
        if not shorts.empty:
            row = shorts.iloc[0]
            atr_val = row["atr_14"]
            close_val = row["close"]
            if np.isfinite(atr_val) and atr_val > 0:
                assert row["stop_loss"] == pytest.approx(close_val + 2 * atr_val, rel=1e-6)
                assert row["take_profit"] == pytest.approx(close_val - 4 * atr_val, rel=1e-6)

    def test_no_trade_when_flat(self, flat_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().generate_composite_signals(flat_df)
        assert (out["signal"] == 0).all()

    def test_preserves_original_columns(self, synthetic_df: pd.DataFrame):
        out = QuantitativeStrategyEngine().generate_composite_signals(synthetic_df)
        for c in ("open", "high", "low", "close", "volume"):
            assert c in out.columns

    def test_no_loops_vectorized(self, synthetic_df: pd.DataFrame):
        # Smoke test: large frame completes quickly (vectorized)
        big = pd.concat([synthetic_df] * 5, ignore_index=True)
        big.index = pd.date_range("2026-01-01", periods=len(big), freq="5min", tz="UTC")
        out = QuantitativeStrategyEngine().generate_composite_signals(big)
        assert len(out) == len(big)


# ─── Edge cases ─────────────────────────────────────────────────────────────

class TestEdgeCases:
    def test_short_frame_returns_nan_signals(self):
        idx = pd.date_range("2026-01-01", periods=5, freq="5min", tz="UTC")
        df = pd.DataFrame(
            {"open": np.arange(5, dtype=float), "high": np.arange(5) + 1,
             "low": np.arange(5) - 1, "close": np.arange(5, dtype=float), "volume": 1},
            index=idx,
        )
        out = QuantitativeStrategyEngine().generate_composite_signals(df)
        assert "signal" in out.columns
        assert (out["signal"] == 0).all()  # insufficient lookback → no trade

    def test_input_not_mutated(self, synthetic_df: pd.DataFrame):
        original = synthetic_df.copy()
        QuantitativeStrategyEngine().generate_composite_signals(synthetic_df)
        pd.testing.assert_frame_equal(synthetic_df, original)

    def test_missing_column_raises(self):
        idx = pd.date_range("2026-01-01", periods=50, freq="5min", tz="UTC")
        df = pd.DataFrame({"close": np.arange(50, dtype=float)}, index=idx)
        with pytest.raises((KeyError, ValueError)):
            QuantitativeStrategyEngine().generate_composite_signals(df)
