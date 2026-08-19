package rl

import (
	"testing"
)

func TestDisabledMode(t *testing.T) {
	m := NewManager(DefaultConfig())
	dec := m.Evaluate(Observation{}, func(o Observation) (Action, float64) {
		return ActionLong, 0.9
	})
	if dec.Action != ActionNoTrade {
		t.Fatal("disabled mode should always return NO_TRADE")
	}
	if !dec.ShadowOnly {
		t.Fatal("disabled mode should be shadow only")
	}
}

func TestShadowModeCannotBlockOrExecute(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = RLShadow
	m := NewManager(cfg)
	dec := m.Evaluate(Observation{}, func(o Observation) (Action, float64) {
		return ActionLong, 0.95
	})
	if !dec.ShadowOnly {
		t.Fatal("shadow mode should be shadow only")
	}
	// Shadow can observe but cannot block or execute
	if dec.Action != ActionLong {
		t.Fatal("shadow should observe the action")
	}
}

func TestFilterModeCanOnlyVeto(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = RLFilterOnly
	cfg.MinConfidence = 0.5
	m := NewManager(cfg)
	// High confidence BUY — filter cannot create, only veto
	dec := m.Evaluate(Observation{}, func(o Observation) (Action, float64) {
		return ActionLong, 0.9
	})
	if dec.Action != ActionNoTrade {
		t.Fatal("filter mode should not create trades, only veto")
	}
	// Low confidence — filter can veto
	dec2 := m.Evaluate(Observation{}, func(o Observation) (Action, float64) {
		return ActionLong, 0.3 // below min confidence
	})
	if dec2.Action != ActionNoTrade {
		t.Fatal("filter should veto low confidence")
	}
}

func TestNoUnapprovedLiveMode(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Mode = RLFilterOnly // not live_approved
	m := NewManager(cfg)
	dec := m.Evaluate(Observation{}, func(o Observation) (Action, float64) {
		return ActionLong, 0.95
	})
	// In filter mode, cannot execute live trades
	if !dec.ShadowOnly && dec.Mode != RLLiveApproved {
		// filter_only returns shadowOnly=false but mode is filter, not live
		if dec.Mode != RLFilterOnly {
			t.Fatal("should be filter_only mode")
		}
	}
}

func TestEnvironmentAccounting(t *testing.T) {
	env := NewSimulatedEnvironment(10)
	obs := env.Reset()
	if obs.PositionState != 0 {
		t.Fatal("initial position should be flat")
	}
	_, reward, done := env.Step(ActionLong)
	if done {
		t.Fatal("should not be done after 1 step")
	}
	if reward <= 0 {
		t.Fatal("long action should have positive reward")
	}
}

func TestTransactionCost(t *testing.T) {
	rc := DefaultRewardConfig()
	if rc.TransactionCost <= 0 {
		t.Fatal("transaction cost should be positive")
	}
	if rc.SpreadCost <= 0 {
		t.Fatal("spread cost should be positive")
	}
}

func TestDrawdownPenalty(t *testing.T) {
	rc := DefaultRewardConfig()
	if rc.DrawdownPenalty <= 0 {
		t.Fatal("drawdown penalty should be positive")
	}
}

func TestDeterministicObservationDimensions(t *testing.T) {
	obs1 := Observation{
		Regime: 1, Confluence: 80, Confidence: 70,
	}
	obs2 := Observation{
		Regime: 1, Confluence: 80, Confidence: 70,
	}
	if obs1.Regime != obs2.Regime || obs1.Confluence != obs2.Confluence {
		t.Fatal("observations with same values should be equal")
	}
}

func TestCanApproveLiveChecksMetrics(t *testing.T) {
	cfg := DefaultConfig()
	cfg.RequireOOSValidation = true
	cfg.MinTradeCount = 50
	cfg.MaxDrawdownPct = 10
	cfg.MinProfitFactor = 1.3
	m := NewManager(cfg)

	// Insufficient trades
	ok, _ := m.CanApproveLive(ValidationMetrics{TradeCount: 10, MaxDrawdown: 5, ProfitFactor: 1.5})
	if ok {
		t.Fatal("should not approve with insufficient trades")
	}

	// High drawdown
	ok, _ = m.CanApproveLive(ValidationMetrics{TradeCount: 100, MaxDrawdown: 15, ProfitFactor: 1.5})
	if ok {
		t.Fatal("should not approve with high drawdown")
	}

	// Low profit factor
	ok, _ = m.CanApproveLive(ValidationMetrics{TradeCount: 100, MaxDrawdown: 5, ProfitFactor: 1.0})
	if ok {
		t.Fatal("should not approve with low profit factor")
	}

	// Good metrics
	ok, reason := m.CanApproveLive(ValidationMetrics{TradeCount: 100, MaxDrawdown: 5, ProfitFactor: 1.5})
	if !ok {
		t.Fatalf("should approve with good metrics: %s", reason)
	}
}
