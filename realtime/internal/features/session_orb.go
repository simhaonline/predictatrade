// Package features — Session ORB (Opening Range Breakout) Engine (P2-001)
// Tracks candle prices within each trading session to compute opening ranges,
// breakout detection, and compression metrics.
//
// Reference inspiration: GOLD_ORB (no license, reference-only) —
// opening range computation per session. Clean-room reimplementation.
//
// Active mode: features are computed and available for strategy evidence scoring.
package features

import (
	"time"

	"github.com/shopspring/decimal"
)

// sessionRange tracks high/low boundaries for a single session.
type sessionRange struct {
	high decimal.Decimal
	low  decimal.Decimal
	open decimal.Decimal // first bar close
	barCount int
}

// SessionORBEngine computes opening range metrics for Asian, London, and NY sessions.
type SessionORBEngine struct {
	asian  sessionRange
	london sessionRange
	ny     sessionRange
	lastDay int // YYYYDDD day-of-year
}

// NewSessionORBEngine creates a new session ORB engine.
func NewSessionORBEngine() *SessionORBEngine {
	return &SessionORBEngine{}
}

// Process evaluates a candle for session ORB features.
// It accumulates high/low for each session and resets on new day.
// Session boundaries are evaluated in BROKER time (GMT+3) — see BrokerLocation.
func (e *SessionORBEngine) Process(candleTime time.Time, price decimal.Decimal) SessionORBFeatures {
	brokerNow := candleTime.In(BrokerLocation())
	dayOfYear := brokerNow.YearDay()

	// Reset on new day (broker-local day)
	if dayOfYear != e.lastDay {
		e.asian = sessionRange{}
		e.london = sessionRange{}
		e.ny = sessionRange{}
		e.lastDay = dayOfYear
	}

	hour := brokerNow.Hour()

	// Determine session and update range
	e.updateSessionRange(hour, price)

	feat := SessionORBFeatures{}
	if !e.asian.low.IsZero() {
		feat.AsianHigh = e.asian.high
		feat.AsianLow = e.asian.low
		feat.AsianRange = e.asian.high.Sub(e.asian.low)
	}
	if !e.london.low.IsZero() {
		feat.LondonHigh = e.london.high
		feat.LondonLow = e.london.low
		feat.LondonRange = e.london.high.Sub(e.london.low)
	}
	if !e.ny.low.IsZero() {
		feat.NYHigh = e.ny.high
		feat.NYLow = e.ny.low
		feat.NYRange = e.ny.high.Sub(e.ny.low)
	}

	// Compute breakout direction relative to current session's range
	currentRange := e.nyRange()
	if hour >= 13 && hour < 17 {
		currentRange = e.londonRange()
	}
	if !currentRange.high.IsZero() {
		if price.GreaterThan(currentRange.high) {
			feat.BreakoutDir = "BUY"
		} else if price.LessThan(currentRange.low) {
			feat.BreakoutDir = "SELL"
		}
		rangeSize := currentRange.high.Sub(currentRange.low)
		if rangeSize.GreaterThan(decimal.Zero) {
			feat.DistFromHi = price.Sub(currentRange.high).Div(rangeSize)
			feat.DistFromLo = currentRange.low.Sub(price).Div(rangeSize)
		}
	}

	return feat
}

func (e *SessionORBEngine) updateSessionRange(hour int, price decimal.Decimal) {
	switch {
	case hour >= 0 && hour < 8: // Asian/Tokyo 00:00-08:00 (broker GMT+3)
		e.updateRange(&e.asian, price)
	case hour >= 8 && hour < 17: // London 08:00-17:00 (broker GMT+3)
		e.updateRange(&e.london, price)
	case hour >= 13 && hour < 22: // NY 13:00-22:00 (broker GMT+3)
		e.updateRange(&e.ny, price)
	}
}

func (e *SessionORBEngine) updateRange(sr *sessionRange, price decimal.Decimal) {
	if sr.low.IsZero() || price.LessThan(sr.low) {
		sr.low = price
	}
	if sr.high.IsZero() || price.GreaterThan(sr.high) {
		sr.high = price
	}
	if sr.open.IsZero() {
		sr.open = price // first bar in session = opening range reference
	}
	sr.barCount++
}

func (e *SessionORBEngine) nyRange() sessionRange    { return e.ny }
func (e *SessionORBEngine) londonRange() sessionRange { return e.london }
