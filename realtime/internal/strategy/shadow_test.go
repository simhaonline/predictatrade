package strategy

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func TestShadowEvaluation_RegimeMismatchProducesShadow(t *testing.T) {
	trendSwing := NewTrendSwing()

	// Create state with MEAN_REVERSION regime (not accepted by TrendSwing)
	state := &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		Timestamp:    time.Now(),
		CurrentPrice: decimal.NewFromFloat(2400),
		Bid:          decimal.NewFromFloat(2399),
		Ask:          decimal.NewFromFloat(2401),
		Spread:       decimal.NewFromFloat(2),
		Mid:          decimal.NewFromFloat(2400),
		Indicators: features.IndicatorFeatures{
			ATR:        decimal.NewFromFloat(2),
			RSI:        decimal.NewFromFloat(48),
			ADX:        decimal.NewFromFloat(15),
			EMA9:       decimal.NewFromFloat(2400),
			EMA21:      decimal.NewFromFloat(2400),
			EMA50:      decimal.NewFromFloat(2400),
			EMA100:     decimal.NewFromFloat(2400),
			EMA200:     decimal.NewFromFloat(2400),
			SMA200:     decimal.NewFromFloat(2400),
			MACDMain:   decimal.NewFromFloat(1),
			MACDSignal: decimal.NewFromFloat(0.5),
			OsMA:       decimal.NewFromFloat(0.5),
			CCI:        decimal.NewFromFloat(50),
		},
		Regime: features.RegimeFeatures{
			Current:    types.RegimeMeanReversion,
			Confidence: 0.7,
		},
		Session: features.SessionFeatures{
			CurrentSession: "LONDON",
			NewsRisk:       "LOW",
		},
		Quality: types.QualityAuthoritative,
	}

	// Production evaluation should reject
	prodResult := trendSwing.Evaluate(state)
	if prodResult.Direction != types.DirectionNoTrade {
		t.Errorf("Production should reject on regime mismatch, got %s", prodResult.Direction)
	}

	// Shadow evaluation should produce hypothetical result
	shadow := EvaluateShadow(trendSwing, state)
	if shadow == nil {
		t.Fatal("Shadow evaluation should produce a result for regime-mismatched strategy")
	}
	if !shadow.ShadowOnly {
		t.Error("Shadow result must be marked ShadowOnly=true")
	}
	if shadow.Executable {
		t.Error("Shadow result must be marked Executable=false")
	}
	if shadow.FailedProductionReason == "" {
		t.Error("Shadow result must have FailedProductionReason")
	}
}

func TestShadowEvaluation_RegimeMatchProducesNil(t *testing.T) {
	trendSwing := NewTrendSwing()

	state := &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		Timestamp:    time.Now(),
		CurrentPrice: decimal.NewFromFloat(2400),
		Indicators: features.IndicatorFeatures{
			ATR:    decimal.NewFromFloat(2),
			ADX:    decimal.NewFromFloat(32),
			EMA9:   decimal.NewFromFloat(2415),
			EMA21:  decimal.NewFromFloat(2405),
			EMA50:  decimal.NewFromFloat(2395),
			EMA100: decimal.NewFromFloat(2400),
			EMA200: decimal.NewFromFloat(2395),
			SMA200: decimal.NewFromFloat(2400),
		},
		Regime: features.RegimeFeatures{
			Current: types.RegimeTrendingBullish, // Accepted by TrendSwing
		},
		Session: features.SessionFeatures{
			CurrentSession: "LONDON",
			NewsRisk:       "LOW",
		},
		Quality: types.QualityAuthoritative,
	}

	shadow := EvaluateShadow(trendSwing, state)
	if shadow != nil {
		t.Error("Shadow evaluation should return nil when regime matches")
	}
}

func TestShadowEvaluation_AllShadows(t *testing.T) {
	strategies := AllStrategies()

	state := &features.MarketState{
		Symbol:       types.SymbolXAUUSD,
		Timestamp:    time.Now(),
		CurrentPrice: decimal.NewFromFloat(2400),
		Indicators: features.IndicatorFeatures{
			ATR:    decimal.NewFromFloat(2),
			RSI:    decimal.NewFromFloat(48),
			ADX:    decimal.NewFromFloat(15),
			EMA9:   decimal.NewFromFloat(2400),
			EMA21:  decimal.NewFromFloat(2400),
			EMA50:  decimal.NewFromFloat(2400),
			SMA200: decimal.NewFromFloat(2400),
		},
		Regime: features.RegimeFeatures{
			Current: types.RegimeRange, // Not accepted by TrendSwing or UltraScalping (StandardScalping/Swing accept it)
		},
		Session: features.SessionFeatures{
			CurrentSession: "LONDON",
			NewsRisk:       "LOW",
		},
		Quality: types.QualityAuthoritative,
	}

	shadows := EvaluateAllShadows(strategies, state)

	// At least TrendSwing and UltraScalping should produce shadows
	// (StandardScalping and StandardSwing accept MEAN_REVERSION)
	if len(shadows) < 1 {
		t.Errorf("Expected at least 1 shadow result, got %d", len(shadows))
	}

	for _, s := range shadows {
		if !s.ShadowOnly {
			t.Error("All shadow results must be ShadowOnly=true")
		}
		if s.Executable {
			t.Error("All shadow results must be Executable=false")
		}
	}
}
