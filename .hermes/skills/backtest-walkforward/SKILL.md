---
name: backtest-walkforward
description: "Walk-forward backtesting with OOS validation."
---

# backtest-walkforward

Use for PAT walk-forward backtesting.

Location: research/src/patresearch/backtesting/
Scripts: scripts/oos_walkforward_calibrate.py, scripts/walk_forward_baseline.py

Steps: split train/test, optimize in-sample, freeze, test OOS, slide, repeat

Metrics: return, Sharpe, Sortino, max DD, profit factor, win rate, expectancy, trades count, Brier/ECE

Go/No-Go: Sharpe > 0.5 OOS, PF > 1.3, DD < 30%, > 100 trades, ECE < 0.1
