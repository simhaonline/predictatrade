# Strategy Playbooks
## v1.16.0 — 26 August 2026

### Engine Inventory

| # | Engine | ID | TFs | Min Score | Expiry | Mode |
|---|--------|----|-----|:---------:|:------:|:----:|
| 1 | Standard Scalping | STANDARD_SCALPING | M1/M5 | 65 | 10m | LIVE |
| 2 | Ultra Scalping | ULTRA_SCALPING | M1 | 60 | 5m | LIVE |
| 3 | Standard Swing | STANDARD_SWING | M15/H1 | 68 | 30m | LIVE |
| 4 | Trend Swing | TREND_SWING | H1/H4 | 70 | 60m | LIVE |
| 5 | MARNIE_FIB | MARNIE_FIB | H1 | 70 | 60m | SHADOW |

### Standard Scalping (M1/M5)
- Personality: Quick scalping, high-frequency, low-exposure
- Decision TF: M1, HTF confirmation: M5
- Min ATR: 5 pips, Max spread: 2.5 pips
- SL buffer: 1.5x ATR, TP1: 1.0x, TP2: 2.0x, TP3: 3.0x
- Regime: TRENDING_BULLISH, TRENDING_BEARISH, RANGE

### Ultra Scalping (M1)
- Personality: Ultra-fast, lower thresholds, quick exits
- Decision TF: M1
- Min ATR: 3 pips, Max spread: 2.0 pips
- SL buffer: 1.0x ATR, TP1: 0.75x, TP2: 1.5x

### Standard Swing (M15/H1)
- Personality: Medium-term swings, structure-focused
- Decision TF: M15, HTF: H1
- Min ATR: 8 pips, Max spread: 3.0 pips
- SL buffer: 2.0x ATR, TP1: 1.5x, TP2: 3.0x

### Trend Swing (H1/H4)
- Personality: Long-term trend following, high conviction
- Decision TF: H1, HTF: H4
- Min ATR: 12 pips, Max spread: 3.5 pips
- SL buffer: 2.5x ATR, TP1: 2.0x, TP2: 4.0x

### MARNIE_FIB (H1) — SHADOW
- Personality: Fibonacci confluence trader
- Fibonacci levels: 38.2%, 50%, 61.8%, 78.6%
- Requires BOS/CHoCH + 2+ fib confluences
- Accumulating outcomes before activation
