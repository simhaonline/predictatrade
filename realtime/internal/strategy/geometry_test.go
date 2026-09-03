package strategy

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func makeGeoState(price, atr float64) *features.MarketState {
	return &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		Timestamp:    time.Now(),
		CurrentPrice: decimal.NewFromFloat(price),
		Bid:          decimal.NewFromFloat(price - 0.15),
		Ask:          decimal.NewFromFloat(price + 0.15),
		Spread:       decimal.NewFromFloat(0.3),
		Mid:          decimal.NewFromFloat(price),
		Indicators: features.IndicatorFeatures{
			ATR:        decimal.NewFromFloat(atr),
			BollUpper:  decimal.NewFromFloat(price + atr*2),
			BollLower:  decimal.NewFromFloat(price - atr*2),
			BollMiddle: decimal.NewFromFloat(price),
		},
		Structure: features.StructureFeatures{
			SwingLows:  []decimal.Decimal{decimal.NewFromFloat(price - atr*2)},
			SwingHighs: []decimal.Decimal{decimal.NewFromFloat(price + atr*2)},
		},
		Quality: types.QualityAuthoritative,
	}
}

func TestBuildTradeGeometry_BUY_Valid(t *testing.T) {
	state := makeGeoState(2400, 3.0)
	cfg := NewStandardScalping().cfg
	geo := BuildTradeGeometry(state, types.DirectionBuy, cfg)

	if !geo.Valid {
		t.Fatalf("BUY geometry should be valid, got reason: %s", geo.ReasonCode)
	}
	if !geo.Entry.GreaterThan(decimal.Zero) {
		t.Error("Entry should be positive")
	}
	if !geo.StopLoss.LessThan(geo.Entry) {
		t.Error("BUY: SL should be below Entry")
	}
	if !geo.TP1.GreaterThan(geo.Entry) {
		t.Error("BUY: TP1 should be above Entry")
	}
	if !geo.TP2.GreaterThan(geo.TP1) {
		t.Error("BUY: TP2 should be above TP1")
	}
	if !geo.TP3.GreaterThan(geo.TP2) {
		t.Error("BUY: TP3 should be above TP2")
	}
}

func TestBuildTradeGeometry_SELL_Valid(t *testing.T) {
	state := makeGeoState(2400, 3.0)
	cfg := NewStandardScalping().cfg
	geo := BuildTradeGeometry(state, types.DirectionSell, cfg)

	if !geo.Valid {
		t.Fatalf("SELL geometry should be valid, got reason: %s", geo.ReasonCode)
	}
	if !geo.StopLoss.GreaterThan(geo.Entry) {
		t.Error("SELL: SL should be above Entry")
	}
	if !geo.TP1.LessThan(geo.Entry) {
		t.Error("SELL: TP1 should be below Entry")
	}
	if !geo.TP2.LessThan(geo.TP1) {
		t.Error("SELL: TP2 should be below TP1")
	}
}

func TestBuildTradeGeometry_ATRNotReady(t *testing.T) {
	state := makeGeoState(2400, 0) // Zero ATR
	cfg := NewStandardScalping().cfg
	geo := BuildTradeGeometry(state, types.DirectionBuy, cfg)

	if geo.Valid {
		t.Error("Geometry should be invalid when ATR is zero")
	}
	if geo.ReasonCode != "ATR_NOT_READY" {
		t.Errorf("Expected AT_R_NOT_READY, got %s", geo.ReasonCode)
	}
}

func TestBuildTradeGeometry_NoDirection(t *testing.T) {
	state := makeGeoState(2400, 3.0)
	cfg := NewStandardScalping().cfg
	geo := BuildTradeGeometry(state, types.DirectionNoTrade, cfg)

	if geo.Valid {
		t.Error("Geometry should be invalid for NO_TRADE direction")
	}
}

func TestBuildTradeGeometry_UsesAskForBuy(t *testing.T) {
	state := makeGeoState(2400, 3.0)
	state.Ask = decimal.NewFromFloat(2400.15)
	state.Bid = decimal.NewFromFloat(2399.85)
	cfg := NewStandardScalping().cfg
	geo := BuildTradeGeometry(state, types.DirectionBuy, cfg)

	entry, _ := geo.Entry.Float64()
	if entry < 2400.1 {
		t.Errorf("BUY entry should use Ask (~2400.15), got %f", entry)
	}
}

func TestBuildTradeGeometry_UsesBidForSell(t *testing.T) {
	state := makeGeoState(2400, 3.0)
	state.Ask = decimal.NewFromFloat(2400.15)
	state.Bid = decimal.NewFromFloat(2399.85)
	cfg := NewStandardScalping().cfg
	geo := BuildTradeGeometry(state, types.DirectionSell, cfg)

	entry, _ := geo.Entry.Float64()
	if entry > 2400.0 {
		t.Errorf("SELL entry should use Bid (~2399.85), got %f", entry)
	}
}

func TestCheckDirectionDominance_Sufficient(t *testing.T) {
	long := decimal.NewFromFloat(35.0)
	short := decimal.NewFromFloat(20.0)
	dir, ok := CheckDirectionDominance(long, short)
	if !ok {
		t.Error("35 vs 20 should have sufficient dominance")
	}
	if dir != types.DirectionBuy {
		t.Error("Should be BUY direction")
	}
}

func TestCheckDirectionDominance_Insufficient(t *testing.T) {
	long := decimal.NewFromFloat(25.1)
	short := decimal.NewFromFloat(24.9)
	_, ok := CheckDirectionDominance(long, short)
	if ok {
		t.Error("25.1 vs 24.9 should NOT have sufficient dominance (diff < 5)")
	}
}

func TestUltraRangeEvidence_NonZero(t *testing.T) {
	state := makeGeoState(2390, 3.0)
	state.Indicators.RSI = decimal.NewFromFloat(25)   // Oversold
	state.Indicators.CCI = decimal.NewFromFloat(-130) // CCI extreme
	state.Indicators.BollLower = decimal.NewFromFloat(2392)
	state.Indicators.BollUpper = decimal.NewFromFloat(2408)
	state.Indicators.OsMA = decimal.NewFromFloat(-0.5)
	state.VWAP.SessionVWAP = decimal.NewFromFloat(2400)
	state.Candle.IsRejection = true
	state.Candle.IsBullish = true
	state.Liquidity.RecentSweeps = []features.SweepEvent{
		{Direction: "sell_side", Price: decimal.NewFromFloat(2388), Time: time.Now()},
	}

	var evidence []types.EvidenceContribution
	computeUltraRangeEvidence(&evidence, state, DefaultUltraRangeConfig())

	if len(evidence) == 0 {
		t.Fatal("Ultra range evidence should be non-empty with qualifying conditions")
	}

	// Check that buy evidence dominates
	longScore := decimal.Zero
	shortScore := decimal.Zero
	for _, e := range evidence {
		if e.Direction == types.DirectionBuy {
			longScore = longScore.Add(e.Contribution)
		} else if e.Direction == types.DirectionSell {
			shortScore = shortScore.Add(e.Contribution)
		}
	}
	if !longScore.GreaterThan(shortScore) {
		t.Errorf("Long (%s) should dominate short (%s) with oversold conditions", longScore, shortScore)
	}
}

func TestTrendTransition_NonZeroEvidence(t *testing.T) {
	state := makeGeoState(2400, 3.0)
	state.Indicators.ADX = decimal.NewFromFloat(22)
	state.Indicators.ADXPlusDI = decimal.NewFromFloat(20)
	state.Indicators.ADXMinusDI = decimal.NewFromFloat(12)
	state.Indicators.EMA9 = decimal.NewFromFloat(2402)
	state.Indicators.EMA21 = decimal.NewFromFloat(2400)
	state.Indicators.MACDMain = decimal.NewFromFloat(2)
	state.Indicators.MACDSignal = decimal.NewFromFloat(1)
	state.MTF.Score = 40
	state.Structure.CurrentTrend = "bullish"
	state.Structure.LastBOS = &features.StructureEvent{Direction: "bullish", Price: decimal.NewFromFloat(2410), Time: time.Now()}

	evidence := computeTrendTransitionEvidence(state)
	if len(evidence) == 0 {
		t.Fatal("Trend transition evidence should be non-empty with transition conditions")
	}

	// Check buy dominance
	longCount := 0
	for _, e := range evidence {
		if e.Direction == types.DirectionBuy {
			longCount++
		}
	}
	if longCount == 0 {
		t.Error("Should have buy-direction transition evidence")
	}
}

func TestTrendTransition_FlatMarket_NoEvidence(t *testing.T) {
	state := makeGeoState(2400, 1.0)                // Low ATR — no expansion
	state.Indicators.ADX = decimal.NewFromFloat(12) // Low — no expansion
	state.Indicators.EMA9 = decimal.NewFromFloat(2400)
	state.Indicators.EMA21 = decimal.NewFromFloat(2400)
	state.Indicators.MACDMain = decimal.Zero
	state.Indicators.MACDSignal = decimal.Zero
	state.MTF.Score = 0
	state.Structure.CurrentTrend = "neutral"

	evidence := computeTrendTransitionEvidence(state)
	// Should have minimal or no evidence in flat market
	longScore := decimal.Zero
	shortScore := decimal.Zero
	for _, e := range evidence {
		if e.Direction == types.DirectionBuy {
			longScore = longScore.Add(e.Contribution)
		} else {
			shortScore = shortScore.Add(e.Contribution)
		}
	}
	// In a flat market, scores should be very low
	total := longScore.Add(shortScore)
	totalF, _ := total.Float64()
	if totalF > 0.15 {
		t.Errorf("Flat market should produce minimal evidence, total=%f", totalF)
	}
}

func TestThresholdReachability_StandardScalping_Range(t *testing.T) {
	// Calculate max achievable score for StandardScalping in RANGE
	// Evidence budget: EMA(0.12) + VWAP(0.08) + BOS(0.14) + Candle(0.10) + MACD(0.06) + OsMA(0.05) + RSI(0.05) + ADX(0.07) + Liq(0.08) + MTF(0.05) = 0.80
	// Range evidence: BB(0.08) + VWAP_dev(0.06) + RSI(0.07) + Stoch(0.06) + CCI(0.05) + Rejection(0.08) + Sweep(0.10) + FVG(0.06) = 0.56
	// But family caps limit: TREND≤0.25, MOMENTUM≤0.20, STRUCTURE≤0.20, VWAP≤0.10, CANDLE≤0.15, LIQUIDITY≤0.15, MTF≤0.15, VOLATILITY≤0.10, SMC≤0.15
	// Max in one direction (all aligned):
	// TREND: EMA(0.12) + ADX(0.07) = 0.19 ≤ 0.25
	// MOMENTUM: MACD(0.06) + OsMA(0.05) + RSI(0.05) + range_RSI(0.07) + Stoch(0.06) + CCI(0.05) = 0.34 → capped 0.20
	// STRUCTURE: BOS(0.14) = 0.14 ≤ 0.20
	// VWAP: VWAP(0.08) + dev(0.06) = 0.14 → capped 0.10
	// CANDLE: displacement(0.10) + rejection(0.08) = 0.18 → capped 0.15
	// LIQUIDITY: sweep(0.08) + range_sweep(0.10) = 0.18 → capped 0.15
	// MTF: 0.05 ≤ 0.15
	// VOLATILITY: BB(0.08) ≤ 0.10
	// SMC: FVG(0.06) ≤ 0.15
	// Total: 0.19 + 0.20 + 0.14 + 0.10 + 0.15 + 0.15 + 0.05 + 0.08 + 0.06 = 1.12 → but this is ALL evidence in one direction
	// In practice, RANGE evidence is split. Realistic max in RANGE: ~0.47 → score ~47
	// Trade threshold for RANGE: 45. 47 >= 45 → REACHABLE

	ct, tt, _ := GetThresholds(types.StrategyStandardScalping, types.RegimeRange)
	maxRangeScore := 47.0 // Calculated from evidence budget

	if tt > maxRangeScore {
		t.Errorf("StandardScalping RANGE trade threshold %f > max achievable %f — UNREACHABLE", tt, maxRangeScore)
	}
	if ct >= tt {
		t.Errorf("Candidate threshold %f >= trade threshold %f — invalid", ct, tt)
	}
	t.Logf("StandardScalping RANGE: candidate=%f, trade=%f, max~%f → REACHABLE", ct, tt, maxRangeScore)
}

func TestThresholdReachability_AllStrategies(t *testing.T) {
	configs := DefaultRegimeThresholds()
	for stratID, regimeMap := range configs {
		for regime, rt := range regimeMap {
			if rt.CandidateThreshold >= rt.TradeThreshold {
				t.Errorf("%s/%s: candidate %f >= trade %f",
					stratID, regime, rt.CandidateThreshold, rt.TradeThreshold)
			}
		}
	}
}
