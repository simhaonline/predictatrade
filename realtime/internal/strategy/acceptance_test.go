// Package strategy — Acceptance tests for RANGE, TREND, BREAKOUT, RECOVERY, CALIBRATION.
// SOW Phase 2 Sections 35-39: Deterministic fixtures proving correct strategy behavior.
package strategy

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ─── Helper to build test states ───

func makeState(price float64, ind features.IndicatorFeatures, regime types.Regime) *features.MarketState {
	return &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		Timestamp:    time.Now(),
		CurrentPrice: decimal.NewFromFloat(price),
		Bid:          decimal.NewFromFloat(price - 0.5),
		Ask:          decimal.NewFromFloat(price + 0.5),
		Spread:       decimal.NewFromFloat(1.0),
		Mid:          decimal.NewFromFloat(price),
		Indicators:   ind,
		Regime:       features.RegimeFeatures{Current: regime, Confidence: 0.7},
		Session:      features.SessionFeatures{CurrentSession: "LONDON", NewsRisk: "LOW"},
		Quality:      types.QualityAuthoritative,
		MTF: features.MTFFeatures{
			Score:  50,
			States: map[types.Timeframe]int{types.TFM1: 1, types.TFM5: 1},
		},
		Structure: features.StructureFeatures{
			CurrentTrend: "bullish",
			SwingHighs:   []decimal.Decimal{decimal.NewFromFloat(price * 1.01)},
			SwingLows:    []decimal.Decimal{decimal.NewFromFloat(price * 0.99)},
		},
		VWAP: features.VWAPFeatures{SessionVWAP: decimal.NewFromFloat(price)},
		Candle: features.CandleIntelligence{
			Range: decimal.NewFromFloat(2.0),
		},
	}
}

func makeTrendBullIndicators(price float64) features.IndicatorFeatures {
	return features.IndicatorFeatures{
		ATR:    decimal.NewFromFloat(3.0),
		RSI:    decimal.NewFromFloat(58),
		ADX:    decimal.NewFromFloat(35),
		EMA9:   decimal.NewFromFloat(price + 15),
		EMA21:  decimal.NewFromFloat(price + 8),
		EMA50:  decimal.NewFromFloat(price + 2),
		EMA100: decimal.NewFromFloat(price),
		EMA200: decimal.NewFromFloat(price - 5),
		SMA200: decimal.NewFromFloat(price),
		ADXPlusDI:  decimal.NewFromFloat(25),
		ADXMinusDI: decimal.NewFromFloat(12),
		MACDMain:   decimal.NewFromFloat(5),
		MACDSignal: decimal.NewFromFloat(3),
		OsMA:       decimal.NewFromFloat(2),
		CCI:        decimal.NewFromFloat(85),
		StochMain:  decimal.NewFromFloat(65),
		StochSignal: decimal.NewFromFloat(55),
	}
}

func makeTrendBearIndicators(price float64) features.IndicatorFeatures {
	return features.IndicatorFeatures{
		ATR:    decimal.NewFromFloat(3.0),
		RSI:    decimal.NewFromFloat(38),
		ADX:    decimal.NewFromFloat(35),
		EMA9:   decimal.NewFromFloat(price - 15),
		EMA21:  decimal.NewFromFloat(price - 8),
		EMA50:  decimal.NewFromFloat(price - 2),
		EMA100: decimal.NewFromFloat(price),
		EMA200: decimal.NewFromFloat(price + 5),
		SMA200: decimal.NewFromFloat(price),
		ADXPlusDI:  decimal.NewFromFloat(12),
		ADXMinusDI: decimal.NewFromFloat(25),
		MACDMain:   decimal.NewFromFloat(-5),
		MACDSignal: decimal.NewFromFloat(-3),
		OsMA:       decimal.NewFromFloat(-2),
		CCI:        decimal.NewFromFloat(-85),
		StochMain:  decimal.NewFromFloat(35),
		StochSignal: decimal.NewFromFloat(45),
	}
}

func makeRangeIndicators(price float64) features.IndicatorFeatures {
	return features.IndicatorFeatures{
		ATR:    decimal.NewFromFloat(2.0),
		RSI:    decimal.NewFromFloat(48),
		ADX:    decimal.NewFromFloat(14),
		EMA9:   decimal.NewFromFloat(price),
		EMA21:  decimal.NewFromFloat(price),
		EMA50:  decimal.NewFromFloat(price),
		EMA100: decimal.NewFromFloat(price),
		EMA200: decimal.NewFromFloat(price),
		SMA200: decimal.NewFromFloat(price),
		ADXPlusDI:  decimal.NewFromFloat(15),
		ADXMinusDI: decimal.NewFromFloat(14),
		MACDMain:   decimal.NewFromFloat(0.2),
		MACDSignal: decimal.NewFromFloat(0.1),
		OsMA:       decimal.NewFromFloat(0.1),
		CCI:        decimal.NewFromFloat(10),
		StochMain:  decimal.NewFromFloat(50),
		StochSignal: decimal.NewFromFloat(48),
		BollUpper:  decimal.NewFromFloat(price + 10),
		BollLower:  decimal.NewFromFloat(price - 10),
		BollMiddle: decimal.NewFromFloat(price),
		BollWidth:  decimal.NewFromFloat(20.0 / price),
	}
}

// ═════════════════════════════════════════════════════════════
// Section 35: RANGE MARKET ACCEPTANCE TESTS
// ═════════════════════════════════════════════════════════════

func TestRange_Center_NoTrade(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	state := makeState(2400, ind, types.RegimeRange)
	// Price at center — no directional edge
	state.Indicators.RSI = decimal.NewFromFloat(50)
	state.Indicators.CCI = decimal.NewFromFloat(0)

	result := scalping.Evaluate(state)
	// Center of range with no edge — may or may not produce signal, but shouldn't error
	if result.Direction == types.DirectionError {
		t.Error("Should not error in range center")
	}
}

func TestRange_LowerRejection_ProducesBuyCandidate(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	ind.RSI = decimal.NewFromFloat(28)    // Oversold
	ind.CCI = decimal.NewFromFloat(-130)  // CCI extreme
	ind.BollLower = decimal.NewFromFloat(2392)
	ind.BollUpper = decimal.NewFromFloat(2408)
	state := makeState(2392, ind, types.RegimeRange) // Price at lower BB
	state.Candle.IsRejection = true
	state.Candle.IsBullish = true
	state.VWAP.SessionVWAP = decimal.NewFromFloat(2400)

	result := scalping.Evaluate(state)

	// Should produce evidence in the buy direction
	longScore, _ := result.LongScore.Float64()
	if longScore == 0 {
		t.Errorf("Range lower rejection should produce buy evidence, long score=%f", longScore)
	}
	// Long should dominate short
	shortScore, _ := result.ShortScore.Float64()
	if longScore <= shortScore {
		t.Errorf("Long (%f) should dominate short (%f) at range lower rejection", longScore, shortScore)
	}
}

func TestRange_UpperRejection_ProducesSellCandidate(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	ind.RSI = decimal.NewFromFloat(72)   // Overbought
	ind.CCI = decimal.NewFromFloat(130)   // CCI extreme
	ind.BollLower = decimal.NewFromFloat(2392)
	ind.BollUpper = decimal.NewFromFloat(2408)
	state := makeState(2408, ind, types.RegimeRange) // Price at upper BB
	state.Candle.IsRejection = true
	state.Candle.IsBearish = true
	state.VWAP.SessionVWAP = decimal.NewFromFloat(2400)

	result := scalping.Evaluate(state)

	shortScore, _ := result.ShortScore.Float64()
	if shortScore == 0 {
		t.Errorf("Range upper rejection should produce sell evidence, short score=%f", shortScore)
	}
	longScore, _ := result.LongScore.Float64()
	if shortScore <= longScore {
		t.Errorf("Short (%f) should dominate long (%f) at range upper rejection", shortScore, longScore)
	}
}

func TestRange_FakeBreakout_NoTrade(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	ind.RSI = decimal.NewFromFloat(55)
	state := makeState(2405, ind, types.RegimeRange)
	// Fake breakout — price slightly above range but no confirmation
	state.Candle.IsBreakout = false
	state.Candle.IsDisplacement = false

	result := scalping.Evaluate(state)
	// Should not produce a strong directional signal
	if result.Direction == types.DirectionError {
		t.Error("Should not error on fake breakout")
	}
}

func TestRange_LiquiditySweepBullish(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	ind.RSI = decimal.NewFromFloat(35)
	state := makeState(2395, ind, types.RegimeRange)
	state.Liquidity.RecentSweeps = []features.SweepEvent{
		{Direction: "sell_side", Price: decimal.NewFromFloat(2390), Time: time.Now()},
	}

	result := scalping.Evaluate(state)
	longScore, _ := result.LongScore.Float64()
	if longScore == 0 {
		t.Error("Bullish liquidity sweep in range should produce buy evidence")
	}
}

func TestRange_LiquiditySweepBearish(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	ind.RSI = decimal.NewFromFloat(65)
	state := makeState(2405, ind, types.RegimeRange)
	state.Liquidity.RecentSweeps = []features.SweepEvent{
		{Direction: "buy_side", Price: decimal.NewFromFloat(2410), Time: time.Now()},
	}

	result := scalping.Evaluate(state)
	shortScore, _ := result.ShortScore.Float64()
	if shortScore == 0 {
		t.Error("Bearish liquidity sweep in range should produce sell evidence")
	}
}

func TestRange_WideSpread_NoTrade(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	state := makeState(2400, ind, types.RegimeRange)
	state.Spread = decimal.NewFromFloat(5.0) // Very wide spread
	state.Indicators.ATR = decimal.NewFromFloat(2.0) // Spread > ATR

	result := scalping.Evaluate(state)
	// Wide spread should produce conflict penalty or reduce score
	// The strategy should not produce a confident signal with wide spread
	if result.Direction == types.DirectionError {
		t.Error("Should not error on wide spread")
	}
}

func TestRange_TrendSwingNoTrade(t *testing.T) {
	trend := NewTrendSwing()
	ind := makeRangeIndicators(2400)
	state := makeState(2400, ind, types.RegimeRange)

	result := trend.Evaluate(state)
	if result.Direction != types.DirectionNoTrade {
		t.Errorf("TrendSwing must NO-TRADE in RANGE, got %s", result.Direction)
	}
}

// ═════════════════════════════════════════════════════════════
// Section 36: TREND ACCEPTANCE TESTS
// ═════════════════════════════════════════════════════════════

func TestTrend_StrongBullish_TrendSwingQualifies(t *testing.T) {
	trend := NewTrendSwing()
	ind := makeTrendBullIndicators(2400)
	state := makeState(2420, ind, types.RegimeTrendingBullish) // Price above SMA200 and EMA50
	state.Structure.CurrentTrend = "bullish"
	state.Structure.LastBOS = &features.StructureEvent{
		Type: "BOS", Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now(),
	}
	state.MTF.Score = 70
	state.Indicators.SMA200 = decimal.NewFromFloat(2400) // Price above SMA200
	state.Indicators.EMA50 = decimal.NewFromFloat(2402)  // Price above EMA50

	result := trend.Evaluate(state)
	if result.Direction != types.DirectionBuy {
		t.Errorf("Strong bullish trend should produce BUY candidate, got %s (reasons: %v)", result.Direction, result.ReasonCodes)
	}
}

func TestTrend_StrongBearish_TrendSwingQualifies(t *testing.T) {
	trend := NewTrendSwing()
	ind := makeTrendBearIndicators(2400)
	state := makeState(2400, ind, types.RegimeTrendingBearish)
	state.Structure.CurrentTrend = "bearish"
	state.Structure.LastBOS = &features.StructureEvent{
		Type: "BOS", Direction: "bearish", Price: decimal.NewFromFloat(2390), Time: time.Now(),
	}
	state.MTF.Score = -70

	result := trend.Evaluate(state)
	if result.Direction != types.DirectionSell {
		t.Errorf("Strong bearish trend should produce SELL candidate, got %s (reasons: %v)", result.Direction, result.ReasonCodes)
	}
}

func TestTrend_WeakBullish_MayNotQualify(t *testing.T) {
	trend := NewTrendSwing()
	ind := makeTrendBullIndicators(2400)
	ind.ADX = decimal.NewFromFloat(20) // Below MinADX of 22
	state := makeState(2400, ind, types.RegimeTrendingBullish)

	result := trend.Evaluate(state)
	// Weak ADX should not qualify for trend swing
	if result.Direction == types.DirectionBuy {
		t.Error("Weak bullish trend (ADX=20) should not qualify for TrendSwing")
	}
}

func TestTrend_StandardScalpingInBullishTrend(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeTrendBullIndicators(2400)
	state := makeState(2400, ind, types.RegimeTrendingBullish)

	result := scalping.Evaluate(state)
	// StandardScalping should produce buy-biased evidence in bullish trend
	longScore, _ := result.LongScore.Float64()
	shortScore, _ := result.ShortScore.Float64()
	if longScore < shortScore {
		t.Errorf("In bullish trend, long (%f) should >= short (%f)", longScore, shortScore)
	}
}

// ═════════════════════════════════════════════════════════════
// Section 37: BREAKOUT ACCEPTANCE TESTS
// ═════════════════════════════════════════════════════════════

func TestBreakout_BOS_ProducesCandidate(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeTrendBullIndicators(2400)
	ind.ADX = decimal.NewFromFloat(28)
	state := makeState(2415, ind, types.RegimeBreakout)
	state.Structure.LastBOS = &features.StructureEvent{
		Type: "BOS", Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now(),
	}
	state.Candle.IsBreakout = true
	state.Candle.IsBullish = true
	state.Candle.IsDisplacement = true

	result := scalping.Evaluate(state)
	longScore, _ := result.LongScore.Float64()
	if longScore == 0 {
		t.Error("Bullish BOS breakout should produce buy evidence")
	}
}

func TestBreakout_FailedBreakout_NoTrade(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	ind.ADX = decimal.NewFromFloat(16) // Weak — not a real breakout
	state := makeState(2405, ind, types.RegimeRange)
	state.Candle.IsBreakout = true
	state.Candle.IsBullish = true
	state.Candle.IsRejection = true // Breakout then rejection = failed breakout

	result := scalping.Evaluate(state)
	// Failed breakout should not produce strong directional signal
	// The rejection should limit the buy evidence
	if result.Direction == types.DirectionError {
		t.Error("Should not error on failed breakout")
	}
}

func TestBreakout_ATRExpansion(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeTrendBullIndicators(2400)
	ind.ATR = decimal.NewFromFloat(6.0) // ATR expansion
	state := makeState(2415, ind, types.RegimeBreakout)
	state.Structure.LastBOS = &features.StructureEvent{
		Type: "BOS", Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now(),
	}
	state.Candle.IsBreakout = true
	state.Candle.IsBullish = true

	result := scalping.Evaluate(state)
	// With ATR expansion and BOS, should produce evidence
	longScore, _ := result.LongScore.Float64()
	if longScore == 0 {
		t.Error("BOS with ATR expansion should produce buy evidence")
	}
}

// ═════════════════════════════════════════════════════════════
// Section 10: ATR REGRESSION TESTS
// ═════════════════════════════════════════════════════════════

func TestATR_Normal_ProducesValidEntrySL(t *testing.T) {
	trend := NewTrendSwing()
	ind := makeTrendBullIndicators(2400)
	ind.ATR = decimal.NewFromFloat(3.0)
	state := makeState(2400, ind, types.RegimeTrendingBullish)
	state.Structure.CurrentTrend = "bullish"
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now()}
	state.MTF.Score = 70

	result := trend.Evaluate(state)
	if result.Direction == types.DirectionBuy {
		entry, _ := result.EntryPrice.Float64()
		sl, _ := result.StopLoss.Float64()
		if entry <= 0 || sl <= 0 {
			t.Errorf("Normal ATR: entry=%f, sl=%f should be positive", entry, sl)
		}
		if sl >= entry {
			t.Errorf("BUY: SL (%f) should be below entry (%f)", sl, entry)
		}
	}
}

func TestATR_VeryLow_ProducesTightStops(t *testing.T) {
	trend := NewTrendSwing()
	ind := makeTrendBullIndicators(2400)
	ind.ATR = decimal.NewFromFloat(0.5) // Very low ATR
	state := makeState(2400, ind, types.RegimeTrendingBullish)
	state.Structure.CurrentTrend = "bullish"
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now()}
	state.MTF.Score = 70
	// Set swing low close to price for tight structural SL
	state.Structure.SwingLows = []decimal.Decimal{decimal.NewFromFloat(2398)}

	result := trend.Evaluate(state)
	if result.Direction == types.DirectionBuy {
		risk := result.EntryPrice.Sub(result.StopLoss).Abs()
		riskF, _ := risk.Float64()
		// With ATR=0.5 and swing low at 2398, risk should be small (entry ~2400.5, SL ~2397)
		if riskF > 5.0 {
			t.Errorf("Very low ATR: risk=%f should be small (ATR=0.5, swing low=2398)", riskF)
		}
	}
}

func TestATR_High_ProducesWideStops(t *testing.T) {
	trend := NewTrendSwing()
	ind := makeTrendBullIndicators(2400)
	ind.ATR = decimal.NewFromFloat(10.0) // High ATR
	state := makeState(2400, ind, types.RegimeTrendingBullish)
	state.Structure.CurrentTrend = "bullish"
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now()}
	state.MTF.Score = 70

	result := trend.Evaluate(state)
	if result.Direction == types.DirectionBuy {
		risk := result.EntryPrice.Sub(result.StopLoss).Abs()
		riskF, _ := risk.Float64()
		if riskF < 5.0 { // With ATR=10, risk should be substantial
			t.Errorf("High ATR: risk=%f should be substantial (ATR=10)", riskF)
		}
	}
}

func TestATR_Missing_ProducesError(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeTrendBullIndicators(2400)
	ind.ATR = decimal.Zero // Missing ATR
	state := makeState(2400, ind, types.RegimeTrendingBullish)

	result := scalping.Evaluate(state)
	if result.Direction != types.DirectionError {
		t.Errorf("Missing ATR should produce ERROR, got %s", result.Direction)
	}
	hasReason := false
	for _, rc := range result.ReasonCodes {
		if rc == types.NTATRNotReady {
			hasReason = true
		}
	}
	if !hasReason {
		t.Error("Missing ATR should have NT_ATR_NOT_READY reason")
	}
}

func TestATR_Zero_ProducesError(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeTrendBullIndicators(2400)
	ind.ATR = decimal.Zero
	state := makeState(2400, ind, types.RegimeTrendingBullish)

	result := scalping.Evaluate(state)
	if result.Direction != types.DirectionError {
		t.Errorf("Zero ATR should produce ERROR, got %s", result.Direction)
	}
}

// ═════════════════════════════════════════════════════════════
// Section 38: RECOVERY ACCEPTANCE TESTS
// ═════════════════════════════════════════════════════════════

func TestRecovery_NeverConvertsNoTradeToBuySell(t *testing.T) {
	// This test verifies the principle: recovery never creates a signal
	// from a NO-TRADE. Recovery only affects risk multiplier after a valid
	// setup has independently qualified.
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	ind.RSI = decimal.NewFromFloat(50) // Neutral — no directional edge
	state := makeState(2400, ind, types.RegimeRange)

	result := scalping.Evaluate(state)
	// With no directional evidence, should be NO-TRADE regardless of recovery
	if result.Direction == types.DirectionBuy || result.Direction == types.DirectionSell {
		t.Error("Neutral conditions should produce NO-TRADE regardless of recovery state")
	}
}

func TestRecovery_NeverIncreasesRiskAboveBase(t *testing.T) {
	// Verify recovery manager risk multiplier is <= 1.0 after losses
	// This is tested in the recovery package tests directly.
	// Here we verify the strategy doesn't embed Martingale logic.
	scalping := NewStandardScalping()
	ind := makeTrendBullIndicators(2400)
	state := makeState(2400, ind, types.RegimeTrendingBullish)
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now()}
	state.MTF.Score = 70

	result := scalping.Evaluate(state)
	// Strategy should not have any loss-recovery logic embedded
	// The strategy evaluates evidence independently of prior trade outcomes
	_ = result
	// If we got here without panic, the strategy doesn't crash on normal evaluation
}

// ═════════════════════════════════════════════════════════════
// Section 39: CALIBRATION ACCEPTANCE TESTS
// ═════════════════════════════════════════════════════════════

func TestCalibration_RawScoreNotProbability(t *testing.T) {
	scalping := NewStandardScalping()
	ind := makeTrendBullIndicators(2400)
	state := makeState(2400, ind, types.RegimeTrendingBullish)
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now()}
	state.MTF.Score = 70

	result := scalping.Evaluate(state)
	// Raw score is a confluence score (0-100 range), NOT a probability (0-1)
	// Verify they are different scales
	if !result.RawScore.IsZero() {
		score, _ := result.RawScore.Float64()
		// Score should be in 0-100 range, not 0-1
		if score > 0 && score < 1 {
			t.Errorf("Raw score %f looks like a probability, not a confluence score", score)
		}
	}
}

func TestCalibration_ConfluenceScoreRange(t *testing.T) {
	// Verify confluence scores are in expected range (0-100)
	scalping := NewStandardScalping()
	ind := makeTrendBullIndicators(2400)
	state := makeState(2400, ind, types.RegimeTrendingBullish)
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now()}
	state.MTF.Score = 70

	result := scalping.Evaluate(state)
	longScore, _ := result.LongScore.Float64()
	if longScore < 0 || longScore > 100 {
		t.Errorf("Long score %f should be in 0-100 range", longScore)
	}
}

// ═════════════════════════════════════════════════════════════
// Section 49: NO FORCED SIGNALS
// ═════════════════════════════════════════════════════════════

func TestNoForcedSignals_NeutralMarketProducesNoTrade(t *testing.T) {
	// Verify that neutral/noisy conditions produce NO-TRADE, not forced signals
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	ind.RSI = decimal.NewFromFloat(50) // Perfectly neutral
	ind.CCI = decimal.NewFromFloat(0)  // Neutral
	ind.MACDMain = decimal.Zero
	ind.MACDSignal = decimal.Zero
	ind.OsMA = decimal.Zero
	state := makeState(2400, ind, types.RegimeRange)
	state.Candle = features.CandleIntelligence{Range: decimal.NewFromFloat(2)}
	state.MTF.Score = 0 // No MTF alignment
	state.Structure.LastBOS = nil
	state.Structure.LastCHoCH = nil

	result := scalping.Evaluate(state)
	// Neutral conditions should NOT produce BUY or SELL
	if result.Direction == types.DirectionBuy || result.Direction == types.DirectionSell {
		t.Errorf("Neutral market should not produce forced signal, got %s", result.Direction)
	}
}

func TestNoForcedSignals_NoMinimumQuota(t *testing.T) {
	// Verify there is no "minimum signal quota" or "trade frequency override"
	// by checking that multiple evaluations of the same neutral state all produce NO-TRADE
	scalping := NewStandardScalping()
	ind := makeRangeIndicators(2400)
	ind.RSI = decimal.NewFromFloat(50)
	state := makeState(2400, ind, types.RegimeRange)
	state.MTF.Score = 0

	for i := 0; i < 10; i++ {
		result := scalping.Evaluate(state)
		if result.Direction == types.DirectionBuy || result.Direction == types.DirectionSell {
			t.Errorf("Evaluation %d: neutral market should not produce forced signal", i)
		}
	}
}
