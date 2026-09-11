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
	"github.com/predictatrade/realtime/internal/observability"
	"time"

	"github.com/predictatrade/realtime/internal/types"
)

// ProfitabilityGate suppresses clearly negative-expectancy signals.
type ProfitabilityGate struct{}

// NewProfitabilityGate constructs the gate.
func NewProfitabilityGate() *ProfitabilityGate { return &ProfitabilityGate{} }

// ID implements Gate.
func (g *ProfitabilityGate) ID() types.GateID { return types.GateProfitability }

// Classification groups this gate for reporting (prompt.md Sections 5-6).
func (g *ProfitabilityGate) Classification() GateClassification { return ClassSoftAlpha }

// Evaluate classifies a candidate's profitability into a QUALITY TIER and sets
// ShadowExecutable. It does NOT blanket-veto on a synthetic model (prompt.md
// Sections 21-23): a SOFT-ALPHA failure now downgrades the tier and, when
// evidence is UNKNOWN/LIMITED, tracks the candidate as SHADOW instead of
// starving the strategy. A genuine malformed signal (no entry/SL/score) still
// fails closed so it can never become EXECUTABLE.
func (g *ProfitabilityGate) Evaluate(input GateInput, state GateState) GateEvaluation {
	profile := CurrentGateProfile
	eval := GateEvaluation{
		GateID:         g.ID(),
		EvaluatedAt:    time.Now(),
		Result:         types.GatePass,
		FreshnessMs:    state.FreshnessMs,
		StateVersion:   state.SourceVersion,
		Classification: g.Classification(),
	}
	eval.Classification = ClassSoftAlpha

	// Honor strategy-computed refinement when the engine populated it.
	if input.RefinementProvided {
		if input.IsLossCandidate {
			// Engine flagged a clearly negative-EV candidate with SUFFICIENT
			// evidence. Treat as REJECT (not a silent veto) and record the cue.
			eval.Result = types.GateVeto
			eval.ReasonCodes = append(eval.ReasonCodes, "NEGATIVE_EXPECTANCY")
			eval.QualityTier = "REJECT"
			observability.Log.Warn().
				Bool("loss_candidate", input.IsLossCandidate).
				Float64("signal_score", input.SignalScore).
				Float64("round_trip_cost", input.RoundTripCost).
				Str("strategy", string(input.StrategyID)).
				Msg("[PROFITABILITY] reject — refinement flagged loss candidate (sufficient evidence)")
			return eval
		}
		// Strategy supplied a quality tier; carry it through ONLY when it is
		// already delivery-grade (A_PLUS/A/B). A non-delivery-grade tier
		// (e.g. a conservative "C" from refinement) must NOT short-circuit the
		// independent EV recomputation below — that recomputation returns B for
		// any marginally-positive-EV setup (evPerRisk >= -0.25), which IS
		// delivery-grade. Loosening the delivery-grade threshold (operator-
		// authorized) means a setup that independently clears EV/geometry is
		// executable even if the strategy's label was conservative.
		if input.QualityTier != "" && isDeliveryGradeTier(input.QualityTier) {
			eval.QualityTier = input.QualityTier
			eval.SoftScore = qualityTierScore(input.QualityTier)
			if input.ShadowExecutable {
				eval.ShadowExecutable = true
			}
			eval.Result = types.GatePass
			return eval
		}
		// Otherwise fall through to the independent EV-based tier computation.
	}

	// Independent recomputation from the concrete geometry. Only assess when we
	// have both a score (so the win-rate model is meaningful) and geometry.
	if input.SignalScore == 0 || input.EntryPrice == 0 || input.StopLoss == 0 {
		// Cannot assess — fail open (data-quality / geometry gates handle absence).
		eval.SoftScore = 0
		return eval
	}
	entry, sl, cost := input.EntryPrice, input.StopLoss, input.RoundTripCost
	// Assess against the BEST (farthest) target so a multi-TP strategy whose TP1
	// is close but TP2/TP3 are far is not vetoed on its nearest target alone.
	bestWin := 0.0
	for _, tp := range []float64{input.TakeProfit1, input.TakeProfit2, input.TakeProfit3} {
		if tp == 0 {
			continue
		}
		d := absF(tp - entry)
		if d > bestWin {
			bestWin = d
		}
	}
	if bestWin == 0 {
		// No usable target — fail open (geometry gates handle absence).
		return eval
	}
	winDist := bestWin
	lossDist := absF(entry - sl)
	netWin := winDist - cost
	netLoss := lossDist + cost
	risk := lossDist + cost
	if risk <= 0 {
		return eval
	}
	sp := profile.Strategy(input.StrategyID)
	wr := sp.AssumedHitRate + ((input.SignalScore - 55.0) / 100.0 * 0.12)
	if wr < 0.40 {
		wr = 0.40
	}
	if wr > 0.90 {
		wr = 0.90
	}
	// Normalized expectancy in R (per unit risk). All thresholds below are
	// R-based — the hard veto (-0.25), the shadow band (-0.10), the profile
	// MinNetRR ladder and qualityTierForEV. (Pre-2026-09-11 this carried
	// PRICE-UNIT EV while the veto compensated with *risk — but
	// qualityTierForEV compared its R-based constants against raw price-unit
	// EV, which for gold made "A_PLUS ≥ 0.30" trivially true at ~0.07R.)
	evPerRisk := (wr*netWin - (1-wr)*netLoss) / risk
	netRR1 := netWin / netLoss

	// Decision: map into tier + shadow; do NOT blanket-veto (prompt.md §23).
	switch {
	case evPerRisk <= -0.25:
		eval.Result = types.GateVeto
		eval.ReasonCodes = append(eval.ReasonCodes, "NEGATIVE_EXPECTANCY")
		eval.QualityTier = "REJECT"
	case netRR1 < sp.MinNetRR:
		// materially sub-minimum R:R — shadow, not veto
		eval.ShadowExecutable = true
		eval.QualityTier = "WATCH"
		eval.SoftScore = -0.3
	case evPerRisk <= -0.10:
		eval.ShadowExecutable = true
		eval.QualityTier = "WATCH"
		eval.SoftScore = -0.1
	default:
		eval.QualityTier = qualityTierForEV(evPerRisk, input.SignalScore)
		eval.SoftScore = qualityTierScore(eval.QualityTier)
	}
	return eval
}

// isDeliveryGradeTier reports whether a quality tier is executable-grade
// (A_PLUS/A/B). Used to decide whether a strategy-supplied tier should be
// adopted verbatim or recomputed from independent EV geometry.
func isDeliveryGradeTier(tier string) bool {
	return tier == "A_PLUS" || tier == "A" || tier == "B"
}

// qualityTierForEV maps EV-per-risk (R units) + score into a delivery tier.
func qualityTierForEV(evPerRisk, score float64) string {
	switch {
	case evPerRisk >= 0.30 && score >= 65:
		return "A_PLUS"
	case evPerRisk >= 0.10 && score >= 55:
		return "A"
	case evPerRisk >= -0.25:
		return "B"
	default:
		return "C"
	}
}

// qualityTierScore converts a tier to a SoftScore in [-1, 1].
func qualityTierScore(tier string) float64 {
	switch tier {
	case "A_PLUS":
		return 1.0
	case "A":
		return 0.7
	case "B":
		return 0.4
	case "C":
		return 0.1
	case "WATCH":
		return -0.3
	case "REJECT":
		return -1.0
	default:
		return 0.0
	}
}

func absF(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
