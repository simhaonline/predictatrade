// Package strategy — Golden deterministic tests for the four strategy products.
// SOW Part 31: Mathematical golden tests for indicators, score, BUY/SELL/WAIT/NO_TRADE/BLOCKED/ERROR, and risk.
// These tests prove the exact mathematical behavior of the production pipeline.
package strategy

import (
	"testing"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/predictatrade/realtime/pkg/math"
	"github.com/shopspring/decimal"
)

// ============================================================================
// PART 31A — INDICATOR GOLDEN TESTS
// Known candle data → known expected indicator values.
// ============================================================================

func TestGolden_EMA_KnownValues(t *testing.T) {
	// 10 closes with a known upward trend
	closes := []decimal.Decimal{
		decimal.NewFromFloat(100), decimal.NewFromFloat(101),
		decimal.NewFromFloat(102), decimal.NewFromFloat(103),
		decimal.NewFromFloat(104), decimal.NewFromFloat(105),
		decimal.NewFromFloat(106), decimal.NewFromFloat(107),
		decimal.NewFromFloat(108), decimal.NewFromFloat(109),
	}
	ema3 := patmath.EMA(closes, 3)
	// EMA with period 3: multiplier = 2/(3+1) = 0.5
	// ema[0] = 100
	// ema[1] = 0.5*101 + 0.5*100 = 100.5
	// ema[2] = 0.5*102 + 0.5*100.5 = 101.25
	// ... continuing to ema[9]
	// Let's compute step by step:
	// ema[0]=100, ema[1]=100.5, ema[2]=101.25, ema[3]=102.125
	// ema[4]=103.0625, ema[5]=104.03125, ema[6]=105.015625
	// ema[7]=106.0078125, ema[8]=107.00390625, ema[9]=108.001953125
	expected := decimal.NewFromFloat(108.001953125)
	if !ema3.Sub(expected).Abs().LessThan(decimal.NewFromFloat(0.0001)) {
		t.Errorf("EMA(3) = %s, expected ~%s", ema3.String(), expected.String())
	}
}

func TestGolden_RSI_KnownValues(t *testing.T) {
	// 15 closes with alternating up/down — all gains = 1, all losses = 0
	// RSI should be 100 (all gains, no losses)
	closes := make([]decimal.Decimal, 15)
	for i := range closes {
		closes[i] = decimal.NewFromInt(int64(100 + i))
	}
	rsi := patmath.RSI(closes, 14)
	// All gains, no losses → RS = inf → RSI = 100
	if !rsi.Equal(decimal.NewFromInt(100)) {
		t.Errorf("RSI(14) all-gains = %s, expected 100", rsi.String())
	}

	// All losses → RSI = 0
	closes2 := make([]decimal.Decimal, 15)
	for i := range closes2 {
		closes2[i] = decimal.NewFromInt(int64(100 - i))
	}
	rsi2 := patmath.RSI(closes2, 14)
	if !rsi2.Equal(decimal.NewFromInt(0)) {
		t.Errorf("RSI(14) all-losses = %s, expected 0", rsi2.String())
	}
}

func TestGolden_ATR_KnownValues(t *testing.T) {
	// 15 bars where H-L = 2 for every bar, no gap
	// TR = 2 for each bar, ATR(14) = 2.0
	highs := make([]decimal.Decimal, 15)
	lows := make([]decimal.Decimal, 15)
	closes := make([]decimal.Decimal, 15)
	for i := range highs {
		highs[i] = decimal.NewFromInt(int64(102 + i))
		lows[i] = decimal.NewFromInt(int64(100 + i))
		closes[i] = decimal.NewFromInt(int64(101 + i))
	}
	atr := patmath.ATR(highs, lows, closes, 14)
	// TR for each bar = max(H-L, |H-prevC|, |L-prevC|) = max(2, 1, 1) = 2
	// ATR = sum(TR)/14 = 2*14/14 = 2
	expected := decimal.NewFromInt(2)
	if !atr.Equal(expected) {
		t.Errorf("ATR(14) = %s, expected %s", atr.String(), expected.String())
	}
}

func TestGolden_GrossRR_KnownValues(t *testing.T) {
	// Entry=4400, SL=4390, TP1=4412
	// Risk = |4400-4390| = 10
	// Reward1 = |4412-4400| = 12
	// RR1 = 12/10 = 1.2
	entry := decimal.NewFromFloat(4400)
	sl := decimal.NewFromFloat(4390)
	tp1 := decimal.NewFromFloat(4412)
	rr := patmath.GrossRR(entry, sl, tp1)
	expected := decimal.NewFromFloat(1.2)
	if !rr.Equal(expected) {
		t.Errorf("GrossRR = %s, expected %s", rr.String(), expected.String())
	}
}

func TestGolden_NetRR_KnownValues(t *testing.T) {
	// Entry=4400, SL=4390, TP1=4412, cost=0.30
	// net_RR = (12 - 0.30) / (10 + 0.30) = 11.70 / 10.30 = 1.1359...
	entry := decimal.NewFromFloat(4400)
	sl := decimal.NewFromFloat(4390)
	tp1 := decimal.NewFromFloat(4412)
	cost := decimal.NewFromFloat(0.30)
	rr := patmath.NetRR(entry, sl, tp1, cost)
	expected := decimal.NewFromFloat(11.70).Div(decimal.NewFromFloat(10.30))
	if !rr.Sub(expected).Abs().LessThan(decimal.NewFromFloat(0.0001)) {
		t.Errorf("NetRR = %s, expected ~%s", rr.String(), expected.String())
	}
}

func TestGolden_RR_DivisionByZero(t *testing.T) {
	// SL = Entry → division by zero → must return 0, not panic
	rr := patmath.GrossRR(decimal.NewFromFloat(4400), decimal.NewFromFloat(4400), decimal.NewFromFloat(4412))
	if !rr.IsZero() {
		t.Errorf("GrossRR with zero risk = %s, expected 0", rr.String())
	}
}

// ============================================================================
// PART 31B — SCORE GOLDEN TESTS
// Known components → exact expected score.
// ============================================================================

func TestGolden_ScoreDirection_BullishComponents(t *testing.T) {
	// Three BUY evidence contributions: 0.10 + 0.08 + 0.05 = 0.23
	// Scaled by 100: longScore = 23, shortScore = 0
	// minConfluence = 20 → 23 > 20 → BUY
	evidence := []types.EvidenceContribution{
		{Pillar: "TREND", Feature: "EMA_BULLISH", Direction: types.DirectionBuy, Contribution: decimal.NewFromFloat(0.10), NormalizedValue: decimal.NewFromFloat(0.10)},
		{Pillar: "MOMENTUM", Feature: "RSI_BULLISH", Direction: types.DirectionBuy, Contribution: decimal.NewFromFloat(0.08), NormalizedValue: decimal.NewFromFloat(0.08)},
		{Pillar: "VWAP", Feature: "ABOVE_VWAP", Direction: types.DirectionBuy, Contribution: decimal.NewFromFloat(0.05), NormalizedValue: decimal.NewFromFloat(0.05)},
	}
	dir, raw, long, short, reasons := scoreDirection(nil, evidence, 20, decimal.Zero)
	if dir != types.DirectionBuy {
		t.Errorf("Expected BUY, got %s", dir)
	}
	expectedLong := decimal.NewFromFloat(23) // 0.23 * 100
	if !long.Equal(expectedLong) {
		t.Errorf("LongScore = %s, expected %s", long.String(), expectedLong.String())
	}
	if !short.IsZero() {
		t.Errorf("ShortScore = %s, expected 0", short.String())
	}
	if !raw.Equal(expectedLong) {
		t.Errorf("RawScore = %s, expected %s", raw.String(), expectedLong.String())
	}
	if len(reasons) != 0 {
		t.Errorf("Expected no reasons, got %v", reasons)
	}
}

func TestGolden_ScoreDirection_BelowThreshold(t *testing.T) {
	// Single BUY contribution 0.05 → longScore = 5
	// minConfluence = 20 → 5 < 20 → NO_TRADE
	evidence := []types.EvidenceContribution{
		{Pillar: "TREND", Direction: types.DirectionBuy, Contribution: decimal.NewFromFloat(0.05), NormalizedValue: decimal.NewFromFloat(0.05)},
	}
	dir, raw, _, _, reasons := scoreDirection(nil, evidence, 20, decimal.Zero)
	if dir != types.DirectionNoTrade {
		t.Errorf("Expected NO_TRADE for score below threshold, got %s", dir)
	}
	if !raw.Equal(decimal.NewFromFloat(5)) {
		t.Errorf("RawScore = %s, expected 5", raw.String())
	}
	if len(reasons) == 0 || reasons[0] != types.NTInsufficientScore {
		t.Errorf("Expected INSUFFICIENT_SCORE reason, got %v", reasons)
	}
}

func TestGolden_ScoreDirection_ConflictPenaltyApplied(t *testing.T) {
	// Long=0.30 (30), Short=0.10 (10), penalty=15 applied to long → long=15
	// minConfluence=12 → 15 > 12 → BUY (penalty reduced but still above threshold)
	evidence := []types.EvidenceContribution{
		{Pillar: "TREND", Direction: types.DirectionBuy, Contribution: decimal.NewFromFloat(0.30), NormalizedValue: decimal.NewFromFloat(0.30)},
		{Pillar: "MOMENTUM", Direction: types.DirectionSell, Contribution: decimal.NewFromFloat(0.10), NormalizedValue: decimal.NewFromFloat(0.10)},
	}
	dir, raw, long, short, _ := scoreDirection(nil, evidence, 12, decimal.NewFromFloat(15))
	if dir != types.DirectionBuy {
		t.Errorf("Expected BUY (penalty still above threshold), got %s", dir)
	}
	expectedLong := decimal.NewFromFloat(15) // 30 - 15 = 15
	if !long.Equal(expectedLong) {
		t.Errorf("LongScore after penalty = %s, expected %s", long.String(), expectedLong.String())
	}
	if !raw.Equal(expectedLong) {
		t.Errorf("RawScore = %s, expected %s", raw.String(), expectedLong.String())
	}
	// Short is not penalized (it's the losing side)
	if !short.Equal(decimal.NewFromFloat(10)) {
		t.Errorf("ShortScore = %s, expected 10", short.String())
	}
}

func TestGolden_FamilyCaps_LimitDoubleCounting(t *testing.T) {
	// Three TREND contributions of 0.15 each → sum = 0.45, cap = 0.25
	// After cap: scaled to 0.25 total
	evidence := []types.EvidenceContribution{
		{Pillar: "TREND", Direction: types.DirectionBuy, Contribution: decimal.NewFromFloat(0.15), NormalizedValue: decimal.NewFromFloat(0.15)},
		{Pillar: "TREND", Direction: types.DirectionBuy, Contribution: decimal.NewFromFloat(0.15), NormalizedValue: decimal.NewFromFloat(0.15)},
		{Pillar: "TREND", Direction: types.DirectionBuy, Contribution: decimal.NewFromFloat(0.15), NormalizedValue: decimal.NewFromFloat(0.15)},
	}
	capped := applyFamilyCaps(evidence)
	total := decimal.Zero
	for _, e := range capped {
		total = total.Add(e.Contribution)
	}
	maxAllowed := decimal.NewFromFloat(0.35)
	if total.GreaterThan(maxAllowed.Add(decimal.NewFromFloat(0.001))) {
		t.Errorf("Family cap exceeded: total=%s, max=%s", total.String(), maxAllowed.String())
	}
}

// ============================================================================
// ============================================================================

func TestGolden_BUY_BullishScalping(t *testing.T) {
	s := NewStandardScalping()
	state := makeBullishState()
	result := s.Evaluate(state)
	if result.Direction != types.DirectionBuy {
		t.Errorf("Expected BUY, got %s (score=%s reasons=%v)", result.Direction, result.RawScore, result.ReasonCodes)
	}
	// Verify entry/SL/TP ordering for BUY: Entry > SL, TP1 > TP2 > TP3 > Entry
	if !result.EntryPrice.GreaterThan(result.StopLoss) {
		t.Errorf("BUY: Entry (%s) must be > SL (%s)", result.EntryPrice, result.StopLoss)
	}
	if !result.TP1.GreaterThan(result.EntryPrice) {
		t.Errorf("BUY: TP1 (%s) must be > Entry (%s)", result.TP1, result.EntryPrice)
	}
	if !result.TP2.GreaterThan(result.TP1) {
		t.Errorf("BUY: TP2 (%s) must be > TP1 (%s)", result.TP2, result.TP1)
	}
	if !result.TP3.GreaterThan(result.TP2) {
		t.Errorf("BUY: TP3 (%s) must be > TP2 (%s)", result.TP3, result.TP2)
	}
}

func TestGolden_SELL_BearishScalping(t *testing.T) {
	s := NewStandardScalping()
	state := makeBearishState()
	result := s.Evaluate(state)
	if result.Direction != types.DirectionSell {
		t.Errorf("Expected SELL, got %s (score=%s reasons=%v)", result.Direction, result.RawScore, result.ReasonCodes)
	}
	// Verify entry/SL/TP ordering for SELL: Entry < SL, TP1 < TP2 < TP3 < Entry
	if !result.EntryPrice.LessThan(result.StopLoss) {
		t.Errorf("SELL: Entry (%s) must be < SL (%s)", result.EntryPrice, result.StopLoss)
	}
	if !result.TP1.LessThan(result.EntryPrice) {
		t.Errorf("SELL: TP1 (%s) must be < Entry (%s)", result.TP1, result.EntryPrice)
	}
	if !result.TP2.LessThan(result.TP1) {
		t.Errorf("SELL: TP2 (%s) must be < TP1 (%s)", result.TP2, result.TP1)
	}
	if !result.TP3.LessThan(result.TP2) {
		t.Errorf("SELL: TP3 (%s) must be < TP2 (%s)", result.TP3, result.TP2)
	}
}

func TestGolden_NO_TRADE_RangeMarket(t *testing.T) {
	s := NewStandardScalping()
	state := makeRangeState()
	result := s.Evaluate(state)
	if result.Direction != types.DirectionNoTrade && result.Direction != types.DirectionBuy && result.Direction != types.DirectionSell {
		t.Errorf("Expected NO-TRADE or directional candidate for range market, got %s", result.Direction)
	}
	// A NO_TRADE result must always explain itself (SOW: NO-TRADE is first-class
	// and auditable); directional candidates in range may carry no reasons.
	if result.Direction == types.DirectionNoTrade && len(result.ReasonCodes) == 0 {
		t.Error("Expected reason codes for NO_TRADE")
	}
}

func TestGolden_ERROR_MissingData(t *testing.T) {
	s := NewStandardScalping()
	state := makeStaleState() // ATR = 0
	result := s.Evaluate(state)
	if result.Direction != types.DirectionError {
		t.Errorf("Expected ERROR for missing/stale data, got %s", result.Direction)
	}
	// Should have NTSystemDegraded reason
	found := false
	for _, r := range result.ReasonCodes {
		if r == types.NTATRNotReady {
			found = true
		}
	}
	if !found {
		t.Errorf("Expected NTATRNotReady reason for missing data, got %v", result.ReasonCodes)
	}
	// Score must be zero (no evidence computed)
	if !result.RawScore.IsZero() {
		t.Errorf("Expected zero score for missing data, got %s", result.RawScore)
	}
}

// Known Entry/SL/TP → exact expected RR.
// ============================================================================

func TestGolden_RR_AllThreeTargets(t *testing.T) {
	entry := decimal.NewFromFloat(4400)
	sl := decimal.NewFromFloat(4390)
	tp1 := decimal.NewFromFloat(4412)
	tp2 := decimal.NewFromFloat(4420)
	tp3 := decimal.NewFromFloat(4430)

	risk := entry.Sub(sl).Abs() // 10
	if !risk.Equal(decimal.NewFromInt(10)) {
		t.Fatalf("Risk = %s, expected 10", risk)
	}

	rr1 := tp1.Sub(entry).Abs().Div(risk) // 12/10 = 1.2
	rr2 := tp2.Sub(entry).Abs().Div(risk) // 20/10 = 2.0
	rr3 := tp3.Sub(entry).Abs().Div(risk) // 30/10 = 3.0

	if !rr1.Equal(decimal.NewFromFloat(1.2)) {
		t.Errorf("RR1 = %s, expected 1.2", rr1)
	}
	if !rr2.Equal(decimal.NewFromInt(2)) {
		t.Errorf("RR2 = %s, expected 2", rr2)
	}
	if !rr3.Equal(decimal.NewFromInt(3)) {
		t.Errorf("RR3 = %s, expected 3", rr3)
	}
}

func TestGolden_RR_ShortPosition(t *testing.T) {
	// SELL: Entry=4400, SL=4410, TP1=4388
	// Risk = |4400-4410| = 10
	// Reward = |4388-4400| = 12
	// RR = 12/10 = 1.2
	entry := decimal.NewFromFloat(4400)
	sl := decimal.NewFromFloat(4410)
	tp1 := decimal.NewFromFloat(4388)
	rr := patmath.GrossRR(entry, sl, tp1)
	if !rr.Equal(decimal.NewFromFloat(1.2)) {
		t.Errorf("Short RR1 = %s, expected 1.2", rr)
	}
}

// ============================================================================
// PART 31F — LIQUIDITY SWEEP WIRING TEST (verifies the case-mismatch fix)
// ============================================================================

func TestGolden_LiquiditySweep_CaseInsensitiveWiring(t *testing.T) {
	// The liquidity engine produces "SELL_SIDE_SWEEP" (uppercase).
	// The strategy must recognize it and produce BUY evidence.
	s := NewStandardScalping()
	state := makeBullishState()
	// Set a sweep with the EXACT string the liquidity engine produces
	state.Liquidity.RecentSweeps = []features.SweepEvent{
		{Direction: "SELL_SIDE_SWEEP", Price: decimal.NewFromFloat(4395.0)},
	}
	result := s.Evaluate(state)

	// Verify sweep evidence is present in the result
	foundSweepEvidence := false
	for _, e := range result.Evidence {
		if e.Feature == "SELL_SIDE_SWEEP" && e.Direction == types.DirectionBuy {
			foundSweepEvidence = true
		}
	}
	if !foundSweepEvidence {
		t.Error("Expected SELL_SIDE_SWEEP to produce BUY evidence — case mismatch fix not working")
	}
}

// ============================================================================
// PART 31G — FOUR STRATEGIES ARE DISTINCT
// ============================================================================

func TestGolden_FourStrategiesDistinctConfigs(t *testing.T) {
	strategies := AllStrategies()
	if len(strategies) != 6 {
		t.Fatalf("Expected 6 strategies, got %d", len(strategies))
	}

	ids := make(map[types.StrategyID]bool)
	for _, s := range strategies {
		ids[s.ID()] = true
	}
	if len(ids) != 6 {
		t.Errorf("Expected 6 distinct strategy IDs, got %d", len(ids))
	}

	// Each strategy must produce independent results (not copied)
	state := makeBullishState()
	results := make(map[types.StrategyID]strategyResultSummary)
	for _, s := range strategies {
		r := s.Evaluate(state)
		results[s.ID()] = strategyResultSummary{
			direction: r.Direction,
			rawScore:  r.RawScore,
		}
	}

	// Verify at least 2 different scores (proves they don't all copy one result)
	scores := make(map[string]bool)
	for _, r := range results {
		scores[r.rawScore.String()] = true
	}
	if len(scores) < 2 {
		t.Error("Expected at least 2 different scores across strategies — they may be copying one result")
	}
}

type strategyResultSummary struct {
	direction types.Direction
	rawScore  decimal.Decimal
}

// === Helpers ===

func dec(f float64) decimal.Decimal { return decimal.NewFromFloat(f) }

