package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// IchimokuEngine computes the Ichimoku Kinko Hyo (Ichimoku Cloud) indicator.
// SOW Section 6: Complete Ichimoku implementation.
// Components: Tenkan-sen, Kijun-sen, Senkou Span A, Senkou Span B, Chikou Span.
// Displacement: Senkou spans are displaced forward by the conversion period (26).
// Chikou span is the current close displaced backward by 26 periods.
// No look-ahead bias: future-displaced plotted values are never treated as contemporaneously known.
type IchimokuEngine struct {
	tenkanPeriod  int // Conversion line period (default 9)
	kijunPeriod   int // Base line period (default 26)
	senkouBPeriod int // Senkou Span B period (default 52)
	displacement  int // Displacement period (default 26)

	highs  []decimal.Decimal
	lows   []decimal.Decimal
	closes []decimal.Decimal

	// Historical Senkou A/B values for displacement (stored at index = bar index)
	senkouAHist []decimal.Decimal
	senkouBHist []decimal.Decimal

	tenkan     decimal.Decimal
	kijun      decimal.Decimal
	senkouA    decimal.Decimal // Current plotted (displaced) Senkou A
	senkouB    decimal.Decimal // Current plotted (displaced) Senkou B
	chikou     decimal.Decimal
	ready      bool
}

// NewIchimokuEngine creates a new Ichimoku engine with standard or custom periods.
func NewIchimokuEngine(tenkan, kijun, senkouB, displacement int) *IchimokuEngine {
	if tenkan <= 0 {
		tenkan = 9
	}
	if kijun <= 0 {
		kijun = 26
	}
	if senkouB <= 0 {
		senkouB = 52
	}
	if displacement <= 0 {
		displacement = 26
	}
	return &IchimokuEngine{
		tenkanPeriod:  tenkan,
		kijunPeriod:   kijun,
		senkouBPeriod: senkouB,
		displacement:  displacement,
	}
}

// IchimokuFeatures holds Ichimoku output values.
type IchimokuFeatures struct {
	Tenkan       decimal.Decimal // Conversion line (9-period midpoint)
	Kijun        decimal.Decimal // Base line (26-period midpoint)
	SenkouA      decimal.Decimal // Leading Span A (displaced)
	SenkouB      decimal.Decimal // Leading Span B (displaced)
	Chikou       decimal.Decimal // Lagging Span (close displaced -26)
	CloudTop     decimal.Decimal // max(SenkouA, SenkouB)
	CloudBottom  decimal.Decimal // min(SenkouA, SenkouB)
	AboveCloud   bool            // Price above cloud
	BelowCloud   bool            // Price below cloud
	InCloud      bool            // Price inside cloud
	Ready        bool
}

// Process updates the Ichimoku engine with a new candle.
func (e *IchimokuEngine) Process(candle *types.Candle) IchimokuFeatures {
	if candle == nil {
		return IchimokuFeatures{Ready: e.ready}
	}

	e.highs = append(e.highs, candle.High)
	e.lows = append(e.lows, candle.Low)
	e.closes = append(e.closes, candle.Close)

	// Keep enough history for all calculations
	maxNeeded := e.senkouBPeriod + e.displacement + 10
	if len(e.highs) > maxNeeded {
		e.highs = e.highs[len(e.highs)-maxNeeded:]
		e.lows = e.lows[len(e.lows)-maxNeeded:]
		e.closes = e.closes[len(e.closes)-maxNeeded:]
		// Also trim senkou history
		excess := len(e.senkouAHist) - maxNeeded
		if excess > 0 {
			e.senkouAHist = e.senkouAHist[excess:]
			e.senkouBHist = e.senkouBHist[excess:]
		}
	}

	// Calculate Tenkan-sen: (highest_high(9) + lowest_low(9)) / 2
	if len(e.highs) >= e.tenkanPeriod {
		hh := maxSlice(e.highs[len(e.highs)-e.tenkanPeriod:])
		ll := minSlice(e.lows[len(e.lows)-e.tenkanPeriod:])
		e.tenkan = hh.Add(ll).Div(decimal.NewFromInt(2))
	}

	// Calculate Kijun-sen: (highest_high(26) + lowest_low(26)) / 2
	if len(e.highs) >= e.kijunPeriod {
		hh := maxSlice(e.highs[len(e.highs)-e.kijunPeriod:])
		ll := minSlice(e.lows[len(e.lows)-e.kijunPeriod:])
		e.kijun = hh.Add(ll).Div(decimal.NewFromInt(2))
	}

	// Calculate Senkou Span A: (Tenkan + Kijun) / 2, displaced forward by 26
	// We store it now; the plotted value is from 26 bars ago
	if !e.tenkan.IsZero() && !e.kijun.IsZero() {
		senkouA := e.tenkan.Add(e.kijun).Div(decimal.NewFromInt(2))
		e.senkouAHist = append(e.senkouAHist, senkouA)
	}

	// Calculate Senkou Span B: (highest_high(52) + lowest_low(52)) / 2, displaced forward by 26
	if len(e.highs) >= e.senkouBPeriod {
		hh := maxSlice(e.highs[len(e.highs)-e.senkouBPeriod:])
		ll := minSlice(e.lows[len(e.lows)-e.senkouBPeriod:])
		senkouB := hh.Add(ll).Div(decimal.NewFromInt(2))
		e.senkouBHist = append(e.senkouBHist, senkouB)
	}

	// Chikou Span: current close displaced backward by 26
	// We look at the close from displacement bars ago and plot it at current time
	if len(e.closes) > e.displacement {
		chikouIdx := len(e.closes) - 1 - e.displacement
		if chikouIdx >= 0 {
			e.chikou = e.closes[chikouIdx]
		}
	}

	// Extract displaced Senkou values (from displacement bars ago)
	if len(e.senkouAHist) > e.displacement {
		idx := len(e.senkouAHist) - 1 - e.displacement
		if idx >= 0 {
			e.senkouA = e.senkouAHist[idx]
		}
	}
	if len(e.senkouBHist) > e.displacement {
		idx := len(e.senkouBHist) - 1 - e.displacement
		if idx >= 0 {
			e.senkouB = e.senkouBHist[idx]
		}
	}

	// Determine readiness — need at least senkouB + displacement bars
	totalNeeded := e.senkouBPeriod + e.displacement
	e.ready = len(e.highs) >= totalNeeded && !e.senkouA.IsZero() && !e.senkouB.IsZero()

	// Build features
	feat := IchimokuFeatures{
		Tenkan:  e.tenkan,
		Kijun:   e.kijun,
		SenkouA: e.senkouA,
		SenkouB: e.senkouB,
		Chikou:  e.chikou,
		Ready:   e.ready,
	}

	// Cloud boundaries and price position
	if !e.senkouA.IsZero() && !e.senkouB.IsZero() {
		if e.senkouA.GreaterThan(e.senkouB) {
			feat.CloudTop = e.senkouA
			feat.CloudBottom = e.senkouB
		} else {
			feat.CloudTop = e.senkouB
			feat.CloudBottom = e.senkouA
		}

		close := candle.Close
		if close.GreaterThan(feat.CloudTop) {
			feat.AboveCloud = true
		} else if close.LessThan(feat.CloudBottom) {
			feat.BelowCloud = true
		} else {
			feat.InCloud = true
		}
	}

	return feat
}
