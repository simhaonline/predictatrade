// Package strategy — Signal quality grading and expectancy computation.
// prompt.md Sections 12-14: Quality classification (A+/A/B/REJECTED)
// and expectancy engine (EV_R).
package strategy

import (
	"math"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ComputeQualityGrade assigns a quality grade (A+/A/B/REJECTED) to a signal
// based on its score, R:R profile, regime alignment, and structural confirmation.
//
// Grade semantics (prompt.md Section 12):
//
//	A+  — Exceptional setup: strong structural confirmation, excellent
//	      strategy/regime alignment, strong probability, favorable expectancy
//	A   — Normal production-quality signal
//	B   — Below delivery threshold, store for shadow/calibration
//	REJECTED — Failed a hard rule or unacceptable risk/expectancy
func ComputeQualityGrade(
	score float64,
	rrTP1, rrTP2, rrTP3 float64,
	expectancyR float64,
	regimeOK bool,
	structureConfirmed bool,
	isCandidate bool,
) types.SignalGrade {
	// Hard reject conditions
	if !regimeOK {
		return GradeRejected
	}
	// ATR-based RR must be positive
	if rrTP1 <= 0 {
		return GradeRejected
	}
	// Negative expectancy = auto reject
	if expectancyR < -0.5 {
		return GradeRejected
	}

	// A+ criteria: high score + strong R:R + confirmed structure + positive expectancy
	if score >= 70 && rrTP2 >= 2.0 && structureConfirmed && expectancyR > 0.3 {
		return types.GradeAPlus
	}

	// A criteria: good score + decent R:R + positive expectancy
	if score >= 55 && rrTP1 >= 1.3 && expectancyR >= 0.0 {
		return types.GradeA
	}

	// B criteria: moderate score or marginal expectancy — shadow, don't deliver
	if score >= 30 && !isCandidate {
		return types.GradeB
	}

	// Candidate signals with decent score
	if isCandidate && score >= 20 {
		return types.GradeC
	}

	return types.GradeNoTrade
}

// GradeRejected is an alias for GradeNoTrade used in rejection context
const GradeRejected = types.GradeNoTrade

// ComputeExpectancyR calculates expected value in R units.
//
// EV_R = (P_win × AvgWinR) − (P_loss × AvgLossR) − CostR
//
// Where:
//
//	P_win = calibrated win probability (0-1)
//	P_loss = 1 - P_win
//	AvgWinR = average of TP1/TP2/TP3 in R units (weighted toward TP1)
//	AvgLossR = 1.0 (full SL loss in R units)
//	CostR = round-trip spread cost in R units
//
// Falls back to score-based estimate when calibration unavailable.
func ComputeExpectancyR(
	calibratedProb float64,
	rrTP1, rrTP2, rrTP3 float64,
	costR float64,
) decimal.Decimal {
	// If no calibrated probability, use conservative estimate from R:R
	// Conservative: P_win ≈ 1 / (1 + RR) is a break-even probability
	if calibratedProb <= 0 || calibratedProb > 1 {
		avgRR := (rrTP1 + rrTP2 + rrTP3) / 3.0
		if avgRR <= 0 {
			return decimal.NewFromFloat(-costR)
		}
		calibratedProb = 1.0 / (1.0 + avgRR)
		// Floor at 0.25, ceiling at 0.75
		calibratedProb = math.Max(0.25, math.Min(0.75, calibratedProb))
	}

	pWin := calibratedProb
	pLoss := 1.0 - pWin

	// Weighted average win: TP1 gets 50% weight, TP2 30%, TP3 20%
	avgWinR := 0.5*rrTP1 + 0.3*rrTP2 + 0.2*rrTP3
	if avgWinR <= 0 {
		// Fallback to simple average
		avgWinR = (rrTP1 + rrTP2 + rrTP3) / 3.0
	}

	avgLossR := 1.0 // Full SL loss

	evR := (pWin * avgWinR) - (pLoss * avgLossR) - costR

	return decimal.NewFromFloat(evR)
}

// ComputeExpectancyScore converts EV_R to a 0-100 scale for sorting/filtering.
// EV_R >= 1.0 → 100, EV_R <= -0.5 → 0, linear interpolation between.
func ComputeExpectancyScore(evR decimal.Decimal) float64 {
	ev, _ := evR.Float64()
	if ev >= 1.0 {
		return 100
	}
	if ev <= -0.5 {
		return 0
	}
	return ((ev + 0.5) / 1.5) * 100
}

// ClassifyRejectionReason determines the primary reason a signal was rejected.
// This provides machine-readable diagnostics (prompt.md Sections 17-18).
func ClassifyRejectionReason(
	reasonCodes []types.NoTradeReason,
	score float64,
	rrTP1 float64,
	spreadPips float64,
	regimeOK, sessionOK bool,
	atrReady bool,
) (primary string, allReasons []string) {
	allReasons = make([]string, 0)

	for _, rc := range reasonCodes {
		switch rc {
		case types.NTATRNotReady:
			allReasons = append(allReasons, "stale_data")
		case types.NTSessionUnsuitable, types.NTSessionFilter:
			allReasons = append(allReasons, "session_filter")
		case types.NTHTFBearishVeto, types.NTHTFBullishVeto:
			allReasons = append(allReasons, "htf_contradiction")
		case types.NTInsufficientScore, types.NTScoreBelowTradeThreshold,
			types.NTScoreBelowCandidate:
			allReasons = append(allReasons, "low_score")
		case types.NTLowExpectancy:
			allReasons = append(allReasons, "low_expectancy")
		case types.NTInvalidSL, "INVALID_SL_FOR_BUY", "INVALID_SL_FOR_SELL":
			allReasons = append(allReasons, "invalid_sl")
		case types.NTInvalidTP, "INVALID_TP_FOR_BUY", "INVALID_TP_FOR_SELL":
			allReasons = append(allReasons, "invalid_tp")
		case types.NTRRBelowMin:
			allReasons = append(allReasons, "rr_below_min")
		case types.NTSpreadTooHigh:
			allReasons = append(allReasons, "spread_too_high")
		case types.NTDuplicate:
			allReasons = append(allReasons, "duplicate")
		case types.NTCooldown:
			allReasons = append(allReasons, "cooldown")
		case types.NTVolatilityFilter:
			allReasons = append(allReasons, "volatility_filter")
		case types.NTLiquidityFilter:
			allReasons = append(allReasons, "liquidity_filter")
		case types.NTRegimeMismatch, "NT_REGIME_UNKNOWN":
			allReasons = append(allReasons, "regime_mismatch")
		case types.NTDataStale:
			allReasons = append(allReasons, "data_stale")
		case types.NTNoSetup, "FIB_NO_SWING_ANCHORS":
			allReasons = append(allReasons, "no_setup")
		default:
			allReasons = append(allReasons, "other")
		}
	}

	// Add score-based reasons
	if score < 20 && !contains(allReasons, "low_score") {
		allReasons = append(allReasons, "low_score")
	}
	if rrTP1 < 1.0 && rrTP1 > 0 && !contains(allReasons, "rr_below_min") {
		allReasons = append(allReasons, "rr_below_min")
	}
	if spreadPips > 5.0 && !contains(allReasons, "spread_too_high") {
		allReasons = append(allReasons, "spread_too_high")
	}

	primary = "other"
	if len(allReasons) > 0 {
		primary = allReasons[0]
	}

	// Deduplicate
	seen := map[string]bool{}
	deduped := make([]string, 0, len(allReasons))
	for _, r := range allReasons {
		if !seen[r] {
			seen[r] = true
			deduped = append(deduped, r)
		}
	}

	return primary, deduped
}

// RejectionReasonSummary aggregates rejection statistics for monitoring.
type RejectionReasonSummary struct {
	StrategyID       string
	TotalCandidates  int
	TotalRejected    int
	TotalQualified   int
	RejectionCounts  map[string]int // reason → count
	RejectionPercent float64
	SignalsPerHour   float64
	SignalsPerDay    float64
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
