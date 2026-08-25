package engines

import (
	"testing"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func makeTestState(atr float64, price float64, regime types.Regime) *features.MarketState {
	return &features.MarketState{
		CurrentPrice: decimal.NewFromFloat(price),
		Spread:       decimal.NewFromFloat(0.25),
		Indicators: features.IndicatorFeatures{
			ATR: decimal.NewFromFloat(atr),
		},
		Regime: features.RegimeFeatures{
			Current: regime,
		},
	}
}

func makeBuyResult(score float64, entry float64) strategy.StrategyResult {
	return strategy.StrategyResult{
		Direction:  types.DirectionBuy,
		RawScore:   decimal.NewFromFloat(score),
		EntryPrice: decimal.NewFromFloat(entry),
		StopLoss:   decimal.NewFromFloat(entry - 20),
		TP1:        decimal.NewFromFloat(entry + 30),
		TP2:        decimal.NewFromFloat(entry + 50),
		TP3:        decimal.NewFromFloat(entry + 80),
	}
}

func TestFactory_ReturnsEngineForKnownStrategy(t *testing.T) {
	tests := []struct {
		name   types.StrategyID
		engine EngineType
	}{
		{types.StrategyUltraScalping, UltraScalp},
		{types.StrategyStandardScalping, StdScalp},
		{types.StrategyStandardSwing, StdSwing},
		{types.StrategyTrendSwing, TrendSwng},
	}
	for _, tt := range tests {
		engine, err := GetEngine(tt.name)
		if err != nil {
			t.Errorf("GetEngine(%s) error: %v", tt.name, err)
		}
		if engine == nil {
			t.Errorf("GetEngine(%s) returned nil", tt.name)
		}
		if engine.Type() != tt.engine {
			t.Errorf("GetEngine(%s) type = %s, want %s", tt.name, engine.Type(), tt.engine)
		}
	}
}

func TestFactory_ReturnsNilForUnknownStrategy(t *testing.T) {
	engine, err := GetEngine("UNKNOWN_STRATEGY")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if engine != nil {
		t.Errorf("expected nil for unknown strategy, got %v", engine)
	}
}

func TestUltraScalp_LowATR_Rejects(t *testing.T) {
	engine, _ := GetEngine(types.StrategyUltraScalping)
	state := makeTestState(1.0, 4600, types.RegimeTrendingBullish) // ATR=1.0 < MinAbsATR=3.0
	result := engine.Evaluate(makeBuyResult(70, 4600), state)
	if result.RejectReason == "" {
		t.Error("expected rejection for low ATR (1.0 < 3.0)")
	}
}

func TestUltraScalp_RegimeMismatch_NoLongerRejects(t *testing.T) {
	// Regime gate was removed — scoring system handles regime filtering via thresholds.
	// RANGE regime signals are allowed but will have lower scores due to regime-specific thresholds.
	engine, _ := GetEngine(types.StrategyUltraScalping)
	state := makeTestState(15.0, 4600, types.RegimeRange)
	result := engine.Evaluate(makeBuyResult(70, 4600), state)
	if result.RejectReason != "" {
		t.Errorf("expected no rejection for regime mismatch (regime gate removed), got: %s", result.RejectReason)
	}
}

func TestUltraScalp_LowGrade_NoLongerRejects(t *testing.T) {
	// Grade gate was removed — scoring system handles quality via thresholds.
	engine, _ := GetEngine(types.StrategyUltraScalping)
	state := makeTestState(15.0, 4600, types.RegimeTrendingBullish)
	result := engine.Evaluate(makeBuyResult(50, 4600), state)
	if result.RejectReason != "" {
		t.Errorf("expected no rejection for low grade (grade gate removed), got: %s", result.RejectReason)
	}
}

func TestUltraScalp_ValidSignal_AppliesOverrides(t *testing.T) {
	engine, _ := GetEngine(types.StrategyUltraScalping)
	state := makeTestState(15.0, 4600, types.RegimeTrendingBullish)
	result := engine.Evaluate(makeBuyResult(70, 4600), state)
	if !result.Applied {
		t.Error("expected engine to apply overrides")
	}
	if result.Result.ExpiryMinutes != 5 {
		t.Errorf("expiry=%d, want 5", result.Result.ExpiryMinutes)
	}
}

func TestStdScalp_LowATR_Rejects(t *testing.T) {
	engine, _ := GetEngine(types.StrategyStandardScalping)
	state := makeTestState(1.0, 4600, types.RegimeTrendingBullish) // ATR=1.0 < MinAbsATR=2.0
	result := engine.Evaluate(makeBuyResult(70, 4600), state)
	if result.RejectReason == "" {
		t.Error("expected rejection for low ATR")
	}
}

func TestStdScalp_AllRegimesAllowed(t *testing.T) {
	engine, _ := GetEngine(types.StrategyStandardScalping)
	for _, regime := range []types.Regime{types.RegimeTrendingBullish, types.RegimeRange, types.RegimeMeanReversion, types.RegimeBreakout} {
		state := makeTestState(10.0, 4600, regime)
		result := engine.Evaluate(makeBuyResult(70, 4600), state)
		if result.RejectReason != "" {
			t.Errorf("StdScalp should allow regime %s, got rejection: %s", regime, result.RejectReason)
		}
	}
}

// Regime gate was deliberately removed from engines (commit 52063c1): scoring
// thresholds handle regime filtering. The pipeline-level TrendSwing/range
// separation is enforced by the legacy strategy layer (AcceptedRegimes +
// TestTrendSwing_RangeRejected), so the engine must not double-reject here.
func TestTrendSwing_RegimeGateRemoved_NoEngineRejection(t *testing.T) {
	engine, _ := GetEngine(types.StrategyTrendSwing)
	state := makeTestState(15.0, 4600, types.RegimeRange)
	result := engine.Evaluate(makeBuyResult(70, 4600), state)
	if result.RejectReason != "" {
		t.Errorf("expected no engine-level rejection for RANGE (gate removed), got: %s", result.RejectReason)
	}
}

func TestTrendSwing_IgnoreStructure_True(t *testing.T) {
	engine, _ := GetEngine(types.StrategyTrendSwing)
	state := makeTestState(15.0, 4600, types.RegimeTrendingBullish)
	state.Structure.SwingLows = []decimal.Decimal{decimal.NewFromFloat(4500)}
	result := engine.Evaluate(makeBuyResult(70, 4600), state)
	if !result.Applied {
		t.Error("expected engine to apply")
	}
}

func TestFallback_NoTradeResult(t *testing.T) {
	engine, _ := GetEngine(types.StrategyUltraScalping)
	state := makeTestState(15.0, 4600, types.RegimeTrendingBullish)
	noTradeResult := strategy.StrategyResult{Direction: types.DirectionNoTrade}
	result := engine.Evaluate(noTradeResult, state)
	if !result.Fallback {
		t.Error("expected fallback for NO-TRADE legacy result")
	}
}

func TestGetEngineConfig_ReturnsConfig(t *testing.T) {
	cfg := GetEngineConfig(types.StrategyUltraScalping)
	if cfg == nil {
		t.Fatal("expected config for UltraScalping")
	}
	if cfg.MinAbsATR != 3.0 {
		t.Errorf("MinAbsATR=%.1f, want 3.0 (Phase 6 config: micro-profit scalping)", cfg.MinAbsATR)
	}
	if !cfg.IgnoreStructure {
		t.Error("IgnoreStructure should be true for UltraScalp")
	}
	if cfg.OverrideTPs[0] != 0.5 {
		t.Errorf("OverrideTPs[0]=%.1f, want 0.5 (Phase 6 micro-profit TP distances)", cfg.OverrideTPs[0])
	}
}
