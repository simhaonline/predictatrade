---
name: python-data-quality
description: "Validate research data provenance, gaps, and drift."
---

# python-data-quality

Use when validating Predict-A-Trade research datasets for quality, provenance, gaps, duplicates, or distribution drift.

## Data Sources
- Twelve Data API (XAUUSD spot, DXY)
- FMP API (COT positioning)
- Broker MT5 tick/candle feed

## Commands
cd research
python scripts/quant_validation.py
python tests/backtesting/test_data.py
python scripts/verify_math_parity.py

## Checks
1. Timestamp monotonicity
2. OHLC consistency: high >= max(open,close), low <= min(open,close)
3. No zero/negative prices
4. Gap detection: > 5x median candle spread
5. Duplicate timestamps: keep latest, flag
6. Session coverage: 24/5 XAUUSD, exclude weekends
7. Broker vs API divergence: flag > 0.1% price difference

## Drift Detection
KS test, PSI (Population Stability Index)
Alert if any feature drift > 0.25 PSI

## Provenance
Every dataset: source, date range, broker, preprocessing steps, export timestamp.
Store in research/data/manifest.json or database.
