---
name: python-numpy-scipy
description: "NumPy/SciPy patterns for quant research in PAT."
---

# python-numpy-scipy

Use when writing quantitative analysis, signal computation, or statistical tests in Predict-A-Trade Python.

## Core Patterns

### Vectorized EMA
alpha = 2 / (span + 1)
df['close'].ewm(span=span, adjust=False).mean()

### Brier Score
brier = ((probs - outcomes) ** 2).mean()

### Wilson CI for Win Rate
ci = stats.binom.interval(0.95, n=n_trades, p=win_rate) / n_trades

### Feature Scaling
scaler = StandardScaler()
features_scaled = scaler.fit_transform(features)
Export: json.dump(mean, scale) to scaler.json

### Walk-Forward
Train on [t-N, t], test on [t, t+M], slide forward.
Use TimeSeriesSplit — never shuffle time series.

## Key Scripts
scripts/quant_validation.py — full validation
scripts/oos_walkforward_calibrate.py — OOS
scripts/walk_forward_baseline.py — baseline
scripts/strategy_change_gate.py — change approval

## Pitfalls
pd.rolling on M1 is O(N^2) without pre-computed index
sklearn cross_val_score shuffles by default — use TimeSeriesSplit
np.nan in features breaks ONNX — always fillna(0)
