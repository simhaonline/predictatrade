// Package patmath — Wilder's smoothing functions for RSI, ATR, and ADX.
//
// Wilder's smoothing is a specific EMA with alpha = 1/period (not 2/(period+1)).
// First value is seeded with a simple average of the first `period` values,
// then subsequent values use the recursive formula:
//   smoothed_t = (smoothed_{t-1} * (period - 1) + value_t) / period
//
// This file implements the corrected indicator maths per prompt.md Section 1.
package patmath

import (
	"math"
	"github.com/shopspring/decimal"
)

// wilderSmooth applies Wilder's smoothing to a series of values.
// The first `period` values are averaged to seed the initial smoothed value.
// Subsequent values use the recursive Wilder formula:
//   smoothed_t = (smoothed_{t-1} * (period - 1) + value_t) / period
//
// Returns the final smoothed value, or zero if insufficient data.
func wilderSmooth(values []decimal.Decimal, period int) decimal.Decimal {
	if len(values) < period || period <= 0 {
		return decimal.Zero
	}

	// Seed: simple average of first `period` values
	sum := decimal.Zero
	for i := 0; i < period; i++ {
		sum = sum.Add(values[i])
	}
	smoothed := sum.Div(decimal.NewFromInt(int64(period)))

	// Recursive Wilder smoothing for remaining values
	periodMinus1 := decimal.NewFromInt(int64(period - 1))
	periodDec := decimal.NewFromInt(int64(period))
	for i := period; i < len(values); i++ {
		smoothed = smoothed.Mul(periodMinus1).Add(values[i]).Div(periodDec)
	}
	return smoothed
}

// TrueRangeSeries computes the True Range for each bar in the series.
// TR_t = max(H_t - L_t, |H_t - C_{t-1}|, |L_t - C_{t-1}|)
// The first bar has no previous close, so TR_0 = H_0 - L_0.
func TrueRangeSeries(highs, lows, closes []decimal.Decimal) []decimal.Decimal {
	n := len(highs)
	if len(lows) < n {
		n = len(lows)
	}
	if len(closes) < n {
		n = len(closes)
	}
	if n == 0 {
		return nil
	}
	tr := make([]decimal.Decimal, n)
	tr[0] = highs[0].Sub(lows[0]).Abs()
	for i := 1; i < n; i++ {
		tr[i] = TrueRange(highs[i], lows[i], closes[i-1])
	}
	return tr
}

// ATRWilder computes ATR using Wilder's smoothing (NOT simple average).
// First ATR = mean(TR, period)
// Subsequent: ATR_t = (ATR_{t-1} * (period - 1) + TR_t) / period
//
// This replaces the old simple-average ATR implementation.
func ATRWilder(highs, lows, closes []decimal.Decimal, period int) decimal.Decimal {
	n := len(highs)
	if len(lows) < n {
		n = len(lows)
	}
	if len(closes) < n {
		n = len(closes)
	}
	if n <= period || period <= 0 {
		return decimal.Zero
	}
	// Use float64 to prevent precision explosion
	trs := make([]float64, n-1)
	for i := 1; i < n; i++ {
		h, _ := highs[i].Float64()
		l, _ := lows[i].Float64()
		pc, _ := closes[i-1].Float64()
		hl := math.Abs(h - l)
		hc := math.Abs(h - pc)
		lc := math.Abs(l - pc)
		tr := hl
		if hc > tr { tr = hc }
		if lc > tr { tr = lc }
		trs[i-1] = tr
	}
	// Wilder smoothing: first avg = SMA, then avg = (prev*(n-1) + TR) / n
	atr := 0.0
	for i := 0; i < period && i < len(trs); i++ {
		atr += trs[i]
	}
	atr /= float64(period)
	for i := period; i < len(trs); i++ {
		atr = (atr*float64(period-1) + trs[i]) / float64(period)
	}
	return decimal.NewFromFloat(atr)
}

// RSIWilder computes RSI using Wilder's smoothing method.
//
// delta = P_t - P_{t-1}
// gain = max(delta, 0), loss = max(-delta, 0)
// First avg_gain = mean(gain, 14), first avg_loss = mean(loss, 14)
// Subsequent: avg_gain_t = (avg_gain_{t-1} * 13 + gain_t) / 14
//             avg_loss_t = (avg_loss_{t-1} * 13 + loss_t) / 14
// RS = avg_gain / avg_loss
// RSI = 100 - (100 / (1 + RS))
//
// If avg_loss is zero, RSI = 100. If both are zero, RSI = 50 (undefined).
func RSIWilder(closes []decimal.Decimal, period int) decimal.Decimal {
	n := len(closes)
	if n <= period || period <= 0 {
		return decimal.Zero
	}
	// Use float64 to prevent precision explosion
	changes := make([]float64, n-1)
	for i := 1; i < n; i++ {
		c, _ := closes[i].Float64()
		pc, _ := closes[i-1].Float64()
		changes[i-1] = c - pc
	}
	var avgGain, avgLoss float64
	for i := 0; i < period; i++ {
		if changes[i] > 0 {
			avgGain += changes[i]
		} else {
			avgLoss += -changes[i]
		}
	}
	avgGain /= float64(period)
	avgLoss /= float64(period)
	for i := period; i < len(changes); i++ {
		gain := 0.0
		loss := 0.0
		if changes[i] > 0 {
			gain = changes[i]
		} else {
			loss = -changes[i]
		}
		avgGain = (avgGain*float64(period-1) + gain) / float64(period)
		avgLoss = (avgLoss*float64(period-1) + loss) / float64(period)
	}
	if avgLoss == 0 && avgGain == 0 {
		return decimal.NewFromInt(50)// flat price — undefined
	}
	if avgLoss == 0 {
		return decimal.NewFromInt(100)
	}
	rs := avgGain / avgLoss
	rsi := 100.0 - 100.0/(1.0+rs)
	return decimal.NewFromFloat(rsi)
}

// ADXWilder computes ADX using full Wilder's smoothing for TR, +DM, -DM, and DX.
//
// +DM_t = H_t - H_{t-1}  if > 0 and > -(L_t - L_{t-1}), else 0
// -DM_t = L_{t-1} - L_t  if > 0 and >  (H_t - H_{t-1}), else 0
//
// Smoothed +DM = Wilder(+DM, period)
// Smoothed -DM = Wilder(-DM, period)
// Smoothed TR  = Wilder(TR, period)
//
// +DI = 100 * smoothed +DM / smoothed TR
// -DI = 100 * smoothed -DM / smoothed TR
// DX = 100 * |+DI - -DI| / (+DI + -DI)
// ADX = Wilder(DX, period)
//
// Returns (adx, plusDI, minusDI). ADX is a trend filter, never an entry signal alone.
func ADXWilder(highs, lows, closes []decimal.Decimal, period int) (adx, plusDI, minusDI decimal.Decimal) {
	n := len(highs)
	if len(lows) < n {
		n = len(lows)
	}
	if len(closes) < n {
		n = len(closes)
	}
	if n <= period*2 || period <= 0 {
		return decimal.Zero, decimal.Zero, decimal.Zero
	}
	// Use float64 to prevent precision explosion
	plusDMs := make([]float64, n-1)
	minusDMs := make([]float64, n-1)
	trs := make([]float64, n-1)
	for i := 1; i < n; i++ {
		h, _ := highs[i].Float64()
		l, _ := lows[i].Float64()
		ph, _ := highs[i-1].Float64()
		pl, _ := lows[i-1].Float64()
		pc, _ := closes[i-1].Float64()
		upMove := h - ph
		downMove := pl - l
		var plusDM, minusDM float64
		if upMove > downMove && upMove > 0 {
			plusDM = upMove
		}
		if downMove > upMove && downMove > 0 {
			minusDM = downMove
		}
		hl := math.Abs(h - l)
		hc := math.Abs(h - pc)
		lc := math.Abs(l - pc)
		tr := hl
		if hc > tr { tr = hc }
		if lc > tr { tr = lc }
		plusDMs[i-1] = plusDM
		minusDMs[i-1] = minusDM
		trs[i-1] = tr
	}
	// Wilder smoothing
	avgTR := 0.0
	avgPlusDM := 0.0
	avgMinusDM := 0.0
	for i := 0; i < period; i++ {
		avgTR += trs[i]
		avgPlusDM += plusDMs[i]
		avgMinusDM += minusDMs[i]
	}
	avgTR /= float64(period)
	avgPlusDM /= float64(period)
	avgMinusDM /= float64(period)
	dxValues := make([]float64, 0, n-period)
	for i := period; i < len(trs); i++ {
		avgTR = (avgTR*float64(period-1) + trs[i]) / float64(period)
		avgPlusDM = (avgPlusDM*float64(period-1) + plusDMs[i]) / float64(period)
		avgMinusDM = (avgMinusDM*float64(period-1) + minusDMs[i]) / float64(period)
		var plusDI, minusDI float64
		if avgTR > 0 {
			plusDI = 100.0 * avgPlusDM / avgTR
			minusDI = 100.0 * avgMinusDM / avgTR
		}
		denom := plusDI + minusDI
		var dx float64
		if denom > 0 {
			dx = 100.0 * math.Abs(plusDI-minusDI) / denom
		}
		dxValues = append(dxValues, dx)
		// Store latest DI values
		if i == len(trs)-1 {
			plusDIF := plusDI
			minusDIF := minusDI
			_ = plusDIF
			_ = minusDIF
		}
	}
	// ADX = Wilder smoothed average of DX
	if len(dxValues) == 0 {
		return decimal.Zero, decimal.Zero, decimal.Zero
	}
	adxF := 0.0
	for i := 0; i < period && i < len(dxValues); i++ {
		adxF += dxValues[i]
	}
	adxF /= float64(period)
	for i := period; i < len(dxValues); i++ {
		adxF = (adxF*float64(period-1) + dxValues[i]) / float64(period)
	}
	// Get latest DI values
	var lastPlusDI, lastMinusDI float64
	if avgTR > 0 {
		lastPlusDI = 100.0 * avgPlusDM / avgTR
		lastMinusDI = 100.0 * avgMinusDM / avgTR
	}
	return decimal.NewFromFloat(adxF), decimal.NewFromFloat(lastPlusDI), decimal.NewFromFloat(lastMinusDI)
}
