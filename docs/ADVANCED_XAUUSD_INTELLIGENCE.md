# Advanced XAUUSD Intelligence Module Status

## Version: v1.8.0 — Trade Management Audit + Broker Stop Validation + Cost-Aware Break-Even

## PTB Architecture

The Professional Trader Brain (PTB) is a shared intelligence layer that enriches — but does NOT replace — the existing four strategy engines. PTB provides context, evidence, and recommendations. Final trading decisions remain through the existing canonical path: strategy → gates → risk → persistence → MT4/MT5.

All modules start in SHADOW mode (calculate, persist, observe — zero score impact).

## Module Status Table

| Module | Data Source | Status | Live Score? | Strategy Use | Limitation |
|--------|------------|--------|-------------|-------------|-----------|
| Liquidity Void | Real candles | SHADOW | No | All (future) | Needs multi-candle history |
| Wick Fill | Real candle history | SHADOW | No | All (future) | Needs accumulated stats |
| Session Imbalance | Real session data | SHADOW | No | All (future) | None |
| Candle Range Projector | Real history | SHADOW | No | All (future) | Needs historical accumulation |
| Time At Mode | Real observations | SHADOW | No | All (future) | Needs price-bin history |
| Engineered Liquidity Proxy | Real structure | SHADOW | No | All (future) | Proxy only |
| Market Phase | Real candles | SHADOW | No | All (future) | None |
| Relative Volume Flow | Real tick volume | SHADOW | No | All (future) | Tick volume proxy, NOT real volume |
| Price Delivery | Real session history | SHADOW | No | All (future) | Needs accumulated data |
| Stop Hunt Proxy | Real price behavior | SHADOW | No | All (future) | Proxy only |
| Institutional Footprint | N/A | UNSUPPORTED | No | None | No DOM/Level2/Time&Sales feed |
| Time Cycle Analytics | Real history | SHADOW | No | All (future) | Needs accumulated data |
| Algo Activity Proxy | Real tick data | SHADOW | No | All (future) | Cannot prove algorithmic origin |
| Complete Liquidity Map | Real structure | SHADOW | No | All (future) | INFERRED_PRICE_STRUCTURE only |
| Manipulation Proxy | Real price behavior | SHADOW | No | All (future) | Proxy only |
| MTF Bias Engine | Real candles | SHADOW | No | All (future) | None |
| Volatility Regime Engine | Real history | SHADOW | No | All (future) | Needs percentile accumulation |
| S/R Quality Engine | Real structure | SHADOW | No | All (future) | Needs touch count history |
| Microstructure Engine | Real ticks | SHADOW | No | All (future) | No order book claims |
| Statistical Performance | Real outcomes | SHADOW | No | All (future) | Needs outcome data |
| Data Quality Engine | Real attributes | ACTIVE | No (info) | All | None |
| Synthesis Engine | All components | SHADOW | No | All (future) | Advisory only |
| Gold Correlation Engine | Awaiting live feed | SHADOW | No | All (future) | DXY/silver/yields not connected |
| Gold Role Classification | Awaiting live feed | SHADOW | No | All (future) | Returns UNKNOWN without data |
| ML Pattern Layer | N/A | DISABLED | No | None | No trained model |

## Synthesis Engine Output

```
timestamp, symbol, analysis_id
regime, gold_role, volatility_state, manipulation_index
bias (STRONG_LONG → STRONG_SHORT), bias_strength, confidence
support_levels, resistance_levels, liquidity_targets
optimal_entry_time, time_confidence
market_narrative, key_drivers, risk_factors
setup_quality (A+ → F), position_size_multiplier, stop_distance_multiplier
confluence_score, action (ENTER/WAIT/AVOID/EXIT)
positive_factors, negative_factors, vetoes, reason_codes
component_scores, data_quality
model_version, config_version, shadow_mode
```

## Enhanced Regime States

```
STRONG_TREND_UP, STRONG_TREND_DOWN, WEAK_TREND_UP, WEAK_TREND_DOWN,
RANGE_BOUND, HIGH_VOLATILITY, LOW_VOLATILITY, TRANSITIONING, MANIPULATION
```

## Configuration

All thresholds centralized in `ptb/config.go` with environment variable overrides:
- `PTB_ENABLED` — master switch
- `PTB_SHADOW_MODE` — shadow vs active
- Grade thresholds, position multipliers, stop multipliers, manipulation thresholds
- Correlation windows, freshness limits, MTF weights, family caps
