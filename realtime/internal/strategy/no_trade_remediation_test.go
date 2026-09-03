package strategy

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// Test ATR propagation fix
func TestATRPropagationToGates(t *testing.T) {
	engine := NewTrendSwing()

	state := &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		Timestamp:    time.Now(),
		CurrentPrice: decimal.NewFromFloat(2400),
		Indicators: features.IndicatorFeatures{
			ATR:        decimal.NewFromFloat(3.5), // Real ATR
			RSI:        decimal.NewFromFloat(55),
			ADX:        decimal.NewFromFloat(32),
			EMA9:       decimal.NewFromFloat(2415),
			EMA21:      decimal.NewFromFloat(2405),
			EMA50:      decimal.NewFromFloat(2395),
			EMA100:     decimal.NewFromFloat(2400),
			EMA200:     decimal.NewFromFloat(2395),
			SMA200:     decimal.NewFromFloat(2400),
			MACDMain:   decimal.NewFromFloat(5),
			MACDSignal: decimal.NewFromFloat(3),
			OsMA:       decimal.NewFromFloat(2),
			CCI:        decimal.NewFromFloat(80),
		},
		Regime:    features.RegimeFeatures{Current: types.RegimeTrendingBullish, Confidence: 0.8},
		Session:   features.SessionFeatures{CurrentSession: "LONDON", NewsRisk: "LOW"},
		Quality:   types.QualityAuthoritative,
		MTF:       features.MTFFeatures{Score: 60, States: map[types.Timeframe]int{types.TFM1: 1, types.TFM5: 1}},
		Structure: features.StructureFeatures{CurrentTrend: "bullish"},
	}

	result := engine.Evaluate(state)

	// Entry and SL should be non-zero (ATR propagated)
	if result.Direction == types.DirectionBuy || result.Direction == types.DirectionSell {
		if result.EntryPrice.IsZero() {
			t.Error("EntryPrice should not be zero when ATR is valid")
		}
		if result.StopLoss.IsZero() {
			t.Error("StopLoss should not be zero when ATR is valid")
		}
	}
}

// Test RANGE regime no longer starves StandardScalping
func TestRangeRegimeStandardScalpingProducesEvidence(t *testing.T) {
	scalping := NewStandardScalping()

	state := &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		Timestamp:    time.Now(),
		CurrentPrice: decimal.NewFromFloat(2395), // Below BB lower
		Indicators: features.IndicatorFeatures{
			ATR:         decimal.NewFromFloat(3.0),
			RSI:         decimal.NewFromFloat(28), // Oversold in range
			ADX:         decimal.NewFromFloat(14), // Range
			EMA9:        decimal.NewFromFloat(2400),
			EMA21:       decimal.NewFromFloat(2400),
			EMA50:       decimal.NewFromFloat(2400),
			BollUpper:   decimal.NewFromFloat(2410),
			BollLower:   decimal.NewFromFloat(2390),
			BollMiddle:  decimal.NewFromFloat(2400),
			StochMain:   decimal.NewFromFloat(15),
			StochSignal: decimal.NewFromFloat(20),
			CCI:         decimal.NewFromFloat(-120),
			MACDMain:    decimal.NewFromFloat(-1),
			MACDSignal:  decimal.NewFromFloat(-0.5),
			OsMA:        decimal.NewFromFloat(-0.5),
		},
		Regime:  features.RegimeFeatures{Current: types.RegimeRange, Confidence: 0.6},
		Session: features.SessionFeatures{CurrentSession: "LONDON", NewsRisk: "LOW"},
		Quality: types.QualityAuthoritative,
		MTF:     features.MTFFeatures{Score: 10, States: map[types.Timeframe]int{types.TFM1: 1, types.TFM5: 0}},
		VWAP:    features.VWAPFeatures{SessionVWAP: decimal.NewFromFloat(2400)},
		Candle: features.CandleIntelligence{
			IsRejection: true,
			IsBullish:   true,
			Range:       decimal.NewFromFloat(3),
		},
	}

	result := scalping.Evaluate(state)

	// Should NOT be regime-rejected — RANGE is accepted by StandardScalping
	hasRegimeMismatch := false
	for _, rc := range result.ReasonCodes {
		if rc == types.NTRegimeMismatch {
			hasRegimeMismatch = true
		}
	}
	if hasRegimeMismatch {
		t.Error("StandardScalping should accept RANGE regime, but got NTRegimeMismatch")
	}

	// Should have some evidence (range evidence was added)
	if len(result.Evidence) == 0 {
		t.Error("StandardScalping should produce evidence in RANGE regime")
	}

	// Long score should be non-zero (BB lower touch + RSI oversold + CCI oversold + rejection)
	longScore, _ := result.LongScore.Float64()
	if longScore == 0 {
		t.Errorf("LongScore should be non-zero in range with oversold conditions, got %f", longScore)
	}
}

// Test RANGE regime with UltraScalping now accepts MEAN_REVERSION
func TestUltraScalpingAcceptsMeanReversion(t *testing.T) {
	ultra := NewUltraScalping()

	state := &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		Timestamp:    time.Now(),
		CurrentPrice: decimal.NewFromFloat(2395),
		Indicators: features.IndicatorFeatures{
			ATR:         decimal.NewFromFloat(2.0),
			RSI:         decimal.NewFromFloat(25),
			ADX:         decimal.NewFromFloat(12),
			EMA9:        decimal.NewFromFloat(2400),
			EMA21:       decimal.NewFromFloat(2400),
			EMA50:       decimal.NewFromFloat(2400),
			BollUpper:   decimal.NewFromFloat(2410),
			BollLower:   decimal.NewFromFloat(2390),
			StochMain:   decimal.NewFromFloat(10),
			StochSignal: decimal.NewFromFloat(15),
			OsMA:        decimal.NewFromFloat(-0.3),
		},
		Regime:  features.RegimeFeatures{Current: types.RegimeMeanReversion, Confidence: 0.7},
		Session: features.SessionFeatures{CurrentSession: "LONDON", NewsRisk: "LOW"},
		Quality: types.QualityAuthoritative,
		MTF:     features.MTFFeatures{Score: 5, States: map[types.Timeframe]int{types.TFM1: 1, types.TFM5: 0}},
		Candle:  features.CandleIntelligence{Range: decimal.NewFromFloat(2)},
	}

	result := ultra.Evaluate(state)

	// Should NOT be regime rejected
	hasRegimeMismatch := false
	for _, rc := range result.ReasonCodes {
		if rc == types.NTRegimeMismatch {
			hasRegimeMismatch = true
		}
	}
	if hasRegimeMismatch {
		t.Error("UltraScalping should accept MEAN_REVERSION regime after fix")
	}
}

// Test TrendSwing remains selective in RANGE
func TestTrendSwingRejectsRangeRegime(t *testing.T) {
	trend := NewTrendSwing()

	state := &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		Timestamp:    time.Now(),
		CurrentPrice: decimal.NewFromFloat(2400),
		Indicators: features.IndicatorFeatures{
			ATR:   decimal.NewFromFloat(2.0),
			RSI:   decimal.NewFromFloat(50),
			ADX:   decimal.NewFromFloat(14),
			EMA9:  decimal.NewFromFloat(2400),
			EMA21: decimal.NewFromFloat(2400),
			EMA50: decimal.NewFromFloat(2400),
		},
		Regime:  features.RegimeFeatures{Current: types.RegimeRange},
		Session: features.SessionFeatures{CurrentSession: "LONDON", NewsRisk: "LOW"},
		Quality: types.QualityAuthoritative,
	}

	result := trend.Evaluate(state)

	// Should be NO-TRADE with regime mismatch
	if result.Direction != types.DirectionNoTrade {
		t.Errorf("TrendSwing should reject RANGE regime, got %s", result.Direction)
	}

	// prompt.md Section 10: RANGE now goes through transition analysis
	// NTNoTrendTransition is the correct reason when no transition evidence exists
	hasValidReason := false
	for _, rc := range result.ReasonCodes {
		if rc == types.NTRegimeMismatch || rc == types.NTNoTrendTransition {
			hasValidReason = true
		}
	}
	if !hasValidReason {
		t.Error("TrendSwing should have NTRegimeMismatch or NTNoTrendTransition reason code in RANGE")
	}
}

// Test feature readiness
func TestFeatureReadinessDetection(t *testing.T) {
	state := &features.MarketState{
		Indicators: features.IndicatorFeatures{
			ATR:   decimal.NewFromFloat(2.0),
			RSI:   decimal.NewFromFloat(50),
			ADX:   decimal.Zero, // Not ready
			EMA9:  decimal.NewFromFloat(2400),
			EMA21: decimal.NewFromFloat(2400),
			EMA50: decimal.Zero, // Not ready
		},
	}

	readiness := CheckFeatureReadiness(state)

	if !readiness.ATR {
		t.Error("ATR should be ready")
	}
	if !readiness.RSI {
		t.Error("RSI should be ready")
	}
	if readiness.ADX {
		t.Error("ADX should NOT be ready (zero)")
	}
	if readiness.EMA50 {
		t.Error("EMA50 should NOT be ready (zero)")
	}

	missing := readiness.MissingFeatures()
	if len(missing) == 0 {
		t.Error("Should have missing features")
	}

	pct := readiness.ReadinessPercent()
	if pct <= 0 || pct > 100 {
		t.Errorf("ReadinessPercent should be 0-100, got %f", pct)
	}
}

// Test ATR zero produces NTATRNotReady not NTSystemDegraded
func TestATRZeroProducesCorrectReasonCode(t *testing.T) {
	scalping := NewStandardScalping()

	state := &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		CurrentPrice: decimal.NewFromFloat(2400),
		Indicators: features.IndicatorFeatures{
			ATR: decimal.Zero, // ATR not ready
		},
		Regime:  features.RegimeFeatures{Current: types.RegimeTrendingBullish},
		Session: features.SessionFeatures{CurrentSession: "LONDON", NewsRisk: "LOW"},
		Quality: types.QualityAuthoritative,
	}

	result := scalping.Evaluate(state)

	if result.Direction != types.DirectionError {
		t.Errorf("Should be ERROR when ATR is zero, got %s", result.Direction)
	}

	hasATRNotReady := false
	for _, rc := range result.ReasonCodes {
		if rc == types.NTATRNotReady {
			hasATRNotReady = true
		}
	}
	if !hasATRNotReady {
		t.Error("Should have NTATRNotReady reason code when ATR is zero")
	}
}

// Test contribution trace
func TestContributionTrace(t *testing.T) {
	strat := NewStandardScalping()
	state := &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		CurrentPrice: decimal.NewFromFloat(2400),
		Indicators: features.IndicatorFeatures{
			ATR:        decimal.NewFromFloat(2),
			RSI:        decimal.NewFromFloat(55),
			ADX:        decimal.NewFromFloat(25),
			EMA9:       decimal.NewFromFloat(2410),
			EMA21:      decimal.NewFromFloat(2400),
			EMA50:      decimal.NewFromFloat(2395),
			MACDMain:   decimal.NewFromFloat(2),
			MACDSignal: decimal.NewFromFloat(1),
			OsMA:       decimal.NewFromFloat(1),
		},
		Regime:  features.RegimeFeatures{Current: types.RegimeTrendingBullish},
		Session: features.SessionFeatures{CurrentSession: "LONDON", NewsRisk: "LOW"},
		Quality: types.QualityAuthoritative,
		MTF:     features.MTFFeatures{Score: 50, States: map[types.Timeframe]int{types.TFM1: 1, types.TFM5: 1}},
		Candle:  features.CandleIntelligence{Range: decimal.NewFromFloat(2)},
	}

	result := strat.Evaluate(state)
	readiness := CheckFeatureReadiness(state)
	trace := BuildContributionTrace(strat.ID(), result.Evidence, result, readiness)

	if trace.StrategyID != types.StrategyStandardScalping {
		t.Error("Trace should have correct strategy ID")
	}
	if trace.ReadinessPercent <= 0 {
		t.Error("Readiness percent should be positive")
	}
}
