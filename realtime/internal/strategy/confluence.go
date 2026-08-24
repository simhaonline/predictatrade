// Package strategy implements the four canonical strategy confluence engines.
// SOW Sections 12A-12F: Each strategy must produce DISTINCT versioned behavior.
package strategy

import (
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ConfluenceProfile defines the scoring configuration for a strategy.
// SOW Section 12C
type ConfluenceProfile struct {
	StrategyID                 types.StrategyID
	Version                    string
	MandatoryPillars           []string
	OptionalPillars            []string
	Weights                    map[string]decimal.Decimal
	MinimumScore               decimal.Decimal
	MinimumLongShortSeparation decimal.Decimal
	MinimumConfluenceCount     int
	MaxMissingOptionalWeight   decimal.Decimal
	GradeCeilingByCapability   map[string]types.SignalGrade
}

// ConfluenceInput provides the evidence for scoring.
type ConfluenceInput struct {
	Evidence []types.EvidenceContribution
}

// ConfluenceResult is the output of the confluence engine.
type ConfluenceResult struct {
	LongScore        decimal.Decimal
	ShortScore       decimal.Decimal
	ScoreSeparation  decimal.Decimal
	TotalScore       decimal.Decimal
	PassThreshold    bool
	PassSeparation   bool
	PassMinCount     bool
	MandatoryMet     bool
	Evidence         []types.EvidenceContribution
	ReasonCodes      []types.NoTradeReason
}

// Evaluate runs the deterministic confluence scoring.
// SOW Section 12C: Scoring shall be deterministic from the stored feature snapshot.
func Evaluate(profile ConfluenceProfile, input ConfluenceInput) ConfluenceResult {
	result := ConfluenceResult{
		LongScore:  decimal.Zero,
		ShortScore: decimal.Zero,
		Evidence:   input.Evidence,
	}

	longContribs := decimal.Zero
	shortContribs := decimal.Zero
	confluenceCount := 0
	mandatoryMet := make(map[string]bool)
	missingOptionalWeight := decimal.Zero

	// Initialize mandatory check
	for _, m := range profile.MandatoryPillars {
		mandatoryMet[m] = false
	}

	for _, ev := range input.Evidence {
		weight, ok := profile.Weights[ev.Pillar]
		if !ok {
			// Use the weight from the evidence itself
			weight = ev.Weight
		}

		contrib := ev.NormalizedValue.Mul(weight).Div(decimal.NewFromInt(100))
		ev.Contribution = contrib
		ev.Weight = weight

		if ev.Direction == types.DirectionBuy {
			longContribs = longContribs.Add(contrib)
		} else if ev.Direction == types.DirectionSell {
			shortContribs = shortContribs.Add(contrib)
		}

		if ev.Contribution.GreaterThan(decimal.Zero) {
			confluenceCount++
		}

		// Check mandatory pillars
		if ev.Mandatory && ev.Quality != types.QualityUnavailable && ev.Quality != types.QualityInvalid {
			mandatoryMet[ev.Pillar] = true
		}

		// Track missing optional
		if !ev.Mandatory {
			isOptional := false
			for _, opt := range profile.OptionalPillars {
				if opt == ev.Pillar {
					isOptional = true
					break
				}
			}
			if isOptional && (ev.Quality == types.QualityUnavailable || ev.Quality == types.QualityInvalid) {
				missingOptionalWeight = missingOptionalWeight.Add(weight)
			}
		}
	}

	result.LongScore = longContribs
	result.ShortScore = shortContribs
	result.ScoreSeparation = longContribs.Sub(shortContribs).Abs()
	result.TotalScore = longContribs.Add(shortContribs)
	result.PassThreshold = result.TotalScore.GreaterThanOrEqual(profile.MinimumScore)
	result.PassSeparation = result.ScoreSeparation.GreaterThanOrEqual(profile.MinimumLongShortSeparation)
	result.PassMinCount = confluenceCount >= profile.MinimumConfluenceCount

	// Check all mandatory pillars met
	allMandatoryMet := true
	for _, met := range mandatoryMet {
		if !met {
			allMandatoryMet = false
			break
		}
	}
	result.MandatoryMet = allMandatoryMet

	// Generate reason codes
	if !result.PassThreshold {
		result.ReasonCodes = append(result.ReasonCodes, types.NTInsufficientScore)
	}
	if !result.PassSeparation {
		result.ReasonCodes = append(result.ReasonCodes, types.NTInsufficientScore)
	}
	if !result.PassMinCount {
		result.ReasonCodes = append(result.ReasonCodes, types.NTInsufficientScore)
	}
	if !allMandatoryMet {
		result.ReasonCodes = append(result.ReasonCodes, types.NTUnclearStructure)
	}
	if missingOptionalWeight.GreaterThan(profile.MaxMissingOptionalWeight) {
		result.ReasonCodes = append(result.ReasonCodes, types.NTSystemDegraded)
	}

	return result
}

// Direction determines the final direction from the confluence result.
// SOW Section 16: Final decision considers score, separation, and all evidence.
func Direction(result ConfluenceResult) types.Direction {
	if !result.PassThreshold || !result.PassSeparation || !result.MandatoryMet {
		return types.DirectionNoTrade
	}
	if result.LongScore.GreaterThan(result.ShortScore) {
		return types.DirectionBuy
	}
	if result.ShortScore.GreaterThan(result.LongScore) {
		return types.DirectionSell
	}
	return types.DirectionNoTrade
}

// SeedProfiles returns the initial seed confluence profiles for all four strategies.
// SOW Section 12C.1 — these are implementation baselines to be validated and versioned.
func SeedProfiles() map[types.StrategyID]ConfluenceProfile {
	return map[types.StrategyID]ConfluenceProfile{
		// STANDARD_SCALPING: threshold 75, separation 20
		types.StrategyStandardScalping: {
			StrategyID: types.StrategyStandardScalping,
			Version:    "1.0.0",
			MandatoryPillars: []string{"liquidity", "structure"},
			OptionalPillars:  []string{"fvg_ob", "flow_volume", "regime_volatility", "macro_news"},
			Weights: map[string]decimal.Decimal{
				"liquidity":          decimal.NewFromInt(25),
				"structure":          decimal.NewFromInt(20),
				"fvg_ob":             decimal.NewFromInt(15),
				"flow_volume":        decimal.NewFromInt(20),
				"regime_volatility":  decimal.NewFromInt(10),
				"macro_news":         decimal.NewFromInt(10),
			},
			MinimumScore:               decimal.NewFromInt(75),
			MinimumLongShortSeparation: decimal.NewFromInt(20),
			MinimumConfluenceCount:     3,
			MaxMissingOptionalWeight:   decimal.NewFromInt(15),
			GradeCeilingByCapability:   map[string]types.SignalGrade{},
		},
		// ULTRA_SCALPING: threshold 85, separation 25
		types.StrategyUltraScalping: {
			StrategyID: types.StrategyUltraScalping,
			Version:    "1.0.0",
			MandatoryPillars: []string{"flow_microstructure", "liquidity_event", "execution_cost_quality"},
			OptionalPillars:  []string{"structure_mtf", "imbalance_fvg_vwap", "macro_news"},
			Weights: map[string]decimal.Decimal{
				"flow_microstructure":      decimal.NewFromInt(30),
				"liquidity_event":          decimal.NewFromInt(25),
				"structure_mtf":            decimal.NewFromInt(15),
				"imbalance_fvg_vwap":       decimal.NewFromInt(15),
				"execution_cost_quality":   decimal.NewFromInt(10),
				"macro_news":               decimal.NewFromInt(5),
			},
			MinimumScore:               decimal.NewFromInt(85),
			MinimumLongShortSeparation: decimal.NewFromInt(25),
			MinimumConfluenceCount:     3,
			MaxMissingOptionalWeight:   decimal.NewFromInt(10),
			GradeCeilingByCapability:   map[string]types.SignalGrade{},
		},
		// STANDARD_SWING: threshold 70, separation 15
		types.StrategyStandardSwing: {
			StrategyID: types.StrategyStandardSwing,
			Version:    "1.0.0",
			MandatoryPillars: []string{"d1_h4_structure"},
			OptionalPillars:  []string{"macro_dxy_yield", "htf_liquidity", "mtf_alignment", "volume_profile_flow", "regime_volatility", "rr_carry_cost"},
			Weights: map[string]decimal.Decimal{
				"d1_h4_structure":    decimal.NewFromInt(20),
				"htf_liquidity":      decimal.NewFromInt(15),
				"macro_dxy_yield":    decimal.NewFromInt(20),
				"mtf_alignment":      decimal.NewFromInt(15),
				"volume_profile_flow": decimal.NewFromInt(10),
				"regime_volatility":  decimal.NewFromInt(10),
				"rr_carry_cost":      decimal.NewFromInt(10),
			},
			MinimumScore:               decimal.NewFromInt(70),
			MinimumLongShortSeparation: decimal.NewFromInt(15),
			MinimumConfluenceCount:     3,
			MaxMissingOptionalWeight:   decimal.NewFromInt(20),
			GradeCeilingByCapability:   map[string]types.SignalGrade{},
		},
		// TREND_SWING: threshold 75, separation 15
		types.StrategyTrendSwing: {
			StrategyID: types.StrategyTrendSwing,
			Version:    "1.0.0",
			MandatoryPillars: []string{"w1_d1_h4_trend_structure"},
			OptionalPillars:  []string{"macro_real_yield_dxy", "cot_etf_flow", "mtf_alignment", "major_liquidity_htf_profile", "trend_persistence_volatility", "carry_execution_cost"},
			Weights: map[string]decimal.Decimal{
				"w1_d1_h4_trend_structure":  decimal.NewFromInt(25),
				"macro_real_yield_dxy":      decimal.NewFromInt(20),
				"cot_etf_flow":              decimal.NewFromInt(15),
				"mtf_alignment":             decimal.NewFromInt(15),
				"major_liquidity_htf_profile": decimal.NewFromInt(10),
				"trend_persistence_volatility": decimal.NewFromInt(10),
				"carry_execution_cost":      decimal.NewFromInt(5),
			},
			MinimumScore:               decimal.NewFromInt(75),
			MinimumLongShortSeparation: decimal.NewFromInt(15),
			MinimumConfluenceCount:     3,
			MaxMissingOptionalWeight:   decimal.NewFromInt(15),
			GradeCeilingByCapability:   map[string]types.SignalGrade{},
		},
	}
}

// SeedRiskProfiles returns initial seed risk profiles for all four strategies.
// SOW Section 25A.1 — conservative starting configurations.
func SeedRiskProfiles() map[types.StrategyID]RiskProfile {
	return map[types.StrategyID]RiskProfile{
		types.StrategyStandardScalping: {
			MinGrossRR:          decimal.NewFromFloat(1.20),
			MaxSpreadAbsolute:   decimal.NewFromFloat(0.35),
			MaxSpreadToATR:      decimal.NewFromFloat(0.50),
			MaxNewTradesPerDay:  3,
			LossCooldownMinutes: 30,
			MaxTotalCostToTarget: decimal.NewFromFloat(0.25),
		},
		types.StrategyUltraScalping: {
			MinGrossRR:          decimal.NewFromFloat(1.00),
			MaxSpreadAbsolute:   decimal.NewFromFloat(0.25),
			MaxSpreadToATR:      decimal.NewFromFloat(0.40),
			MaxNewTradesPerDay:  5,
			LossCooldownMinutes: 30,
			MaxTotalCostToTarget: decimal.NewFromFloat(0.20),
		},
		types.StrategyStandardSwing: {
			MinGrossRR:          decimal.NewFromFloat(1.80),
			MaxSpreadAbsolute:   decimal.NewFromFloat(0.45),
			MaxSpreadToATR:      decimal.NewFromFloat(0.60),
			MaxNewTradesPerDay:  2,
			LossCooldownMinutes: 120,
			MaxTotalCostToTarget: decimal.NewFromFloat(0.25),
		},
		types.StrategyTrendSwing: {
			MinGrossRR:          decimal.NewFromFloat(2.50),
			MaxSpreadAbsolute:   decimal.NewFromFloat(0.50),
			MaxSpreadToATR:      decimal.NewFromFloat(0.70),
			MaxNewTradesPerDay:  1,
			LossCooldownMinutes: 240,
			MaxTotalCostToTarget: decimal.NewFromFloat(0.25),
		},
		types.StrategyMarnieFib: {
			MinGrossRR:          decimal.NewFromFloat(2.00),
			MaxSpreadAbsolute:   decimal.NewFromFloat(0.45),
			MaxSpreadToATR:      decimal.NewFromFloat(0.55),
			MaxNewTradesPerDay:  2,
			LossCooldownMinutes: 180,
			MaxTotalCostToTarget: decimal.NewFromFloat(0.25),
		},
	}
}

// RiskProfile defines strategy-specific risk parameters.
// SOW Section 25A
type RiskProfile struct {
	MinGrossRR           decimal.Decimal
	MaxSpreadAbsolute    decimal.Decimal
	MaxSpreadToATR       decimal.Decimal
	MaxExpectedSlippage  decimal.Decimal
	MaxTotalCostToTarget decimal.Decimal
	MinMarginHeadroom    decimal.Decimal
	MaxNewTradesPerDay   int
	LossCooldownMinutes  int
	MaxPositions         int
	MaxDailyLoss         decimal.Decimal
	MaxDrawdown          decimal.Decimal
}
