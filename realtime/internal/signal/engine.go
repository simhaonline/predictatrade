// Package signal implements the signal lifecycle and decision engine.
// SOW Sections 19, 24: Master Decision Hierarchy
package signal

import (
	"time"

	"github.com/google/uuid"
	"github.com/predictatrade/realtime/internal/gates"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Engine orchestrates the full decision pipeline.
// SOW Section 165: Practical Signal Decision Order
type Engine struct {
	gateRegistry     *gates.Registry
	strategyProfiles map[types.StrategyID]strategy.ConfluenceProfile
	riskProfiles     map[types.StrategyID]strategy.RiskProfile
}

// NewEngine creates a new signal engine with seeded strategy profiles.
func NewEngine(gateReg *gates.Registry) *Engine {
	return &Engine{
		gateRegistry:     gateReg,
		strategyProfiles: strategy.SeedProfiles(),
		riskProfiles:     strategy.SeedRiskProfiles(),
	}
}

// DecisionInput provides all inputs for a master decision.
type DecisionInput struct {
	StrategyID    types.StrategyID
	Tick          *types.Tick
	Regime        types.Regime
	Session       string
	SessionAllowed bool
	NewsRisk      string
	Evidence      []types.EvidenceContribution
	EntryPrice    decimal.Decimal
	StopLoss      decimal.Decimal
	TP1           decimal.Decimal
	TP2           decimal.Decimal
	TP3           decimal.Decimal
	RoundTripCost decimal.Decimal
	CurrentExposure float64
	MaxExposure    float64
	EntitlementOK  bool
	LicenseActive  bool
	ExecutionPermitted bool
}

// DecisionResult is the final output of the master decision hierarchy.
type DecisionResult struct {
	Signal       *types.Signal
	AllGatesPass bool
	GateResults  []gates.GateEvaluation
	FirstVeto    *gates.GateEvaluation
	NoTradeReasons []types.NoTradeReason
}

// Decide runs the complete master decision hierarchy (SOW Section 24).
// 1. Evaluate confluence scoring
// 2. Determine direction
// 3. Run hard gates (short-circuit)
// 4. Produce BUY/SELL/NO-TRADE
func (e *Engine) Decide(input DecisionInput) DecisionResult {
	result := DecisionResult{}

	// Step 1: Confluence scoring
	profile, ok := e.strategyProfiles[input.StrategyID]
	if !ok {
		result.NoTradeReasons = append(result.NoTradeReasons, types.NTSystemDegraded)
		return result
	}

	confluenceResult := strategy.Evaluate(profile, strategy.ConfluenceInput{
		Evidence: input.Evidence,
	})

	// Step 2: Determine direction
	direction := strategy.Direction(confluenceResult)
	if direction == types.DirectionNoTrade {
		result.NoTradeReasons = append(result.NoTradeReasons, confluenceResult.ReasonCodes...)
		// Still create a NO-TRADE signal for audit
		result.Signal = &types.Signal{
			ID:         uuid.New().String(),
			Symbol:     types.SymbolXAUUSD,
			StrategyID: input.StrategyID,
			Direction:  types.DirectionNoTrade,
			Grade:      types.GradeNoTrade,
			Status:     types.SignalDetected,
			RawScore:   confluenceResult.TotalScore,
			LongScore:  confluenceResult.LongScore,
			ShortScore: confluenceResult.ShortScore,
			ReasonCodes: result.NoTradeReasons,
			Evidence:   input.Evidence,
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().Add(time.Minute * 15),
		}
		return result
	}

	// Step 3: Run hard gates (SOW Section 131 — short-circuit)
	spread := 0.0
	atr := 0.0
	if input.Tick != nil {
		spread, _ = input.Tick.Spread.Float64()
	}

	gateInput := gates.GateInput{
		Tick:            input.Tick,
		StrategyID:      input.StrategyID,
		Regime:          input.Regime,
		Spread:          spread,
		ATR:             atr,
		EntryPrice:      toFloat(input.EntryPrice),
		StopLoss:        toFloat(input.StopLoss),
		TakeProfit1:     toFloat(input.TP1),
		RoundTripCost:   toFloat(input.RoundTripCost),
		CurrentExposure: input.CurrentExposure,
		MaxExposure:     input.MaxExposure,
		NewsRisk:        input.NewsRisk,
		SessionAllowed:  input.SessionAllowed,
		EntitlementOK:   input.EntitlementOK,
		LicenseActive:   input.LicenseActive,
		ExecutionPermitted: input.ExecutionPermitted,
	}

	allPass, gateEvals, firstVeto := e.gateRegistry.EvaluateAll(gateInput)
	result.AllGatesPass = allPass
	result.GateResults = gateEvals
	result.FirstVeto = firstVeto

	// Step 4: Final decision
	if !allPass {
		// Hard gate veto → NO-TRADE
		result.NoTradeReasons = append(result.NoTradeReasons, types.NTRiskLimitReached)
		if firstVeto != nil {
			for _, rc := range firstVeto.ReasonCodes {
				result.NoTradeReasons = append(result.NoTradeReasons, types.NoTradeReason(rc))
			}
		}
		result.Signal = &types.Signal{
			ID:         uuid.New().String(),
			Symbol:     types.SymbolXAUUSD,
			StrategyID: input.StrategyID,
			Direction:  types.DirectionNoTrade,
			Grade:      types.GradeNoTrade,
			Status:     types.SignalDetected,
			RawScore:   confluenceResult.TotalScore,
			LongScore:  confluenceResult.LongScore,
			ShortScore: confluenceResult.ShortScore,
			EntryPrice: input.EntryPrice,
			StopLoss:   input.StopLoss,
			TP1:        input.TP1,
			TP2:        input.TP2,
			TP3:        input.TP3,
			ReasonCodes: result.NoTradeReasons,
			Evidence:   input.Evidence,
			GateResults: convertGateEvals(gateEvals),
			CreatedAt:  time.Now(),
			ExpiresAt:  time.Now().Add(time.Minute * 15),
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
		ID:         uuid.New().String(),
		Symbol:     types.SymbolXAUUSD,
		StrategyID: input.StrategyID,
		Direction:  direction,
		Grade:      types.GradeUnrated, // Before calibration sufficiency (SOW Section 17A)
		Status:     types.SignalCandidate,
		RawScore:   confluenceResult.TotalScore,
		LongScore:  confluenceResult.LongScore,
		ShortScore: confluenceResult.ShortScore,
		EntryPrice: input.EntryPrice,
		StopLoss:   input.StopLoss,
		TP1:        input.TP1,
		TP2:        input.TP2,
		TP3:        input.TP3,
		GrossRRTP1: grossRR1,
		ExpectedCost: input.RoundTripCost,
		Regime:     input.Regime,
		Session:    input.Session,
		NewsRisk:   input.NewsRisk,
		ReasonCodes: nil,
		Evidence:   input.Evidence,
		GateResults: convertGateEvals(gateEvals),
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(time.Minute * 15),
	}

	return result
}

func toFloat(d decimal.Decimal) float64 {
	f, _ := d.Float64()
	return f
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
