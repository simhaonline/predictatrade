---
name: mt5-strategy-tester
description: "MT5 Strategy Tester optimization, WFO, and validation."
---

# mt5-strategy-tester

Use when running MT5 Strategy Tester backtests, optimization, walk-forward, or genetic optimization of Predict-A-Trade EAs.

## MT5 Strategy Tester Modes
1. Single backtest — one pass with fixed parameters
2. Optimization — brute-force or genetic parameter sweep
3. Walk-Forward (WFO) — in-sample optimization then out-of-sample validation
4. Forward test on demo/paper after backtest passes

## Key Settings
- Symbol: XAUUSD (or broker suffix like XAUUSDm, XAUUSD.)
- Timeframe: match strategy TF (M1/M5/H1/H4)
- Model: Every tick (required for scalping; most realistic)
- Spread: current or fixed at max witnessed
- Deposit: realistic starting capital
- Criterion: custom (Sharpe + profit factor + recovery factor)

## Strategy Test Matrix
STANDARD_SCALPING: M5, 3mo, Every tick, min 200 trades
ULTRA_SCALPING: M1, 1mo, Every tick, min 500 trades
STANDARD_SWING: H1, 12mo, Every tick, min 100 trades
TREND_SWING: H4, 12mo, 1-min OHLC, min 50 trades

## Post-Test Checks
1. No look-ahead (bar[0] in OnTick? use bar[1] for closed bars)
2. Slippage >= 2 points for XAUUSD
3. Profit factor > 1.3 min
4. Max drawdown documented
5. Sharpe ratio computed

## Pitfalls
- MT5 tester uses server time, not UTC — DST shifts possible
- Genetic optimization overfits — always WFO validate
- Every-tick model is 50-100x slower than OHLC
- XAUUSD tick size varies 0.01-0.1 by broker
