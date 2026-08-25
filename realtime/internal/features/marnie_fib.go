package features

import (
	"math"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// MarnieFibEngine implements the Marnie Fib retracement/extension system.
// Ported from research/src/patresearch/indicators/marnie_fib.py
//
// Marnie's custom rules:
//   Retracement: 0.236, 0.382, 0.5, 0.618, 0.786, 1.0
//   Extension:   1.272, 1.618, 2.618
//   Golden Zone: 0.618 - 0.786 (high-probability reversal area)
//
// The engine computes:
//   - Retracement levels from confirmed swing high/low
//   - Extension levels for profit targets
//   - Golden zone boundaries (0.618-0.786)
//   - Confluence score (0-100) based on price proximity to golden zone
//   - Multi-level confluence (previous Fib level alignment)
type MarnieFibEngine struct {
	retracementRatios []float64
	extensionRatios   []float64
	swingHigh         decimal.Decimal
	swingLow          decimal.Decimal
	direction         string // "bullish" or "bearish"
	anchorValid       bool
	prevLevels        []decimal.Decimal // previous Fib levels for confluence
}

// MarnieFibFeatures holds the complete Marnie Fib output.
type MarnieFibFeatures struct {
	RetracementLevels map[string]decimal.Decimal // "0.236" → price, etc.
	ExtensionLevels   map[string]decimal.Decimal // "1.272" → price, etc.
	GoldenZoneLow     decimal.Decimal            // 0.618 level
	GoldenZoneHigh    decimal.Decimal            // 0.786 level
	Direction         string                     // "bullish" or "bearish"
	SwingHigh         decimal.Decimal
	SwingLow          decimal.Decimal
	Range             decimal.Decimal
	ConfluenceScore   float64 // 0-100
	InGoldenZone      bool    // price is within 0.618-0.786
	NearestLevel      string  // nearest Fib ratio label
	NearestLevelPrice decimal.Decimal
	Ready             bool
}

// NewMarnieFibEngine creates a Marnie Fib engine with Marnie's custom ratios.
func NewMarnieFibEngine() *MarnieFibEngine {
	return &MarnieFibEngine{
		retracementRatios: []float64{0.236, 0.382, 0.5, 0.618, 0.786, 1.0},
		extensionRatios:   []float64{1.272, 1.618, 2.618},
	}
}

// Process computes Marnie Fib levels from confirmed structural swings.
func (e *MarnieFibEngine) Process(candle *types.Candle, structure StructureFeatures, currentPrice decimal.Decimal) MarnieFibFeatures {
	feat := MarnieFibFeatures{
		RetracementLevels: make(map[string]decimal.Decimal),
		ExtensionLevels:   make(map[string]decimal.Decimal),
		Ready:             false,
	}

	// NOTE: candle is intentionally not required here. The realtime strategy
	// pipeline does not carry a single Candle into MarnieFibStrategy.Evaluate,
	// and the candle argument is unused by the Fib math (it derives everything
	// from confirmed SwingHighs/SwingLows + currentPrice). Previously a
	// `candle == nil` guard made the engine permanently NOT-ready, which meant
	// MARNIE_FIB could never emit a signal in live. Only the structural anchors
	// are required for the engine to be Ready.
	if len(structure.SwingHighs) == 0 || len(structure.SwingLows) == 0 {
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

	if !swingHigh.GreaterThan(swingLow) {
		return feat
	}

	// Check if anchors changed (invalidation)
	if e.anchorValid && (!e.swingHigh.Equal(swingHigh) || !e.swingLow.Equal(swingLow)) {
		// Store old levels for confluence check
		for r, p := range feat.RetracementLevels {
			e.prevLevels = append(e.prevLevels, p)
			_ = r
		}
		e.anchorValid = false
	}

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
	for _, ratio := range e.retracementRatios {
		retracement := decimal.NewFromFloat(ratio).Mul(rangeVal)
		var price decimal.Decimal
		if direction == "bullish" {
			price = swingHigh.Sub(retracement)
		} else {
			price = swingLow.Add(retracement)
		}
		feat.RetracementLevels[formatMarnieLevel(ratio)] = price
	}

	// Compute extension levels
	for _, ratio := range e.extensionRatios {
		extension := decimal.NewFromFloat(ratio - 1.0).Mul(rangeVal)
		var price decimal.Decimal
		if direction == "bullish" {
			price = swingHigh.Add(extension)
		} else {
			price = swingLow.Sub(extension)
		}
		feat.ExtensionLevels[formatMarnieLevel(ratio)] = price
	}

	// Golden zone: 0.618 - 0.786
	goldenLow := decimal.NewFromFloat(0.618).Mul(rangeVal)
	goldenHigh := decimal.NewFromFloat(0.786).Mul(rangeVal)
	if direction == "bullish" {
		feat.GoldenZoneLow = swingHigh.Sub(goldenLow)
		feat.GoldenZoneHigh = swingHigh.Sub(goldenHigh)
	} else {
		feat.GoldenZoneLow = swingLow.Add(goldenHigh)
		feat.GoldenZoneHigh = swingLow.Add(goldenLow)
	}

	// Ensure golden zone low < high
	if feat.GoldenZoneLow.GreaterThan(feat.GoldenZoneHigh) {
		feat.GoldenZoneLow, feat.GoldenZoneHigh = feat.GoldenZoneHigh, feat.GoldenZoneLow
	}

	// Confluence scoring
	feat.ConfluenceScore = computeMarnieConfluence(currentPrice, feat.GoldenZoneLow, feat.GoldenZoneHigh, rangeVal)
	feat.InGoldenZone = currentPrice.GreaterThanOrEqual(feat.GoldenZoneLow) && currentPrice.LessThanOrEqual(feat.GoldenZoneHigh)

	// Find nearest Fib level
	feat.NearestLevel, feat.NearestLevelPrice = findNearestMarnieLevel(currentPrice, feat.RetracementLevels, feat.ExtensionLevels)

	// Multi-level confluence: check alignment with previous Fib levels
	if len(e.prevLevels) > 0 {
		for _, prevLevel := range e.prevLevels {
			for _, currPrice := range append(mapValues(feat.RetracementLevels), mapValues(feat.ExtensionLevels)...) {
				diff := currPrice.Sub(prevLevel).Abs()
				if rangeVal.GreaterThan(decimal.Zero) {
					pct := diff.Div(rangeVal)
					if pct.LessThan(decimal.NewFromFloat(0.02)) {
						feat.ConfluenceScore = math.Min(100, feat.ConfluenceScore+10)
					}
				}
			}
		}
	}

	return feat
}

// computeMarnieConfluence calculates confluence score from price proximity to golden zone.
func computeMarnieConfluence(price, goldenLow, goldenHigh, rangeVal decimal.Decimal) float64 {
	if price.GreaterThanOrEqual(goldenLow) && price.LessThanOrEqual(goldenHigh) {
		return 100.0
	}

	distFromGolden := price.Sub(goldenLow).Abs()
	distHigh := price.Sub(goldenHigh).Abs()
	if distHigh.LessThan(distFromGolden) {
		distFromGolden = distHigh
	}

	if rangeVal.GreaterThan(decimal.Zero) {
		pct := distFromGolden.Div(rangeVal)
		pctF, _ := pct.Float64()
		return math.Max(0, 100-(pctF*200))
	}
	return 0
}

// findNearestMarnieLevel finds the closest Fib level to current price.
func findNearestMarnieLevel(price decimal.Decimal, retracements, extensions map[string]decimal.Decimal) (string, decimal.Decimal) {
	nearestLabel := ""
	nearestPrice := decimal.Zero
	minDist := decimal.Zero

	checkLevel := func(label string, levelPrice decimal.Decimal) {
		dist := price.Sub(levelPrice).Abs()
		if nearestLabel == "" || dist.LessThan(minDist) {
			nearestLabel = label
			nearestPrice = levelPrice
			minDist = dist
		}
	}

	for label, p := range retracements {
		checkLevel(label, p)
	}
	for label, p := range extensions {
		checkLevel(label, p)
	}

	return nearestLabel, nearestPrice
}

func formatMarnieLevel(ratio float64) string {
	d := decimal.NewFromFloat(ratio)
	return d.String()
}

func mapValues(m map[string]decimal.Decimal) []decimal.Decimal {
	vals := make([]decimal.Decimal, 0, len(m))
	for _, v := range m {
		vals = append(vals, v)
	}
	return vals
}
