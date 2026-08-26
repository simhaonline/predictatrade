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
**Personality:** Quick scalping, high-frequency, low-exposure
- Decision TF: M1, HTF confirmation: M5
- Min ATR: 5 pips
- Max spread: 2.5 pips
- SL buffer: 1.5× ATR
- TP1: 1.0× ATR, TP2: 2.0× ATR, TP3: 3.0× ATR
- Regime: TRENDING_BULLISH, TRENDING_BEARISH, RANGE
- News risk: LOW only

### Ultra Scalping (M1)
**Personality:** Ultra-fast, lower thresholds, quick exits
- Decision TF: M1
- Min ATR: 3 pips
- Max spread: 2.0 pips
- SL buffer: 1.0× ATR
- TP1: 0.75× ATR, TP2: 1.5× ATR
- Regime: all (fast adaptation)
- News risk: NONE only

### Standard Swing (M15/H1)
**Personality:** Medium-term swings, structure-focused
- Decision TF: M15, HTF: H1
- Min ATR: 8 pips
- Max spread: 3.0 pips
- SL buffer: 2.0× ATR (wider)
- TP1: 1.5× ATR, TP2: 3.0× ATR
- Requires HTF trend alignment

### Trend Swing (H1/H4)
**Personality:** Long-term trend following, high conviction
- Decision TF: H1, HTF: H4
- Min ATR: 12 pips
- Max spread: 3.5 pips
- SL buffer: 2.5× ATR (widest)
- TP1: 2.0× ATR, TP2: 4.0× ATR
- Min score: 70
- Requires H4 trend alignment

### MARNIE_FIB (H1) — SHADOW
**Personality:** Fibonacci confluence trader
- Decision TF: H1
- Fibonacci levels: 38.2%, 50%, 61.8%, 78.6%
- Requires: BOS/CHoCH + at least 2 fib confluences
- Status: SHADOW — accumulating outcomes before activation