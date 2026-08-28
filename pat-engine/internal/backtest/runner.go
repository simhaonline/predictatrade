package backtest

import (
	"pat-engine/internal/broker"
	"pat-engine/internal/config"
	"pat-engine/internal/license"
	"pat-engine/internal/signal"
	"pat-engine/internal/strategy"
	"pat-engine/internal/types"
)

// Result summarises a strategy's backtest over a dataset.
type Result struct {
	Strategy   string
	Trades     int
	Wins       int
	GrossWin   float64
	GrossLoss  float64
	PF         float64
	WinRate    float64
	ExcludedBy string // populated when the broker policy excludes the strategy
}

// RunAll evaluates every strategy on the snapshot series using the live signal
// pipeline (strategy + broker policy + license + gates) and simulates outcomes.
func RunAll(states []*types.MarketState, pol *broker.BrokerPolicy, lic *license.License) []Result {
	cfgs := config.AllDefaults()
	strats := strategy.All()
	results := make([]Result, 0, len(strats))

	exec := broker.ExecutionProfile{}
	if pol != nil {
		exec = pol.Execution
	}

	for _, st := range strats {
		cfg := cfgs[string(st.ID())]
		r := Result{Strategy: string(st.ID())}

		// Entitlement gate: a license may only narrow what the broker policy allows.
		if lic != nil && !lic.AllowsStrategy(string(st.ID())) {
			r.ExcludedBy = "LICENSE_STRATEGY_NOT_ALLOWED"
			results = append(results, r)
			continue
		}

		// Broker-policy eligibility (mirrors live: a no-scalping broker excludes scalpers).
		if pol != nil {
			if ok, why := pol.StrategyAllowed(string(st.ID())); !ok {
				r.ExcludedBy = why
				results = append(results, r)
				continue
			}
		}

		maxBars := cfg.BacktestMaxBars
		if maxBars <= 0 {
			maxBars = 50
		}

		for i, s := range states {
			if s == nil {
				continue
			}
			d := signal.Decide(s, st, cfg, pol)
			if !d.Signal.Executable {
				continue
			}
			pnl := Simulate(states, i, d.Signal.Direction, d.Signal.EntryPrice, d.Signal.StopLoss, d.Signal.TP1, maxBars, exec)
			r.Trades++
			if pnl > 0 {
				r.Wins++
				r.GrossWin += pnl
			} else {
				r.GrossLoss += -pnl
			}
		}
		if r.GrossLoss > 0 {
			r.PF = r.GrossWin / r.GrossLoss
		}
		if r.Trades > 0 {
			r.WinRate = float64(r.Wins) / float64(r.Trades)
		}
		results = append(results, r)
	}
	return results
}
