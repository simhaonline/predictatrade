// Package signal ties strategy + broker policy + gates into a single decision.
// The engine only ever emits an EXECUTABLE signal when: the strategy is
// directional, the broker policy permits the strategy, and every hard gate passes.
package signal

import (
	"pat-engine/internal/broker"
	"pat-engine/internal/config"
	"pat-engine/internal/gates"
	"pat-engine/internal/strategy"
	"pat-engine/internal/types"
)

// Decision is the result of evaluating one MarketState.
type Decision struct {
	Signal  types.Signal
	Blocked bool
	Reasons []string
}

// Decide runs the full v1 decision pipeline for a single snapshot.
func Decide(state *types.MarketState, strat strategy.Strategy, cfg config.StrategyConfig, pol *broker.BrokerPolicy) Decision {
	d := Decision{}
	res := strat.Evaluate(state)

	// No directional idea -> blocked by the strategy itself.
	if res.Direction != types.DirBuy && res.Direction != types.DirSell {
		d.Blocked = true
		d.Reasons = append(d.Reasons, res.ReasonCodes...)
		d.Signal = types.Signal{
			StrategyID:  res.StrategyID,
			Direction:   res.Direction,
			ReasonCodes: res.ReasonCodes,
		}
		return d
	}

	// Broker policy gate (scalping allowed? min-hold honoured?).
	if pol != nil {
		if ok, why := pol.StrategyAllowed(string(res.StrategyID)); !ok {
			d.Blocked = true
			d.Reasons = append(d.Reasons, why)
			d.Signal = types.Signal{
				StrategyID:  res.StrategyID,
				Direction:   res.Direction,
				EntryPrice:  res.EntryPrice,
				StopLoss:    res.StopLoss,
				TP1:         res.TP1,
				ReasonCodes: []string{why},
			}
			return d
		}
	}

	// Hard risk gates.
	v := gates.EvaluateAll(state, res, cfg, pol)
	if len(v) > 0 {
		d.Blocked = true
		for _, x := range v {
			d.Reasons = append(d.Reasons, x.ID+": "+x.Reason)
		}
		d.Signal = types.Signal{
			StrategyID: res.StrategyID,
			Direction:  res.Direction,
			EntryPrice: res.EntryPrice,
			StopLoss:   res.StopLoss,
			TP1:        res.TP1,
			TP2:        res.TP2,
			TP3:        res.TP3,
		}
		return d
	}

	// Executable.
	d.Signal = types.Signal{
		StrategyID:  res.StrategyID,
		Direction:   res.Direction,
		EntryPrice:  res.EntryPrice,
		StopLoss:    res.StopLoss,
		TP1:         res.TP1,
		TP2:         res.TP2,
		TP3:         res.TP3,
		RawScore:    res.RawScore,
		ReasonCodes: res.ReasonCodes,
		Executable:  true,
	}
	return d
}
