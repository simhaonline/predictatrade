// Package strategy — Candidate threshold and advisory signal system.
// SOW Phase 2 Sections 7-10: Separate candidate detection from execution qualification.
//
// A BUY_CANDIDATE / SELL_CANDIDATE is a meaningful directional opportunity
// that has NOT been qualified for automatic execution.
//
// score < CandidateThreshold        → NO-TRADE (no meaningful opportunity)
// CandidateThreshold <= score < TradeThreshold → BUY_CANDIDATE / SELL_CANDIDATE
// score >= TradeThreshold           → BUY / SELL (qualified for gate evaluation)
package strategy

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// CandidateThresholdConfig defines candidate thresholds per strategy.
// CandidateThreshold < TradeThreshold (MinConfluence).
// CandidateThreshold has ZERO authority over AutoExecute.
type CandidateThresholdConfig struct {
	StrategyID        types.StrategyID
	CandidateThreshold float64
	TradeThreshold    float64
	Version           string
}

// DefaultCandidateThresholds returns candidate thresholds for all four strategies.
// These are derived from evidence weight budget analysis:
// StandardScalping max theoretical score ~80, trade threshold 50, candidate 30 (v1.1: lowered for reachability)
// UltraScalping max theoretical score ~78, trade threshold 50, candidate 30 (v1.1: lowered for reachability)
// StandardSwing max theoretical score ~92, trade threshold 40, candidate 25 (v1.1: lowered for reachability)
// TrendSwing max theoretical score ~75, trade threshold 35, candidate 20 (v1.1: lowered for reachability)
func DefaultCandidateThresholds() map[types.StrategyID]CandidateThresholdConfig {
	return map[types.StrategyID]CandidateThresholdConfig{
		types.StrategyStandardScalping: {
			StrategyID:        types.StrategyStandardScalping,
			CandidateThreshold: 35,  // raised from 30
			TradeThreshold:    55,   // raised from 50 — balanced quality vs frequency
			Version:           "1.2.0",
		},
		types.StrategyUltraScalping: {
			StrategyID:        types.StrategyUltraScalping,
			CandidateThreshold: 35,  // raised from 30
			TradeThreshold:    55,   // raised from 50 — balanced quality vs frequency
			Version:           "1.2.0",
		},
		types.StrategyStandardSwing: {
			StrategyID:        types.StrategyStandardSwing,
			CandidateThreshold: 30,  // raised from 25
			TradeThreshold:    50,   // raised from 40 — only high-confidence signals execute
			Version:           "1.2.0",
		},
		types.StrategyTrendSwing: {
			StrategyID:        types.StrategyTrendSwing,
			CandidateThreshold: 25,  // raised from 20
			TradeThreshold:    45,   // raised from 35 — only high-confidence signals execute
			Version:           "1.2.0",
		},
		types.StrategyMarnieFib: {
			StrategyID:        types.StrategyMarnieFib,
			CandidateThreshold: 20,  // raised from 15
			TradeThreshold:    40,   // raised from 30 — only high-confidence signals execute
			Version:           "1.2.0",
		},
	}
}

// SignalClass represents the classification of a signal.
type SignalClass string

const (
	SignalClassAdvisory   SignalClass = "ADVISORY"
	SignalClassExecutable SignalClass = "EXECUTABLE"
)

// EvaluateCandidateThreshold determines the signal class from score and thresholds.
// Returns: direction, signalClass, isCandidate, reasonCode
func EvaluateCandidateThreshold(
	direction types.Direction,
	rawScore decimal.Decimal,
	candidateThreshold float64,
	tradeThreshold float64,
) (types.Direction, SignalClass, bool, string) {
	score, _ := rawScore.Float64()

	// If already NO-TRADE/ERROR from strategy, keep it
	if direction != types.DirectionBuy && direction != types.DirectionSell {
		// Check if score is meaningful enough for a candidate
		if score >= candidateThreshold {
			// We have a directional score but strategy returned NO-TRADE
			// This could be because score < tradeThreshold
			// Determine the dominant direction from the score
			// In this case, the strategy already determined no clear direction
			return direction, SignalClassAdvisory, false, "SCORE_BELOW_TRADE_THRESHOLD"
		}
		return direction, "", false, "NT_SCORE_BELOW_THRESHOLD"
	}

	// Direction is BUY or SELL — check if it's a candidate or qualified
	if score >= tradeThreshold {
		return direction, SignalClassExecutable, false, "QUALIFIED"
	}

	// Score is between candidate and trade threshold → advisory candidate
	if score >= candidateThreshold {
		// Convert BUY to BUY_CANDIDATE, SELL to SELL_CANDIDATE
		return direction, SignalClassAdvisory, true, "SCORE_BELOW_TRADE_THRESHOLD"
	}

	// Score below candidate threshold — still a directional signal but weak
	return direction, SignalClassAdvisory, false, "SCORE_BELOW_CANDIDATE_THRESHOLD"
}

// CandidateDirection converts a BUY/SELL to BUY_CANDIDATE/SELL_CANDIDATE.
func CandidateDirection(direction types.Direction) types.Direction {
	switch direction {
	case types.DirectionBuy:
		return types.Direction("BUY_CANDIDATE")
	case types.DirectionSell:
		return types.Direction("SELL_CANDIDATE")
	default:
		return direction
	}
}

// DistanceToThreshold calculates how far the score is from the trade threshold.
func DistanceToThreshold(rawScore decimal.Decimal, tradeThreshold float64) float64 {
	score, _ := rawScore.Float64()
	return score - tradeThreshold
}
