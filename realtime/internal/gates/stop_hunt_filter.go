package gates

import (
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// StopHuntFilterGate prevents entering trades when price is sitting right on a
// structural level (swing high/low). If price is within 1.5 × ATR of the structural
// level, it's likely a stop-hunt trap — wait for a break and reclaim instead.
//
// This addresses Reason 2 from the summary: "SL Placed at Structural Low Gets Hunted".
type StopHuntFilterGate struct {
	MinDistanceATR float64 // Multiplier (default 1.5): price must be > MinDistanceATR × ATR from structure
}

func (g *StopHuntFilterGate) ID() types.GateID { return types.GateStopHuntFilter }

func (g *StopHuntFilterGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	minDist := g.MinDistanceATR
	if minDist <= 0 {
		minDist = 1.5
	}

	// Need ATR and entry price to evaluate
	if input.ATR <= 0 || input.EntryPrice <= 0 {
		eval.Result = types.GatePass
		return eval
	}

	// Get structural levels from gate state (passed by the engine)
	// GateState.Value can carry a struct with StructuralLow/StructuralHigh
	// Use structural data from GateInput (populated by the signal pipeline)
	swingLow := input.StructuralLow
	swingHigh := input.StructuralHigh

	// If no structural data available, pass gracefully (can't evaluate)
	if swingLow == 0 && swingHigh == 0 {
		eval.Result = types.GatePass
		return eval
	}

	threshold := minDist * input.ATR

	// For BUY: check distance to structural low (below entry)
	// For SELL: check distance to structural high (above entry)
	// We check both — if price is too close to either, it's a trap
	if swingLow > 0 {
		distToLow := input.EntryPrice - swingLow
		if distToLow > 0 && distToLow < threshold {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{string(types.NTStructuralStopHunt)}
			return eval
		}
	}

	if swingHigh > 0 {
		distToHigh := swingHigh - input.EntryPrice
		if distToHigh > 0 && distToHigh < threshold {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{string(types.NTStructuralStopHunt)}
			return eval
		}
	}

	eval.Result = types.GatePass
	return eval
}

// MinAbsoluteATRGate rejects signals when ATR is below a strategy-specific minimum.
// Low ATR means the TP distance is too small relative to spread/slippage cost.
//
// This addresses Reason 1 from the summary: "Spread + Slippage Eats the Profit".
//
// ATR magnitude differs by orders of magnitude across timeframes (M1 vs H4), so a
// single global minimum is meaningless. The gate therefore supports per-timeframe
// thresholds (MinATRByTF), falling back to the global MinATR default when a
// timeframe is not explicitly configured.
type MinAbsoluteATRGate struct {
	MinATR float64 // Default minimum ATR (used when no per-timeframe override exists)
	// MinATRByTF overrides MinATR for a specific decision timeframe.
	MinATRByTF map[types.Timeframe]float64
}

func (g *MinAbsoluteATRGate) ID() types.GateID { return types.GateMinATR }

// threshold returns the effective ATR minimum for the input's timeframe.
func (g *MinAbsoluteATRGate) threshold(tf types.Timeframe) float64 {
	if tf != "" {
		if v, ok := g.MinATRByTF[tf]; ok && v > 0 {
			return v
		}
	}
	return g.MinATR
}

func (g *MinAbsoluteATRGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if min := g.threshold(input.Timeframe); min > 0 && input.ATR < min {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{string(types.NTLowATR)}
		return eval
	}

	eval.Result = types.GatePass
	return eval
}

// ValidateMinimumLots checks if the calculated lot size is feasible for the account.
// If lots < 0.01 and the broker doesn't support micro-lots, returns an error.
func ValidateMinimumLots(equity decimal.Decimal, riskPercent float64, slDistance decimal.Decimal, minLot decimal.Decimal) (decimal.Decimal, error) {
	if equity.IsZero() || slDistance.IsZero() {
		return decimal.Zero, nil
	}

	riskAmount := equity.Mul(decimal.NewFromFloat(riskPercent / 100.0))
	lots := riskAmount.Div(slDistance)

	if lots.LessThan(minLot) {
		if equity.LessThan(decimal.NewFromInt(200)) {
			riskAmount = equity.Mul(decimal.NewFromFloat(0.005))
			lots = riskAmount.Div(slDistance)
			if lots.LessThan(minLot) {
				return decimal.Zero, &MinLotExceedsEquityError{Equity: equity, RiskPct: 0.5, SLDistance: slDistance, MinLot: minLot}
			}
			return lots, nil
		}
		return decimal.Zero, &MinLotExceedsEquityError{Equity: equity, RiskPct: riskPercent, SLDistance: slDistance, MinLot: minLot}
	}

	return lots, nil
}

// MinLotExceedsEquityError indicates the account equity is too small for min lot.
type MinLotExceedsEquityError struct {
	Equity     decimal.Decimal
	RiskPct    float64
	SLDistance decimal.Decimal
	MinLot     decimal.Decimal
}

func (e *MinLotExceedsEquityError) Error() string {
	return "ERR_MIN_LOT_EXCEEDS_EQUITY: equity too small for minimum lot size"
}
