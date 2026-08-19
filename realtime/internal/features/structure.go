package features

import (
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// StructureEngine detects market structure: BOS, CHoCH, MSS.
// SOW Section 4, 12A: Market Structure Analysis
// Improved: proper fractal-based swing detection with right-side confirmation (no look-ahead bias).
// A swing high is confirmed only after N bars on each side have lower highs.
// A swing low is confirmed only after N bars on each side have higher lows.
type StructureEngine struct {
	lookback     int
	confirmBars  int // Number of bars on each side to confirm a swing
	candles      []candleInfo
	swings       []swingPoint
	lastTrend    string
	lastBOS      *StructureEvent
	lastCHoCH    *StructureEvent
	structureBreak bool
}

type candleInfo struct {
	high      decimal.Decimal
	low       decimal.Decimal
	close     decimal.Decimal
	timestamp time.Time
	index     int
}

type swingPoint struct {
	price     decimal.Decimal
	isHigh    bool
	timestamp time.Time
	index     int
	confirmed bool
}

func NewStructureEngine(lookback int) *StructureEngine {
	confirmBars := 2 // Standard fractal: 2 bars on each side
	if lookback < 10 {
		lookback = 50
	}
	return &StructureEngine{
		lookback:    lookback,
		confirmBars: confirmBars,
		lastTrend:   "neutral",
	}
}

// Process updates the structure engine with a new candle.
// Implements fractal-based swing detection with right-side confirmation.
// No look-ahead bias: a swing is only confirmed after enough right-side bars exist.
func (e *StructureEngine) Process(candle *types.Candle) StructureFeatures {
	feat := StructureFeatures{CurrentTrend: e.lastTrend}

	if candle == nil {
		if e.lastBOS != nil {
			feat.LastBOS = e.lastBOS
		}
		if e.lastCHoCH != nil {
			feat.LastCHoCH = e.lastCHoCH
		}
		return feat
	}

	// Add candle to history
	idx := len(e.candles)
	e.candles = append(e.candles, candleInfo{
		high:      candle.High,
		low:       candle.Low,
		close:     candle.Close,
		timestamp: candle.Time,
		index:     idx,
	})

	// Keep enough history for swing detection
	maxCandles := e.lookback * 2
	if len(e.candles) > maxCandles {
		e.candles = e.candles[len(e.candles)-maxCandles:]
		// Adjust indices
		for i := range e.candles {
			e.candles[i].index = i
		}
	}

	// Detect confirmed swings using fractal logic
	// A swing high at index i is confirmed if:
	// - There are at least confirmBars bars before and after i
	// - All bars in [i-confirmBars, i-1] have lower highs
	// - All bars in [i+1, i+confirmBars] have lower highs
	e.detectSwings()

	// Extract recent confirmed swing highs and lows
	for _, s := range e.swings {
		if s.confirmed {
			if s.isHigh {
				feat.SwingHighs = append(feat.SwingHighs, s.price)
			} else {
				feat.SwingLows = append(feat.SwingLows, s.price)
			}
		}
	}
	// Keep only last 5
	if len(feat.SwingHighs) > 5 {
		feat.SwingHighs = feat.SwingHighs[len(feat.SwingHighs)-5:]
	}
	if len(feat.SwingLows) > 5 {
		feat.SwingLows = feat.SwingLows[len(feat.SwingLows)-5:]
	}

	// BOS detection: close breaks the most recent confirmed swing high (bullish)
	// or the most recent confirmed swing low (bearish)
	if len(feat.SwingHighs) > 0 && candle.Close.GreaterThan(feat.SwingHighs[len(feat.SwingHighs)-1]) {
		if e.lastTrend != "bullish" {
			e.lastBOS = &StructureEvent{
				Type:      "BOS",
				Direction: "bullish",
				Price:     feat.SwingHighs[len(feat.SwingHighs)-1],
				Time:      candle.Time,
			}
			feat.StructureBreak = true
			feat.LastBOS = e.lastBOS
			e.lastTrend = "bullish"
		}
	} else {
		feat.LastBOS = e.lastBOS
	}

	if len(feat.SwingLows) > 0 && candle.Close.LessThan(feat.SwingLows[len(feat.SwingLows)-1]) {
		if e.lastTrend == "bullish" {
			// CHoCH: was bullish, now breaking below swing low → bearish change
			e.lastCHoCH = &StructureEvent{
				Type:      "CHoCH",
				Direction: "bearish",
				Price:     feat.SwingLows[len(feat.SwingLows)-1],
				Time:      candle.Time,
			}
			feat.StructureBreak = true
			feat.LastCHoCH = e.lastCHoCH
			e.lastTrend = "bearish"
		} else if e.lastTrend != "bearish" {
			// BOS bearish from neutral
			e.lastBOS = &StructureEvent{
				Type:      "BOS",
				Direction: "bearish",
				Price:     feat.SwingLows[len(feat.SwingLows)-1],
				Time:      candle.Time,
			}
			feat.StructureBreak = true
			feat.LastBOS = e.lastBOS
			e.lastTrend = "bearish"
		}
	} else {
		feat.LastCHoCH = e.lastCHoCH
	}

	feat.CurrentTrend = e.lastTrend
	return feat
}

// detectSwings uses fractal logic to find confirmed swing highs and lows.
// A swing high at index i requires confirmBars on each side with lower highs.
// A swing low at index i requires confirmBars on each side with higher lows.
// No look-ahead: only marks as confirmed when right-side bars are available.
func (e *StructureEngine) detectSwings() {
	n := len(e.candles)
	if n < e.confirmBars*2+1 {
		return
	}

	// Check for new swing points at positions that now have enough right-side confirmation
	// We only need to check the most recent unconfirmed potential swing
	startCheck := n - e.confirmBars - 1 // The most recent bar that could now be confirmed
	if startCheck < e.confirmBars {
		startCheck = e.confirmBars
	}

	for i := startCheck; i < n-e.confirmBars; i++ {
		// Check if already in swings
		alreadyFound := false
		for _, s := range e.swings {
			if s.index == e.candles[i].index {
				alreadyFound = true
				break
			}
		}
		if alreadyFound {
			continue
		}

		// Check for swing high
		isSwingHigh := true
		for j := 1; j <= e.confirmBars; j++ {
			if !e.candles[i].high.GreaterThan(e.candles[i-j].high) ||
				!e.candles[i].high.GreaterThan(e.candles[i+j].high) {
				isSwingHigh = false
				break
			}
		}
		if isSwingHigh {
			e.swings = append(e.swings, swingPoint{
				price:     e.candles[i].high,
				isHigh:    true,
				timestamp: e.candles[i].timestamp,
				index:     e.candles[i].index,
				confirmed: true,
			})
		}

		// Check for swing low
		isSwingLow := true
		for j := 1; j <= e.confirmBars; j++ {
			if !e.candles[i].low.LessThan(e.candles[i-j].low) ||
				!e.candles[i].low.LessThan(e.candles[i+j].low) {
				isSwingLow = false
				break
			}
		}
		if isSwingLow {
			e.swings = append(e.swings, swingPoint{
				price:     e.candles[i].low,
				isHigh:    false,
				timestamp: e.candles[i].timestamp,
				index:     e.candles[i].index,
				confirmed: true,
			})
		}
	}

	// Keep only recent swings
	maxSwings := e.lookback
	if len(e.swings) > maxSwings {
		e.swings = e.swings[len(e.swings)-maxSwings:]
	}
}
