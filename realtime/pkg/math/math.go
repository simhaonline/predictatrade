// Package math provides canonical quantitative math functions for the real-time engine.
// SOW Sections 134, 136 — production equivalents of the Python reference_math.py
package patmath

import (
	"math"

	"github.com/shopspring/decimal"
)

// GrossRR computes the gross risk-reward ratio for a target.
// SOW Section 134.1: gross_RR_TPi = abs(TPi - Entry) / abs(Entry - SL)
func GrossRR(entry, stopLoss, takeProfit decimal.Decimal) decimal.Decimal {
	targetDistance := takeProfit.Sub(entry).Abs()
	stopDistance := entry.Sub(stopLoss).Abs()
	if stopDistance.IsZero() {
		return decimal.Zero
	}
	return targetDistance.Div(stopDistance)
}

// NetRR computes the net (cost-adjusted) risk-reward ratio.
// SOW Section 134.1: net_RR = (target_dist - cost) / (stop_dist + cost)
func NetRR(entry, stopLoss, takeProfit, roundTripCost decimal.Decimal) decimal.Decimal {
	targetDistance := takeProfit.Sub(entry).Abs()
	stopDistance := entry.Sub(stopLoss).Abs()
	numerator := targetDistance.Sub(roundTripCost)
	denominator := stopDistance.Add(roundTripCost)
	if denominator.IsZero() {
		return decimal.Zero
	}
	return numerator.Div(denominator)
}

// Expectancy computes the expected R value.
// SOW Section 134.2: E_R = P(win) × AvgWinR - P(loss) × AvgLossR
func Expectancy(pWin, avgWinR, avgLossR decimal.Decimal) decimal.Decimal {
	pLoss := decimal.NewFromInt(1).Sub(pWin)
	return pWin.Mul(avgWinR).Sub(pLoss.Mul(avgLossR))
}

// ProfitFactor computes the profit factor.
// SOW Section 134.2
func ProfitFactor(grossProfit, grossLoss decimal.Decimal) decimal.Decimal {
	if grossLoss.IsZero() {
		return decimal.Zero
	}
	return grossProfit.Div(grossLoss.Abs())
}

// CostToTarget computes the cost-to-target ratio.
// SOW Section 134.11: cost_to_target = total_expected_round_trip_cost / abs(TP1 - Entry)
func CostToTarget(entry, tp1, roundTripCost decimal.Decimal) decimal.Decimal {
	targetDistance := tp1.Sub(entry).Abs()
	if targetDistance.IsZero() {
		return decimal.Zero
	}
	return roundTripCost.Div(targetDistance)
}

// WilsonLower computes the Wilson score interval lower bound for a binomial proportion.
// SOW Section 134.7
func WilsonLower(successes, total int64, zScore float64) float64 {
	if total == 0 {
		return 0
	}
	n := float64(total)
	pHat := float64(successes) / n
	z2 := zScore * zScore
	denominator := 1 + z2/n
	numerator := pHat + z2/(2*n) - zScore*math.Sqrt(pHat*(1-pHat)/n+z2/(4*n*n))
	return numerator / denominator
}

// WilsonUpper computes the Wilson score interval upper bound.
func WilsonUpper(successes, total int64, zScore float64) float64 {
	if total == 0 {
		return 1
	}
	n := float64(total)
	pHat := float64(successes) / n
	z2 := zScore * zScore
	denominator := 1 + z2/n
	numerator := pHat + z2/(2*n) + zScore*math.Sqrt(pHat*(1-pHat)/n+z2/(4*n*n))
	return numerator / denominator
}

// BrierScore computes the Brier score for calibration evaluation.
// SOW Section 134.6: Brier = mean((p - outcome)^2)
func BrierScore(probabilities []float64, outcomes []bool) float64 {
	if len(probabilities) != len(outcomes) || len(probabilities) == 0 {
		return 0
	}
	sum := 0.0
	for i, p := range probabilities {
		outcome := 0.0
		if outcomes[i] {
			outcome = 1.0
		}
		diff := p - outcome
		sum += diff * diff
	}
	return sum / float64(len(probabilities))
}

// ECE computes the Expected Calibration Error.
// SOW Section 134.6: ECE = Σ_bin (n_bin/N) × |observed_freq_bin - mean_forecast_bin|
func ECE(binCounts []int, binMeanForecasts []float64, binObservedFreqs []float64) float64 {
	total := 0
	for _, c := range binCounts {
		total += c
	}
	if total == 0 {
		return 0
	}
	sum := 0.0
	for i, count := range binCounts {
		weight := float64(count) / float64(total)
		diff := math.Abs(binObservedFreqs[i] - binMeanForecasts[i])
		sum += weight * diff
	}
	return sum
}

// MTFAlignmentScore computes the multi-timeframe alignment score.
// SOW Section 134.10: mtf_alignment_score = 100 × Σ(weight_tf × state_tf) / Σ(weight_tf)
// state_tf ∈ {-1, 0, +1}
func MTFAlignmentScore(weights []float64, states []int) float64 {
	if len(weights) != len(states) || len(weights) == 0 {
		return 0
	}
	weightSum := 0.0
	weightedState := 0.0
	for i, w := range weights {
		weightSum += w
		weightedState += w * float64(states[i])
	}
	if weightSum == 0 {
		return 0
	}
	return 100.0 * weightedState / weightSum
}

// ATR computes the Average True Range.
// True Range = max(H-L, |H-Cprev|, |L-Cprev|)
func ATR(highs, lows, closes []decimal.Decimal, period int) decimal.Decimal {
	if len(highs) < period || period <= 0 {
		return decimal.Zero
	}
	trSum := decimal.Zero
	for i := 1; i <= period && i < len(highs); i++ {
		tr := TrueRange(highs[i], lows[i], closes[i-1])
		trSum = trSum.Add(tr)
	}
	return trSum.Div(decimal.NewFromInt(int64(period)))
}

// TrueRange computes the true range for a single bar.
func TrueRange(high, low, prevClose decimal.Decimal) decimal.Decimal {
	hl := high.Sub(low).Abs()
	hc := high.Sub(prevClose).Abs()
	lc := low.Sub(prevClose).Abs()
	max := hl
	if hc.GreaterThan(max) {
		max = hc
	}
	if lc.GreaterThan(max) {
		max = lc
	}
	return max
}

// EMA computes the Exponential Moving Average.
func EMA(values []decimal.Decimal, period int) decimal.Decimal {
	if len(values) == 0 || period <= 0 {
		return decimal.Zero
	}
	multiplier := decimal.NewFromInt(2).Div(decimal.NewFromInt(int64(period + 1)))
	ema := values[0]
	for i := 1; i < len(values); i++ {
		ema = values[i].Mul(multiplier).Add(ema.Mul(decimal.NewFromInt(1).Sub(multiplier)))
	}
	return ema
}

// RSI computes the Relative Strength Index.
func RSI(closes []decimal.Decimal, period int) decimal.Decimal {
	if len(closes) <= period || period <= 0 {
		return decimal.NewFromInt(50)
	}
	avgGain := decimal.Zero
	avgLoss := decimal.Zero
	for i := 1; i <= period; i++ {
		change := closes[i].Sub(closes[i-1])
		if change.GreaterThan(decimal.Zero) {
			avgGain = avgGain.Add(change)
		} else {
			avgLoss = avgLoss.Add(change.Abs())
		}
	}
	avgGain = avgGain.Div(decimal.NewFromInt(int64(period)))
	avgLoss = avgLoss.Div(decimal.NewFromInt(int64(period)))
	if avgLoss.IsZero() {
		return decimal.NewFromInt(100)
	}
	rs := avgGain.Div(avgLoss)
	rsi := decimal.NewFromInt(100).Sub(decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)))
	return rsi
}

// ADX computes the Average Directional Index (simplified).
func ADX(highs, lows, closes []decimal.Decimal, period int) decimal.Decimal {
	if len(highs) <= period*2 || period <= 0 {
		return decimal.Zero
	}
	// Simplified ADX: average of |DX| values
	dxSum := decimal.Zero
	count := 0
	for i := period; i < len(highs)-1; i++ {
		plusDM := highs[i].Sub(highs[i-1])
		minusDM := lows[i-1].Sub(lows[i])
		if plusDM.LessThan(decimal.Zero) {
			plusDM = decimal.Zero
		}
		if minusDM.LessThan(decimal.Zero) {
			minusDM = decimal.Zero
		}
		if plusDM.GreaterThan(minusDM) {
			minusDM = decimal.Zero
		} else {
			plusDM = decimal.Zero
		}
		tr := TrueRange(highs[i], lows[i], closes[i-1])
		if tr.IsZero() {
			continue
		}
		plusDI := plusDM.Div(tr).Mul(decimal.NewFromInt(100))
		minusDI := minusDM.Div(tr).Mul(decimal.NewFromInt(100))
		diSum := plusDI.Add(minusDI)
		if diSum.IsZero() {
			continue
		}
		dx := plusDI.Sub(minusDI).Abs().Div(diSum).Mul(decimal.NewFromInt(100))
		dxSum = dxSum.Add(dx)
		count++
	}
	if count == 0 {
		return decimal.Zero
	}
	return dxSum.Div(decimal.NewFromInt(int64(count)))
}
