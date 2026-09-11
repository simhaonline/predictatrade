// Package gates — versioned GateProfiles (prompt.md Phase 3 / Section 69-77).
//
// A GateProfile is the single source of truth for SOFT-ALPHA qualification
// thresholds. It is version-controlled and selected at runtime via the
// GATE_PROFILE_VERSION environment variable so a changed profile can be rolled
// back WITHOUT recompiling/redeploying the platform (prompt.md Section 226).
//
// CRITICAL INVARIANT: every profile shares the SAME hard capital-safety
// thresholds. Profiles differ ONLY in validated soft-alpha qualification
// (win-rate priors, R:R floors, cost caps, evidence requirements). No profile
// may weaken a HARD gate. CONSERVATIVE ≈ prior behaviour; BALANCED is the
// recommended production profile that removes unnecessary soft vetoes;
// OPPORTUNITY is a more permissive profile for Shadow/Paper canary testing.
package gates

import (
	"os"
	"strings"

	"github.com/predictatrade/realtime/internal/types"
)

// GateClassification groups gates so operators/reports can distinguish
// "no trade opportunity" from "good trade existed but could not be delivered"
// (prompt.md Sections 5-6, 233).
type GateClassification string

const (
	ClassHardSafety       GateClassification = "HARD_SAFETY"       // fail-closed; never relaxed for volume
	ClassExecutionQuality GateClassification = "EXECUTION_QUALITY" // context-adaptive, broker/session/vol aware
	ClassSoftAlpha       GateClassification = "SOFT_ALPHA"        // profitability/confluence/score — REBUILD TARGET
	ClassStrategySpecific GateClassification = "STRATEGY_SPECIFIC" // per-strategy entry/exit economics
	ClassEntitlement     GateClassification = "ENTITLEMENT"       // plan/license WHO may receive
	ClassDelivery        GateClassification = "DELIVERY"          // device/queue/ack transport
)

// Classification returns the taxonomy group for a gate.
func (r *Registry) Classification(id types.GateID) GateClassification {
	switch id {
	case types.GateDataQuality, types.GateSession, types.GateNews,
		types.GateExposure, types.GateMargin, types.GateRRNetExpectancy,
		types.GateWrongSideSL, types.GateRiskOversize, types.GatePositionCaps,
		types.GateDailyLoss, types.GateProfitTarget, types.GateMartingaleBan,
		types.GateEdgeValidation, types.GateBrokerSymbolValidation:
		return ClassHardSafety
	case types.GateSpread, types.GateSlippage, types.GateTotalCost, types.GateStopHuntFilter, types.GateMinATR:
		return ClassExecutionQuality
	case types.GateProfitability:
		return ClassSoftAlpha
	case types.GateEntitlement, types.GateLicense, types.GateExecutionPermit:
		return ClassEntitlement
	default:
		return ClassHardSafety
	}
}

// StrategyProfile holds the soft-alpha economics for ONE strategy. These are
// validated assumptions, not fabricated probabilities (prompt.md Sections 23, 63).
// AssumedHitRate is the prior win probability used ONLY for EV computation when
// no per-strategy outcome sample exists; it is strategy-appropriate (scalpers
// win more often at smaller R:R; swing traders need higher R:R).
type StrategyProfile struct {
	StrategyID        types.StrategyID
	AssumedHitRate    float64 // prior P(win) for EV when evidence insufficient
	MinNetRR          float64 // minimum NET R:R (after cost) to qualify
	CostToTP1MaxPct   float64 // cost must be <= this fraction of TP1 distance (scalping strictness)
	MinScore          float64 // minimum raw strategy score to be a candidate
	// MinEvidenceTrades: below this many observed outcomes the profile treats
	// the strategy's own stats as UNKNOWN and uses the prior (never auto-fails).
	MinEvidenceTrades int
}

// GateProfile is a named, versioned bundle of soft-alpha thresholds.
type GateProfile struct {
	Version string
	Label   string
	// Strategies keyed by StrategyID. Missing strategies fall back to the
	// default entry.
	Strategies map[types.StrategyID]StrategyProfile
	Default    StrategyProfile
	// Global soft-alpha switches.
	// RequireSufficientEvidence: when true, a candidate with UNKNOWN evidence
	// and negative model EV is REJECTED (conservative). When false, it becomes
	// SHADOW (tracked, not delivered) so we measure the miss instead of
	// starving the strategy (balanced/opportunity).
	RequireSufficientEvidence bool
	// MaxVetoReasonCodes: if a candidate fails more than this many soft gates,
	// it is REJECTED rather than SHADOWED (guards against garbage signals).
	MaxSoftFailForShadow int
}

// conservativeProfile reproduces the PRE-FIX behaviour as closely as possible:
// the profitability gate is allowed to veto on the synthetic model, and unknown
// evidence fails closed.
func conservativeProfile() GateProfile {
	return GateProfile{
		Version:                  "CONSERVATIVE",
		Label:                    "Conservative (pre-fix baseline)",
		RequireSufficientEvidence: true,
		MaxSoftFailForShadow:     1,
		Default: StrategyProfile{
			AssumedHitRate:  0.50,
			MinNetRR:        1.0,
			CostToTP1MaxPct: 0.30,
			MinScore:        55,
		},
		Strategies: map[types.StrategyID]StrategyProfile{
			types.StrategyStandardScalping: {StrategyID: types.StrategyStandardScalping, AssumedHitRate: 0.50, MinNetRR: 1.0, CostToTP1MaxPct: 0.30, MinScore: 55, MinEvidenceTrades: 30},
			types.StrategyUltraScalping:    {StrategyID: types.StrategyUltraScalping, AssumedHitRate: 0.50, MinNetRR: 1.0, CostToTP1MaxPct: 0.30, MinScore: 55, MinEvidenceTrades: 30},
		},
	}
}

// balancedProfile is the recommended production profile. It removes the
// unnecessary soft vetoes identified in the forensic audit:
//   - uses strategy-appropriate hit-rate priors (scalping ~0.58, swing ~0.52)
//   - treats UNKNOWN evidence as SHADOW, never as a hard REJECT
//   - allows up to MaxSoftFailForShadow soft failures to be tracked as shadow
func balancedProfile() GateProfile {
	return GateProfile{
		Version:                  "BALANCED",
		Label:                    "Balanced (recommended; removes unnecessary soft vetoes)",
		RequireSufficientEvidence: false,
		MaxSoftFailForShadow:     2,
		Default: StrategyProfile{
			AssumedHitRate:  0.52,
			MinNetRR:        0.8,
			CostToTP1MaxPct: 0.35,
			MinScore:        50,
		},
		Strategies: map[types.StrategyID]StrategyProfile{
			types.StrategyStandardScalping: {StrategyID: types.StrategyStandardScalping, AssumedHitRate: 0.58, MinNetRR: 0.6, CostToTP1MaxPct: 0.40, MinScore: 50, MinEvidenceTrades: 20},
			types.StrategyUltraScalping:    {StrategyID: types.StrategyUltraScalping, AssumedHitRate: 0.60, MinNetRR: 0.5, CostToTP1MaxPct: 0.45, MinScore: 50, MinEvidenceTrades: 20},
			types.StrategyStandardSwing:    {StrategyID: types.StrategyStandardSwing, AssumedHitRate: 0.52, MinNetRR: 1.2, CostToTP1MaxPct: 0.20, MinScore: 52, MinEvidenceTrades: 20},
			types.StrategyTrendSwing:       {StrategyID: types.StrategyTrendSwing, AssumedHitRate: 0.52, MinNetRR: 1.3, CostToTP1MaxPct: 0.20, MinScore: 52, MinEvidenceTrades: 20},
			types.StrategyMarnieFib:        {StrategyID: types.StrategyMarnieFib, AssumedHitRate: 0.54, MinNetRR: 1.0, CostToTP1MaxPct: 0.25, MinScore: 52, MinEvidenceTrades: 20},
		},
	}
}

// opportunityProfile is more permissive for Shadow/Paper canary testing only.
func opportunityProfile() GateProfile {
	p := balancedProfile()
	p.Version = "OPPORTUNITY"
	p.Label = "Opportunity (shadow/paper canary only)"
	p.MaxSoftFailForShadow = 4
	p.Default.MinScore = 45
	p.Default.MinNetRR = 0.5
	p.Strategies[types.StrategyStandardScalping] = StrategyProfile{StrategyID: types.StrategyStandardScalping, AssumedHitRate: 0.58, MinNetRR: 0.4, CostToTP1MaxPct: 0.50, MinScore: 45, MinEvidenceTrades: 10}
	p.Strategies[types.StrategyUltraScalping] = StrategyProfile{StrategyID: types.StrategyUltraScalping, AssumedHitRate: 0.60, MinNetRR: 0.3, CostToTP1MaxPct: 0.55, MinScore: 45, MinEvidenceTrades: 10}
	return p
}

// profileRegistry is the set of selectable profiles, keyed by Version (upper).
var profileRegistry = map[string]GateProfile{
	"CONSERVATIVE": conservativeProfile(),
	"BALANCED":     balancedProfile(),
	"OPPORTUNITY":  opportunityProfile(),
}

// CurrentGateProfile is the active profile, selected once at startup from the
// GATE_PROFILE_VERSION env var (default BALANCED). Rollback is a one-line env
// change + restart — no recompile.
var CurrentGateProfile = LoadGateProfile(os.Getenv("GATE_PROFILE_VERSION"))

// LoadGateProfile resolves a profile by version string (case-insensitive).
// Unknown/empty → BALANCED (the recommended production default).
func LoadGateProfile(version string) GateProfile {
	v := strings.TrimSpace(strings.ToUpper(version))
	if v == "" {
		return profileRegistry["BALANCED"]
	}
	if p, ok := profileRegistry[v]; ok {
		return p
	}
	return profileRegistry["BALANCED"]
}

// Strategy returns the per-strategy soft-alpha profile, falling back to Default.
func (p GateProfile) Strategy(id types.StrategyID) StrategyProfile {
	if s, ok := p.Strategies[id]; ok {
		return s
	}
	return p.Default
}
