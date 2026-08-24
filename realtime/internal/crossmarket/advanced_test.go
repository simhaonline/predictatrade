package crossmarket

import (
	"math"
	"time"
	"testing"
)

func TestLeadLagDetector_BasicLag(t *testing.T) {
	detector := NewLeadLagDetector(5)
	// XAU returns lag asset returns by 2 bars
	xau := []float64{0.01, 0.01, 0.02, 0.02, 0.03, 0.03, 0.04, 0.04, 0.05, 0.05,
		0.06, 0.06, 0.07, 0.07, 0.08, 0.08, 0.09, 0.09, 0.10, 0.10,
		0.11, 0.11, 0.12, 0.12, 0.13, 0.13, 0.14, 0.14, 0.15, 0.15}
	asset := []float64{0.03, 0.03, 0.05, 0.05, 0.07, 0.07, 0.09, 0.09, 0.11, 0.11,
		0.13, 0.13, 0.15, 0.15, 0.17, 0.17, 0.19, 0.19, 0.21, 0.21,
		0.23, 0.23, 0.25, 0.25, 0.27, 0.27, 0.29, 0.29, 0.31, 0.31}

	result := detector.Analyze(xau, asset, "test_asset")
	if result.SampleCount < 20 {
		t.Errorf("need at least 20 samples, got %d", result.SampleCount)
	}
	// The best correlation should be positive (direct) or negative (inverse)
	// Just verify it doesn't crash and produces a result
	if math.IsNaN(result.Coefficient) {
		t.Error("coefficient should not be NaN")
	}
}

func TestLeadLagDetector_InsufficientData(t *testing.T) {
	detector := NewLeadLagDetector(5)
	result := detector.Analyze([]float64{0.01, 0.02}, []float64{0.01}, "test")
	if result.Confidence != 0 {
		t.Errorf("insufficient data should produce zero confidence, got %.2f", result.Confidence)
	}
}

func TestDXYExhaustion_MomentumDeceleration(t *testing.T) {
	// DXY values with clear momentum deceleration
	dxy := []float64{100, 100.5, 101, 101.3, 101.5, 101.55, 101.56, 101.57, 101.58, 101.59}
	rsi := []float64{60, 62, 65, 64, 63, 62, 61, 60, 59, 58} // RSI declining while price rises

	result := CalculateDXYExhaustion(dxy, rsi, 18.0) // low ADX
	if result.Score < 50 {
		t.Errorf("clear exhaustion should produce score >= 50, got %.1f", result.Score)
	}
	if len(result.Evidence) < 2 {
		t.Error("should have multiple evidence factors")
	}
}

func TestDXYExhaustion_InsufficientData(t *testing.T) {
	result := CalculateDXYExhaustion([]float64{100, 101}, nil, 0)
	if result.Confidence != 0 {
		t.Errorf("insufficient data should produce zero confidence, got %.2f", result.Confidence)
	}
}

func TestBTCShock_NormalRange(t *testing.T) {
	// Normal returns within 1 std
	returns := make([]float64, 50)
	for i := range returns {
		returns[i] = 0.001 * float64(i%3-1) // small oscillation
	}
	result := DetectBTCShock(returns, 50)
	if result.State != BTCShockNormal {
		t.Errorf("normal returns should produce NORMAL state, got %s", result.State)
	}
}

func TestBTCShock_ExtremeShock(t *testing.T) {
	// Build a series with a large outlier at the end
	returns := make([]float64, 50)
	for i := 0; i < 49; i++ {
		returns[i] = 0.001
	}
	returns[49] = 0.15 // 15% return — extreme outlier

	result := DetectBTCShock(returns, 50)
	if result.State != BTCShockExtreme && result.State != BTCShockShock {
		t.Errorf("extreme outlier should produce SHOCK or EXTREME, got %s", result.State)
	}
	if result.ZScore < 3 {
		t.Errorf("extreme outlier should have z-score > 3, got %.2f", result.ZScore)
	}
}

func TestBTCShock_InsufficientData(t *testing.T) {
	result := DetectBTCShock([]float64{0.01}, 50)
	if result.State != BTCShockNormal {
		t.Errorf("insufficient data should produce NORMAL, got %s", result.State)
	}
}

func TestCircuitBreaker_ValidPrice(t *testing.T) {
	cb := NewCircuitBreaker()
	valid, reason := cb.Validate("DXY", 98.5, timeNow())
	if !valid {
		t.Errorf("valid price should pass, got rejection: %s", reason)
	}
}

func TestCircuitBreaker_ZeroPrice(t *testing.T) {
	cb := NewCircuitBreaker()
	valid, reason := cb.Validate("DXY", 0, timeNow())
	if valid {
		t.Error("zero price should be rejected")
	}
	if reason != "zero_or_negative_price" {
		t.Errorf("expected zero_or_negative_price, got %s", reason)
	}
}

func TestCircuitBreaker_NaNPrice(t *testing.T) {
	cb := NewCircuitBreaker()
	valid, _ := cb.Validate("DXY", math.NaN(), timeNow())
	if valid {
		t.Error("NaN price should be rejected")
	}
}

func TestCircuitBreaker_ImpossibleJump(t *testing.T) {
	cb := NewCircuitBreaker()
	cb.Validate("DXY", 100.0, timeNow())
	valid, reason := cb.Validate("DXY", 200.0, timeNow()) // 100% jump
	if valid {
		t.Error("100% price jump should be rejected")
	}
	if reason != "impossible_price_jump" {
		t.Errorf("expected impossible_price_jump, got %s", reason)
	}
}

func TestCircuitBreaker_TimestampInversion(t *testing.T) {
	cb := NewCircuitBreaker()
	t1 := timeNow()
	t2 := t1.Add(-1 * time.Hour) // earlier timestamp
	cb.Validate("DXY", 100.0, t1)
	valid, reason := cb.Validate("DXY", 100.1, t2)
	if valid {
		t.Error("timestamp inversion should be rejected")
	}
	if reason != "timestamp_inversion" {
		t.Errorf("expected timestamp_inversion, got %s", reason)
	}
}

func TestReturnsCalculator_LogReturns(t *testing.T) {
	rc := NewReturnsCalculator()
	prices := []float64{100, 101, 102, 103}
	returns := rc.LogReturns(prices)
	if len(returns) != 3 {
		t.Errorf("expected 3 returns, got %d", len(returns))
	}
	if returns[0] <= 0 {
		t.Errorf("rising prices should produce positive log returns, got %.4f", returns[0])
	}
}

func TestReturnsCalculator_SimpleReturns(t *testing.T) {
	rc := NewReturnsCalculator()
	prices := []float64{100, 105, 100}
	returns := rc.SimpleReturns(prices)
	if returns[0] != 0.05 {
		t.Errorf("expected 0.05, got %.4f", returns[0])
	}
	if returns[1] != -5.0/105.0 {
		t.Errorf("expected negative return, got %.4f", returns[1])
	}
}

func TestZScore(t *testing.T) {
	values := []float64{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 10}
	z := ZScore(values, 10)
	if z < 1.0 {
		t.Errorf("latest value at top of range should have high z-score, got %.2f", z)
	}
}

func TestRollingCorrelation(t *testing.T) {
	// Perfectly correlated series
	x := make([]float64, 60)
	y := make([]float64, 60)
	for i := range x {
		x[i] = float64(i)
		y[i] = float64(i) * 2
	}
	corrs := RollingCorrelation(x, y, 20)
	if len(corrs) == 0 {
		t.Error("should produce correlation values")
	}
	if corrs[len(corrs)-1] < 0.99 {
		t.Errorf("perfectly correlated series should have corr > 0.99, got %.4f", corrs[len(corrs)-1])
	}
}

func TestCorrelationStability(t *testing.T) {
	// Stable correlations around 0.8
	stableCorrs := []float64{0.79, 0.81, 0.80, 0.82, 0.78, 0.80, 0.81, 0.79, 0.80, 0.82}
	stability := CorrelationStability(stableCorrs)
	if stability < 0.8 {
		t.Errorf("stable correlations should produce high stability, got %.2f", stability)
	}

	// Unstable correlations
	unstableCorrs := []float64{0.8, -0.3, 0.6, -0.5, 0.9, -0.2, 0.4, -0.4, 0.7, -0.1}
	stability = CorrelationStability(unstableCorrs)
	if stability > 0.5 {
		t.Errorf("unstable correlations should produce low stability, got %.2f", stability)
	}
}

func TestCalculateMomentum(t *testing.T) {
	prices := make([]float64, 50)
	for i := range prices {
		prices[i] = 100 + float64(i) // steadily rising
	}
	mom := CalculateMomentum(prices)
	if mom.Direction != DirBullish {
		t.Errorf("rising prices should be bullish, got %s", mom.Direction)
	}
	if mom.Return1 <= 0 {
		t.Errorf("rising prices should have positive return, got %.4f", mom.Return1)
	}
	if mom.ZScore <= 0 {
		t.Errorf("rising prices should have positive z-score, got %.2f", mom.ZScore)
	}
}

func TestStrategyWeights_AllStrategies(t *testing.T) {
	weights := DefaultStrategyWeights()
	if len(weights) != 4 {
		t.Errorf("expected 4 strategy weight configs, got %d", len(weights))
	}

	// Ultra scalping should have lower max contribution than trend swing
	if weights[StrategyUltraScalping].MaxContribution >= weights[StrategyTrendSwing].MaxContribution {
		t.Error("ultra scalping max contribution should be less than trend swing")
	}

	// Trend swing should have highest real yield weight
	trendYieldWeight := weights[StrategyTrendSwing].Weights[DriverRealYields]
	ultraYieldWeight := weights[StrategyUltraScalping].Weights[DriverRealYields]
	if trendYieldWeight <= ultraYieldWeight {
		t.Error("trend swing should weight real yields higher than ultra scalping")
	}

	// BTC weight should be minimal for trend swing
	btcTrendWeight := weights[StrategyTrendSwing].Weights[DriverBTC]
	if btcTrendWeight > 3.0 {
		t.Errorf("BTC weight for trend swing should be minimal (≤3), got %.1f", btcTrendWeight)
	}
}

func TestMacroSignalFields_FromConfluenceResult(t *testing.T) {
	result := ConfluenceResult{
		Score:      65,
		Confidence: 0.8,
		Regime:     SHSafeHavenGold,
		DataQuality: QualityConnected,
		DriverSnapshot: []DriverSnapshot{
			{Name: DriverDXY, Direction: DirBullish},
			{Name: DriverBTC, Direction: DirBearish},
		},
	}
	fields := FromConfluenceResult(&result)
	if fields.MacroScore != 65 {
		t.Errorf("expected macro_score=65, got %.1f", fields.MacroScore)
	}
	if fields.DXYBias != "BULLISH" {
		t.Errorf("expected DXYBias=BULLISH, got %s", fields.DXYBias)
	}
	if fields.CrossAssetState != "SAFE_HAVEN_ROTATION" {
		t.Errorf("expected SAFE_HAVEN_ROTATION, got %s", fields.CrossAssetState)
	}
	if fields.MacroExplanation == "" {
		t.Error("explanation should not be empty")
	}
}

func TestRecordMetrics(t *testing.T) {
	result := ConfluenceResult{
		Score:       50,
		Confidence:  0.75,
		DataQuality: QualityConnected,
		Regime:      SHNORMAL,
		DriverSnapshot: []DriverSnapshot{
			{Name: DriverDXY, Source: "twelvedata"},
		},
	}
	// Just verify it doesn't panic
	RecordMetrics(&result)
}

// Helper
func timeNow() time.Time { return time.Now() }

