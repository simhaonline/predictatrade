package features

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// SAREngine computes Parabolic SAR (Stop and Reverse).
// SOW Section 5: Parabolic SAR implementation.
// Algorithm: Wilder's Parabolic SAR with acceleration factor.
// - Start with AF = step, increment on new extreme, cap at maxAF.
// - Bullish SAR rises toward price; bearish SAR falls toward price.
// - Reversal when price touches SAR; switch direction, reset AF.
// Deterministic warm-up: requires at least 2 candles.
type SAREngine struct {
	af        decimal.Decimal // Current acceleration factor
	step      decimal.Decimal // AF increment (default 0.02)
	maxAF     decimal.Decimal // Maximum AF (default 0.20)
	sar       decimal.Decimal // Current SAR value
	ep        decimal.Decimal // Extreme point (highest high or lowest low)
	isLong    bool             // Current trend direction
	prevHigh  decimal.Decimal
	prevLow   decimal.Decimal
	ready     bool
	count     int
}

// NewSAREngine creates a new Parabolic SAR engine.
func NewSAREngine(step, maxAF float64) *SAREngine {
	if step <= 0 {
		step = 0.02
	}
	if maxAF <= 0 {
		maxAF = 0.20
	}
	return &SAREngine{
		step:  decimal.NewFromFloat(step),
		maxAF: decimal.NewFromFloat(maxAF),
		af:    decimal.NewFromFloat(step),
	}
}

// SARFeatures holds Parabolic SAR output.
type SARFeatures struct {
	Value    decimal.Decimal // Current SAR value
	IsLong   bool            // True = bullish (long), false = bearish (short)
	Reversed bool            // True if reversal occurred this bar
	Ready    bool            // True if SAR has warmed up
}

// Process updates the SAR with a new candle.
func (e *SAREngine) Process(candle *types.Candle) SARFeatures {
	if candle == nil {
		return SARFeatures{Ready: e.ready}
	}

	e.count++

	// Warm-up: first candle initializes, second candle sets initial direction
	if e.count == 1 {
		e.prevHigh = candle.High
		e.prevLow = candle.Low
		e.sar = candle.Low // Assume long initially
		e.ep = candle.High
		e.isLong = true
		e.ready = false
		return SARFeatures{Value: e.sar, IsLong: e.isLong, Ready: false}
	}

	if e.count == 2 {
		// Determine initial direction from first two candles
		if candle.Close.GreaterThan(e.prevHigh) {
			e.isLong = true
			e.sar = e.prevLow // SAR at previous low for long
			e.ep = candle.High
		} else if candle.Close.LessThan(e.prevLow) {
			e.isLong = false
			e.sar = e.prevHigh // SAR at previous high for short
			e.ep = candle.Low
		} else {
			// No clear direction, keep long
			e.sar = e.prevLow
			e.ep = candle.High
		}
		e.af = e.step
		e.ready = true
		e.prevHigh = candle.High
		e.prevLow = candle.Low
		return SARFeatures{Value: e.sar, IsLong: e.isLong, Ready: true}
	}

	// Main SAR calculation
	reversed := false
	newSAR := e.sar

	if e.isLong {
		// Bullish SAR: SAR = SAR + AF * (EP - SAR)
		newSAR = e.sar.Add(e.af.Mul(e.ep.Sub(e.sar)))

		// SAR cannot be above the prior two lows
		if newSAR.GreaterThan(e.prevLow) {
			newSAR = e.prevLow
		}
		if newSAR.GreaterThan(e.prevLow) {
			newSAR = e.prevLow
		}

		// Check for reversal: if low breaks SAR
		if candle.Low.LessThan(newSAR) {
			// Reverse to short
			reversed = true
			e.isLong = false
			newSAR = e.ep // SAR jumps to extreme point
			e.ep = candle.Low
			e.af = e.step
		} else {
			// Update extreme point and acceleration factor
			if candle.High.GreaterThan(e.ep) {
				e.ep = candle.High
				e.af = e.af.Add(e.step)
				if e.af.GreaterThan(e.maxAF) {
					e.af = e.maxAF
				}
			}
		}
	} else {
		// Bearish SAR: SAR = SAR + AF * (EP - SAR) [EP < SAR, so SAR decreases]
		newSAR = e.sar.Add(e.af.Mul(e.ep.Sub(e.sar)))

		// SAR cannot be below the prior two highs
		if newSAR.LessThan(e.prevHigh) {
			newSAR = e.prevHigh
		}

		// Check for reversal: if high breaks SAR
		if candle.High.GreaterThan(newSAR) {
			// Reverse to long
			reversed = true
			e.isLong = true
			newSAR = e.ep // SAR jumps to extreme point
			e.ep = candle.High
			e.af = e.step
		} else {
			// Update extreme point and acceleration factor
			if candle.Low.LessThan(e.ep) {
				e.ep = candle.Low
				e.af = e.af.Add(e.step)
				if e.af.GreaterThan(e.maxAF) {
					e.af = e.maxAF
				}
			}
		}
	}

	e.sar = newSAR
	e.prevHigh = candle.High
	e.prevLow = candle.Low
	e.ready = true

	return SARFeatures{
		Value:    e.sar,
		IsLong:   e.isLong,
		Reversed: reversed,
		Ready:    e.ready,
	}
}
