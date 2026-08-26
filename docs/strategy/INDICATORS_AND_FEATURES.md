# Indicators & Features
## v1.16.0 — 26 August 2026

### Evidence Pillars (13 pillars, family-capped)

| Pillar | Cap | Features |
|--------|:---:|----------|
| TREND | 0.35 | EMA 9/21/50/100/200, SMA, ADX |
| MOMENTUM | 0.30 | MACD, OsMA, RSI, Stoch, CCI |
| STRUCTURE | 0.25 | BOS/CHoCH, BSL/SSL, pivots, pullback |
| LIQUIDITY | 0.20 | Sweep detection, order blocks |
| SMC | 0.20 | FVG, imbalance, displacement |
| MTF | 0.20 | HTF alignment, trend direction |
| CANDLE | 0.20 | Rejection, displacement, pin bar |
| REGIME | 0.15 | Regime engine output |
| VWAP | 0.15 | VWAP deviation, bands |
| VOLATILITY | 0.15 | ATR, BB width, range compression |
| ML | 0.25 | ONNX model inference |
| SENTIMENT | 0.25 | Ollama sentiment analysis |
| SESSION_ORB | 0.15 | Asian/London/NY opening ranges |

### Indicator Status: 35/42 LIVE, 7 warming up

Live indicators include: EMA, MACD, ADX, RSI, ATR, VWAP, SAR, Ichimoku, Bollinger, Stoch, CCI, OBV, pivots (D/W/M), Fibonacci

### P2 Features (ACTIVE, v1.16.0)
- **Session ORB:** Asian/London/NY opening ranges, breakout direction, compression
- **Pin Bar:** Body/wick ratios, rejection direction, quality score (0-1)
- **Pullback:** Depth %, ATR-normalized retracement, continuation confirmation
- **Trade Group ID:** Multi-position signal split tracking
