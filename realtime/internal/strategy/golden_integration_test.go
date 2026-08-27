// Package strategy_test — External integration tests for golden verification.
// These tests use the signal engine, gates, and calibration from outside the
// strategy package to avoid import cycles.
package strategy_test

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/calibration"
	"github.com/predictatrade/realtime/internal/features"
	"github.com/predictatrade/realtime/internal/gates"
	sigengine "github.com/predictatrade/realtime/internal/signal"
	"github.com/predictatrade/realtime/internal/strategy"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

// markValidated promotes the seeded default model to VALIDATED so Calibrate can
// return a real (non-fabricated) probability. Under BE-5, Calibrate only emits a
// probability for a VALIDATED model; the PROVISIONAL seed must not be surfaced.
func markValidated(c *calibration.Consumer, sid types.StrategyID) {
	m := c.GetModel(sid)
	if m == nil {
		return
	}
	m.IsActive = true
	m.Status = "VALIDATED"
}

func TestGolden_Calibration_ClampsHighScore(t *testing.T) {
	c := calibration.NewConsumer()
	c.SeedDefaultModels()
	markValidated(c, types.StrategyStandardScalping)
	prob, ok := c.Calibrate(types.StrategyStandardScalping, decimal.NewFromFloat(209.7))
	if !ok {
		t.Fatalf("expected a validated calibration probability, got ok=false")
	}
	probF, _ := prob.Float64()
	if probF > 0.90 {
		t.Errorf("Clamped probability for score 209.7 = %.4f, expected < 0.90 (no sigmoid saturation)", probF)
	}
	if probF < 0.80 {
		t.Errorf("Clamped probability for score 209.7 = %.4f, expected > 0.80", probF)
	}
}

func TestGolden_Calibration_ClampsZeroScore(t *testing.T) {
	c := calibration.NewConsumer()
	c.SeedDefaultModels()
	markValidated(c, types.StrategyStandardScalping)
	prob, ok := c.Calibrate(types.StrategyStandardScalping, decimal.Zero)
	if !ok {
		t.Fatalf("expected a validated calibration probability, got ok=false")
	}
	probF, _ := prob.Float64()
	if probF < 0.30 || probF > 0.50 {
		t.Errorf("Probability for score 0 = %.4f, expected ~0.378", probF)
	}
}

func TestGolden_Calibration_BoundedZeroToOne(t *testing.T) {
	c := calibration.NewConsumer()
	c.SeedDefaultModels()
	markValidated(c, types.StrategyStandardScalping)
	for _, score := range []float64{0, 25, 50, 75, 100, 150, 200, 500, -50} {
		prob, ok := c.Calibrate(types.StrategyStandardScalping, decimal.NewFromFloat(score))
		if !ok {
			t.Fatalf("expected a validated calibration probability for score %.1f, got ok=false", score)
		}
		probF, _ := prob.Float64()
		if probF < 0.0 || probF > 1.0 {
			t.Errorf("Probability for score %.1f = %.4f, out of [0,1] bounds", score, probF)
		}
	}
}

// TestGolden_Calibration_NoFabrication verifies BE-5: a PROVISIONAL (unvalidated)
// model must NOT produce a probability. Calibrate must return (0, false) so the
// engine never surfaces a fabricated confidence value to subscribers.
func TestGolden_Calibration_NoFabrication(t *testing.T) {
	c := calibration.NewConsumer()
	c.SeedDefaultModels() // seeds are PROVISIONAL, not VALIDATED
	prob, ok := c.Calibrate(types.StrategyStandardScalping, decimal.NewFromFloat(80))
	if ok {
		t.Errorf("PROVISIONAL seed model must not yield a probability (ok=false), got prob=%s", prob.String())
	}
	if !prob.IsZero() {
		t.Errorf("expected zero sentinel probability for unvalidated model, got %s", prob.String())
	}
}

func TestGolden_BLOCKED_HardGateVeto(t *testing.T) {
	gateReg := gates.NewRegistry()
	gateReg.Register(&gates.DataQualityGate{})
	gateReg.Register(&gates.SessionGate{})
	gateReg.Register(&gates.NewsGate{})
	gateReg.Register(&gates.SpreadGate{MaxSpreadAbsolute: 0.10, MaxSpreadToATR: 0.10})
	gateReg.Register(&gates.SlippageGate{MaxSlippage: 0.10})
	gateReg.Register(&gates.TotalCostGate{MaxCostToTarget: 0.25})
	gateReg.Register(&gates.ExposureGate{MaxExposure: 5})
	gateReg.Register(&gates.MarginGate{})
	gateReg.Register(&gates.RRNetExpectancyGate{MinGrossRR: 1.0})
	gateReg.Register(&gates.EntitlementGate{})
	gateReg.Register(&gates.LicenseGate{})
	gateReg.Register(&gates.ExecutionPermissionGate{})

	now := time.Date(2026, 1, 15, 10, 0, 0, 0, time.UTC)
	for _, gid := range []types.GateID{
		types.GateDataQuality, types.GateSession, types.GateNews,
		types.GateSpread, types.GateSlippage, types.GateTotalCost,
		types.GateExposure, types.GateMargin, types.GateRRNetExpectancy,
		types.GateEntitlement, types.GateLicense, types.GateExecutionPermit,
	} {
		gateReg.UpdateState(gid, gates.GateState{
			State: types.GatePass, EvaluatedAt: now, ValidUntil: now.Add(60 * time.Second),
		})
	}

	engine := sigengine.NewEngine(gateReg)
	tick := &types.Tick{
		Symbol: "XAUUSD", Bid: decimal.NewFromFloat(4399.8), Ask: decimal.NewFromFloat(4400.3),
		Mid: decimal.NewFromFloat(4400.05), Spread: decimal.NewFromFloat(0.50),
		Quality: types.QualityAuthoritative,
	}
	decision := engine.Decide(sigengine.DecisionInput{
		StrategyID: types.StrategyStandardScalping,
		Direction:  types.DirectionBuy,
		RawScore:   decimal.NewFromFloat(85), LongScore: decimal.NewFromFloat(85), ShortScore: decimal.NewFromFloat(5),
		Tick: tick, Regime: types.RegimeTrendingBullish,
		Session: "LONDON", SessionAllowed: true,
		NewsRisk: "NONE",
		EntryPrice: decimal.NewFromFloat(4400), StopLoss: decimal.NewFromFloat(4390),
		TP1: decimal.NewFromFloat(4412), TP2: decimal.NewFromFloat(4418), TP3: decimal.NewFromFloat(4424),
		RoundTripCost: decimal.NewFromFloat(0.30), CurrentExposure: 0, MaxExposure: 5,
		EntitlementOK: true, LicenseActive: true, ExecutionPermitted: true,
	})

	if decision.Signal == nil {
		t.Fatal("Expected non-nil signal")
	}
	// prompt.md Section 17: Direction preserves market thesis (BUY), status/grade = BLOCKED
	if decision.Signal.Direction != types.DirectionBuy {
		t.Errorf("Expected direction BUY (preserved when blocked), got %s", decision.Signal.Direction)
	}
	if decision.Signal.Grade != types.GradeBlocked {
		t.Errorf("Expected grade BLOCKED, got %s", decision.Signal.Grade)
	}
	if decision.AllGatesPass {
		t.Error("Expected gates to NOT all pass (spread veto)")
	}
	if decision.FirstVeto == nil {
		t.Error("Expected a first veto from spread gate")
	}
}

func TestGolden_LiquiditySweep_CaseInsensitiveWiring(t *testing.T) {
	s := strategy.NewStandardScalping()
	state := makeBullishStateExt()
	state.Liquidity.RecentSweeps = []features.SweepEvent{
		{Direction: "SELL_SIDE_SWEEP", Price: decimal.NewFromFloat(4395.0)},
	}
	result := s.Evaluate(state)
	found := false
	for _, e := range result.Evidence {
		if e.Feature == "SELL_SIDE_SWEEP" && e.Direction == types.DirectionBuy {
			found = true
		}
	}
	if !found {
		t.Error("Expected SELL_SIDE_SWEEP to produce BUY evidence -- case mismatch fix not working")
	}
}

func makeBullishStateExt() *features.MarketState {
	return &features.MarketState{
		Symbol: "XAUUSD", CurrentPrice: decimal.NewFromFloat(4400.0),
		Bid: decimal.NewFromFloat(4399.8), Ask: decimal.NewFromFloat(4400.2),
		Spread: decimal.NewFromFloat(0.4), Mid: decimal.NewFromFloat(4400.0),
		Quality: types.QualityAuthoritative,
		Indicators: features.IndicatorFeatures{
			ATR: decimal.NewFromFloat(15.0), RSI: decimal.NewFromFloat(62.0),
			EMA9: decimal.NewFromFloat(4402.0), EMA21: decimal.NewFromFloat(4398.0),
			EMA50: decimal.NewFromFloat(4395.0), SMA200: decimal.NewFromFloat(4380.0),
			ADX: decimal.NewFromFloat(28.0), ADXPlusDI: decimal.NewFromFloat(25.0),
			ADXMinusDI: decimal.NewFromFloat(15.0), MACDMain: decimal.NewFromFloat(2.5),
			MACDSignal: decimal.NewFromFloat(1.0), StochMain: decimal.NewFromFloat(65.0),
			StochSignal: decimal.NewFromFloat(60.0), OsMA: decimal.NewFromFloat(1.5),
			CCI: decimal.NewFromFloat(50.0), BollUpper: decimal.NewFromFloat(4420.0),
			BollLower: decimal.NewFromFloat(4380.0), BollMiddle: decimal.NewFromFloat(4400.0),
		},
		VWAP: features.VWAPFeatures{SessionVWAP: decimal.NewFromFloat(4395.0)},
		Structure: features.StructureFeatures{
			CurrentTrend: "bullish",
			LastBOS: &features.StructureEvent{Type: "BOS", Direction: "bullish", Price: decimal.NewFromFloat(4398.0)},
		},
		Liquidity: features.LiquidityFeatures{},
		FVG: features.FVGFeatures{
			FVGs: []features.FVGZone{{Type: "BULLISH", Upper: decimal.NewFromFloat(4401.0), Lower: decimal.NewFromFloat(4399.0)}},
			OrderBlocks: []features.OrderBlock{{Type: "BULLISH", Upper: decimal.NewFromFloat(4397.0), Lower: decimal.NewFromFloat(4395.0)}},
		},
		Regime: features.RegimeFeatures{Current: types.RegimeTrendingBullish},
		MTF: features.MTFFeatures{
			Score: 86.67, States: map[types.Timeframe]int{
				types.TFM1: 1, types.TFM5: 1, types.TFM15: 1, types.TFM30: 0,
				types.TFH1: 1, types.TFH4: 1, types.TFD1: 0,
			},
		},
		Session: features.SessionFeatures{CurrentSession: "LONDON", NewsRisk: "NONE"},
		Candle: features.CandleIntelligence{IsBullish: true, IsDisplacement: true},
	}
}
