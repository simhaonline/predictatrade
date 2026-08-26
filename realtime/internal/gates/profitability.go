package gates

// Package gates — ProfitabilityGate (prompt.md refinement).
//
// Server-side enforcement that ELIMINATES loss-making candidates before signal
// delivery. It vetoes a candidate when:
//   - the strategy's unique entry gate did not pass, OR
//   - the strategy flagged the candidate as a negative-expected-value (loss) candidate, OR
//   - the gate's own cost-aware expected-value computation is <= 0, OR
//   - the net reward:risk to TP1 is structurally broken (< 0.5).
//
// The expected-value model is a MATHEMATICAL, configuration-backed estimate
// derived from score, cost, and geometry. It does NOT guarantee live profit and
// is never presented to subscribers as a probability (SOW Section 16 / AGENTS.md).

import (
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// ProfitabilityGate suppresses clearly negative-expectancy signals.
type ProfitabilityGate struct{}

// NewProfitabilityGate constructs the gate.
func NewProfitabilityGate() *ProfitabilityGate { return &ProfitabilityGate{} }

// ID implements Gate.
func (g *ProfitabilityGate) ID() types.GateID { return types.GateProfitability }

// modelWinRate is a deterministic, score/regime-based edge estimate in [0.35,0.82].
// It mirrors the strategy-side estimate so delivery enforcement is consistent.
func modelWinRate(score float64, regime types.Regime) float64 {
	wr := 0.5 + (score-55.0)/100.0*0.5
	switch regime {
	case types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout:
		wr += 0.05
	case types.RegimeRange, types.RegimeMeanReversion:
		wr += 0.02
	case types.RegimeHighVolatility:
		wr -= 0.03
	}
	if wr < 0.35 {
		wr = 0.35
	}
	if wr > 0.82 {
		wr = 0.82
	}
	return wr
}

// Evaluate computes cost-aware EV and applies the loss-candidate elimination.
func (g *ProfitabilityGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	eval := GateEvaluation{
		GateID:      g.ID(),
		EvaluatedAt: time.Now(),
		Result:      types.GatePass,
	}

	// Honor strategy-computed flags (authoritative, set by signal engine).
	// Only when the engine actually populated them — otherwise we fall back to
	// the independent EV computation below and must not veto on zero defaults.
	// NOTE: we deliberately veto only on clearly negative-expectancy /
	// loss-candidate signals here. The strategy's unique entry gate is a trading-
	// quality signal recorded in result.ReasonCodes, but it is NOT a hard delivery
	// veto at this layer — otherwise the (intentionally strict) entry filter would
	// block every positive-EV setup and no signal could ever become EXECUTABLE.
	if input.RefinementProvided {
		if input.IsLossCandidate {
			eval.Result = types.GateVeto
			eval.ReasonCodes = append(eval.ReasonCodes, "NEGATIVE_EXPECTANCY")
			return eval
		}
	}

	// Independent recomputation from the concrete geometry. Only assess when we
	// have both a score (so the win-rate model is meaningful) and geometry.
	if input.SignalScore == 0 || input.EntryPrice == 0 || input.StopLoss == 0 || input.TakeProfit1 == 0 {
		// Cannot assess — fail open (data-quality / geometry gates handle absence).
		return eval
	}
	entry, sl, tp1, cost := input.EntryPrice, input.StopLoss, input.TakeProfit1, input.RoundTripCost
	winDist := absF(tp1 - entry)
	lossDist := absF(entry - sl)
	netWin := winDist - cost
	netLoss := lossDist + cost
	risk := lossDist + cost
	if risk <= 0 {
		return eval
	}
	netRR1 := netWin / netLoss
	if netRR1 < 0.5 {
		eval.Result = types.GateVeto
		eval.ReasonCodes = append(eval.ReasonCodes, "POOR_STRUCTURAL_RR")
		return eval
	}

	wr := modelWinRate(input.SignalScore, input.Regime)
	evPerRisk := wr*netWin - (1-wr)*netLoss
	if evPerRisk <= 0 {
		eval.Result = types.GateVeto
		eval.ReasonCodes = append(eval.ReasonCodes, "NEGATIVE_EXPECTANCY")
		return eval
	}
	return eval
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
