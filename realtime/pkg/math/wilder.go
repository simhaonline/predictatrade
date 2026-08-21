// Package patmath — Wilder's smoothing functions for RSI, ATR, and ADX.
//
// Wilder's smoothing is a specific EMA with alpha = 1/period (not 2/(period+1)).
// First value is seeded with a simple average of the first `period` values,
// then subsequent values use the recursive formula:
//   smoothed_t = (smoothed_{t-1} * (period - 1) + value_t) / period
//
// This file implements the corrected indicator maths per prompt.md Section 1.
package patmath

import "github.com/shopspring/decimal"

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
	if len(highs) <= period || period <= 0 {
		return decimal.Zero
	}
	tr := TrueRangeSeries(highs, lows, closes)
	return wilderSmooth(tr, period)
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
	if len(closes) <= period || period <= 0 {
		return decimal.NewFromInt(50)
	}

	// Compute gains and losses
	gains := make([]decimal.Decimal, len(closes)-1)
	losses := make([]decimal.Decimal, len(closes)-1)
	for i := 1; i < len(closes); i++ {
		change := closes[i].Sub(closes[i-1])
		if change.GreaterThan(decimal.Zero) {
			gains[i-1] = change
			losses[i-1] = decimal.Zero
		} else {
			gains[i-1] = decimal.Zero
			losses[i-1] = change.Abs()
		}
	}

	// Seed: simple average of first `period` gains/losses
	avgGain := decimal.Zero
	avgLoss := decimal.Zero
	for i := 0; i < period; i++ {
		avgGain = avgGain.Add(gains[i])
		avgLoss = avgLoss.Add(losses[i])
	}
	avgGain = avgGain.Div(decimal.NewFromInt(int64(period)))
	avgLoss = avgLoss.Div(decimal.NewFromInt(int64(period)))

	// Wilder's recursive smoothing for remaining values
	periodMinus1 := decimal.NewFromInt(int64(period - 1))
	periodDec := decimal.NewFromInt(int64(period))
	for i := period; i < len(gains); i++ {
		avgGain = avgGain.Mul(periodMinus1).Add(gains[i]).Div(periodDec)
		avgLoss = avgLoss.Mul(periodMinus1).Add(losses[i]).Div(periodDec)
	}

	// RSI calculation
	if avgLoss.IsZero() {
		if avgGain.IsZero() {
			return decimal.NewFromInt(50) // Flat price — undefined
		}
		return decimal.NewFromInt(100) // No losses → RSI = 100
	}
	rs := avgGain.Div(avgLoss)
	rsi := decimal.NewFromInt(100).Sub(
		decimal.NewFromInt(100).Div(
			decimal.NewFromInt(1).Add(rs),
		),
	)
	return rsi
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
	// Need at least 2*period + 1 bars for meaningful ADX
	if n <= period*2 || period <= 0 {
		return decimal.Zero, decimal.Zero, decimal.Zero
	}

	// Compute +DM, -DM, and TR series (starting from bar 1)
	plusDMs := make([]decimal.Decimal, n-1)
	minusDMs := make([]decimal.Decimal, n-1)
	trSeries := make([]decimal.Decimal, n-1)

	for i := 1; i < n; i++ {
		upMove := highs[i].Sub(highs[i-1])
		downMove := lows[i-1].Sub(lows[i])

		if upMove.GreaterThan(downMove) && upMove.GreaterThan(decimal.Zero) {
			plusDMs[i-1] = upMove
		} else {
			plusDMs[i-1] = decimal.Zero
		}
		if downMove.GreaterThan(upMove) && downMove.GreaterThan(decimal.Zero) {
			minusDMs[i-1] = downMove
		} else {
			minusDMs[i-1] = decimal.Zero
		}

		trSeries[i-1] = TrueRange(highs[i], lows[i], closes[i-1])
	}

	// Wilder smoothing of +DM, -DM, TR
	smoothedPlusDM := wilderSmooth(plusDMs, period)
	smoothedMinusDM := wilderSmooth(minusDMs, period)
	smoothedTR := wilderSmooth(trSeries, period)

	if smoothedTR.IsZero() {
		return decimal.Zero, decimal.Zero, decimal.Zero
	}

	// +DI and -DI
	plusDI = smoothedPlusDM.Div(smoothedTR).Mul(decimal.NewFromInt(100))
	minusDI = smoothedMinusDM.Div(smoothedTR).Mul(decimal.NewFromInt(100))

	// DX
	diSum := plusDI.Add(minusDI)
	if diSum.IsZero() {
		return decimal.Zero, plusDI, minusDI
	}
	dx := plusDI.Sub(minusDI).Abs().Div(diSum).Mul(decimal.NewFromInt(100))

	// For a proper ADX, we need a series of DX values.
	// Compute DX for each bar using Wilder-smoothed components, then Wilder-smooth the DX series.
	dxSeries := make([]decimal.Decimal, len(trSeries)-period+1)
	dxIdx := 0
	for start := period - 1; start < len(trSeries); start++ {
		sPlusDM := wilderSmooth(plusDMs[:start+1], period)
		sMinusDM := wilderSmooth(minusDMs[:start+1], period)
		sTR := wilderSmooth(trSeries[:start+1], period)
		if sTR.IsZero() {
			dxSeries[dxIdx] = decimal.Zero
			dxIdx++
			continue
		}
		pDI := sPlusDM.Div(sTR).Mul(decimal.NewFromInt(100))
		mDI := sMinusDM.Div(sTR).Mul(decimal.NewFromInt(100))
		diS := pDI.Add(mDI)
		if diS.IsZero() {
			dxSeries[dxIdx] = decimal.Zero
		} else {
			dxSeries[dxIdx] = pDI.Sub(mDI).Abs().Div(diS).Mul(decimal.NewFromInt(100))
		}
		dxIdx++
	}

	if len(dxSeries) < period {
		return dx, plusDI, minusDI // Fallback to single DX if not enough DX values
	}
	adx = wilderSmooth(dxSeries, period)
	return adx, plusDI, minusDI
}
