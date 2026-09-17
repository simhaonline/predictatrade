// indicators_volatility.go — P1 (prompt.md): volatility-state indicators.
//
// New, non-overlapping additions (existing verified features untouched):
//   - ChoppinessIndex: Dreissax-style range compression measure [0,100]
//   - SqueezeState: Bollinger-inside-Keltner squeeze state + release detection
//   - ADXSlope: ADX momentum (delta across bars)
//
// Formulas (documented per prompt.md requirements):
//
//	Chop(n) = 100 * Log10( Σ TR_i / (MaxHigh(n) − MinLow(n)) ) / Log10(n)
//	  TR_i  = max(H−L, |H−prevC|, |L−prevC|); the FIRST bar of the window uses
//	          H−L when no prior close exists (standard warmup fallback).
//	  Warmup: fewer than n bars → zero (omitted).
//
//	SqueezeOn      = BBUpper < KCUpper AND BBLower > KCLower
//	SqueezeRelease = prevOn && !On   (direction taken by caller from close vs BB middle)
//
//	ADXSlope = ADX_now − ADX_prev   (plain delta; sign = momentum of trend strength)
package features

import (
	math "math"

	"github.com/shopspring/decimal"
)

var (
	decTen  = decimal.NewFromInt(10)
	decHund = decimal.NewFromInt(100)
)

// ChoppinessIndex computes the Choppiness Index over the last n bars.
// highs/lows/closes must be equal-length ascending-time series; n ≤ len-1.
// Returns zero during warmup (insufficient bars) — callers treat zero as
// "not ready" (the feature-snapshot builder omits zero values).
func ChoppinessIndex(highs, lows, closes []decimal.Decimal, n int) decimal.Decimal {
	if n <= 1 || len(highs) != len(lows) || len(lows) != len(closes) || len(highs) < n {
		return decimal.Zero
	}

	// Window = last n bars. TR of the first window bar uses H−L when no prior
	// close exists (len == n); otherwise it uses the standard prev-close form.
	start := len(highs) - n
	havePrev := start > 0

	maxHi, minLo := highs[start], lows[start]
	for i := start + 1; i < len(highs); i++ {
		if highs[i].GreaterThan(maxHi) {
			maxHi = highs[i]
		}
		if lows[i].LessThan(minLo) {
			minLo = lows[i]
		}
	}
	rng := maxHi.Sub(minLo)
	if rng.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	sumTR := decimal.Zero
	if !havePrev {
		sumTR = sumTR.Add(highs[start].Sub(lows[start])) // TR[0] = H−L fallback
	}
	for i := start + 1; i < len(highs); i++ {
		prevC := closes[i-1]
		tr := highs[i].Sub(lows[i])
		if d := highs[i].Sub(prevC).Abs(); d.GreaterThan(tr) {
			tr = d
		}
		if d := lows[i].Sub(prevC).Abs(); d.GreaterThan(tr) {
			tr = d
		}
		sumTR = sumTR.Add(tr)
	}
	if sumTR.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero
	}

	// 100 * Log10(sumTR/rng) / Log10(n)
	ratio := sumTR.Div(rng)
	if ratio.LessThanOrEqual(decimal.NewFromInt(1)) {
		return decimal.Zero // perfectly trending: log10(≤1) ≤ 0 → chop floor
	}
	num := log10Decimal(ratio)
	den := log10Decimal(decimal.NewFromInt(int64(n)))
	if den.IsZero() {
		return decimal.Zero
	}
	return num.Div(den).Mul(decHund)
}

// log10Decimal computes base-10 log via math.Log10 with decimal round-trip.
func log10Decimal(d decimal.Decimal) decimal.Decimal {
	f, _ := d.Float64()
	if f <= 0 {
		return decimal.Zero
	}
	return decimal.NewFromFloat(math.Log10(f))
}

// decsFromFloats converts a float64 window to decimals (one alloc per call;
// called per closed bar — the windows are bounded by lookback).
func decsFromFloats(fs []float64) []decimal.Decimal {
	out := make([]decimal.Decimal, len(fs))
	for i, f := range fs {
		out[i] = decimal.NewFromFloat(f)
	}
	return out
}

// SqueezeStateResult carries the squeeze family state.
type SqueezeStateResult struct {
	On      bool // BB fully inside KC
	Release bool // transitioned from On→Off this bar
}

// SqueezeState evaluates the BB-inside-KC squeeze.
// bbUpper/bbLower: Bollinger bands; kcUpper/kcLower: Keltner channels.
// prevOn: squeeze state of the previous bar. Zero-valued bands (warmup)
// produce On=false, Release=false.
func SqueezeState(bbUpper, bbLower, kcUpper, kcLower decimal.Decimal, prevOn bool) SqueezeStateResult {
	if bbUpper.IsZero() || bbLower.IsZero() || kcUpper.IsZero() || kcLower.IsZero() {
		return SqueezeStateResult{}
	}
	on := bbUpper.LessThan(kcUpper) && bbLower.GreaterThan(kcLower)
	return SqueezeStateResult{On: on, Release: prevOn && !on}
}

// ADXSlope returns the ADX delta (current − previous). Warmup (either value
// zero) returns zero — the trend-strength momentum is not yet measurable.
func ADXSlope(current, previous decimal.Decimal) decimal.Decimal {
	if current.IsZero() || previous.IsZero() {
		return decimal.Zero
	}
	return current.Sub(previous)
}