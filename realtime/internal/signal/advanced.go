// Package signal — Advanced integration layer.
// Wires recovery, adaptation, hedging, ML, RL, and sentiment into the signal pipeline.
//
// The signal hot path uses ONLY cached state — no synchronous external I/O.
// All external data (sentiment, ML inference) is pre-computed by background goroutines.
package signal

import (
	"github.com/predictatrade/realtime/internal/adaptation"
	"github.com/predictatrade/realtime/internal/hedging"
	"github.com/predictatrade/realtime/internal/ml"
	"github.com/predictatrade/realtime/internal/recovery"
	"github.com/predictatrade/realtime/internal/rl"
	"github.com/predictatrade/realtime/internal/sentiment"
)

// AdvancedManagers holds the optional advanced intelligence managers.
// Any nil manager means that subsystem is disabled — the signal engine continues safely.
type AdvancedManagers struct {
	Recovery  *recovery.Manager
	Adaptation *adaptation.Manager
	Hedging   *hedging.Manager
	ML        *ml.AdaptationManager
	RL        *rl.Manager
	Sentiment *sentiment.Engine
}

// AdvancedDecisionInput extends DecisionInput with advanced context.
type AdvancedDecisionInput struct {
	DecisionInput
	// Recovery context
	AccountID     string
	Confluence    float64
	SetupGrade    string
	Confidence    float64
	// Adaptation context
	MarketContext adaptation.ContextInput
	// RL context (for filter mode)
	RLObservation rl.Observation
	RLInferenceFn func(rl.Observation) (rl.Action, float64)
}

// AdvancedDecisionResult extends DecisionResult with advanced fields.
type AdvancedDecisionResult struct {
	DecisionResult
	// Recovery fields (additive, backward-compatible)
	RecoveryState    recovery.RecoveryState
	RecoveryBlockReason recovery.BlockReason
	SizeMultiplier   float64
	// Adaptation fields
	AdaptationPhase  adaptation.MarketPhase
	AdaptationResult *adaptation.AdaptationResult
	// ML fields
	MLPrediction *ml.MLPrediction
	// RL fields
	RLDecision *rl.RLDecision
	// Sentiment
	SentimentScore float64
	// Blocked by advanced gates
	BlockedByAdvanced bool
	AdvancedBlockReason string
}

// DecideWithAdvanced runs the complete decision pipeline including advanced gates.
// The standard hard gates run first, then recovery/adaptation/RL/sentiment apply.
//
// This NEVER weakens existing hard gates. Advanced gates are ADDITIONAL.
// A signal must pass ALL standard gates AND all applicable advanced gates.
func (e *Engine) DecideWithAdvanced(input AdvancedDecisionInput, adv *AdvancedManagers) AdvancedDecisionResult {
	result := AdvancedDecisionResult{}

	// Step 1: Run standard decision pipeline
	baseResult := e.Decide(input.DecisionInput)
	result.DecisionResult = baseResult

	// If base gates already vetoed, return immediately
	if !baseResult.AllGatesPass {
		return result
	}

	// If base decision is not BUY/SELL, return
	if baseResult.Signal == nil ||
		(baseResult.Signal.Direction != "BUY" && baseResult.Signal.Direction != "SELL") {
		return result
	}

	// Step 2: Apply recovery gate
	if adv.Recovery != nil {
		key := recovery.AccountStrategyKey{
			AccountID:  input.AccountID,
			StrategyID: string(input.StrategyID),
			Symbol:     "XAUUSD",
		}
		allowed, blockReason := adv.Recovery.CheckSignal(key, input.Confluence, input.SetupGrade, input.Confidence)
		result.RecoveryState = adv.Recovery.GetState(key)
		result.SizeMultiplier = adv.Recovery.GetSizeMultiplier(key)

		if !allowed {
			result.BlockedByAdvanced = true
			result.AdvancedBlockReason = string(blockReason)
			// Convert to BLOCKED signal
			result.Signal.Direction = "BLOCKED"
			result.Signal.Grade = "BLOCKED"
			result.NoTradeReasons = append(result.NoTradeReasons, "RECOVERY_BLOCKED")
			return result
		}
	} else {
		result.SizeMultiplier = 1.0
	}

	// Step 3: Apply adaptation
	if adv.Adaptation != nil && adv.Adaptation.IsEnabled() {
		adaptResult := adv.Adaptation.Adapt(input.MarketContext, nil)
		result.AdaptationPhase = adaptResult.Phase
		result.AdaptationResult = &adaptResult
	}

	// Step 4: Apply ML prediction (inference only, with fallback)
	if adv.ML != nil && adv.ML.Config().Enabled {
		features := ml.FeatureVector{
			Regime:            regimeToFloat(input.MarketContext.Regime),
			Confluence:        input.Confluence,
			Confidence:        input.Confidence,
			ManipulationIndex: input.MarketContext.ManipulationIndex,
			Volatility:        strToFloat(input.MarketContext.VolatilityState),
			Spread:            input.MarketContext.Spread,
			ATR:               input.MarketContext.ATR,
		}
		pred := adv.ML.Predict(features)
		result.MLPrediction = &pred
	}

	// Step 5: Apply RL filter (if enabled and in filter or live mode)
	if adv.RL != nil {
		mode := adv.RL.GetMode()
		if mode == rl.RLFilterOnly || mode == rl.RLLiveApproved {
			if input.RLInferenceFn != nil {
				rlDec := adv.RL.Evaluate(input.RLObservation, input.RLInferenceFn)
				result.RLDecision = &rlDec
				if rlDec.Action == rl.ActionNoTrade && mode == rl.RLFilterOnly {
					result.BlockedByAdvanced = true
					result.AdvancedBlockReason = "RL_FILTER_VETO"
					result.Signal.Direction = "BLOCKED"
					result.Signal.Grade = "BLOCKED"
					return result
				}
			}
		}
	}

	// Step 6: Apply sentiment influence
	if adv.Sentiment != nil {
		result.SentimentScore = adv.Sentiment.GetInfluence()
	}

	return result
}

func regimeToFloat(regime string) float64 {
	if regime == "TRENDING_BULLISH" || regime == "TRENDING_BEARISH" || regime == "BREAKOUT" {
		return 1
	}
	return 0
}

func strToFloat(s string) float64 {
	if s == "HIGH" || s == "EXTREME" {
		return 0.005
	}
	if s == "LOW" {
		return 0.0005
	}
	return 0.001
}
