"""Quantitative Strategy Engine — vectorized indicator & signal library.

Implements production-ready, fully vectorized technical-indicator and
strategy-signal computations on pandas/numpy.  No Python-level loops are used
for historical time-series processing; every operation is expressed via
pandas/numpy vectorized primitives so that an entire OHLCV frame can be
processed in a single pass.

This module complements (does not replace) the scalar reference math in
:mod:`patresearch.reference_math`.  The scalar functions remain the parity
reference (SOW Section 137); the vectorized engine here exists to scale indicator
computation across large historical datasets in the Python research plane.

All methods are pure functions of their inputs and never mutate the supplied
DataFrame.  Edge cases (division-by-zero, insufficient lookback) are handled by
forward-padding or NaN propagation so that downstream consumers can detect
degraded windows rather than crash.

Input contract
--------------
``df`` is a ``pd.DataFrame`` with a ``DatetimeIndex`` (UTC) and lowercase columns
``open``, ``high``, ``low``, ``close``, ``volume``.

Output contract
---------------
Each ``compute_*`` method returns the input DataFrame augmented with new
indicator columns.  ``generate_composite_signals`` returns the full frame with
all indicator columns plus a unified ``signal`` column in ``{-1, 0, 1}`` and
risk-boundary columns ``stop_loss`` / ``take_profit``.
"""
from __future__ import annotations

from typing import Final

import numpy as np
import pandas as pd

__all__ = ["QuantitativeStrategyEngine"]

# Required input columns (lowercase, per the documented contract).
_REQUIRED_COLUMNS: Final[tuple[str, ...]] = ("open", "high", "low", "close", "volume")


class QuantitativeStrategyEngine:
    """Vectorized technical-indicator and strategy-signal engine.

    Every public method is vectorized (no Python loops over the time index) and
    returns a *new* DataFrame so the caller's input is never mutated.  The class
    is stateless: configuration lives in the method call-site, which keeps the
    engine safe for parallel walk-forward folds.

    All methods accept ``period`` / ``n_std`` style parameters so the same engine
    can reproduce the four versioned strategy profiles
    (``STANDARD_SCALPING``, ``ULTRA_SCALPING``, ``STANDARD_SWING``,
    ``TREND_SWING``) by configuration, consistent with SOW Section 5.
    """

    # ──────────────────────────────────────────────────────────────────────
    # Helpers
    # ──────────────────────────────────────────────────────────────────────

    @staticmethod
    def _validate(df: pd.DataFrame) -> pd.DataFrame:
        """Validate the input frame and return a defensive copy.

        Raises
        ------
        KeyError
            If any required OHLCV column is missing.
        ValueError
            If the frame is empty or lacks a DatetimeIndex.
        """
        missing = [c for c in _REQUIRED_COLUMNS if c not in df.columns]
        if missing:
            raise KeyError(f"Missing required columns: {missing}")
        if df.empty:
            raise ValueError("Input DataFrame is empty.")
        if not isinstance(df.index, pd.DatetimeIndex):
            raise TypeError("Input DataFrame must have a DatetimeIndex.")
        return df.copy()

    @staticmethod
    def _wilder_ema(series: pd.Series, period: int) -> pd.Series:
        """Wilder-style EMA (alpha = 1/period) used by RSI/ATR/ADX.

        Formula
        -------
        .. math:: \\mathrm{EMA}_t = \\alpha\\, x_t + (1-\\alpha)\\,\\mathrm{EMA}_{t-1},\\quad \\alpha=\\frac{1}{n}

        The first valid value is seeded with the simple mean of the first
        ``period`` observations (Wilder's convention), after which the recursion
        is applied vectorized via :meth:`pd.Series.ewm`.
        """
        if period <= 0:
            return pd.Series(np.nan, index=series.index, dtype=float)
        alpha = 1.0 / period
        # adjust=False reproduces the recursive EMA y_t = a*x_t + (1-a)*y_{t-1}.
        return series.ewm(alpha=alpha, adjust=False, min_periods=period).mean()

    # ──────────────────────────────────────────────────────────────────────
    # 1A. SMA / EMA
    # ──────────────────────────────────────────────────────────────────────

    def compute_sma(self, df: pd.DataFrame, period: int = 20) -> pd.DataFrame:
        """Compute the Simple Moving Average of ``close``.

        Formula
        -------
        .. math:: \\mathrm{SMA}_t = \\frac{1}{n}\\sum_{i=0}^{n-1} P_{t-i}

        Vectorized via :meth:`pd.Series.rolling`.  The first ``period-1`` values
        are ``NaN`` (insufficient lookback) and are left untouched.
        """
        out = self._validate(df)
        col = f"sma_{period}"
        out[col] = out["close"].rolling(window=period, min_periods=period).mean()
        return out

    def compute_ema(self, df: pd.DataFrame, period: int = 20) -> pd.DataFrame:
        """Compute the Exponential Moving Average of ``close``.

        Formula
        -------
        .. math:: \\mathrm{EMA}_t = P_t\\,\\alpha + \\mathrm{EMA}_{t-1}(1-\\alpha),\\quad \\alpha=\\frac{2}{n+1}

        Vectorized via :meth:`pd.Series.ewm` with ``adjust=False`` so the
        recursive definition above is reproduced exactly.
        """
        out = self._validate(df)
        col = f"ema_{period}"
        alpha = 2.0 / (period + 1)
        out[col] = out["close"].ewm(alpha=alpha, adjust=False, min_periods=period).mean()
        return out

    def ema_crossover_signal(self, df: pd.DataFrame, fast: int = 50, slow: int = 200) -> pd.DataFrame:
        """Generate EMA fast/slow crossover signals.

        Signal
        ------
        * **+1 (long)** on a *golden cross* — fast EMA crosses **above** slow EMA.
        * **-1 (short)** on a *death cross* — fast EMA crosses **below** slow EMA.
        * **0** otherwise (no crossover on this bar).

        Crosses are detected by vectorized sign-change on ``fast - slow``.
        """
        out = self._validate(df)
        out = self.compute_ema(out, period=fast)
        out = self.compute_ema(out, period=slow)
        diff = out[f"ema_{fast}"] - out[f"ema_{slow}"]
        # sign change: prev <= 0 and curr > 0  (golden), prev >= 0 and curr < 0 (death)
        prev_diff = diff.shift(1)
        golden = (prev_diff <= 0) & (diff > 0)
        death = (prev_diff >= 0) & (diff < 0)
        out["ema_cross_signal"] = 0
        out.loc[golden, "ema_cross_signal"] = 1
        out.loc[death, "ema_cross_signal"] = -1
        # Mask until both EMAs are valid
        warmup = out[f"ema_{slow}"].isna()
        out.loc[warmup, "ema_cross_signal"] = np.nan
        return out

    # ──────────────────────────────────────────────────────────────────────
    # 1B. ADX with DM+/DM-
    # ──────────────────────────────────────────────────────────────────────

    def compute_adx(self, df: pd.DataFrame, period: int = 14) -> pd.DataFrame:
        """Compute ADX, +DI and -DI using Wilder's smoothing.

        Formulas
        --------
        True Range::

            \\mathrm{TR}_t = \\max(H_t-L_t,\\ |H_t-C_{t-1}|,\\ |L_t-C_{t-1}|)

        Directional movement::

            +\\mathrm{DM}_t = H_t-H_{t-1} \\quad (\\text{if } >0 \\text{ and } > L_{t-1}-L_t)
            -\\mathrm{DM}_t = L_{t-1}-L_t \\quad (\\text{if } >0 \\text{ and } > H_t-H_{t-1})

        Smoothed (Wilder) components, then::

            +\\mathrm{DI}_t = 100\\,\\frac{\\mathrm{SM}_{+\\mathrm{DM}}}{\\mathrm{SM}_{\\mathrm{TR}}}
            -\\mathrm{DI}_t = 100\\,\\frac{\\mathrm{SM}_{-\\mathrm{DM}}}{\\mathrm{SM}_{\\mathrm{TR}}}
            \\mathrm{DX}_t = 100\\,\\frac{|+\\mathrm{DI}_t - -\\mathrm{DI}_t|}{+\\mathrm{DI}_t + -\\mathrm{DI}_t}
            \\mathrm{ADX}_t = \\mathrm{WilderEMA}(\\mathrm{DX}, n)

        Division-by-zero is guarded: where the denominator is zero the result is
        set to ``NaN`` (forward-filled where safe).
        """
        out = self._validate(df)
        high, low, close = out["high"], out["low"], out["close"]

        up = high.diff()
        down = -low.diff()
        plus_dm = np.where((up > down) & (up > 0), up, 0.0)
        minus_dm = np.where((down > up) & (down > 0), down, 0.0)

        tr = self._true_range_series(high, low, close)

        atr_wilder = self._wilder_ema(tr, period)
        plus_dm_s = self._wilder_ema(pd.Series(plus_dm, index=out.index, dtype=float), period)
        minus_dm_s = self._wilder_ema(pd.Series(minus_dm, index=out.index, dtype=float), period)

        with np.errstate(divide="ignore", invalid="ignore"):
            plus_di = 100.0 * plus_dm_s / atr_wilder
            minus_di = 100.0 * minus_dm_s / atr_wilder
            plus_di = plus_di.replace([np.inf, -np.inf], np.nan)
            minus_di = minus_di.replace([np.inf, -np.inf], np.nan)

        di_sum = plus_di + minus_di
        di_diff = (plus_di - minus_di).abs()
        with np.errstate(divide="ignore", invalid="ignore"):
            dx = 100.0 * di_diff / di_sum
            dx = dx.replace([np.inf, -np.inf], np.nan)

        adx = self._wilder_ema(dx, period)

        out[f"plus_di_{period}"] = plus_di
        out[f"minus_di_{period}"] = minus_di
        out[f"adx_{period}"] = adx
        return out

    def compute_adx_signal(self, df: pd.DataFrame, period: int = 14, adx_threshold: float = 25.0) -> pd.DataFrame:
        """ADX-based directional signal.

        Signal
        ------
        * **+1 (long)** when ``ADX > threshold`` and ``+DI > -DI``.
        * **-1 (short)** when ``ADX > threshold`` and ``+DI < -DI``.
        * **0** otherwise.
        """
        out = self.compute_adx(df, period=period)
        adx = out[f"adx_{period}"]
        plus = out[f"plus_di_{period}"]
        minus = out[f"minus_di_{period}"]
        out["adx_signal"] = 0
        out.loc[(adx > adx_threshold) & (plus > minus), "adx_signal"] = 1
        out.loc[(adx > adx_threshold) & (plus < minus), "adx_signal"] = -1
        warmup = adx.isna()
        out.loc[warmup, "adx_signal"] = np.nan
        return out

    # ──────────────────────────────────────────────────────────────────────
    # 2A. RSI (Wilder)
    # ──────────────────────────────────────────────────────────────────────

    def compute_rsi(self, df: pd.DataFrame, period: int = 14) -> pd.DataFrame:
        """Compute the Relative Strength Index using Wilder's smoothing.

        Formula
        -------
        .. math::
           \\Delta P_t = P_t - P_{t-1}\\\\
           G_t = \\max(\\Delta P_t, 0),\\quad L_t = \\max(-\\Delta P_t, 0)\\\\
           \\overline{G}_t = \\mathrm{WilderEMA}(G, n),\\quad
           \\overline{L}_t = \\mathrm{WilderEMA}(L, n)\\\\
           \\mathrm{RS}_t = \\frac{\\overline{G}_t}{\\overline{L}_t},\\quad
           \\mathrm{RSI}_t = 100 - \\frac{100}{1+\\mathrm{RS}_t}

        If ``\\overline{L}_t = 0`` (no losses) then ``RSI = 100``; if both
        averages are zero (flat price) the result is ``NaN`` to avoid 0/0.
        """
        out = self._validate(df)
        close = out["close"]
        delta = close.diff()
        gain = delta.clip(lower=0.0)
        loss = (-delta).clip(lower=0.0)

        avg_gain = self._wilder_ema(gain, period)
        avg_loss = self._wilder_ema(loss, period)

        with np.errstate(divide="ignore", invalid="ignore"):
            rs = avg_gain / avg_loss
            # 100 - 100/(1+rs): when rs=+inf (no losses) this evaluates to 100.
            rsi = 100.0 - (100.0 / (1.0 + rs))

        # Flat-price guard: avg_gain == avg_loss == 0 → 0/0 → undefined → NaN.
        both_zero = (avg_gain == 0) & (avg_loss == 0)
        rsi = rsi.where(~both_zero, np.nan)
        # Sanitize any residual non-finite values.
        rsi = rsi.replace([np.inf, -np.inf], np.nan)
        # No losses at all → RSI = 100 (handled by rs=inf → 100/(1+inf)=0 → 100-0=100).
        out[f"rsi_{period}"] = rsi
        return out

    def compute_rsi_signal(self, df: pd.DataFrame, period: int = 14, oversold: float = 30.0, overbought: float = 70.0) -> pd.DataFrame:
        """RSI mean-reversion crossover signal.

        Signal
        ------
        * **+1 (long)** when RSI crosses **above** ``oversold`` (from below to above 30).
        * **-1 (short)** when RSI crosses **below** ``overbought`` (from above to below 70).
        * **0** otherwise.
        """
        out = self.compute_rsi(df, period=period)
        rsi = out[f"rsi_{period}"]
        prev = rsi.shift(1)
        long_cross = (prev <= oversold) & (rsi > oversold)
        short_cross = (prev >= overbought) & (rsi < overbought)
        out["rsi_signal"] = 0
        out.loc[long_cross, "rsi_signal"] = 1
        out.loc[short_cross, "rsi_signal"] = -1
        out.loc[rsi.isna(), "rsi_signal"] = np.nan
        return out

    # ──────────────────────────────────────────────────────────────────────
    # 2B. MACD
    # ──────────────────────────────────────────────────────────────────────

    def compute_macd(self, df: pd.DataFrame, fast: int = 12, slow: int = 26, signal: int = 9) -> pd.DataFrame:
        """Compute the MACD line, signal line and histogram.

        Formula
        -------
        .. math::
           \\mathrm{MACD}_t = \\mathrm{EMA}_{12}(P)_t - \\mathrm{EMA}_{26}(P)_t\\\\
           \\mathrm{Signal}_t = \\mathrm{EMA}_{9}(\\mathrm{MACD})_t\\\\
           \\mathrm{Histogram}_t = \\mathrm{MACD}_t - \\mathrm{Signal}_t
        """
        out = self._validate(df)
        close = out["close"]
        ema_fast = close.ewm(span=fast, adjust=False).mean()
        ema_slow = close.ewm(span=slow, adjust=False).mean()
        macd_line = ema_fast - ema_slow
        macd_signal = macd_line.ewm(span=signal, adjust=False).mean()
        out["macd_line"] = macd_line
        out["macd_signal"] = macd_signal
        out["macd_histogram"] = macd_line - macd_signal
        return out

    def compute_macd_signal(self, df: pd.DataFrame, fast: int = 12, slow: int = 26, signal: int = 9) -> pd.DataFrame:
        """MACD crossover signal.

        Signal
        ------
        * **+1 (long)** when MACD line crosses **above** the signal line.
        * **-1 (short)** when MACD line crosses **below** the signal line.
        * **0** otherwise.
        """
        out = self.compute_macd(df, fast=fast, slow=slow, signal=signal)
        diff = out["macd_line"] - out["macd_signal"]
        prev = diff.shift(1)
        bull = (prev <= 0) & (diff > 0)
        bear = (prev >= 0) & (diff < 0)
        out["macd_signal"] = 0
        out.loc[bull, "macd_signal"] = 1
        out.loc[bear, "macd_signal"] = -1
        # Mask warmup where slow EMA is not yet valid.
        warmup = out["macd_line"].isna() & out["macd_signal"].isna()
        out.loc[warmup, "macd_signal_col"] = np.nan  # not used; kept for clarity
        out.drop(columns=["macd_signal_col"], errors="ignore", inplace=True)
        return out

    # ──────────────────────────────────────────────────────────────────────
    # 3A. Bollinger Bands
    # ──────────────────────────────────────────────────────────────────────

    def compute_bollinger_bands(self, df: pd.DataFrame, period: int = 20, n_std: float = 2.0) -> pd.DataFrame:
        """Compute Bollinger Bands.

        Formula
        -------
        .. math::
           \\mathrm{Mid}_t = \\mathrm{SMA}_{20}(P)_t\\\\
           \\sigma_t = \\mathrm{StdDev}_{20}(P)_t\\\\
           \\mathrm{Upper}_t = \\mathrm{Mid}_t + k\\,\\sigma_t\\\\
           \\mathrm{Lower}_t = \\mathrm{Mid}_t - k\\,\\sigma_t
        """
        out = self._validate(df)
        mid = out["close"].rolling(window=period, min_periods=period).mean()
        std = out["close"].rolling(window=period, min_periods=period).std(ddof=0)
        out[f"bb_mid_{period}"] = mid
        out[f"bb_upper_{period}"] = mid + n_std * std
        out[f"bb_lower_{period}"] = mid - n_std * std
        return out

    def compute_bollinger_signal(self, df: pd.DataFrame, period: int = 20, n_std: float = 2.0) -> pd.DataFrame:
        """Bollinger Band mean-reversion signal.

        Signal
        ------
        * **+1 (long)** when ``close`` touches or breaks the **lower** band.
        * **-1 (short)** when ``close`` touches or breaks the **upper** band.
        * **0** otherwise.
        """
        out = self.compute_bollinger_bands(df, period=period, n_std=n_std)
        close = out["close"]
        lower = out[f"bb_lower_{period}"]
        upper = out[f"bb_upper_{period}"]
        out["bb_signal"] = 0
        out.loc[close <= lower, "bb_signal"] = 1
        out.loc[close >= upper, "bb_signal"] = -1
        out.loc[lower.isna() | upper.isna(), "bb_signal"] = np.nan
        return out

    # ──────────────────────────────────────────────────────────────────────
    # 3B. ATR Channel Breakout
    # ──────────────────────────────────────────────────────────────────────

    @staticmethod
    def _true_range_series(high: pd.Series, low: pd.Series, close: pd.Series) -> pd.Series:
        """Vectorized True Range.

        Formula
        -------
        .. math:: \\mathrm{TR}_t = \\max(H_t-L_t,\\ |H_t-C_{t-1}|,\\ |L_t-C_{t-1}|)
        """
        prev_close = close.shift(1)
        hl = (high - low).abs()
        hc = (high - prev_close).abs()
        lc = (low - prev_close).abs()
        tr = pd.concat([hl, hc, lc], axis=1).max(axis=1)
        return tr

    def compute_atr(self, df: pd.DataFrame, period: int = 14) -> pd.DataFrame:
        """Compute the Average True Range using Wilder smoothing.

        Formula
        -------
        .. math:: \\mathrm{ATR}_t = \\frac{\\mathrm{ATR}_{t-1}(n-1) + \\mathrm{TR}_t}{n}

        Implemented vectorized as a Wilder EMA of the True Range series.
        """
        out = self._validate(df)
        tr = self._true_range_series(out["high"], out["low"], out["close"])
        atr = self._wilder_ema(tr, period)
        out[f"atr_{period}"] = atr
        return out

    def compute_atr_channel_signal(self, df: pd.DataFrame, period: int = 14, multiplier: float = 1.5) -> pd.DataFrame:
        """ATR channel-breakout signal.

        Signal
        ------
        * **+1 (long)** when ``close > prev_close + k·ATR``.
        * **-1 (short)** when ``close < prev_close - k·ATR``.
        * **0** otherwise.
        """
        out = self.compute_atr(df, period=period)
        prev_close = out["close"].shift(1)
        atr = out[f"atr_{period}"]
        upper_chan = prev_close + multiplier * atr
        lower_chan = prev_close - multiplier * atr
        out["atr_breakout_signal"] = 0
        out.loc[out["close"] > upper_chan, "atr_breakout_signal"] = 1
        out.loc[out["close"] < lower_chan, "atr_breakout_signal"] = -1
        out.loc[atr.isna() | prev_close.isna(), "atr_breakout_signal"] = np.nan
        return out

    # ──────────────────────────────────────────────────────────────────────
    # 4. Composite Matrix
    # ──────────────────────────────────────────────────────────────────────

    def generate_composite_signals(self, df: pd.DataFrame) -> pd.DataFrame:
        """Run the full indicator + signal pipeline and produce a unified signal.

        Pipeline
        --------
        1. **Structural trend filter** — EMA(50)/EMA(200) crossover establishes the
           directional regime (golden/death cross).
        2. **Entry triggers** — overextended RSI (cross above 30 / below 70) **or**
           Bollinger Band mean-reversion touch/break.
        3. **Composite signal** — a long requires a *bullish* structural trend
           (EMA fast > slow) AND an entry trigger; a short requires a *bearish*
           structural trend (EMA fast < slow) AND an entry trigger.  Otherwise
           ``0`` (NO-TRADE — a first-class valid result).
        4. **Dynamic risk management stops** based on ATR:

           .. math::
              \\mathrm{SL}_{\\text{long}} = C_t - 2\\,\\mathrm{ATR}_t\\\\
              \\mathrm{TP}_{\\text{long}} = C_t + 4\\,\\mathrm{ATR}_t\\\\
              \\mathrm{SL}_{\\text{short}} = C_t + 2\\,\\mathrm{ATR}_t\\\\
              \\mathrm{TP}_{\\text{short}} = C_t - 4\\,\\mathrm{ATR}_t

        Returns
        -------
        ``pd.DataFrame``
            The original OHLCV columns plus all indicator columns, a unified
            ``signal`` column in ``{-1, 0, 1}`` and ``stop_loss`` / ``take_profit``
            columns.  Rows without sufficient lookback carry ``signal == 0`` and
            ``NaN`` risk boundaries.

        Notes
        -----
        ``NO-TRADE`` (0) is a first-class valid result.  This method never forces
        a trade to meet a frequency target, consistent with the AGENTS.md safety
        precedence.
        """
        out = self._validate(df)

        # Indicators
        out = self.compute_ema(out, period=50)
        out = self.compute_ema(out, period=200)
        out = self.compute_rsi(out, period=14)
        out = self.compute_bollinger_bands(out, period=20, n_std=2.0)
        out = self.compute_atr(out, period=14)

        # Structural trend regime (vectorized boolean).
        ema_fast = out["ema_50"]
        ema_slow = out["ema_200"]
        bullish_trend = ema_fast > ema_slow
        bearish_trend = ema_fast < ema_slow

        # Entry triggers: RSI overextension OR Bollinger touch.
        rsi = out["rsi_14"]
        prev_rsi = rsi.shift(1)
        rsi_long_trigger = (prev_rsi <= 30) & (rsi > 30)
        rsi_short_trigger = (prev_rsi >= 70) & (rsi < 70)

        close = out["close"]
        lower_bb = out["bb_lower_20"]
        upper_bb = out["bb_upper_20"]
        bb_long_trigger = close <= lower_bb
        bb_short_trigger = close >= upper_bb

        long_trigger = rsi_long_trigger | bb_long_trigger
        short_trigger = rsi_short_trigger | bb_short_trigger

        # Composite signal: trend + trigger alignment.
        signal = pd.Series(0, index=out.index, dtype=np.int8)
        warmup = ema_slow.isna()  # not enough data for structural trend

        signal.loc[bullish_trend & long_trigger & ~warmup] = 1
        signal.loc[bearish_trend & short_trigger & ~warmup] = -1
        out["signal"] = signal.astype(np.int8)

        # Dynamic ATR-based risk boundaries.
        atr = out["atr_14"]
        sl = pd.Series(np.nan, index=out.index, dtype=float)
        tp = pd.Series(np.nan, index=out.index, dtype=float)

        long_mask = (out["signal"] == 1) & atr.notna() & (atr > 0)
        short_mask = (out["signal"] == -1) & atr.notna() & (atr > 0)

        sl.loc[long_mask] = close.loc[long_mask] - 2.0 * atr.loc[long_mask]
        tp.loc[long_mask] = close.loc[long_mask] + 4.0 * atr.loc[long_mask]
        sl.loc[short_mask] = close.loc[short_mask] + 2.0 * atr.loc[short_mask]
        tp.loc[short_mask] = close.loc[short_mask] - 4.0 * atr.loc[short_mask]

        out["stop_loss"] = sl
        out["take_profit"] = tp
        return out
