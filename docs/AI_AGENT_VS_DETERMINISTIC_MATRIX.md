# AI Agent vs Deterministic Matrix

## Version: v1.0.0 — Stage 4 PTB

## Conclusion

The entire production trading pipeline is **deterministic**. No AI, ML, LLM, or VLM inference occurs in the runtime path.

## Classification

| Component | Classification | Notes |
|-----------|---------------|-------|
| EMA/RSI/MACD/ADX/ATR/Bollinger | Deterministic Mathematical | Canonical formulas |
| Stochastic | Deterministic Mathematical | %K + 3-period SMA signal |
| Structure (BOS/CHoCH) | Rule-Based Analytical | Fractal swing detection |
| Liquidity (pools/sweeps) | Rule-Based Analytical | Equal highs/lows + wick sweeps |
| FVG/Order Blocks | Rule-Based Analytical | 3-candle gap detection |
| Regime | Rule-Based Analytical | EMA stack + ADX + RSI + ATR |
| MTF Alignment | Deterministic Mathematical | Weighted close>open per TF |
| Session | Rule-Based Analytical | UTC hour-based session map |
| Candle Intelligence | Rule-Based Analytical | Body/wick ratios |
| VWAP | Deterministic Mathematical | Session-anchored from MT5 |
| Volume Profile | UNSUPPORTED | Broker tick volume only |
| Cumulative Delta | UNSUPPORTED | No centralized order-flow |
| Institutional Footprint | UNSUPPORTED | No DOM/Level2/Time&Sales |
| Scoring Engine | Rule-Based Analytical | Weighted evidence summation |
| Calibration | Deterministic Mathematical | Sigmoid(a*x+b) with clamping |
| Final Decision | Rule-Based Analytical | Threshold + conflict + gates |
| ML | DISABLED/RESEARCH | No trained model loaded |

## PTB Module Classifications

| Module | Type | AI? | Status |
|--------|------|-----|--------|
| Liquidity Void | Deterministic | No | SHADOW |
| Wick Fill | Statistical | No | SHADOW |
| Session Imbalance | Rule-Based | No | SHADOW |
| Candle Range Projector | Statistical | No | SHADOW |
| Time At Mode | Statistical | No | SHADOW |
| Engineered Liquidity Proxy | Rule-Based Proxy | No | SHADOW |
| Market Phase | Rule-Based | No | SHADOW |
| Relative Volume Flow | Deterministic Proxy | No — tick volume proxy | SHADOW |
| Price Delivery | Statistical | No | SHADOW |
| Stop Hunt Proxy | Rule-Based Proxy | No | SHADOW |
| Institutional Footprint | UNSUPPORTED | N/A | UNSUPPORTED |
| Time Cycle | Statistical | No | SHADOW |
| Algo Activity Proxy | Rule-Based Proxy | No | SHADOW |
| Complete Liquidity Map | Rule-Based | No — INFERRED_PRICE_STRUCTURE | SHADOW |
| Manipulation Proxy | Rule-Based Proxy | No | SHADOW |
| MTF Bias Engine | Deterministic | No | SHADOW |
| Volatility Regime Engine | Deterministic | No | SHADOW |
| S/R Quality Engine | Rule-Based | No | SHADOW |
| Microstructure Engine | Deterministic | No | SHADOW |
| Data Quality Engine | Deterministic | No | ACTIVE |
| Statistical Performance | Statistical | No | SHADOW |

## PTB Synthesis Engine Classification

| Component | Classification | AI? |
|-----------|---------------|-----|
| Synthesis Engine | Rule-Based Analytical | No |
| Confluence Scoring | Deterministic Mathematical | No |
| Bias Determination | Rule-Based | No |
| Setup Quality Grading | Rule-Based | No |
| Position Size Multiplier | Rule-Based | No |
| Stop Distance Multiplier | Rule-Based | No |
| Action Determination | Rule-Based | No |
| Market Narrative | Deterministic Template | No |
| Reason Codes | Rule-Based | No |
| Gold Correlation Engine | Deterministic Mathematical (Pearson) | No |
| Gold Role Classification | Rule-Based | No — returns UNKNOWN without data |
| ML Pattern Layer | DISABLED/RESEARCH | No inference occurs |

## Key Principle

Something is ML only if it:
- Loads a real trained model artifact
- Has documented features
- Has training dataset provenance
- Has validation results
- Has model version
- Performs real inference

Until then: `ML STATUS = DISABLED / RESEARCH`
