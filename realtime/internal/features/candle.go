// Package features — Candle intelligence engine.
// SOW Section 12F: Research-Derived Candle Patterns and Strategy Variant Registry.
// Analyzes candle structure: body/wick ratios, displacement, rejection, engulfing, etc.
package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// CandleEngine analyzes candle structure for signal generation.
type CandleEngine struct {
	prevCandle *types.Candle
	consecutiveBull int
	consecutiveBear int
}

func NewCandleEngine() *CandleEngine {
	return &CandleEngine{}
}

// Process analyzes a candle and produces CandleIntelligence.
func (e *CandleEngine) Process(candle *types.Candle, atr decimal.Decimal) CandleIntelligence {
	ci := CandleIntelligence{}
	if candle == nil {
		return ci
	}

	body := candle.Close.Sub(candle.Open).Abs()
	range_ := candle.High.Sub(candle.Low)
	if range_.LessThanOrEqual(decimal.Zero) {
		return ci
	}

	ci.BodySize = body
	ci.Range = range_
	ci.IsBullish = candle.Close.GreaterThan(candle.Open)
	ci.IsBearish = candle.Close.LessThan(candle.Open)

	// Wicks
	highBody := candle.High.Sub(decimal.Max(candle.Close, candle.Open))
	lowBody := decimal.Min(candle.Close, candle.Open).Sub(candle.Low)
	ci.UpperWick = highBody
	ci.LowerWick = lowBody

	// Body/range ratio
	if range_.GreaterThan(decimal.Zero) {
		ci.BodyRangeRatio = body.Div(range_)
	}

	// ATR normalized
	if atr.GreaterThan(decimal.Zero) {
		ci.ATRNormalized = range_.Div(atr)
		ci.IsDisplacement = body.GreaterThan(atr.Mul(decimal.NewFromFloat(2.0)))
		ci.IsCompression = range_.LessThan(atr.Mul(decimal.NewFromFloat(0.5)))
		ci.IsExpansion = range_.GreaterThan(atr.Mul(decimal.NewFromFloat(2.0)))
	}

	// Doji: body < 10% of range
	ci.IsDoji = body.LessThan(range_.Mul(decimal.NewFromFloat(0.1)))

	// Pin bar: long wick > 60% of range
	longWick := highBody
	if lowBody.GreaterThan(longWick) {
		longWick = lowBody
	}
	ci.IsPinBar = longWick.GreaterThan(range_.Mul(decimal.NewFromFloat(0.6)))

	// Rejection: long wick against close direction
	if ci.IsBullish {
		ci.IsRejection = lowBody.GreaterThan(range_.Mul(decimal.NewFromFloat(0.5)))
	} else if ci.IsBearish {
		ci.IsRejection = highBody.GreaterThan(range_.Mul(decimal.NewFromFloat(0.5)))
	}

	// Consecutive directional closes
	if ci.IsBullish {
		e.consecutiveBull++
		e.consecutiveBear = 0
	} else if ci.IsBearish {
		e.consecutiveBear++
		e.consecutiveBull = 0
	}
	ci.ConsecutiveBull = e.consecutiveBull
	ci.ConsecutiveBear = e.consecutiveBear

	// P2-002: Pin Bar geometry features — structured computation for shadow mode
	ci.PinBarBodyRatio = body.Div(range_)
	ci.PinBarUppWickRatio = highBody.Div(range_)
	ci.PinBarLowWickRatio = lowBody.Div(range_)
	if atr.GreaterThan(decimal.Zero) {
		ci.PinBarRngATRRatio = range_.Div(atr)
	}
	// Rejection direction
	if highBody.GreaterThan(lowBody) {
		ci.PinBarRejDirection = "SELL"
	} else if lowBody.GreaterThan(highBody) {
		ci.PinBarRejDirection = "BUY"
	}
	// Quality: body smallness + wick dominance composite
	one := decimal.NewFromInt(1)
	bodyQuality := one.Sub(ci.PinBarBodyRatio)
	dominantWick := highBody
	if lowBody.GreaterThan(highBody) {
		dominantWick = lowBody
	}
	wickDominance := dominantWick.Div(range_)
	ci.PinBarQuality = bodyQuality.Mul(decimal.NewFromFloat(0.4)).
		Add(wickDominance.Mul(decimal.NewFromFloat(0.4))).
		Add(ci.ATRNormalized.Mul(decimal.NewFromFloat(0.2)))

	// Engulfing and inside/outside bars need previous candle
	if e.prevCandle != nil {
		prevBody := e.prevCandle.Close.Sub(e.prevCandle.Open).Abs()
	
		// Engulfing: current body engulfs previous body
		ci.IsEngulfing = body.GreaterThan(prevBody) &&
			candle.Close.GreaterThan(e.prevCandle.Close) && candle.Open.LessThan(e.prevCandle.Open) ||
			body.GreaterThan(prevBody) &&
				candle.Close.LessThan(e.prevCandle.Close) && candle.Open.GreaterThan(e.prevCandle.Open)

		// Inside bar: current range within previous range
		ci.IsInsideBar = candle.High.LessThanOrEqual(e.prevCandle.High) &&
			candle.Low.GreaterThanOrEqual(e.prevCandle.Low)

		// Outside bar: current range engulfs previous range
		ci.IsOutsideBar = candle.High.GreaterThan(e.prevCandle.High) &&
			candle.Low.LessThan(e.prevCandle.Low)

		// Breakout: close beyond previous high/low
		ci.IsBreakout = candle.Close.GreaterThan(e.prevCandle.High) ||
			candle.Close.LessThan(e.prevCandle.Low)
	}

	e.prevCandle = candle
	return ci
}
