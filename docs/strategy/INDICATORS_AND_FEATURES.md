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

### P2 Features (ACTIVE, v1.16.0 — all promoted from SHADOW)

All P2 features are now wired into the live signal scoring pipeline:

- **Session ORB (P2-001):** Asian/London/NY opening ranges with breakout direction detection and range compression analysis. Session boundaries use broker-local time (GMT+3 configurable via BROKER_TIMEZONE). ORB reset logic now uses BrokerLocation() for correct day rollover at broker midnight.
- **Pin Bar (P2-002):** Body/wick ratio analysis, rejection direction scoring, quality score (0-1)
- **Pullback (P2-003):** Depth %, ATR-normalized retracement, continuation confirmation
- **Trade Group ID (P2-004):** Multi-position signal split tracking for grouped trade execution
- **SLO Targets (P2-005):** Availability, latency, error budget monitoring

### Broker Timezone (v1.16.x)

Session classification and ORB range boundaries now use broker-local time (GMT+3) instead of UTC. This fixes a 3-hour offset that affected Tokyo/London/New York session boundary detection and displayed signal timestamps. Controlled by `BROKER_TIMEZONE` environment variable.

### Signal Quality & Rejection Diagnostics (v1.16.x)

Every signal now carries:
- **QualityGrade:** A+, A, B, or REJECTED based on evidence confluence and R:R quality
- **ExpectancyR (EV_R):** Expected value per unit risk
- **ExpectancyScore:** 0-100 normalized quality score
- **PrimaryRejectionReason:** Machine-readable reason for signal rejection (LOW_EXPECTANCY, INVALID_SL, INVALID_TP, RR_BELOW_MIN, SPREAD_TOO_HIGH, DUPLICATE, etc.)
- **RejectionReasons:** All contributing rejection factors
