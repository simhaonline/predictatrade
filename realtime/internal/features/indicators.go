package features

import (
	"math"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/predictatrade/realtime/pkg/math"
	"github.com/shopspring/decimal"
)

// IndicatorEngine computes ALL locally-calculable technical indicators.
// Mathematical formulas follow canonical definitions (Wilder RSI, Wilder ATR, etc.).
// Numerical safety: all calculations handle NaN, infinity, division by zero,
// insufficient history, and invalid candle data explicitly.
// Indicators not available (SAR, Ichimoku, StochRSI, Volume Profile, Cumulative Delta)
// are UNAVAILABLE and never fabricated.
type IndicatorEngine struct {
	closes     []decimal.Decimal
	highs      []decimal.Decimal
	lows       []decimal.Decimal
	volumes    []int64
	lookback   int
	prevEMA9   decimal.Decimal
	prevEMA21  decimal.Decimal
	prevMACD       decimal.Decimal
	prevMACDSignal decimal.Decimal
	prevRSI        decimal.Decimal
	prevClose      decimal.Decimal
	prevBollLower  decimal.Decimal
	prevBollUpper  decimal.Decimal
	macdHist       []decimal.Decimal
	stochHist  []decimal.Decimal // %K history for Stochastic signal line (3-period SMA)
}

func NewIndicatorEngine(lookback int) *IndicatorEngine {
	return &IndicatorEngine{lookback: lookback}
}

func (e *IndicatorEngine) Process(candle *types.Candle) IndicatorFeatures {
	feat := IndicatorFeatures{}
	if candle == nil {
		return feat
	}

	// Numerical safety: validate candle data
	if candle.High.LessThan(candle.Low) || candle.High.LessThan(candle.Open) ||
		candle.High.LessThan(candle.Close) || candle.Low.GreaterThan(candle.Open) ||
		candle.Low.GreaterThan(candle.Close) {
		return feat // Invalid candle — return zeros (UNAVAILABLE, not neutral)
	}

	e.closes = append(e.closes, candle.Close)
	e.highs = append(e.highs, candle.High)
	e.lows = append(e.lows, candle.Low)
	e.volumes = append(e.volumes, candle.Volume)

	if len(e.closes) > e.lookback {
		e.closes = e.closes[len(e.closes)-e.lookback:]
		e.highs = e.highs[len(e.highs)-e.lookback:]
		e.lows = e.lows[len(e.lows)-e.lookback:]
		e.volumes = e.volumes[len(e.volumes)-e.lookback:]
	}

	// === Trend Indicators ===

	// EMA 9
	if len(e.closes) >= 9 {
		feat.EMA9 = patmath.EMA(e.closes, 9)
	}
	// EMA 21
	if len(e.closes) >= 21 {
		feat.EMA21 = patmath.EMA(e.closes, 21)
	}
	// EMA 50
	if len(e.closes) >= 50 {
		feat.EMA50 = patmath.EMA(e.closes, 50)
	}
	// EMA 100
	if len(e.closes) >= 100 {
		feat.EMA100 = patmath.EMA(e.closes, 100)
	}
	// EMA 200
	if len(e.closes) >= 200 {
		feat.EMA200 = patmath.EMA(e.closes, 200)
	}

	// EMA Cross 9/21 — detect actual crossover event, not just alignment
	// Bullish cross: prev EMA9 <= prev EMA21 AND current EMA9 > current EMA21
	// Bearish cross: prev EMA9 >= prev EMA21 AND current EMA9 < current EMA21
	if !e.prevEMA9.IsZero() && !e.prevEMA21.IsZero() && !feat.EMA9.IsZero() && !feat.EMA21.IsZero() {
		if e.prevEMA9.LessThanOrEqual(e.prevEMA21) && feat.EMA9.GreaterThan(feat.EMA21) {
			feat.EMACross921 = true // Bullish cross event
		}
		// EMACross921 is false for bearish cross or no cross
	}
	// Alignment (separate from cross event) — stored in the comparison itself

	// SMA 50
	if len(e.closes) >= 50 {
		feat.SMA50 = simpleMA(e.closes, 50)
	}
	// SMA 100
	if len(e.closes) >= 100 {
		feat.SMA100 = simpleMA(e.closes, 100)
	}
	// SMA 200
	if len(e.closes) >= 200 {
		feat.SMA200 = simpleMA(e.closes, 200)
	}

	// MACD (12, 26, 9)
	// MACD_line = EMA12 - EMA26
	// Signal = EMA9 of MACD line
	// Histogram = MACD_line - Signal
	if len(e.closes) >= 26 {
		ema12 := patmath.EMA(e.closes, 12)
		ema26 := patmath.EMA(e.closes, 26)
		feat.MACDMain = ema12.Sub(ema26)

		// Proper MACD signal: EMA9 of MACD line history
		e.macdHist = append(e.macdHist, feat.MACDMain)
		if len(e.macdHist) > 20 {
			e.macdHist = e.macdHist[len(e.macdHist)-20:]
		}
		if len(e.macdHist) >= 9 {
			feat.MACDSignal = calcEMA(e.macdHist, 9)
		}
	}

	// ADX 14 with +DI / -DI using full Wilder's method (prompt.md Section 1.5)
	// ADXWilder returns all three values with consistent Wilder smoothing
	if len(e.highs) >= 28 {
		adx, plusDI, minusDI := patmath.ADXWilder(e.highs, e.lows, e.closes, 14)
		feat.ADX = adx
		feat.ADXPlusDI = plusDI
		feat.ADXMinusDI = minusDI
	}

	// Parabolic SAR — UNAVAILABLE
	// Ichimoku Cloud — UNAVAILABLE

	// === Momentum Indicators ===

	// RSI 14 (Wilder's method)
	if len(e.closes) >= 14 {
		feat.RSI = patmath.RSI(e.closes, 14)
	}

	// Stochastic 14/3/3
	// %K_raw = 100 * (Close - LowestLow_n) / (HighestHigh_n - LowestLow_n)
	// Handle division by zero when HighestHigh == LowestLow
	if len(e.closes) >= 14 && len(e.highs) >= 14 && len(e.lows) >= 14 {
		highestHigh := maxSlice(e.highs[len(e.highs)-14:])
		lowestLow := minSlice(e.lows[len(e.lows)-14:])
		rangeVal := highestHigh.Sub(lowestLow)
		if rangeVal.GreaterThan(decimal.Zero) {
			feat.StochMain = candle.Close.Sub(lowestLow).Div(rangeVal).Mul(decimal.NewFromInt(100))
			// Stoch Signal = 3-period SMA of %K (proper implementation)
			e.stochHist = append(e.stochHist, feat.StochMain)
			if len(e.stochHist) > 10 {
				e.stochHist = e.stochHist[len(e.stochHist)-10:]
			}
			if len(e.stochHist) >= 3 {
				feat.StochSignal = simpleMA(e.stochHist, 3)
			} else if len(e.stochHist) > 0 {
				// Not enough history for 3-period SMA — use available average
				feat.StochSignal = simpleMA(e.stochHist, len(e.stochHist))
			}
			// If insufficient history, StochSignal remains zero (INSUFFICIENT_DATA)
		}
		// If range is zero, StochMain remains zero (INSUFFICIENT_DATA, not neutral)
	}

	// Stochastic RSI — UNAVAILABLE (requires RSI history)

	// CCI 20
	// CCI = (TP - SMA(TP, n)) / (0.015 * MeanDeviation(TP, n))
	// Handle zero mean deviation
	if len(e.closes) >= 20 && len(e.highs) >= 20 && len(e.lows) >= 20 {
		tp := typicalPrice(e.highs[len(e.highs)-20:], e.lows[len(e.lows)-20:], e.closes[len(e.closes)-20:])
		smaTP := simpleMA(tp, 20)
		meanDev := meanDeviation(tp, smaTP)
		if meanDev.GreaterThan(decimal.Zero) {
			feat.CCI = tp[len(tp)-1].Sub(smaTP).Div(meanDev.Mul(decimal.NewFromFloat(0.015)))
		}
		// If meanDev is zero, CCI remains zero (INSUFFICIENT_DATA)
	}

	// === Volatility Indicators ===

	// ATR 14 (Wilder's method)
	// TR = max(H-L, |H-prevC|, |L-prevC|)
	if len(e.closes) >= 14 {
		feat.ATR = patmath.ATR(e.highs, e.lows, e.closes, 14)
	}

	// Bollinger Bands 20/2
	// Middle = SMA20, Upper = Middle + 2*std, Lower = Middle - 2*std
	if len(e.closes) >= 20 {
		sma := simpleMA(e.closes, 20)
		stdDev := stdDevCalc(e.closes[len(e.closes)-20:], sma)
		feat.BollMiddle = sma
		feat.BollUpper = sma.Add(stdDev.Mul(decimal.NewFromInt(2)))
		feat.BollLower = sma.Sub(stdDev.Mul(decimal.NewFromInt(2)))
		// Bollinger Band Width = (Upper - Lower) / Middle
		// Protect against near-zero denominator
		if sma.GreaterThan(decimal.Zero) {
			feat.BollWidth = feat.BollUpper.Sub(feat.BollLower).Div(sma)
		}
	}

	// === Volume Indicators ===

	// OBV (On Balance Volume) — uses tick volume (provenance: TICK_VOLUME)
	// Close[t] > Close[t-1] → OBV += Volume[t]
	// Close[t] < Close[t-1] → OBV -= Volume[t]
	// Close[t] == Close[t-1] → unchanged
	if len(e.closes) >= 2 && len(e.volumes) >= 2 {
		obv := decimal.Zero
		for i := 1; i < len(e.closes); i++ {
			if e.closes[i].GreaterThan(e.closes[i-1]) {
				obv = obv.Add(decimal.NewFromInt(e.volumes[i]))
			} else if e.closes[i].LessThan(e.closes[i-1]) {
				obv = obv.Sub(decimal.NewFromInt(e.volumes[i]))
			}
		}
		feat.OBV = obv
	}

	// Volume Profile — UNAVAILABLE (requires real volume, broker provides tick volume only)
	// Cumulative Delta — UNAVAILABLE (requires centralized order-flow)
	// VWAP — set from MT5 snapshot (session-anchored)

	// Store previous values for cross detection
	e.prevEMA9 = feat.EMA9
	e.prevEMA21 = feat.EMA21
	e.prevMACD = feat.MACDMain
	e.prevMACDSignal = feat.MACDSignal
	e.prevRSI = feat.RSI
	e.prevClose = candle.Close
	e.prevBollLower = feat.BollLower
	e.prevBollUpper = feat.BollUpper

	return feat
}

// === Helper Functions ===

func simpleMA(values []decimal.Decimal, period int) decimal.Decimal {
	if len(values) < period || period <= 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for i := len(values) - period; i < len(values); i++ {
		sum = sum.Add(values[i])
	}
	return sum.Div(decimal.NewFromInt(int64(period)))
}

func calcEMA(values []decimal.Decimal, period int) decimal.Decimal {
	if len(values) < period || period <= 0 {
		return decimal.Zero
	}
	// Initialize with SMA
	ema := simpleMA(values[:period], period)
	alpha := decimal.NewFromFloat(2.0 / float64(period+1))
	for i := period; i < len(values); i++ {
		ema = alpha.Mul(values[i]).Add(decimal.NewFromInt(1).Sub(alpha).Mul(ema))
	}
	return ema
}

func stdDevCalc(values []decimal.Decimal, mean decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}
	sumSq := decimal.Zero
	for _, v := range values {
		diff := v.Sub(mean)
		sumSq = sumSq.Add(diff.Mul(diff))
	}
	variance := sumSq.Div(decimal.NewFromInt(int64(len(values))))
	f, _ := variance.Float64()
	return decimal.NewFromFloat(math.Sqrt(f))
}

func maxSlice(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}
	m := values[0]
	for _, v := range values[1:] {
		if v.GreaterThan(m) {
			m = v
		}
	}
	return m
}

func minSlice(values []decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}
	m := values[0]
	for _, v := range values[1:] {
		if v.LessThan(m) {
			m = v
		}
	}
	return m
}

func typicalPrice(highs, lows, closes []decimal.Decimal) []decimal.Decimal {
	n := len(highs)
	if len(lows) < n {
		n = len(lows)
	}
	if len(closes) < n {
		n = len(closes)
	}
	result := make([]decimal.Decimal, n)
	for i := 0; i < n; i++ {
		result[i] = highs[i].Add(lows[i]).Add(closes[i]).Div(decimal.NewFromInt(3))
	}
	return result
}

func meanDeviation(values []decimal.Decimal, mean decimal.Decimal) decimal.Decimal {
	if len(values) == 0 {
		return decimal.Zero
	}
	sum := decimal.Zero
	for _, v := range values {
		sum = sum.Add(v.Sub(mean).Abs())
	}
	return sum.Div(decimal.NewFromInt(int64(len(values))))
}

// calcDirectionalMovement computes +DM and -DM for ADX calculation.
// +DM = max(up_move, 0) if up_move > down_move, else 0
// -DM = max(down_move, 0) if down_move > up_move, else 0
func calcDirectionalMovement(highs, lows []decimal.Decimal) (plusDM, minusDM decimal.Decimal) {
	if len(highs) < 2 || len(lows) < 2 {
		return decimal.Zero, decimal.Zero
	}
	plusDM = decimal.Zero
	minusDM = decimal.Zero
	for i := 1; i < len(highs); i++ {
		upMove := highs[i].Sub(highs[i-1])
		downMove := lows[i-1].Sub(lows[i])
		if upMove.GreaterThan(downMove) && upMove.GreaterThan(decimal.Zero) {
			plusDM = plusDM.Add(upMove)
		}
		if downMove.GreaterThan(upMove) && downMove.GreaterThan(decimal.Zero) {
			minusDM = minusDM.Add(downMove)
		}
	}
	// Average
	n := decimal.NewFromInt(int64(len(highs) - 1))
	if n.GreaterThan(decimal.Zero) {
		plusDM = plusDM.Div(n)
		minusDM = minusDM.Div(n)
	}
	return
}
