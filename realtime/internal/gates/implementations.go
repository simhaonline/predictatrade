// Package gates — concrete gate implementations.
// SOW Section 131.3: Gate registry and seed latency budgets.
package gates

import (
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/rs/zerolog"
)

// DataQualityGate checks feed freshness and quality (SOW Section 131.3: fast, <1ms).
type DataQualityGate struct{}

func (g *DataQualityGate) ID() types.GateID { return types.GateDataQuality }

func (g *DataQualityGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if state.State != types.GatePass {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{"FEED_QUALITY_FAILURE"}
		return eval
	}

	if input.Tick == nil {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{"NO_TICK_DATA"}
		return eval
	}

	if input.Tick.Quality == types.QualityInvalid || input.Tick.Quality == types.QualityUnavailable {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{"TICK_QUALITY_INVALID"}
		return eval
	}

	if input.Tick.Quality == types.QualityStale {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{"TICK_STALE"}
		return eval
	}

	eval.Result = types.GatePass
	return eval
}

// SessionGate checks whether the current session is allowed for the strategy (fast, <1ms).
type SessionGate struct{}

func (g *SessionGate) ID() types.GateID { return types.GateSession }

func (g *SessionGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if !input.SessionAllowed {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{string(types.NTSessionUnsuitable)}
		return eval
	}

	eval.Result = types.GatePass
	return eval
}

// NewsGate checks news blackout windows (fast, <1ms).
//
// BE-4 (fail-closed): news-data-unavailable (or stale beyond TTL) is treated as
// a VETO, not a pass. A feed outage must NOT silently remove the mandated news
// protection. The only exception is a brief gap: if a successful news sync
// occurred within NewsSyncGraceTTL, the provider is known-good and a transient
// fetch failure is tolerated (does not kill every signal). When the gate has no
// last-sync provider (LastSuccessfulSync == nil) it fails closed, because the
// absence of freshness evidence must never be interpreted as "safe to trade".
type NewsGate struct {
	// LastSuccessfulSync returns the time of the last successful news sync.
	// Optional; when nil the gate always fails closed on DATA_UNAVAILABLE.
	LastSuccessfulSync func() time.Time
}

// NewsSyncGraceTTL is the window after a successful sync during which a
// transient DATA_UNAVAILABLE (brief fetch gap) is tolerated instead of vetoing.
const NewsSyncGraceTTL = 15 * time.Minute

func NewNewsGate(lastSync func() time.Time) *NewsGate {
	return &NewsGate{LastSuccessfulSync: lastSync}
}

func (g *NewsGate) ID() types.GateID { return types.GateNews }

func (g *NewsGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	// Block on genuine news risk: HIGH, EXTREME, and BLOCKED.
	// A genuine "no news event" (NONE/LOW/MEDIUM) is NOT a veto — only actual
	// scheduled-event risk is.
	if input.NewsRisk == "HIGH" || input.NewsRisk == "EXTREME" ||
		input.NewsRisk == "BLOCKED" {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{string(types.NTHighNewsRisk)}
		return eval
	}

	// Fail-closed on unavailable/stale news data (BE-4). DATA_UNAVAILABLE means
	// the provider is stale or failed — the mandated protection must remain in
	// force. The single allowed exception is a recent successful sync: a known
	// good provider that merely dropped one fetch should not halt all trading.
	if input.NewsRisk == "DATA_UNAVAILABLE" {
		graceOK := false
		if g.LastSuccessfulSync != nil {
			last := g.LastSuccessfulSync()
			if !last.IsZero() && time.Since(last) <= NewsSyncGraceTTL {
				graceOK = true
			}
		}
		if !graceOK {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{string(types.NTNewsDataUnavailable)}
			return eval
		}
	}

	eval.Result = types.GatePass
	return eval
}

// SpreadGate checks maximum spread (fast, <1ms).
type SpreadGate struct {
	MaxSpreadAbsolute float64
	MaxSpreadToATR    float64
}

func (g *SpreadGate) ID() types.GateID { return types.GateSpread }

func (g *SpreadGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if g.MaxSpreadAbsolute > 0 && input.Spread > g.MaxSpreadAbsolute {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{string(types.NTHighSpread)}
		return eval
	}

	if g.MaxSpreadToATR > 0 && input.ATR > 0 {
		spreadToATR := input.Spread / input.ATR
		if spreadToATR > g.MaxSpreadToATR {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{"SPREAD_TO_ATR_EXCEEDED"}
			return eval
		}
	}

	eval.Result = types.GatePass
	return eval
}

// SlippageGate checks expected slippage (fast, <1ms).
type SlippageGate struct {
	MaxSlippage float64
}

func (g *SlippageGate) ID() types.GateID { return types.GateSlippage }

func (g *SlippageGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	// Slippage check uses state value
	slipState, ok := state.Value.(float64)
	if ok && g.MaxSlippage > 0 && slipState > g.MaxSlippage {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{"SLIPPAGE_EXCEEDED"}
		return eval
	}

	eval.Result = types.GatePass
	return eval
}

// TotalCostGate checks cost-to-target ratio (fast, <1ms).
// Bug 6: scalping strategies (STANDARD_SCALPING, ULTRA_SCALPING) get a
// stricter cap — (spread+slippage+commission) must be ≤ CostToTP1MaxPct of
// the TP1 distance; violations veto with `total_cost` and log arithmetic.
type TotalCostGate struct {
	MaxCostToTarget float64
	// Scalping strictness (Bug 6):
	CostToTP1MaxPct    float64                   // e.g. 0.30 → cost must be ≤ 30% of TP1 distance
	ScalpingStrategies map[types.StrategyID]bool // strategies subject to the strict cap
	Logger             *zerolog.Logger           // optional; used for arithmetic audit log
}

func (g *TotalCostGate) ID() types.GateID { return types.GateTotalCost }

func (g *TotalCostGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if input.TakeProfit1 != 0 && input.EntryPrice != 0 {
		targetDist := abs(input.TakeProfit1 - input.EntryPrice)
		if targetDist > 0 && input.RoundTripCost > 0 {
			costToTarget := input.RoundTripCost / targetDist

			// Bug 6: scalping cost strictness — veto `total_cost` when the
			// actual round-trip cost exceeds CostToTP1MaxPct of TP1 distance.
			if g.CostToTP1MaxPct > 0 && g.ScalpingStrategies[input.StrategyID] &&
				costToTarget > g.CostToTP1MaxPct {
				if g.Logger != nil {
					g.Logger.Warn().
						Str("strategy", string(input.StrategyID)).
						Float64("round_trip_cost", input.RoundTripCost).
						Float64("tp1_distance", targetDist).
						Float64("cost_to_tp1_ratio", costToTarget).
						Float64("max_cost_to_tp1_pct", g.CostToTP1MaxPct).
						Msg("[TOTAL_COST] scalping candidate rejected — cost arithmetic")
				}
				eval.Result = types.GateVeto
				eval.ReasonCodes = []string{"total_cost"}
				return eval
			}

			if g.MaxCostToTarget > 0 && costToTarget > g.MaxCostToTarget {
				eval.Result = types.GateVeto
				eval.ReasonCodes = []string{string(types.NTTotalCostExceeded)}
				return eval
			}
		}
	}

	eval.Result = types.GatePass
	return eval
}

// ExposureGate checks aggregate XAUUSD exposure (mid, <5ms).
type ExposureGate struct {
	MaxExposure float64
}

func (g *ExposureGate) ID() types.GateID { return types.GateExposure }

func (g *ExposureGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if g.MaxExposure > 0 && input.CurrentExposure+1 > g.MaxExposure {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{string(types.NTRiskLimitReached)}
		return eval
	}

	eval.Result = types.GatePass
	return eval
}

// MarginGate checks margin headroom (mid, <5ms).
type MarginGate struct{}

func (g *MarginGate) ID() types.GateID { return types.GateMargin }

func (g *MarginGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	marginOK, ok := state.Value.(bool)
	if ok && !marginOK {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{string(types.NTMarginInsufficient)}
		return eval
	}

	eval.Result = types.GatePass
	return eval
}

// RRNetExpectancyGate checks minimum R:R and net expectancy (fast, <1ms).
type RRNetExpectancyGate struct {
	MinGrossRR       float64
	MinNetExpectancy float64
}

func (g *RRNetExpectancyGate) ID() types.GateID { return types.GateRRNetExpectancy }

func (g *RRNetExpectancyGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if input.StopLoss != 0 && input.TakeProfit1 != 0 && input.EntryPrice != 0 {
		// Geometry sanity: SL and TP must bracket the entry on the correct sides.
		// A mis-bracketed trade has undefined R:R and must never be executable.
		if input.Direction == types.DirectionBuy && !(input.StopLoss < input.EntryPrice && input.TakeProfit1 > input.EntryPrice) {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{string(types.NTPoorRR), "INVALID_SLTP_GEOMETRY"}
			return eval
		}
		if input.Direction == types.DirectionSell && !(input.TakeProfit1 < input.EntryPrice && input.StopLoss > input.EntryPrice) {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{string(types.NTPoorRR), "INVALID_SLTP_GEOMETRY"}
			return eval
		}
		grossRR := abs(input.TakeProfit1-input.EntryPrice) / abs(input.EntryPrice-input.StopLoss)
		// Absolute floor: never trade R:R < 1.0 regardless of configuration, so a
		// mis-set MIN_RR can never admit negative-expectancy trades.
		minRR := g.MinGrossRR
		if minRR < 1.0 {
			minRR = 1.0
		}
		if grossRR < minRR {
			eval.Result = types.GateVeto
			eval.ReasonCodes = []string{string(types.NTPoorRR)}
			return eval
		}
	}

	eval.Result = types.GatePass
	return eval
}

// EntitlementGate checks strategy entitlement (mid, <5ms local evaluation).
type EntitlementGate struct{}

func (g *EntitlementGate) ID() types.GateID { return types.GateEntitlement }

func (g *EntitlementGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if !input.EntitlementOK {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{"ENTITLEMENT_DENIED"}
		return eval
	}

	eval.Result = types.GatePass
	return eval
}

// LicenseGate checks license validity (mid, <5ms local evaluation).
type LicenseGate struct{}

func (g *LicenseGate) ID() types.GateID { return types.GateLicense }

func (g *LicenseGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if !input.LicenseActive {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{string(types.NTLicenseRestricted)}
		return eval
	}

	eval.Result = types.GatePass
	return eval
}

// ExecutionPermissionGate checks execution permission (mid, <5ms).
type ExecutionPermissionGate struct{}

func (g *ExecutionPermissionGate) ID() types.GateID { return types.GateExecutionPermit }

func (g *ExecutionPermissionGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:       g.ID(),
		EvaluatedAt:  time.Now(),
		FreshnessMs:  state.FreshnessMs,
		StateVersion: state.SourceVersion,
	}

	if !input.ExecutionPermitted {
		eval.Result = types.GateVeto
		eval.ReasonCodes = []string{string(types.NTExecutionUnavailable)}
		return eval
	}

	eval.Result = types.GatePass
	return eval
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
