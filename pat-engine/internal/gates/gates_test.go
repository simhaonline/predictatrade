package gates

import (
	"testing"

	"pat-engine/internal/broker"
	"pat-engine/internal/config"
	"pat-engine/internal/strategy"
	"pat-engine/internal/types"
)

func TestNetRRAfterCostPasses(t *testing.T) {
	res := strategy.StrategyResult{
		Direction:  types.DirBuy,
		EntryPrice: 2000.0,
		StopLoss:   1999.5,
		TP1:        2001.5,
	}
	st := types.MarketState{Spread: 0.2}
	pol := &broker.BrokerPolicy{Digits: 2, Execution: broker.DefaultXAUUSDExecution()}
	cfg := config.StrategyConfig{MinRR: 1.5}

	v := EvaluateAll(&st, res, cfg, pol)
	if len(v) > 0 {
		t.Fatalf("expected no vetoes, got %+v", v)
	}
}

func TestNetRRNegativeAfterCost(t *testing.T) {
	res := strategy.StrategyResult{
		Direction:  types.DirBuy,
		EntryPrice: 2000.0,
		StopLoss:   1999.5,
		TP1:        2000.2, // reward 0.2 < spread+commission+swap cost => net negative
	}
	st := types.MarketState{Spread: 0.2}
	pol := &broker.BrokerPolicy{Digits: 2, Execution: broker.DefaultXAUUSDExecution()}
	cfg := config.StrategyConfig{MinRR: 0.1} // pass gross gate; net must fail

	v := EvaluateAll(&st, res, cfg, pol)
	for _, vt := range v {
		if vt.ID == "NET_RR_NEGATIVE" || vt.ID == "NET_RR_BELOW_MIN" {
			return
		}
	}
	t.Fatalf("expected a net R:R veto, got %+v", v)
}
