// indicators_geom.go — P1 (prompt.md): geometry/regression indicators.
//
// New, non-overlapping additions (existing verified features untouched):
//   - LinRegSlopeR2: linear-regression slope + R² over the last n closes
//   - LinSlopeATR: slope normalized by ATR (per-bar drift in ATR units)
//   - AsianSweep: wick-through-Asian-range-then-close-back (reuses the
//     existing SessionORB engine's AsianHigh/Low — no new state, no overlap)
//
// Formulas (documented per prompt.md requirements):
//
//	slope = Σ((x−x̄)(y−ȳ)) / Σ((x−x̄)²),   x = 0..n−1 bar index
//	R²    = 1 − SS_res/SS_tot             ∈ [0,1]; 1 = perfect linear fit
//	LinSlopeATR = slope / ATR; zero ATR (warmup) → zero, never infinite
//	AsianSweep(bull) = Low < AsianLow  AND Close > AsianLow
//	AsianSweep(bear) = High > AsianHigh AND Close < AsianHigh
//	Warmup: fewer than n bars, or zero Asian levels → zero/false.
package features

import "github.com/shopspring/decimal"

// LinRegSlopeR2 computes the OLS slope and R² of closes over the last n bars.
// Warmup (len < n) returns zeros.
func LinRegSlopeR2(closes []decimal.Decimal, n int) (slope, r2 decimal.Decimal) {
	if n <= 1 || len(closes) < n {
		return decimal.Zero, decimal.Zero
	}
	w := closes[len(closes)-n:]

	var sx, sy, sxx, sxy, syy float64
	for i, c := range w {
		yf, _ := c.Float64()
		sx += float64(i)
		sy += yf
		sxx += float64(i) * float64(i)
		sxy += float64(i) * yf
		syy += yf * yf
	}
	fl := float64(n)
	sxyBar := sxy - sx*sy/fl
	sxxBar := sxx - sx*sx/fl
	if sxxBar == 0 {
		return decimal.Zero, decimal.Zero
	}
	slopeF := sxyBar / sxxBar

	// R² from the correlation of a perfect-fit prediction: for OLS,
	// R² = sxyBar² / (sxxBar * syyTot) where syyTot = syy − sy²/n
	syyTot := syy - sy*sy/fl
	r2F := 0.0
	if syyTot > 0 {
		r2F = (sxyBar * sxyBar) / (sxxBar * syyTot)
		if r2F < 0 {
			r2F = 0
		}
		if r2F > 1 {
			r2F = 1
		}
	}
	return decimal.NewFromFloat(slopeF), decimal.NewFromFloat(r2F)
}

// LinSlopeATR normalizes the regression slope by ATR. Zero ATR (warmup) →
// zero — documented fallback instead of +Inf.
func LinSlopeATR(slope, atr decimal.Decimal) decimal.Decimal {
	if atr.IsZero() || slope.IsZero() {
		return decimal.Zero
	}
	return slope.Div(atr)
}

// AsianSweep detects a liquidity sweep of the Asian session range.
// bull=true: Low pierced BELOW AsianLow and Close is back INSIDE the range.
// bull=false: High pierced ABOVE AsianHigh and Close is back INSIDE.
// Zero Asian levels (ORB warmup) → false.
func AsianSweep(high, low, close, asianHigh, asianLow decimal.Decimal, bull bool) bool {
	if asianHigh.IsZero() || asianLow.IsZero() {
		return false
	}
	if bull {
		return low.LessThan(asianLow) && close.GreaterThan(asianLow)
	}
	return high.GreaterThan(asianHigh) && close.LessThan(asianHigh)
}