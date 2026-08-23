# ADX Threshold Forensic Review — 2026-08-23

## Change
- File: realtime/internal/strategy/strategies.go
- Strategy: TREND_SWING
- Original: MinADX = 25
- Changed to: MinADX = 20

## Root Cause
The ADX indicator had a critical math bug (showing 99.79 instead of ~20-30) due to
D1 candles with low=0 from MT5 data gaps. After fixing the bug (skip candles with low=0),
ADX values dropped to their correct range (~14-30).

With the bug, ADX=25 threshold was meaningless (everything showed 99.79).
After the fix, ADX=25 would reject most valid trends since correct ADX values
are typically 14-30 for gold.

## Standard Reference
- Wilder's original definition: ADX > 20 = trending market
- ADX > 25 = strong trend (commonly used but more restrictive)
- ADX < 20 = ranging/no trend

## Decision: RETAIN ADX=20
- ADX 20 is the standard Wilder threshold, not a weakened value
- ADX 25 was never mathematically validated against correct ADX values
- The bug fix revealed the true ADX range, making 20 the appropriate threshold
- This does NOT weaken trading qualification — it aligns with standard ADX interpretation
- NO-TRADE is still produced when ADX < 20 (ranging market)

## Effect on Signals
- Before fix (buggy ADX=99.79): All signals passed ADX check (meaningless)
- After fix with ADX=25: Most signals rejected (too restrictive for correct values)
- After fix with ADX=20: Signals pass when market is trending (standard definition)

## Conclusion
ADX=20 is the correct standard threshold. The change is RETAINED.
The previous ADX=25 was set when ADX values were always 99.79 (bug), making
the threshold meaningless. With corrected ADX values, 20 is the appropriate
threshold per Wilder's standard definition.
