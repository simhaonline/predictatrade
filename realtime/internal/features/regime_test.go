package features

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func TestRegimeEngine_RSIZeroDoesNotTriggerMeanReversion(t *testing.T) {
	engine := NewRegimeEngine()
	candle := &types.Candle{
		Symbol:    types.SymbolXAUUSD,
		Timeframe: types.TFM1,
		Time:      time.Now(),
		Open:      decimal.NewFromFloat(2400),
		High:      decimal.NewFromFloat(2401),
		Low:       decimal.NewFromFloat(2399),
		Close:     decimal.NewFromFloat(2400),
	}

	// RSI=0 should NOT trigger oversold/MEAN_REVERSION
	ind := IndicatorFeatures{
		RSI:  decimal.Zero, // Uninitialized
		ADX:  decimal.NewFromFloat(15),
		ATR:  decimal.NewFromFloat(2),
		EMA9:  decimal.NewFromFloat(2400),
		EMA21: decimal.NewFromFloat(2400),
		EMA50: decimal.NewFromFloat(2400),
	}

	result := engine.Process(candle, ind)
	if result.Current == types.RegimeMeanReversion {
		t.Fatalf("RSI=0 should NOT trigger MEAN_REVERSION, got %s with confidence %f", result.Current, result.Confidence)
	}
	// With ADX=15, should be RANGE
	if result.Current != types.RegimeRange {
		t.Errorf("Expected RANGE for RSI=0 ADX=15, got %s", result.Current)
	}
}

func TestRegimeEngine_RSI48_ADX15_ProducesRangeNotMeanReversion(t *testing.T) {
	engine := NewRegimeEngine()
	candle := &types.Candle{
		Symbol:    types.SymbolXAUUSD,
		Timeframe: types.TFM1,
		Time:      time.Now(),
		Open:      decimal.NewFromFloat(2400),
		High:      decimal.NewFromFloat(2401),
		Low:       decimal.NewFromFloat(2399),
		Close:     decimal.NewFromFloat(2400),
	}

	ind := IndicatorFeatures{
		RSI:  decimal.NewFromFloat(48), // Mid-range, not extreme
		ADX:  decimal.NewFromFloat(15), // Below 20, ranging
		ATR:  decimal.NewFromFloat(2),
		EMA9:  decimal.NewFromFloat(2400),
		EMA21: decimal.NewFromFloat(2400),
		EMA50: decimal.NewFromFloat(2400),
	}

	result := engine.Process(candle, ind)
	if result.Current != types.RegimeRange {
		t.Errorf("RSI=48 ADX=15 should produce RANGE, got %s (confidence=%f, reason=%s)",
			result.Current, result.Confidence, result.EntryReason)
	}
}

func TestRegimeEngine_OverboughtTriggersMeanReversion(t *testing.T) {
	engine := NewRegimeEngine()
	candle := &types.Candle{
		Symbol:    types.SymbolXAUUSD,
		Timeframe: types.TFM1,
		Time:      time.Now(),
		Close:     decimal.NewFromFloat(2400),
	}

	ind := IndicatorFeatures{
		RSI:  decimal.NewFromFloat(75), // Overbought
		ADX:  decimal.NewFromFloat(15),
		ATR:  decimal.NewFromFloat(2),
		EMA9:  decimal.NewFromFloat(2400),
		EMA21: decimal.NewFromFloat(2400),
		EMA50: decimal.NewFromFloat(2400),
	}

	result := engine.Process(candle, ind)
	if result.Current != types.RegimeMeanReversion {
		t.Errorf("RSI=75 should produce MEAN_REVERSION, got %s", result.Current)
	}
	if result.Confidence != 0.7 {
		t.Errorf("Expected confidence 0.7, got %f", result.Confidence)
	}
}

func TestRegimeEngine_Hysteresis_PreventsFlickering(t *testing.T) {
	// Use custom config: 10 min hold, 2 confirmation candles
	engine := NewRegimeEngineWithConfig(10*time.Minute, 2, 0.92, 0.25)

	t0 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	candle := func(mins int) *types.Candle {
		return &types.Candle{Time: t0.Add(time.Duration(mins) * time.Minute), Close: decimal.NewFromFloat(2400)}
	}

	// Candle 1: RSI=75 → MEAN_REVERSION
	ind1 := IndicatorFeatures{RSI: decimal.NewFromFloat(75), ADX: decimal.NewFromFloat(15), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2400), EMA21: decimal.NewFromFloat(2400), EMA50: decimal.NewFromFloat(2400)}
	r1 := engine.Process(candle(0), ind1)
	if r1.Current != types.RegimeMeanReversion {
		t.Fatalf("Expected MEAN_REVERSION, got %s", r1.Current)
	}

	// Candle 2: RSI drops to 48 → should NOT immediately transition to RANGE (hysteresis)
	ind2 := IndicatorFeatures{RSI: decimal.NewFromFloat(48), ADX: decimal.NewFromFloat(15), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2400), EMA21: decimal.NewFromFloat(2400), EMA50: decimal.NewFromFloat(2400)}
	r2 := engine.Process(candle(1), ind2)
	if r2.Current != types.RegimeMeanReversion {
		t.Errorf("Hysteresis should prevent immediate transition from MEAN_REVERSION to %s, got %s", types.RegimeRange, r2.Current)
	}
	// Confidence should decay
	if r2.Confidence >= 0.7 {
		t.Errorf("Confidence should decay from 0.7, got %f", r2.Confidence)
	}
}

func TestRegimeEngine_ConfidenceDecay(t *testing.T) {
	// Use long min hold to see decay before transition occurs
	engine := NewRegimeEngineWithConfig(100*time.Minute, 10, 0.9, 0.25)

	t0 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	candle := func(mins int) *types.Candle {
		return &types.Candle{Time: t0.Add(time.Duration(mins) * time.Minute), Close: decimal.NewFromFloat(2400)}
	}

	// Enter MEAN_REVERSION with RSI=75
	ind1 := IndicatorFeatures{RSI: decimal.NewFromFloat(75), ADX: decimal.NewFromFloat(15), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2400), EMA21: decimal.NewFromFloat(2400), EMA50: decimal.NewFromFloat(2400)}
	engine.Process(candle(0), ind1)

	// RSI normalizes to 48 — confidence should decay each candle
	ind2 := IndicatorFeatures{RSI: decimal.NewFromFloat(48), ADX: decimal.NewFromFloat(15), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2400), EMA21: decimal.NewFromFloat(2400), EMA50: decimal.NewFromFloat(2400)}

	r1 := engine.Process(candle(1), ind2)
	conf1 := r1.Confidence
	r2 := engine.Process(candle(2), ind2)
	conf2 := r2.Confidence
	r3 := engine.Process(candle(3), ind2)
	conf3 := r3.Confidence

	if conf2 >= conf1 {
		t.Errorf("Confidence should decay: %f → %f → %f", conf1, conf2, conf3)
	}
	if conf3 >= conf2 {
		t.Errorf("Confidence should continue decaying: %f → %f → %f", conf1, conf2, conf3)
	}
	// Should not go below minimum
	if conf3 < 0.25 {
		t.Errorf("Confidence should not go below minimum 0.25, got %f", conf3)
	}
}

func TestRegimeEngine_TransitionToTrendingBullish(t *testing.T) {
	engine := NewRegimeEngineWithConfig(1*time.Minute, 1, 0.5, 0.25)

	t0 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	candle := func(mins int) *types.Candle {
		return &types.Candle{Time: t0.Add(time.Duration(mins) * time.Minute), Close: decimal.NewFromFloat(2400)}
	}

	// Start with MEAN_REVERSION (RSI=75)
	ind1 := IndicatorFeatures{RSI: decimal.NewFromFloat(75), ADX: decimal.NewFromFloat(15), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2410), EMA21: decimal.NewFromFloat(2400), EMA50: decimal.NewFromFloat(2390)}
	engine.Process(candle(0), ind1)

	// Transition to TRENDING_BULLISH (ADX=32, bullish EMA alignment)
	ind2 := IndicatorFeatures{
		RSI: decimal.NewFromFloat(55), ADX: decimal.NewFromFloat(32), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2415), EMA21: decimal.NewFromFloat(2405), EMA50: decimal.NewFromFloat(2395),
	}

	// Feed multiple candles to pass min hold + confirmation
	var result RegimeFeatures
	for i := 1; i <= 10; i++ {
		result = engine.Process(candle(i), ind2)
	}

	if result.Current != types.RegimeTrendingBullish {
		t.Errorf("Should transition to TRENDING_BULLISH after confirmation, got %s (reason: %s, hold: %s)",
			result.Current, result.EntryReason, result.HoldReason)
	}
}

func TestRegimeEngine_TransitionHistory(t *testing.T) {
	engine := NewRegimeEngineWithConfig(1*time.Minute, 1, 0.5, 0.25)

	t0 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	candle := func(mins int) *types.Candle {
		return &types.Candle{Time: t0.Add(time.Duration(mins) * time.Minute), Close: decimal.NewFromFloat(2400)}
	}

	// Start with MEAN_REVERSION
	engine.Process(candle(0), IndicatorFeatures{
		RSI: decimal.NewFromFloat(75), ADX: decimal.NewFromFloat(15), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2400), EMA21: decimal.NewFromFloat(2400), EMA50: decimal.NewFromFloat(2400),
	})

	// Transition to TRENDING_BULLISH
	ind := IndicatorFeatures{
		RSI: decimal.NewFromFloat(55), ADX: decimal.NewFromFloat(32), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2415), EMA21: decimal.NewFromFloat(2405), EMA50: decimal.NewFromFloat(2395),
	}
	for i := 1; i <= 10; i++ {
		engine.Process(candle(i), ind)
	}

	transitions := engine.GetTransitions()
	if len(transitions) < 1 {
		t.Fatalf("Expected at least 1 transition, got %d", len(transitions))
	}

	lastTransition := transitions[len(transitions)-1]
	if lastTransition.From != types.RegimeMeanReversion {
		t.Errorf("Expected transition FROM MEAN_REVERSION, got %s", lastTransition.From)
	}
	if lastTransition.To != types.RegimeTrendingBullish {
		t.Errorf("Expected transition TO TRENDING_BULLISH, got %s", lastTransition.To)
	}
}

func TestRegimeEngine_Versioning(t *testing.T) {
	engine := NewRegimeEngine()
	candle := &types.Candle{
		Time:  time.Now(),
		Close: decimal.NewFromFloat(2400),
	}
	ind := IndicatorFeatures{
		RSI: decimal.NewFromFloat(48), ADX: decimal.NewFromFloat(15), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2400), EMA21: decimal.NewFromFloat(2400), EMA50: decimal.NewFromFloat(2400),
	}
	result := engine.Process(candle, ind)
	if result.RegimeEngineVersion != RegimeEngineVersion {
		t.Errorf("Expected version %s, got %s", RegimeEngineVersion, result.RegimeEngineVersion)
	}
}

func TestRegimeEngine_DiagnosticsFields(t *testing.T) {
	engine := NewRegimeEngine()
	candle := &types.Candle{
		Time:  time.Now(),
		Close: decimal.NewFromFloat(2400),
	}
	ind := IndicatorFeatures{
		RSI: decimal.NewFromFloat(75), ADX: decimal.NewFromFloat(15), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2400), EMA21: decimal.NewFromFloat(2400), EMA50: decimal.NewFromFloat(2400),
	}
	result := engine.Process(candle, ind)

	// Verify all diagnostic fields are populated
	if result.EntryReason == "" {
		t.Error("EntryReason should be populated")
	}
	if result.EnteredAt.IsZero() {
		t.Error("EnteredAt should be populated")
	}
	if result.RawRegime == "" {
		t.Error("RawRegime should be populated")
	}
	if result.RegimeEngineVersion == "" {
		t.Error("RegimeEngineVersion should be populated")
	}
}

func TestRegimeEngine_Reset(t *testing.T) {
	engine := NewRegimeEngine()
	candle := &types.Candle{
		Time:  time.Now(),
		Close: decimal.NewFromFloat(2400),
	}
	ind := IndicatorFeatures{
		RSI: decimal.NewFromFloat(75), ADX: decimal.NewFromFloat(15), ATR: decimal.NewFromFloat(2),
		EMA9: decimal.NewFromFloat(2400), EMA21: decimal.NewFromFloat(2400), EMA50: decimal.NewFromFloat(2400),
	}
	engine.Process(candle, ind)
	engine.Reset()

	if engine.GetCurrentRegime() != types.RegimeRange {
		t.Errorf("After reset, regime should be RANGE, got %s", engine.GetCurrentRegime())
	}
	if len(engine.GetTransitions()) != 0 {
		t.Errorf("After reset, transitions should be empty, got %d", len(engine.GetTransitions()))
	}
}
