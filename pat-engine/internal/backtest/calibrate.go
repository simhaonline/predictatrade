package backtest

import (
	"fmt"

	"pat-engine/internal/broker"
	"pat-engine/internal/calibrate"
	"pat-engine/internal/config"
	"pat-engine/internal/license"
	"pat-engine/internal/signal"
	"pat-engine/internal/strategy"
	"pat-engine/internal/types"
)

// LabeledOutcomes runs the full live pipeline over the snapshot series and returns
// one Outcome per EXECUTABLE signal, labeled by whether the honest simulator made it
// to the 1R partial (the TP1_BEFORE_SL target). This is the training/validation label
// set for the calibrator — it uses the exact same Decide + Simulate code as live, so
// calibration transfers to production without a research drift.
func LabeledOutcomes(states []*types.MarketState, pol *broker.BrokerPolicy, lic *license.License) []calibrate.Outcome {
	cfgs := config.AllDefaults()
	strats := strategy.All()
	exec := broker.ExecutionProfile{}
	if pol != nil {
		exec = pol.Execution
	}

	var outs []calibrate.Outcome
	for _, st := range strats {
		cfg := cfgs[string(st.ID())]
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
			pnl := Simulate(states, i, d.Signal.Direction, d.Signal.EntryPrice, d.Signal.StopLoss,
				d.Signal.TP1, d.Signal.TP2, d.Signal.TP3, maxBars, exec)
			outs = append(outs, calibrate.Outcome{
				StrategyID: string(st.ID()),
				Regime:     s.Regime,
				RawScore:   d.Signal.RawScore,
				Win:        pnl > 0,
			})
		}
	}
	return outs
}

// FitCalibration fits an empirical calibrator on a (training) window of states.
func FitCalibration(states []*types.MarketState, pol *broker.BrokerPolicy, lic *license.License) *calibrate.EmpiricalModel {
	m := calibrate.NewEmpirical(10)
	m.Fit(LabeledOutcomes(states, pol, lic))
	return m
}

// ValidateCalibration measures calibration QUALITY on a held-out window: it buckets
// the predicted probability and checks the realized win rate inside each bucket is
// close to the prediction (reliability). A model that is well-calibrated has
// realized≈predicted across buckets. Uncalibrated setups (ok=false) are excluded.
func ValidateCalibration(m *calibrate.EmpiricalModel, states []*types.MarketState, pol *broker.BrokerPolicy, lic *license.License) string {
	outs := LabeledOutcomes(states, pol, lic)
	const nb = 5
	type cell struct {
		predSum float64
		n       int
		wins    int
	}
	cells := make([]cell, nb)
	skipped := 0
	for _, o := range outs {
		p, _, _, ok := m.Predict(o.StrategyID, o.Regime, o.RawScore)
		if !ok {
			skipped++
			continue
		}
		bi := int(p * float64(nb))
		if bi >= nb {
			bi = nb - 1
		}
		if bi < 0 {
			bi = 0
		}
		cells[bi].predSum += p
		cells[bi].n++
		if o.Win {
			cells[bi].wins++
		}
	}
	out := "\n=== CALIBRATION RELIABILITY (held-out) ===\n"
	out += fmt.Sprintf("%-12s %8s %8s %10s %12s\n", "PROB BUCKET", "N", "PRED", "REALIZED", "GAP")
	for i := 0; i < nb; i++ {
		c := cells[i]
		if c.n == 0 {
			continue
		}
		pred := c.predSum / float64(c.n)
		realized := float64(c.wins) / float64(c.n)
		gap := realized - pred
		out += fmt.Sprintf("[%.1f-%.1f) %8d %8.2f %10.2f %12.2f\n",
			float64(i)/float64(nb), float64(i+1)/float64(nb), c.n, pred, realized, gap)
	}
	out += fmt.Sprintf("excluded (UNCALIBRATED): %d of %d setups\n", skipped, len(outs))
	return out
}
