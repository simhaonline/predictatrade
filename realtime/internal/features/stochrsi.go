package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// StochRSIEngine computes the Stochastic RSI indicator.
// SOW Section 7: Stochastic RSI implementation.
// Algorithm:
// 1. Compute RSI (Wilder's method, period 14)
// 2. Compute rolling min/max of RSI over stochPeriod (default 14)
// 3. StochRSI = (RSI - minRSI) / (maxRSI - minRSI)
// 4. Optional K smoothing (SMA of StochRSI, default 3)
// 5. Optional D smoothing (SMA of K, default 3)
// Handles: zero denominator, insufficient history, NaN, warm-up, incremental updates.
type StochRSIEngine struct {
	rsiPeriod    int
	stochPeriod  int
	kSmooth      int
	dSmooth      int

	closes       []decimal.Decimal
	rsiHistory   []decimal.Decimal
	avgGain      decimal.Decimal
	avgLoss      decimal.Decimal
	rsiReady     bool
	prevClose    decimal.Decimal

	stochHist    []decimal.Decimal // StochRSI raw values for K/D smoothing
}

// NewStochRSIEngine creates a new Stochastic RSI engine.
func NewStochRSIEngine(rsiPeriod, stochPeriod, kSmooth, dSmooth int) *StochRSIEngine {
	if rsiPeriod <= 0 {
		rsiPeriod = 14
	}
	if stochPeriod <= 0 {
		stochPeriod = 14
	}
	if kSmooth <= 0 {
		kSmooth = 3
	}
	if dSmooth <= 0 {
		dSmooth = 3
	}
	return &StochRSIEngine{
		rsiPeriod:   rsiPeriod,
		stochPeriod: stochPeriod,
		kSmooth:     kSmooth,
		dSmooth:     dSmooth,
	}
}

// StochRSIFeatures holds Stochastic RSI output.
type StochRSIFeatures struct {
	Raw      decimal.Decimal // Raw StochRSI (0-1 range, sometimes *100)
	K        decimal.Decimal // K smoothed line
	D        decimal.Decimal // D smoothed line
	Ready    bool            // True when fully warmed up
}

// Process updates the StochRSI engine with a new candle.
func (e *StochRSIEngine) Process(candle *types.Candle) StochRSIFeatures {
	if candle == nil {
		return StochRSIFeatures{Ready: e.rsiReady && len(e.rsiHistory) >= e.stochPeriod}
	}

	e.closes = append(e.closes, candle.Close)
	maxLen := e.rsiPeriod + e.stochPeriod + e.kSmooth + e.dSmooth + 10
	if len(e.closes) > maxLen {
		e.closes = e.closes[len(e.closes)-maxLen:]
	}

	// Step 1: Compute RSI using Wilder's smoothing
	rsi := e.computeRSI(candle.Close)

	if !rsi.IsZero() {
		e.rsiHistory = append(e.rsiHistory, rsi)
		if len(e.rsiHistory) > e.stochPeriod+e.kSmooth+e.dSmooth+5 {
			e.rsiHistory = e.rsiHistory[len(e.rsiHistory)-(e.stochPeriod+e.kSmooth+e.dSmooth+5):]
		}
	}

	e.prevClose = candle.Close

	// Check readiness: need rsiPeriod + stochPeriod RSI values minimum
	totalNeeded := e.rsiPeriod + e.stochPeriod
	ready := len(e.rsiHistory) >= e.stochPeriod && e.rsiReady
	_ = totalNeeded

	if !ready || len(e.rsiHistory) < e.stochPeriod {
		return StochRSIFeatures{Ready: false}
	}

	// Step 2: Rolling min/max of RSI over stochPeriod
	rsiWindow := e.rsiHistory[len(e.rsiHistory)-e.stochPeriod:]
	minRSI := minSlice(rsiWindow)
	maxRSI := maxSlice(rsiWindow)

	// Step 3: StochRSI = (RSI - minRSI) / (maxRSI - minRSI)
	// Handle zero denominator: if maxRSI == minRSI, StochRSI is undefined → return 0
	rangeVal := maxRSI.Sub(minRSI)
	var stochRaw decimal.Decimal
	if rangeVal.GreaterThan(decimal.Zero) {
		stochRaw = rsi.Sub(minRSI).Div(rangeVal)
	} else {
		// RSI is constant in the window — StochRSI undefined, return 0.5 (neutral)
		stochRaw = decimal.NewFromFloat(0.5)
	}

	// Store raw StochRSI for smoothing
	e.stochHist = append(e.stochHist, stochRaw)
	if len(e.stochHist) > e.kSmooth+e.dSmooth+5 {
		e.stochHist = e.stochHist[len(e.stochHist)-(e.kSmooth+e.dSmooth+5):]
	}

	// Step 4: K smoothing = SMA of StochRSI over kSmooth
	var kLine decimal.Decimal
	if len(e.stochHist) >= e.kSmooth {
		kLine = simpleMA(e.stochHist[len(e.stochHist)-e.kSmooth:], e.kSmooth)
	} else {
		kLine = stochRaw
	}

	// Step 5: D smoothing = SMA of K over dSmooth
	// We need K history — approximate using recent stochHist
	var dLine decimal.Decimal
	if len(e.stochHist) >= e.kSmooth+e.dSmooth-1 {
		// Compute K for last dSmooth bars and average
		kValues := make([]decimal.Decimal, 0, e.dSmooth)
		for i := 0; i < e.dSmooth; i++ {
			endIdx := len(e.stochHist) - i
			startIdx := endIdx - e.kSmooth
			if startIdx >= 0 && endIdx <= len(e.stochHist) {
				kVal := simpleMA(e.stochHist[startIdx:endIdx], e.kSmooth)
				kValues = append(kValues, kVal)
			}
		}
		if len(kValues) == e.dSmooth {
			sum := decimal.Zero
			for _, v := range kValues {
				sum = sum.Add(v)
			}
			dLine = sum.Div(decimal.NewFromInt(int64(e.dSmooth)))
		} else {
			dLine = kLine
		}
	} else {
		dLine = kLine
	}

	return StochRSIFeatures{
		Raw:   stochRaw,
		K:     kLine,
		D:     dLine,
		Ready: true,
	}
}

// computeRSI computes Wilder's RSI incrementally.
func (e *StochRSIEngine) computeRSI(close decimal.Decimal) decimal.Decimal {
	if len(e.closes) < 2 {
		return decimal.Zero
	}

	if !e.rsiReady {
		// Still in initial RSI calculation period
		if len(e.closes) <= e.rsiPeriod {
			return decimal.Zero
		}

		// First RSI: compute initial avgGain/avgLoss from first rsiPeriod changes
		if len(e.closes) == e.rsiPeriod+1 && e.avgGain.IsZero() {
			gainSum := decimal.Zero
			lossSum := decimal.Zero
			for i := 1; i <= e.rsiPeriod; i++ {
				change := e.closes[i].Sub(e.closes[i-1])
				if change.GreaterThan(decimal.Zero) {
					gainSum = gainSum.Add(change)
				} else {
					lossSum = lossSum.Add(change.Abs())
				}
			}
			e.avgGain = gainSum.Div(decimal.NewFromInt(int64(e.rsiPeriod)))
			e.avgLoss = lossSum.Div(decimal.NewFromInt(int64(e.rsiPeriod)))
			e.rsiReady = true
		} else {
			return decimal.Zero
		}
	} else {
		// Wilder's smoothing: avgGain = (prevAvgGain * (n-1) + currentGain) / n
		change := close.Sub(e.prevClose)
		var gain, loss decimal.Decimal
		if change.GreaterThan(decimal.Zero) {
			gain = change
		} else {
			loss = change.Abs()
		}
		n := decimal.NewFromInt(int64(e.rsiPeriod))
		e.avgGain = e.avgGain.Mul(n.Sub(decimal.NewFromInt(1))).Add(gain).Div(n)
		e.avgLoss = e.avgLoss.Mul(n.Sub(decimal.NewFromInt(1))).Add(loss).Div(n)
	}

	// RSI = 100 - 100/(1 + RS), where RS = avgGain/avgLoss
	if e.avgLoss.IsZero() {
		return decimal.NewFromInt(100)
	}
	rs := e.avgGain.Div(e.avgLoss)
	rsi := decimal.NewFromInt(100).Sub(
		decimal.NewFromInt(100).Div(decimal.NewFromInt(1).Add(rs)),
	)
	return rsi
}
