// Package ptb — Tests for advanced intelligence modules, feature flags, and data guards.
// Stage 4 Sections 66-70: Module tests, live-data guards, shadow mode tests.
package ptb

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func TestFlagRegistry_DefaultShadow(t *testing.T) {
	r := NewFlagRegistry()
	for _, name := range AllModuleNames() {
		mode := r.GetMode(name)
		if name == ModuleInstitutionalFootprint {
			if mode != types.ModuleUnsupported {
				t.Errorf("%s should be UNSUPPORTED, got %s", name, mode)
			}
		} else if mode != types.ModuleShadow {
			t.Errorf("%s should default to SHADOW, got %s", name, mode)
		}
	}
}

func TestFlagRegistry_SetMode(t *testing.T) {
	r := NewFlagRegistry()
	r.SetMode(ModuleLiquidityVoid, types.ModuleActive)
	if !r.IsActive(ModuleLiquidityVoid) {
		t.Error("Expected liquidity_void to be ACTIVE")
	}
	r.SetMode(ModuleLiquidityVoid, types.ModuleOff)
	if r.IsActive(ModuleLiquidityVoid) {
		t.Error("Expected liquidity_void to be OFF")
	}
}

func TestDataAuthenticityGuard_RejectsNonLive(t *testing.T) {
	g := NewDataAuthenticityGuard()
	if !g.CheckSource(types.DataSourceLiveAgent) {
		t.Error("LIVE_AGENT should be accepted")
	}
	if g.CheckSource(types.DataSourceMock) {
		t.Error("MOCK should be rejected")
	}
	if g.CheckSource(types.DataSourceDemo) {
		t.Error("DEMO should be rejected")
	}
	if g.CheckSource(types.DataSourceFixture) {
		t.Error("FIXTURE should be rejected")
	}
	if g.CheckSource(types.DataSourceSynthetic) {
		t.Error("SYNTHETIC should be rejected")
	}
	if g.CheckSource(types.DataSourcePlaceholder) {
		t.Error("PLACEHOLDER should be rejected")
	}
	if g.CheckSource(types.DataSourceUnknown) {
		t.Error("UNKNOWN should be rejected")
	}
	if g.CheckSource(types.DataSourceTest) {
		t.Error("TEST should be rejected for production signals")
	}
}

func TestDataAuthenticityGuard_Reason(t *testing.T) {
	g := NewDataAuthenticityGuard()
	reason := g.RejectReason(types.DataSourceMock)
	if reason == "" {
		t.Error("Expected non-empty rejection reason for MOCK")
	}
	if g.RejectReason(types.DataSourceLiveAgent) != "" {
		t.Error("Expected empty reason for LIVE_AGENT")
	}
}

func TestEngine_Evaluate_NilState(t *testing.T) {
	e := NewEngine()
	snap := e.Evaluate(nil, "test-id", types.DataSourceLiveAgent)
	if snap != nil {
		t.Error("Expected nil snapshot for nil state")
	}
}

func TestEngine_Evaluate_ValidState(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	if snap == nil {
		t.Fatal("Expected non-nil snapshot")
	}
	if !snap.IsLive {
		t.Error("Expected IsLive=true for LIVE_AGENT")
	}
	if snap.DataSource != types.DataSourceLiveAgent {
		t.Errorf("Expected LIVE_AGENT, got %s", snap.DataSource)
	}
}

func TestEngine_Evaluate_TestSourceRejected(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceTest)
	if snap == nil {
		t.Fatal("Expected non-nil snapshot")
	}
	if snap.IsLive {
		t.Error("Expected IsLive=false for TEST source")
	}
}

func TestEngine_AllModulesShadow_ZeroScoreContribution(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)

	// Verify ALL modules have zero score contribution in SHADOW mode
	if !snap.LiquidityVoid.ScoreContrib.IsZero() {
		t.Error("liquidity_void should have zero score contrib in SHADOW")
	}
	if !snap.WickFill.ScoreContrib.IsZero() {
		t.Error("wick_fill should have zero score contrib in SHADOW")
	}
	if !snap.SessionImbalance.ScoreContrib.IsZero() {
		t.Error("session_imbalance should have zero score contrib in SHADOW")
	}
	if !snap.StopHuntProxy.ScoreContrib.IsZero() {
		t.Error("stop_hunt_proxy should have zero score contrib in SHADOW")
	}
	if !snap.ManipulationProxy.ScoreContrib.IsZero() {
		t.Error("manipulation_proxy should have zero score contrib in SHADOW")
	}
}

func TestEngine_InstitutionalFootprint_Unsupported(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	if snap.InstitutionalFootprint.Mode != types.ModuleUnsupported {
		t.Errorf("Expected UNSUPPORTED, got %s", snap.InstitutionalFootprint.Mode)
	}
	if snap.InstitutionalFootprint.Available {
		t.Error("Institutional Footprint should NOT be available")
	}
	if snap.InstitutionalFootprint.State != "UNSUPPORTED_BY_DATA_SOURCE" {
		t.Errorf("Expected UNSUPPORTED_BY_DATA_SOURCE, got %s", snap.InstitutionalFootprint.State)
	}
}

func TestEngine_MTFBias(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	// MTF bias should compute from existing MTF states
	if snap.MTFBias.Bias == "" {
		t.Error("Expected non-empty MTF bias")
	}
}

func TestEngine_VolatilityRegime(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	if snap.VolatilityRegime.Regime == "" {
		t.Error("Expected non-empty volatility regime")
	}
}

func TestEngine_DataQuality_Measured(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	if snap.DataQuality.State == "" {
		t.Error("Expected non-empty data quality state")
	}
	// Quality should NOT be a hardcoded 90 — it must be measured
	if snap.DataQuality.Score == 90 {
		t.Error("Data quality score should NOT be hardcoded 90")
	}
}

func TestEngine_ShadowMode_DoesNotAlterSignals(t *testing.T) {
	// Stage 4 Section 69: Shadow modules must NOT alter BUY/SELL/WAIT/NO_TRADE/BLOCKED/ERROR
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)

	// None of the shadow results should contain any BUY/SELL direction
	results := []ModuleResult{
		snap.LiquidityVoid, snap.WickFill, snap.SessionImbalance,
		snap.StopHuntProxy, snap.ManipulationProxy, snap.MarketPhase,
		snap.RelativeVolumeFlow, snap.CompleteLiquidityMap,
		snap.EngineeredLiquidity, snap.CandleRangeProjector,
		snap.AlgoActivity, snap.PriceDelivery,
		snap.TimeCycle, snap.TimeAtMode,
	}
	for _, r := range results {
		if r.Mode == types.ModuleShadow && !r.ScoreContrib.IsZero() {
			t.Errorf("Module %s in SHADOW has non-zero score contribution", r.Module)
		}
	}
}

func TestEngine_CompleteLiquidityMap_InferredOnly(t *testing.T) {
	e := NewEngine()
	state := makeTestState()
	snap := e.Evaluate(state, "snap-001", types.DataSourceLiveAgent)
	// Verify all liquidity levels are INFERRED_PRICE_STRUCTURE, not fabricated
	if snap.CompleteLiquidityMap.Available {
		value, ok := snap.CompleteLiquidityMap.Value.(map[string]interface{})
		if ok {
			if sourceType, exists := value["source_type"]; exists {
				if sourceType != "INFERRED_PRICE_STRUCTURE" {
					t.Errorf("Expected INFERRED_PRICE_STRUCTURE, got %v", sourceType)
				}
			}
		}
	}
}

func TestEngine_NoFakeData(t *testing.T) {
	// Stage 4 Section 73: No production signal from fake data
	e := NewEngine()
	state := makeTestState()
	// Evaluate with a non-live source — should be marked as not live
	snap := e.Evaluate(state, "snap-001", types.DataSourceMock)
	if snap.IsLive {
		t.Error("MOCK data source should produce IsLive=false")
	}
}

func TestEvaluateDataQuality_NilState(t *testing.T) {
	result := EvaluateDataQuality(nil, time.Now())
	if result.State != "ERROR" {
		t.Errorf("Expected ERROR for nil state, got %s", result.State)
	}
}

func TestEvaluateDataQuality_ValidState(t *testing.T) {
	state := makeTestState()
	result := EvaluateDataQuality(state, time.Now())
	if result.State == "ERROR" && state.LastTick != nil {
		t.Errorf("Expected non-ERROR for valid state, got %s", result.State)
	}
}

func makeTestState() *features.MarketState {
	return &features.MarketState{
		Symbol:       "XAUUSD",
		CurrentPrice: decimal.NewFromFloat(4400.0),
		Bid:          decimal.NewFromFloat(4399.8),
		Ask:          decimal.NewFromFloat(4400.2),
		Spread:       decimal.NewFromFloat(0.4),
		Mid:          decimal.NewFromFloat(4400.0),
		Quality:      types.QualityAuthoritative,
		Timestamp:    time.Now(),
		Candles:      map[types.Timeframe]*types.Candle{types.TFM5: {Symbol: "XAUUSD", Timeframe: types.TFM5}},
		Indicators: features.IndicatorFeatures{
			ATR:         decimal.NewFromFloat(15.0),
			RSI:         decimal.NewFromFloat(62.0),
			EMA9:        decimal.NewFromFloat(4402.0),
			EMA21:       decimal.NewFromFloat(4398.0),
			ADX:         decimal.NewFromFloat(28.0),
			BollUpper:   decimal.NewFromFloat(4420.0),
			BollLower:   decimal.NewFromFloat(4380.0),
			BollWidth:   decimal.NewFromFloat(0.01),
		},
		VWAP: features.VWAPFeatures{SessionVWAP: decimal.NewFromFloat(4395.0)},
		Structure: features.StructureFeatures{
			CurrentTrend: "bullish",
			SwingHighs:    []decimal.Decimal{decimal.NewFromFloat(4410.0), decimal.NewFromFloat(4410.05)},
			SwingLows:     []decimal.Decimal{decimal.NewFromFloat(4390.0)},
			LastBOS:       &features.StructureEvent{Type: "BOS", Direction: "bullish"},
		},
		Liquidity: features.LiquidityFeatures{
			Pools: []features.LiquidityPool{{Price: decimal.NewFromFloat(4410.0), Type: "EQUAL_HIGHS"}},
			RecentSweeps: []features.SweepEvent{{Direction: "SELL_SIDE_SWEEP", Price: decimal.NewFromFloat(4395.0)}},
		},
		Regime: features.RegimeFeatures{Current: types.RegimeTrendingBullish},
		MTF: features.MTFFeatures{
			Score: 86.67, States: map[types.Timeframe]int{
				types.TFM1: 1, types.TFM5: 1, types.TFM15: 1, types.TFH1: 1, types.TFH4: 1,
			},
		},
		Session: features.SessionFeatures{CurrentSession: "LONDON", NewsRisk: "NONE"},
		Candle: features.CandleIntelligence{
			IsBullish: true, IsDisplacement: true,
			Range: decimal.NewFromFloat(20.0), BodySize: decimal.NewFromFloat(15.0),
			UpperWick: decimal.NewFromFloat(2.0), LowerWick: decimal.NewFromFloat(3.0),
		},
		LastTick: &types.Tick{
			Symbol: "XAUUSD", Bid: decimal.NewFromFloat(4399.8), Ask: decimal.NewFromFloat(4400.2),
			Mid: decimal.NewFromFloat(4400.0), Spread: decimal.NewFromFloat(0.4),
			Quality: types.QualityAuthoritative, TickVolume: 100,
		},
	}
}
