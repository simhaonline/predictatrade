// Package strategy — Geometry validation for every signal before broadcast.
//
// Validates SL/TP ordering, R:R ratios, and strategy config tolerance.
// If validation fails, the signal is demoted to NO_TRADE.
package strategy

import (
	"fmt"

	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ValidateGeometry checks that a signal's SL/TP/RR values are mathematically correct.
// Returns (valid, reason) — if invalid, reason explains the failure.
func ValidateGeometry(direction types.Direction, entry, sl, tp1, tp2, tp3 decimal.Decimal, cfg strategy.StrategyConfig) (bool, string) {
	if entry.IsZero() {
		return false, "ENTRY_IS_ZERO"
	}
	if sl.IsZero() {
		return false, "SL_IS_ZERO"
	}

	// Check SL is on correct side of entry
	if direction == types.DirectionBuy || string(direction) == "BUY_CANDIDATE" {
		if !sl.LessThan(entry) {
			return false, fmt.Sprintf("SL_NOT_BELOW_ENTRY: SL=%s Entry=%s", sl, entry)
		}
		if tp1.GreaterThan(decimal.Zero) && !tp1.GreaterThan(entry) {
			return false, fmt.Sprintf("TP1_NOT_ABOVE_ENTRY: TP1=%s Entry=%s", tp1, entry)
		}
		// Check TP ordering: TP1 < TP2 < TP3 for BUY
		if tp2.GreaterThan(decimal.Zero) && !tp2.GreaterThan(tp1) {
			return false, fmt.Sprintf("TP2_NOT_ABOVE_TP1: TP2=%s TP1=%s", tp2, tp1)
		}
		if tp3.GreaterThan(decimal.Zero) && !tp3.GreaterThan(tp2) {
			return false, fmt.Sprintf("TP3_NOT_ABOVE_TP2: TP3=%s TP2=%s", tp3, tp2)
		}
	} else if direction == types.DirectionSell || string(direction) == "SELL_CANDIDATE" {
		if !sl.GreaterThan(entry) {
			return false, fmt.Sprintf("SL_NOT_ABOVE_ENTRY: SL=%s Entry=%s", sl, entry)
		}
		if tp1.GreaterThan(decimal.Zero) && !tp1.LessThan(entry) {
			return false, fmt.Sprintf("TP1_NOT_BELOW_ENTRY: TP1=%s Entry=%s", tp1, entry)
		}
		if tp2.GreaterThan(decimal.Zero) && !tp2.LessThan(tp1) {
			return false, fmt.Sprintf("TP2_NOT_BELOW_TP1: TP2=%s TP1=%s", tp2, tp1)
		}
		if tp3.GreaterThan(decimal.Zero) && !tp3.LessThan(tp2) {
			return false, fmt.Sprintf("TP3_NOT_BELOW_TP2: TP3=%s TP2=%s", tp3, tp2)
		}
	}

	// Check R:R ratios are positive
	riskDist := entry.Sub(sl).Abs()
	if riskDist.IsZero() {
		return false, "ZERO_RISK_DISTANCE"
	}
	if tp1.GreaterThan(decimal.Zero) {
		rr1 := tp1.Sub(entry).Abs().Div(riskDist)
		if rr1.LessThanOrEqual(decimal.Zero) {
			return false, "RR1_NOT_POSITIVE"
		}
		// Check R:R1 matches strategy config within 0.01 tolerance
		expectedRR1 := cfg.ATRMultiplierTP1 / cfg.ATRMultiplierSL
		actualRR1, _ := rr1.Float64()
		if absFloat(actualRR1-expectedRR1) > 0.01 {
			return false, fmt.Sprintf("RR1_MISMATCH: expected=%.2f actual=%.2f", expectedRR1, actualRR1)
		}
	}

	return true, ""
}

func absFloat(f float64) float64 {
	if f < 0 {
		return -f
	}
	return f
}
