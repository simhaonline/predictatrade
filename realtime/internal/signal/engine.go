// Package signal implements the signal lifecycle and decision engine.
// SOW Sections 19, 24: Master Decision Hierarchy
package signal

import (
	"time"

	"github.com/google/uuid"
	"github.com/predictatrade/realtime/internal/gates"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Engine orchestrates the full decision pipeline.
// SOW Section 165: Practical Signal Decision Order
type Engine struct {
	gateRegistry *gates.Registry
}

// NewEngine creates a new signal engine.
func NewEngine(gateReg *gates.Registry) *Engine {
	return &Engine{
		gateRegistry: gateReg,
	}
}

// DecisionInput provides all inputs for a master decision.
type DecisionInput struct {
	StrategyID         types.StrategyID
	Direction          types.Direction // Pre-computed by strategy
	RawScore           decimal.Decimal // Pre-computed by strategy
	LongScore          decimal.Decimal // Pre-computed by strategy
	ShortScore         decimal.Decimal // Pre-computed by strategy
	Tick               *types.Tick
	Regime             types.Regime
	Session            string
	SessionAllowed     bool
	NewsRisk           string
	Evidence           []types.EvidenceContribution
	EntryPrice         decimal.Decimal
	StopLoss           decimal.Decimal
	TP1                decimal.Decimal
	TP2                decimal.Decimal
	TP3                decimal.Decimal
	RoundTripCost      decimal.Decimal
	CurrentExposure    float64
	MaxExposure        float64
	EntitlementOK      bool
	LicenseActive      bool
	ExecutionPermitted bool
	ATR                decimal.Decimal // Phase 2: Real ATR propagation to gates
	// Phase 3: Structural levels for StopHuntFilterGate
	StructuralLow  float64
	StructuralHigh float64

	// Capital protection (R1-R7): broker snapshot + sizing inputs
	AccountEquity         float64
	AccountFreeMargin     float64
	AccountLeverage       float64
	SymbolTickValue       float64
	SymbolTickSize        float64
	LotStep               float64
	RequestedLot          float64
	PositionsKnown        bool
	OpenBuyPositions      int
	OpenSellPositions     int
	StrategyOpenPositions int

	// Strategy-provided reason codes propagated into engine decision
	// for audit traceability (fixes GAP-5: engine discards strategy reasons).
	DecisionReasons []types.NoTradeReason

	// P1-001: Broker precision — digits for price rounding (e.g., 2 for XAUUSD)
	BrokerDigits int32

	// ─── Refinement (prompt.md): micro profit-taking + profitability ───
	// Propagated from the strategy evaluation so the delivery layer can
	// eliminate loss-making candidates and surface micro profit-taking.
	MicroTP          decimal.Decimal
	PartialClosePct  float64
	EdgeScore        float64
	ExpectedValue    float64
	IsLossCandidate  bool
	EntryGatePassed  bool
}

// DecisionResult is the final output of the master decision hierarchy.
type DecisionResult struct {
	Signal         *types.Signal
	AllGatesPass   bool
	GateResults    []gates.GateEvaluation
	FirstVeto      *gates.GateEvaluation
	NoTradeReasons []types.NoTradeReason
}

// Decide runs the complete master decision hierarchy (SOW Section 24).
// The strategy has already done: evidence scoring, direction determination,
// conflict detection, MTF check, regime/session check.
// This function runs the 12 hard gates on the strategy's pre-computed result.
// 1. Accept strategy direction and score
// 2. Run hard gates (short-circuit)
// 3. Produce BUY/SELL/NO-TRADE
func (e *Engine) Decide(input DecisionInput) DecisionResult {
	result := DecisionResult{}

	// P1-001: Round prices to broker digits before gate evaluation and signal output.
	// This prevents impossible price levels that don't match broker tick size.
	digits := input.BrokerDigits
	if digits > 0 {
		input.EntryPrice = roundToDigits(input.EntryPrice, digits)
		input.StopLoss = roundToDigits(input.StopLoss, digits)
		input.TP1 = roundToDigits(input.TP1, digits)
		input.TP2 = roundToDigits(input.TP2, digits)
		input.TP3 = roundToDigits(input.TP3, digits)
	}

	// Step 1: Use the strategy's pre-computed direction
	// The strategy has already evaluated all evidence, conflicts, MTF, regime, session
	direction := input.Direction
	if direction != types.DirectionBuy && direction != types.DirectionSell {
		// NO_TRADE, WAIT, ERROR, BLOCKED — skip gates, persist as-is
		// Preserve strategy-level reason codes for traceability (audit GAP-5).
		// Only append NTInsufficientScore if the strategy provided no reasons.
		result.NoTradeReasons = input.DecisionReasons
		if len(result.NoTradeReasons) == 0 {
			result.NoTradeReasons = append(result.NoTradeReasons, types.NTInsufficientScore)
		}
		grade := types.GradeNoTrade
		if direction == types.DirectionWait {
			grade = types.GradeWait
		} else if direction == types.DirectionError {
			grade = types.GradeError
		} else if direction == types.DirectionBlocked {
			grade = types.GradeBlocked
		}
		result.Signal = &types.Signal{
			ID:          uuid.New().String(),
			Symbol:      types.SymbolXAUUSD,
			StrategyID:  input.StrategyID,
			Direction:   direction,
			Grade:       grade,
			Status:      types.SignalDetected,
			RawScore:    input.RawScore,
			LongScore:   input.LongScore,
			ShortScore:  input.ShortScore,
			EntryPrice:  input.EntryPrice,
			StopLoss:    input.StopLoss,
			TP1:         input.TP1,
			TP2:         input.TP2,
			TP3:         input.TP3,
			Regime:      input.Regime,
			Session:     input.Session,
			NewsRisk:    input.NewsRisk,
			ReasonCodes: result.NoTradeReasons,
			Evidence:    input.Evidence,
			MicroTP:          input.MicroTP,
			PartialClosePct:  input.PartialClosePct,
			EdgeScore:        input.EdgeScore,
			ExpectedValue:    input.ExpectedValue,
			IsLossCandidate:  input.IsLossCandidate,
			CreatedAt:   time.Now().UTC(),
			ExpiresAt:   time.Now().UTC().Add(time.Minute * 15),
		}
		return result
	}

	// Step 3: Run hard gates (SOW Section 131 — short-circuit)
	spread := 0.0
	atr := 0.0
	if input.Tick != nil {
		spread, _ = input.Tick.Spread.Float64()
	}
	// CRITICAL FIX: Propagate real ATR to gates (was hardcoded to 0.0)
	if !input.ATR.IsZero() {
		atr, _ = input.ATR.Float64()
	}

	gateInput := gates.GateInput{
		Tick:               input.Tick,
		StrategyID:         input.StrategyID,
		Direction:          input.Direction,
		Regime:             input.Regime,
		Spread:             spread,
		ATR:                atr,
		EntryPrice:         toFloat(input.EntryPrice),
		StopLoss:           toFloat(input.StopLoss),
		TakeProfit1:        toFloat(input.TP1),
		RoundTripCost:      toFloat(input.RoundTripCost),
		CurrentExposure:    input.CurrentExposure,
		MaxExposure:        input.MaxExposure,
		NewsRisk:           input.NewsRisk,
		SessionAllowed:     input.SessionAllowed,
		EntitlementOK:      input.EntitlementOK,
		LicenseActive:      input.LicenseActive,
		ExecutionPermitted: input.ExecutionPermitted,
		StructuralLow:      input.StructuralLow,
		StructuralHigh:     input.StructuralHigh,
		// Capital protection (R1-R7)
		AccountEquity:         input.AccountEquity,
		AccountFreeMargin:     input.AccountFreeMargin,
		AccountLeverage:       input.AccountLeverage,
		SymbolTickValue:       input.SymbolTickValue,
		SymbolTickSize:        input.SymbolTickSize,
		LotStep:               input.LotStep,
		RequestedLot:          input.RequestedLot,
		PositionsKnown:        input.PositionsKnown,
		OpenBuyPositions:      input.OpenBuyPositions,
		OpenSellPositions:     input.OpenSellPositions,
		StrategyOpenPositions: input.StrategyOpenPositions,
	}

	gateInput.IsLossCandidate = input.IsLossCandidate
	gateInput.EntryGatePassed = input.EntryGatePassed
	gateInput.RefinementProvided = true
	allPass, gateEvals, firstVeto := e.gateRegistry.EvaluateAll(gateInput)
	result.AllGatesPass = allPass
	result.GateResults = gateEvals
	result.FirstVeto = firstVeto

	// Step 4: Final decision
	if !allPass {
		// Hard gate veto → NO-TRADE
		// prompt.md Section 17: Preserve market direction (BUY/SELL), set status to BLOCKED
		// Do NOT set Direction=BLOCKED — that loses the market thesis
		if firstVeto != nil {
			for _, rc := range firstVeto.ReasonCodes {
				result.NoTradeReasons = append(result.NoTradeReasons, types.NoTradeReason(rc))
			}
		} else {
			// No specific veto found — generic gate failure
			result.NoTradeReasons = append(result.NoTradeReasons, types.NTGateDegraded)
		}
		result.Signal = &types.Signal{
			ID:          uuid.New().String(),
			Symbol:      types.SymbolXAUUSD,
			StrategyID:  input.StrategyID,
			Direction:   direction, // Keep BUY/SELL — do NOT set BLOCKED
			Grade:       types.GradeBlocked,
			Status:      types.SignalDetected,
			RawScore:    input.RawScore,
			LongScore:   input.LongScore,
			ShortScore:  input.ShortScore,
			EntryPrice:  input.EntryPrice,
			StopLoss:    input.StopLoss,
			TP1:         input.TP1,
			TP2:         input.TP2,
			TP3:         input.TP3,
			Regime:      input.Regime,
			Session:     input.Session,
			NewsRisk:    input.NewsRisk,
			ReasonCodes: result.NoTradeReasons,
			Evidence:    input.Evidence,
			GateResults: convertGateEvals(gateEvals),
			MicroTP:          input.MicroTP,
			PartialClosePct:  input.PartialClosePct,
			EdgeScore:        input.EdgeScore,
			ExpectedValue:    input.ExpectedValue,
			IsLossCandidate:  input.IsLossCandidate,
			CreatedAt:   time.Now().UTC(),
			ExpiresAt:   time.Now().UTC().Add(time.Minute * 15),
		}
		return result
	}

	// All gates pass → produce BUY/SELL signal
	// Compute gross R:R
	grossRR1 := decimal.Zero
	if !input.StopLoss.IsZero() {
		grossRR1 = input.TP1.Sub(input.EntryPrice).Abs().Div(input.EntryPrice.Sub(input.StopLoss).Abs())
	}

	result.Signal = &types.Signal{
		ID:           uuid.New().String(),
		Symbol:       types.SymbolXAUUSD,
		StrategyID:   input.StrategyID,
		Direction:    direction,
		Grade:        types.GradeUnrated, // Before calibration sufficiency (SOW Section 17A)
		Status:       types.SignalCandidate,
		RawScore:     input.RawScore,
		LongScore:    input.LongScore,
		ShortScore:   input.ShortScore,
		EntryPrice:   input.EntryPrice,
		StopLoss:     input.StopLoss,
		TP1:          input.TP1,
		TP2:          input.TP2,
		TP3:          input.TP3,
		GrossRRTP1:   grossRR1,
		ExpectedCost: input.RoundTripCost,
		Regime:       input.Regime,
		Session:      input.Session,
		NewsRisk:     input.NewsRisk,
		ReasonCodes:  nil,
		Evidence:     input.Evidence,
		GateResults:  convertGateEvals(gateEvals),
		CreatedAt:    time.Now().UTC(),
		ExpiresAt:    time.Now().UTC().Add(time.Minute * 15),
	}

	return result
}

func toFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
}

// roundToDigits rounds a decimal to the broker-specified number of digits.
// P1-001: Broker precision validation — prevents impossible price levels
// that don't match broker tick size (e.g., XAUUSD at 2 digits = 0.01 resolution).
func roundToDigits(d decimal.Decimal, digits int32) decimal.Decimal {
	return d.Round(digits)
}

func convertGateEvals(evals []gates.GateEvaluation) []types.GateEvaluation {
	result := make([]types.GateEvaluation, len(evals))
	for i, e := range evals {
		result[i] = types.GateEvaluation{
			GateID:       e.GateID,
			Result:       e.Result,
			ReasonCodes:  e.ReasonCodes,
			EvaluatedAt:  e.EvaluatedAt,
			FreshnessMs:  e.FreshnessMs,
			StateVersion: e.StateVersion,
		}
	}
	return result
}
