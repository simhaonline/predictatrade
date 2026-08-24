package crossmarket

import (
	"math"
	"time"
)

// DXYExhaustionScore calculates exhaustion evidence for DXY.
// Does NOT trigger trades solely from this metric.
type DXYExhaustionResult struct {
	Score       float64 // 0-100, higher = more exhausted
	Confidence  float64
	Evidence    []string
}

// CalculateDXYExhaustion analyzes DXY momentum deceleration, RSI divergence,
// failed breakout, and ADX decline patterns.
func CalculateDXYExhaustion(dxyValues, rsiValues []float64, adxValue float64) DXYExhaustionResult {
	n := len(dxyValues)
	if n < 10 {
		return DXYExhaustionResult{Confidence: 0}
	}

	score := 0.0
	var evidence []string

	// 1. Momentum deceleration: recent returns slowing
	recentRet := dxyValues[n-1] - dxyValues[n-5]
	olderRet := dxyValues[n-5] - dxyValues[n-10]
	if math.Abs(olderRet) > math.Abs(recentRet)*1.5 && math.Abs(olderRet) > 0.001 {
		score += 25
		evidence = append(evidence, "momentum_deceleration")
	}

	// 2. RSI divergence: price making new highs but RSI declining
	if len(rsiValues) >= 10 {
		priceHigher := dxyValues[n-1] > dxyValues[n-5]
		rsiLower := rsiValues[len(rsiValues)-1] < rsiValues[len(rsiValues)-5]
		if priceHigher && rsiLower {
			score += 30
			evidence = append(evidence, "rsi_bearish_divergence")
		}
	}

	// 3. ADX decline (falling trend strength)
	if adxValue > 0 && adxValue < 20 {
		score += 20
		evidence = append(evidence, "low_adx")
	}

	// 4. Rejection: last bar reversed significantly
	if n >= 3 {
		barRange := math.Abs(dxyValues[n-1] - dxyValues[n-2])
		reversal := dxyValues[n-1] - dxyValues[n-2]
		if barRange > 0 && math.Abs(reversal) < barRange*0.3 {
			score += 15
			evidence = append(evidence, "rejection_wick")
		}
	}

	score = math.Min(100, score)
	confidence := score / 100.0 * 0.7 // exhaustion is speculative

	return DXYExhaustionResult{
		Score:      score,
		Confidence: confidence,
		Evidence:   evidence,
	}
}

// BTCShockState classifies statistically unusual BTC movements.
type BTCShockState string

const (
	BTCShockNormal      BTCShockState = "NORMAL"
	BTCShockElevated    BTCShockState = "ELEVATED"
	BTCShockShock       BTCShockState = "SHOCK"
	BTCShockExtreme     BTCShockState = "EXTREME_SHOCK"
)

// BTCShockResult holds the result of BTC shock detection.
type BTCShockResult struct {
	State       BTCShockState
	Score       float64 // 0-100
	Direction   string  // UP, DOWN, NEUTRAL
	Confidence  float64
	ZScore      float64
	Reason      string
}

// DetectBTCShock uses rolling statistics (not hard-coded thresholds) to detect unusual BTC moves.
func DetectBTCShock(btcReturns []float64, lookback int) BTCShockResult {
	if lookback <= 0 {
		lookback = 50
	}
	n := len(btcReturns)
	if n < lookback {
		return BTCShockResult{State: BTCShockNormal, Confidence: 0}
	}

	// Use the last `lookback` returns for statistics
	sample := btcReturns[n-lookback:]

	// Calculate mean and std
	mean := 0.0
	for _, r := range sample {
		mean += r
	}
	mean /= float64(len(sample))

	variance := 0.0
	for _, r := range sample {
		variance += (r - mean) * (r - mean)
	}
	variance /= float64(len(sample))
	std := math.Sqrt(variance)

	if std == 0 {
		return BTCShockResult{State: BTCShockNormal, Confidence: 0}
	}

	// Latest return
	latestRet := btcReturns[n-1]
	zScore := (latestRet - mean) / std

	// ATR-normalized move (use std as proxy for ATR)
	atrNormMove := math.Abs(latestRet) / std

	// Classify using z-score (dynamic, not hard-coded percentage)
	var state BTCShockState
	score := 0.0
	direction := "NEUTRAL"
	reason := ""

	switch {
	case math.Abs(zScore) > 4:
		state = BTCShockExtreme
		score = 100
		reason = "extreme z-score shock"
	case math.Abs(zScore) > 3:
		state = BTCShockShock
		score = 75
		reason = "significant z-score shock"
	case math.Abs(zScore) > 2:
		state = BTCShockElevated
		score = 50
		reason = "elevated volatility"
	default:
		state = BTCShockNormal
		score = math.Min(25, atrNormMove*10)
		reason = "normal range"
	}

	if zScore > 0 {
		direction = "UP"
	} else if zScore < 0 {
		direction = "DOWN"
	}

	confidence := math.Min(1.0, math.Abs(zScore)/5.0)

	return BTCShockResult{
		State:      state,
		Score:      score,
		Direction:  direction,
		Confidence: confidence,
		ZScore:     zScore,
		Reason:     reason,
	}
}

// CircuitBreaker protects against corrupted cross-asset input.
type CircuitBreaker struct {
	lastPrice map[string]float64
	lastTime  map[string]time.Time
}

func NewCircuitBreaker() *CircuitBreaker {
	return &CircuitBreaker{
		lastPrice: make(map[string]float64),
		lastTime:  make(map[string]time.Time),
	}
}

// Validate checks an incoming observation for anomalies.
// Returns (valid, reason) — if invalid, the observation must be rejected.
func (cb *CircuitBreaker) Validate(symbol string, price float64, ts time.Time) (bool, string) {
	if price <= 0 {
		return false, "zero_or_negative_price"
	}
	if math.IsNaN(price) || math.IsInf(price, 0) {
		return false, "nan_or_inf_price"
	}

	if prev, ok := cb.lastPrice[symbol]; ok {
		// Check for impossible price jump (>50% in one tick)
		change := math.Abs(price-prev) / prev
		if change > 0.5 {
			return false, "impossible_price_jump"
		}

		// Check for frozen feed (same price for too long)
		if prev == price {
			if prevTime, ok := cb.lastTime[symbol]; ok {
				if time.Since(prevTime) > 10*time.Minute {
					return false, "frozen_feed"
				}
			}
		}

		// Check for timestamp inversion
		if prevTime, ok := cb.lastTime[symbol]; ok {
			if ts.Before(prevTime) {
				return false, "timestamp_inversion"
			}
		}
	}

	// Check for unreasonably large price (>1M or <0.01 for any asset)
	if price > 1000000 || price < 0.01 {
		return false, "price_out_of_sane_range"
	}

	cb.lastPrice[symbol] = price
	cb.lastTime[symbol] = ts
	return true, ""
}

// ReturnsCalculator computes returns from a price series.
type ReturnsCalculator struct{}

func NewReturnsCalculator() *ReturnsCalculator { return &ReturnsCalculator{} }

// LogReturns computes log returns from price series.
func (r *ReturnsCalculator) LogReturns(prices []float64) []float64 {
	if len(prices) < 2 {
		return nil
	}
	returns := make([]float64, len(prices)-1)
	for i := 0; i < len(prices)-1; i++ {
		if prices[i] <= 0 || prices[i+1] <= 0 {
			returns[i] = 0
			continue
		}
		returns[i] = math.Log(prices[i+1] / prices[i])
	}
	return returns
}

// SimpleReturns computes simple returns from price series.
func (r *ReturnsCalculator) SimpleReturns(prices []float64) []float64 {
	if len(prices) < 2 {
		return nil
	}
	returns := make([]float64, len(prices)-1)
	for i := 0; i < len(prices)-1; i++ {
		if prices[i] == 0 {
			returns[i] = 0
			continue
		}
		returns[i] = (prices[i+1] - prices[i]) / prices[i]
	}
	return returns
}

// ZScore calculates the z-score of the latest value relative to a window.
func ZScore(values []float64, window int) float64 {
	if window <= 0 || len(values) < window {
		return 0
	}
	sample := values[len(values)-window:]
	mean := 0.0
	for _, v := range sample {
		mean += v
	}
	mean /= float64(len(sample))

	variance := 0.0
	for _, v := range sample {
		variance += (v - mean) * (v - mean)
	}
	variance /= float64(len(sample))
	std := math.Sqrt(variance)

	if std == 0 {
		return 0
	}
	return (values[len(values)-1] - mean) / std
}

// RollingCorrelation computes rolling Pearson correlation.
func RollingCorrelation(x, y []float64, window int) []float64 {
	if window < 5 || len(x) < window || len(y) < window {
		return nil
	}
	n := len(x)
	if len(y) < n {
		n = len(y)
	}

	result := make([]float64, 0, n-window+1)
	for i := window; i <= n; i++ {
		corr := pearsonCorrelation(x[i-window:i], y[i-window:i])
		if math.IsNaN(corr) {
			corr = 0
		}
		result = append(result, corr)
	}
	return result
}

// CorrelationStability measures how stable a correlation series is.
func CorrelationStability(correlations []float64) float64 {
	if len(correlations) < 5 {
		return 0
	}
	// Stability = 1 - (std of correlations / 1.0)
	mean := 0.0
	for _, c := range correlations {
		mean += c
	}
	mean /= float64(len(correlations))

	variance := 0.0
	for _, c := range correlations {
		variance += (c - mean) * (c - mean)
	}
	variance /= float64(len(correlations))
	std := math.Sqrt(variance)

	stability := 1.0 - std
	return math.Max(0, math.Min(1, stability))
}

// CrossAssetMomentum calculates momentum metrics for a macro asset.
type CrossAssetMomentum struct {
	Return1   float64
	Return5   float64
	Return15  float64
	Return30  float64
	ROC       float64
	ATRNorm   float64
	ZScore    float64
	VolPct    float64
	Direction Direction
}

// CalculateMomentum computes momentum from a price series.
func CalculateMomentum(prices []float64) CrossAssetMomentum {
	momentum := CrossAssetMomentum{}
	n := len(prices)
	if n < 2 {
		return momentum
	}

	if n >= 2 {
		momentum.Return1 = (prices[n-1] - prices[n-2]) / prices[n-2]
	}
	if n >= 6 {
		momentum.Return5 = (prices[n-1] - prices[n-6]) / prices[n-6]
	}
	if n >= 16 {
		momentum.Return15 = (prices[n-1] - prices[n-16]) / prices[n-16]
	}
	if n >= 31 {
		momentum.Return30 = (prices[n-1] - prices[n-31]) / prices[n-31]
	}

	momentum.ROC = momentum.Return5
	momentum.ZScore = ZScore(prices, 50)

	if n >= 15 {
		// ATR proxy: average absolute return
		returns := make([]float64, 0, 14)
		for i := n - 15; i < n-1; i++ {
			if prices[i] != 0 {
				returns = append(returns, math.Abs((prices[i+1]-prices[i])/prices[i]))
			}
		}
		if len(returns) > 0 {
			avgATR := 0.0
			for _, r := range returns {
				avgATR += r
			}
			avgATR /= float64(len(returns))
			if avgATR > 0 {
				momentum.ATRNorm = math.Abs(momentum.Return1) / avgATR
			}
		}
	}

	if momentum.Return5 > 0 {
		momentum.Direction = DirBullish
	} else if momentum.Return5 < 0 {
		momentum.Direction = DirBearish
	}

	return momentum
}
