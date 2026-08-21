# Predict-A-Trade Project Details

## Scan scope

Scanned 783 tracked files, including 654 implementation, configuration, and documentation files across Go, Python, TypeScript, MQL, SQL, JSON/TOML/YAML, shell, PowerShell, HTML, and Markdown.

## Indicators

### Production Go indicators

Defined in `realtime/internal/features/indicators.go` and assembled by `realtime/internal/features/registry.go`.

- Trend: EMA 9/21/50/100/200; SMA 50/100/200; EMA 9/21 crossover.
- MACD: main, signal, histogram, bullish cross, bearish cross; 12/26/9.
- Directional strength: ADX 14, +DI, -DI.
- Momentum: RSI 14; Stochastic %K/signal 14/3/3; CCI 20; Momentum; OsMA.
- Volatility: ATR 14; Bollinger upper/middle/lower bands, width, bullish/bearish reversals.
- Volume: OBV from broker tick volume; OBV Z-score; tick-volume Z-score; Bollinger-width Z-score.
- Price/context: session VWAP, rolling VWAP, VWAP upper/lower bands.
- Additional local indicators: Parabolic SAR 0.02/0.20; Ichimoku Tenkan, Kijun, Senkou A/B, Chikou, cloud state; StochRSI raw/K/D.
- Derived indicators: EMA cross, MACD crosses, Bollinger reversals, SAR direction, and Ichimoku cloud position.

The values are stored in `realtime/internal/features/state.go` in `IndicatorFeatures`.

### Python research indicators

`research/src/patresearch/quantitative_strategy_engine.py` provides a separate vectorized research implementation:

- SMA, EMA, RSI, ADX, +DI, -DI, MACD, Bollinger Bands, ATR.
- EMA crossover, ADX directional, RSI mean-reversion, MACD crossover, Bollinger reversal, and ATR-channel breakout signals.
- Composite EMA/RSI/Bollinger/ATR signal pipeline.

Tests are in `research/tests/test_quantitative_strategy_engine.py`.

### Unsupported or unavailable indicators

- Volume Profile.
- Cumulative Delta/CVD.
- DOM/order-book depth and institutional footprint.
- Centralized real volume.
- COT unless an external provider is configured.
- DXY, silver, and yield context unless feeds are connected.

The code explicitly prevents fabricating these capabilities.

## Features

### Market structure

Implemented by `realtime/internal/features/structure.go`:

- Confirmed fractal swing highs/lows.
- Break of Structure (BOS).
- Change of Character (CHoCH).
- Market Structure Shift fields (MSS).
- Current trend and structure-break state.
- Two-bar confirmation with no look-ahead.

### Liquidity and SMC/ICT-style features

Implemented by `liquidity.go` and `fvg.go`:

- Equal-high/equal-low liquidity pools.
- Buy-side and sell-side liquidity sweeps.
- Fair Value Gaps (FVG) and Inverted FVG fields.
- Order Blocks and Breaker Blocks.
- FVG fill and mitigation state.

### Candle, price, and geometry features

- Candle body, range, upper/lower wick, ATR-normalized range.
- Bullish, bearish, doji, pin-bar, engulfing, inside-bar, and outside-bar patterns.
- Displacement, rejection, breakout, compression, and expansion.
- Consecutive bullish/bearish candles.
- Fibonacci retracement and daily/weekly pivots.
- Support/resistance levels.
- Entry, stop-loss, TP1/TP2/TP3 geometry.
- Risk/reward and cost-to-target geometry.

### Session and timeframe features

Implemented by `session.go` and `mtf.go`:

- TOKYO, LONDON, OVERLAP, NEW_YORK, and SYDNEY sessions.
- Weekend and off-hours detection.
- News-risk state.
- Per-timeframe bullish/neutral/bearish state.
- Weighted multi-timeframe alignment score.

### Regime features

Implemented by `regime.go`:

- Strong/weak trend up/down.
- Range-bound, high-volatility, low-volatility, transitioning, and manipulation regimes.
- Raw versus confirmed regime.
- Confidence, age, entry reason, transition candidate, confirmation count, hysteresis, and confidence decay.

### PTB features and modules

Implemented by `realtime/internal/ptb/modules.go`:

- Liquidity Void.
- Wick Fill.
- Session Imbalance.
- Candle Range Projector.
- Time At Mode.
- Engineered Liquidity Proxy.
- Market Phase.
- Relative Volume Flow.
- Price Delivery.
- Stop Hunt Proxy.
- Time Cycle Analytics.
- Algo Activity Proxy.
- Complete Liquidity Map.
- Manipulation Proxy.
- MTF Bias.
- Volatility Regime.
- Support/Resistance Quality.
- Microstructure.
- Statistical Performance.
- Data Quality.

Most PTB modules are shadow or warming-up; institutional footprint is unsupported.

PTB synthesis also produces confluence, bias, setup grade, position-size and stop-distance multipliers, actions, reason codes, deterministic narratives, enhanced regime, gold-role classification, correlations, liquidity targeting, and component-score breakdowns.

### ML/RL feature vectors

ML adaptation features include regime, confluence, confidence, manipulation index, volatility, liquidity, spread, ATR, session, recent returns, and sentiment. The persisted model schema is `models/feature_columns.json`.

RL observations include regime, confluence, confidence, manipulation, volatility, liquidity, sentiment, DXY, real yields, session, spread, ATR, recent returns, and position state.

## Engines

### Feature and indicator engines

Constructed and run by `realtime/internal/features/registry.go`:

- `IndicatorEngine`
- `StructureEngine`
- `LiquidityEngine`
- `FVGEngine`
- `VWAPEngine`
- `RegimeEngine`
- `MTFEngine`
- `SessionEngine`
- `SAREngine`
- `IchimokuEngine`
- `StochRSIEngine`
- `FibonacciEngine`
- `PivotEngine`
- `CandleEngine`
- Rolling-statistics/Z-score engine

`Registry.Evaluate()` runs these engines and builds `MarketState`.

### Strategy engines

Implemented under `realtime/internal/strategy/engines`:

- `StdScalpEngine` → `STANDARD_SCALPING`.
- `UltraScalpEngine` → `ULTRA_SCALPING`.
- `StdSwingEngine` → `STANDARD_SWING`.
- `TrendSwingEngine` → `TREND_SWING`.

Legacy/base strategy implementations remain in `realtime/internal/strategy/strategies.go`. Specialized engines are selected through the factory and applied in `realtime/cmd/realtime-engine/main.go`.

### Signal, market, and research engines

- Signal decision engine: `realtime/internal/signal/engine.go`.
- Hard-gate registry: `realtime/internal/gates/gates.go`.
- PTB engine: `realtime/internal/ptb/modules.go`.
- PTB correlation engine: `realtime/internal/ptb/correlation.go`.
- News risk engine: `realtime/pkg/news/risk_engine.go`.
- News breakout engine: `realtime/internal/breakout/breakout.go`.
- Replay engine: `realtime/internal/replay/engine.go`.
- Python event-driven backtest engine: `research/src/patresearch/backtesting/engine/core.py`.
- Python vectorized quantitative strategy engine: `research/src/patresearch/quantitative_strategy_engine.py`.

### ML, RL, and adaptive engines

- ONNX `MLEngine`: loads XGBoost and LSTM models from `models/`.
- ML `AdaptationManager` and `ModelRegistry`.
- RL strategy optimizer.
- Asynchronous sentiment engine.
- Offline ML training pipeline.
- Offline RL environment and training pipeline.

### Risk and lifecycle managers

- Loss recovery manager.
- Market adaptation manager.
- Controlled hedge manager.
- OCO manager.
- Signal delivery manager.
- Cooldown and duplicate-signal managers.
- Reconciliation engine.
- Maintenance scheduler.

These are wired through `realtime/internal/signal/advanced.go` and the realtime command process.

## LLMs

Only one LLM integration was found: the Ollama client in `realtime/pkg/ollama/client.go`.

- Service: Ollama HTTP API.
- Default host: `http://localhost:11434`.
- Default model: `deepseek-v4-pro:cloud`.
- Configuration: `OLLAMA_ENABLED`, `OLLAMA_HOST`, `OLLAMA_MODEL`, and `OLLAMA_TIMEOUT`.
- Purpose: news/headline sentiment scoring.
- Output: sentiment score from -1.0 to +1.0.
- Evidence tag: `OLLAMA_LLM`.
- Runtime integration: `realtime/cmd/realtime-engine/main.go`.

No direct OpenAI, Anthropic/Claude, Gemini, Llama, Mistral, or other named LLM API integration was found.

## Runtime and status notes

- Deterministic Go indicators and strategies are the authoritative live path.
- ML defaults to disabled through `ML_ENABLED=false`.
- RL defaults to `disabled`.
- Ollama defaults to disabled through `OLLAMA_ENABLED=false`.
- XGBoost and LSTM ONNX artifacts exist, but activation is configuration-controlled.
- `realtime/web/live.html` contains a separate 14-indicator dashboard shell. Its demo initialization uses synthetic/random display data and is not authoritative live computation.
- Repository README and deployment reports describe Ollama as active, but the source-code default is disabled; actual status depends on environment configuration.

## Primary usage path

`realtime/cmd/realtime-engine/main.go` follows the main runtime flow:

1. `features.Registry.Evaluate()` builds market features.
2. PTB evaluates shared intelligence.
3. Each of the four strategies evaluates independently.
4. Specialized strategy-engine overrides are applied when available.
5. ML and Ollama sentiment are optionally injected.
6. Hard gates and advanced safety managers evaluate the candidate.
7. The signal engine decides and delivers the result through WebSocket/Windows/MT adapters.

