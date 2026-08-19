// Package adaptation implements the rule-based adaptation manager.
//
// It modifies strategy/scoring parameters based on the current XAUUSD environment.
// Adaptation may make the system MORE CONSERVATIVE but NEVER increases risk
// above central hard account-risk limits.
//
// Safety hierarchy:
//   adaptive proposed risk → strategy risk limit → account risk limit → global hard maximum
//   The most restrictive valid limit wins.
package adaptation

import (
	"math"
	"sync"

	"github.com/shopspring/decimal"
)

// MarketPhase represents the conceptual market phase.
// Uses existing regime enums from the repository where available.
type MarketPhase string

const (
	PhaseTrending       MarketPhase = "TRENDING"
	PhaseRanging        MarketPhase = "RANGING"
	PhaseHighVolatility MarketPhase = "HIGH_VOLATILITY"
	PhaseLowVolatility  MarketPhase = "LOW_VOLATILITY"
	PhaseManipulative   MarketPhase = "MANIPULATIVE"
	PhaseUncertain      MarketPhase = "UNCERTAIN"
)

// Config holds adaptation configuration.
type Config struct {
	Enabled                bool
	StopDistanceMultiplier float64
	RiskMultiplier         float64
	MinConfluence          float64
	MinConfidence          float64
	MaxRiskMultiplier      float64 // hard ceiling — adaptation cannot exceed this
	GlobalHardMaxRisk      float64 // absolute maximum risk fraction
}

// DefaultConfig returns safe defaults. Adaptation is conservative.
func DefaultConfig() Config {
	return Config{
		Enabled:                true,
		StopDistanceMultiplier: 1.0,
		RiskMultiplier:         1.0,
		MinConfluence:          70.0,
		MinConfidence:          65.0,
		MaxRiskMultiplier:      1.0, // never increase risk above baseline
		GlobalHardMaxRisk:      0.02, // 2% max risk per trade
	}
}

// ContextInput provides the market environment for adaptation decisions.
type ContextInput struct {
	Regime            string
	VolatilityState   string
	ManipulationIndex float64
	LiquidityScore    float64
	Spread            float64
	ATR               float64
	Session           string
	MarketStructure   string
	DXYContext        string
	RealYields        string
}

// AdaptationResult is the output of an adaptation evaluation.
type AdaptationResult struct {
	Phase                  MarketPhase
	StopDistanceMultiplier float64
	RiskMultiplier         float64
	MinConfluence          float64
	MinConfidence          float64
	PreferredStrategies    []string
	WeightAdjustments      map[string]float64
	Reason                 string
	Source                 string // RULE_BASED, ML_BASED, FALLBACK
}

// Manager applies rule-based adaptation to strategy parameters.
type Manager struct {
	mu     sync.RWMutex
	config Config
}

// NewManager creates an adaptation manager.
func NewManager(cfg Config) *Manager {
	return &Manager{config: cfg}
}

// Adapt evaluates the market context and returns adapted parameters.
// It operates on a deep copy — it never mutates base configuration globally.
func (m *Manager) Adapt(ctx ContextInput, baseWeights map[string]decimal.Decimal) AdaptationResult {
	m.mu.RLock()
	defer m.mu.RUnlock()

	phase := classifyPhase(ctx)
	result := AdaptationResult{
		Phase:                  phase,
		StopDistanceMultiplier: m.config.StopDistanceMultiplier,
		RiskMultiplier:         m.config.RiskMultiplier,
		MinConfluence:          m.config.MinConfluence,
		MinConfidence:           m.config.MinConfidence,
		Source:                 "RULE_BASED",
	}

	// Deep copy weights — never mutate the original
	adjustedWeights := make(map[string]float64, len(baseWeights))
	for k, v := range baseWeights {
		f, _ := v.Float64()
		adjustedWeights[k] = f
	}

	switch phase {
	case PhaseTrending:
		// In trends: favor trend-following, normal risk
		result.RiskMultiplier = clampRisk(m.config.RiskMultiplier * 1.0, m.config)
		result.StopDistanceMultiplier = 1.0
		// Boost trend/structure weights
		adjustWeight(adjustedWeights, "trend", 1.15)
		adjustWeight(adjustedWeights, "structure", 1.10)
		adjustWeight(adjustedWeights, "mtf_alignment", 1.10)
		result.Reason = "Trending market: favor trend/structure, normal risk"

	case PhaseRanging:
		// In ranges: reduce risk, favor mean-reversion, widen stops slightly
		result.RiskMultiplier = clampRisk(m.config.RiskMultiplier*0.7, m.config)
		result.StopDistanceMultiplier = 1.1
		result.MinConfluence = m.config.MinConfluence + 5 // require more confluence
		adjustWeight(adjustedWeights, "momentum", 0.8)
		adjustWeight(adjustedWeights, "structure", 1.2) // S/R more important
		adjustWeight(adjustedWeights, "vwap", 1.15)
		result.Reason = "Ranging market: reduce risk, favor S/R structure, higher confluence"

	case PhaseHighVolatility:
		// High vol: reduce risk significantly, widen stops
		result.RiskMultiplier = clampRisk(m.config.RiskMultiplier*0.5, m.config)
		result.StopDistanceMultiplier = 1.5
		result.MinConfluence = m.config.MinConfluence + 10
		result.MinConfidence = m.config.MinConfidence + 5
		adjustWeight(adjustedWeights, "volatility", 1.3)
		adjustWeight(adjustedWeights, "liquidity", 1.2)
		result.Reason = "High volatility: significantly reduce risk, widen stops, higher thresholds"

	case PhaseLowVolatility:
		// Low vol: slightly tighter stops, normal risk
		result.RiskMultiplier = clampRisk(m.config.RiskMultiplier*0.9, m.config)
		result.StopDistanceMultiplier = 0.8
		adjustWeight(adjustedWeights, "momentum", 1.15)
		result.Reason = "Low volatility: tighter stops, slightly reduced risk"

	case PhaseManipulative:
		// Manipulation detected: maximum caution
		result.RiskMultiplier = clampRisk(m.config.RiskMultiplier*0.3, m.config)
		result.StopDistanceMultiplier = 1.5
		result.MinConfluence = m.config.MinConfluence + 15
		result.MinConfidence = m.config.MinConfidence + 10
		adjustWeight(adjustedWeights, "manipulation", 1.5)
		adjustWeight(adjustedWeights, "liquidity", 1.3)
		result.Reason = "Manipulative market: maximum caution, reduced risk, higher thresholds"

	case PhaseUncertain:
		// Uncertain: fall back to conservative defaults
		result.RiskMultiplier = clampRisk(m.config.RiskMultiplier*0.6, m.config)
		result.StopDistanceMultiplier = 1.2
		result.MinConfluence = m.config.MinConfluence + 10
		result.Source = "FALLBACK"
		result.Reason = "Uncertain market: conservative fallback, reduced risk"
	}

	// Normalize weights
	normalizeWeights(adjustedWeights)

	// Validate: reject NaN/Inf
	for k, v := range adjustedWeights {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			adjustedWeights[k] = 1.0 // safe fallback
		}
	}

	result.WeightAdjustments = adjustedWeights
	return result
}

// classifyPhase determines the market phase from context inputs.
// Uses existing regime enums where possible.
func classifyPhase(ctx ContextInput) MarketPhase {
	// High manipulation score → manipulative
	if ctx.ManipulationIndex > 70 {
		return PhaseManipulative
	}

	// High volatility
	if ctx.VolatilityState == "HIGH" || ctx.VolatilityState == "EXTREME" {
		return PhaseHighVolatility
	}

	// Low volatility
	if ctx.VolatilityState == "LOW" {
		return PhaseLowVolatility
	}

	// Regime-based
	switch ctx.Regime {
	case "TRENDING_BULLISH", "TRENDING_BEARISH", "BREAKOUT":
		return PhaseTrending
	case "RANGE", "MEAN_REVERSION":
		return PhaseRanging
	case "HIGH_VOLATILITY":
		return PhaseHighVolatility
	case "LOW_VOLATILITY":
		return PhaseLowVolatility
	case "UNSTABLE", "NO_TRADE":
		return PhaseUncertain
	}

	// Default: uncertain if we can't classify
	return PhaseUncertain
}

// clampRisk ensures risk never exceeds the hard limit.
// The most restrictive valid limit wins.
func clampRisk(proposed float64, cfg Config) float64 {
	// 1. Cannot exceed MaxRiskMultiplier (adaptation ceiling)
	if proposed > cfg.MaxRiskMultiplier {
		proposed = cfg.MaxRiskMultiplier
	}
	// 2. Cannot exceed GlobalHardMaxRisk (absolute maximum)
	if proposed > cfg.GlobalHardMaxRisk {
		proposed = cfg.GlobalHardMaxRisk
	}
	// 3. Never below zero
	if proposed < 0 {
		proposed = 0
	}
	return proposed
}

func adjustWeight(weights map[string]float64, key string, factor float64) {
	if v, ok := weights[key]; ok {
		weights[key] = v * factor
	}
}

func normalizeWeights(weights map[string]float64) {
	sum := 0.0
	for _, v := range weights {
		sum += v
	}
	if sum <= 0 {
		return
	}
	for k, v := range weights {
		weights[k] = v / sum * float64(len(weights))
	}
}

// IsEnabled returns whether adaptation is enabled.
func (m *Manager) IsEnabled() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.config.Enabled
}
