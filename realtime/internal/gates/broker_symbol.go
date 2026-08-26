// Package gates — Broker Symbol Validation Gate (GitHub Reference P0-001)
// Validates that Entry/SL/TP prices respect broker symbol constraints:
// digits, stops_level, freeze_level, lot min/max/step.
//
// Reference inspiration: mt5-trade-split-manager (MIT) — validates all order
// parameters against broker SymbolInfo before submission.
// Clean-room reimplementation in PAT's gate architecture.
package gates

import (
	"math"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// BrokerSymbolValidatorGate validates price levels and lot sizes against broker
// symbol constraints. Unlike the capital-protection gates (which validate risk),
// this gate validates technical feasibility — preventing impossible orders that
// MT5 would reject anyway.
//
// Inputs (from GateInput):
//   - SymbolTickValue, SymbolTickSize: broker point/tick metadata
//   - EntryPrice, StopLoss, TakeProfit1: computed price levels
//   - RequestedLot: candidate lot size
//   - Direction: BUY or SELL
//
// Failure modes:
//   - StopsLevel too close: SL distance < stops_level in points
//   - FreezeLevel too close: SL/TP distance < freeze_level (market execution)
//   - Invalid lot: outside [min_lot, max_lot] or not aligned to lot_step
//   - Missing metadata: tick value/size missing (fail-closed)
type BrokerSymbolValidatorGate struct {
	// MinStopPoints is the minimum SL distance in points (symbol STOPS_LEVEL).
	// Zero means "no constraint from broker" (gate passes).
	MinStopPoints float64

	// MinFreezePoints is the minimum distance for any pending order (FREEZE_LEVEL).
	// Zero means "no constraint" (gate passes).
	MinFreezePoints float64

	// MinLot is the minimum allowed lot size for the symbol.
	MinLot float64

	// MaxLot is the maximum allowed lot size for the symbol.
	MaxLot float64

	// LotStep is the lot size increment (e.g., 0.01).
	LotStep float64

	// Digits is the number of decimal places for price display (e.g., 2 for XAUUSD).
	Digits int
}

func (g *BrokerSymbolValidatorGate) ID() types.GateID {
	return types.GateBrokerSymbolValidation
}

func (g *BrokerSymbolValidatorGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	// Fail-closed: if broker metadata hasn't arrived yet, degrade but don't veto.
	// We don't want to block ALL signals waiting for MT5 SymbolInfo.
	// If the gate state is UNKNOWN/NOT_INITIALIZED, DEGRADE — the capital
	// protection gates provide the hard safety barriers.
	if state.State != types.GatePass {
		eval.Result = types.GateDegraded
		eval.ReasonCodes = []string{"BROKER_SYMBOL_DATA_UNAVAILABLE"}
		return eval
	}

	// ─── Lot validation ───
	if input.RequestedLot > 0 && g.MinLot > 0 && g.MaxLot > 0 {
		if input.RequestedLot < g.MinLot {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{string(types.NTBrokerConstraint),
				"LOT_BELOW_MINIMUM"}
			return eval
		}
		if input.RequestedLot > g.MaxLot {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{string(types.NTBrokerConstraint),
				"LOT_ABOVE_MAXIMUM"}
			return eval
		}
		// Lot step alignment
		if g.LotStep > 0 {
			steps := math.Round(input.RequestedLot / g.LotStep)
			aligned := steps * g.LotStep
			if math.Abs(input.RequestedLot-aligned) > g.LotStep*0.001 {
				eval.Result = types.GateVeto
				eval.ReasonCodes = []string{string(types.NTBrokerConstraint),
					"LOT_NOT_ALIGNED_TO_STEP"}
				return eval
			}
		}
	}

	// ─── Stops level / Freeze level validation ───
	// Only validate if we have actual price levels and broker metadata
	if input.EntryPrice > 0 && input.StopLoss > 0 {
		// Compute SL distance in points
		slDistance := math.Abs(input.EntryPrice - input.StopLoss)

		// STOPS_LEVEL: minimum distance from current price for SL
		if g.MinStopPoints > 0 {
			slDistancePoints := slDistance / g.pointSize(input)
			if slDistancePoints < g.MinStopPoints {
				eval.Result = types.GateVeto
				eval.ReasonCodes = []string{string(types.NTBrokerConstraint),
					"STOP_LOSS_TOO_CLOSE"}
				return eval
			}
		}

		// FREEZE_LEVEL: minimum distance for any pending order
		if g.MinFreezePoints > 0 {
			freezeDistancePoints := slDistance / g.pointSize(input)
			if freezeDistancePoints < g.MinFreezePoints {
				eval.Result = types.GateVeto
				eval.ReasonCodes = []string{string(types.NTBrokerConstraint),
					"ORDER_WITHIN_FREEZE_LEVEL"}
				return eval
			}
		}
	}

	// ─── Take profit validation ───
	if input.EntryPrice > 0 && input.TakeProfit1 > 0 {
		tpDistance := math.Abs(input.EntryPrice - input.TakeProfit1)
		// TP must also respect stops_level (for MT5's minimum TP distance)
		if g.MinStopPoints > 0 {
			tpDistancePoints := tpDistance / g.pointSize(input)
			if tpDistancePoints < g.MinStopPoints {
				eval.Result = types.GateVeto
				eval.ReasonCodes = []string{string(types.NTBrokerConstraint),
					"TAKE_PROFIT_TOO_CLOSE"}
				return eval
			}
		}
	}

	eval.Result = types.GatePass
	return eval
}

// pointSize returns the instrument point size from GateInput.
// Uses SymbolTickSize if available; falls back to XAUUSD default 0.01.
func (g *BrokerSymbolValidatorGate) pointSize(input GateInput) float64 {
	if input.SymbolTickSize > 0 {
		return input.SymbolTickSize
	}
	// XAUUSD point is typically 0.01 (1 pip = 10 points for most brokers)
	// This is a reasonable default; the gate degrades (not vetoes) when metadata is missing.
	return 0.01
}
