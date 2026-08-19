# XAUUSD Accuracy Enhancement Report

## Version: v1.0.0 — Stage 4 PTB

## Existing Baseline

The verified backend uses deterministic mathematical indicators and rule-based scoring. No accuracy percentage claims are made without real outcome validation.

## PTB Synthesis Engine

The synthesis engine combines all evidence into a unified assessment with:
- Confluence score (weighted average of component scores)
- Bias determination (STRONG_LONG → STRONG_SHORT)
- Setup quality grading (A+ through F)
- Position size multiplier (advisory only)
- Stop distance multiplier (context-aware)
- Action recommendation (ENTER, WAIT, AVOID)
- Machine-readable reason codes
- Deterministic market narrative

**Status: SHADOW** — Score Contribution = 0 until validated.

## Gold Correlation Engine

Computes rolling Pearson correlations between gold and:
- DXY (US Dollar Index)
- Silver (XAGUSD)
- US10Y yields

**Status: SHADOW (awaiting live feed)** — All external feeds return UNAVAILABLE. No fabricated correlations.

## Gold Role Classification

Determines what drives XAUUSD: CURRENCY, SAFE_HAVEN, MONETARY_ASSET, COMMODITY, INFLATION_HEDGE, or UNKNOWN.

**Status: Returns UNKNOWN without macro data** — No forced classification.

## Advanced Module Status

| Module | Status | Data Source | Score Impact |
|--------|--------|------------|-------------|
| Liquidity Void | SHADOW | Real candles | 0 |
| Wick Fill | SHADOW | Real candle history | 0 |
| Session Imbalance | SHADOW | Real session data | 0 |
| Candle Range Projector | SHADOW | Real history | 0 |
| Time At Mode | SHADOW | Real observations | 0 |
| Engineered Liquidity Proxy | SHADOW | Real structure | 0 |
| Market Phase | SHADOW | Real candles | 0 |
| Relative Tick Volume Flow | SHADOW | Real tick volume | 0 |
| Price Delivery | SHADOW | Real session history | 0 |
| Stop Hunt Proxy | SHADOW | Real price behavior | 0 |
| Institutional Footprint | UNSUPPORTED | N/A | 0 |
| Time Cycle Analytics | SHADOW | Real history | 0 |
| Algo Activity Proxy | SHADOW | Real tick data | 0 |
| Complete Liquidity Map | SHADOW | Real structure | 0 |
| Manipulation Proxy | SHADOW | Real price behavior | 0 |
| MTF Bias Engine | SHADOW | Real candles | 0 |
| Volatility Regime Engine | SHADOW | Real history | 0 |
| S/R Quality Engine | SHADOW | Real structure | 0 |
| Microstructure Engine | SHADOW | Real ticks | 0 |
| Statistical Performance | SHADOW | Real outcomes | 0 |
| Data Quality Engine | ACTIVE | Real attributes | 0 (informational) |
| Synthesis Engine | SHADOW | All components | 0 |
| Gold Correlation Engine | SHADOW | Awaiting live feed | 0 |
| Gold Role Classification | SHADOW | Awaiting live feed | 0 |
| ML Pattern Layer | DISABLED | N/A | 0 |

## No Accuracy Claims

No percentage accuracy improvement is claimed. Shadow modules calculate and persist for future statistical validation. Activation requires measured evidence of incremental value through the performance feedback loop.

## Performance Feedback Loop

The `trading.signal_performance` table (migration 013) stores real outcomes:
- Entry/exit prices, PnL, MAE, MFE
- Time in trade, slippage, execution quality
- TP1/TP2/TP3/SL hit status
- Regime, bias, gold role, manipulation at entry

Future weight adaptation requires validated statistical evidence, not short-term PnL.

## ML Readiness

```
ML STATUS = DISABLED / RESEARCH
```

Infrastructure ready: feature schema, extraction, dataset builder, model interface, shadow-mode output. Activation requires real training data from persisted outcomes, walk-forward validation, and out-of-sample testing.
