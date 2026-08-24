package signal

import (
	"testing"
	"time"

	"github.com/predictatrade/realtime/internal/adaptation"
	"github.com/predictatrade/realtime/internal/gates"
	"github.com/predictatrade/realtime/internal/hedging"
	"github.com/predictatrade/realtime/internal/ml"
	"github.com/predictatrade/realtime/internal/recovery"
	"github.com/predictatrade/realtime/internal/rl"
	"github.com/predictatrade/realtime/internal/sentiment"
	"github.com/predictatrade/realtime/internal/types"
	"github.com/shopspring/decimal"
)

func setupEngineWithGates() *Engine {
	reg := gates.NewRegistry()
	// Register all gates with PASS state
	now := time.Now()
	for _, gateID := range []types.GateID{
		types.GateDataQuality, types.GateSession, types.GateNews,
		types.GateSpread, types.GateSlippage, types.GateTotalCost,
		types.GateExposure, types.GateMargin, types.GateRRNetExpectancy,
		types.GateEntitlement, types.GateLicense, types.GateExecutionPermit,
		// Self-evaluating precision gates are part of the canonical order
		// (main.go registers + seeds them; mirror that here).
		types.GateMinATR, types.GateStopHuntFilter,
	} {
		reg.Register(&passGate{gateID: gateID})
		reg.UpdateState(gateID, gates.GateState{
			GateID:        gateID,
			State:         types.GatePass,
			EvaluatedAt:   now,
			ValidUntil:    now.Add(time.Hour),
			SourceVersion: "test",
		})
	}
	return NewEngine(reg)
}

// passGate always passes
type passGate struct {
	gateID types.GateID
}

func (g *passGate) ID() types.GateID { return g.gateID }
func (g *passGate) Evaluate(input gates.GateInput, state gates.GateState) gates.GateEvaluation {
	return gates.GateEvaluation{
		GateID:      g.gateID,
		Result:      types.GatePass,
		EvaluatedAt: time.Now(),
	}
}

func makeBaseInput(strat types.StrategyID) DecisionInput {
	return DecisionInput{
		StrategyID: strat,
		Direction:  types.DirectionBuy,
		RawScore:   decimal.NewFromInt(80),
		LongScore:  decimal.NewFromInt(80),
		ShortScore: decimal.NewFromInt(20),
		Tick: &types.Tick{
			Symbol:          "XAUUSD",
			Bid:             decimal.NewFromFloat(2400),
			Ask:             decimal.NewFromFloat(2400.3),
			Mid:             decimal.NewFromFloat(2400.15),
			Spread:          decimal.NewFromFloat(0.3),
			Source:          "LIVE_MASTER_NODE",
			SourceTimestamp: time.Now(),
			Quality:         types.QualityAuthoritative,
		},
		Regime:             types.RegimeTrendingBullish,
		Session:            "LONDON",
		SessionAllowed:     true,
		NewsRisk:           "LOW",
		EntryPrice:         decimal.NewFromFloat(2400),
		StopLoss:           decimal.NewFromFloat(2390),
		TP1:                decimal.NewFromFloat(2415),
		TP2:                decimal.NewFromFloat(2430),
		TP3:                decimal.NewFromFloat(2450),
		RoundTripCost:      decimal.NewFromFloat(0.3),
		CurrentExposure:    1,
		MaxExposure:        5,
		EntitlementOK:      true,
		LicenseActive:      true,
		ExecutionPermitted: true,
	}
}

func makeAdvancedInput(base DecisionInput, strat types.StrategyID) AdvancedDecisionInput {
	return AdvancedDecisionInput{
		DecisionInput: base,
		AccountID:     "test-acc",
		Confluence:    85,
		SetupGrade:    "A",
		Confidence:    80,
		MarketContext: adaptation.ContextInput{
			Regime: "TRENDING_BULLISH",
		},
	}
}

func TestIntegrationAllFourStrategies(t *testing.T) {
	engine := setupEngineWithGates()

	adv := &AdvancedManagers{
		Recovery:   recovery.NewManager(recovery.DefaultConfig()),
		Adaptation: adaptation.NewManager(adaptation.DefaultConfig()),
		Hedging:    hedging.NewManager(hedging.DefaultConfig()),
		ML:         ml.NewAdaptationManager(ml.DefaultConfig(), ml.NewModelRegistry()),
		RL:         rl.NewManager(rl.DefaultConfig()),
		Sentiment:  sentiment.NewEngine(sentiment.DefaultConfig(), nil),
	}

	strategies := types.AllStrategies()
	for _, strat := range strategies {
		base := makeBaseInput(strat)
		advInput := makeAdvancedInput(base, strat)
		result := engine.DecideWithAdvanced(advInput, adv)

		if result.Signal == nil {
			t.Fatalf("strategy %s: signal should not be nil", strat)
		}
		// Verify signal is not blocked when all gates pass and recovery is NORMAL
		if result.Signal.Direction == "BLOCKED" {
			t.Fatalf("strategy %s: should not be blocked in NORMAL state, reason: %s", strat, result.AdvancedBlockReason)
		}
		// Verify recovery state is NORMAL
		if result.RecoveryState != recovery.StateNormal {
			t.Fatalf("strategy %s: recovery should be NORMAL, got %s", strat, result.RecoveryState)
		}
	}
}

func TestIntegrationRecoveryBlocksSignal(t *testing.T) {
	engine := setupEngineWithGates()

	recCfg := recovery.DefaultConfig()
	recMgr := recovery.NewManager(recCfg)
	key := recovery.AccountStrategyKey{
		AccountID:  "test-acc",
		StrategyID: "STANDARD_SCALPING",
		Symbol:     "XAUUSD",
	}
	recMgr.SetStateRecord(&recovery.StateRecord{
		Key:        key,
		State:      recovery.StateHalted,
		HaltUntil:  time.Now().Add(1 * time.Hour),
		TradingDay: time.Now(),
	})

	adv := &AdvancedManagers{
		Recovery: recMgr,
	}

	base := makeBaseInput(types.StrategyStandardScalping)
	advInput := makeAdvancedInput(base, types.StrategyStandardScalping)
	result := engine.DecideWithAdvanced(advInput, adv)

	if !result.BlockedByAdvanced {
		t.Fatal("signal should be blocked by recovery halt")
	}
	if result.Signal.Direction != "BLOCKED" {
		t.Fatalf("expected BLOCKED, got %s", result.Signal.Direction)
	}
}

func TestIntegrationNOTradeRemainsValid(t *testing.T) {
	engine := setupEngineWithGates()

	adv := &AdvancedManagers{
		Recovery: recovery.NewManager(recovery.DefaultConfig()),
	}

	base := makeBaseInput(types.StrategyStandardScalping)
	base.Direction = types.DirectionNoTrade // strategy says NO-TRADE
	advInput := makeAdvancedInput(base, types.StrategyStandardScalping)
	result := engine.DecideWithAdvanced(advInput, adv)

	if result.Signal == nil {
		t.Fatal("signal should not be nil")
	}
	if result.Signal.Direction != types.DirectionNoTrade {
		t.Fatalf("expected NO-TRADE, got %s", result.Signal.Direction)
	}
}

func TestIntegrationSentimentInfluenceApplied(t *testing.T) {
	engine := setupEngineWithGates()

	sentCfg := sentiment.DefaultConfig()
	sentCfg.Enabled = true
	sentEngine := sentiment.NewEngine(sentCfg, nil)

	adv := &AdvancedManagers{
		Sentiment: sentEngine,
	}

	base := makeBaseInput(types.StrategyStandardScalping)
	advInput := makeAdvancedInput(base, types.StrategyStandardScalping)
	result := engine.DecideWithAdvanced(advInput, adv)

	// Sentiment with no providers should give 0 influence (neutral)
	if result.SentimentScore != 0 {
		t.Fatalf("expected 0 sentiment influence, got %f", result.SentimentScore)
	}
}

func TestIntegrationRLFilterCanVeto(t *testing.T) {
	engine := setupEngineWithGates()

	rlCfg := rl.DefaultConfig()
	rlCfg.Mode = rl.RLFilterOnly
	rlCfg.MinConfidence = 0.5
	rlMgr := rl.NewManager(rlCfg)

	adv := &AdvancedManagers{
		RL: rlMgr,
	}

	base := makeBaseInput(types.StrategyStandardScalping)
	advInput := makeAdvancedInput(base, types.StrategyStandardScalping)
	// RL inference returns NO_TRADE with high confidence — filter should veto
	advInput.RLInferenceFn = func(obs rl.Observation) (rl.Action, float64) {
		return rl.ActionNoTrade, 0.9
	}
	result := engine.DecideWithAdvanced(advInput, adv)

	if !result.BlockedByAdvanced {
		t.Fatal("RL filter should veto with NO_TRADE action")
	}
	if result.AdvancedBlockReason != "RL_FILTER_VETO" {
		t.Fatalf("expected RL_FILTER_VETO, got %s", result.AdvancedBlockReason)
	}
}

func TestIntegrationSizeMultiplierFromRecovery(t *testing.T) {
	engine := setupEngineWithGates()

	recMgr := recovery.NewManager(recovery.DefaultConfig())
	key := recovery.AccountStrategyKey{
		AccountID:  "test-acc",
		StrategyID: "STANDARD_SCALPING",
		Symbol:     "XAUUSD",
	}
	recMgr.SetStateRecord(&recovery.StateRecord{
		Key:        key,
		State:      recovery.StateRecovery,
		TradingDay: time.Now(),
	})

	adv := &AdvancedManagers{
		Recovery: recMgr,
	}

	base := makeBaseInput(types.StrategyStandardScalping)
	advInput := makeAdvancedInput(base, types.StrategyStandardScalping)
	// In recovery with valid confluence/grade/confidence, signal should pass with reduced size
	advInput.Confluence = 85
	advInput.SetupGrade = "A"
	advInput.Confidence = 80
	result := engine.DecideWithAdvanced(advInput, adv)

	if result.SizeMultiplier != 0.5 {
		t.Fatalf("expected 0.5 size multiplier in recovery, got %f", result.SizeMultiplier)
	}
}

func TestIntegrationAdaptationApplied(t *testing.T) {
	engine := setupEngineWithGates()

	advMgr := adaptation.NewManager(adaptation.DefaultConfig())

	adv := &AdvancedManagers{
		Adaptation: advMgr,
	}

	base := makeBaseInput(types.StrategyStandardScalping)
	advInput := makeAdvancedInput(base, types.StrategyStandardScalping)
	advInput.MarketContext = adaptation.ContextInput{
		Regime:          "TRENDING_BULLISH",
		VolatilityState: "NORMAL",
	}
	result := engine.DecideWithAdvanced(advInput, adv)

	if result.AdaptationPhase != adaptation.PhaseTrending {
		t.Fatalf("expected TRENDING phase, got %s", result.AdaptationPhase)
	}
}

func TestIntegrationMLFallbackWhenDisabled(t *testing.T) {
	engine := setupEngineWithGates()

	mlMgr := ml.NewAdaptationManager(ml.DefaultConfig(), ml.NewModelRegistry())

	adv := &AdvancedManagers{
		ML: mlMgr,
	}

	base := makeBaseInput(types.StrategyStandardScalping)
	advInput := makeAdvancedInput(base, types.StrategyStandardScalping)
	result := engine.DecideWithAdvanced(advInput, adv)

	// ML is disabled — no prediction should be set (or fallback)
	if result.MLPrediction != nil && !result.MLPrediction.FallbackUsed {
		t.Fatal("ML should use fallback when disabled")
	}
}
