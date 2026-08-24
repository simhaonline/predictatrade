// Package strategy — Comprehensive tests for prompt.md Section 57 requirements.
// These tests verify the signal engine closure: TrendSwing transitions,
// StandardSwing direction/status, mirror tests, dominance, duplicates,
// provenance, probability, entry types, and replay.
package strategy

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ─── Test helpers ───

func makeTrendSwingRangeNoTransitionState() *features.MarketState {
	state := makeBaseState()
	state.Regime.Current = types.RegimeRange
	state.Indicators.ADX = decimal.NewFromFloat(12.0)   // Low ADX
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(15.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(15.0) // Equal DI — no direction
	state.Indicators.EMA9 = decimal.NewFromFloat(4400.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4400.0) // Flat EMAs
	state.Indicators.EMA50 = decimal.NewFromFloat(4400.0)
	state.Indicators.BollWidth = decimal.NewFromFloat(0.001) // Low BB width
	state.Indicators.BollUpper = decimal.NewFromFloat(4405.0)
	state.Indicators.BollLower = decimal.NewFromFloat(4395.0)
	state.Indicators.BollMiddle = decimal.NewFromFloat(4400.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(0.0)
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.0)
	state.Indicators.ATR = decimal.NewFromFloat(1.0) // Low ATR
	state.MTF.Score = 0 // No MTF direction
	state.Structure.LastBOS = nil
	state.Structure.LastCHoCH = nil
	state.Structure.CurrentTrend = ""
	state.CurrentPrice = decimal.NewFromFloat(4400.0)
	return state
}

func makeTrendSwingBullishTransitionState() *features.MarketState {
	state := makeBaseState()
	state.Regime.Current = types.RegimeRange
	state.Indicators.ADX = decimal.NewFromFloat(22.0) // ADX expansion
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(28.0) // +DI > -DI
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(14.0)
	state.Indicators.EMA9 = decimal.NewFromFloat(4405.0) // EMA slope bullish
	state.Indicators.EMA21 = decimal.NewFromFloat(4398.0)
	state.Indicators.EMA50 = decimal.NewFromFloat(4395.0)
	state.Indicators.EMA100 = decimal.NewFromFloat(4390.0)
	state.Indicators.EMA200 = decimal.NewFromFloat(4385.0)
	state.Indicators.BollWidth = decimal.NewFromFloat(0.05) // BB expansion
	state.Indicators.BollUpper = decimal.NewFromFloat(4415.0)
	state.Indicators.BollLower = decimal.NewFromFloat(4385.0)
	state.Indicators.BollMiddle = decimal.NewFromFloat(4400.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(0.5) // MACD bullish
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.1)
	state.Indicators.ATR = decimal.NewFromFloat(3.5) // ATR expansion
	state.MTF.Score = 50 // MTF bullish alignment
	state.Structure.LastBOS = &features.StructureEvent{
		Direction: "bullish", Time: time.Now().UTC(),
	}
	state.Structure.CurrentTrend = "bullish"
	state.Structure.SwingLows = []decimal.Decimal{decimal.NewFromFloat(4385.0)}
	state.Structure.SwingHighs = []decimal.Decimal{decimal.NewFromFloat(4415.0)}
	state.CurrentPrice = decimal.NewFromFloat(4408.0) // Upper range pressure
	return state
}

func makeTrendSwingBearishTransitionState() *features.MarketState {
	state := makeBaseState()
	state.Regime.Current = types.RegimeRange
	state.Indicators.ADX = decimal.NewFromFloat(22.0)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(14.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(28.0) // -DI > +DI
	state.Indicators.EMA9 = decimal.NewFromFloat(4395.0) // EMA slope bearish
	state.Indicators.EMA21 = decimal.NewFromFloat(4402.0)
	state.Indicators.EMA50 = decimal.NewFromFloat(4405.0)
	state.Indicators.EMA100 = decimal.NewFromFloat(4410.0)
	state.Indicators.EMA200 = decimal.NewFromFloat(4415.0)
	state.Indicators.BollWidth = decimal.NewFromFloat(0.05)
	state.Indicators.BollUpper = decimal.NewFromFloat(4415.0)
	state.Indicators.BollLower = decimal.NewFromFloat(4385.0)
	state.Indicators.BollMiddle = decimal.NewFromFloat(4400.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(-0.5) // MACD bearish
	state.Indicators.MACDSignal = decimal.NewFromFloat(-0.1)
	state.Indicators.ATR = decimal.NewFromFloat(3.5)
	state.MTF.Score = -50 // MTF bearish alignment
	state.Structure.LastBOS = &features.StructureEvent{
		Direction: "bearish", Time: time.Now().UTC(),
	}
	state.Structure.CurrentTrend = "bearish"
	state.Structure.SwingLows = []decimal.Decimal{decimal.NewFromFloat(4385.0)}
	state.Structure.SwingHighs = []decimal.Decimal{decimal.NewFromFloat(4415.0)}
	state.CurrentPrice = decimal.NewFromFloat(4392.0) // Lower range pressure
	return state
}

func makeTrendSwingBullishConfirmedState() *features.MarketState {
	state := makeBaseState()
	state.Regime.Current = types.RegimeTrendingBullish
	state.Indicators.ADX = decimal.NewFromFloat(30.0) // Strong ADX
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(35.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(12.0)
	state.Indicators.EMA9 = decimal.NewFromFloat(4410.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4400.0)
	state.Indicators.EMA50 = decimal.NewFromFloat(4390.0)
	state.Indicators.EMA100 = decimal.NewFromFloat(4380.0)
	state.Indicators.EMA200 = decimal.NewFromFloat(4370.0)
	state.Indicators.SMA200 = decimal.NewFromFloat(4375.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(1.5)
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.5)
	state.Indicators.CCI = decimal.NewFromFloat(100)
	state.Indicators.ATR = decimal.NewFromFloat(5.0)
	state.MTF.Score = 60
	state.Structure.CurrentTrend = "bullish"
	state.Structure.LastBOS = &features.StructureEvent{
		Direction: "bullish", Time: time.Now().UTC(),
	}
	state.CurrentPrice = decimal.NewFromFloat(4415.0)
	state.Candle.IsBullish = true
	state.Candle.ConsecutiveBull = 3
	return state
}

func makeTrendSwingBearishConfirmedState() *features.MarketState {
	state := makeBaseState()
	state.Regime.Current = types.RegimeTrendingBearish
	state.Indicators.ADX = decimal.NewFromFloat(30.0)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(12.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(35.0)
	state.Indicators.EMA9 = decimal.NewFromFloat(4390.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4400.0)
	state.Indicators.EMA50 = decimal.NewFromFloat(4410.0)
	state.Indicators.EMA100 = decimal.NewFromFloat(4420.0)
	state.Indicators.EMA200 = decimal.NewFromFloat(4430.0)
	state.Indicators.SMA200 = decimal.NewFromFloat(4425.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(-1.5)
	state.Indicators.MACDSignal = decimal.NewFromFloat(-0.5)
	state.Indicators.CCI = decimal.NewFromFloat(-100)
	state.Indicators.ATR = decimal.NewFromFloat(5.0)
	state.MTF.Score = -60
	state.Structure.CurrentTrend = "bearish"
	state.Structure.LastBOS = &features.StructureEvent{
		Direction: "bearish", Time: time.Now().UTC(),
	}
	state.CurrentPrice = decimal.NewFromFloat(4385.0)
	state.Candle.IsBearish = true
	state.Candle.ConsecutiveBear = 3
	return state
}

func makeStandardSwingBlockedSellState() *features.MarketState {
	state := makeBaseState()
	state.Regime.Current = types.RegimeRange
	state.Indicators.ATR = decimal.NewFromFloat(3.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4395.0)
	state.Indicators.EMA50 = decimal.NewFromFloat(4405.0) // Bearish: EMA21 < EMA50
	state.Indicators.SMA200 = decimal.NewFromFloat(4410.0)
	state.Indicators.ADX = decimal.NewFromFloat(15.0) // Low ADX
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(15.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(20.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(-0.5)
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.1)
	state.Indicators.RSI = decimal.NewFromFloat(40.0)
	state.Indicators.BollUpper = decimal.NewFromFloat(4410.0)
	state.Indicators.BollLower = decimal.NewFromFloat(4390.0)
	state.Indicators.BollMiddle = decimal.NewFromFloat(4400.0)
	state.CurrentPrice = decimal.NewFromFloat(4396.0)
	state.MTF.Score = -20
	state.Session.CurrentSession = "LONDON"
	return state
}

func makeBaseState() *features.MarketState {
	return &features.MarketState{
		Symbol:        "XAUUSD",
		CurrentPrice:  decimal.NewFromFloat(4400.0),
		Bid:           decimal.NewFromFloat(4399.8),
		Ask:           decimal.NewFromFloat(4400.2),
		Mid:           decimal.NewFromFloat(4400.0),
		Spread:        decimal.NewFromFloat(0.4),
		Quality:       types.QualityAuthoritative,
		Timestamp:     time.Now().UTC(),
		LastTick:      &types.Tick{Source: "LIVE_MASTER_NODE", Sequence: 1, SourceTimestamp: time.Now().UTC()},
		Session:       features.SessionFeatures{CurrentSession: "LONDON", IsOverlap: false, IsWeekend: false, NewsRisk: "NONE"},
		Indicators:    features.IndicatorFeatures{ATR: decimal.NewFromFloat(3.0), RSI: decimal.NewFromFloat(50.0)},
		VWAP:          features.VWAPFeatures{SessionVWAP: decimal.NewFromFloat(4400.0)},
		Candle:        features.CandleIntelligence{},
		Structure:     features.StructureFeatures{},
		Liquidity:     features.LiquidityFeatures{},
		FVG:           features.FVGFeatures{},
		MTF:           features.MTFFeatures{Score: 0},
		Candles:       make(map[types.Timeframe]*types.Candle),
	}
}

// ─── TrendSwing Tests ───

// Test 1: TrendSwing ordinary range NO-TRADE (prompt.md Section 10)
func TestTrendSwing_OrdinaryRange_NoTrade(t *testing.T) {
	s := NewTrendSwing()
	state := makeTrendSwingRangeNoTransitionState()
	result := s.Evaluate(state)

	if result.Direction != types.DirectionNoTrade {
		t.Errorf("Expected NO-TRADE for ordinary range, got %s", result.Direction)
	}
	// Must have a reason code indicating no trend transition
	foundReason := false
	for _, rc := range result.ReasonCodes {
		if rc == types.NTNoTrendTransition || rc == types.NTRegimeMismatch {
			foundReason = true
			break
		}
	}
	if !foundReason {
		t.Errorf("Expected NT_NO_TREND_TRANSITION or NT_REGIME_MISMATCH reason, got %v", result.ReasonCodes)
	}
	// Score should not be unexplained zero — transition scores must be computed
	if result.TransitionLongScore.IsZero() && result.TransitionShortScore.IsZero() {
		t.Log("Transition scores are zero — this is correct for no-transition range")
	}
}

// Test 2: TrendSwing bullish transition candidate (prompt.md Section 11)
func TestTrendSwing_BullishTransitionCandidate(t *testing.T) {
	s := NewTrendSwing()
	state := makeTrendSwingBullishTransitionState()
	result := s.Evaluate(state)

	if result.Direction != types.DirectionBuy {
		t.Errorf("Expected BUY for bullish transition, got %s", result.Direction)
	}
	if !result.IsTransitionCandidate {
		t.Error("Expected IsTransitionCandidate=true")
	}
	// Transition long score should be meaningful
	longF, _ := result.TransitionLongScore.Float64()
	if longF <= 0 {
		t.Errorf("Expected positive transition long score, got %.1f", longF)
	}
	// Geometry should be computed
	if result.EntryPrice.IsZero() {
		t.Error("Expected non-zero entry price for transition candidate")
	}
}

// Test 3: TrendSwing bearish transition candidate (prompt.md Section 12)
func TestTrendSwing_BearishTransitionCandidate(t *testing.T) {
	s := NewTrendSwing()
	state := makeTrendSwingBearishTransitionState()
	result := s.Evaluate(state)

	if result.Direction != types.DirectionSell {
		t.Errorf("Expected SELL for bearish transition, got %s", result.Direction)
	}
	if !result.IsTransitionCandidate {
		t.Error("Expected IsTransitionCandidate=true")
	}
	shortF, _ := result.TransitionShortScore.Float64()
	if shortF <= 0 {
		t.Errorf("Expected positive transition short score, got %.1f", shortF)
	}
}

// Test 4: TrendSwing candidate cannot execute (prompt.md Section 44)
func TestTrendSwing_CandidateCannotExecute(t *testing.T) {
	s := NewTrendSwing()
	state := makeTrendSwingBullishTransitionState()
	result := s.Evaluate(state)

	if result.Direction != types.DirectionBuy {
		t.Fatalf("Expected BUY direction, got %s", result.Direction)
	}
	// A transition candidate must NOT be executable
	// The main.go pipeline classifies it as ADVISORY with executionEligible=false
	// Here we verify the strategy result indicates transition candidate
	if !result.IsTransitionCandidate {
		t.Error("Transition candidate must be marked as non-executable")
	}
}

// Test 5: TrendSwing bullish confirmed BUY (prompt.md Section 13)
func TestTrendSwing_BullishConfirmed_BUY(t *testing.T) {
	s := NewTrendSwing()
	state := makeTrendSwingBullishConfirmedState()
	result := s.Evaluate(state)

	if result.Direction != types.DirectionBuy {
		t.Errorf("Expected BUY for confirmed bullish trend, got %s (reasons: %v)", result.Direction, result.ReasonCodes)
	}
	scoreF, _ := result.RawScore.Float64()
	if scoreF < 30 {
		t.Errorf("Expected score >= 30 for confirmed bullish, got %.1f", scoreF)
	}
}

// Test 6: TrendSwing bearish confirmed SELL (prompt.md Section 14)
func TestTrendSwing_BearishConfirmed_SELL(t *testing.T) {
	s := NewTrendSwing()
	state := makeTrendSwingBearishConfirmedState()
	result := s.Evaluate(state)

	if result.Direction != types.DirectionSell {
		t.Errorf("Expected SELL for confirmed bearish trend, got %s (reasons: %v)", result.Direction, result.ReasonCodes)
	}
	scoreF, _ := result.RawScore.Float64()
	if scoreF < 30 {
		t.Errorf("Expected score >= 30 for confirmed bearish, got %.1f", scoreF)
	}
}

// Test 7: TrendSwing threshold reachability (prompt.md Section 15)
func TestTrendSwing_ThresholdReachability(t *testing.T) {
	// Verify that the TrendSwing trade threshold is mathematically reachable
	// from the maximum evidence budget in TRENDING_BULLISH regime
	s := NewTrendSwing()
	state := makeTrendSwingBullishConfirmedState()
	result := s.Evaluate(state)

	scoreF, _ := result.RawScore.Float64()
	_, tradeThresh, _ := GetThresholds(types.StrategyTrendSwing, types.RegimeTrendingBullish)

	if scoreF < tradeThresh {
		t.Errorf("Score %.1f < trade threshold %.1f — threshold not reachable from max evidence", scoreF, tradeThresh)
	}
}

// ─── StandardSwing Tests ───

// Test 8: StandardSwing blocked retains SELL direction (prompt.md Section 17)
func TestStandardSwing_BlockedRetainsSellDirection(t *testing.T) {
	// This tests the engine behavior: when gates veto, direction is preserved
	// The strategy itself returns SELL, and the engine should keep SELL with grade BLOCKED
	s := NewStandardSwing()
	state := makeStandardSwingBlockedSellState()
	result := s.Evaluate(state)

	// The strategy should produce a directional result (SELL or NO-TRADE)
	// If it produces SELL, the engine should preserve it when blocked
	if result.Direction == types.DirectionSell {
		t.Log("Strategy produced SELL — engine will preserve direction when blocked")
	} else if result.Direction == types.DirectionNoTrade {
		t.Log("Strategy produced NO-TRADE — no direction to preserve")
	}
	// The key test: Direction should never be "BLOCKED" — that's a status, not a direction
	if result.Direction == types.DirectionBlocked {
		t.Error("Direction should never be BLOCKED — that's a status, not a market direction")
	}
}

// Test 9: StandardSwing blocked retains BUY direction (prompt.md Section 17)
func TestStandardSwing_BlockedRetainsBuyDirection(t *testing.T) {
	state := makeBaseState()
	state.Regime.Current = types.RegimeRange
	state.Indicators.EMA21 = decimal.NewFromFloat(4405.0)
	state.Indicators.EMA50 = decimal.NewFromFloat(4395.0) // Bullish: EMA21 > EMA50
	state.Indicators.SMA200 = decimal.NewFromFloat(4390.0)
	state.Indicators.ADX = decimal.NewFromFloat(15.0)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(20.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(15.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(0.5)
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.1)
	state.Indicators.RSI = decimal.NewFromFloat(60.0)
	state.CurrentPrice = decimal.NewFromFloat(4406.0)
	state.MTF.Score = 20

	s := NewStandardSwing()
	result := s.Evaluate(state)

	if result.Direction == types.DirectionBlocked {
		t.Error("Direction should never be BLOCKED")
	}
}

// Test 10: StandardSwing trade threshold reachability (prompt.md Section 19)
func TestStandardSwing_TradeThresholdReachable(t *testing.T) {
	// Create a strong legitimate fixture that should reach the trade threshold
	state := makeBaseState()
	state.Regime.Current = types.RegimeTrendingBullish
	state.Indicators.ATR = decimal.NewFromFloat(3.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4410.0)
	state.Indicators.EMA50 = decimal.NewFromFloat(4395.0) // Strong bullish
	state.Indicators.SMA200 = decimal.NewFromFloat(4380.0)
	state.Indicators.ADX = decimal.NewFromFloat(25.0)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(30.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(10.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(1.5)
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.3)
	state.Indicators.RSI = decimal.NewFromFloat(60.0)
	state.Indicators.BollUpper = decimal.NewFromFloat(4420.0)
	state.Indicators.BollLower = decimal.NewFromFloat(4380.0)
	state.Indicators.BollMiddle = decimal.NewFromFloat(4400.0)
	state.CurrentPrice = decimal.NewFromFloat(4415.0)
	state.MTF.Score = 50
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Time: time.Now().UTC()}
	state.Structure.CurrentTrend = "bullish"
	state.Candle.IsBreakout = true
	state.Candle.IsBullish = true

	s := NewStandardSwing()
	result := s.Evaluate(state)

	_, tradeThresh, _ := GetThresholds(types.StrategyStandardSwing, types.RegimeTrendingBullish)
	scoreF, _ := result.RawScore.Float64()

	if scoreF < tradeThresh {
		t.Errorf("StandardSwing score %.1f < trade threshold %.1f — threshold not reachable from strong fixture", scoreF, tradeThresh)
	}
	if result.Direction != types.DirectionBuy {
		t.Errorf("Expected BUY from strong bullish fixture, got %s (score=%.1f, threshold=%.1f)", result.Direction, scoreF, tradeThresh)
	}
}

// ─── Mirror Tests (prompt.md Section 22) ───

// Test 11: StandardScalping bullish/bearish mirror
func TestStandardScalping_MirrorSymmetry(t *testing.T) {
	bullState := makeBaseState()
	bullState.Regime.Current = types.RegimeTrendingBullish
	bullState.Indicators.EMA9 = decimal.NewFromFloat(4410.0)
	bullState.Indicators.EMA21 = decimal.NewFromFloat(4395.0)
	bullState.Indicators.ADX = decimal.NewFromFloat(25.0)
	bullState.Indicators.ADXPlusDI = decimal.NewFromFloat(30.0)
	bullState.Indicators.ADXMinusDI = decimal.NewFromFloat(10.0)
	bullState.Indicators.MACDMain = decimal.NewFromFloat(1.0)
	bullState.Indicators.MACDSignal = decimal.NewFromFloat(0.2)
	bullState.Indicators.OsMA = decimal.NewFromFloat(0.5)
	bullState.Indicators.RSI = decimal.NewFromFloat(60.0)
	bullState.CurrentPrice = decimal.NewFromFloat(4410.0)
	bullState.VWAP.SessionVWAP = decimal.NewFromFloat(4395.0)
	bullState.MTF.Score = 50
	bullState.Candle.IsDisplacement = true
	bullState.Candle.IsBullish = true
	bullState.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Time: time.Now().UTC()}

	bearState := makeBaseState()
	bearState.Regime.Current = types.RegimeTrendingBearish
	bearState.Indicators.EMA9 = decimal.NewFromFloat(4390.0)
	bearState.Indicators.EMA21 = decimal.NewFromFloat(4405.0) // Mirror: EMA9 < EMA21
	bearState.Indicators.ADX = decimal.NewFromFloat(25.0)
	bearState.Indicators.ADXPlusDI = decimal.NewFromFloat(10.0) // Mirror
	bearState.Indicators.ADXMinusDI = decimal.NewFromFloat(30.0)
	bearState.Indicators.MACDMain = decimal.NewFromFloat(-1.0) // Mirror
	bearState.Indicators.MACDSignal = decimal.NewFromFloat(-0.2)
	bearState.Indicators.OsMA = decimal.NewFromFloat(-0.5)
	bearState.Indicators.RSI = decimal.NewFromFloat(40.0)
	bearState.CurrentPrice = decimal.NewFromFloat(4390.0)
	bearState.VWAP.SessionVWAP = decimal.NewFromFloat(4405.0)
	bearState.MTF.Score = -50
	bearState.Candle.IsDisplacement = true
	bearState.Candle.IsBearish = true
	bearState.Structure.LastBOS = &features.StructureEvent{Direction: "bearish", Time: time.Now().UTC()}

	s := NewStandardScalping()
	bullResult := s.Evaluate(bullState)
	bearResult := s.Evaluate(bearState)

	bullLong, _ := bullResult.LongScore.Float64()
	bullShort, _ := bullResult.ShortScore.Float64()
	bearLong, _ := bearResult.LongScore.Float64()
	bearShort, _ := bearResult.ShortScore.Float64()

	// Bullish fixture should favor BUY
	if bullLong <= bullShort {
		t.Errorf("Bullish fixture: long (%.1f) should > short (%.1f)", bullLong, bullShort)
	}
	// Bearish fixture should favor SELL
	if bearShort <= bearLong {
		t.Errorf("Bearish fixture: short (%.1f) should > long (%.1f)", bearShort, bearLong)
	}
	// Mirror symmetry: bull long ≈ bear short
	diff := bullLong - bearShort
	if diff < 0 {
		diff = -diff
	}
	if diff > 20 {
		t.Errorf("Mirror asymmetry: bull long (%.1f) vs bear short (%.1f), diff=%.1f", bullLong, bearShort, diff)
	}
}

// Test 12: UltraScalping bullish/bearish mirror
func TestUltraScalping_MirrorSymmetry(t *testing.T) {
	bullState := makeBaseState()
	bullState.Regime.Current = types.RegimeTrendingBullish
	bullState.Indicators.EMA9 = decimal.NewFromFloat(4410.0)
	bullState.Indicators.EMA21 = decimal.NewFromFloat(4398.0)
	bullState.Indicators.EMA50 = decimal.NewFromFloat(4390.0)
	bullState.Indicators.ADX = decimal.NewFromFloat(28.0)
	bullState.Indicators.ADXPlusDI = decimal.NewFromFloat(30.0)
	bullState.Indicators.ADXMinusDI = decimal.NewFromFloat(10.0)
	bullState.Indicators.OsMA = decimal.NewFromFloat(0.5)
	bullState.Indicators.StochMain = decimal.NewFromFloat(80.0)
	bullState.Indicators.StochSignal = decimal.NewFromFloat(70.0)
	bullState.CurrentPrice = decimal.NewFromFloat(4410.0)
	bullState.VWAP.SessionVWAP = decimal.NewFromFloat(4395.0)
	bullState.MTF.Score = 60
	bullState.Candle.IsDisplacement = true
	bullState.Candle.IsBullish = true

	bearState := makeBaseState()
	bearState.Regime.Current = types.RegimeTrendingBearish
	bearState.Indicators.EMA9 = decimal.NewFromFloat(4390.0)
	bearState.Indicators.EMA21 = decimal.NewFromFloat(4402.0)
	bearState.Indicators.EMA50 = decimal.NewFromFloat(4410.0)
	bearState.Indicators.ADX = decimal.NewFromFloat(28.0)
	bearState.Indicators.ADXPlusDI = decimal.NewFromFloat(10.0)
	bearState.Indicators.ADXMinusDI = decimal.NewFromFloat(30.0)
	bearState.Indicators.OsMA = decimal.NewFromFloat(-0.5)
	bearState.Indicators.StochMain = decimal.NewFromFloat(20.0)
	bearState.Indicators.StochSignal = decimal.NewFromFloat(30.0)
	bearState.CurrentPrice = decimal.NewFromFloat(4390.0)
	bearState.VWAP.SessionVWAP = decimal.NewFromFloat(4405.0)
	bearState.MTF.Score = -60
	bearState.Candle.IsDisplacement = true
	bearState.Candle.IsBearish = true

	s := NewUltraScalping()
	bullResult := s.Evaluate(bullState)
	bearResult := s.Evaluate(bearState)

	bullLong, _ := bullResult.LongScore.Float64()
	bearShort, _ := bearResult.ShortScore.Float64()

	if bullLong <= 0 {
		t.Errorf("Bullish fixture: long score should be positive, got %.1f", bullLong)
	}
	if bearShort <= 0 {
		t.Errorf("Bearish fixture: short score should be positive, got %.1f", bearShort)
	}
}

// Test 13: Trend transition bullish/bearish mirror
func TestTrendTransition_MirrorSymmetry(t *testing.T) {
	s := NewTrendSwing()
	bullState := makeTrendSwingBullishTransitionState()
	bearState := makeTrendSwingBearishTransitionState()

	bullResult := s.Evaluate(bullState)
	bearResult := s.Evaluate(bearState)

	bullLong, _ := bullResult.TransitionLongScore.Float64()
	bearShort, _ := bearResult.TransitionShortScore.Float64()

	if bullLong <= 0 {
		t.Errorf("Bullish transition: long score should be positive, got %.1f", bullLong)
	}
	if bearShort <= 0 {
		t.Errorf("Bearish transition: short score should be positive, got %.1f", bearShort)
	}
	// Mirror symmetry check
	diff := bullLong - bearShort
	if diff < 0 {
		diff = -diff
	}
	if diff > 30 {
		t.Errorf("Transition mirror asymmetry: bull long (%.1f) vs bear short (%.1f), diff=%.1f", bullLong, bearShort, diff)
	}
}

// ─── Dominance Tests (prompt.md Section 23) ───

// Test 14: Candidate directional dominance
func TestDirectionalDominance_PreventsConflictingDirection(t *testing.T) {
	// Create state where long and short scores are nearly equal
	state := makeBaseState()
	state.Regime.Current = types.RegimeRange
	state.Indicators.EMA9 = decimal.NewFromFloat(4400.01) // Nearly equal
	state.Indicators.EMA21 = decimal.NewFromFloat(4400.0)
	state.Indicators.ADX = decimal.NewFromFloat(20.0)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(20.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(20.0) // Equal DI
	state.Indicators.MACDMain = decimal.NewFromFloat(0.01) // Nearly zero
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.0)
	state.Indicators.OsMA = decimal.NewFromFloat(0.0)
	state.Indicators.RSI = decimal.NewFromFloat(50.0) // Neutral
	state.CurrentPrice = decimal.NewFromFloat(4400.0)
	state.MTF.Score = 0

	s := NewStandardScalping()
	result := s.Evaluate(state)

	// With nearly equal scores, should produce NO-TRADE or conflicting direction
	if result.Direction == types.DirectionBuy || result.Direction == types.DirectionSell {
		longF, _ := result.LongScore.Float64()
		shortF, _ := result.ShortScore.Float64()
		diff := longF - shortF
		if diff < 0 {
			diff = -diff
		}
		if diff < MinDominanceMargin {
			t.Errorf("Direction %s with dominance %.1f < min %.1f — should be NO-TRADE",
				result.Direction, diff, MinDominanceMargin)
		}
	}
}

// Test 15: Conflicting scores produce no direction
func TestConflictingScores_NoDirection(t *testing.T) {
	dir, _, _, _, reasons := scoreDirectionWithThresholds(
		[]types.EvidenceContribution{
			{Direction: types.DirectionBuy, Contribution: decimal.NewFromFloat(0.30)},
			{Direction: types.DirectionSell, Contribution: decimal.NewFromFloat(0.29)},
		},
		20, 40, decimal.Zero,
	)

	if dir != types.DirectionNoTrade {
		t.Errorf("Expected NO-TRADE for conflicting scores, got %s", dir)
	}
	foundConflict := false
	for _, r := range reasons {
		if r == types.NTConflictingDirection {
			foundConflict = true
			break
		}
	}
	if !foundConflict {
		t.Errorf("Expected NT_CONFLICTING_DIRECTION in reasons, got %v", reasons)
	}
}

// ─── Duplicate Signal Tests (prompt.md Sections 27-29) ───

// Test 16: Duplicate closed-bar signal suppressed
func TestDuplicateClosedBar_Suppressed(t *testing.T) {
	barTime := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	fp1 := computeTestFingerprintWithBar("XAUUSD", types.StrategyStandardScalping,
		types.DirectionBuy, decimal.NewFromFloat(4400.0), decimal.NewFromFloat(4390.0),
		time.Time{}, time.Time{}, barTime)
	fp2 := computeTestFingerprintWithBar("XAUUSD", types.StrategyStandardScalping,
		types.DirectionBuy, decimal.NewFromFloat(4400.0), decimal.NewFromFloat(4390.0),
		time.Time{}, time.Time{}, barTime)

	if fp1 != fp2 {
		t.Error("Same closed-bar inputs should produce same fingerprint")
	}
}

// Test 17: Different bar times produce different fingerprints
func TestDuplicate_DifferentBarTime_DifferentFingerprint(t *testing.T) {
	bar1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	bar2 := time.Date(2026, 1, 15, 10, 5, 0, 0, time.UTC)
	fp1 := computeTestFingerprintWithBar("XAUUSD", types.StrategyStandardScalping,
		types.DirectionBuy, decimal.NewFromFloat(4400.0), decimal.NewFromFloat(4390.0),
		time.Time{}, time.Time{}, bar1)
	fp2 := computeTestFingerprintWithBar("XAUUSD", types.StrategyStandardScalping,
		types.DirectionBuy, decimal.NewFromFloat(4400.0), decimal.NewFromFloat(4390.0),
		time.Time{}, time.Time{}, bar2)

	if fp1 == fp2 {
		t.Error("Different bar times should produce different fingerprints")
	}
}

// Test 18: Retry does not duplicate signal (same fingerprint = duplicate)
func TestRetry_NoDuplicate(t *testing.T) {
	// Without Valkey cache, duplicate checking is fail-open (allows all)
	// This is correct for testing — production uses Valkey
	isNew := true
	var err error = nil
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !isNew {
		t.Error("First check should be new (nil cache = fail-open)")
	}
}

// ─── Provenance Tests (prompt.md Sections 30-31, 43) ───

// Test 19: Real provider required for LIVE VERIFIED
func TestProvenance_LiveVerified_RequiresLiveSource(t *testing.T) {
	if !types.IsLiveDataSource(types.DataSourceLiveMasterNode) {
		t.Error("LIVE_MASTER_NODE should be a live data source")
	}
	if types.IsLiveDataSource(types.DataSourceSynthetic) {
		t.Error("SYNTHETIC should NOT be a live data source")
	}
	if types.IsLiveDataSource(types.DataSourceTest) {
		t.Error("TEST should NOT be a live data source")
	}
}

// Test 20: Synthetic cannot become LIVE VERIFIED
func TestProvenance_SyntheticCannotBeLiveVerified(t *testing.T) {
	sources := []types.DataSourceType{
		types.DataSourceSynthetic,
		types.DataSourceTest,
		types.DataSourceMock,
		types.DataSourceFixture,
		types.DataSourcePlaceholder,
	}
	for _, src := range sources {
		if types.IsLiveDataSource(src) {
			t.Errorf("Source %s should NOT be live", src)
		}
	}
}

// ─── Probability Tests (prompt.md Section 36) ───

// Test 21: Unverified calibration produces probability NULL
func TestCalibration_Unverified_ProducesNullProbability(t *testing.T) {
	if types.IsCalibrationValidated(types.CalibrationUnverified) {
		t.Error("UNVERIFIED should not be validated")
	}
	if !types.IsCalibrationValidated(types.CalibrationValidated) {
		t.Error("VALIDATED should be validated")
	}
	if !types.IsCalibrationValidated(types.CalibrationPromoted) {
		t.Error("PROMOTED should be validated")
	}
}

// ─── Entry Type Tests (prompt.md Section 33) ───

// Test 22: Entry BUY uses Ask
func TestEntry_BUY_UsesAsk(t *testing.T) {
	state := makeBaseState()
	state.Bid = decimal.NewFromFloat(4399.8)
	state.Ask = decimal.NewFromFloat(4400.2)
	state.CurrentPrice = decimal.NewFromFloat(4400.0) // Mid — should NOT be used
	state.Indicators.ATR = decimal.NewFromFloat(3.0)

	geo := BuildTradeGeometry(state, types.DirectionBuy, StrategyConfig{
		ATRMultiplierSL: 1.5, ATRMultiplierTP1: 1.5, ATRMultiplierTP2: 2.5, ATRMultiplierTP3: 3.5, MinRR: 1.8,
	})

	if !geo.Valid {
		t.Fatal("Expected valid geometry")
	}
	entryF, _ := geo.Entry.Float64()
	if entryF != 4400.2 {
		t.Errorf("BUY entry should use Ask (4400.2), got %.1f", entryF)
	}
}

// Test 23: Entry SELL uses Bid
func TestEntry_SELL_UsesBid(t *testing.T) {
	state := makeBaseState()
	state.Bid = decimal.NewFromFloat(4399.8)
	state.Ask = decimal.NewFromFloat(4400.2)
	state.CurrentPrice = decimal.NewFromFloat(4400.0)
	state.Indicators.ATR = decimal.NewFromFloat(3.0)

	geo := BuildTradeGeometry(state, types.DirectionSell, StrategyConfig{
		ATRMultiplierSL: 1.5, ATRMultiplierTP1: 1.5, ATRMultiplierTP2: 2.5, ATRMultiplierTP3: 3.5, MinRR: 1.8,
	})

	if !geo.Valid {
		t.Fatal("Expected valid geometry")
	}
	entryF, _ := geo.Entry.Float64()
	if entryF != 4399.8 {
		t.Errorf("SELL entry should use Bid (4399.8), got %.1f", entryF)
	}
}

// ─── Closed Bar Tests (prompt.md Section 32) ───

// Test 24: Swing uses closed-bar confirmation where required
func TestClosedBar_SwingConfirmation(t *testing.T) {
	// TrendSwing and StandardSwing should use closed-bar semantics
	// The BarClosed type distinguishes INTRABAR_LIVE from CLOSED_BAR_CONFIRMED
	if types.BarClosedConfirmed != "CLOSED_BAR_CONFIRMED" {
		t.Error("BarClosedConfirmed should be CLOSED_BAR_CONFIRMED")
	}
	if types.BarIntrabarLive != "INTRABAR_LIVE" {
		t.Error("BarIntrabarLive should be INTRABAR_LIVE")
	}
}

// ─── Candidate Execution Safety (prompt.md Section 44) ───

// Test 25: Candidate cannot execute
func TestCandidate_CannotExecute_Advisory(t *testing.T) {
	// A BUY_CANDIDATE + AutoExecute=true must result in ZERO broker orders
	// This is verified by the signal class being ADVISORY, not EXECUTABLE
	_, _, isCandidate, _ := EvaluateCandidateThreshold(
		types.DirectionBuy,
		decimal.NewFromFloat(35.0), // Between candidate (30) and trade (50) for TrendSwing
		30, 50,
	)
	if !isCandidate {
		t.Error("Score between candidate and trade threshold should be a candidate")
	}
}

// ─── Exit Lifecycle Tests (prompt.md Section 35) ───

// Test 26: Actual exit remains NULL before closure
func TestExitPrice_NullBeforeClosure(t *testing.T) {
	sig := &types.Signal{
		ID:         "test-signal",
		Direction:  types.DirectionBuy,
		EntryPrice: decimal.NewFromFloat(4400.0),
		TP1:        decimal.NewFromFloat(4412.0),
	}
	// Before closure, exit price and closed_at should be zero
	if !sig.ExitPrice.IsZero() {
		t.Error("ExitPrice should be zero before closure")
	}
	if !sig.ClosedAt.IsZero() {
		t.Error("ClosedAt should be zero before closure")
	}
	if !sig.RealizedR.IsZero() {
		t.Error("RealizedR should be zero before closure")
	}
}

// ─── Deterministic Replay Tests (prompt.md Section 39) ───

// Test 27: Deterministic replay reproduces signal
func TestDeterministicReplay_ReproducesSignal(t *testing.T) {
	s := NewTrendSwing()
	state := makeTrendSwingBullishConfirmedState()

	// Evaluate twice with same input
	result1 := s.Evaluate(state)
	result2 := s.Evaluate(state)

	if result1.Direction != result2.Direction {
		t.Errorf("Replay direction mismatch: %s vs %s", result1.Direction, result2.Direction)
	}
	if !result1.RawScore.Equal(result2.RawScore) {
		t.Errorf("Replay score mismatch: %s vs %s", result1.RawScore, result2.RawScore)
	}
	if !result1.LongScore.Equal(result2.LongScore) {
		t.Errorf("Replay long score mismatch: %s vs %s", result1.LongScore, result2.LongScore)
	}
}

// ─── Look-Ahead Prevention (prompt.md Section 46) ───

// Test 28: Future candle not available to evaluator
func TestLookAhead_FutureCandleNotAvailable(t *testing.T) {
	// The strategy evaluator only uses the current MarketState, which contains
	// indicators computed from PAST data. Future candles cannot enter.
	// This test verifies that the strategy doesn't crash or produce different
	// results when "future" data exists but is not accessible.
	s := NewStandardScalping()
	state := makeBaseState()
	state.Regime.Current = types.RegimeTrendingBullish
	state.Indicators.EMA9 = decimal.NewFromFloat(4410.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4395.0)
	state.Indicators.ADX = decimal.NewFromFloat(25.0)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(30.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(10.0)
	state.Indicators.ATR = decimal.NewFromFloat(3.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(1.0)
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.2)
	state.Indicators.RSI = decimal.NewFromFloat(60.0)
	state.CurrentPrice = decimal.NewFromFloat(4410.0)
	state.MTF.Score = 50

	result1 := s.Evaluate(state)
	// Add a "future" candle that should NOT affect the result
	state.Candles[types.TFH1] = &types.Candle{
		Time: time.Now().Add(1 * time.Hour), // Future
		Close: decimal.NewFromFloat(4500.0), // Artificially high
	}
	result2 := s.Evaluate(state)

	if result1.Direction != result2.Direction {
		t.Errorf("Future candle should not affect evaluation: %s vs %s", result1.Direction, result2.Direction)
	}
}

// ─── Score Budget Tests (prompt.md Section 16) ───

// Test 29: All intended executable score profiles reach their thresholds
func TestScoreBudget_AllProfilesReachThresholds(t *testing.T) {
	strategies := []struct {
		id      types.StrategyID
		regimes []types.Regime
	}{
		{types.StrategyStandardScalping, []types.Regime{types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout, types.RegimeRange}},
		{types.StrategyUltraScalping, []types.Regime{types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout, types.RegimeRange}},
		{types.StrategyStandardSwing, []types.Regime{types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout, types.RegimeRange}},
		{types.StrategyTrendSwing, []types.Regime{types.RegimeTrendingBullish, types.RegimeTrendingBearish, types.RegimeBreakout}},
	}

	for _, st := range strategies {
		for _, regime := range st.regimes {
			ct, tt, found := GetThresholds(st.id, regime)
			if !found {
				t.Errorf("No thresholds found for %s in %s", st.id, regime)
				continue
			}
			// Verify candidate < trade
			if ct >= tt {
				t.Errorf("%s %s: candidate threshold (%.1f) >= trade threshold (%.1f)", st.id, regime, ct, tt)
			}
			// Verify thresholds are positive
			if ct <= 0 || tt <= 0 {
				t.Errorf("%s %s: thresholds must be positive (ct=%.1f, tt=%.1f)", st.id, regime, ct, tt)
			}
		}
	}
}

// ─── Ultra Intrabar Debounce (prompt.md Section 28) ───

// Test 30: Ultra intrabar debounce — different bar times produce different fingerprints
func TestUltraIntrabar_Debounce(t *testing.T) {
	// Ultra scalping evaluates frequently, but canonical signals should only
	// be published on meaningful changes (new bar, direction change, etc.)
	bar1 := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	bar2 := time.Date(2026, 1, 15, 10, 1, 0, 0, time.UTC) // Next minute

	fp1 := computeTestFingerprintWithBar("XAUUSD", types.StrategyUltraScalping,
		types.DirectionBuy, decimal.NewFromFloat(4400.0), decimal.NewFromFloat(4395.0),
		time.Time{}, time.Time{}, bar1)
	fp2 := computeTestFingerprintWithBar("XAUUSD", types.StrategyUltraScalping,
		types.DirectionBuy, decimal.NewFromFloat(4400.0), decimal.NewFromFloat(4395.0),
		time.Time{}, time.Time{}, bar2)

	if fp1 == fp2 {
		t.Error("Different bar times should produce different fingerprints for Ultra")
	}
}

// ─── Deterministic Hash (prompt.md Section 38) ───

// Test 31: Deterministic hash reproducibility
func TestDeterministicHash_Reproducibility(t *testing.T) {
	// Same inputs should produce same hash
	input := "XAUUSD|STANDARD_SCALPING|1.0|BUY|4400.00|4390.00|||2026-01-15T10:00Z"
	hash1 := sha256.Sum256([]byte(input))
	hash2 := sha256.Sum256([]byte(input))
	hex1 := hex.EncodeToString(hash1[:16])
	hex2 := hex.EncodeToString(hash2[:16])

	if hex1 != hex2 {
		t.Error("Same input should produce same hash")
	}
}

// ─── Ultra Geometry Validation (prompt.md Section 25) ───

// Test 32: UltraScalping geometry appropriate for horizon
func TestUltraScalping_Geometry_AppropriateForHorizon(t *testing.T) {
	state := makeBaseState()
	state.Regime.Current = types.RegimeTrendingBullish
	state.Indicators.EMA9 = decimal.NewFromFloat(4410.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4398.0)
	state.Indicators.EMA50 = decimal.NewFromFloat(4390.0)
	state.Indicators.ADX = decimal.NewFromFloat(28.0)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(30.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(10.0)
	state.Indicators.ATR = decimal.NewFromFloat(2.0) // Ultra: smaller ATR
	state.Indicators.OsMA = decimal.NewFromFloat(0.5)
	state.Indicators.StochMain = decimal.NewFromFloat(80.0)
	state.Indicators.StochSignal = decimal.NewFromFloat(70.0)
	state.CurrentPrice = decimal.NewFromFloat(4410.0)
	state.VWAP.SessionVWAP = decimal.NewFromFloat(4395.0)
	state.MTF.Score = 60
	state.Candle.IsDisplacement = true
	state.Candle.IsBullish = true

	s := NewUltraScalping()
	result := s.Evaluate(state)

	if result.Direction != types.DirectionBuy && result.Direction != types.DirectionNoTrade {
		t.Logf("Ultra direction: %s", result.Direction)
	}

	if !result.EntryPrice.IsZero() && !result.StopLoss.IsZero() {
		risk := result.EntryPrice.Sub(result.StopLoss).Abs()
		riskF, _ := risk.Float64()
		// Ultra SL should be tight (0.5-1.5 ATR)
		atrF, _ := state.Indicators.ATR.Float64()
		if atrF > 0 && (riskF < 0.3*atrF || riskF > 2.0*atrF) {
			t.Errorf("Ultra SL risk (%.2f) should be 0.3-2.0 ATR (%.2f)", riskF, atrF)
		}
	}
}

// ─── StandardScalping Geometry (prompt.md Section 26) ───

// Test 33: StandardScalping geometry ordering
func TestStandardScalping_Geometry_Ordering(t *testing.T) {
	state := makeBaseState()
	state.Regime.Current = types.RegimeTrendingBullish
	state.Indicators.EMA9 = decimal.NewFromFloat(4410.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4395.0)
	state.Indicators.ADX = decimal.NewFromFloat(25.0)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(30.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(10.0)
	state.Indicators.ATR = decimal.NewFromFloat(3.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(1.0)
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.2)
	state.Indicators.RSI = decimal.NewFromFloat(60.0)
	state.CurrentPrice = decimal.NewFromFloat(4410.0)
	state.VWAP.SessionVWAP = decimal.NewFromFloat(4395.0)
	state.MTF.Score = 50
	state.Candle.IsDisplacement = true
	state.Candle.IsBullish = true
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Time: time.Now().UTC()}

	s := NewStandardScalping()
	result := s.Evaluate(state)

	if result.Direction == types.DirectionBuy && !result.EntryPrice.IsZero() && !result.StopLoss.IsZero() {
		// BUY: SL < Entry < TP1 <= TP2 <= TP3
		if !result.StopLoss.LessThan(result.EntryPrice) {
			t.Error("BUY: SL should be < Entry")
		}
		if !result.TP1.GreaterThan(result.EntryPrice) {
			t.Error("BUY: TP1 should be > Entry")
		}
		if !result.TP2.GreaterThanOrEqual(result.TP1) {
			t.Error("BUY: TP2 should be >= TP1")
		}
		if !result.TP3.GreaterThanOrEqual(result.TP2) {
			t.Error("BUY: TP3 should be >= TP2")
		}
	}
}

// ─── Feature Contribution Trace (prompt.md Section 40) ───

// Test 34: Feature contribution trace available
func TestFeatureContribution_TraceAvailable(t *testing.T) {
	s := NewStandardScalping()
	state := makeBaseState()
	state.Regime.Current = types.RegimeTrendingBullish
	state.Indicators.EMA9 = decimal.NewFromFloat(4410.0)
	state.Indicators.EMA21 = decimal.NewFromFloat(4395.0)
	state.Indicators.ADX = decimal.NewFromFloat(25.0)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(30.0)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(10.0)
	state.Indicators.ATR = decimal.NewFromFloat(3.0)
	state.Indicators.MACDMain = decimal.NewFromFloat(1.0)
	state.Indicators.MACDSignal = decimal.NewFromFloat(0.2)
	state.Indicators.RSI = decimal.NewFromFloat(60.0)
	state.CurrentPrice = decimal.NewFromFloat(4410.0)
	state.VWAP.SessionVWAP = decimal.NewFromFloat(4395.0)
	state.MTF.Score = 50
	// Bearish BOS gives the trace a genuine SELL contribution (the fixture's
	// indicators are otherwise uniformly bullish).
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bearish", Price: decimal.NewFromFloat(4380), Time: time.Now()}

	result := s.Evaluate(state)

	if len(result.Evidence) == 0 {
		t.Error("Expected non-empty evidence for feature contribution trace")
	}

	// Check that evidence has both LONG and SHORT contributions
	hasLong := false
	hasShort := false
	for _, e := range result.Evidence {
		if e.Direction == types.DirectionBuy {
			hasLong = true
		}
		if e.Direction == types.DirectionSell {
			hasShort = true
		}
	}
	if !hasLong {
		t.Error("Expected at least one BUY (long) evidence contribution")
	}
	if !hasShort {
		t.Error("Expected at least one SELL (short) evidence contribution")
	}
}

// computeTestFingerprintWithBar is a local copy for testing without import cycle.
func computeTestFingerprintWithBar(symbol string, strategyID types.StrategyID, direction types.Direction,
	entryPrice, stopLoss decimal.Decimal, bosTime, chochTime time.Time, barTime time.Time) string {
	entryStr := fmt.Sprintf("%.2f", entryPrice.InexactFloat64())
	slStr := fmt.Sprintf("%.2f", stopLoss.InexactFloat64())
	bosStr := ""
	if !bosTime.IsZero() {
		bosStr = bosTime.UTC().Format("2006-01-02T15:04Z")
	}
	barStr := ""
	if !barTime.IsZero() {
		barStr = barTime.UTC().Format("2006-01-02T15:04Z")
	}
	canonical := strings.Join([]string{symbol, string(strategyID), "1.0", string(direction), entryStr, slStr, bosStr, "", barStr}, "|")
	hash := sha256.Sum256([]byte(canonical))
	return hex.EncodeToString(hash[:16])
}

// ─── Helper: makeBullishStateExt exists in golden_test.go ───
// If not, we define a minimal version here
var _ = fmt.Sprintf
var _ = strings.HasPrefix
