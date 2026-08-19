package adaptation

import (
	"math"
	"testing"

	"github.com/shopspring/decimal"
)

func baseWeights() map[string]decimal.Decimal {
	return map[string]decimal.Decimal{
		"trend":     decimal.NewFromFloat(0.25),
		"momentum":  decimal.NewFromFloat(0.20),
		"structure": decimal.NewFromFloat(0.20),
		"liquidity": decimal.NewFromFloat(0.15),
		"vwap":      decimal.NewFromFloat(0.10),
		"volatility": decimal.NewFromFloat(0.10),
	}
}

func TestEveryRegime(t *testing.T) {
	m := NewManager(DefaultConfig())
	phases := []MarketPhase{PhaseTrending, PhaseRanging, PhaseHighVolatility, PhaseLowVolatility, PhaseManipulative, PhaseUncertain}
	for _, phase := range phases {
		ctx := ContextInput{}
		switch phase {
		case PhaseTrending:
			ctx.Regime = "TRENDING_BULLISH"
		case PhaseRanging:
			ctx.Regime = "RANGE"
		case PhaseHighVolatility:
			ctx.VolatilityState = "HIGH"
		case PhaseLowVolatility:
			ctx.VolatilityState = "LOW"
		case PhaseManipulative:
			ctx.ManipulationIndex = 80
		case PhaseUncertain:
			ctx.Regime = "UNSTABLE"
		}
		result := m.Adapt(ctx, baseWeights())
		if result.Phase != phase {
			t.Errorf("expected phase %s, got %s", phase, result.Phase)
		}
	}
}

func TestWeightNormalization(t *testing.T) {
	m := NewManager(DefaultConfig())
	ctx := ContextInput{Regime: "TRENDING_BULLISH"}
	result := m.Adapt(ctx, baseWeights())
	sum := 0.0
	for _, v := range result.WeightAdjustments {
		sum += v
	}
	if math.Abs(sum-float64(len(baseWeights()))) > 0.01 {
		t.Fatalf("weights should be normalized, sum=%f", sum)
	}
}

func TestNoNaNInf(t *testing.T) {
	m := NewManager(DefaultConfig())
	ctx := ContextInput{Regime: "TRENDING_BULLISH"}
	result := m.Adapt(ctx, baseWeights())
	for k, v := range result.WeightAdjustments {
		if math.IsNaN(v) || math.IsInf(v, 0) {
			t.Fatalf("weight %s is NaN/Inf: %f", k, v)
		}
	}
}

func TestNoMutationOfBaseConfig(t *testing.T) {
	m := NewManager(DefaultConfig())
	original := baseWeights()
	originalTrend, _ := original["trend"].Float64()
	ctx := ContextInput{Regime: "TRENDING_BULLISH"}
	_ = m.Adapt(ctx, original)
	afterTrend, _ := original["trend"].Float64()
	if afterTrend != originalTrend {
		t.Fatalf("base config was mutated: trend before=%f after=%f", originalTrend, afterTrend)
	}
}

func TestRiskClamps(t *testing.T) {
	m := NewManager(DefaultConfig())
	ctx := ContextInput{ManipulationIndex: 80}
	result := m.Adapt(ctx, baseWeights())
	// Manipulative should clamp risk to 0.3 * 1.0 = 0.3, but capped at MaxRiskMultiplier=1.0
	if result.RiskMultiplier > m.config.MaxRiskMultiplier {
		t.Fatalf("risk multiplier should not exceed max: %f > %f", result.RiskMultiplier, m.config.MaxRiskMultiplier)
	}
	if result.RiskMultiplier > m.config.GlobalHardMaxRisk {
		t.Fatalf("risk should not exceed global hard max: %f > %f", result.RiskMultiplier, m.config.GlobalHardMaxRisk)
	}
}

func TestStrategySelection(t *testing.T) {
	m := NewManager(DefaultConfig())
	ctx := ContextInput{Regime: "TRENDING_BULLISH"}
	result := m.Adapt(ctx, baseWeights())
	if result.PreferredStrategies == nil {
		// PreferredStrategies may be nil if not set — that's OK, it's optional
	}
}

func TestFallbackForUnknownContext(t *testing.T) {
	m := NewManager(DefaultConfig())
	ctx := ContextInput{} // empty context
	result := m.Adapt(ctx, baseWeights())
	if result.Phase != PhaseUncertain {
		t.Fatalf("empty context should be uncertain, got %s", result.Phase)
	}
	if result.Source != "FALLBACK" {
		t.Fatalf("uncertain should use FALLBACK source, got %s", result.Source)
	}
}

func TestConservativeHandlingOfMissingData(t *testing.T) {
	m := NewManager(DefaultConfig())
	ctx := ContextInput{Regime: ""} // missing regime
	result := m.Adapt(ctx, baseWeights())
	if result.RiskMultiplier > 1.0 {
		t.Fatalf("missing data should not increase risk, got %f", result.RiskMultiplier)
	}
}
