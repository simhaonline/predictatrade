package features

import (
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// PivotEngine computes Daily and Weekly pivot points.
// SOW Section 13: Pivots must use previous completed period OHLC, not current incomplete.
// Daily pivots use the previous completed trading day's OHLC.
// Weekly pivots use the previous completed trading week's OHLC.
// All calculations are UTC-based with proper day/week boundary handling.
// Standard pivot formula: P = (H + L + C) / 3
// R1 = 2*P - L, S1 = 2*P - H
// R2 = P + (H - L), S2 = P - (H - L)
// R3 = H + 2*(P - L), S3 = L - 2*(H - P)
type PivotEngine struct {
	// Current accumulating period data
	currentDayOHLC   *periodOHLC
	currentWeekOHLC  *periodOHLC
	// Previous completed period data
	prevDayOHLC  *periodOHLC
	prevWeekOHLC *periodOHLC
	// Computed pivots
	dailyPivots  PivotSet
	weeklyPivots PivotSet
}

type periodOHLC struct {
	Open   decimal.Decimal
	High   decimal.Decimal
	Low    decimal.Decimal
	Close  decimal.Decimal
	Date   time.Time // Day boundary or week start (UTC)
}

// PivotSet holds a set of pivot levels.
type PivotSet struct {
	P     decimal.Decimal // Pivot point
	R1    decimal.Decimal
	R2    decimal.Decimal
	R3    decimal.Decimal
	S1    decimal.Decimal
	S2    decimal.Decimal
	S3    decimal.Decimal
	Ready bool
}

// PivotFeatures holds both daily and weekly pivots.
type PivotFeatures struct {
	Daily  PivotSet
	Weekly PivotSet
	Ready  bool
}

// NewPivotEngine creates a new pivot engine.
func NewPivotEngine() *PivotEngine {
	return &PivotEngine{}
}

// Process updates pivot calculations with a new completed candle.
// Only completed candles (IsClosed = true) should be used for period aggregation.
func (e *PivotEngine) Process(candle *types.Candle) PivotFeatures {
	if candle == nil {
		return PivotFeatures{Daily: e.dailyPivots, Weekly: e.weeklyPivots, Ready: e.dailyPivots.Ready}
	}

	candleTime := candle.Time.UTC()
	candleDate := truncateToDay(candleTime)
	weekStart := truncateToWeek(candleTime)

	// --- Daily pivot processing ---
	if e.currentDayOHLC == nil {
		// First candle ever — start current day
		e.currentDayOHLC = &periodOHLC{
			Open: candle.Open, High: candle.High, Low: candle.Low, Close: candle.Close,
			Date: candleDate,
		}
	} else if candleDate.After(e.currentDayOHLC.Date) {
		// New day — current day's data is now complete → save as previous
		e.prevDayOHLC = e.currentDayOHLC
		e.dailyPivots = computePivotSet(e.prevDayOHLC.High, e.prevDayOHLC.Low, e.prevDayOHLC.Close)
		// Start new day
		e.currentDayOHLC = &periodOHLC{
			Open: candle.Open, High: candle.High, Low: candle.Low, Close: candle.Close,
			Date: candleDate,
		}
	} else if candleDate.Equal(e.currentDayOHLC.Date) {
		// Same day — update OHLC
		if candle.High.GreaterThan(e.currentDayOHLC.High) {
			e.currentDayOHLC.High = candle.High
		}
		if candle.Low.LessThan(e.currentDayOHLC.Low) {
			e.currentDayOHLC.Low = candle.Low
		}
		e.currentDayOHLC.Close = candle.Close
	}

	// --- Weekly pivot processing ---
	if e.currentWeekOHLC == nil {
		e.currentWeekOHLC = &periodOHLC{
			Open: candle.Open, High: candle.High, Low: candle.Low, Close: candle.Close,
			Date: weekStart,
		}
	} else if weekStart.After(e.currentWeekOHLC.Date) {
		// New week — current week is complete → save as previous
		e.prevWeekOHLC = e.currentWeekOHLC
		e.weeklyPivots = computePivotSet(e.prevWeekOHLC.High, e.prevWeekOHLC.Low, e.prevWeekOHLC.Close)
		// Start new week
		e.currentWeekOHLC = &periodOHLC{
			Open: candle.Open, High: candle.High, Low: candle.Low, Close: candle.Close,
			Date: weekStart,
		}
	} else if weekStart.Equal(e.currentWeekOHLC.Date) {
		if candle.High.GreaterThan(e.currentWeekOHLC.High) {
			e.currentWeekOHLC.High = candle.High
		}
		if candle.Low.LessThan(e.currentWeekOHLC.Low) {
			e.currentWeekOHLC.Low = candle.Low
		}
		e.currentWeekOHLC.Close = candle.Close
	}

	return PivotFeatures{
		Daily:  e.dailyPivots,
		Weekly: e.weeklyPivots,
		Ready:  e.dailyPivots.Ready,
	}
}

// truncateToDay returns the UTC midnight timestamp for the given time.
func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

// truncateToWeek returns the Monday UTC midnight for the given time's week.
func truncateToWeek(t time.Time) time.Time {
	t = truncateToDay(t)
	offset := (int(t.Weekday()) - int(time.Monday) + 7) % 7
	return t.AddDate(0, 0, -offset)
}

// computePivotSet computes the standard pivot levels from a period's OHLC.
func computePivotSet(h, l, c decimal.Decimal) PivotSet {
	p := h.Add(l).Add(c).Div(decimal.NewFromInt(3))
	rangeVal := h.Sub(l)

	return PivotSet{
		P:  p,
		R1: p.Mul(decimal.NewFromInt(2)).Sub(l),
		S1: p.Mul(decimal.NewFromInt(2)).Sub(h),
		R2: p.Add(rangeVal),
		S2: p.Sub(rangeVal),
		R3: h.Add(p.Sub(l).Mul(decimal.NewFromInt(2))),
		S3: l.Sub(h.Sub(p).Mul(decimal.NewFromInt(2))),
		Ready: true,
	}
}
