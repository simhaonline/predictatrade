// Package gates — Trade management invariants and helpers.
//
// This file implements the CENTRAL SAFETY INVARIANTS for trade management:
//   - Monotonic SL: never move SL backward (prompt requirement #6)
//   - Minimum improvement hysteresis (prompt requirement #7)
//   - Immutable initial R (prompt requirement #8)
//   - Unrealized R calculation (prompt requirement #9)
//   - Management stage state machine (prompt requirement #11)
//   - Broker acknowledgment state model (prompt requirement #12)
//
// This does NOT duplicate EA-side trailing logic — the EAs own the actual
// broker OrderModify/PositionModify calls. This file provides:
//  1. Server-side validation invariants for SL proposals
//  2. A state machine for tracking management lifecycle
//  3. R calculation helpers
//  4. Persistence models for audit trail
package gates

import (
	"github.com/shopspring/decimal"
)

// ManagementStage represents the lifecycle stage of an open trade.
type ManagementStage string

const (
	StageOpenInitialRisk    ManagementStage = "OPEN_INITIAL_RISK"
	StageProfitDeveloping   ManagementStage = "PROFIT_DEVELOPING"
	StageBreakEvenEligible  ManagementStage = "BREAK_EVEN_ELIGIBLE"
	StageBreakEvenProtected ManagementStage = "BREAK_EVEN_PROTECTED"
	StageProfitLocked       ManagementStage = "PROFIT_LOCKED"
	StageTrailingActive     ManagementStage = "TRAILING_ACTIVE"
	StageExited             ManagementStage = "EXITED"
)

// BrokerAckStatus represents the broker acknowledgment state for an SL modification.
type BrokerAckStatus string

const (
	BrokerAckNone         BrokerAckStatus = "NONE"
	BrokerAckPending      BrokerAckStatus = "PENDING"
	BrokerAckAcknowledged BrokerAckStatus = "ACKNOWLEDGED"
	BrokerAckRejected     BrokerAckStatus = "REJECTED"
)

// TradeManagementConfig defines strategy-specific trade management profiles.
type TradeManagementConfig struct {
	StrategyID          string
	Enabled             bool
	BreakEvenTriggerR   float64 // R multiples to trigger break-even
	ProfitLockTriggerR  float64 // R multiples to trigger profit lock
	ProfitLockR         float64 // R multiple to lock as guaranteed profit
	TrailingActivationR float64 // R multiples to activate trailing
	TrailingMethod      string  // "ATR", "STRUCTURE", "ATR_STRUCTURE", "HYBRID"
	ATRMultiplier       float64 // ATR multiplier for trailing distance
	StructureBuffer     float64 // buffer for structure-based stops
	MinImprovement      float64 // minimum SL improvement to warrant broker modification
	MaxHoldSeconds      int     // max holding time (0 = unlimited)
}

// DefaultTradeManagementConfigs returns per-strategy management profiles.
// These extend the existing EA inputs with strategy-specific behavior.
// The EAs still own the actual broker calls — this provides server-side
// validation and the persistence/audit model.
func DefaultTradeManagementConfigs() map[string]TradeManagementConfig {
	return map[string]TradeManagementConfig{
		"STANDARD_SCALPING": {
			StrategyID: "STANDARD_SCALPING", Enabled: true,
			BreakEvenTriggerR: 1.0, ProfitLockTriggerR: 1.5, ProfitLockR: 0.5,
			TrailingActivationR: 2.0, TrailingMethod: "ATR", ATRMultiplier: 2.0,
			MinImprovement: 0.01, MaxHoldSeconds: 600,
		},
		"ULTRA_SCALPING": {
			StrategyID: "ULTRA_SCALPING", Enabled: true,
			BreakEvenTriggerR: 1.0, ProfitLockTriggerR: 1.2, ProfitLockR: 0.5,
			TrailingActivationR: 1.5, TrailingMethod: "ATR", ATRMultiplier: 1.5,
			MinImprovement: 0.01, MaxHoldSeconds: 900,
		},
		"STANDARD_SWING": {
			StrategyID: "STANDARD_SWING", Enabled: true,
			BreakEvenTriggerR: 1.0, ProfitLockTriggerR: 2.0, ProfitLockR: 1.0,
			TrailingActivationR: 2.0, TrailingMethod: "ATR_STRUCTURE", ATRMultiplier: 2.0,
			MinImprovement: 0.05, MaxHoldSeconds: 14400,
		},
		"TREND_SWING": {
			StrategyID: "TREND_SWING", Enabled: true,
			BreakEvenTriggerR: 1.0, ProfitLockTriggerR: 2.5, ProfitLockR: 1.5,
			TrailingActivationR: 2.5, TrailingMethod: "ATR_STRUCTURE", ATRMultiplier: 2.5,
			MinImprovement: 0.10, MaxHoldSeconds: 86400,
		},
	}
}

// SLProposal represents a proposed SL modification with validation.
type SLProposal struct {
	Direction         string // "BUY" or "SELL"
	EntryPrice        decimal.Decimal
	InitialSL         decimal.Decimal // immutable original SL (1R reference)
	ConfirmedSL       decimal.Decimal // broker-confirmed current SL
	ProposedSL        decimal.Decimal // new proposed SL
	CurrentBid        decimal.Decimal
	CurrentAsk        decimal.Decimal
	StrategyID        string
	CurrentATR        decimal.Decimal
	BrokerStopsLevel  decimal.Decimal // broker minimum stop distance
	BrokerFreezeLevel decimal.Decimal
	TickSize          decimal.Decimal
}

// ValidateMonotonicSL checks the central safety invariant:
// SL must NEVER move backward.
// BUY: proposed_sl > confirmed_sl (must be higher)
// SELL: proposed_sl < confirmed_sl (must be lower)
func ValidateMonotonicSL(proposal SLProposal) (bool, string) {
	if proposal.Direction == "BUY" {
		if !proposal.ProposedSL.GreaterThan(proposal.ConfirmedSL) {
			return false, "MONOTONIC_VIOLATION_BUY: proposed SL must be > confirmed SL"
		}
		// SL must not be at or above current bid (would be past market)
		if !proposal.ProposedSL.LessThan(proposal.CurrentBid) {
			return false, "SL_AT_OR_ABOVE_BID: proposed SL must be below current bid"
		}
	} else if proposal.Direction == "SELL" {
		if !proposal.ProposedSL.LessThan(proposal.ConfirmedSL) {
			return false, "MONOTONIC_VIOLATION_SELL: proposed SL must be < confirmed SL"
		}
		if !proposal.ProposedSL.GreaterThan(proposal.CurrentAsk) {
			return false, "SL_AT_OR_BELOW_ASK: proposed SL must be above current ask"
		}
	} else {
		return false, "INVALID_DIRECTION"
	}
	return true, ""
}

// ValidateMinimumImprovement checks hysteresis: avoid modifying SL on every tick.
func ValidateMinimumImprovement(proposal SLProposal, minImprovement decimal.Decimal) (bool, string) {
	diff := proposal.ProposedSL.Sub(proposal.ConfirmedSL).Abs()
	if !diff.GreaterThanOrEqual(minImprovement) {
		return false, "INSUFFICIENT_IMPROVEMENT: below minimum threshold"
	}
	return true, ""
}

// ValidateBrokerStopLevel checks that the proposed SL respects broker stops level.
func ValidateBrokerStopLevel(proposal SLProposal) (bool, string) {
	if proposal.Direction == "BUY" {
		minSL := proposal.CurrentBid.Sub(proposal.BrokerStopsLevel)
		if !proposal.ProposedSL.LessThan(minSL) {
			return false, "BROKER_STOP_LEVEL_VIOLATION: SL too close to bid"
		}
	} else {
		maxSL := proposal.CurrentAsk.Add(proposal.BrokerStopsLevel)
		if !proposal.ProposedSL.GreaterThan(maxSL) {
			return false, "BROKER_STOP_LEVEL_VIOLATION: SL too close to ask"
		}
	}
	return true, ""
}

// CalculateInitialR computes the immutable 1R risk distance.
// initial_risk_distance = abs(entry_price - initial_stop_loss)
func CalculateInitialR(entryPrice, initialSL decimal.Decimal) decimal.Decimal {
	return entryPrice.Sub(initialSL).Abs()
}

// CalculateUnrealizedR computes current R multiple from current price.
// BUY: current_R = (current_bid - entry) / initial_risk_distance
// SELL: current_R = (entry - current_ask) / initial_risk_distance
func CalculateUnrealizedR(direction string, entryPrice, initialRiskDistance, currentBid, currentAsk decimal.Decimal) decimal.Decimal {
	if initialRiskDistance.IsZero() || initialRiskDistance.LessThanOrEqual(decimal.Zero) {
		return decimal.Zero // fail safe
	}
	if direction == "BUY" {
		return currentBid.Sub(entryPrice).Div(initialRiskDistance)
	} else if direction == "SELL" {
		return entryPrice.Sub(currentAsk).Div(initialRiskDistance)
	}
	return decimal.Zero
}

// DetermineManagementStage determines the trade management stage based on current R.
func DetermineManagementStage(currentR decimal.Decimal, config TradeManagementConfig) ManagementStage {
	r, _ := currentR.Float64()

	if r < 0 {
		return StageOpenInitialRisk
	}
	if r > float64(config.TrailingActivationR) {
		return StageTrailingActive
	}
	if r >= float64(config.ProfitLockTriggerR) && r <= float64(config.TrailingActivationR) {
		return StageProfitLocked
	}
	if r >= float64(config.BreakEvenTriggerR) {
		return StageBreakEvenProtected
	}
	if r > 0 {
		return StageProfitDeveloping
	}
	return StageOpenInitialRisk
}

// NormalizeSLPrice normalizes SL to broker digits and tick size.
func NormalizeSLPrice(sl decimal.Decimal, tickSize decimal.Decimal, digits int) decimal.Decimal {
	if tickSize.IsZero() {
		return sl
	}
	// Round to nearest tick size
	rounded := sl.Div(tickSize).Round(0).Mul(tickSize)
	return rounded
}

// ValidateSLProposal runs all validations and returns a combined result.
func ValidateSLProposal(proposal SLProposal, minImprovement decimal.Decimal) (bool, []string) {
	var reasons []string

	// 1. Monotonic check — SL must never move backward
	if ok, reason := ValidateMonotonicSL(proposal); !ok {
		reasons = append(reasons, reason)
	}

	// 2. Minimum improvement hysteresis
	if ok, reason := ValidateMinimumImprovement(proposal, minImprovement); !ok {
		reasons = append(reasons, reason)
	}

	// 3. Broker stop level validation
	if !proposal.BrokerStopsLevel.IsZero() {
		if ok, reason := ValidateBrokerStopLevel(proposal); !ok {
			reasons = append(reasons, reason)
		}
	}

	if len(reasons) > 0 {
		return false, reasons
	}
	return true, nil
}
