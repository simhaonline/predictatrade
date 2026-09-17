// indicators_candle.go — P1 (prompt.md): candle-anatomy indicators.
//
// New, non-overlapping additions (existing verified features untouched):
//   - CloseLocationValue (CLV): where the close sits inside the bar range
//   - CloseStreak: consecutive rising/falling closes (bull/bear streak)
//   - MicroReclaim: wick pierces a level, close back on the original side
//
// Formulas (documented per prompt.md requirements):
//
//	CLV = (2C − H − L) / (H − L)   ∈ [−1, 1];  0 when H == L (flat bar)
//	Streak: sign of close-to-close deltas; a zero delta resets the run to 0.
//	  bullStreak = +N (N consecutive rising closes), bearStreak = −N.
//	MicroReclaim(bull) = Low < Level AND Close > Level
//	MicroReclaim(bear) = High > Level AND Close < Level
package features

import "github.com/shopspring/decimal"

// CloseLocationValue returns CLV ∈ [−1, 1]; zero for flat bars (H == L) —
// documented fallback instead of a division-by-zero panic/NaN.
func CloseLocationValue(high, low, close decimal.Decimal) decimal.Decimal {
	rng := high.Sub(low)
	if rng.IsZero() {
		return decimal.Zero
	}
	return close.Sub(close).Add(close.Mul(decimal.NewFromInt(2))).Sub(high).Sub(low).Div(rng)
}

// CloseStreak scans closes ascending in time and returns +N for N consecutive
// rising closes (bull streak), −N for falling (bear streak). A zero delta
// resets the count. Warmup (nil/empty) → 0.
func CloseStreak(closes []decimal.Decimal) int {
	if len(closes) < 2 {
		return 0
	}
	streak := 0
	for i := 1; i < len(closes); i++ {
		switch closes[i].Compare(closes[i-1]) {
		case 1:
			if streak < 0 {
				streak = 1
			} else {
				streak++
			}
		case -1:
			if streak > 0 {
				streak = -1
			} else {
				streak--
			}
		default:
			streak = 0
		}
	}
	return streak
}

// MicroReclaim detects a wick-through-and-close-back pattern at a level.
// bull=true: Low pierced BELOW the level and Close is back ABOVE it.
// bull=false: High pierced ABOVE the level and Close is back BELOW it.
func MicroReclaim(high, low, close, level decimal.Decimal, bull bool) bool {
	if high.IsZero() || low.IsZero() || close.IsZero() || level.IsZero() {
		return false
	}
	if bull {
		return low.LessThan(level) && close.GreaterThan(level)
	}
	return high.GreaterThan(level) && close.LessThan(level)
}