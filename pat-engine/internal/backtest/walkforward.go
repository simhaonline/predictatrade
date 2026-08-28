package backtest

import (
	"fmt"
	"pat-engine/internal/broker"
	"pat-engine/internal/license"
	"pat-engine/internal/types"
)

// FoldResult holds the out-of-sample backtest for one walk-forward fold.
type FoldResult struct {
	Index    int
	From     int
	To       int
	Results  []Result
}

// WalkForward runs the fixed (research-derived, NOT data-mined) strategy set over
// contiguous, non-overlapping out-of-sample folds of the historical series. Because
// the strategy parameters are configured from the scalping research playbook rather
// than learned from the data, every fold is a genuine out-of-sample test — there is
// no look-ahead and no parameter re-fitting between folds. This is the honest
// stability check the SOW requires before any performance claim.
func WalkForward(states []*types.MarketState, warmup, foldBars int, pol *broker.BrokerPolicy, lic *license.License) []FoldResult {
	if foldBars <= 0 {
		foldBars = 40000
	}
	if warmup < 200 {
		warmup = 200
	}
	var folds []FoldResult
	idx := 0
	for start := warmup; start+foldBars < len(states); start += foldBars {
		end := start + foldBars
		foldStates := states[start:end]
		res := RunAll(foldStates, pol, lic)
		folds = append(folds, FoldResult{Index: idx, From: start, To: end, Results: res})
		idx++
	}
	return folds
}

// SummarizeFolds prints an honest, plain-text OOS report. No numbers are smoothed;
// folds that fail are reported as such.
func SummarizeFolds(folds []FoldResult) string {
	out := ""
	for _, f := range folds {
		out += fmt.Sprintf("\n=== OOS Fold %d (bars %d..%d) ===\n", f.Index, f.From, f.To)
		for _, r := range f.Results {
			if r.ExcludedBy != "" {
				out += fmt.Sprintf("  %-20s EXCLUDED (%s)\n", r.Strategy, r.ExcludedBy)
				continue
			}
			out += fmt.Sprintf("  %-20s trades=%-6d win=%.1f%% PF=%.3f\n",
				r.Strategy, r.Trades, r.WinRate*100, r.PF)
		}
	}
	// Aggregate per strategy across folds (requires each fold to contain the strategy).
	type agg struct {
		trades, wins int
		grossWin, grossLoss float64
	}
	byStrat := map[string]*agg{}
	for _, f := range folds {
		for _, r := range f.Results {
			if r.ExcludedBy != "" {
				continue
			}
			a := byStrat[r.Strategy]
			if a == nil {
				a = &agg{}
				byStrat[r.Strategy] = a
			}
			a.trades += r.Trades
			a.wins += r.Wins
			a.grossWin += r.GrossWin
			a.grossLoss += r.GrossLoss
		}
	}
	out += "\n=== Aggregated OOS (all folds) ===\n"
	for s, a := range byStrat {
		pf := 0.0
		if a.grossLoss > 0 {
			pf = a.grossWin / a.grossLoss
		}
		wr := 0.0
		if a.trades > 0 {
			wr = float64(a.wins) / float64(a.trades)
		}
		out += fmt.Sprintf("  %-20s trades=%-6d win=%.1f%% PF=%.3f\n", s, a.trades, wr*100, pf)
	}
	return out
}
