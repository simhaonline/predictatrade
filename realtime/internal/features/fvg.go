package features

import (
	"github.com/predictatrade/realtime/internal/types"
)

// FVGEngine detects Fair Value Gaps, Order Blocks, and Breakers.
// SOW Section 12A.4-12A.7
type FVGEngine struct {
	fvgs     []FVGZone
	ifvgs    []FVGZone
	obList   []OrderBlock
	breakers []OrderBlock
	recent   []*types.Candle
	lookback int
}

func NewFVGEngine(lookback int) *FVGEngine {
	return &FVGEngine{lookback: lookback}
}

func (e *FVGEngine) Process(candle *types.Candle) FVGFeatures {
	feat := FVGFeatures{}
	if candle == nil {
		return feat
	}

	e.recent = append(e.recent, candle)
	if len(e.recent) > e.lookback {
		e.recent = e.recent[len(e.recent)-e.lookback:]
	}

	// FVG detection: 3-candle imbalance
	// Bullish FVG: candle[i-1].high < candle[i+1].low
	// Bearish FVG: candle[i-1].low > candle[i+1].high
	if len(e.recent) >= 3 {
		i := len(e.recent) - 1
		c1 := e.recent[i-2]
		c3 := e.recent[i]

		// Bullish FVG
		if c1.High.LessThan(c3.Low) {
			fvg := FVGZone{
				Upper: c3.Low,
				Lower: c1.High,
				Type:  "BULLISH",
				Time:  candle.Time,
			}
			e.fvgs = append(e.fvgs, fvg)
		}

		// Bearish FVG
		if c1.Low.GreaterThan(c3.High) {
			fvg := FVGZone{
				Upper: c1.Low,
				Lower: c3.High,
				Type:  "BEARISH",
				Time:  candle.Time,
			}
			e.fvgs = append(e.fvgs, fvg)
		}

		// Order Block detection: last opposite-color candle before a strong move
		if c3.Close.GreaterThan(c1.High) && c1.Close.LessThan(c1.Open) {
			// Bearish candle before bullish move = bullish OB
			ob := OrderBlock{
				Upper: c1.High,
				Lower: c1.Low,
				Type:  "BULLISH",
				Time:  c1.Time,
			}
			e.obList = append(e.obList, ob)
		}
		if c3.Close.LessThan(c1.Low) && c1.Close.GreaterThan(c1.Open) {
			// Bullish candle before bearish move = bearish OB
			ob := OrderBlock{
				Upper: c1.High,
				Lower: c1.Low,
				Type:  "BEARISH",
				Time:  c1.Time,
			}
			e.obList = append(e.obList, ob)
		}
	}

	// Check if FVGs have been filled
	for i, fvg := range e.fvgs {
		if !fvg.Filled {
			if fvg.Type == "BULLISH" && candle.Low.LessThanOrEqual(fvg.Lower) {
				e.fvgs[i].Filled = true
			}
			if fvg.Type == "BEARISH" && candle.High.GreaterThanOrEqual(fvg.Upper) {
				e.fvgs[i].Filled = true
			}
		}
	}

	// Keep recent items
	if len(e.fvgs) > 20 {
		e.fvgs = e.fvgs[len(e.fvgs)-20:]
	}
	if len(e.obList) > 10 {
		e.obList = e.obList[len(e.obList)-10:]
	}

	// Filter unfilled FVGs
	var activeFVGs, activeIFVGs []FVGZone
	for _, f := range e.fvgs {
		if !f.Filled {
			activeFVGs = append(activeFVGs, f)
		}
	}

	feat.FVGs = activeFVGs
	feat.IFVGs = activeIFVGs
	feat.OrderBlocks = e.obList
	feat.Breakers = e.breakers
	return feat
}
