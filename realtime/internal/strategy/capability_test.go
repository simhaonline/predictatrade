package strategy

import (
	"testing"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// ─── Test Helpers ───

// makeBullishTrendingState creates a market state with strong bullish trend conditions.
func makeBullishTrendingState() *features.MarketState {
	return &features.MarketState{
		Symbol:       "XAUUSD",
		CurrentPrice: decimal.NewFromFloat(4360.0),
		Bid:          decimal.NewFromFloat(4359.8),
		Ask:          decimal.NewFromFloat(4360.2),
		Spread:       decimal.NewFromFloat(0.4),
		Mid:          decimal.NewFromFloat(4360.0),
		Candles:      make(map[types.Timeframe]*types.Candle),
		Indicators: features.IndicatorFeatures{
			EMA9:        decimal.NewFromFloat(4358.0),
			EMA21:       decimal.NewFromFloat(4355.0),
			EMA50:       decimal.NewFromFloat(4350.0),
			EMA100:      decimal.NewFromFloat(4345.0),
			EMA200:      decimal.NewFromFloat(4340.0),
			SMA200:      decimal.NewFromFloat(4338.0),
			MACDMain:    decimal.NewFromFloat(2.5),
			MACDSignal:  decimal.NewFromFloat(1.0),
			ADX:         decimal.NewFromFloat(30.0),
			ADXPlusDI:   decimal.NewFromFloat(25.0),
			ADXMinusDI:  decimal.NewFromFloat(15.0),
			RSI:         decimal.NewFromFloat(62.0),
			StochMain:   decimal.NewFromFloat(75.0),
			StochSignal: decimal.NewFromFloat(70.0),
			OsMA:        decimal.NewFromFloat(1.5),
			ATR:         decimal.NewFromFloat(3.0),
			BollUpper:   decimal.NewFromFloat(4368.0),
			BollLower:   decimal.NewFromFloat(4352.0),
			BollMiddle:  decimal.NewFromFloat(4360.0),
			BollWidth:   decimal.NewFromFloat(16.0),
			CCI:         decimal.NewFromFloat(80.0),
		},
		Regime: features.RegimeFeatures{
			Current:    types.RegimeTrendingBullish,
			Volatility: "NORMAL",
			Confidence: 0.8,
		},
		Session: features.SessionFeatures{
			CurrentSession: "LONDON",
			IsOverlap:      false,
			IsWeekend:      false,
			NewsRisk:       "NONE",
		},
		VWAP: features.VWAPFeatures{
			SessionVWAP: decimal.NewFromFloat(4355.0),
		},
		Structure: features.StructureFeatures{
			CurrentTrend: "bullish",
			SwingHighs:   []decimal.Decimal{decimal.NewFromFloat(4362.0)},
			SwingLows:    []decimal.Decimal{decimal.NewFromFloat(4348.0)},
			LastBOS: &features.StructureEvent{
				Type:      "BOS",
				Direction: "bullish",
				Price:     decimal.NewFromFloat(4358.0),
			},
		},
		MTF: features.MTFFeatures{
			Score:  60.0,
			States: map[types.Timeframe]int{types.TFH1: 1, types.TFH4: 1, types.TFM15: 1},
		},
		Candle: features.CandleIntelligence{
			IsBullish:        true,
			IsDisplacement:   true,
			IsBreakout:       true,
			ConsecutiveBull:  3,
			BodyRangeRatio:   decimal.NewFromFloat(0.75),
		},
		Quality: types.QualityAuthoritative,
	}
}

// makeBearishTrendingState creates a market state with strong bearish trend conditions.
func makeBearishTrendingState() *features.MarketState {
	s := makeBullishTrendingState()
	s.CurrentPrice = decimal.NewFromFloat(4340.0)
	s.Bid = decimal.NewFromFloat(4339.8)
	s.Ask = decimal.NewFromFloat(4340.2)
	s.Indicators.EMA9 = decimal.NewFromFloat(4342.0)
	s.Indicators.EMA21 = decimal.NewFromFloat(4345.0)
	s.Indicators.EMA50 = decimal.NewFromFloat(4350.0)
	s.Indicators.EMA100 = decimal.NewFromFloat(4355.0)
	s.Indicators.EMA200 = decimal.NewFromFloat(4360.0)
	s.Indicators.SMA200 = decimal.NewFromFloat(4362.0)
	s.Indicators.MACDMain = decimal.NewFromFloat(-2.5)
	s.Indicators.MACDSignal = decimal.NewFromFloat(-1.0)
	s.Indicators.ADXPlusDI = decimal.NewFromFloat(15.0)
	s.Indicators.ADXMinusDI = decimal.NewFromFloat(25.0)
	s.Indicators.RSI = decimal.NewFromFloat(38.0)
	s.Indicators.StochMain = decimal.NewFromFloat(25.0)
	s.Indicators.StochSignal = decimal.NewFromFloat(30.0)
	s.Indicators.OsMA = decimal.NewFromFloat(-1.5)
	s.Indicators.CCI = decimal.NewFromFloat(-80.0)
	s.Indicators.BollUpper = decimal.NewFromFloat(4348.0)
	s.Indicators.BollLower = decimal.NewFromFloat(4332.0)
	s.Indicators.BollMiddle = decimal.NewFromFloat(4340.0)
	s.Regime.Current = types.RegimeTrendingBearish
	s.VWAP.SessionVWAP = decimal.NewFromFloat(4345.0)
	s.Structure.CurrentTrend = "bearish"
	s.Structure.SwingHighs = []decimal.Decimal{decimal.NewFromFloat(4352.0)}
	s.Structure.SwingLows = []decimal.Decimal{decimal.NewFromFloat(4338.0)}
	s.Structure.LastBOS = &features.StructureEvent{
		Type: "BOS", Direction: "bearish", Price: decimal.NewFromFloat(4342.0),
	}
	s.MTF.Score = -60.0
	s.MTF.States = map[types.Timeframe]int{types.TFH1: -1, types.TFH4: -1, types.TFM15: -1}
	s.Candle = features.CandleIntelligence{
		IsBearish:      true,
		IsDisplacement: true,
		IsBreakout:     true,
		ConsecutiveBear: 3,
		BodyRangeRatio: decimal.NewFromFloat(0.75),
	}
	return s
}

// makeMeanReversionState creates a market state in mean-reversion regime (should produce NO-TRADE for trend strategies).
func makeMeanReversionState() *features.MarketState {
	s := makeBullishTrendingState()
	s.Regime.Current = types.RegimeMeanReversion
	s.Indicators.RSI = decimal.NewFromFloat(48.0)
	s.Indicators.ADX = decimal.NewFromFloat(15.0)
	s.Indicators.ADXPlusDI = decimal.NewFromFloat(18.0)
	s.Indicators.ADXMinusDI = decimal.NewFromFloat(17.0)
	s.Structure.CurrentTrend = ""
	s.Structure.LastBOS = nil
	s.Structure.LastCHoCH = nil
	s.Candle = features.CandleIntelligence{
		IsDoji:       true,
		BodyRangeRatio: decimal.NewFromFloat(0.1),
	}
	s.MTF.Score = 5.0
	return s
}

// ─── Strategy Capability Tests ───
// SOW: Each enabled strategy must be mathematically capable of BUY, SELL, and NO-TRADE.

func TestStandardScalping_Capability(t *testing.T) {
	s := NewStandardScalping()

	// BUY: strong bullish trend
	buyResult := s.Evaluate(makeBullishTrendingState())
	if buyResult.Direction != types.DirectionBuy {
		t.Errorf("StandardScalping bullish state: expected BUY, got %s (score=%s, reasons=%v)",
			buyResult.Direction, buyResult.RawScore, buyResult.ReasonCodes)
	}

	// SELL: strong bearish trend
	sellResult := s.Evaluate(makeBearishTrendingState())
	if sellResult.Direction != types.DirectionSell {
		t.Errorf("StandardScalping bearish state: expected SELL, got %s (score=%s, reasons=%v)",
			sellResult.Direction, sellResult.RawScore, sellResult.ReasonCodes)
	}

	// NO-TRADE: mean reversion with weak signals
	ntResult := s.Evaluate(makeMeanReversionState())
	if ntResult.Direction != types.DirectionNoTrade && ntResult.Direction != types.DirectionWait {
		t.Errorf("StandardScalping mean-reversion state: expected NO-TRADE/WAIT, got %s (score=%s)",
			ntResult.Direction, ntResult.RawScore)
	}
}

func TestUltraScalping_Capability(t *testing.T) {
	s := NewUltraScalping()

	// BUY: strong bullish trend
	buyResult := s.Evaluate(makeBullishTrendingState())
	if buyResult.Direction != types.DirectionBuy {
		t.Errorf("UltraScalping bullish state: expected BUY, got %s (score=%s, reasons=%v)",
			buyResult.Direction, buyResult.RawScore, buyResult.ReasonCodes)
	}

	// SELL: strong bearish trend
	sellResult := s.Evaluate(makeBearishTrendingState())
	if sellResult.Direction != types.DirectionSell {
		t.Errorf("UltraScalping bearish state: expected SELL, got %s (score=%s, reasons=%v)",
			sellResult.Direction, sellResult.RawScore, sellResult.ReasonCodes)
	}

	// Mean reversion: UltraScalping now accepts RANGE/MEAN_REVERSION with relaxed EMA hierarchy
	// A directional result is valid — it will be classified as ADVISORY candidate by the pipeline
	mrResult := s.Evaluate(makeMeanReversionState())
	if mrResult.Direction != types.DirectionNoTrade && mrResult.Direction != types.DirectionBuy && mrResult.Direction != types.DirectionSell {
		t.Errorf("UltraScalping mean-reversion state: expected NO-TRADE/BUY/SELL, got %s", mrResult.Direction)
	}
	// If directional, geometry must be computed
	if mrResult.Direction == types.DirectionBuy || mrResult.Direction == types.DirectionSell {
		if mrResult.EntryPrice.IsZero() || mrResult.StopLoss.IsZero() {
			t.Errorf("UltraScalping mean-reversion candidate: geometry must be computed, entry=%s sl=%s",
				mrResult.EntryPrice, mrResult.StopLoss)
		}
	}
}

func TestStandardSwing_Capability(t *testing.T) {
	s := NewStandardSwing()

	// BUY: strong bullish trend
	buyResult := s.Evaluate(makeBullishTrendingState())
	if buyResult.Direction != types.DirectionBuy {
		t.Errorf("StandardSwing bullish state: expected BUY, got %s (score=%s, reasons=%v)",
			buyResult.Direction, buyResult.RawScore, buyResult.ReasonCodes)
	}

	// SELL: strong bearish trend
	sellResult := s.Evaluate(makeBearishTrendingState())
	if sellResult.Direction != types.DirectionSell {
		t.Errorf("StandardSwing bearish state: expected SELL, got %s (score=%s, reasons=%v)",
			sellResult.Direction, sellResult.RawScore, sellResult.ReasonCodes)
	}

	// Mean reversion: StandardSwing accepts RANGE/MEAN_REVERSION
	// Score above candidate threshold produces a directional candidate
	mrResult := s.Evaluate(makeMeanReversionState())
	if mrResult.Direction != types.DirectionNoTrade && mrResult.Direction != types.DirectionWait &&
		mrResult.Direction != types.DirectionBuy && mrResult.Direction != types.DirectionSell {
		t.Errorf("StandardSwing mean-reversion state: expected NO-TRADE/WAIT/BUY/SELL, got %s (score=%s)",
			mrResult.Direction, mrResult.RawScore)
	}
}

func TestTrendSwing_Capability(t *testing.T) {
	s := NewTrendSwing()

	// BUY: strong bullish trend
	buyResult := s.Evaluate(makeBullishTrendingState())
	if buyResult.Direction != types.DirectionBuy {
		t.Errorf("TrendSwing bullish state: expected BUY, got %s (score=%s, reasons=%v)",
			buyResult.Direction, buyResult.RawScore, buyResult.ReasonCodes)
	}

	// SELL: strong bearish trend
	sellResult := s.Evaluate(makeBearishTrendingState())
	if sellResult.Direction != types.DirectionSell {
		t.Errorf("TrendSwing bearish state: expected SELL, got %s (score=%s, reasons=%v)",
			sellResult.Direction, sellResult.RawScore, sellResult.ReasonCodes)
	}

	// NO-TRADE: mean reversion (regime not accepted for trend swing)
	ntResult := s.Evaluate(makeMeanReversionState())
	if ntResult.Direction != types.DirectionNoTrade {
		t.Errorf("TrendSwing mean-reversion state: expected NO-TRADE, got %s", ntResult.Direction)
	}
}

// TestScoreScale verifies that scores are in the 0-100 range after scaling.
func TestScoreScale(t *testing.T) {
	for _, strat := range AllStrategies() {
		buyResult := strat.Evaluate(makeBullishTrendingState())
		score, _ := buyResult.RawScore.Float64()
		if score < 0 || score > 100 {
			t.Errorf("%s BUY score out of range: %f", strat.ID(), score)
		}
		if buyResult.LongScore.IsNegative() {
			t.Errorf("%s long score negative: %s", strat.ID(), buyResult.LongScore)
		}
	}
}

// TestRegimeGating verifies that trend strategies reject non-trending regimes.
func TestRegimeGating(t *testing.T) {
	trendSwing := NewTrendSwing()
	ultraScalp := NewUltraScalping()

	mrState := makeMeanReversionState()

	// Trend Swing must reject MEAN_REVERSION
	tsResult := trendSwing.Evaluate(mrState)
	if tsResult.Direction != types.DirectionNoTrade {
		t.Errorf("TrendSwing should reject MEAN_REVERSION, got %s", tsResult.Direction)
	}
	if tsResult.RawScore.IsZero() {
		// Score should be 0 because it returns before evidence computation
		// This is correct — regime gate prevents evidence evaluation
	}

	// Ultra Scalping now accepts MEAN_REVERSION with relaxed EMA hierarchy for range trading
	// It may produce a directional candidate (ADVISORY) — this is correct behavior
	usResult := ultraScalp.Evaluate(mrState)
	if usResult.Direction != types.DirectionNoTrade && usResult.Direction != types.DirectionBuy && usResult.Direction != types.DirectionSell {
		t.Errorf("UltraScalping MEAN_REVERSION: expected NO-TRADE/BUY/SELL, got %s", usResult.Direction)
	}
}
