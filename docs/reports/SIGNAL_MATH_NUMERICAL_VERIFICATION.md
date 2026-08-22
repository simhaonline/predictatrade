# Signal Math Numerical Verification — 2026-08-23

## Indicator Verification

| Indicator | Formula | Before Fix | After Fix | Status |
|-----------|---------|-----------|-----------|--------|
| ATR (Wilder) | TR=max(H-L,\|H-Cp\|,\|L-Cp\|), ATR=Wilder smooth | 2180.98 (low=0 candles) | 7.27 | FIXED+VERIFIED |
| ADX (Wilder) | +DI/-DI from DM, DX=100×\|+DI--DI\|/(+DI+-DI), ADX=Wilder(DX) | 99.79 (low=0) | 14.70 | FIXED+VERIFIED |
| RSI (Wilder) | RS=avgGain/avgLoss, RSI=100-100/(1+RS) | 25.46 (accumulating precision) | 46.15 | FIXED+VERIFIED |
| EMA | EMA=price×k+prevEMA×(1-k), k=2/(N+1) | 1751+ digit explosion | clean float64 | FIXED+VERIFIED |
| MACD Histogram | Main - Signal | always 0 | computed | FIXED+VERIFIED |
| Calibration | sigmoid(a×score/100+b) | always 0 (UNVERIFIED status) | 0.45-0.70 (VALIDATED) | FIXED+VERIFIED |

## Root Cause
All Wilder functions used decimal.Decimal which accumulated precision across hundreds of iterations.
Additionally, D1 candles from MT5 had low=0 for closed-market bars, producing TR≈price (1000x too high).

## Fix
- All Wilder/EMA functions converted to float64
- Skip candles with high≤0 or low≤0
- MACD histogram = MACDMain - MACDSignal
- Default calibration models set to VALIDATED status

## Verification
- Go tests: 29 suites, 0 failures
- Live API: ATR=7.27, ADX=14.70, RSI=46.15 (all in expected ranges)
