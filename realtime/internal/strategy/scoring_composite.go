// scoring_composite.go — P1 (prompt.md): 8-family signed-sum composite with
// per-family diagnostics, the reference tier ladder, and RegimeAllows.
//
// NON-OVERLAP GUARANTEE (the reason this file exists in this shape):
// the engine's canonical scoring path is confluence.go Evaluate() — weighted
// pillar contributions with mandatory/optional logic, family caps
// (applyFamilyCaps) and per-strategy thresholds. That path is UNCHANGED.
// This file ADDS a pure, read-only diagnostics layer over the same
// []EvidenceContribution rows:
//
//	FamilySubScores — signed per-family sums (TREND/MTF/MOMENTUM/VOLUME/
//	                   SMC/GEOMETRY/CANDLE/MACRO)
//	FamilyComposite — Σ families × 1.5, clamped ±100 (reference formula)
//	ReferenceTier   — A+/A/B/C/WATCH ladder (55/42/30/22/12), flag-gated
//	RegimeAllowsDirection — trending regimes admit the aligned direction
//	                         only; squeeze admits nothing; others both.
//
// Nothing here mutates evidence, scores, or delivery. Consumers opt in via
// REFERENCE_TIER_LADDER (tier ladder) and the diagnostics payload; the
// decision path keeps using ComputeQualityGrade until the operator flips.
package strategy

import (
	"os"
	"strings"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// FamilySubScores holds the signed per-family diagnostic sums.
type FamilySubScores struct {
	Trend    decimal.Decimal
	MTF      decimal.Decimal
	Momentum decimal.Decimal
	Volume   decimal.Decimal
	SMC      decimal.Decimal
	Geometry decimal.Decimal
	Candle   decimal.Decimal
	Macro    decimal.Decimal
}

// familyPillars maps evidence pillars → composite families. Pillars that
// don't belong to the 8 reference families (REGIME, ML, SENTIMENT,
// SESSION_ORB, WESTERN_ASTRO…) are excluded from the composite — they keep
// their role in the canonical confluence path.
var familyPillars = map[string]string{
	"TREND":    "TREND",
	"MTF":      "MTF",
	"MOMENTUM": "MOMENTUM",
	"VOLUME":   "VOLUME",
	"SMC":      "SMC",
	"GEOMETRY": "GEOMETRY",
	"CANDLE":   "CANDLE",
	"MACRO":    "MACRO",
	// aliases used by the live engines (map to the same families, no overlap):
	"VOLATILITY":  "GEOMETRY", // BB/ATR-state reads feed the geometry family
	"STRUCTURE":   "SMC",      // BOS/CHoCH are SMC family per the reference
	"LIQUIDITY":   "SMC",
	"WESTERN_ASTRO": "MACRO",
	"DI_DASHA":      "MACRO",
	"DI_HORA":       "MACRO",
	"DI_NAKSHATRA":  "MACRO",
	"CROSS_MARKET":  "MACRO",
}

// FamilySubScores aggregates evidence rows into signed per-family sums.
// The sign follows the evidence direction (BUY positive, SELL negative) —
// matching the reference's signed family scores.
func ComputeFamilySubScores(evidence []types.EvidenceContribution) FamilySubScores {
	var fs FamilySubScores
	for _, ev := range evidence {
		fam, ok := familyPillars[ev.Pillar]
		if !ok {
			continue
		}
		v := ev.Contribution
		if ev.Direction == types.DirectionSell && v.GreaterThan(decimal.Zero) {
			v = v.Neg()
		}
		switch fam {
		case "TREND":
			fs.Trend = fs.Trend.Add(v)
		case "MTF":
			fs.MTF = fs.MTF.Add(v)
		case "MOMENTUM":
			fs.Momentum = fs.Momentum.Add(v)
		case "VOLUME":
			fs.Volume = fs.Volume.Add(v)
		case "SMC":
			fs.SMC = fs.SMC.Add(v)
		case "GEOMETRY":
			fs.Geometry = fs.Geometry.Add(v)
		case "CANDLE":
			fs.Candle = fs.Candle.Add(v)
		case "MACRO":
			fs.Macro = fs.Macro.Add(v)
		}
	}
	return fs
}

// CompositeMultiplier is the reference scaling factor (×1.5) applied at
// composite level before the ±100 clamp (prompt.md P1: "apply at composite
// level before clamp").
const CompositeMultiplier = 1.5

// FamilyComposite returns the signed-sum composite of the family sub-scores,
// scaled ×1.5 and clamped to ±100. Deterministic, pure.
func FamilyComposite(fs FamilySubScores) decimal.Decimal {
	total := fs.Trend.Add(fs.MTF).Add(fs.Momentum).Add(fs.Volume).
		Add(fs.SMC).Add(fs.Geometry).Add(fs.Candle).Add(fs.Macro)
	scaled := total.Mul(decimal.NewFromFloat(CompositeMultiplier))
	hundred := decimal.NewFromInt(100)
	if scaled.GreaterThan(hundred) {
		return hundred
	}
	if scaled.LessThan(hundred.Neg()) {
		return hundred.Neg()
	}
	return scaled
}

// ReferenceTier implements the prompt.md P1 ladder: A+ 55, A 42, B 30,
// C 22, Watch 12. Scores below the Watch floor return "" (no tier).
// Returns "" when the reference ladder is not enabled (feature flag
// REFERENCE_TIER_LADDER) — callers then keep ComputeQualityGrade.
func ReferenceTier(score float64) string {
	if !referenceTierLadderEnabled() {
		return ""
	}
	switch {
	case score >= 55:
		return string(types.GradeAPlus)
	case score >= 42:
		return string(types.GradeA)
	case score >= 30:
		return "B"
	case score >= 22:
		return "C"
	case score >= 12:
		return "WATCH"
	default:
		return ""
	}
}

// referenceTierLadderEnabled reads the operator feature flag once per call —
// env-driven so rollout risk stays with the operator (prompt.md acceptance).
func referenceTierLadderEnabled() bool {
	return strings.EqualFold(strings.TrimSpace(os.Getenv("REFERENCE_TIER_LADDER")), "true")
}

// ReferenceTierLadderEnabled exposes the flag for the diagnostics payload.
func ReferenceTierLadderEnabled() bool { return referenceTierLadderEnabled() }

// TierReason explains a tier decision: dominant positive family, dominant
// negative family, and the composite. Diagnostics-only; no gate behavior.
func TierReason(fs FamilySubScores, comp decimal.Decimal) string {
	type kv struct {
		name string
		v    decimal.Decimal
	}
	all := []kv{
		{"TREND", fs.Trend}, {"MTF", fs.MTF}, {"MOMENTUM", fs.Momentum},
		{"VOLUME", fs.Volume}, {"SMC", fs.SMC}, {"GEOMETRY", fs.Geometry},
		{"CANDLE", fs.Candle}, {"MACRO", fs.Macro},
	}
	best, worst := all[0], all[0]
	for _, x := range all[1:] {
		if x.v.GreaterThan(best.v) {
			best = x
		}
		if x.v.LessThan(worst.v) {
			worst = x
		}
	}
	cf, _ := comp.Float64()
	return "composite=" + strings.TrimRight(strings.TrimRight(
		formatFloat(cf), "0"), ".") +
		" dominant=" + best.name + " opposing=" + worst.name
}

func formatFloat(f float64) string {
	return strings.TrimSpace(strings.Replace(
		strings.Replace( // avoid importing strconv for one call
			decimal.NewFromFloat(f).StringFixed(2), "", "", 1), "", "", 1))
}

// RegimeAllowsDirection implements the reference RegimeAllows():
//   - TRENDING_BULLISH admits BUY only
//   - TRENDING_BEARISH admits SELL only
//   - RANGE (the reference's "squeeze") admits NOTHING — the squeeze veto
//   - all other regimes (BREAKOUT, HIGH_VOLATILITY, …) admit both directions
//
// This is the gate-level helper; the actual gate is RegimeDirectionGate
// (gates package) so the check is fail-closed, ordered, and per-(strategy,
// timeframe) isolated like every other hard gate.
func RegimeAllowsDirection(regime types.Regime, dir types.Direction) bool {
	if dir != types.DirectionBuy && dir != types.DirectionSell {
		return false
	}
	switch regime {
	case types.RegimeTrendingBullish:
		return dir == types.DirectionBuy
	case types.RegimeTrendingBearish:
		return dir == types.DirectionSell
	case types.RegimeRange:
		return false // reference squeeze: no trades while compressed
	default:
		return true
	}
}