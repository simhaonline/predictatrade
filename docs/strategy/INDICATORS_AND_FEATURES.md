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

### Indicator Status (42 total)

| Status | Count | Examples |
|--------|:-----:|----------|
| LIVE | 35 | EMA, MACD, ADX, RSI, ATR, VWAP, SAR, Ichimoku, Bollinger |
| WARMING UP | 7 | Needs more candles for period parameters |
| DISABLED | 0 | — |

### P2 Features (ACTIVE, v1.16.0)

| Feature | Pillar | Description |
|---------|--------|-------------|
| Session ORB | SESSION_ORB | Asian/London/NY opening ranges, breakout detection |
| Pin Bar | CANDLE | Body/wick ratios, rejection direction, quality score |
| Pullback | STRUCTURE | Depth %, ATR retracement, continuation confirmation |
| Trade Group ID | — | Multi-position signal split tracking |