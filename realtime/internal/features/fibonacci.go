package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// FibonacciEngine computes Fibonacci retracement levels based on confirmed structural swings.
// SOW Section 12: Fibonacci retracement from confirmed market structure, not arbitrary windows.
// Uses the StructureEngine's confirmed swing highs/lows as valid swing anchors.
// Levels: 0.236, 0.382, 0.500, 0.618, 0.786 (configurable).
// Invalidation: when structural anchor changes, old levels are invalidated.
type FibonacciEngine struct {
	levels      []float64
	swingHigh   decimal.Decimal
	swingLow    decimal.Decimal
	direction   string // "bullish" or "bearish"
	anchorValid bool
}

// NewFibonacciEngine creates a new Fibonacci retracement engine.
func NewFibonacciEngine(levels []float64) *FibonacciEngine {
	if len(levels) == 0 {
		levels = []float64{0.236, 0.382, 0.500, 0.618, 0.786}
	}
	return &FibonacciEngine{levels: levels}
}

// FibonacciFeatures holds Fibonacci retracement levels.
type FibonacciFeatures struct {
	Levels    map[string]decimal.Decimal // "0.236" → price, etc.
	Direction string                      // "bullish" or "bearish"
	SwingHigh decimal.Decimal
	SwingLow  decimal.Decimal
	Range     decimal.Decimal
	Ready     bool
}

// Process computes Fibonacci levels from confirmed structural swings.
// The structure features must contain at least one confirmed swing high and one swing low.
func (e *FibonacciEngine) Process(candle *types.Candle, structure StructureFeatures) FibonacciFeatures {
	feat := FibonacciFeatures{
		Levels: make(map[string]decimal.Decimal),
		Ready:  false,
	}

	if candle == nil || len(structure.SwingHighs) == 0 || len(structure.SwingLows) == 0 {
		return feat
	}

	// Use the most recent confirmed swing high and swing low
	swingHigh := structure.SwingHighs[0]
	for _, h := range structure.SwingHighs {
		if h.GreaterThan(swingHigh) {
			swingHigh = h
		}
	}
	swingLow := structure.SwingLows[0]
	for _, l := range structure.SwingLows {
		if l.LessThan(swingLow) {
			swingLow = l
		}
	}

	// Require swing high > swing low (valid range)
	if !swingHigh.GreaterThan(swingLow) {
		return feat
	}

	// Check if anchors have changed (invalidation)
	if e.anchorValid && (!e.swingHigh.Equal(swingHigh) || !e.swingLow.Equal(swingLow)) {
		// Structural anchor changed — old levels invalidated, recompute
		e.anchorValid = false
	}

	// Determine direction: if current trend is bullish, draw retracement from low to high (bullish)
	// If bearish, draw from high to low (bearish)
	direction := "bullish"
	if structure.CurrentTrend == "bearish" {
		direction = "bearish"
	}

	e.swingHigh = swingHigh
	e.swingLow = swingLow
	e.direction = direction
	e.anchorValid = true

	rangeVal := swingHigh.Sub(swingLow)
	feat.SwingHigh = swingHigh
	feat.SwingLow = swingLow
	feat.Range = rangeVal
	feat.Direction = direction
	feat.Ready = true

	// Compute retracement levels
	for _, level := range e.levels {
		retracement := decimal.NewFromFloat(level).Mul(rangeVal)
		var price decimal.Decimal
		if direction == "bullish" {
			// Bullish retracement: levels measured down from swing high
			price = swingHigh.Sub(retracement)
		} else {
			// Bearish retracement: levels measured up from swing low
			price = swingLow.Add(retracement)
		}
		feat.Levels[formatLevel(level)] = price
	}

	return feat
}

func formatLevel(level float64) string {
	switch level {
	case 0.236:
		return "0.236"
	case 0.382:
		return "0.382"
	case 0.500:
		return "0.500"
	case 0.618:
		return "0.618"
	case 0.786:
		return "0.786"
	default:
		// Format any custom level
		d := decimal.NewFromFloat(level)
		return d.String()
	}
}
