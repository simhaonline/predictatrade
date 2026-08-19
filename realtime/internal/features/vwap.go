package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// VWAPEngine computes session VWAP with standard deviation bands.
// SOW Section 12B: VWAP calculation
type VWAPEngine struct {
	cumPV   decimal.Decimal // cumulative price × volume
	cumVol  decimal.Decimal // cumulative volume
	sessionStart *types.Candle
}

func NewVWAPEngine() *VWAPEngine {
	return &VWAPEngine{}
}

func (e *VWAPEngine) Process(candle *types.Candle) VWAPFeatures {
	feat := VWAPFeatures{}
	if candle == nil || candle.Volume == 0 {
		return feat
	}

	// Track session start (simple: first candle of the day)
	if e.sessionStart == nil || candle.Time.Day() != e.sessionStart.Time.Day() {
		e.cumPV = decimal.Zero
		e.cumVol = decimal.Zero
		e.sessionStart = candle
	}

	typicalPrice := candle.High.Add(candle.Low).Add(candle.Close).Div(decimal.NewFromInt(3))
	vol := decimal.NewFromInt(candle.Volume)

	e.cumPV = e.cumPV.Add(typicalPrice.Mul(vol))
	e.cumVol = e.cumVol.Add(vol)

	if e.cumVol.GreaterThan(decimal.Zero) {
		vwap := e.cumPV.Div(e.cumVol)
		feat.SessionVWAP = vwap

		// Simple bands: 1 standard deviation approximation
		// In production, calculate proper stddev from all candles
		spread := candle.High.Sub(candle.Low)
		feat.UpperBand = vwap.Add(spread)
		feat.LowerBand = vwap.Sub(spread)
	}

	return feat
}
