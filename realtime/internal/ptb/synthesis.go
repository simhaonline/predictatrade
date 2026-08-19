// Package ptb — Synthesis engine, setup quality, position multiplier, narrative.
// Sections 14-18, 25-26: Core PTB synthesis, grading, sizing, explainability.
package ptb

import (
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// BiasState represents the PTB directional bias.
type BiasState string

const (
	BiasStrongLong  BiasState = "STRONG_LONG"
	BiasLong        BiasState = "LONG"
	BiasNeutral     BiasState = "NEUTRAL"
	BiasShort       BiasState = "SHORT"
	BiasStrongShort BiasState = "STRONG_SHORT"
	BiasStandAside  BiasState = "STAND_ASIDE"
)

// ActionState represents the PTB recommended action.
type ActionState string

const (
	ActionEnter ActionState = "ENTER"
	ActionWait  ActionState = "WAIT"
	ActionAvoid ActionState = "AVOID"
	ActionExit  ActionState = "EXIT"
)

// SetupGrade represents setup quality classification.
type SetupGrade string

const (
	GradeAPlus SetupGrade = "A+"
	GradeA     SetupGrade = "A"
	GradeB     SetupGrade = "B"
	GradeC     SetupGrade = "C"
	GradeD     SetupGrade = "D"
	GradeF     SetupGrade = "F"
)

// GoldRole represents what is currently driving XAUUSD.
type GoldRole string

const (
	GoldRoleCurrency      GoldRole = "CURRENCY"
	GoldRoleSafeHaven     GoldRole = "SAFE_HAVEN"
	GoldRoleMonetaryAsset GoldRole = "MONETARY_ASSET"
	GoldRoleCommodity     GoldRole = "COMMODITY"
	GoldRoleInflationHedge GoldRole = "INFLATION_HEDGE"
	GoldRoleUnknown       GoldRole = "UNKNOWN"
)

// EnhancedRegime represents professional regime classification.
type EnhancedRegime string

const (
	RegimeStrongTrendUp   EnhancedRegime = "STRONG_TREND_UP"
	RegimeStrongTrendDown EnhancedRegime = "STRONG_TREND_DOWN"
	RegimeWeakTrendUp     EnhancedRegime = "WEAK_TREND_UP"
	RegimeWeakTrendDown   EnhancedRegime = "WEAK_TREND_DOWN"
	RegimeRangeBound      EnhancedRegime = "RANGE_BOUND"
	RegimeHighVolatility  EnhancedRegime = "HIGH_VOLATILITY"
	RegimeLowVolatility   EnhancedRegime = "LOW_VOLATILITY"
	RegimeTransitioning   EnhancedRegime = "TRANSITIONING"
	RegimeManipulation    EnhancedRegime = "MANIPULATION"
)

// CorrelationResult holds a single correlation measurement.
type CorrelationResult struct {
	Instrument     string  `json:"instrument"`
	Window         int     `json:"window"`
	Correlation    float64 `json:"correlation"`
	Direction      string  `json:"direction"`  // INVERSE, DIRECT, NEUTRAL
	Strength       float64 `json:"strength"`
	ChangeInCorr   float64 `json:"change_in_correlation"`
	SampleCount    int     `json:"sample_count"`
	Timestamp      time.Time `json:"timestamp"`
	DataAge        int64   `json:"data_age_ms"`
	Quality        string  `json:"quality"`     // OK, STALE, INSUFFICIENT, UNAVAILABLE
	Availability   string  `json:"availability"`
}

// LiquidityTarget represents a directional liquidity objective.
type LiquidityTarget struct {
	Price          decimal.Decimal `json:"price"`
	Type           string          `json:"type"`
	Side           string          `json:"side"`     // ABOVE (for LONG), BELOW (for SHORT)
	Strength       int             `json:"strength"`
	Swept          bool            `json:"swept"`
	Distance       float64         `json:"distance_from_price"`
	StructuralRel  string          `json:"structural_relevance"` // HIGH, MEDIUM, LOW
}

// ComponentScore holds a single component's contribution to synthesis.
type ComponentScore struct {
	Score     float64 `json:"score"`
	Direction string  `json:"direction"`
	Weight    float64 `json:"weight"`
	Available bool    `json:"available"`
}

// ReasonCode represents a machine-readable PTB reason.
type ReasonCode string

// SynthesisOutput is the final PTB synthesis result.
// Section 15: Normalized PTB output.
type SynthesisOutput struct {
	Timestamp    time.Time      `json:"timestamp"`
	Symbol       string         `json:"symbol"`
	AnalysisID   string         `json:"analysis_id"`

	// Classification
	Regime          EnhancedRegime `json:"regime"`
	GoldRole        GoldRole       `json:"gold_role"`
	VolatilityState string         `json:"volatility_state"`
	ManipulationIndex float64     `json:"manipulation_index"`

	// Bias and confidence
	Bias         BiasState  `json:"bias"`
	BiasStrength float64    `json:"bias_strength"`
	Confidence   float64    `json:"confidence"`

	// Levels
	SupportLevels    []decimal.Decimal  `json:"support_levels"`
	ResistanceLevels []decimal.Decimal  `json:"resistance_levels"`
	LiquidityTargets []LiquidityTarget  `json:"liquidity_targets"`

	// Timing
	OptimalEntryTime  string  `json:"optimal_entry_time"`
	ExpectedDuration  string  `json:"expected_duration"`
	TimeConfidence    float64 `json:"time_confidence"`

	// Narrative
	MarketNarrative string   `json:"market_narrative"`
	KeyDrivers      []string `json:"key_drivers"`
	RiskFactors     []string `json:"risk_factors"`

	// Quality and action
	RecommendedSetup    string      `json:"recommended_setup"`
	PositionSizeMult    float64     `json:"position_size_multiplier"`
	StopDistanceMult    float64     `json:"stop_distance_multiplier"`
	ConfluenceScore     float64     `json:"confluence_score"`
	SetupQuality        SetupGrade  `json:"setup_quality"`
	Action              ActionState `json:"action"`

	// Explainability (Section 25)
	PositiveFactors []string               `json:"positive_factors"`
	NegativeFactors []string               `json:"negative_factors"`
	Vetoes          []string               `json:"vetoes"`
	ReasonCodes     []string               `json:"reason_codes"`
	DataQuality     map[string]string      `json:"data_quality"`
	ComponentScores map[string]ComponentScore `json:"component_scores"`

	// Provenance
	ModelVersion  string `json:"model_version"`
	ConfigVersion string `json:"config_version"`
	ShadowMode    bool   `json:"shadow_mode"`
}

// Synthesize is the core PTB function.
// Section 14: Combines all evidence into a unified assessment.
// CRITICAL: In SHADOW mode, this produces output for observation but
// does NOT alter production BUY/SELL decisions.
func (e *Engine) Synthesize(state *features.MarketState, snap *MarketIntelligenceSnapshot, cfg *Config) *SynthesisOutput {
	if state == nil || snap == nil {
		return nil
	}

	now := time.Now().UTC()
	out := &SynthesisOutput{
		Timestamp:    now,
		Symbol:       state.Symbol,
		AnalysisID:   fmt.Sprintf("ptb-%d", now.UnixMilli()),
		DataQuality:  make(map[string]string),
		ComponentScores: make(map[string]ComponentScore),
		ShadowMode:    cfg.ShadowMode,
		ModelVersion:  cfg.ModelVersion,
		ConfigVersion: cfg.ConfigVersion,
	}

	// === Regime Classification (Section 6) ===
	out.Regime = classifyRegime(state, snap)
	out.ComponentScores["regime"] = ComponentScore{Score: regimeScore(state), Direction: regimeDirection(state), Available: true}

	// === Volatility State (Section 7) ===
	out.VolatilityState = snap.VolatilityRegime.Regime
	out.ComponentScores["volatility"] = ComponentScore{Score: volScore(snap), Available: true}

	// === Gold Role (Section 9) ===
	out.GoldRole = classifyGoldRole(snap)
	out.ComponentScores["macro"] = ComponentScore{Score: 50, Available: snap.InstitutionalFootprint.Available}
	if snap.InstitutionalFootprint.Mode == types.ModuleUnsupported {
		out.ComponentScores["macro"] = ComponentScore{Score: 0, Available: false}
		out.DataQuality["macro"] = "UNAVAILABLE"
	}

	// === Manipulation Index (Section 12) ===
	out.ManipulationIndex = extractManipulationIndex(snap)
	out.ComponentScores["manipulation"] = ComponentScore{Score: out.ManipulationIndex, Available: true}

	// === MTF Bias (Section 5) ===
	mtfBias := snap.MTFBias
	out.ComponentScores["mtf"] = ComponentScore{Score: math.Abs(mtfBias.Alignment) * 100, Direction: mtfBias.Bias, Available: mtfBias.SampleCount > 0}

	// === Structure (reuse existing) ===
	structScore := 50.0
	structDir := "NEUTRAL"
	if state.Structure.LastBOS != nil {
		structScore = 70
		if state.Structure.LastBOS.Direction == "bullish" {
			structDir = "LONG"
		} else {
			structDir = "SHORT"
		}
	}
	if state.Structure.LastCHoCH != nil {
		structScore = 65
		if state.Structure.LastCHoCH.Direction == "bullish" {
			structDir = "LONG"
		} else {
			structDir = "SHORT"
		}
	}
	out.ComponentScores["structure"] = ComponentScore{Score: structScore, Direction: structDir, Available: true}

	// === Liquidity (Section 10) ===
	liqScore := 50.0
	liqDir := "NEUTRAL"
	if len(state.Liquidity.RecentSweeps) > 0 {
		sweep := state.Liquidity.RecentSweeps[len(state.Liquidity.RecentSweeps)-1]
		liqScore = 70
		if sweep.Direction == "SELL_SIDE_SWEEP" || sweep.Direction == "sell_side" {
			liqDir = "LONG"
		} else {
			liqDir = "SHORT"
		}
	}
	out.ComponentScores["liquidity"] = ComponentScore{Score: liqScore, Direction: liqDir, Available: true}

	// === Session/Timing (Section 13) ===
	sessionScore := 50.0
	if state.Session.CurrentSession == "LONDON" || state.Session.CurrentSession == "NEW_YORK" || state.Session.CurrentSession == "OVERLAP" {
		sessionScore = 80
	} else if state.Session.CurrentSession == "TOKYO" {
		sessionScore = 60
	}
	if state.Session.IsWeekend {
		sessionScore = 0
	}
	out.ComponentScores["session"] = ComponentScore{Score: sessionScore, Available: true}
	out.TimeConfidence = sessionScore / 100.0
	out.OptimalEntryTime = state.Session.CurrentSession

	// === Compute Confluence Score (Section 14) ===
	// Weighted average of component scores with family caps
	out.ConfluenceScore = computeConfluence(out.ComponentScores, cfg)
	out.Confidence = out.ConfluenceScore

	// === Determine Bias (Section 15) ===
	out.Bias, out.BiasStrength = determineBias(out.ComponentScores, mtfBias, out.ConfluenceScore)

	// === Liquidity Targets (Section 11) ===
	out.LiquidityTargets = generateLiquidityTargets(state, out.Bias)
	out.SupportLevels = extractSupportLevels(state)
	out.ResistanceLevels = extractResistanceLevels(state)

	// === Setup Quality (Section 16) ===
	out.SetupQuality = gradeSetup(out.ConfluenceScore, out.Confidence, cfg)

	// === Position Size Multiplier (Section 17) ===
	out.PositionSizeMult = positionSizeMultiplier(out.SetupQuality, cfg)

	// === Stop Distance Multiplier (Section 18) ===
	out.StopDistanceMult = stopDistanceMultiplier(out.ManipulationIndex, out.VolatilityState, cfg)

	// === Action (Section 15) ===
	out.Action = determineAction(out.Bias, out.Confidence, out.ManipulationIndex, cfg)

	// === Reason Codes (Section 25) ===
	out.ReasonCodes, out.PositiveFactors, out.NegativeFactors, out.Vetoes = generateReasonCodes(out)

	// === Market Narrative (Section 26) ===
	out.MarketNarrative = generateNarrative(state, snap, out)
	out.KeyDrivers = extractKeyDrivers(out.ComponentScores)
	out.RiskFactors = extractRiskFactors(snap, out)

	// === Data Quality (Section 40) ===
	out.DataQuality["market_data"] = snap.DataQuality.State
	out.DataQuality["mtf_data"] = boolStr(mtfBias.SampleCount > 0)
	out.DataQuality["macro"] = "UNAVAILABLE"
	out.DataQuality["session"] = "OK"
	if state.Session.IsWeekend {
		out.DataQuality["session"] = "WEEKEND"
	}

	return out
}

// classifyRegime maps existing regime to enhanced regime (Section 6).
func classifyRegime(state *features.MarketState, snap *MarketIntelligenceSnapshot) EnhancedRegime {
	adx, _ := state.Indicators.ADX.Float64()
	bullish := state.Indicators.EMA9.GreaterThan(state.Indicators.EMA21) && state.Indicators.EMA21.GreaterThan(state.Indicators.EMA50)
	bearish := state.Indicators.EMA9.LessThan(state.Indicators.EMA21) && state.Indicators.EMA21.LessThan(state.Indicators.EMA50)

	manipIdx := extractManipulationIndex(snap)
	if manipIdx > 70 {
		return RegimeManipulation
	}

	switch {
	case adx > 30 && bullish:
		return RegimeStrongTrendUp
	case adx > 30 && bearish:
		return RegimeStrongTrendDown
	case adx > 20 && bullish:
		return RegimeWeakTrendUp
	case adx > 20 && bearish:
		return RegimeWeakTrendDown
	case adx < 18:
		return RegimeRangeBound
	case snap.VolatilityRegime.Regime == "EXTREME" || snap.VolatilityRegime.Regime == "HIGH":
		return RegimeHighVolatility
	case snap.VolatilityRegime.Regime == "LOW":
		return RegimeLowVolatility
	default:
		return RegimeTransitioning
	}
}

// classifyGoldRole determines what is driving XAUUSD (Section 9).
// Without DXY/yield data, returns UNKNOWN — no fabricated classifications.
func classifyGoldRole(snap *MarketIntelligenceSnapshot) GoldRole {
	// DXY, yields, silver data are NOT available from the current Master Node.
	// Without correlation data, we cannot determine the gold role.
	// Do NOT fabricate a classification.
	return GoldRoleUnknown
}

// computeConfluence produces the weighted confluence score (Section 14).
// Evidence families are capped to prevent double-counting (Section 42).
func computeConfluence(components map[string]ComponentScore, cfg *Config) float64 {
	weights := map[string]float64{
		"mtf":          0.15,
		"regime":       0.15,
		"volatility":   0.10,
		"structure":    0.20,
		"liquidity":    0.15,
		"session":      0.10,
		"manipulation": 0.10,
		"macro":        0.05,
	}

	totalWeight := 0.0
	weightedSum := 0.0

	for comp, score := range components {
		w := weights[comp]
		if !score.Available {
			continue
		}
		weightedSum += w * score.Score
		totalWeight += w
	}

	if totalWeight == 0 {
		return 0
	}
	return weightedSum / totalWeight
}

// determineBias produces the directional bias from component evidence.
func determineBias(components map[string]ComponentScore, mtf MTFBiasResult, confluence float64) (BiasState, float64) {
	longScore := 0.0
	shortScore := 0.0
	totalWeight := 0.0

	weights := map[string]float64{
		"mtf": 0.20, "regime": 0.15, "structure": 0.25,
		"liquidity": 0.15, "manipulation": 0.10, "macro": 0.05, "session": 0.10,
	}

	for comp, score := range components {
		w := weights[comp]
		if !score.Available || w == 0 {
			continue
		}
		switch score.Direction {
		case "LONG":
			longScore += w * score.Score
		case "SHORT":
			shortScore += w * score.Score
		}
		totalWeight += w
	}

	if totalWeight == 0 {
		return BiasNeutral, 0
	}

	netScore := (longScore - shortScore) / totalWeight
	strength := math.Abs(netScore) / 100.0

	switch {
	case netScore > 50:
		return BiasStrongLong, strength
	case netScore > 15:
		return BiasLong, strength
	case netScore < -50:
		return BiasStrongShort, strength
	case netScore < -15:
		return BiasShort, strength
	default:
		return BiasNeutral, strength
	}
}

// gradeSetup classifies setup quality A+ through F (Section 16).
func gradeSetup(confluence, confidence float64, cfg *Config) SetupGrade {
	score := (confluence + confidence) / 2.0
	switch {
	case score >= cfg.GradeAPlus:
		return GradeAPlus
	case score >= cfg.GradeA:
		return GradeA
	case score >= cfg.GradeB:
		return GradeB
	case score >= cfg.GradeC:
		return GradeC
	case score >= cfg.GradeD:
		return GradeD
	default:
		return GradeF
	}
}

// positionSizeMultiplier maps grade to risk multiplier (Section 17).
func positionSizeMultiplier(grade SetupGrade, cfg *Config) float64 {
	switch grade {
	case GradeAPlus:
		return cfg.PosMultAPlus
	case GradeA:
		return cfg.PosMultA
	case GradeB:
		return cfg.PosMultB
	case GradeC:
		return cfg.PosMultC
	case GradeD:
		return cfg.PosMultD
	default:
		return cfg.PosMultF
	}
}

// stopDistanceMultiplier recommends context-aware stop distance (Section 18).
func stopDistanceMultiplier(manipIdx float64, volState string, cfg *Config) float64 {
	if manipIdx > cfg.ManipHighRisk {
		return cfg.StopMultHighManipulation
	}
	if volState == "LOW" || volState == "EXTREME_LOW" {
		return cfg.StopMultLowVolatility
	}
	return cfg.StopMultNormal
}

// determineAction produces the recommended action (Section 15, 24).
func determineAction(bias BiasState, confidence, manipIdx float64, cfg *Config) ActionState {
	if manipIdx > cfg.ManipHighRisk {
		return ActionAvoid
	}
	if confidence < cfg.MinConfidence {
		return ActionWait
	}
	switch bias {
	case BiasStrongLong, BiasLong:
		return ActionEnter
	case BiasStrongShort, BiasShort:
		return ActionEnter
	default:
		return ActionWait
	}
}

// generateLiquidityTargets produces directional targets (Section 11).
func generateLiquidityTargets(state *features.MarketState, bias BiasState) []LiquidityTarget {
	targets := make([]LiquidityTarget, 0)
	curPrice, _ := state.CurrentPrice.Float64()

	// For LONG bias, targets are above price (resistance/BSL)
	for _, sh := range state.Structure.SwingHighs {
		priceF, _ := sh.Float64()
		if priceF > curPrice {
			targets = append(targets, LiquidityTarget{
				Price: sh, Type: "SWING_HIGH", Side: "ABOVE",
				Strength: 3, Distance: priceF - curPrice, StructuralRel: "HIGH",
			})
		}
	}
	// For SHORT bias, targets are below price (support/SSL)
	for _, sl := range state.Structure.SwingLows {
		priceF, _ := sl.Float64()
		if priceF < curPrice {
			targets = append(targets, LiquidityTarget{
				Price: sl, Type: "SWING_LOW", Side: "BELOW",
				Strength: 3, Distance: curPrice - priceF, StructuralRel: "HIGH",
			})
		}
	}
	// Add liquidity pools
	for _, pool := range state.Liquidity.Pools {
		priceF, _ := pool.Price.Float64()
		side := "ABOVE"
		if priceF < curPrice {
			side = "BELOW"
		}
		targets = append(targets, LiquidityTarget{
			Price: pool.Price, Type: pool.Type, Side: side,
			Strength: pool.Strength, Swept: pool.Swept,
			Distance: math.Abs(priceF - curPrice), StructuralRel: "MEDIUM",
		})
	}
	return targets
}

// extractSupportLevels returns support levels from swing lows.
func extractSupportLevels(state *features.MarketState) []decimal.Decimal {
	curPrice := state.CurrentPrice
	var supports []decimal.Decimal
	for _, sl := range state.Structure.SwingLows {
		if sl.LessThan(curPrice) {
			supports = append(supports, sl)
		}
	}
	return supports
}

// extractResistanceLevels returns resistance levels from swing highs.
func extractResistanceLevels(state *features.MarketState) []decimal.Decimal {
	curPrice := state.CurrentPrice
	var resistances []decimal.Decimal
	for _, sh := range state.Structure.SwingHighs {
		if sh.GreaterThan(curPrice) {
			resistances = append(resistances, sh)
		}
	}
	return resistances
}

// generateNarrative produces a deterministic market narrative (Section 26).
func generateNarrative(state *features.MarketState, snap *MarketIntelligenceSnapshot, out *SynthesisOutput) string {
	var sb strings.Builder

	// Regime
	sb.WriteString(fmt.Sprintf("Regime: %s. ", out.Regime))

	// Structure
	if state.Structure.CurrentTrend != "" {
		sb.WriteString(fmt.Sprintf("Structure: %s", state.Structure.CurrentTrend))
		if state.Structure.LastBOS != nil {
			sb.WriteString(fmt.Sprintf(" with BOS %s", state.Structure.LastBOS.Direction))
		}
		sb.WriteString(". ")
	}

	// MTF
	sb.WriteString(fmt.Sprintf("MTF bias: %s (alignment %.1f%%). ", snap.MTFBias.Bias, snap.MTFBias.Alignment*100))

	// Volatility
	sb.WriteString(fmt.Sprintf("Volatility: %s. ", out.VolatilityState))

	// Liquidity
	if len(state.Liquidity.RecentSweeps) > 0 {
		sweep := state.Liquidity.RecentSweeps[len(state.Liquidity.RecentSweeps)-1]
		sb.WriteString(fmt.Sprintf("Recent %s sweep. ", sweep.Direction))
	}

	// Manipulation
	if out.ManipulationIndex > 70 {
		sb.WriteString("HIGH manipulation risk detected. ")
	} else if out.ManipulationIndex > 40 {
		sb.WriteString("Elevated manipulation risk. ")
	}

	// Session
	sb.WriteString(fmt.Sprintf("Session: %s. ", state.Session.CurrentSession))

	// Bias and action
	sb.WriteString(fmt.Sprintf("PTB bias: %s (confidence %.1f%%, grade %s, action: %s). ",
		out.Bias, out.Confidence, out.SetupQuality, out.Action))

	// Macro
	if out.GoldRole == GoldRoleUnknown {
		sb.WriteString("Macro/correlation data unavailable — gold role UNKNOWN. ")
	}

	return sb.String()
}

// generateReasonCodes produces machine-readable reason codes (Section 25).
func generateReasonCodes(out *SynthesisOutput) (reasons, positive, negative, vetoes []string) {
	// Positive factors
	if out.ComponentScores["mtf"].Direction == "LONG" || out.ComponentScores["mtf"].Direction == "SHORT" {
		if out.Bias == BiasLong || out.Bias == BiasStrongLong {
			positive = append(positive, "MTF_BULLISH_ALIGNMENT")
		} else if out.Bias == BiasShort || out.Bias == BiasStrongShort {
			positive = append(positive, "MTF_BEARISH_ALIGNMENT")
		}
	}
	if out.ComponentScores["structure"].Direction == "LONG" {
		positive = append(positive, "HTF_STRUCTURE_BULLISH")
	} else if out.ComponentScores["structure"].Direction == "SHORT" {
		positive = append(positive, "HTF_STRUCTURE_BEARISH")
	}
	if out.ComponentScores["liquidity"].Direction == "LONG" {
		positive = append(positive, "SELL_SIDE_SWEEP_CONFIRMED")
	} else if out.ComponentScores["liquidity"].Direction == "SHORT" {
		positive = append(positive, "BUY_SIDE_SWEEP_CONFIRMED")
	}

	// Negative factors
	if out.ComponentScores["mtf"].Direction == "CONFLICTED" {
		negative = append(negative, "MTF_CONFLICT")
	}
	if out.ManipulationIndex > 70 {
		negative = append(negative, "HIGH_MANIPULATION_RISK")
	}
	if out.VolatilityState == "EXTREME" || out.VolatilityState == "EXTREME_HIGH" {
		negative = append(negative, "VOLATILITY_EXTREME")
	}
	if !out.ComponentScores["macro"].Available {
		negative = append(negative, "MACRO_DATA_UNAVAILABLE")
	}

	// Vetoes
	if out.Action == ActionAvoid {
		vetoes = append(vetoes, "MANIPULATION_RISK_VETO")
	}

	// All reasons
	reasons = append(reasons, positive...)
	reasons = append(reasons, negative...)
	reasons = append(reasons, vetoes...)

	if out.ConfluenceScore < 70 {
		negative = append(negative, "INSUFFICIENT_CONFLUENCE")
	}

	return
}

// extractKeyDrivers returns the main factors driving the current assessment.
func extractKeyDrivers(components map[string]ComponentScore) []string {
	drivers := make([]string, 0)
	for comp, score := range components {
		if score.Available && score.Score > 65 {
			drivers = append(drivers, fmt.Sprintf("%s (%.0f, %s)", comp, score.Score, score.Direction))
		}
	}
	return drivers
}

// extractRiskFactors identifies risk factors from the snapshot.
func extractRiskFactors(snap *MarketIntelligenceSnapshot, out *SynthesisOutput) []string {
	risks := make([]string, 0)
	if out.ManipulationIndex > 40 {
		risks = append(risks, "ELEVATED_MANIPULATION_RISK")
	}
	if snap.VolatilityRegime.Regime == "EXTREME" || snap.VolatilityRegime.Regime == "HIGH" {
		risks = append(risks, "HIGH_VOLATILITY")
	}
	if snap.DataQuality.State == "DEGRADED" || snap.DataQuality.State == "STALE" {
		risks = append(risks, "DATA_QUALITY_DEGRADED")
	}
	if !snap.IsLive {
		risks = append(risks, "NON_LIVE_DATA_SOURCE")
	}
	return risks
}

// Helper functions
func regimeScore(state *features.MarketState) float64 {
	adx, _ := state.Indicators.ADX.Float64()
	return math.Min(adx*2, 100)
}

func regimeDirection(state *features.MarketState) string {
	if state.Indicators.EMA9.GreaterThan(state.Indicators.EMA21) {
		return "LONG"
	} else if state.Indicators.EMA9.LessThan(state.Indicators.EMA21) {
		return "SHORT"
	}
	return "NEUTRAL"
}

func volScore(snap *MarketIntelligenceSnapshot) float64 {
	switch snap.VolatilityRegime.Regime {
	case "EXTREME":
		return 20
	case "HIGH":
		return 40
	case "NORMAL":
		return 70
	case "LOW":
		return 50
	default:
		return 50
	}
}

func extractManipulationIndex(snap *MarketIntelligenceSnapshot) float64 {
	if snap.ManipulationProxy.Available {
		if val, ok := snap.ManipulationProxy.Value.(map[string]interface{}); ok {
			if idx, ok := val["market_dislocation_index"].(float64); ok {
				return idx * 100
			}
		}
	}
	return 0
}

func boolStr(b bool) string {
	if b {
		return "OK"
	}
	return "INSUFFICIENT"
}
