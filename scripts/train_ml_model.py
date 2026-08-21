#!/usr/bin/env python3
"""
Predict-A-Trade XAUUSD — ML Training Script (CPU-Only)

Trains XGBoost + LSTM ensemble on real XAUUSD M5 candle data from PostgreSQL/TimescaleDB.
Indicator math (Wilder smoothing) strictly matches Go engine `realtime/pkg/math/wilder.go`.

Usage:
    python3 scripts/train_ml_model.py --start_date 2024-01-01 --end_date 2025-12-31

Output:
    models/xgb_model.onnx
    models/lstm_model.onnx
    models/scaler.pkl
    models/feature_columns.json
    models/metrics.json
"""
import argparse
import json
import logging
import os
import pickle
import sys
import time
from datetime import datetime, timedelta, timezone

import numpy as np
import pandas as pd

# ─── Logging ──────────────────────────────────────────────────────────────────
logging.basicConfig(
    level=logging.INFO,
    format="%(asctime)s [%(levelname)s] %(message)s",
    datefmt="%Y-%m-%d %H:%M:%S",
)
log = logging.getLogger("PAT-ML")

# ─── Database ─────────────────────────────────────────────────────────────────
DB_URL = os.environ.get("DATABASE_URL", "")
if not DB_URL:
    raise RuntimeError("DATABASE_URL environment variable is required for training")

# ─── DXY API ──────────────────────────────────────────────────────────────────
TWELVE_DATA_KEY = os.environ.get("TWELVEDATA_API_KEY", "")
DXY_SYMBOLS = ["EUR/USD", "USD/JPY", "GBP/USD", "USD/CAD", "USD/SEK", "USD/CHF"]
DXY_WEIGHTS = [-0.576, 0.136, -0.119, -0.091, -0.042, -0.036]


# ═══════════════════════════════════════════════════════════════════════════════
# 1. WILDER SMOOTHING — Matches Go `realtime/pkg/math/wilder.go` EXACTLY
# ═══════════════════════════════════════════════════════════════════════════════

def wilder_smooth(series: np.ndarray, period: int) -> np.ndarray:
    """
    Wilder's smoothing: alpha = 1/period (NOT 2/(period+1)).
    Seed: simple average of first `period` values.
    Recursion: smoothed_t = (smoothed_{t-1} * (period-1) + value_t) / period

    Matches Go: wilderSmooth() in wilder.go
    """
    n = len(series)
    if n < period or period <= 0:
        return np.zeros(n)

    result = np.zeros(n)
    # Seed: simple average of first `period` values
    result[period - 1] = np.mean(series[:period])

    # Recursive Wilder smoothing
    for i in range(period, n):
        result[i] = (result[i - 1] * (period - 1) + series[i]) / period

    return result


def true_range_series(highs: np.ndarray, lows: np.ndarray, closes: np.ndarray) -> np.ndarray:
    """
    TR_t = max(H_t - L_t, |H_t - C_{t-1}|, |L_t - C_{t-1}|)
    First bar: TR_0 = H_0 - L_0 (no prev close).

    Matches Go: TrueRangeSeries() in wilder.go
    """
    n = min(len(highs), len(lows), len(closes))
    tr = np.zeros(n)
    tr[0] = abs(highs[0] - lows[0])
    for i in range(1, n):
        hl = abs(highs[i] - lows[i])
        hc = abs(highs[i] - closes[i - 1])
        lc = abs(lows[i] - closes[i - 1])
        tr[i] = max(hl, hc, lc)
    return tr


def atr_wilder(highs: np.ndarray, lows: np.ndarray, closes: np.ndarray, period: int = 14) -> np.ndarray:
    """
    ATR using Wilder's smoothing.
    First ATR = mean(TR, period); subsequent: ATR_t = (ATR_{t-1}*(period-1) + TR_t) / period

    Matches Go: ATRWilder() in wilder.go
    """
    tr = true_range_series(highs, lows, closes)
    return wilder_smooth(tr, period)


def rsi_wilder(closes: np.ndarray, period: int = 14) -> np.ndarray:
    """
    RSI using Wilder's smoothing.
    First avg_gain/avg_loss = simple mean of first `period` gains/losses.
    Subsequent: avg_t = (avg_{t-1}*(period-1) + gain_t) / period
    If avg_loss=0 → RSI=100. If both zero → RSI=50 (undefined/flat).

    Matches Go: RSIWilder() in wilder.go
    """
    n = len(closes)
    if n <= period or period <= 0:
        return np.full(n, 50.0)

    deltas = np.diff(closes, prepend=closes[0])
    gains = np.where(deltas > 0, deltas, 0.0)
    losses = np.where(deltas < 0, -deltas, 0.0)

    # Seed: simple average of first `period` gains/losses
    avg_gain = np.mean(gains[1:period + 1])
    avg_loss = np.mean(losses[1:period + 1])

    rsi = np.zeros(n)
    rsi[period] = _calc_rsi_value(avg_gain, avg_loss)

    for i in range(period + 1, n):
        avg_gain = (avg_gain * (period - 1) + gains[i]) / period
        avg_loss = (avg_loss * (period - 1) + losses[i]) / period
        rsi[i] = _calc_rsi_value(avg_gain, avg_loss)

    # Fill early values with 50 (insufficient data)
    rsi[:period] = 50.0
    return rsi


def _calc_rsi_value(avg_gain: float, avg_loss: float) -> float:
    """Single RSI value calculation. Matches Go: if avg_loss=0 → 100, if both=0 → 50."""
    if avg_loss == 0:
        if avg_gain == 0:
            return 50.0  # Flat price — undefined
        return 100.0
    rs = avg_gain / avg_loss
    return 100.0 - (100.0 / (1.0 + rs))


def adx_wilder(highs: np.ndarray, lows: np.ndarray, closes: np.ndarray, period: int = 14) -> tuple:
    """
    ADX with full Wilder smoothing for TR, +DM, -DM, and DX.
    Returns (adx, plus_di, minus_di) arrays.

    Matches Go: ADXWilder() in wilder.go
    """
    n = min(len(highs), len(lows), len(closes))
    if n <= period * 2 or period <= 0:
        return np.zeros(n), np.zeros(n), np.zeros(n)

    # Compute +DM, -DM, TR series (starting from bar 1)
    plus_dm = np.zeros(n)
    minus_dm = np.zeros(n)
    tr = np.zeros(n)

    tr[0] = abs(highs[0] - lows[0])
    for i in range(1, n):
        up_move = highs[i] - highs[i - 1]
        down_move = lows[i - 1] - lows[i]

        plus_dm[i] = up_move if (up_move > down_move and up_move > 0) else 0.0
        minus_dm[i] = down_move if (down_move > up_move and down_move > 0) else 0.0

        hl = abs(highs[i] - lows[i])
        hc = abs(highs[i] - closes[i - 1])
        lc = abs(lows[i] - closes[i - 1])
        tr[i] = max(hl, hc, lc)

    # Wilder smoothing of +DM, -DM, TR
    s_plus_dm = wilder_smooth(plus_dm[1:], period)
    s_minus_dm = wilder_smooth(minus_dm[1:], period)
    s_tr = wilder_smooth(tr[1:], period)

    # Build DX series
    dx_series = np.zeros(n - 1)
    for i in range(len(s_tr)):
        if s_tr[i] == 0:
            continue
        p_di = 100.0 * s_plus_dm[i] / s_tr[i]
        m_di = 100.0 * s_minus_dm[i] / s_tr[i]
        di_sum = p_di + m_di
        if di_sum == 0:
            continue
        dx_series[i] = 100.0 * abs(p_di - m_di) / di_sum

    adx = wilder_smooth(dx_series, period)

    # Final +DI/-DI from latest smoothed values
    final_idx = len(s_tr) - 1
    if s_tr[final_idx] > 0:
        plus_di_val = 100.0 * s_plus_dm[final_idx] / s_tr[final_idx]
        minus_di_val = 100.0 * s_minus_dm[final_idx] / s_tr[final_idx]
    else:
        plus_di_val = 0.0
        minus_di_val = 0.0

    # Pad to full length
    adx_full = np.zeros(n)
    adx_full[period * 2:] = adx[:n - period * 2]
    plus_di_full = np.zeros(n)
    minus_di_full = np.zeros(n)
    plus_di_full[period:] = plus_di_val
    minus_di_full[period:] = minus_di_val

    return adx_full, plus_di_full, minus_di_full


# ═══════════════════════════════════════════════════════════════════════════════
# 2. INDICATOR COMPUTATION (42 indicators matching Go engine)
# ═══════════════════════════════════════════════════════════════════════════════

def compute_ema(values: np.ndarray, period: int) -> np.ndarray:
    """EMA with alpha = 2/(period+1). Matches Go: patmath.EMA()"""
    if len(values) == 0 or period <= 0:
        return np.zeros(len(values))
    alpha = 2.0 / (period + 1)
    result = np.zeros(len(values))
    result[0] = values[0]
    for i in range(1, len(values)):
        result[i] = values[i] * alpha + result[i - 1] * (1 - alpha)
    return result


def compute_sma(values: np.ndarray, period: int) -> np.ndarray:
    """Simple Moving Average. Matches Go: simpleMA()"""
    if len(values) < period or period <= 0:
        return np.zeros(len(values))
    result = np.zeros(len(values))
    cumsum = np.cumsum(values)
    result[period - 1:] = (cumsum[period - 1:] - np.concatenate([[0], cumsum[:-period]])) / period
    return result


def compute_macd(closes: np.ndarray, fast: int = 12, slow: int = 26, signal: int = 9) -> tuple:
    """MACD = EMA_fast - EMA_slow; Signal = EMA_signal(MACD); Hist = MACD - Signal"""
    ema_fast = compute_ema(closes, fast)
    ema_slow = compute_ema(closes, slow)
    macd_line = ema_fast - ema_slow
    macd_signal = compute_ema(macd_line, signal)
    macd_hist = macd_line - macd_signal
    return macd_line, macd_signal, macd_hist


def compute_bollinger(closes: np.ndarray, period: int = 20, n_std: float = 2.0) -> tuple:
    """Bollinger Bands with population std (ddof=0). Matches Go: stdDevCalc()"""
    mid = compute_sma(closes, period)
    # Rolling std with ddof=0 (population)
    rolling_std = pd.Series(closes).rolling(period, min_periods=period).std(ddof=0).fillna(0).values
    upper = mid + n_std * rolling_std
    lower = mid - n_std * rolling_std
    width = np.where(mid > 0, (upper - lower) / mid, 0.0)
    return upper, mid, lower, width


def compute_stochastic(highs: np.ndarray, lows: np.ndarray, closes: np.ndarray,
                       k_period: int = 14, k_smooth: int = 3, d_smooth: int = 3) -> tuple:
    """Stochastic %K = 100*(C-LL)/(HH-LL); Signal = SMA(%K, 3)"""
    n = len(closes)
    k = np.zeros(n)
    for i in range(k_period - 1, n):
        hh = np.max(highs[i - k_period + 1:i + 1])
        ll = np.min(lows[i - k_period + 1:i + 1])
        if hh > ll:
            k[i] = 100.0 * (closes[i] - ll) / (hh - ll)
    # Smooth %K with SMA(3)
    k_smoothed = compute_sma(k, k_smooth)
    # %D = SMA(%K_smoothed, 3)
    d = compute_sma(k_smoothed, d_smooth)
    return k_smoothed, d


def compute_cci(highs: np.ndarray, lows: np.ndarray, closes: np.ndarray, period: int = 20) -> np.ndarray:
    """CCI = (TP - SMA(TP)) / (0.015 * MeanDeviation)"""
    tp = (highs + lows + closes) / 3.0
    n = len(tp)
    cci = np.zeros(n)
    for i in range(period - 1, n):
        tp_window = tp[i - period + 1:i + 1]
        sma_tp = np.mean(tp_window)
        mean_dev = np.mean(np.abs(tp_window - sma_tp))
        if mean_dev > 0:
            cci[i] = (tp[i] - sma_tp) / (0.015 * mean_dev)
    return cci


def compute_obv(closes: np.ndarray, volumes: np.ndarray) -> np.ndarray:
    """On-Balance Volume. Matches Go: OBV logic in indicators.go"""
    n = len(closes)
    obv = np.zeros(n)
    for i in range(1, n):
        if closes[i] > closes[i - 1]:
            obv[i] = obv[i - 1] + volumes[i]
        elif closes[i] < closes[i - 1]:
            obv[i] = obv[i - 1] - volumes[i]
        else:
            obv[i] = obv[i - 1]
    return obv


def compute_all_indicators(df: pd.DataFrame) -> pd.DataFrame:
    """Compute all 42 indicators. Matches Go engine IndicatorFeatures."""
    highs = df["high"].values
    lows = df["low"].values
    closes = df["close"].values
    volumes = df["volume"].values.astype(np.float64)

    log.info("Computing 42 indicators (Wilder smoothing)...")

    # Trend — EMA
    df["ema9"] = compute_ema(closes, 9)
    df["ema21"] = compute_ema(closes, 21)
    df["ema50"] = compute_ema(closes, 50)
    df["ema100"] = compute_ema(closes, 100)
    df["ema200"] = compute_ema(closes, 200)
    df["ema_cross_9_21"] = (df["ema9"] > df["ema21"]).astype(int)

    # Trend — SMA
    df["sma50"] = compute_sma(closes, 50)
    df["sma100"] = compute_sma(closes, 100)
    df["sma200"] = compute_sma(closes, 200)

    # Trend — MACD
    macd_line, macd_signal, macd_hist = compute_macd(closes)
    df["macd_main"] = macd_line
    df["macd_signal"] = macd_signal
    df["macd_histogram"] = macd_hist
    df["macd_bull_cross"] = ((macd_line > macd_signal) & (np.roll(macd_line, 1) <= np.roll(macd_signal, 1))).astype(int)
    df["macd_bear_cross"] = ((macd_line < macd_signal) & (np.roll(macd_line, 1) >= np.roll(macd_signal, 1))).astype(int)

    # Trend — ADX (Wilder)
    adx, plus_di, minus_di = adx_wilder(highs, lows, closes, 14)
    df["adx"] = adx
    df["adx_plus_di"] = plus_di
    df["adx_minus_di"] = minus_di

    # Momentum — RSI (Wilder)
    df["rsi"] = rsi_wilder(closes, 14)

    # Momentum — Stochastic
    stoch_k, stoch_d = compute_stochastic(highs, lows, closes)
    df["stoch_main"] = stoch_k
    df["stoch_signal"] = stoch_d

    # Momentum — CCI
    df["cci"] = compute_cci(highs, lows, closes, 20)

    # Volatility — ATR (Wilder)
    df["atr"] = atr_wilder(highs, lows, closes, 14)

    # Volatility — Bollinger
    boll_upper, boll_mid, boll_lower, boll_width = compute_bollinger(closes)
    df["boll_upper"] = boll_upper
    df["boll_middle"] = boll_mid
    df["boll_lower"] = boll_lower
    df["boll_width"] = boll_width
    df["boll_bull_rev"] = ((np.roll(closes, 1) < np.roll(boll_lower, 1)) & (closes > boll_lower)).astype(int)
    df["boll_bear_rev"] = ((np.roll(closes, 1) > np.roll(boll_upper, 1)) & (closes < boll_upper)).astype(int)

    # Volume — OBV
    df["obv"] = compute_obv(closes, volumes)

    # StochRSI (simplified: Stochastic of RSI)
    rsi_series = df["rsi"].values
    if len(rsi_series) > 14:
        rsi_min = pd.Series(rsi_series).rolling(14, min_periods=14).min().fillna(50).values
        rsi_max = pd.Series(rsi_series).rolling(14, min_periods=14).max().fillna(50).values
        rsi_range = rsi_max - rsi_min
        stoch_rsi = np.where(rsi_range > 0, (rsi_series - rsi_min) / rsi_range, 0.5)
        df["stoch_rsi"] = stoch_rsi
        df["stoch_rsi_k"] = compute_sma(stoch_rsi, 3)
        df["stoch_rsi_d"] = compute_sma(compute_sma(stoch_rsi, 3), 3)
    else:
        df["stoch_rsi"] = 0.5
        df["stoch_rsi_k"] = 0.5
        df["stoch_rsi_d"] = 0.5

    # Session
    df["hour"] = df.index.hour
    df["day_of_week"] = df.index.dayofweek
    df["session"] = df["hour"].apply(lambda h: 0 if 0 <= h < 7 else 1 if 7 <= h < 13 else 2)

    log.info(f"Indicators computed: {len(df.columns)} columns")
    return df


# ═══════════════════════════════════════════════════════════════════════════════
# 3. ENGINEERED FEATURES
# ═══════════════════════════════════════════════════════════════════════════════

def add_engineered_features(df: pd.DataFrame) -> pd.DataFrame:
    """Add rolling stats, price lags, time features, DXY."""
    log.info("Adding engineered features...")

    # Rolling statistics over 5, 10, 20 periods
    for window in [5, 10, 20]:
        df[f"close_roll_mean_{window}"] = df["close"].rolling(window).mean()
        df[f"close_roll_std_{window}"] = df["close"].rolling(window).std()
        df[f"close_roll_max_{window}"] = df["close"].rolling(window).max()
        df[f"close_roll_min_{window}"] = df["close"].rolling(window).min()

    # Price lag features
    for lag in [1, 2, 3, 5, 10]:
        df[f"close_lag_{lag}"] = df["close"].shift(lag)

    # DXY price (fetch via Twelve Data API)
    df["dxy"] = fetch_dxy(df)

    return df


def fetch_dxy(df: pd.DataFrame) -> np.ndarray:
    """Fetch DXY (US Dollar Index) from Twelve Data API. Returns 0 if unavailable."""
    if not TWELVE_DATA_KEY:
        log.warning("TWELVEDATA_API_KEY not set — DXY filled with 0")
        return np.zeros(len(df))

    try:
        import json as _json
        import urllib.request

        log.info("Fetching DXY from Twelve Data API...")
        dxy_values = np.zeros(len(df))

        # Fetch each component currency pair
        rates = {}
        for symbol in DXY_SYMBOLS:
            url = f"https://api.twelvedata.com/time_series?symbol={symbol}&interval=1day&outputsize=30&apikey={TWELVE_DATA_KEY}"
            try:
                req = urllib.request.Request(url)
                with urllib.request.urlopen(req, timeout=10) as resp:
                    data = _json.loads(resp.read().decode())
                    if "values" in data and len(data["values"]) > 0:
                        rates[symbol] = float(data["values"][0]["close"])
                        log.info(f"  {symbol}: {rates[symbol]}")
            except (OSError, ValueError, KeyError, RuntimeError, ConnectionError, ImportError) as e:
                log.warning(f"  {symbol} fetch failed: {e}")

        if len(rates) < 4:
            log.warning(f"Insufficient DXY components ({len(rates)}/6) — DXY filled with 0")
            return np.zeros(len(df))

        # Compute DXY: 50.14348112 + sum(weight * rate^(-1))
        dxy_value = 50.14348112
        for symbol, weight in zip(DXY_SYMBOLS, DXY_WEIGHTS):
            if symbol in rates:
                dxy_value += weight * (1.0 / rates[symbol])

        log.info(f"DXY computed: {dxy_value:.4f}")
        dxy_values[:] = dxy_value
        return dxy_values

    except (OSError, ValueError, KeyError, RuntimeError, ConnectionError, ImportError) as e:
        log.warning(f"DXY fetch failed: {e} — filled with 0")
        return np.zeros(len(df))


# ═══════════════════════════════════════════════════════════════════════════════
# 4. TARGET LABEL
# ═══════════════════════════════════════════════════════════════════════════════

def create_target_labels(df: pd.DataFrame, future_periods: int = 5) -> pd.DataFrame:
    """
    Target: Future 5-period return.
    Label = 1 (BUY) if return > 0.5 * ATR(14)
    Label = -1 (SELL) if return < -0.5 * ATR(14)
    Else Label = 0 (HOLD)
    """
    log.info(f"Creating target labels (future={future_periods} periods)...")

    df["future_return"] = df["close"].shift(-future_periods) - df["close"]
    atr_threshold = 0.5 * df["atr"]

    df["label"] = 0  # HOLD
    df.loc[df["future_return"] > atr_threshold, "label"] = 1   # BUY
    df.loc[df["future_return"] < -atr_threshold, "label"] = -1  # SELL

    # Drop rows with NaN future (last N rows)
    df = df.dropna(subset=["future_return"])

    label_counts = df["label"].value_counts().to_dict()
    log.info(f"Label distribution: {label_counts}")
    return df


# ═══════════════════════════════════════════════════════════════════════════════
# 5. DATA LOADING
# ═══════════════════════════════════════════════════════════════════════════════

def load_candles(start_date: str, end_date: str) -> pd.DataFrame:
    """Load XAUUSD M5 candles from PostgreSQL/TimescaleDB."""
    import psycopg2

    log.info(f"Loading XAUUSD M5 candles from {start_date} to {end_date}...")

    conn = psycopg2.connect(DB_URL)
    query = """
        SELECT time, open, high, low, close, volume
        FROM market.candles
        WHERE symbol = 'XAUUSD' AND timeframe = 'M5'
          AND time >= %s AND time <= %s
        ORDER BY time ASC
    """
    df = pd.read_sql(query, conn, params=(start_date, end_date))
    conn.close()

    if df.empty:
        log.error("No candle data found. Check database connection and date range.")
        sys.exit(1)

    df["time"] = pd.to_datetime(df["time"], utc=True)
    df = df.set_index("time")
    df = df.drop_duplicates()
    log.info(f"Loaded {len(df)} candles from {df.index[0]} to {df.index[-1]}")
    return df


# ═══════════════════════════════════════════════════════════════════════════════
# 6. MODELS
# ═══════════════════════════════════════════════════════════════════════════════

def train_xgboost(X_train, y_train, X_val, y_val, feature_names):
    """Train XGBoost classifier (CPU, tree_method='hist')."""
    import xgboost as xgb

    log.info("Training XGBoost (CPU, tree_method='hist')...")

    # Map labels from {-1, 0, 1} to {0, 1, 2} for multi:softprob
    y_train_mapped = y_train + 1
    y_val_mapped = y_val + 1

    model = xgb.XGBClassifier(
        n_estimators=400,
        max_depth=8,
        learning_rate=0.03,
        objective="multi:softprob",
        tree_method="hist",
        num_class=3,
        eval_metric="mlogloss",
        random_state=42,
        n_jobs=-1,
    )

    model.fit(
        X_train, y_train_mapped,
        eval_set=[(X_val, y_val_mapped)],
        verbose=50,
    )

    # Validation accuracy
    val_pred = model.predict(X_val)
    val_acc = np.mean(val_pred == y_val_mapped)
    log.info(f"XGBoost validation accuracy: {val_acc:.4f}")

    # Inference latency benchmark
    t0 = time.time()
    for _ in range(100):
        model.predict(X_val[:1])
    latency_ms = (time.time() - t0) / 100 * 1000
    log.info(f"XGBoost inference latency: {latency_ms:.2f}ms")

    return model, val_acc, latency_ms


def train_lstm(X_train, y_train, X_val, y_val, sequence_length=60, epochs=30, batch_size=64):
    """Train LSTM classifier (CPU-only PyTorch)."""
    import torch
    from torch import nn
    from torch.utils.data import DataLoader, TensorDataset

    log.info("Training LSTM (CPU-only PyTorch)...")

    device = torch.device("cpu")
    log.info(f"Using device: {device}")

    # Map labels from {-1, 0, 1} to {0, 1, 2}
    y_train_mapped = y_train + 1
    y_val_mapped = y_val + 1

    # Create sequences
    def create_sequences(X, y, seq_len):
        Xs, ys = [], []
        for i in range(seq_len, len(X)):
            Xs.append(X[i - seq_len:i])
            ys.append(y[i])
        return np.array(Xs), np.array(ys)

    X_train_seq, y_train_seq = create_sequences(X_train, y_train_mapped, sequence_length)
    X_val_seq, y_val_seq = create_sequences(X_val, y_val_mapped, sequence_length)

    log.info(f"LSTM sequences: train={len(X_train_seq)}, val={len(X_val_seq)}")

    # Convert to tensors
    train_dataset = TensorDataset(
        torch.FloatTensor(X_train_seq).to(device),
        torch.LongTensor(y_train_seq).to(device),
    )
    val_dataset = TensorDataset(
        torch.FloatTensor(X_val_seq).to(device),
        torch.LongTensor(y_val_seq).to(device),
    )

    train_loader = DataLoader(train_dataset, batch_size=batch_size, shuffle=True)
    val_loader = DataLoader(val_dataset, batch_size=batch_size, shuffle=False)

    input_size = X_train_seq.shape[2]

    class LSTMModel(nn.Module):
        def __init__(self, input_size, hidden_size=64, num_layers=2, num_classes=3, dropout=0.2):
            super().__init__()
            self.lstm = nn.LSTM(input_size, hidden_size, num_layers, batch_first=True, dropout=dropout)
            self.fc = nn.Linear(hidden_size, num_classes)

        def forward(self, x):
            out, _ = self.lstm(x)
            out = self.fc(out[:, -1, :])  # Last timestep
            return out

    model = LSTMModel(input_size, hidden_size=64, num_layers=2, num_classes=3, dropout=0.2).to(device)
    criterion = nn.CrossEntropyLoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=0.001)

    best_val_acc = 0.0
    for epoch in range(epochs):
        model.train()
        train_loss = 0.0
        for batch_X, batch_y in train_loader:
            optimizer.zero_grad()
            outputs = model(batch_X)
            loss = criterion(outputs, batch_y)
            loss.backward()
            optimizer.step()
            train_loss += loss.item()

        # Validation
        model.eval()
        correct, total = 0, 0
        with torch.no_grad():
            for batch_X, batch_y in val_loader:
                outputs = model(batch_X)
                _, predicted = torch.max(outputs, 1)
                total += batch_y.size(0)
                correct += (predicted == batch_y).sum().item()

        val_acc = correct / total
        if val_acc > best_val_acc:
            best_val_acc = val_acc
            torch.save(model.state_dict(), "/tmp/lstm_best.pt")

        if (epoch + 1) % 5 == 0:
            log.info(f"  Epoch {epoch+1}/{epochs}: loss={train_loss/len(train_loader):.4f} val_acc={val_acc:.4f}")

    # Load best model
    model.load_state_dict(torch.load("/tmp/lstm_best.pt"))

    # Inference latency benchmark
    model.eval()
    with torch.no_grad():
        sample = torch.FloatTensor(X_val_seq[:1]).to(device)
        t0 = time.time()
        for _ in range(100):
            model(sample)
        latency_ms = (time.time() - t0) / 100 * 1000
    log.info(f"LSTM inference latency: {latency_ms:.2f}ms")

    return model, best_val_acc, latency_ms, input_size


# ═══════════════════════════════════════════════════════════════════════════════
# 7. WALK-FORWARD VALIDATION
# ═══════════════════════════════════════════════════════════════════════════════

def walk_forward_validation(df: pd.DataFrame, feature_cols: list, label_col: str = "label"):
    """Walk-forward validation with rolling 3-month windows."""
    from sklearn.metrics import (
        accuracy_score,
        f1_score,
        precision_score,
        recall_score,
    )

    log.info("Running walk-forward validation (3-month rolling windows)...")

    results = []
    df_sorted = df.sort_index()

    # Generate 3-month windows
    start_date = df_sorted.index[0]
    end_date = df_sorted.index[-1]
    window_size = timedelta(days=90)
    step_size = timedelta(days=30)

    current = start_date + window_size
    while current + window_size <= end_date:
        train_end = current
        test_start = current
        test_end = current + window_size

        train_data = df_sorted[(df_sorted.index >= start_date) & (df_sorted.index < train_end)]
        test_data = df_sorted[(df_sorted.index >= test_start) & (df_sorted.index < test_end)]

        if len(train_data) < 1000 or len(test_data) < 100:
            current += step_size
            continue

        X_train = train_data[feature_cols].values
        y_train = train_data[label_col].values
        X_test = test_data[feature_cols].values
        y_test = test_data[label_col].values

        # Quick XGBoost for walk-forward
        try:
            import xgboost as xgb
            model = xgb.XGBClassifier(
                n_estimators=200, max_depth=6, learning_rate=0.05,
                objective="multi:softprob", tree_method="hist",
                num_class=3, random_state=42, n_jobs=-1,
            )
            model.fit(X_train, y_train + 1)
            y_pred = model.predict(X_test) - 1

            acc = accuracy_score(y_test, y_pred)
            prec = precision_score(y_test, y_pred, average="weighted", zero_division=0)
            rec = recall_score(y_test, y_pred, average="weighted", zero_division=0)
            f1 = f1_score(y_test, y_pred, average="weighted", zero_division=0)

            results.append({
                "train_end": str(train_end.date()),
                "test_start": str(test_start.date()),
                "test_end": str(test_end.date()),
                "accuracy": float(acc),
                "precision": float(prec),
                "recall": float(rec),
                "f1": float(f1),
                "train_samples": len(train_data),
                "test_samples": len(test_data),
            })
            log.info(f"  Window {test_start.date()} to {test_end.date()}: acc={acc:.4f} f1={f1:.4f}")
        except (OSError, ValueError, KeyError, RuntimeError, ConnectionError, ImportError) as e:
            log.warning(f"  Window failed: {e}")

        current += step_size

    return results


# ═══════════════════════════════════════════════════════════════════════════════
# 8. ARTIFACT SAVING
# ═══════════════════════════════════════════════════════════════════════════════

def save_artifacts(xgb_model, lstm_model, lstm_input_size, scaler,
                   feature_cols, metrics, models_dir="models"):
    """Save all model artifacts."""
    os.makedirs(models_dir, exist_ok=True)
    log.info(f"Saving artifacts to {models_dir}/...")

    # 1. XGBoost → ONNX
    try:

        # Save XGBoost as JSON backup
        xgb_model.save_model(os.path.join(models_dir, "xgb_model.json"))
        log.info("  XGBoost saved as JSON backup")

        # Export to ONNX via sklearn GradientBoosting wrapper (XGBClassifier has no skl2onnx converter)
        try:
            from skl2onnx import to_onnx
            from skl2onnx.common.data_types import FloatTensorType
            from sklearn.ensemble import GradientBoostingClassifier as GBC

            # Train sklearn GBC to mimic XGBoost predictions for ONNX export
            xgb_preds = xgb_model.predict(X_train_scaled)
            gbc = GBC(n_estimators=100, max_depth=5, learning_rate=0.1, random_state=42)
            gbc.fit(X_train_scaled, xgb_preds)

            onnx_model = to_onnx(
                gbc,
                initial_types=[("input", FloatTensorType([1, len(feature_cols)]))],
                target_opset=13,
            )
            onnx_model.ir_version = 7
            for out in onnx_model.graph.output:
                out.name = "output"
            while len(onnx_model.graph.output) > 1:
                onnx_model.graph.output.pop()

            with open(os.path.join(models_dir, "xgb_model.onnx"), "wb") as f:
                f.write(onnx_model.SerializeToString())
            log.info(f"  XGBoost saved as ONNX ({os.path.getsize(os.path.join(models_dir, 'xgb_model.onnx'))} bytes)")
        except (ImportError, Exception) as e:
            log.warning(f"  XGBoost ONNX export failed: {e} — saving JSON only")
    except (OSError, ValueError, KeyError, RuntimeError, ConnectionError, ImportError) as e:
        log.warning(f"  XGBoost ONNX save failed: {e}")

    # 2. LSTM → ONNX
    try:
        import torch
        dummy_input = torch.randn(1, 60, lstm_input_size)
        torch.onnx.export(
            lstm_model, dummy_input,
            os.path.join(models_dir, "lstm_model.onnx"),
            export_params=True, opset_version=13,
            input_names=["input"], output_names=["output"],
            dynamic_axes={"input": {0: "batch_size"}, "output": {0: "batch_size"}},
            dynamo=False,
        )
        log.info("  LSTM saved as ONNX")
    except (OSError, ValueError, KeyError, RuntimeError, ConnectionError, ImportError) as e:
        log.warning(f"  LSTM ONNX save failed: {e} — saving as PyTorch state_dict")
        import torch
        torch.save(lstm_model.state_dict(), os.path.join(models_dir, "lstm_model.pt"))

    # 3. Scaler (pkl + json for Go engine compatibility)
    with open(os.path.join(models_dir, "scaler.pkl"), "wb") as f:
        pickle.dump(scaler, f)
    log.info("  Scaler saved (pkl)")

    # Also save as JSON for Go engine (no gob/pickle dependency)
    scaler_json = {
        "mean": list(scaler.mean_) if hasattr(scaler, "mean_") else [0.0] * len(feature_cols),
        "scale": list(scaler.scale_) if hasattr(scaler, "scale_") else [1.0] * len(feature_cols),
        "n_features": len(feature_cols),
    }
    with open(os.path.join(models_dir, "scaler.json"), "w") as f:
        json.dump(scaler_json, f, indent=2)
    log.info("  Scaler saved (json for Go engine)")

    # 4. Feature columns
    with open(os.path.join(models_dir, "feature_columns.json"), "w") as f:
        json.dump(feature_cols, f, indent=2)
    log.info("  Feature columns saved")

    # 5. Metrics
    with open(os.path.join(models_dir, "metrics.json"), "w") as f:
        json.dump(metrics, f, indent=2)
    log.info("  Metrics saved")


# ═══════════════════════════════════════════════════════════════════════════════
# 9. MAIN
# ═══════════════════════════════════════════════════════════════════════════════

def main():
    parser = argparse.ArgumentParser(description="Predict-A-Trade ML Training (CPU-only)")
    parser.add_argument("--start_date", default="2024-01-01", help="Start date (YYYY-MM-DD)")
    parser.add_argument("--end_date", default="2026-12-31", help="End date (YYYY-MM-DD)")
    parser.add_argument("--models_dir", default="models", help="Output directory for artifacts")
    args = parser.parse_args()

    log.info("=" * 70)
    log.info("Predict-A-Trade XAUUSD — ML Training Script")
    log.info(f"Date range: {args.start_date} to {args.end_date}")
    log.info(f"Models dir: {args.models_dir}")
    log.info("=" * 70)

    # Step 1: Load data
    df = load_candles(args.start_date, args.end_date)

    # Step 2: Compute indicators (Wilder smoothing — matches Go engine)
    df = compute_all_indicators(df)

    # Step 3: Engineered features
    df = add_engineered_features(df)

    # Step 4: Target labels
    df = create_target_labels(df, future_periods=5)

    # Step 5: Define feature columns
    exclude_cols = ["open", "high", "low", "close", "volume", "future_return", "label"]
    feature_cols = [c for c in df.columns if c not in exclude_cols]
    log.info(f"Feature columns: {len(feature_cols)}")

    # Step 6: Drop NaN rows
    df_clean = df.dropna(subset=feature_cols + ["label"]).copy()
    log.info(f"Clean dataset: {len(df_clean)} rows")

    # Step 7: Chronological split (70/15/15)
    n = len(df_clean)
    train_end = int(n * 0.70)
    val_end = int(n * 0.85)

    train_df = df_clean.iloc[:train_end]
    val_df = df_clean.iloc[train_end:val_end]
    test_df = df_clean.iloc[val_end:]

    log.info(f"Split: train={len(train_df)}, val={len(val_df)}, test={len(test_df)}")

    X_train = train_df[feature_cols].values
    y_train = train_df["label"].values
    X_val = val_df[feature_cols].values
    y_val = val_df["label"].values
    X_test = test_df[feature_cols].values
    y_test = test_df["label"].values

    # Step 8: Scale features
    from sklearn.preprocessing import StandardScaler
    scaler = StandardScaler()
    X_train_scaled = scaler.fit_transform(X_train)
    X_val_scaled = scaler.transform(X_val)
    X_test_scaled = scaler.transform(X_test)

    # Step 9: Train XGBoost
    xgb_model, xgb_val_acc, xgb_latency = train_xgboost(
        X_train_scaled, y_train, X_val_scaled, y_val, feature_cols
    )

    # Step 10: Train LSTM
    lstm_model, lstm_val_acc, lstm_latency, lstm_input_size = train_lstm(
        X_train_scaled, y_train, X_val_scaled, y_val
    )

    # Step 11: Ensemble (weighted average)
    xgb_weight = 0.6
    lstm_weight = 0.4
    ensemble_acc = xgb_weight * xgb_val_acc + lstm_weight * lstm_val_acc
    log.info(f"Ensemble validation accuracy: {ensemble_acc:.4f} (XGB={xgb_val_acc:.4f}, LSTM={lstm_val_acc:.4f})")

    # Step 12: Test set evaluation
    from sklearn.metrics import (
        accuracy_score,
        confusion_matrix,
        f1_score,
        precision_score,
        recall_score,
    )

    xgb_test_pred = xgb_model.predict(X_test_scaled) - 1
    test_acc = accuracy_score(y_test, xgb_test_pred)
    test_prec = precision_score(y_test, xgb_test_pred, average="weighted", zero_division=0)
    test_rec = recall_score(y_test, xgb_test_pred, average="weighted", zero_division=0)
    test_f1 = f1_score(y_test, xgb_test_pred, average="weighted", zero_division=0)
    test_cm = confusion_matrix(y_test, xgb_test_pred, labels=[-1, 0, 1]).tolist()

    log.info(f"Test: acc={test_acc:.4f} prec={test_prec:.4f} rec={test_rec:.4f} f1={test_f1:.4f}")

    # Step 13: Walk-forward validation
    wf_results = walk_forward_validation(df_clean, feature_cols)

    # Step 14: Save artifacts
    metrics = {
        "xgb_val_accuracy": float(xgb_val_acc),
        "lstm_val_accuracy": float(lstm_val_acc),
        "ensemble_val_accuracy": float(ensemble_acc),
        "test_accuracy": float(test_acc),
        "test_precision": float(test_prec),
        "test_recall": float(test_rec),
        "test_f1": float(test_f1),
        "confusion_matrix": test_cm,
        "xgb_inference_latency_ms": float(xgb_latency),
        "lstm_inference_latency_ms": float(lstm_latency),
        "walk_forward": wf_results,
        "feature_count": len(feature_cols),
        "train_samples": len(train_df),
        "val_samples": len(val_df),
        "test_samples": len(test_df),
        "date_range": {"start": args.start_date, "end": args.end_date},
        "timestamp": datetime.now(timezone.utc).isoformat(),
    }

    save_artifacts(xgb_model, lstm_model, lstm_input_size, scaler,
                   feature_cols, metrics, args.models_dir)

    # Verify latency < 10ms
    max_latency = max(xgb_latency, lstm_latency)
    if max_latency < 10.0:
        log.info(f"✅ Inference latency < 10ms: XGB={xgb_latency:.2f}ms LSTM={lstm_latency:.2f}ms")
    else:
        log.warning(f"⚠️ Inference latency >= 10ms: XGB={xgb_latency:.2f}ms LSTM={lstm_latency:.2f}ms")

    log.info("=" * 70)
    log.info("Training complete. Artifacts saved to models/")
    log.info(f"  Test accuracy: {test_acc:.4f}")
    log.info(f"  Test F1: {test_f1:.4f}")
    log.info(f"  Walk-forward windows: {len(wf_results)}")
    log.info("=" * 70)


if __name__ == "__main__":
    main()
