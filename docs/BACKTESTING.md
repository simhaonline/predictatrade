# Predict-A-Trade Backtesting Framework

## Architecture

The backtesting framework reproduces production trading logic through faithful Python adapters, ensuring live/backtest parity.

```
HISTORICAL DATA
      │
      ▼
DATA VALIDATION (quality checks, UTC, OHLC, duplicates, gaps)
      │
      ▼
TIMEFRAME ALIGNMENT (no look-ahead, closed candles only)
      │
      ▼
MARKET SNAPSHOT (indicators, regime, session)
      │
      ▼
PRODUCTION STRATEGY/PTB (evidence → confluence → direction → SL/TP)
      │
      ▼
ADAPTATION LAYER (phase-based parameter adjustment)
      │
      ▼
RISK GATES (spread, session, R:R, exposure, daily loss, recovery)
      │
      ├── BLOCK → no fill, block reason persisted
      │
      └── ALLOW
            │
            ▼
EXECUTION SIMULATOR (spread, slippage, commission, partial fills)
      │
      ▼
PORTFOLIO (positions, equity, trailing stop, break-even, time exit)
      │
      ▼
ANALYTICS (metrics, walk-forward, Monte Carlo, sensitivity)
      │
      ▼
REPORTS (JSON, CSV, run manifest)
```

## Data Requirements

- XAUUSD candle data (M1, M5, M15, H1, H4, D1)
- All timestamps must be UTC
- CSV or in-memory data supported
- Data quality validation runs before backtest (errors abort)

## No-Lookahead Guarantees

- Higher-timeframe candles are only included if they CLOSED before the primary candle timestamp
- A partially formed higher-TF candle is never used
- Automated tests verify no look-ahead contamination

## Execution Model

| Feature | Default | Configurable |
|---------|---------|-------------|
| Spread | Fixed 0.30 | fixed, dynamic, historical |
| Slippage | Fixed 0.05 | fixed, percentage, ATR-based |
| Commission | $7/lot | yes |
| Latency | 0ms | yes |
| Partial fills | 0% probability | yes |
| Rejections | 0% probability | yes |
| Direction correctness | LONG=ask, SHORT=bid | automatic |
| Same-bar SL/TP | Conservative (SL first) | configurable |

## Exit Management

| Feature | Default | Configurable |
|---------|---------|-------------|
| Stop-loss | Per signal | yes |
| Take-profit | Per signal | yes |
| Trailing stop | ATR × 2.0 | yes |
| Break-even | 1R trigger | yes |
| Time exit | Disabled (0) | configurable bars |
| End-of-test close | Market close | automatic |

## PTB/Live Parity

The PTB strategy adapter reproduces the Go production strategy logic:
- Evidence generation (trend, momentum, structure, liquidity, candle patterns)
- Confluence scoring (family caps, PHI threshold, score separation)
- Direction determination (long/short score comparison)
- Conflict detection (MTF, regime, spread)
- Entry/SL/TP calculation (ATR-based)
- Risk gate evaluation (spread, session, R:R, exposure)

## Risk Gates (Production Reuse)

- Max risk per trade (2% default)
- Max daily loss percent (2%)
- Max consecutive losses (2 → recovery mode)
- Max positions (3)
- Max exposure (5 lots)
- Min R:R (strategy-specific)
- Spread-to-ATR gate
- Session restrictions

## Feature Precomputation

Precompute PTB features once, replay without re-computing:
```bash
python -m patresearch.backtesting.cli precompute --symbol XAUUSD --timeframe M5
```

## Walk-Forward

```bash
python -m patresearch.backtesting.cli walk-forward --strategy STANDARD_SCALPING
```

## Monte Carlo

```bash
python -m patresearch.backtesting.cli monte-carlo --strategy STANDARD_SCALPING --runs 1000
```

## Sensitivity Analysis

```bash
python -m patresearch.backtesting.cli sensitivity --strategy STANDARD_SCALPING
```

## CLI Commands

```bash
# Run a backtest
python -m patresearch.backtesting.cli run --symbol XAUUSD --strategy STANDARD_SCALPING --timeframe M5 --seed 42

# Validate data
python -m patresearch.backtesting.cli validate-data --data-file candles.csv --symbol XAUUSD --timeframe M5

# Precompute features
python -m patresearch.backtesting.cli precompute --symbol XAUUSD --timeframe M5

# Walk-forward analysis
python -m patresearch.backtesting.cli walk-forward --strategy STANDARD_SCALPING

# Monte Carlo
python -m patresearch.backtesting.cli monte-carlo --strategy STANDARD_SCALPING --runs 1000

# Sensitivity analysis
python -m patresearch.backtesting.cli sensitivity --strategy STANDARD_SCALPING

# List runs
python -m patresearch.backtesting.cli list

# Show run details
python -m patresearch.backtesting.cli show <run-id>
```

## Reproducibility

Every backtest generates a `run_manifest.json` with:
- Run ID, symbol, strategy, timeframe
- Data hash, feature version, model version
- Configuration, random seed, execution assumptions
- Git commit SHA, application version
- Artifact locations

Same data + config + seed = same results.

## Database Persistence

Migration 015 creates tables:
- `trading.backtest_runs` — top-level run tracking
- `trading.backtest_trades` — individual trade records
- `trading.backtest_fold_results` — walk-forward fold outcomes
- `trading.backtest_artifacts` — file locations
- `trading.backtest_parameter_sets` — parameter search grid


## Vectorized Indicator Engine (v1.5.0)

A fully vectorized pandas/numpy indicator engine (`QuantitativeStrategyEngine`)
is available alongside the event-driven backtesting framework for fast batch
indicator computation across large historical datasets:

```python
from patresearch import QuantitativeStrategyEngine

engine = QuantitativeStrategyEngine()
result = engine.generate_composite_signals(df)  # df: OHLCV with DatetimeIndex
# Returns: original columns + all indicators + 'signal' (-1/0/1) + stop_loss/take_profit
```

- 7 vectorized indicators: SMA, EMA, ADX, RSI, MACD, Bollinger Bands, ATR
- 6 signal methods: EMA crossover, ADX directional, RSI mean-reversion, MACD crossover, Bollinger reversal, ATR breakout
- Composite pipeline: EMA(50/200) trend filter → RSI/BB entry triggers → ATR-based dynamic stops
- Module: `research/src/patresearch/quantitative_strategy_engine.py`
- Tests: `research/tests/test_quantitative_strategy_engine.py` (29 tests)

This complements (does not replace) the scalar `reference_math.py` used for Go parity verification.

## Testing

```bash
cd research && python3 -m pytest tests/ -v
```

## Limitations

- Synthetic data produces NO_TRADE (correct behavior with strict thresholds)
- Live broker execution is a separate concern from backtesting
- RL model files require external training before use
- Historical data from TimescaleDB requires database connection
