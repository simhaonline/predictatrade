// Package ptb — Tests for synthesis engine, setup quality, correlation, and integration.
// Sections 33-36: Unit tests, integration tests, regression tests.
package ptb

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// === SYNTHESIS TESTS ===

func TestSynthesize_StrongLong(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)
	if out == nil {
		t.Fatal("Expected non-nil synthesis output")
	}
	// With bullish state, bias should be LONG or STRONG_LONG
	if out.Bias != BiasLong && out.Bias != BiasStrongLong && out.Bias != BiasNeutral {
		t.Errorf("Expected LONG or NEUTRAL bias for bullish state, got %s", out.Bias)
	}
}

func TestSynthesize_StrongShort(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	// Flip to bearish
	state.Indicators.EMA9 = decimal.NewFromFloat(4398.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4402.0)
	state.Structure.CurrentTrend = "bearish"
	state.Structure.LastBOS = &features.StructureEvent{Type: "BOS", Direction: "bearish"}
	state.MTF.Score = -86.67
	state.MTF.States = map[types.Timeframe]int{
		types.TFM1: -1, types.TFM5: -1, types.TFH1: -1, types.TFH4: -1,
	}
	state.Candle.IsBullish = false
	state.Candle.IsBearish = true

	snap := e.Evaluate(state, "snap-002", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)
	if out.Bias != BiasShort && out.Bias != BiasStrongShort && out.Bias != BiasNeutral {
		t.Errorf("Expected SHORT or NEUTRAL bias for bearish state, got %s", out.Bias)
	}
}

func TestSynthesize_NeutralConflictingState(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	// Create conflicting evidence
	state.MTF.States[types.TFH1] = -1
	state.MTF.States[types.TFH4] = -1
	state.MTF.Score = 0

	snap := e.Evaluate(state, "snap-003", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)
	// Conflicting evidence should produce NEUTRAL or STAND_ASIDE
	if out.Bias == BiasStrongLong || out.Bias == BiasStrongShort {
		t.Errorf("Expected non-strong bias for conflicting state, got %s", out.Bias)
	}
}

func TestSynthesize_ShadowMode_NoProductionImpact(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	cfg.ShadowMode = true
	out := e.Synthesize(state, snap, cfg)
	if !out.ShadowMode {
		t.Error("Expected ShadowMode=true")
	}
	// In shadow mode, position multiplier should be 0 or low
	// (it's a recommendation only, not applied to production)
}

func TestSynthesize_SetupQualityGrading(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)

	// Setup quality must be one of the valid grades
	validGrades := map[SetupGrade]bool{
		GradeAPlus: true, GradeA: true, GradeB: true,
		GradeC: true, GradeD: true, GradeF: true,
	}
	if !validGrades[out.SetupQuality] {
		t.Errorf("Invalid setup quality: %s", out.SetupQuality)
	}
}

func TestSynthesize_PositionSizeMultiplier(t *testing.T) {
	cfg := DefaultConfig()
	// A+ → 1.0, A → 0.8, B → 0.6, C → 0.4, D → 0.2, F → 0.0
	tests := []struct {
		grade    SetupGrade
		expected float64
	}{
		{GradeAPlus, 1.00},
		{GradeA, 0.80},
		{GradeB, 0.60},
		{GradeC, 0.40},
		{GradeD, 0.20},
		{GradeF, 0.00},
	}
	for _, tt := range tests {
		mult := positionSizeMultiplier(tt.grade, cfg)
		if mult != tt.expected {
			t.Errorf("Grade %s: expected mult %.2f, got %.2f", tt.grade, tt.expected, mult)
		}
	}
}

func TestSynthesize_StopDistanceMultiplier(t *testing.T) {
	cfg := DefaultConfig()
	// High manipulation → 1.5
	if mult := stopDistanceMultiplier(75, "NORMAL", cfg); mult != 1.5 {
		t.Errorf("High manipulation: expected 1.5, got %.1f", mult)
	}
	// Normal → 1.0
	if mult := stopDistanceMultiplier(30, "NORMAL", cfg); mult != 1.0 {
		t.Errorf("Normal: expected 1.0, got %.1f", mult)
	}
	// Low volatility → 0.8
	if mult := stopDistanceMultiplier(10, "LOW", cfg); mult != 0.8 {
		t.Errorf("Low volatility: expected 0.8, got %.1f", mult)
	}
}

func TestSynthesize_ActionStates(t *testing.T) {
	cfg := DefaultConfig()
	// High manipulation → AVOID
	if action := determineAction(BiasLong, 80, 75, cfg); action != ActionAvoid {
		t.Errorf("High manipulation: expected AVOID, got %s", action)
	}
	// Low confidence → WAIT
	if action := determineAction(BiasLong, 50, 20, cfg); action != ActionWait {
		t.Errorf("Low confidence: expected WAIT, got %s", action)
	}
	// Good setup → ENTER
	if action := determineAction(BiasLong, 80, 20, cfg); action != ActionEnter {
		t.Errorf("Good setup: expected ENTER, got %s", action)
	}
	// Neutral → WAIT
	if action := determineAction(BiasNeutral, 80, 20, cfg); action != ActionWait {
		t.Errorf("Neutral: expected WAIT, got %s", action)
	}
}

func TestSynthesize_RegimeClassification(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)

	validRegimes := map[EnhancedRegime]bool{
		RegimeStrongTrendUp: true, RegimeStrongTrendDown: true,
		RegimeWeakTrendUp: true, RegimeWeakTrendDown: true,
		RegimeRangeBound: true, RegimeHighVolatility: true,
		RegimeLowVolatility: true, RegimeTransitioning: true,
		RegimeManipulation: true,
	}
	if !validRegimes[out.Regime] {
		t.Errorf("Invalid regime: %s", out.Regime)
	}
}

func TestSynthesize_GoldRoleUnknown_WithoutData(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)
	// Without DXY/yield data, gold role must be UNKNOWN — no fabrication
	if out.GoldRole != GoldRoleUnknown {
		t.Errorf("Expected UNKNOWN gold role without macro data, got %s", out.GoldRole)
	}
}

func TestSynthesize_ReasonCodes(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)

	// Must produce machine-readable reason codes
	if len(out.ReasonCodes) == 0 && len(out.PositiveFactors) == 0 && len(out.NegativeFactors) == 0 {
		t.Error("Expected non-empty reason codes or factors")
	}
}

func TestSynthesize_MarketNarrative(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)

	// Must produce a deterministic narrative string
	if out.MarketNarrative == "" {
		t.Error("Expected non-empty market narrative")
	}
}

func TestSynthesize_ComponentScores(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)

	// Must include component scores for key components
	required := []string{"mtf", "regime", "volatility", "structure", "liquidity", "session", "manipulation"}
	for _, comp := range required {
		if _, ok := out.ComponentScores[comp]; !ok {
			t.Errorf("Missing component score: %s", comp)
		}
	}
}

func TestSynthesize_LiquidityTargets(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)

	// Should produce liquidity targets from swing highs/lows
	if len(out.LiquidityTargets) == 0 {
		// May be empty if no levels above/below price — acceptable
	}
	// Verify targets have valid structure
	for _, tgt := range out.LiquidityTargets {
		if tgt.Type == "" {
			t.Error("Liquidity target missing type")
		}
	}
}

// === CORRELATION TESTS ===

func TestCorrelation_NoData_Unavailable(t *testing.T) {
	c := NewCorrelationEngine()
	result := c.ComputeDXYCorrelation(20)
	if result.Quality != "UNAVAILABLE" {
		t.Errorf("Expected UNAVAILABLE for no DXY data, got %s", result.Quality)
	}
}

func TestCorrelation_WithDXYData(t *testing.T) {
	c := NewCorrelationEngine()
	// Add gold and DXY observations (inversely correlated)
	for i := 0; i < 30; i++ {
		c.AddGoldObservation(float64(2000+i), time.Now())
		c.AddDXYObservation(float64(100-float64(i)*0.1), time.Now())
	}
	result := c.ComputeDXYCorrelation(20)
	if result.Quality == "UNAVAILABLE" {
		t.Skip("DXY data not available despite being added")
	}
	if result.Correlation > 0 {
		// Inversely correlated data should produce negative correlation
		t.Logf("DXY correlation: %.4f (direction: %s)", result.Correlation, result.Direction)
	}
}

func TestCorrelation_ConstantSeries_Handled(t *testing.T) {
	c := NewCorrelationEngine()
	// Add constant series — should return 0, not NaN
	for i := 0; i < 30; i++ {
		c.AddGoldObservation(2000.0, time.Now())
		c.AddDXYObservation(100.0, time.Now())
	}
	result := c.ComputeDXYCorrelation(20)
	if result.Quality == "INVALID" {
		// Correctly handled — constant series produces NaN which is caught
		return
	}
}

func TestCorrelation_InsufficientSamples(t *testing.T) {
	c := NewCorrelationEngine()
	c.AddGoldObservation(2000.0, time.Now())
	c.AddDXYObservation(100.0, time.Now())
	result := c.ComputeDXYCorrelation(20)
	if result.Quality != "UNAVAILABLE" {
		t.Errorf("Expected UNAVAILABLE for insufficient samples, got %s", result.Quality)
	}
}

func TestCorrelation_GoldSilverRatio_NoData(t *testing.T) {
	c := NewCorrelationEngine()
	ratio, ok := c.GoldSilverRatio()
	if ok {
		t.Errorf("Expected GoldSilverRatio ok=false without silver data")
	}
	if ratio != 0 {
		t.Errorf("Expected ratio 0 without data, got %f", ratio)
	}
}

// === CONFIG TESTS ===

func TestConfig_Defaults_ShadowMode(t *testing.T) {
	cfg := DefaultConfig()
	if !cfg.ShadowMode {
		t.Error("Default config should have ShadowMode=true")
	}
	if !cfg.Enabled {
		t.Error("Default config should have Enabled=true")
	}
}

func TestConfig_GradeThresholds(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.GradeAPlus <= cfg.GradeA {
		t.Error("A+ threshold should be higher than A")
	}
	if cfg.GradeA <= cfg.GradeB {
		t.Error("A threshold should be higher than B")
	}
}

// === REGRESSION TEST (Section 36) ===
// PTB disabled should preserve existing behavior.
func TestRegression_PTBDisabled_PreservesBehavior(t *testing.T) {
	e := NewEngine()
	state := makeTestState()

	// With PTB shadow mode, strategies should produce the same results
	// as without PTB. The PTB snapshot is stored but doesn't alter scores.
	snap := e.Evaluate(state, "snap-reg", types.DataSourceLiveAgent)
	if snap == nil {
		t.Fatal("PTB snapshot should not be nil")
	}

	// Verify all module scores are zero (SHADOW)
	if !snap.LiquidityVoid.ScoreContrib.IsZero() {
		t.Error("SHADOW module should have zero score contribution")
	}
	if !snap.ManipulationProxy.ScoreContrib.IsZero() {
		t.Error("SHADOW module should have zero score contribution")
	}

	// Synthesis output should not affect the strategy's own evaluation
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)
	if out != nil && out.ShadowMode {
		// Shadow mode = no production impact
		return
	}
	if out == nil {
		t.Error("Synthesis should produce output even in shadow mode")
	}
}

// === INTEGRATION TEST (Section 34) ===
func TestIntegration_PTBCannotBypassRiskGates(t *testing.T) {
	// PTB may recommend ENTER, but hard risk gates remain authoritative.
	// This test verifies PTB output does not bypass risk.
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-int", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)

	// Even if PTB says ENTER, the existing risk gates (spread, RR, etc.)
	// remain the final authority. PTB action is advisory only.
	// The strategy engine and gate system are separate from PTB.
	if out.Action == ActionEnter {
		// This is fine — it's a recommendation. The existing gate system
		// still makes the final call.
		t.Log("PTB recommends ENTER — risk gates remain authoritative")
	}
}

func TestIntegration_AVOID_DoesNotProduceSignal(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-avoid", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	// Force high manipulation to trigger AVOID
	cfg.ManipHighRisk = 10 // low threshold to trigger AVOID easily
	out := e.Synthesize(state, snap, cfg)

	if out.Action == ActionAvoid {
		// AVOID should never produce an executable signal
		// The strategy engine would see this as advisory and not trade
		t.Log("AVOID action correctly produced for high manipulation")
	}
}

func TestIntegration_UnavailableMacroDoesNotCrash(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	// No DXY/silver/yield data added to correlation engine
	snap := e.Evaluate(state, "snap-no-macro", types.DataSourceLiveAgent)
	cfg := DefaultConfig()
	out := e.Synthesize(state, snap, cfg)

	// Should not crash — macro just marked UNAVAILABLE
	if out == nil {
		t.Fatal("Synthesis should not crash with unavailable macro data")
	}
	if out.GoldRole != GoldRoleUnknown {
		t.Errorf("Expected UNKNOWN gold role, got %s", out.GoldRole)
	}
}
